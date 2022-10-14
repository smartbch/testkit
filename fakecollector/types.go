package main

import (
	gethcmn "github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
)

type OperatorInfo struct {
	Address gethcmn.Address `json:"address"`
	Pubkey  hexutil.Bytes   `json:"pubkey"`
	RpcUrl  string          `json:"rpc_url"`
	Intro   string          `json:"intro"`
}

type MonitorInfo struct {
	Address gethcmn.Address `json:"address"`
	Pubkey  hexutil.Bytes   `json:"pubkey"`
	Intro   string          `json:"intro"`
}

type CcCovenantInfo struct {
	Operators    []OperatorInfo `json:"operators"`
	Monitors     []MonitorInfo  `json:"monitors"`
	OldOperators []OperatorInfo `json:"old_operators"`
	OldMonitors  []MonitorInfo  `json:"old_monitors"`
}

type UtxoInfo struct {
	OwnerOfLost      gethcmn.Address `json:"owner_of_lost"`
	CovenantAddr     gethcmn.Address `json:"covenant_addr"`
	IsRedeemed       bool            `json:"is_redeemed"`
	RedeemTarget     gethcmn.Address `json:"redeem_target"`
	ExpectedSignTime int64           `json:"expected_sign_time"`
	Txid             gethcmn.Hash    `json:"txid"`
	Index            uint32          `json:"index"`
	Amount           hexutil.Uint64  `json:"amount"` // in satoshi
	TxSigHash        hexutil.Bytes   `json:"tx_sig_hash"`
}

type OperatorResp struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
	Result  string `json:"result,omitempty"`
}
