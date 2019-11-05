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
	"encoding/binary"
	"math/big"
	"math/rand"
	"testing"

	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
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
	parent := newDiffLayer(emptyLayer(), common.Hash{}, accounts, storage)
	child := newDiffLayer(parent, common.Hash{}, accounts, storage)
	child = newDiffLayer(child, common.Hash{}, accounts, storage)
	child = newDiffLayer(child, common.Hash{}, accounts, storage)
	child = newDiffLayer(child, common.Hash{}, accounts, storage)
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
	parent := newDiffLayer(emptyLayer(), common.Hash{}, flip(), storage)
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
		parent = newDiffLayer(emptyLayer(), common.Hash{}, accounts, storage)
	}
	{
		var accounts = make(map[common.Hash][]byte)
		var storage = make(map[common.Hash]map[common.Hash][]byte)
		accounts[acc] = randomAccount()
		accstorage := make(map[common.Hash][]byte)
		storage[acc] = accstorage
		storage[acc][slot] = []byte{0x01}
		child = newDiffLayer(parent, common.Hash{}, accounts, storage)
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

func emptyLayer() *diskLayer {
	return &diskLayer{
		diskdb: memorydb.New(),
		cache:  fastcache.New(500 * 1024),
	}
}

// BenchmarkSearch checks how long it takes to find a non-existing key
// BenchmarkSearch-6   	  200000	     10481 ns/op (1K per layer)
// BenchmarkSearch-6   	  200000	     10760 ns/op (10K per layer)
// BenchmarkSearch-6   	  100000	     17866 ns/op
//
// BenchmarkSearch-6   	  500000	      3723 ns/op (10k per layer, only top-level RLock()
func BenchmarkSearch(b *testing.B) {
	// First, we set up 128 diff layers, with 1K items each
	fill := func(parent snapshot) *diffLayer {
		accounts := make(map[common.Hash][]byte)
		storage := make(map[common.Hash]map[common.Hash][]byte)

		for i := 0; i < 10000; i++ {
			accounts[randomHash()] = randomAccount()
		}
		return newDiffLayer(parent, common.Hash{}, accounts, storage)
	}
	var layer snapshot
	layer = emptyLayer()
	for i := 0; i < 128; i++ {
		layer = fill(layer)
	}

	key := crypto.Keccak256Hash([]byte{0x13, 0x38})
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
// With bloom filter:
// BenchmarkSearchSlot-6   	 3467835	       351 ns/op
func BenchmarkSearchSlot(b *testing.B) {
	// First, we set up 128 diff layers, with 1K items each
	accountKey := crypto.Keccak256Hash([]byte{0x13, 0x37})
	storageKey := crypto.Keccak256Hash([]byte{0x13, 0x37})
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
		return newDiffLayer(parent, common.Hash{}, accounts, storage)
	}
	var layer snapshot
	layer = emptyLayer()
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
	fill := func(parent snapshot) *diffLayer {
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
		return newDiffLayer(parent, common.Hash{}, accounts, storage)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		b.StopTimer()
		var layer snapshot
		layer = emptyLayer()
		for i := 1; i < 128; i++ {
			layer = fill(layer)
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

// This test writes ~324M of diff layers to disk, spread over
// - 128 individual layers,
// - each with 200 accounts
// - containing 200 slots
//
// BenchmarkJournal-6   	       1	1471373923 ns/ops
// BenchmarkJournal-6   	       1	1208083335 ns/op // bufio writer
func BenchmarkJournal(b *testing.B) {
	fill := func(parent snapshot) *diffLayer {
		accounts := make(map[common.Hash][]byte)
		storage := make(map[common.Hash]map[common.Hash][]byte)

		for i := 0; i < 200; i++ {
			accountKey := randomHash()
			accounts[accountKey] = randomAccount()

			accStorage := make(map[common.Hash][]byte)
			for i := 0; i < 200; i++ {
				value := make([]byte, 32)
				rand.Read(value)
				accStorage[randomHash()] = value

			}
			storage[accountKey] = accStorage
		}
		return newDiffLayer(parent, common.Hash{}, accounts, storage)
	}
	layer := snapshot(new(diskLayer))
	for i := 1; i < 128; i++ {
		layer = fill(layer)
	}
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		layer.Journal(new(bytes.Buffer))
	}
}

// TestIteratorBasics tests some simple single-layer iteration
func TestIteratorBasics(t *testing.T) {
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
	parent := newDiffLayer(emptyLayer{}, common.Hash{}, accounts, storage)
	it := parent.newIterator()
	verifyIterator(t, 100, it)
}

type testIterator struct {
	values []byte
}

func newTestIterator(values ...byte) *testIterator {
	return &testIterator{values}
}
func (ti *testIterator) Next() bool {
	ti.values = ti.values[1:]
	if len(ti.values) == 0 {
		return false
	}
	return true
}

func (ti *testIterator) Key() common.Hash {
	return common.BytesToHash([]byte{ti.values[0]})
}

func (ti *testIterator) Seek(common.Hash) {
	panic("implement me")
}

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
		var iterators []Iterator
		for _, data := range tc.lists {
			iterators = append(iterators, newTestIterator(data...))

		}
		fi := &fastIterator{
			iterators: iterators,
			initiated: false,
		}
		count := 0
		for fi.Next() {
			if got, exp := fi.Key()[31], tc.expKeys[count]; exp != got {
				t.Errorf("tc %d, [%d]: got %d exp %d", i, count, got, exp)
			}
			count++
		}
	}
}

func verifyIterator(t *testing.T, expCount int, it Iterator) {
	var (
		i    = 0
		last = common.Hash{}
	)
	for it.Next() {
		v := it.Key()
		if bytes.Compare(last[:], v[:]) >= 0 {
			t.Errorf("Wrong order:\n%x \n>=\n%x", last, v)
		}
		i++
	}
	if i != expCount {
		t.Errorf("iterator len wrong, expected %d, got %d", expCount, i)
	}
}

// TestIteratorTraversal tests some simple multi-layer iteration
func TestIteratorTraversal(t *testing.T) {
	var (
		storage = make(map[common.Hash]map[common.Hash][]byte)
	)

	mkAccounts := func(args ...string) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for _, h := range args {
			accounts[common.HexToHash(h)] = randomAccount()
		}
		return accounts
	}
	// entries in multiple layers should only become output once
	parent := newDiffLayer(emptyLayer{}, common.Hash{},
		mkAccounts("0xaa", "0xee", "0xff", "0xf0"), storage)

	child := parent.Update(common.Hash{},
		mkAccounts("0xbb", "0xdd", "0xf0"), storage)

	child = child.Update(common.Hash{},
		mkAccounts("0xcc", "0xf0", "0xff"), storage)

	// single layer iterator
	verifyIterator(t, 3, child.newIterator())
	// multi-layered binary iterator
	verifyIterator(t, 7, child.newBinaryIterator())
	// multi-layered fast iterator
	verifyIterator(t, 7, child.newFastIterator())
}

func TestIteratorLargeTraversal(t *testing.T) {
	// This testcase is a bit notorious -- all layers contain the exact
	// same 200 accounts.
	var storage = make(map[common.Hash]map[common.Hash][]byte)
	mkAccounts := func(num int) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for i := 0; i < num; i++ {
			h := common.Hash{}
			binary.BigEndian.PutUint64(h[:], uint64(i+1))
			accounts[h] = randomAccount()
		}
		return accounts
	}
	parent := newDiffLayer(emptyLayer{}, common.Hash{},
		mkAccounts(200), storage)
	child := parent.Update(common.Hash{},
		mkAccounts(200), storage)
	for i := 2; i < 100; i++ {
		child = child.Update(common.Hash{},
			mkAccounts(200), storage)
	}
	// single layer iterator
	verifyIterator(t, 200, child.newIterator())
	// multi-layered binary iterator
	verifyIterator(t, 200, child.newBinaryIterator())
	// multi-layered fast iterator
	verifyIterator(t, 200, child.newFastIterator())
}

// BenchmarkIteratorTraversal is a bit a bit notorious -- all layers contain the exact
// same 200 accounts. That means that we need to process 2000 items, but only
// spit out 200 values eventually.
//
//BenchmarkIteratorTraversal/binary_iterator-6         	    2008	    573290 ns/op	    9520 B/op	     199 allocs/op
//BenchmarkIteratorTraversal/fast_iterator-6           	    1946	    575596 ns/op	   20146 B/op	     134 allocs/op
func BenchmarkIteratorTraversal(b *testing.B) {

	var storage = make(map[common.Hash]map[common.Hash][]byte)

	mkAccounts := func(num int) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for i := 0; i < num; i++ {
			h := common.Hash{}
			binary.BigEndian.PutUint64(h[:], uint64(i+1))
			accounts[h] = randomAccount()
		}
		return accounts
	}
	parent := newDiffLayer(emptyLayer{}, common.Hash{},
		mkAccounts(200), storage)

	child := parent.Update(common.Hash{},
		mkAccounts(200), storage)

	for i := 2; i < 100; i++ {
		child = child.Update(common.Hash{},
			mkAccounts(200), storage)

	}
	// We call this once before the benchmark, so the creation of
	// sorted accountlists are not included in the results.
	child.newBinaryIterator()
	b.Run("binary iterator", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := 0
			it := child.newBinaryIterator()
			for it.Next() {
				got++
			}
			if exp := 200; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
	b.Run("fast iterator", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := 0
			it := child.newFastIterator()
			for it.Next() {
				got++
			}
			if exp := 200; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
}

// BenchmarkIteratorLargeBaselayer is a pretty realistic benchmark, where
// the baselayer is a lot larger than the upper layer.
//
// This is heavy on the binary iterator, which in most cases will have to
// call recursively 100 times for the majority of the values
//
// BenchmarkIteratorLargeBaselayer/binary_iterator-6    	     585	   2067377 ns/op	    9520 B/op	     199 allocs/op
// BenchmarkIteratorLargeBaselayer/fast_iterator-6      	   13198	     91043 ns/op	    8601 B/op	     118 allocs/op
func BenchmarkIteratorLargeBaselayer(b *testing.B) {
	var storage = make(map[common.Hash]map[common.Hash][]byte)

	mkAccounts := func(num int) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for i := 0; i < num; i++ {
			h := common.Hash{}
			binary.BigEndian.PutUint64(h[:], uint64(i+1))
			accounts[h] = randomAccount()
		}
		return accounts
	}

	parent := newDiffLayer(emptyLayer{}, common.Hash{},
		mkAccounts(2000), storage)

	child := parent.Update(common.Hash{},
		mkAccounts(20), storage)

	for i := 2; i < 100; i++ {
		child = child.Update(common.Hash{},
			mkAccounts(20), storage)

	}
	// We call this once before the benchmark, so the creation of
	// sorted accountlists are not included in the results.
	child.newBinaryIterator()
	b.Run("binary iterator", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := 0
			it := child.newBinaryIterator()
			for it.Next() {
				got++
			}
			if exp := 2000; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
	b.Run("fast iterator", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := 0
			it := child.newFastIterator()
			for it.Next() {
				got++
			}
			if exp := 2000; got != exp {
				b.Errorf("iterator len wrong, expected %d, got %d", exp, got)
			}
		}
	})
}

// TestIteratorFlatting tests what happens when we
// - have a live iterator on child C (parent C1 -> C2 .. CN)
// - flattens C2 all the way into CN
// - continues iterating
// Right now, this "works" simply because the keys do not change -- the
// iterator is not aware that a layer has become stale. This naive
// solution probably won't work in the long run, however
func TestIteratorFlattning(t *testing.T) {
	var (
		storage = make(map[common.Hash]map[common.Hash][]byte)
	)
	mkAccounts := func(args ...string) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for _, h := range args {
			accounts[common.HexToHash(h)] = randomAccount()
		}
		return accounts
	}
	// entries in multiple layers should only become output once
	parent := newDiffLayer(emptyLayer{}, common.Hash{},
		mkAccounts("0xaa", "0xee", "0xff", "0xf0"), storage)

	child := parent.Update(common.Hash{},
		mkAccounts("0xbb", "0xdd", "0xf0"), storage)

	child = child.Update(common.Hash{},
		mkAccounts("0xcc", "0xf0", "0xff"), storage)

	it := child.newFastIterator()
	child.parent.(*diffLayer).flatten()
	// The parent should now be stale
	verifyIterator(t, 7, it)
}

func TestIteratorSeek(t *testing.T) {
	storage := make(map[common.Hash]map[common.Hash][]byte)
	mkAccounts := func(args ...string) map[common.Hash][]byte {
		accounts := make(map[common.Hash][]byte)
		for _, h := range args {
			accounts[common.HexToHash(h)] = randomAccount()
		}
		return accounts
	}
	parent := newDiffLayer(emptyLayer{}, common.Hash{},
		mkAccounts("0xaa", "0xee", "0xff", "0xf0"), storage)
	it := parent.newIterator()
	// expected: ee, f0, ff
	it.Seek(common.HexToHash("0xdd"))
	verifyIterator(t, 3, it)

	it = parent.newIterator().(*dlIterator)
	// expected: ee, f0, ff
	it.Seek(common.HexToHash("0xaa"))
	verifyIterator(t, 3, it)

	it = parent.newIterator().(*dlIterator)
	// expected: nothing
	it.Seek(common.HexToHash("0xff"))
	verifyIterator(t, 0, it)

	child := parent.Update(common.Hash{},
		mkAccounts("0xbb", "0xdd", "0xf0"), storage)

	child = child.Update(common.Hash{},
		mkAccounts("0xcc", "0xf0", "0xff"), storage)

	it = child.newFastIterator()
	// expected: cc, dd, ee, f0, ff
	it.Seek(common.HexToHash("0xbb"))
	verifyIterator(t, 5, it)

	it = child.newFastIterator()
	it.Seek(common.HexToHash("0xef"))
	// exp: f0, ff
	verifyIterator(t, 2, it)

	it = child.newFastIterator()
	it.Seek(common.HexToHash("0xf0"))
	verifyIterator(t, 1, it)

	it.Seek(common.HexToHash("0xff"))
	verifyIterator(t, 0, it)

}
