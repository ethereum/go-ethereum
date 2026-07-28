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
	"testing"

	"github.com/ethereum/go-ethereum/crypto/blake3"
)

// FuzzTrieOps decodes the fuzz input as an operation stream over a small
// key universe and checks the engine against the rebuild model after every
// operation.
func FuzzTrieOps(f *testing.F) {
	f.Add([]byte{0x01, 0x02, 0x83, 0x44, 0x05, 0xc6, 0x07, 0x88, 0x09, 0x0a})
	f.Add([]byte{0xff, 0x00, 0xff, 0x00, 0xff, 0x00})
	f.Fuzz(func(t *testing.T, data []byte) {
		tr := newTestTrie()
		model := make(map[string][]byte)
		for i := 0; i+2 < len(data); i += 3 {
			entity, sub, val := data[i]&0x07, data[i+1], data[i+2]
			var key []byte
			seed := [1]byte{entity}
			switch entity & 0x03 {
			case 0:
				h := blake3.Sum256(seed[:])
				key = append(append([]byte{AccountZone}, h[:]...), sub)
			case 1:
				h := blake3.Sum256(append(seed[:], 0xC0))
				key = append(append([]byte{CodeZone}, h[:]...), sub)
			default:
				p := blake3.Sum256(seed[:])
				q := blake3.Sum256(append(seed[:], entity>>2))
				key = append([]byte{StorageZone}, p[:]...)
				key = append(key, q[:]...)
				key = append(key, sub)
			}
			if val == 0 {
				delete(model, string(key))
				deleteKey(t, tr, key)
			} else {
				value := make([]byte, 32)
				value[31] = val
				model[string(key)] = value
				setKey(t, tr, key, value)
			}
			if got, want := tr.Hash(), modelRoot(model); got != want {
				t.Fatalf("op %d: engine %x model %x", i/3, got, want)
			}
		}
	})
}

// FuzzNodeDecode throws arbitrary bytes at the record decoder: it must never
// panic, and anything that decodes must re-encode to the same hash.
func FuzzNodeDecode(f *testing.F) {
	f.Add([]byte{0x00})
	f.Add([]byte{0x01, 0x00, 0x03, 0xa0})
	f.Add([]byte{0x02, 0x00, 0x10, 0x21})
	f.Fuzz(func(t *testing.T, data []byte) {
		n, err := decodeNode(data)
		if err != nil {
			return
		}
		switch nn := n.(type) {
		case *groupNode:
			pos := nn.cachedAt
			blob := serializeNode(nn, pos)
			redec, err := decodeNode(blob)
			if err != nil {
				t.Fatalf("re-decode failed: %v", err)
			}
			if redec.hashAt(pos) != n.hashAt(pos) {
				t.Fatal("re-encoded group hash mismatch")
			}
		case *branchNode:
			blob := serializeNode(nn, 0)
			if _, err := decodeNode(blob); err != nil {
				t.Fatalf("re-decode failed: %v", err)
			}
		}
	})
}
