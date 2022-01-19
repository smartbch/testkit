package main

import (
	"encoding/hex"
	"fmt"
	"os"

	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/smartbch/watcher"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Printf("Usage: %s <rpc-url> <username> <password>\n", os.Args[0])
		return
	}

	rpcURL := os.Args[1]
	username := os.Args[2]
	password := os.Args[3]

	client := watcher.NewRpcClient(rpcURL, username, password, "text/plain;",
		log.NewNopLogger())
	h := client.GetLatestHeight()
	fmt.Println("latest height:", h)

	for ; h > 0; h-- {
		bHash, err := client.GetBlockHash(h)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		bInfo, err := client.GetBlockInfo(bHash)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		cbTx, err := client.GetTxInfo(bInfo.Tx[0].Hash, bHash)
		if err != nil {
			fmt.Println(err.Error())
			continue
		}

		pubKey, ok := cbTx.GetValidatorPubKey()
		if ok {
			fmt.Printf("block: %d, vote for pubkey: %s\n",
				h, hex.EncodeToString(pubKey[:]))
		} else {
			fmt.Printf("block: %d, no votes\n", h)
		}
	}
}
