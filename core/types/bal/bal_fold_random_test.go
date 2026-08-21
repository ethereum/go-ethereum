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
	"math/rand"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

// fuzzState mirrors ApplyBlockAccessList: last write per field wins, and an
// account left empty is removed together with its storage.
type fuzzState map[common.Address]*fuzzAccount

type fuzzAccount struct {
	balance uint64
	nonce   uint64
	code    []byte
	storage map[[32]byte]uint64
}

func (st fuzzState) apply(list *BlockAccessList) {
	for i := range *list {
		acc := &(*list)[i]
		if !mutated(acc) {
			continue
		}
		obj := st[acc.Address]
		if obj == nil {
			obj = &fuzzAccount{storage: make(map[[32]byte]uint64)}
			st[acc.Address] = obj
		}
		for _, sc := range acc.StorageChanges {
			if n := len(sc.SlotChanges); n > 0 {
				value := sc.SlotChanges[n-1].PostValue.Uint64()
				if value == 0 {
					delete(obj.storage, sc.Slot.Bytes32())
				} else {
					obj.storage[sc.Slot.Bytes32()] = value
				}
			}
		}
		if n := len(acc.BalanceChanges); n > 0 {
			obj.balance = acc.BalanceChanges[n-1].PostBalance.Uint64()
		}
		if n := len(acc.NonceChanges); n > 0 {
			obj.nonce = acc.NonceChanges[n-1].PostNonce
		}
		if n := len(acc.CodeChanges); n > 0 {
			obj.code = acc.CodeChanges[n-1].NewCode
		}
		if obj.balance == 0 && obj.nonce == 0 && len(obj.code) == 0 {
			delete(st, acc.Address)
		}
	}
}

// randomList builds a small canonical-shaped access list.
func randomList(rng *rand.Rand) *BlockAccessList {
	var (
		pool = []common.Address{{0x01}, {0x02}, {0x03}}
		list BlockAccessList
	)
	for _, addr := range pool {
		if rng.Intn(2) == 0 {
			continue
		}
		acc := AccountAccess{Address: addr}
		idx := uint32(rng.Intn(8))
		if rng.Intn(2) == 0 {
			acc.BalanceChanges = []encodingBalanceChange{{BlockAccessIndex: idx, PostBalance: uint256.NewInt(uint64(rng.Intn(3)))}}
		}
		if rng.Intn(2) == 0 {
			acc.NonceChanges = []encodingAccountNonce{{BlockAccessIndex: idx, PostNonce: uint64(rng.Intn(2))}}
		}
		if rng.Intn(3) == 0 {
			code := []byte{}
			if rng.Intn(2) == 0 {
				code = []byte{0x60}
			}
			acc.CodeChanges = []encodingCodeChange{{BlockAccessIndex: idx, NewCode: code}}
		}
		for slot := 0; slot < 2; slot++ {
			if rng.Intn(2) == 0 {
				acc.StorageChanges = append(acc.StorageChanges, encodingSlotChanges{
					Slot:        uint256.NewInt(uint64(slot)),
					SlotChanges: []encodingStorageWrite{{BlockAccessIndex: idx, PostValue: uint256.NewInt(uint64(rng.Intn(3)))}},
				})
			}
		}
		// Pin Fold's canonical-list invariant: bare storage writes carry
		// either alive metadata or an explicit removal marker.
		if len(acc.StorageChanges) > 0 && len(acc.BalanceChanges) == 0 && len(acc.NonceChanges) == 0 && len(acc.CodeChanges) == 0 {
			if rng.Intn(2) == 0 {
				acc.CodeChanges = []encodingCodeChange{{BlockAccessIndex: idx, NewCode: []byte{0x60}}}
			} else {
				acc.BalanceChanges = []encodingBalanceChange{{BlockAccessIndex: idx, PostBalance: uint256.NewInt(0)}}
			}
		}
		if mutated(&acc) {
			list = append(list, acc)
		}
	}
	return &list
}

// TestFoldMatchesSequentialApply drives random block sequences through the
// interpreter list by list and through Fold's batching loop; the end states
// must agree, or a fold leaked storage a removal wiped.
func TestFoldMatchesSequentialApply(t *testing.T) {
	rng := rand.New(rand.NewSource(0x8347))
	for round := 0; round < 500; round++ {
		lists := make([]*BlockAccessList, 3+rng.Intn(5))
		for i := range lists {
			lists[i] = randomList(rng)
		}
		sequential := make(fuzzState)
		for _, list := range lists {
			sequential.apply(list)
		}
		folded := make(fuzzState)
		for rest := lists; len(rest) > 0; {
			batch, taken := Fold(rest)
			if taken == 0 {
				t.Fatalf("round %d: fold made no progress", round)
			}
			folded.apply(batch)
			rest = rest[taken:]
		}
		if !reflect.DeepEqual(sequential, folded) {
			t.Fatalf("round %d (seed 0x8347): folded state diverged\nlists: %+v", round, lists)
		}
	}
}
