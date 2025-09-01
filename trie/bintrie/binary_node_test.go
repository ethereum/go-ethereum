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
	// Create an internal node with two hashed children
	leftHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	rightHash := common.HexToHash("0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321")

	node := &InternalNode{
		depth: 5,
		left:  HashedNode(leftHash),
		right: HashedNode(rightHash),
	}

	// Serialize the node
	serialized := SerializeNode(node)

	// Check the serialized format
	if serialized[0] != nodeTypeInternal {
		t.Errorf("Expected type byte to be %d, got %d", nodeTypeInternal, serialized[0])
	}

	if len(serialized) != 65 {
		t.Errorf("Expected serialized length to be 65, got %d", len(serialized))
	}

	// Deserialize the node
	deserialized, err := DeserializeNode(serialized, 5)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Check that it's an internal node
	internalNode, ok := deserialized.(*InternalNode)
	if !ok {
		t.Fatalf("Expected InternalNode, got %T", deserialized)
	}

	// Check the depth
	if internalNode.depth != 5 {
		t.Errorf("Expected depth 5, got %d", internalNode.depth)
	}

	// Check the left and right hashes
	if internalNode.left.Hash() != leftHash {
		t.Errorf("Left hash mismatch: expected %x, got %x", leftHash, internalNode.left.Hash())
	}

	if internalNode.right.Hash() != rightHash {
		t.Errorf("Right hash mismatch: expected %x, got %x", rightHash, internalNode.right.Hash())
	}
}

// TestSerializeDeserializeStemNode tests serialization and deserialization of StemNode
func TestSerializeDeserializeStemNode(t *testing.T) {
	// Create a stem node with some values
	stem := make([]byte, 31)
	for i := range stem {
		stem[i] = byte(i)
	}

	var values [256][]byte
	// Add some values at different indices
	values[0] = common.HexToHash("0x0101010101010101010101010101010101010101010101010101010101010101").Bytes()
	values[10] = common.HexToHash("0x0202020202020202020202020202020202020202020202020202020202020202").Bytes()
	values[255] = common.HexToHash("0x0303030303030303030303030303030303030303030303030303030303030303").Bytes()

	node := &StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  10,
	}

	// Serialize the node
	serialized := SerializeNode(node)

	// Check the serialized format
	if serialized[0] != nodeTypeStem {
		t.Errorf("Expected type byte to be %d, got %d", nodeTypeStem, serialized[0])
	}

	// Check the stem is correctly serialized
	if !bytes.Equal(serialized[1:32], stem) {
		t.Errorf("Stem mismatch in serialized data")
	}

	// Deserialize the node
	deserialized, err := DeserializeNode(serialized, 10)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Check that it's a stem node
	stemNode, ok := deserialized.(*StemNode)
	if !ok {
		t.Fatalf("Expected StemNode, got %T", deserialized)
	}

	// Check the stem
	if !bytes.Equal(stemNode.Stem, stem) {
		t.Errorf("Stem mismatch after deserialization")
	}

	// Check the values
	if !bytes.Equal(stemNode.Values[0], values[0]) {
		t.Errorf("Value at index 0 mismatch")
	}
	if !bytes.Equal(stemNode.Values[10], values[10]) {
		t.Errorf("Value at index 10 mismatch")
	}
	if !bytes.Equal(stemNode.Values[255], values[255]) {
		t.Errorf("Value at index 255 mismatch")
	}

	// Check that other values are nil
	for i := range NodeWidth {
		if i == 0 || i == 10 || i == 255 {
			continue
		}
		if stemNode.Values[i] != nil {
			t.Errorf("Expected nil value at index %d, got %x", i, stemNode.Values[i])
		}
	}
}

// TestDeserializeEmptyNode tests deserialization of empty node
func TestDeserializeEmptyNode(t *testing.T) {
	// Empty byte slice should deserialize to Empty node
	deserialized, err := DeserializeNode([]byte{}, 0)
	if err != nil {
		t.Fatalf("Failed to deserialize empty node: %v", err)
	}

	_, ok := deserialized.(Empty)
	if !ok {
		t.Fatalf("Expected Empty node, got %T", deserialized)
	}
}

// TestDeserializeInvalidType tests deserialization with invalid type byte
func TestDeserializeInvalidType(t *testing.T) {
	// Create invalid serialized data with unknown type byte
	invalidData := []byte{99, 0, 0, 0} // Type byte 99 is invalid

	_, err := DeserializeNode(invalidData, 0)
	if err == nil {
		t.Fatal("Expected error for invalid type byte, got nil")
	}
}

// TestDeserializeInvalidLength tests deserialization with invalid data length
func TestDeserializeInvalidLength(t *testing.T) {
	// InternalNode with type byte 1 but wrong length
	invalidData := []byte{nodeTypeInternal, 0, 0} // Too short for internal node

	_, err := DeserializeNode(invalidData, 0)
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
			key:      []byte{0x80}, // 10000000 in binary
			expected: []byte{1},
			wantErr:  false,
		},
		{
			name:     "depth 7",
			depth:    7,
			key:      []byte{0xFF}, // 11111111 in binary
			expected: []byte{1, 1, 1, 1, 1, 1, 1, 1},
			wantErr:  false,
		},
		{
			name:     "depth crossing byte boundary",
			depth:    10,
			key:      []byte{0xFF, 0x00}, // 11111111 00000000 in binary
			expected: []byte{1, 1, 1, 1, 1, 1, 1, 1, 0, 0, 0},
			wantErr:  false,
		},
		{
			name:     "max valid depth",
			depth:    31 * 8,
			key:      make([]byte, 32),
			expected: make([]byte, 31*8+1),
			wantErr:  false,
		},
		{
			name:    "depth too large",
			depth:   31*8 + 1,
			key:     make([]byte, 32),
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
