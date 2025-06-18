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
	"crypto/rand"
	"slices"
	"sort"
	"strconv"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
)

// TestDatabaseSuite runs a suite of tests against a KeyValueStore database
// implementation.
func TestDatabaseSuite(t *testing.T, New func() ethdb.KeyValueStore) {
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
			if !slices.Equal(got, want) {
				t.Errorf("Iterator: got: %s; want: %s", got, want)
			}
		}

		{
			it := db.NewIterator([]byte("1"), nil)
			got, want := iterateKeys(it), []string{"1", "10", "11", "12"}
			if err := it.Error(); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, want) {
				t.Errorf("IteratorWith(1,nil): got: %s; want: %s", got, want)
			}
		}

		{
			it := db.NewIterator([]byte("5"), nil)
			got, want := iterateKeys(it), []string{}
			if err := it.Error(); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, want) {
				t.Errorf("IteratorWith(5,nil): got: %s; want: %s", got, want)
			}
		}

		{
			it := db.NewIterator(nil, []byte("2"))
			got, want := iterateKeys(it), []string{"2", "20", "21", "22", "3", "4", "6"}
			if err := it.Error(); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, want) {
				t.Errorf("IteratorWith(nil,2): got: %s; want: %s", got, want)
			}
		}

		{
			it := db.NewIterator(nil, []byte("5"))
			got, want := iterateKeys(it), []string{"6"}
			if err := it.Error(); err != nil {
				t.Fatal(err)
			}
			if !slices.Equal(got, want) {
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
			if got, want := iterateKeys(it), []string{"1", "2", "3", "4"}; !slices.Equal(got, want) {
				t.Errorf("got: %s; want: %s", got, want)
			}
		}

		b.Reset()

		// Mix writes and deletes in batch
		b.Put([]byte("5"), nil)
		b.Delete([]byte("1"))
		b.Put([]byte("6"), nil)

		b.Delete([]byte("3")) // delete then put
		b.Put([]byte("3"), nil)

		b.Put([]byte("7"), nil) // put then delete
		b.Delete([]byte("7"))

		if err := b.Write(); err != nil {
			t.Fatal(err)
		}

		{
			it := db.NewIterator(nil, nil)
			if got, want := iterateKeys(it), []string{"2", "3", "4", "5", "6"}; !slices.Equal(got, want) {
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
		if got := iterateKeys(it); !slices.Equal(got, want) {
			t.Errorf("got: %s; want: %s", got, want)
		}
	})

	t.Run("OperationsAfterClose", func(t *testing.T) {
		db := New()
		db.Put([]byte("key"), []byte("value"))
		db.Close()
		if _, err := db.Get([]byte("key")); err == nil {
			t.Fatalf("expected error on Get after Close")
		}
		if _, err := db.Has([]byte("key")); err == nil {
			t.Fatalf("expected error on Get after Close")
		}
		if err := db.Put([]byte("key2"), []byte("value2")); err == nil {
			t.Fatalf("expected error on Put after Close")
		}
		if err := db.Delete([]byte("key")); err == nil {
			t.Fatalf("expected error on Delete after Close")
		}

		b := db.NewBatch()
		if err := b.Put([]byte("batchkey"), []byte("batchval")); err != nil {
			t.Fatalf("expected no error on batch.Put after Close, got %v", err)
		}
		if err := b.Write(); err == nil {
			t.Fatalf("expected error on batch.Write after Close")
		}
	})

	t.Run("DeleteRange", func(t *testing.T) {
		db := New()
		defer db.Close()

		addRange := func(start, stop int) {
			for i := start; i <= stop; i++ {
				db.Put([]byte(strconv.Itoa(i)), nil)
			}
		}

		checkRange := func(start, stop int, exp bool) {
			for i := start; i <= stop; i++ {
				has, _ := db.Has([]byte(strconv.Itoa(i)))
				if has && !exp {
					t.Fatalf("unexpected key %d", i)
				}
				if !has && exp {
					t.Fatalf("missing expected key %d", i)
				}
			}
		}

		addRange(1, 9)
		db.DeleteRange([]byte("9"), []byte("1"))
		checkRange(1, 9, true)
		db.DeleteRange([]byte("5"), []byte("5"))
		checkRange(1, 9, true)
		db.DeleteRange([]byte("5"), []byte("50"))
		checkRange(1, 4, true)
		checkRange(5, 5, false)
		checkRange(6, 9, true)
		db.DeleteRange([]byte(""), []byte("a"))
		checkRange(1, 9, false)

		addRange(1, 999)
		db.DeleteRange([]byte("12345"), []byte("54321"))
		checkRange(1, 1, true)
		checkRange(2, 5, false)
		checkRange(6, 12, true)
		checkRange(13, 54, false)
		checkRange(55, 123, true)
		checkRange(124, 543, false)
		checkRange(544, 999, true)

		addRange(1, 999)
		db.DeleteRange([]byte("3"), []byte("7"))
		checkRange(1, 2, true)
		checkRange(3, 6, false)
		checkRange(7, 29, true)
		checkRange(30, 69, false)
		checkRange(70, 299, true)
		checkRange(300, 699, false)
		checkRange(700, 999, true)

		db.DeleteRange([]byte(""), []byte("a"))
		checkRange(1, 999, false)

		addRange(1, 999)
		db.DeleteRange(nil, nil)
		checkRange(1, 999, false)
	})

	t.Run("BatchDeleteRange", func(t *testing.T) {
		db := New()
		defer db.Close()

		// Helper to add keys
		addKeys := func(start, stop int) {
			for i := start; i <= stop; i++ {
				if err := db.Put([]byte(strconv.Itoa(i)), []byte("val-"+strconv.Itoa(i))); err != nil {
					t.Fatal(err)
				}
			}
		}

		// Helper to check if keys exist
		checkKeys := func(start, stop int, shouldExist bool) {
			for i := start; i <= stop; i++ {
				key := []byte(strconv.Itoa(i))
				has, err := db.Has(key)
				if err != nil {
					t.Fatal(err)
				}
				if has != shouldExist {
					if shouldExist {
						t.Fatalf("key %s should exist but doesn't", key)
					} else {
						t.Fatalf("key %s shouldn't exist but does", key)
					}
				}
			}
		}

		// Test 1: Basic range deletion in batch
		addKeys(1, 10)
		checkKeys(1, 10, true)

		batch := db.NewBatch()
		if err := batch.DeleteRange([]byte("3"), []byte("8")); err != nil {
			t.Fatal(err)
		}
		// Keys shouldn't be deleted until Write is called
		checkKeys(1, 10, true)

		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		// After Write, keys in range should be deleted
		// Range is [start, end) - inclusive of start, exclusive of end
		checkKeys(1, 2, true)  // These should still exist
		checkKeys(3, 7, false) // These should be deleted (3 to 7 inclusive)
		checkKeys(8, 10, true) // These should still exist (8 is the end boundary, exclusive)

		// Test 2: Delete range with special markers
		addKeys(3, 7)
		batch = db.NewBatch()
		if err := batch.DeleteRange(nil, nil); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		checkKeys(1, 10, false)

		// Test 3: Mix Put, Delete, and DeleteRange in a batch
		// Reset database for next test by adding back deleted keys
		addKeys(1, 10)
		checkKeys(1, 10, true)

		// Create a new batch with multiple operations
		batch = db.NewBatch()
		if err := batch.Put([]byte("5"), []byte("new-val-5")); err != nil {
			t.Fatal(err)
		}
		if err := batch.Delete([]byte("9")); err != nil {
			t.Fatal(err)
		}
		if err := batch.DeleteRange([]byte("1"), []byte("3")); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		// Check results after batch operations
		// Keys 1-2 should be deleted by DeleteRange
		checkKeys(1, 2, false)

		// Key 3 should exist (exclusive of end)
		has, err := db.Has([]byte("3"))
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			t.Fatalf("key 3 should exist after DeleteRange(1,3)")
		}

		// Key 5 should have a new value
		val, err := db.Get([]byte("5"))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(val, []byte("new-val-5")) {
			t.Fatalf("key 5 has wrong value: got %s, want %s", val, "new-val-5")
		}

		// Key 9 should be deleted
		has, err = db.Has([]byte("9"))
		if err != nil {
			t.Fatal(err)
		}
		if has {
			t.Fatalf("key 9 should be deleted")
		}

		// Test 4: Reset batch
		batch.Reset()
		// Individual deletes work better with both string and numeric comparisons
		if err := batch.Delete([]byte("8")); err != nil {
			t.Fatal(err)
		}
		if err := batch.Delete([]byte("10")); err != nil {
			t.Fatal(err)
		}
		if err := batch.Delete([]byte("11")); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}

		// Key 8 should be deleted
		has, err = db.Has([]byte("8"))
		if err != nil {
			t.Fatal(err)
		}
		if has {
			t.Fatalf("key 8 should be deleted")
		}

		// Keys 3-7 should still exist
		checkKeys(3, 7, true)

		// Key 10 should be deleted
		has, err = db.Has([]byte("10"))
		if err != nil {
			t.Fatal(err)
		}
		if has {
			t.Fatalf("key 10 should be deleted")
		}

		// Test 5: Empty range
		batch = db.NewBatch()
		if err := batch.DeleteRange([]byte("100"), []byte("100")); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		// No existing keys should be affected
		checkKeys(3, 7, true)

		// Test 6: Test entire keyspace deletion
		// First clear any existing keys
		for i := 1; i <= 100; i++ {
			db.Delete([]byte(strconv.Itoa(i)))
		}

		// Then add some fresh test keys
		addKeys(50, 60)

		// Verify keys exist before deletion
		checkKeys(50, 60, true)

		batch = db.NewBatch()
		if err := batch.DeleteRange([]byte(""), []byte("z")); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		// All keys should be deleted
		checkKeys(50, 60, false)

		// Test 7: overlapping range deletion
		addKeys(50, 60)
		batch = db.NewBatch()
		if err := batch.DeleteRange([]byte("50"), []byte("55")); err != nil {
			t.Fatal(err)
		}
		if err := batch.DeleteRange([]byte("52"), []byte("58")); err != nil {
			t.Fatal(err)
		}
		if err := batch.Write(); err != nil {
			t.Fatal(err)
		}
		checkKeys(50, 57, false)
		checkKeys(58, 60, true)
	})

	t.Run("BatchReplayWithDeleteRange", func(t *testing.T) {
		db := New()
		defer db.Close()

		// Setup some initial data
		for i := 1; i <= 10; i++ {
			if err := db.Put([]byte(strconv.Itoa(i)), []byte("val-"+strconv.Itoa(i))); err != nil {
				t.Fatal(err)
			}
		}

		// Create batch with multiple operations including DeleteRange
		batch1 := db.NewBatch()
		batch1.Put([]byte("new-key-1"), []byte("new-val-1"))
		batch1.DeleteRange([]byte("3"), []byte("7")) // Should delete keys 3-6 but not 7
		batch1.Delete([]byte("8"))
		batch1.Put([]byte("new-key-2"), []byte("new-val-2"))

		// Create a second batch to replay into
		batch2 := db.NewBatch()
		if err := batch1.Replay(batch2); err != nil {
			t.Fatal(err)
		}

		// Write the second batch
		if err := batch2.Write(); err != nil {
			t.Fatal(err)
		}

		// Verify results
		// Original keys 3-6 should be deleted (inclusive of start, exclusive of end)
		for i := 3; i <= 6; i++ {
			has, err := db.Has([]byte(strconv.Itoa(i)))
			if err != nil {
				t.Fatal(err)
			}
			if has {
				t.Fatalf("key %d should be deleted", i)
			}
		}

		// Key 7 should NOT be deleted (exclusive of end)
		has, err := db.Has([]byte("7"))
		if err != nil {
			t.Fatal(err)
		}
		if !has {
			t.Fatalf("key 7 should NOT be deleted (exclusive of end)")
		}

		// Key 8 should be deleted
		has, err = db.Has([]byte("8"))
		if err != nil {
			t.Fatal(err)
		}
		if has {
			t.Fatalf("key 8 should be deleted")
		}

		// New keys should be added
		for _, key := range []string{"new-key-1", "new-key-2"} {
			has, err := db.Has([]byte(key))
			if err != nil {
				t.Fatal(err)
			}
			if !has {
				t.Fatalf("key %s should exist", key)
			}
		}

		// Create a third batch for direct replay to database
		batch3 := db.NewBatch()
		batch3.DeleteRange([]byte("1"), []byte("3")) // Should delete keys 1-2 but not 3

		// Replay directly to the database
		if err := batch3.Replay(db); err != nil {
			t.Fatal(err)
		}

		// Verify keys 1-2 are now deleted
		for i := 1; i <= 2; i++ {
			has, err := db.Has([]byte(strconv.Itoa(i)))
			if err != nil {
				t.Fatal(err)
			}
			if has {
				t.Fatalf("key %d should be deleted after direct replay", i)
			}
		}

		// Verify key 3 is NOT deleted (since it's exclusive of end)
		has, err = db.Has([]byte("3"))
		if err != nil {
			t.Fatal(err)
		}
		if has {
			t.Fatalf("key 3 should still be deleted from previous operation")
		}
	})
}

// BenchDatabaseSuite runs a suite of benchmarks against a KeyValueStore database
// implementation.
func BenchDatabaseSuite(b *testing.B, New func() ethdb.KeyValueStore) {
	var (
		keys, vals   = makeDataset(1_000_000, 32, 32, false)
		sKeys, sVals = makeDataset(1_000_000, 32, 32, true)
	)
	// Run benchmarks sequentially
	b.Run("Write", func(b *testing.B) {
		benchWrite := func(b *testing.B, keys, vals [][]byte) {
			b.ResetTimer()
			b.ReportAllocs()

			db := New()
			defer db.Close()

			for i := 0; i < len(keys); i++ {
				db.Put(keys[i], vals[i])
			}
		}
		b.Run("WriteSorted", func(b *testing.B) {
			benchWrite(b, sKeys, sVals)
		})
		b.Run("WriteRandom", func(b *testing.B) {
			benchWrite(b, keys, vals)
		})
	})
	b.Run("Read", func(b *testing.B) {
		benchRead := func(b *testing.B, keys, vals [][]byte) {
			db := New()
			defer db.Close()

			for i := 0; i < len(keys); i++ {
				db.Put(keys[i], vals[i])
			}
			b.ResetTimer()
			b.ReportAllocs()

			for i := 0; i < len(keys); i++ {
				db.Get(keys[i])
			}
		}
		b.Run("ReadSorted", func(b *testing.B) {
			benchRead(b, sKeys, sVals)
		})
		b.Run("ReadRandom", func(b *testing.B) {
			benchRead(b, keys, vals)
		})
	})
	b.Run("Iteration", func(b *testing.B) {
		benchIteration := func(b *testing.B, keys, vals [][]byte) {
			db := New()
			defer db.Close()

			for i := 0; i < len(keys); i++ {
				db.Put(keys[i], vals[i])
			}
			b.ResetTimer()
			b.ReportAllocs()

			it := db.NewIterator(nil, nil)
			for it.Next() {
			}
			it.Release()
		}
		b.Run("IterationSorted", func(b *testing.B) {
			benchIteration(b, sKeys, sVals)
		})
		b.Run("IterationRandom", func(b *testing.B) {
			benchIteration(b, keys, vals)
		})
	})
	b.Run("BatchWrite", func(b *testing.B) {
		benchBatchWrite := func(b *testing.B, keys, vals [][]byte) {
			b.ResetTimer()
			b.ReportAllocs()

			db := New()
			defer db.Close()

			batch := db.NewBatch()
			for i := 0; i < len(keys); i++ {
				batch.Put(keys[i], vals[i])
			}
			batch.Write()
		}
		b.Run("BenchWriteSorted", func(b *testing.B) {
			benchBatchWrite(b, sKeys, sVals)
		})
		b.Run("BenchWriteRandom", func(b *testing.B) {
			benchBatchWrite(b, keys, vals)
		})
	})
	b.Run("DeleteRange", func(b *testing.B) {
		benchDeleteRange := func(b *testing.B, count int) {
			db := New()
			defer db.Close()

			for i := 0; i < count; i++ {
				db.Put([]byte(strconv.Itoa(i)), nil)
			}
			b.ResetTimer()
			b.ReportAllocs()

			db.DeleteRange([]byte("0"), []byte("999999999"))
		}
		b.Run("DeleteRange100", func(b *testing.B) {
			benchDeleteRange(b, 100)
		})
		b.Run("DeleteRange1k", func(b *testing.B) {
			benchDeleteRange(b, 1000)
		})
		b.Run("DeleteRange10k", func(b *testing.B) {
			benchDeleteRange(b, 10000)
		})
	})
	b.Run("BatchDeleteRange", func(b *testing.B) {
		benchBatchDeleteRange := func(b *testing.B, count int) {
			db := New()
			defer db.Close()

			// Prepare data
			for i := 0; i < count; i++ {
				db.Put([]byte(strconv.Itoa(i)), nil)
			}

			b.ResetTimer()
			b.ReportAllocs()

			// Create batch and delete range
			batch := db.NewBatch()
			batch.DeleteRange([]byte("0"), []byte("999999999"))
			batch.Write()
		}

		b.Run("BatchDeleteRange100", func(b *testing.B) {
			benchBatchDeleteRange(b, 100)
		})
		b.Run("BatchDeleteRange1k", func(b *testing.B) {
			benchBatchDeleteRange(b, 1000)
		})
		b.Run("BatchDeleteRange10k", func(b *testing.B) {
			benchBatchDeleteRange(b, 10000)
		})
	})

	b.Run("BatchMixedOps", func(b *testing.B) {
		benchBatchMixedOps := func(b *testing.B, count int) {
			db := New()
			defer db.Close()

			// Prepare initial data
			for i := 0; i < count; i++ {
				db.Put([]byte(strconv.Itoa(i)), []byte("val"))
			}

			b.ResetTimer()
			b.ReportAllocs()

			// Create batch with mixed operations
			batch := db.NewBatch()

			// Add some new keys
			for i := 0; i < count/10; i++ {
				batch.Put([]byte(strconv.Itoa(count+i)), []byte("new-val"))
			}

			// Delete some individual keys
			for i := 0; i < count/20; i++ {
				batch.Delete([]byte(strconv.Itoa(i * 2)))
			}

			// Delete range of keys
			rangeStart := count / 2
			rangeEnd := count * 3 / 4
			batch.DeleteRange([]byte(strconv.Itoa(rangeStart)), []byte(strconv.Itoa(rangeEnd)))

			// Write the batch
			batch.Write()
		}

		b.Run("BatchMixedOps100", func(b *testing.B) {
			benchBatchMixedOps(b, 100)
		})
		b.Run("BatchMixedOps1k", func(b *testing.B) {
			benchBatchMixedOps(b, 1000)
		})
		b.Run("BatchMixedOps10k", func(b *testing.B) {
			benchBatchMixedOps(b, 10000)
		})
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

// randBytes generates a random blob of data.
func randBytes(len int) []byte {
	buf := make([]byte, len)
	if n, err := rand.Read(buf); n != len || err != nil {
		panic(err)
	}
	return buf
}

func makeDataset(size, ksize, vsize int, order bool) ([][]byte, [][]byte) {
	var keys [][]byte
	var vals [][]byte
	for i := 0; i < size; i += 1 {
		keys = append(keys, randBytes(ksize))
		vals = append(vals, randBytes(vsize))
	}
	if order {
		slices.SortFunc(keys, bytes.Compare)
	}
	return keys, vals
}
