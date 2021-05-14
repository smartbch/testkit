package client

import (
	"encoding/json"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
)

type RpcTxs []Transaction

func (t *Transaction) PrintJson() ([]byte, error) {
	return json.Marshal(t)
}

type Transaction struct {
	BlockHash        *common.Hash    `json:"blockHash"`
	BlockNumber      *hexutil.Big    `json:"blockNumber"`
	From             common.Address  `json:"from"`
	Gas              hexutil.Uint64  `json:"gas"`
	GasPrice         *hexutil.Big    `json:"gasPrice"`
	Hash             common.Hash     `json:"hash"`
	Input            hexutil.Bytes   `json:"input"`
	Nonce            hexutil.Uint64  `json:"nonce"`
	To               *common.Address `json:"to"`
	TransactionIndex *hexutil.Uint64 `json:"transactionIndex"`
	Value            *hexutil.Big    `json:"value"`
	V                *hexutil.Big    `json:"v"`
	R                *hexutil.Big    `json:"r"`
	S                *hexutil.Big    `json:"s"`
}

func ConvertTx(tx Transaction) *types.Transaction {
	t := &types.LegacyTx{
		Nonce:    uint64(tx.Nonce),
		GasPrice: tx.GasPrice.ToInt(),
		Gas:      uint64(tx.Gas),
		To:       tx.To,
		Value:    tx.Value.ToInt(),
		Data:     tx.Input,
		//V:        tx.V.ToInt(),
		//R:        tx.R.ToInt(),
		//S:        tx.S.ToInt(),
	}
	return types.NewTx(t)
}
