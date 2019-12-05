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
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

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
	parent := newDiffLayer(emptyLayer(), common.Hash{}, accounts, storage)
	it := parent.newAccountIterator()
	verifyIterator(t, 100, it)
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
	if len(ti.values) == 0 {
		return false
	}
	return true
}

func (ti *testIterator) Error() error {
	panic("implement me")
}

func (ti *testIterator) Key() common.Hash {
	return common.BytesToHash([]byte{ti.values[0]})
}

func (ti *testIterator) Value() []byte {
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
		var iterators []AccountIterator
		for _, data := range tc.lists {
			iterators = append(iterators, newTestIterator(data...))

		}
		fi := &fastAccountIterator{
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

func verifyIterator(t *testing.T, expCount int, it AccountIterator) {
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
	parent := newDiffLayer(emptyLayer(), common.Hash{},
		mkAccounts("0xaa", "0xee", "0xff", "0xf0"), storage)

	child := parent.Update(common.Hash{},
		mkAccounts("0xbb", "0xdd", "0xf0"), storage)

	child = child.Update(common.Hash{},
		mkAccounts("0xcc", "0xf0", "0xff"), storage)

	// single layer iterator
	verifyIterator(t, 3, child.newAccountIterator())
	// multi-layered binary iterator
	verifyIterator(t, 7, child.newBinaryAccountIterator())
	// multi-layered fast iterator
	verifyIterator(t, 7, child.newFastAccountIterator())
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
	parent := newDiffLayer(emptyLayer(), common.Hash{},
		mkAccounts(200), storage)
	child := parent.Update(common.Hash{},
		mkAccounts(200), storage)
	for i := 2; i < 100; i++ {
		child = child.Update(common.Hash{},
			mkAccounts(200), storage)
	}
	// single layer iterator
	verifyIterator(t, 200, child.newAccountIterator())
	// multi-layered binary iterator
	verifyIterator(t, 200, child.newBinaryAccountIterator())
	// multi-layered fast iterator
	verifyIterator(t, 200, child.newFastAccountIterator())
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
	parent := newDiffLayer(emptyLayer(), common.Hash{},
		mkAccounts(200), storage)

	child := parent.Update(common.Hash{},
		mkAccounts(200), storage)

	for i := 2; i < 100; i++ {
		child = child.Update(common.Hash{},
			mkAccounts(200), storage)

	}
	// We call this once before the benchmark, so the creation of
	// sorted accountlists are not included in the results.
	child.newBinaryAccountIterator()
	b.Run("binary iterator", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := 0
			it := child.newBinaryAccountIterator()
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
			it := child.newFastAccountIterator()
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

	parent := newDiffLayer(emptyLayer(), common.Hash{},
		mkAccounts(2000), storage)

	child := parent.Update(common.Hash{},
		mkAccounts(20), storage)

	for i := 2; i < 100; i++ {
		child = child.Update(common.Hash{},
			mkAccounts(20), storage)

	}
	// We call this once before the benchmark, so the creation of
	// sorted accountlists are not included in the results.
	child.newBinaryAccountIterator()
	b.Run("binary iterator", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			got := 0
			it := child.newBinaryAccountIterator()
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
			it := child.newFastAccountIterator()
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
	parent := newDiffLayer(emptyLayer(), common.Hash{},
		mkAccounts("0xaa", "0xee", "0xff", "0xf0"), storage)

	child := parent.Update(common.Hash{},
		mkAccounts("0xbb", "0xdd", "0xf0"), storage)

	child = child.Update(common.Hash{},
		mkAccounts("0xcc", "0xf0", "0xff"), storage)

	it := child.newFastAccountIterator()
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
	parent := newDiffLayer(emptyLayer(), common.Hash{},
		mkAccounts("0xaa", "0xee", "0xff", "0xf0"), storage)
	it := AccountIterator(parent.newAccountIterator())
	// expected: ee, f0, ff
	it.Seek(common.HexToHash("0xdd"))
	verifyIterator(t, 3, it)

	it = parent.newAccountIterator()
	// expected: ee, f0, ff
	it.Seek(common.HexToHash("0xaa"))
	verifyIterator(t, 3, it)

	it = parent.newAccountIterator()
	// expected: nothing
	it.Seek(common.HexToHash("0xff"))
	verifyIterator(t, 0, it)

	child := parent.Update(common.Hash{},
		mkAccounts("0xbb", "0xdd", "0xf0"), storage)

	child = child.Update(common.Hash{},
		mkAccounts("0xcc", "0xf0", "0xff"), storage)

	it = child.newFastAccountIterator()
	// expected: cc, dd, ee, f0, ff
	it.Seek(common.HexToHash("0xbb"))
	verifyIterator(t, 5, it)

	it = child.newFastAccountIterator()
	it.Seek(common.HexToHash("0xef"))
	// exp: f0, ff
	verifyIterator(t, 2, it)

	it = child.newFastAccountIterator()
	it.Seek(common.HexToHash("0xf0"))
	verifyIterator(t, 1, it)

	it.Seek(common.HexToHash("0xff"))
	verifyIterator(t, 0, it)
}
