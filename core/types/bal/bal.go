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

// CodeChange contains the runtime bytecode deployed at an address and the
// transaction index where the deployment took place.
type CodeChange struct {
	TxIndex uint16
	Code    []byte `json:"code,omitempty"`
}

// AccountAccess contains post-block account state for mutations as well as
// all storage keys that were read during execution.
type AccountAccess struct {
	// StorageWrites is the post-state values of an account's storage slots
	// that were modified in a block, keyed by the slot key and the tx index
	// where the modification occurred.
	StorageWrites map[common.Hash]map[uint16]common.Hash `json:"storageWrites,omitempty"`

	// StorageReads is the set of slot keys that were accessed during block
	// execution.
	//
	// Storage slots which are both read and written (with changed values)
	// appear only in StorageWrites.
	StorageReads map[common.Hash]struct{} `json:"storageReads,omitempty"`

	// BalanceChanges contains the post-transaction balances of an account,
	// keyed by transaction indices where it was changed.
	BalanceChanges map[uint16]*uint256.Int `json:"balanceChanges,omitempty"`

	// NonceChanges contains the post-state nonce values of an account keyed
	// by tx index.
	NonceChanges map[uint16]uint64 `json:"nonceChanges,omitempty"`

	// CodeChange is only set for contract accounts which were deployed in
	// the block.
	CodeChange *CodeChange `json:"codeChange,omitempty"`
}

// NewAccountAccess initializes the account access object.
func NewAccountAccess() *AccountAccess {
	return &AccountAccess{
		StorageWrites:  make(map[common.Hash]map[uint16]common.Hash),
		StorageReads:   make(map[common.Hash]struct{}),
		BalanceChanges: make(map[uint16]*uint256.Int),
		NonceChanges:   make(map[uint16]uint64),
	}
}

// BlockAccessList contains post-block modified state and some state accessed
// in execution (account addresses and storage keys).
type BlockAccessList struct {
	Accounts map[common.Address]*AccountAccess
}

// NewBlockAccessList instantiates an empty access list.
func NewBlockAccessList() BlockAccessList {
	return BlockAccessList{
		Accounts: make(map[common.Address]*AccountAccess),
	}
}

// AccountRead records the address of an account that has been read during execution.
func (b *BlockAccessList) AccountRead(addr common.Address) {
	if _, ok := b.Accounts[addr]; !ok {
		b.Accounts[addr] = NewAccountAccess()
	}
}

// StorageRead records a storage key read during execution.
func (b *BlockAccessList) StorageRead(address common.Address, key common.Hash) {
	if _, ok := b.Accounts[address]; !ok {
		b.Accounts[address] = NewAccountAccess()
	}
	if _, ok := b.Accounts[address].StorageWrites[key]; ok {
		return
	}
	b.Accounts[address].StorageReads[key] = struct{}{}
}

// StorageWrite records the post-transaction value of a mutated storage slot.
// The storage slot is removed from the list of read slots.
func (b *BlockAccessList) StorageWrite(txIdx uint16, address common.Address, key, value common.Hash) {
	if _, ok := b.Accounts[address]; !ok {
		b.Accounts[address] = NewAccountAccess()
	}
	if _, ok := b.Accounts[address].StorageWrites[key]; !ok {
		b.Accounts[address].StorageWrites[key] = make(map[uint16]common.Hash)
	}
	b.Accounts[address].StorageWrites[key][txIdx] = value

	delete(b.Accounts[address].StorageReads, key)
}

// CodeChange records the code of a newly-created contract.
func (b *BlockAccessList) CodeChange(address common.Address, txIndex uint16, code []byte) {
	if _, ok := b.Accounts[address]; !ok {
		b.Accounts[address] = NewAccountAccess()
	}
	b.Accounts[address].CodeChange = &CodeChange{
		TxIndex: txIndex,
		Code:    bytes.Clone(code),
	}
}

// NonceChange records tx post-state nonce of any contract-like accounts whose
// nonce was incremented.
func (b *BlockAccessList) NonceChange(address common.Address, txIdx uint16, postNonce uint64) {
	if _, ok := b.Accounts[address]; !ok {
		b.Accounts[address] = NewAccountAccess()
	}
	b.Accounts[address].NonceChanges[txIdx] = postNonce
}

// BalanceChange records the post-transaction balance of an account whose
// balance changed.
func (b *BlockAccessList) BalanceChange(txIdx uint16, address common.Address, balance *uint256.Int) {
	if _, ok := b.Accounts[address]; !ok {
		b.Accounts[address] = NewAccountAccess()
	}
	b.Accounts[address].BalanceChanges[txIdx] = balance.Clone()
}

// PrettyPrint returns a human-readable representation of the access list
func (b *BlockAccessList) PrettyPrint() string {
	enc := b.toEncodingObj()
	return enc.prettyPrint()
}

// Hash computes the SSZ hash of the access list
func (b *BlockAccessList) Hash() common.Hash {
	hash, err := b.toEncodingObj().HashTreeRoot()
	if err != nil {
		// errors here are related to BAL values exceeding maximum size defined
		// by the spec. Hard-fail because these cases are not expected to be hit
		// under reasonable conditions.
		panic(err)
	}
	return hash
}

// Copy returns a deep copy of the access list.
func (b *BlockAccessList) Copy() *BlockAccessList {
	res := NewBlockAccessList()
	for addr, aa := range b.Accounts {
		var aaCopy AccountAccess

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

		if aa.CodeChange != nil {
			aaCopy.CodeChange = &CodeChange{
				TxIndex: aa.CodeChange.TxIndex,
				Code:    bytes.Clone(aa.CodeChange.Code),
			}
		}
		res.Accounts[addr] = &aaCopy
	}
	return &res
}
