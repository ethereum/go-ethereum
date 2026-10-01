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

package bal

import (
	"sort"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// accountLookup references an account's per-index mutations. The slices are the
// ones from the encoded access list, which the spec requires to be sorted
// ascending (and unique) by block-access index, so they can be binary-searched
// directly without copying.
type accountLookup struct {
	balances []encodingBalanceChange
	nonces   []encodingAccountNonce
	codes    []encodingCodeChange
	storage  map[common.Hash][]encodingStorageWrite
}

// Lookup is a read-optimized, index-addressable view over a block access list.
type Lookup struct {
	accounts map[common.Address]*accountLookup
}

// Lookup builds a Lookup over the access list. The returned view aliases the
// receiver's slices, so the access list must not be mutated while it is in use.
func (e *BlockAccessList) Lookup() *Lookup {
	l := &Lookup{
		accounts: make(map[common.Address]*accountLookup, len(*e)),
	}
	for i := range *e {
		acc := &(*e)[i]
		al := &accountLookup{
			balances: acc.BalanceChanges,
			nonces:   acc.NonceChanges,
			codes:    acc.CodeChanges,
			storage:  make(map[common.Hash][]encodingStorageWrite, len(acc.StorageChanges)),
		}
		for j := range acc.StorageChanges {
			sc := &acc.StorageChanges[j]
			al.storage[sc.Slot.Bytes32()] = sc.SlotChanges
		}
		l.accounts[acc.Address] = al
	}
	return l
}

// searchLatest returns the entry with the highest block-access index strictly
// below limit, relying on entries being sorted ascending by that index.
func searchLatest[E any](entries []E, limit uint32, index func(E) uint32) (E, bool) {
	i := sort.Search(len(entries), func(i int) bool {
		return index(entries[i]) >= limit
	})
	// All entries satisfy the condition (index >= limit)
	if i == 0 {
		var zero E
		return zero, false
	}
	return entries[i-1], true
}

// AccountChanges returns the account field values observed at block-access index
// limit (i.e. the latest mutation recorded strictly before limit). Each boolean
// reports whether the corresponding field was mutated before limit.
func (l *Lookup) AccountChanges(addr common.Address, limit uint32) (balance *uint256.Int, nonce uint64, code []byte, hasBalance, hasNonce, hasCode bool) {
	acc, ok := l.accounts[addr]
	if !ok {
		return nil, 0, nil, false, false, false
	}
	if e, ok := searchLatest(acc.balances, limit, func(e encodingBalanceChange) uint32 { return e.BlockAccessIndex }); ok {
		balance, hasBalance = e.PostBalance, true
	}
	if e, ok := searchLatest(acc.nonces, limit, func(e encodingAccountNonce) uint32 { return e.BlockAccessIndex }); ok {
		nonce, hasNonce = e.PostNonce, true
	}
	if e, ok := searchLatest(acc.codes, limit, func(e encodingCodeChange) uint32 { return e.BlockAccessIndex }); ok {
		code, hasCode = e.NewCode, true
	}
	return balance, nonce, code, hasBalance, hasNonce, hasCode
}

// Code returns the contract code observed at block-access index limit, and
// whether the code was set before limit.
func (l *Lookup) Code(addr common.Address, limit uint32) ([]byte, bool) {
	acc, ok := l.accounts[addr]
	if !ok {
		return nil, false
	}
	if e, ok := searchLatest(acc.codes, limit, func(e encodingCodeChange) uint32 { return e.BlockAccessIndex }); ok {
		return e.NewCode, true
	}
	return nil, false
}

// Storage returns the value of the storage slot observed at block-access index
// limit, and whether the slot was written before limit.
func (l *Lookup) Storage(addr common.Address, slot common.Hash, limit uint32) (common.Hash, bool) {
	acc, ok := l.accounts[addr]
	if !ok {
		return common.Hash{}, false
	}
	writes, ok := acc.storage[slot]
	if !ok {
		return common.Hash{}, false
	}
	if e, ok := searchLatest(writes, limit, func(e encodingStorageWrite) uint32 { return e.BlockAccessIndex }); ok {
		return e.PostValue.Bytes32(), true
	}
	return common.Hash{}, false
}
