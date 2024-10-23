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
	"encoding/binary"
	"fmt"
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

	// TODO reenable these tests once the persistent state iteration
	// is implemented.

	//db := rawdb.NewMemoryDatabase()
	//batch := db.NewBatch()
	//states.write(db, batch, nil, nil)
	//batch.Write()
	//it = newDiskAccountIterator(db, common.Hash{})
	//verifyIterator(t, 100, it, verifyNothing) // Nil is allowed for single layer iterator
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

	// TODO reenable these tests once the persistent state iteration
	// is implemented.

	//db := rawdb.NewMemoryDatabase()
	//batch := db.NewBatch()
	//states.write(db, batch, nil, nil)
	//batch.Write()
	//for account := range accounts {
	//	it := newDiskStorageIterator(db, account, common.Hash{})
	//	verifyIterator(t, 100-nilStorage[account], it, verifyNothing) // Nil is allowed for single layer iterator
	//}
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
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

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

	// TODO reenable these tests once the persistent state iteration
	// is implemented.

	// Test after persist some bottom-most layers into the disk,
	// the functionalities still work.
	//db.tree.cap(common.HexToHash("0x04"), 2)

	//head = db.tree.get(common.HexToHash("0x04"))
	//verifyIterator(t, 7, head.(*diffLayer).newBinaryAccountIterator(), verifyAccount)
	//
	//it, _ = db.AccountIterator(common.HexToHash("0x04"), common.Hash{})
	//verifyIterator(t, 7, it, verifyAccount)
	//it.Release()
}

func TestStorageIteratorTraversal(t *testing.T) {
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

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

	// TODO reenable these tests once the persistent state iteration
	// is implemented.

	// Test after persist some bottom-most layers into the disk,
	// the functionalities still work.
	//db.tree.cap(common.HexToHash("0x04"), 2)
	//verifyIterator(t, 6, head.(*diffLayer).newBinaryStorageIterator(common.HexToHash("0xaa")), verifyStorage)
	//
	//it, _ = db.StorageIterator(common.HexToHash("0x04"), common.HexToHash("0xaa"), common.Hash{})
	//verifyIterator(t, 6, it, verifyStorage)
	//it.Release()
}

// TestAccountIteratorTraversalValues tests some multi-layer iteration, where we
// also expect the correct values to show up.
func TestAccountIteratorTraversalValues(t *testing.T) {
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

	// Create a batch of account sets to seed subsequent layers with
	var (
		a = make(map[common.Hash][]byte)
		b = make(map[common.Hash][]byte)
		c = make(map[common.Hash][]byte)
		d = make(map[common.Hash][]byte)
		e = make(map[common.Hash][]byte)
		f = make(map[common.Hash][]byte)
		g = make(map[common.Hash][]byte)
		h = make(map[common.Hash][]byte)
	)
	for i := byte(2); i < 0xff; i++ {
		a[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 0, i))
		if i > 20 && i%2 == 0 {
			b[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 1, i))
		}
		if i%4 == 0 {
			c[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 2, i))
		}
		if i%7 == 0 {
			d[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 3, i))
		}
		if i%8 == 0 {
			e[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 4, i))
		}
		if i > 50 || i < 85 {
			f[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 5, i))
		}
		if i%64 == 0 {
			g[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 6, i))
		}
		if i%128 == 0 {
			h[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 7, i))
		}
	}
	// Assemble a stack of snapshots from the account layers
	db.Update(common.HexToHash("0x02"), types.EmptyRootHash, 2, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, a, nil, nil, nil))
	db.Update(common.HexToHash("0x03"), common.HexToHash("0x02"), 3, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, b, nil, nil, nil))
	db.Update(common.HexToHash("0x04"), common.HexToHash("0x03"), 4, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, c, nil, nil, nil))
	db.Update(common.HexToHash("0x05"), common.HexToHash("0x04"), 5, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, d, nil, nil, nil))
	db.Update(common.HexToHash("0x06"), common.HexToHash("0x05"), 6, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, e, nil, nil, nil))
	db.Update(common.HexToHash("0x07"), common.HexToHash("0x06"), 7, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, f, nil, nil, nil))
	db.Update(common.HexToHash("0x08"), common.HexToHash("0x07"), 8, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, g, nil, nil, nil))
	db.Update(common.HexToHash("0x09"), common.HexToHash("0x08"), 9, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, h, nil, nil, nil))

	it, _ := db.AccountIterator(common.HexToHash("0x09"), common.Hash{})
	head, _ := db.StateReader(common.HexToHash("0x09"))
	for it.Next() {
		hash := it.Hash()
		want, err := head.(*reader).AccountRLP(hash)
		if err != nil {
			t.Fatalf("failed to retrieve expected account: %v", err)
		}
		if have := it.Account(); !bytes.Equal(want, have) {
			t.Fatalf("hash %x: account mismatch: have %x, want %x", hash, have, want)
		}
	}
	it.Release()

	// TODO reenable these tests once the persistent state iteration
	// is implemented.

	// Test after persist some bottom-most layers into the disk,
	// the functionalities still work.
	//db.tree.cap(common.HexToHash("0x09"), 2)
	//
	//it, _ = db.AccountIterator(common.HexToHash("0x09"), common.Hash{})
	//for it.Next() {
	//	hash := it.Hash()
	//	account, err := head.Account(hash)
	//	if err != nil {
	//		t.Fatalf("failed to retrieve expected account: %v", err)
	//	}
	//	want, _ := rlp.EncodeToBytes(account)
	//	if have := it.Account(); !bytes.Equal(want, have) {
	//		t.Fatalf("hash %x: account mismatch: have %x, want %x", hash, have, want)
	//	}
	//}
	//it.Release()
}

func TestStorageIteratorTraversalValues(t *testing.T) {
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

	wrapStorage := func(storage map[common.Hash][]byte) map[common.Hash]map[common.Hash][]byte {
		return map[common.Hash]map[common.Hash][]byte{
			common.HexToHash("0xaa"): storage,
		}
	}
	// Create a batch of storage sets to seed subsequent layers with
	var (
		a = make(map[common.Hash][]byte)
		b = make(map[common.Hash][]byte)
		c = make(map[common.Hash][]byte)
		d = make(map[common.Hash][]byte)
		e = make(map[common.Hash][]byte)
		f = make(map[common.Hash][]byte)
		g = make(map[common.Hash][]byte)
		h = make(map[common.Hash][]byte)
	)
	for i := byte(2); i < 0xff; i++ {
		a[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 0, i))
		if i > 20 && i%2 == 0 {
			b[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 1, i))
		}
		if i%4 == 0 {
			c[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 2, i))
		}
		if i%7 == 0 {
			d[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 3, i))
		}
		if i%8 == 0 {
			e[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 4, i))
		}
		if i > 50 || i < 85 {
			f[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 5, i))
		}
		if i%64 == 0 {
			g[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 6, i))
		}
		if i%128 == 0 {
			h[common.Hash{i}] = []byte(fmt.Sprintf("layer-%d, key %d", 7, i))
		}
	}
	// Assemble a stack of snapshots from the account layers
	db.Update(common.HexToHash("0x02"), types.EmptyRootHash, 2, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), wrapStorage(a), nil, nil))
	db.Update(common.HexToHash("0x03"), common.HexToHash("0x02"), 3, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), wrapStorage(b), nil, nil))
	db.Update(common.HexToHash("0x04"), common.HexToHash("0x03"), 4, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), wrapStorage(c), nil, nil))
	db.Update(common.HexToHash("0x05"), common.HexToHash("0x04"), 5, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), wrapStorage(d), nil, nil))
	db.Update(common.HexToHash("0x06"), common.HexToHash("0x05"), 6, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), wrapStorage(e), nil, nil))
	db.Update(common.HexToHash("0x07"), common.HexToHash("0x06"), 7, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), wrapStorage(f), nil, nil))
	db.Update(common.HexToHash("0x08"), common.HexToHash("0x07"), 8, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), wrapStorage(g), nil, nil))
	db.Update(common.HexToHash("0x09"), common.HexToHash("0x08"), 9, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), wrapStorage(h), nil, nil))

	it, _ := db.StorageIterator(common.HexToHash("0x09"), common.HexToHash("0xaa"), common.Hash{})
	head, _ := db.StateReader(common.HexToHash("0x09"))
	for it.Next() {
		hash := it.Hash()
		want, err := head.Storage(common.HexToHash("0xaa"), hash)
		if err != nil {
			t.Fatalf("failed to retrieve expected storage slot: %v", err)
		}
		if have := it.Slot(); !bytes.Equal(want, have) {
			t.Fatalf("hash %x: slot mismatch: have %x, want %x", hash, have, want)
		}
	}
	it.Release()

	// TODO reenable these tests once the persistent state iteration
	// is implemented.

	// Test after persist some bottom-most layers into the disk,
	// the functionalities still work.
	//db.tree.cap(common.HexToHash("0x09"), 2)
	//
	//it, _ = db.StorageIterator(common.HexToHash("0x09"), common.HexToHash("0xaa"), common.Hash{})
	//for it.Next() {
	//	hash := it.Hash()
	//	want, err := head.Storage(common.HexToHash("0xaa"), hash)
	//	if err != nil {
	//		t.Fatalf("failed to retrieve expected slot: %v", err)
	//	}
	//	if have := it.Slot(); !bytes.Equal(want, have) {
	//		t.Fatalf("hash %x: slot mismatch: have %x, want %x", hash, have, want)
	//	}
	//}
	//it.Release()
}

// This testcase is notorious, all layers contain the exact same 200 accounts.
func TestAccountIteratorLargeTraversal(t *testing.T) {
	// Create a custom account factory to recreate the same addresses
	makeAccounts := func(num int) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for i := 0; i < num; i++ {
			h := common.Hash{}
			binary.BigEndian.PutUint64(h[:], uint64(i+1))
			accounts[h] = randomAccount()
		}
		return accounts
	}
	// Build up a large stack of snapshots
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()
	for i := 1; i < 128; i++ {
		parent := types.EmptyRootHash
		if i == 1 {
			parent = common.HexToHash(fmt.Sprintf("0x%02x", i))
		}
		db.Update(common.HexToHash(fmt.Sprintf("0x%02x", i+1)), parent, uint64(i), trienode.NewMergedNodeSet(),
			NewStateSetWithOrigin(nil, makeAccounts(200), nil, nil, nil))
	}
	// Iterate the entire stack and ensure everything is hit only once
	head := db.tree.get(common.HexToHash("0x80"))
	verifyIterator(t, 200, newDiffAccountIterator(common.Hash{}, head.(*diffLayer).states.stateSet, nil), verifyNothing)
	verifyIterator(t, 200, head.(*diffLayer).newBinaryAccountIterator(), verifyAccount)

	it, _ := db.AccountIterator(common.HexToHash("0x80"), common.Hash{})
	verifyIterator(t, 200, it, verifyAccount)
	it.Release()

	// TODO reenable these tests once the persistent state iteration
	// is implemented.

	// Test after persist some bottom-most layers into the disk,
	// the functionalities still work.
	//db.tree.cap(common.HexToHash("0x80"), 2)
	//
	//verifyIterator(t, 200, head.(*diffLayer).newBinaryAccountIterator(), verifyAccount)
	//
	//it, _ = db.AccountIterator(common.HexToHash("0x80"), common.Hash{})
	//verifyIterator(t, 200, it, verifyAccount)
	//it.Release()
}

// TestAccountIteratorFlattening tests what happens when we
// - have a live iterator on child C (parent C1 -> C2 .. CN)
// - flattens C2 all the way into CN
// - continues iterating
func TestAccountIteratorFlattening(t *testing.T) {
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

	// Create a stack of diffs on top
	db.Update(common.HexToHash("0x02"), types.EmptyRootHash, 1, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa", "0xee", "0xff", "0xf0"), nil, nil, nil))

	db.Update(common.HexToHash("0x03"), common.HexToHash("0x02"), 2, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xbb", "0xdd", "0xf0"), nil, nil, nil))

	db.Update(common.HexToHash("0x04"), common.HexToHash("0x03"), 3, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xcc", "0xf0", "0xff"), nil, nil, nil))

	// Create an iterator and flatten the data from underneath it
	it, _ := db.AccountIterator(common.HexToHash("0x04"), common.Hash{})
	defer it.Release()

	if err := db.tree.cap(common.HexToHash("0x04"), 1); err != nil {
		t.Fatalf("failed to flatten snapshot stack: %v", err)
	}
	//verifyIterator(t, 7, it)
}

func TestAccountIteratorSeek(t *testing.T) {
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

	db.Update(common.HexToHash("0x02"), types.EmptyRootHash, 1, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa", "0xee", "0xff", "0xf0"), nil, nil, nil))

	db.Update(common.HexToHash("0x03"), common.HexToHash("0x02"), 2, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xbb", "0xdd", "0xf0"), nil, nil, nil))

	db.Update(common.HexToHash("0x04"), common.HexToHash("0x03"), 3, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xcc", "0xf0", "0xff"), nil, nil, nil))

	// Account set is now
	// 02: aa, ee, f0, ff
	// 03: aa, bb, dd, ee, f0 (, f0), ff
	// 04: aa, bb, cc, dd, ee, f0 (, f0), ff (, ff)
	// Construct various iterators and ensure their traversal is correct
	it, _ := db.AccountIterator(common.HexToHash("0x02"), common.HexToHash("0xdd"))
	defer it.Release()
	verifyIterator(t, 3, it, verifyAccount) // expected: ee, f0, ff

	it, _ = db.AccountIterator(common.HexToHash("0x02"), common.HexToHash("0xaa"))
	defer it.Release()
	verifyIterator(t, 4, it, verifyAccount) // expected: aa, ee, f0, ff

	it, _ = db.AccountIterator(common.HexToHash("0x02"), common.HexToHash("0xff"))
	defer it.Release()
	verifyIterator(t, 1, it, verifyAccount) // expected: ff

	it, _ = db.AccountIterator(common.HexToHash("0x02"), common.HexToHash("0xff1"))
	defer it.Release()
	verifyIterator(t, 0, it, verifyAccount) // expected: nothing

	it, _ = db.AccountIterator(common.HexToHash("0x04"), common.HexToHash("0xbb"))
	defer it.Release()
	verifyIterator(t, 6, it, verifyAccount) // expected: bb, cc, dd, ee, f0, ff

	it, _ = db.AccountIterator(common.HexToHash("0x04"), common.HexToHash("0xef"))
	defer it.Release()
	verifyIterator(t, 2, it, verifyAccount) // expected: f0, ff

	it, _ = db.AccountIterator(common.HexToHash("0x04"), common.HexToHash("0xf0"))
	defer it.Release()
	verifyIterator(t, 2, it, verifyAccount) // expected: f0, ff

	it, _ = db.AccountIterator(common.HexToHash("0x04"), common.HexToHash("0xff"))
	defer it.Release()
	verifyIterator(t, 1, it, verifyAccount) // expected: ff

	it, _ = db.AccountIterator(common.HexToHash("0x04"), common.HexToHash("0xff1"))
	defer it.Release()
	verifyIterator(t, 0, it, verifyAccount) // expected: nothing
}

func TestStorageIteratorSeek(t *testing.T) {
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

	// Stack three diff layers on top with various overlaps
	db.Update(common.HexToHash("0x02"), types.EmptyRootHash, 1, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), randomStorageSet([]string{"0xaa"}, [][]string{{"0x01", "0x03", "0x05"}}, nil), nil, nil))

	db.Update(common.HexToHash("0x03"), common.HexToHash("0x02"), 2, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), randomStorageSet([]string{"0xaa"}, [][]string{{"0x02", "0x05", "0x06"}}, nil), nil, nil))

	db.Update(common.HexToHash("0x04"), common.HexToHash("0x03"), 3, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), randomStorageSet([]string{"0xaa"}, [][]string{{"0x01", "0x05", "0x08"}}, nil), nil, nil))

	// Account set is now
	// 02: 01, 03, 05
	// 03: 01, 02, 03, 05 (, 05), 06
	// 04: 01(, 01), 02, 03, 05(, 05, 05), 06, 08
	// Construct various iterators and ensure their traversal is correct
	it, _ := db.StorageIterator(common.HexToHash("0x02"), common.HexToHash("0xaa"), common.HexToHash("0x01"))
	defer it.Release()
	verifyIterator(t, 3, it, verifyStorage) // expected: 01, 03, 05

	it, _ = db.StorageIterator(common.HexToHash("0x02"), common.HexToHash("0xaa"), common.HexToHash("0x02"))
	defer it.Release()
	verifyIterator(t, 2, it, verifyStorage) // expected: 03, 05

	it, _ = db.StorageIterator(common.HexToHash("0x02"), common.HexToHash("0xaa"), common.HexToHash("0x5"))
	defer it.Release()
	verifyIterator(t, 1, it, verifyStorage) // expected: 05

	it, _ = db.StorageIterator(common.HexToHash("0x02"), common.HexToHash("0xaa"), common.HexToHash("0x6"))
	defer it.Release()
	verifyIterator(t, 0, it, verifyStorage) // expected: nothing

	it, _ = db.StorageIterator(common.HexToHash("0x04"), common.HexToHash("0xaa"), common.HexToHash("0x01"))
	defer it.Release()
	verifyIterator(t, 6, it, verifyStorage) // expected: 01, 02, 03, 05, 06, 08

	it, _ = db.StorageIterator(common.HexToHash("0x04"), common.HexToHash("0xaa"), common.HexToHash("0x05"))
	defer it.Release()
	verifyIterator(t, 3, it, verifyStorage) // expected: 05, 06, 08

	it, _ = db.StorageIterator(common.HexToHash("0x04"), common.HexToHash("0xaa"), common.HexToHash("0x08"))
	defer it.Release()
	verifyIterator(t, 1, it, verifyStorage) // expected: 08

	it, _ = db.StorageIterator(common.HexToHash("0x04"), common.HexToHash("0xaa"), common.HexToHash("0x09"))
	defer it.Release()
	verifyIterator(t, 0, it, verifyStorage) // expected: nothing
}

// TestAccountIteratorDeletions tests that the iterator behaves correct when there are
// deleted accounts (where the Account() value is nil). The iterator
// should not output any accounts or nil-values for those cases.
func TestAccountIteratorDeletions(t *testing.T) {
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

	// Stack three diff layers on top with various overlaps
	db.Update(common.HexToHash("0x02"), types.EmptyRootHash, 1, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0x11", "0x22", "0x33"), nil, nil, nil))

	deleted := common.HexToHash("0x22")
	destructed := map[common.Hash]struct{}{
		deleted: {},
	}
	db.Update(common.HexToHash("0x03"), common.HexToHash("0x02"), 2, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(destructed, randomAccountSet("0x11", "0x33"), nil, nil, nil))

	db.Update(common.HexToHash("0x04"), common.HexToHash("0x03"), 3, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0x33", "0x44", "0x55"), nil, nil, nil))

	// The output should be 11,33,44,55
	it, _ := db.AccountIterator(common.HexToHash("0x04"), common.Hash{})
	// Do a quick check
	verifyIterator(t, 4, it, verifyAccount)
	it.Release()

	// And a more detailed verification that we indeed do not see '0x22'
	it, _ = db.AccountIterator(common.HexToHash("0x04"), common.Hash{})
	defer it.Release()
	for it.Next() {
		hash := it.Hash()
		if it.Account() == nil {
			t.Errorf("iterator returned nil-value for hash %x", hash)
		}
		if hash == deleted {
			t.Errorf("expected deleted elem %x to not be returned by iterator", deleted)
		}
	}
}

func TestStorageIteratorDeletions(t *testing.T) {
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

	// Stack three diff layers on top with various overlaps
	db.Update(common.HexToHash("0x02"), types.EmptyRootHash, 1, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), randomStorageSet([]string{"0xaa"}, [][]string{{"0x01", "0x03", "0x05"}}, nil), nil, nil))

	db.Update(common.HexToHash("0x03"), common.HexToHash("0x02"), 2, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), randomStorageSet([]string{"0xaa"}, [][]string{{"0x02", "0x04", "0x06"}}, [][]string{{"0x01", "0x03"}}), nil, nil))

	// The output should be 02,04,05,06
	it, _ := db.StorageIterator(common.HexToHash("0x03"), common.HexToHash("0xaa"), common.Hash{})
	verifyIterator(t, 4, it, verifyStorage)
	it.Release()

	// The output should be 04,05,06
	it, _ = db.StorageIterator(common.HexToHash("0x03"), common.HexToHash("0xaa"), common.HexToHash("0x03"))
	verifyIterator(t, 3, it, verifyStorage)
	it.Release()

	// Destruct the whole storage
	destructed := map[common.Hash]struct{}{
		common.HexToHash("0xaa"): {},
	}
	db.Update(common.HexToHash("0x04"), common.HexToHash("0x03"), 3, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(destructed, nil, nil, nil, nil))

	it, _ = db.StorageIterator(common.HexToHash("0x04"), common.HexToHash("0xaa"), common.Hash{})
	verifyIterator(t, 0, it, verifyStorage)
	it.Release()

	// Re-insert the slots of the same account
	db.Update(common.HexToHash("0x05"), common.HexToHash("0x04"), 4, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(nil, randomAccountSet("0xaa"), randomStorageSet([]string{"0xaa"}, [][]string{{"0x07", "0x08", "0x09"}}, nil), nil, nil))

	// The output should be 07,08,09
	it, _ = db.StorageIterator(common.HexToHash("0x05"), common.HexToHash("0xaa"), common.Hash{})
	verifyIterator(t, 3, it, verifyStorage)
	it.Release()

	// Destruct the whole storage but re-create the account in the same layer
	db.Update(common.HexToHash("0x06"), common.HexToHash("0x05"), 5, trienode.NewMergedNodeSet(),
		NewStateSetWithOrigin(destructed, randomAccountSet("0xaa"), randomStorageSet([]string{"0xaa"}, [][]string{{"0x11", "0x12"}}, nil), nil, nil))

	it, _ = db.StorageIterator(common.HexToHash("0x06"), common.HexToHash("0xaa"), common.Hash{})
	verifyIterator(t, 2, it, verifyStorage) // The output should be 11,12
	it.Release()

	verifyIterator(t, 2, db.tree.get(common.HexToHash("0x06")).(*diffLayer).newBinaryStorageIterator(common.HexToHash("0xaa")), verifyStorage)
}

// BenchmarkAccountIteratorTraversal is a bit notorious -- all layers contain the
// exact same 200 accounts. That means that we need to process 2000 items, but
// only spit out 200 values eventually.
//
// The value-fetching benchmark is easy on the binary iterator, since it never has to reach
// down at any depth for retrieving the values -- all are on the topmost layer
//
// BenchmarkAccountIteratorTraversal/binary_iterator_keys-8         	  759984	      1566 ns/op
// BenchmarkAccountIteratorTraversal/binary_iterator_values-8       	  150028	      7900 ns/op
// BenchmarkAccountIteratorTraversal/fast_iterator_keys-8           	  172809	      7006 ns/op
// BenchmarkAccountIteratorTraversal/fast_iterator_values-8         	  165112	      7658 ns/op
func BenchmarkAccountIteratorTraversal(b *testing.B) {
	// Create a custom account factory to recreate the same addresses
	makeAccounts := func(num int) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for i := 0; i < num; i++ {
			h := common.Hash{}
			binary.BigEndian.PutUint64(h[:], uint64(i+1))
			accounts[h] = randomAccount()
		}
		return accounts
	}
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

	for i := 1; i <= 100; i++ {
		parent := types.EmptyRootHash
		if i == 1 {
			parent = common.HexToHash(fmt.Sprintf("0x%02x", i))
		}
		db.Update(common.HexToHash(fmt.Sprintf("0x%02x", i+1)), parent, uint64(i), trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, makeAccounts(200), nil, nil, nil))
	}
	// We call this once before the benchmark, so the creation of
	// sorted accountlists are not included in the results.
	head := db.tree.get(common.HexToHash("0x65"))
	head.(*diffLayer).newBinaryAccountIterator()

	b.Run("binary iterator keys", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := 0
			it := head.(*diffLayer).newBinaryAccountIterator()
			for it.Next() {
				got++
			}
			if exp := 200; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
	b.Run("binary iterator values", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := 0
			it := head.(*diffLayer).newBinaryAccountIterator()
			for it.Next() {
				got++
				head.(*diffLayer).account(it.Hash(), 0)
			}
			if exp := 200; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
	b.Run("fast iterator keys", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			it, _ := db.AccountIterator(common.HexToHash("0x65"), common.Hash{})
			defer it.Release()

			got := 0
			for it.Next() {
				got++
			}
			if exp := 200; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
	b.Run("fast iterator values", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			it, _ := db.AccountIterator(common.HexToHash("0x65"), common.Hash{})
			defer it.Release()

			got := 0
			for it.Next() {
				got++
				it.Account()
			}
			if exp := 200; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
}

// BenchmarkAccountIteratorLargeBaselayer is a pretty realistic benchmark, where
// the baselayer is a lot larger than the upper layer.
//
// This is heavy on the binary iterator, which in most cases will have to
// call recursively 100 times for the majority of the values
//
// BenchmarkAccountIteratorLargeBaselayer/binary_iterator_(keys)-6         	     514	   1971999 ns/op
// BenchmarkAccountIteratorLargeBaselayer/binary_iterator_(values)-6       	      61	  18997492 ns/op
// BenchmarkAccountIteratorLargeBaselayer/fast_iterator_(keys)-6           	   10000	    114385 ns/op
// BenchmarkAccountIteratorLargeBaselayer/fast_iterator_(values)-6         	    4047	    296823 ns/op
func BenchmarkAccountIteratorLargeBaselayer(b *testing.B) {
	// Create a custom account factory to recreate the same addresses
	makeAccounts := func(num int) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for i := 0; i < num; i++ {
			h := common.Hash{}
			binary.BigEndian.PutUint64(h[:], uint64(i+1))
			accounts[h] = randomAccount()
		}
		return accounts
	}
	config := &Config{
		WriteBufferSize: 0,
	}
	db := New(rawdb.NewMemoryDatabase(), config, false)
	// db.WaitGeneration()

	db.Update(common.HexToHash("0x02"), types.EmptyRootHash, 1, trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, makeAccounts(2000), nil, nil, nil))
	for i := 2; i <= 100; i++ {
		db.Update(common.HexToHash(fmt.Sprintf("0x%02x", i+1)), common.HexToHash(fmt.Sprintf("0x%02x", i)), uint64(i), trienode.NewMergedNodeSet(), NewStateSetWithOrigin(nil, makeAccounts(20), nil, nil, nil))
	}
	// We call this once before the benchmark, so the creation of
	// sorted accountlists are not included in the results.
	head := db.tree.get(common.HexToHash("0x65"))
	head.(*diffLayer).newBinaryAccountIterator()

	b.Run("binary iterator (keys)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := 0
			it := head.(*diffLayer).newBinaryAccountIterator()
			for it.Next() {
				got++
			}
			if exp := 2000; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
	b.Run("binary iterator (values)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := 0
			it := head.(*diffLayer).newBinaryAccountIterator()
			for it.Next() {
				got++
				v := it.Hash()
				head.(*diffLayer).account(v, 0)
			}
			if exp := 2000; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
	b.Run("fast iterator (keys)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			it, _ := db.AccountIterator(common.HexToHash("0x65"), common.Hash{})
			defer it.Release()

			got := 0
			for it.Next() {
				got++
			}
			if exp := 2000; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
	b.Run("fast iterator (values)", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			it, _ := db.AccountIterator(common.HexToHash("0x65"), common.Hash{})
			defer it.Release()

			got := 0
			for it.Next() {
				it.Account()
				got++
			}
			if exp := 2000; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
}

/*
func BenchmarkBinaryAccountIteration(b *testing.B) {
	benchmarkAccountIteration(b, func(snap snapshot) AccountIterator {
		return snap.(*diffLayer).newBinaryAccountIterator()
	})
}

func BenchmarkFastAccountIteration(b *testing.B) {
	benchmarkAccountIteration(b, newFastAccountIterator)
}

func benchmarkAccountIteration(b *testing.B, iterator func(snap snapshot) AccountIterator) {
	// Create a diff stack and randomize the accounts across them
	layers := make([]map[common.Hash][]byte, 128)
	for i := 0; i < len(layers); i++ {
		layers[i] = make(map[common.Hash][]byte)
	}
	for i := 0; i < b.N; i++ {
		depth := rand.Intn(len(layers))
		layers[depth][randomHash()] = randomAccount()
	}
	stack := snapshot(emptyLayer())
	for _, layer := range layers {
		stack = stack.Update(common.Hash{}, layer, nil, nil)
	}
	// Reset the timers and report all the stats
	it := iterator(stack)

	b.ResetTimer()
	b.ReportAllocs()

	for it.Next() {
	}
}
*/
