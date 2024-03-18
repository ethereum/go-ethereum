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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package state

import "github.com/ethereum/go-ethereum/common"

// Origin represents the prev-state for a state transition.
type Origin struct {
	// Accounts represents the account data before the state transition, keyed
	// by the account address. The nil value means the account was not present
	// before.
	Accounts map[common.Address][]byte

	// Storages represents the storage data before the state transition, keyed
	// by the account address and slot key hash. The nil value means the slot
	// was not present.
	Storages map[common.Address]map[common.Hash][]byte

	size common.StorageSize // Approximate size of set
}

// NewOrigin constructs the state set with provided data.
func NewOrigin(accounts map[common.Address][]byte, storages map[common.Address]map[common.Hash][]byte) *Origin {
	return &Origin{
		Accounts: accounts,
		Storages: storages,
	}
}

// Size returns the approximate memory size occupied by the set.
func (s *Origin) Size() common.StorageSize {
	if s.size != 0 {
		return s.size
	}
	for _, account := range s.Accounts {
		s.size += common.StorageSize(common.AddressLength + len(account))
	}
	for _, slots := range s.Storages {
		for _, val := range slots {
			s.size += common.StorageSize(common.HashLength + len(val))
		}
		s.size += common.StorageSize(common.AddressLength)
	}
	return s.size
}
