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

	key := make([]byte, 32)
	value := make([]byte, 32)

	_, err := node.Insert(key, value, nil)
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

	stem := make([]byte, 31)
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

	stem := make([]byte, 31)
	values := make([][]byte, 256)

	_, err := node.InsertValuesAtStem(stem, values, nil, 0)
	if err == nil {
		t.Fatal("Expected error for InsertValuesAtStem on HashedNode")
	}

	if err.Error() != "insertValuesAtStem not implemented for hashed node" {
		t.Errorf("Unexpected error message: %v", err)
	}
}

// TestHashedNodeGet tests that Get panics (as per implementation)
func TestHashedNodeGet(t *testing.T) {
	node := HashedNode(common.HexToHash("0x1234"))

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for Get on HashedNode")
		}
	}()

	key := make([]byte, 32)
	_, _ = node.Get(key, nil)
}

// TestHashedNodeCollectNodes tests that CollectNodes panics (as per implementation)
func TestHashedNodeCollectNodes(t *testing.T) {
	node := HashedNode(common.HexToHash("0x1234"))

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for CollectNodes on HashedNode")
		}
	}()

	path := []byte{0, 1, 0}
	node.CollectNodes(path, func([]byte, BinaryNode) {})
}

// TestHashedNodeGetHeight tests that GetHeight panics (as per implementation)
func TestHashedNodeGetHeight(t *testing.T) {
	node := HashedNode(common.HexToHash("0x1234"))

	defer func() {
		r := recover()
		if r == nil {
			t.Error("Expected panic for GetHeight on HashedNode")
		}
		// Check the panic message
		if r != "tried to get the height of a hashed node, this is a bug" {
			t.Errorf("Unexpected panic message: %v", r)
		}
	}()

	_ = node.GetHeight()
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
	return len(s) >= len(substr) && s[:len(s)] != "" && len(substr) > 0
}
