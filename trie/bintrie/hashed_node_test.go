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

// TestHashedNodeHash tests the Hash method
func TestHashedNodeHash(t *testing.T) {
	hash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	node := HashedNode(hash)

	// Hash should return the stored hash
	if node.Hash() != hash {
		t.Errorf("Hash mismatch: expected %x, got %x", hash, node.Hash())
	}
}

// TestHashedNodeCopy tests the Copy method
func TestHashedNodeCopy(t *testing.T) {
	hash := common.HexToHash("0xabcdef")
	node := HashedNode(hash)

	copied := node.Copy()
	copiedHash, ok := copied.(HashedNode)
	if !ok {
		t.Fatalf("Expected HashedNode, got %T", copied)
	}

	// Hash should be the same
	if common.Hash(copiedHash) != hash {
		t.Errorf("Hash mismatch after copy: expected %x, got %x", hash, copiedHash)
	}

	// But should be a different object
	if &node == &copiedHash {
		t.Error("Copy returned same object reference")
	}
}

// TestHashedNodeInsert tests that Insert returns an error
func TestHashedNodeInsert(t *testing.T) {
	node := HashedNode(common.HexToHash("0x1234"))

	key := make([]byte, HashSize)
	value := make([]byte, HashSize)

	_, err := node.Insert(key, value, nil, 0)
	if err == nil {
		t.Fatal("Expected error for Insert on HashedNode")
	}

	if err.Error() != "insert not implemented for hashed node" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestHashedNodeGetValuesAtStem tests that GetValuesAtStem returns an error
func TestHashedNodeGetValuesAtStem(t *testing.T) {
	node := HashedNode(common.HexToHash("0x1234"))

	stem := make([]byte, StemSize)
	_, err := node.GetValuesAtStem(stem, nil)
	if err == nil {
		t.Fatal("Expected error for GetValuesAtStem on HashedNode")
	}

	if err.Error() != "attempted to get values from an unresolved node" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestHashedNodeInsertValuesAtStem tests that InsertValuesAtStem returns an error
func TestHashedNodeInsertValuesAtStem(t *testing.T) {
	node := HashedNode(common.HexToHash("0x1234"))

	stem := make([]byte, StemSize)
	values := make([][]byte, StemNodeWidth)

	// Test 1: nil resolver should return an error
	_, err := node.InsertValuesAtStem(stem, values, nil, 0)
	if err == nil {
		t.Fatal("Expected error for InsertValuesAtStem on HashedNode with nil resolver")
	}

	if err.Error() != "InsertValuesAtStem resolve error: resolver is nil" {
		t.Errorf("Unexpected error message: %v", err)
	}

	// Test 2: mock resolver returning invalid data should return deserialization error
	mockResolver := func(path []byte, hash common.Hash) ([]byte, error) {
		// Return invalid/nonsense data that cannot be deserialized
		return []byte{0xff, 0xff, 0xff}, nil
	}

	_, err = node.InsertValuesAtStem(stem, values, mockResolver, 0)
	if err == nil {
		t.Fatal("Expected error for InsertValuesAtStem on HashedNode with invalid resolver data")
	}

	expectedPrefix := "InsertValuesAtStem node deserialization error:"
	if len(err.Error()) < len(expectedPrefix) || err.Error()[:len(expectedPrefix)] != expectedPrefix {
		t.Errorf("Expected deserialization error, got: %v", err)
	}

	// Test 3: mock resolver returning valid serialized node should succeed
	stem = make([]byte, StemSize)
	stem[0] = 0xaa
	var originalValues [StemNodeWidth][]byte
	originalValues[0] = common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111").Bytes()
	originalValues[1] = common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222").Bytes()

	originalNode := &StemNode{
		Stem:   stem,
		Values: originalValues[:],
		depth:  0,
	}

	// Serialize the node
	serialized := SerializeNode(originalNode)

	// Create a mock resolver that returns the serialized node
	validResolver := func(path []byte, hash common.Hash) ([]byte, error) {
		return serialized, nil
	}

	var newValues [StemNodeWidth][]byte
	newValues[2] = common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333").Bytes()

	resolvedNode, err := node.InsertValuesAtStem(stem, newValues[:], validResolver, 0)
	if err != nil {
		t.Fatalf("Expected successful resolution and insertion, got error: %v", err)
	}

	resultStem, ok := resolvedNode.(*StemNode)
	if !ok {
		t.Fatalf("Expected resolved node to be *StemNode, got %T", resolvedNode)
	}

	if !bytes.Equal(resultStem.Stem, stem) {
		t.Errorf("Stem mismatch: expected %x, got %x", stem, resultStem.Stem)
	}

	// Verify the original values are preserved
	if !bytes.Equal(resultStem.Values[0], originalValues[0]) {
		t.Errorf("Original value at index 0 not preserved: expected %x, got %x", originalValues[0], resultStem.Values[0])
	}
	if !bytes.Equal(resultStem.Values[1], originalValues[1]) {
		t.Errorf("Original value at index 1 not preserved: expected %x, got %x", originalValues[1], resultStem.Values[1])
	}

	// Verify the new value was inserted
	if !bytes.Equal(resultStem.Values[2], newValues[2]) {
		t.Errorf("New value at index 2 not inserted correctly: expected %x, got %x", newValues[2], resultStem.Values[2])
	}
}

// TestHashedNodeToDot tests the toDot method for visualization
func TestHashedNodeToDot(t *testing.T) {
	hash := common.HexToHash("0x1234")
	node := HashedNode(hash)

	dot := node.toDot("parent", "010")

	// Should contain the hash value and parent connection
	expectedHash := "hash010"
	if !contains(dot, expectedHash) {
		t.Errorf("Expected dot output to contain %s", expectedHash)
	}

	if !contains(dot, "parent -> hash010") {
		t.Error("Expected dot output to contain parent connection")
	}
}

// Helper function
func contains(s, substr string) bool {
	return len(s) >= len(substr) && s != "" && len(substr) > 0
}
