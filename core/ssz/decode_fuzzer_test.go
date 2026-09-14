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

package ssz_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/core/ssz"
)

// FuzzSSZDecode drives arbitrary input through the strict decoder of every
// composite shape in the spec type set and enforces three invariants:
//
//  1. no panic: malformed input is always rejected with an error wrapping a
//     named sentinel of the package;
//  2. strict decode is injective: when input does decode, re-encoding
//     reproduces it byte for byte, so no two byte strings decode to the same
//     value and hashing a decoded value is equivalent to hashing the wire
//     bytes;
//  3. hash_tree_root of every decoded value can be computed without panic.
func FuzzSSZDecode(f *testing.F) {
	seed := func(obj specObject) []byte {
		blob, err := ssz.Encode(obj)
		if err != nil {
			f.Fatal(err)
		}
		return blob
	}
	f.Add(seed(&varTestStruct{A: 1, B: []uint16{2, 3}, C: 4}))
	f.Add(seed(&complexTestStruct{
		A: 1, B: []uint16{2, 3}, C: 4, D: hexBlob{5, 6, 7},
		E: varTestStruct{A: 8, B: []uint16{9}, C: 10},
		F: [4]fixedTestStruct{{A: 1, B: 2, C: 3}},
		G: [2]varTestStruct{{A: 11, B: []uint16{12, 13}, C: 14}, {A: 15}},
	}))
	f.Add(seed(&bitsStruct{
		A: yamlBitlist{bits: []byte{0x15}, nbits: 5},
		B: hexBlob{0x03}, C: hexBlob{0x01},
		D: yamlBitlist{bits: []byte{0x2a}, nbits: 6},
		E: hexBlob{0xff},
	}))
	// Adversarial seeds: truncated fixed part, offset out of range, first
	// offset not equal to the fixed size.
	f.Add([]byte{0x07, 0x00, 0x00})
	f.Add([]byte{0xff, 0xff, 0xff, 0xff, 0x00})
	f.Add([]byte{0x02, 0x00, 0x08, 0x00, 0x00, 0x00})

	f.Fuzz(func(t *testing.T, data []byte) {
		targets := []specObject{
			new(fixedTestStruct),
			new(varTestStruct),
			new(complexTestStruct),
			new(bitsStruct),
			&testBitlist{limit: 64},
			&testBasicVector{kind: kindUint64, length: 3},
		}
		for _, obj := range targets {
			if err := ssz.Decode(data, obj); err != nil {
				if !wrapsSentinel(err) {
					t.Fatalf("%T: error %q wraps no named sentinel", obj, err)
				}
				continue
			}
			reencoded, err := ssz.Encode(obj)
			if err != nil {
				t.Fatalf("%T: decoded value failed to re-encode: %v", obj, err)
			}
			if !bytes.Equal(reencoded, data) {
				t.Fatalf("%T: decode not strict:\n  in  %x\n  out %x", obj, data, reencoded)
			}
			obj.HashTreeRoot()
		}
	})
}

func wrapsSentinel(err error) bool {
	for _, s := range sentinels {
		if errors.Is(err, s) {
			return true
		}
	}
	return false
}
