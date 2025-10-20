// Copyright 2023 The go-ethereum Authors
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

package pathdb

import (
	"bytes"
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/rlp"
)

// randomStateSet generates a random state change set.
func randomStateSet(n int) (map[common.Address][]byte, map[common.Address]map[common.Hash][]byte) {
	var (
		accounts = make(map[common.Address][]byte)
		storages = make(map[common.Address]map[common.Hash][]byte)
	)
	for i := 0; i < n; i++ {
		addr := testrand.Address()
		storages[addr] = make(map[common.Hash][]byte)
		for j := 0; j < 3; j++ {
			v, _ := rlp.EncodeToBytes(common.TrimLeftZeroes(testrand.Bytes(32)))
			storages[addr][testrand.Hash()] = v
		}
		account := generateAccount(types.EmptyRootHash)
		accounts[addr] = types.SlimAccountRLP(account)
	}
	return accounts, storages
}

func makeStateHistory(rawStorageKey bool) *stateHistory {
	accounts, storages := randomStateSet(3)
	return newStateHistory(testrand.Hash(), types.EmptyRootHash, 0, accounts, storages, rawStorageKey)
}

func makeStateHistories(n int) []*stateHistory {
	var (
		parent = types.EmptyRootHash
		result []*stateHistory
	)
	for i := 0; i < n; i++ {
		root := testrand.Hash()
		accounts, storages := randomStateSet(3)
		h := newStateHistory(root, parent, uint64(i), accounts, storages, false)
		parent = root
		result = append(result, h)
	}
	return result
}

func TestEncodeDecodeStateHistory(t *testing.T) {
	testEncodeDecodeStateHistory(t, false)
	testEncodeDecodeStateHistory(t, true)
}

func testEncodeDecodeStateHistory(t *testing.T, rawStorageKey bool) {
	var (
		m   meta
		dec stateHistory
		obj = makeStateHistory(rawStorageKey)
	)
	// check if meta data can be correctly encode/decode
	blob := obj.meta.encode()
	if err := m.decode(blob); err != nil {
		t.Fatalf("Failed to decode %v", err)
	}
	if !reflect.DeepEqual(&m, obj.meta) {
		t.Fatal("meta is mismatched")
	}

	// check if account/storage data can be correctly encode/decode
	accountData, storageData, accountIndexes, storageIndexes := obj.encode()
	if err := dec.decode(accountData, storageData, accountIndexes, storageIndexes); err != nil {
		t.Fatalf("Failed to decode, err: %v", err)
	}
	if !compareSet(dec.accounts, obj.accounts) {
		t.Fatal("account data is mismatched")
	}
	if !compareMapSet(dec.storages, obj.storages) {
		t.Fatal("storage data is mismatched")
	}
	if !compareList(dec.accountList, obj.accountList) {
		t.Fatal("account list is mismatched")
	}
	if !compareMapList(dec.storageList, obj.storageList) {
		t.Fatal("storage list is mismatched")
	}
}

func checkStateHistory(t *testing.T, freezer ethdb.AncientReader, id uint64, exist bool) {
	blob := rawdb.ReadStateHistoryMeta(freezer, id)
	if exist && len(blob) == 0 {
		t.Fatalf("Failed to load trie history, %d", id)
	}
	if !exist && len(blob) != 0 {
		t.Fatalf("Unexpected trie history, %d", id)
	}
}

func checkHistoriesInRange(t *testing.T, freezer ethdb.AncientReader, from, to uint64, exist bool) {
	for i := from; i <= to; i = i + 1 {
		checkStateHistory(t, freezer, i, exist)
	}
}

func TestTruncateHeadStateHistory(t *testing.T) {
	var (
		hs         = makeStateHistories(10)
		freezer, _ = rawdb.NewStateFreezer(t.TempDir(), false, false)
	)
	defer freezer.Close()

	for i := 0; i < len(hs); i++ {
		accountData, storageData, accountIndex, storageIndex := hs[i].encode()
		rawdb.WriteStateHistory(freezer, uint64(i+1), hs[i].meta.encode(), accountIndex, storageIndex, accountData, storageData)
	}
	for size := len(hs); size > 0; size-- {
		pruned, err := truncateFromHead(freezer, typeStateHistory, uint64(size-1))
		if err != nil {
			t.Fatalf("Failed to truncate from head %v", err)
		}
		if pruned != 1 {
			t.Error("Unexpected pruned items", "want", 1, "got", pruned)
		}
		checkHistoriesInRange(t, freezer, uint64(size), uint64(10), false)
		checkHistoriesInRange(t, freezer, uint64(1), uint64(size-1), true)
	}
}

func TestTruncateTailStateHistory(t *testing.T) {
	var (
		hs         = makeStateHistories(10)
		freezer, _ = rawdb.NewStateFreezer(t.TempDir(), false, false)
	)
	defer freezer.Close()

	for i := 0; i < len(hs); i++ {
		accountData, storageData, accountIndex, storageIndex := hs[i].encode()
		rawdb.WriteStateHistory(freezer, uint64(i+1), hs[i].meta.encode(), accountIndex, storageIndex, accountData, storageData)
	}
	for newTail := 1; newTail < len(hs); newTail++ {
		pruned, _ := truncateFromTail(freezer, typeStateHistory, uint64(newTail))
		if pruned != 1 {
			t.Error("Unexpected pruned items", "want", 1, "got", pruned)
		}
		checkHistoriesInRange(t, freezer, uint64(1), uint64(newTail), false)
		checkHistoriesInRange(t, freezer, uint64(newTail+1), uint64(10), true)
	}
}

func TestTruncateTailStateHistories(t *testing.T) {
	var cases = []struct {
		limit       uint64
		expPruned   int
		maxPruned   uint64
		minUnpruned uint64
		empty       bool
	}{
		// history: id [10]
		{
			limit:     1,
			expPruned: 9,
			maxPruned: 9, minUnpruned: 10, empty: false,
		},
		// history: none
		{
			limit:     0,
			expPruned: 10,
			empty:     true,
		},
		// history: id [1:10]
		{
			limit:       10,
			expPruned:   0,
			maxPruned:   0,
			minUnpruned: 1,
		},
	}
	for i, c := range cases {
		var (
			hs         = makeStateHistories(10)
			freezer, _ = rawdb.NewStateFreezer(t.TempDir()+fmt.Sprintf("%d", i), false, false)
		)
		defer freezer.Close()

		for i := 0; i < len(hs); i++ {
			accountData, storageData, accountIndex, storageIndex := hs[i].encode()
			rawdb.WriteStateHistory(freezer, uint64(i+1), hs[i].meta.encode(), accountIndex, storageIndex, accountData, storageData)
		}
		pruned, _ := truncateFromTail(freezer, typeStateHistory, uint64(10)-c.limit)
		if pruned != c.expPruned {
			t.Error("Unexpected pruned items", "want", c.expPruned, "got", pruned)
		}
		if c.empty {
			checkHistoriesInRange(t, freezer, uint64(1), uint64(10), false)
		} else {
			checkHistoriesInRange(t, freezer, uint64(1), c.maxPruned, false)
			checkHistoriesInRange(t, freezer, c.minUnpruned, uint64(10), true)
		}
	}
}

func TestTruncateOutOfRange(t *testing.T) {
	var (
		hs         = makeStateHistories(10)
		freezer, _ = rawdb.NewStateFreezer(t.TempDir(), false, false)
	)
	defer freezer.Close()

	for i := 0; i < len(hs); i++ {
		accountData, storageData, accountIndex, storageIndex := hs[i].encode()
		rawdb.WriteStateHistory(freezer, uint64(i+1), hs[i].meta.encode(), accountIndex, storageIndex, accountData, storageData)
	}
	truncateFromTail(freezer, typeStateHistory, uint64(len(hs)/2))

	// Ensure of-out-range truncations are rejected correctly.
	head, _ := freezer.Ancients()
	tail, _ := freezer.Tail()

	cases := []struct {
		mode   int
		target uint64
		expErr error
	}{
		{0, head, nil}, // nothing to delete
		{0, head + 1, errHeadTruncationOutOfRange},
		{0, tail - 1, errHeadTruncationOutOfRange},
		{1, tail, nil}, // nothing to delete
		{1, head + 1, errTailTruncationOutOfRange},
		{1, tail - 1, errTailTruncationOutOfRange},
	}
	for _, c := range cases {
		var gotErr error
		if c.mode == 0 {
			_, gotErr = truncateFromHead(freezer, typeStateHistory, c.target)
		} else {
			_, gotErr = truncateFromTail(freezer, typeStateHistory, c.target)
		}
		if !errors.Is(gotErr, c.expErr) {
			t.Errorf("Unexpected error, want: %v, got: %v", c.expErr, gotErr)
		}
	}
}

func compareSet[k comparable](a, b map[k][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for key, valA := range a {
		valB, ok := b[key]
		if !ok {
			return false
		}
		if !bytes.Equal(valA, valB) {
			return false
		}
	}
	return true
}

func compareList[k comparable](a, b []k) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func compareMapSet[K1 comparable, K2 comparable](a, b map[K1]map[K2][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for key, subsetA := range a {
		subsetB, ok := b[key]
		if !ok {
			return false
		}
		if !compareSet(subsetA, subsetB) {
			return false
		}
	}
	return true
}

func compareMapList[K comparable, V comparable](a, b map[K][]V) bool {
	if len(a) != len(b) {
		return false
	}
	for key, listA := range a {
		listB, ok := b[key]
		if !ok {
			return false
		}
		if !compareList(listA, listB) {
			return false
		}
	}
	return true
}
