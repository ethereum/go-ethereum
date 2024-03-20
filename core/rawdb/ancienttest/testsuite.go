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

package ancienttest

import (
	"bytes"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
)

// TestAncientSuite runs a suite of tests against an ancient database
// implementation.
func TestAncientSuite(t *testing.T, newFn func(kinds []string) ethdb.AncientStore) {
	// Test basic read methods
	t.Run("BasicRead", func(t *testing.T) { basicRead(t, newFn) })

	// Test batch read method
	t.Run("BatchRead", func(t *testing.T) { batchRead(t, newFn) })

	// Test basic write methods
	t.Run("BasicWrite", func(t *testing.T) { basicWrite(t, newFn) })

	// Test if data mutation is allowed after db write
	t.Run("nonMutable", func(t *testing.T) { nonMutable(t, newFn) })
}

func basicRead(t *testing.T, newFn func(kinds []string) ethdb.AncientStore) {
	var (
		db   = newFn([]string{"a"})
		data = makeDataset(100, 32)
	)
	defer db.Close()

	db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := 0; i < len(data); i++ {
			op.AppendRaw("a", uint64(i), data[i])
		}
		return nil
	})
	db.TruncateTail(10)
	db.TruncateHead(90)

	// Test basic tail and head retrievals
	tail, err := db.Tail()
	if err != nil || tail != 10 {
		t.Fatal("Failed to retrieve tail")
	}
	ancient, err := db.Ancients()
	if err != nil || ancient != 90 {
		t.Fatal("Failed to retrieve ancient")
	}

	// Test the deleted items shouldn't be reachable
	var cases = []struct {
		start int
		limit int
	}{
		{0, 10},
		{90, 100},
	}
	for _, c := range cases {
		for i := c.start; i < c.limit; i++ {
			exist, err := db.HasAncient("a", uint64(i))
			if err != nil {
				t.Fatalf("Failed to check presence, %v", err)
			}
			if exist {
				t.Fatalf("Item %d is already truncated", uint64(i))
			}
			_, err = db.Ancient("a", uint64(i))
			if err == nil {
				t.Fatal("Error is expected for non-existent item")
			}
		}
	}

	// Test the items in range should be reachable
	for i := 10; i < 90; i++ {
		exist, err := db.HasAncient("a", uint64(i))
		if err != nil {
			t.Fatalf("Failed to check presence, %v", err)
		}
		if !exist {
			t.Fatalf("Item %d is missing", uint64(i))
		}
		blob, err := db.Ancient("a", uint64(i))
		if err != nil {
			t.Fatalf("Failed to retrieve item, %v", err)
		}
		if !bytes.Equal(blob, data[i]) {
			t.Fatalf("Unexpected item content, want: %v, got: %v", data[i], blob)
		}
	}

	// Test the items in unknown table shouldn't be reachable
	exist, err := db.HasAncient("b", uint64(0))
	if err != nil {
		t.Fatalf("Failed to check presence, %v", err)
	}
	if exist {
		t.Fatal("Item in unknown table shouldn't be found")
	}
	_, err = db.Ancient("b", uint64(0))
	if err == nil {
		t.Fatal("Error is expected for unknown table")
	}
}

func batchRead(t *testing.T, newFn func(kinds []string) ethdb.AncientStore) {
	var (
		db   = newFn([]string{"a"})
		data = makeDataset(100, 32)
	)
	defer db.Close()

	db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := 0; i < 100; i++ {
			op.AppendRaw("a", uint64(i), data[i])
		}
		return nil
	})
	db.TruncateTail(10)
	db.TruncateHead(90)

	// Test the items in range should be reachable
	var cases = []struct {
		start    uint64
		count    uint64
		maxSize  uint64
		expStart int
		expLimit int
	}{
		// Items in range [10, 90) with no size limitation
		{
			10, 80, 0, 10, 90,
		},
		// Items in range [10, 90) with 32 size cap, single item is expected
		{
			10, 80, 32, 10, 11,
		},
		// Items in range [10, 90) with 31 size cap, single item is expected
		{
			10, 80, 31, 10, 11,
		},
		// Items in range [10, 90) with 32*80 size cap, all items are expected
		{
			10, 80, 32 * 80, 10, 90,
		},
		// Extra items above the last item are not returned
		{
			10, 90, 0, 10, 90,
		},
	}
	for i, c := range cases {
		batch, err := db.AncientRange("a", c.start, c.count, c.maxSize)
		if err != nil {
			t.Fatalf("Failed to retrieve item in range, %v", err)
		}
		if !reflect.DeepEqual(batch, data[c.expStart:c.expLimit]) {
			t.Fatalf("Case %d, Batch content is not matched", i)
		}
	}

	// Test out-of-range / zero-size retrieval should be rejected
	_, err := db.AncientRange("a", 0, 1, 0)
	if err == nil {
		t.Fatal("Out-of-range retrieval should be rejected")
	}
	_, err = db.AncientRange("a", 90, 1, 0)
	if err == nil {
		t.Fatal("Out-of-range retrieval should be rejected")
	}
	_, err = db.AncientRange("a", 10, 0, 0)
	if err == nil {
		t.Fatal("Zero-size retrieval should be rejected")
	}

	// Test item in unknown table shouldn't be reachable
	_, err = db.AncientRange("b", 10, 1, 0)
	if err == nil {
		t.Fatal("Item in unknown table shouldn't be found")
	}
}

func basicWrite(t *testing.T, newFn func(kinds []string) ethdb.AncientStore) {
	var (
		db    = newFn([]string{"a", "b"})
		dataA = makeDataset(100, 32)
		dataB = makeDataset(100, 32)
	)
	defer db.Close()

	// The ancient write to tables should be aligned
	_, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := 0; i < 100; i++ {
			op.AppendRaw("a", uint64(i), dataA[i])
		}
		return nil
	})
	if err == nil {
		t.Fatal("Unaligned ancient write should be rejected")
	}

	// Test normal ancient write
	size, err := db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := 0; i < 100; i++ {
			op.AppendRaw("a", uint64(i), dataA[i])
			op.AppendRaw("b", uint64(i), dataB[i])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to write ancient data %v", err)
	}
	wantSize := int64(6400)
	if size != wantSize {
		t.Fatalf("Ancient write size is not expected, want: %d, got: %d", wantSize, size)
	}

	// Write should work after head truncating
	db.TruncateHead(90)
	_, err = db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := 90; i < 100; i++ {
			op.AppendRaw("a", uint64(i), dataA[i])
			op.AppendRaw("b", uint64(i), dataB[i])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to write ancient data %v", err)
	}

	// Write should work after truncating everything
	db.TruncateTail(0)
	_, err = db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := 0; i < 100; i++ {
			op.AppendRaw("a", uint64(i), dataA[i])
			op.AppendRaw("b", uint64(i), dataB[i])
		}
		return nil
	})
	if err != nil {
		t.Fatalf("Failed to write ancient data %v", err)
	}
}

func nonMutable(t *testing.T, newFn func(kinds []string) ethdb.AncientStore) {
	db := newFn([]string{"a"})
	defer db.Close()

	// We write 100 zero-bytes to the freezer and immediately mutate the slice
	db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		data := make([]byte, 100)
		op.AppendRaw("a", uint64(0), data)
		for i := range data {
			data[i] = 0xff
		}
		return nil
	})
	// Now read it.
	data, err := db.Ancient("a", uint64(0))
	if err != nil {
		t.Fatal(err)
	}
	for k, v := range data {
		if v != 0 {
			t.Fatalf("byte %d != 0: %x", k, v)
		}
	}
}

// TestResettableAncientSuite runs a suite of tests against a resettable ancient
// database implementation.
func TestResettableAncientSuite(t *testing.T, newFn func(kinds []string) ethdb.ResettableAncientStore) {
	t.Run("Reset", func(t *testing.T) {
		var (
			db   = newFn([]string{"a"})
			data = makeDataset(100, 32)
		)
		defer db.Close()

		db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
			for i := 0; i < 100; i++ {
				op.AppendRaw("a", uint64(i), data[i])
			}
			return nil
		})
		db.TruncateTail(10)
		db.TruncateHead(90)

		// Ancient write should work after resetting
		db.Reset()
		db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
			for i := 0; i < 100; i++ {
				op.AppendRaw("a", uint64(i), data[i])
			}
			return nil
		})
	})
}

func makeDataset(size, value int) [][]byte {
	var vals [][]byte
	for i := 0; i < size; i += 1 {
		vals = append(vals, testrand.Bytes(value))
	}
	return vals
}
