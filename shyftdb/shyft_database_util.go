package shyftdb

import (
    "fmt"
    
    "github.com/syndtr/goleveldb/leveldb"
	"github.com/ethereum/go-ethereum/core/types"
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
    err := db.Put(hash, []byte("GOOD MORNING WORLD"), nil)
    fmt.Println("the error is: ++++++++++++++")
    fmt.Println(err)
	return nil
}