package api

import (
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/smartbch/testkit/bchnode/generator"
)

type BlockCountService struct {
	ctx *generator.Context
}

func MakeBlockCountService(ctx *generator.Context) BlockCountService {
	return BlockCountService{ctx: ctx}
}

func (b *BlockCountService) Call(r *http.Request, _ *string, result *int64) error {
	b.ctx.RWLock.RLock()
	*result = b.ctx.NextBlockHeight - 1
	b.ctx.RWLock.RUnlock()
	return nil
}

type BlockHashService struct {
	ctx *generator.Context
}

func MakeBlockHashService(ctx *generator.Context) BlockHashService {
	return BlockHashService{ctx: ctx}
}

func (b *BlockHashService) Call(r *http.Request, args *int64, result *string) error {
	var ok bool
	b.ctx.RWLock.RLock()
	*result, ok = b.ctx.BlkHashByHeight[*args]
	b.ctx.RWLock.RUnlock()
	if !ok {
		return errors.New("no such height")
	}
	return nil
}

type BlockService struct {
	ctx *generator.Context
}

func MakeBlockService(ctx *generator.Context) BlockService {
	return BlockService{ctx: ctx}
}

func (b *BlockService) Call(r *http.Request, args *string, result *generator.BlockInfo) error {
	b.ctx.RWLock.RLock()
	info, ok := b.ctx.BlkByHash[*args]
	b.ctx.RWLock.RUnlock()
	if !ok {
		return errors.New("no such block hash")
	}
	*result = *info
	return nil
}

type TxService struct {
	ctx *generator.Context
}

func MakeTxService(ctx *generator.Context) TxService {
	return TxService{ctx: ctx}
}

func (t *TxService) Call(r *http.Request, args *string, result *generator.TxInfo) error {
	t.ctx.RWLock.RLock()
	info, ok := t.ctx.TxByHash[*args]
	t.ctx.RWLock.RUnlock()
	if !ok {
		return errors.New("No such tx hash")
	}
	*result = *info
	return nil
}

type PubKeyService struct {
	ctx *generator.Context
}

func MakePubKeyService(ctx *generator.Context) PubKeyService {
	return PubKeyService{ctx: ctx}
}

func (p *PubKeyService) Call(r *http.Request, args *string, result *string) error {
	s := strings.Split(*args, "-")
	if len(s) != 3 {
		return errors.New("invalid format")
	}
	info := &generator.PubKeyInfo{
		Pubkey: s[0],
	}
	vp, err := strconv.Atoi(s[1])
	fmt.Printf("voting power: %d\n", vp)
	if err != nil {
		return errors.New("invalid voting power")
	}
	info.RemainCount = int64(vp)
	info.VotingPower = int64(vp)
	p.ctx.RWLock.Lock()
	if s[2] == "add" || s[2] == "edit" {
		if info.VotingPower <= 0 {
			p.ctx.RWLock.Unlock()
			return errors.New("voting power should be positive when add or edit an validator")
		}
		p.ctx.PubkeyInfoByPubkey[info.Pubkey] = info
	} else if s[2] == "retire" {
		delete(p.ctx.PubkeyInfoByPubkey, info.Pubkey)
	} else {
		p.ctx.RWLock.Unlock()
		return errors.New("invalid action")
	}
	p.ctx.RWLock.Unlock()
	*result = "send success"
	return nil
}

type BlockIntervalService struct {
	ctx *generator.Context
}

func MakeBlockIntervalService(ctx *generator.Context) BlockIntervalService {
	return BlockIntervalService{ctx: ctx}
}

func (b *BlockIntervalService) Call(r *http.Request, args *int64, result *string) error {
	b.ctx.Producer.Lock.Lock()
	b.ctx.Producer.BlockIntervalTime = *args
	b.ctx.Producer.Lock.Unlock()
	*result = "send success"
	return nil
}

type BlockReorgService struct {
	ctx *generator.Context
}

func MakeBlockReorgService(ctx *generator.Context) BlockReorgService {
	return BlockReorgService{ctx: ctx}
}

func (b *BlockReorgService) Call(r *http.Request, args *int64, result *string) error {
	b.ctx.Producer.ReorgChan <- true
	*result = "send success"
	return nil
}

type CCService struct {
	ctx *generator.Context
}

func MakeCCService(ctx *generator.Context) CCService {
	return CCService{ctx: ctx}
}

func (c *CCService) Call(r *http.Request, args *string, result *string) error {
	pubkey := *args
	pubkeyBytes, err := hex.DecodeString(pubkey)
	if err != nil || len(pubkeyBytes) != 33 {
		*result = "must 33bytes pubkey hex string without 0x"
	}
	c.ctx.Producer.MonitorPubkeyChan <- *args
	*result = "send success"
	return nil
}
