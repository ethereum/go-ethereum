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

// TestHashedNodeHash tests the Hash method via nodeStore.
func TestHashedNodeHash(t *testing.T) {
	hash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	s := newNodeStore()
	ref := s.newHashedRef(hash)

	if s.computeHash(ref) != hash {
		t.Errorf("Hash mismatch: expected %x, got %x", hash, s.computeHash(ref))
	}
}

// TestHashedNodeCopy tests the Copy method via nodeStore.
func TestHashedNodeCopy(t *testing.T) {
	hash := common.HexToHash("0xabcdef")
	s := newNodeStore()
	ref := s.newHashedRef(hash)
	s.root = ref

	ns := s.Copy()
	copiedHash := ns.computeHash(ns.root)

	if copiedHash != hash {
		t.Errorf("Hash mismatch after copy: expected %x, got %x", hash, copiedHash)
	}
}

// TestHashedNodeInsertValuesAtStem tests InsertValuesAtStem resolution via nodeStore.
func TestHashedNodeInsertValuesAtStem(t *testing.T) {
	// Test 1: nil resolver should return an error
	s := newNodeStore()
	hashedRef := s.newHashedRef(common.HexToHash("0x1234"))
	s.root = hashedRef

	stem := make([]byte, StemSize)
	values := make([][]byte, StemNodeWidth)

	err := s.InsertValuesAtStem(stem, values, nil)
	if err == nil {
		t.Fatal("Expected error for InsertValuesAtStem with nil resolver")
	}

	// Test 2: mock resolver returning invalid data should return deserialization error
	mockResolver := func(path []byte, hash common.Hash) ([]byte, error) {
		return []byte{0xff, 0xff, 0xff}, nil
	}

	s2 := newNodeStore()
	hashedRef2 := s2.newHashedRef(common.HexToHash("0x1234"))
	s2.root = hashedRef2

	err = s2.InsertValuesAtStem(stem, values, mockResolver)
	if err == nil {
		t.Fatal("Expected error for InsertValuesAtStem with invalid resolver data")
	}

	// Test 3: mock resolver returning valid serialized node should succeed
	stem = make([]byte, StemSize)
	stem[0] = 0xaa
	originalValues := make([][]byte, StemNodeWidth)
	originalValues[0] = common.HexToHash("0x1111111111111111111111111111111111111111111111111111111111111111").Bytes()
	originalValues[1] = common.HexToHash("0x2222222222222222222222222222222222222222222222222222222222222222").Bytes()

	// Build the serialized node
	rs := newNodeStore()
	ref := rs.newStemRef(stem, 0)
	sn := rs.getStem(ref.Index())
	for i, v := range originalValues {
		if v != nil {
			sn.setValue(byte(i), v)
		}
	}
	serialized := rs.serializeNode(ref)

	validResolver := func(path []byte, hash common.Hash) ([]byte, error) {
		return serialized, nil
	}

	s3 := newNodeStore()
	hashedRef3 := s3.newHashedRef(common.HexToHash("0x1234"))
	s3.root = hashedRef3

	newValues := make([][]byte, StemNodeWidth)
	newValues[2] = common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333").Bytes()

	err = s3.InsertValuesAtStem(stem, newValues, validResolver)
	if err != nil {
		t.Fatalf("Expected successful resolution and insertion, got error: %v", err)
	}

	// Verify original values are preserved
	retrieved, err := s3.GetValuesAtStem(stem, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retrieved[0], originalValues[0]) {
		t.Errorf("Original value at index 0 not preserved")
	}
	if !bytes.Equal(retrieved[1], originalValues[1]) {
		t.Errorf("Original value at index 1 not preserved")
	}
	if !bytes.Equal(retrieved[2], newValues[2]) {
		t.Errorf("New value at index 2 not inserted correctly")
	}
}

// TestHashedNodeGetError tests that getting through an unresolved HashedNode root returns error.
func TestHashedNodeGetError(t *testing.T) {
	s := newNodeStore()
	// Create root as hashed, then try to resolve through InternalNode parent
	rootRef := s.newInternalRef(0)
	rootNode := s.getInternal(rootRef.Index())
	hashedLeft := s.newHashedRef(common.HexToHash("0x1234"))
	rootNode.left = hashedLeft
	rootNode.right = emptyRef
	s.root = rootRef

	key := make([]byte, 32) // goes left
	key[31] = 5

	resolver := func(path []byte, hash common.Hash) ([]byte, error) {
		return nil, errors.New("node not found")
	}

	_, err := s.Get(key, resolver)
	if err == nil {
		t.Fatal("Expected error when resolver fails")
	}
}
