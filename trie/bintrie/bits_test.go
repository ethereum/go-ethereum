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

package bintrie

import (
	"bytes"
	"math/rand"
	"testing"
)

// TestEncodeBitPrefixVectors pins the EIP-8297 encode_bit_prefix layout.
//
// The expected bytes are hand-written from the spec, not exported from the
// reference implementation: eip8297_vectors.json carries no bit-prefix section,
// so nothing here is machine-checked against EELS. Read them as a guard against
// this package changing the layout, not as evidence that the layout is right.
func TestEncodeBitPrefixVectors(t *testing.T) {
	cases := []struct {
		bits []byte
		want []byte
	}{
		{nil, []byte{0x00, 0x00}},
		{[]byte{1, 0, 1}, []byte{0x00, 0x03, 0xa0}},
		{[]byte{1, 1, 1, 1, 1, 1, 1, 1, 1}, []byte{0x00, 0x09, 0xff, 0x80}},
		{[]byte{0, 0, 0}, []byte{0x00, 0x03, 0x00}},
		{bytes.Repeat([]byte{1}, 8), []byte{0x00, 0x08, 0xff}},
	}
	for _, c := range cases {
		got := encodeBitPrefix(bitsToBitstr(c.bits))
		if !bytes.Equal(got, c.want) {
			t.Fatalf("bits %v: got %x want %x", c.bits, got, c.want)
		}
	}
}

// TestBitPrefixTrailingZeroDistinct pins that prefixes differing only in
// trailing zero bits encode differently (the injectivity the bit count
// exists to provide).
func TestBitPrefixTrailingZeroDistinct(t *testing.T) {
	a := encodeBitPrefix(bitsToBitstr([]byte{1, 0}))
	b := encodeBitPrefix(bitsToBitstr([]byte{1, 0, 0}))
	if bytes.Equal(a, b) {
		t.Fatal("trailing-zero prefixes must encode distinctly")
	}
}

func TestBitPrefixRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(8297))
	for i := 0; i < 200; i++ {
		n := rng.Intn(530)
		bits := make([]byte, n)
		for j := range bits {
			bits[j] = byte(rng.Intn(2))
		}
		p := bitsToBitstr(bits)
		enc := encodeBitPrefix(p)
		dec, consumed, err := decodeBitPrefix(enc)
		if err != nil {
			t.Fatal(err)
		}
		if consumed != len(enc) || !dec.equal(p) {
			t.Fatalf("round trip failed for %d bits", n)
		}
	}
	// Non-canonical padding must be rejected.
	if _, _, err := decodeBitPrefix([]byte{0x00, 0x03, 0xa1}); err == nil {
		t.Fatal("expected non-canonical padding rejection")
	}
	// Truncated payloads must be rejected.
	if _, _, err := decodeBitPrefix([]byte{0x00, 0x09, 0xff}); err == nil {
		t.Fatal("expected truncation rejection")
	}
}

func TestSliceConcatMatch(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for i := 0; i < 500; i++ {
		key := make([]byte, 1+rng.Intn(66))
		rng.Read(key)
		from := rng.Intn(8 * len(key))
		n := rng.Intn(8*len(key) - from + 1)
		p := slice(key, from, n)
		if p.n != n {
			t.Fatalf("slice length %d want %d", p.n, n)
		}
		for j := 0; j < n; j++ {
			if p.bit(j) != bitAt(key, from+j) {
				t.Fatalf("slice bit %d mismatch", j)
			}
		}
		// matchKey of the extracted slice against its own source is exact.
		if m := p.matchKey(key, from); m != n {
			t.Fatalf("matchKey %d want %d", m, n)
		}
	}
	// concat reassembles a split prefix around its split bit.
	full := slice([]byte{0xde, 0xad, 0xbe, 0xef}, 3, 20)
	for cut := 0; cut < full.n; cut++ {
		head := slice([]byte{0xde, 0xad, 0xbe, 0xef}, 3, cut)
		tail := slice([]byte{0xde, 0xad, 0xbe, 0xef}, 3+cut+1, full.n-cut-1)
		if got := head.concat(full.bit(cut), tail); !got.equal(full) {
			t.Fatalf("concat mismatch at cut %d", cut)
		}
	}
}

func TestCommonPrefixLen(t *testing.T) {
	rng := rand.New(rand.NewSource(2))
	for i := 0; i < 500; i++ {
		a := make([]byte, 4+rng.Intn(63))
		rng.Read(a)
		b := append([]byte{}, a...)
		from := rng.Intn(8*len(a) - 1)
		// Reference: brute-force bit compare.
		brute := func() int {
			max := 8 * min(len(a), len(b))
			for j := from; j < max; j++ {
				if bitAt(a, j) != bitAt(b, j) {
					return j - from
				}
			}
			return max - from
		}
		if got, want := commonPrefixLen(a, b, from), brute(); got != want {
			t.Fatalf("identical: got %d want %d", got, want)
		}
		// Flip one bit after from and re-check.
		flip := from + rng.Intn(8*len(a)-from)
		b[flip>>3] ^= 1 << (7 - flip&7)
		if got, want := commonPrefixLen(a, b, from), brute(); got != want {
			t.Fatalf("flipped: got %d want %d (from %d flip %d)", got, want, from, flip)
		}
	}
}

func TestEncodePathRoot(t *testing.T) {
	if got := encodePath([]byte{0xff}, 0); len(got) != 0 {
		t.Fatalf("root path must be empty, got %x", got)
	}
	if got := encodePath([]byte{0b10100000}, 3); !bytes.Equal(got, []byte{0x00, 0x03, 0xa0}) {
		t.Fatalf("path encoding mismatch: %x", got)
	}
}
