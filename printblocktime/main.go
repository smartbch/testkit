package main

import (
	"fmt"
	"os"
	"time"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/staking"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("Usage: %s <rpc-url> <username> <password>\n", os.Args[0])
		return
	}

	rpcURL := os.Args[1]
	username := os.Args[2]
	password := os.Args[3]

	client := staking.NewRpcClient(rpcURL, username, password, "text/plain;",
		log.NewNopLogger())
	h := client.GetLatestHeight()
	fmt.Println("latest height:", h)

	for ; h > 0; h-- {
		b := client.GetBlockByHeight(h)
		fmt.Println("block:", h, "time:", time.Unix(b.Timestamp, 0).String())
	}
}
