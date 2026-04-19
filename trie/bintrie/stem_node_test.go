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

// TestStemNodeInsertSameStem tests inserting values with the same stem via nodeStore.
func TestStemNodeInsertSameStem(t *testing.T) {
	s := newNodeStore()

	stem := make([]byte, 31)
	for i := range stem {
		stem[i] = byte(i)
	}

	// Insert first value
	key1 := make([]byte, 32)
	copy(key1[:31], stem)
	key1[31] = 0
	value1 := common.HexToHash("0x0101").Bytes()
	if err := s.Insert(key1, value1, nil); err != nil {
		t.Fatal(err)
	}

	// Insert another value with the same stem but different last byte
	key2 := make([]byte, 32)
	copy(key2[:31], stem)
	key2[31] = 10
	value2 := common.HexToHash("0x0202").Bytes()
	if err := s.Insert(key2, value2, nil); err != nil {
		t.Fatal(err)
	}

	// Root should still be a StemNode
	if s.root.Kind() != kindStem {
		t.Fatalf("Expected kindStem root, got kind %d", s.root.Kind())
	}

	// Check that both values are present
	v1, _ := s.Get(key1, nil)
	if !bytes.Equal(v1, value1) {
		t.Errorf("Value at index 0 mismatch")
	}
	v2, _ := s.Get(key2, nil)
	if !bytes.Equal(v2, value2) {
		t.Errorf("Value at index 10 mismatch")
	}
}

// TestStemNodeInsertDifferentStem tests inserting values with different stems via nodeStore.
func TestStemNodeInsertDifferentStem(t *testing.T) {
	s := newNodeStore()

	// Insert first value with stem of all zeros
	key1 := make([]byte, 32)
	key1[31] = 0
	value1 := common.HexToHash("0x0101").Bytes()
	if err := s.Insert(key1, value1, nil); err != nil {
		t.Fatal(err)
	}

	// Insert with a different stem (first bit different)
	key2 := make([]byte, 32)
	key2[0] = 0x80 // First bit is 1 instead of 0
	value2 := common.HexToHash("0x0202").Bytes()
	if err := s.Insert(key2, value2, nil); err != nil {
		t.Fatal(err)
	}

	// Should now be an InternalNode
	if s.root.Kind() != kindInternal {
		t.Fatalf("Expected kindInternal root, got kind %d", s.root.Kind())
	}

	// Check depth
	rootNode := s.getInternal(s.root.Index())
	if rootNode.depth != 0 {
		t.Errorf("Expected depth 0, got %d", rootNode.depth)
	}

	// Verify both values are retrievable
	v1, _ := s.Get(key1, nil)
	if !bytes.Equal(v1, value1) {
		t.Error("Value 1 mismatch")
	}
	v2, _ := s.Get(key2, nil)
	if !bytes.Equal(v2, value2) {
		t.Error("Value 2 mismatch")
	}
}

// TestStemNodeInsertInvalidValueLength tests inserting value with invalid length via nodeStore.
func TestStemNodeInsertInvalidValueLength(t *testing.T) {
	s := newNodeStore()

	key := make([]byte, 32)
	invalidValue := []byte{1, 2, 3} // Not 32 bytes

	err := s.Insert(key, invalidValue, nil)
	if err == nil {
		t.Fatal("Expected error for invalid value length")
	}

	if err.Error() != "invalid insertion: value length" {
		t.Errorf("Expected 'invalid insertion: value length' error, got: %v", err)
	}
}

// TestStemNodeCopy tests the Copy method via nodeStore.
func TestStemNodeCopy(t *testing.T) {
	s := newNodeStore()

	key1 := make([]byte, 32)
	for i := range 31 {
		key1[i] = byte(i)
	}
	key1[31] = 0
	value1 := common.HexToHash("0x0101").Bytes()

	key2 := make([]byte, 32)
	copy(key2[:31], key1[:31])
	key2[31] = 255
	value2 := common.HexToHash("0x0202").Bytes()

	if err := s.Insert(key1, value1, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Insert(key2, value2, nil); err != nil {
		t.Fatal(err)
	}

	ns := s.Copy()

	// Check that values are equal
	v1, _ := ns.Get(key1, nil)
	if !bytes.Equal(v1, value1) {
		t.Errorf("Value at index 0 mismatch after copy")
	}
	v2, _ := ns.Get(key2, nil)
	if !bytes.Equal(v2, value2) {
		t.Errorf("Value at index 255 mismatch after copy")
	}
}

// TestStemNodeHash tests the Hash method.
func TestStemNodeHash(t *testing.T) {
	s := newNodeStore()

	key := make([]byte, 32)
	key[31] = 0
	value := common.HexToHash("0x0101").Bytes()
	if err := s.Insert(key, value, nil); err != nil {
		t.Fatal(err)
	}

	hash1 := s.computeHash(s.root)

	// Hash should be deterministic
	hash2 := s.computeHash(s.root)
	if hash1 != hash2 {
		t.Errorf("Hash not deterministic: %x != %x", hash1, hash2)
	}

	// Changing a value should change the hash
	key2 := make([]byte, 32)
	key2[31] = 1
	value2 := common.HexToHash("0x0202").Bytes()
	if err := s.Insert(key2, value2, nil); err != nil {
		t.Fatal(err)
	}
	hash3 := s.computeHash(s.root)
	if hash1 == hash3 {
		t.Error("Hash didn't change after modifying values")
	}
}

// TestStemNodeGetValuesAtStem tests GetValuesAtStem method via nodeStore.
func TestStemNodeGetValuesAtStem(t *testing.T) {
	s := newNodeStore()

	stem := make([]byte, 31)
	for i := range stem {
		stem[i] = byte(i)
	}

	values := make([][]byte, 256)
	values[0] = common.HexToHash("0x0101").Bytes()
	values[10] = common.HexToHash("0x0202").Bytes()
	values[255] = common.HexToHash("0x0303").Bytes()

	if err := s.InsertValuesAtStem(stem, values, nil); err != nil {
		t.Fatal(err)
	}

	// GetValuesAtStem with matching stem
	retrievedValues, err := s.GetValuesAtStem(stem, nil)
	if err != nil {
		t.Fatalf("Failed to get values: %v", err)
	}

	if !bytes.Equal(retrievedValues[0], values[0]) {
		t.Error("Value at index 0 mismatch")
	}
	if !bytes.Equal(retrievedValues[10], values[10]) {
		t.Error("Value at index 10 mismatch")
	}
	if !bytes.Equal(retrievedValues[255], values[255]) {
		t.Error("Value at index 255 mismatch")
	}

	// GetValuesAtStem with different stem should return nil values
	differentStem := make([]byte, 31)
	differentStem[0] = 0xFF

	shouldBeEmpty, err := s.GetValuesAtStem(differentStem, nil)
	if err != nil {
		t.Fatalf("Failed to get values with different stem: %v", err)
	}

	allNil := true
	for _, v := range shouldBeEmpty {
		if v != nil {
			allNil = false
			break
		}
	}
	if !allNil {
		t.Error("Expected all nil values for different stem")
	}
}

// TestStemNodeInsertValuesAtStem tests InsertValuesAtStem method via nodeStore.
func TestStemNodeInsertValuesAtStem(t *testing.T) {
	s := newNodeStore()

	stem := make([]byte, 31)
	values := make([][]byte, 256)
	values[0] = common.HexToHash("0x0101").Bytes()

	if err := s.InsertValuesAtStem(stem, values, nil); err != nil {
		t.Fatal(err)
	}

	// Insert new values at the same stem
	newValues := make([][]byte, 256)
	newValues[1] = common.HexToHash("0x0202").Bytes()
	newValues[2] = common.HexToHash("0x0303").Bytes()

	if err := s.InsertValuesAtStem(stem, newValues, nil); err != nil {
		t.Fatal(err)
	}

	// Check that all values are present
	retrieved, err := s.GetValuesAtStem(stem, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(retrieved[0], values[0]) {
		t.Error("Original value at index 0 missing")
	}
	if !bytes.Equal(retrieved[1], newValues[1]) {
		t.Error("New value at index 1 missing")
	}
	if !bytes.Equal(retrieved[2], newValues[2]) {
		t.Error("New value at index 2 missing")
	}
}

// TestStemNodeGetHeight tests GetHeight method via nodeStore.
func TestStemNodeGetHeight(t *testing.T) {
	s := newNodeStore()

	key := make([]byte, 32)
	value := common.HexToHash("0x01").Bytes()
	if err := s.Insert(key, value, nil); err != nil {
		t.Fatal(err)
	}

	height := s.getHeight(s.root)
	if height != 1 {
		t.Errorf("Expected height 1, got %d", height)
	}
}

// TestStemNodeCollectNodes tests CollectNodes method via nodeStore.
func TestStemNodeCollectNodes(t *testing.T) {
	s := newNodeStore()

	stem := make([]byte, 31)
	values := make([][]byte, 256)
	values[0] = common.HexToHash("0x0101").Bytes()

	if err := s.InsertValuesAtStem(stem, values, nil); err != nil {
		t.Fatal(err)
	}

	var collectedPaths [][]byte
	flushFn := func(path []byte, hash common.Hash, serialized []byte) {
		pathCopy := make([]byte, len(path))
		copy(pathCopy, path)
		collectedPaths = append(collectedPaths, pathCopy)
	}

	err := s.collectNodes(s.root, []byte{0, 1, 0}, flushFn)
	if err != nil {
		t.Fatalf("Failed to collect nodes: %v", err)
	}

	// Should have collected one node (itself)
	if len(collectedPaths) != 1 {
		t.Errorf("Expected 1 collected node, got %d", len(collectedPaths))
	}

	// Check the path
	if !bytes.Equal(collectedPaths[0], []byte{0, 1, 0}) {
		t.Errorf("Path mismatch: expected [0, 1, 0], got %v", collectedPaths[0])
	}
}
