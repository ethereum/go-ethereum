package utils

import (
	"os"

	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

var (
	TaikoFlag = &cli.BoolFlag{
		Name:     "taiko",
		Usage:    "Taiko network",
		Category: flags.TaikoCategory,
	}
	PreconfirmationForwardingURLFlag = &cli.StringFlag{
		Name:     "taiko.preconfirmationForwardingUrl",
		Usage:    "URL to forward RPC requests before confirmation",
		Category: flags.TaikoCategory,
	}
)

// RegisterTaikoAPIs initializes and registers the Taiko RPC APIs.
func RegisterTaikoAPIs(stack *node.Node, cfg *ethconfig.Config, backend *eth.Ethereum) {
	if os.Getenv("TAIKO_TEST") != "" {
		return
	}
	// Add methods under "taiko_" RPC namespace to the available APIs list
	stack.RegisterAPIs([]rpc.API{
		{
			Namespace: "taiko",
			Version:   params.VersionWithMeta,
			Service:   eth.NewTaikoAPIBackend(backend),
			Public:    true,
		},
		{
			Namespace:     "taikoAuth",
			Version:       params.VersionWithMeta,
			Service:       eth.NewTaikoAuthAPIBackend(backend),
			Authenticated: true,
		},
	})
}
