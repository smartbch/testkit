package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/holiman/uint256"

	"github.com/smartbch/testkit/cctester/config"
)

func ExecuteWithContinuousOutPut(exe string, params ...string) {
	cmd := exec.Command(exe, params...)
	stdout, err := cmd.StdoutPipe()
	cmd.Stderr = cmd.Stdout
	if err != nil {
		fmt.Println(err.Error())
		//panic(err)
	}
	if err = cmd.Start(); err != nil {
		fmt.Println(err.Error())
		//panic(err)
	}

	_, exeFile := path.Split(exe)
	for {
		tmp := make([]byte, 1024)
		_, err := stdout.Read(tmp)
		//_out := string(tmp)
		_out := strings.ReplaceAll(string(tmp), "\n", "\n"+exeFile+": ")
		fmt.Print(_out)
		if err != nil {
			break
		}
	}
}

func Execute(exe string, params ...string) string {
	cmd := exec.Command(exe, params...)
	out, err := cmd.Output()
	if err != nil {
		panic(err.Error())
	}
	return string(out)
}

func ExecuteGovScript(exe string, params ...string) string {
	cmd := exec.Command(exe, params...)
	cmd.Dir = config.CcContractsPath
	cmd.Env = append(cmd.Env, "HARDHAT_NETWORK=sbch_local")
	cmd.Env = append(cmd.Env, "KEY=0xa3ff378a8d766931575df674fbb1024f09f7072653e1aa91641f310b3e1c5275")
	out, err := cmd.Output()
	if err != nil {
		panic(err.Error())
	}
	return string(out)
}

func SendCcTxToFakeNode(tx string) {
	tx = strings.ReplaceAll(tx, "\"", "\\\"")
	data := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"method\":\"cc\",\"params\":[\"%s\"],\"id\":1}", tx)
	fmt.Println(data)
	args := []string{"-X", "POST", "--data", data, "-H", "Content-Type: application/json", "http://127.0.0.1:1234", "-v"}
	ExecuteWithContinuousOutPut("curl", args...)
}

func SendMonitorVoteToFakeNode(monitorPubkey string) {
	data := fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"method\":\"monitor\",\"params\":[\"%s\"],\"id\":1}", monitorPubkey)
	fmt.Println(data)
	args := []string{"-X", "POST", "--data", data, "-H", "Content-Type: application/json", "http://127.0.0.1:1234", "-v"}
	ExecuteWithContinuousOutPut("curl", args...)
}

func StartRescan(mainHeight string) {
	args := []string{"exec", "scripts/startrescan.js", "--network=sbch_local", mainHeight}
	ExecuteWithContinuousOutPut("truffle", args...)
}

func HandleCCUTXOs() {
	args := []string{"exec", "scripts/handleutxo.js", "--network=sbch_local"}
	ExecuteWithContinuousOutPut("truffle", args...)
}

func Redeem(txid, receiver, amount string) {
	args := []string{"exec", "scripts/redeem.js", "--network=sbch_local", txid, "0", receiver, amount}
	ExecuteWithContinuousOutPut("truffle", args...)
}

type UtxoInfo struct {
	OwnerOfLost      common.Address `json:"owner_of_lost"`
	CovenantAddr     common.Address `json:"covenant_addr"`
	IsRedeemed       bool           `json:"is_redeemed"`
	RedeemTarget     common.Address `json:"redeem_target"`
	ExpectedSignTime int64          `json:"expected_sign_time"`
	Txid             common.Hash    `json:"txid"`
	Index            uint32         `json:"index"`
	Amount           hexutil.Uint64 `json:"amount"` // in satoshi
	TxSigHash        hexutil.Bytes  `json:"tx_sig_hash"`
}

func GetRedeemingUTXOs() []*UtxoInfo {
	args := []string{"-X", "POST", "--data", "{\"jsonrpc\":\"2.0\",\"method\":\"sbch_getRedeemingUtxosForMonitors\",\"params\":[],\"id\":1}", "-H", "Content-Type: application/json", "http://127.0.0.1:8545"}
	out := Execute("curl", args...)
	//fmt.Println(out)
	type serverResponse struct {
		Result []*UtxoInfo      `json:"result"`
		Error  interface{}      `json:"error"`
		Id     *json.RawMessage `json:"id"`
	}
	var res serverResponse
	fmt.Println(out)
	err := json.Unmarshal([]byte(out), &res)
	if err != nil {
		panic(err)
	}
	if res.Error != nil {
		panic(res.Error)
	}
	return res.Result
}

func GetRedeemableUTXOs() []*UtxoInfo {
	args := []string{"-X", "POST", "--data", "{\"jsonrpc\":\"2.0\",\"method\":\"sbch_getRedeemableUtxos\",\"params\":[],\"id\":1}", "-H", "Content-Type: application/json", "http://127.0.0.1:8545"}
	out := Execute("curl", args...)
	//fmt.Println(out)
	type serverResponse struct {
		Result []*UtxoInfo      `json:"result"`
		Error  interface{}      `json:"error"`
		Id     *json.RawMessage `json:"id"`
	}
	var res serverResponse
	fmt.Println(out)
	err := json.Unmarshal([]byte(out), &res)
	if err != nil {
		panic(err)
	}
	if res.Error != nil {
		panic(res.Error)
	}
	return res.Result
}

func GetToBeConvertedUTXOs() []*UtxoInfo {
	args := []string{"-X", "POST", "--data", "{\"jsonrpc\":\"2.0\",\"method\":\"sbch_getToBeConvertedUtxosForMonitors\",\"params\":[],\"id\":1}", "-H", "Content-Type: application/json", "http://127.0.0.1:8545"}
	out := Execute("curl", args...)
	//fmt.Println(out)
	type serverResponse struct {
		Result []*UtxoInfo      `json:"result"`
		Error  interface{}      `json:"error"`
		Id     *json.RawMessage `json:"id"`
	}
	var res serverResponse
	fmt.Println(out)
	err := json.Unmarshal([]byte(out), &res)
	if err != nil {
		panic(err)
	}
	if res.Error != nil {
		panic(res.Error)
	}
	return res.Result
}

func GetAccBalance(address string) *uint256.Int {
	args := []string{"-X", "POST", "--data", fmt.Sprintf("{\"jsonrpc\":\"2.0\",\"method\":\"eth_getBalance\",\"params\":[\"%s\",\"latest\"],\"id\":1}", address), "-H", "Content-Type: application/json", "http://127.0.0.1:8545"}
	out := Execute("curl", args...)
	//fmt.Println(out)
	type serverResponse struct {
		Result string           `json:"result"`
		Error  interface{}      `json:"error"`
		Id     *json.RawMessage `json:"id"`
	}
	var res serverResponse
	err := json.Unmarshal([]byte(out), &res)
	if err != nil {
		panic(err)
	}
	balance, err := uint256.FromHex(res.Result)
	if err != nil {
		panic(err)
	}
	return balance
}

func GetSideChainBlockHeight() uint64 {
	args := []string{"-X", "POST", "--data", "{\"jsonrpc\":\"2.0\",\"method\":\"eth_blockNumber\",\"params\":[],\"id\":1}", "-H", "Content-Type: application/json", "http://127.0.0.1:8545"}
	out := Execute("curl", args...)
	fmt.Println(out)
	type serverResponse struct {
		Result string           `json:"result"`
		Error  interface{}      `json:"error"`
		Id     *json.RawMessage `json:"id"`
	}
	var res serverResponse
	err := json.Unmarshal([]byte(out), &res)
	if err != nil {
		panic(err)
	}
	balance, err := hexutil.DecodeUint64(res.Result)
	if err != nil {
		panic(err)
	}
	return balance
}

func GetLatestBlockHeight() string {
	args := []string{"-X", "POST", "--data", "{\"jsonrpc\":\"2.0\",\"method\":\"getblockcount\",\"params\":[],\"id\":1}", "-H", "Content-Type: application/json", "http://127.0.0.1:1234", "-v"}
	out := Execute("curl", args...)
	//fmt.Println(out)
	type serverResponse struct {
		Result float64          `json:"result"`
		Error  interface{}      `json:"error"`
		Id     *json.RawMessage `json:"id"`
	}
	var res serverResponse
	err := json.Unmarshal([]byte(out), &res)
	if err != nil {
		panic("not get the block height")
	}
	return fmt.Sprintf("%d", int64(res.Result))
}

func StartSideChainNode() {
	sideNodeParams := []string{
		"start",
		"--home", os.ExpandEnv("$HOME/.smartbchd"),
		"--unlock", "0xe3d9be2e6430a9db8291ab1853f5ec2467822b33a1a08825a22fab1425d2bff9",
		"--https.addr=off",
		"--wss.addr=off",
		"--http.api=eth,web3,net,txpool,sbch,debug",
		"--log_level=json-rpc:debug,watcher:debug,app:debug",
		"--skip-sanity-check=true",
		"--with-syncdb=false",
	}
	ExecuteWithContinuousOutPut(config.SideNodePath, sideNodeParams...)
}

func DeployGovContracts() string {
	output := ExecuteGovScript("node",
		"scripts/int_test_tool.js",
		"deploy-gov-contracts")
	fmt.Println(output)

	// find nodesGovAddr
	idx := strings.Index(output, "CCNodesGov deployed to: ")
	if idx < 0 {
		panic(output)
	}
	nodesGovAddr := output[idx+24 : idx+24+42]
	fmt.Println("nodesGovAddr:", nodesGovAddr)
	return nodesGovAddr
}

func InitSbchNodesGov(govAddr string) {
	output := ExecuteGovScript("node",
		"scripts/int_test_tool.js",
		"add-sbchd-node",
		"--gov="+govAddr,
		"--rpc-url=http://127.0.0.1:8545",
		"--cert-url='http://127.0.0.1/cert'", // not used
		"--cert-hash=0xd86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994", // not used
	)
	fmt.Println(output)

	output = ExecuteGovScript("node",
		"scripts/int_test_tool.js",
		"add-sbchd-node",
		"--gov="+govAddr,
		"--rpc-url=http://127.0.0.1:8545",
		"--cert-url='http://127.0.0.1/cert'", // not used
		"--cert-hash=0xd86b49e3424e557beebf67bd06842cdb88e314c44887f3f265b7f81107dd6994", // not used
	)
	fmt.Println(output)
}

func StartOperators(nodesGovAddr string) {
	ExecuteWithContinuousOutPut(config.OperatorPath,
		"--listenAddr=0.0.0.0:8801",
		"--bootstrapRpcURL=http://localhost:8545",
		"--nodesGovAddr="+nodesGovAddr,
	)
}

func StartFakeCollector() {
	ExecuteWithContinuousOutPut(config.CollectorPath)
}
