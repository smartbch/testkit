package generator

import (
	"fmt"
	"sync"
	"time"
)

var (
	crosschainTransferDefaultAmount int64 = 16
)

type Producer struct {
	ExitChan          chan bool
	ReorgChan         chan bool
	MonitorPubkeyChan chan string
	Lock              sync.Mutex
	BlockIntervalTime int64 //uint: second
}

func (p *Producer) Start(ctx *Context) {
	for {
		select {
		case <-p.ExitChan:
			return
		case <-p.ReorgChan:
			ctx.ReorgBlock()
		case pubkey := <-p.MonitorPubkeyChan:
			ctx.MonitorPubkey = pubkey
		default:
			//BuildBlockWithCrossChainTx("034872060af10ec594db868ce81e16763828e30441916b37e5c31ea2154b46639a")
			bi := ctx.BuildBlockRespWithCoinbaseTx(ctx.getPubkey())
			if bi == nil {
				time.Sleep(10 * time.Second)
				ctx.Log.Println("no pubkey info")
				continue
			}
			time.Sleep(time.Duration(p.BlockIntervalTime) * time.Second)
		}
	}
}

func (ctx *Context) getPubkey() string {
	ctx.RWLock.RLock()
	if ctx.PubkeyInfoIndex == 0 {
		ctx.reloadPubkeyInfo()
	} else if ctx.PubkeyInfoIndex == len(ctx.PubKeyInfoSet) {
		ctx.PubkeyInfoIndex = 0
		ctx.reloadPubkeyInfo()
	}
	if len(ctx.PubKeyInfoSet) == 0 {
		ctx.RWLock.RUnlock()
		return ""
	}
	fmt.Printf("len of pubkeyInfoSet: %d\n", len(ctx.PubKeyInfoSet))
	ctx.RWLock.RUnlock()
	pi := &ctx.PubKeyInfoSet[ctx.PubkeyInfoIndex]
	if pi.RemainCount == 0 {
		panic("voting power remain should be positive")
	}
	pi.RemainCount--
	if pi.RemainCount == 0 {
		pi.RemainCount = pi.VotingPower
		ctx.PubkeyInfoIndex++
	}
	return pi.Pubkey
}

func (ctx *Context) reloadPubkeyInfo() {
	ctx.PubKeyInfoSet = make([]PubKeyInfo, 0, len(ctx.PubkeyInfoByPubkey))
	for _, in := range ctx.PubkeyInfoByPubkey {
		ctx.PubKeyInfoSet = append(ctx.PubKeyInfoSet, *in)
	}
}
