package historydb

import (
	"context"
	"fmt"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// go test -c .
// HISTORYDBTEST=YES ./historydb.test -test.run TestStep0
// HISTORYDBTEST=YES ./historydb.test -test.run TestStep1
// HISTORYDBTEST=YES ./historydb.test -test.run TestStep2

func TestStep0(t *testing.T) {
	if os.Getenv("HISTORYDBTEST") != "YES" {
		return
	}
	testTheOnlyTxInBlocks("./modb", "http://127.0.0.1:8545", 100000)
}

func TestStep1(t *testing.T) {
	if os.Getenv("HISTORYDBTEST") != "YES" {
		return
	}
	generateHisDb("/mnt/nvme/smartbchd/data/modb", "./hisdb", 2650514)
}

func TestStep2(t *testing.T) {
	if os.Getenv("HISTORYDBTEST") != "YES" {
		return
	}
	runTestcases("./hisdb", "http://127.0.0.1:8545", 2650514)
}

func TestStep_getAccount(t *testing.T) {
	ethCli := getEthClient("http://127.0.0.1:8545")

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	addr := common.HexToAddress("0x0608037fd563fA0d159adb934FFc9035dF56FB88")
	nonce, err := ethCli.NonceAt(ctx, addr, nil)
	fmt.Printf("Nonce A %d %#v\n", nonce, err)
	nonce, err = ethCli.NonceAt(ctx, addr, big.NewInt(2650514))
	fmt.Printf("Nonce B %d %#v\n", nonce, err)
}
