package main

import (
	"context"
	"time"

	"github.com/ethereum/go-ethereum/rpc"

	sbchrpc "github.com/smartbch/smartbch/rpc/api"
)

const getTimeout = time.Second * 15

type SbchClient struct {
	url    string
	rpcCli *rpc.Client
	//ethCli *ethclient.Client
}

func NewSbchClient(rpcUrl string) (*SbchClient, error) {
	rpcCli, err := rpc.DialContext(context.Background(), rpcUrl)
	if err != nil {
		return nil, err
	}
	//ethCli := ethclient.NewClient(rpcCli)

	return &SbchClient{
		url:    rpcUrl,
		rpcCli: rpcCli,
		//ethCli: ethCli,
	}, nil
}

func (cli *SbchClient) GetCcCovenantInfo() (info sbchrpc.CcCovenantInfo, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), getTimeout)
	defer cancel()

	err = cli.rpcCli.CallContext(ctx, &info, "sbch_getCcCovenantInfo")
	return
}

func (cli *SbchClient) GetRedeemingUtxosForOperators() (utxos []*sbchrpc.UtxoInfo, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), getTimeout)
	defer cancel()

	err = cli.rpcCli.CallContext(ctx, &utxos, "sbch_getRedeemingUtxosForOperators")
	return
}

func (cli *SbchClient) GetToBeConvertedUtxosForOperators() (utxos []*sbchrpc.UtxoInfo, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), getTimeout)
	defer cancel()

	err = cli.rpcCli.CallContext(ctx, &utxos, "sbch_getToBeConvertedUtxosForOperators")
	return
}

func getOperatorPubkeys(operators []sbchrpc.OperatorInfo) [][]byte {
	pubkeys := make([][]byte, len(operators))
	for i, operator := range operators {
		pubkeys[i] = operator.Pubkey
	}
	return pubkeys
}
func getMonitorPubkeys(monitors []sbchrpc.MonitorInfo) [][]byte {
	pubkeys := make([][]byte, len(monitors))
	for i, monitor := range monitors {
		pubkeys[i] = monitor.Pubkey
	}
	return pubkeys
}
