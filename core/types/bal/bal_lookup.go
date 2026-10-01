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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// accountLookup references an account's per-index mutations. The slices are the
// ones from the encoded access list, which the spec requires to be sorted
// ascending (and unique) by block-access index, so they can be binary-searched
// directly without copying.
type accountLookup struct {
	balances   []encodingBalanceChange
	nonces     []encodingAccountNonce
	codes      []encodingCodeChange
	codeHashes []codeHash // lazily computed hashes, parallel to codes
	storage    map[common.Hash][]encodingStorageWrite
}

// codeHash memoizes the hash of a code change. The lookup is shared by every
// transaction of the block, so the hash is computed at most once per change
// rather than once per transaction loading the account.
type codeHash struct {
	once sync.Once
	hash common.Hash
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
			balances:   acc.BalanceChanges,
			nonces:     acc.NonceChanges,
			codes:      acc.CodeChanges,
			codeHashes: make([]codeHash, len(acc.CodeChanges)),
			storage:    make(map[common.Hash][]encodingStorageWrite, len(acc.StorageChanges)),
		}
		for j := range acc.StorageChanges {
			sc := &acc.StorageChanges[j]
			al.storage[sc.Slot.Bytes32()] = sc.SlotChanges
		}
		l.accounts[acc.Address] = al
	}
	return l
}

// searchLatest returns the position of the entry with the highest block-access
// index strictly below limit, relying on entries being sorted ascending by that
// index. It returns -1 if there is no such entry.
func searchLatest[E any](entries []E, limit uint32, index func(E) uint32) int {
	// The entry before the first one satisfying (index >= limit)
	return sort.Search(len(entries), func(i int) bool {
		return index(entries[i]) >= limit
	}) - 1
}

// AccountChanges returns the account field values observed at block-access index
// limit (i.e. the latest mutation recorded strictly before limit). Each boolean
// reports whether the corresponding field was mutated before limit.
func (l *Lookup) AccountChanges(addr common.Address, limit uint32) (balance *uint256.Int, nonce uint64, codeHash common.Hash, hasBalance, hasNonce, hasCode bool) {
	acc, ok := l.accounts[addr]
	if !ok {
		return nil, 0, common.Hash{}, false, false, false
	}
	if i := searchLatest(acc.balances, limit, func(e encodingBalanceChange) uint32 { return e.BlockAccessIndex }); i >= 0 {
		balance, hasBalance = acc.balances[i].PostBalance, true
	}
	if i := searchLatest(acc.nonces, limit, func(e encodingAccountNonce) uint32 { return e.BlockAccessIndex }); i >= 0 {
		nonce, hasNonce = acc.nonces[i].PostNonce, true
	}
	if i := searchLatest(acc.codes, limit, func(e encodingCodeChange) uint32 { return e.BlockAccessIndex }); i >= 0 {
		h := &acc.codeHashes[i]
		h.once.Do(func() { h.hash = crypto.Keccak256Hash(acc.codes[i].NewCode) })
		codeHash, hasCode = h.hash, true
	}
	return balance, nonce, codeHash, hasBalance, hasNonce, hasCode
}

// Code returns the contract code observed at block-access index limit, and
// whether the code was set before limit.
func (l *Lookup) Code(addr common.Address, limit uint32) ([]byte, bool) {
	acc, ok := l.accounts[addr]
	if !ok {
		return nil, false
	}
	if i := searchLatest(acc.codes, limit, func(e encodingCodeChange) uint32 { return e.BlockAccessIndex }); i >= 0 {
		return acc.codes[i].NewCode, true
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
	if i := searchLatest(writes, limit, func(e encodingStorageWrite) uint32 { return e.BlockAccessIndex }); i >= 0 {
		return writes[i].PostValue.Bytes32(), true
	}
	return common.Hash{}, false
}
