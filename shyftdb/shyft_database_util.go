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
	fmt.Println("The tx_strs is")
	fmt.Println(tx_strs)
	//strs := []string{"foo", "bar"}
	buf := &bytes.Buffer{}
	gob.NewEncoder(buf).Encode(tx_strs)
	bs := buf.Bytes()
	hash := block.Header().Hash().Bytes()
	if err := db.Put(hash, bs, nil); err != nil {
		log.Crit("Failed to store block", "err", err)
		return nil // Do we want to force an exit here?
	}
	if block.Transactions().Len() > 0 {
		WriteTransactions(db, block.Transactions(), hash)
	}
	return nil
}

func WriteTransactions(db *leveldb.DB, transactions []*types.Transaction, blockHash []byte) error {
	for _, tx := range transactions {
		var from = GenerateFromAddr()
		txData := txEntry{
			TxHash:    tx.Hash(),
			To:   	   tx.To(),
			From: 	   from,
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
		GetTransaction(db, tx)
	}
	GetAllTransactions(db)
	return nil
}

// Helper functions

func GenerateFromAddr(tx *types.Transaction) *common.Address {
	var from *common.Address
	signer := deriveSigner(tx.data.V)
	if f, err := Sender(signer, tx); err != nil { // derive but don't cache
		from = "[invalid sender: invalid sig]"
	} else {
		from = fmt.Sprintf("%x", f[:])
	}
}

// Meant for internal tests

func GetBlock(db *leveldb.DB, block *types.Block) {
	hash := block.Header().Hash().Bytes()
	data, err := db.Get(hash, nil)
	if err != nil {
		log.Crit("Could not retrieve block", "err", err)
	}
	fmt.Println("\nBLOCK Value: " + string(data))
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