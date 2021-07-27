package main

import (
	"crypto/elliptic"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gcash/bchd/chaincfg"
	"github.com/gcash/bchutil"
	"github.com/smartbch/smartbch/staking"
)

func test() {
	privKey, _ := crypto.GenerateKey()
	pubKey := privKey.PublicKey
	bz := elliptic.Marshal(pubKey.Curve, pubKey.X, pubKey.Y)
	fmt.Printf("haha %d %#v\n", len(bz), bz)
}

func main() {
	test()
	//bitcoincash:qqqsjenhvqrf2024vapeuh3elp4q6femac89k2dn50
	bz, _ := hex.DecodeString("0450863AD64A87AE8A2FE83C1AF1A8403CB53F53E486D8511DAD8A04887E5B23522CD470243453A299FA9E77237716103ABC11A1DF38855ED6F2EE187E9C582BA6")
	fmt.Println(pubkey2cashaddr(bz))
	if len(os.Args) != 6 {
		fmt.Printf("Usage: %s <rpcURL> <username> <password> <startHeight> <endHeight>\n", os.Args[0])
		return
	}

	rpcURL := os.Args[1]
	rpcUsername := os.Args[2]
	rpcPassword := os.Args[3]

	startH, err := strconv.ParseInt(os.Args[4], 10, 32)
	if err != nil {
		panic(err)
	}

	endH, err := strconv.ParseInt(os.Args[5], 10, 32)
	if err != nil {
		panic(err)
	}

	client := staking.NewRpcClient(rpcURL, rpcUsername, rpcPassword, "text/plain;")
	collectKeys(client, startH, endH)
	//client.PrintAllOpReturn(519995, 679995)
}

func collectKeys(client *staking.RpcClient, startHeight, endHeight int64) {
	for h := startHeight; h < endHeight; h++ {
		fmt.Printf("Height: %d\n", h)
		hash, err := client.GetBlockHash(h)
		if err != nil {
			fmt.Printf("Error when getBlockHashOfHeight %d %s\n", h, err.Error())
			continue
		}
		bi, err := client.GetBlockInfo(hash)
		if err != nil {
			fmt.Printf("Error when getBlock %d %s\n", h, err.Error())
			continue
		}
		for _, txid := range bi.Tx {
			fmt.Printf("now tx %d %s\n", len(txid), txid)
			tx, err := client.GetTxInfo(txid)
			if err != nil {
				fmt.Printf("Error when getTx %s %s\n", txid, err.Error())
				continue
			}
			fmt.Printf("%#v\n", tx)
			for _, vout := range tx.VoutList {
				hexStr, ok := vout.ScriptPubKey["hex"].(string)
				fmt.Printf("hex %v %d %s\n", ok, len(hexStr), hexStr)
			}
			for _, vin := range tx.VinList {
				scriptSig, ok := vin["scriptSig"].(map[string]interface{})
				if !ok {
					continue
				}
				asm := scriptSig["asm"].(string)
				fields := strings.Split(asm, " ")
				if len(fields) != 2 {
					continue
				}
				pubkey := fields[1]
				var isPub33 bool
				var isPub65 bool
				isPub33 = len(pubkey) == 66 &&
					(strings.HasPrefix(pubkey, "02") || strings.HasPrefix(pubkey, "03"))
				isPub65 = len(pubkey) == 130 && strings.HasPrefix(pubkey, "04")
				if !(isPub65 || isPub33) {
					continue
				}
				bz, err := hex.DecodeString(pubkey)
				if err != nil {
					panic(err)
				}
				addr, err := bchutil.NewAddressPubKey(bz, &chaincfg.MainNetParams)
				if err != nil {
					fmt.Println(err)
					return
				}

				fmt.Printf("Pubkey %s %s\n", pubkey, addr.String())
				fmt.Println(pubkey2cashaddr(bz), pubkey2ethaddr(bz))
			}
		}
	}
}

func pubkey2cashaddr(pub []byte) string {
	pkhAddr, err := bchutil.NewAddressPubKeyHash(bchutil.Hash160(pub), &chaincfg.MainNetParams)
	if err != nil {
		panic(err)
	}
	return pkhAddr.EncodeAddress()
}

func pubkey2ethaddr(pub []byte) common.Address {
	if len(pub) == 65 {
		pubkey, err := crypto.UnmarshalPubkey(pub)
		if err != nil {
			panic(err)
		}
		return crypto.PubkeyToAddress(*pubkey)
	} else if len(pub) == 33 {
		pubkey, err := crypto.DecompressPubkey(pub)
		if err != nil {
			panic(err)
		}
		return crypto.PubkeyToAddress(*pubkey)
	}
	panic("invalid pubkey length")
}
