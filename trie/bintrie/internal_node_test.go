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

// TestInternalNodeGet tests the Get method via NodeStore
func TestInternalNodeGet(t *testing.T) {
	s := NewNodeStore()

	leftStem := make([]byte, 31)
	rightStem := make([]byte, 31)
	rightStem[0] = 0x80

	var leftValues, rightValues [256][]byte
	leftValues[0] = common.HexToHash("0x0101").Bytes()
	rightValues[0] = common.HexToHash("0x0202").Bytes()

	left := s.allocStem(StemNode{Stem: leftStem, Values: leftValues[:], depth: 1})
	right := s.allocStem(StemNode{Stem: rightStem, Values: rightValues[:], depth: 1})
	ref := s.allocInternal(InternalNode{depth: 0, left: left, right: right})

	// Get value from left subtree
	leftKey := make([]byte, 32)
	leftKey[31] = 0
	value, err := s.Get(ref, leftKey, nil)
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
	value, err = s.Get(ref, rightKey, nil)
	if err != nil {
		t.Fatalf("Failed to get right value: %v", err)
	}
	if !bytes.Equal(value, rightValues[0]) {
		t.Errorf("Right value mismatch: expected %x, got %x", rightValues[0], value)
	}
}

// TestInternalNodeGetWithResolver tests Get with HashedNode resolution
func TestInternalNodeGetWithResolver(t *testing.T) {
	s := NewNodeStore()
	hashedRef := s.allocHashed(HashedNode{hash: common.HexToHash("0x1234")})
	ref := s.allocInternal(InternalNode{depth: 0, left: hashedRef, right: EmptyRef})

	resolver := func(path []byte, hash common.Hash) ([]byte, error) {
		if hash == common.HexToHash("0x1234") {
			stem := make([]byte, 31)
			var values [256][]byte
			values[5] = common.HexToHash("0xabcd").Bytes()
			tmpStore := NewNodeStore()
			tmpRef := tmpStore.allocStem(StemNode{Stem: stem, Values: values[:], depth: 1})
			return tmpStore.SerializeNode(tmpRef), nil
		}
		return nil, errors.New("node not found")
	}

	key := make([]byte, 32)
	key[31] = 5
	value, err := s.Get(ref, key, resolver)
	if err != nil {
		t.Fatalf("Failed to get value: %v", err)
	}

	expectedValue := common.HexToHash("0xabcd").Bytes()
	if !bytes.Equal(value, expectedValue) {
		t.Errorf("Value mismatch: expected %x, got %x", expectedValue, value)
	}
}

// TestInternalNodeInsert tests the Insert method via NodeStore
func TestInternalNodeInsert(t *testing.T) {
	s := NewNodeStore()
	ref := s.allocInternal(InternalNode{depth: 0, left: EmptyRef, right: EmptyRef})

	leftKey := make([]byte, 32)
	leftKey[31] = 10
	leftValue := common.HexToHash("0x0101").Bytes()

	newRef, err := s.Insert(ref, leftKey, leftValue, nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	if newRef.Kind() != KindInternal {
		t.Fatalf("Expected KindInternal, got %v", newRef.Kind())
	}

	n := s.getInternal(newRef.Index())
	if n.left.Kind() != KindStem {
		t.Fatalf("Expected left child to be KindStem, got %v", n.left.Kind())
	}

	leftStem := s.getStem(n.left.Index())
	if !bytes.Equal(leftStem.Values[10], leftValue) {
		t.Errorf("Value mismatch: expected %x, got %x", leftValue, leftStem.Values[10])
	}

	if !n.right.IsEmpty() {
		t.Errorf("Expected right child to remain empty")
	}
}

// TestInternalNodeCopy tests the Copy method
func TestInternalNodeCopy(t *testing.T) {
	s := NewNodeStore()

	leftStem := s.allocStem(StemNode{
		Stem:   make([]byte, 31),
		Values: make([][]byte, 256),
		depth:  1,
	})
	s.getStem(leftStem.Index()).Values[0] = common.HexToHash("0x0101").Bytes()

	rightStemBytes := make([]byte, 31)
	rightStemBytes[0] = 0x80
	rightStem := s.allocStem(StemNode{
		Stem:   rightStemBytes,
		Values: make([][]byte, 256),
		depth:  1,
	})
	s.getStem(rightStem.Index()).Values[0] = common.HexToHash("0x0202").Bytes()

	ref := s.allocInternal(InternalNode{depth: 0, left: leftStem, right: rightStem})

	// Create a copy
	ns := s.Copy()
	n := ns.getInternal(ref.Index())

	if n.depth != 0 {
		t.Errorf("Depth mismatch: expected 0, got %d", n.depth)
	}

	copiedLeft := ns.getStem(n.left.Index())
	copiedRight := ns.getStem(n.right.Index())

	if !bytes.Equal(copiedLeft.Values[0], common.HexToHash("0x0101").Bytes()) {
		t.Error("Left child value mismatch after copy")
	}
	if !bytes.Equal(copiedRight.Values[0], common.HexToHash("0x0202").Bytes()) {
		t.Error("Right child value mismatch after copy")
	}

	// Modify copied store - should not affect original
	copiedLeft.Values[0] = common.HexToHash("0x9999").Bytes()
	origLeft := s.getStem(leftStem.Index())
	if bytes.Equal(origLeft.Values[0], common.HexToHash("0x9999").Bytes()) {
		t.Error("Copy is not independent from original")
	}
}

// TestInternalNodeHash tests the ComputeHash method
func TestInternalNodeHash(t *testing.T) {
	s := NewNodeStore()
	left := s.allocHashed(HashedNode{hash: common.HexToHash("0x1111")})
	right := s.allocHashed(HashedNode{hash: common.HexToHash("0x2222")})
	ref := s.allocInternal(InternalNode{
		depth:         0,
		left:          left,
		right:         right,
		mustRecompute: true,
	})

	hash1 := s.ComputeHash(ref)
	hash2 := s.ComputeHash(ref)
	if hash1 != hash2 {
		t.Errorf("Hash not deterministic: %x != %x", hash1, hash2)
	}

	// Changing a child should change the hash
	n := s.getInternal(ref.Index())
	n.left = s.allocHashed(HashedNode{hash: common.HexToHash("0x3333")})
	n.mustRecompute = true
	hash3 := s.ComputeHash(ref)
	if hash1 == hash3 {
		t.Error("Hash didn't change after modifying left child")
	}
}

// TestInternalNodeGetValuesAtStem tests GetValuesAtStem
func TestInternalNodeGetValuesAtStem(t *testing.T) {
	s := NewNodeStore()

	leftStem := make([]byte, 31)
	rightStem := make([]byte, 31)
	rightStem[0] = 0x80

	var leftValues, rightValues [256][]byte
	leftValues[0] = common.HexToHash("0x0101").Bytes()
	leftValues[10] = common.HexToHash("0x0102").Bytes()
	rightValues[0] = common.HexToHash("0x0201").Bytes()
	rightValues[20] = common.HexToHash("0x0202").Bytes()

	left := s.allocStem(StemNode{Stem: leftStem, Values: leftValues[:], depth: 1})
	right := s.allocStem(StemNode{Stem: rightStem, Values: rightValues[:], depth: 1})
	ref := s.allocInternal(InternalNode{depth: 0, left: left, right: right})

	values, err := s.GetValuesAtStem(ref, leftStem, nil)
	if err != nil {
		t.Fatalf("Failed to get left values: %v", err)
	}
	if !bytes.Equal(values[0], leftValues[0]) {
		t.Error("Left value at index 0 mismatch")
	}
	if !bytes.Equal(values[10], leftValues[10]) {
		t.Error("Left value at index 10 mismatch")
	}

	values, err = s.GetValuesAtStem(ref, rightStem, nil)
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

// TestInternalNodeInsertValuesAtStem tests InsertValuesAtStem
func TestInternalNodeInsertValuesAtStem(t *testing.T) {
	s := NewNodeStore()
	ref := s.allocInternal(InternalNode{depth: 0, left: EmptyRef, right: EmptyRef})

	stem := make([]byte, 31)
	var values [256][]byte
	values[5] = common.HexToHash("0x0505").Bytes()
	values[10] = common.HexToHash("0x1010").Bytes()

	newRef, err := s.InsertValuesAtStem(ref, stem, values[:], nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert values: %v", err)
	}

	n := s.getInternal(newRef.Index())
	if n.left.Kind() != KindStem {
		t.Fatalf("Expected left child to be KindStem, got %v", n.left.Kind())
	}

	leftStem := s.getStem(n.left.Index())
	if !bytes.Equal(leftStem.Values[5], values[5]) {
		t.Error("Value at index 5 mismatch")
	}
	if !bytes.Equal(leftStem.Values[10], values[10]) {
		t.Error("Value at index 10 mismatch")
	}
}

// TestInternalNodeCollectNodes tests CollectNodes
func TestInternalNodeCollectNodes(t *testing.T) {
	s := NewNodeStore()
	left := s.allocStem(StemNode{Stem: make([]byte, 31), Values: make([][]byte, 256), depth: 1})
	rightStemBytes := make([]byte, 31)
	rightStemBytes[0] = 0x80
	right := s.allocStem(StemNode{Stem: rightStemBytes, Values: make([][]byte, 256), depth: 1})
	ref := s.allocInternal(InternalNode{depth: 0, left: left, right: right})

	var collectedPaths [][]byte
	var collectedRefs []NodeRef

	flushFn := func(path []byte, r NodeRef) {
		pathCopy := make([]byte, len(path))
		copy(pathCopy, path)
		collectedPaths = append(collectedPaths, pathCopy)
		collectedRefs = append(collectedRefs, r)
	}

	err := s.CollectNodes(ref, []byte{1}, flushFn)
	if err != nil {
		t.Fatalf("Failed to collect nodes: %v", err)
	}

	if len(collectedRefs) != 3 {
		t.Errorf("Expected 3 collected nodes, got %d", len(collectedRefs))
	}

	expectedPaths := [][]byte{
		{1, 0},
		{1, 1},
		{1},
	}

	for i, expectedPath := range expectedPaths {
		if !bytes.Equal(collectedPaths[i], expectedPath) {
			t.Errorf("Path %d mismatch: expected %v, got %v", i, expectedPath, collectedPaths[i])
		}
	}
}

// TestInternalNodeGetHeight tests GetHeight
func TestInternalNodeGetHeight(t *testing.T) {
	s := NewNodeStore()

	deepLeft := s.allocStem(StemNode{Stem: make([]byte, 31), Values: make([][]byte, 256), depth: 2})
	leftInternal := s.allocInternal(InternalNode{depth: 1, left: deepLeft, right: EmptyRef})
	rightStem := s.allocStem(StemNode{Stem: make([]byte, 31), Values: make([][]byte, 256), depth: 1})
	ref := s.allocInternal(InternalNode{depth: 0, left: leftInternal, right: rightStem})

	height := s.GetHeight(ref)
	if height != 3 {
		t.Errorf("Expected height 3, got %d", height)
	}
}

// TestInternalNodeDepthTooLarge tests handling of excessive depth
func TestInternalNodeDepthTooLarge(t *testing.T) {
	s := NewNodeStore()
	ref := s.allocInternal(InternalNode{
		depth: 31*8 + 1,
		left:  EmptyRef,
		right: EmptyRef,
	})

	stem := make([]byte, 31)
	_, err := s.GetValuesAtStem(ref, stem, nil)
	if err == nil {
		t.Fatal("Expected error for excessive depth")
	}
	if err.Error() != "node too deep" {
		t.Errorf("Expected 'node too deep' error, got: %v", err)
	}
}
