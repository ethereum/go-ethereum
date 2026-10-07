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

	// balancePre and storagePre hold the balance and slot values from before
	// the first change recorded at an index, so changes recorded in separate
	// scopes of one index can be netted (see DropUnchanged).
	balancePre map[uint32]*uint256.Int
	storagePre map[common.Hash]map[uint32]common.Hash
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

// StorageWriteFrom is StorageWrite for a slot whose value was prev before
// the write.
func (b *ConstructionBlockAccessList) StorageWriteFrom(txIdx uint32, address common.Address, key, prev, value common.Hash) {
	b.StorageWrite(txIdx, address, key, value)
	b.Accounts[address].setStoragePre(key, txIdx, prev)
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

// BalanceChangeFrom is BalanceChange for an account whose balance was prev
// before the change.
func (b *ConstructionBlockAccessList) BalanceChangeFrom(txIdx uint32, address common.Address, prev, balance *uint256.Int) {
	b.BalanceChange(txIdx, address, balance)
	b.Accounts[address].setBalancePre(txIdx, prev)
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
// in order strictly.
//
// Values from before a change (see DropUnchanged) keep the earliest one.
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
		for key, writes := range otherAcc.StorageWrites {
			existing, ok := acc.StorageWrites[key]
			if !ok {
				acc.StorageWrites[key] = writes
			} else {
				for txIdx, value := range writes {
					existing[txIdx] = value
				}
			}
			for txIdx, pre := range otherAcc.storagePre[key] {
				acc.setStoragePre(key, txIdx, pre)
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
		for txIdx, pre := range otherAcc.balancePre {
			acc.setBalancePre(txIdx, pre)
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

		for index, balance := range aa.balancePre {
			aaCopy.setBalancePre(index, balance)
		}
		for key, m := range aa.storagePre {
			for index, value := range m {
				aaCopy.setStoragePre(key, index, value)
			}
		}
		res.Accounts[addr] = &aaCopy
	}
	return res
}

// DropUnchanged removes the balance changes and storage writes recorded at
// index that restore the value from before the index. A slot left without
// writes stays in the list as a read.
func (b *ConstructionBlockAccessList) DropUnchanged(index uint32) {
	for _, acc := range b.Accounts {
		if pre, ok := acc.balancePre[index]; ok {
			if balance, ok := acc.BalanceChanges[index]; ok && balance.Eq(pre) {
				delete(acc.BalanceChanges, index)
			}
		}
		for key, writes := range acc.StorageWrites {
			pre, ok := acc.storagePre[key][index]
			if value, written := writes[index]; !ok || !written || value != pre {
				continue
			}
			delete(writes, index)
			if len(writes) == 0 {
				delete(acc.StorageWrites, key)
				acc.StorageReads[key] = struct{}{}
			}
		}
	}
}

// setBalancePre records the balance from before the first change at index.
func (a *ConstructionAccountAccess) setBalancePre(index uint32, balance *uint256.Int) {
	if a.balancePre == nil {
		a.balancePre = make(map[uint32]*uint256.Int)
	}
	if _, ok := a.balancePre[index]; !ok {
		a.balancePre[index] = balance.Clone()
	}
}

// setStoragePre records the slot value from before the first write at index.
func (a *ConstructionAccountAccess) setStoragePre(key common.Hash, index uint32, value common.Hash) {
	if a.storagePre == nil {
		a.storagePre = make(map[common.Hash]map[uint32]common.Hash)
	}
	if a.storagePre[key] == nil {
		a.storagePre[key] = make(map[uint32]common.Hash)
	}
	if _, ok := a.storagePre[key][index]; !ok {
		a.storagePre[key][index] = value
	}
}
