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
	"fmt"
	"slices"
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

type revision struct {
	id           int
	journalIndex int
}

// journalMutationKind indicates the type of account mutation.
type journalMutationKind uint8

const (
	// journalMutationKindNone is the zero value returned by mutation() for
	// entries that don't carry a tracked account mutation. The accompanying
	// bool is false in that case; callers must gate on it before using the
	// kind.
	journalMutationKindNone journalMutationKind = iota
	journalMutationKindTouch
	journalMutationKindCreate
	journalMutationKindSelfDestruct
	journalMutationKindBalance
	journalMutationKindNonce
	journalMutationKindCode
	journalMutationKindStorage
	journalMutationKindCount // sentinel, must stay last
)

type journalMutationCounts [journalMutationKindCount]int

// journalMutationState tracks, per account, both the per-kind count of mutation
// entries currently present in the journal and the pre-tx value of each
// metadata field captured on its first touch (balance/nonce/code).
// The *Set flags indicate whether the corresponding field has been mutated
// at least once in the current tx window; they are cleared when all entries
// of that kind are reverted. Storage slots are tracked elsewhere.
type journalMutationState struct {
	counts journalMutationCounts

	balance    *uint256.Int
	balanceSet bool
	nonce      uint64
	nonceSet   bool
	code       []byte
	codeSet    bool
}

func (s *journalMutationState) add(kind journalMutationKind) {
	s.counts.add(kind)
}

// remove drops one occurrence of the given mutation kind. It returns a flag
// indicating whether no entries of any kind remain.
func (s *journalMutationState) remove(kind journalMutationKind) bool {
	if s.counts.remove(kind) {
		// No entries of this kind remain for this account; drop the
		// corresponding stashed original so the state mirrors the
		// live mutation set.
		s.clearKind(kind)
	}
	return s.counts == (journalMutationCounts{})
}

// clearKind drops the stashed original for the given mutation kind. It is
// invoked during revert once no journal entries of that kind remain for the
// account. Kinds that don't correspond to a tracked metadata field are no-ops.
func (s *journalMutationState) clearKind(kind journalMutationKind) {
	switch kind {
	case journalMutationKindBalance:
		s.balance = nil
		s.balanceSet = false
	case journalMutationKindNonce:
		s.nonce = 0
		s.nonceSet = false
	case journalMutationKindCode:
		s.code = nil
		s.codeSet = false
	}
}

func (s *journalMutationState) copy() *journalMutationState {
	cpy := *s
	if s.balance != nil {
		cpy.balance = new(uint256.Int).Set(s.balance)
	}
	if s.code != nil {
		cpy.code = slices.Clone(s.code)
	}
	return &cpy
}

func (c *journalMutationCounts) add(kind journalMutationKind) {
	c[kind]++
}

func (c *journalMutationCounts) remove(kind journalMutationKind) bool {
	c[kind]--
	return c[kind] == 0
}

// journalEntry is a modification entry in the state change journal that can be
// reverted on demand.
type journalEntry interface {
	// revert undoes the changes introduced by this journal entry.
	revert(*StateDB)

	// mutation returns the account mutation introduced by this entry.
	// It indicates false if no tracked account mutation was made.
	mutation() (common.Address, journalMutationKind, bool)

	// copy returns a deep-copied journal entry.
	copy() journalEntry
}

// stashBalance records prev as the pre-tx balance of addr, iff this is the
// first balance touch seen in the current tx. Subsequent balance writes are
// ignored so the stored value remains the true pre-tx original.
func (j *journal) stashBalance(addr common.Address, prev *uint256.Int) {
	s := j.mutationStateFor(addr)
	if s.balanceSet {
		return
	}
	// The balance is already deep-copied and safe to hold the object here.
	s.balance = prev
	s.balanceSet = true
}

// stashNonce records prev as the pre-tx nonce of addr on first touch.
func (j *journal) stashNonce(addr common.Address, prev uint64) {
	s := j.mutationStateFor(addr)
	if s.nonceSet {
		return
	}
	s.nonce = prev
	s.nonceSet = true
}

// stashCode records prev as the pre-tx code of addr on first touch.
func (j *journal) stashCode(addr common.Address, prev []byte) {
	s := j.mutationStateFor(addr)
	if s.codeSet {
		return
	}
	// The code is already deep-copied in the StateDB, safe to
	// hold the reference here.
	s.code = prev
	s.codeSet = true
}

// mutationStateFor returns the mutation state for addr, creating an empty one
// if absent.
func (j *journal) mutationStateFor(addr common.Address) *journalMutationState {
	s := j.mutations[addr]
	if s == nil {
		s = new(journalMutationState)
		j.mutations[addr] = s
	}
	return s
}

// journal contains the list of state modifications applied since the last state
// commit. These are tracked to be able to be reverted in the case of an execution
// exception or request for reversal.
type journal struct {
	entries   []journalEntry                           // Current changes tracked by the journal
	mutations map[common.Address]*journalMutationState // Per-account mutation kinds and pre-tx originals

	validRevisions []revision
	nextRevisionId int
}

// newJournal creates a new initialized journal.
func newJournal() *journal {
	return &journal{
		mutations: make(map[common.Address]*journalMutationState),
	}
}

// reset clears the journal, after this operation the journal can be used anew.
// It is semantically similar to calling 'newJournal', but the underlying slices
// can be reused.
func (j *journal) reset() {
	j.entries = j.entries[:0]
	j.validRevisions = j.validRevisions[:0]
	clear(j.mutations)
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
func (j *journal) revertToSnapshot(revid int, s *StateDB) {
	// Find the snapshot in the stack of valid snapshots.
	idx := sort.Search(len(j.validRevisions), func(i int) bool {
		return j.validRevisions[i].id >= revid
	})
	if idx == len(j.validRevisions) || j.validRevisions[idx].id != revid {
		panic(fmt.Errorf("revision id %v cannot be reverted", revid))
	}
	snapshot := j.validRevisions[idx].journalIndex

	// Replay the journal to undo changes and remove invalidated snapshots
	j.revert(s, snapshot)
	j.validRevisions = j.validRevisions[:idx]
}

// append inserts a new modification entry to the end of the change journal.
func (j *journal) append(entry journalEntry) {
	j.entries = append(j.entries, entry)
	if addr, kind, dirty := entry.mutation(); dirty {
		state := j.mutations[addr]
		if state == nil {
			state = new(journalMutationState)
			j.mutations[addr] = state
		}
		state.add(kind)
	}
}

// revert undoes a batch of journalled modifications along with any reverted
// mutation tracking too.
func (j *journal) revert(statedb *StateDB, snapshot int) {
	for i := len(j.entries) - 1; i >= snapshot; i-- {
		// Undo the changes made by the operation
		j.entries[i].revert(statedb)

		// Drop any mutation tracking induced by the change.
		if addr, kind, dirty := j.entries[i].mutation(); dirty {
			state := j.mutations[addr]
			if state == nil {
				panic(fmt.Errorf("journal mutation tracking missing for %x", addr[:]))
			}
			if state.remove(kind) {
				delete(j.mutations, addr)
			}
		}
	}
	j.entries = j.entries[:snapshot]
}

// ripemdMagic explicitly keeps RIPEMD160 in the mutation set with a touch change.
//
// Ethereum Mainnet contains an old empty-account touch/revert quirk for address
// 0x03. If we only relied on the journal entry above, the revert path would
// remove the account from the mutation set together with the touch.
//
// Keep an explicit touch marker so tx finalisation still sees RIPEMD160
// on the mutation pass when replaying that historical case.
func (j *journal) ripemdMagic() {
	state := j.mutations[ripemd]
	if state == nil {
		state = new(journalMutationState)
		j.mutations[ripemd] = state
	}
	state.add(journalMutationKindTouch)
}

// length returns the current number of entries in the journal.
func (j *journal) length() int {
	return len(j.entries)
}

// copy returns a deep-copied journal.
func (j *journal) copy() *journal {
	entries := make([]journalEntry, 0, j.length())
	for i := 0; i < j.length(); i++ {
		entries = append(entries, j.entries[i].copy())
	}
	mutations := make(map[common.Address]*journalMutationState, len(j.mutations))
	for addr, state := range j.mutations {
		mutations[addr] = state.copy()
	}
	return &journal{
		entries:        entries,
		mutations:      mutations,
		validRevisions: slices.Clone(j.validRevisions),
		nextRevisionId: j.nextRevisionId,
	}
}

func (j *journal) logChange(txHash common.Hash) {
	j.append(addLogChange{txhash: txHash})
}

func (j *journal) createObject(addr common.Address) {
	j.append(createObjectChange{account: addr})
}

func (j *journal) createContract(addr common.Address) {
	j.append(createContractChange{account: addr})
}

func (j *journal) destruct(addr common.Address) {
	j.append(selfDestructChange{account: addr})
}

func (j *journal) storageChange(addr common.Address, key, prev, origin common.Hash) {
	j.append(storageChange{
		account:   addr,
		key:       key,
		prevvalue: prev,
		origvalue: origin,
	})
}

func (j *journal) transientStateChange(addr common.Address, key, prev common.Hash) {
	j.append(transientStorageChange{
		account:  addr,
		key:      key,
		prevalue: prev,
	})
}

func (j *journal) refundChange(previous uint64) {
	j.append(refundChange{prev: previous})
}

func (j *journal) balanceChange(addr common.Address, previous *uint256.Int) {
	prev := previous.Clone()
	j.stashBalance(addr, prev)
	j.append(balanceChange{
		account: addr,
		prev:    prev,
	})
}

func (j *journal) setCode(address common.Address, prevCode []byte) {
	j.stashCode(address, prevCode)
	j.append(codeChange{
		account:  address,
		prevCode: prevCode,
	})
}

func (j *journal) nonceChange(address common.Address, prev uint64) {
	j.stashNonce(address, prev)
	j.append(nonceChange{
		account: address,
		prev:    prev,
	})
}

func (j *journal) touchChange(address common.Address) {
	j.append(touchChange{
		account: address,
	})
	if address == ripemd {
		// Preserve the historical RIPEMD160 precompile consensus exception.
		//
		// Mainnet contains an old empty-account touch/revert quirk for address
		// 0x03. If we only relied on the journal entry above, the revert path
		// would remove the account from the dirty set together with the touch.
		// Keep an explicit dirty marker so tx finalisation still sees the
		// account on the dirty pass when replaying that historical case.
		//
		// This does not force deletion by itself: Finalise will still delete the
		// account only if the state object is present at tx end and qualifies for
		// deletion there.
		j.ripemdMagic()
	}
}

func (j *journal) accessListAddAccount(addr common.Address) {
	j.append(accessListAddAccountChange{addr})
}

func (j *journal) accessListAddSlot(addr common.Address, slot common.Hash) {
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
	// This event happens prior to executing initcode. The journal-event simply
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

func (ch createObjectChange) mutation() (common.Address, journalMutationKind, bool) {
	return ch.account, journalMutationKindCreate, true
}

func (ch createObjectChange) copy() journalEntry {
	return createObjectChange{
		account: ch.account,
	}
}

func (ch createContractChange) revert(s *StateDB) {
	s.getStateObject(ch.account).newContract = false
}

func (ch createContractChange) mutation() (common.Address, journalMutationKind, bool) {
	return common.Address{}, journalMutationKindNone, false
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

func (ch selfDestructChange) mutation() (common.Address, journalMutationKind, bool) {
	return ch.account, journalMutationKindSelfDestruct, true
}

func (ch selfDestructChange) copy() journalEntry {
	return selfDestructChange{
		account: ch.account,
	}
}

var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")

func (ch touchChange) revert(s *StateDB) {
}

func (ch touchChange) mutation() (common.Address, journalMutationKind, bool) {
	return ch.account, journalMutationKindTouch, true
}

func (ch touchChange) copy() journalEntry {
	return touchChange{
		account: ch.account,
	}
}

func (ch balanceChange) revert(s *StateDB) {
	s.getStateObject(ch.account).setBalance(ch.prev)
}

func (ch balanceChange) mutation() (common.Address, journalMutationKind, bool) {
	return ch.account, journalMutationKindBalance, true
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

func (ch nonceChange) mutation() (common.Address, journalMutationKind, bool) {
	return ch.account, journalMutationKindNonce, true
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

func (ch codeChange) mutation() (common.Address, journalMutationKind, bool) {
	return ch.account, journalMutationKindCode, true
}

func (ch codeChange) copy() journalEntry {
	return codeChange{
		account:  ch.account,
		prevCode: ch.prevCode,
	}
}

func (ch storageChange) revert(s *StateDB) {
	s.getStateObject(ch.account).setState(ch.key, ch.prevvalue, ch.origvalue)
}

func (ch storageChange) mutation() (common.Address, journalMutationKind, bool) {
	return ch.account, journalMutationKindStorage, true
}

func (ch storageChange) copy() journalEntry {
	return storageChange{
		account:   ch.account,
		key:       ch.key,
		prevvalue: ch.prevvalue,
		origvalue: ch.origvalue,
	}
}

func (ch transientStorageChange) revert(s *StateDB) {
	s.setTransientState(ch.account, ch.key, ch.prevalue)
}

func (ch transientStorageChange) mutation() (common.Address, journalMutationKind, bool) {
	return common.Address{}, journalMutationKindNone, false
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

func (ch refundChange) mutation() (common.Address, journalMutationKind, bool) {
	return common.Address{}, journalMutationKindNone, false
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

func (ch addLogChange) mutation() (common.Address, journalMutationKind, bool) {
	return common.Address{}, journalMutationKindNone, false
}

func (ch addLogChange) copy() journalEntry {
	return addLogChange{
		txhash: ch.txhash,
	}
}

func (ch accessListAddAccountChange) revert(s *StateDB) {
	/*
		One important invariant here, is that whenever a (addr, slot) is added, if the
		addr is not already present, the add causes two journal entries:
		- one for the address,
		- one for the (address,slot)
		Therefore, when unrolling the change, we can always blindly delete the
		(addr) at this point, since no storage adds can remain when come upon
		a single (addr) change.
	*/
	s.accessList.DeleteAddress(ch.address)
}

func (ch accessListAddAccountChange) mutation() (common.Address, journalMutationKind, bool) {
	return common.Address{}, journalMutationKindNone, false
}

func (ch accessListAddAccountChange) copy() journalEntry {
	return accessListAddAccountChange{
		address: ch.address,
	}
}

func (ch accessListAddSlotChange) revert(s *StateDB) {
	s.accessList.DeleteSlot(ch.address, ch.slot)
}

func (ch accessListAddSlotChange) mutation() (common.Address, journalMutationKind, bool) {
	return common.Address{}, journalMutationKindNone, false
}

func (ch accessListAddSlotChange) copy() journalEntry {
	return accessListAddSlotChange{
		address: ch.address,
		slot:    ch.slot,
	}
}
