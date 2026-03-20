// Copyright 2025 go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestSerializeDeserializeInternalNode tests serialization and deserialization of InternalNode
func TestSerializeDeserializeInternalNode(t *testing.T) {
	leftHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	rightHash := common.HexToHash("0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321")

	s := NewNodeStore()
	left := s.allocHashed(HashedNode{hash: leftHash})
	right := s.allocHashed(HashedNode{hash: rightHash})
	ref := s.allocInternal(InternalNode{
		depth: 5,
		left:  left,
		right: right,
	})

	// Serialize the node
	serialized := s.SerializeNode(ref)

	// Check the serialized format
	if serialized[0] != nodeTypeInternal {
		t.Errorf("Expected type byte to be %d, got %d", nodeTypeInternal, serialized[0])
	}

	if len(serialized) != 65 {
		t.Errorf("Expected serialized length to be 65, got %d", len(serialized))
	}

	// Deserialize the node
	s2 := NewNodeStore()
	deserialized, err := s2.DeserializeNode(serialized, 5)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	if deserialized.Kind() != KindInternal {
		t.Fatalf("Expected KindInternal, got %v", deserialized.Kind())
	}

	n := s2.getInternal(deserialized.Index())
	if n.depth != 5 {
		t.Errorf("Expected depth 5, got %d", n.depth)
	}

	if s2.ComputeHash(n.left) != leftHash {
		t.Errorf("Left hash mismatch: expected %x, got %x", leftHash, s2.ComputeHash(n.left))
	}

	if s2.ComputeHash(n.right) != rightHash {
		t.Errorf("Right hash mismatch: expected %x, got %x", rightHash, s2.ComputeHash(n.right))
	}
}

// TestSerializeDeserializeStemNode tests serialization and deserialization of StemNode
func TestSerializeDeserializeStemNode(t *testing.T) {
	stem := make([]byte, StemSize)
	for i := range stem {
		stem[i] = byte(i)
	}

	var values [StemNodeWidth][]byte
	values[0] = common.HexToHash("0x0101010101010101010101010101010101010101010101010101010101010101").Bytes()
	values[10] = common.HexToHash("0x0202020202020202020202020202020202020202020202020202020202020202").Bytes()
	values[255] = common.HexToHash("0x0303030303030303030303030303030303030303030303030303030303030303").Bytes()

	s := NewNodeStore()
	ref := s.allocStem(StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  10,
	})

	serialized := s.SerializeNode(ref)

	if serialized[0] != nodeTypeStem {
		t.Errorf("Expected type byte to be %d, got %d", nodeTypeStem, serialized[0])
	}

	if !bytes.Equal(serialized[1:1+StemSize], stem) {
		t.Errorf("Stem mismatch in serialized data")
	}

	s2 := NewNodeStore()
	deserialized, err := s2.DeserializeNode(serialized, 10)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	if deserialized.Kind() != KindStem {
		t.Fatalf("Expected KindStem, got %v", deserialized.Kind())
	}

	sn := s2.getStem(deserialized.Index())
	if !bytes.Equal(sn.Stem, stem) {
		t.Errorf("Stem mismatch after deserialization")
	}

	if !bytes.Equal(sn.Values[0], values[0]) {
		t.Errorf("Value at index 0 mismatch")
	}
	if !bytes.Equal(sn.Values[10], values[10]) {
		t.Errorf("Value at index 10 mismatch")
	}
	if !bytes.Equal(sn.Values[255], values[255]) {
		t.Errorf("Value at index 255 mismatch")
	}

	for i := range StemNodeWidth {
		if i == 0 || i == 10 || i == 255 {
			continue
		}
		if sn.Values[i] != nil {
			t.Errorf("Expected nil value at index %d, got %x", i, sn.Values[i])
		}
	}
}

// TestDeserializeEmptyNode tests deserialization of empty node
func TestDeserializeEmptyNode(t *testing.T) {
	s := NewNodeStore()
	deserialized, err := s.DeserializeNode([]byte{}, 0)
	if err != nil {
		t.Fatalf("Failed to deserialize empty node: %v", err)
	}

	if deserialized.Kind() != KindEmpty {
		t.Fatalf("Expected KindEmpty, got %v", deserialized.Kind())
	}
}

// TestDeserializeInvalidType tests deserialization with invalid type byte
func TestDeserializeInvalidType(t *testing.T) {
	s := NewNodeStore()
	invalidData := []byte{99, 0, 0, 0}
	_, err := s.DeserializeNode(invalidData, 0)
	if err == nil {
		t.Fatal("Expected error for invalid type byte, got nil")
	}
}

// TestDeserializeInvalidLength tests deserialization with invalid data length
func TestDeserializeInvalidLength(t *testing.T) {
	s := NewNodeStore()
	invalidData := []byte{nodeTypeInternal, 0, 0}
	_, err := s.DeserializeNode(invalidData, 0)
	if err == nil {
		t.Fatal("Expected error for invalid data length, got nil")
	}

	if err.Error() != "invalid serialized node length" {
		t.Errorf("Expected 'invalid serialized node length' error, got: %v", err)
	}
}

// TestKeyToPath tests the keyToPath function
func TestKeyToPath(t *testing.T) {
	tests := []struct {
		name     string
		depth    int
		key      []byte
		expected []byte
		wantErr  bool
	}{
		{
			name:     "depth 0",
			depth:    0,
			key:      []byte{0x80},
			expected: []byte{1},
			wantErr:  false,
		},
		{
			name:     "depth 7",
			depth:    7,
			key:      []byte{0xFF},
			expected: []byte{1, 1, 1, 1, 1, 1, 1, 1},
			wantErr:  false,
		},
		{
			name:     "depth crossing byte boundary",
			depth:    10,
			key:      []byte{0xFF, 0x00},
			expected: []byte{1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0},
			wantErr:  false,
		},
		{
			name:     "max valid depth",
			depth:    StemSize * 8,
			key:      make([]byte, HashSize),
			expected: make([]byte, StemSize*8+1),
			wantErr:  false,
		},
		{
			name:    "depth too large",
			depth:   StemSize*8 + 1,
			key:     make([]byte, HashSize),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path, err := keyToPath(tt.depth, tt.key)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Expected error for depth %d, got nil", tt.depth)
				}
				return
			}
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}
			if !bytes.Equal(path, tt.expected) {
				t.Errorf("Path mismatch: expected %v, got %v", tt.expected, path)
			}
		})
	}
}
