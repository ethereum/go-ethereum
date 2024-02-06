package live

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/eth/tracers/directory"
	"github.com/ethereum/go-ethereum/params"
)

func init() {
	directory.LiveDirectory.Register("liveNoop", newLiveNoopTracer)
}

// liveNoop is a no-op live tracer. It's there to
// catch changes in the tracing interface, as well as
// for testing live tracing performance. Can be removed
// as soon as we have a real live tracer.
type liveNoop struct{}

func newLiveNoopTracer() (core.BlockchainLogger, error) {
	return &liveNoop{}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *liveNoop) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *liveNoop) CaptureEnd(output []byte, gasUsed uint64, err error, reverted bool) {
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *liveNoop) CaptureState(pc uint64, op vm.OpCode, gas, cost uint64, scope *vm.ScopeContext, rData []byte, depth int, err error) {
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *liveNoop) CaptureFault(pc uint64, op vm.OpCode, gas, cost uint64, _ *vm.ScopeContext, depth int, err error) {
}

// CaptureKeccakPreimage is called during the KECCAK256 opcode.
func (t *liveNoop) CaptureKeccakPreimage(hash common.Hash, data []byte) {}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *liveNoop) CaptureEnter(typ vm.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *liveNoop) CaptureExit(output []byte, gasUsed uint64, err error, reverted bool) {
}

func (t *liveNoop) OnBeaconBlockRootStart(root common.Hash) {}
func (t *liveNoop) OnBeaconBlockRootEnd()                   {}

func (t *liveNoop) CaptureTxStart(env *vm.EVM, tx *types.Transaction, from common.Address) {
}

func (t *liveNoop) CaptureTxEnd(receipt *types.Receipt, err error) {
}

func (t *liveNoop) OnBlockStart(b *types.Block, td *big.Int, finalized, safe *types.Header) {
}

func (t *liveNoop) OnBlockEnd(err error) {
}

func (t *liveNoop) OnBlockchainInit(chainConfig *params.ChainConfig) {
}

func (t *liveNoop) OnGenesisBlock(b *types.Block, alloc core.GenesisAlloc) {
}

func (t *liveNoop) OnBalanceChange(a common.Address, prev, new *big.Int, reason state.BalanceChangeReason) {
}

func (t *liveNoop) OnNonceChange(a common.Address, prev, new uint64) {
}

func (t *liveNoop) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
}

func (t *liveNoop) OnStorageChange(a common.Address, k, prev, new common.Hash) {
}

func (t *liveNoop) OnLog(l *types.Log) {

}

func (t *liveNoop) OnNewAccount(a common.Address, reset bool) {
}

func (t *liveNoop) OnGasChange(old, new uint64, reason vm.GasChangeReason) {
}
