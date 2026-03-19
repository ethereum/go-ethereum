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

// TestHashedNodeHash tests the hash of a HashedNode via the store
func TestHashedNodeHash(t *testing.T) {
	hash := common.HexToHash("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	s := NewNodeStore()
	ref := s.allocHashed(HashedNode{hash: hash})

	if s.ComputeHash(ref) != hash {
		t.Errorf("Hash mismatch: expected %x, got %x", hash, s.ComputeHash(ref))
	}
}

// TestHashedNodeInsertValuesAtStem tests that InsertValuesAtStem on a hashed ref
// resolves via the store
func TestHashedNodeInsertValuesAtStem(t *testing.T) {
	hash := common.HexToHash("0x1234")
	s := NewNodeStore()
	ref := s.allocHashed(HashedNode{hash: hash})

	stem := make([]byte, StemSize)
	values := make([][]byte, StemNodeWidth)

	// Test 1: nil resolver should return an error
	_, err := s.InsertValuesAtStem(ref, stem, values, nil, 0)
	if err == nil {
		t.Fatal("Expected error for InsertValuesAtStem on HashedNode with nil resolver")
	}

	if err.Error() != "InsertValuesAtStem resolve error: resolver is nil" {
		t.Errorf("Unexpected error message: %v", err)
	}

	// Test 2: mock resolver returning invalid data should return deserialization error
	mockResolver := func(path []byte, hash common.Hash) ([]byte, error) {
		return []byte{0xff, 0xff, 0xff}, nil
	}

	_, err = s.InsertValuesAtStem(ref, stem, values, mockResolver, 0)
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

	// Create a temporary store to serialize the original node
	tmpStore := NewNodeStore()
	origRef := tmpStore.allocStem(StemNode{
		Stem:   stem,
		Values: originalValues[:],
		depth:  0,
	})
	serialized := tmpStore.SerializeNode(origRef)

	validResolver := func(path []byte, hash common.Hash) ([]byte, error) {
		return serialized, nil
	}

	var newValues [StemNodeWidth][]byte
	newValues[2] = common.HexToHash("0x3333333333333333333333333333333333333333333333333333333333333333").Bytes()

	s2 := NewNodeStore()
	ref2 := s2.allocHashed(HashedNode{hash: hash})
	resolvedRef, err := s2.InsertValuesAtStem(ref2, stem, newValues[:], validResolver, 0)
	if err != nil {
		t.Fatalf("Expected successful resolution and insertion, got error: %v", err)
	}

	if resolvedRef.Kind() != KindStem {
		t.Fatalf("Expected KindStem, got %v", resolvedRef.Kind())
	}

	resultStem := s2.getStem(resolvedRef.Index())
	if !bytes.Equal(resultStem.Stem, stem) {
		t.Errorf("Stem mismatch: expected %x, got %x", stem, resultStem.Stem)
	}

	if !bytes.Equal(resultStem.Values[0], originalValues[0]) {
		t.Errorf("Original value at index 0 not preserved")
	}
	if !bytes.Equal(resultStem.Values[1], originalValues[1]) {
		t.Errorf("Original value at index 1 not preserved")
	}
	if !bytes.Equal(resultStem.Values[2], newValues[2]) {
		t.Errorf("New value at index 2 not inserted correctly")
	}
}

// TestHashedNodeGetValuesAtStem tests that GetValuesAtStem on a hashed ref returns error
func TestHashedNodeGetValuesAtStem(t *testing.T) {
	s := NewNodeStore()
	ref := s.allocHashed(HashedNode{hash: common.HexToHash("0x1234")})

	stem := make([]byte, StemSize)
	_, err := s.GetValuesAtStem(ref, stem, nil)
	if err == nil {
		t.Fatal("Expected error for GetValuesAtStem on HashedNode")
	}

	if err.Error() != "attempted to get values from an unresolved node" {
		t.Errorf("Unexpected error message: %v", err)
	}
}
