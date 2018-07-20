package eth


import (
	"github.com/ShyftNetwork/go-empyrean/common"
	"github.com/ShyftNetwork/go-empyrean/params"
	"context"
)

var EthereumObject interface{}

type ShyftTracer struct {}

var PrivateAPI *PrivateDebugAPI
var Context context.Context
var TracerConfig *TraceConfig

func InitTracerEnv() {
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
	fullNode, _ := SNew(Global_config)
	privateAPI := NewPrivateDebugAPI(config2, fullNode)
	PrivateAPI = privateAPI
}

func (st ShyftTracer) GetTracerToRun (hash common.Hash) (interface{}, error) {
	return PrivateAPI.STraceTransaction(Context, hash, TracerConfig)
}

func setEthObject(ethobj interface{}){
	EthereumObject = ethobj
}

var Global_config *Config

func SetGlobalConfig(c *Config) {
	Global_config = c
}