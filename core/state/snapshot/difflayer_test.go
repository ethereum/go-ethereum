// Copyright 2019 The go-ethereum Authors
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

package snapshot

import (
	"bytes"
	"math/big"
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func randomAccount() []byte {
	root := randomHash()
	a := Account{
		Balance:  big.NewInt(rand.Int63()),
		Nonce:    rand.Uint64(),
		Root:     root[:],
		CodeHash: emptyCode[:],
	}
	data, _ := rlp.EncodeToBytes(a)
	return data
}

// TestMergeBasics tests some simple merges
func TestMergeBasics(t *testing.T) {
	var (
		accounts = make(map[common.Hash][]byte)
		storage  = make(map[common.Hash]map[common.Hash][]byte)
	)
	// Fill up a parent
	for i := 0; i < 100; i++ {
		h := randomHash()
		data := randomAccount()

		accounts[h] = data
		if rand.Intn(20) < 10 {
			accStorage := make(map[common.Hash][]byte)
			value := make([]byte, 32)
			rand.Read(value)
			accStorage[randomHash()] = value
			storage[h] = accStorage
		}
	}
	// Add some (identical) layers on top
	parent := newDiffLayer(emptyLayer{}, 1, common.Hash{}, accounts, storage)
	child := newDiffLayer(parent, 1, common.Hash{}, accounts, storage)
	child = newDiffLayer(child, 1, common.Hash{}, accounts, storage)
	child = newDiffLayer(child, 1, common.Hash{}, accounts, storage)
	child = newDiffLayer(child, 1, common.Hash{}, accounts, storage)
	// And flatten
	merged := (child.flatten()).(*diffLayer)

	{ // Check account lists
		// Should be zero/nil first
		if got, exp := len(merged.accountList), 0; got != exp {
			t.Errorf("accountList wrong, got %v exp %v", got, exp)
		}
		// Then set when we call AccountList
		if got, exp := len(merged.AccountList()), len(accounts); got != exp {
			t.Errorf("AccountList() wrong, got %v exp %v", got, exp)
		}
		if got, exp := len(merged.accountList), len(accounts); got != exp {
			t.Errorf("accountList [2] wrong, got %v exp %v", got, exp)
		}
	}
	{ // Check storage lists
		i := 0
		for aHash, sMap := range storage {
			if got, exp := len(merged.storageList), i; got != exp {
				t.Errorf("[1] storageList wrong, got %v exp %v", got, exp)
			}
			if got, exp := len(merged.StorageList(aHash)), len(sMap); got != exp {
				t.Errorf("[2] StorageList() wrong, got %v exp %v", got, exp)
			}
			if got, exp := len(merged.storageList[aHash]), len(sMap); got != exp {
				t.Errorf("storageList wrong, got %v exp %v", got, exp)
			}
			i++
		}
	}
}

// TestMergeDelete tests some deletion
func TestMergeDelete(t *testing.T) {
	var (
		storage = make(map[common.Hash]map[common.Hash][]byte)
	)
	// Fill up a parent
	h1 := common.HexToHash("0x01")
	h2 := common.HexToHash("0x02")

	flip := func() map[common.Hash][]byte {
		accs := make(map[common.Hash][]byte)
		accs[h1] = randomAccount()
		accs[h2] = nil
		return accs
	}
	flop := func() map[common.Hash][]byte {
		accs := make(map[common.Hash][]byte)
		accs[h1] = nil
		accs[h2] = randomAccount()
		return accs
	}

	// Add some flip-flopping layers on top
	parent := newDiffLayer(emptyLayer{}, 1, common.Hash{}, flip(), storage)
	child := parent.Update(common.Hash{}, flop(), storage)
	child = child.Update(common.Hash{}, flip(), storage)
	child = child.Update(common.Hash{}, flop(), storage)
	child = child.Update(common.Hash{}, flip(), storage)
	child = child.Update(common.Hash{}, flop(), storage)
	child = child.Update(common.Hash{}, flip(), storage)

	if data, _ := child.Account(h1); data == nil {
		t.Errorf("last diff layer: expected %x to be non-nil", h1)
	}
	if data, _ := child.Account(h2); data != nil {
		t.Errorf("last diff layer: expected %x to be nil", h2)
	}
	// And flatten
	merged := (child.flatten()).(*diffLayer)

	// check number
	if got, exp := merged.number, child.number; got != exp {
		t.Errorf("merged layer: wrong number - exp %d got %d", exp, got)
	}
	if data, _ := merged.Account(h1); data == nil {
		t.Errorf("merged layer: expected %x to be non-nil", h1)
	}
	if data, _ := merged.Account(h2); data != nil {
		t.Errorf("merged layer: expected %x to be nil", h2)
	}
	// If we add more granular metering of memory, we can enable this again,
	// but it's not implemented for now
	//if got, exp := merged.memory, child.memory; got != exp {
	//	t.Errorf("mem wrong, got %d, exp %d", got, exp)
	//}
}

// This tests that if we create a new account, and set a slot, and then merge
// it, the lists will be correct.
func TestInsertAndMerge(t *testing.T) {
	// Fill up a parent
	var (
		acc    = common.HexToHash("0x01")
		slot   = common.HexToHash("0x02")
		parent *diffLayer
		child  *diffLayer
	)
	{
		var accounts = make(map[common.Hash][]byte)
		var storage = make(map[common.Hash]map[common.Hash][]byte)
		parent = newDiffLayer(emptyLayer{}, 1, common.Hash{}, accounts, storage)
	}
	{
		var accounts = make(map[common.Hash][]byte)
		var storage = make(map[common.Hash]map[common.Hash][]byte)
		accounts[acc] = randomAccount()
		accstorage := make(map[common.Hash][]byte)
		storage[acc] = accstorage
		storage[acc][slot] = []byte{0x01}
		child = newDiffLayer(parent, 2, common.Hash{}, accounts, storage)
	}
	// And flatten
	merged := (child.flatten()).(*diffLayer)
	{ // Check that slot value is present
		got, _ := merged.Storage(acc, slot)
		if exp := []byte{0x01}; bytes.Compare(got, exp) != 0 {
			t.Errorf("merged slot value wrong, got %x, exp %x", got, exp)
		}
	}
}

type emptyLayer struct{}

func (emptyLayer) Update(blockRoot common.Hash, accounts map[common.Hash][]byte, storage map[common.Hash]map[common.Hash][]byte) *diffLayer {
	panic("implement me")
}

func (emptyLayer) Journal() error {
	panic("implement me")
}

func (emptyLayer) Info() (uint64, common.Hash) {
	return 0, common.Hash{}
}
func (emptyLayer) Number() uint64 {
	return 0
}

func (emptyLayer) Account(hash common.Hash) (*Account, error) {
	return nil, nil
}

func (emptyLayer) AccountRLP(hash common.Hash) ([]byte, error) {
	return nil, nil
}

func (emptyLayer) Storage(accountHash, storageHash common.Hash) ([]byte, error) {
	return nil, nil
}

// BenchmarkSearch checks how long it takes to find a non-existing key
// BenchmarkSearch-6   	  200000	     10481 ns/op (1K per layer)
// BenchmarkSearch-6   	  200000	     10760 ns/op (10K per layer)
// BenchmarkSearch-6   	  100000	     17866 ns/op
//
// BenchmarkSearch-6   	  500000	      3723 ns/op (10k per layer, only top-level RLock()
func BenchmarkSearch(b *testing.B) {
	// First, we set up 128 diff layers, with 1K items each

	blocknum := uint64(0)
	fill := func(parent snapshot) *diffLayer {
		accounts := make(map[common.Hash][]byte)
		storage := make(map[common.Hash]map[common.Hash][]byte)

		for i := 0; i < 10000; i++ {
			accounts[randomHash()] = randomAccount()
		}
		blocknum++
		return newDiffLayer(parent, blocknum, common.Hash{}, accounts, storage)
	}

	var layer snapshot
	layer = emptyLayer{}
	for i := 0; i < 128; i++ {
		layer = fill(layer)
	}

	key := common.Hash{}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		layer.AccountRLP(key)
	}
}

// BenchmarkSearchSlot checks how long it takes to find a non-existing key
// - Number of layers: 128
// - Each layers contains the account, with a couple of storage slots
// BenchmarkSearchSlot-6   	  100000	     14554 ns/op
// BenchmarkSearchSlot-6   	  100000	     22254 ns/op (when checking parent root using mutex)
// BenchmarkSearchSlot-6   	  100000	     14551 ns/op (when checking parent number using atomic)
func BenchmarkSearchSlot(b *testing.B) {
	// First, we set up 128 diff layers, with 1K items each

	blocknum := uint64(0)
	accountKey := common.Hash{}
	storageKey := common.HexToHash("0x1337")
	accountRLP := randomAccount()
	fill := func(parent snapshot) *diffLayer {
		accounts := make(map[common.Hash][]byte)
		accounts[accountKey] = accountRLP
		storage := make(map[common.Hash]map[common.Hash][]byte)

		accStorage := make(map[common.Hash][]byte)
		for i := 0; i < 5; i++ {
			value := make([]byte, 32)
			rand.Read(value)
			accStorage[randomHash()] = value
			storage[accountKey] = accStorage
		}
		blocknum++
		return newDiffLayer(parent, blocknum, common.Hash{}, accounts, storage)
	}

	var layer snapshot
	layer = emptyLayer{}
	for i := 0; i < 128; i++ {
		layer = fill(layer)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		layer.Storage(accountKey, storageKey)
	}
}

// With accountList and sorting
//BenchmarkFlatten-6   	      50	  29890856 ns/op
//
// Without sorting and tracking accountlist
// BenchmarkFlatten-6   	     300	   5511511 ns/op
func BenchmarkFlatten(b *testing.B) {
	fill := func(parent snapshot, blocknum int) *diffLayer {
		accounts := make(map[common.Hash][]byte)
		storage := make(map[common.Hash]map[common.Hash][]byte)

		for i := 0; i < 100; i++ {
			accountKey := randomHash()
			accounts[accountKey] = randomAccount()

			accStorage := make(map[common.Hash][]byte)
			for i := 0; i < 20; i++ {
				value := make([]byte, 32)
				rand.Read(value)
				accStorage[randomHash()] = value

			}
			storage[accountKey] = accStorage
		}
		return newDiffLayer(parent, uint64(blocknum), common.Hash{}, accounts, storage)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var layer snapshot
		layer = emptyLayer{}
		for i := 1; i < 128; i++ {
			layer = fill(layer, i)
		}
		b.StartTimer()

		for i := 1; i < 128; i++ {
			dl, ok := layer.(*diffLayer)
			if !ok {
				break
			}

			layer = dl.flatten()
		}
		b.StopTimer()
	}
}
