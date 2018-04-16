package shyftdb

import (
    "fmt"
    
    "github.com/syndtr/goleveldb/leveldb"
    "github.com/syndtr/goleveldb/leveldb/util"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

func WriteBlock(db *leveldb.DB, block *types.Block) error {
    hash := block.Header().Hash().Bytes()
    fmt.Println(hash)
    if err := db.Put(hash, []byte("GOOD MORNING WORLD"), nil); err != nil {
    	log.Crit("Failed to store block", "err", err)
	}
	// If the block was not succesfully stored we will not store the tx data
    if block.Transactions().Len() > 0 {
		WriteTransactions(db, block.Transactions(), hash)
	}
	return nil
}

func WriteTransactions(db *leveldb.DB, transactions []*types.Transaction, blockHash []byte) error {
	for _, tx := range transactions {
		key := append([]byte("tx-")[:], tx.Hash().Bytes()[:]...)
		if err := db.Put(key, []byte("Hello hello"), nil); err != nil {
			log.Crit("Failed to store TX", "err", err)
		}
		GetTransaction(db, tx)
	}
	GetAllTransactions(db)
	return nil
}

// Meant for internal tests

func GetBlock(db *leveldb.DB, block *types.Block) {
	hash := block.Header().Hash().Bytes()
	bar, err := db.Get(hash, nil)
	fmt.Println("Our error is ")
	fmt.Println(err)
	fmt.Println("Our result is ")
	fmt.Println(bar)

}

func GetAllTransactions(db *leveldb.DB) {
	//iter := db.NewIterator(util.BytesPrefix([]byte("tx-")), nil)
	iter := db.NewIterator(util.BytesPrefix([]byte("tx-")), nil)
	for iter.Next() {
		fmt.Println(iter.Key())
		fmt.Println(iter.Value())
	}
	iter.Release()
}
func GetTransaction (db *leveldb.DB, tx *types.Transaction) {
	key := append([]byte("tx-")[:], tx.Hash().Bytes()[:]...)
	value, err := db.Get(key, nil)
	if err != nil {
		log.Crit("Could not retrieve data")
	}
	if len(value) > 0 {
		fmt.Println(value)
	}
}