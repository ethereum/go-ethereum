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

package bal

import (
	"bytes"
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// ConstructionAccountAccess contains post-block account state for mutations as well as
// all storage keys that were read during execution. It is used when building block
// access list during execution.
type ConstructionAccountAccess struct {
	// StorageWrites is the post-state values of an account's storage slots
	// that were modified in a block, keyed by the slot key and the tx index
	// where the modification occurred.
	StorageWrites map[common.Hash]map[uint32]common.Hash `json:"storageWrites,omitempty"`

	// StorageReads is the set of slot keys that were accessed during block
	// execution.
	//
	// Storage slots which are both read and written (with changed values)
	// appear only in StorageWrites.
	StorageReads map[common.Hash]struct{} `json:"storageReads,omitempty"`

	// BalanceChanges contains the post-transaction balances of an account,
	// keyed by transaction indices where it was changed.
	BalanceChanges map[uint32]*uint256.Int `json:"balanceChanges,omitempty"`

	// NonceChanges contains the post-state nonce values of an account keyed
	// by tx index.
	NonceChanges map[uint32]uint64 `json:"nonceChanges,omitempty"`

	// CodeChange contains the post-state contract code of an account keyed
	// by tx index.
	CodeChange map[uint32][]byte `json:"codeChange,omitempty"`

	// restores marks, keyed by tx index, the entries which were set back to
	// the value they held before that index. They only matter to Merge, which
	// drops the entries they shadow. It is nil unless there is any.
	restores map[uint32]*constructionRestores
}

// constructionRestores is the set of entries of an account which ended a
// finalisation scope with the value they held before the block access index,
// after a previous scope of the same index might have recorded a change.
type constructionRestores struct {
	balance, nonce, code bool
	slots                []common.Hash
}

// restoresAt returns the restore markers of the account for the given index,
// creating them if absent.
func (a *ConstructionAccountAccess) restoresAt(txIdx uint32) *constructionRestores {
	if a.restores == nil {
		a.restores = make(map[uint32]*constructionRestores)
	}
	r := a.restores[txIdx]
	if r == nil {
		r = new(constructionRestores)
		a.restores[txIdx] = r
	}
	return r
}

// dropStorageWrite removes the write of the slot at the given index. A slot
// left with no write at all is demoted to a read.
func (a *ConstructionAccountAccess) dropStorageWrite(txIdx uint32, key common.Hash) {
	writes, ok := a.StorageWrites[key]
	if !ok {
		return
	}
	delete(writes, txIdx)
	if len(writes) == 0 {
		delete(a.StorageWrites, key)
		a.StorageReads[key] = struct{}{}
	}
}

// NewConstructionAccountAccess initializes the account access object.
func NewConstructionAccountAccess() *ConstructionAccountAccess {
	return &ConstructionAccountAccess{
		StorageWrites:  make(map[common.Hash]map[uint32]common.Hash),
		StorageReads:   make(map[common.Hash]struct{}),
		BalanceChanges: make(map[uint32]*uint256.Int),
		NonceChanges:   make(map[uint32]uint64),
		CodeChange:     make(map[uint32][]byte),
	}
}

// ConstructionBlockAccessList contains post-block modified state and some state accessed
// in execution (account addresses and storage keys).
type ConstructionBlockAccessList struct {
	Accounts map[common.Address]*ConstructionAccountAccess
}

// NewConstructionBlockAccessList instantiates an empty access list.
func NewConstructionBlockAccessList() *ConstructionBlockAccessList {
	return &ConstructionBlockAccessList{
		Accounts: make(map[common.Address]*ConstructionAccountAccess),
	}
}

// AccountRead records the address of an account that has been read during execution.
func (b *ConstructionBlockAccessList) AccountRead(addr common.Address) {
	if _, ok := b.Accounts[addr]; !ok {
		b.Accounts[addr] = NewConstructionAccountAccess()
	}
}

// StorageRead records a storage key read during execution.
func (b *ConstructionBlockAccessList) StorageRead(address common.Address, key common.Hash) {
	if _, ok := b.Accounts[address]; !ok {
		b.Accounts[address] = NewConstructionAccountAccess()
	}
	if _, ok := b.Accounts[address].StorageWrites[key]; ok {
		return
	}
	b.Accounts[address].StorageReads[key] = struct{}{}
}

// StorageWrite records the post-transaction value of a mutated storage slot.
// The storage slot is removed from the list of read slots.
func (b *ConstructionBlockAccessList) StorageWrite(txIdx uint32, address common.Address, key, value common.Hash) {
	if _, ok := b.Accounts[address]; !ok {
		b.Accounts[address] = NewConstructionAccountAccess()
	}
	if _, ok := b.Accounts[address].StorageWrites[key]; !ok {
		b.Accounts[address].StorageWrites[key] = make(map[uint32]common.Hash)
	}
	b.Accounts[address].StorageWrites[key][txIdx] = value

	delete(b.Accounts[address].StorageReads, key)
}

// CodeChange records the code of a newly-created contract.
func (b *ConstructionBlockAccessList) CodeChange(address common.Address, txIndex uint32, code []byte) {
	if _, ok := b.Accounts[address]; !ok {
		b.Accounts[address] = NewConstructionAccountAccess()
	}
	// TODO(rjl493456442) is it essential to deep-copy the code?
	b.Accounts[address].CodeChange[txIndex] = bytes.Clone(code)
}

// NonceChange records tx post-state nonce of any contract-like accounts whose
// nonce was incremented.
func (b *ConstructionBlockAccessList) NonceChange(address common.Address, txIdx uint32, postNonce uint64) {
	if _, ok := b.Accounts[address]; !ok {
		b.Accounts[address] = NewConstructionAccountAccess()
	}
	b.Accounts[address].NonceChanges[txIdx] = postNonce
}

// BalanceChange records the post-transaction balance of an account whose
// balance changed.
func (b *ConstructionBlockAccessList) BalanceChange(txIdx uint32, address common.Address, balance *uint256.Int) {
	if _, ok := b.Accounts[address]; !ok {
		b.Accounts[address] = NewConstructionAccountAccess()
	}
	b.Accounts[address].BalanceChanges[txIdx] = balance.Clone()
}

// The Restore methods below record that an entry ended a finalisation scope
// with the value it held immediately before the given index. Changes are net
// of the whole index, so such an entry is not a change, and whatever an
// earlier scope sharing the index recorded for it is dropped on Merge. The
// entries of other indices are left alone.

// BalanceRestore records that the balance of the account is back to the one
// it had before the given index.
func (b *ConstructionBlockAccessList) BalanceRestore(txIdx uint32, address common.Address) {
	acc := b.account(address)
	delete(acc.BalanceChanges, txIdx)
	acc.restoresAt(txIdx).balance = true
}

// NonceRestore records that the nonce of the account is back to the one it
// had before the given index.
func (b *ConstructionBlockAccessList) NonceRestore(address common.Address, txIdx uint32) {
	acc := b.account(address)
	delete(acc.NonceChanges, txIdx)
	acc.restoresAt(txIdx).nonce = true
}

// CodeRestore records that the code of the account is back to the one it had
// before the given index.
func (b *ConstructionBlockAccessList) CodeRestore(address common.Address, txIdx uint32) {
	acc := b.account(address)
	delete(acc.CodeChange, txIdx)
	acc.restoresAt(txIdx).code = true
}

// StorageRestore records that the storage slot is back to the value it had
// before the given index. The slot counts as read, unless it was written at
// another index.
func (b *ConstructionBlockAccessList) StorageRestore(txIdx uint32, address common.Address, key common.Hash) {
	acc := b.account(address)
	if _, ok := acc.StorageWrites[key]; ok {
		acc.dropStorageWrite(txIdx, key)
	} else {
		acc.StorageReads[key] = struct{}{}
	}
	r := acc.restoresAt(txIdx)
	r.slots = append(r.slots, key)
}

// account returns the access object of the given account, creating it if absent.
func (b *ConstructionBlockAccessList) account(address common.Address) *ConstructionAccountAccess {
	acc, ok := b.Accounts[address]
	if !ok {
		acc = NewConstructionAccountAccess()
		b.Accounts[address] = acc
	}
	return acc
}

// PrettyPrint returns a human-readable representation of the access list
func (b *ConstructionBlockAccessList) PrettyPrint() string {
	enc := b.ToEncodingObj()
	return enc.PrettyPrint()
}

// Merge applies other on top of the local block access list. For colliding
// entries (a (slot, txIdx) write or a txIdx-keyed balance/nonce/code change),
// the value from other wins, matching the semantics of applying the local
// effects first and then other's. Storage reads are unioned; any slot
// written by either side is dropped from StorageReads.
//
// Typically each list covers its own tx index, so txIdx-level collisions are
// not expected; the exception is pre/post-transition system calls, which
// share a single tx index. In that case callers must pass block-accessList
// in order strictly. Entries which other marks as restored to their value
// from before a tx index are dropped from the local list for that index, so
// that the merged list is net of all the scopes sharing the index.
//
// other is referenced (not deep copied), after the call both lists share
// inner maps and other must not be mutated.
func (b *ConstructionBlockAccessList) Merge(other *ConstructionBlockAccessList) {
	if other == nil {
		return
	}
	for addr, otherAcc := range other.Accounts {
		acc, ok := b.Accounts[addr]
		if !ok {
			b.Accounts[addr] = otherAcc
			continue
		}
		// Drop the entries restored by other first: anything other carries
		// for the same index was recorded after the restore.
		for txIdx, restored := range otherAcc.restores {
			local := acc.restoresAt(txIdx)
			if restored.balance {
				delete(acc.BalanceChanges, txIdx)
				local.balance = true
			}
			if restored.nonce {
				delete(acc.NonceChanges, txIdx)
				local.nonce = true
			}
			if restored.code {
				delete(acc.CodeChange, txIdx)
				local.code = true
			}
			for _, key := range restored.slots {
				acc.dropStorageWrite(txIdx, key)
			}
			local.slots = append(local.slots, restored.slots...)
		}
		for key, writes := range otherAcc.StorageWrites {
			existing, ok := acc.StorageWrites[key]
			if !ok {
				acc.StorageWrites[key] = writes
			} else {
				for txIdx, value := range writes {
					existing[txIdx] = value
				}
			}
			delete(acc.StorageReads, key)
		}
		for key := range otherAcc.StorageReads {
			if _, ok := acc.StorageWrites[key]; ok {
				continue
			}
			acc.StorageReads[key] = struct{}{}
		}
		for txIdx, balance := range otherAcc.BalanceChanges {
			acc.BalanceChanges[txIdx] = balance
		}
		for txIdx, nonce := range otherAcc.NonceChanges {
			acc.NonceChanges[txIdx] = nonce
		}
		for txIdx, code := range otherAcc.CodeChange {
			acc.CodeChange[txIdx] = code
		}
	}
}

// Copy returns a deep copy of the access list.
func (b *ConstructionBlockAccessList) Copy() *ConstructionBlockAccessList {
	res := NewConstructionBlockAccessList()
	for addr, aa := range b.Accounts {
		var aaCopy ConstructionAccountAccess

		slotWrites := make(map[common.Hash]map[uint32]common.Hash, len(aa.StorageWrites))
		for key, m := range aa.StorageWrites {
			slotWrites[key] = maps.Clone(m)
		}
		aaCopy.StorageWrites = slotWrites
		aaCopy.StorageReads = maps.Clone(aa.StorageReads)

		balances := make(map[uint32]*uint256.Int, len(aa.BalanceChanges))
		for index, balance := range aa.BalanceChanges {
			balances[index] = balance.Clone()
		}
		aaCopy.BalanceChanges = balances
		aaCopy.NonceChanges = maps.Clone(aa.NonceChanges)

		codes := make(map[uint32][]byte, len(aa.CodeChange))
		for index, code := range aa.CodeChange {
			codes[index] = bytes.Clone(code)
		}
		aaCopy.CodeChange = codes

		if aa.restores != nil {
			aaCopy.restores = make(map[uint32]*constructionRestores, len(aa.restores))
			for index, restored := range aa.restores {
				cpy := *restored
				cpy.slots = slices.Clone(restored.slots)
				aaCopy.restores[index] = &cpy
			}
		}
		res.Accounts[addr] = &aaCopy
	}
	return res
}
