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

type StateMutations struct {
	list map[common.Address]AccountMutations
}

func NewStateMutations() *StateMutations {
	return &StateMutations{make(map[common.Address]AccountMutations)}
}

func (s StateMutations) String() string {
	b, _ := json.MarshalIndent(s, "", "    ")
	return string(b)
}

// Merge merges the state changes present in next into the caller.  After,
// the state of the caller is the aggregate diff through next.
func (s *StateMutations) Merge(next *StateMutations) {
	if next == nil {
		return
	}
	for account, diff := range next.list {
		if mut, ok := s.list[account]; ok {
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
			s.list[account] = mut
		} else {
			s.list[account] = *diff.Copy()
		}
	}
}

func (s *StateMutations) Eq(other *StateMutations) bool {
	if s == nil && other == nil {
		return true
	} else if s == nil && other != nil {
		return false
	} else if s != nil && other == nil {
		return false
	} else if len(s.list) != len(other.list) {
		return false
	}

	for addr, mut := range s.list {
		otherMut, ok := other.list[addr]
		if !ok {
			return false
		}

		if !mut.Eq(&otherMut) {
			return false
		}
	}

	return true
}

func (s *StateMutations) Set(addr common.Address, mut *AccountMutations) {
	s.list[addr] = *mut
}

type ConstructionBlockAccessList struct {
	list             map[common.Address]*ConstructionAccountAccesses
	transactionCount int
}

func NewConstructionBlockAccessList() *ConstructionBlockAccessList {
	return &ConstructionBlockAccessList{
		make(map[common.Address]*ConstructionAccountAccesses),
		0}
}

func (c *ConstructionBlockAccessList) Copy() *ConstructionBlockAccessList {
	if c == nil {
		return nil
	}
	res := NewConstructionBlockAccessList()
	for addr, accountAccess := range c.list {
		aaCopy := accountAccess.Copy()
		res.list[addr] = &aaCopy
	}
	return res
}

// must be called after all txs are added
func (c *ConstructionBlockAccessList) AddBlockFinalizeMutations(muts *StateMutations) {
	c.addMutations(muts, c.transactionCount+1)
}

func (c *ConstructionBlockAccessList) AddBlockInitMutations(muts *StateMutations) {
	c.addMutations(muts, 0)
}

func (c *ConstructionBlockAccessList) AddTransactionMutations(muts *StateMutations, txIdx int) {
	c.transactionCount = max(c.transactionCount, txIdx+1)
	c.addMutations(muts, c.transactionCount)
}

func (c *ConstructionBlockAccessList) addMutations(muts *StateMutations, index int) {
	if muts == nil {
		return
	}
	// TO
	idx := uint16(index)
	for addr, mut := range muts.list {
		if _, exist := c.list[addr]; !exist {
			c.list[addr] = newConstructionAccountAccesses()
		}
		if mut.Nonce != nil {
			if c.list[addr].NonceChanges == nil {
				c.list[addr].NonceChanges = make(map[uint16]uint64)
			}
			c.list[addr].NonceChanges[idx] = *mut.Nonce
		}
		if mut.Balance != nil {
			if c.list[addr].BalanceChanges == nil {
				c.list[addr].BalanceChanges = make(map[uint16]*uint256.Int)
			}
			c.list[addr].BalanceChanges[idx] = mut.Balance.Clone()
		}
		if mut.Code != nil {
			if c.list[addr].CodeChanges == nil {
				c.list[addr].CodeChanges = make(map[uint16][]byte)
			}
			c.list[addr].CodeChanges[idx] = bytes.Clone(mut.Code)
		}
		if len(mut.StorageWrites) > 0 {
			for key, val := range mut.StorageWrites {
				if c.list[addr].StorageWrites[key] == nil {
					c.list[addr].StorageWrites[key] = make(map[uint16]common.Hash)
				}
				c.list[addr].StorageWrites[key][idx] = val

				// delete the key from the tracked reads if it was previously read.
				delete(c.list[addr].StorageReads, key)
			}
		}
	}
}

func (c *ConstructionBlockAccessList) AddAccesses(reads *StateAccessList) {
	if reads == nil {
		return
	}
	for addr, addrReads := range reads.list {
		if _, ok := c.list[addr]; !ok {
			c.list[addr] = newConstructionAccountAccesses()
		}
		for storageKey, _ := range addrReads {
			if c.list[addr].StorageWrites != nil {
				if _, ok := c.list[addr].StorageWrites[storageKey]; ok {
					continue
				}
			}
			if c.list[addr].StorageReads == nil {
				c.list[addr].StorageReads = make(map[common.Hash]struct{})
			}
			c.list[addr].StorageReads[storageKey] = struct{}{}
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

// Copy returns a deep copy of the access list
func (e BlockAccessList) Copy() *BlockAccessList {
	var res BlockAccessList
	for _, accountAccess := range e {
		res = append(res, accountAccess.Copy())
	}
	return &res
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
