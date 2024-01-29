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

package state

import (
	"bytes"
	"fmt"
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/holiman/uint256"
)

var (
	_ journal = (*sparseJournal)(nil)
)

// journalAccount represents the 'journable state' of a types.Account.
// Which means, all the normal fields except storage root, but also with a
// destruction-flag.
type journalAccount struct {
	nonce       uint64
	balance     uint256.Int
	codeHash    []byte // nil == emptyCodeHAsh
	destructed  bool
	newContract bool
}

type addrSlot struct {
	addr common.Address
	slot common.Hash
}

type doubleHash struct {
	origin common.Hash
	prev   common.Hash
}

// scopedJournal represents all changes within a single callscope. These changes
// are either all reverted, or all committed -- they cannot be partially applied.
type scopedJournal struct {
	accountChanges map[common.Address]*journalAccount
	refund         int64
	logs           []common.Hash

	accessListAddresses []common.Address
	accessListAddrSlots []addrSlot

	storageChanges  map[common.Address]map[common.Hash]doubleHash
	tStorageChanges map[common.Address]map[common.Hash]common.Hash
}

func newScopedJournal() *scopedJournal {
	return &scopedJournal{
		refund: -1,
	}
}

func (j *scopedJournal) deepCopy() *scopedJournal {
	var cpy = &scopedJournal{
		// The accountChanges copy will copy the pointers to
		// journalAccount objects: thus not actually deep copy those
		// objects. That is fine: we never mutate journalAccount.
		accountChanges:      maps.Clone(j.accountChanges),
		refund:              j.refund,
		logs:                slices.Clone(j.logs),
		accessListAddresses: slices.Clone(j.accessListAddresses),
		accessListAddrSlots: slices.Clone(j.accessListAddrSlots),
	}
	if j.storageChanges != nil {
		cpy.storageChanges = make(map[common.Address]map[common.Hash]doubleHash)
		for addr, changes := range j.storageChanges {
			cpy.storageChanges[addr] = maps.Clone(changes)
		}
	}
	if j.tStorageChanges != nil {
		cpy.tStorageChanges = make(map[common.Address]map[common.Hash]common.Hash)
		for addr, changes := range j.tStorageChanges {
			cpy.tStorageChanges[addr] = maps.Clone(changes)
		}
	}
	return cpy
}

func (j *scopedJournal) journalRefundChange(prev uint64) {
	if j.refund == -1 {
		// We convert from uint64 to int64 here, so that we can use -1
		// to represent "no previous value set".
		// Treating refund as int64 is fine, there's no possibility for
		// refund to ever exceed maxInt64.
		j.refund = int64(prev)
	}
}

// journalAccountChange is the common shared implementation for all account-changes.
// These changes all fall back to this method:
// - balance change
// - nonce change
// - destruct-change
// - code change
// - touch change
// - creation change (in this case, the account is nil)
func (j *scopedJournal) journalAccountChange(address common.Address, account *types.StateAccount, destructed, newContract bool) {
	if j.accountChanges == nil {
		j.accountChanges = make(map[common.Address]*journalAccount)
	}
	// If the account has already been journalled, we're done here
	if _, ok := j.accountChanges[address]; ok {
		return
	}
	if account == nil {
		j.accountChanges[address] = nil // created now, previously non-existent
		return
	}
	ja := &journalAccount{
		nonce:       account.Nonce,
		balance:     *account.Balance,
		destructed:  destructed,
		newContract: newContract,
	}
	if !bytes.Equal(account.CodeHash, types.EmptyCodeHash[:]) {
		ja.codeHash = account.CodeHash
	}
	j.accountChanges[address] = ja
}

func (j *scopedJournal) journalLog(txHash common.Hash) {
	j.logs = append(j.logs, txHash)
}

func (j *scopedJournal) journalAccessListAddAccount(addr common.Address) {
	j.accessListAddresses = append(j.accessListAddresses, addr)
}

func (j *scopedJournal) journalAccessListAddSlot(addr common.Address, slot common.Hash) {
	j.accessListAddrSlots = append(j.accessListAddrSlots, addrSlot{addr, slot})
}

func (j *scopedJournal) journalSetState(addr common.Address, key, prev, origin common.Hash) {
	if j.storageChanges == nil {
		j.storageChanges = make(map[common.Address]map[common.Hash]doubleHash)
	}
	changes, ok := j.storageChanges[addr]
	if !ok {
		changes = make(map[common.Hash]doubleHash)
		j.storageChanges[addr] = changes
	}
	// Do not overwrite a previous value!
	if _, ok := changes[key]; !ok {
		changes[key] = doubleHash{origin: origin, prev: prev}
	}
}

func (j *scopedJournal) journalSetTransientState(addr common.Address, key, prev common.Hash) {
	if j.tStorageChanges == nil {
		j.tStorageChanges = make(map[common.Address]map[common.Hash]common.Hash)
	}
	changes, ok := j.tStorageChanges[addr]
	if !ok {
		changes = make(map[common.Hash]common.Hash)
		j.tStorageChanges[addr] = changes
	}
	// Do not overwrite a previous value!
	if _, ok := changes[key]; !ok {
		changes[key] = prev
	}
}

func (j *scopedJournal) revert(s *StateDB) {
	// Revert refund
	if j.refund != -1 {
		s.refund = uint64(j.refund)
	}
	// Revert storage changes
	for addr, changes := range j.storageChanges {
		obj := s.getStateObject(addr)
		for key, val := range changes {
			obj.setState(key, val.prev, val.origin)
		}
	}
	// Revert t-store changes
	for addr, changes := range j.tStorageChanges {
		for key, val := range changes {
			s.setTransientState(addr, key, val)
		}
	}

	// Revert changes to accounts
	for addr, data := range j.accountChanges {
		if data == nil { // Reverting a create
			delete(s.stateObjects, addr)
			continue
		}
		obj := s.getStateObject(addr)
		obj.setNonce(data.nonce)
		// Setting 'code' to nil means it will be loaded from disk
		// next time it is needed. We avoid nilling it unless required
		journalHash := data.codeHash
		if data.codeHash == nil {
			if !bytes.Equal(obj.CodeHash(), types.EmptyCodeHash[:]) {
				obj.setCode(types.EmptyCodeHash, nil)
			}
		} else {
			if !bytes.Equal(obj.CodeHash(), journalHash) {
				obj.setCode(common.BytesToHash(data.codeHash), nil)
			}
		}
		obj.setBalance(&data.balance)
		obj.selfDestructed = data.destructed
		obj.newContract = data.newContract
	}
	// Revert logs
	for _, txhash := range j.logs {
		logs := s.logs[txhash]
		if len(logs) == 1 {
			delete(s.logs, txhash)
		} else {
			s.logs[txhash] = logs[:len(logs)-1]
		}
		s.logSize--
	}
	// Revert access list additions
	for i := len(j.accessListAddrSlots) - 1; i >= 0; i-- {
		item := j.accessListAddrSlots[i]
		s.accessList.DeleteSlot(item.addr, item.slot)
	}
	for i := len(j.accessListAddresses) - 1; i >= 0; i-- {
		s.accessList.DeleteAddress(j.accessListAddresses[i])
	}
}

func (j *scopedJournal) merge(parent *scopedJournal) {
	if parent.refund == -1 {
		parent.refund = j.refund
	}
	// Revert changes to accounts
	if parent.accountChanges == nil {
		parent.accountChanges = j.accountChanges
	} else {
		for addr, data := range j.accountChanges {
			if _, present := parent.accountChanges[addr]; present {
				// Nothing to do here, it's already stored in parent scope
				continue
			}
			parent.accountChanges[addr] = data
		}
	}
	// Revert logs
	parent.logs = append(parent.logs, j.logs...)

	// Revert access list additions
	parent.accessListAddrSlots = append(parent.accessListAddrSlots, j.accessListAddrSlots...)
	parent.accessListAddresses = append(parent.accessListAddresses, j.accessListAddresses...)

	if parent.storageChanges == nil {
		parent.storageChanges = j.storageChanges
	} else {
		// Merge storage changes
		for addr, changes := range j.storageChanges {
			prevChanges, ok := parent.storageChanges[addr]
			if !ok {
				parent.storageChanges[addr] = changes
				continue
			}
			for k, v := range changes {
				if _, ok := prevChanges[k]; !ok {
					prevChanges[k] = v
				}
			}
		}
	}
	if parent.tStorageChanges == nil {
		parent.tStorageChanges = j.tStorageChanges
	} else {
		// Revert t-store changes
		for addr, changes := range j.tStorageChanges {
			prevChanges, ok := parent.tStorageChanges[addr]
			if !ok {
				parent.tStorageChanges[addr] = changes
				continue
			}
			for k, v := range changes {
				if _, ok := prevChanges[k]; !ok {
					prevChanges[k] = v
				}
			}
		}
	}
}

func (j *scopedJournal) addDirtyAccounts(set map[common.Address]any) {
	// Changes due to account changes
	for addr := range j.accountChanges {
		set[addr] = []interface{}{}
	}
	// Changes due to storage changes
	for addr := range j.storageChanges {
		set[addr] = []interface{}{}
	}
}

// sparseJournal contains the list of state modifications applied since the last state
// commit. These are tracked to be able to be reverted in the case of an execution
// exception or request for reversal.
type sparseJournal struct {
	entries   []*scopedJournal // Current changes tracked by the journal
	ripeMagic bool
}

// newJournal creates a new initialized journal.
func newSparseJournal() *sparseJournal {
	s := new(sparseJournal)
	s.snapshot() // create snaphot zero
	return s
}

// reset clears the journal, after this operation the journal can be used
// anew. It is semantically similar to calling 'newJournal', but the underlying
// slices can be reused
func (j *sparseJournal) reset() {
	j.entries = j.entries[:0]
	j.snapshot()
}

func (j *sparseJournal) copy() journal {
	cp := &sparseJournal{
		entries: make([]*scopedJournal, 0, len(j.entries)),
	}
	for _, entry := range j.entries {
		cp.entries = append(cp.entries, entry.deepCopy())
	}
	return cp
}

// snapshot returns an identifier for the current revision of the state.
// OBS: A call to Snapshot is _required_ in order to initialize the journalling,
// invoking the journal-methods without having invoked Snapshot will lead to
// panic.
func (j *sparseJournal) snapshot() int {
	id := len(j.entries)
	j.entries = append(j.entries, newScopedJournal())
	return id
}

// revertToSnapshot reverts all state changes made since the given revision.
func (j *sparseJournal) revertToSnapshot(id int, s *StateDB) {
	if id >= len(j.entries) {
		panic(fmt.Errorf("revision id %v cannot be reverted", id))
	}
	// Revert the entries sequentially
	for i := len(j.entries) - 1; i >= id; i-- {
		entry := j.entries[i]
		entry.revert(s)
	}
	j.entries = j.entries[:id]
}

func (j *sparseJournal) DiscardSnapshot(id int) {
	if id == 0 {
		return
	}
	// here we must merge the 'id' with it's parent.
	want := len(j.entries) - 1
	have := id
	if want != have {
		if want == 0 && id == 1 {
			// If a transcation is applied successfully, the statedb.Finalize will
			// end by clearing and resetting the journal. Invoking a DiscardSnapshot
			// afterwards will lead us here.
			// Let's not panic, but it's ok to complain a bit
			log.Error("Extraneous invocation to discard snapshot")
			return
		} else {
			panic(fmt.Sprintf("journalling error, want discard(%d), have discard(%d)", want, have))
		}
	}
	entry := j.entries[id]
	parent := j.entries[id-1]
	entry.merge(parent)
	j.entries = j.entries[:id]
}

func (j *sparseJournal) journalAccountChange(addr common.Address, account *types.StateAccount, destructed, newContract bool) {
	j.entries[len(j.entries)-1].journalAccountChange(addr, account, destructed, newContract)
}

func (j *sparseJournal) nonceChange(addr common.Address, account *types.StateAccount, destructed, newContract bool) {
	j.journalAccountChange(addr, account, destructed, newContract)
}

func (j *sparseJournal) balanceChange(addr common.Address, account *types.StateAccount, destructed, newContract bool) {
	j.journalAccountChange(addr, account, destructed, newContract)
}

func (j *sparseJournal) setCode(addr common.Address, account *types.StateAccount) {
	j.journalAccountChange(addr, account, false, true)
}

func (j *sparseJournal) createObject(addr common.Address) {
	// Creating an account which is destructed, hence already exists, is not
	// allowed, hence we know destructed == 'false'.
	// Also, if we are creating the account now, it cannot yet be a
	// newContract (that might come later)
	j.journalAccountChange(addr, nil, false, false)
}

func (j *sparseJournal) createContract(addr common.Address, account *types.StateAccount) {
	// Creating an account which is destructed, hence already exists, is not
	// allowed, hence we know it to be 'false'.
	// Also: if we create the contract now, it cannot be previously created
	j.journalAccountChange(addr, account, false, false)
}

func (j *sparseJournal) destruct(addr common.Address, account *types.StateAccount) {
	// destructing an already destructed account must not be journalled. Hence we
	// know it to be 'false'.
	// Also: if we're allowed to destruct it, it must be `newContract:true`, OR
	// the concept of newContract is unused and moot.
	j.journalAccountChange(addr, account, false, true)
}

// var ripemd = common.HexToAddress("0000000000000000000000000000000000000003")
func (j *sparseJournal) touchChange(addr common.Address, account *types.StateAccount, destructed, newContract bool) {
	j.journalAccountChange(addr, account, destructed, newContract)
	if addr == ripemd {
		// Explicitly put it in the dirty-cache one extra time. Ripe magic.
		j.ripeMagic = true
	}
}

func (j *sparseJournal) logChange(txHash common.Hash) {
	j.entries[len(j.entries)-1].journalLog(txHash)
}

func (j *sparseJournal) refundChange(prev uint64) {
	j.entries[len(j.entries)-1].journalRefundChange(prev)
}

func (j *sparseJournal) accessListAddAccount(addr common.Address) {
	j.entries[len(j.entries)-1].journalAccessListAddAccount(addr)
}

func (j *sparseJournal) accessListAddSlot(addr common.Address, slot common.Hash) {
	j.entries[len(j.entries)-1].journalAccessListAddSlot(addr, slot)
}

func (j *sparseJournal) storageChange(addr common.Address, key, prev, origin common.Hash) {
	j.entries[len(j.entries)-1].journalSetState(addr, key, prev, origin)
}

func (j *sparseJournal) transientStateChange(addr common.Address, key, prev common.Hash) {
	j.entries[len(j.entries)-1].journalSetTransientState(addr, key, prev)
}

func (j *sparseJournal) dirtyAccounts() []common.Address {
	// The dirty-set should encompass all layers
	var dirty = make(map[common.Address]any)
	for _, scope := range j.entries {
		scope.addDirtyAccounts(dirty)
	}
	if j.ripeMagic {
		dirty[ripemd] = []interface{}{}
	}
	var dirtyList = make([]common.Address, 0, len(dirty))
	for addr := range dirty {
		dirtyList = append(dirtyList, addr)
	}
	return dirtyList
}
