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
	Exit              chan bool
	Reorg             chan bool
	Tx                chan string
	Lock              sync.Mutex
	BlockIntervalTime int64 //uint: second
}

func (p *Producer) Start() {
	for {
		select {
		case <-p.Exit:
			return
		case <-p.Reorg:
			ReorgBlock()
		case pubkey := <-p.Tx:
			BuildBlockWithCrossChainTx(pubkey)
		default:
			//BuildBlockWithCrossChainTx("034872060af10ec594db868ce81e16763828e30441916b37e5c31ea2154b46639a")
			bi := BuildBlockRespWithCoinbaseTx(getPubkey())
			if bi == nil {
				time.Sleep(10 * time.Second)
				Ctx.Log.Println("no pubkey info")
				continue
			}
			time.Sleep(time.Duration(p.BlockIntervalTime) * time.Second)
		}
	}
}

func getPubkey() string {
	Ctx.RWLock.RLock()
	if Ctx.PubkeyInfoIndex == 0 {
		reloadPubkeyInfo()
	} else if Ctx.PubkeyInfoIndex == len(Ctx.PubKeyInfoSet) {
		Ctx.PubkeyInfoIndex = 0
		reloadPubkeyInfo()
	}
	if len(Ctx.PubKeyInfoSet) == 0 {
		Ctx.RWLock.RUnlock()
		return ""
	}
	fmt.Printf("len of pubkeyInfoSet: %d\n", len(Ctx.PubKeyInfoSet))
	Ctx.RWLock.RUnlock()
	pi := &Ctx.PubKeyInfoSet[Ctx.PubkeyInfoIndex]
	if pi.RemainCount == 0 {
		panic("voting power remain should be positive")
	}
	pi.RemainCount--
	if pi.RemainCount == 0 {
		pi.RemainCount = pi.VotingPower
		Ctx.PubkeyInfoIndex++
	}
	return pi.Pubkey
}

func reloadPubkeyInfo() {
	Ctx.PubKeyInfoSet = make([]PubKeyInfo, 0, len(Ctx.PubkeyInfoByPubkey))
	for _, in := range Ctx.PubkeyInfoByPubkey {
		Ctx.PubKeyInfoSet = append(Ctx.PubKeyInfoSet, *in)
	}
}
