package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/rlp"
	"log"
	"os"
	"sync"
	"time"

	"github.com/smartbch/testkit/chaintracker/client"
)

var url = "http://135.181.219.10:8545"
var url1 = "http://106.75.244.31:8545"
var url2 = "http://106.75.214.131:8545"
var url3 = "http://158.247.192.195:8545"
var urlLocal = "http://127.0.0.1:8545"

var txList []client.Transaction
var txListPath = "./out.json"
var rawTxPath = "./out.txt"

type Info struct {
	TotalNewAccount uint
	TotalContract   uint
	TotalTx         uint
}

var info Info
var addressSet map[common.Address]bool
var lock sync.Mutex
var w sync.WaitGroup

//func main() {
//	c, err := client.New(url)
//	if err != nil {
//		panic(err)
//	}
//	defer c.Close()
//	c1, err := client.New(url1)
//	if err != nil {
//		panic(err)
//	}
//	defer c1.Close()
//	c2, err := client.New(url2)
//	if err != nil {
//		panic(err)
//	}
//	defer c2.Close()
//
//	txList = make([]client.Transaction, 20000)
//	addressSet = make(map[common.Address]bool)
//	currentHeight, err := c.BlockNumber()
//	if err != nil {
//		panic(err)
//	}
//	fmt.Println("current height: ", currentHeight)
//	w.Add(3)
//	go getChainInfo(1, currentHeight/3, c)
//	go getChainInfo(currentHeight/3, 2*currentHeight/3, c1)
//	go getChainInfo(2*currentHeight/3, currentHeight, c2)
//	w.Wait()
//	infoJson, _ := json.MarshalIndent(info, "", "    ")
//	fmt.Println("Total info: ", string(infoJson))
//	writeResult()
//}

func getChainInfo(start, end uint64, c *client.Client) {
	var i uint64
	for i = start; i < end; i++ {
		if i%1000 == 0 {
			fmt.Println("height: ", i)
		}
		txs, err := c.GetTxListByHeight(i)
		if err != nil {
			panic(err)
		}
		if len(txs) == 0 {
			continue
		}
		lock.Lock()
		txList = append(txList, txs...)
		for _, tx := range txs {
			//out, err := tx.PrintJson()
			//if err != nil {
			//	panic(err)
			//}
			//fmt.Printf("height:%d, tx:%s\n", i, string(out))
			//t := client.ConvertTx(tx)
			//err = c.SendRawTransaction(t)
			//if err != nil {
			//	panic(err)
			//}
			updateInfo(&tx)
			//infoJson, _ := json.MarshalIndent(info, "", "    ")
			//fmt.Println("Total info: ", string(infoJson))
		}
		lock.Unlock()
	}
	w.Done()
}

func updateInfo(tx *client.Transaction) {
	zeroAddress := common.Address{}
	to := common.Address{}
	from := common.Address{}
	to.SetBytes(tx.To.Bytes())
	from.SetBytes(tx.From.Bytes())
	if to == zeroAddress {
		info.TotalContract++
	} else if !addressSet[from] {
		addressSet[from] = true
		info.TotalNewAccount++
	}
	info.TotalTx++
}

func writeResult() {
	file, err := os.OpenFile(txListPath, os.O_WRONLY|os.O_CREATE, 0666)
	if err != nil {
		panic(err)
	}
	defer file.Close()
	w := bufio.NewWriter(file)
	out, err := json.Marshal(txList)
	if err != nil {
		panic(err)
	}
	_, _ = w.Write(out)
	_ = w.Flush()
}

func main() {
	c, err := client.New(url3)
	if err != nil {
		panic(err)
	}
	defer c.Close()
	fi, err := os.Open(rawTxPath)
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	defer fi.Close()
	br := bufio.NewScanner(fi)
	i := 0
	for br.Scan() {
		s := br.Text()
		raw, err := hex.DecodeString(s)
		if err != nil {
			panic(err)
		}
		for {
			err = c.Rpc.CallContext(context.Background(), nil, "eth_sendRawTransaction", hexutil.Encode(raw))
			if err != nil && err.Error() == "method handler crashed" {
				time.Sleep(500 * time.Millisecond)
				fmt.Println("try again")
				continue
			} else if err != nil {
				fmt.Println(err)
			}
			break
		}
		time.Sleep(15 * time.Millisecond)
		i++
		if i%1000 == 0 {
			fmt.Println("another 1000 txs sent!")
		}
	}
	if err := br.Err(); err != nil {
		log.Fatal(err)
	}
}

func DecodeTx(data []byte) (*types.Transaction, error) {
	tx := &types.Transaction{}
	err := tx.DecodeRLP(rlp.NewStream(bytes.NewReader(data), 0))
	return tx, err
}
