// Copyright 2024 The go-ethereum Authors
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
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type revision struct {
	id           int
	journalIndex int
}

// journal is a state change journal to be wrapped around a tracer.
// It will emit the state change hooks with reverse values when a call reverts.
type journal struct {
	entries []entry
	hooks   *Hooks

	validRevisions []revision
	nextRevisionId int
	curRevisionId  int
}

type entry interface {
	revert(tracer *Hooks)
}

// WrapWithJournal wraps the given tracer with a journaling layer.
func WrapWithJournal(hooks *Hooks) (*Hooks, error) {
	if hooks == nil {
		return nil, fmt.Errorf("wrapping nil tracer")
	}
	// No state change to journal.
	if hooks.OnBalanceChange == nil && hooks.OnNonceChange == nil && hooks.OnCodeChange == nil && hooks.OnStorageChange == nil {
		return hooks, nil
	}
	var (
		j       = &journal{entries: make([]entry, 0), hooks: hooks}
		wrapped = &Hooks{
			OnTxEnd: j.OnTxEnd,
			OnEnter: j.OnEnter,
			OnExit:  j.OnExit,
		}
	)
	// State change hooks.
	if hooks.OnBalanceChange != nil {
		wrapped.OnBalanceChange = j.OnBalanceChange
	}
	if hooks.OnNonceChange != nil {
		wrapped.OnNonceChange = j.OnNonceChange
	}
	if hooks.OnCodeChange != nil {
		wrapped.OnCodeChange = j.OnCodeChange
	}
	if hooks.OnStorageChange != nil {
		wrapped.OnStorageChange = j.OnStorageChange
	}
	// Pass through the remaining hooks.
	if hooks.OnTxStart != nil {
		wrapped.OnTxStart = hooks.OnTxStart
	}
	if hooks.OnOpcode != nil {
		wrapped.OnOpcode = hooks.OnOpcode
	}
	if hooks.OnFault != nil {
		wrapped.OnFault = hooks.OnFault
	}
	if hooks.OnGasChange != nil {
		wrapped.OnGasChange = hooks.OnGasChange
	}
	if hooks.OnBlockchainInit != nil {
		wrapped.OnBlockchainInit = hooks.OnBlockchainInit
	}
	if hooks.OnClose != nil {
		wrapped.OnClose = hooks.OnClose
	}
	if hooks.OnBlockStart != nil {
		wrapped.OnBlockStart = hooks.OnBlockStart
	}
	if hooks.OnBlockEnd != nil {
		wrapped.OnBlockEnd = hooks.OnBlockEnd
	}
	if hooks.OnSkippedBlock != nil {
		wrapped.OnSkippedBlock = hooks.OnSkippedBlock
	}
	if hooks.OnGenesisBlock != nil {
		wrapped.OnGenesisBlock = hooks.OnGenesisBlock
	}
	if hooks.OnReorg != nil {
		wrapped.OnReorg = hooks.OnReorg
	}
	if hooks.OnSystemCallStart != nil {
		wrapped.OnSystemCallStart = hooks.OnSystemCallStart
	}
	if hooks.OnSystemCallEnd != nil {
		wrapped.OnSystemCallEnd = hooks.OnSystemCallEnd
	}
	if hooks.OnLog != nil {
		wrapped.OnLog = hooks.OnLog
	}
	if hooks.OnBalanceRead != nil {
		wrapped.OnBalanceRead = hooks.OnBalanceRead
	}
	if hooks.OnNonceRead != nil {
		wrapped.OnNonceRead = hooks.OnNonceRead
	}
	if hooks.OnCodeRead != nil {
		wrapped.OnCodeRead = hooks.OnCodeRead
	}
	if hooks.OnCodeSizeRead != nil {
		wrapped.OnCodeSizeRead = hooks.OnCodeSizeRead
	}
	if hooks.OnCodeHashRead != nil {
		wrapped.OnCodeHashRead = hooks.OnCodeHashRead
	}
	if hooks.OnStorageRead != nil {
		wrapped.OnStorageRead = hooks.OnStorageRead
	}
	if hooks.OnBlockHashRead != nil {
		wrapped.OnBlockHashRead = hooks.OnBlockHashRead
	}
	return wrapped, nil
}

// reset clears the journal, after this operation the journal can be used anew.
// It is semantically similar to calling 'NewJournal', but the underlying slices
// can be reused.
func (j *journal) reset() {
	j.entries = j.entries[:0]
	j.validRevisions = j.validRevisions[:0]
	j.nextRevisionId = 0
}

// snapshot returns an identifier for the current revision of the state.
func (j *journal) snapshot() int {
	id := j.nextRevisionId
	j.nextRevisionId++
	j.validRevisions = append(j.validRevisions, revision{id, j.length()})
	return id
}

// revertToSnapshot reverts all state changes made since the given revision.
func (j *journal) revertToSnapshot(revid int, hooks *Hooks) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(j.validRevisions), func(i int) bool {
		return j.validRevisions[i].id >= revid
	})
	if idx == len(j.validRevisions) || j.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := j.validRevisions[idx].journalIndex

	// Replay the journal to undo changes and remove invalidated snapshots
	j.revert(hooks, snapshot)
	j.validRevisions = j.validRevisions[:idx]
}

// revert undoes a batch of journaled modifications.
func (j *journal) revert(hooks *Hooks, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		// Undo the changes made by the operation
		j.entries[i].revert(hooks)
	}
	j.entries = j.entries[:snapshot]
}

// length returns the current number of entries in the journal.
func (j *journal) length() int {
	return len(j.entries)
}

func (j *journal) OnTxEnd(receipt *types.Receipt, err error) {
	j.reset()
}

func (j *journal) OnEnter(depth int, typ byte, from common.Address, to common.Address, input []byte, gas uint64, value *big.Int) {
	j.curRevisionId = j.snapshot()
	if j.hooks != nil && j.hooks.OnEnter != nil {
		j.hooks.OnEnter(depth, typ, from, to, input, gas, value)
	}
}

func (j *journal) OnExit(depth int, output []byte, gasUsed uint64, err error, reverted bool) {
	if reverted {
		j.revertToSnapshot(j.curRevisionId, j.hooks)
	}
	j.curRevisionId--
	if j.hooks != nil && j.hooks.OnExit != nil {
		j.hooks.OnExit(depth, output, gasUsed, err, reverted)
	}
}

func (j *journal) OnBalanceChange(addr common.Address, prev, new *big.Int, reason BalanceChangeReason) {
	j.entries = append(j.entries, balanceChange{addr: addr, prev: prev, new: new})
	if j.hooks != nil && j.hooks.OnBalanceChange != nil {
		j.hooks.OnBalanceChange(addr, prev, new, reason)
	}
}

func (j *journal) OnNonceChange(addr common.Address, prev, new uint64) {
	j.entries = append(j.entries, nonceChange{addr: addr, prev: prev, new: new})
	if j.hooks != nil && j.hooks.OnNonceChange != nil {
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
	if j.hooks != nil && j.hooks.OnCodeChange != nil {
		j.hooks.OnCodeChange(addr, prevCodeHash, prevCode, codeHash, code)
	}
}

func (j *journal) OnStorageChange(addr common.Address, slot common.Hash, prev, new common.Hash) {
	j.entries = append(j.entries, storageChange{addr: addr, slot: slot, prev: prev, new: new})
	if j.hooks != nil && j.hooks.OnStorageChange != nil {
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
	if hooks.OnNonceChange != nil {
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
