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

// CodeChange contains the runtime bytecode deployed at an address and the
// transaction index where the deployment took place.
type CodeChange struct {
	TxIdx uint16
	Code  []byte `json:"code,omitempty"`
}

// ConstructionAccountAccess contains post-block account state for mutations as well as
// all storage keys that were read during execution.  It is used when building block
// access list during execution.
type ConstructionAccountAccess struct {
	// StorageWrites is the post-state values of an account's storage slots
	// that were modified in a block, keyed by the slot key and the tx index
	// where the modification occurred.
	StorageWrites map[common.Hash]map[uint16]common.Hash

	// StorageReads is the set of slot keys that were accessed during block
	// execution.
	//
	// Storage slots which are both read and written (with changed values)
	// appear only in StorageWrites.
	StorageReads map[common.Hash]struct{}

	// BalanceChanges contains the post-transaction balances of an account,
	// keyed by transaction indices where it was changed.
	BalanceChanges map[uint16]*uint256.Int

	// NonceChanges contains the post-state nonce values of an account keyed
	// by tx index.
	NonceChanges map[uint16]uint64

	CodeChanges map[uint16]CodeChange
}

// NewConstructionAccountAccess initializes the account access object.
func NewConstructionAccountAccess() *ConstructionAccountAccess {
	return &ConstructionAccountAccess{
		StorageWrites:  make(map[common.Hash]map[uint16]common.Hash),
		StorageReads:   make(map[common.Hash]struct{}),
		BalanceChanges: make(map[uint16]*uint256.Int),
		NonceChanges:   make(map[uint16]uint64),
		CodeChanges:    make(map[uint16]CodeChange),
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
func (c *ConstructionBlockAccessList) AccountRead(addr common.Address) {
	if _, ok := c.Accounts[addr]; !ok {
		c.Accounts[addr] = NewConstructionAccountAccess()
	}
}

// StorageRead records a storage key read during execution.
func (c *ConstructionBlockAccessList) StorageRead(address common.Address, key common.Hash) {
	if _, ok := c.Accounts[address]; !ok {
		c.Accounts[address] = NewConstructionAccountAccess()
	}
	if _, ok := c.Accounts[address].StorageWrites[key]; ok {
		return
	}
	c.Accounts[address].StorageReads[key] = struct{}{}
}

// StorageWrite records the post-transaction value of a mutated storage slot.
// The storage slot is removed from the list of read slots.
func (c *ConstructionBlockAccessList) StorageWrite(txIdx uint16, address common.Address, key, value common.Hash) {
	if _, ok := c.Accounts[address]; !ok {
		c.Accounts[address] = NewConstructionAccountAccess()
	}
	if _, ok := c.Accounts[address].StorageWrites[key]; !ok {
		c.Accounts[address].StorageWrites[key] = make(map[uint16]common.Hash)
	}
	c.Accounts[address].StorageWrites[key][txIdx] = value

	delete(c.Accounts[address].StorageReads, key)
}

// CodeChange records the code of a newly-created contract.
func (c *ConstructionBlockAccessList) CodeChange(address common.Address, txIndex uint16, code []byte) {
	if _, ok := c.Accounts[address]; !ok {
		c.Accounts[address] = NewConstructionAccountAccess()
	}
	c.Accounts[address].CodeChanges[txIndex] = CodeChange{
		TxIdx: txIndex,
		Code:  bytes.Clone(code),
	}
}

// NonceChange records tx post-state nonce of any contract-like accounts whose
// nonce was incremented.
func (c *ConstructionBlockAccessList) NonceChange(address common.Address, txIdx uint16, postNonce uint64) {
	if _, ok := c.Accounts[address]; !ok {
		c.Accounts[address] = NewConstructionAccountAccess()
	}
	c.Accounts[address].NonceChanges[txIdx] = postNonce
}

// BalanceChange records the post-transaction balance of an account whose
// balance changed.
func (c *ConstructionBlockAccessList) BalanceChange(txIdx uint16, address common.Address, balance *uint256.Int) {
	if _, ok := c.Accounts[address]; !ok {
		c.Accounts[address] = NewConstructionAccountAccess()
	}
	c.Accounts[address].BalanceChanges[txIdx] = balance.Clone()
}

// Copy returns a deep copy of the access list.
func (c *ConstructionBlockAccessList) Copy() *ConstructionBlockAccessList {
	res := NewConstructionBlockAccessList()
	for addr, aa := range c.Accounts {
		var aaCopy ConstructionAccountAccess

		slotWrites := make(map[common.Hash]map[uint16]common.Hash, len(aa.StorageWrites))
		for key, m := range aa.StorageWrites {
			slotWrites[key] = maps.Clone(m)
		}
		aaCopy.StorageWrites = slotWrites
		aaCopy.StorageReads = maps.Clone(aa.StorageReads)

		balances := make(map[uint16]*uint256.Int, len(aa.BalanceChanges))
		for index, balance := range aa.BalanceChanges {
			balances[index] = balance.Clone()
		}
		aaCopy.BalanceChanges = balances
		aaCopy.NonceChanges = maps.Clone(aa.NonceChanges)

		codeChangesCopy := make(map[uint16]CodeChange)
		for idx, codeChange := range aa.CodeChanges {
			codeChangesCopy[idx] = CodeChange{
				TxIdx: idx,
				Code:  bytes.Clone(codeChange.Code),
			}
		}
		res.Accounts[addr] = &aaCopy
	}
	return res
}

// ApplyDiff includes the given changes in the block access list at the given index.
func (c *ConstructionBlockAccessList) ApplyDiff(i uint, diff *StateDiff) {
	idx := uint16(i)
	for addr, acctDiff := range diff.Mutations {
		if _, ok := c.Accounts[addr]; !ok {
			c.Accounts[addr] = &ConstructionAccountAccess{}
		}
		if acctDiff.Balance != nil {
			if c.Accounts[addr].BalanceChanges == nil {
				c.Accounts[addr].BalanceChanges = make(map[uint16]*uint256.Int)
			}
			c.Accounts[addr].BalanceChanges[idx] = acctDiff.Balance
		}
		if acctDiff.Nonce != nil {
			if c.Accounts[addr].NonceChanges == nil {
				c.Accounts[addr].NonceChanges = make(map[uint16]uint64)
			}
			c.Accounts[addr].NonceChanges[uint16(idx)] = *acctDiff.Nonce
		}
		if acctDiff.Code != nil {
			if c.Accounts[addr].CodeChanges == nil {
				c.Accounts[addr].CodeChanges = make(map[uint16]CodeChange)
			}
			// TODO: make the CodeChanges value just be []byte
			c.Accounts[addr].CodeChanges[uint16(idx)] = CodeChange{idx, acctDiff.Code}
		}
		if acctDiff.StorageWrites != nil {
			if c.Accounts[addr].StorageWrites == nil {
				// TODO: can we instantiate all these maps in the constructor?
				c.Accounts[addr].StorageWrites = make(map[common.Hash]map[uint16]common.Hash)
			}

			for slot, val := range acctDiff.StorageWrites {
				if c.Accounts[addr].StorageWrites[slot] == nil {
					c.Accounts[addr].StorageWrites[slot] = make(map[uint16]common.Hash)
				}

				c.Accounts[addr].StorageWrites[slot][idx] = val

				delete(c.Accounts[addr].StorageReads, slot)
			}
		}
	}
}

// ApplyAccesses records the given account/storage accesses in the BAL.
func (c *ConstructionBlockAccessList) ApplyAccesses(accesses StateAccesses) {
	for addr, acctAccesses := range accesses {
		if c.Accounts[addr] == nil {
			c.Accounts[addr] = &ConstructionAccountAccess{}
		}
		if len(acctAccesses) > 0 {

			if c.Accounts[addr].StorageReads == nil {
				c.Accounts[addr].StorageReads = make(map[common.Hash]struct{})
			}
			for key, _ := range acctAccesses {
				// if any of the accessed keys were previously written, they
				// appear in the written set only and not also in accesses.
				if len(c.Accounts[addr].StorageWrites) > 0 {
					if _, ok := c.Accounts[addr].StorageWrites[key]; ok {
						continue
					}
				}
				c.Accounts[addr].StorageReads[key] = struct{}{}
			}
		}
	}
}

type StateDiff struct {
	Mutations map[common.Address]*AccountState `json:"Mutations,omitempty"`
}

type StateAccesses map[common.Address]map[common.Hash]struct{}

type AccountState struct {
	Balance       *uint256.Int                `json:"Balance,omitempty"`
	Nonce         *uint64                     `json:"Nonce,omitempty"`
	Code          ContractCode                `json:"Code,omitempty"`
	StorageWrites map[common.Hash]common.Hash `json:"StorageWrites,omitempty"`
}

func (a *AccountState) String() string {
	var res bytes.Buffer
	enc := json.NewEncoder(&res)
	enc.SetIndent("", "    ")
	enc.Encode(a)
	return res.String()
}

// Merge the changes of a future AccountState into the caller, resulting in the
// combined state changes through next.
func (a *AccountState) Merge(next *AccountState) {
	if next.Balance != nil {
		a.Balance = next.Balance
	}
	if next.Nonce != nil {
		a.Nonce = next.Nonce
	}
	if next.Code != nil {
		a.Code = next.Code
	}
	if next.StorageWrites != nil {
		if a.StorageWrites == nil {
			a.StorageWrites = maps.Clone(next.StorageWrites)
		} else {
			for key, val := range next.StorageWrites {
				a.StorageWrites[key] = val
			}
		}
	}
}

func NewEmptyAccountState() *AccountState {
	return &AccountState{
		nil,
		nil,
		nil,
		nil,
	}
}

func (a *AccountState) Eq(other *AccountState) bool {
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
		if a.StorageWrites == nil || other.StorageWrites == nil {
			return false
		}

		if !maps.Equal(a.StorageWrites, other.StorageWrites) {
			return false
		}
	}
	return true
}

func (a *AccountState) Copy() *AccountState {
	res := NewEmptyAccountState()
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

func (s *StateDiff) String() string {
	var res bytes.Buffer
	enc := json.NewEncoder(&res)
	enc.SetIndent("", "    ")
	enc.Encode(s)
	return res.String()
}

// Merge merges the state changes present in next into the caller.  After,
// the state of the caller is the aggregate diff through next.
func (s *StateDiff) Merge(next *StateDiff) {
	for account, diff := range next.Mutations {
		if mut, ok := s.Mutations[account]; ok {
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
			s.Mutations[account] = diff.Copy()
		}
	}
}

func (s *StateDiff) Copy() *StateDiff {
	res := &StateDiff{make(map[common.Address]*AccountState)}
	for addr, accountDiff := range s.Mutations {
		cpy := accountDiff.Copy()
		res.Mutations[addr] = cpy
	}
	return res
}

// Copy returns a deep copy of the access list
func (e BlockAccessList) Copy() (res BlockAccessList) {
	for _, accountAccess := range e {
		res = append(res, accountAccess.Copy())
	}
	return
}
