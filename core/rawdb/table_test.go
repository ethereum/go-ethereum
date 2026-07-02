// Copyright 2020 The go-ethereum Authors
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

package rawdb

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
)

func TestTableDatabase(t *testing.T)            { testTableDatabase(t, "prefix") }
func TestEmptyPrefixTableDatabase(t *testing.T) { testTableDatabase(t, "") }

type testReplayer struct {
	puts      [][]byte
	dels      [][]byte
	delRanges [][2][]byte
}

func (r *testReplayer) Put(key []byte, value []byte) error {
	r.puts = append(r.puts, key)
	return nil
}

func (r *testReplayer) Delete(key []byte) error {
	r.dels = append(r.dels, key)
	return nil
}

func (r *testReplayer) DeleteRange(start, end []byte) error {
	r.delRanges = append(r.delRanges, [2][]byte{start, end})
	return nil
}

func testTableDatabase(t *testing.T, prefix string) {
	db := NewTable(NewMemoryDatabase(), prefix)

	var entries = []struct {
		key   []byte
		value []byte
	}{
		{[]byte{0x01, 0x02}, []byte{0x0a, 0x0b}},
		{[]byte{0x03, 0x04}, []byte{0x0c, 0x0d}},
		{[]byte{0x05, 0x06}, []byte{0x0e, 0x0f}},

		{[]byte{0xff, 0xff, 0x01}, []byte{0x1a, 0x1b}},
		{[]byte{0xff, 0xff, 0x02}, []byte{0x1c, 0x1d}},
		{[]byte{0xff, 0xff, 0x03}, []byte{0x1e, 0x1f}},
	}

	// Test Put/Get operation
	for _, entry := range entries {
		db.Put(entry.key, entry.value)
	}
	for _, entry := range entries {
		got, err := db.Get(entry.key)
		if err != nil {
			t.Fatalf("Failed to get value: %v", err)
		}
		if !bytes.Equal(got, entry.value) {
			t.Fatalf("Value mismatch: want=%v, got=%v", entry.value, got)
		}
	}

	// Test batch operation
	db = NewTable(NewMemoryDatabase(), prefix)
	batch := db.NewBatch()
	for _, entry := range entries {
		batch.Put(entry.key, entry.value)
	}
	batch.Write()
	for _, entry := range entries {
		got, err := db.Get(entry.key)
		if err != nil {
			t.Fatalf("Failed to get value: %v", err)
		}
		if !bytes.Equal(got, entry.value) {
			t.Fatalf("Value mismatch: want=%v, got=%v", entry.value, got)
		}
	}

	// Test batch replayer
	r := &testReplayer{}
	batch.Replay(r)
	for index, entry := range entries {
		got := r.puts[index]
		if !bytes.Equal(got, entry.key) {
			t.Fatalf("Key mismatch: want=%v, got=%v", entry.key, got)
		}
	}

	check := func(iter ethdb.Iterator, expCount, index int) {
		count := 0
		for iter.Next() {
			key, value := iter.Key(), iter.Value()
			if !bytes.Equal(key, entries[index].key) {
				t.Fatalf("Key mismatch: want=%v, got=%v", entries[index].key, key)
			}
			if !bytes.Equal(value, entries[index].value) {
				t.Fatalf("Value mismatch: want=%v, got=%v", entries[index].value, value)
			}
			index += 1
			count++
		}
		if count != expCount {
			t.Fatalf("Wrong number of elems, exp %d got %d", expCount, count)
		}
		iter.Release()
	}
	// Test iterators
	check(db.NewIterator(nil, nil), 6, 0)
	// Test iterators with prefix
	check(db.NewIterator([]byte{0xff, 0xff}, nil), 3, 3)
	// Test iterators with start point
	check(db.NewIterator(nil, []byte{0xff, 0xff, 0x02}), 2, 4)
	// Test iterators with prefix and start point
	check(db.NewIterator([]byte{0xee}, nil), 0, 0)
	check(db.NewIterator(nil, []byte{0x00}), 6, 0)

	// Test batch replayer with DeleteRange
	db2 := NewTable(NewMemoryDatabase(), prefix)
	for _, entry := range entries {
		db2.Put(entry.key, entry.value)
	}
	batch2 := db2.NewBatch()
	batch2.Put([]byte{0x07, 0x08}, []byte{0x10, 0x11})
	batch2.DeleteRange([]byte{0x01, 0x02}, []byte{0x05, 0x06})
	batch2.Delete([]byte{0xff, 0xff, 0x03})

	// Replay into another batch (tests tableReplayer.DeleteRange via batch-to-batch)
	batch3 := db2.NewBatch()
	if err := batch2.Replay(batch3); err != nil {
		t.Fatalf("Failed to replay batch with DeleteRange: %v", err)
	}
	if err := batch3.Write(); err != nil {
		t.Fatalf("Failed to write replayed batch: %v", err)
	}
	// Keys in range [0x01,0x02 .. 0x05,0x06) should be deleted
	for _, key := range [][]byte{{0x01, 0x02}, {0x03, 0x04}} {
		if _, err := db2.Get(key); err == nil {
			t.Fatalf("Key %x should be deleted after replayed DeleteRange", key)
		}
	}
	// Key 0x05,0x06 should still exist (exclusive end)
	if _, err := db2.Get([]byte{0x05, 0x06}); err != nil {
		t.Fatalf("Key 0x0506 should exist (exclusive end): %v", err)
	}
	// New key should exist
	if _, err := db2.Get([]byte{0x07, 0x08}); err != nil {
		t.Fatalf("Key 0x0708 should exist after replay: %v", err)
	}
	// Deleted single key should be gone
	if _, err := db2.Get([]byte{0xff, 0xff, 0x03}); err == nil {
		t.Fatal("Key 0xffff03 should be deleted after replay")
	}

	// Replay into a testReplayer to verify prefix stripping
	r2 := &testReplayer{}
	batch2.Replay(r2)
	if len(r2.delRanges) != 1 {
		t.Fatalf("Expected 1 DeleteRange in replay, got %d", len(r2.delRanges))
	}
	if !bytes.Equal(r2.delRanges[0][0], []byte{0x01, 0x02}) {
		t.Fatalf("DeleteRange start mismatch: want=%x, got=%x", []byte{0x01, 0x02}, r2.delRanges[0][0])
	}
	if !bytes.Equal(r2.delRanges[0][1], []byte{0x05, 0x06}) {
		t.Fatalf("DeleteRange end mismatch: want=%x, got=%x", []byte{0x05, 0x06}, r2.delRanges[0][1])
	}

	// Test range deletion
	db.DeleteRange(nil, nil)
	for _, entry := range entries {
		_, err := db.Get(entry.key)
		if err == nil {
			t.Fatal("Unexpected item after deletion")
		}
	}
	// Test range deletion by batch
	batch = db.NewBatch()
	for _, entry := range entries {
		batch.Put(entry.key, entry.value)
	}
	batch.Write()
	batch.Reset()
	batch.DeleteRange(nil, nil)
	batch.Write()
	for _, entry := range entries {
		_, err := db.Get(entry.key)
		if err == nil {
			t.Fatal("Unexpected item after deletion")
		}
	}
}
