package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"math"
	"math/big"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gcash/bchd/bchec"
	"github.com/gcash/bchd/btcjson"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchd/chaincfg/chainhash"
	"github.com/gcash/bchd/rpcclient"
	"github.com/gcash/bchd/txscript"
	"github.com/gcash/bchd/wire"
	"github.com/gcash/bchutil"
	"github.com/holiman/uint256"
	"github.com/smartbch/smartbch/crosschain/abi"
	"github.com/smartbch/smartbch/rpc/client"
	"github.com/smartbch/smartbch/rpc/types"

	"github.com/smartbch/testkit/ethutils"
)

var (
	ccContractAddress = common.HexToAddress("0x0000000000000000000000000000000000002714")
)

type Sender struct {
	chainId *uint256.Int

	mainChainClient *rpcclient.Client
	smartbchClient  *client.Client

	covenantAddress    string
	oldCovenantAddress string
	from               bchutil.Address // main chain tx sender
	wif                *bchutil.WIF    // main chain tx sender wif
	to                 bchutil.Address // covenant address
	oldTo              bchutil.Address // old covenant address
	fee                int64           // main chain tx fee

	sideChainReceiver    string
	sideChainReceiverKey *ecdsa.PrivateKey
	targetAddress        common.Address // pubkey hash160 respond to Sender.from

	minCCAmount float64 // 0.001bch for test
	maxCCAmount float64 // 0.01bch for test

	utxoSeparationMode                     bool // separation utxo for more utxos
	redeemAllMode                          bool // redeem all redeemable utxo
	lostAndFoundAboveMaxAmountMode         bool
	lostAndFoundBelowMinAmountMode         bool
	lostAndFoundWithOldCovenantAddressMode bool
	redeemAllLostAndFoundMode              bool
	transferByBurnMode                     bool
	aggregationMode                        bool

	// data for analysis
	totalTxNumsM2S uint64
	totalTxNumsS2M uint64
	totalAmountM2S *uint256.Int
	totalAmountS2M *uint256.Int
}

func newSender() *Sender {
	var mainChainClientInfo string
	var sideChainUrl string = "http://127.0.0.1:8545"
	var wif string
	var sideChainReceiverK string
	var utxoSeparationMode bool = false
	var redeemAllMode bool = false
	var lostAndFoundAboveMaxAmountMode bool = false
	var lostAndFoundBelowMinAmountMode bool = false
	var redeemAllLostAndFoundMode bool = false
	var transferByBurnMode bool = false
	var lostAndFoundWithOldCovenantAddressMode bool = false
	var aggregationMode bool = false

	flag.StringVar(&mainChainClientInfo, "mainChainClientInfo", mainChainClientInfo, "main chain client info: url,username,password")
	flag.StringVar(&sideChainUrl, "sideChainUrl", sideChainUrl, "side chain url")
	flag.StringVar(&wif, "wif", wif, "main chain wif")
	flag.StringVar(&sideChainReceiverK, "sideChainReceiverKey", sideChainReceiverK, "side chain sender key")
	flag.BoolVar(&utxoSeparationMode, "utxoSeparationMode", utxoSeparationMode, "utxo separation mode")
	flag.BoolVar(&redeemAllMode, "redeemAllMode", redeemAllMode, "redeem all mode")
	flag.BoolVar(&redeemAllLostAndFoundMode, "redeemAllLostAndFoundMode", redeemAllLostAndFoundMode, "redeem all lost and found mode")
	flag.BoolVar(&lostAndFoundAboveMaxAmountMode, "lostAndFoundAboveMaxAmountMode", lostAndFoundAboveMaxAmountMode, "lost and found above max amount mode")
	flag.BoolVar(&lostAndFoundBelowMinAmountMode, "lostAndFoundBelowMinAmountMode", lostAndFoundBelowMinAmountMode, "lost and found below min amount mode")
	flag.BoolVar(&lostAndFoundWithOldCovenantAddressMode, "lostAndFoundWithOldCovenantAddressMode", lostAndFoundWithOldCovenantAddressMode, "lost and found as of transfer to old covenant address mode")
	flag.BoolVar(&transferByBurnMode, "transferByBurnMode", transferByBurnMode, "transfer by burn mode")
	flag.BoolVar(&aggregationMode, "aggregationMode", aggregationMode, "aggregation mode")

	flag.Parse()
	if len(wif) == 0 {
		flag.Usage()
	}
	s := Sender{
		chainId:            uint256.NewInt(0x2712),
		covenantAddress:    "6Ad3f81523c87aa17f1dFA08271cF57b6277C98e",
		utxoSeparationMode: utxoSeparationMode,

		redeemAllMode:                          redeemAllMode,
		lostAndFoundAboveMaxAmountMode:         lostAndFoundAboveMaxAmountMode,
		lostAndFoundBelowMinAmountMode:         lostAndFoundBelowMinAmountMode,
		lostAndFoundWithOldCovenantAddressMode: lostAndFoundWithOldCovenantAddressMode,
		redeemAllLostAndFoundMode:              redeemAllLostAndFoundMode,
		transferByBurnMode:                     transferByBurnMode,
		aggregationMode:                        aggregationMode,
		minCCAmount:                            0.0001,
		maxCCAmount:                            0.01,
		fee:                                    700,
		totalAmountM2S:                         uint256.NewInt(0),
		totalAmountS2M:                         uint256.NewInt(0),
	}
	s.initSideChainFields(sideChainReceiverK)

	var err error
	s.mainChainClient = makeMainChainClient(mainChainClientInfo)
	s.smartbchClient, err = client.Dial(sideChainUrl)
	if err != nil {
		panic(err)
	}
	ccInfo, err := s.smartbchClient.CcInfo(context.Background())
	if err != nil {
		panic(err)
	}
	s.covenantAddress = strings.TrimPrefix(ccInfo.CurrCovenantAddress, "0x")
	if ccInfo.LastCovenantAddress != "0x0000000000000000000000000000000000000000" {
		s.oldCovenantAddress = strings.TrimPrefix(ccInfo.LastCovenantAddress, "0x")
	}
	s.initMainChainFields(wif)
	fmt.Printf(`
Sender basic infos:
covenantAddress:%s
from:%s
to:%s
sideChainReceiver:%s
targetAddress:%s
`, s.covenantAddress, s.from.String(), s.to.String(), s.sideChainReceiver, s.targetAddress.String())
	return &s
}

type crossUtxoInfo struct {
	txid   *big.Int
	amount *big.Int
}

type redeemInfo struct {
	txid common.Hash
}

func makeMainChainClient(mainChainClientInfo string) *rpcclient.Client {
	bchClientParams := strings.Split(mainChainClientInfo, ",")
	if len(bchClientParams) != 3 {
		panic("invalid main chain client param")
	}
	connCfg := &rpcclient.ConnConfig{
		Host:         bchClientParams[0],
		User:         bchClientParams[1],
		Pass:         bchClientParams[2],
		HTTPPostMode: true,
		DisableTLS:   true,
	}
	mainChainClient, err := rpcclient.New(connCfg, nil)
	if err != nil {
		panic(err)
	}
	return mainChainClient
}

func (s *Sender) initMainChainFields(wif string) {
	w, err := bchutil.DecodeWIF(wif)
	if err != nil {
		panic(err)
	}
	s.wif = w
	pkhFrom := bchutil.Hash160(w.SerializePubKey())
	s.targetAddress = common.HexToAddress(hex.EncodeToString(pkhFrom))
	from, err := bchutil.NewAddressPubKeyHash(pkhFrom, &chaincfg.TestNet3Params)
	if err != nil {
		panic(err)
	}
	s.from = from
	pkhTo, err := hex.DecodeString(s.covenantAddress)
	if err != nil {
		panic(err)
	}
	to, err := bchutil.NewAddressScriptHashFromHash(pkhTo, &chaincfg.TestNet3Params)
	s.to = to
	if s.oldCovenantAddress != "" {
		oldPkhTo, err := hex.DecodeString(s.oldCovenantAddress)
		if err != nil {
			panic(err)
		}
		oldTo, err := bchutil.NewAddressScriptHashFromHash(oldPkhTo, &chaincfg.TestNet3Params)
		if err != nil {
			panic(err)
		}
		s.oldTo = oldTo
	}
}

func (s *Sender) initSideChainFields(sideChainReceiverK string) {
	keyBz, err := hex.DecodeString(sideChainReceiverK)
	if err != nil {
		panic(err)
	}
	privateKey, err := crypto.ToECDSA(keyBz)
	if err != nil {
		panic(err)
	}
	s.sideChainReceiverKey = privateKey
	s.sideChainReceiver = crypto.PubkeyToAddress(privateKey.PublicKey).String()[2:]
}

func main() {
	s := newSender()
	if s.redeemAllMode {
		fmt.Println("In redeem all mode !!!")
		_, _ = s.redeemAll()
		return
	}
	if s.redeemAllLostAndFoundMode {
		fmt.Println("redeem all lost and found mode !!!")
		s.redeemAllLostAndFound()
		return
	}
	if s.utxoSeparationMode {
		fmt.Println("In utxo separation mode !!!")
		unspentUtxos := s.listUnspentUtxo(s.from)
		s.utxoSeparation(unspentUtxos)
		return
	}
	s.simpleRun()
}

//
//func main() {
//	s := newSender()
//	time.Sleep(10 * time.Second)
//	if s.redeemAllMode {
//		fmt.Println("In redeem all mode !!!")
//		_, _ = s.redeemAll()
//		return
//	}
//	if s.redeemAllLostAndFoundMode {
//		fmt.Println("redeem all lost and found mode !!!")
//		s.redeemAllLostAndFound()
//		return
//	}
//	for {
//		timeBegin := time.Now().Unix()
//		balanceBefore := s.getSideChainBalance(s.sideChainReceiver)
//		// step 1. send main => side tx
//		fmt.Println("send main chain to side chain transaction...")
//		unspentUtxos := s.listUnspentUtxo(s.from)
//		for i := 0; i < 20; i++ {
//			if len(unspentUtxos) != 0 {
//				break
//			}
//			fmt.Println("Not find unspent utxo, retry 30s later...")
//			time.Sleep(30 * time.Second)
//			unspentUtxos = s.listUnspentUtxo(s.from)
//		}
//		if len(unspentUtxos) == 0 {
//			s.redeemAll()
//			s.redeemAllLostAndFound()
//			time.Sleep(20 * time.Minute)
//			unspentUtxos = s.listUnspentUtxo(s.from)
//		}
//		if s.transferByBurnMode {
//			fmt.Println("In transfer by burn mode !!!")
//			count := 0
//			for _, unspent := range unspentUtxos {
//				err := s.transferByBurn(unspent)
//				if err == nil {
//					count++
//				}
//				if count == 10 {
//					return
//				}
//			}
//		}
//		if s.lostAndFoundWithOldCovenantAddressMode {
//			fmt.Println("In lostAndFoundWithOldCovenantAddress mode !!!")
//			count := 0
//			if s.oldCovenantAddress == "" {
//				fmt.Println("old covenant address is zero")
//				return
//			}
//			for _, unspent := range unspentUtxos {
//				_, err := s.transferToSideChain(unspent, s.oldTo)
//				if err == nil {
//					count++
//				}
//				if count == 10 {
//					return
//				}
//			}
//		}
//		if s.aggregationMode {
//			fmt.Println("In utxo aggregation mode !!!")
//			s.UtxosAggregation(unspentUtxos, 0.003, 0.001)
//			return
//		}
//		if s.utxoSeparationMode {
//			fmt.Println("In utxo separation mode !!!")
//			s.utxoSeparation(unspentUtxos)
//			return
//		}
//		if s.lostAndFoundAboveMaxAmountMode {
//			fmt.Println("lost and found above max amount mode !!!")
//		}
//		if s.lostAndFoundBelowMinAmountMode {
//			fmt.Println("lost and found below min amount mode !!!")
//		}
//		for {
//			var utxoInfos []*crossUtxoInfo
//			count := 0
//			for _, unspentUtxo := range unspentUtxos {
//				if s.lostAndFoundAboveMaxAmountMode {
//					if unspentUtxo.Amount < s.maxCCAmount+0.00001 {
//						continue
//					}
//				}
//				if s.lostAndFoundBelowMinAmountMode {
//					// we should make the amount in (0.0006, 0.001), so the utxo will trigger lost and found below minCCAmount
//					if unspentUtxo.Amount >= 0.0008 {
//						txid, _ := s.transferSingleInput(unspentUtxo, s.from, s.to, int64(70000), s.fee, []byte(s.sideChainReceiver), s.wif.PrivKey, s.wif.SerializePubKey())
//						fmt.Printf("txid:%s\n", txid.String())
//						return
//					} else {
//						continue
//					}
//				}
//				txHash, err := s.transferToSideChain(unspentUtxo, s.to)
//				if err != nil {
//					fmt.Printf("transfer to side chain failed: %s\n", err.Error())
//					continue
//				}
//				txid, ok := big.NewInt(0).SetString(txHash.String(), 16)
//				if !ok {
//					panic(fmt.Sprintf("convert tx hash %s to big.Int failed", txHash.String()))
//				}
//				utxoInfos = append(utxoInfos, &crossUtxoInfo{
//					txid:   txid,
//					amount: big.NewInt(int64(math.Round(unspentUtxo.Amount*1e8)) - s.fee),
//				})
//				count++
//			}
//			if count != 0 {
//				break
//			}
//			// there has no valid utxo, maybe should redeem some.
//			for k := 0; ; k++ {
//				s.redeemAll()
//				s.redeemAllLostAndFound()
//				time.Sleep(time.Duration(k) * 12 * time.Minute)
//				for i := 0; i < 20; i++ {
//					unspentUtxos = s.listUnspentUtxo(s.from)
//					if len(unspentUtxos) == 0 {
//						time.Sleep(10 * time.Second)
//						continue
//					}
//					break
//				}
//				if len(unspentUtxos) != 0 {
//					break
//				}
//				fmt.Println("unspentUtxos length is zero, continue!!!")
//			}
//		}
//		// step 2. wait side chain handle these cross chain utxo
//		for {
//			fmt.Println("waiting main to side tx be handled by side chain...")
//			time.Sleep(10 * time.Minute)
//			balanceAfter := s.getSideChainBalance(s.sideChainReceiver)
//			if balanceAfter.Gt(balanceBefore) {
//				// it means cross chain utxo handled
//				fmt.Printf("side chain receiver:%s, balance increase:%s\n", s.sideChainReceiver, uint256.NewInt(0).Sub(balanceAfter, balanceBefore).String())
//				break
//			}
//		}
//		// step 3. send redeem txs
//		redeemTimes := 0
//		successRedeemNums := 0
//		for {
//			totalRedeemableNums, redeemInfos := s.redeemAll()
//			time.Sleep(10 * time.Second)
//			for _, info := range redeemInfos {
//				receipt, err := s.smartbchClient.TransactionReceipt(context.Background(), info.txid)
//				if err != nil {
//					fmt.Printf("get %s receipt failed:%s\n", info.txid, err.Error())
//					continue
//				}
//				if receipt.Status != uint64(1) {
//					out, _ := json.MarshalIndent(receipt, "", "  ")
//					fmt.Printf("redeem tx failed, receipt:%s\n", string(out))
//					continue
//				}
//				s.totalTxNumsS2M++
//				successRedeemNums++
//			}
//			if !(totalRedeemableNums != 0 && successRedeemNums == 0) {
//				break
//			}
//			successRedeemNums = 0
//			redeemTimes++
//			if redeemTimes > 6 {
//				redeemTimes = 6 // 1hour
//			}
//			// not wasting gas fee if cc pause, todo: change to check ccInfo.MonitorsWithPauseCommand
//			time.Sleep(time.Duration(redeemTimes) * 10 * time.Minute)
//			fmt.Println("redeem nothing, new round for redeem!!!")
//		}
//		// redeem all lostAndFound by the way, not check result
//		s.redeemAllLostAndFound()
//		// wait redeem tx mint in main chain
//		time.Sleep(10 * time.Minute)
//
//		timeAfter := time.Now().Unix()
//		fmt.Printf(`
//	Summary:
//	transfer %d cross tx this round
//	total bch from main chain to side chain:%d
//	total bch from side chain to main chain:%d
//	total txs from main chain to side chain:%d
//	total txs from side chain to main chain:%d
//	total time:%d
//	`, successRedeemNums, s.totalAmountM2S.Uint64(), s.totalAmountS2M.Uint64(), s.totalTxNumsM2S, s.totalTxNumsS2M, timeAfter-timeBegin)
//		fmt.Printf("\nAnother New Round Start !!!\n")
//	}
//}

func (s *Sender) getSideChainBalance(address string) *uint256.Int {
	var out *uint256.Int
	for {
		balance, err := s.smartbchClient.BalanceAt(context.Background(), common.HexToAddress(address), nil)
		if err != nil {
			fmt.Println(err)
			time.Sleep(10 * time.Second)
			continue
		}
		out, _ = uint256.FromBig(balance)
		break
	}
	return out
}

func (s *Sender) getNonce() uint64 {
	var nonce uint64
	var err error
	for {
		nonce, err = s.smartbchClient.NonceAt(context.Background(), common.HexToAddress(s.sideChainReceiver), nil)
		if err != nil {
			fmt.Println(err)
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}
	return nonce
}

func (s *Sender) getMainChainBalance(account string) *uint256.Int {
	// todo: not work
	balance, err := s.mainChainClient.GetBalance(account)
	if err != nil {
		panic(err)
	}
	return uint256.NewInt(uint64(balance))
}

func (s *Sender) listUnspentUtxo(address bchutil.Address) []btcjson.ListUnspentResult {
	var unspentList []btcjson.ListUnspentResult
	var err error
	for {
		unspentList, err = s.mainChainClient.ListUnspentMinMaxAddresses(1, 9999, []bchutil.Address{address})
		if err != nil {
			fmt.Println(err)
			time.Sleep(10 * time.Second)
			continue
		}
		fmt.Printf("unspent utxos length:%d\n", len(unspentList))
		break
	}
	return unspentList
}

func (s *Sender) transferToSideChain(unspentUtxo btcjson.ListUnspentResult, to bchutil.Address) (*chainhash.Hash, error) {
	mul := 1e8
	if int64(math.Round(unspentUtxo.Amount*mul)) > s.fee {
		txid, err := s.transferSingleInput(unspentUtxo, s.from, to, int64(math.Round(unspentUtxo.Amount*mul))-s.fee, s.fee, []byte(s.sideChainReceiver), s.wif.PrivKey, s.wif.SerializePubKey())
		if err == nil {
			s.totalAmountM2S = uint256.NewInt(0).Add(s.totalAmountM2S, uint256.NewInt(uint64(unspentUtxo.Amount*mul)-uint64(s.fee)))
			s.totalTxNumsM2S++
			fmt.Println("send main chain to side chain transaction success, txid:" + txid.String())
		}
		return txid, err
	}
	return nil, errors.New("unspent utxo amount not cover fee")
}

func (s *Sender) transferSingleInput(unspent btcjson.ListUnspentResult, from, to bchutil.Address, amount, fee int64, nullData []byte, privateKey *bchec.PrivateKey, fromPubkey []byte) (*chainhash.Hash, error) {
	tx := wire.NewMsgTx(2)
	if int64(math.Round(unspent.Amount*(1e8))) < amount+fee {
		return nil, errors.New("unspent amount not enough")
	}
	// add input
	hash, _ := chainhash.NewHashFromStr(unspent.TxID)
	outPoint := wire.NewOutPoint(hash, unspent.Vout)
	txIn := wire.NewTxIn(outPoint, nil)
	tx.AddTxIn(txIn)
	// add output
	pkScript, err := txscript.PayToAddrScript(to)
	if err != nil {
		return nil, err
	}
	txOut := wire.NewTxOut(amount, pkScript)
	tx.AddTxOut(txOut)
	change := int64(math.Round(unspent.Amount*1e8)) - amount - fee
	if change > 0 {
		// add change receiver
		pkScript, err := txscript.PayToAddrScript(from)
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(wire.NewTxOut(change, pkScript))
	}
	if len(nullData) != 0 {
		nullScript, err := txscript.NullDataScript(nullData)
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(wire.NewTxOut(0, nullScript))
	}
	// sign
	scriptPubkey, err := hex.DecodeString(unspent.ScriptPubKey)
	if err != nil {
		return nil, err
	}
	sigHashes := txscript.NewTxSigHashes(tx)
	hashType := txscript.SigHashAll | txscript.SigHashForkID
	sigHash, err := txscript.CalcSignatureHash(scriptPubkey, sigHashes, hashType, tx, 0, int64(math.Round(unspent.Amount*1e8)), true)
	if err != nil {
		return nil, err
	}
	sig, err := privateKey.SignECDSA(sigHash)
	if err != nil {
		panic(err)
	}
	sigScript, err := txscript.NewScriptBuilder().AddData(append(sig.Serialize(), byte(hashType))).AddData(fromPubkey).Script()
	tx.TxIn[0].SignatureScript = sigScript
	var buf bytes.Buffer
	_ = tx.Serialize(&buf)
	//fmt.Println(hex.EncodeToString(buf.Bytes()))
	txHash, err := s.mainChainClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	return txHash, nil
}

func (s *Sender) checkMainChainTxStatus(txHash *chainhash.Hash) error {
	res, err := s.mainChainClient.GetRawTransactionVerbose(txHash)
	if err != nil {
		return err
	}
	if res.Confirmations < 1 {
		return errors.New("not get enough confirm")
	}
	return nil
}

func (s *Sender) utxoSeparation(unspentUtxos []btcjson.ListUnspentResult) {
	for _, unspent := range unspentUtxos {
		// 8x minCCAmount
		if int64(math.Round(unspent.Amount*1e8)) > int64(s.minCCAmount*1e8*8)+1000 {
			outAmount := (int64(math.Round(unspent.Amount*1e8)) - 1000) / 4
			fmt.Printf("separate tx %s to 4 parts, which amount is %d\n", unspent.TxID, outAmount)
			_, _ = s.transferForSeparation(unspent, s.from, outAmount, 4, 1000, s.wif.PrivKey, s.wif.SerializePubKey())
		} else if int64(math.Round(unspent.Amount*1e8)) > int64(s.minCCAmount*1e8*4)+1000 /* 4x minCCAmount */ {
			outAmount := (int64(math.Round(unspent.Amount*1e8)) - 1000) / 2
			fmt.Printf("separate tx %s to 2 parts, which amount is %d\n", unspent.TxID, outAmount)
			_, _ = s.transferForSeparation(unspent, s.from, outAmount, 2, 1000, s.wif.PrivKey, s.wif.SerializePubKey())
		}
	}
}

func (s *Sender) UtxosAggregation(unspentUtxos []btcjson.ListUnspentResult, maxAmountToCollect, minAmountToCollect float64) {
	var utxosToSpend []btcjson.ListUnspentResult
	count := 0
	for _, unspent := range unspentUtxos {
		if unspent.Amount >= minAmountToCollect && unspent.Amount <= maxAmountToCollect {
			utxosToSpend = append(utxosToSpend, unspent)
			count++
		}
		if count == 1 {
			fmt.Println("hit")
			break
		}
	}
	if count != 1 {
		fmt.Println("not hit")
		return
	}
	for _, unspent := range unspentUtxos {
		if unspent.Amount == 0.00000001 {
			utxosToSpend = append(utxosToSpend, unspent)
			count++
		}
		if count == 50 {
			break
		}
	}
	hash, err := s.transferForAggregation(utxosToSpend, s.from, 7500, s.wif.PrivKey, s.wif.SerializePubKey())
	if err != nil {
		fmt.Println(err)
	} else {
		fmt.Println(hash.String())
	}
}

func (s *Sender) transferByBurn(unspent btcjson.ListUnspentResult) error {
	mul := 1e8
	if int64(math.Round(unspent.Amount*mul)) > s.fee+100 {
		txid, err := s.transferSingleInput(unspent, s.from, s.to, 100, s.fee, []byte(s.sideChainReceiver), s.wif.PrivKey, s.wif.SerializePubKey())
		if err != nil {
			fmt.Println(err)
		} else {
			fmt.Println("send transferByBurn success:" + txid.String())
		}
		return err
	}
	return errors.New("amount not enough")
}

// transfer to self for get many tiny utxo
func (s *Sender) transferForSeparation(unspent btcjson.ListUnspentResult, from bchutil.Address, amount, outputNums, fee int64, privateKey *bchec.PrivateKey, fromPubkey []byte) (*chainhash.Hash, error) {
	tx := wire.NewMsgTx(2)
	if int64(math.Round(unspent.Amount*(1e8))) < (amount*outputNums + fee) {
		return nil, errors.New("unspent amount not enough")
	}
	// add input
	hash, _ := chainhash.NewHashFromStr(unspent.TxID)
	outPoint := wire.NewOutPoint(hash, unspent.Vout)
	txIn := wire.NewTxIn(outPoint, nil)
	tx.AddTxIn(txIn)
	// add output
	pkScript, err := txscript.PayToAddrScript(from)
	if err != nil {
		return nil, err
	}
	txOut := wire.NewTxOut(amount, pkScript)
	for i := int64(0); i < outputNums; i++ {
		tx.AddTxOut(txOut)
	}
	change := int64(math.Round(unspent.Amount*1e8)) - amount*outputNums - fee
	if change > 0 {
		// add change receiver
		pkScript, err := txscript.PayToAddrScript(from)
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(wire.NewTxOut(change, pkScript))
	}
	// sign
	scriptPubkey, err := hex.DecodeString(unspent.ScriptPubKey)
	if err != nil {
		return nil, err
	}
	sigHashes := txscript.NewTxSigHashes(tx)
	hashType := txscript.SigHashAll | txscript.SigHashForkID
	sigHash, err := txscript.CalcSignatureHash(scriptPubkey, sigHashes, hashType, tx, 0, int64(math.Round(unspent.Amount*1e8)), true)
	if err != nil {
		return nil, err
	}
	sig, err := privateKey.SignECDSA(sigHash)
	if err != nil {
		panic(err)
	}
	sigScript, err := txscript.NewScriptBuilder().AddData(append(sig.Serialize(), byte(hashType))).AddData(fromPubkey).Script()
	tx.TxIn[0].SignatureScript = sigScript
	var buf bytes.Buffer
	_ = tx.Serialize(&buf)
	txHash, err := s.mainChainClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	return txHash, nil
}

// transfer to self for get many tiny utxo
func (s *Sender) transferForAggregation(unspentList []btcjson.ListUnspentResult, from bchutil.Address, fee int64, privateKey *bchec.PrivateKey, fromPubkey []byte) (*chainhash.Hash, error) {
	tx := wire.NewMsgTx(2)
	// add input
	totalInAmount := 0.0
	for _, unspent := range unspentList {
		hash, _ := chainhash.NewHashFromStr(unspent.TxID)
		outPoint := wire.NewOutPoint(hash, unspent.Vout)
		txIn := wire.NewTxIn(outPoint, nil)
		tx.AddTxIn(txIn)
		totalInAmount += unspent.Amount
	}
	if int64(math.Round(totalInAmount*1e8)) <= fee {
		return nil, errors.New("not cover fee")
	}
	// add output
	change := int64(math.Round(totalInAmount*1e8)) - fee
	if change > 0 {
		// add change receiver
		pkScript, err := txscript.PayToAddrScript(from)
		if err != nil {
			return nil, err
		}
		tx.AddTxOut(wire.NewTxOut(change, pkScript))
	}
	// sign
	sigHashes := txscript.NewTxSigHashes(tx)
	for idx, unspent := range unspentList {
		scriptPubkey, err := hex.DecodeString(unspent.ScriptPubKey)
		if err != nil {
			return nil, err
		}
		hashType := txscript.SigHashSingle | txscript.SigHashForkID
		sigHash, err := txscript.CalcSignatureHash(scriptPubkey, sigHashes, hashType, tx, idx, int64(math.Round(unspent.Amount*1e8)), true)
		if err != nil {
			return nil, err
		}

		sig, err := privateKey.SignECDSA(sigHash)
		if err != nil {
			panic(err)
		}
		sigScript, err := txscript.NewScriptBuilder().AddData(append(sig.Serialize(), byte(hashType))).AddData(fromPubkey).Script()
		tx.TxIn[idx].SignatureScript = sigScript
	}

	var buf bytes.Buffer
	_ = tx.Serialize(&buf)
	txHash, err := s.mainChainClient.SendRawTransaction(tx, false)
	if err != nil {
		return nil, err
	}
	return txHash, nil
}

//(successRedeemCount, allRedeemableCount int)
func (s *Sender) redeemAll() (totalRedeemableNums int, redeemInfos []*redeemInfo) {
	fmt.Println("redeem all")
	var utxoInfos *types.UtxoInfos
	var err error
	for {
		utxoInfos, err = s.smartbchClient.RedeemableUtxos(context.Background())
		if err != nil {
			fmt.Println(err)
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}
	totalRedeemableNums = len(utxoInfos.Infos)
	nonce := s.getNonce()
	for _, info := range utxoInfos.Infos {
		for {
			txid, err := s.redeem(info.Txid.Big(), big.NewInt(0).Mul(big.NewInt(int64(info.Amount)), big.NewInt(1e10)), nonce)
			if err != nil {
				fmt.Println("redeem failed, retry 10s later!!!")
				time.Sleep(10 * time.Second)
				nonce = s.getNonce()
				continue
			}
			redeemInfos = append(redeemInfos, &redeemInfo{txid: txid})
			break
		}
		nonce++
		time.Sleep(6 * time.Second)
	}
	return
}

func (s *Sender) redeem(txid, amount *big.Int, nonce uint64) (common.Hash, error) {
	data := abi.PackRedeemFunc(txid, big.NewInt(0), s.targetAddress)
	gasLimit := 4000_000
	gasPrice := uint256.NewInt(10_000_000_000)
	tx := ethutils.NewTx(nonce, &ccContractAddress, amount, uint64(gasLimit), gasPrice.ToBig(), data)
	signedTx, err := ethutils.SignTx(tx, s.chainId.ToBig(), s.sideChainReceiverKey)
	if err != nil {
		panic(err)
	}
	out, err := signedTx.MarshalJSON()
	if err != nil {
		panic(err)
	}
	fmt.Printf("redeem tx:%s\n", string(out))
	err = s.smartbchClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		fmt.Println("SendTransaction:" + err.Error())
		return common.Hash{}, err
	}
	return signedTx.Hash(), nil
}

func (s *Sender) redeemAllLostAndFound() {
	fmt.Println("redeem all lost and found")
	for {
		utxoInfos, err := s.smartbchClient.LostAndFoundUtxos(context.Background())
		if err != nil {
			fmt.Println(err)
			time.Sleep(10 * time.Second)
			continue
		}
		nonce := s.getNonce()
		for _, info := range utxoInfos.Infos {
			_, err := s.redeem(info.Txid.Big(), big.NewInt(0), nonce)
			if err != nil {
				time.Sleep(10 * time.Second)
				nonce = s.getNonce()
				continue
			}
			nonce++
			time.Sleep(6 * time.Second)
		}
		break
	}
}

func (s *Sender) simpleRun() {
	fmt.Println("simple run...")
	go func() {
		for {
			if !s.isPaused() {
				s.redeemAll()
				s.redeemAllLostAndFound()
			}
			time.Sleep(20 * time.Minute)
		}
	}()
	go func() {
		for {
			unspentUtxos := s.listUnspentUtxo(s.from)
			s.updateTo()
			for _, unspent := range unspentUtxos {
				_, err := s.transferToSideChain(unspent, s.to)
				if err != nil {
					fmt.Println(err)
				}
			}
			time.Sleep(10 * time.Minute)
		}
	}()
	select {}
}

func (s *Sender) updateTo() {
	var ccInfo *types.CcInfo
	var err error
	for {
		ccInfo, err = s.smartbchClient.CcInfo(context.Background())
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}
	currCovenantAddressInCcInfo := strings.TrimPrefix(strings.ToLower(ccInfo.CurrCovenantAddress), "0x")
	currCovenantAddressLocal := strings.TrimPrefix(strings.ToLower(s.covenantAddress), "0x")
	if currCovenantAddressInCcInfo != currCovenantAddressLocal {
		s.covenantAddress = currCovenantAddressInCcInfo
		pkhTo, err := hex.DecodeString(s.covenantAddress)
		if err != nil {
			panic(err)
		}
		s.to, _ = bchutil.NewAddressScriptHashFromHash(pkhTo, &chaincfg.TestNet3Params)
		fmt.Printf("change covenant address from %s to %s\n", currCovenantAddressLocal, currCovenantAddressInCcInfo)
	}
}

func (s *Sender) isPaused() bool {
	var info *types.CcInfo
	var err error
	for {
		info, err = s.smartbchClient.CcInfo(context.Background())
		if err != nil {
			time.Sleep(10 * time.Second)
			continue
		}
		break
	}
	if len(info.MonitorsWithPauseCommand) == 0 {
		return false
	} else {
		fmt.Println("isPaused")
		return true
	}
}
