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
	"errors"
	"fmt"
	"math/bits"

	"github.com/ethereum/go-ethereum/common"
)

// Bintrie stem blob layout
// ------------------------
//
// The flat-state representation of a bintrie stem packs the populated
// (offset, 32-byte value) pairs at that stem into a single on-disk blob.
// A stem holds up to 256 offsets (per EIP-7864, the full "stem group"),
// but in practice only a handful are populated for any given account
// (BasicData at offset 0, CodeHash at offset 1, a few storage slots, or
// code chunks). A dense encoding would waste 8 KB per stem; this layout
// scales linearly with the number of populated offsets.
//
// Layout:
//
//	[ 0 .. 31 ]   32-byte bitmap; bit i set iff offset i has a value
//	[32 .. 63 ]   first populated offset's 32-byte value
//	[64 .. 95 ]   second populated offset's 32-byte value
//	...
//	[32 + 32*(N-1) .. 32 + 32*N - 1]  N-th populated offset's value
//
// where N = popcount(bitmap). Values appear in increasing offset order,
// which is the iteration order of the bitmap bits from least- to
// most-significant byte (byte 0 first, then byte 1, etc.), and within
// each byte from MSB (offset b*8) to LSB (offset b*8+7).
//
// An "absent" offset is one whose bitmap bit is clear; an offset whose
// value is 32 zero bytes is "present with zero value" — that is the
// tombstone convention used by BinaryTrie.DeleteStorage, which writes
// 32 zero bytes to mark a slot as cleared without removing it from the
// underlying StemNode's Values slice.
//
// An empty stem (all bits clear) is represented by a zero-length blob,
// and callers must delete the on-disk key rather than write a zero-length
// value.
const (
	stemBlobBitmapSize = 32                      // bytes
	stemBlobBitmapBits = stemBlobBitmapSize * 8  // 256
	stemBlobValueSize  = common.HashLength       // 32
)

// stemOffsetMax is the highest valid offset within a bintrie stem.
const stemOffsetMax = stemBlobBitmapBits - 1 // 255

var (
	errStemBlobTooShort        = errors.New("stem blob shorter than bitmap")
	errStemBlobMalformed       = errors.New("stem blob length does not match bitmap popcount")
	errStemBlobValueOutOfRange = errors.New("stem blob value slice out of range")
)

// encodeStemBlob encodes a bitmap and a dense values slice (one entry per
// set bit, in ascending offset order) into the wire format described at
// the top of this file.
//
// The caller must ensure len(values) == popcount(bitmap) and that every
// entry in values has len == 32. If every bitmap bit is clear the function
// returns nil so the caller knows to delete the on-disk key.
func encodeStemBlob(bitmap [stemBlobBitmapSize]byte, values [][]byte) ([]byte, error) {
	count := bitmapPopcount(bitmap)
	if count != len(values) {
		return nil, fmt.Errorf("stem blob popcount=%d values=%d: %w", count, len(values), errStemBlobMalformed)
	}
	if count == 0 {
		return nil, nil
	}
	out := make([]byte, stemBlobBitmapSize+count*stemBlobValueSize)
	copy(out, bitmap[:])
	for i, v := range values {
		if len(v) != stemBlobValueSize {
			return nil, fmt.Errorf("stem blob value %d has len %d: %w", i, len(v), errStemBlobMalformed)
		}
		copy(out[stemBlobBitmapSize+i*stemBlobValueSize:], v)
	}
	return out, nil
}

// decodeStemBlob parses a raw stem blob into its bitmap and an ordered
// slice of populated 32-byte values. The returned values alias the input
// slice; callers must not retain or mutate them without copying first.
//
// A nil or zero-length blob decodes to a zero bitmap and no values
// (equivalent to "no offsets present").
func decodeStemBlob(blob []byte) ([stemBlobBitmapSize]byte, [][]byte, error) {
	var bitmap [stemBlobBitmapSize]byte
	if len(blob) == 0 {
		return bitmap, nil, nil
	}
	if len(blob) < stemBlobBitmapSize {
		return bitmap, nil, errStemBlobTooShort
	}
	copy(bitmap[:], blob[:stemBlobBitmapSize])
	count := bitmapPopcount(bitmap)
	expected := stemBlobBitmapSize + count*stemBlobValueSize
	if len(blob) != expected {
		return bitmap, nil, fmt.Errorf("stem blob len=%d popcount=%d expected=%d: %w", len(blob), count, expected, errStemBlobMalformed)
	}
	if count == 0 {
		return bitmap, nil, nil
	}
	values := make([][]byte, count)
	for i := range values {
		start := stemBlobBitmapSize + i*stemBlobValueSize
		values[i] = blob[start : start+stemBlobValueSize]
	}
	return bitmap, values, nil
}

// extractStemOffset returns the 32-byte value at the given offset within
// a stem blob, or nil if the offset is not present. It does not allocate;
// the returned slice aliases the input blob and must not be mutated.
//
// Returns an error only if the blob itself is malformed. An absent offset
// in a well-formed blob is (nil, nil) — not an error.
func extractStemOffset(blob []byte, offset byte) ([]byte, error) {
	if len(blob) == 0 {
		return nil, nil
	}
	if len(blob) < stemBlobBitmapSize {
		return nil, errStemBlobTooShort
	}
	var bitmap [stemBlobBitmapSize]byte
	copy(bitmap[:], blob[:stemBlobBitmapSize])

	// Is the offset present at all?
	if !bitmapGet(bitmap, offset) {
		return nil, nil
	}
	// Count how many set bits precede this offset to find the value slot.
	idx := bitmapRank(bitmap, offset)
	start := stemBlobBitmapSize + idx*stemBlobValueSize
	end := start + stemBlobValueSize
	if end > len(blob) {
		return nil, errStemBlobValueOutOfRange
	}
	return blob[start:end], nil
}

// stemBuilder accumulates (offset, value) pairs and produces a stem blob.
// It supports loading an existing blob, setting individual offsets, and
// emitting the final encoded form.
//
// Setting a value of nil or an empty slice clears the corresponding bit
// from the bitmap (the offset becomes "absent"). Setting a non-nil
// 32-byte slice — including 32 zero bytes — marks the offset present
// with that value. This preserves the distinction between absent and
// tombstoned-with-zero used elsewhere in the bintrie code.
//
// A stemBuilder is not safe for concurrent use.
type stemBuilder struct {
	bitmap [stemBlobBitmapSize]byte
	// values stores the current value at each offset, or nil if absent.
	// Using a fixed 256-entry array avoids allocation churn as offsets
	// are set and cleared.
	values [stemBlobBitmapBits][]byte
}

// newStemBuilder returns an empty stemBuilder.
func newStemBuilder() *stemBuilder {
	return &stemBuilder{}
}

// loadFromBlob merges the entries of the given stem blob into the builder.
// Existing entries at the same offsets are overwritten. An empty blob is
// a no-op.
func (b *stemBuilder) loadFromBlob(blob []byte) error {
	if len(blob) == 0 {
		return nil
	}
	bitmap, values, err := decodeStemBlob(blob)
	if err != nil {
		return err
	}
	// Walk the bitmap and copy each populated offset into the builder,
	// stepping the values index in sync.
	var vi int
	for offset := range stemBlobBitmapBits {
		if !bitmapGet(bitmap, byte(offset)) {
			continue
		}
		// decodeStemBlob returns slices aliasing the input blob; we take
		// an owning copy so the builder survives the caller mutating or
		// releasing the source blob.
		v := make([]byte, stemBlobValueSize)
		copy(v, values[vi])
		b.values[offset] = v
		b.bitmap[offset/8] |= 1 << (7 - uint(offset%8))
		vi++
	}
	return nil
}

// set writes value at the given offset. A nil or empty-length value
// clears the offset (bitmap bit cleared). A non-nil 32-byte value sets
// the offset present with that value. Setting with any other length
// panics — callers are expected to always pass 32-byte values.
func (b *stemBuilder) set(offset byte, value []byte) {
	if len(value) == 0 {
		b.values[offset] = nil
		b.bitmap[offset/8] &^= 1 << (7 - uint(offset%8))
		return
	}
	if len(value) != stemBlobValueSize {
		panic(fmt.Sprintf("stemBuilder: value at offset %d has len %d, want %d", offset, len(value), stemBlobValueSize))
	}
	// Own the bytes so later caller mutations don't aliasing-surprise us.
	owned := make([]byte, stemBlobValueSize)
	copy(owned, value)
	b.values[offset] = owned
	b.bitmap[offset/8] |= 1 << (7 - uint(offset%8))
}

// empty reports whether no offsets are currently populated in the builder.
func (b *stemBuilder) empty() bool {
	return bitmapPopcount(b.bitmap) == 0
}

// encode produces the stem blob encoding for the builder's current state.
// Returns nil for an empty builder so the caller can decide to delete the
// on-disk key rather than write a zero-length value.
func (b *stemBuilder) encode() []byte {
	count := bitmapPopcount(b.bitmap)
	if count == 0 {
		return nil
	}
	out := make([]byte, stemBlobBitmapSize+count*stemBlobValueSize)
	copy(out, b.bitmap[:])

	// Walk the bitmap in ascending order, copying each populated value.
	pos := stemBlobBitmapSize
	for offset := range stemBlobBitmapBits {
		if b.values[offset] == nil {
			continue
		}
		copy(out[pos:], b.values[offset])
		pos += stemBlobValueSize
	}
	return out
}

// reset clears all entries in the builder.
func (b *stemBuilder) reset() {
	b.bitmap = [stemBlobBitmapSize]byte{}
	b.values = [stemBlobBitmapBits][]byte{}
}

// stemOffsetValue is a single (offset, value) pair passed to mergeStemBlob.
// A nil Value clears the offset.
type stemOffsetValue struct {
	Offset byte
	Value  []byte
}

// mergeStemBlob performs a read-modify-write on a stem blob: it decodes
// the existing blob (if any), applies the given writes in order, and
// returns a freshly encoded blob. Returns (nil, nil) when the result is
// empty — the caller should delete the on-disk key in that case.
func mergeStemBlob(existing []byte, writes []stemOffsetValue) ([]byte, error) {
	b := newStemBuilder()
	if err := b.loadFromBlob(existing); err != nil {
		return nil, err
	}
	for _, w := range writes {
		b.set(w.Offset, w.Value)
	}
	return b.encode(), nil
}

// bitmapPopcount returns the number of set bits in the 32-byte bitmap.
func bitmapPopcount(bitmap [stemBlobBitmapSize]byte) int {
	var n int
	for _, b := range bitmap {
		n += bits.OnesCount8(b)
	}
	return n
}

// bitmapGet returns whether bit `offset` is set in the bitmap. The
// convention mirrors the bintrie: bit index `offset` lives in byte
// `offset/8`, with the MSB of that byte corresponding to the lowest
// in-byte offset (`offset%8 == 0`).
func bitmapGet(bitmap [stemBlobBitmapSize]byte, offset byte) bool {
	return bitmap[offset/8]&(1<<(7-uint(offset%8))) != 0
}

// bitmapRank returns the number of set bits that come strictly before
// `offset` (in ascending offset order). The offset itself does not count.
func bitmapRank(bitmap [stemBlobBitmapSize]byte, offset byte) int {
	// Full whole bytes before the target.
	byteIdx := int(offset) / 8
	var rank int
	for i := range byteIdx {
		rank += bits.OnesCount8(bitmap[i])
	}
	// Bits within the target byte that are above the target's bit.
	bitIdx := offset % 8
	if bitIdx > 0 {
		// The MSB is offset%8==0. We want bits 0..bitIdx-1 in that layout,
		// which are the top bitIdx bits of the byte.
		mask := byte(0xFF << (8 - bitIdx))
		rank += bits.OnesCount8(bitmap[byteIdx] & mask)
	}
	return rank
}
