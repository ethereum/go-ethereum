package live

import (
	"encoding/json"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/tracers"
	"github.com/ethereum/go-ethereum/params"
)

func init() {
	tracers.LiveDirectory.Register("noop", newNoopTracer)
}

// noop is a no-op live tracer. It's there to
// catch changes in the tracing interface, as well as
// for testing live tracing performance. Can be removed
// as soon as we have a real live tracer.
type noop struct{}

func newNoopTracer(_ json.RawMessage) (*tracing.Hooks, error) {
	t := &noop{}
	return &tracing.Hooks{
		OnTxStart:        t.OnTxStart,
		OnTxEnd:          t.OnTxEnd,
		OnEnter:          t.OnEnter,
		OnExit:           t.OnExit,
		OnOpcode:         t.OnOpcode,
		OnFault:          t.OnFault,
		OnGasChange:      t.OnGasChange,
		OnBlockchainInit: t.OnBlockchainInit,
		OnBlockStart:     t.OnBlockStart,
		OnBlockEnd:       t.OnBlockEnd,
		OnSkippedBlock:   t.OnSkippedBlock,
		OnGenesisBlock:   t.OnGenesisBlock,
		OnBalanceChange:  t.OnBalanceChange,
		OnNonceChange:    t.OnNonceChange,
		OnCodeChange:     t.OnCodeChange,
		OnStorageChange:  t.OnStorageChange,
		OnLog:            t.OnLog,
	}, nil
}

func (t *noop) OnOpcode(pc uint64, op byte, gas, cost uint64, scope tracing.OpContext, rData []byte, depth int, err error) {
}

func (t *noop) OnFault(pc uint64, op byte, gas, cost uint64, _ tracing.OpContext, depth int, err error) {
}

func (t *noop) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
}

func (t *noop) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
}

func (t *noop) OnTxStart(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
}

func (t *noop) OnTxEnd(receipt *types.Receipt, err error) {
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
