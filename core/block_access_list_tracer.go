package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/holiman/uint256"
	"math/big"
)

type accountPrestate struct {
	balance *uint256.Int
	nonce   *uint64
	code    []byte
}

// BlockAccessListTracer is a tracer which gathers state accesses/mutations
// from the execution of a block.  It is used for constructing and verifying
// EIP-7928 block access lists.
type BlockAccessListTracer struct {
	builder *bal.BlockAccessListBuilder

	// the access list index that changes are currently being recorded into
	balIdx uint16
}

// NewBlockAccessListTracer returns an BlockAccessListTracer and a set of hooks
func NewBlockAccessListTracer() (*BlockAccessListTracer, *tracing.Hooks) {
	balTracer := &BlockAccessListTracer{
		builder: bal.NewConstructionBlockAccessList(),
	}
	hooks := &tracing.Hooks{
		OnBlockFinalization:  balTracer.OnBlockFinalization,
		OnPreTxExecutionDone: balTracer.OnPreTxExecutionDone,
		OnTxEnd:              balTracer.TxEndHook,
		OnEnter:              balTracer.OnEnter,
		OnExit:               balTracer.OnExit,
		OnCodeChangeV2:       balTracer.OnCodeChange,
		OnBalanceChange:      balTracer.OnBalanceChange,
		OnNonceChangeV2:      balTracer.OnNonceChange,
		OnStorageChange:      balTracer.OnStorageChange,
		OnStorageRead:        balTracer.OnStorageRead,
		OnAccountRead:        balTracer.OnAcountRead,
		OnSelfDestructChange: balTracer.OnSelfDestruct,
	}
	wrappedHooks, err := tracing.WrapWithJournal(hooks)
	if err != nil {
		panic(err) // TODO: ....
	}
	return balTracer, wrappedHooks
}

// AccessList returns the constructed access list.
// It is assumed that this is only called after all the block state changes
// have been executed and the block has been finalized.
func (a *BlockAccessListTracer) AccessList() *bal.BlockAccessListBuilder {
	return a.builder
}

func (a *BlockAccessListTracer) OnPreTxExecutionDone() {
	a.builder.FinalisePendingChanges(0)
	a.balIdx++
}

func (a *BlockAccessListTracer) TxEndHook(receipt *types.Receipt, err error) {
	a.builder.FinalisePendingChanges(a.balIdx)
	a.balIdx++
}

func (a *BlockAccessListTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	a.builder.EnterScope()
}

func (a *BlockAccessListTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	a.builder.ExitScope(reverted)
}

func (a *BlockAccessListTracer) OnCodeChange(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte, reason tracing.CodeChangeReason) {
	// TODO: if we don't have this equality check, some tests fail.  should be investigated.
	// probably the tracer shouldn't invoke code change if the code didn't actually change tho.
	if prevCodeHash != codeHash {
		a.builder.CodeChange(addr, prevCode, code)
	}
}

func (a *BlockAccessListTracer) OnSelfDestruct(addr common.Address) {
	a.builder.SelfDestruct(addr)
}

func (a *BlockAccessListTracer) OnBlockFinalization() {
	a.builder.FinalisePendingChanges(a.balIdx)
}

func (a *BlockAccessListTracer) OnBalanceChange(addr common.Address, prevBalance, newBalance *big.Int, _ tracing.BalanceChangeReason) {
	newU256 := new(uint256.Int).SetBytes(newBalance.Bytes())
	prevU256 := new(uint256.Int).SetBytes(prevBalance.Bytes())
	a.builder.BalanceChange(addr, prevU256, newU256)
}

func (a *BlockAccessListTracer) OnNonceChange(addr common.Address, prev uint64, new uint64, reason tracing.NonceChangeReason) {
	a.builder.NonceChange(addr, prev, new)
}

func (a *BlockAccessListTracer) OnStorageRead(addr common.Address, key common.Hash) {
	a.builder.StorageRead(addr, key)
}

func (a *BlockAccessListTracer) OnAcountRead(addr common.Address) {
	a.builder.AccountRead(addr)
}

func (a *BlockAccessListTracer) OnStorageChange(addr common.Address, slot common.Hash, prev common.Hash, new common.Hash) {
	a.builder.StorageWrite(addr, slot, prev, new)
}
