package main

import (
	"context"
	"fmt"
	"math/big"
	"crypto/ecdsa"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/core/types"

	"github.com/smartbch/testkit/chaintracker/client"
	"github.com/smartbch/testkit/ethutils"
)

var url = "http://localhost:8545"

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
		GasPrice: big.NewInt(1),
		Gas:      10000000,
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

func repeateNonce(repeatCount int64, nonceStart, nonceEnd uint64, toAddrList []common.Address, privKeys []string) {
	if len(toAddrList) != len(privKeys) {
		panic("length mismatch")
	}
	c, err := client.New(url)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	for nonce := nonceStart; nonce < nonceEnd; nonce++ {
		fmt.Printf("=============== nonce %d sleep %d ==============\n", nonce, time.Now().UnixNano())
		time.Sleep(2*time.Second)
		fmt.Printf("=============== nonce %d %d ==============\n", nonce, time.Now().UnixNano())
		for i := int64(1); i <= repeatCount; i++ {
			for j, toAddr := range toAddrList {
				tx := GetTx(&toAddr, i, nonce, privKeys[j])
				err = c.Rpc.CallContext(context.Background(), nil, "eth_sendRawTransaction", hexutil.Encode(tx))
				if err != nil {
					//fmt.Printf("E(%d-v%d-n%d): %s\n", j,i,nonce, err.Error())
					continue
				}
			}
		}
	}
}

var toKeys = []string{
"48d540e4ab73c814edd3d3a7509a70589dc3f0254866342c42d96b66ae235cd8",
"fce1bfe924327cd8f8afa6903ec25038c4532c604b34b4e13882e0ff41f045c6",
"e36b2e4af6cea8309b82c4ab9140a1ea6a7c3f1b0ec645e872685b5d728c5ffc",
"ba7deb8e1ad8fe676a7a2a5a91350d4f53514f10f4596ddcfe55712019777d26",
"b3f2bdbd8d1b9a4e29de3091cae54ca64f2d896b45f42e964004c3a7e8464976",
"7d359c3ada40c3b5d21e46787346901cb793efdfff2d2d263a291bfcc70905e0",
"00ced8b432ab7a85c03225fb69f2d9c2d4f16644ca5e8a3594b6edb3cb8bff41",
"4bd2afcd3b67bb6c5b84065e03633c053c636c5033b0bf83c6d616922ba029c5",
"1cd5f8f98d4475794a0f8034f0ba02a11a7015c95ea9641d9d94eb8b44d4ccc8",
"56a5efb09e9135e21e0bb8ff07cd6fb9f4d5ff528f92f8f60a2ab2a977c19821",
}

var fromKeys = []string{
"7fc6cf51adb430d9220c9f3ed4e992e75b5d1e8e52fe2bc99183cadc141725bc",
"08c65e04cd27b03d8bb8d19ffadadd82c2dd0935e3f23f313857a2c9629bba43",
"594d82ba88e52b2e037da8513493074eee5e6a6820d836afee5764fb78830285",
"433721d2f0e5c90d0a67a91153eaac3aa9db974ba9b4b9a7be219f02c12c015d",
"ff1f7f7276b877274043a42d17258b79dd4fd32ca17c48a5dc75049c1f931841",
"bab883ae3c7578be66ba5f1c1798dd52ab84ff9403a62c0b478491264df4a50e",
"2698171de1409b229fa14b71fa982507b276c7234c34cee8c42ac0713a614a4f",
"cb7883806fa970ef34b10286b80122b3188b09a24d154d2b81fb30e61c8b99ad",
"e58d53577a8c30b550db1b461c5aee5c8368946be945819cdfdd77dd990e55cd",
"fbb4694007aff7a979f46e76f9ec522015ed74702594864bde419a6c4a24f377",
}
/*

./smartbchd init mynode --chain-id 0x2711 \
  --init-balance=10000000000000000000 \
  --test-keys="7fc6cf51adb430d9220c9f3ed4e992e75b5d1e8e52fe2bc99183cadc141725bc,\
08c65e04cd27b03d8bb8d19ffadadd82c2dd0935e3f23f313857a2c9629bba43,\
594d82ba88e52b2e037da8513493074eee5e6a6820d836afee5764fb78830285,\
433721d2f0e5c90d0a67a91153eaac3aa9db974ba9b4b9a7be219f02c12c015d,\
ff1f7f7276b877274043a42d17258b79dd4fd32ca17c48a5dc75049c1f931841,\
bab883ae3c7578be66ba5f1c1798dd52ab84ff9403a62c0b478491264df4a50e,\
2698171de1409b229fa14b71fa982507b276c7234c34cee8c42ac0713a614a4f,\
cb7883806fa970ef34b10286b80122b3188b09a24d154d2b81fb30e61c8b99ad,\
e58d53577a8c30b550db1b461c5aee5c8368946be945819cdfdd77dd990e55cd,\
fbb4694007aff7a979f46e76f9ec522015ed74702594864bde419a6c4a24f377"

*/
func main() {
	toAddrs := make([]common.Address, len(toKeys))
	for i, key := range toKeys {
		toAddrs[i] = KeyToAddr(key)
	}
	repeateNonce(3000, 0, 1000, toAddrs, fromKeys)
}

