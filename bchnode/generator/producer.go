package generator

import (
	"sync"
	"time"

	"github.com/smartbch/testkit/bchnode/generator/types"
)

type Producer struct {
	ExitChan          chan bool
	ReorgChan         chan bool
	MonitorPubkeyChan chan string
	CCTxChan          chan types.TxInfo
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
		case tx := <-p.CCTxChan:
			ctx.CCTxs = append(ctx.CCTxs, tx)
		default:
			bi := ctx.BuildBlockRespWithCoinbaseTx(ctx.getPubkey())
			if bi == nil {
				time.Sleep(10 * time.Second)
				//ctx.Log.Println("no validator pubkey and monitor pubkey info both")
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
	//fmt.Printf("len of pubkeyInfoSet: %d\n", len(ctx.PubKeyInfoSet))
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
