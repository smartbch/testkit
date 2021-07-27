package p2pkhdb

import (
	"bytes"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/gcash/bchutil"
	"github.com/smartbch/moeingads/indextree"
	"github.com/smartbch/smartbch/staking"
)

const (
	EthAddrToFreeGasByte    byte = 50 // eth address => remained free gas
	BchAddrUTXOToAmountByte byte = 70 // bch address, utxo-id => amount
	BchAddr2EthAddrByte     byte = 80
	HeightByte              byte = 90
)

type RocksDB = indextree.RocksDB

type BalanceInfo struct {
	EthAddr common.Address
	Balance uint64
}

type P2pkhDB struct {
	rdb *RocksDB
}

func (db *P2pkhDB) BeginBlock(h int64) {
	db.rdb.OpenNewBatch()
	var buf [8]byte
	binary.LittleEndian.PutUint64(buf[:], uint64(h))
	db.rdb.CurrBatch().Set([]byte{HeightByte}, buf[:])
}

func (db *P2pkhDB) LatestHeight() int64 {
	bz := db.rdb.Get([]byte{HeightByte})
	return int64(binary.LittleEndian.Uint64(bz))
}

func (db *P2pkhDB) EndBlock() {
	db.rdb.CloseOldBatch()
}

func (db *P2pkhDB) AddUTXO(utxoId [32 + 4]byte, addr []byte, amount uint64) {
	key := make([]byte, 21, 21+32+4)
	key[0] = BchAddrUTXOToAmountByte
	copy(key[1:], addr)
	key = append(key, utxoId[:]...)
	var amountBz [8]byte
	binary.LittleEndian.PutUint64(amountBz[:], amount)
	db.rdb.CurrBatch().Set(key, amountBz[:])
}

func (db *P2pkhDB) RemoveUTXO(utxoId [32 + 4]byte, pub []byte) {
	ethAddr, err := pubkey2ethaddr(pub)
	if err != nil {
		fmt.Printf("Err in AddUTXO %#v\n", err)
		return
	}
	bchAddr := bchutil.Hash160(pub)
	db.rdb.CurrBatch().Set(append([]byte{BchAddr2EthAddrByte}, bchAddr...), ethAddr[:])

	key := make([]byte, 21, 21+32+4)
	key[0] = BchAddrUTXOToAmountByte
	copy(key[1:], bchAddr)
	key = append(key, utxoId[:]...)
	db.rdb.CurrBatch().Delete(key)
}

func (db *P2pkhDB) addNewBalanceInfo(bchAddr []byte, info BalanceInfo, balanceChan chan BalanceInfo) {
	ethAddr := db.rdb.Get(append([]byte{BchAddr2EthAddrByte}, bchAddr...))
	if len(ethAddr) == 0 {
		return
	}
	copy(info.EthAddr[:], ethAddr)
	balanceChan <- info
}

func (db *P2pkhDB) ScanEthAddrAndBalance(balanceChan chan BalanceInfo) {
	currBchAddr := []byte{}
	var info BalanceInfo
	iter := db.rdb.Iterator([]byte{BchAddrUTXOToAmountByte}, []byte{BchAddrUTXOToAmountByte + 1})
	defer iter.Close()
	for ; iter.Valid(); iter.Next() {
		key := iter.Key()
		if bytes.Equal(currBchAddr, key[:20]) {
			info.Balance += binary.LittleEndian.Uint64(iter.Value())
		} else {
			db.addNewBalanceInfo(currBchAddr, info, balanceChan)
			currBchAddr = append([]byte{}, key[:20]...)
			info.Balance = 0
		}
	}
	db.addNewBalanceInfo(currBchAddr, info, balanceChan)
	close(balanceChan)
}

func pubkey2ethaddr(pub []byte) (common.Address, error) {
	if len(pub) == 65 {
		pubkey, err := crypto.UnmarshalPubkey(pub)
		if err != nil {
			return common.Address{}, err
		}
		return crypto.PubkeyToAddress(*pubkey), nil
	} else if len(pub) == 33 {
		pubkey, err := crypto.DecompressPubkey(pub)
		if err != nil {
			return common.Address{}, err
		}
		return crypto.PubkeyToAddress(*pubkey), nil
	}
	return common.Address{}, errors.New("Invalid length of public key")
}

// ====================================================

type GasDB struct {
	rdb *RocksDB
}

func (db *GasDB) BeginBlock() {
	db.rdb.OpenNewBatch()
}

func (db *GasDB) EndBlock() {
	db.rdb.CloseOldBatch()
}

func (db *GasDB) SetGas(ethAddr common.Address, gas uint64) {
	var gasBz [8]byte
	binary.LittleEndian.PutUint64(gasBz[:], gas)
	db.rdb.CurrBatch().Set(ethAddr[:], gasBz[:])
}

func (db *GasDB) DeductGas(ethAddr common.Address, gas uint64) bool {
	gasBz := db.rdb.Get(ethAddr[:])
	if len(gasBz) == 0 {
		return false
	}
	remainedGas := binary.LittleEndian.Uint64(gasBz[:])
	if remainedGas < gas {
		return false
	}
	db.SetGas(ethAddr, remainedGas-gas)
	return true
}

// ====================================================

type DBKeeper struct {
	db    *P2pkhDB
	gasDb *GasDB
}

func (keeper *DBKeeper) UpdateToHeight(client *staking.RpcClient, h int64) {
	hash, err := client.GetBlockHash(h)
	if err != nil {
		fmt.Printf("Error when getBlockHashOfHeight %d %s\n", h, err.Error())
		return
	}
	bi, err := client.GetBlockInfo(hash)
	if err != nil {
		fmt.Printf("Error when getBlock %d %s\n", h, err.Error())
		return
	}
	for _, txid := range bi.Tx {
		tx, err := client.GetTxInfo(txid)
		if err != nil {
			fmt.Printf("Error when getTx %s %s\n", txid, err.Error())
			continue
		}
		var utxoId [36]byte
		txBz, err := hex.DecodeString(txid)
		if err != nil {
			fmt.Printf("err in decode txid %#v\n", err)
			return
		}
		copy(utxoId[:], txBz)
		for _, vout := range tx.VoutList {
			binary.LittleEndian.PutUint32(utxoId[32:], uint32(vout.N))
			keeper.handleVout(utxoId, vout)
		}
	}
}

func (keeper *DBKeeper) handleVout(utxoId [36]byte, vout staking.Vout) {
	typeStr, ok := vout.ScriptPubKey["type"].(string)
	if !ok || typeStr != "pubkeyhash" {
		return
	}
	hexStr, ok := vout.ScriptPubKey["hex"].(string)
	if !ok || len(hexStr) != 50 {
		return
	}
	bchAddr, err := hex.DecodeString(hexStr[6:46])
	if err != nil {
		fmt.Printf("err in decode hex address %#v\n", err)
		return
	}
	keeper.db.AddUTXO(utxoId, bchAddr, uint64(vout.Value*1e8))
}

func (keeper *DBKeeper) handleVin(vin map[string]interface{}) {
	txid, ok := vin["txid"].(string)
	if !ok {
		fmt.Printf("no txid\n")
		return
	}
	n, ok := vin["vout"].(int)
	if !ok {
		fmt.Printf("no vout\n")
		return
	}
	var utxoId [36]byte
	txBz, err := hex.DecodeString(txid)
	if err != nil {
		fmt.Printf("err in decode txid %#v\n", err)
		return
	}
	copy(utxoId[:], txBz)
	binary.LittleEndian.PutUint32(utxoId[32:], uint32(n))

	scriptSig, ok := vin["scriptSig"].(map[string]interface{})
	if !ok {
		return
	}
	asm := scriptSig["asm"].(string)
	fields := strings.Split(asm, " ")
	if len(fields) != 2 {
		return
	}
	pubkey := fields[1]
	bz, err := hex.DecodeString(pubkey)
	if err != nil {
		fmt.Printf("Error in DecodeString %s\n", err)
		return
	}
	isPub33 := len(bz) == 33 && (bz[0] == 0x2 || bz[0] == 0x3)
	isPub65 := len(bz) == 65 && bz[0] == 0x4
	if !(isPub65 || isPub33) {
		return
	}
	keeper.db.RemoveUTXO(utxoId, bz)
}
