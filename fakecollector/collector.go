package main

import (
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	gethcmn "github.com/ethereum/go-ethereum/common"
	sbchrpc "github.com/smartbch/smartbch/rpc/api"
	"github.com/smartbch/testkit/cctester/testcase"
)

func RunFake(sbchRpcUrl, operatorUrl string) {
	sbchClient, err := NewSbchClient(sbchRpcUrl)
	if err != nil {
		fmt.Println("failed to create smartBCH RPC client:", err.Error())
		return
	}

	for {
		handleAllPendingUTXOs(sbchClient, operatorUrl)
		time.Sleep(1 * time.Minute)
	}
}

func handleAllPendingUTXOs(sbchClient *SbchClient, operatorUrl string) {
	redeemingUtxos, err := sbchClient.GetRedeemingUtxosForOperators()
	if err != nil {
		fmt.Println("failed to get redeeming UTXOs:", err.Error())
		return
	}
	if len(redeemingUtxos) > 0 {
		for _, utxo := range redeemingUtxos {
			handleRedeemingUTXO(operatorUrl, utxo)
		}
	}

	toBeConvertedUtxos, err := sbchClient.GetToBeConvertedUtxosForOperators()
	if err != nil {
		fmt.Println("failed to get redeeming UTXOs:", err.Error())
		return
	}
	if len(toBeConvertedUtxos) > 0 {
		for _, utxo := range toBeConvertedUtxos {
			handleToBeConvertedUTXO(operatorUrl, utxo)
		}
	}
}

func handleRedeemingUTXO(operatorUrl string, utxo *sbchrpc.UtxoInfo) {
	sig, err := getSigByHash(operatorUrl, utxo.TxSigHash)
	if err != nil {
		fmt.Println("failed to get sig by hash:", err.Error())
	}
	fmt.Println("sig:", hex.EncodeToString(sig))

	testcase.BuildAndSendMainnetRedeemTx(hex.EncodeToString(utxo.Txid[:]))
}

func handleToBeConvertedUTXO(operatorUrl string, utxo *sbchrpc.UtxoInfo) {
	sig, err := getSigByHash(operatorUrl, utxo.TxSigHash)
	if err != nil {
		fmt.Println("failed to get sig by hash:", err.Error())
	}
	fmt.Println("sig:", hex.EncodeToString(sig))
	// TODO
}

func getSigByHash(operatorUrl string, txSigHash []byte) ([]byte, error) {
	fullUrl := operatorUrl + "?hash=" + hex.EncodeToString(txSigHash)
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
