// Copyright 2024 The go-ethereum Authors
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
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/testrand"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/holiman/uint256"
)

type verifyContent int

const (
	verifyNothing verifyContent = iota
	verifyAccount
	verifyStorage
)

func verifyIterator(t *testing.T, expCount int, it Iterator, verify verifyContent) {
	t.Helper()

	var (
		count = 0
		last  = common.Hash{}
	)
	for it.Next() {
		hash := it.Hash()
		if bytes.Compare(last[:], hash[:]) >= 0 {
			t.Errorf("wrong order: %x >= %x", last, hash)
		}
		count++
		if verify == verifyAccount && len(it.(AccountIterator).Account()) == 0 {
			t.Errorf("iterator returned nil-value for hash %x", hash)
		} else if verify == verifyStorage && len(it.(StorageIterator).Slot()) == 0 {
			t.Errorf("iterator returned nil-value for hash %x", hash)
		}
		last = hash
	}
	if count != expCount {
		t.Errorf("iterator count mismatch: have %d, want %d", count, expCount)
	}
	if err := it.Error(); err != nil {
		t.Errorf("iterator failed: %v", err)
	}
}

// randomAccount generates a random account and returns it RLP encoded.
func randomAccount() []byte {
	a := &types.StateAccount{
		Balance:  uint256.NewInt(rand.Uint64()),
		Nonce:    rand.Uint64(),
		Root:     testrand.Hash(),
		CodeHash: types.EmptyCodeHash[:],
	}
	data, _ := rlp.EncodeToBytes(a)
	return data
}

// randomAccountSet generates a set of random accounts with the given strings as
// the account address hashes.
func randomAccountSet(hashes ...string) map[common.Hash][]byte {
	accounts := make(map[common.Hash][]byte)
	for _, hash := range hashes {
		accounts[common.HexToHash(hash)] = randomAccount()
	}
	return accounts
}

// randomStorageSet generates a set of random slots with the given strings as
// the slot addresses.
func randomStorageSet(accounts []string, hashes [][]string, nilStorage [][]string) map[common.Hash]map[common.Hash][]byte {
	storages := make(map[common.Hash]map[common.Hash][]byte)
	for index, account := range accounts {
		storages[common.HexToHash(account)] = make(map[common.Hash][]byte)

		if index < len(hashes) {
			hashes := hashes[index]
			for _, hash := range hashes {
				storages[common.HexToHash(account)][common.HexToHash(hash)] = testrand.Bytes(32)
			}
		}
		if index < len(nilStorage) {
			nils := nilStorage[index]
			for _, hash := range nils {
				storages[common.HexToHash(account)][common.HexToHash(hash)] = nil
			}
		}
	}
	return storages
}

// TestAccountIteratorBasics tests some simple single-layer(diff and disk) iteration
func TestAccountIteratorBasics(t *testing.T) {
	var (
		destructs = make(map[common.Hash]struct{})
		accounts  = make(map[common.Hash][]byte)
		storage   = make(map[common.Hash]map[common.Hash][]byte)
	)
	// Fill up a parent
	for i := 0; i < 100; i++ {
		hash := testrand.Hash()
		data := testrand.Bytes(32)

		accounts[hash] = data
		if rand.Intn(4) == 0 {
			destructs[hash] = struct{}{}
		}
		if rand.Intn(2) == 0 {
			accStorage := make(map[common.Hash][]byte)
			accStorage[testrand.Hash()] = testrand.Bytes(32)
			storage[hash] = accStorage
		}
	}
	states := newStates(destructs, accounts, storage)
	it := newDiffAccountIterator(common.Hash{}, states, nil)
	verifyIterator(t, 100, it, verifyNothing) // Nil is allowed for single layer iterator

	db := rawdb.NewMemoryDatabase()
	batch := db.NewBatch()
	states.write(db, batch, nil)
	batch.Write()
	it = newDiskAccountIterator(db, common.Hash{})
	verifyIterator(t, 100, it, verifyNothing) // Nil is allowed for single layer iterator
}

// TestStorageIteratorBasics tests some simple single-layer(diff and disk) iteration for storage
func TestStorageIteratorBasics(t *testing.T) {
	var (
		nilStorage = make(map[common.Hash]int)
		accounts   = make(map[common.Hash][]byte)
		storage    = make(map[common.Hash]map[common.Hash][]byte)
	)
	// Fill some random data
	for i := 0; i < 10; i++ {
		hash := testrand.Hash()
		accounts[hash] = testrand.Bytes(32)

		accStorage := make(map[common.Hash][]byte)

		var nilstorage int
		for i := 0; i < 100; i++ {
			if rand.Intn(2) == 0 {
				accStorage[testrand.Hash()] = testrand.Bytes(32)
			} else {
				accStorage[testrand.Hash()] = nil // delete slot
				nilstorage += 1
			}
		}
		storage[hash] = accStorage
		nilStorage[hash] = nilstorage
	}
	states := newStates(nil, accounts, storage)
	for account := range accounts {
		it, _ := newDiffStorageIterator(account, common.Hash{}, states, nil)
		verifyIterator(t, 100, it, verifyNothing) // Nil is allowed for single layer iterator
	}

	db := rawdb.NewMemoryDatabase()
	batch := db.NewBatch()
	states.write(db, batch, nil)
	batch.Write()
	for account := range accounts {
		it := newDiskStorageIterator(db, account, common.Hash{})
		verifyIterator(t, 100-nilStorage[account], it, verifyNothing) // Nil is allowed for single layer iterator
	}
}

type testIterator struct {
	values []byte
}

func newTestIterator(values ...byte) *testIterator {
	return &testIterator{values}
}

func (ti *testIterator) Seek(common.Hash) {
	panic("implement me")
}

func (ti *testIterator) Next() bool {
	ti.values = ti.values[1:]
	return len(ti.values) > 0
}

func (ti *testIterator) Error() error {
	return nil
}

func (ti *testIterator) Hash() common.Hash {
	return common.BytesToHash([]byte{ti.values[0]})
}

func (ti *testIterator) Account() []byte {
	return nil
}

func (ti *testIterator) Slot() []byte {
	return nil
}

func (ti *testIterator) Release() {}

func TestFastIteratorBasics(t *testing.T) {
	type testCase struct {
		lists   [][]byte
		expKeys []byte
	}
	for i, tc := range []testCase{
		{lists: [][]byte{{0, 1, 8}, {1, 2, 8}, {2, 9}, {4},
			{7, 14, 15}, {9, 13, 15, 16}},
			expKeys: []byte{0, 1, 2, 4, 7, 8, 9, 13, 14, 15, 16}},
		{lists: [][]byte{{0, 8}, {1, 2, 8}, {7, 14, 15}, {8, 9},
			{9, 10}, {10, 13, 15, 16}},
			expKeys: []byte{0, 1, 2, 7, 8, 9, 10, 13, 14, 15, 16}},
	} {
		var iterators []*weightedIterator
		for i, data := range tc.lists {
			it := newTestIterator(data...)
			iterators = append(iterators, &weightedIterator{it, i})
		}
		fi := &fastIterator{
			iterators: iterators,
			initiated: false,
		}
		count := 0
		for fi.Next() {
			if got, exp := fi.Hash()[31], tc.expKeys[count]; exp != got {
				t.Errorf("tc %d, [%d]: got %d exp %d", i, count, got, exp)
			}
			count++
		}
	}
}

// TestAccountIteratorTraversal tests some simple multi-layer iteration.
func TestAccountIteratorTraversal(t *testing.T) {
	config := &Config{
		DirtyCacheSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	db.WaitGeneration()

	// Stack three diff layers on top with various overlaps
	db.Update(common.HexToHash("0x02"), types.EmptyRootHash, 0, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa", "0xee", "0xff", "0xf0"), nil, nil, nil))

	db.Update(common.HexToHash("0x03"), common.HexToHash("0x02"), 0, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xbb", "0xdd", "0xf0"), nil, nil, nil))

	db.Update(common.HexToHash("0x04"), common.HexToHash("0x03"), 0, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xcc", "0xf0", "0xff"), nil, nil, nil))

	// Verify the single and multi-layer iterators
	head := db.tree.get(common.HexToHash("0x04"))

	it := newDiffAccountIterator(common.Hash{}, head.(*diffLayer).states.stateSet, nil)
	verifyIterator(t, 3, it, verifyNothing)
	verifyIterator(t, 7, head.(*diffLayer).newBinaryAccountIterator(), verifyAccount)

	it, _ = db.AccountIterator(common.HexToHash("0x04"), common.Hash{})
	verifyIterator(t, 7, it, verifyAccount)
	it.Release()

	// Test after persist some bottom-most layers into the disk,
	// the functionalities still work.
	db.tree.cap(common.HexToHash("0x04"), 2)

	head = db.tree.get(common.HexToHash("0x04"))
	verifyIterator(t, 7, head.(*diffLayer).newBinaryAccountIterator(), verifyAccount)

	it, _ = db.AccountIterator(common.HexToHash("0x04"), common.Hash{})
	verifyIterator(t, 7, it, verifyAccount)
	it.Release()
}

func TestStorageIteratorTraversal(t *testing.T) {
	config := &Config{
		DirtyCacheSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	db.WaitGeneration()

	// Stack three diff layers on top with various overlaps
	db.Update(common.HexToHash("0x02"), types.EmptyRootHash, 0, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), randomStorageSet([]string{"0xaa"}, [][]string{{"0x01", "0x02", "0x03"}}, nil), nil, nil))

	db.Update(common.HexToHash("0x03"), common.HexToHash("0x02"), 0, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), randomStorageSet([]string{"0xaa"}, [][]string{{"0x04", "0x05", "0x06"}}, nil), nil, nil))

	db.Update(common.HexToHash("0x04"), common.HexToHash("0x03"), 0, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), randomStorageSet([]string{"0xaa"}, [][]string{{"0x01", "0x02", "0x03"}}, nil), nil, nil))

	// Verify the single and multi-layer iterators
	head := db.tree.get(common.HexToHash("0x04"))

	diffIter, _ := newDiffStorageIterator(common.HexToHash("0xaa"), common.Hash{}, head.(*diffLayer).states.stateSet, nil)
	verifyIterator(t, 3, diffIter, verifyNothing)
	verifyIterator(t, 6, head.(*diffLayer).newBinaryStorageIterator(common.HexToHash("0xaa")), verifyStorage)

	it, _ := db.StorageIterator(common.HexToHash("0x04"), common.HexToHash("0xaa"), common.Hash{})
	verifyIterator(t, 6, it, verifyStorage)
	it.Release()

	// Test after persist some bottom-most layers into the disk,
	// the functionalities still work.
	db.tree.cap(common.HexToHash("0x04"), 2)
	verifyIterator(t, 6, head.(*diffLayer).newBinaryStorageIterator(common.HexToHash("0xaa")), verifyStorage)

	it, _ = db.StorageIterator(common.HexToHash("0x04"), common.HexToHash("0xaa"), common.Hash{})
	verifyIterator(t, 6, it, verifyStorage)
	it.Release()
}
