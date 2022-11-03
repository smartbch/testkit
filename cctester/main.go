package main

import (
	"encoding/hex"
	"fmt"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartbch/testkit/cctester/config"
	"github.com/smartbch/testkit/cctester/testcase"
	"github.com/smartbch/testkit/cctester/utils"
)

func main() {
	key, _ := crypto.GenerateKey()
	rpcKey := hex.EncodeToString(crypto.FromECDSA(key))
	fmt.Printf("rpc key: %s\n", rpcKey)
	_ = os.Remove("out.log")
	_ = os.Remove("block.log")
	fmt.Println("-------------- start fake node --------------")
	go utils.ExecuteWithContinuousOutPut(config.FakeNodePath)
	time.Sleep(4 * time.Second)
	fmt.Println("-------------- send monitor vote --------------")
	go utils.SendMonitorVoteToFakeNode("000000000000000000000000000000000000000000000000000000000000000002")
	time.Sleep(6 * time.Second)
	fmt.Println("-------------- start side node --------------")
	go utils.StartSideChainNode()
	time.Sleep(6 * time.Second)
	utils.SetRpcKey(rpcKey)
	time.Sleep(1 * time.Second)
	fmt.Println("-------------- deploy Gov contracts --------------")
	nodesGovAddr := utils.DeployGovContracts()
	utils.InitSbchNodesGov(nodesGovAddr)
	time.Sleep(3 * time.Second)
	fmt.Println("-------------- start operators --------------")
	go utils.StartOperators(nodesGovAddr)
	time.Sleep(3 * time.Second)
	fmt.Println("-------------- start fake collector --------------")
	go utils.StartFakeCollector()
	time.Sleep(3 * time.Second)
	fmt.Println("-------------- start test --------------")
	go testcase.Test()
	select {}
}
