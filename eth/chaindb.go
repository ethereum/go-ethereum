package eth

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"fmt"
	"github.com/ethereum/go-ethereum/core"
)

func chaindb(ctx *node.ServiceContext, config *Config) (ethdb.Database, error) {
	fmt.Println("DO WE GET HERE ++++++++++++++++++++++++++++++++")
	fmt.Printf("%+v", ctx)
	if core.Chaindb_global != nil {
		return core.Chaindb_global, nil
	}
	fmt.Println("FAR OUT ++++++++++++++++++++++++++++++++")

	chainDb, err := CreateDB(ctx, config, "chaindata")
	fmt.Println("CHECK CHECK ++++++++++++++++")
	if err == nil {
		core.SetChainDB(chainDb)
		return core.Chaindb_global, nil
	}
	return nil, err
}