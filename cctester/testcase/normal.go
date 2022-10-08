package testcase

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"

	"github.com/smartbch/testkit/cctester/config"
	"github.com/smartbch/testkit/cctester/utils"
)

func Test() {
	fmt.Printf("--------- Test convert -----------\n")
	TestConvert()
	time.Sleep(5 * time.Second)
	fmt.Printf("-------------- TestLostAndFoundWithBelowMinAmount -------------\n")
	TestLostAndFoundWithBelowMinAmount()
	time.Sleep(5 * time.Second)
	fmt.Printf("-------------- TestLostAndFoundWithAboveMaxAmount -------------\n")
	TestLostAndFoundWithAboveMaxAmount()
	time.Sleep(5 * time.Second)
	fmt.Printf("-------------- TestLostAndFoundWithOldCovenantAddress -------------\n")
	TestLostAndFoundWithOldCovenantAddress()
	time.Sleep(5 * time.Second)
	fmt.Printf("-------------- TestNormal -------------\n")
	TestNormal()
	fmt.Printf("-------------- TestRedeemableWithBelowMinAmount -------------\n")
	TestRedeemableWithBelowMinAmount()
	os.Exit(0)
}

func TestRedeemableWithBelowMinAmount() {
	var txid = "0x0000000000000000000000000000000000000000000000000000000000000002"
	var covenantAddress = "0x0000000000000000000000000000000000000002"
	//0xab5d62788e207646fa60eb3eebdc4358c7f5686c
	var receiver string = "0xab5d62788e207646fa60eb3eebdc4358c7f5686c"
	var amount string = "0.1"
	var amountInSideChain = uint256.NewInt(0).Mul(uint256.NewInt(1e7), uint256.NewInt(1e10))
	var normalGasFee = uint256.NewInt(0).Mul(uint256.NewInt(4000000) /*gas*/, uint256.NewInt(20000000000) /*gas price*/)
	fmt.Println(`-------------------- send cc transfer tx -------------------`)
	buildAndSendTransferTx(txid, covenantAddress, receiver, amount)
	time.Sleep(5 * time.Second)
	fmt.Println(`-------------------- send startRescan tx -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(10 * time.Second)
	balance := utils.GetAccBalance(receiver)
	fmt.Println(`-------------------- send handle utxo tx -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(5 * time.Second)
	balance1 := utils.GetAccBalance(receiver)
	receiveAmount := uint256.NewInt(0).Sub(uint256.NewInt(0).Add(balance1, normalGasFee), balance)
	fmt.Printf("balance: %s\n", receiveAmount.String())
	if !receiveAmount.Eq(amountInSideChain) {
		panic("receive amount not match")
	}
	fmt.Println(`-------------------- check utxo record from rpc -------------------`)
	utxoRecords := utils.GetRedeemingUTXOs()
	fmt.Printf("utxoRecords: len:%d\n", len(utxoRecords))
	for _, utxo := range utxoRecords {
		fmt.Printf("utxo: txid:%s\n", utxo.Txid.String())
	}
	if len(utxoRecords) != 1 {
		panic("")
	}
	if utxoRecords[0].Txid.String() != txid {
		panic("")
	}
	//fmt.Printf("utxoRecords[0].OwnerOfLost:%s\n", utxoRecords[0].OwnerOfLost.String())
	zeroAddress := common.Address{}
	if strings.ToLower(utxoRecords[0].OwnerOfLost.String()) != zeroAddress.String() {
		panic("")
	}
	fmt.Println(`--------------------- send main chain redeem tx -------------------`)
	buildAndSendMainnetRedeemTx(txid[2:])
	time.Sleep(5 * time.Second)
	fmt.Println(`--------------------- send startRescan tx second time -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(15 * time.Second)
	fmt.Println(`--------------------- send handle utxo tx second time -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(4 * time.Second)
	utxoRecords = utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 0 {
		panic("")
	}
}

func TestLostAndFoundWithAboveMaxAmount() {
	var txid = "0x0000000000000000000000000000000000000000000000000000000000000001"
	var covenantAddress = "0x0000000000000000000000000000000000000002"
	var receiver string = "0xab5d62788e207646fa60eb3eebdc4358c7f5686c"
	var amount string = "2000"
	//var amountInSatoshi = "0x2e90edd000" //2000_00000000
	//var amountInSideChain = uint256.NewInt(0).Mul(uint256.NewInt(1e7), uint256.NewInt(1e10))
	var normalGasFee = uint256.NewInt(0).Mul(uint256.NewInt(4000000) /*gas*/, uint256.NewInt(20000000000) /*gas price*/)
	fmt.Println(`-------------------- send cc transfer tx -------------------`)
	buildAndSendTransferTx(txid, covenantAddress, receiver, amount)
	time.Sleep(5 * time.Second)
	fmt.Println(`-------------------- send startRescan tx -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(10 * time.Second)
	balance := utils.GetAccBalance(receiver)
	fmt.Println(`-------------------- send handle utxo tx -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(4 * time.Second)
	balance1 := utils.GetAccBalance(receiver)
	receiveAmount := uint256.NewInt(0).Sub(uint256.NewInt(0).Add(balance1, normalGasFee), balance)
	fmt.Printf("balance: %s\n", receiveAmount.String())
	if !receiveAmount.IsZero() {
		panic("receive amount should be zero")
	}
	fmt.Println(`-------------------- send redeem tx -------------------`)
	buildAndSendRedeemTx(txid, receiver, "0")
	time.Sleep(4 * time.Second)
	balance2 := utils.GetAccBalance(receiver)
	burnAmount := uint256.NewInt(0).Sub(balance1, uint256.NewInt(0).Add(balance2, normalGasFee))
	if !burnAmount.IsZero() {
		panic("burn amount should be zero")
	}
	fmt.Println(`-------------------- check utxo record from rpc -------------------`)
	utxoRecords := utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 1 {
		panic("")
	}
	if utxoRecords[0].Txid.String() != txid {
		panic("")
	}
	fmt.Printf("utxoRecords[0].Amount:%s\n", utxoRecords[0].Amount.String())
	//if amountInSatoshi != utxoRecords[0].Amount.String() {
	//panic("")
	//}
	//fmt.Printf("utxoRecords[0].OwnerOfLost:%s\n", utxoRecords[0].OwnerOfLost.String())
	if strings.ToLower(utxoRecords[0].OwnerOfLost.String()) != receiver {
		panic("")
	}
	fmt.Println(`--------------------- send main chain redeem tx -------------------`)
	buildAndSendMainnetRedeemTx(txid[2:])
	time.Sleep(5 * time.Second)
	fmt.Println(`--------------------- send startRescan tx second time -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(10 * time.Second)
	fmt.Println(`--------------------- send handle utxo tx second time -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(5 * time.Second)
	utxoRecords = utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 0 {
		panic("")
	}
}

func TestLostAndFoundWithBelowMinAmount() {
	var txid = "0x0000000000000000000000000000000000000000000000000000000000000002"
	var covenantAddress = "0x0000000000000000000000000000000000000002"
	var receiver string = "0xab5d62788e207646fa60eb3eebdc4358c7f5686c"
	var amount string = "0.9"
	//var amountInSideChain = uint256.NewInt(0).Mul(uint256.NewInt(1e7), uint256.NewInt(1e10))
	var normalGasFee = uint256.NewInt(0).Mul(uint256.NewInt(4000000) /*gas*/, uint256.NewInt(20000000000) /*gas price*/)
	fmt.Println(`-------------------- send cc transfer tx -------------------`)
	buildAndSendTransferTx(txid, covenantAddress, receiver, amount)
	time.Sleep(5 * time.Second)
	fmt.Println(`-------------------- send startRescan tx -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(10 * time.Second)
	balance := utils.GetAccBalance(receiver)
	fmt.Println(`-------------------- send handle utxo tx -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(5 * time.Second)
	balance1 := utils.GetAccBalance(receiver)
	receiveAmount := uint256.NewInt(0).Sub(uint256.NewInt(0).Add(balance1, normalGasFee), balance)
	fmt.Printf("balance: %s\n", receiveAmount.String())
	if !receiveAmount.IsZero() {
		panic("receive amount should be zero")
	}
	fmt.Println(`-------------------- send redeem tx -------------------`)
	buildAndSendRedeemTx(txid, receiver, "0")
	time.Sleep(4 * time.Second)
	balance2 := utils.GetAccBalance(receiver)
	burnAmount := uint256.NewInt(0).Sub(balance1, uint256.NewInt(0).Add(balance2, normalGasFee))
	if !burnAmount.IsZero() {
		panic("burn amount should be zero")
	}
	fmt.Println(`-------------------- check utxo record from rpc -------------------`)
	utxoRecords := utils.GetRedeemingUTXOs()
	fmt.Printf("utxoRecords: len:%d\n", len(utxoRecords))
	for _, utxo := range utxoRecords {
		fmt.Printf("utxo: txid:%s\n", utxo.Txid.String())
	}
	if len(utxoRecords) != 1 {
		panic("")
	}
	if utxoRecords[0].Txid.String() != txid {
		panic("")
	}
	//fmt.Printf("utxoRecords[0].OwnerOfLost:%s\n", utxoRecords[0].OwnerOfLost.String())
	if strings.ToLower(utxoRecords[0].OwnerOfLost.String()) != receiver {
		panic("")
	}
	fmt.Println(`--------------------- send main chain redeem tx -------------------`)
	buildAndSendMainnetRedeemTx(txid[2:])
	time.Sleep(5 * time.Second)
	fmt.Println(`--------------------- send startRescan tx second time -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(15 * time.Second)
	fmt.Println(`--------------------- send handle utxo tx second time -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(4 * time.Second)
	utxoRecords = utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 0 {
		panic("")
	}
}

func TestLostAndFoundWithOldCovenantAddress() {
	var txid = "0x0000000000000000000000000000000000000000000000000000000000000003"
	var receiver string = "0xab5d62788e207646fa60eb3eebdc4358c7f5686c"
	var amount string = "1"
	var normalGasFee = uint256.NewInt(0).Mul(uint256.NewInt(4000000) /*gas*/, uint256.NewInt(20000000000) /*gas price*/)
	var lastCovenantAddress = "0x0000000000000000000000000000000000000001"

	fmt.Println(`-------------------- send cc transfer tx -------------------`)
	buildAndSendTransferTx(txid, lastCovenantAddress, receiver, amount)
	time.Sleep(5 * time.Second)
	fmt.Println(`-------------------- send startRescan tx -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(15 * time.Second)
	balance := utils.GetAccBalance(receiver)
	fmt.Println(`-------------------- send handle utxo tx -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(4 * time.Second)
	balance1 := utils.GetAccBalance(receiver)
	receiveAmount := uint256.NewInt(0).Sub(uint256.NewInt(0).Add(balance1, normalGasFee), balance)
	fmt.Printf("balance: %s\n", receiveAmount.String())
	if !receiveAmount.IsZero() {
		panic("receive amount should be zero")
	}
	fmt.Println(`-------------------- send redeem tx -------------------`)
	buildAndSendRedeemTx(txid, receiver, "0")
	time.Sleep(4 * time.Second)
	balance2 := utils.GetAccBalance(receiver)
	burnAmount := uint256.NewInt(0).Sub(balance1, uint256.NewInt(0).Add(balance2, normalGasFee))
	if !burnAmount.IsZero() {
		panic("burn amount should be zero")
	}
	fmt.Println(`-------------------- check utxo record from rpc -------------------`)
	utxoRecords := utils.GetRedeemingUTXOs()
	fmt.Println(len(utxoRecords))
	if len(utxoRecords) != 1 {
		panic("")
	}
	if utxoRecords[0].Txid.String() != txid {
		panic("")
	}
	//fmt.Printf("utxoRecords[0].OwnerOfLost:%s\n", utxoRecords[0].OwnerOfLost.String())
	if strings.ToLower(utxoRecords[0].OwnerOfLost.String()) != receiver {
		panic("")
	}
	fmt.Println(`--------------------- send main chain redeem tx -------------------`)
	buildAndSendMainnetRedeemTx(txid[2:])
	time.Sleep(5 * time.Second)
	fmt.Println(`--------------------- send startRescan tx second time -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(15 * time.Second)
	fmt.Println(`--------------------- send handle utxo tx second time -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(4 * time.Second)
	utxoRecords = utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 0 {
		panic("")
	}
}

func TestNormal() {
	var txid = "0x0000000000000000000000000000000000000000000000000000000000000004"
	var covenantAddress = "0x0000000000000000000000000000000000000002"
	var receiver string = "0xab5d62788e207646fa60eb3eebdc4358c7f5686c"
	var amount string = "1"
	var amountInSideChain = uint256.NewInt(0).Mul(uint256.NewInt(1e8), uint256.NewInt(1e10))
	var normalGasFee = uint256.NewInt(0).Mul(uint256.NewInt(4000000) /*gas*/, uint256.NewInt(20000000000) /*gas price*/)
	fmt.Println(`-------------------- send cc transfer tx -------------------`)
	buildAndSendTransferTx(txid, covenantAddress, receiver, amount)
	time.Sleep(5 * time.Second)
	fmt.Println(`-------------------- send startRescan tx -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(10 * time.Second)
	balance := utils.GetAccBalance(receiver)
	fmt.Println(`-------------------- send handle utxo tx -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(4 * time.Second)
	balance1 := utils.GetAccBalance(receiver)
	receiveAmount := uint256.NewInt(0).Sub(uint256.NewInt(0).Add(balance1, normalGasFee), balance)
	fmt.Printf("balance: %s\n", receiveAmount.String())
	if !receiveAmount.Eq(amountInSideChain) {
		panic("balance not match")
	}
	fmt.Println(`-------------------- send redeem tx -------------------`)
	buildAndSendRedeemTx(txid, receiver, "1000000000000000000")
	time.Sleep(6 * time.Second)
	balance2 := utils.GetAccBalance(receiver)
	burnAddress := uint256.NewInt(0).Sub(balance1, uint256.NewInt(0).Add(balance2, normalGasFee))
	if !burnAddress.Eq(amountInSideChain) {
		panic("balance not match after redeem")
	}
	fmt.Println(`-------------------- check utxo record from rpc -------------------`)
	utxoRecords := utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 1 {
		panic("")
	}
	if utxoRecords[0].Txid.String() != txid {
		panic("")
	}
	fmt.Println(`--------------------- send main chain redeem tx -------------------`)
	buildAndSendMainnetRedeemTx(txid[2:])
	time.Sleep(5 * time.Second)
	fmt.Println(`--------------------- send startRescan tx second time -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(15 * time.Second)
	fmt.Println(`--------------------- send handle utxo tx second time -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(4 * time.Second)
	utxoRecords = utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 0 {
		panic("")
	}
}

func TestConvert() {
	var txid = "0x0000000000000000000000000000000000000000000000000000000000000005"
	var newTxid = "0x0000000000000000000000000000000000000000000000000000000000000006"
	var covenantAddress = "0x0000000000000000000000000000000000000001"
	var newCovenantAddress = "0x0000000000000000000000000000000000000002"

	var receiver string = "0xab5d62788e207646fa60eb3eebdc4358c7f5686c"
	var amount string = "1"
	var amountInSideChain = uint256.NewInt(0).Mul(uint256.NewInt(1e8), uint256.NewInt(1e10))
	var newAmount string = "0.9999"
	var newAmountInSideChain = uint256.NewInt(0).Mul(uint256.NewInt(9999e4), uint256.NewInt(1e10))

	var normalGasFee = uint256.NewInt(0).Mul(uint256.NewInt(4000000) /*gas*/, uint256.NewInt(20000000000) /*gas price*/)
	fmt.Println(`-------------------- send cc transfer tx -------------------`)
	buildAndSendTransferTx(txid, covenantAddress, receiver, amount)
	time.Sleep(5 * time.Second)
	fmt.Println(`-------------------- send startRescan tx -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(10 * time.Second)
	balance := utils.GetAccBalance(receiver)
	fmt.Println(`-------------------- send handle utxo tx -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(4 * time.Second)
	balance1 := utils.GetAccBalance(receiver)
	receiveAmount := uint256.NewInt(0).Sub(uint256.NewInt(0).Add(balance1, normalGasFee), balance)
	fmt.Printf("balance: %s\n", receiveAmount.String())
	if !receiveAmount.Eq(amountInSideChain) {
		panic("balance not match")
	}
	fmt.Println(`-------------------- send startRescan tx to change covenant address -------------------`)
	latestSideChainHeight := utils.GetSideChainBlockHeight()
	for latestSideChainHeight <= 30 {
		fmt.Printf("side chain height:%d\n", latestSideChainHeight)
		time.Sleep(5 * time.Second)
		latestSideChainHeight = utils.GetSideChainBlockHeight()
	}
	buildAndSendStartRescanTx()
	time.Sleep(6 * time.Second)
	fmt.Println(`-------------------- check utxo record from rpc -------------------`)
	utxoRecords := utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 0 {
		panic("")
	}
	toBeConvertedUtxoRecords := utils.GetToBeConvertedUTXOs()
	if len(toBeConvertedUtxoRecords) != 1 {
		panic("")
	}
	if toBeConvertedUtxoRecords[0].Txid.String() != txid {
		panic("")
	}
	fmt.Println(`--------------------- send main chain convert tx -------------------`)
	buildAndSendConvertTx(txid[2:], newTxid[2:], newCovenantAddress, newAmount)
	time.Sleep(5 * time.Second)
	fmt.Println(`--------------------- send startRescan tx second time -------------------`)
	latestSideChainHeight = utils.GetSideChainBlockHeight()
	for latestSideChainHeight <= 60 {
		fmt.Printf("side chain height:%d\n", latestSideChainHeight)
		time.Sleep(5 * time.Second)
		latestSideChainHeight = utils.GetSideChainBlockHeight()
	}
	buildAndSendStartRescanTx()
	time.Sleep(15 * time.Second)
	fmt.Println(`--------------------- send handle utxo tx second time -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(4 * time.Second)
	utxoRecords = utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 0 {
		panic("")
	}
	fmt.Println(`-------------------- send redeem tx second time -------------------`)
	balance2 := utils.GetAccBalance(receiver)
	buildAndSendRedeemTx(newTxid, receiver, "999900000000000000")
	time.Sleep(4 * time.Second)
	balance3 := utils.GetAccBalance(receiver)
	burnAddress := uint256.NewInt(0).Sub(balance2, uint256.NewInt(0).Add(balance3, normalGasFee))
	fmt.Println(burnAddress.String())
	fmt.Println(newAmountInSideChain.String())
	if !burnAddress.Eq(newAmountInSideChain) {
		panic("balance not match after redeem")
	}
	fmt.Println(`-------------------- check utxo record from rpc second time -------------------`)
	utxoRecords = utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 1 {
		panic("")
	}
	if utxoRecords[0].Txid.String() != newTxid {
		panic("")
	}
	if utxoRecords[0].CovenantAddr.String() != newCovenantAddress {
		panic("")
	}
	fmt.Println(`--------------------- send main chain redeem tx -------------------`)
	buildAndSendMainnetRedeemTx(newTxid[2:])
	time.Sleep(5 * time.Second)
	fmt.Println(`--------------------- send startRescan tx second time -------------------`)
	buildAndSendStartRescanTx()
	time.Sleep(15 * time.Second)
	fmt.Println(`--------------------- send handle utxo tx second time -------------------`)
	buildAndSendHandleUTXOTx()
	time.Sleep(4 * time.Second)
	utxoRecords = utils.GetRedeemingUTXOs()
	if len(utxoRecords) != 0 {
		panic("")
	}
}

func buildAndSendTransferTx(txid, covenantAddress, receiver, amount string) {
	out := utils.Execute(config.TxMakerPath, "make-cc-utxo",
		fmt.Sprintf("--txid=%s", txid),
		fmt.Sprintf("--cc-covenant-addr=%s", covenantAddress),
		fmt.Sprintf("--amt=%s", amount),
		fmt.Sprintf("--op-return=%s", receiver))
	//fmt.Printf(out)
	utils.SendCcTxToFakeNode(out)
}

func buildAndSendConvertTx(inTxid, txid, covenantAddress, amount string) {
	out := utils.Execute(config.TxMakerPath, "convert-by-operators",
		fmt.Sprintf("--txid=%s", txid),
		fmt.Sprintf("--in-txid=%s", inTxid),
		"--in-vout=0",
		fmt.Sprintf("--cc-covenant-addr=%s", covenantAddress),
		fmt.Sprintf("--amt=%s", amount))
	//fmt.Printf(out)
	utils.SendCcTxToFakeNode(out)
}

func buildAndSendMainnetRedeemTx(txid string) {
	out := utils.Execute(config.TxMakerPath, "redeem-cc-utxo", fmt.Sprintf("--in-txid=%s", txid), fmt.Sprintf("--txid=%s", txid), "--in-vout=0")
	fmt.Println(out)
	utils.SendCcTxToFakeNode(out)
}

func buildAndSendStartRescanTx() {
	height := utils.GetLatestBlockHeight()
	fmt.Println(height)
	utils.StartRescan(height)
}

func buildAndSendHandleUTXOTx() {
	utils.HandleCCUTXOs()
}

func buildAndSendRedeemTx(txid, receiver, amount string) {
	utils.Redeem(txid, receiver, amount)
}
