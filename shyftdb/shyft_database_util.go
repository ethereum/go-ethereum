package shyftdb

import (
    "fmt"
    "bytes"
    "encoding/gob"
	"math/big"

    "github.com/syndtr/goleveldb/leveldb"
    "github.com/syndtr/goleveldb/leveldb/util"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/common"
)

type txEntry struct {
	TxHash    common.Hash
	To   	  *common.Address
	From 	  *common.Address
	BlockHash []byte
	Amount 	  *big.Int
	GasPrice  *big.Int
	Gas 	  uint64
	Nonce     uint64
	Data      []byte
}

func WriteBlock(db *leveldb.DB, block *types.Block) error {
	leng := block.Transactions().Len()
	var tx_strs = make([]string, leng)
	//var tx_bytes = make([]byte, leng)

    hash := block.Header().Hash().Bytes()
	if block.Transactions().Len() > 0 {
		for i, tx := range block.Transactions() {
 			tx_strs[i] = WriteTransactions(db, tx, hash)
			//tx_bytes[i] = tx.Hash().Bytes()
 		}
	}
	
	fmt.Println("The tx_strs is")
	fmt.Println(tx_strs)
	//strs := []string{"foo", "bar"}
	buf := &bytes.Buffer{}
	gob.NewEncoder(buf).Encode(tx_strs)
	bs := buf.Bytes()
	
    key := append([]byte("bk-")[:], hash[:]...)
	if err := db.Put(key, bs, nil); err != nil {
		log.Crit("Failed to store block", "err", err)
		return nil // Do we want to force an exit here?
	}
	return nil
}

func WriteTransactions(db *leveldb.DB, tx *types.Transaction, blockHash []byte) string {
	txData := txEntry{
		TxHash:    tx.Hash(),
		To:   	   tx.To(),
		From: 	   tx.From(),
		BlockHash: blockHash,
		Amount:    tx.Value(),
		GasPrice:  tx.GasPrice(),
		Gas:   	   tx.Gas(),
		Nonce:     tx.Nonce(),
		Data:      tx.Data(),
	}
	fmt.Println(txData)
	key := append([]byte("tx-")[:], tx.Hash().Bytes()[:]...)
	if err := db.Put(key, []byte("Hello hello"), nil); err != nil {
		log.Crit("Failed to store TX", "err", err)
	}
	return tx.Hash().String()
}

// Meant for internal tests

func GetBlock(db *leveldb.DB, block *types.Block) []byte {
	hash := block.Header().Hash().Bytes()
	key := append([]byte("bk-")[:], hash[:]...)
	data, err := db.Get(key, nil)
	if err != nil {
		log.Crit("Could not retrieve block", "err", err)
	}
	fmt.Println("\nBLOCK Value: " + string(data))
	return data
}

func GetAllTransactions(db *leveldb.DB) {
	iter := db.NewIterator(util.BytesPrefix([]byte("tx-")), nil)
	for iter.Next() {
		fmt.Println("\nALL TX VALUE: " + string(iter.Value()))
	}
	iter.Release()
}

func GetTransaction (db *leveldb.DB, tx *types.Transaction) {
	key := append([]byte("tx-")[:], tx.Hash().Bytes()[:]...)
	data, err := db.Get(key, nil)
	if err != nil {
		log.Crit("Could not retrieve TX", "err", err)
	}
	if len(data) > 0 {
		fmt.Println("\nTX Value: " + string(data))
	}
}