package backends

import (
	"context"
	"math"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
	"github.com/ethereum/go-ethereum/eth/downloader"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/eth/filters"
	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
)

var _ bind.ContractBackend = (*SimulatedBackend)(nil)

type SimulatedBackend struct {
	eth *eth.Ethereum
	*catalyst.SimulatedBeacon
	*ethclient.Client
}

// NewSimulatedBackend creates a new binding backend using a simulated blockchain
// for testing purposes.
// A simulated backend always uses chainID 1337.
func NewSimulatedBackend(alloc core.GenesisAlloc, gasLimit uint64) (*SimulatedBackend, error) {
	// Setup the node object
	nodeConf := &node.DefaultConfig
	nodeConf.DataDir = ""
	nodeConf.P2P = p2p.Config{DiscAddr: "", ListenAddr: ""}
	stack, err := node.New(nodeConf)
	if err != nil {
		return nil, err
	}

	// Setup ethereum
	genesis := core.Genesis{
		Config:   params.AllDevChainProtocolChanges,
		GasLimit: gasLimit,
		Alloc:    alloc,
	}
	conf := &ethconfig.Defaults
	conf.Genesis = &genesis
	conf.SyncMode = downloader.FullSync
	return NewSimWithNode(stack, conf, math.MaxUint64)
}

// NewSimWithNode sets up a simulated backend on an existing node
// this allows users to do persistent simulations.
func NewSimWithNode(stack *node.Node, conf *eth.Config, blockPeriod uint64) (*SimulatedBackend, error) {
	backend, err := eth.New(stack, conf)
	if err != nil {
		return nil, err
	}

	// Register the filter system
	filterSystem := filters.NewFilterSystem(backend.APIBackend, filters.Config{})
	stack.RegisterAPIs([]rpc.API{{
		Namespace: "eth",
		Service:   filters.NewFilterAPI(filterSystem, false),
	}})

	// Start the node
	if err := stack.Start(); err != nil {
		return nil, err
	}

	// Set up the simulated beacon
	beacon, err := catalyst.NewSimulatedBeacon(blockPeriod, backend)
	if err != nil {
		return nil, err
	}

	// Reorg our chain back to genesis
	if err := beacon.Fork(context.Background(), backend.BlockChain().GetCanonicalHash(0)); err != nil {
		return nil, err
	}

	return &SimulatedBackend{
		eth:             backend,
		SimulatedBeacon: beacon,
		Client:          ethclient.NewClient(stack.Attach()),
	}, nil
}

func (n *SimulatedBackend) Close() error {
	n.Client.Close()
	return n.SimulatedBeacon.Stop()
}

// Blockchain is needed for LES-tests
func (n *SimulatedBackend) Blockchain() *core.BlockChain {
	return n.eth.BlockChain()
}

// ChainDB is needed for LES-tests
func (n *SimulatedBackend) ChainDB() ethdb.Database {
	return n.eth.ChainDb()
}
