package backends

import (
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/catalyst"
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
	genesis := core.Genesis{
		Config:   params.AllEthashProtocolChanges,
		GasLimit: 0,
		Alloc:    alloc,
	}
	conf := &ethconfig.Defaults
	conf.Genesis = &genesis

	stack, err := node.New(&node.DefaultConfig)
	if err != nil {
		return nil, err
	}

	backend, err := eth.New(stack, conf)
	if err != nil {
		return nil, err
	}

	beacon, err := catalyst.NewSimulatedBeacon(12, backend)
	if err != nil {
		return nil, err
	}

	client, err := ethclient.Dial(stack.IPCEndpoint())
	if err != nil {
		return nil, err
	}

	return &NewSim{
		SimulatedBeacon: beacon,
		Client:          client,
	}, nil
}
