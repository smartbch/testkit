package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingdb/modb"
	"github.com/smartbch/moeingevm/types"
	sbchrpc "github.com/smartbch/smartbch/rpc/api"
)

func testTheOnlyTxInBlocks(modbDir, rpcUrl string, endHeight uint64) {
	modb := modb.NewMoDB(modbDir, log.NewNopLogger())
	ctx := types.NewContext(nil, modb)
	sbchCli := newSbchClient(rpcUrl)
	for h := uint64(1); h < endHeight; h++ {
		blk, err := ctx.GetBlockByHeight(h)
		if err != nil {
			panic(err)
		}
		fmt.Println("block:", h, "txs:", len(blk.Transactions))
		if len(blk.Transactions) != 1 {
			continue
		}
		txHash := blk.Transactions[0]
		tx, _, err := ctx.GetTxByHash(txHash)
		if err != nil {
			panic(err)
		}
		testTheOnlyTx(tx, sbchCli, h)
	}
}

func testTheOnlyTx(tx *types.Transaction, sbchCli *SbchClient, height uint64) {
	to := common.Address(tx.To)
	toPtr := &to
	if to == [20]byte{} {
		toPtr = nil
	}

	h := big.NewInt(int64(height - 1))
	callMsg := ethereum.CallMsg{
		From:     tx.From,
		To:       toPtr,
		Gas:      tx.Gas,
		GasPrice: big.NewInt(0).SetBytes(tx.GasPrice[:]),
		Value:    big.NewInt(0).SetBytes(tx.Value[:]),
		Data:     tx.Input,
	}

	callDetail, err := sbchCli.sbchCall(callMsg, h)
	if err != nil {
		panic(err)
	}

	compareCallDetail(tx, callDetail)
}

func compareCallDetail(tx *types.Transaction, callDetail *sbchrpc.CallDetail) {
	callDetail1 := sbchrpc.TxToRpcCallDetail(tx)

	json1, err := json.Marshal(callDetail1)
	if err != nil {
		panic(err)
	}

	json2, err := json.Marshal(callDetail)
	if err != nil {
		panic(err)
	}

	if !bytes.Equal(json1, json2) {
		fmt.Println("----- json1 -----")
		fmt.Println(string(json1))
		fmt.Println("----- json2 -----")
		fmt.Println(string(json2))

		panic("callDetail not match!")
	}
}
