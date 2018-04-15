package shyftdb

import (
    "fmt"
    
    "github.com/syndtr/goleveldb/leveldb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

func GetBlock(db *leveldb.DB, block *types.Block) {
    hash := block.Header().Hash().Bytes()
	bar, err := db.Get(hash, nil)
	fmt.Println("Our error is ")
	fmt.Println(err)
	fmt.Println("Our result is ")
	fmt.Println(bar)
	
}

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
		txHash := tx.Hash().Bytes()
		if err := db.Put(txHash, []byte("Hello hello"), nil); err != nil {
			log.Crit("Failed to store TX", "err", err)
		}
		GetTransaction(db, tx)
	}
	return nil
}

func GetTransaction(db *leveldb.DB, tx *types.Transaction) {
	data, err := db.Get(tx.Hash().Bytes(), nil)
	if err != nil {
		log.Crit("Could not retrieve data")
	}
	if len(data) > 0 {
		fmt.Println("\n\t\t\t DATA \n")
		fmt.Println(data)
	}
}