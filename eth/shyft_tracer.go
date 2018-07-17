package eth


import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"context"
	"fmt"
)

var EthereumObject interface{}

type ShyftTracer struct {}

var PrivateAPI *PrivateDebugAPI
var Context context.Context
var TracerConfig *TraceConfig

func InitTracerEnv() {
	fmt.Println("INIT tracer Env")
	config2 := params.ShyftNetworkChainConfig

	jsTracer := "callTracer"
	var ctx2 context.Context
	Context = ctx2
	config := &TraceConfig{
		LogConfig: nil,
		Tracer: &jsTracer,  // needs to be non-nil
		Timeout: nil,
		Reexec: nil,
	}
	TracerConfig = config
	fmt.Println("+++++++++++++++++++++++++++ before FULLNODE", Global_config)
	// called here
	fullNode, _ := SNew(Global_config)
	fmt.Println("+++++++++++++++++++++++++++ after FULLNODE")
	privateAPI := NewPrivateDebugAPI(config2, fullNode)
	PrivateAPI = privateAPI
	fmt.Println("+++++++++++++++++++++++++++ after privateAPI")
}

func (st ShyftTracer) GetTracerToRun (hash common.Hash) (interface{}, error) {
	return PrivateAPI.STraceTransaction(Context, hash, TracerConfig)
}

func setEthObject(ethobj interface{}){
	EthereumObject = ethobj
}

var Global_config *Config

func SetGlobalConfig(c *Config) {
	fmt.Println("SETGLOBAL")
	Global_config = c
}