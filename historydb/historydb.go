package historydb

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"math"
	"math/big"
	"math/rand"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
	it "github.com/smartbch/moeingads/indextree"
	adstypes "github.com/smartbch/moeingads/types"
	"github.com/smartbch/moeingdb/modb"
	"github.com/tendermint/tendermint/libs/log"

	"github.com/smartbch/moeingevm/types"
)

const (
	CreationCounterByte = byte(100)
	AccountByte         = byte(102)
	BytecodeByte        = byte(104)
	StorageByte         = byte(106)

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
		if "0x0206422a8140C674203cB357D2807fC89ccC4B4C" == common.Address(op.Addr).String() {
			fmt.Printf("height %d 0206 %#v\n", height, op.Account)
		}
		copy(key[1:], op.Addr[:])
		copy(key[1+20:], db.currHeight[:])
		db.batch.Set(key[:], op.Account)
		if len(op.Account) == 0 {
			//delete contract account (EOA cannot be deleted)
			key[0] = BytecodeByte
			db.batch.Set(key[:], op.Account)
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
	defer iter.Close()
	if !iter.Valid() {
		return
	}
	currRec := getRecord(iter.Key(), iter.Value())
	for iter.Valid() {
		iter.Next()
		currRec.EndHeight = latestHeight
		key, value := iter.Key(), iter.Value()
		if (currRec.Key == "account" && key[0] == AccountByte) ||
			(currRec.Key == "bytecode" && key[0] == StorageByte) {
			if bytes.Equal(currRec.Addr[:], key[1:1+20]) {
				currRec.EndHeight = binary.BigEndian.Uint64(key[1+20:])
			}
		} else if len(currRec.Key) == 32 && key[0] == StorageByte {
			if bytes.Equal(currRec.Addr[:], key[1:1+20]) && currRec.Key == string(key[1+20:1+20+32]) {
				currRec.EndHeight = binary.BigEndian.Uint64(key[1+20+32:])
			}
		}
		recChan <- currRec
		currRec = getRecord(key, value)
	}
	recChan <- currRec
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
	for rec := range recChan {
		fmt.Printf("StartEnd %d %d\n", rec.StartHeight, rec.EndHeight)
		mid := rand.Intn(int(rec.EndHeight)-int(rec.StartHeight)) + int(rec.StartHeight)
		if rec.Key == "account" {
			runAccountTestcase(rec, ethCli, rec.StartHeight)
			//runAccountTestcase(rec, ethCli, uint64(mid))
			//runAccountTestcase(rec, ethCli, rec.EndHeight-1)
		} else if rec.Key == "bytecode" {
			runBytecodeTestcase(rec, ethCli, rec.StartHeight)
			runBytecodeTestcase(rec, ethCli, uint64(mid))
			runBytecodeTestcase(rec, ethCli, rec.EndHeight-1)
		} else if len(rec.Key) == 32 {
			runStorageTestcase(rec, ethCli, rec.StartHeight)
			runStorageTestcase(rec, ethCli, uint64(mid))
			runStorageTestcase(rec, ethCli, rec.EndHeight-1)
		} else {
			panic("Invalid rec.Key")
		}
	}
}

func runAccountTestcase(rec HistoricalRecord, ethCli *ethclient.Client, height uint64) {
	h := big.NewInt(int64(height))
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	nonce, err := ethCli.NonceAt(ctx, common.Address(rec.Addr), h)
	balance, err := ethCli.BalanceAt(ctx, common.Address(rec.Addr), h)
	if len(rec.Value) == 0 && nonce == 0 && balance.IsInt64() && balance.Int64() == 0 { //deleted contract account
		return
	}
	//fmt.Printf("Here %#v %d\n", rec.Addr, len(rec.Value))
	accInfo := types.NewAccountInfo(rec.Value)
	if err != nil {
		panic(err)
	}
	if accInfo.Nonce() != nonce {
		fmt.Printf("height %d acc %s\n", height, common.Address(rec.Addr))
		fmt.Printf("nonce ref %d imp %d\n", accInfo.Nonce(), nonce)
		h = big.NewInt(int64(height - 1))
		nonce, _ = ethCli.NonceAt(ctx, common.Address(rec.Addr), h)
		fmt.Printf("nonce imp+h-1 %d\n", nonce)

		h = big.NewInt(int64(height + 1))
		nonce, _ = ethCli.NonceAt(ctx, common.Address(rec.Addr), h)
		fmt.Printf("nonce imp+h+1 %d\n", nonce)
	} else {
		fmt.Printf("nonce SAME! h %d nonce %d %s\n", height, nonce, common.Address(rec.Addr))
	}
	//balance, err := ethCli.BalanceAt(ctx, common.Address(rec.Addr), h)
	//if err != nil {
	//	panic(err)
	//}
	//if accInfo.Balance().ToBig().Cmp(balance) != 0 {
	//	fmt.Printf("account %d acc %s\n", height, common.Address(rec.Addr))
	//	fmt.Printf("balance ref %s imp %s\n", accInfo.Balance(), balance)
	//	balance, err = ethCli.BalanceAt(ctx, common.Address(rec.Addr), nil)
	//	fmt.Printf("balance+latest imp %s\n", balance)
	//	h := big.NewInt(int64(height+1))
	//	balance, err = ethCli.BalanceAt(ctx, common.Address(rec.Addr), h)
	//	fmt.Printf("BALANCE+1 imp %s\n", balance)
	//}
}

func runBytecodeTestcase(rec HistoricalRecord, ethCli *ethclient.Client, height uint64) {
	h := big.NewInt(int64(height))
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	bytecode, err := ethCli.CodeAt(ctx, common.Address(rec.Addr), h)
	if len(rec.Value) == 0 && err != nil { //deleted contract code
		return
	}
	if err != nil {
		panic(err)
	}
	bcInfo := types.NewBytecodeInfo(rec.Value)
	if !bytes.Equal(bcInfo.BytecodeSlice(), bytecode) {
		fmt.Printf("bytecode %d acc %s\n", height, common.Address(rec.Addr))
		fmt.Printf("ref %#v\n", bcInfo.BytecodeSlice())
		fmt.Printf("imp %#v\n", bytecode)
	}
}

func runStorageTestcase(rec HistoricalRecord, ethCli *ethclient.Client, height uint64) {
	h := big.NewInt(int64(height))
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	var key common.Hash
	copy(key[:], rec.Key)
	value, err := ethCli.StorageAt(ctx, common.Address(rec.Addr), key, h)
	if len(rec.Value) == 0 && err != nil { //deleted stoarge
		return
	}
	if err != nil {
		panic(err)
	}
	if !bytes.Equal(rec.Value, value) {
		fmt.Printf("storage %d acc %s\n", height, common.Address(rec.Addr))
		fmt.Printf("ref %#v\n", rec.Value)
		fmt.Printf("imp %#v\n", value)
	}
}

func testTheOnlyTxInBlocks(modbDir, rpcUrl string, endHeight uint64) {
	modb := modb.NewMoDB(modbDir, log.NewNopLogger())
	ctx := types.NewContext(nil, modb)
	ethCli := getEthClient(rpcUrl)
	for h := uint64(0); h < endHeight; h++ {
		blk, err := ctx.GetBlockByHeight(h)
		if err != nil {
			panic(err)
		}
		if len(blk.Transactions) != 1 {
			continue
		}
		txHash := blk.Transactions[0]
		tx, _, err := ctx.GetTxByHash(txHash)
		if err != nil {
			panic(err)
		}
		testTheOnlyTx(tx, ethCli)
	}
}

func testTheOnlyTx(tx *types.Transaction, ethCli *ethclient.Client) {
	// TODO: check sbch_call using tx.RwLists
}
