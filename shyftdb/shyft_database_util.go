package shyftdb

import (
    "fmt"
    "bytes"
    "encoding/gob"
    "github.com/syndtr/goleveldb/leveldb"
	"github.com/ethereum/go-ethereum/core/types"
)

func GetBlock(db *leveldb.DB, block *types.Block) []byte {
    hash := block.Header().Hash().Bytes()
	bar, err := db.Get(hash, nil)
	fmt.Println("Our error is ")
	fmt.Println(err)
	fmt.Println("Our result is ")
	fmt.Println(bar)
	
	return bar
}

func WriteBlock(db *leveldb.DB, block *types.Block) error {
   	leng := block.Transactions().Len()
    var tx_strs = make([]string, leng)
    //var tx_bytes = make([]byte, leng)
	if block.Transactions().Len() > 0 {
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
    hash := block.Header().Hash().Bytes()
    fmt.Println(hash)
    err := db.Put(hash, bs, nil)
    fmt.Println("the error is: ++++++++++++++")
    fmt.Println(err)
	return nil
}