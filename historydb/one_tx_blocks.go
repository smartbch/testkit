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

func (db *OneTxDb) getAllTxs(cb func(height uint64, tx *moevmtypes.Transaction)) {
	iter := db.rocksdb.Iterator([]byte{0}, []byte{255})
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

func testTxsInOneTxDb(oneTxDbDir, rpcUrl string, endHeight uint64) {
	sbchCli := newSbchClient(rpcUrl)
	oneTxDb := newOneTxDb(oneTxDbDir)

	cb := func(height uint64, tx *moevmtypes.Transaction) {
		fmt.Println("height:", height, "tx:", hex.EncodeToString(tx.Hash[:]))
		testTheOnlyTx(tx, sbchCli, height)
	}
	oneTxDb.getAllTxs(cb)
}

func testTheOnlyTxInBlocks(modbDir, rpcUrl string, endHeight uint64) {
	sbchCli := newSbchClient(rpcUrl)
	cb := func(height uint64, tx *moevmtypes.Transaction) {
		testTheOnlyTx(tx, sbchCli, height)
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

func testTheOnlyTx(tx *moevmtypes.Transaction, sbchCli *SbchClient, height uint64) {
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

func compareCallDetail(tx *moevmtypes.Transaction, callDetail *sbchrpc.CallDetail) {
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
