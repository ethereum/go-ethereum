package eth


import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/params"
	"context"
	"fmt"
)

var EthereumObject interface{}

type ShyftTracer struct {}

func (st ShyftTracer) GetTracerToRun (hash common.Hash) (interface{}, error) {
	config2 := params.ShyftNetworkChainConfig

	jsTracer := "callTracer"
	var ctx2 context.Context
	config := &TraceConfig{
		LogConfig: nil,
		Tracer: &jsTracer,  // needs to be non-nil
		Timeout: nil,
		Reexec: nil,
	}
	fmt.Println("+++++++++++++++++++++++++++ before FULLNODE", Global_config)
	fullNode, _ := SNew(Global_config)
	fmt.Println("+++++++++++++++++++++++++++ after FULLNODE")
	privateAPI := NewPrivateDebugAPI(config2, fullNode)
	fmt.Println("+++++++++++++++++++++++++++ after privateAPI")
	return privateAPI.STraceTransaction(ctx2, hash, config)
}

func setEthObject(ethobj interface{}){
	EthereumObject = ethobj
}

var Global_config *Config

func SetGlobalConfig(c *Config) {
	fmt.Println("SETGLOBAL")
	Global_config = c
}