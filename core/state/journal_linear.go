// Copyright 2016 The go-ethereum Authors
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

package state

import (
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// journalEntry is a modification entry in the state change linear journal that can be
// reverted on demand.
type journalEntry interface {
	// revert undoes the changes introduced by this entry.
	revert(*StateDB)

	// dirtied returns the Ethereum address modified by this entry.
	dirtied() *common.Address

	// copy returns a deep-copied entry.
	copy() journalEntry
}

// linearJournal contains the list of state modifications applied since the last state
// commit. These are tracked to be able to be reverted in the case of an execution
// exception or request for reversal.
type linearJournal struct {
	entries []journalEntry         // Current changes tracked by the linearJournal
	dirties map[common.Address]int // Dirty accounts and the number of changes

	revisions []int // sequence of indexes to points in time designating snapshots
}

// compile-time interface check
var _ journal = (*linearJournal)(nil)

// newLinearJournal creates a new initialized linearJournal.
func newLinearJournal() *linearJournal {
	s := &linearJournal{
		dirties: make(map[common.Address]int),
	}
	s.snapshot() // create snaphot zero
	return s
}

// reset clears the journal, after this operation the journal can be used anew.
// It is semantically similar to calling 'newJournal', but the underlying slices
// can be reused.
func (j *linearJournal) reset() {
	j.entries = j.entries[:0]
	j.revisions = j.revisions[:0]
	clear(j.dirties)
	j.snapshot()
}

func (j linearJournal) dirtyAccounts() []common.Address {
	dirty := make([]common.Address, 0, len(j.dirties))
	// flatten into list
	for addr := range j.dirties {
		dirty = append(dirty, addr)
	}
	return dirty
}

// snapshot starts a new journal scope which can be reverted or discarded.
func (j *linearJournal) snapshot() {
	j.revisions = append(j.revisions, len(j.entries))
}

// revertSnapshot reverts all state changes made since the last call to snapshot().
func (j *linearJournal) revertSnapshot(s *StateDB) {
	id := len(j.revisions) - 1
	if id < 0 {
		j.snapshot()
		return
	}
	revision := j.revisions[id]
	// Replay the linearJournal to undo changes and remove invalidated snapshots
	j.revertTo(s, revision)
	j.revisions = j.revisions[:id]
	if id == 0 {
		j.snapshot()
	}
}

// discardSnapshot removes the latest snapshot; after calling this
// method, it is no longer possible to revert to that particular snapshot, the
// changes are considered part of the parent scope.
func (j *linearJournal) discardSnapshot() {
	id := len(j.revisions) - 1
	if id <= 0 {
		// If a transaction is applied successfully, the statedb.Finalize will
		// end by clearing and resetting the journal. Invoking a discardSnapshot
		// afterwards will land here: calling discard on an empty journal.
		// This is fine
		return
	}
	j.revisions = j.revisions[:id]
}

// append inserts a new modification entry to the end of the change linearJournal.
func (j *linearJournal) append(entry journalEntry) {
	j.entries = append(j.entries, entry)
	if addr := entry.dirtied(); addr != nil {
		j.dirties[*addr]++
	}
}

// revert undoes a batch of journalled modifications along with any reverted
// dirty handling too.
func (j *linearJournal) revertTo(statedb *StateDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		// Undo the changes made by the operation
		j.entries[i].revert(statedb)

		// Drop any dirty tracking induced by the change
		if addr := j.entries[i].dirtied(); addr != nil {
			if j.dirties[*addr]--; j.dirties[*addr] == 0 {
				delete(j.dirties, *addr)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}

// dirty explicitly sets an address to dirty, even if the change entries would
// otherwise suggest it as clean. This method is an ugly hack to handle the RIPEMD
// precompile consensus exception.
func (j *linearJournal) dirty(addr common.Address) {
	j.dirties[addr]++
}

// length returns the current number of entries in the linearJournal.
func (j *linearJournal) length() int {
	return len(j.entries)
}

// copy returns a deep-copied journal.
func (j *linearJournal) copy() journal {
	entries := make([]journalEntry, 0, j.length())
	for i := 0; i < j.length(); i++ {
		entries = append(entries, j.entries[i].copy())
	}
	return &linearJournal{
		entries:   entries,
		dirties:   maps.Clone(j.dirties),
		revisions: slices.Clone(j.revisions),
	}
}

func (j *linearJournal) logChange(txHash common.Hash) {
	j.append(addLogChange{txhash: txHash})
}

func (j *linearJournal) createObject(addr common.Address) {
	j.append(createObjectChange{account: addr})
}

func (j *linearJournal) createContract(addr common.Address, account *types.StateAccount) {
	j.append(createContractChange{account: addr})
}

func (j *linearJournal) destruct(addr common.Address, account *types.StateAccount) {
	j.append(selfDestructChange{account: addr})
}

func (j *linearJournal) storageChange(addr common.Address, key, prev, origin common.Hash) {
	j.append(storageChange{
		account:   addr,
		key:       key,
		prevvalue: prev,
		origvalue: origin,
	})
}

func (j *linearJournal) transientStateChange(addr common.Address, key, prev common.Hash) {
	j.append(transientStorageChange{
		account:  addr,
		key:      key,
		prevalue: prev,
	})
}

func (j *linearJournal) refundChange(previous uint64) {
	j.append(refundChange{prev: previous})
}

func (j *linearJournal) balanceChange(addr common.Address, account *types.StateAccount, destructed, newContract bool) {
	j.append(balanceChange{
		account: addr,
		prev:    account.Balance.Clone(),
	})
}

func (j *linearJournal) setCode(address common.Address, account *types.StateAccount, prevCode []byte) {
	j.append(codeChange{
		account:  address,
		prevCode: prevCode,
	})
}

func (j *linearJournal) nonceChange(address common.Address, account *types.StateAccount, destructed, newContract bool) {
	j.append(nonceChange{
		account: address,
		prev:    account.Nonce,
	})
}

func (j *linearJournal) touchChange(address common.Address, account *types.StateAccount, destructed, newContract bool) {
	j.append(touchChange{
		account: address,
	})
	if address == ripemd {
		// Explicitly put it in the dirty-cache, which is otherwise generated from
		// flattened journals.
		j.dirty(address)
	}
}

func (j *linearJournal) accessListAddAccount(addr common.Address) {
	j.append(accessListAddAccountChange{addr})
}

func (j *linearJournal) accessListAddSlot(addr common.Address, slot common.Hash) {
	j.append(accessListAddSlotChange{
		address: addr,
		slot:    slot,
	})
}

type (
	// Changes to the account trie.
	createObjectChange struct {
		account common.Address
	}
	// createContractChange represents an account becoming a contract-account.
	// This event happens prior to executing initcode. The linearJournal-event simply
	// manages the created-flag, in order to allow same-tx destruction.
	createContractChange struct {
		account common.Address
	}
	selfDestructChange struct {
		account common.Address
	}

	// Changes to individual accounts.
	balanceChange struct {
		account common.Address
		prev    *uint256.Int
	}
	nonceChange struct {
		account common.Address
		prev    uint64
	}
	storageChange struct {
		account   common.Address
		key       common.Hash
		prevvalue common.Hash
		origvalue common.Hash
	}
	codeChange struct {
		account  common.Address
		prevCode []byte
	}

	// Changes to other state values.
	refundChange struct {
		prev uint64
	}
	addLogChange struct {
		txhash common.Hash
	}
	touchChange struct {
		account common.Address
	}

	// Changes to the access list
	accessListAddAccountChange struct {
		address common.Address
	}
	accessListAddSlotChange struct {
		address common.Address
		slot    common.Hash
	}

	// Changes to transient storage
	transientStorageChange struct {
		account       common.Address
		key, prevalue common.Hash
	}
)

func (ch createObjectChange) revert(s *StateDB) {
	delete(s.stateObjects, ch.account)
}

func (ch createObjectChange) dirtied() *common.Address {
	return &ch.account
}

func (ch createObjectChange) copy() journalEntry {
	return createObjectChange{
		account: ch.account,
	}
}

func (ch createContractChange) revert(s *StateDB) {
	s.getStateObject(ch.account).newContract = false
}

func (ch createContractChange) dirtied() *common.Address {
	// This method returns nil, since the transformation from non-contract to
	// contract is not an operation which has an effect on the trie:
	// it does not make the account part of the dirty-set.
	// Creating the account (createObject) or setting the code (setCode)
	// however, do, and are.
	return nil
}

func (ch createContractChange) copy() journalEntry {
	return createContractChange{
		account: ch.account,
	}
}

func (ch selfDestructChange) revert(s *StateDB) {
	obj := s.getStateObject(ch.account)
	if obj != nil {
		obj.selfDestructed = false
	}
}

func (ch selfDestructChange) dirtied() *common.Address {
	return &ch.account
}

func (ch selfDestructChange) copy() journalEntry {
	return selfDestructChange{
		account: ch.account,
	}
}

var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) revert(s *StateDB) {
}

func (ch touchChange) dirtied() *common.Address {
	return &ch.account
}

func (ch touchChange) copy() journalEntry {
	return touchChange{
		account: ch.account,
	}
}

func (ch balanceChange) revert(s *StateDB) {
	s.getStateObject(ch.account).setBalance(ch.prev)
}

func (ch balanceChange) dirtied() *common.Address {
	return &ch.account
}

func (ch balanceChange) copy() journalEntry {
	return balanceChange{
		account: ch.account,
		prev:    new(uint256.Int).Set(ch.prev),
	}
}

func (ch nonceChange) revert(s *StateDB) {
	s.getStateObject(ch.account).setNonce(ch.prev)
}

func (ch nonceChange) dirtied() *common.Address {
	return &ch.account
}

func (ch nonceChange) copy() journalEntry {
	return nonceChange{
		account: ch.account,
		prev:    ch.prev,
	}
}

func (ch codeChange) revert(s *StateDB) {
	s.getStateObject(ch.account).setCode(crypto.Keccak256Hash(ch.prevCode), ch.prevCode)
}

func (ch codeChange) dirtied() *common.Address {
	return &ch.account
}

func (ch codeChange) copy() journalEntry {
	return codeChange{
		account:  ch.account,
		prevCode: ch.prevCode}
}

func (ch storageChange) revert(s *StateDB) {
	s.getStateObject(ch.account).setState(ch.key, ch.prevvalue, ch.origvalue)
}

func (ch storageChange) dirtied() *common.Address {
	return &ch.account
}

func (ch storageChange) copy() journalEntry {
	return storageChange{
		account:   ch.account,
		key:       ch.key,
		prevvalue: ch.prevvalue,
	}
}

func (ch transientStorageChange) revert(s *StateDB) {
	s.setTransientState(ch.account, ch.key, ch.prevalue)
}

func (ch transientStorageChange) dirtied() *common.Address {
	return nil
}

func (ch transientStorageChange) copy() journalEntry {
	return transientStorageChange{
		account:  ch.account,
		key:      ch.key,
		prevalue: ch.prevalue,
	}
}

func (ch refundChange) revert(s *StateDB) {
	s.refund = ch.prev
}

func (ch refundChange) dirtied() *common.Address {
	return nil
}

func (ch refundChange) copy() journalEntry {
	return refundChange{
		prev: ch.prev,
	}
}

func (ch addLogChange) revert(s *StateDB) {
	logs := s.logs[ch.txhash]
	if len(logs) == 1 {
		delete(s.logs, ch.txhash)
	} else {
		s.logs[ch.txhash] = logs[:len(logs)-1]
	}
	s.logSize--
}

func (ch addLogChange) dirtied() *common.Address {
	return nil
}

func (ch addLogChange) copy() journalEntry {
	return addLogChange{
		txhash: ch.txhash,
	}
}

func (ch accessListAddAccountChange) revert(s *StateDB) {
	/*
		One important invariant here, is that whenever a (addr, slot) is added, if the
		addr is not already present, the add causes two linearJournal entries:
		- one for the address,
		- one for the (address,slot)
		Therefore, when unrolling the change, we can always blindly delete the
		(addr) at this point, since no storage adds can remain when come upon
		a single (addr) change.
	*/
	s.accessList.DeleteAddress(ch.address)
}

func (ch accessListAddAccountChange) dirtied() *common.Address {
	return nil
}

func (ch accessListAddAccountChange) copy() journalEntry {
	return accessListAddAccountChange{
		address: ch.address,
	}
}

func (ch accessListAddSlotChange) revert(s *StateDB) {
	s.accessList.DeleteSlot(ch.address, ch.slot)
}

func (ch accessListAddSlotChange) dirtied() *common.Address {
	return nil
}

func (ch accessListAddSlotChange) copy() journalEntry {
	return accessListAddSlotChange{
		address: ch.address,
		slot:    ch.slot,
	}
}
