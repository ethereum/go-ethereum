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
	return privateAPI.STraceTransaction(ctx2, hash, config)
}

func setEthObject(ethobj interface{}){
	EthereumObject = ethobj
}

var Global_config *Config

func setGlobalConfig(c *Config) {
	Global_config = c
}