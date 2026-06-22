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
	"encoding/json"
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
		res.Accounts[addr] = &aaCopy
	}
	return res
}

type StorageMutations map[common.Hash]common.Hash

// AccountMutations contains mutations that were made to an account across
// one or more access list indices.
type AccountMutations struct {
	Balance       *uint256.Int     `json:"Balance,omitempty"`
	Nonce         *uint64          `json:"Nonce,omitempty"`
	Code          []byte           `json:"Code,omitempty"`
	StorageWrites StorageMutations `json:"StorageWrites,omitempty"`
}

// String returns a human-readable JSON representation of the account mutations.
func (a *AccountMutations) String() string {
	var res bytes.Buffer
	enc := json.NewEncoder(&res)
	enc.SetIndent("", "    ")
	enc.Encode(a)
	return res.String()
}

// Copy returns a deep-copy of the instance.
func (a *AccountMutations) Copy() *AccountMutations {
	res := &AccountMutations{
		nil,
		nil,
		nil,
		nil,
	}
	if a.Nonce != nil {
		res.Nonce = new(uint64)
		*res.Nonce = *a.Nonce
	}
	if a.Code != nil {
		res.Code = bytes.Clone(a.Code)
	}
	if a.Balance != nil {
		res.Balance = new(uint256.Int).Set(a.Balance)
	}
	if a.StorageWrites != nil {
		res.StorageWrites = maps.Clone(a.StorageWrites)
	}
	return res
}

// Eq returns whether the calling instance is equal to the provided one.
func (a *AccountMutations) Eq(other *AccountMutations) bool {
	if a.Balance != nil || other.Balance != nil {
		if a.Balance == nil || other.Balance == nil {
			return false
		}

		if !a.Balance.Eq(other.Balance) {
			return false
		}
	}

	if (len(a.Code) != 0 || len(other.Code) != 0) && !bytes.Equal(a.Code, other.Code) {
		return false
	}

	if a.Nonce != nil || other.Nonce != nil {
		if a.Nonce == nil || other.Nonce == nil {
			return false
		}

		if *a.Nonce != *other.Nonce {
			return false
		}
	}

	if a.StorageWrites != nil || other.StorageWrites != nil {
		if !maps.Equal(a.StorageWrites, other.StorageWrites) {
			return false
		}
	}
	return true
}

type BALExecutionMode int

const (
	BALExecutionOptimized BALExecutionMode = iota
	BALExecutionNoBatchIO
	BALExecutionSequential
)

// WrittenCounts groups per-block aggregate write counts derived from the BAL.
type WrittenCounts struct {
	Accounts     int
	StorageSlots int
	Codes        int
	CodeBytes    int
}

// WrittenCounts walks the BAL once and returns the aggregate write counts.
func (e BlockAccessList) WrittenCounts() WrittenCounts {
	var w WrittenCounts
	for i := range e {
		a := &e[i]
		if len(a.StorageChanges) > 0 || len(a.BalanceChanges) > 0 ||
			len(a.NonceChanges) > 0 || len(a.CodeChanges) > 0 {
			w.Accounts++
		}
		w.StorageSlots += len(a.StorageChanges)
		if n := len(a.CodeChanges); n > 0 {
			w.Codes++
			w.CodeBytes += len(a.CodeChanges[n-1].NewCode)
		}
	}
	return w
}

type StateMutations map[common.Address]AccountMutations

type StorageKeySet map[common.Hash]struct{}
type StateAccesses map[common.Address]StorageKeySet

func (s StateAccesses) Eq(other StateAccesses) bool {
	if len(s) != len(other) {
		return false
	}

	for addr, set := range s {
		otherSet, ok := other[addr]
		if !ok {
			return false
		}
		if !maps.Equal(set, otherSet) {
			return false
		}
	}
	return true
}
