/*
File Usage:

This tool is used to test CheckTx(), good behavior should include:
	1. same nonce and smaller nonce should not in pool and block
	2. tx with same nonce should not in next block although that tx hasn't been executed yet.
	2. can send tx into pool with continuous nonce, not support nonce disorder or nonce hole;
	3. not care if there already has a pending tx in block

Test Method:
	1. run main, to test different nonce tx in different block,
       for example: tx_with_nonce_1 in block_1, tx_with_nonce2 in block_2
	2. change delayTime to 500 to test continuous nonce tx in one block and also in continuous blocks
	3. change delayTime to 50 to test continuous nonce tx in only one block
*/

package main

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"

	"github.com/smartbch/testkit/chaintracker/client"
	"github.com/smartbch/testkit/ethutils"
)

var url = "http://localhost:8545"
var delayTime = 2000

func MustEncodeTx(tx *types.Transaction) []byte {
	if data, err := ethutils.EncodeTx(tx); err == nil {
		return data
	} else {
		panic(err)
	}
}

func MustHexToPrivKey(key string) *ecdsa.PrivateKey {
	if k, _, err := ethutils.HexToPrivKey(key); err == nil {
		return k
	} else {
		panic(err)
	}
}

func MustSignTx(tx *types.Transaction, chainID *big.Int, privKey string) *types.Transaction {
	key := MustHexToPrivKey(privKey)
	if tx, err := ethutils.SignTx(tx, chainID, key); err == nil {
		return tx
	} else {
		panic(err)
	}
}

func GetTx(toAddr *common.Address, value int64, nonce uint64, privKey string) []byte {
	txData := &types.LegacyTx{
		Nonce:    nonce,
		GasPrice: big.NewInt(150_000_000_000),
		Gas:      100000,
		To:       toAddr,
		Value:    big.NewInt(value),
	}
	tx := types.NewTx(txData)
	tx = MustSignTx(tx, big.NewInt(0x2711), privKey)
	return MustEncodeTx(tx)
}

func KeyToAddr(keyStr string) common.Address {
	key, err := crypto.HexToECDSA(keyStr)
	if err != nil {
		panic(err)
	}
	return crypto.PubkeyToAddress(key.PublicKey)
}

func repeatNonce(repeatCount int64, nonceStart, nonceEnd uint64, toAddrList []common.Address, privKeys []string) {
	if len(toAddrList) != len(privKeys) {
		panic("length mismatch")
	}
	fromAddrs := make([]common.Address, len(privKeys))
	for i, key := range privKeys {
		fromAddrs[i] = KeyToAddr(key)
	}
	c, err := client.New(url)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	for nonce := nonceStart; nonce < nonceEnd; nonce++ {
		time.Sleep(time.Duration(delayTime) * time.Millisecond))
		fmt.Printf("=============== nonce %d begin ==============\n", nonce)
		for i := int64(1); i <= repeatCount; i++ {
			for j, toAddr := range toAddrList {
				tx := GetTx(&toAddr, i, nonce, privKeys[j])
				err = c.Rpc.CallContext(context.Background(), nil, "eth_sendRawTransaction", hexutil.Encode(tx))
				if err != nil {
					fmt.Printf("TestRepeatNonceE(address:%s-nonce:%d-index:%d): %s\n", fromAddrs[j], nonce, i, err.Error())
				} else {
					fmt.Printf("TestRepeatNonceOk(address:%s-nonce:%d-index:%d): %s\n", fromAddrs[j], nonce, i, "success")
				}
				if i == repeatCount && nonce != 0 {
					//test smaller nonce tx should not in pool
					tx := GetTx(&toAddr, i, nonce-1, privKeys[j])
					err = c.Rpc.CallContext(context.Background(), nil, "eth_sendRawTransaction", hexutil.Encode(tx))
					if err != nil {
						fmt.Printf("TestSmallerNonceE(address:%s-nonce:%d-index:%d): %s\n", fromAddrs[j], nonce, i, err.Error())
						continue
					} else {
						fmt.Printf("TestSmallerNonceOk(address:%s-nonce:%d-index:%d): %s\n", fromAddrs[j], nonce, i, "success")
					}
				}
			}
		}
		fmt.Printf("=============== nonce %d end ==============\n", nonce)
	}
	fmt.Printf("=============== test same nonce tx in continuous block ==============\n")
	for i := int64(1); i <= 10; i++ {
		time.Sleep(400 * time.Millisecond)
		tx := GetTx(&toAddrList[0], i+20, nonceEnd-1, privKeys[0])
		err = c.Rpc.CallContext(context.Background(), nil, "eth_sendRawTransaction", hexutil.Encode(tx))
		if err != nil {
			fmt.Printf("TestSmallerNonceE(address:%s-nonce:%d-index:%d): %s\n", fromAddrs[0], nonceEnd-1, i, err.Error())
			continue
		} else {
			fmt.Printf("TestSmallerNonceOk(address:%s-nonce:%d-index:%d): %s\n", fromAddrs[0], nonceEnd-1, i, "success")
		}
	}
	fmt.Printf("=============== test same nonce tx in continuous block end==============\n")
}

var toKeys = []string{
	"48d540e4ab73c814edd3d3a7509a70589dc3f0254866342c42d96b66ae235cd8",
	//"fce1bfe924327cd8f8afa6903ec25038c4532c604b34b4e13882e0ff41f045c6",
	//"e36b2e4af6cea8309b82c4ab9140a1ea6a7c3f1b0ec645e872685b5d728c5ffc",
	//"ba7deb8e1ad8fe676a7a2a5a91350d4f53514f10f4596ddcfe55712019777d26",
	//"b3f2bdbd8d1b9a4e29de3091cae54ca64f2d896b45f42e964004c3a7e8464976",
	//"7d359c3ada40c3b5d21e46787346901cb793efdfff2d2d263a291bfcc70905e0",
	//"00ced8b432ab7a85c03225fb69f2d9c2d4f16644ca5e8a3594b6edb3cb8bff41",
	//"4bd2afcd3b67bb6c5b84065e03633c053c636c5033b0bf83c6d616922ba029c5",
	//"1cd5f8f98d4475794a0f8034f0ba02a11a7015c95ea9641d9d94eb8b44d4ccc8",
	//"56a5efb09e9135e21e0bb8ff07cd6fb9f4d5ff528f92f8f60a2ab2a977c19821",
}

//be same with script restart_from_h0.sh in smartbch repo
var fromKeys = []string{
	"e3d9be2e6430a9db8291ab1853f5ec2467822b33a1a08825a22fab1425d2bff9",
	//"5a09e9d6be2cdc7de8f6beba300e52823493cd23357b1ca14a9c36764d600f5e",
	//"7e01af236f9c9536d9d28b07cea24ccf21e21c9bc9f2b2c11471cd82dbb63162",
	//"1f67c31733dc3fd02c1f9ce9cb9e05b1d2f1b7b5463fef8acf6cf17f3bd01467",
	//"8aa75c97b22e743e2d14a0472406f03cc5b4a050e8d4300040002096f50c0c6f",
	//"84a453fe127ae889de1cfc28590bf5168d2843b50853ab3c5080cd5cf9e18b4b",
	//"40580320383dbedba7a5305a593ee2c46581a4fd56ff357204c3894e91fbaf48",
	//"0e3e6ba041d8ad56b0825c549b610e447ec55a72bb90762d281956c56146c4b3",
	//"867b73f28bea9a0c83dfc233b8c4e51e0d58197de7482ebf666e40dd7947e2b6",
	//"a3ff378a8d766931575df674fbb1024f09f7072653e1aa91641f310b3e1c5275",
}

/*

./smartbchd init mynode --chain-id 0x2711 \
  --init-balance=10000000000000000000 \
  --test-keys="\
0xe3d9be2e6430a9db8291ab1853f5ec2467822b33a1a08825a22fab1425d2bff9,\
0x5a09e9d6be2cdc7de8f6beba300e52823493cd23357b1ca14a9c36764d600f5e,\
0x7e01af236f9c9536d9d28b07cea24ccf21e21c9bc9f2b2c11471cd82dbb63162,\
0x1f67c31733dc3fd02c1f9ce9cb9e05b1d2f1b7b5463fef8acf6cf17f3bd01467,\
0x8aa75c97b22e743e2d14a0472406f03cc5b4a050e8d4300040002096f50c0c6f,\
0x84a453fe127ae889de1cfc28590bf5168d2843b50853ab3c5080cd5cf9e18b4b,\
0x40580320383dbedba7a5305a593ee2c46581a4fd56ff357204c3894e91fbaf48,\
0x0e3e6ba041d8ad56b0825c549b610e447ec55a72bb90762d281956c56146c4b3,\
0x867b73f28bea9a0c83dfc233b8c4e51e0d58197de7482ebf666e40dd7947e2b6,\
0xa3ff378a8d766931575df674fbb1024f09f7072653e1aa91641f310b3e1c5275"

*/
func main() {
	toAddrs := make([]common.Address, len(toKeys))
	for i, key := range toKeys {
		toAddrs[i] = KeyToAddr(key)
	}
	repeatNonce(2, 0, 15, toAddrs, fromKeys)
}
