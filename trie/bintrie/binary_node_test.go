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
// with the grouped subtree format through NodeStore.
func TestSerializeDeserializeInternalNode(t *testing.T) {
	leftHash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	rightHash := common.HexToHash("0xfedcba0987654321fedcba0987654321fedcba0987654321fedcba0987654321")

	s := NewNodeStore()
	leftRef := s.newHashedRef(leftHash)
	rightRef := s.newHashedRef(rightHash)

	rootRef := s.newInternalRef(0)
	rootNode := s.getInternal(rootRef.Index())
	rootNode.left = leftRef
	rootNode.right = rightRef
	s.SetRoot(rootRef)

	// Serialize the node with default group depth of 8
	serialized := s.SerializeNode(rootRef, MaxGroupDepth)

	// Check the serialized format
	if serialized[0] != nodeTypeInternal {
		t.Errorf("Expected type byte to be %d, got %d", nodeTypeInternal, serialized[0])
	}
	if serialized[1] != MaxGroupDepth {
		t.Errorf("Expected group depth to be %d, got %d", MaxGroupDepth, serialized[1])
	}

	bitmapSize := BitmapSizeForDepth(MaxGroupDepth)
	expectedLen := 1 + 1 + bitmapSize + 2*HashSize
	if len(serialized) != expectedLen {
		t.Errorf("Expected serialized length to be %d, got %d", expectedLen, len(serialized))
	}

	// Check bitmap bits
	bitmap := serialized[2 : 2+bitmapSize]
	if bitmap[0]&0x80 == 0 {
		t.Error("Expected bit 0 to be set in bitmap (left child)")
	}
	if bitmap[16]&0x80 == 0 {
		t.Error("Expected bit 128 to be set in bitmap (right child)")
	}

	// Deserialize into a new store
	ds := NewNodeStore()
	deserialized, err := ds.DeserializeNode(serialized, 0)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	// Root should be an InternalNode
	if deserialized.Kind() != KindInternal {
		t.Fatalf("Expected KindInternal, got kind %d", deserialized.Kind())
	}

	internalNode := ds.getInternal(deserialized.Index())
	if internalNode.depth != 0 {
		t.Errorf("Expected depth 0, got %d", internalNode.depth)
	}

	// Navigate to position 0 (8 left turns) to find the left hash
	node0 := navigateToLeafRef(ds, deserialized, 0, 8)
	if ds.ComputeHash(node0) != leftHash {
		t.Errorf("Left hash mismatch: expected %x, got %x", leftHash, ds.ComputeHash(node0))
	}

	// Navigate to position 128 (right, then 7 lefts) to find the right hash
	node128 := navigateToLeafRef(ds, deserialized, 128, 8)
	if ds.ComputeHash(node128) != rightHash {
		t.Errorf("Right hash mismatch: expected %x, got %x", rightHash, ds.ComputeHash(node128))
	}
}

// navigateToLeafRef navigates to a specific position in the tree using NodeStore.
func navigateToLeafRef(s *NodeStore, ref NodeRef, position, depth int) NodeRef {
	cur := ref
	for d := 0; d < depth; d++ {
		if cur.Kind() != KindInternal {
			return cur
		}
		in := s.getInternal(cur.Index())
		bit := (position >> (depth - 1 - d)) & 1
		if bit == 0 {
			cur = in.left
		} else {
			cur = in.right
		}
	}
	return cur
}

// TestSerializeDeserializeStemNode tests serialization and deserialization of StemNode through NodeStore.
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
	ref := s.newStemRef(stem, 10)
	sn := s.getStem(ref.Index())
	for i, v := range values {
		if v != nil {
			sn.setValue(byte(i), v)
		}
	}

	// Serialize the node
	serialized := s.SerializeNode(ref, MaxGroupDepth)

	// Check the serialized format
	if serialized[0] != nodeTypeStem {
		t.Errorf("Expected type byte to be %d, got %d", nodeTypeStem, serialized[0])
	}

	// Check the stem is correctly serialized
	if !bytes.Equal(serialized[1:1+StemSize], stem) {
		t.Errorf("Stem mismatch in serialized data")
	}

	// Deserialize into a new store
	ds := NewNodeStore()
	deserializedRef, err := ds.DeserializeNode(serialized, 10)
	if err != nil {
		t.Fatalf("Failed to deserialize node: %v", err)
	}

	if deserializedRef.Kind() != KindStem {
		t.Fatalf("Expected KindStem, got kind %d", deserializedRef.Kind())
	}

	stemNode := ds.getStem(deserializedRef.Index())

	// Check the stem
	if !bytes.Equal(stemNode.Stem[:], stem) {
		t.Errorf("Stem mismatch after deserialization")
	}

	// Check the values
	if !bytes.Equal(stemNode.getValue(0), values[0]) {
		t.Errorf("Value at index 0 mismatch")
	}
	if !bytes.Equal(stemNode.getValue(10), values[10]) {
		t.Errorf("Value at index 10 mismatch")
	}
	if !bytes.Equal(stemNode.getValue(255), values[255]) {
		t.Errorf("Value at index 255 mismatch")
	}

	// Check that other values are nil
	for i := range StemNodeWidth {
		if i == 0 || i == 10 || i == 255 {
			continue
		}
		if stemNode.hasValue(byte(i)) {
			t.Errorf("Expected no value at index %d, got %x", i, stemNode.getValue(byte(i)))
		}
	}
}

// TestDeserializeEmptyNode tests deserialization of empty node.
func TestDeserializeEmptyNode(t *testing.T) {
	s := NewNodeStore()
	deserialized, err := s.DeserializeNode([]byte{}, 0)
	if err != nil {
		t.Fatalf("Failed to deserialize empty node: %v", err)
	}

	if !deserialized.IsEmpty() {
		t.Fatalf("Expected EmptyRef, got kind %d", deserialized.Kind())
	}
}

// TestDeserializeInvalidType tests deserialization with invalid type byte.
func TestDeserializeInvalidType(t *testing.T) {
	s := NewNodeStore()
	invalidData := []byte{99, 0, 0, 0} // Type byte 99 is invalid

	_, err := s.DeserializeNode(invalidData, 0)
	if err == nil {
		t.Fatal("Expected error for invalid type byte, got nil")
	}
}

// TestDeserializeInvalidLength tests deserialization with invalid data length.
func TestDeserializeInvalidLength(t *testing.T) {
	s := NewNodeStore()
	// InternalNode with valid type byte and group depth but too short for bitmap
	invalidData := []byte{nodeTypeInternal, 8, 0, 0} // Too short for bitmap (needs 32 bytes)

	_, err := s.DeserializeNode(invalidData, 0)
	if err == nil {
		t.Fatal("Expected error for invalid data length, got nil")
	}

	if err.Error() != "invalid serialized node length" {
		t.Errorf("Expected 'invalid serialized node length' error, got: %v", err)
	}
}

// TestKeyToPath tests the keyToPath function.
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
			depth:    StemSize*8 - 1,
			key:      make([]byte, HashSize),
			expected: make([]byte, StemSize*8),
			wantErr:  false,
		},
		{
			name:    "depth too large",
			depth:   StemSize * 8,
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
