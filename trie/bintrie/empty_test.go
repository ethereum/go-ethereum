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

// TestEmptyGet tests the Get method
func TestEmptyGet(t *testing.T) {
	node := Empty{}

	key := make([]byte, 32)
	value, err := node.Get(key, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if value != nil {
		t.Errorf("Expected nil value from empty node, got %x", value)
	}
}

// TestEmptyInsert tests the Insert method
func TestEmptyInsert(t *testing.T) {
	node := Empty{}

	key := make([]byte, 32)
	key[0] = 0x12
	key[31] = 0x34
	value := common.HexToHash("0xabcd").Bytes()

	newNode, err := node.Insert(key, value, nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Should create a StemNode
	stemNode, ok := newNode.(*StemNode)
	if !ok {
		t.Fatalf("Expected StemNode, got %T", newNode)
	}

	// Check the stem (first 31 bytes of key)
	if !bytes.Equal(stemNode.Stem, key[:31]) {
		t.Errorf("Stem mismatch: expected %x, got %x", key[:31], stemNode.Stem)
	}

	// Check the value at the correct index (last byte of key)
	if !bytes.Equal(stemNode.Values[key[31]], value) {
		t.Errorf("Value mismatch at index %d: expected %x, got %x", key[31], value, stemNode.Values[key[31]])
	}

	// Check that other values are nil
	for i := 0; i < 256; i++ {
		if i != int(key[31]) && stemNode.Values[i] != nil {
			t.Errorf("Expected nil value at index %d, got %x", i, stemNode.Values[i])
		}
	}
}

// TestEmptyCopy tests the Copy method
func TestEmptyCopy(t *testing.T) {
	node := Empty{}

	copied := node.Copy()
	copiedEmpty, ok := copied.(Empty)
	if !ok {
		t.Fatalf("Expected Empty, got %T", copied)
	}

	// Both should be empty
	if node != copiedEmpty {
		// Empty is a zero-value struct, so copies should be equal
		t.Errorf("Empty nodes should be equal")
	}
}

// TestEmptyHash tests the Hash method
func TestEmptyHash(t *testing.T) {
	node := Empty{}

	hash := node.Hash()

	// Empty node should have zero hash
	if hash != (common.Hash{}) {
		t.Errorf("Expected zero hash for empty node, got %x", hash)
	}
}

// TestEmptyGetValuesAtStem tests the GetValuesAtStem method
func TestEmptyGetValuesAtStem(t *testing.T) {
	node := Empty{}

	stem := make([]byte, 31)
	values, err := node.GetValuesAtStem(stem, nil)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should return an array of 256 nil values
	if len(values) != 256 {
		t.Errorf("Expected 256 values, got %d", len(values))
	}

	for i, v := range values {
		if v != nil {
			t.Errorf("Expected nil value at index %d, got %x", i, v)
		}
	}
}

// TestEmptyInsertValuesAtStem tests the InsertValuesAtStem method
func TestEmptyInsertValuesAtStem(t *testing.T) {
	node := Empty{}

	stem := make([]byte, 31)
	stem[0] = 0x42

	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()
	values[10] = common.HexToHash("0x0202").Bytes()
	values[255] = common.HexToHash("0x0303").Bytes()

	newNode, err := node.InsertValuesAtStem(stem, values[:], nil, 5)
	if err != nil {
		t.Fatalf("Failed to insert values: %v", err)
	}

	// Should create a StemNode
	stemNode, ok := newNode.(*StemNode)
	if !ok {
		t.Fatalf("Expected StemNode, got %T", newNode)
	}

	// Check the stem
	if !bytes.Equal(stemNode.Stem, stem) {
		t.Errorf("Stem mismatch: expected %x, got %x", stem, stemNode.Stem)
	}

	// Check the depth
	if stemNode.depth != 5 {
		t.Errorf("Depth mismatch: expected 5, got %d", stemNode.depth)
	}

	// Check the values
	if !bytes.Equal(stemNode.Values[0], values[0]) {
		t.Error("Value at index 0 mismatch")
	}
	if !bytes.Equal(stemNode.Values[10], values[10]) {
		t.Error("Value at index 10 mismatch")
	}
	if !bytes.Equal(stemNode.Values[255], values[255]) {
		t.Error("Value at index 255 mismatch")
	}

	// Check that values is the same slice (not a copy)
	if &stemNode.Values[0] != &values[0] {
		t.Error("Expected values to be the same slice reference")
	}
}

// TestEmptyCollectNodes tests the CollectNodes method
func TestEmptyCollectNodes(t *testing.T) {
	node := Empty{}

	var collected []BinaryNode
	flushFn := func(path []byte, n BinaryNode) {
		collected = append(collected, n)
	}

	err := node.CollectNodes([]byte{0, 1, 0}, flushFn)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	// Should not collect anything for empty node
	if len(collected) != 0 {
		t.Errorf("Expected no collected nodes for empty, got %d", len(collected))
	}
}

// TestEmptyToDot tests the toDot method
func TestEmptyToDot(t *testing.T) {
	node := Empty{}

	dot := node.toDot("parent", "010")

	// Should return empty string for empty node
	if dot != "" {
		t.Errorf("Expected empty string for empty node toDot, got %s", dot)
	}
}

// TestEmptyGetHeight tests the GetHeight method
func TestEmptyGetHeight(t *testing.T) {
	node := Empty{}

	height := node.GetHeight()

	// Empty node should have height 0
	if height != 0 {
		t.Errorf("Expected height 0 for empty node, got %d", height)
	}
}
