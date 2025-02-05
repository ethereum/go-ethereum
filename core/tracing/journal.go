// Copyright 2025 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package tracing

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// journal is a state change journal to be wrapped around a tracer.
// It will emit the state change hooks with reverse values when a call reverts.
type journal struct {
	hooks     *Hooks
	entries   []entry
	revisions []int
}

type entry interface {
	revert(tracer *Hooks)
}

// WrapWithJournal wraps the given tracer with a journaling layer.
func WrapWithJournal(hooks *Hooks) (*Hooks, error) {
	if hooks == nil {
		return nil, fmt.Errorf("wrapping nil tracer")
	}
	// No state change to journal, return the wrapped hooks as is
	if hooks.OnBalanceChange == nil && hooks.OnNonceChange == nil && hooks.OnNonceChangeV2 == nil && hooks.OnCodeChange == nil && hooks.OnStorageChange == nil {
		return hooks, nil
	}
	if hooks.OnNonceChange != nil && hooks.OnNonceChangeV2 != nil {
		return nil, fmt.Errorf("cannot have both OnNonceChange and OnNonceChangeV2")
	}

	// Create a new Hooks instance and copy all hooks
	wrapped := *hooks

	// Create journal
	j := &journal{hooks: hooks}
	// Scope hooks need to be re-implemented.
	wrapped.OnTxEnd = j.OnTxEnd
	wrapped.OnEnter = j.OnEnter
	wrapped.OnExit = j.OnExit
	// Wrap state change hooks.
	if hooks.OnBalanceChange != nil {
		wrapped.OnBalanceChange = j.OnBalanceChange
	}
	if hooks.OnNonceChange != nil || hooks.OnNonceChangeV2 != nil {
		// Regardless of which hook version is used in the tracer,
		// the journal will want to capture the nonce change reason.
		wrapped.OnNonceChangeV2 = j.OnNonceChangeV2
		// A precaution to ensure EVM doesn't call both hooks.
		wrapped.OnNonceChange = nil
	}
	if hooks.OnCodeChange != nil {
		wrapped.OnCodeChange = j.OnCodeChange
	}
	if hooks.OnStorageChange != nil {
		wrapped.OnStorageChange = j.OnStorageChange
	}

	return &wrapped, nil
}

// reset clears the journal, after this operation the journal can be used anew.
// It is semantically similar to calling 'NewJournal', but the underlying slices
// can be reused.
func (j *journal) reset() {
	j.entries = j.entries[:0]
	j.revisions = j.revisions[:0]
}

// snapshot records a revision and stores it to the revision stack.
func (j *journal) snapshot() {
	rev := len(j.entries)
	j.revisions = append(j.revisions, rev)
}

// revert reverts all state changes up to the last tracked revision.
func (j *journal) revert(hooks *Hooks) {
	// Replay the journal entries above the last revision to undo changes,
	// then remove the reverted changes from the journal.
	rev := j.revisions[len(j.revisions)-1]
	for i := len(j.entries) - 1; i >= rev; i-- {
		j.entries[i].revert(hooks)
	}
	j.entries = j.entries[:rev]
	j.popRevision()
}

// popRevision removes an item from the revision stack. This basically forgets about
// the last call to snapshot() and moves to the one prior.
func (j *journal) popRevision() {
	j.revisions = j.revisions[:len(j.revisions)-1]
}

// OnTxEnd resets the journal since each transaction has its own EVM call stack.
func (j *journal) OnTxEnd(receipt *types.Receipt, err error) {
	j.reset()
	if j.hooks.OnTxEnd != nil {
		j.hooks.OnTxEnd(receipt, err)
	}
}

// OnEnter is invoked for each EVM call frame and records a journal revision.
func (j *journal) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	j.snapshot()
	if j.hooks.OnEnter != nil {
		j.hooks.OnEnter(depth, typ, from, to, input, gas, value)
	}
}

// OnExit is invoked when an EVM call frame ends.
// If the call has reverted, all state changes made by that frame are undone.
// If the call did not revert, we forget about changes in that revision.
func (j *journal) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if reverted {
		j.revert(j.hooks)
	} else {
		j.popRevision()
	}
	if j.hooks.OnExit != nil {
		j.hooks.OnExit(depth, output, gasUsed, err, reverted)
	}
}

func (j *journal) OnBalanceChange(addr common.Address, prev, new *big.Int, reason BalanceChangeReason) {
	j.entries = append(j.entries, balanceChange{addr: addr, prev: prev, new: new})
	if j.hooks.OnBalanceChange != nil {
		j.hooks.OnBalanceChange(addr, prev, new, reason)
	}
}

func (j *journal) OnNonceChangeV2(addr common.Address, prev, new uint64, reason NonceChangeReason) {
	// When a contract is created, the nonce of the creator is incremented.
	// This change is not reverted when the creation fails.
	if reason != NonceChangeContractCreator {
		j.entries = append(j.entries, nonceChange{addr: addr, prev: prev, new: new})
	}
	if j.hooks.OnNonceChangeV2 != nil {
		j.hooks.OnNonceChangeV2(addr, prev, new, reason)
	} else if j.hooks.OnNonceChange != nil {
		j.hooks.OnNonceChange(addr, prev, new)
	}
}

func (j *journal) OnCodeChange(addr common.Address, prevCodeHash common.Hash, prevCode []byte, codeHash common.Hash, code []byte) {
	j.entries = append(j.entries, codeChange{
		addr:         addr,
		prevCodeHash: prevCodeHash,
		prevCode:     prevCode,
		newCodeHash:  codeHash,
		newCode:      code,
	})
	if j.hooks.OnCodeChange != nil {
		j.hooks.OnCodeChange(addr, prevCodeHash, prevCode, codeHash, code)
	}
}

func (j *journal) OnStorageChange(addr common.Address, slot common.Hash, prev, new common.Hash) {
	j.entries = append(j.entries, storageChange{addr: addr, slot: slot, prev: prev, new: new})
	if j.hooks.OnStorageChange != nil {
		j.hooks.OnStorageChange(addr, slot, prev, new)
	}
}

type (
	balanceChange struct {
		addr common.Address
		prev *big.Int
		new  *big.Int
	}

	nonceChange struct {
		addr common.Address
		prev uint64
		new  uint64
	}

	codeChange struct {
		addr         common.Address
		prevCodeHash common.Hash
		prevCode     []byte
		newCodeHash  common.Hash
		newCode      []byte
	}

	storageChange struct {
		addr common.Address
		slot common.Hash
		prev common.Hash
		new  common.Hash
	}
)

func (b balanceChange) revert(hooks *Hooks) {
	if hooks.OnBalanceChange != nil {
		hooks.OnBalanceChange(b.addr, b.new, b.prev, BalanceChangeRevert)
	}
}

func (n nonceChange) revert(hooks *Hooks) {
	if hooks.OnNonceChangeV2 != nil {
		hooks.OnNonceChangeV2(n.addr, n.new, n.prev, NonceChangeRevert)
	} else if hooks.OnNonceChange != nil {
		hooks.OnNonceChange(n.addr, n.new, n.prev)
	}
}

func (c codeChange) revert(hooks *Hooks) {
	if hooks.OnCodeChange != nil {
		hooks.OnCodeChange(c.addr, c.newCodeHash, c.newCode, c.prevCodeHash, c.prevCode)
	}
}

func (s storageChange) revert(hooks *Hooks) {
	if hooks.OnStorageChange != nil {
		hooks.OnStorageChange(s.addr, s.slot, s.new, s.prev)
	}
}
