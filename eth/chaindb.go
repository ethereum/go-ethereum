package eth

import (
	"github.com/ShyftNetwork/go-empyrean/ethdb"
	"github.com/ShyftNetwork/go-empyrean/node"
	"io/ioutil"
	"os"
)

var Chaindb_global ethdb.Database

func SetChainDB(db ethdb.Database){
	Chaindb_global = db
}

func chaindb(ctx *node.ServiceContext, config *Config) (ethdb.Database, error) {
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

//(*ethdb.LDBDatabase, func())
func NewShyftTestLDB() {
	dirname, err := ioutil.TempDir(os.TempDir(), "shyftdb_test_")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}
	db, err := ethdb.NewLDBDatabase(dirname, 0, 0)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}
	Chaindb_global = db

	//return db, func() {
	//	db.Close()
	//	os.RemoveAll(dirname)
	//}
}