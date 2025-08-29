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

// TestStemNodeInsertSameStem tests inserting values with the same stem
func TestStemNodeInsertSameStem(t *testing.T) {
	stem := make([]byte, 31)
	for i := range stem {
		stem[i] = byte(i)
	}

	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()

	node := &StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  0,
	}

	// Insert another value with the same stem but different last byte
	key := make([]byte, 32)
	copy(key[:31], stem)
	key[31] = 10
	value := common.HexToHash("0x0202").Bytes()

	newNode, err := node.Insert(key, value, nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Should still be a StemNode
	stemNode, ok := newNode.(*StemNode)
	if !ok {
		t.Fatalf("Expected StemNode, got %T", newNode)
	}

	// Check that both values are present
	if !bytes.Equal(stemNode.Values[0], values[0]) {
		t.Errorf("Value at index 0 mismatch")
	}
	if !bytes.Equal(stemNode.Values[10], value) {
		t.Errorf("Value at index 10 mismatch")
	}
}

// TestStemNodeInsertDifferentStem tests inserting values with different stems
func TestStemNodeInsertDifferentStem(t *testing.T) {
	stem1 := make([]byte, 31)
	for i := range stem1 {
		stem1[i] = 0x00
	}

	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()

	node := &StemNode{
		Stem:   stem1,
		Values: values[:],
		depth:  0,
	}

	// Insert with a different stem (first bit different)
	key := make([]byte, 32)
	key[0] = 0x80 // First bit is 1 instead of 0
	value := common.HexToHash("0x0202").Bytes()

	newNode, err := node.Insert(key, value, nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	// Should now be an InternalNode
	internalNode, ok := newNode.(*InternalNode)
	if !ok {
		t.Fatalf("Expected InternalNode, got %T", newNode)
	}

	// Check depth
	if internalNode.depth != 0 {
		t.Errorf("Expected depth 0, got %d", internalNode.depth)
	}

	// Original stem should be on the left (bit 0)
	leftStem, ok := internalNode.left.(*StemNode)
	if !ok {
		t.Fatalf("Expected left child to be StemNode, got %T", internalNode.left)
	}
	if !bytes.Equal(leftStem.Stem, stem1) {
		t.Errorf("Left stem mismatch")
	}

	// New stem should be on the right (bit 1)
	rightStem, ok := internalNode.right.(*StemNode)
	if !ok {
		t.Fatalf("Expected right child to be StemNode, got %T", internalNode.right)
	}
	if !bytes.Equal(rightStem.Stem, key[:31]) {
		t.Errorf("Right stem mismatch")
	}
}

// TestStemNodeInsertInvalidValueLength tests inserting value with invalid length
func TestStemNodeInsertInvalidValueLength(t *testing.T) {
	stem := make([]byte, 31)
	var values [256][]byte

	node := &StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  0,
	}

	// Try to insert value with wrong length
	key := make([]byte, 32)
	copy(key[:31], stem)
	invalidValue := []byte{1, 2, 3} // Not 32 bytes

	_, err := node.Insert(key, invalidValue, nil, 0)
	if err == nil {
		t.Fatal("Expected error for invalid value length")
	}

	if err.Error() != "invalid insertion: value length" {
		t.Errorf("Expected 'invalid insertion: value length' error, got: %v", err)
	}
}

// TestStemNodeCopy tests the Copy method
func TestStemNodeCopy(t *testing.T) {
	stem := make([]byte, 31)
	for i := range stem {
		stem[i] = byte(i)
	}

	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()
	values[255] = common.HexToHash("0x0202").Bytes()

	node := &StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  10,
	}

	// Create a copy
	copied := node.Copy()
	copiedStem, ok := copied.(*StemNode)
	if !ok {
		t.Fatalf("Expected StemNode, got %T", copied)
	}

	// Check that values are equal but not the same slice
	if !bytes.Equal(copiedStem.Stem, node.Stem) {
		t.Errorf("Stem mismatch after copy")
	}
	if &copiedStem.Stem[0] == &node.Stem[0] {
		t.Error("Stem slice not properly cloned")
	}

	// Check values
	if !bytes.Equal(copiedStem.Values[0], node.Values[0]) {
		t.Errorf("Value at index 0 mismatch after copy")
	}
	if !bytes.Equal(copiedStem.Values[255], node.Values[255]) {
		t.Errorf("Value at index 255 mismatch after copy")
	}

	// Check that value slices are cloned
	if copiedStem.Values[0] != nil && &copiedStem.Values[0][0] == &node.Values[0][0] {
		t.Error("Value slice not properly cloned")
	}

	// Check depth
	if copiedStem.depth != node.depth {
		t.Errorf("Depth mismatch: expected %d, got %d", node.depth, copiedStem.depth)
	}
}

// TestStemNodeHash tests the Hash method
func TestStemNodeHash(t *testing.T) {
	stem := make([]byte, 31)
	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()

	node := &StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  0,
	}

	hash1 := node.Hash()

	// Hash should be deterministic
	hash2 := node.Hash()
	if hash1 != hash2 {
		t.Errorf("Hash not deterministic: %x != %x", hash1, hash2)
	}

	// Changing a value should change the hash
	node.Values[1] = common.HexToHash("0x0202").Bytes()
	hash3 := node.Hash()
	if hash1 == hash3 {
		t.Error("Hash didn't change after modifying values")
	}
}

// TestStemNodeGetValuesAtStem tests GetValuesAtStem method
func TestStemNodeGetValuesAtStem(t *testing.T) {
	stem := make([]byte, 31)
	for i := range stem {
		stem[i] = byte(i)
	}

	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()
	values[10] = common.HexToHash("0x0202").Bytes()
	values[255] = common.HexToHash("0x0303").Bytes()

	node := &StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  0,
	}

	// GetValuesAtStem with matching stem
	retrievedValues, err := node.GetValuesAtStem(stem, nil)
	if err != nil {
		t.Fatalf("Failed to get values: %v", err)
	}

	// Check that all values match
	for i := 0; i < 256; i++ {
		if !bytes.Equal(retrievedValues[i], values[i]) {
			t.Errorf("Value mismatch at index %d", i)
		}
	}

	// GetValuesAtStem with different stem also returns the same values
	// (implementation ignores the stem parameter)
	differentStem := make([]byte, 31)
	differentStem[0] = 0xFF

	retrievedValues2, err := node.GetValuesAtStem(differentStem, nil)
	if err != nil {
		t.Fatalf("Failed to get values with different stem: %v", err)
	}

	// Should still return the same values (stem is ignored)
	for i := 0; i < 256; i++ {
		if !bytes.Equal(retrievedValues2[i], values[i]) {
			t.Errorf("Value mismatch at index %d with different stem", i)
		}
	}
}

// TestStemNodeInsertValuesAtStem tests InsertValuesAtStem method
func TestStemNodeInsertValuesAtStem(t *testing.T) {
	stem := make([]byte, 31)
	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()

	node := &StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  0,
	}

	// Insert new values at the same stem
	var newValues [256][]byte
	newValues[1] = common.HexToHash("0x0202").Bytes()
	newValues[2] = common.HexToHash("0x0303").Bytes()

	newNode, err := node.InsertValuesAtStem(stem, newValues[:], nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert values: %v", err)
	}

	stemNode, ok := newNode.(*StemNode)
	if !ok {
		t.Fatalf("Expected StemNode, got %T", newNode)
	}

	// Check that all values are present
	if !bytes.Equal(stemNode.Values[0], values[0]) {
		t.Error("Original value at index 0 missing")
	}
	if !bytes.Equal(stemNode.Values[1], newValues[1]) {
		t.Error("New value at index 1 missing")
	}
	if !bytes.Equal(stemNode.Values[2], newValues[2]) {
		t.Error("New value at index 2 missing")
	}
}

// TestStemNodeGetHeight tests GetHeight method
func TestStemNodeGetHeight(t *testing.T) {
	node := &StemNode{
		Stem:   make([]byte, 31),
		Values: make([][]byte, 256),
		depth:  0,
	}

	height := node.GetHeight()
	if height != 1 {
		t.Errorf("Expected height 1, got %d", height)
	}
}

// TestStemNodeCollectNodes tests CollectNodes method
func TestStemNodeCollectNodes(t *testing.T) {
	stem := make([]byte, 31)
	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()

	node := &StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  0,
	}

	var collectedPaths [][]byte
	var collectedNodes []BinaryNode

	flushFn := func(path []byte, n BinaryNode) {
		// Make a copy of the path
		pathCopy := make([]byte, len(path))
		copy(pathCopy, path)
		collectedPaths = append(collectedPaths, pathCopy)
		collectedNodes = append(collectedNodes, n)
	}

	err := node.CollectNodes([]byte{0, 1, 0}, flushFn)
	if err != nil {
		t.Fatalf("Failed to collect nodes: %v", err)
	}

	// Should have collected one node (itself)
	if len(collectedNodes) != 1 {
		t.Errorf("Expected 1 collected node, got %d", len(collectedNodes))
	}

	// Check that the collected node is the same
	if collectedNodes[0] != node {
		t.Error("Collected node doesn't match original")
	}

	// Check the path
	if !bytes.Equal(collectedPaths[0], []byte{0, 1, 0}) {
		t.Errorf("Path mismatch: expected [0, 1, 0], got %v", collectedPaths[0])
	}
}
