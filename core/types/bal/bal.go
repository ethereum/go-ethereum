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
	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
	"maps"
)

// ConstructionAccountAccesses contains post-block account state for mutations as well as
// all storage keys that were read during execution.  It is used when building block
// access list during execution.
type ConstructionAccountAccesses struct {
	// StorageWrites is the post-state values of an account's storage slots
	// that were modified in a block, keyed by the slot key and the tx index
	// where the modification occurred.
	StorageWrites map[common.Hash]map[uint16]common.Hash

	// StorageReads is the set of slot keys that were accessed during block
	// execution.
	//
	// storage slots which are both read and written (with changed values)
	// appear only in StorageWrites.
	StorageReads map[common.Hash]struct{}

	// BalanceChanges contains the post-transaction balances of an account,
	// keyed by transaction indices where it was changed.
	BalanceChanges map[uint16]*uint256.Int

	// NonceChanges contains the post-state nonce values of an account keyed
	// by tx index.
	NonceChanges map[uint16]uint64

	CodeChanges map[uint16][]byte
}

func (c *ConstructionAccountAccesses) Copy() (res ConstructionAccountAccesses) {
	if c.StorageWrites != nil {
		res.StorageWrites = make(map[common.Hash]map[uint16]common.Hash)
		for slot, writes := range c.StorageWrites {
			res.StorageWrites[slot] = maps.Clone(writes)
		}
	}
	if c.StorageReads != nil {
		res.StorageReads = maps.Clone(c.StorageReads)
	}
	if c.BalanceChanges != nil {
		res.BalanceChanges = maps.Clone(c.BalanceChanges)
	}
	if c.NonceChanges != nil {
		res.NonceChanges = maps.Clone(c.NonceChanges)
	}
	if c.CodeChanges != nil {
		res.CodeChanges = maps.Clone(c.CodeChanges)
	}
	return res
}

type StateMutations map[common.Address]AccountMutations

func (s StateMutations) String() string {
	b, _ := json.MarshalIndent(s, "", "    ")
	return string(b)
}

// Merge merges the state changes present in next into the caller.  After,
// the state of the caller is the aggregate diff through next.
func (s StateMutations) Merge(next StateMutations) {
	for account, diff := range next {
		if mut, ok := s[account]; ok {
			if diff.Balance != nil {
				mut.Balance = diff.Balance
			}
			if diff.Code != nil {
				mut.Code = diff.Code
			}
			if diff.Nonce != nil {
				mut.Nonce = diff.Nonce
			}
			if len(diff.StorageWrites) > 0 {
				if mut.StorageWrites == nil {
					mut.StorageWrites = maps.Clone(diff.StorageWrites)
				} else {
					for key, val := range diff.StorageWrites {
						mut.StorageWrites[key] = val
					}
				}
			}
		} else {
			s[account] = *diff.Copy()
		}
	}
}

func (s StateMutations) Eq(other StateMutations) bool {
	if len(s) != len(other) {
		return false
	}

	for addr, mut := range s {
		otherMut, ok := other[addr]
		if !ok {
			return false
		}

		if !mut.Eq(&otherMut) {
			return false
		}
	}

	return true
}

type ConstructionBlockAccessList map[common.Address]*ConstructionAccountAccesses

func (c ConstructionBlockAccessList) Copy() ConstructionBlockAccessList {
	res := make(ConstructionBlockAccessList)
	for addr, accountAccess := range c {
		aaCopy := accountAccess.Copy()
		res[addr] = &aaCopy
	}
	return res
}

func (c ConstructionBlockAccessList) AccumulateMutations(muts StateMutations, idx uint16) {
	for addr, mut := range muts {
		if _, exist := c[addr]; !exist {
			c[addr] = newConstructionAccountAccesses()
		}
		if mut.Nonce != nil {
			if c[addr].NonceChanges == nil {
				c[addr].NonceChanges = make(map[uint16]uint64)
			}
			c[addr].NonceChanges[idx] = *mut.Nonce
		}
		if mut.Balance != nil {
			if c[addr].BalanceChanges == nil {
				c[addr].BalanceChanges = make(map[uint16]*uint256.Int)
			}
			c[addr].BalanceChanges[idx] = mut.Balance.Clone()
		}
		if mut.Code != nil {
			if c[addr].CodeChanges == nil {
				c[addr].CodeChanges = make(map[uint16][]byte)
			}
			c[addr].CodeChanges[idx] = bytes.Clone(mut.Code)
		}
		if len(mut.StorageWrites) > 0 {
			for key, val := range mut.StorageWrites {
				if c[addr].StorageWrites[key] == nil {
					c[addr].StorageWrites[key] = make(map[uint16]common.Hash)
				}
				c[addr].StorageWrites[key][idx] = val
			}
		}
	}
}

func (c ConstructionBlockAccessList) AccumulateReads(reads StateAccesses) {
	for addr, addrReads := range reads {
		if _, ok := c[addr]; !ok {
			c[addr] = newConstructionAccountAccesses()
		}
		for storageKey, _ := range addrReads {
			if c[addr].StorageWrites != nil {
				if _, ok := c[addr].StorageWrites[storageKey]; ok {
					continue
				}
			}
			if c[addr].StorageReads == nil {
				c[addr].StorageReads = make(map[common.Hash]struct{})
			}
			c[addr].StorageReads[storageKey] = struct{}{}
		}
	}
}

func newConstructionAccountAccesses() *ConstructionAccountAccesses {
	return &ConstructionAccountAccesses{
		StorageWrites:  make(map[common.Hash]map[uint16]common.Hash),
		StorageReads:   make(map[common.Hash]struct{}),
		BalanceChanges: make(map[uint16]*uint256.Int),
		NonceChanges:   make(map[uint16]uint64),
		CodeChanges:    make(map[uint16][]byte),
	}
}

// StateDiff contains state mutations occuring over one or more access list
// index.
type StateDiff struct {
	Mutations map[common.Address]*AccountMutations `json:"Mutations,omitempty"`
}

// StateAccesses contains a set of accounts/storage that were accessed during the
// execution of one or more access list indices.
type StateAccesses map[common.Address]StorageAccessList
type StorageAccessList map[common.Hash]struct{}

// Merge combines adds the accesses from other into s.
func (s StateAccesses) Merge(other StateAccesses) {
	for addr, accesses := range other {
		if _, ok := s[addr]; !ok {
			s[addr] = make(map[common.Hash]struct{})
		}
		for slot := range accesses {
			s[addr][slot] = struct{}{}
		}
	}
}

func (s StateAccesses) Eq(other StateAccesses) bool {
	if len(s) != len(other) {
		return false
	}
	for addr, accesses := range s {
		if _, ok := other[addr]; !ok {
			return false
		}
		if !maps.Equal(accesses, other[addr]) {
			return false
		}
	}
	return true
}

type StorageMutations map[common.Hash]common.Hash

// AccountMutations contains mutations that were made to an account across
// one or more access list indices.
type AccountMutations struct {
	Balance       *uint256.Int     `json:"Balance,omitempty"`
	Nonce         *uint64          `json:"Nonce,omitempty"`
	Code          ContractCode     `json:"Code,omitempty"`
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

// String returns the state diff as a formatted JSON string.
func (s *StateDiff) String() string {
	var res bytes.Buffer
	enc := json.NewEncoder(&res)
	enc.SetIndent("", "    ")
	enc.Encode(s)
	return res.String()
}

// Copy returns a deep copy of the StateDiff
func (s *StateDiff) Copy() *StateDiff {
	res := &StateDiff{make(map[common.Address]*AccountMutations)}
	for addr, accountDiff := range s.Mutations {
		cpy := accountDiff.Copy()
		res.Mutations[addr] = cpy
	}
	return res
}

// AccessListReader exposes utilities to read state mutations and accesses from an access list
// TODO: expose this an an interface?
type AccessListReader map[common.Address]*AccountAccess

func NewAccessListReader(bal BlockAccessList) (reader AccessListReader) {
	reader = make(AccessListReader)
	for _, accountAccess := range bal {
		reader[accountAccess.Address] = &accountAccess
	}
	return
}

func (a AccessListReader) Accesses() (accesses StateAccesses) {
	accesses = make(StateAccesses)
	for addr, acctAccess := range a {
		if len(acctAccess.StorageReads) > 0 {
			accesses[addr] = make(StorageAccessList)
			for _, key := range acctAccess.StorageReads {
				accesses[addr][key.ToHash()] = struct{}{}
			}
		} else if len(acctAccess.CodeChanges) == 0 && len(acctAccess.StorageChanges) == 0 && len(acctAccess.BalanceChanges) == 0 && len(acctAccess.NonceChanges) == 0 {
			accesses[addr] = make(StorageAccessList)
		}
	}
	return
}

// TODO: these methods should return the mutations accrued before the execution of the given index

// TODO: strip the storage mutations from the returned result
// the returned object should be able to be modified
func (a AccessListReader) accountMutationsAt(addr common.Address, idx int) (res *AccountMutations) {
	acct, exist := a[addr]
	if !exist {
		return nil
	}

	res = &AccountMutations{}
	// TODO: remove the reverse iteration here to clean the code up

	for i := len(acct.BalanceChanges) - 1; i >= 0; i-- {
		if acct.BalanceChanges[i].TxIdx == uint16(idx) {
			res.Balance = acct.BalanceChanges[i].Balance
		}
		if acct.BalanceChanges[i].TxIdx < uint16(idx) {
			break
		}
	}

	for i := len(acct.CodeChanges) - 1; i >= 0; i-- {
		if acct.CodeChanges[i].TxIndex == uint16(idx) {
			res.Code = bytes.Clone(acct.CodeChanges[i].Code)
			break
		}
		if acct.CodeChanges[i].TxIndex < uint16(idx) {
			break
		}
	}

	for i := len(acct.NonceChanges) - 1; i >= 0; i-- {
		if acct.NonceChanges[i].TxIdx == uint16(idx) {
			res.Nonce = new(uint64)
			*res.Nonce = acct.NonceChanges[i].Nonce
			break
		}
		if acct.NonceChanges[i].TxIdx < uint16(idx) {
			break
		}
	}

	for i := len(acct.StorageChanges) - 1; i >= 0; i-- {
		if res.StorageWrites == nil {
			res.StorageWrites = make(map[common.Hash]common.Hash)
		}
		slotWrites := acct.StorageChanges[i]

		for j := len(slotWrites.Accesses) - 1; j >= 0; j-- {
			if slotWrites.Accesses[j].TxIdx == uint16(idx) {
				res.StorageWrites[slotWrites.Slot.ToHash()] = slotWrites.Accesses[j].ValueAfter.ToHash()
				break
			}
			if slotWrites.Accesses[j].TxIdx < uint16(idx) {
				break
			}
		}
		if len(res.StorageWrites) == 0 {
			res.StorageWrites = nil
		}
	}

	if res.Code == nil && res.Nonce == nil && len(res.StorageWrites) == 0 && res.Balance == nil {
		return nil
	}
	return res
}

func (a AccessListReader) AccountMutations(addr common.Address, idx int) (res *AccountMutations) {
	diff, exist := a[addr]
	if !exist {
		return nil
	}

	res = &AccountMutations{}

	for i := 0; i < len(diff.BalanceChanges) && diff.BalanceChanges[i].TxIdx < uint16(idx); i++ {
		res.Balance = diff.BalanceChanges[i].Balance.Clone()
	}

	for i := 0; i < len(diff.CodeChanges) && diff.CodeChanges[i].TxIndex < uint16(idx); i++ {
		res.Code = bytes.Clone(diff.CodeChanges[i].Code)
	}

	for i := 0; i < len(diff.NonceChanges) && diff.NonceChanges[i].TxIdx < uint16(idx); i++ {
		res.Nonce = new(uint64)
		*res.Nonce = diff.NonceChanges[i].Nonce
	}

	if len(diff.StorageChanges) > 0 {
		res.StorageWrites = make(map[common.Hash]common.Hash)
		for _, slotWrites := range diff.StorageChanges {
			for i := 0; i < len(slotWrites.Accesses) && slotWrites.Accesses[i].TxIdx < uint16(idx); i++ {
				res.StorageWrites[slotWrites.Slot.ToHash()] = slotWrites.Accesses[i].ValueAfter.ToHash()
			}
		}
	}

	if res.Code == nil && res.Nonce == nil && len(res.StorageWrites) == 0 && res.Balance == nil {
		return nil
	}
	return res
}

// Mutations returns the aggregate state mutations from [0, idx)
func (a AccessListReader) Mutations(idx int) *StateMutations {
	res := make(StateMutations)
	for addr := range a {
		if mut := a.AccountMutations(addr, idx); mut != nil {
			res[addr] = *mut
		}
	}
	return &res
}

// MutationsAt returns the state mutations from an index
func (a AccessListReader) MutationsAt(idx int) *StateMutations {
	res := make(StateMutations)
	for addr := range a {
		if mut := a.accountMutationsAt(addr, idx); mut != nil {
			res[addr] = *mut
		}
	}
	return &res
}

type StorageKeys map[common.Address][]common.Hash

// StorageKeys returns the set of accounts and storage keys mutated in the access list.
// If reads is set, the un-mutated accounts/keys are included in the result.
func (a AccessListReader) StorageKeys(reads bool) (keys StorageKeys) {
	keys = make(StorageKeys)
	for addr, acct := range a {
		for _, storageChange := range acct.StorageChanges {
			keys[addr] = append(keys[addr], storageChange.Slot.ToHash())
		}
		if !(reads && len(acct.StorageReads) > 0) {
			continue
		}
		for _, storageRead := range acct.StorageReads {
			keys[addr] = append(keys[addr], storageRead.ToHash())
		}
	}
	return
}

// Storage returns the value of a storage key at the start of executing an index.
// If the slot has no mutations in the access list, it returns nil.
func (a AccessListReader) Storage(addr common.Address, key common.Hash, idx int) (val *common.Hash) {
	storageMuts := a.AccountMutations(addr, idx)
	if storageMuts != nil {
		res, ok := storageMuts.StorageWrites[key]
		if ok {
			return &res
		}
	}
	return nil
}

// Copy returns a deep copy of the access list
func (e BlockAccessList) Copy() (res BlockAccessList) {
	for _, accountAccess := range e {
		res = append(res, accountAccess.Copy())
	}
	return
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
