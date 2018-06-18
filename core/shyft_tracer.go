package core

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/common"
	"fmt"
)

var Chaindb_global ethdb.Database

func SetChainDB(db ethdb.Database){
	Chaindb_global = db
}



func MyTraceTransaction(hash string) (interface{}) {
	common_hash := common.StringToHash(hash)
	fmt.Println("the hash is")
	fmt.Println(common_hash)

	fmt.Println("the chain db is")
	fmt.Println(Chaindb_global)
	if Chaindb_global == nil {
		return nil
	}

	tx, blockHash, _, index := GetTransaction(Chaindb_global, common_hash)

	fmt.Println("the tx is ")
	fmt.Println(tx)
	fmt.Println(blockHash)
	fmt.Println(index)
	return 42
}