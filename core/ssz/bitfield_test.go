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

package ssz

import (
	"bytes"
	"errors"
	"math/rand"
	"testing"
)

func TestBitvectorSize(t *testing.T) {
	for _, c := range []struct {
		nbits uint64
		size  int
	}{{1, 1}, {7, 1}, {8, 1}, {9, 2}, {16, 2}, {256, 32}, {257, 33}} {
		if got := BitvectorSize(c.nbits); got != c.size {
			t.Errorf("BitvectorSize(%d) = %d, want %d", c.nbits, got, c.size)
		}
	}
}

func TestValidateBitvector(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		nbits uint64
		err   error
	}{
		{"full byte", []byte{0xff}, 8, nil},
		{"partial byte in range", []byte{0x1f}, 5, nil},
		{"two bytes", []byte{0xff, 0x01}, 9, nil},
		{"too short", []byte{0xff}, 9, ErrSize},
		{"too long", []byte{0xff, 0x00}, 8, ErrSize},
		{"excess bit", []byte{0x20}, 5, ErrExcessBits},
		{"excess high bit", []byte{0x80}, 1, ErrExcessBits},
		{"excess in second byte", []byte{0xff, 0x02}, 9, ErrExcessBits},
	}
	for _, tc := range tests {
		if err := ValidateBitvector(tc.data, tc.nbits); !errors.Is(err, tc.err) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.err)
		}
	}
}

func TestBitlistSize(t *testing.T) {
	for _, c := range []struct {
		nbits uint64
		size  int
	}{{0, 1}, {1, 1}, {7, 1}, {8, 2}, {15, 2}, {16, 3}} {
		if got := BitlistSize(c.nbits); got != c.size {
			t.Errorf("BitlistSize(%d) = %d, want %d", c.nbits, got, c.size)
		}
	}
}

// TestBitlistDelimiter pins the delimiter to each bit position of the last
// byte, including the case where it opens a new byte.
func TestBitlistDelimiter(t *testing.T) {
	for nbits := uint64(0); nbits <= 9; nbits++ {
		bits := []byte{0xff, 0xff}[:(nbits+7)/8]
		if rem := nbits % 8; rem != 0 {
			bits[len(bits)-1] &= 1<<rem - 1 // no bits at or above nbits
		}
		got := AppendBitlist([]byte{0xee}, bits, nbits)
		want := append([]byte{0xee}, bits...)
		if nbits%8 == 0 {
			want = append(want, 1)
		} else {
			want[len(want)-1] |= 1 << (nbits % 8)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("nbits=%d: %x, want %x", nbits, got, want)
		}
		if len(got)-1 != BitlistSize(nbits) {
			t.Errorf("nbits=%d: %d bytes, BitlistSize says %d", nbits, len(got)-1, BitlistSize(nbits))
		}
	}
}

func TestBitlistRoundTrip(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	for nbits := uint64(0); nbits <= 70; nbits++ {
		bits := make([]byte, (nbits+7)/8)
		rng.Read(bits)
		if rem := nbits % 8; rem != 0 {
			bits[len(bits)-1] &= 1<<rem - 1
		}
		ser := AppendBitlist(nil, bits, nbits)
		gotBits, gotN, err := DecodeBitlist(ser, nbits)
		if err != nil {
			t.Fatalf("nbits=%d: %v", nbits, err)
		}
		if gotN != nbits || !bytes.Equal(gotBits, bits) {
			t.Errorf("nbits=%d: decoded (%x, %d), want (%x, %d)", nbits, gotBits, gotN, bits, nbits)
		}
	}
}

func TestDecodeBitlistErrors(t *testing.T) {
	tests := []struct {
		name  string
		data  []byte
		limit uint64
		err   error
	}{
		{"empty", nil, 8, ErrBadBitlist},
		{"zero last byte", []byte{0xff, 0x00}, 16, ErrBadBitlist},
		{"over limit", []byte{0x10}, 3, ErrTooBig},
		{"at limit", []byte{0x08}, 3, nil},
		{"zero limit ok", []byte{0x01}, 0, nil},
	}
	for _, tc := range tests {
		if _, _, err := DecodeBitlist(tc.data, tc.limit); !errors.Is(err, tc.err) {
			t.Errorf("%s: err = %v, want %v", tc.name, err, tc.err)
		}
	}
}
