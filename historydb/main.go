package main

import (
	"fmt"
	"os"
	"strconv"
)

const usage = `Usage:
  historydb generateOneTxDb <modbDir> <oneTxDbDir> <endHeight>
  historydb testTxsInOneTxDb <modbDir> <rpcUrl> <endHeight>
  historydb testTheOnlyTxInBlocks <oneTxDbDir> <rpcUrl> <endHeight>
  historydb generateHisDb <modbDir> <hisdbDir> <endHeight>
  historydb runTestcases <hisdbDir> <rpcUrl> <latestHeight>`

func main() {
	if len(os.Args) != 5 {
		fmt.Println(usage)
		return
	}

	switch os.Args[1] {
	case "generateOneTxDb":
		generateOneTxDb(os.Args[2], os.Args[3], parseUint64(os.Args[4]))
	case "testTxsInOneTxDb":
		testTxsInOneTxDb(os.Args[2], os.Args[3], parseUint64(os.Args[4]))
	case "testTheOnlyTxInBlocks":
		testTheOnlyTxInBlocks(os.Args[2], os.Args[3], parseUint64(os.Args[4]))
	case "generateHisDb":
		generateHisDb(os.Args[2], os.Args[3], parseUint64(os.Args[4]))
	case "runTestcases":
		runTestcases(os.Args[2], os.Args[3], parseUint64(os.Args[4]))
	default:
		fmt.Println(usage)
	}
}

func parseUint64(s string) uint64 {
	n, err := strconv.ParseUint(s, 10, 64)
	if err != nil {
		panic(err)
	}
	return n
}
