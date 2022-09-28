package main

import (
	"fmt"
	"github.com/smartbch/testkit/cctester/testcase"
	"os"
	"time"

	"github.com/smartbch/testkit/cctester/config"
	"github.com/smartbch/testkit/cctester/utils"
)

/*

#./smartbchd start --home $NODE_HOME --unlock $TEST_KEYS --https.addr=off --wss.addr=off \
#  --http.api='eth,web3,net,txpool,sbch,debug' \
#  --log_level='json-rpc:debug,watcher:debug,app:debug' \
#  --skip-sanity-check=true \
#  --with-syncdb=false
*/

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
	sideNodeParams := []string{
		"start",
		"--home", "/Users/bear/.smartbchd",
		"--unlock", "0xe3d9be2e6430a9db8291ab1853f5ec2467822b33a1a08825a22fab1425d2bff9",
		"--https.addr=off",
		"--wss.addr=off",
		"--http.api=eth,web3,net,txpool,sbch,debug",
		"--log_level=json-rpc:debug,watcher:debug,app:debug",
		"--skip-sanity-check=true",
		"--with-syncdb=false",
	}
	go utils.ExecuteWithContinuousOutPut(config.SideNodePath, sideNodeParams...)
	time.Sleep(10 * time.Second)
	fmt.Println("-------------- start test --------------")
	go testcase.Test()
	select {}
}
