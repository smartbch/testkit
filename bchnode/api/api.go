package api

import (
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/smartbch/testkit/bchnode/generator"
	"github.com/smartbch/testkit/bchnode/generator/types"
)

var ctx *generator.Context

func InitContext(c *generator.Context) {
	ctx = c
}

type BlockCountService struct{}

func (_ *BlockCountService) Call(r *http.Request, _ *string, result *int64) error {
	ctx.RWLock.RLock()
	*result = ctx.NextBlockHeight - 1
	ctx.RWLock.RUnlock()
	return nil
}

type BlockHashService struct{}

func (_ *BlockHashService) Call(r *http.Request, args *int64, result *string) error {
	var ok bool
	ctx.RWLock.RLock()
	*result, ok = ctx.BlkHashByHeight[*args]
	ctx.RWLock.RUnlock()
	if !ok {
		return errors.New("no such height")
	}
	return nil
}

type BlockService struct{}

func (_ *BlockService) Call(r *http.Request, args *string, result *types.BlockInfo) error {
	ctx.RWLock.RLock()
	info, ok := ctx.BlkByHash[*args]
	ctx.RWLock.RUnlock()
	if !ok {
		return errors.New("no such block hash")
	}
	*result = *info
	return nil
}

type TxService struct{}

func (_ *TxService) Call(r *http.Request, args *string, result *types.TxInfo) error {
	ctx.RWLock.RLock()
	info, ok := ctx.TxByHash[*args]
	ctx.RWLock.RUnlock()
	if !ok {
		return errors.New("No such tx hash")
	}
	*result = *info
	return nil
}

type PubKeyService struct{}

func (_ *PubKeyService) Call(r *http.Request, args *string, result *string) error {
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
	ctx.RWLock.Lock()
	if s[2] == "add" || s[2] == "edit" {
		if info.VotingPower <= 0 {
			ctx.RWLock.Unlock()
			return errors.New("voting power should be positive when add or edit an validator")
		}
		ctx.PubkeyInfoByPubkey[info.Pubkey] = info
	} else if s[2] == "retire" {
		delete(ctx.PubkeyInfoByPubkey, info.Pubkey)
	} else {
		ctx.RWLock.Unlock()
		return errors.New("invalid action")
	}
	ctx.RWLock.Unlock()
	*result = "send success"
	return nil
}

type BlockIntervalService struct{}

func (_ *BlockIntervalService) Call(r *http.Request, args *int64, result *string) error {
	ctx.Producer.Lock.Lock()
	ctx.Producer.BlockIntervalTime = *args
	ctx.Producer.Lock.Unlock()
	*result = "send success"
	return nil
}

type BlockReorgService struct{}

func (_ *BlockReorgService) Call(r *http.Request, args *int64, result *string) error {
	ctx.Producer.ReorgChan <- true
	*result = "send success"
	return nil
}

type MonitorVoteService struct{}

func (_ *MonitorVoteService) Call(r *http.Request, args *string, result *string) error {
	pubkey := *args
	pubkeyBytes, err := hex.DecodeString(pubkey)
	if err != nil || len(pubkeyBytes) != 33 {
		return errors.New("must 33bytes pubkey hex string without 0x")
	}
	ctx.Producer.MonitorPubkeyChan <- *args
	*result = "send success"
	return nil
}

type CCService struct{}

func (_ *CCService) Call(r *http.Request, args *string, result *string) error {
	tx := types.TxInfo{}
	if args == nil {
		fmt.Println("args is nil")
	}
	fmt.Println(*args)
	err := json.Unmarshal([]byte(*args), &tx)
	if err != nil {
		return errors.New("must bch tx json format")
	}
	ctx.Producer.CCTxChan <- tx
	*result = "send success"
	return nil
}
