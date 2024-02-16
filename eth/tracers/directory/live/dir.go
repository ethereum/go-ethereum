package live

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

type LiveLogger struct {
	VMLogger vm.EVMLogger

	/*
		- Chain events -
	*/
	OnBlockchainInit func(chainConfig *params.ChainConfig)
	// OnBlockStart is called before executing `block`.
	// `td` is the total difficulty prior to `block`.
	OnBlockStart func(event core.BlockEvent)
	OnBlockEnd   func(err error)
	// OnSkippedBlock indicates a block was skipped during processing
	// due to it being known previously. This can happen e.g. when recovering
	// from a crash.
	OnSkippedBlock func(event core.BlockEvent)
	OnGenesisBlock func(genesis *types.Block, alloc core.GenesisAlloc)

	/*
		- State events -
	*/
	OnBalanceChange func(addr common.Address, prev, new *big.Int, reason state.BalanceChangeReason)
	OnNonceChange   func(addr common.Address, prev, new uint64)
	OnCodeChange    func(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte)
	OnStorageChange func(addr common.Address, slot common.Hash, prev, new common.Hash)
	OnLog           func(log *types.Log)
}

type ctorFunc func(config json.RawMessage) (*LiveLogger, error)

// Directory is the collection of tracers which can be used
// during normal block import operations.
var Directory = directory{elems: make(map[string]ctorFunc)}

type directory struct {
	elems map[string]ctorFunc
}

// Register registers a tracer constructor by name.
func (d *directory) Register(name string, f ctorFunc) {
	d.elems[name] = f
}

// New instantiates a tracer by name.
func (d *directory) New(name string, config json.RawMessage) (*LiveLogger, error) {
	if f, ok := d.elems[name]; ok {
		return f(config)
	}
	return nil, errors.New("not found")
}
