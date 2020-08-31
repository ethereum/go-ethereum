// Copyright 2020 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common"
)

type accessList struct {
	addresses map[common.Address]int
	slots     []map[common.Hash]struct{}
}

// Contains returns true if the address is in the access list
func (al *accessList) ContainsAddr(address common.Address) bool {
	_, ok := al.addresses[address]
	return ok
}

// Contains returns whether the (address, slot) is present.
func (al *accessList) Contains(address common.Address, slot common.Hash) (addressPresent bool, slotPresent bool) {
	idx, ok := al.addresses[address]
	if !ok {
		// no such address (and hence zero slots)
		return false, false
	}
	if idx == -1 {
		// address yes, but no slots
		return true, false
	}
	_, slotPresent = al.slots[idx][slot]
	return true, slotPresent
}

// NewAcessList creates a new accessList
func NewAccessList() *accessList {
	return &accessList{
		addresses: make(map[common.Address]int),
		slots:     nil,
	}
}

// Clear cleans out the access list. This can be used instead of creating a new object
// at every transacton
func (a *accessList) Clear() {
	a.addresses = make(map[common.Address]int)
	a.slots = a.slots[:0]
}

// Copy creates an independent copy of a
func (a *accessList) Copy() *accessList {
	cp := NewAccessList()
	for k, v := range a.addresses {
		cp.addresses[k] = v
	}
	for _, slotMap := range a.slots {
		newSlotmap := make(map[common.Hash]struct{})
		for k := range slotMap {
			newSlotmap[k] = struct{}{}
		}
		cp.slots = append(cp.slots, newSlotmap)
	}
	return cp
}

// AddAddr adds an address to the access list, and returns 'true' if the operation
// caused a change (addr was not previously in the list)
func (al *accessList) AddAddr(address common.Address) bool {
	if _, present := al.addresses[address]; present {
		return false
	}
	al.addresses[address] = -1
	return true
}

// AddSlot adds the specified (addr, slot) combo to the access list.
// Return values are:
// - address added
// - slot added
// For any 'true' value returned, a corresponding journal entry must be made
func (al *accessList) AddSlot(address common.Address, slot common.Hash) (addrChange bool, slotChange bool) {
	idx, addrOk := al.addresses[address]
	if !addrOk || idx == -1 {
		// Address not present, or addr present but no slots there
		slotmap := make(map[common.Hash]struct{})
		slotmap[slot] = struct{}{}
		idx = len(al.slots)
		al.addresses[address] = idx
		al.slots = append(al.slots, slotmap)

		if !addrOk {
			addrChange = true
		}
		// Journal add slot change
		return addrChange, true
	}
	// There is already an (address,slot) mapping
	slotmap := al.slots[idx]
	if _, ok := slotmap[slot]; !ok {
		slotmap[slot] = struct{}{}
		// Journal add slot change
		return false, true
	}
	// No changes required
	return false, false
}
