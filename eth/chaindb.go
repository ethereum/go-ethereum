package eth

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"fmt"
)

var Chaindb_global ethdb.Database

func SetChainDB(db ethdb.Database){
	Chaindb_global = db
}

func chaindb(ctx *node.ServiceContext, config *Config) (ethdb.Database, error) {
	fmt.Println("DO WE GET HERE ++++++++++++++++++++++++++++++++")
	fmt.Printf("%+v", ctx)
	if Chaindb_global != nil {
		return Chaindb_global, nil
	}
	fmt.Println("FAR OUT ++++++++++++++++++++++++++++++++")

	chainDb, err := CreateDB(ctx, config, "chaindata")
	fmt.Println("CHECK CHECK ++++++++++++++++")
	if err == nil {
		SetChainDB(chainDb)
		return Chaindb_global, nil
	}
	return nil, err
}