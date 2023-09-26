package backends

import (
	"context"
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/params"
)

var _ bind.ContractBackend = (*NewSim)(nil)

type NewSim struct {
	*catalyst.SimulatedBeacon
	*ethclient.Client
}

func NewNewSim(alloc core.GenesisAlloc) (*NewSim, error) {
	// Setup the node object
	nodeConf := &node.DefaultConfig
	nodeConf.IPCPath = "geth.ipc"
	nodeConf.DataDir = filepath.Join(os.TempDir(), "simulated-geth")
	stack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	// Setup ethereum
	genesis := core.Genesis{
		Config:   params.AllDevChainProtocolChanges,
		GasLimit: 30_000_000,
		Alloc:    alloc,
	}
	conf := &ethconfig.Defaults
	conf.Genesis = &genesis
	conf.SyncMode = downloader.FullSync

	backend, err := eth.New(stack, conf)
	if err != nil {
		return nil, err
	}

	// Start the node
	if err := stack.Start(); err != nil {
		return nil, err
	}

	// Set up the simulated beacon
	beacon, err := catalyst.NewSimulatedBeacon(12, backend)
	if err != nil {
		return nil, err
	}

	// Reorg our chain back to genesis
	if err := beacon.Fork(context.Background(), backend.BlockChain().GetCanonicalHash(0)); err != nil {
		return nil, err
	}

	return &NewSim{
		SimulatedBeacon: beacon,
		Client:          ethclient.NewClient(stack.Attach()),
	}, nil
}
