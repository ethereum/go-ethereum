// Copyright 2026 The go-ethereum Authors
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

package bal

import (
	"maps"

	"github.com/ethereum/go-ethereum/common"
)

// StorageAccessList represents a set of storage slots accessed within an account.
type StorageAccessList map[common.Hash]struct{}

// StateAccessList records the set of accounts and storage slots that have been
// accessed. An entry with an empty StorageAccessList denotes an account access
// without any storage slot access.
type StateAccessList struct {
	list map[common.Address]StorageAccessList
}

// NewStateAccessList returns an empty StateAccessList ready for use.
func NewStateAccessList() *StateAccessList {
	return &StateAccessList{
		list: make(map[common.Address]StorageAccessList),
	}
}

// AddAccount records an access to the given account. It is a no-op if the
// account is already present.
func (s *StateAccessList) AddAccount(addr common.Address) {
	if s == nil {
		return
	}
	if _, exists := s.list[addr]; !exists {
		s.list[addr] = make(StorageAccessList)
	}
}

// AddState records an access to the given storage slot. The owning account is
// implicitly recorded as well.
func (s *StateAccessList) AddState(addr common.Address, slot common.Hash) {
	if s == nil {
		return
	}
	slots, exists := s.list[addr]
	if !exists {
		slots = make(StorageAccessList)
		s.list[addr] = slots
	}
	slots[slot] = struct{}{}
}

// Merge merges the entries from other into the receiver.
func (s *StateAccessList) Merge(other *StateAccessList) {
	if s == nil || other == nil {
		return
	}
	for addr, otherSlots := range other.list {
		slots, exists := s.list[addr]
		if !exists {
			s.list[addr] = otherSlots
			continue
		}
		maps.Copy(slots, otherSlots)
	}
}

func (s *StateAccessList) Eq(other StateAccessList) bool {
	if len(s.list) != len(other.list) {
		return false
	}
	for addr, accesses := range s.list {
		if _, ok := other.list[addr]; !ok {
			return false
		}
		if !maps.Equal(accesses, other.list[addr]) {
			return false
		}
	}
	return true
}
