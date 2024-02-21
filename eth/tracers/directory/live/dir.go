package live

import (
	"encoding/json"
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

type ScopeContext interface {
	GetMemoryData() []byte
	GetStackData() []uint256.Int
	GetCaller() common.Address
	GetAddress() common.Address
	GetCallValue() *uint256.Int
	GetCallInput() []byte
}

type StateDB interface {
	GetBalance(common.Address) *uint256.Int
	GetNonce(common.Address) uint64
	GetCode(common.Address) []byte
	GetState(common.Address, common.Hash) common.Hash
	Exist(common.Address) bool
	GetRefund() uint64
}

// Canceler is an interface that wraps the Cancel method.
// It allows loggers to cancel EVM processing.
type Canceler interface {
	Cancel()
}

type VMContext struct {
	Coinbase    common.Address
	BlockNumber *big.Int
	Time        uint64
	Random      *common.Hash
	// Effective tx gas price
	GasPrice    *big.Int
	ChainConfig *params.ChainConfig
	StateDB     StateDB
	VM          Canceler
}

// BlockEvent is emitted upon tracing an incoming block.
// It contains the block as well as consensus related information.
type BlockEvent struct {
	Block     *types.Block
	TD        *big.Int
	Finalized *types.Header
	Safe      *types.Header
}

// OpCode is an EVM opcode
// TODO: provide utils for consumers
type OpCode byte

type LiveLogger struct {
	/*
		- VM events -
	*/
	// Transaction level
	// Call simulations don't come with a valid signature. `from` field
	// to be used for address of the caller.
	CaptureTxStart func(vm *VMContext, tx *types.Transaction, from common.Address)
	CaptureTxEnd   func(receipt *types.Receipt, err error)
	// Top call frame
	CaptureStart func(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int)
	// CaptureEnd is invoked when the processing of the top call ends.
	// See docs for `CaptureExit` for info on the `reverted` parameter.
	CaptureEnd func(output []byte, gasUsed uint64, err error, reverted bool)
	// Rest of call frames
	CaptureEnter func(typ OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int)
	// CaptureExit is invoked when the processing of a message ends.
	// `revert` is true when there was an error during the execution.
	// Exceptionally, before the homestead hardfork a contract creation that
	// ran out of gas when attempting to persist the code to database did not
	// count as a call failure and did not cause a revert of the call. This will
	// be indicated by `reverted == false` and `err == ErrCodeStoreOutOfGas`.
	CaptureExit func(output []byte, gasUsed uint64, err error, reverted bool)
	// Opcode level
	CaptureState          func(pc uint64, op OpCode, gas, cost uint64, scope ScopeContext, rData []byte, depth int, err error)
	CaptureFault          func(pc uint64, op OpCode, gas, cost uint64, scope ScopeContext, depth int, err error)
	CaptureKeccakPreimage func(hash common.Hash, data []byte)
	// Misc
	OnGasChange func(old, new uint64, reason GasChangeReason)

	/*
		- Chain events -
	*/
	OnBlockchainInit func(chainConfig *params.ChainConfig)
	// OnBlockStart is called before executing `block`.
	// `td` is the total difficulty prior to `block`.
	OnBlockStart func(event BlockEvent)
	OnBlockEnd   func(err error)
	// OnSkippedBlock indicates a block was skipped during processing
	// due to it being known previously. This can happen e.g. when recovering
	// from a crash.
	OnSkippedBlock func(event BlockEvent)
	OnGenesisBlock func(genesis *types.Block, alloc types.GenesisAlloc)

	/*
		- State events -
	*/
	OnBalanceChange func(addr common.Address, prev, new *big.Int, reason BalanceChangeReason)
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
