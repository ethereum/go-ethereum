package core

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/holiman/uint256"
	"math/big"
)

// BlockAccessListTracer constructs an EIP-7928 block access list from the
// execution of a block
type BlockAccessListTracer struct {
	// this is a set of access lists for each call scope. the overall block access lists
	// is accrued at index 0, while the access lists of various nested execution
	// scopes are in the proceeding indices.
	// When an execution scope terminates in a non-reverting fashion, the changes are
	// merged into the access list of the parent scope.
	callAccessLists []*bal.ConstructionBlockAccessList
	txIdx           uint16

	// if non-nil, it's the address of the account which just self-destructed.
	// reset at the end of the call-scope which self-destructed.
	selfdestructedAccount *common.Address
}

// NewBlockAccessListTracer returns an BlockAccessListTracer and a set of hooks
func NewBlockAccessListTracer() (*BlockAccessListTracer, *tracing.Hooks) {
	balTracer := &BlockAccessListTracer{
		callAccessLists: []*bal.ConstructionBlockAccessList{bal.NewConstructionBlockAccessList()},
		txIdx:           0,
	}
	hooks := &tracing.Hooks{
		OnTxEnd:           balTracer.TxEndHook,
		OnTxStart:         balTracer.TxStartHook,
		OnEnter:           balTracer.OnEnter,
		OnExit:            balTracer.OnExit,
		OnCodeChangeV2:    balTracer.OnCodeChange,
		OnBalanceChange:   balTracer.OnBalanceChange,
		OnNonceChange:     balTracer.OnNonceChange,
		OnStorageChange:   balTracer.OnStorageChange,
		OnColdAccountRead: balTracer.OnColdAccountRead,
		OnColdStorageRead: balTracer.OnColdStorageRead,
	}
	return balTracer, hooks
}

// AccessList returns the constructed access list
func (a *BlockAccessListTracer) AccessList() *bal.ConstructionBlockAccessList {
	return a.callAccessLists[0]
}

func (a *BlockAccessListTracer) TxEndHook(receipt *types.Receipt, err error) {
	a.txIdx++
}

func (a *BlockAccessListTracer) TxStartHook(vm *tracing.VMContext, tx *types.Transaction, from common.Address) {
	if a.txIdx == 0 {
		a.txIdx++
	}
}

func (a *BlockAccessListTracer) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	a.callAccessLists = append(a.callAccessLists, bal.NewConstructionBlockAccessList())
}

func (a *BlockAccessListTracer) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	// any self-destructed accounts must have been created in the same transaction
	// so there is no difference between the pre/post tx state of those accounts
	if a.selfdestructedAccount != nil {
		delete(a.callAccessLists[len(a.callAccessLists)-1].Accounts, *a.selfdestructedAccount)
	}
	if !reverted {
		parentAccessList := a.callAccessLists[len(a.callAccessLists)-2]
		scopeAccessList := a.callAccessLists[len(a.callAccessLists)-1]
		parentAccessList.Merge(scopeAccessList)
	}

	a.callAccessLists = a.callAccessLists[:len(a.callAccessLists)-1]
}

func (a *BlockAccessListTracer) OnCodeChange(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte, reason tracing.CodeChangeReason) {
	if reason == tracing.CodeChangeSelfDestruct {
		a.selfdestructedAccount = &addr
		return
	}
	a.callAccessLists[len(a.callAccessLists)-1].CodeChange(addr, uint16(a.txIdx), code)
}

func (a *BlockAccessListTracer) OnBalanceChange(addr common.Address, prevBalance, newBalance *big.Int, _ tracing.BalanceChangeReason) {
	a.callAccessLists[len(a.callAccessLists)-1].BalanceChange(a.txIdx, addr, new(uint256.Int).SetBytes(newBalance.Bytes()))
}

func (a *BlockAccessListTracer) OnNonceChange(addr common.Address, prev uint64, new uint64) {
	a.callAccessLists[len(a.callAccessLists)-1].NonceChange(addr, a.txIdx, new)
}

func (a *BlockAccessListTracer) OnColdStorageRead(addr common.Address, key common.Hash) {
	a.callAccessLists[len(a.callAccessLists)-1].StorageRead(addr, key)
}

func (a *BlockAccessListTracer) OnColdAccountRead(addr common.Address) {
	a.callAccessLists[len(a.callAccessLists)-1].AccountRead(addr)
}

func (a *BlockAccessListTracer) OnStorageChange(addr common.Address, slot common.Hash, prev common.Hash, new common.Hash) {
	a.callAccessLists[len(a.callAccessLists)-1].StorageWrite(a.txIdx, addr, slot, new)
}
