package main

import (
	"context"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/rpc"

	sbchrpc "github.com/smartbch/smartbch/rpc/api"
)

type SbchClient struct {
	rpcCli *rpc.Client
	ethCli *ethclient.Client
}

func newSbchClient(url string) *SbchClient {
	rpcCli, err := rpc.DialContext(context.Background(), url)
	if err != nil {
		panic(err)
	}

	ethCli := ethclient.NewClient(rpcCli)
	return &SbchClient{
		rpcCli: rpcCli,
		ethCli: ethCli,
	}
}

func (cli *SbchClient) sbchCall(msg ethereum.CallMsg, blockNumber *big.Int) (*sbchrpc.CallDetail, error) {
	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()

	var callDetail sbchrpc.CallDetail
	err := cli.rpcCli.CallContext(ctx, &callDetail, "sbch_call", toCallArg(msg), toBlockNumArg(blockNumber))
	if err != nil {
		return nil, err
	}
	return &callDetail, nil
}

func toBlockNumArg(number *big.Int) string {
	if number == nil {
		return "latest"
	}
	pending := big.NewInt(-1)
	if number.Cmp(pending) == 0 {
		return "pending"
	}
	return hexutil.EncodeBig(number)
}

func toCallArg(msg ethereum.CallMsg) interface{} {
	arg := map[string]interface{}{
		"from": msg.From,
		"to":   msg.To,
	}
	if len(msg.Data) > 0 {
		arg["data"] = hexutil.Bytes(msg.Data)
	}
	if msg.Value != nil {
		arg["value"] = (*hexutil.Big)(msg.Value)
	}
	if msg.Gas != 0 {
		arg["gas"] = hexutil.Uint64(msg.Gas)
	}
	if msg.GasPrice != nil {
		arg["gasPrice"] = (*hexutil.Big)(msg.GasPrice)
	}
	return arg
}
