// Copyright 2026 The go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie/bintrie"
)

// TestStemBlobOffset127_128Boundary tests the bitmap byte boundary
// between offset 127 (last header storage slot) and offset 128
// (first code chunk). Off-by-one in bitmapRank at this boundary
// would cause extractStemOffset to return the wrong value.
func TestStemBlobOffset127_128Boundary(t *testing.T) {
	b := newStemBuilder()
	val127 := bytes.Repeat([]byte{0x7F}, stemBlobValueSize)
	val128 := bytes.Repeat([]byte{0x80}, stemBlobValueSize)
	b.set(127, val127)
	b.set(128, val128)

	blob := b.encode()
	if blob == nil {
		t.Fatal("encode returned nil for 2-offset builder")
	}

	got127, err := extractStemOffset(blob, 127)
	if err != nil {
		t.Fatalf("extract offset 127: %v", err)
	}
	if !bytes.Equal(got127, val127) {
		t.Errorf("offset 127: got %x, want %x", got127, val127)
	}

	got128, err := extractStemOffset(blob, 128)
	if err != nil {
		t.Fatalf("extract offset 128: %v", err)
	}
	if !bytes.Equal(got128, val128) {
		t.Errorf("offset 128: got %x, want %x", got128, val128)
	}

	// Verify bitmapRank correctness at the byte boundary.
	var bitmap [stemBlobBitmapSize]byte
	copy(bitmap[:], blob[:stemBlobBitmapSize])
	if r := bitmapRank(bitmap, 127); r != 0 {
		t.Errorf("bitmapRank(127) = %d, want 0", r)
	}
	if r := bitmapRank(bitmap, 128); r != 1 {
		t.Errorf("bitmapRank(128) = %d, want 1", r)
	}
}

// TestStemBlobFull256DeleteMiddle tests a fully-populated stem (all 256
// offsets) where one offset in the middle is deleted.
func TestStemBlobFull256DeleteMiddle(t *testing.T) {
	b := newStemBuilder()
	for i := range 256 {
		val := bytes.Repeat([]byte{byte(i)}, stemBlobValueSize)
		b.set(byte(i), val)
	}
	if bitmapPopcount(b.bitmap) != 256 {
		t.Fatalf("full builder has popcount %d, want 256", bitmapPopcount(b.bitmap))
	}

	b.set(128, nil) // delete the middle
	if bitmapPopcount(b.bitmap) != 255 {
		t.Fatalf("after delete: popcount %d, want 255", bitmapPopcount(b.bitmap))
	}

	blob := b.encode()
	expectedSize := stemBlobBitmapSize + 255*stemBlobValueSize
	if len(blob) != expectedSize {
		t.Fatalf("blob size %d, want %d", len(blob), expectedSize)
	}

	got128, _ := extractStemOffset(blob, 128)
	if got128 != nil {
		t.Errorf("offset 128 should be absent, got %x", got128)
	}

	got127, _ := extractStemOffset(blob, 127)
	if !bytes.Equal(got127, bytes.Repeat([]byte{127}, stemBlobValueSize)) {
		t.Errorf("offset 127 corrupted after deleting 128")
	}
	got129, _ := extractStemOffset(blob, 129)
	if !bytes.Equal(got129, bytes.Repeat([]byte{129}, stemBlobValueSize)) {
		t.Errorf("offset 129 corrupted after deleting 128")
	}
	got0, _ := extractStemOffset(blob, 0)
	if !bytes.Equal(got0, bytes.Repeat([]byte{0}, stemBlobValueSize)) {
		t.Errorf("offset 0 corrupted")
	}
	got255, _ := extractStemOffset(blob, 255)
	if !bytes.Equal(got255, bytes.Repeat([]byte{255}, stemBlobValueSize)) {
		t.Errorf("offset 255 corrupted")
	}
}

// TestFlushIdempotency verifies that flushing the same data twice
// produces an identical on-disk blob.
func TestFlushIdempotency(t *testing.T) {
	codec, db := newTestBintrieCodec(t)
	stem := bytes.Repeat([]byte{0x55}, bintrie.StemSize)
	mkKey := func(offset byte) common.Hash {
		var k common.Hash
		copy(k[:bintrie.StemSize], stem)
		k[bintrie.StemSize] = offset
		return k
	}
	val := bytes.Repeat([]byte{0xAA}, stemBlobValueSize)

	batch := db.NewBatch()
	codec.Flush(batch, nil, map[common.Hash][]byte{mkKey(5): val}, nil, nil)
	flushBatch(t, batch)
	blob1 := rawdb.ReadBinTrieStem(db, stem)

	batch = db.NewBatch()
	codec.Flush(batch, nil, map[common.Hash][]byte{mkKey(5): val}, nil, nil)
	flushBatch(t, batch)
	blob2 := rawdb.ReadBinTrieStem(db, stem)

	if !bytes.Equal(blob1, blob2) {
		t.Errorf("Flush is not idempotent: blob1 len=%d blob2 len=%d", len(blob1), len(blob2))
	}
}
