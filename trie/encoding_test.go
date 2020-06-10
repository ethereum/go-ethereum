// Copyright 2014 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"testing"
)

func TestBinCompact(t *testing.T) {
	tests := []struct{ bin, compact []byte }{
		// empty keys, with and without terminator
		{bin: []byte{}, compact: []byte{0x40}},  // 0100 0000
		{bin: []byte{2}, compact: []byte{0xc0}}, // 1100 0000

		// length 1 with and without terminator
		{bin: []byte{1}, compact: []byte{0x38}},    // 0011 1000
		{bin: []byte{1, 2}, compact: []byte{0xb8}}, // 1011 1000

		// length 2 with and without terminator
		{bin: []byte{0, 1}, compact: []byte{0x24}},    // 0010 0100
		{bin: []byte{0, 1, 2}, compact: []byte{0xa4}}, // 1010 0100

		// length 3 with and without terminator
		{bin: []byte{1, 0, 1}, compact: []byte{0x1a}},    // 0001 1010
		{bin: []byte{1, 0, 1, 2}, compact: []byte{0x9a}}, // 1001 1010

		// length 4 with and without terminator
		{bin: []byte{1, 0, 1, 0}, compact: []byte{0x0a}},    // 0000 1010
		{bin: []byte{1, 0, 1, 0, 2}, compact: []byte{0x8a}}, // 1000 1010

		// length 5 with and without terminator
		{bin: []byte{1, 0, 1, 0, 1}, compact: []byte{0x7a, 0x80}},    // 0111 1010 1000 0000
		{bin: []byte{1, 0, 1, 0, 1, 2}, compact: []byte{0xfa, 0x80}}, // 1111 1010 1000 0000

		// length 6 with and without terminator
		{bin: []byte{1, 0, 1, 0, 1, 0}, compact: []byte{0x6a, 0x80}},    // 0110 1010 1000 0000
		{bin: []byte{1, 0, 1, 0, 1, 0, 2}, compact: []byte{0xea, 0x80}}, // 1110 1010 1000 0000

		// length 7 with and without terminator
		{bin: []byte{1, 0, 1, 0, 1, 0, 1}, compact: []byte{0x5a, 0xa0}},    // 0101 1010 1010 0000
		{bin: []byte{1, 0, 1, 0, 1, 0, 1, 2}, compact: []byte{0xda, 0xa0}}, // 1101 1010 1010 0000

		// length 8 with and without terminator
		{bin: []byte{1, 0, 1, 0, 1, 0, 1, 0}, compact: []byte{0x4a, 0xa0}},    // 0100 1010 1010 0000
		{bin: []byte{1, 0, 1, 0, 1, 0, 1, 0, 2}, compact: []byte{0xca, 0xa0}}, // 1100 1010 1010 0000

		// 32-byte key with and without terminator
		{
			bin:     bytes.Repeat([]byte{1, 0}, 4*32),
			compact: append(append([]byte{0x4a}, bytes.Repeat([]byte{0xaa}, 31)...), 0xa0),
		},
		{
			bin:     append(bytes.Repeat([]byte{1, 0}, 4*32), 0x2),
			compact: append(append([]byte{0xca}, bytes.Repeat([]byte{0xaa}, 31)...), 0xa0),
		},
	}
	for _, test := range tests {
		if c := binaryKeyToCompactKey(test.bin); !bytes.Equal(c, test.compact) {
			t.Errorf("binaryKeyToCompactKey(%x) -> %x, want %x", test.bin, c, test.compact)
		}
		if h := compactKeyToBinaryKey(test.compact); !bytes.Equal(h, test.bin) {
			t.Errorf("compactKeyToBinaryKey(%x) -> %x, want %x", test.compact, h, test.bin)
		}
	}
}

func TestBinaryKeyBytes(t *testing.T) {
	tests := []struct{ key, binaryIn, binaryOut []byte }{
		{key: []byte{}, binaryIn: []byte{2}, binaryOut: []byte{2}},
		{key: []byte{}, binaryIn: []byte{}, binaryOut: []byte{2}},
		{
			key:       []byte{0x12, 0x34, 0x56},
			binaryIn:  []byte{0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 0, 0, 1, 0, 1, 0, 1, 1, 0, 2},
			binaryOut: []byte{0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 0, 0, 1, 0, 1, 0, 1, 1, 0, 2},
		},
		{
			key:       []byte{0x12, 0x34, 0x5},
			binaryIn:  []byte{0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 2},
			binaryOut: []byte{0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 0, 0, 0, 0, 0, 0, 1, 0, 1, 2},
		},
		{
			key:       []byte{0x12, 0x34, 0x56},
			binaryIn:  []byte{0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 0, 0, 1, 0, 1, 0, 1, 1, 0},
			binaryOut: []byte{0, 0, 0, 1, 0, 0, 1, 0, 0, 0, 1, 1, 0, 1, 0, 0, 0, 1, 0, 1, 0, 1, 1, 0, 2},
		},
	}
	for _, test := range tests {
		if h := keyBytesToBinaryKey(test.key); !bytes.Equal(h, test.binaryOut) {
			t.Errorf("keyBytesToBinaryKey(%x) -> %b, want %b", test.key, h, test.binaryOut)
		}
		if k := binaryKeyToKeyBytes(test.binaryIn); !bytes.Equal(k, test.key) {
			t.Errorf("binaryKeyToKeyBytes(%b) -> %x, want %x", test.binaryIn, k, test.key)
		}
	}
}
