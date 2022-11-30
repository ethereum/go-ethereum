package utils

import (
	"os"

	"github.com/ethereum/go-ethereum/core"
	taikoGenesis "github.com/ethereum/go-ethereum/core/taiko_genesis"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

var (
	TaikoFlag = cli.BoolFlag{
		Name:  "taiko",
		Usage: "Taiko network",
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
	})
}

func SetTaikoDevelopNetwork(cfg *ethconfig.Config) {
	var allocJSON []byte
	switch cfg.NetworkId {
	case params.TaikoAlpha1NetworkID.Uint64():
		allocJSON = taikoGenesis.Alpha1GenesisAllocJSON
	case params.TaikoAlpha2NetworkID.Uint64():
		allocJSON = taikoGenesis.Alpha2GenesisAllocJSON
	default:
		log.Crit("can not find alloc json file for network")
	}
	var alloc core.GenesisAlloc
	if err := alloc.UnmarshalJSON(allocJSON); err != nil {
		log.Crit("unmarshal alloc json error", "error", err)
	}
	cfg.Genesis.Alloc = alloc
}
