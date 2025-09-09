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

// TestInternalNodeGet tests the Get method
func TestInternalNodeGet(t *testing.T) {
	// Create a simple tree structure
	leftStem := make([]byte, 31)
	rightStem := make([]byte, 31)
	rightStem[0] = 0x80 // First bit is 1

	var leftValues, rightValues [256][]byte
	leftValues[0] = common.HexToHash("0x0101").Bytes()
	rightValues[0] = common.HexToHash("0x0202").Bytes()

	node := &InternalNode{
		depth: 0,
		left: &StemNode{
			Stem:   leftStem,
			Values: leftValues[:],
			depth:  1,
		},
		right: &StemNode{
			Stem:   rightStem,
			Values: rightValues[:],
			depth:  1,
		},
	}

	// Get value from left subtree
	leftKey := make([]byte, 32)
	leftKey[31] = 0
	value, err := node.Get(leftKey, nil)
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
	value, err = node.Get(rightKey, nil)
	if err != nil {
		t.Fatalf("Failed to get right value: %v", err)
	}
	if !bytes.Equal(value, rightValues[0]) {
		t.Errorf("Right value mismatch: expected %x, got %x", rightValues[0], value)
	}
}

// TestInternalNodeGetWithResolver tests Get with HashedNode resolution
func TestInternalNodeGetWithResolver(t *testing.T) {
	// Create an internal node with a hashed child
	hashedChild := HashedNode(common.HexToHash("0x1234"))

	node := &InternalNode{
		depth: 0,
		left:  hashedChild,
		right: Empty{},
	}

	// Mock resolver that returns a stem node
	resolver := func(path []byte, hash common.Hash) ([]byte, error) {
		if hash == common.Hash(hashedChild) {
			stem := make([]byte, 31)
			var values [256][]byte
			values[5] = common.HexToHash("0xabcd").Bytes()
			stemNode := &StemNode{
				Stem:   stem,
				Values: values[:],
				depth:  1,
			}
			return SerializeNode(stemNode), nil
		}
		return nil, errors.New("node not found")
	}

	// Get value through the hashed node
	key := make([]byte, 32)
	key[31] = 5
	value, err := node.Get(key, resolver)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	expectedValue := common.HexToHash("0xabcd").Bytes()
	if !bytes.Equal(value, expectedValue) {
		t.Errorf("Value mismatch: expected %x, got %x", expectedValue, value)
	}
}

// TestInternalNodeInsert tests the Insert method
func TestInternalNodeInsert(t *testing.T) {
	// Start with an internal node with empty children
	node := &InternalNode{
		depth: 0,
		left:  Empty{},
		right: Empty{},
	}

	// Insert a value into the left subtree
	leftKey := make([]byte, 32)
	leftKey[31] = 10
	leftValue := common.HexToHash("0x0101").Bytes()

	newNode, err := node.Insert(leftKey, leftValue, nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	internalNode, ok := newNode.(*InternalNode)
	if !ok {
		t.Fatalf("Expected InternalNode, got %T", newNode)
	}

	// Check that left child is now a StemNode
	leftStem, ok := internalNode.left.(*StemNode)
	if !ok {
		t.Fatalf("Expected left child to be StemNode, got %T", internalNode.left)
	}

	// Check the inserted value
	if !bytes.Equal(leftStem.Values[10], leftValue) {
		t.Errorf("Value mismatch: expected %x, got %x", leftValue, leftStem.Values[10])
	}

	// Right child should still be Empty
	_, ok = internalNode.right.(Empty)
	if !ok {
		t.Errorf("Expected right child to remain Empty, got %T", internalNode.right)
	}
}

// TestInternalNodeCopy tests the Copy method
func TestInternalNodeCopy(t *testing.T) {
	// Create an internal node with stem children
	leftStem := &StemNode{
		Stem:   make([]byte, 31),
		Values: make([][]byte, 256),
		depth:  1,
	}
	leftStem.Values[0] = common.HexToHash("0x0101").Bytes()

	rightStem := &StemNode{
		Stem:   make([]byte, 31),
		Values: make([][]byte, 256),
		depth:  1,
	}
	rightStem.Stem[0] = 0x80
	rightStem.Values[0] = common.HexToHash("0x0202").Bytes()

	node := &InternalNode{
		depth: 0,
		left:  leftStem,
		right: rightStem,
	}

	// Create a copy
	copied := node.Copy()
	copiedInternal, ok := copied.(*InternalNode)
	if !ok {
		t.Fatalf("Expected InternalNode, got %T", copied)
	}

	// Check depth
	if copiedInternal.depth != node.depth {
		t.Errorf("Depth mismatch: expected %d, got %d", node.depth, copiedInternal.depth)
	}

	// Check that children are copied
	copiedLeft, ok := copiedInternal.left.(*StemNode)
	if !ok {
		t.Fatalf("Expected left child to be StemNode, got %T", copiedInternal.left)
	}

	copiedRight, ok := copiedInternal.right.(*StemNode)
	if !ok {
		t.Fatalf("Expected right child to be StemNode, got %T", copiedInternal.right)
	}

	// Verify deep copy (children should be different objects)
	if copiedLeft == leftStem {
		t.Error("Left child not properly copied")
	}
	if copiedRight == rightStem {
		t.Error("Right child not properly copied")
	}

	// But values should be equal
	if !bytes.Equal(copiedLeft.Values[0], leftStem.Values[0]) {
		t.Error("Left child value mismatch after copy")
	}
	if !bytes.Equal(copiedRight.Values[0], rightStem.Values[0]) {
		t.Error("Right child value mismatch after copy")
	}
}

// TestInternalNodeHash tests the Hash method
func TestInternalNodeHash(t *testing.T) {
	// Create an internal node
	node := &InternalNode{
		depth: 0,
		left:  HashedNode(common.HexToHash("0x1111")),
		right: HashedNode(common.HexToHash("0x2222")),
	}

	hash1 := node.Hash()

	// Hash should be deterministic
	hash2 := node.Hash()
	if hash1 != hash2 {
		t.Errorf("Hash not deterministic: %x != %x", hash1, hash2)
	}

	// Changing a child should change the hash
	node.left = HashedNode(common.HexToHash("0x3333"))
	hash3 := node.Hash()
	if hash1 == hash3 {
		t.Error("Hash didn't change after modifying left child")
	}

	// Test with nil children (should use zero hash)
	nodeWithNil := &InternalNode{
		depth: 0,
		left:  nil,
		right: HashedNode(common.HexToHash("0x4444")),
	}
	hashWithNil := nodeWithNil.Hash()
	if hashWithNil == (common.Hash{}) {
		t.Error("Hash shouldn't be zero even with nil child")
	}
}

// TestInternalNodeGetValuesAtStem tests GetValuesAtStem method
func TestInternalNodeGetValuesAtStem(t *testing.T) {
	// Create a tree with values at different stems
	leftStem := make([]byte, 31)
	rightStem := make([]byte, 31)
	rightStem[0] = 0x80

	var leftValues, rightValues [256][]byte
	leftValues[0] = common.HexToHash("0x0101").Bytes()
	leftValues[10] = common.HexToHash("0x0102").Bytes()
	rightValues[0] = common.HexToHash("0x0201").Bytes()
	rightValues[20] = common.HexToHash("0x0202").Bytes()

	node := &InternalNode{
		depth: 0,
		left: &StemNode{
			Stem:   leftStem,
			Values: leftValues[:],
			depth:  1,
		},
		right: &StemNode{
			Stem:   rightStem,
			Values: rightValues[:],
			depth:  1,
		},
	}

	// Get values from left stem
	values, err := node.GetValuesAtStem(leftStem, nil)
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
	values, err = node.GetValuesAtStem(rightStem, nil)
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

// TestInternalNodeInsertValuesAtStem tests InsertValuesAtStem method
func TestInternalNodeInsertValuesAtStem(t *testing.T) {
	// Start with an internal node with empty children
	node := &InternalNode{
		depth: 0,
		left:  Empty{},
		right: Empty{},
	}

	// Insert values at a stem in the left subtree
	stem := make([]byte, 31)
	var values [256][]byte
	values[5] = common.HexToHash("0x0505").Bytes()
	values[10] = common.HexToHash("0x1010").Bytes()

	newNode, err := node.InsertValuesAtStem(stem, values[:], nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert values: %v", err)
	}

	internalNode, ok := newNode.(*InternalNode)
	if !ok {
		t.Fatalf("Expected InternalNode, got %T", newNode)
	}

	// Check that left child is now a StemNode with the values
	leftStem, ok := internalNode.left.(*StemNode)
	if !ok {
		t.Fatalf("Expected left child to be StemNode, got %T", internalNode.left)
	}

	if !bytes.Equal(leftStem.Values[5], values[5]) {
		t.Error("Value at index 5 mismatch")
	}
	if !bytes.Equal(leftStem.Values[10], values[10]) {
		t.Error("Value at index 10 mismatch")
	}
}

// TestInternalNodeCollectNodes tests CollectNodes method
func TestInternalNodeCollectNodes(t *testing.T) {
	// Create an internal node with two stem children
	leftStem := &StemNode{
		Stem:   make([]byte, 31),
		Values: make([][]byte, 256),
		depth:  1,
	}

	rightStem := &StemNode{
		Stem:   make([]byte, 31),
		Values: make([][]byte, 256),
		depth:  1,
	}
	rightStem.Stem[0] = 0x80

	node := &InternalNode{
		depth: 0,
		left:  leftStem,
		right: rightStem,
	}

	var collectedPaths [][]byte
	var collectedNodes []BinaryNode

	flushFn := func(path []byte, n BinaryNode) {
		pathCopy := make([]byte, len(path))
		copy(pathCopy, path)
		collectedPaths = append(collectedPaths, pathCopy)
		collectedNodes = append(collectedNodes, n)
	}

	err := node.CollectNodes([]byte{1}, flushFn)
	if err != nil {
		t.Fatalf("Failed to collect nodes: %v", err)
	}

	// Should have collected 3 nodes: left stem, right stem, and the internal node itself
	if len(collectedNodes) != 3 {
		t.Errorf("Expected 3 collected nodes, got %d", len(collectedNodes))
	}

	// Check paths
	expectedPaths := [][]byte{
		{1, 0}, // left child
		{1, 1}, // right child
		{1},    // internal node itself
	}

	for i, expectedPath := range expectedPaths {
		if !bytes.Equal(collectedPaths[i], expectedPath) {
			t.Errorf("Path %d mismatch: expected %v, got %v", i, expectedPath, collectedPaths[i])
		}
	}
}

// TestInternalNodeGetHeight tests GetHeight method
func TestInternalNodeGetHeight(t *testing.T) {
	// Create a tree with different heights
	// Left subtree: depth 2 (internal -> stem)
	// Right subtree: depth 1 (stem)
	leftInternal := &InternalNode{
		depth: 1,
		left: &StemNode{
			Stem:   make([]byte, 31),
			Values: make([][]byte, 256),
			depth:  2,
		},
		right: Empty{},
	}

	rightStem := &StemNode{
		Stem:   make([]byte, 31),
		Values: make([][]byte, 256),
		depth:  1,
	}

	node := &InternalNode{
		depth: 0,
		left:  leftInternal,
		right: rightStem,
	}

	height := node.GetHeight()
	// Height should be max(left height, right height) + 1
	// Left height: 2, Right height: 1, so total: 3
	if height != 3 {
		t.Errorf("Expected height 3, got %d", height)
	}
}

// TestInternalNodeDepthTooLarge tests handling of excessive depth
func TestInternalNodeDepthTooLarge(t *testing.T) {
	// Create an internal node at max depth
	node := &InternalNode{
		depth: 31*8 + 1,
		left:  Empty{},
		right: Empty{},
	}

	stem := make([]byte, 31)
	_, err := node.GetValuesAtStem(stem, nil)
	if err == nil {
		t.Fatal("Expected error for excessive depth")
	}
	if err.Error() != "node too deep" {
		t.Errorf("Expected 'node too deep' error, got: %v", err)
	}
}
