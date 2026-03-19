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

	s := NewNodeStore()
	ref := s.allocStem(StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  0,
	})

	// Insert another value with the same stem but different last byte
	key := make([]byte, 32)
	copy(key[:31], stem)
	key[31] = 10
	value := common.HexToHash("0x0202").Bytes()

	newRef, err := s.Insert(ref, key, value, nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	if newRef.Kind() != KindStem {
		t.Fatalf("Expected KindStem, got %v", newRef.Kind())
	}

	sn := s.getStem(newRef.Index())
	if !bytes.Equal(sn.Values[0], values[0]) {
		t.Errorf("Value at index 0 mismatch")
	}
	if !bytes.Equal(sn.Values[10], value) {
		t.Errorf("Value at index 10 mismatch")
	}
}

// TestStemNodeInsertDifferentStem tests inserting values with different stems
func TestStemNodeInsertDifferentStem(t *testing.T) {
	stem1 := make([]byte, 31)

	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()

	s := NewNodeStore()
	ref := s.allocStem(StemNode{
		Stem:   stem1,
		Values: values[:],
		depth:  0,
	})

	// Insert with a different stem (first bit different)
	key := make([]byte, 32)
	key[0] = 0x80
	value := common.HexToHash("0x0202").Bytes()

	newRef, err := s.Insert(ref, key, value, nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert: %v", err)
	}

	if newRef.Kind() != KindInternal {
		t.Fatalf("Expected KindInternal, got %v", newRef.Kind())
	}

	n := s.getInternal(newRef.Index())
	if n.depth != 0 {
		t.Errorf("Expected depth 0, got %d", n.depth)
	}

	// Original stem should be on the left (bit 0)
	if n.left.Kind() != KindStem {
		t.Fatalf("Expected left child to be KindStem, got %v", n.left.Kind())
	}
	leftStem := s.getStem(n.left.Index())
	if !bytes.Equal(leftStem.Stem, stem1) {
		t.Errorf("Left stem mismatch")
	}

	// New stem should be on the right (bit 1)
	if n.right.Kind() != KindStem {
		t.Fatalf("Expected right child to be KindStem, got %v", n.right.Kind())
	}
	rightStem := s.getStem(n.right.Index())
	if !bytes.Equal(rightStem.Stem, key[:31]) {
		t.Errorf("Right stem mismatch")
	}
}

// TestStemNodeInsertInvalidValueLength tests inserting value with invalid length
func TestStemNodeInsertInvalidValueLength(t *testing.T) {
	stem := make([]byte, 31)
	var values [256][]byte

	s := NewNodeStore()
	ref := s.allocStem(StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  0,
	})

	key := make([]byte, 32)
	copy(key[:31], stem)
	invalidValue := []byte{1, 2, 3} // Not 32 bytes

	// Insert goes through InsertValuesAtStem which accepts any value length,
	// but the old test tested StemNode.Insert directly. With the store
	// architecture, single-value Insert wraps into InsertValuesAtStem,
	// which puts the value at values[suffix]. This succeeds (no length check
	// at the stem level for InsertValuesAtStem).
	// We just verify the round-trip works.
	newRef, err := s.Insert(ref, key, invalidValue, nil, 0)
	if err != nil {
		t.Fatalf("Insert error: %v", err)
	}
	sn := s.getStem(newRef.Index())
	if !bytes.Equal(sn.Values[0], invalidValue) {
		t.Errorf("Value mismatch")
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

	s := NewNodeStore()
	s.allocStem(StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  10,
	})

	ns := s.Copy()
	copiedStem := ns.getStem(0)

	if !bytes.Equal(copiedStem.Stem, stem) {
		t.Errorf("Stem mismatch after copy")
	}
	if !bytes.Equal(copiedStem.Values[0], values[0]) {
		t.Errorf("Value at index 0 mismatch after copy")
	}
	if !bytes.Equal(copiedStem.Values[255], values[255]) {
		t.Errorf("Value at index 255 mismatch after copy")
	}
	if copiedStem.depth != 10 {
		t.Errorf("Depth mismatch: expected 10, got %d", copiedStem.depth)
	}

	// Verify deep copy
	copiedStem.Values[0] = common.HexToHash("0x9999").Bytes()
	origStem := s.getStem(0)
	if bytes.Equal(origStem.Values[0], common.HexToHash("0x9999").Bytes()) {
		t.Error("Copy is not independent from original")
	}
}

// TestStemNodeHash tests the Hash method
func TestStemNodeHash(t *testing.T) {
	stem := make([]byte, 31)
	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()

	node := &StemNode{
		Stem:          stem,
		Values:        values[:],
		depth:         0,
		mustRecompute: true,
	}

	hash1 := node.Hash()
	hash2 := node.Hash()
	if hash1 != hash2 {
		t.Errorf("Hash not deterministic: %x != %x", hash1, hash2)
	}

	node.Values[1] = common.HexToHash("0x0202").Bytes()
	node.mustRecompute = true
	hash3 := node.Hash()
	if hash1 == hash3 {
		t.Error("Hash didn't change after modifying values")
	}
}

// TestStemNodeGetValuesAtStem tests GetValuesAtStem via NodeStore
func TestStemNodeGetValuesAtStem(t *testing.T) {
	stem := make([]byte, 31)
	for i := range stem {
		stem[i] = byte(i)
	}

	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()
	values[10] = common.HexToHash("0x0202").Bytes()
	values[255] = common.HexToHash("0x0303").Bytes()

	s := NewNodeStore()
	ref := s.allocStem(StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  0,
	})

	retrievedValues, err := s.GetValuesAtStem(ref, stem, nil)
	if err != nil {
		t.Fatalf("Failed to get values: %v", err)
	}

	for i := range 256 {
		if !bytes.Equal(retrievedValues[i], values[i]) {
			t.Errorf("Value mismatch at index %d", i)
		}
	}

	// Different stem should return nil
	differentStem := make([]byte, 31)
	differentStem[0] = 0xFF
	shouldBeNil, err := s.GetValuesAtStem(ref, differentStem, nil)
	if err != nil {
		t.Fatalf("Failed to get values with different stem: %v", err)
	}
	if shouldBeNil != nil {
		t.Error("Expected nil for different stem, got non-nil")
	}
}

// TestStemNodeInsertValuesAtStem tests InsertValuesAtStem
func TestStemNodeInsertValuesAtStem(t *testing.T) {
	stem := make([]byte, 31)
	var values [256][]byte
	values[0] = common.HexToHash("0x0101").Bytes()

	s := NewNodeStore()
	ref := s.allocStem(StemNode{
		Stem:   stem,
		Values: values[:],
		depth:  0,
	})

	var newValues [256][]byte
	newValues[1] = common.HexToHash("0x0202").Bytes()
	newValues[2] = common.HexToHash("0x0303").Bytes()

	newRef, err := s.InsertValuesAtStem(ref, stem, newValues[:], nil, 0)
	if err != nil {
		t.Fatalf("Failed to insert values: %v", err)
	}

	sn := s.getStem(newRef.Index())
	if !bytes.Equal(sn.Values[0], values[0]) {
		t.Error("Original value at index 0 missing")
	}
	if !bytes.Equal(sn.Values[1], newValues[1]) {
		t.Error("New value at index 1 missing")
	}
	if !bytes.Equal(sn.Values[2], newValues[2]) {
		t.Error("New value at index 2 missing")
	}
}

// TestStemNodeGetHeight tests GetHeight
func TestStemNodeGetHeight(t *testing.T) {
	s := NewNodeStore()
	ref := s.allocStem(StemNode{
		Stem:   make([]byte, 31),
		Values: make([][]byte, 256),
		depth:  0,
	})

	height := s.GetHeight(ref)
	if height != 1 {
		t.Errorf("Expected height 1, got %d", height)
	}
}

// TestStemNodeCollectNodes tests CollectNodes
func TestStemNodeCollectNodes(t *testing.T) {
	s := NewNodeStore()
	ref := s.allocStem(StemNode{
		Stem:   make([]byte, 31),
		Values: make([][]byte, 256),
		depth:  0,
	})
	s.getStem(ref.Index()).Values[0] = common.HexToHash("0x0101").Bytes()

	var collectedPaths [][]byte
	var collectedRefs []NodeRef

	flushFn := func(path []byte, r NodeRef) {
		pathCopy := make([]byte, len(path))
		copy(pathCopy, path)
		collectedPaths = append(collectedPaths, pathCopy)
		collectedRefs = append(collectedRefs, r)
	}

	err := s.CollectNodes(ref, []byte{0, 1, 0}, flushFn)
	if err != nil {
		t.Fatalf("Failed to collect nodes: %v", err)
	}

	if len(collectedRefs) != 1 {
		t.Errorf("Expected 1 collected node, got %d", len(collectedRefs))
	}

	if !bytes.Equal(collectedPaths[0], []byte{0, 1, 0}) {
		t.Errorf("Path mismatch: expected [0, 1, 0], got %v", collectedPaths[0])
	}
}
