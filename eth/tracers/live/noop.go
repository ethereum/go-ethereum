package live

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers/directory/live"
	"github.com/ethereum/go-ethereum/params"
)

func init() {
	live.Directory.Register("noop", newNoopTracer)
}

// noop is a no-op live tracer. It's there to
// catch changes in the tracing interface, as well as
// for testing live tracing performance. Can be removed
// as soon as we have a real live tracer.
type noop struct{}

func newNoopTracer(_ json.RawMessage) (*tracing.Hooks, error) {
	t := &noop{}
	return &tracing.Hooks{
		CaptureTxStart:        t.CaptureTxStart,
		CaptureTxEnd:          t.CaptureTxEnd,
		CaptureStart:          t.CaptureStart,
		CaptureEnd:            t.CaptureEnd,
		CaptureEnter:          t.CaptureEnter,
		CaptureExit:           t.CaptureExit,
		CaptureState:          t.CaptureState,
		CaptureFault:          t.CaptureFault,
		CaptureKeccakPreimage: t.CaptureKeccakPreimage,
		OnGasChange:           t.OnGasChange,
		OnBlockchainInit:      t.OnBlockchainInit,
		OnBlockStart:          t.OnBlockStart,
		OnBlockEnd:            t.OnBlockEnd,
		OnSkippedBlock:        t.OnSkippedBlock,
		OnGenesisBlock:        t.OnGenesisBlock,
		OnBalanceChange:       t.OnBalanceChange,
		OnNonceChange:         t.OnNonceChange,
		OnCodeChange:          t.OnCodeChange,
		OnStorageChange:       t.OnStorageChange,
		OnLog:                 t.OnLog,
	}, nil
}

// CaptureStart implements the EVMLogger interface to initialize the tracing operation.
func (t *noop) CaptureStart(from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

// CaptureEnd is called after the call finishes to finalize the tracing.
func (t *noop) CaptureEnd(output []byte, gasUsed uint64, err error, reverted bool) {
}

// CaptureState implements the EVMLogger interface to trace a single step of VM execution.
func (t *noop) CaptureState(pc uint64, op tracing.OpCode, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
}

// CaptureFault implements the EVMLogger interface to trace an execution fault.
func (t *noop) CaptureFault(pc uint64, op tracing.OpCode, gas, cost uint64, _ tracing.OpContext, depth int, err error) {
}

// CaptureKeccakPreimage is called during the KECCAK256 opcode.
func (t *noop) CaptureKeccakPreimage(hash common.Hash, data []byte) {}

// CaptureEnter is called when EVM enters a new scope (via call, create or selfdestruct).
func (t *noop) CaptureEnter(typ tracing.OpCode, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

// CaptureExit is called when EVM exits a scope, even if the scope didn't
// execute any code.
func (t *noop) CaptureExit(output []byte, gasUsed uint64, err error, reverted bool) {
}

func (t *noop) CaptureTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
}

func (t *noop) CaptureTxEnd(receipt *types.Receipt, err error) {
}

func (t *noop) OnBlockStart(ev tracing.BlockEvent) {
}

func (t *noop) OnBlockEnd(err error) {
}

func (t *noop) OnSkippedBlock(ev tracing.BlockEvent) {}

func (t *noop) OnBlockchainInit(chainConfig *params.ChainConfig) {
}

func (t *noop) OnGenesisBlock(b *types.Block, alloc types.GenesisAlloc) {
}

func (t *noop) OnBalanceChange(a common.Address, prev, new *big.Int, reason tracing.BalanceChangeReason) {
}

func (t *noop) OnNonceChange(a common.Address, prev, new uint64) {
}

func (t *noop) OnCodeChange(a common.Address, prevCodeHash common.Hash, prev []byte, codeHash common.Hash, code []byte) {
}

func (t *noop) OnStorageChange(a common.Address, k, prev, new common.Hash) {
}

func (t *noop) OnLog(l *types.Log) {

}

func (t *noop) OnGasChange(old, new uint64, reason tracing.GasChangeReason) {
}
