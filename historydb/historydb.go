package main

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/tendermint/tendermint/libs/log"

	it "github.com/smartbch/moeingads/indextree"
	adstypes "github.com/smartbch/moeingads/types"
	"github.com/smartbch/moeingdb/modb"
	"github.com/smartbch/moeingevm/types"
	sbchrpc "github.com/smartbch/smartbch/rpc/api"
)

const (
	CreationCounterByte = byte(100)
	AccountByte         = byte(102)
	BytecodeByte        = byte(104)
	StorageByte         = byte(106)
	DelHeightByte       = byte(108)

	Timeout = time.Second * 15
)

// You can use HistoricalRecord to send requests to RPC and check the result
// When key is 32-byte long, Value is StorageSlot
// When key is "account", Value is AccountInfo bytes
// When key is "bytecode", Value is BytecodeInfo bytes
type HistoricalRecord struct {
	Addr        [20]byte
	Key         string
	Value       []byte
	StartHeight uint64 //when this record was created
	EndHeight   uint64 //when this record was overwritten
}

type HistoryDb struct {
	rocksdb    *it.RocksDB
	batch      adstypes.Batch
	currHeight [8]byte
}

func NewHisDb(dirname string) *HistoryDb {
	db := &HistoryDb{}
	var err error
	db.rocksdb, err = it.NewRocksDB(dirname, ".")
	if err != nil {
		panic(err)
	}
	return db
}

func (db *HistoryDb) Close() {
	db.rocksdb.Close()
}

func (db *HistoryDb) BeginWrite(height uint64) {
	db.batch = db.rocksdb.NewBatch()
	binary.BigEndian.PutUint64(db.currHeight[:], height)
}

func (db *HistoryDb) EndWrite() {
	db.batch.WriteSync()
	db.batch.Close()
	db.batch = nil
}

func (db *HistoryDb) AddRwLists0(height uint64, rwLists *types.ReadWriteLists) {
	for _, op := range rwLists.AccountWList {
		if len(op.Account) != 49 {
			fmt.Printf("Invalid account len %d %#v addr %#v\n", len(op.Account), op.Account, op.Addr)
			return
		}
		accInfo := types.NewAccountInfo(op.Account)
		seq := accInfo.Sequence()
		if seq == math.MaxUint64 {
			continue
		}
		fmt.Printf("seq %d %#v\n", seq, op.Addr)
		//seq2Addr[accInfo.Sequence()] = op.Addr
	}
}

func (db *HistoryDb) AddRwLists(height uint64, rwLists *types.ReadWriteLists) {
	db.BeginWrite(height)
	seq2Addr := make(map[uint64][20]byte, len(rwLists.AccountRList)+len(rwLists.AccountWList))
	seq2Addr[2000] = [20]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x27, 0x11}
	for _, op := range rwLists.AccountRList {
		accInfo := types.NewAccountInfo(op.Account)
		seq2Addr[accInfo.Sequence()] = op.Addr
	}
	for _, op := range rwLists.AccountWList {
		if len(op.Account) == 0 {
			continue //deletion
		}
		accInfo := types.NewAccountInfo(op.Account)
		seq2Addr[accInfo.Sequence()] = op.Addr
	}
	for _, op := range rwLists.CreationCounterWList {
		var key [1 + 1 + 8]byte
		key[0] = CreationCounterByte
		key[1] = op.Lsb
		copy(key[2:], db.currHeight[:])
		var value [8]byte
		binary.LittleEndian.PutUint64(value[:], op.Counter)
		db.batch.Set(key[:], value[:])
	}
	for _, op := range rwLists.AccountWList {
		var key [1 + 20 + 8]byte
		key[0] = AccountByte
		//focus := common.Address(op.Addr).String()
		//if("0x5D0171c4AB2745412B148aF5C803C62605b19cD6" == focus ||
		//	"0x42A02Ab30C79247D96689C3776aA2faCC1F19dc3" == focus ) {
		//	fmt.Printf("focus %s height %d value %#v\n", focus, height, op.Account)
		//}
		copy(key[1:], op.Addr[:])
		copy(key[1+20:], db.currHeight[:])
		db.batch.Set(key[:], op.Account)
		if len(op.Account) == 0 {
			//delete contract account (EOA cannot be deleted)
			key[0] = BytecodeByte // bytecode is also deleted
			db.batch.Set(key[:], op.Account)
			key[0] = DelHeightByte //record the deletion height
			db.batch.Set(key[:1+20], db.currHeight[:])
		}
	}
	for _, op := range rwLists.BytecodeWList {
		var key [1 + 20 + 8]byte
		key[0] = BytecodeByte
		copy(key[1:], op.Addr[:])
		copy(key[1+20:], db.currHeight[:])
		db.batch.Set(key[:], op.Bytecode)
	}
	for _, op := range rwLists.StorageWList {
		var key [1 + 20 + 32 + 8]byte
		key[0] = StorageByte
		addr, ok := seq2Addr[op.Seq]
		if !ok {
			panic("Cannot find seq's addr")
		}
		copy(key[1:], addr[:])
		if len(op.Key) != 32 {
			panic("Invalid Key Length")
		}
		//focus := common.Address(addr).String()
		//if("0x5D0171c4AB2745412B148aF5C803C62605b19cD6" == focus ||
		//	"0x42A02Ab30C79247D96689C3776aA2faCC1F19dc3" == focus ) {
		//	fmt.Printf("FOCUS %s height %d key %#v value %#v\n", focus, height, op.Key, op.Value)
		//}
		copy(key[1+20:], op.Key)
		copy(key[1+20+32:], db.currHeight[:])
		db.batch.Set(key[:], op.Value)
	}
	db.EndWrite()
}

func (db *HistoryDb) AddRwListAtHeight(ctx *types.Context, height uint64) {
	blk, err := ctx.GetBlockByHeight(height)
	if err != nil {
		panic(err)
	}
	for _, txHash := range blk.Transactions {
		tx, _, err := ctx.GetTxByHash(txHash)
		if err != nil {
			fmt.Printf("txHash %#v\n", txHash)
			panic(err)
		}
		if tx.StatusStr == "incorrect nonce" {
			//fmt.Printf("%#v\n", tx)
			continue
		}
		db.AddRwLists(height, tx.RwLists)
	}
}

func (db *HistoryDb) Fill(ctx *types.Context, endHeight uint64) {
	for h := uint64(1); h < endHeight; h++ {
		if h%10000 == 0 {
			fmt.Printf("Height %d\n", h)
		}
		db.AddRwListAtHeight(ctx, h)
	}
}

func getRecord(key, value []byte) (rec HistoricalRecord) {
	if key[0] == AccountByte {
		copy(rec.Addr[:], key[1:])
		rec.Key = "account"
		rec.Value = append([]byte{}, value...)
		rec.StartHeight = binary.BigEndian.Uint64(key[1+20:])
		//fmt.Printf("account StartHeight %d\n", rec.StartHeight)
	} else if key[0] == BytecodeByte {
		copy(rec.Addr[:], key[1:])
		rec.Key = "bytecode"
		rec.Value = append([]byte{}, value...)
		rec.StartHeight = binary.BigEndian.Uint64(key[1+20:])
	} else if key[0] == StorageByte {
		copy(rec.Addr[:], key[1:])
		rec.Key = string(key[1+20 : 1+20+32])
		rec.Value = append([]byte{}, value...)
		rec.StartHeight = binary.BigEndian.Uint64(key[1+20+32:])
	} else {
		panic("invalid key[0]")
	}
	return
}

func (db *HistoryDb) GenerateRecords(recChan chan HistoricalRecord, latestHeight uint64) {
	iter := db.rocksdb.Iterator([]byte{AccountByte}, []byte{StorageByte + 1})
	//iter := db.rocksdb.Iterator([]byte{AccountByte}, []byte{AccountByte + 1})
	//iter := db.rocksdb.Iterator([]byte{BytecodeByte}, []byte{BytecodeByte + 1})
	//iter := db.rocksdb.Iterator([]byte{StorageByte, 0x42, 0xA0}, []byte{StorageByte + 1})
	defer iter.Close()
	if !iter.Valid() {
		close(recChan)
		return
	}
	currRec := getRecord(iter.Key(), iter.Value())
	currRec.EndHeight = latestHeight
	for iter.Next(); iter.Valid(); iter.Next() {
		key, value := iter.Key(), iter.Value()
		if (currRec.Key == "account" && key[0] == AccountByte) ||
			(currRec.Key == "bytecode" && key[0] == BytecodeByte) {
			if bytes.Equal(currRec.Addr[:], key[1:1+20]) {
				currRec.EndHeight = binary.BigEndian.Uint64(key[1+20:])
			}
		} else if len(currRec.Key) == 32 && key[0] == StorageByte {
			if bytes.Equal(currRec.Addr[:], key[1:1+20]) && currRec.Key == string(key[1+20:1+20+32]) {
				currRec.EndHeight = binary.BigEndian.Uint64(key[1+20+32:])
			} else {
				heightBz := db.rocksdb.Get(append([]byte{DelHeightByte}, currRec.Addr[:]...))
				if len(heightBz) != 0 {
					currRec.EndHeight = binary.BigEndian.Uint64(heightBz)
				}
			}
		}
		recChan <- currRec
		currRec = getRecord(key, value)
		currRec.EndHeight = latestHeight
	}
	recChan <- currRec
	close(recChan)
}

// -------------------------------------------------------------------------------

func generateHisDb(modbDir, hisdbDir string, endHeight uint64) {
	modb := modb.NewMoDB(modbDir, log.NewNopLogger())
	ctx := types.NewContext(nil, modb)
	hisDb := NewHisDb(hisdbDir)
	hisDb.Fill(ctx, endHeight)
	hisDb.Close()
}

func getEthClient(rpcUrl string) *ethclient.Client {
	rpcCli, err := rpc.DialContext(context.Background(), rpcUrl)
	if err != nil {
		panic(err)
	}
	return ethclient.NewClient(rpcCli)
}

func runTestcases(hisdbDir, rpcUrl string, latestHeight uint64) {
	ethCli := getEthClient(rpcUrl)
	recChan := make(chan HistoricalRecord, 100)
	hisDb := NewHisDb(hisdbDir)
	go hisDb.GenerateRecords(recChan, latestHeight)
	count := 0
	for rec := range recChan {
		if count%100000 == 0 {
			fmt.Printf("========== %d ============\n", count)
		}
		if rec.EndHeight == rec.StartHeight {
			continue
		}
		if int(rec.EndHeight) <= int(rec.StartHeight) {
			fmt.Printf("StartEnd %d %d\n", rec.StartHeight, rec.EndHeight)
		}
		mid := rand.Intn(int(rec.EndHeight)-int(rec.StartHeight)) + int(rec.StartHeight)
		if rec.Key == "account" {
			runNonceTestcase(rec, ethCli, rec.StartHeight, rec.StartHeight, rec.EndHeight)
			if rec.StartHeight+1 < rec.EndHeight {
				//We are not aware of balance-change caused by Prepare
				//So rec.StartHeight+1 == rec.EndHeight cannot be tested
				runAccountTestcase(rec, ethCli, rec.StartHeight, rec.StartHeight, rec.EndHeight)
			}
			if rec.StartHeight+4 <= rec.EndHeight { // must avoid rec.EndHeight-1
				mid = rand.Intn(int(rec.EndHeight)-int(rec.StartHeight)-3) + int(rec.StartHeight) + 1
				runNonceTestcase(rec, ethCli, uint64(mid), rec.StartHeight, rec.EndHeight)
				runAccountTestcase(rec, ethCli, uint64(mid), rec.StartHeight, rec.EndHeight)
			}
			runNonceTestcase(rec, ethCli, rec.EndHeight-1, rec.StartHeight, rec.EndHeight)
			//We are not aware of balance-change caused by Prepare
			//So balance at EndHeight cannot be tested
		} else if rec.Key == "bytecode" {
			runBytecodeTestcase(rec, ethCli, rec.StartHeight, rec.StartHeight, rec.EndHeight)
			runBytecodeTestcase(rec, ethCli, uint64(mid), rec.StartHeight, rec.EndHeight)
			runBytecodeTestcase(rec, ethCli, rec.EndHeight-1, rec.StartHeight, rec.EndHeight)
		} else if len(rec.Key) == 32 {
			runStorageTestcase(rec, ethCli, rec.StartHeight, rec.StartHeight, rec.EndHeight)
			runStorageTestcase(rec, ethCli, uint64(mid), rec.StartHeight, rec.EndHeight)
			runStorageTestcase(rec, ethCli, rec.EndHeight-1, rec.StartHeight, rec.EndHeight)
		} else {
			panic("Invalid rec.Key")
		}
		count++
	}
}

func runAccountTestcase(rec HistoricalRecord, ethCli *ethclient.Client, height, s, e uint64) {
	h := big.NewInt(int64(height))
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	balance, err := ethCli.BalanceAt(ctx, common.Address(rec.Addr), h)
	if len(rec.Value) == 0 && balance.IsInt64() && balance.Int64() == 0 { //deleted contract account
		return
	}
	//fmt.Printf("Here %#v %d\n", rec.Addr, len(rec.Value))
	accInfo := types.NewAccountInfo(rec.Value)
	if err != nil {
		panic(err)
	}
	if accInfo.Balance().ToBig().Cmp(balance) != 0 {
		fmt.Printf("SE %d %d\n", s, e)
		fmt.Printf("account %d acc %s\n", height, common.Address(rec.Addr))
		fmt.Printf("balance ref %s imp %s\n", accInfo.Balance(), balance)
		balance, err = ethCli.BalanceAt(ctx, common.Address(rec.Addr), nil)
		fmt.Printf("balance+latest imp %s\n", balance)
		h := big.NewInt(int64(height + 1))
		balance, err = ethCli.BalanceAt(ctx, common.Address(rec.Addr), h)
		fmt.Printf("BALANCE+1 imp %s\n", balance)
	}
}
func runNonceTestcase(rec HistoricalRecord, ethCli *ethclient.Client, height, s, e uint64) {
	h := big.NewInt(int64(height))
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	nonce, err := ethCli.NonceAt(ctx, common.Address(rec.Addr), h)
	if len(rec.Value) == 0 && nonce == 0 { //deleted contract account
		return
	}
	//fmt.Printf("Here %#v %d\n", rec.Addr, len(rec.Value))
	accInfo := types.NewAccountInfo(rec.Value)
	if err != nil {
		panic(err)
	}
	if accInfo.Nonce() != nonce {
		fmt.Printf("SE %d %d\n", s, e)
		fmt.Printf("height %d acc %s\n", height, common.Address(rec.Addr))
		fmt.Printf("nonce ref %d imp %d\n", accInfo.Nonce(), nonce)
		h = big.NewInt(int64(height - 1))
		nonce, _ = ethCli.NonceAt(ctx, common.Address(rec.Addr), h)
		fmt.Printf("nonce imp+h-1 %d\n", nonce)

		h = big.NewInt(int64(height + 1))
		nonce, _ = ethCli.NonceAt(ctx, common.Address(rec.Addr), h)
		fmt.Printf("nonce imp+h+1 %d\n", nonce)
	} else {
		if "0x498F4F6cb582B9839e3dA48E18734286FBFFa7e0" == common.Address(rec.Addr).String() ||
			"0x79f3F9F9c0E860341e5A20C6801Ad28fb1FBD924" == common.Address(rec.Addr).String() {
			fmt.Printf("nonce SAME! h %d nonce %d %s\n", height, nonce, common.Address(rec.Addr))
		}
	}
}

func runBytecodeTestcase(rec HistoricalRecord, ethCli *ethclient.Client, height, s, e uint64) {
	h := big.NewInt(int64(height))
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	//focus := common.Address(rec.Addr).String()
	//if("0x5D0171c4AB2745412B148aF5C803C62605b19cD6" == focus ||
	//	"0x42A02Ab30C79247D96689C3776aA2faCC1F19dc3" == focus ) {
	//	fmt.Printf("SE %d %d\n", s, e)
	//	fmt.Printf("Focus height %d acc %s value %#v\n", height, common.Address(rec.Addr), rec.Value)
	//}
	bytecode, err := ethCli.CodeAt(ctx, common.Address(rec.Addr), h)
	if len(rec.Value) == 0 && len(bytecode) == 0 { //deleted contract code
		return
	}
	if err != nil {
		panic(err)
	}
	bcInfo := types.NewBytecodeInfo(rec.Value)
	if !bytes.Equal(bcInfo.BytecodeSlice(), bytecode) {
		fmt.Printf("SE %d %d\n", s, e)
		fmt.Printf("bytecode %d acc %s\n", height, common.Address(rec.Addr))
		fmt.Printf("ref %#v\n", bcInfo.BytecodeSlice())
		fmt.Printf("imp %#v\n", bytecode)
	}
}

func runStorageTestcase(rec HistoricalRecord, ethCli *ethclient.Client, height, s, e uint64) {
	if rec.Addr == [20]byte{0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x0, 0x27, 0x10} { // The storage of staking contract cannot be correctly tested
		return
	}
	h := big.NewInt(int64(height))
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	//focus := common.Address(rec.Addr).String()
	//if("0x5D0171c4AB2745412B148aF5C803C62605b19cD6" == focus ||
	//	"0x42A02Ab30C79247D96689C3776aA2faCC1F19dc3" == focus ) {
	//	fmt.Printf("SE %d %d\n", s, e)
	//	fmt.Printf("Focus height %d acc %s key %#v value %#v\n", height, common.Address(rec.Addr), rec.Key, rec.Value)
	//}

	var key common.Hash
	copy(key[:], rec.Key)
	value, err := ethCli.StorageAt(ctx, common.Address(rec.Addr), key, h)
	var zero32 [32]byte
	if len(rec.Value) == 0 && bytes.Equal(value, zero32[:]) { //deleted stoarge
		return
	}
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(rec.Value, value) {
		fmt.Printf("SE %d %d\n", s, e)
		fmt.Printf("storage %d acc %s %#v\n", height, common.Address(rec.Addr), rec.Key)
		fmt.Printf("ref %#v\n", rec.Value)
		fmt.Printf("imp %#v\n", value)
	}
}

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
	// TODO: check sbch_call using tx.RwLists
	to := common.Address(tx.To)
	toPtr := &to
	if to == [20]byte{} {
		toPtr = nil
	}

	h := big.NewInt(int64(height))
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
