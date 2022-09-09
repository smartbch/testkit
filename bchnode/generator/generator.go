package generator

import (
	"bufio"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"io"
	"log"
	"os"
	"os/signal"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

var Validator = "00"
var Monitor = "01"
var Identifier = "73424348"
var ShaGateAddress = "14f8c7e99fd4e867c34cbd5968e35575fd5919a4"

type Context struct {
	RWLock sync.RWMutex

	Log      *log.Logger
	BlockLog *log.Logger

	Producer *Producer

	TxByHash           map[string]*TxInfo
	BlkByHash          map[string]*BlockInfo
	BlkHashByHeight    map[int64]string
	PubkeyInfoByPubkey map[string]*PubKeyInfo
	NextBlockHeight    int64

	//internal
	PubKeyInfoSet   []PubKeyInfo
	PubkeyInfoIndex int
	MonitorPubkey   string
}

func Init() *Context {
	var ctx Context
	ctx.NextBlockHeight = 1
	ctx.TxByHash = make(map[string]*TxInfo)
	ctx.BlkByHash = make(map[string]*BlockInfo)
	ctx.BlkHashByHeight = make(map[int64]string)
	ctx.PubkeyInfoByPubkey = make(map[string]*PubKeyInfo)

	//inti logger
	file, err := os.OpenFile("out.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	ctx.Log = log.New(io.MultiWriter(file, os.Stdout), "INFO: ", log.Ltime|log.Lshortfile)

	blockFile, err := os.OpenFile("block.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	ctx.BlockLog = log.New(blockFile, "", log.Ltime|log.Lshortfile)

	//init producer
	ctx.Producer = &Producer{
		ExitChan:          make(chan bool),
		ReorgChan:         make(chan bool, 1),
		MonitorPubkeyChan: make(chan string, 1),
		BlockIntervalTime: 3,
	}
	ctx.loadBlocksFromLog()
	ctx.logPubKeysOnExit()
	go ctx.Producer.Start(&ctx)
	return &ctx
}

type JsonRpcError struct {
	Code    int `json:"code"`
	Message int `json:"messsage"`
}

type BlockCountResp struct {
	Result int64         `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

type BlockHashResp struct {
	Result string        `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

type BlockInfo struct {
	Hash              string   `json:"hash"`
	Confirmations     int      `json:"confirmations"`
	Size              int      `json:"size"`
	Height            int64    `json:"height"`
	Version           int      `json:"version"`
	VersionHex        string   `json:"versionHex"`
	Merkleroot        string   `json:"merkleroot"`
	Tx                []string `json:"tx"`
	Time              int64    `json:"time"`
	MedianTime        int64    `json:"mediantime"`
	Nonce             int      `json:"nonce"`
	Bits              string   `json:"bits"`
	Difficulty        float64  `json:"difficulty"`
	Chainwork         string   `json:"chainwork"`
	NumTx             int      `json:"nTx"`
	PreviousBlockhash string   `json:"previousblockhash"`
}

type BlockInfoResp struct {
	Result BlockInfo     `json:"result"`
	Error  *JsonRpcError `json:"error"`
	Id     string        `json:"id"`
}

type CoinbaseVin struct {
	Coinbase string `json:"coinbase"`
	Sequence int    `json:"sequence"`
}

type Vout struct {
	Value        int64                  `json:"value"`
	N            int                    `json:"n"`
	ScriptPubKey map[string]interface{} `json:"scriptPubKey"`
}

type TxInfo struct {
	TxID          string                   `json:"txid"`
	Hash          string                   `json:"hash"`
	Version       int                      `json:"version"`
	Size          int                      `json:"size"`
	Locktime      int                      `json:"locktime"`
	VinList       []map[string]interface{} `json:"vin"`
	VoutList      []Vout                   `json:"vout"`
	Hex           string                   `json:"hex"`
	Blockhash     string                   `json:"blockhash"`
	Confirmations int                      `json:"confirmations"`
	Time          int64                    `json:"time"`
	BlockTime     int64                    `json:"blocktime"`
}

var reorgBlockNumbers int64 = 8

func (ctx *Context) ReorgBlock() {
	h := ctx.NextBlockHeight
	initHeight := h - reorgBlockNumbers
	if initHeight <= 1 {
		return
	}
	ctx.RWLock.Lock()
	for i := int64(0); i < reorgBlockNumbers; i++ {
		bi := &BlockInfo{
			Hash:          buildBlockHash(initHeight),
			Confirmations: 1,      //1 confirm
			Size:          100000, //100k
			Height:        initHeight,
			Version:       8888, //for test
			Time:          time.Now().Unix(),
			NumTx:         1,
		}
		bi.Tx = append(bi.Tx, buildTxHash(bi.Hash, 0))
		ti := ctx.BuildTxWithPubkey(0, bi.Hash, "reorg_tx")
		//change ctx
		if bi.Height > 1 {
			bi.PreviousBlockhash = ctx.BlkByHash[ctx.BlkHashByHeight[bi.Height-1]].Hash
		}
		ctx.BlkByHash[bi.Hash] = bi
		ctx.BlkHashByHeight[initHeight] = bi.Hash
		ctx.TxByHash[ti.Hash] = ti
		initHeight++

		ctx.Log.Printf("ReorgChan: new block: %d, %s; coinbase tx: hash:%s, pubkey:%s, parentHash:%s\n", bi.Height, bi.Hash, ti.Hash, "reorg_tx", bi.PreviousBlockhash)
		ctx.logBlock(bi, ti)
	}
	ctx.RWLock.Unlock()
	return
}

func (ctx *Context) BuildBlockRespWithCoinbaseTx(pubkey string /*hex without 0x, len 64B*/) *BlockInfo {
	if pubkey == "" {
		return nil
	}
	bi := &BlockInfo{
		Hash:          buildBlockHash(ctx.NextBlockHeight),
		Confirmations: 1,      //1 confirm
		Size:          100000, //100k
		Height:        ctx.NextBlockHeight,
		Version:       8888, //for test
		Time:          time.Now().Unix(),
		NumTx:         1,
	}
	bi.Tx = append(bi.Tx, buildTxHash(bi.Hash, 0))
	//change ctx
	ctx.RWLock.Lock()
	if bi.Height > 1 {
		bi.PreviousBlockhash = ctx.BlkByHash[ctx.BlkHashByHeight[bi.Height-1]].Hash
	}
	ctx.BlkByHash[bi.Hash] = bi
	ctx.BlkHashByHeight[ctx.NextBlockHeight] = bi.Hash
	ti := ctx.BuildTxWithPubkey(0, bi.Hash, pubkey)
	ctx.TxByHash[ti.Hash] = ti
	ctx.NextBlockHeight++
	ctx.RWLock.Unlock()
	//limit log amount
	if bi.Height%20 == 1 {
		ctx.Log.Printf("new block: %d, %s; coinbase tx: hash:%s, pubkey:%s\n", bi.Height, bi.Hash, ti.Hash, pubkey)
	}
	ctx.logBlock(bi, ti)
	return bi
}

func (ctx *Context) BuildTxWithPubkey(txIndex int64, blockHash, pubkey string) *TxInfo {
	ti := &TxInfo{
		Hash:      buildTxHash(blockHash, txIndex),
		Size:      100,
		Blockhash: blockHash,
	}
	v := Vout{
		ScriptPubKey: make(map[string]interface{}),
	}
	v.ScriptPubKey["asm"] = "OP_RETURN " + Identifier + Validator + pubkey
	ti.VoutList = append(ti.VoutList, v)
	if ctx.MonitorPubkey != "" {
		v = Vout{
			ScriptPubKey: make(map[string]interface{}),
		}
		v.ScriptPubKey["asm"] = "OP_RETURN " + Identifier + Monitor + ctx.MonitorPubkey
		ti.VoutList = append(ti.VoutList, v)
	}
	return ti
}

func buildTxHash(blockHash string, txIndex int64) string {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(txIndex))
	return blockHash + hex.EncodeToString(b[:])
	//return fmt.Sprintf("%s-%d", blockHash, txIndex)
}

func buildBlockHash(height int64) string {
	var b [8]byte
	binary.BigEndian.PutUint64(b[:], uint64(height))
	var hash [32]byte
	copy(hash[:], hex.EncodeToString(b[:])+hexutil.EncodeUint64(uint64(time.Now().Unix()))[2:])
	return hex.EncodeToString(hash[:])
}

type PubKeyInfo struct {
	Pubkey      string
	VotingPower int64
	RemainCount int64 //init same with Voting power
}

func (ctx *Context) logBlock(bi *BlockInfo, ti *TxInfo) {
	biJSON, _ := json.Marshal(bi)
	ctx.BlockLog.Println("block: ", string(biJSON))

	tiJSON, _ := json.Marshal(ti)
	ctx.BlockLog.Println("tx: ", string(tiJSON))
}

func (ctx *Context) loadBlocksFromLog() {
	ctx.Log.Println("loading blocks from log ...")

	f, err := os.Open("block.log")
	if err != nil {
		ctx.Log.Println(err.Error())
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	scanner.Split(bufio.ScanLines)

	for scanner.Scan() {
		line := scanner.Text()
		if idx := strings.Index(line, "block:"); idx > 0 {
			bi := &BlockInfo{}
			err := json.Unmarshal([]byte(line[idx+6:]), bi)
			if err != nil {
				panic(err)
			}
			ctx.Log.Printf("loaded block: %d\n", bi.Height)
			ctx.BlkByHash[bi.Hash] = bi
			ctx.BlkHashByHeight[bi.Height] = bi.Hash
			ctx.NextBlockHeight = bi.Height + 1
		}
		if idx := strings.Index(line, "tx:"); idx > 0 {
			ti := &TxInfo{}
			err := json.Unmarshal([]byte(line[idx+3:]), ti)
			if err != nil {
				panic(err)
			}
			ctx.Log.Printf("loaded tx: %s\n", ti.Hash)
			ctx.TxByHash[ti.Hash] = ti
		}
		if idx := strings.Index(line, "pubkey:"); idx > 0 {
			pubkeys := map[string]*PubKeyInfo{}
			err := json.Unmarshal([]byte(line[idx+7:]), &pubkeys)
			if err != nil {
				panic(err)
			}
			ctx.Log.Println("loaded pubkeys: ", line[idx+7:])
			ctx.PubkeyInfoByPubkey = pubkeys
		}
	}
}

func (ctx *Context) logPubKeysOnExit() {
	trapSignal(func() {
		ctx.Log.Println("saving pubkeys ...")
		bytes, err := json.Marshal(ctx.PubkeyInfoByPubkey)
		if err == nil {
			ctx.BlockLog.Println("pubkey: ", string(bytes))
		} else {
			ctx.Log.Println(err.Error())
		}
	})
}

func trapSignal(cleanupFunc func()) {
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM, syscall.SIGKILL)
	go func() {
		sig := <-sigs
		if cleanupFunc != nil {
			cleanupFunc()
		}
		exitCode := 128
		switch sig {
		case syscall.SIGINT:
			exitCode += int(syscall.SIGINT)
		case syscall.SIGTERM:
			exitCode += int(syscall.SIGTERM)
		case syscall.SIGKILL:
			exitCode += int(syscall.SIGKILL)
		}
		os.Exit(exitCode)
	}()
}
