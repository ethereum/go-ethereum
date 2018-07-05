package eth

import (
	"github.com/ethereum/go-ethereum/common"
	"fmt"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"context"
)

var EthereumObject interface{}

type ShyftTracer struct {}

func (st ShyftTracer) GetTracerToRun(hash common.Hash, stack *node.Node) (interface{}, error) {
	config2 := params.ShyftNetworkChainConfig
	var cfg *Config
	var ctx context.Context
	var config *TraceConfig
	var fullNode *Ethereum

	err := stack.Register(func(ctx *node.ServiceContext) (node.Service, error) {
		fullNode, err := New(ctx, cfg)
		return fullNode, err
	})
	fmt.Println(err)
	privateAPI := NewPrivateDebugAPI(config2, fullNode)
	return privateAPI.TraceTransaction(ctx, hash, config)
}

func setEthObject(ethobj interface{}){
	EthereumObject = ethobj
}