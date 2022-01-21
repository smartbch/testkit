package main

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/tendermint/tendermint/libs/log"

	it "github.com/smartbch/moeingads/indextree"
	"github.com/smartbch/moeingdb/modb"
	moevmtypes "github.com/smartbch/moeingevm/types"
	sbchrpc "github.com/smartbch/smartbch/rpc/api"
)

type OneTxDb struct {
	rocksdb *it.RocksDB
}

func newOneTxDb(dirname string) *OneTxDb {
	rocksdb, err := it.NewRocksDB(dirname, ".")
	if err != nil {
		panic(err)
	}
	return &OneTxDb{rocksdb: rocksdb}
}

func (db *OneTxDb) addTx(height uint64, tx *moevmtypes.Transaction) {
	key := make([]byte, 8)
	binary.BigEndian.PutUint64(key, height)
	val, err := tx.MarshalMsg(nil)
	if err != nil {
		panic(err)
	}
	db.rocksdb.Set(key, val)
}

func (db *OneTxDb) getAllTxs(startHeight uint64, cb func(height uint64, tx *moevmtypes.Transaction)) {
	end := []byte{255, 255, 255, 255, 255, 255, 255, 255}
	start := make([]byte, 8)
	binary.BigEndian.PutUint64(start, startHeight)

	iter := db.rocksdb.Iterator(start, end)
	defer iter.Close()

	for ; iter.Valid(); iter.Next() {
		key, val := iter.Key(), iter.Value()
		h := binary.BigEndian.Uint64(key)
		tx := &moevmtypes.Transaction{}
		_, err := tx.UnmarshalMsg(val)
		if err != nil {
			panic(err)
		}
		cb(h, tx)
	}
}

func generateOneTxDb(modbDir, oneTxDbDir string, endHeight uint64) {
	oneTxDb := newOneTxDb(oneTxDbDir)
	n := 0
	cb := func(height uint64, tx *moevmtypes.Transaction) {
		oneTxDb.addTx(height, tx)
		n++
	}
	scanTheOnlyTxInBlocks(modbDir, endHeight, cb)
	fmt.Println("oneTx count:", n)
}

func testTxsInOneTxDb(oneTxDbDir, rpcUrl string, startHeight uint64, stopOnErr bool) {
	sbchCli := newSbchClient(rpcUrl)
	oneTxDb := newOneTxDb(oneTxDbDir)

	cb := func(height uint64, tx *moevmtypes.Transaction) {
		// if len(tx.Input) == 0 { // skip ether transfers
		// 	return
		// }

		fmt.Print("height: ", height, " tx: 0x", hex.EncodeToString(tx.Hash[:]))
		if testTheOnlyTx(tx, sbchCli, height, stopOnErr) {
			fmt.Println(" OK")
		} else {
			fmt.Println(" FAIL")
			if stopOnErr {
				panic("callDetails not match!")
			}
		}
	}
	oneTxDb.getAllTxs(startHeight, cb)
}

func testTheOnlyTxInBlocks(modbDir, rpcUrl string, endHeight uint64) {
	sbchCli := newSbchClient(rpcUrl)
	cb := func(height uint64, tx *moevmtypes.Transaction) {
		if !testTheOnlyTx(tx, sbchCli, height, true) {
			panic("callDetails not match!")
		}
	}
	scanTheOnlyTxInBlocks(modbDir, endHeight, cb)
}

func scanTheOnlyTxInBlocks(modbDir string, endHeight uint64, cb func(height uint64, tx *moevmtypes.Transaction)) {
	_modb := modb.NewMoDB(modbDir, log.NewNopLogger())
	ctx := moevmtypes.NewContext(nil, _modb)
	for h := uint64(1); h < endHeight; h++ {
		blk, err := ctx.GetBlockByHeight(h)
		if err != nil {
			//panic(err)
			fmt.Println(err.Error())
			continue
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
		cb(h, tx)
	}
}

func testTheOnlyTx(tx *moevmtypes.Transaction, sbchCli *SbchClient, height uint64, printsDetail bool) bool {
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

	return compareCallDetail(tx, callDetail, printsDetail)
}

func compareCallDetail(tx *moevmtypes.Transaction, rpcCallDetail *sbchrpc.CallDetail, printsDetail bool) bool {
	txCallDetail := sbchrpc.TxToRpcCallDetail(tx)

	json1, err := json.Marshal(txCallDetail)
	if err != nil {
		panic(err)
	}

	json2, err := json.Marshal(rpcCallDetail)
	if err != nil {
		panic(err)
	}

	if bytes.Equal(json1, json2) {
		return true
	}

	if printsDetail {
		fmt.Println("\n----- txCallDetail  -----")
		fmt.Println(string(json1))
		fmt.Println("\n----- rpcCallDetail -----")
		fmt.Println(string(json2))
	}
	return false
}
