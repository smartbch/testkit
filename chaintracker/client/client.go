package client

import (
	"context"
	"github.com/ethereum/go-ethereum/common/hexutil"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"
)

var _ API = Client{}

type Client struct {
	ethClient *ethclient.Client
	Rpc       *rpc.Client
}

func (c Client) BlockNumber() (uint64, error) {
	return c.ethClient.BlockNumber(context.Background())
}

func (c Client) SendRawTransaction(tx *types.Transaction) error {
	return c.ethClient.SendTransaction(context.Background(), tx)
}

func (c Client) GetTxListByHeight(height uint64) (RpcTxs, error) {
	var r RpcTxs
	err := c.Rpc.CallContext(context.Background(), &r, "sbch_getTxListByHeight", hexutil.Uint64(height))
	if err != nil {
		return nil, err
	}
	return r, err
}

func (c Client) Close() {
	c.ethClient.Close()
}

func New(url string) (*Client, error) {
	r, err := rpc.Dial(url)
	if err != nil {
		return nil, err
	}
	c := &Client{
		Rpc:       r,
		ethClient: ethclient.NewClient(r),
	}
	return c, nil
}
