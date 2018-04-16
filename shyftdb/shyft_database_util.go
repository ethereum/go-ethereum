package shyftdb

import (
    "fmt"
    "bytes"
    "encoding/gob"
    "github.com/syndtr/goleveldb/leveldb"
    "github.com/syndtr/goleveldb/leveldb/util"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

func WriteBlock(db *leveldb.DB, block *types.Block) error {
	leng := block.Transactions().Len()
	var tx_strs = make([]string, leng)
	//var tx_bytes = make([]byte, leng)

    hash := block.Header().Hash().Bytes()
	if block.Transactions().Len() > 0 {
	    // this is inefficient, there are 2 loops over 
	    // block.Transactions
	    // TODO: Fix this so there is only one loop
		WriteTransactions(db, block.Transactions(), hash)
		for i, tx := range block.Transactions() {
 			fmt.Println(tx.Hash())
 			fmt.Println("TX HASH")
 			fmt.Println(tx.To().Hex())
 			tx_strs[i] = tx.Hash().String()
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
func GetAllBlocks(db *leveldb.DB) {
	iter := db.NewIterator(util.BytesPrefix([]byte("bk-")), nil)
	for iter.Next() {
	    result := iter.Value()
	    buf := bytes.NewBuffer(result)
		strs2 := []string{}
		gob.NewDecoder(buf).Decode(&strs2)
		fmt.Println("the key is")
		hash := common.BytesToHash(iter.Key())
		hex := hash.Hex()
		fmt.Println(hex)
		if(len(strs2) > 0){
			fmt.Println("ALL TRANSACTIONS:")
			fmt.Printf("%v", strs2)
	    	fmt.Println("")		
		}

		//fmt.Println("\n ALL BK BK VALUE" + string(result))
	}
	
	iter.Release()
}

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