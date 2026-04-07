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
)

// mkval constructs a 32-byte value where the first byte is tag and the
// rest are zero. Used to make test assertions easy to read.
func mkval(tag byte) []byte {
	v := make([]byte, stemBlobValueSize)
	v[0] = tag
	return v
}

// TestStemBlobEmpty verifies that a builder with no entries encodes to
// nil (so callers delete the key) and decodes back to a zero bitmap and
// no values.
func TestStemBlobEmpty(t *testing.T) {
	b := newStemBuilder()
	if !b.empty() {
		t.Fatal("fresh builder should be empty")
	}
	blob := b.encode()
	if blob != nil {
		t.Fatalf("empty builder should encode to nil, got %x", blob)
	}

	// Decode nil and empty slice both yield an empty result.
	for _, input := range [][]byte{nil, {}} {
		bitmap, values, err := decodeStemBlob(input)
		if err != nil {
			t.Fatalf("decode empty: %v", err)
		}
		if values != nil {
			t.Fatalf("decode empty values: got %v, want nil", values)
		}
		for i, b := range bitmap {
			if b != 0 {
				t.Fatalf("decode empty bitmap byte %d: got 0x%02x, want 0", i, b)
			}
		}
	}
}

// TestStemBlobBasicDataAndCodeHash verifies the "account header" encoding
// pattern: offsets 0 and 1 populated. This is the common case for every
// account update.
func TestStemBlobBasicDataAndCodeHash(t *testing.T) {
	b := newStemBuilder()
	basicData := mkval(0xAA)
	codeHash := mkval(0xBB)
	b.set(0, basicData)
	b.set(1, codeHash)

	if b.empty() {
		t.Fatal("builder should not be empty after two sets")
	}

	blob := b.encode()
	if blob == nil {
		t.Fatal("encode should not return nil for populated builder")
	}
	if got, want := len(blob), stemBlobBitmapSize+2*stemBlobValueSize; got != want {
		t.Fatalf("blob length: got %d, want %d", got, want)
	}

	// Roundtrip through decodeStemBlob.
	bitmap, values, err := decodeStemBlob(blob)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got := bitmapPopcount(bitmap); got != 2 {
		t.Fatalf("popcount: got %d, want 2", got)
	}
	if !bitmapGet(bitmap, 0) || !bitmapGet(bitmap, 1) {
		t.Fatalf("bitmap missing offset 0 or 1: %x", bitmap)
	}
	if !bytes.Equal(values[0], basicData) {
		t.Fatalf("value[0]: got %x, want %x", values[0], basicData)
	}
	if !bytes.Equal(values[1], codeHash) {
		t.Fatalf("value[1]: got %x, want %x", values[1], codeHash)
	}

	// Point reads via extractStemOffset.
	got, err := extractStemOffset(blob, 0)
	if err != nil {
		t.Fatalf("extract offset 0: %v", err)
	}
	if !bytes.Equal(got, basicData) {
		t.Fatalf("extract 0: got %x, want %x", got, basicData)
	}
	got, err = extractStemOffset(blob, 1)
	if err != nil {
		t.Fatalf("extract offset 1: %v", err)
	}
	if !bytes.Equal(got, codeHash) {
		t.Fatalf("extract 1: got %x, want %x", got, codeHash)
	}
	// An unset offset returns (nil, nil).
	got, err = extractStemOffset(blob, 42)
	if err != nil {
		t.Fatalf("extract unset offset: %v", err)
	}
	if got != nil {
		t.Fatalf("extract unset: got %x, want nil", got)
	}
}

// TestStemBlobAllOffsets verifies that a fully-populated stem (all 256
// offsets) encodes and decodes correctly. This is the worst-case size.
func TestStemBlobAllOffsets(t *testing.T) {
	b := newStemBuilder()
	for i := range stemBlobBitmapBits {
		b.set(byte(i), mkval(byte(i)))
	}
	blob := b.encode()
	expectedLen := stemBlobBitmapSize + stemBlobBitmapBits*stemBlobValueSize
	if len(blob) != expectedLen {
		t.Fatalf("blob length: got %d, want %d", len(blob), expectedLen)
	}

	bitmap, _, err := decodeStemBlob(blob)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if bitmapPopcount(bitmap) != stemBlobBitmapBits {
		t.Fatalf("popcount: got %d, want %d", bitmapPopcount(bitmap), stemBlobBitmapBits)
	}
	for i := range stemBlobBitmapBits {
		got, err := extractStemOffset(blob, byte(i))
		if err != nil {
			t.Fatalf("extract %d: %v", i, err)
		}
		if got[0] != byte(i) {
			t.Fatalf("extract %d: tag 0x%02x, want 0x%02x", i, got[0], byte(i))
		}
	}
}

// TestStemBlobSparseHighOffsets verifies that non-contiguous offsets
// (typical for storage slots scattered across the stem) round-trip
// correctly.
func TestStemBlobSparseHighOffsets(t *testing.T) {
	b := newStemBuilder()
	offsets := []byte{3, 17, 64, 127, 128, 200, 255}
	for _, o := range offsets {
		b.set(o, mkval(o))
	}
	blob := b.encode()
	if len(blob) != stemBlobBitmapSize+len(offsets)*stemBlobValueSize {
		t.Fatalf("unexpected blob length: %d", len(blob))
	}

	// Extract each and verify, including some absent offsets in between.
	for _, o := range offsets {
		got, err := extractStemOffset(blob, o)
		if err != nil {
			t.Fatalf("extract %d: %v", o, err)
		}
		if got[0] != o {
			t.Fatalf("extract %d: tag 0x%02x, want 0x%02x", o, got[0], o)
		}
	}
	// Spot-check absent offsets between populated ones.
	for _, o := range []byte{0, 1, 2, 4, 18, 63, 126, 129, 199, 254} {
		got, err := extractStemOffset(blob, o)
		if err != nil {
			t.Fatalf("extract absent %d: %v", o, err)
		}
		if got != nil {
			t.Fatalf("extract absent %d: got %x, want nil", o, got)
		}
	}
}

// TestStemBlobSetClearRoundtrip verifies that setting and then clearing
// an offset leaves the builder in the same state as never setting it.
func TestStemBlobSetClearRoundtrip(t *testing.T) {
	b := newStemBuilder()
	b.set(5, mkval(0xCD))
	if b.empty() {
		t.Fatal("should not be empty after set")
	}
	b.set(5, nil)
	if !b.empty() {
		t.Fatal("should be empty after clearing the only entry")
	}
	if blob := b.encode(); blob != nil {
		t.Fatalf("encode after clear: got %x, want nil", blob)
	}
}

// TestStemBlobLoadFromBlob verifies that an existing blob can be loaded
// into a fresh builder for read-modify-write semantics.
func TestStemBlobLoadFromBlob(t *testing.T) {
	// Build an initial blob with two entries.
	b1 := newStemBuilder()
	b1.set(0, mkval(0x11))
	b1.set(64, mkval(0x22))
	initial := b1.encode()

	// Load into a fresh builder, modify, encode.
	b2 := newStemBuilder()
	if err := b2.loadFromBlob(initial); err != nil {
		t.Fatalf("loadFromBlob: %v", err)
	}
	b2.set(0, mkval(0x33))  // overwrite offset 0
	b2.set(64, nil)         // clear offset 64
	b2.set(128, mkval(0x44)) // add offset 128
	updated := b2.encode()

	// Offset 0 should have the new value.
	got, err := extractStemOffset(updated, 0)
	if err != nil || got == nil || got[0] != 0x33 {
		t.Fatalf("offset 0 after update: got %x err=%v, want tag 0x33", got, err)
	}
	// Offset 64 should be absent.
	got, err = extractStemOffset(updated, 64)
	if err != nil {
		t.Fatalf("offset 64 after clear: %v", err)
	}
	if got != nil {
		t.Fatalf("offset 64 after clear: got %x, want nil", got)
	}
	// Offset 128 should have the new value.
	got, err = extractStemOffset(updated, 128)
	if err != nil || got == nil || got[0] != 0x44 {
		t.Fatalf("offset 128 after update: got %x err=%v, want tag 0x44", got, err)
	}
}

// TestStemBlobMergeHelper verifies mergeStemBlob: read existing, apply
// writes, produce new blob in one call.
func TestStemBlobMergeHelper(t *testing.T) {
	// Start with a blob containing offset 0.
	b := newStemBuilder()
	b.set(0, mkval(0x01))
	initial := b.encode()

	// Merge: overwrite 0, add 1, clear a non-existent offset (no-op).
	result, err := mergeStemBlob(initial, []stemOffsetValue{
		{Offset: 0, Value: mkval(0x02)},
		{Offset: 1, Value: mkval(0x03)},
		{Offset: 100, Value: nil},
	})
	if err != nil {
		t.Fatalf("merge: %v", err)
	}
	got, _ := extractStemOffset(result, 0)
	if got == nil || got[0] != 0x02 {
		t.Fatalf("merged offset 0: got %x, want tag 0x02", got)
	}
	got, _ = extractStemOffset(result, 1)
	if got == nil || got[0] != 0x03 {
		t.Fatalf("merged offset 1: got %x, want tag 0x03", got)
	}
}

// TestStemBlobMergeToEmpty verifies that clearing every populated entry
// via merge returns a nil blob (so the caller deletes the key).
func TestStemBlobMergeToEmpty(t *testing.T) {
	b := newStemBuilder()
	b.set(0, mkval(0x01))
	b.set(5, mkval(0x02))
	initial := b.encode()

	result, err := mergeStemBlob(initial, []stemOffsetValue{
		{Offset: 0, Value: nil},
		{Offset: 5, Value: nil},
	})
	if err != nil {
		t.Fatalf("merge to empty: %v", err)
	}
	if result != nil {
		t.Fatalf("merge to empty: got %x, want nil", result)
	}
}

// TestStemBlobTombstoneZeroBytes verifies that a 32-byte zero value is
// preserved as "present with zero value" — not confused with "absent".
// DeleteStorage uses this convention.
func TestStemBlobTombstoneZeroBytes(t *testing.T) {
	b := newStemBuilder()
	zeros := make([]byte, stemBlobValueSize)
	b.set(64, zeros)
	if b.empty() {
		t.Fatal("zero-value entry should count as populated")
	}
	blob := b.encode()
	got, err := extractStemOffset(blob, 64)
	if err != nil {
		t.Fatalf("extract tombstone: %v", err)
	}
	if !bytes.Equal(got, zeros) {
		t.Fatalf("extract tombstone: got %x, want 32 zero bytes", got)
	}
}

// TestStemBlobMalformedInput verifies that decodeStemBlob detects
// malformed blobs with wrong lengths.
func TestStemBlobMalformedInput(t *testing.T) {
	// Shorter than bitmap.
	if _, _, err := decodeStemBlob(make([]byte, 10)); err == nil {
		t.Fatal("expected error for too-short blob")
	}
	// Bitmap claims 2 entries but blob only has room for 1.
	var bitmap [stemBlobBitmapSize]byte
	bitmap[0] = 0xC0 // bits 0 and 1 set → 2 entries
	short := make([]byte, stemBlobBitmapSize+stemBlobValueSize)
	copy(short, bitmap[:])
	if _, _, err := decodeStemBlob(short); err == nil {
		t.Fatal("expected error for blob shorter than bitmap implies")
	}
}

// TestBitmapRank sanity-checks the bit-to-index helper used by
// extractStemOffset for single-offset reads.
func TestBitmapRank(t *testing.T) {
	var bitmap [stemBlobBitmapSize]byte
	// Set bits at offsets 0, 1, 5, 64, 200.
	for _, o := range []byte{0, 1, 5, 64, 200} {
		bitmap[o/8] |= 1 << (7 - uint(o%8))
	}
	cases := []struct {
		offset byte
		want   int
	}{
		{0, 0},   // first set bit is at index 0
		{1, 1},   // second set bit
		{5, 2},   // third
		{64, 3},  // fourth
		{200, 4}, // fifth
		// For an unset offset, rank returns the number of set bits < it.
		{2, 2},    // bits 0 and 1 are before 2
		{100, 4},  // bits 0,1,5,64 are before 100
		{255, 5},  // all five bits are before 255
	}
	for _, c := range cases {
		if got := bitmapRank(bitmap, c.offset); got != c.want {
			t.Errorf("bitmapRank(%d) = %d, want %d", c.offset, got, c.want)
		}
	}
}
