// Copyright 2021 The go-ethereum Authors
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

package vm

import (
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type accessList struct {
	addresses map[common.Address]int
	slots     []map[common.Hash]struct{}
}

func (al *accessList) AddAddress(address common.Address) {
	// Set address if not previously present
	if _, present := al.addresses[address]; !present {
		al.addresses[address] = -1
	}
}

func (al *accessList) AddSlot(address common.Address, slot common.Hash) {
	idx, addrPresent := al.addresses[address]
	if !addrPresent || idx == -1 {
		// Address not present, or addr present but no slots there
		al.addresses[address] = len(al.slots)
		slotmap := map[common.Hash]struct{}{slot: {}}
		al.slots = append(al.slots, slotmap)
	}
	// There is already an (address,slot) mapping
	slotmap := al.slots[idx]
	if _, ok := slotmap[slot]; !ok {
		slotmap[slot] = struct{}{}
	}
	// No changes require
}

func (al *accessList) DeleteAddressIfNoSlotSet(address common.Address) {
	idx, addrPresent := al.addresses[address]
	if !addrPresent || idx == -1 {
		delete(al.addresses, address)
	}
}

// Copy creates an independent copy of an accessList.
func (a *accessList) Copy() *accessList {
	cp := new(accessList)
	for k, v := range a.addresses {
		cp.addresses[k] = v
	}
	cp.slots = make([]map[common.Hash]struct{}, len(a.slots))
	for i, slotMap := range a.slots {
		newSlotmap := make(map[common.Hash]struct{}, len(slotMap))
		for k := range slotMap {
			newSlotmap[k] = struct{}{}
		}
		cp.slots[i] = newSlotmap
	}
	return cp
}

func (a *accessList) ToAccessList() *types.AccessList {
	acl := make([]types.AccessTuple, 0, len(a.addresses))
	for addr, idx := range a.addresses {
		var tuple types.AccessTuple
		tuple.Address = addr
		// addresses without slots are saved as -1
		if idx > 0 {
			keys := make([]common.Hash, 0, len(a.slots[idx]))
			for key := range a.slots[idx] {
				keys = append(keys, key)
			}
			tuple.StorageKeys = keys
		}
		acl = append(acl, tuple)
	}
	cast := types.AccessList(acl)
	return &cast
}

type AccessListTracer struct {
	list accessList
	err  error
}

func (a *AccessListTracer) CaptureStart(env *EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
}

func (a *AccessListTracer) CaptureState(env *EVM, pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, rData []byte, depth int, err error) {
	stack := scope.Stack
	if (op == SLOAD || op == SSTORE) && stack.len() >= 1 {
		slot := common.Hash(stack.data[stack.len()-1].Bytes32())
		a.list.AddSlot(scope.Contract.Address(), slot)
	}
	if (op == EXTCODECOPY || op == EXTCODEHASH || op == EXTCODESIZE || op == BALANCE || op == SELFDESTRUCT) && stack.len() >= 1 {
		stackElem := stack.data[stack.len()-1].Bytes32()
		address := common.BytesToAddress(stackElem[:])
		a.list.AddAddress(address)
	}
	if (op == DELEGATECALL || op == CALL || op == STATICCALL || op == CALLCODE) && stack.len() >= 5 {
		stackElem := stack.data[stack.len()-2].Bytes32()
		address := common.BytesToAddress(stackElem[:])
		a.list.AddAddress(address)
	}
}

func (*AccessListTracer) CaptureFault(env *EVM, pc uint64, op OpCode, gas, cost uint64, scope *ScopeContext, depth int, err error) {
}

func (*AccessListTracer) CaptureEnd(output []byte, gasUsed uint64, t time.Duration, err error) {}

func (a *AccessListTracer) GetAccessList() *types.AccessList {
	return a.list.ToAccessList()
}

func (a *AccessListTracer) GetUnpreparedAccessList(sender common.Address, dst *common.Address, precompiles []common.Address) *types.AccessList {
	copy := a.list.Copy()
	copy.DeleteAddressIfNoSlotSet(sender)
	if dst != nil {
		copy.DeleteAddressIfNoSlotSet(*dst)
	}
	for _, addr := range precompiles {
		copy.DeleteAddressIfNoSlotSet(addr)
	}
	return copy.ToAccessList()
}
