package main

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
)

func run(sbchRpcUrl, operatorUrl string) {
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
	fmt.Println("handleAllPendingUTXOs...")

	fmt.Println("GetRedeemingUtxosForOperators...")
	redeemingUtxos, err := sbchClient.GetRedeemingUtxosForOperators()
	if err != nil {
		fmt.Println("failed to get redeeming UTXOs:", err.Error())
		return
	}

	utxosJson, _ := json.MarshalIndent(redeemingUtxos, "", "  ")
	fmt.Println("UTXOS:", string(utxosJson))
	if len(redeemingUtxos) > 0 {
		for _, utxo := range redeemingUtxos {
			handleRedeemingUTXO(operatorUrl, utxo)
		}
	}

	fmt.Println("GetToBeConvertedUtxosForOperators...")
	toBeConvertedUtxos, err := sbchClient.GetToBeConvertedUtxosForOperators()
	if err != nil {
		fmt.Println("failed to get redeeming UTXOs:", err.Error())
		return
	}
	utxosJson, _ = json.MarshalIndent(toBeConvertedUtxos, "", "  ")
	fmt.Println("UTXOS:", string(utxosJson))
	if len(toBeConvertedUtxos) > 0 {
		for _, utxo := range toBeConvertedUtxos {
			handleToBeConvertedUTXO(operatorUrl, utxo)
		}
	}
}

func handleRedeemingUTXO(operatorUrl string, utxo *UtxoInfo) {
	fmt.Println("handleRedeemingUTXO, getSigByHash, txSigHash:", hex.EncodeToString(utxo.TxSigHash))
	sig, err := getSigByHash(operatorUrl, utxo.TxSigHash)
	if err != nil {
		fmt.Println("failed to get sig by hash:", err.Error())
	}
	fmt.Println("sig:", hex.EncodeToString(sig))

	// TODO
	//testcase.BuildAndSendMainnetRedeemTx(hex.EncodeToString(utxo.Txid[:]))
}

func handleToBeConvertedUTXO(operatorUrl string, utxo *UtxoInfo) {
	fmt.Println("handleToBeConvertedUTXO, getSigByHash, txSigHash:", hex.EncodeToString(utxo.TxSigHash))
	sig, err := getSigByHash(operatorUrl, utxo.TxSigHash)
	if err != nil {
		fmt.Println("failed to get sig by hash:", err.Error())
	}
	fmt.Println("sig:", hex.EncodeToString(sig))
	// TODO
}

func getSigByHash(operatorUrl string, txSigHash []byte) ([]byte, error) {
	fullUrl := operatorUrl + "/sig?hash=" + hex.EncodeToString(txSigHash)
	fmt.Println("getSigByHash:", fullUrl)
	resp, err := http.Get(fullUrl)
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	sig, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	return gethcmn.FromHex(string(sig)), nil
}
