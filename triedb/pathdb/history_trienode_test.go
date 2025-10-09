// Copyright 2025 The go-ethereum Authors
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

package pathdb

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/internal/testrand"
)

// randomTrienodes generates a random trienode set.
func randomTrienodes(n int) (map[common.Hash]map[string][]byte, common.Hash) {
	var (
		root  common.Hash
		nodes = make(map[common.Hash]map[string][]byte)
	)
	for i := 0; i < n; i++ {
		owner := testrand.Hash()
		if i == 0 {
			owner = common.Hash{}
		}
		nodes[owner] = make(map[string][]byte)

		for j := 0; j < 10; j++ {
			path := testrand.Bytes(rand.Intn(10))
			for z := 0; z < len(path); z++ {
				nodes[owner][string(path[:z])] = testrand.Bytes(rand.Intn(128))
			}
		}
		// zero-size trie node, representing it was non-existent before
		for j := 0; j < 10; j++ {
			path := testrand.Bytes(32)
			nodes[owner][string(path)] = nil
		}
		// root node with zero-size path
		rnode := testrand.Bytes(256)
		nodes[owner][""] = rnode
		if owner == (common.Hash{}) {
			root = crypto.Keccak256Hash(rnode)
		}
	}
	return nodes, root
}

func makeTrienodeHistory() *trienodeHistory {
	nodes, root := randomTrienodes(10)
	return newTrienodeHistory(root, common.Hash{}, 1, nodes)
}

func makeTrienodeHistories(n int) []*trienodeHistory {
	var (
		parent common.Hash
		result []*trienodeHistory
	)
	for i := 0; i < n; i++ {
		nodes, root := randomTrienodes(10)
		result = append(result, newTrienodeHistory(root, parent, uint64(i+1), nodes))
		parent = root
	}
	return result
}

func TestEncodeDecodeTrienodeHistory(t *testing.T) {
	var (
		dec trienodeHistory
		obj = makeTrienodeHistory()
	)
	header, keySection, valueSection, err := obj.encode()
	if err != nil {
		t.Fatalf("Failed to encode trienode history: %v", err)
	}
	if err := dec.decode(header, keySection, valueSection); err != nil {
		t.Fatalf("Failed to decode trienode history: %v", err)
	}

	if !reflect.DeepEqual(obj.meta, dec.meta) {
		t.Fatal("trienode metadata is mismatched")
	}
	if !compareList(dec.owners, obj.owners) {
		t.Fatal("trie owner list is mismatched")
	}
	if !compareMapList(dec.nodeList, obj.nodeList) {
		t.Fatal("trienode list is mismatched")
	}
	if !compareMapSet(dec.nodes, obj.nodes) {
		t.Fatal("trienode content is mismatched")
	}

	// Re-encode again, ensuring the encoded blob still match
	header2, keySection2, valueSection2, err := dec.encode()
	if err != nil {
		t.Fatalf("Failed to encode trienode history: %v", err)
	}
	if !bytes.Equal(header, header2) {
		t.Fatal("re-encoded header is mismatched")
	}
	if !bytes.Equal(keySection, keySection2) {
		t.Fatal("re-encoded key section is mismatched")
	}
	if !bytes.Equal(valueSection, valueSection2) {
		t.Fatal("re-encoded value section is mismatched")
	}
}

func TestTrienodeHistoryReader(t *testing.T) {
	var (
		hs         = makeTrienodeHistories(10)
		freezer, _ = rawdb.NewTrienodeFreezer(t.TempDir(), false, false)
	)
	defer freezer.Close()

	for i, h := range hs {
		header, keySection, valueSection, _ := h.encode()
		if err := rawdb.WriteTrienodeHistory(freezer, uint64(i+1), header, keySection, valueSection); err != nil {
			t.Fatalf("Failed to write trienode history: %v", err)
		}
	}
	for i, h := range hs {
		tr, err := newTrienodeHistoryReader(uint64(i+1), freezer)
		if err != nil {
			t.Fatalf("Failed to construct the history reader: %v", err)
		}
		for _, owner := range h.owners {
			nodes := h.nodes[owner]
			for key, value := range nodes {
				blob, err := tr.read(owner, key)
				if err != nil {
					t.Fatalf("Failed to read trienode history: %v", err)
				}
				if !bytes.Equal(blob, value) {
					t.Fatalf("Unexpected trie node data, want: %v, got: %v", value, blob)
				}
			}
		}
	}
	for i, h := range hs {
		metadata, err := readTrienodeMetadata(freezer, uint64(i+1))
		if err != nil {
			t.Fatalf("Failed to read trienode history metadata: %v", err)
		}
		if !reflect.DeepEqual(h.meta, metadata) {
			t.Fatalf("Unexpected trienode metadata, want: %v, got: %v", h.meta, metadata)
		}
	}
}

// TestEmptyTrienodeHistory tests encoding/decoding of empty trienode history
func TestEmptyTrienodeHistory(t *testing.T) {
	h := newTrienodeHistory(common.Hash{}, common.Hash{}, 1, make(map[common.Hash]map[string][]byte))

	// Test encoding empty history
	header, keySection, valueSection, err := h.encode()
	if err != nil {
		t.Fatalf("Failed to encode empty trienode history: %v", err)
	}

	// Verify sections are minimal but valid
	if len(header) == 0 {
		t.Fatal("Header should not be empty")
	}
	if len(keySection) != 0 {
		t.Fatal("Key section should be empty for empty history")
	}
	if len(valueSection) != 0 {
		t.Fatal("Value section should be empty for empty history")
	}

	// Test decoding empty history
	var decoded trienodeHistory
	if err := decoded.decode(header, keySection, valueSection); err != nil {
		t.Fatalf("Failed to decode empty trienode history: %v", err)
	}

	if len(decoded.owners) != 0 {
		t.Fatal("Decoded history should have no owners")
	}
	if len(decoded.nodeList) != 0 {
		t.Fatal("Decoded history should have no node lists")
	}
	if len(decoded.nodes) != 0 {
		t.Fatal("Decoded history should have no nodes")
	}
}

// TestSingleTrieHistory tests encoding/decoding of history with single trie
func TestSingleTrieHistory(t *testing.T) {
	nodes := make(map[common.Hash]map[string][]byte)
	owner := testrand.Hash()
	nodes[owner] = make(map[string][]byte)

	// Add some nodes with various sizes
	nodes[owner][""] = testrand.Bytes(32)      // empty key
	nodes[owner]["a"] = testrand.Bytes(1)      // small value
	nodes[owner]["bb"] = testrand.Bytes(100)   // medium value
	nodes[owner]["ccc"] = testrand.Bytes(1000) // large value
	nodes[owner]["dddd"] = testrand.Bytes(0)   // empty value

	h := newTrienodeHistory(common.Hash{}, common.Hash{}, 1, nodes)
	testEncodeDecode(t, h)
}

// TestMultipleTries tests multiple tries with different node counts
func TestMultipleTries(t *testing.T) {
	nodes := make(map[common.Hash]map[string][]byte)

	// First trie with many small nodes
	owner1 := testrand.Hash()
	nodes[owner1] = make(map[string][]byte)
	for i := 0; i < 100; i++ {
		key := string(testrand.Bytes(rand.Intn(10)))
		nodes[owner1][key] = testrand.Bytes(rand.Intn(50))
	}

	// Second trie with few large nodes
	owner2 := testrand.Hash()
	nodes[owner2] = make(map[string][]byte)
	for i := 0; i < 5; i++ {
		key := string(testrand.Bytes(rand.Intn(20)))
		nodes[owner2][key] = testrand.Bytes(1000 + rand.Intn(1000))
	}

	// Third trie with nil values (zero-size nodes)
	owner3 := testrand.Hash()
	nodes[owner3] = make(map[string][]byte)
	for i := 0; i < 10; i++ {
		key := string(testrand.Bytes(rand.Intn(15)))
		nodes[owner3][key] = nil
	}

	h := newTrienodeHistory(common.Hash{}, common.Hash{}, 1, nodes)
	testEncodeDecode(t, h)
}

// TestLargeNodeValues tests encoding/decoding with very large node values
func TestLargeNodeValues(t *testing.T) {
	nodes := make(map[common.Hash]map[string][]byte)
	owner := testrand.Hash()
	nodes[owner] = make(map[string][]byte)

	// Test with progressively larger values
	sizes := []int{1024, 10 * 1024, 100 * 1024, 1024 * 1024} // 1KB, 10KB, 100KB, 1MB
	for _, size := range sizes {
		key := string(testrand.Bytes(10))
		nodes[owner][key] = testrand.Bytes(size)

		h := newTrienodeHistory(common.Hash{}, common.Hash{}, 1, nodes)
		testEncodeDecode(t, h)
		t.Logf("Successfully tested encoding/decoding with %dKB value", size/1024)
	}
}

// TestNilNodeValues tests encoding/decoding with nil (zero-length) node values
func TestNilNodeValues(t *testing.T) {
	nodes := make(map[common.Hash]map[string][]byte)
	owner := testrand.Hash()
	nodes[owner] = make(map[string][]byte)

	// Mix of nil and non-nil values
	nodes[owner]["nil"] = nil
	nodes[owner]["data1"] = []byte("some data")
	nodes[owner]["data2"] = []byte("more data")

	h := newTrienodeHistory(common.Hash{}, common.Hash{}, 1, nodes)
	testEncodeDecode(t, h)

	// Verify nil values are preserved
	_, ok := h.nodes[owner]["nil"]
	if !ok {
		t.Fatal("Nil value should be preserved")
	}
}

// TestCorruptedHeader tests error handling for corrupted header data
func TestCorruptedHeader(t *testing.T) {
	h := makeTrienodeHistory()
	header, keySection, valueSection, _ := h.encode()

	// Test corrupted version
	corruptedHeader := make([]byte, len(header))
	copy(corruptedHeader, header)
	corruptedHeader[0] = 0xFF // Invalid version

	var decoded trienodeHistory
	if err := decoded.decode(corruptedHeader, keySection, valueSection); err == nil {
		t.Fatal("Expected error for corrupted version")
	}

	// Test truncated header
	truncatedHeader := header[:len(header)-5]
	if err := decoded.decode(truncatedHeader, keySection, valueSection); err == nil {
		t.Fatal("Expected error for truncated header")
	}

	// Test header with invalid trie header size
	invalidHeader := make([]byte, len(header))
	copy(invalidHeader, header)
	invalidHeader = invalidHeader[:trienodeMetadataSize+5] // Not divisible by trie header size

	if err := decoded.decode(invalidHeader, keySection, valueSection); err == nil {
		t.Fatal("Expected error for invalid header size")
	}
}

// TestCorruptedKeySection tests error handling for corrupted key section data
func TestCorruptedKeySection(t *testing.T) {
	h := makeTrienodeHistory()
	header, keySection, valueSection, _ := h.encode()

	// Test empty key section when header indicates data
	if len(keySection) > 0 {
		var decoded trienodeHistory
		if err := decoded.decode(header, []byte{}, valueSection); err == nil {
			t.Fatal("Expected error for empty key section with non-empty header")
		}
	}

	// Test truncated key section
	if len(keySection) > 10 {
		truncatedKeySection := keySection[:len(keySection)-10]
		var decoded trienodeHistory
		if err := decoded.decode(header, truncatedKeySection, valueSection); err == nil {
			t.Fatal("Expected error for truncated key section")
		}
	}

	// Test corrupted key section with invalid varint
	corruptedKeySection := make([]byte, len(keySection))
	copy(corruptedKeySection, keySection)
	if len(corruptedKeySection) > 5 {
		corruptedKeySection[5] = 0xFF // Corrupt varint encoding
		var decoded trienodeHistory
		if err := decoded.decode(header, corruptedKeySection, valueSection); err == nil {
			t.Fatal("Expected error for corrupted varint in key section")
		}
	}
}

// TestCorruptedValueSection tests error handling for corrupted value section data
func TestCorruptedValueSection(t *testing.T) {
	h := makeTrienodeHistory()
	header, keySection, valueSection, _ := h.encode()

	// Test truncated value section
	if len(valueSection) > 10 {
		truncatedValueSection := valueSection[:len(valueSection)-10]
		var decoded trienodeHistory
		if err := decoded.decode(header, keySection, truncatedValueSection); err == nil {
			t.Fatal("Expected error for truncated value section")
		}
	}

	// Test empty value section when key section indicates data exists
	if len(valueSection) > 0 {
		var decoded trienodeHistory
		if err := decoded.decode(header, keySection, []byte{}); err == nil {
			t.Fatal("Expected error for empty value section with non-empty key section")
		}
	}
}

// TestInvalidOffsets tests error handling for invalid offsets in encoded data
func TestInvalidOffsets(t *testing.T) {
	h := makeTrienodeHistory()
	header, keySection, valueSection, _ := h.encode()

	// Corrupt key offset in header (make it larger than key section)
	corruptedHeader := make([]byte, len(header))
	copy(corruptedHeader, header)
	corruptedHeader[trienodeMetadataSize+common.HashLength] = 0xff

	var dec1 trienodeHistory
	if err := dec1.decode(corruptedHeader, keySection, valueSection); err == nil {
		t.Fatal("Expected error for invalid key offset")
	}

	// Corrupt value offset in header (make it larger than value section)
	corruptedHeader = make([]byte, len(header))
	copy(corruptedHeader, header)
	corruptedHeader[trienodeMetadataSize+common.HashLength+4] = 0xff

	var dec2 trienodeHistory
	if err := dec2.decode(corruptedHeader, keySection, valueSection); err == nil {
		t.Fatal("Expected error for invalid value offset")
	}
}

// TestTrienodeHistoryReaderNonExistentPath tests reading non-existent paths
func TestTrienodeHistoryReaderNonExistentPath(t *testing.T) {
	var (
		h          = makeTrienodeHistory()
		freezer, _ = rawdb.NewTrienodeFreezer(t.TempDir(), false, false)
	)
	defer freezer.Close()

	header, keySection, valueSection, _ := h.encode()
	if err := rawdb.WriteTrienodeHistory(freezer, 1, header, keySection, valueSection); err != nil {
		t.Fatalf("Failed to write trienode history: %v", err)
	}

	tr, err := newTrienodeHistoryReader(1, freezer)
	if err != nil {
		t.Fatalf("Failed to construct history reader: %v", err)
	}

	// Try to read a non-existent path
	_, err = tr.read(testrand.Hash(), "nonexistent")
	if err == nil {
		t.Fatal("Expected error for non-existent trie owner")
	}

	// Try to read from existing owner but non-existent path
	owner := h.owners[0]
	_, err = tr.read(owner, "nonexistent-path")
	if err == nil {
		t.Fatal("Expected error for non-existent path")
	}
}

// TestTrienodeHistoryReaderNilValues tests reading nil (zero-length) values
func TestTrienodeHistoryReaderNilValues(t *testing.T) {
	nodes := make(map[common.Hash]map[string][]byte)
	owner := testrand.Hash()
	nodes[owner] = make(map[string][]byte)

	// Add some nil values
	nodes[owner]["nil1"] = nil
	nodes[owner]["nil2"] = nil
	nodes[owner]["data1"] = []byte("some data")

	h := newTrienodeHistory(common.Hash{}, common.Hash{}, 1, nodes)

	var freezer, _ = rawdb.NewTrienodeFreezer(t.TempDir(), false, false)
	defer freezer.Close()

	header, keySection, valueSection, _ := h.encode()
	if err := rawdb.WriteTrienodeHistory(freezer, 1, header, keySection, valueSection); err != nil {
		t.Fatalf("Failed to write trienode history: %v", err)
	}

	tr, err := newTrienodeHistoryReader(1, freezer)
	if err != nil {
		t.Fatalf("Failed to construct history reader: %v", err)
	}

	// Test reading nil values
	data1, err := tr.read(owner, "nil1")
	if err != nil {
		t.Fatalf("Failed to read nil value: %v", err)
	}
	if len(data1) != 0 {
		t.Fatal("Expected nil data for nil value")
	}

	data2, err := tr.read(owner, "nil2")
	if err != nil {
		t.Fatalf("Failed to read nil value: %v", err)
	}
	if len(data2) != 0 {
		t.Fatal("Expected nil data for nil value")
	}

	// Test reading non-nil value
	data3, err := tr.read(owner, "data1")
	if err != nil {
		t.Fatalf("Failed to read non-nil value: %v", err)
	}
	if !bytes.Equal(data3, []byte("some data")) {
		t.Fatal("Data mismatch for non-nil value")
	}
}

// TestTrienodeHistoryReaderNilKey tests reading nil (zero-length) key
func TestTrienodeHistoryReaderNilKey(t *testing.T) {
	nodes := make(map[common.Hash]map[string][]byte)
	owner := testrand.Hash()
	nodes[owner] = make(map[string][]byte)

	// Add some nil values
	nodes[owner][""] = []byte("some data")
	nodes[owner]["data1"] = []byte("some data")

	h := newTrienodeHistory(common.Hash{}, common.Hash{}, 1, nodes)

	var freezer, _ = rawdb.NewTrienodeFreezer(t.TempDir(), false, false)
	defer freezer.Close()

	header, keySection, valueSection, _ := h.encode()
	if err := rawdb.WriteTrienodeHistory(freezer, 1, header, keySection, valueSection); err != nil {
		t.Fatalf("Failed to write trienode history: %v", err)
	}

	tr, err := newTrienodeHistoryReader(1, freezer)
	if err != nil {
		t.Fatalf("Failed to construct history reader: %v", err)
	}

	// Test reading nil values
	data1, err := tr.read(owner, "")
	if err != nil {
		t.Fatalf("Failed to read nil value: %v", err)
	}
	if !bytes.Equal(data1, []byte("some data")) {
		t.Fatal("Data mismatch for nil key")
	}

	// Test reading non-nil value
	data2, err := tr.read(owner, "data1")
	if err != nil {
		t.Fatalf("Failed to read non-nil value: %v", err)
	}
	if !bytes.Equal(data2, []byte("some data")) {
		t.Fatal("Data mismatch for non-nil key")
	}
}

// TestTrienodeHistoryReaderIterator tests the iterator functionality
func TestTrienodeHistoryReaderIterator(t *testing.T) {
	h := makeTrienodeHistory()

	// Count expected entries
	expectedCount := 0
	expectedNodes := make(map[stateIdent]bool)
	for owner, nodeList := range h.nodeList {
		expectedCount += len(nodeList)
		for _, node := range nodeList {
			expectedNodes[stateIdent{
				typ:         typeTrienode,
				addressHash: owner,
				path:        node,
			}] = true
		}
	}

	// Test the iterator
	actualCount := 0
	for x := range h.forEach() {
		_ = x
		actualCount++
	}
	if actualCount != expectedCount {
		t.Fatalf("Iterator count mismatch: expected %d, got %d", expectedCount, actualCount)
	}

	// Test that iterator yields expected state identifiers
	seen := make(map[stateIdent]bool)
	for ident := range h.forEach() {
		if ident.typ != typeTrienode {
			t.Fatal("Iterator should only yield trienode history identifiers")
		}
		key := stateIdent{typ: ident.typ, addressHash: ident.addressHash, path: ident.path}
		if seen[key] {
			t.Fatal("Iterator yielded duplicate identifier")
		}
		seen[key] = true

		if !expectedNodes[key] {
			t.Fatalf("Unexpected yielded identifier %v", key)
		}
	}
}

// TestSharedLen tests the sharedLen helper function
func TestSharedLen(t *testing.T) {
	tests := []struct {
		a, b     []byte
		expected int
	}{
		// Empty strings
		{[]byte(""), []byte(""), 0},
		// One empty string
		{[]byte(""), []byte("abc"), 0},
		{[]byte("abc"), []byte(""), 0},
		// No common prefix
		{[]byte("abc"), []byte("def"), 0},
		// Partial common prefix
		{[]byte("abc"), []byte("abx"), 2},
		{[]byte("prefix"), []byte("pref"), 4},
		// Complete common prefix (shorter first)
		{[]byte("ab"), []byte("abcd"), 2},
		// Complete common prefix (longer first)
		{[]byte("abcd"), []byte("ab"), 2},
		// Identical strings
		{[]byte("identical"), []byte("identical"), 9},
		// Binary data
		{[]byte{0x00, 0x01, 0x02}, []byte{0x00, 0x01, 0x03}, 2},
		// Large strings
		{bytes.Repeat([]byte("a"), 1000), bytes.Repeat([]byte("a"), 1000), 1000},
		{bytes.Repeat([]byte("a"), 1000), append(bytes.Repeat([]byte("a"), 999), []byte("b")...), 999},
	}

	for i, test := range tests {
		result := sharedLen(test.a, test.b)
		if result != test.expected {
			t.Errorf("Test %d: sharedLen(%q, %q) = %d, expected %d",
				i, test.a, test.b, result, test.expected)
		}
		// Test commutativity
		resultReverse := sharedLen(test.b, test.a)
		if result != resultReverse {
			t.Errorf("Test %d: sharedLen is not commutative: sharedLen(a,b)=%d, sharedLen(b,a)=%d",
				i, result, resultReverse)
		}
	}
}

// TestDecodeHeaderCorruptedData tests decodeHeader with corrupted data
func TestDecodeHeaderCorruptedData(t *testing.T) {
	// Create valid header data first
	h := makeTrienodeHistory()
	header, _, _, _ := h.encode()

	// Test with empty header
	_, _, _, _, err := decodeHeader([]byte{})
	if err == nil {
		t.Fatal("Expected error for empty header")
	}

	// Test with invalid version
	corruptedVersion := make([]byte, len(header))
	copy(corruptedVersion, header)
	corruptedVersion[0] = 0xFF
	_, _, _, _, err = decodeHeader(corruptedVersion)
	if err == nil {
		t.Fatal("Expected error for invalid version")
	}

	// Test with truncated header (not divisible by trie header size)
	truncated := header[:trienodeMetadataSize+5]
	_, _, _, _, err = decodeHeader(truncated)
	if err == nil {
		t.Fatal("Expected error for truncated header")
	}

	// Test with unordered trie owners
	unordered := make([]byte, len(header))
	copy(unordered, header)

	// Swap two owner hashes to make them unordered
	hash1Start := trienodeMetadataSize
	hash2Start := trienodeMetadataSize + trienodeTrieHeaderSize
	hash1 := unordered[hash1Start : hash1Start+common.HashLength]
	hash2 := unordered[hash2Start : hash2Start+common.HashLength]

	// Only swap if they would be out of order
	copy(unordered[hash1Start:hash1Start+common.HashLength], hash2)
	copy(unordered[hash2Start:hash2Start+common.HashLength], hash1)

	_, _, _, _, err = decodeHeader(unordered)
	if err == nil {
		t.Fatal("Expected error for unordered trie owners")
	}
}

// TestDecodeSingleCorruptedData tests decodeSingle with corrupted data
func TestDecodeSingleCorruptedData(t *testing.T) {
	h := makeTrienodeHistory()
	_, keySection, _, _ := h.encode()

	// Test with empty key section
	_, err := decodeSingle([]byte{}, nil)
	if err == nil {
		t.Fatal("Expected error for empty key section")
	}

	// Test with key section too small for trailer
	if len(keySection) > 0 {
		_, err := decodeSingle(keySection[:3], nil) // Less than 4 bytes for trailer
		if err == nil {
			t.Fatal("Expected error for key section too small for trailer")
		}
	}

	// Test with corrupted varint in key section
	corrupted := make([]byte, len(keySection))
	copy(corrupted, keySection)
	corrupted[5] = 0xFF // Corrupt varint
	_, err = decodeSingle(corrupted, nil)
	if err == nil {
		t.Fatal("Expected error for corrupted varint")
	}

	// Test with corrupted trailer (invalid restart count)
	corrupted = make([]byte, len(keySection))
	copy(corrupted, keySection)
	// Set restart count to something too large
	binary.BigEndian.PutUint32(corrupted[len(corrupted)-4:], 10000)
	_, err = decodeSingle(corrupted, nil)
	if err == nil {
		t.Fatal("Expected error for invalid restart count")
	}
}

// Helper function to test encode/decode cycle
func testEncodeDecode(t *testing.T, h *trienodeHistory) {
	header, keySection, valueSection, err := h.encode()
	if err != nil {
		t.Fatalf("Failed to encode trienode history: %v", err)
	}

	var decoded trienodeHistory
	if err := decoded.decode(header, keySection, valueSection); err != nil {
		t.Fatalf("Failed to decode trienode history: %v", err)
	}

	// Compare the decoded history with original
	if !compareList(decoded.owners, h.owners) {
		t.Fatal("Trie owner list mismatch")
	}
	if !compareMapList(decoded.nodeList, h.nodeList) {
		t.Fatal("Trienode list mismatch")
	}
	if !compareMapSet(decoded.nodes, h.nodes) {
		t.Fatal("Trienode content mismatch")
	}
}
