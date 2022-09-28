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
	fmt.Printf("-------------- TestLostAndFoundWithBelowMinAMount -------------\n")
	TestLostAndFoundWithBelowMinAMount()
	time.Sleep(5 * time.Second)
	fmt.Printf("-------------- TestLostAndFoundWithAboveMaxAMount -------------\n")
	TestLostAndFoundWithAboveMaxAMount()
	time.Sleep(5 * time.Second)
	fmt.Printf("-------------- TestLostAndFoundWithOldCovenantAddress -------------\n")
	TestLostAndFoundWithOldCovenantAddress()
	time.Sleep(5 * time.Second)
	fmt.Printf("-------------- TestNormal -------------\n")
	TestNormal()
	fmt.Printf("-------------- TestRedeemableWithBelowMinAMount -------------\n")
	TestRedeemableWithBelowMinAMount()
	os.Exit(0)
}

func TestRedeemableWithBelowMinAMount() {
	var txid = "0x0000000000000000000000000000000000000000000000000000000000000002"
	var covenantAddress = "0x0000000000000000000000000000000000000001"
	var receiver string = "0x09f236e4067f5fca5872d0c09f92ce653377ae41"
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

func TestLostAndFoundWithAboveMaxAMount() {
	var txid = "0x0000000000000000000000000000000000000000000000000000000000000001"
	var covenantAddress = "0x0000000000000000000000000000000000000001"
	var receiver string = "0x09f236e4067f5fca5872d0c09f92ce653377ae41"
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

func TestLostAndFoundWithBelowMinAMount() {
	var txid = "0x0000000000000000000000000000000000000000000000000000000000000002"
	var covenantAddress = "0x0000000000000000000000000000000000000001"
	var receiver string = "0x09f236e4067f5fca5872d0c09f92ce653377ae41"
	var amount string = "0.1"
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
	var receiver string = "0x09f236e4067f5fca5872d0c09f92ce653377ae41"
	var amount string = "1"
	var normalGasFee = uint256.NewInt(0).Mul(uint256.NewInt(4000000) /*gas*/, uint256.NewInt(20000000000) /*gas price*/)
	var lastCovenantAddress = "0x0000000000000000000000000000000000000002"

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
	var covenantAddress = "0x0000000000000000000000000000000000000001"
	var receiver string = "0x09f236e4067f5fca5872d0c09f92ce653377ae41"
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

func buildAndSendTransferTx(txid, covenantAddress, receiver, amount string) {
	out := utils.Execute(config.TxMakerPath, "make-cc-utxo", fmt.Sprintf("--txid=%s", txid),
		fmt.Sprintf("--cc-covenant-addr=%s", covenantAddress),
		fmt.Sprintf("--amt=%s", amount),
		fmt.Sprintf("--op-return=%s", receiver))
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
