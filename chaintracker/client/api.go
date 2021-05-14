package client

import (
	"github.com/ethereum/go-ethereum/core/types"
)

type API interface {
	BlockNumber() (uint64, error)
	SendRawTransaction(tx *types.Transaction) error
	GetTxListByHeight(height uint64) (RpcTxs, error)
}