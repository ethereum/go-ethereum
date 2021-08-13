package main

import (
	"context"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/plugins"
	"github.com/ethereum/go-ethereum/plugins/interfaces"
	"github.com/ethereum/go-ethereum/rpc"
	lru "github.com/hashicorp/golang-lru"
	"gopkg.in/urfave/cli.v1"
)

type MyService struct {
	backend interfaces.Backend
	stack   *node.Node
}

var pl *plugins.PluginLoader
var cache *lru.Cache

func Initialize(ctx *cli.Context, loader *plugins.PluginLoader) {
	pl = loader

	cache, _ = lru.New(128) // TODO: Make size configurable
	if !ctx.GlobalBool(utils.SnapshotFlag.Name) {
		log.Warn("Snapshots are required for StateUpdate plugins, but are currently disabled. State Updates will be unavailable")
	}
	log.Info("loaded is_synced plugin")
}

func GetAPIs(stack *node.Node, backend interfaces.Backend) []rpc.API {
	return []rpc.API{
		{
			Namespace: "mynamespace",
			Version:   "1.0",
			Service:   &MyService{backend, stack},
			Public:    true,
		},
	}
}

var zero = 0

func (h *MyService) IsSynced(ctx context.Context) bool {
	x := h.backend.Downloader()
	return h.stack.Server().PeerCount() > zero && x.Progress().CurrentBlock >= x.Progress().HighestBlock
}
