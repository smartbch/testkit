package main

import (
	"bytes"
	"context"
	"crypto/ecdsa"
	"encoding/hex"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
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

	"github.com/smartbch/testkit/ethutils"
)

var (
	ccContractAddress = common.HexToAddress("0x0000000000000000000000000000000000002714")
)

type Sender struct {
	chainId *uint256.Int

	mainChainClient *rpcclient.Client
	smartbchClient  *client.Client

	covenantAddress string
	from            bchutil.Address // main chain tx sender
	wif             *bchutil.WIF    // main chain tx sender wif
	to              bchutil.Address // covenant address
	fee             int64           // main chain tx fee

	sideChainReceiver    string
	sideChainReceiverKey *ecdsa.PrivateKey
	targetAddress        common.Address // pubkey hash160 respond to Sender.from

	minCCAmount float64 // 0.001bch for test

	utxoSeparationMode bool // separation utxo for more utxos
	redeemAllMode      bool // redeem all redeemable utxo

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

	flag.StringVar(&mainChainClientInfo, "mainChainClientInfo", mainChainClientInfo, "main chain client info: url,username,password")
	flag.StringVar(&sideChainUrl, "sideChainUrl", sideChainUrl, "side chain url")
	flag.StringVar(&wif, "wif", wif, "main chain wif")
	flag.StringVar(&sideChainReceiverK, "sideChainReceiverKey", sideChainReceiverK, "side chain sender key")
	flag.BoolVar(&utxoSeparationMode, "utxoSeparationMode", utxoSeparationMode, "utxo separation mode")
	flag.BoolVar(&redeemAllMode, "redeemAllMode", redeemAllMode, "redeem all mode")
	flag.Parse()
	if len(wif) == 0 {
		flag.Usage()
	}
	s := Sender{
		chainId:            uint256.NewInt(0x2712),
		covenantAddress:    "6ad3f81523c87aa17f1dfa08271cf57b6277c98e",
		sideChainReceiver:  "b24FD9aeCaC7034819FffE8064bA5133e2Ef1a4F",
		utxoSeparationMode: utxoSeparationMode,
		redeemAllMode:      redeemAllMode,
	}
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
	from.ScriptAddress()
	pkhTo, err := hex.DecodeString(s.covenantAddress)
	if err != nil {
		panic(err)
	}
	to, err := bchutil.NewAddressScriptHashFromHash(pkhTo, &chaincfg.TestNet3Params)
	s.to = to
	s.fee = 1000
	keyBz, err := hex.DecodeString(sideChainReceiverK)
	if err != nil {
		panic(err)
	}
	privateKey, err := crypto.ToECDSA(keyBz)
	if err != nil {
		panic(err)
	}
	s.sideChainReceiverKey = privateKey
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
	s.mainChainClient = mainChainClient
	smartbchClient, err := client.Dial(sideChainUrl)
	if err != nil {
		panic(err)
	}
	s.smartbchClient = smartbchClient
	s.minCCAmount = 0.001
	s.totalAmountM2S = uint256.NewInt(0)
	s.totalAmountS2M = uint256.NewInt(0)

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
	txid   common.Hash
	amount *uint256.Int
}

func main() {
	s := newSender()
	for {
		timeBegin := time.Now().Unix()
		balanceBefore := s.getSideChainBalance(s.sideChainReceiver)
		// step 1. send main => side tx
		fmt.Println("send main chain to side chain transaction...")
		unspentUtxos := s.listUnspentUtxo(s.from)
		if len(unspentUtxos) == 0 {
			time.Sleep(30 * time.Second)
			fmt.Println("Not find unspent utxo, retry 30s later...")
			continue
		}
		if s.utxoSeparationMode {
			fmt.Println("In utxo separation mode !!!")
			s.utxoSeparation(unspentUtxos)
			return
		}
		if s.redeemAllMode {
			fmt.Println("In redeem all mode !!!")
			s.redeemAll()
			return
		}
		var utxoInfos []*crossUtxoInfo
		for _, unspentUtxo := range unspentUtxos {
			txHash, err := s.transferToSideChain(unspentUtxo)
			if err != nil {
				fmt.Printf("transfer to side chain failed: %s\n", err.Error())
				continue
			}
			txid, ok := big.NewInt(0).SetString(txHash.String(), 16)
			if !ok {
				panic(fmt.Sprintf("convert tx hash %s to big.Int failed", txHash.String()))
			}
			utxoInfos = append(utxoInfos, &crossUtxoInfo{
				txid:   txid,
				amount: big.NewInt(int64(unspentUtxo.Amount*1e8) - s.fee),
			})
		}
		// step 2. wait side chain handle these cross chain utxo
		for {
			fmt.Println("waiting main to side tx be handled by side chain...")
			time.Sleep(300 * time.Second)
			balanceAfter := s.getSideChainBalance(s.sideChainReceiver)
			if balanceAfter.Gt(balanceBefore) {
				// it means cross chain utxo handled
				fmt.Printf("side chain receiver:%s, balance increase:%s\n", s.sideChainReceiver, uint256.NewInt(0).Sub(balanceAfter, balanceBefore).String())
				break
			}
		}
		unspentUtxos = s.listUnspentUtxo(s.from)
		unspentUtxoNumsBefore := len(unspentUtxos)
		// step 3. send redeem txs
		fmt.Printf("send redeem txs...\n")
		var redeemInfos []*redeemInfo
		nonce, err := s.smartbchClient.NonceAt(context.Background(), common.HexToAddress(s.sideChainReceiver), nil)
		if err != nil {
			panic(err)
		}
		fmt.Printf("nonce is %d\n", nonce)
		for _, info := range utxoInfos {
			txid, err := s.redeem(info.txid, big.NewInt(0).Mul(info.amount, big.NewInt(1e10)), nonce)
			if err != nil {
				fmt.Printf("send redeem tx error:%s\n", err.Error())
				continue
			}
			amount, _ := uint256.FromBig(info.amount)
			redeemInfos = append(redeemInfos, &redeemInfo{
				txid:   txid,
				amount: amount,
			})
			nonce++
			time.Sleep(8 * time.Second)
		}
		time.Sleep(18 * time.Second)
		successRedeemNums := 0
		for _, info := range redeemInfos {
			//fmt.Printf("please check the redeem tx status:%s\n", info.txid)
			receipt, err := s.smartbchClient.TransactionReceipt(context.Background(), info.txid)
			if err != nil {
				fmt.Printf("get %s receipt failed:%s\n", info.txid, err.Error())
				continue
			}
			if receipt.Status != uint64(1) {
				out, _ := json.MarshalIndent(receipt, "", "  ")
				fmt.Printf("redeem tx failed, receipt:%s\n", string(out))
				continue
			}
			s.totalAmountS2M = uint256.NewInt(0).Add(s.totalAmountS2M, info.amount)
			s.totalTxNumsS2M++
			successRedeemNums++
		}
		// step 4. check main chain sender balance, if redeem success, it will increase and amount of unspent utxo will increase too.
		for {
			unspentUtxos = s.listUnspentUtxo(s.from)
			unspentUtxoNumsAfter := len(unspentUtxos)
			if unspentUtxoNumsAfter >= unspentUtxoNumsBefore+successRedeemNums {
				fmt.Printf("redeem successs, we get %d new unspent utxo\n", unspentUtxoNumsAfter-unspentUtxoNumsBefore)
				for _, un := range unspentUtxos {
					out, _ := json.MarshalIndent(un, "", "  ")
					fmt.Println(string(out))
				}
				break
			}
			fmt.Println("waiting operator handle the redeem txs...")
			time.Sleep(300 * time.Second)
		}
		timeAfter := time.Now().Unix()
		fmt.Printf(`
Summary:
transfer %d cross tx this round
total bch from main chain to side chain:%d
total bch from side chain to main chain:%d
total txs from main chain to side chain:%d
total txs from side chain to main chain:%d
total time:%d
`, successRedeemNums, s.totalAmountM2S.Uint64(), s.totalAmountS2M.Uint64(), s.totalTxNumsM2S, s.totalTxNumsS2M, timeAfter-timeBegin)
		time.Sleep(300 * time.Second)
		fmt.Printf("Another New Round Start !!!\n")
	}
}

func (s *Sender) redeem(txid, amount *big.Int, nonce uint64) (common.Hash, error) {
	data := abi.PackRedeemFunc(txid, big.NewInt(0), s.targetAddress)
	gasLimit := 4000_000
	gasPrice := uint256.NewInt(10_000_000_000)
	tx := ethutils.NewTx(nonce, &ccContractAddress, amount, uint64(gasLimit), gasPrice.ToBig(), data)
	out, err := tx.MarshalJSON()
	if err != nil {
		panic(err)
	}
	fmt.Printf("redeem tx:%s\n", string(out))
	signedTx, err := ethutils.SignTx(tx, s.chainId.ToBig(), s.sideChainReceiverKey)
	if err != nil {
		panic(err)
	}
	err = s.smartbchClient.SendTransaction(context.Background(), signedTx)
	if err != nil {
		fmt.Println("SendTransaction:" + err.Error())
		return common.Hash{}, err
	}
	return signedTx.Hash(), nil
}

func (s *Sender) getSideChainBalance(address string) *uint256.Int {
	balance, err := s.smartbchClient.BalanceAt(context.Background(), common.HexToAddress(address), nil)
	if err != nil {
		panic(err)
	}
	out, _ := uint256.FromBig(balance)
	return out
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
	unspentList, err := s.mainChainClient.ListUnspentMinMaxAddresses(1, 9999, []bchutil.Address{address})
	if err != nil {
		panic(err)
	}
	//out, _ := json.MarshalIndent(unspentList, "", "  ")
	//fmt.Printf("unspent utxos: \n%s\n", string(out))
	fmt.Printf("unspent utxos length:%d\n", len(unspentList))
	return unspentList
}

func (s *Sender) transferToSideChain(unspentUtxo btcjson.ListUnspentResult) (*chainhash.Hash, error) {
	mul := 1e8
	if int64(unspentUtxo.Amount*mul) > s.fee {
		txid, err := s.transferSingleInput(unspentUtxo, s.from, s.to, int64(unspentUtxo.Amount*mul)-s.fee, s.fee, []byte(s.sideChainReceiver), s.wif.PrivKey, s.wif.SerializePubKey())
		if err != nil {
			fmt.Printf("send main chain to side chain transaction, err:%s\n", err.Error())
		} else {
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
	if int64(unspent.Amount*(1e8)) < amount+fee {
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
	change := int64(unspent.Amount*1e8) - amount - fee
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
	sigHash, err := txscript.CalcSignatureHash(scriptPubkey, sigHashes, hashType, tx, 0, int64(unspent.Amount*1e8), true)
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
		if int64(unspent.Amount*1e8) > int64(s.minCCAmount*1e8*8)+s.fee {
			outAmount := (int64(unspent.Amount*1e8) - s.fee) / 4
			fmt.Printf("separate tx %s to 4 parts, which amount is %d\n", unspent.TxID, outAmount)
			_, _ = s.transferForSeparation(unspent, s.from, outAmount, 4, s.fee, s.wif.PrivKey, s.wif.SerializePubKey())
		} else if int64(unspent.Amount*1e8) > int64(s.minCCAmount*1e8*4)+s.fee /* 4x minCCAmount */ {
			outAmount := (int64(unspent.Amount*1e8) - s.fee) / 2
			fmt.Printf("separate tx %s to 2 parts, which amount is %d\n", unspent.TxID, outAmount)
			_, _ = s.transferForSeparation(unspent, s.from, outAmount, 2, s.fee, s.wif.PrivKey, s.wif.SerializePubKey())
		}
	}
}

func (s *Sender) transferForSeparation(unspent btcjson.ListUnspentResult, from bchutil.Address, amount, outputNums, fee int64, privateKey *bchec.PrivateKey, fromPubkey []byte) (*chainhash.Hash, error) {
	tx := wire.NewMsgTx(2)
	if int64(unspent.Amount*(1e8)) < (amount*outputNums + fee) {
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
	change := int64(unspent.Amount*1e8) - amount*outputNums - fee
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
	sigHash, err := txscript.CalcSignatureHash(scriptPubkey, sigHashes, hashType, tx, 0, int64(unspent.Amount*1e8), true)
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

func (s *Sender) redeemAll() {
	fmt.Println("redeem all")
	utxoInfos, err := s.smartbchClient.RedeemableUtxos(context.Background())
	if err != nil {
		panic(err)
	}
	nonce, err := s.smartbchClient.NonceAt(context.Background(), common.HexToAddress(s.sideChainReceiver), nil)
	if err != nil {
		panic(err)
	}
	count := 0
	for _, info := range utxoInfos.Infos {
		_, err := s.redeem(info.Txid.Big(), big.NewInt(0).Mul(big.NewInt(int64(info.Amount)), big.NewInt(1e10)), nonce)
		if err != nil {
			continue
		}
		nonce++
		count++
		time.Sleep(6 * time.Second)
	}
	fmt.Printf("redeem %d utxos\n", count)
}
