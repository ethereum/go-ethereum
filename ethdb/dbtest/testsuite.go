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

package dbtest

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"reflect"
	"sort"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

// TestKeyValueStoreSuite runs a suite of tests against a KeyValueStore database
// implementation.
func TestKeyValueStoreSuite(t *testing.T, New func() ethdb.KeyValueStore) {
	t.Run("Iterator", func(t *testing.T) {
		tests := []struct {
			content map[string]string
			prefix  string
			start   string
			order   []string
		}{
			// Empty databases should be iterable
			{map[string]string{}, "", "", nil},
			{map[string]string{}, "non-existent-prefix", "", nil},

			// Single-item databases should be iterable
			{map[string]string{"key": "val"}, "", "", []string{"key"}},
			{map[string]string{"key": "val"}, "k", "", []string{"key"}},
			{map[string]string{"key": "val"}, "l", "", nil},

			// Multi-item databases should be fully iterable
			{
				map[string]string{"k1": "v1", "k5": "v5", "k2": "v2", "k4": "v4", "k3": "v3"},
				"", "",
				[]string{"k1", "k2", "k3", "k4", "k5"},
			},
			{
				map[string]string{"k1": "v1", "k5": "v5", "k2": "v2", "k4": "v4", "k3": "v3"},
				"k", "",
				[]string{"k1", "k2", "k3", "k4", "k5"},
			},
			{
				map[string]string{"k1": "v1", "k5": "v5", "k2": "v2", "k4": "v4", "k3": "v3"},
				"l", "",
				nil,
			},
			// Multi-item databases should be prefix-iterable
			{
				map[string]string{
					"ka1": "va1", "ka5": "va5", "ka2": "va2", "ka4": "va4", "ka3": "va3",
					"kb1": "vb1", "kb5": "vb5", "kb2": "vb2", "kb4": "vb4", "kb3": "vb3",
				},
				"ka", "",
				[]string{"ka1", "ka2", "ka3", "ka4", "ka5"},
			},
			{
				map[string]string{
					"ka1": "va1", "ka5": "va5", "ka2": "va2", "ka4": "va4", "ka3": "va3",
					"kb1": "vb1", "kb5": "vb5", "kb2": "vb2", "kb4": "vb4", "kb3": "vb3",
				},
				"kc", "",
				nil,
			},
			// Multi-item databases should be prefix-iterable with start position
			{
				map[string]string{
					"ka1": "va1", "ka5": "va5", "ka2": "va2", "ka4": "va4", "ka3": "va3",
					"kb1": "vb1", "kb5": "vb5", "kb2": "vb2", "kb4": "vb4", "kb3": "vb3",
				},
				"ka", "3",
				[]string{"ka3", "ka4", "ka5"},
			},
			{
				map[string]string{
					"ka1": "va1", "ka5": "va5", "ka2": "va2", "ka4": "va4", "ka3": "va3",
					"kb1": "vb1", "kb5": "vb5", "kb2": "vb2", "kb4": "vb4", "kb3": "vb3",
				},
				"ka", "8",
				nil,
			},
		}
		for i, tt := range tests {
			// Create the key-value data store
			db := New()
			for key, val := range tt.content {
				if err := db.Put([]byte(key), []byte(val)); err != nil {
					t.Fatalf("test %d: failed to insert item %s:%s into database: %v", i, key, val, err)
				}
			}
			// Iterate over the database with the given configs and verify the results
			it, idx := db.NewIterator([]byte(tt.prefix), []byte(tt.start)), 0
			for it.Next() {
				if len(tt.order) <= idx {
					t.Errorf("test %d: prefix=%q more items than expected: checking idx=%d (key %q), expecting len=%d", i, tt.prefix, idx, it.Key(), len(tt.order))
					break
				}
				if !bytes.Equal(it.Key(), []byte(tt.order[idx])) {
					t.Errorf("test %d: item %d: key mismatch: have %s, want %s", i, idx, string(it.Key()), tt.order[idx])
				}
				if !bytes.Equal(it.Value(), []byte(tt.content[tt.order[idx]])) {
					t.Errorf("test %d: item %d: value mismatch: have %s, want %s", i, idx, string(it.Value()), tt.content[tt.order[idx]])
				}
				idx++
			}
			if err := it.Error(); err != nil {
				t.Errorf("test %d: iteration failed: %v", i, err)
			}
			if idx != len(tt.order) {
				t.Errorf("test %d: iteration terminated prematurely: have %d, want %d", i, idx, len(tt.order))
			}
			db.Close()
		}
	})

	t.Run("IteratorWith", func(t *testing.T) {
		db := New()
		defer db.Close()

		keys := []string{"1", "2", "3", "4", "6", "10", "11", "12", "20", "21", "22"}
		sort.Strings(keys) // 1, 10, 11, etc

		for _, k := range keys {
			if err := db.Put([]byte(k), nil); err != nil {
				t.Fatal(err)
			}
		}

		{
			it := db.NewIterator(nil, nil)
			got, want := iterateKeys(it), keys
			if err := it.Error(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("Iterator: got: %s; want: %s", got, want)
			}
		}

		{
			it := db.NewIterator([]byte("1"), nil)
			got, want := iterateKeys(it), []string{"1", "10", "11", "12"}
			if err := it.Error(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("IteratorWith(1,nil): got: %s; want: %s", got, want)
			}
		}

		{
			it := db.NewIterator([]byte("5"), nil)
			got, want := iterateKeys(it), []string{}
			if err := it.Error(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("IteratorWith(5,nil): got: %s; want: %s", got, want)
			}
		}

		{
			it := db.NewIterator(nil, []byte("2"))
			got, want := iterateKeys(it), []string{"2", "20", "21", "22", "3", "4", "6"}
			if err := it.Error(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("IteratorWith(nil,2): got: %s; want: %s", got, want)
			}
		}

		{
			it := db.NewIterator(nil, []byte("5"))
			got, want := iterateKeys(it), []string{"6"}
			if err := it.Error(); err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(got, want) {
				t.Errorf("IteratorWith(nil,5): got: %s; want: %s", got, want)
			}
		}
	})

	t.Run("KeyValueOperations", func(t *testing.T) {
		db := New()
		defer db.Close()

		key := []byte("foo")

		if got, err := db.Has(key); err != nil {
			t.Error(err)
		} else if got {
			t.Errorf("wrong value: %t", got)
		}

		value := []byte("hello world")
		if err := db.Put(key, value); err != nil {
			t.Error(err)
		}

		if got, err := db.Has(key); err != nil {
			t.Error(err)
		} else if !got {
			t.Errorf("wrong value: %t", got)
		}

		if got, err := db.Get(key); err != nil {
			t.Error(err)
		} else if !bytes.Equal(got, value) {
			t.Errorf("wrong value: %q", got)
		}

		if err := db.Delete(key); err != nil {
			t.Error(err)
		}

		if got, err := db.Has(key); err != nil {
			t.Error(err)
		} else if got {
			t.Errorf("wrong value: %t", got)
		}
	})

	t.Run("Batch", func(t *testing.T) {
		db := New()
		defer db.Close()

		b := db.NewBatch()
		for _, k := range []string{"1", "2", "3", "4"} {
			if err := b.Put([]byte(k), nil); err != nil {
				t.Fatal(err)
			}
		}

		if has, err := db.Has([]byte("1")); err != nil {
			t.Fatal(err)
		} else if has {
			t.Error("db contains element before batch write")
		}

		if err := b.Write(); err != nil {
			t.Fatal(err)
		}

		{
			it := db.NewIterator(nil, nil)
			if got, want := iterateKeys(it), []string{"1", "2", "3", "4"}; !reflect.DeepEqual(got, want) {
				t.Errorf("got: %s; want: %s", got, want)
			}
		}

		b.Reset()

		// Mix writes and deletes in batch
		b.Put([]byte("5"), nil)
		b.Delete([]byte("1"))
		b.Put([]byte("6"), nil)
		b.Delete([]byte("3"))
		b.Put([]byte("3"), nil)

		if err := b.Write(); err != nil {
			t.Fatal(err)
		}

		{
			it := db.NewIterator(nil, nil)
			if got, want := iterateKeys(it), []string{"2", "3", "4", "5", "6"}; !reflect.DeepEqual(got, want) {
				t.Errorf("got: %s; want: %s", got, want)
			}
		}
	})

	t.Run("BatchReplay", func(t *testing.T) {
		db := New()
		defer db.Close()

		want := []string{"1", "2", "3", "4"}
		b := db.NewBatch()
		for _, k := range want {
			if err := b.Put([]byte(k), nil); err != nil {
				t.Fatal(err)
			}
		}

		b2 := db.NewBatch()
		if err := b.Replay(b2); err != nil {
			t.Fatal(err)
		}

		if err := b2.Replay(db); err != nil {
			t.Fatal(err)
		}

		it := db.NewIterator(nil, nil)
		if got := iterateKeys(it); !reflect.DeepEqual(got, want) {
			t.Errorf("got: %s; want: %s", got, want)
		}
	})

	t.Run("Snapshot", func(t *testing.T) {
		db := New()
		defer db.Close()

		initial := map[string]string{
			"k1": "v1", "k2": "v2", "k3": "", "k4": "",
		}
		for k, v := range initial {
			db.Put([]byte(k), []byte(v))
		}
		snapshot, err := db.NewSnapshot()
		if err != nil {
			t.Fatal(err)
		}
		for k, v := range initial {
			got, err := snapshot.Get([]byte(k))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, []byte(v)) {
				t.Fatalf("Unexpected value want: %v, got %v", v, got)
			}
		}

		// Flush more modifications into the database, ensure the snapshot
		// isn't affected.
		var (
			update = map[string]string{"k1": "v1-b", "k3": "v3-b"}
			insert = map[string]string{"k5": "v5-b"}
			delete = map[string]string{"k2": ""}
		)
		for k, v := range update {
			db.Put([]byte(k), []byte(v))
		}
		for k, v := range insert {
			db.Put([]byte(k), []byte(v))
		}
		for k := range delete {
			db.Delete([]byte(k))
		}
		for k, v := range initial {
			got, err := snapshot.Get([]byte(k))
			if err != nil {
				t.Fatal(err)
			}
			if !bytes.Equal(got, []byte(v)) {
				t.Fatalf("Unexpected value want: %v, got %v", v, got)
			}
		}
		for k := range insert {
			got, err := snapshot.Get([]byte(k))
			if err == nil || len(got) != 0 {
				t.Fatal("Unexpected value")
			}
		}
		for k := range delete {
			got, err := snapshot.Get([]byte(k))
			if err != nil || len(got) == 0 {
				t.Fatal("Unexpected deletion")
			}
		}
	})
}

func iterateKeys(it ethdb.Iterator) []string {
	keys := []string{}
	for it.Next() {
		keys = append(keys, string(it.Key()))
	}
	sort.Strings(keys)
	it.Release()
	return keys
}

type AncientCreator func(tables map[string]bool) (store ethdb.AncientStore, closeAndCleanup func())

func TestAncientStoreSuite(t *testing.T, New AncientCreator) {
	const testKind = "test"
	var tableDef = map[string]bool{testKind: true}

	t.Run("Modify", func(t *testing.T) {
		t.Parallel()

		// Create test data.
		var valuesRaw [][]byte
		var valuesRLP []*big.Int
		for x := 0; x < 100; x++ {
			v := getChunk(256, x)
			valuesRaw = append(valuesRaw, v)
			iv := big.NewInt(int64(x))
			iv = iv.Exp(iv, iv, nil)
			valuesRLP = append(valuesRLP, iv)
		}

		tables := map[string]bool{"raw": true, "rlp": false}

		store, closeAndCleanup := New(tables)
		defer closeAndCleanup()

		// Commit test data.
		_, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
			for i := range valuesRaw {
				if err := op.AppendRaw("raw", uint64(i), valuesRaw[i]); err != nil {
					return err
				}
				if err := op.Append("rlp", uint64(i), valuesRLP[i]); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal("ModifyAncients failed:", err)
		}

		// Read back test data.
		checkAncientCount(t, store, "raw", uint64(len(valuesRaw)))
		checkAncientCount(t, store, "rlp", uint64(len(valuesRLP)))
		for i := range valuesRaw {
			v, _ := store.Ancient("raw", uint64(i))
			if !bytes.Equal(v, valuesRaw[i]) {
				t.Fatalf("wrong raw value at %d: %x", i, v)
			}
			ivEnc, _ := store.Ancient("rlp", uint64(i))
			want, _ := rlp.EncodeToBytes(valuesRLP[i])
			if !bytes.Equal(ivEnc, want) {
				t.Fatalf("wrong RLP value at %d: %x", i, ivEnc)
			}
		}
	})

	// This checks that ModifyAncients rolls back updates
	// when the function passed to it returns an error.
	t.Run("ModifyRollback", func(t *testing.T) {
		t.Parallel()

		store, closeAndCleanup := New(tableDef)

		theError := errors.New("oops")
		_, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
			require.NoError(t, op.AppendRaw(testKind, 0, make([]byte, 2048)))
			require.NoError(t, op.AppendRaw(testKind, 1, make([]byte, 2048)))
			require.NoError(t, op.AppendRaw(testKind, 2, make([]byte, 2048)))
			return theError
		})
		if err != theError {
			t.Errorf("ModifyAncients returned wrong error %q", err)
		}
		checkAncientCount(t, store, testKind, 0)
		closeAndCleanup()

		// Reopen and check that the rolled-back data doesn't reappear.
		store, closeAndCleanup = New(tableDef)

		checkAncientCount(t, store, testKind, 0)
		closeAndCleanup()
	})

	t.Run("ConcurrentModifyRetrieve", func(t *testing.T) {
		t.Parallel()

		store, closeAndCleanup := New(tableDef)
		defer closeAndCleanup()

		var (
			numReaders     = 5
			writeBatchSize = uint64(50)
			written        = make(chan uint64, numReaders*6)
			wg             sync.WaitGroup
		)
		wg.Add(numReaders + 1)

		// Launch the writer. It appends 10000 items in batches.
		go func() {
			defer wg.Done()
			defer close(written)
			for item := uint64(0); item < 10000; item += writeBatchSize {
				_, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
					for i := uint64(0); i < writeBatchSize; i++ {
						item := item + i
						value := getChunk(32, int(item))
						if err := op.AppendRaw(testKind, item, value); err != nil {
							return err
						}
					}
					return nil
				})
				if err != nil {
					panic(err)
				}
				for i := 0; i < numReaders; i++ {
					written <- item + writeBatchSize
				}
			}
		}()

		// Launch the readers. They read random items from the AncientStore up to the
		// current frozen item count.
		for i := 0; i < numReaders; i++ {
			go func() {
				defer wg.Done()
				for frozen := range written {
					for rc := 0; rc < 80; rc++ {
						num := uint64(rand.Intn(int(frozen)))
						value, err := store.Ancient(testKind, num)
						if err != nil {
							panic(fmt.Errorf("error reading %d (frozen %d): %v", num, frozen, err))
						}
						if !bytes.Equal(value, getChunk(32, int(num))) {
							panic(fmt.Errorf("wrong value at %d", num))
						}
					}
				}
			}()
		}

		wg.Wait()
	})

	t.Run("ConcurrentModifyTruncate", func(t *testing.T) {
		store, closeAndCleanup := New(tableDef)
		defer closeAndCleanup()

		var item = make([]byte, 256)

		for i := 0; i < 1000; i++ {
			// First reset and write 100 items.
			if err := store.TruncateHead(0); err != nil {
				t.Fatal("truncate failed:", err)
			}
			_, err := store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
				for i := uint64(0); i < 100; i++ {
					if err := op.AppendRaw("test", i, item); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				t.Fatal("modify failed:", err)
			}
			checkAncientCount(t, store, "test", 100)

			// Now append 100 more items and truncate concurrently.
			var (
				wg          sync.WaitGroup
				truncateErr error
				modifyErr   error
			)
			wg.Add(3)
			go func() {
				_, modifyErr = store.ModifyAncients(func(op ethdb.AncientWriteOp) error {
					for i := uint64(100); i < 200; i++ {
						if err := op.AppendRaw("test", i, item); err != nil {
							return err
						}
					}
					return nil
				})
				wg.Done()
			}()
			go func() {
				truncateErr = store.TruncateHead(10)
				wg.Done()
			}()
			go func() {
				store.AncientSize("test")
				wg.Done()
			}()
			wg.Wait()

			// Now check the outcome. If the truncate operation went through first, the append
			// fails, otherwise it succeeds. In either case, the freezer should be positioned
			// at 10 after both operations are done.
			if truncateErr != nil {
				t.Fatal("concurrent truncate failed:", err)
			}
			if !(errors.Is(modifyErr, nil) || errors.Is(modifyErr, ethdb.ErrAncientOutOrderInsertion)) {
				t.Fatal("wrong error from concurrent modify:", modifyErr)
			}
			checkAncientCount(t, store, "test", 10)
		}
	})
}

// checkAncientCount verifies that the AncientStore contains n items.
func checkAncientCount(t *testing.T, store ethdb.AncientStore, kind string, n uint64) {
	t.Helper()

	if frozen, _ := store.Ancients(); frozen != n {
		t.Fatalf("Ancients() returned %d, want %d", frozen, n)
	}

	// Check at index n-1.
	if n > 0 {
		index := n - 1
		if ok, _ := store.HasAncient(kind, index); !ok {
			t.Errorf("HasAncient(%q, %d) returned false unexpectedly", kind, index)
		}
		if _, err := store.Ancient(kind, index); err != nil {
			t.Errorf("Ancient(%q, %d) returned unexpected error %q", kind, index, err)
		}
	}

	// Check at index n.
	index := n
	if ok, _ := store.HasAncient(kind, index); ok {
		t.Errorf("HasAncient(%q, %d) returned true unexpectedly", kind, index)
	}
	if _, err := store.Ancient(kind, index); err == nil {
		t.Errorf("Ancient(%q, %d) didn't return expected error", kind, index)
	} else if err != ethdb.ErrAncientOutOfBounds {
		t.Errorf("Ancient(%q, %d) returned unexpected error %q", kind, index, err)
	}
}

// Gets a chunk of data, filled with 'b'
func getChunk(size int, b int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(b)
	}
	return data
}
