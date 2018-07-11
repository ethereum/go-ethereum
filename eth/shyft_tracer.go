package eth


import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"context"
)

var EthereumObject interface{}

type ShyftTracer struct {}
//(
func (st ShyftTracer) GetTracerToRun (hash common.Hash) (interface{}, error) {
	config2 := params.ShyftNetworkChainConfig

	jsTracer := "callTracer"

	//
	//var cfg *Config
	var ctx2 context.Context
	config := &TraceConfig{
		LogConfig: nil,
		Tracer: &jsTracer,  // needs to be non-nil
		Timeout: nil,
		Reexec: nil,
	}
	//var fullNode *Ethereum
	fullNode, _ := SNew(Global_config)
	privateAPI := NewPrivateDebugAPI(config2, fullNode)
	return privateAPI.TraceTransaction(ctx2, hash, config)
}

func setEthObject(ethobj interface{}){
	EthereumObject = ethobj
}

var Global_config *Config

func setGlobalConfig(c *Config) {
	Global_config = c
}

//
//import (
//	"github.com/ethereum/go-ethereum/common"
//	"fmt"
//	"github.com/ethereum/go-ethereum/core"
//	"runtime"
//	"context"
//)
//
////const (
////	// defaultTraceReexec is the number of blocks the tracer is willing to go back
////	// and reexecute to produce missing historical state necessary to run a specific
////	// trace.
////	defaultTraceReexec = uint64(128)
////)
//
////var api *eth.PrivateDebugAPI
//
//var EthereumObject interface{}
//
//type ShyftTracer struct {}
//
//func (st ShyftTracer) MyTraceTransaction(hash string) (interface{}) {
//	//fmt.Println("THE stringed hash is in MYTRACETRANSACTION")
//	//fmt.Println(hash)
//	//fmt.Println(ethObjectInterface.GetEthObject())
//	fmt.Println("CTXXXXXXXXXXXXXXX")
//	common_hash := common.HexToHash(hash)
//
//
//	fmt.Println("the chain db is in MyTraceTransaction")
//	//fmt.Println(Chaindb_global)
//	if Chaindb_global == nil {
//		return nil
//	}
//
//	tx, blockHash, _, index := core.GetTransaction(Chaindb_global, common_hash)
//
//	if tx == nil {
//		fmt.Println("TRANSACTION NOT FOUND IN MyTraceTransaction +++++++++")
//	}
//
//	reexec := defaultTraceReexec
//	//if config != nil && config.Reexec != nil {
//	//	reexec = *config.Reexec
//	//}
//	msg, vmctx, statedb, err := DebugApi.computeTxEnv(blockHash, int(index), reexec)
//
//	fmt.Println("stuff and nonsense")
//	fmt.Println(msg)
//	fmt.Println(vmctx)
//	fmt.Println(statedb)
//	fmt.Println(err)
//	fmt.Println("THIS CTX")
//	//fmt.Println("the tx is ")
//	//fmt.Println(tx)
//	//fmt.Println(blockHash)
//	//fmt.Println(index)
//	//DebugApi.traceTxTest(msg, vmctx, statedb, config)
//	return 42
//}
//
//func (api *PrivateDebugAPI) TraceTransactionTest(ctx context.Context, hash common.Hash, config *TraceConfig) (interface{}, error) {
//	fmt.Println("TRACE TRANSACTION ++++++!+!+!+!+!+!+!+!+!+!+!+!++!")
//	_, file, no, ok := runtime.Caller(1)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(2)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(3)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(4)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(5)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(6)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(7)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(8)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(9)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(10)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(11)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(12)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(13)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(14)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(15)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(16)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(17)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(18)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(19)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(20)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(21)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(22)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(23)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(24)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(25)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(26)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(27)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(28)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(29)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(30)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(31)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(32)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(33)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(34)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(35)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(36)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(37)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(38)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(39)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(40)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	fmt.Println("////////////// ++++++++++++++++ /////////////////// Phantom *(!*!*!*!*!*!**!*!*")
//	_, file, no, ok = runtime.Caller(41)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(42)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(43)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	_, file, no, ok = runtime.Caller(44)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//	fmt.Println("////////////// ++++++++++++++++ /////////////////// TraceTransaction *(!*!*!*!*!*!**!*!*")
//	_, file, no, ok = runtime.Caller(45)
//	if ok {
//		fmt.Printf("called from %s#%d\n", file, no)
//	}
//
//	tx, blockHash, _, index := core.GetTransaction(api.eth.ChainDb(), hash)
//	if tx == nil {
//		return nil, fmt.Errorf("transaction %x not found", hash)
//	}
//
//	reexec := defaultTraceReexec
//	if config != nil && config.Reexec != nil {
//		reexec = *config.Reexec
//	}
//	msg, vmctx, statedb, err := api.computeTxEnv(blockHash, int(index), reexec)
//	if err != nil {
//		return nil, err
//	}
//	fmt.Println("CTX +++++++++++++++++++++++", ctx)
//	return api.traceTxTest(ctx, msg, vmctx, statedb, config)
//}
//
//func setEthObject(ethobj interface{}){
//	EthereumObject = ethobj
//}
