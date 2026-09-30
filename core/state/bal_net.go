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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package state

import (
	"bytes"
	"maps"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/holiman/uint256"
)

// sameIndexBaseline is the pre-value of each account field and storage slot
// at the start of one block access index. Post-execution withdrawals and
// system calls share that index and each finalise on their own, so the value
// is captured on the first mutation and kept for the rest of the index.
type sameIndexBaseline struct {
	index    uint32
	balances map[common.Address]*uint256.Int
	nonces   map[common.Address]uint64
	codes    map[common.Address][]byte
	storage  map[common.Address]map[common.Hash]common.Hash
}

func newSameIndexBaseline(index uint32) *sameIndexBaseline {
	return &sameIndexBaseline{
		index:    index,
		balances: make(map[common.Address]*uint256.Int),
		nonces:   make(map[common.Address]uint64),
		codes:    make(map[common.Address][]byte),
		storage:  make(map[common.Address]map[common.Hash]common.Hash),
	}
}

func (b *sameIndexBaseline) copy() *sameIndexBaseline {
	if b == nil {
		return nil
	}
	out := &sameIndexBaseline{
		index:    b.index,
		balances: make(map[common.Address]*uint256.Int, len(b.balances)),
		nonces:   maps.Clone(b.nonces),
		codes:    make(map[common.Address][]byte, len(b.codes)),
		storage:  make(map[common.Address]map[common.Hash]common.Hash, len(b.storage)),
	}
	for addr, balance := range b.balances {
		out.balances[addr] = balance.Clone()
	}
	for addr, code := range b.codes {
		out.codes[addr] = bytes.Clone(code)
	}
	for addr, slots := range b.storage {
		out.storage[addr] = maps.Clone(slots)
	}
	return out
}

// concreteState returns the StateDB behind db. Traced execution wraps it in
// hookedStateDB; both have to net, or the block access list hash diverges.
func concreteState(db any) *StateDB {
	switch s := db.(type) {
	case *StateDB:
		return s
	case *hookedStateDB:
		return s.inner
	default:
		return nil
	}
}

// BeginSameIndexBaseline starts capturing pre-values for index. Mutations at
// any other block access index are ignored. db may be a *StateDB or the
// tracing wrapper around one.
func BeginSameIndexBaseline(db any, index uint32) {
	if s := concreteState(db); s != nil {
		s.sameIndexBaseline = newSameIndexBaseline(index)
	}
}

// DiscardSameIndexBaseline drops a capture started by BeginSameIndexBaseline
// without reconciling. Used when post-execution fails before the access list
// is committed.
func DiscardSameIndexBaseline(db any) {
	if s := concreteState(db); s != nil {
		s.sameIndexBaseline = nil
	}
}

// NetSameIndexChanges drops or demotes changes at index whose post-value
// matches the pre-index baseline, then ends the capture.
func NetSameIndexChanges(db any, list *bal.ConstructionBlockAccessList, index uint32) {
	s := concreteState(db)
	if s == nil {
		return
	}
	baseline := s.sameIndexBaseline
	s.sameIndexBaseline = nil
	if baseline == nil || baseline.index != index || list == nil {
		return
	}
	baseline.net(list, index)
}

func (b *sameIndexBaseline) net(list *bal.ConstructionBlockAccessList, index uint32) {
	for addr, acc := range list.Accounts {
		var demote []common.Hash
		for slot, writes := range acc.StorageWrites {
			postVal, ok := writes[index]
			if !ok {
				continue
			}
			slots := b.storage[addr]
			preVal, recorded := slots[slot]
			if recorded && postVal == preVal {
				demote = append(demote, slot)
			}
		}
		for _, slot := range demote {
			list.DemoteStorageWriteToRead(index, addr, slot)
		}
		if postBal, ok := acc.BalanceChanges[index]; ok {
			if preBal, recorded := b.balances[addr]; recorded && postBal.Cmp(preBal) == 0 {
				list.DropBalanceChange(index, addr)
			}
		}
		if postNonce, ok := acc.NonceChanges[index]; ok {
			if preNonce, recorded := b.nonces[addr]; recorded && postNonce == preNonce {
				list.DropNonceChange(index, addr)
			}
		}
		if postCode, ok := acc.CodeChange[index]; ok {
			if preCode, recorded := b.codes[addr]; recorded && bytes.Equal(postCode, preCode) {
				list.DropCodeChange(index, addr)
			}
		}
	}
}

func (s *StateDB) noteStorageBaseline(addr common.Address, key, prev common.Hash) {
	b := s.sameIndexBaseline
	if b == nil || s.blockAccessIndex != b.index {
		return
	}
	slots, ok := b.storage[addr]
	if !ok {
		slots = make(map[common.Hash]common.Hash)
		b.storage[addr] = slots
	}
	if _, exists := slots[key]; exists {
		return
	}
	slots[key] = prev
}

func (s *StateDB) noteBalanceBaseline(addr common.Address, prev *uint256.Int) {
	b := s.sameIndexBaseline
	if b == nil || s.blockAccessIndex != b.index {
		return
	}
	if _, exists := b.balances[addr]; exists {
		return
	}
	b.balances[addr] = prev.Clone()
}

func (s *StateDB) noteNonceBaseline(addr common.Address, prev uint64) {
	b := s.sameIndexBaseline
	if b == nil || s.blockAccessIndex != b.index {
		return
	}
	if _, exists := b.nonces[addr]; exists {
		return
	}
	b.nonces[addr] = prev
}

func (s *StateDB) noteCodeBaseline(addr common.Address, obj *stateObject, newCode []byte) {
	b := s.sameIndexBaseline
	if b == nil || s.blockAccessIndex != b.index {
		return
	}
	if _, exists := b.codes[addr]; exists {
		return
	}
	// Load only after the capture is known to be active, so ordinary
	// code updates do not pull code into memory just to throw it away.
	current := obj.Code()
	if bytes.Equal(current, newCode) {
		return
	}
	b.codes[addr] = bytes.Clone(current)
}
