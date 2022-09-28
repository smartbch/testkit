package main

import (
	"fmt"
	"os"
	"time"

	"github.com/smartbch/testkit/cctester/config"
	"github.com/smartbch/testkit/cctester/testcase"
	"github.com/smartbch/testkit/cctester/utils"
)

func main() {
	_ = os.Remove("out.log")
	_ = os.Remove("block.log")
	fmt.Println("-------------- start fake node --------------")
	go utils.ExecuteWithContinuousOutPut(config.FakeNodePath)
	time.Sleep(4 * time.Second)
	fmt.Println("-------------- send monitor vote --------------")
	go utils.SendMonitorVoteToFakeNode("000000000000000000000000000000000000000000000000000000000000000002")
	time.Sleep(10 * time.Second)
	fmt.Println("-------------- start side node --------------")
	go utils.StartSideChainNode()
	time.Sleep(10 * time.Second)
	fmt.Println("-------------- start test --------------")
	go testcase.Test()
	select {}
}
