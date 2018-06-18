package core

import (
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/common"
	"fmt"

)

const (
	// defaultTraceReexec is the number of blocks the tracer is willing to go back
	// and reexecute to produce missing historical state necessary to run a specific
	// trace.
	defaultTraceReexec = uint64(128)
)

//var api *eth.PrivateDebugAPI

var Chaindb_global ethdb.Database

func SetChainDB(db ethdb.Database){
	Chaindb_global = db
}

func MyTraceTransaction(hash string) (interface{}) {
	//fmt.Println("THE stringed hash is in MYTRACETRANSACTION")
	//fmt.Println(hash)
	//fmt.Println(ethObjectInterface.GetEthObject())
	common_hash := common.HexToHash(hash)


	//fmt.Println("the chain db is in MyTraceTransaction")
	//fmt.Println(Chaindb_global)
	if Chaindb_global == nil {
		return nil
	}

	tx, blockHash, _, index := GetTransaction(Chaindb_global, common_hash)

	//reexec := defaultTraceReexec
	//if config != nil && config.Reexec != nil {
	//	reexec = *config.Reexec
	//}
	//msg, vmctx, statedb, err := eth.api.computeTxEnv(blockHash, int(index), reexec)

	if tx == nil {
		fmt.Println("TRANSACTION NOT FOUND IN MyTraceTransaction +++++++++")
	}

	fmt.Println("the tx is ")
	fmt.Println(tx)
	fmt.Println(blockHash)
	fmt.Println(index)
	return 42
}