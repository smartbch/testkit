package generator

import (
	"encoding/binary"
	"encoding/hex"
	"io"
	"log"
	"os"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
)

var Version = "00"
var Identifier = "73424348"

type Context struct {
	RWLock sync.RWMutex

	Log *log.Logger

	Producer *Producer

	TxByHash           map[string]*TxInfo
	BlkByHash          map[string]*BlockInfo
	BlkHashByHeight    map[int64]string
	PubkeyInfoByPubkey map[string]*PubKeyInfo
	NextBlockHeight    int64

	//internal
	PubKeyInfoSet   []PubKeyInfo
	PubkeyInfoIndex int
}

var Ctx Context

func Init() {
	Ctx.NextBlockHeight = 1
	Ctx.TxByHash = make(map[string]*TxInfo)
	Ctx.BlkByHash = make(map[string]*BlockInfo)
	Ctx.BlkHashByHeight = make(map[int64]string)
	Ctx.PubkeyInfoByPubkey = make(map[string]*PubKeyInfo)
	//inti logger
	file, err := os.OpenFile("out.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		panic(err)
	}
	Ctx.Log = log.New(io.MultiWriter(file, os.Stdout), "INFO: ", log.Ltime|log.Lshortfile)
	//init producer
	Ctx.Producer = &Producer{
		Exit:              make(chan bool),
		Reorg:             make(chan bool, 1),
		BlockIntervalTime: 3,
	}
	go Ctx.Producer.Start()
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
	Value        float64                `json:"value"`
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

func ReorgBlock() {
	h := Ctx.NextBlockHeight
	initHeight := h - reorgBlockNumbers
	if initHeight <= 1 {
		return
	}
	Ctx.RWLock.Lock()
	for i := int64(0); i < reorgBlockNumbers; i++ {
		bi := &BlockInfo{
			Hash: buildBlockHash(initHeight),
			Confirmations: 1,      //1 confirm
			Size:          100000, //100k
			Height:        initHeight,
			Version:       8888, //for test
			Time:          time.Now().Unix(),
			NumTx:         1,
		}
		bi.Tx = append(bi.Tx, buildTxHash(bi.Hash, 0))
		ti := BuildTxWithPubkey(0, bi.Hash, "reorg_tx")
		//change ctx
		if bi.Height > 1 {
			bi.PreviousBlockhash = Ctx.BlkByHash[Ctx.BlkHashByHeight[bi.Height-1]].Hash
		}
		Ctx.BlkByHash[bi.Hash] = bi
		Ctx.BlkHashByHeight[initHeight] = bi.Hash
		Ctx.TxByHash[ti.Hash] = ti
		initHeight++

		Ctx.Log.Printf("Reorg: new block: %d, %s; coinbase tx: hash:%s, pubkey:%s, parentHash:%s\n", bi.Height, bi.Hash, ti.Hash, "reorg_tx", bi.PreviousBlockhash)
	}
	Ctx.RWLock.Unlock()
	return
}

func BuildBlockRespWithCoinbaseTx(pubkey string /*hex without 0x, len 64B*/) *BlockInfo {
	if pubkey == "" {
		return nil
	}
	bi := &BlockInfo{
		Hash:          buildBlockHash(Ctx.NextBlockHeight),
		Confirmations: 1,      //1 confirm
		Size:          100000, //100k
		Height:        Ctx.NextBlockHeight,
		Version:       8888, //for test
		Time:          time.Now().Unix(),
		NumTx:         1,
	}
	bi.Tx = append(bi.Tx, buildTxHash(bi.Hash, 0))
	ti := BuildTxWithPubkey(0, bi.Hash, pubkey)
	//change ctx
	Ctx.RWLock.Lock()
	if bi.Height > 1 {
		bi.PreviousBlockhash = Ctx.BlkByHash[Ctx.BlkHashByHeight[bi.Height-1]].Hash
	}
	Ctx.BlkByHash[bi.Hash] = bi
	Ctx.BlkHashByHeight[Ctx.NextBlockHeight] = bi.Hash
	Ctx.TxByHash[ti.Hash] = ti
	Ctx.NextBlockHeight++
	Ctx.RWLock.Unlock()
	Ctx.Log.Printf("new block: %d, %s; coinbase tx: hash:%s, pubkey:%s\n", bi.Height, bi.Hash, ti.Hash, pubkey)
	return bi
}

func BuildTxWithPubkey(txIndex int64, blockHash, pubkey string) *TxInfo {
	ti := &TxInfo{
		Hash:      buildTxHash(blockHash, txIndex),
		Size:      100,
		Blockhash: blockHash,
	}
	v := Vout{
		ScriptPubKey: make(map[string]interface{}),
	}
	v.ScriptPubKey["asm"] = "OP_RETURN " + Identifier + Version + pubkey
	ti.VoutList = append(ti.VoutList, v)
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
	return hex.EncodeToString(b[:]) + hexutil.EncodeUint64(uint64(time.Now().Unix()))[2:]
	//return fmt.Sprintf("%d.%d", height, time.Now().Unix())
}

type PubKeyInfo struct {
	Pubkey      string
	VotingPower int64
	RemainCount int64 //init same with Voting power
}
