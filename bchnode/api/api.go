package api

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/smartbch/testkit/bchnode/generator"
)

type BlockCountService struct{}

func (_ *BlockCountService) Call(r *http.Request, _ *string, result *int64) error {
	generator.Ctx.RWLock.RLock()
	*result = generator.Ctx.NextBlockHeight - 1
	generator.Ctx.RWLock.RUnlock()
	return nil
}

type BlockHashService struct{}

func (_ *BlockHashService) Call(r *http.Request, args *int64, result *string) error {
	var ok bool
	generator.Ctx.RWLock.RLock()
	*result, ok = generator.Ctx.BlkHashByHeight[*args]
	generator.Ctx.RWLock.RUnlock()
	if !ok {
		return errors.New("no such height")
	}
	return nil
}

type BlockService struct{}

func (_ *BlockService) Call(r *http.Request, args *string, result *generator.BlockInfo) error {
	generator.Ctx.RWLock.RLock()
	info, ok := generator.Ctx.BlkByHash[*args]
	generator.Ctx.RWLock.RUnlock()
	if !ok {
		return errors.New("no such block hash")
	}
	*result = *info
	return nil
}

type TxService struct{}

func (_ *TxService) Call(r *http.Request, args *string, result *generator.TxInfo) error {
	generator.Ctx.RWLock.RLock()
	info, ok := generator.Ctx.TxByHash[*args]
	generator.Ctx.RWLock.RUnlock()
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
	generator.Ctx.RWLock.Lock()
	if s[2] == "add" || s[2] == "modify" {
		generator.Ctx.PubkeyInfoByPubkey[info.Pubkey] = info
	} else {
		delete(generator.Ctx.PubkeyInfoByPubkey, info.Pubkey)
	}
	generator.Ctx.RWLock.Unlock()
	*result = "send success"
	return nil
}

type BlockInternalService struct{}

func (_ *BlockInternalService) Call(r *http.Request, args *int64, result *string) error {
	generator.Ctx.Producer.Lock.Lock()
	generator.Ctx.Producer.BlockInternalTime = *args
	generator.Ctx.Producer.Lock.Unlock()
	*result = "send success"
	return nil
}

type BlockReorgService struct{}

func (_ *BlockReorgService) Call(r *http.Request, args *int64, result *string) error {
	generator.Ctx.Producer.Reorg <- true
	*result = "send success"
	return nil
}
