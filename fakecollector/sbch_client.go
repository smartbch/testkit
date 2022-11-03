package main

import (
	"context"
	"time"

	"github.com/smartbch/smartbch/rpc/client"
	"github.com/smartbch/smartbch/rpc/types"
)

const getTimeout = time.Second * 15

type SbchClient struct {
	url string
	cli *client.Client
}

func NewSbchClient(rpcUrl string) (*SbchClient, error) {
	rpcCli, err := client.Dial(rpcUrl)
	if err != nil {
		return nil, err
	}

	return &SbchClient{
		url: rpcUrl,
		cli: rpcCli,
	}, nil
}

func (s *SbchClient) GetCcInfo() (*types.CcInfo, error) {
	return s.cli.CcInfo(context.Background())
}

func (s *SbchClient) GetRedeemingUtxosForOperators() (utxos *types.UtxoInfos, err error) {
	return s.cli.RedeemingUtxosForOperators(context.Background())
}

func (s *SbchClient) GetToBeConvertedUtxosForOperators() (utxos *types.UtxoInfos, err error) {
	return s.cli.ToBeConvertedUtxosForOperators(context.Background())
}

func getOperatorPubkeys(operators []OperatorInfo) [][]byte {
	pubkeys := make([][]byte, len(operators))
	for i, operator := range operators {
		pubkeys[i] = operator.Pubkey
	}
	return pubkeys
}
func getMonitorPubkeys(monitors []MonitorInfo) [][]byte {
	pubkeys := make([][]byte, len(monitors))
	for i, monitor := range monitors {
		pubkeys[i] = monitor.Pubkey
	}
	return pubkeys
}
