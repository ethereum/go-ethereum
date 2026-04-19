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
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestInternalNodeGet tests the Get method via nodeStore.
func TestInternalNodeGet(t *testing.T) {
	s := newNodeStore()

	leftStem := make([]byte, 31)
	rightStem := make([]byte, 31)
	rightStem[0] = 0x80

	leftValues := make([][]byte, 256)
	leftValues[0] = common.HexToHash("0x0101").Bytes()
	rightValues := make([][]byte, 256)
	rightValues[0] = common.HexToHash("0x0202").Bytes()

	// Build tree: root -> left stem, right stem
	// Insert left stem values
	s.root = emptyRef
	if err := s.InsertValuesAtStem(leftStem, leftValues, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertValuesAtStem(rightStem, rightValues, nil); err != nil {
		t.Fatal(err)
	}

	// Get value from left subtree
	leftKey := make([]byte, 32)
	leftKey[31] = 0
	value, err := s.Get(leftKey, nil)
	if err != nil {
		t.Fatalf("Failed to get left value: %v", err)
	}
	if !bytes.Equal(value, leftValues[0]) {
		t.Errorf("Left value mismatch: expected %x, got %x", leftValues[0], value)
	}

	// Get value from right subtree
	rightKey := make([]byte, 32)
	rightKey[0] = 0x80
	rightKey[31] = 0
	value, err = s.Get(rightKey, nil)
	if err != nil {
		t.Fatalf("Failed to get right value: %v", err)
	}
	if !bytes.Equal(value, rightValues[0]) {
		t.Errorf("Right value mismatch: expected %x, got %x", rightValues[0], value)
	}
}

// TestInternalNodeGetWithResolver tests Get with HashedNode resolution via nodeStore.
func TestInternalNodeGetWithResolver(t *testing.T) {
	// Create a store with an internal node containing a hashed child
	s := newNodeStore()
	hashedChild := s.newHashedRef(common.HexToHash("0x1234"))
	rootRef := s.newInternalRef(0)
	rootNode := s.getInternal(rootRef.Index())
	rootNode.left = hashedChild
	rootNode.right = emptyRef
	s.root = rootRef

	// Mock resolver that returns a stem node
	resolver := func(path []byte, hash common.Hash) ([]byte, error) {
		if hash == common.HexToHash("0x1234") {
			rs := newNodeStore()
			stem := make([]byte, 31)
			ref := rs.newStemRef(stem, 1)
			sn := rs.getStem(ref.Index())
			sn.setValue(5, common.HexToHash("0xabcd").Bytes())
			return rs.serializeNode(ref), nil
		}
		return nil, errors.New("node not found")
	}

	// Get value through the hashed node
	key := make([]byte, 32)
	key[31] = 5
	value, err := s.Get(key, resolver)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	expectedValue := common.HexToHash("0xabcd").Bytes()
	if !bytes.Equal(value, expectedValue) {
		t.Errorf("Value mismatch: expected %x, got %x", expectedValue, value)
	}
}

// TestInternalNodeInsert tests the Insert method via nodeStore.
func TestInternalNodeInsert(t *testing.T) {
	s := newNodeStore()

	leftKey := make([]byte, 32)
	leftKey[31] = 10
	leftValue := common.HexToHash("0x0101").Bytes()

	if err := s.Insert(leftKey, leftValue, nil); err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Verify the value was stored
	value, err := s.Get(leftKey, nil)
	if err != nil {
		t.Fatalf("Failed to get: %v", err)
	}
	if !bytes.Equal(value, leftValue) {
		t.Errorf("Value mismatch: expected %x, got %x", leftValue, value)
	}
}

// TestInternalNodeCopy tests the Copy method via nodeStore.
func TestInternalNodeCopy(t *testing.T) {
	s := newNodeStore()

	leftKey := make([]byte, 32)
	leftKey[31] = 0
	leftValue := common.HexToHash("0x0101").Bytes()

	rightKey := make([]byte, 32)
	rightKey[0] = 0x80
	rightKey[31] = 0
	rightValue := common.HexToHash("0x0202").Bytes()

	if err := s.Insert(leftKey, leftValue, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Insert(rightKey, rightValue, nil); err != nil {
		t.Fatal(err)
	}

	ns := s.Copy()

	// Values should be equal
	v1, _ := ns.Get(leftKey, nil)
	if !bytes.Equal(v1, leftValue) {
		t.Error("Left child value mismatch after copy")
	}
	v2, _ := ns.Get(rightKey, nil)
	if !bytes.Equal(v2, rightValue) {
		t.Error("Right child value mismatch after copy")
	}
}

// TestInternalNodeHash tests the Hash method via nodeStore.
func TestInternalNodeHash(t *testing.T) {
	s := newNodeStore()
	leftRef := s.newHashedRef(common.HexToHash("0x1111"))
	rightRef := s.newHashedRef(common.HexToHash("0x2222"))
	rootRef := s.newInternalRef(0)
	rootNode := s.getInternal(rootRef.Index())
	rootNode.left = leftRef
	rootNode.right = rightRef
	s.root = rootRef

	hash1 := s.computeHash(rootRef)

	// Hash should be deterministic
	hash2 := s.computeHash(rootRef)
	if hash1 != hash2 {
		t.Errorf("Hash not deterministic: %x != %x", hash1, hash2)
	}

	// Changing a child should change the hash
	rootNode.left = s.newHashedRef(common.HexToHash("0x3333"))
	rootNode.mustRecompute = true
	hash3 := s.computeHash(rootRef)
	if hash1 == hash3 {
		t.Error("Hash didn't change after modifying left child")
	}
}

// TestInternalNodeGetValuesAtStem tests GetValuesAtStem method via nodeStore.
func TestInternalNodeGetValuesAtStem(t *testing.T) {
	s := newNodeStore()

	leftStem := make([]byte, 31)
	rightStem := make([]byte, 31)
	rightStem[0] = 0x80

	leftValues := make([][]byte, 256)
	leftValues[0] = common.HexToHash("0x0101").Bytes()
	leftValues[10] = common.HexToHash("0x0102").Bytes()
	rightValues := make([][]byte, 256)
	rightValues[0] = common.HexToHash("0x0201").Bytes()
	rightValues[20] = common.HexToHash("0x0202").Bytes()

	if err := s.InsertValuesAtStem(leftStem, leftValues, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertValuesAtStem(rightStem, rightValues, nil); err != nil {
		t.Fatal(err)
	}

	// Get values from left stem
	values, err := s.GetValuesAtStem(leftStem, nil)
	if err != nil {
		t.Fatalf("Failed to get left values: %v", err)
	}
	if !bytes.Equal(values[0], leftValues[0]) {
		t.Error("Left value at index 0 mismatch")
	}
	if !bytes.Equal(values[10], leftValues[10]) {
		t.Error("Left value at index 10 mismatch")
	}

	// Get values from right stem
	values, err = s.GetValuesAtStem(rightStem, nil)
	if err != nil {
		t.Fatalf("Failed to get right values: %v", err)
	}
	if !bytes.Equal(values[0], rightValues[0]) {
		t.Error("Right value at index 0 mismatch")
	}
	if !bytes.Equal(values[20], rightValues[20]) {
		t.Error("Right value at index 20 mismatch")
	}
}

// TestInternalNodeInsertValuesAtStem tests InsertValuesAtStem method via nodeStore.
func TestInternalNodeInsertValuesAtStem(t *testing.T) {
	s := newNodeStore()

	stem := make([]byte, 31)
	values := make([][]byte, 256)
	values[5] = common.HexToHash("0x0505").Bytes()
	values[10] = common.HexToHash("0x1010").Bytes()

	if err := s.InsertValuesAtStem(stem, values, nil); err != nil {
		t.Fatalf("Failed to insert values: %v", err)
	}

	// Check that the values are stored
	retrieved, err := s.GetValuesAtStem(stem, nil)
	if err != nil {
		t.Fatalf("Failed to get values: %v", err)
	}
	if !bytes.Equal(retrieved[5], values[5]) {
		t.Error("Value at index 5 mismatch")
	}
	if !bytes.Equal(retrieved[10], values[10]) {
		t.Error("Value at index 10 mismatch")
	}
}

// TestInternalNodeCollectNodes tests CollectNodes method via nodeStore.
func TestInternalNodeCollectNodes(t *testing.T) {
	s := newNodeStore()

	leftStem := make([]byte, 31)
	rightStem := make([]byte, 31)
	rightStem[0] = 0x80

	leftValues := make([][]byte, 256)
	rightValues := make([][]byte, 256)

	if err := s.InsertValuesAtStem(leftStem, leftValues, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertValuesAtStem(rightStem, rightValues, nil); err != nil {
		t.Fatal(err)
	}

	var collectedPaths [][]byte
	flushFn := func(path []byte, hash common.Hash, serialized []byte) {
		pathCopy := make([]byte, len(path))
		copy(pathCopy, path)
		collectedPaths = append(collectedPaths, pathCopy)
	}

	err := s.collectNodes(s.root, []byte{1}, flushFn)
	if err != nil {
		t.Fatalf("Failed to collect nodes: %v", err)
	}

	// Should have collected 3 nodes: left stem, right stem, and the internal node itself
	if len(collectedPaths) != 3 {
		t.Errorf("Expected 3 collected nodes, got %d", len(collectedPaths))
	}
}

// TestInternalNodeGetHeight tests GetHeight method via nodeStore.
func TestInternalNodeGetHeight(t *testing.T) {
	s := newNodeStore()

	// Insert values that create a deeper tree
	stem1 := make([]byte, 31) // left
	stem2 := make([]byte, 31)
	stem2[0] = 0x40 // 01... -> goes left at depth 0, right at depth 1

	values1 := make([][]byte, 256)
	values1[0] = common.HexToHash("0x01").Bytes()
	values2 := make([][]byte, 256)
	values2[0] = common.HexToHash("0x02").Bytes()

	if err := s.InsertValuesAtStem(stem1, values1, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.InsertValuesAtStem(stem2, values2, nil); err != nil {
		t.Fatal(err)
	}

	height := s.getHeight(s.root)
	if height < 2 {
		t.Errorf("Expected height >= 2, got %d", height)
	}
}

// TestInternalNodeDepthTooLarge tests handling of excessive depth via nodeStore.
func TestInternalNodeDepthTooLarge(t *testing.T) {
	s := newNodeStore()
	// Creating an internal node beyond max depth should panic
	defer func() {
		if r := recover(); r == nil {
			t.Fatal("Expected panic for excessive depth")
		}
	}()
	s.newInternalRef(31*8 + 1)
}
