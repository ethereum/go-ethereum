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

package core

import (
	"bytes"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types/bal"
)

// accessListMerger consumes the BAL returned by a single Finalise with the
// shared block accessList index.
type accessListMerger interface {
	Merge(*bal.ConstructionBlockAccessList)
}

// postExecutionAccessList merges the per-call lists at one index.
type postExecutionAccessList struct {
	index    uint32
	merged   *bal.ConstructionBlockAccessList
	repeated map[common.Address]*repeatedWrites
}

// newPostExecutionAccessList constructs the object with the given index.
func newPostExecutionAccessList(index uint32) *postExecutionAccessList {
	return &postExecutionAccessList{
		index:    index,
		merged:   bal.NewConstructionBlockAccessList(),
		repeated: make(map[common.Address]*repeatedWrites),
	}
}

type repeatedWrites struct {
	balance, nonce, code bool
	slots                map[common.Hash]struct{}
}

// bothChanged reports whether both scopes contain a change at index.
func bothChanged[T any](a, b map[uint32]T, index uint32) bool {
	_, first := a[index]
	_, second := b[index]
	return first && second
}

func (p *postExecutionAccessList) Merge(scope *bal.ConstructionBlockAccessList) {
	if scope == nil {
		return
	}
	for addr, next := range scope.Accounts {
		prev := p.merged.Accounts[addr]
		if prev == nil {
			continue
		}
		writes := p.repeated[addr]
		if writes == nil {
			writes = new(repeatedWrites)
			p.repeated[addr] = writes
		}
		writes.balance = writes.balance || bothChanged(prev.BalanceChanges, next.BalanceChanges, p.index)
		writes.nonce = writes.nonce || bothChanged(prev.NonceChanges, next.NonceChanges, p.index)
		writes.code = writes.code || bothChanged(prev.CodeChange, next.CodeChange, p.index)

		for key := range next.StorageWrites {
			if _, exists := prev.StorageWrites[key][p.index]; exists {
				if writes.slots == nil {
					writes.slots = make(map[common.Hash]struct{})
				}
				writes.slots[key] = struct{}{}
			}
		}
	}
	p.merged.Merge(scope)
}

// netChanges removes repeated writes that restored the pre-withdrawal value.
func (p *postExecutionAccessList) netChanges(original *state.StateDB) *bal.ConstructionBlockAccessList {
	for addr, changes := range p.repeated {
		account := p.merged.Accounts[addr]
		if changes.balance && account.BalanceChanges[p.index].Eq(original.GetBalance(addr)) {
			delete(account.BalanceChanges, p.index)
		}
		if changes.nonce && account.NonceChanges[p.index] == original.GetNonce(addr) {
			delete(account.NonceChanges, p.index)
		}
		if changes.code && bytes.Equal(account.CodeChange[p.index], original.GetCode(addr)) {
			delete(account.CodeChange, p.index)
		}
		for key := range changes.slots {
			if account.StorageWrites[key][p.index] == original.GetState(addr, key) {
				delete(account.StorageWrites, key)
				account.StorageReads[key] = struct{}{}
			}
		}
	}
	return p.merged
}
