// Copyright 2022 The go-ethereum Authors
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
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ethereum/go-ethereum/common"
)

type transientStorageKey struct {
	addr common.Address
	key  common.Hash
}

// transientStorage is a representation of EIP-1153 "Transient Storage".
type transientStorage map[transientStorageKey]common.Hash

// newTransientStorage creates a new instance of a transientStorage.
func newTransientStorage() transientStorage {
	return make(transientStorage)
}

// Set sets the transient-storage `value` for `key` at the given `addr`.
func (t transientStorage) Set(addr common.Address, key, value common.Hash) {
	tsKey := transientStorageKey{addr: addr, key: key}
	if value == (common.Hash{}) { // this is a 'delete'
		delete(t, tsKey)
	} else {
		t[tsKey] = value
	}
}

// Get gets the transient storage for `key` at the given `addr`.
func (t transientStorage) Get(addr common.Address, key common.Hash) common.Hash {
	tsKey := transientStorageKey{addr: addr, key: key}
	return t[tsKey]
}

// Copy does a deep copy of the transientStorage
func (t transientStorage) Copy() transientStorage {
	return maps.Clone(t)
}

// PrettyPrint prints the contents of the access list in a human-readable form
func (t transientStorage) PrettyPrint() string {
	out := new(strings.Builder)
	sortedTSKeys := slices.Collect(maps.Keys(t))
	slices.SortFunc(sortedTSKeys, func(a, b transientStorageKey) int {
		r := a.addr.Cmp(b.addr)
		if r != 0 {
			return r
		}
		return a.key.Cmp(b.key)
	})

	for i := 0; i < len(sortedTSKeys); {
		tsKey := sortedTSKeys[i]
		fmt.Fprintf(out, "%#x:", tsKey.addr)
		for ; i < len(sortedTSKeys) && sortedTSKeys[i].addr == tsKey.addr; i++ {
			tsKey2 := sortedTSKeys[i]
			fmt.Fprintf(out, "  %X : %X\n", tsKey2.key, t[tsKey2])
		}
	}
	return out.String()
}
