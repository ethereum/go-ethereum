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

// BlockAccessListTracer constructs an EIP-7928 block access list from the
// execution of a block
type BlockAccessListTracer struct {
	// this is a set of access lists for each call scope. the overall block access lists
	// is accrued at index 0, while the access lists of various nested execution
	// scopes are in the proceeding indices.
	// When an execution scope terminates in a non-reverting fashion, the changes are
	// merged into the access list of the parent scope.
	blockTxCount      int
	accessList        *bal.ConstructionBlockAccessList
	balIdx            uint16
	accessListBuilder *bal.AccessListBuilder

	// mutations and state reads from currently-executing bal index
	idxMutations *bal.StateDiff
	idxReads     bal.StateAccesses
}

// NewBlockAccessListTracer returns an BlockAccessListTracer and a set of hooks
func NewBlockAccessListTracer(startIdx int) (*BlockAccessListTracer, *tracing.Hooks) {
	balTracer := &BlockAccessListTracer{
		accessList: bal.NewConstructionBlockAccessList(),
		//balIdx:            uint16(startIdx),
		accessListBuilder: bal.NewAccessListBuilder(),
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
		OnColdAccountRead:    balTracer.OnColdAccountRead,
		OnColdStorageRead:    balTracer.OnColdStorageRead,
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
func (a *BlockAccessListTracer) AccessList() *bal.ConstructionBlockAccessList {
	return a.accessList
}

func (a *BlockAccessListTracer) OnPreTxExecutionDone() {
	a.idxMutations, a.idxReads = a.accessListBuilder.FinaliseIdxChanges()
	a.accessList.Apply(0, a.idxMutations, a.idxReads)
	a.accessListBuilder = bal.NewAccessListBuilder()
	a.balIdx++
}

// TODO: I don't like that AccessList and this do slightly different things,
// and that they mutate the access list builder (not apparent in the naming of the methods)
//
// ^ idea: add Finalize() which returns the diff/accesses, also accumulating them in the BAL.
// AccessList just returns the constructed BAL.
func (a *BlockAccessListTracer) IdxChanges() (*bal.StateDiff, bal.StateAccesses) {
	return a.idxMutations, a.idxReads
}

func (a *BlockAccessListTracer) TxEndHook(receipt *types.Receipt, err error) {
	a.idxMutations, a.idxReads = a.accessListBuilder.FinaliseIdxChanges()
	a.accessList.Apply(a.balIdx, a.idxMutations, a.idxReads)
	a.accessListBuilder = bal.NewAccessListBuilder()
	a.balIdx++
}

func (a *BlockAccessListTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	a.accessListBuilder.EnterScope()
}

func (a *BlockAccessListTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	a.accessListBuilder.ExitScope(reverted)
}

func (a *BlockAccessListTracer) OnCodeChange(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte, reason tracing.CodeChangeReason) {
	// TODO: if we don't have this equality check, some tests fail.  should be investigated.
	// probably the tracer shouldn't invoke code change if the code didn't actually change tho.
	if prevCodeHash != codeHash {
		a.accessListBuilder.CodeChange(addr, prevCode, code)
	}
}

func (a *BlockAccessListTracer) OnSelfDestruct(addr common.Address) {
	a.accessListBuilder.SelfDestruct(addr)
}

func (a *BlockAccessListTracer) OnBlockFinalization() {
	a.idxMutations, a.idxReads = a.accessListBuilder.FinaliseIdxChanges()
	a.accessList.Apply(a.balIdx, a.idxMutations, a.idxReads)
	a.accessListBuilder = bal.NewAccessListBuilder()
}

func (a *BlockAccessListTracer) OnBalanceChange(addr common.Address, prevBalance, newBalance *big.Int, _ tracing.BalanceChangeReason) {
	newU256 := new(uint256.Int).SetBytes(newBalance.Bytes())
	prevU256 := new(uint256.Int).SetBytes(prevBalance.Bytes())
	a.accessListBuilder.BalanceChange(addr, prevU256, newU256)
}

func (a *BlockAccessListTracer) OnNonceChange(addr common.Address, prev uint64, new uint64, reason tracing.NonceChangeReason) {
	a.accessListBuilder.NonceChange(addr, prev, new)
}

func (a *BlockAccessListTracer) OnColdStorageRead(addr common.Address, key common.Hash) {
	a.accessListBuilder.StorageRead(addr, key)
}

func (a *BlockAccessListTracer) OnColdAccountRead(addr common.Address) {
	a.accessListBuilder.AccountRead(addr)
}

func (a *BlockAccessListTracer) OnStorageChange(addr common.Address, slot common.Hash, prev common.Hash, new common.Hash) {
	a.accessListBuilder.StorageWrite(addr, slot, prev, new)
}
