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
	fmt.Printf("CTX v+%", ctx)
	if Chaindb_global != nil {
		return Chaindb_global, nil
	}

	chainDb, err := CreateDB(ctx, config, "chaindata")
	if err == nil {
		SetChainDB(chainDb)
		return Chaindb_global, nil
	}
	return nil, err
}