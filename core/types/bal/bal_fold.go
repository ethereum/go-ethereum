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
	"bytes"
	"maps"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// foldedAccount carries an account's latest value per field across the folded
// range; nil means the field never changed.
type foldedAccount struct {
	storage map[[32]byte]*uint256.Int
	balance *uint256.Int
	nonce   *uint64
	code    []byte
	hasCode bool
}

// mutated reports whether an access entry changes anything.
func mutated(acc *AccountAccess) bool {
	return len(acc.StorageChanges) != 0 || len(acc.BalanceChanges) != 0 ||
		len(acc.NonceChanges) != 0 || len(acc.CodeChanges) != 0
}

// Fold coalesces consecutive canonical (header-proven) lists into one
// installing the range's end state, last write wins. A removed account must
// not fold past its recreation, so a consumed prefix length is returned.
func Fold(lists []*BlockAccessList) (*BlockAccessList, int) {
	if len(lists) == 0 {
		return nil, 0
	}
	folded := make(map[common.Address]*foldedAccount)
	taken := 0
	for _, list := range lists {
		if list != nil && taken > 0 && touchesRemoved(folded, *list) {
			break
		}
		if list != nil {
			for i := range *list {
				acc := &(*list)[i]
				if !mutated(acc) {
					continue
				}
				fa := folded[acc.Address]
				if fa == nil {
					fa = &foldedAccount{storage: make(map[[32]byte]*uint256.Int)}
					folded[acc.Address] = fa
				}
				for _, sc := range acc.StorageChanges {
					if n := len(sc.SlotChanges); n > 0 {
						fa.storage[sc.Slot.Bytes32()] = sc.SlotChanges[n-1].PostValue
					}
				}
				if n := len(acc.BalanceChanges); n > 0 {
					fa.balance = acc.BalanceChanges[n-1].PostBalance
				}
				if n := len(acc.NonceChanges); n > 0 {
					nonce := acc.NonceChanges[n-1].PostNonce
					fa.nonce = &nonce
				}
				if n := len(acc.CodeChanges); n > 0 {
					fa.code = acc.CodeChanges[n-1].NewCode
					fa.hasCode = true
				}
			}
		}
		taken++
	}
	out := make(BlockAccessList, 0, len(folded))
	for _, addr := range slices.SortedFunc(maps.Keys(folded), common.Address.Cmp) {
		fa := folded[addr]
		entry := AccountAccess{Address: addr}
		for _, key := range slices.SortedFunc(maps.Keys(fa.storage), func(a, b [32]byte) int {
			return bytes.Compare(a[:], b[:])
		}) {
			entry.StorageChanges = append(entry.StorageChanges, encodingSlotChanges{
				Slot:        new(uint256.Int).SetBytes32(key[:]),
				SlotChanges: []encodingStorageWrite{{BlockAccessIndex: 0, PostValue: fa.storage[key]}},
			})
		}
		if fa.balance != nil {
			entry.BalanceChanges = []encodingBalanceChange{{BlockAccessIndex: 0, PostBalance: fa.balance}}
		}
		if fa.nonce != nil {
			entry.NonceChanges = []encodingAccountNonce{{BlockAccessIndex: 0, PostNonce: *fa.nonce}}
		}
		if fa.hasCode {
			entry.CodeChanges = []encodingCodeChange{{BlockAccessIndex: 0, NewCode: fa.code}}
		}
		out = append(out, entry)
	}
	return &out, taken
}

// possiblyRemoved reports whether the folded account may be gone at a block
// boundary: something changed, and every changed field is consistent with
// deletion's emptiness trigger. Unchanged fields must count as consistent.
func (fa *foldedAccount) possiblyRemoved() bool {
	if fa.balance == nil && fa.nonce == nil && !fa.hasCode {
		return false
	}
	if fa.balance != nil && !fa.balance.IsZero() {
		return false
	}
	if fa.nonce != nil && *fa.nonce != 0 {
		return false
	}
	if fa.hasCode && len(fa.code) != 0 {
		return false
	}
	return true
}

// touchesRemoved reports whether the list mutates an account the folded range
// may have removed: the removal's storage wipe has to materialize before a
// recreation applies.
func touchesRemoved(folded map[common.Address]*foldedAccount, list BlockAccessList) bool {
	for i := range list {
		acc := &list[i]
		if !mutated(acc) {
			continue
		}
		if fa := folded[acc.Address]; fa != nil && fa.possiblyRemoved() {
			return true
		}
	}
	return false
}
