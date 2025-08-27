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

package stateless

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

func TestIsLeafNode(t *testing.T) {
	tests := []struct {
		name     string
		nodeData []byte
		want     bool
	}{
		{
			name: "leaf node with terminator",
			// Compact encoding: first byte 0x20 means odd length key with terminator
			// This represents a leaf node
			nodeData: mustEncodeNode(t, [][]byte{
				{0x20, 0x01, 0x02, 0x03}, // Key with terminator flag
				{0x01, 0x02, 0x03, 0x04},  // Value
			}),
			want: true,
		},
		{
			name: "leaf node with even key and terminator",
			// Compact encoding: first byte 0x30 means even length key with terminator
			nodeData: mustEncodeNode(t, [][]byte{
				{0x30, 0x01, 0x02}, // Key with terminator flag (even length)
				{0x05, 0x06},        // Value
			}),
			want: true,
		},
		{
			name: "extension node (no terminator)",
			// Compact encoding: first byte 0x00 means even length key without terminator
			// This represents an extension node
			nodeData: mustEncodeNode(t, [][]byte{
				{0x00, 0x01, 0x02},                                                              // Key without terminator flag
				{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, // 32-byte hash
					0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a,
					0x1b, 0x1c, 0x1d, 0x1e, 0x1f},
			}),
			want: false,
		},
		{
			name: "extension node with odd key (no terminator)",
			// Compact encoding: first byte 0x10 means odd length key without terminator
			nodeData: mustEncodeNode(t, [][]byte{
				{0x10, 0x01, 0x02, 0x03}, // Key without terminator flag (odd length)
				{0x01, 0x02, 0x03, 0x04}, // Could be hash reference
			}),
			want: false,
		},
		{
			name: "branch node",
			// Branch nodes have 17 elements
			nodeData: mustEncodeNode(t, [][]byte{
				{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
			}),
			want: false,
		},
		{
			name:     "empty data",
			nodeData: []byte{},
			want:     false,
		},
		{
			name:     "invalid RLP",
			nodeData: []byte{0xff, 0xff, 0xff},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isLeafNode(tt.nodeData)
			if got != tt.want {
				t.Errorf("isLeafNode() = %v, want %v", got, tt.want)
			}
		})
	}
}

func mustEncodeNode(t *testing.T, elems [][]byte) []byte {
	data, err := rlp.EncodeToBytes(elems)
	if err != nil {
		t.Fatalf("Failed to encode node: %v", err)
	}
	return data
}

func TestWitnessStats(t *testing.T) {
	// Create a witness stats collector
	stats := NewWitnessStats()

	// Create witness data with both leaf and non-leaf nodes
	witness := map[string][]byte{
		// Leaf node at depth 4 (path length 4)
		"abcd": mustEncodeNode(t, [][]byte{
			{0x20, 0x01, 0x02}, // Key with terminator
			{0x01, 0x02},        // Value
		}),
		// Extension node at depth 2 (should not be counted)
		"ab": mustEncodeNode(t, [][]byte{
			{0x00, 0x01, 0x02}, // Key without terminator
			{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d,
				0x0e, 0x0f, 0x10, 0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18, 0x19, 0x1a,
				0x1b, 0x1c, 0x1d, 0x1e, 0x1f}, // 31-byte hash (simulated)
		}),
		// Another leaf node at depth 6
		"abcdef": mustEncodeNode(t, [][]byte{
			{0x30, 0x01}, // Key with terminator
			{0x03, 0x04}, // Value
		}),
		// Branch node (should not be counted)
		"a": mustEncodeNode(t, [][]byte{
			{}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {}, {},
		}),
	}

	// Add account trie data (zero owner hash)
	stats.Add(witness, common.Hash{})

	// Verify only leaf nodes were counted
	if stats.accountTrie.samples != 2 {
		t.Errorf("Expected 2 leaf nodes in account trie, got %d", stats.accountTrie.samples)
	}

	// Check the depth statistics
	expectedAvg := int64((4 + 6) / 2) // Average of path lengths 4 and 6
	if stats.accountTrie.totalDepth/stats.accountTrie.samples != expectedAvg {
		t.Errorf("Expected average depth %d, got %d", expectedAvg, stats.accountTrie.totalDepth/stats.accountTrie.samples)
	}
	if stats.accountTrie.minDepth != 4 {
		t.Errorf("Expected min depth 4, got %d", stats.accountTrie.minDepth)
	}
	if stats.accountTrie.maxDepth != 6 {
		t.Errorf("Expected max depth 6, got %d", stats.accountTrie.maxDepth)
	}

	// Test storage trie (non-zero owner hash)
	storageStats := NewWitnessStats()
	storageWitness := map[string][]byte{
		// Leaf node
		"xyz": mustEncodeNode(t, [][]byte{
			{0x20, 0x01}, // Key with terminator
			{0x05, 0x06}, // Value
		}),
	}
	storageStats.Add(storageWitness, common.HexToHash("0x1234"))

	if storageStats.storageTrie.samples != 1 {
		t.Errorf("Expected 1 leaf node in storage trie, got %d", storageStats.storageTrie.samples)
	}
	if storageStats.accountTrie.samples != 0 {
		t.Errorf("Expected 0 nodes in account trie for storage access, got %d", storageStats.accountTrie.samples)
	}
}