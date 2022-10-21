package main

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/smartbch/testkit/cctester/testcase"
)

var redeemUtxoCache map[[32]byte]bool
var convertUtxoCache map[[32]byte]bool

func run(sbchRpcUrl, operatorUrl string) {
	redeemUtxoCache = make(map[[32]byte]bool)
	convertUtxoCache = make(map[[32]byte]bool)
	sbchClient, err := NewSbchClient(sbchRpcUrl)
	if err != nil {
		fmt.Println("failed to create smartBCH RPC client:", err.Error())
		return
	}
	for {
		handleAllPendingUTXOs(sbchClient, operatorUrl)
		time.Sleep(1 * time.Second)
	}
}

func handleAllPendingUTXOs(sbchClient *SbchClient, operatorUrl string) {
	//fmt.Println("handleAllPendingUTXOs...")

	//fmt.Println("GetRedeemingUtxosForOperators...")
	redeemingUtxos, err := sbchClient.GetRedeemingUtxosForOperators()
	if err != nil {
		fmt.Println("failed to get redeeming UTXOs:", err.Error())
		return
	}
	//utxosJson, _ := json.MarshalIndent(redeemingUtxos, "", "  ")
	//fmt.Println("UTXOS:", string(utxosJson))
	if len(redeemingUtxos) > 0 {
		for _, utxo := range redeemingUtxos {
			handleRedeemingUTXO(operatorUrl, utxo)
		}
	}
	//fmt.Println("GetToBeConvertedUtxosForOperators...")
	toBeConvertedUtxos, err := sbchClient.GetToBeConvertedUtxosForOperators()
	if err != nil {
		fmt.Println("failed to get redeeming UTXOs:", err.Error())
		return
	}
	//utxosJson, _ = json.MarshalIndent(toBeConvertedUtxos, "", "  ")
	//fmt.Println("UTXOS:", string(utxosJson))
	if len(toBeConvertedUtxos) > 0 {
		ccInfo, err := sbchClient.GetCcCovenantInfo()
		if err != nil {
			fmt.Println("failed to get ccCovenantInfo:", err.Error())
		}
		for _, utxo := range toBeConvertedUtxos {
			handleToBeConvertedUTXO(ccInfo, operatorUrl, utxo)
		}
	}
}

func handleRedeemingUTXO(operatorUrl string, utxo *UtxoInfo) {
	out, err := json.Marshal(utxo)
	hash := sha256.Sum256(out)
	if _, exist := redeemUtxoCache[hash]; exist {
		//fmt.Println("already exist")
		return
	}
	sig, err := getSigByHash(operatorUrl, utxo.TxSigHash)
	for err != nil {
		fmt.Println("failed to get sig by hash, retry:", err.Error())
		time.Sleep(2 * time.Second)
		sig, err = getSigByHash(operatorUrl, utxo.TxSigHash)
	}
	fmt.Printf("handleRedeemingUTXO, txid:%s, txSigHash:%s, sig:%s\n", utxo.Txid.String(), utxo.TxSigHash.String(), hex.EncodeToString(sig))
	testcase.BuildAndSendMainnetRedeemTx(hex.EncodeToString(utxo.Txid[:]))
	redeemUtxoCache[hash] = true
}

func handleToBeConvertedUTXO(info CcCovenantInfo, operatorUrl string, utxo *UtxoInfo) {
	out, err := json.Marshal(utxo)
	hash := sha256.Sum256(out)
	if _, exist := convertUtxoCache[hash]; exist {
		//fmt.Println("already exist")
		return
	}
	sig, err := getSigByHash(operatorUrl, utxo.TxSigHash)
	for err != nil {
		fmt.Println("failed to get sig by hash, retry:", err.Error())
		time.Sleep(2 * time.Second)
		sig, err = getSigByHash(operatorUrl, utxo.TxSigHash)
	}
	txidB := [32]byte{}
	rand.Read(txidB[:])
	txid := gethcmn.Hash(txidB)
	fmt.Printf("handleRedeemingUTXO, txid:%s, txSigHash:%s, sig:%s, inTxid:%s\n", txid.String(), utxo.TxSigHash.String(), hex.EncodeToString(sig), utxo.Txid.String())
	testcase.BuildAndSendMainnetConvertTx(utxo.Txid.String(), txid.String(), info.CurrCovenantAddress, "0.9999" /*hard code*/)
	convertUtxoCache[hash] = true
}

func getSigByHash(operatorUrl string, txSigHash []byte) ([]byte, error) {
	fullUrl := operatorUrl + "/sig?hash=" + hex.EncodeToString(txSigHash)
	fmt.Println("getSigByHash:", fullUrl)
	resp, err := http.Get(fullUrl)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var respJson OperatorResp
	err = json.Unmarshal(respBytes, &respJson)
	if err != nil {
		return nil, err
	}
	if respJson.Error != "" {
		return nil, errors.New(respJson.Error)
	}

	return gethcmn.FromHex(respJson.Result), nil
}
