package main

import (
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/ethapi"
	"github.com/ethereum/go-ethereum/node"
)

func makeReadOnlyNode(datadir string) (*node.Node, ethapi.Backend) {
	node_cfg := node.DefaultConfig
	eth_cfg := ethconfig.Defaults
	node_cfg.DataDir = datadir
	node_cfg.ReadOnly = true
	node_cfg.LocalLib = true
	stack, err := node.New(&node_cfg)
	if err != nil {
		utils.Fatalf("Failed to create the protocol stack: %v", err)
	}
	backend, _ := utils.RegisterEthService(stack, &eth_cfg, false)
	return stack, backend
}

func StartNode(stack *node.Node) {
	if err := stack.Start(); err != nil {
		utils.Fatalf("Error starting protocol stack: %v", err)
	}
}
