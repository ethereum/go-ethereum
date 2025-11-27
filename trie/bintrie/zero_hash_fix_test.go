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
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

// TestZeroHashSkipsResolver tests that zero-hash HashedNodes don't trigger resolver calls
func TestZeroHashSkipsResolver(t *testing.T) {
	// Create an InternalNode with one real child and one Empty child
	realHash := common.HexToHash("0x1234")

	node := &InternalNode{
		depth: 0,
		left:  HashedNode(realHash),
		right: Empty{},
	}

	// Serialize and deserialize to create zero-hash HashedNode
	serialized := SerializeNode(node)
	deserialized, err := DeserializeNode(serialized, 0)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}

	deserializedInternal := deserialized.(*InternalNode)

	// Verify that right child is a zero-hash HashedNode after deserialization
	if hn, ok := deserializedInternal.right.(HashedNode); ok {
		if common.Hash(hn) != (common.Hash{}) {
			t.Fatal("Expected right child to be zero-hash HashedNode")
		}
	} else {
		t.Fatalf("Expected right child to be HashedNode, got %T", deserializedInternal.right)
	}

	// Track resolver calls
	resolverCalls := 0
	resolver := func(path []byte, hash common.Hash) ([]byte, error) {
		resolverCalls++

		// Zero-hash should never reach resolver
		if hash == (common.Hash{}) {
			t.Error("BUG: Resolver called for zero hash")
			return nil, errors.New("zero hash should not be resolved")
		}

		// Return valid data for real hash
		if hash == realHash {
			stem := make([]byte, 31)
			var values [256][]byte
			values[5] = common.HexToHash("0xabcd").Bytes()
			return SerializeNode(&StemNode{Stem: stem, Values: values[:], depth: 1}), nil
		}

		return nil, errors.New("not found")
	}

	// Access right child (zero-hash) - should not call resolver
	rightStem := make([]byte, 31)
	rightStem[0] = 0x80 // First bit is 1, routes to right child

	values, err := deserializedInternal.GetValuesAtStem(rightStem, resolver)
	if err != nil {
		t.Fatalf("GetValuesAtStem failed: %v", err)
	}

	// All values should be nil for empty node
	for i, v := range values {
		if v != nil {
			t.Errorf("Expected nil value at index %d, got %x", i, v)
		}
	}

	// Verify resolver was not called for zero-hash
	if resolverCalls > 0 {
		t.Errorf("Resolver should not have been called for zero-hash child, but was called %d times", resolverCalls)
	}

	// Now test left child (real hash) - should call resolver
	leftStem := make([]byte, 31)
	_, err = deserializedInternal.GetValuesAtStem(leftStem, resolver)
	if err != nil {
		t.Fatalf("GetValuesAtStem failed for left child: %v", err)
	}

	if resolverCalls != 1 {
		t.Errorf("Expected resolver to be called once for real hash, called %d times", resolverCalls)
	}
}

// TestZeroHashSkipsResolverOnInsert tests that InsertValuesAtStem also skips zero-hash resolver calls
func TestZeroHashSkipsResolverOnInsert(t *testing.T) {
	// Create node after deserialization with zero-hash children
	node := &InternalNode{
		depth: 0,
		left:  HashedNode(common.Hash{}), // Zero-hash
		right: HashedNode(common.Hash{}), // Zero-hash
	}

	resolverCalls := 0
	resolver := func(path []byte, hash common.Hash) ([]byte, error) {
		resolverCalls++

		if hash == (common.Hash{}) {
			t.Error("BUG: Resolver called for zero hash in InsertValuesAtStem")
			return nil, errors.New("zero hash should not be resolved")
		}

		return nil, errors.New("not found")
	}

	// Insert values into left subtree (zero-hash child)
	leftStem := make([]byte, 31)
	var values [256][]byte
	values[10] = common.HexToHash("0x5678").Bytes()

	_, err := node.InsertValuesAtStem(leftStem, values[:], resolver, 0)
	if err != nil {
		t.Fatalf("InsertValuesAtStem failed: %v", err)
	}

	// Verify resolver was not called
	if resolverCalls > 0 {
		t.Errorf("Resolver should not have been called for zero-hash child, but was called %d times", resolverCalls)
	}
}
