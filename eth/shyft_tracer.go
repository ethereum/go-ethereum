package eth

import (
	"github.com/ethereum/go-ethereum/common"
	"fmt"
	"github.com/ethereum/go-ethereum/core"
)

//const (
//	// defaultTraceReexec is the number of blocks the tracer is willing to go back
//	// and reexecute to produce missing historical state necessary to run a specific
//	// trace.
//	defaultTraceReexec = uint64(128)
//)

//var api *eth.PrivateDebugAPI

var EthereumObject interface{}

type ShyftTracer struct {}

func (st ShyftTracer) MyTraceTransaction(hash string) (interface{}) {
	//fmt.Println("THE stringed hash is in MYTRACETRANSACTION")
	//fmt.Println(hash)
	//fmt.Println(ethObjectInterface.GetEthObject())
	common_hash := common.HexToHash(hash)


	//fmt.Println("the chain db is in MyTraceTransaction")
	//fmt.Println(Chaindb_global)
	if Chaindb_global == nil {
		return nil
	}

	tx, blockHash, _, index := core.GetTransaction(Chaindb_global, common_hash)

	if tx == nil {
		fmt.Println("TRANSACTION NOT FOUND IN MyTraceTransaction +++++++++")
	}

	reexec := defaultTraceReexec
	//if config != nil && config.Reexec != nil {
	//	reexec = *config.Reexec
	//}
	msg, vmctx, statedb, err := DebugApi.computeTxEnv(blockHash, int(index), reexec)

	fmt.Println("stuff and nonsense")
	fmt.Println(msg)
	fmt.Println(vmctx)
	fmt.Println(statedb)
	fmt.Println(err)

	//fmt.Println("the tx is ")
	//fmt.Println(tx)
	//fmt.Println(blockHash)
	//fmt.Println(index)
	//DebugApi.traceTx(ctx, msg, vmctx, statedb, config)
	return 42
}

func setEthObject(ethobj interface{}){
	EthereumObject = ethobj
}