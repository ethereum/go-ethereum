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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/holiman/uint256"
)

// accountLookup references an account's per-index mutations. The slices are the
// ones from the encoded access list, which the spec requires to be sorted
// ascending (and unique) by block-access index, so they can be binary-searched
// directly without copying.
type accountLookup struct {
	balances []encodingBalanceChange
	nonces   []encodingAccountNonce
	codes    []lookupCode
	storage  map[common.Hash][]encodingStorageWrite
}

// lookupCode is a code change with its code hash resolved. The code of an
// EIP-8298 adopted code change is taken from another code change in the access
// list carrying the same hash; it is nil if the code was deployed before the
// block and must be read from the state by hash.
type lookupCode struct {
	index uint32
	hash  common.Hash
	code  []byte
}

// Lookup is a read-optimized, index-addressable view over a block access list.
type Lookup struct {
	accounts map[common.Address]*accountLookup
}

// Lookup builds a Lookup over the access list. The returned view aliases the
// receiver's slices, so the access list must not be mutated while it is in use.
func (e *BlockAccessList) Lookup() *Lookup {
	var (
		l = &Lookup{
			accounts: make(map[common.Address]*accountLookup, len(*e)),
		}
		deployed = make(map[common.Hash][]byte)
		adopted  []*lookupCode
	)
	for i := range *e {
		acc := &(*e)[i]
		al := &accountLookup{
			balances: acc.BalanceChanges,
			nonces:   acc.NonceChanges,
			codes:    make([]lookupCode, len(acc.CodeChanges)),
			storage:  make(map[common.Hash][]encodingStorageWrite, len(acc.StorageChanges)),
		}
		for j := range acc.CodeChanges {
			cc := &acc.CodeChanges[j]
			entry := &al.codes[j]
			entry.index = cc.BlockAccessIndex
			if cc.NewCodeHash != nil {
				entry.hash = *cc.NewCodeHash
				adopted = append(adopted, entry)
			} else {
				entry.hash = crypto.Keccak256Hash(cc.NewCode)
				entry.code = cc.NewCode
				if len(cc.NewCode) > 0 {
					deployed[entry.hash] = cc.NewCode
				}
			}
		}
		for j := range acc.StorageChanges {
			sc := &acc.StorageChanges[j]
			al.storage[sc.Slot.Bytes32()] = sc.SlotChanges
		}
		l.accounts[acc.Address] = al
	}
	for _, entry := range adopted {
		entry.code = deployed[entry.hash]
	}
	return l
}

// DeployedCodes returns the bytecode recorded by the non-adopted code changes
// of the access list, keyed by code hash. It resolves the code of EIP-8298
// adopted code changes whose code was deployed earlier in the block; adopted
// code absent from it was deployed before the block. It is nil if the access
// list contains no adopted code change.
func (e *BlockAccessList) DeployedCodes() map[common.Hash][]byte {
	var adopted bool
	for i := range *e {
		for _, cc := range (*e)[i].CodeChanges {
			if cc.NewCodeHash != nil {
				adopted = true
				break
			}
		}
	}
	if !adopted {
		return nil
	}
	codes := make(map[common.Hash][]byte)
	for i := range *e {
		for _, cc := range (*e)[i].CodeChanges {
			if cc.NewCodeHash == nil && len(cc.NewCode) > 0 {
				codes[crypto.Keccak256Hash(cc.NewCode)] = cc.NewCode
			}
		}
	}
	return codes
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
func (l *Lookup) AccountChanges(addr common.Address, limit uint32) (balance *uint256.Int, nonce uint64, codeHash common.Hash, hasBalance, hasNonce, hasCode bool) {
	acc, ok := l.accounts[addr]
	if !ok {
		return nil, 0, common.Hash{}, false, false, false
	}
	if e, ok := searchLatest(acc.balances, limit, func(e encodingBalanceChange) uint32 { return e.BlockAccessIndex }); ok {
		balance, hasBalance = e.PostBalance, true
	}
	if e, ok := searchLatest(acc.nonces, limit, func(e encodingAccountNonce) uint32 { return e.BlockAccessIndex }); ok {
		nonce, hasNonce = e.PostNonce, true
	}
	if e, ok := searchLatest(acc.codes, limit, func(e lookupCode) uint32 { return e.index }); ok {
		codeHash, hasCode = e.hash, true
	}
	return balance, nonce, codeHash, hasBalance, hasNonce, hasCode
}

// Code returns the code hash and contract code observed at block-access index
// limit, and whether the code was set before limit. The returned code is nil
// if it was adopted through EIP-8298 SETCODEFROM from code deployed before the
// block; it must then be read from the state by the returned hash.
func (l *Lookup) Code(addr common.Address, limit uint32) (common.Hash, []byte, bool) {
	acc, ok := l.accounts[addr]
	if !ok {
		return common.Hash{}, nil, false
	}
	if e, ok := searchLatest(acc.codes, limit, func(e lookupCode) uint32 { return e.index }); ok {
		return e.hash, e.code, true
	}
	return common.Hash{}, nil, false
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
