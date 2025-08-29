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
)

func TestWitnessStatsAdd(t *testing.T) {
	tests := []struct {
		name                 string
		nodes                map[string][]byte
		owner                common.Hash
		expectedAccountDepth int64
		expectedStorageDepth int64
	}{
		{
			name:                 "empty nodes",
			nodes:                map[string][]byte{},
			owner:                common.Hash{},
			expectedAccountDepth: 0,
			expectedStorageDepth: 0,
		},
		{
			name: "single account trie leaf",
			nodes: map[string][]byte{
				"abc": []byte("data"),
			},
			owner:                common.Hash{},
			expectedAccountDepth: 3,
			expectedStorageDepth: 0,
		},
		{
			name: "account trie with internal nodes",
			nodes: map[string][]byte{
				"a":   []byte("data1"),
				"ab":  []byte("data2"),
				"abc": []byte("data3"),
			},
			owner:                common.Hash{},
			expectedAccountDepth: 3, // Only "abc" is a leaf
			expectedStorageDepth: 0,
		},
		{
			name: "multiple account trie branches",
			nodes: map[string][]byte{
				"a":   []byte("data1"),
				"ab":  []byte("data2"),
				"abc": []byte("data3"),
				"b":   []byte("data4"),
				"bc":  []byte("data5"),
				"bcd": []byte("data6"),
			},
			owner:                common.Hash{},
			expectedAccountDepth: 6, // "abc" (3) + "bcd" (3) = 6
			expectedStorageDepth: 0,
		},
		{
			name: "siblings are all leaves",
			nodes: map[string][]byte{
				"aa": []byte("data1"),
				"ab": []byte("data2"),
				"ac": []byte("data3"),
			},
			owner:                common.Hash{},
			expectedAccountDepth: 6, // 2 + 2 + 2 = 6
			expectedStorageDepth: 0,
		},
		{
			name: "storage trie leaves",
			nodes: map[string][]byte{
				"1":   []byte("data1"),
				"12":  []byte("data2"),
				"123": []byte("data3"),
				"124": []byte("data4"),
			},
			owner:                common.HexToHash("0x1234"),
			expectedAccountDepth: 0,
			expectedStorageDepth: 6, // "123" (3) + "124" (3) = 6
		},
		{
			name: "complex trie structure",
			nodes: map[string][]byte{
				"1":   []byte("data1"),
				"12":  []byte("data2"),
				"123": []byte("data3"),
				"124": []byte("data4"),
				"2":   []byte("data5"),
				"23":  []byte("data6"),
				"234": []byte("data7"),
				"235": []byte("data8"),
				"3":   []byte("data9"),
			},
			owner:                common.Hash{},
			expectedAccountDepth: 13, // "123"(3) + "124"(3) + "234"(3) + "235"(3) + "3"(1) = 13
			expectedStorageDepth: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats := NewWitnessStats()
			stats.Add(tt.nodes, tt.owner)

			// Check account trie depth
			if stats.accountTrie.totalDepth != tt.expectedAccountDepth {
				t.Errorf("Account trie total depth = %d, want %d", stats.accountTrie.totalDepth, tt.expectedAccountDepth)
			}

			// Check storage trie depth
			if stats.storageTrie.totalDepth != tt.expectedStorageDepth {
				t.Errorf("Storage trie total depth = %d, want %d", stats.storageTrie.totalDepth, tt.expectedStorageDepth)
			}
		})
	}
}

func TestWitnessStatsMinMax(t *testing.T) {
	stats := NewWitnessStats()

	// Add some account trie nodes with varying depths
	stats.Add(map[string][]byte{
		"a":     []byte("data1"),
		"ab":    []byte("data2"),
		"abc":   []byte("data3"),
		"abcd":  []byte("data4"),
		"abcde": []byte("data5"),
	}, common.Hash{})

	// Only "abcde" is a leaf (depth 5)
	if stats.accountTrie.minDepth != 5 {
		t.Errorf("Account trie min depth = %d, want %d", stats.accountTrie.minDepth, 5)
	}
	if stats.accountTrie.maxDepth != 5 {
		t.Errorf("Account trie max depth = %d, want %d", stats.accountTrie.maxDepth, 5)
	}

	// Add more leaves with different depths
	stats.Add(map[string][]byte{
		"x":  []byte("data6"),
		"yz": []byte("data7"),
	}, common.Hash{})

	// Now we have leaves at depths 1, 2, and 5
	if stats.accountTrie.minDepth != 1 {
		t.Errorf("Account trie min depth after update = %d, want %d", stats.accountTrie.minDepth, 1)
	}
	if stats.accountTrie.maxDepth != 5 {
		t.Errorf("Account trie max depth after update = %d, want %d", stats.accountTrie.maxDepth, 5)
	}
}

func TestWitnessStatsAverage(t *testing.T) {
	stats := NewWitnessStats()

	// Add nodes that will create leaves at depths 2, 3, and 4
	stats.Add(map[string][]byte{
		"aa":   []byte("data1"),
		"bb":   []byte("data2"),
		"ccc":  []byte("data3"),
		"dddd": []byte("data4"),
	}, common.Hash{})

	// All are leaves: 2 + 2 + 3 + 4 = 11 total, 4 samples
	expectedAvg := int64(11) / int64(4)
	actualAvg := stats.accountTrie.totalDepth / stats.accountTrie.samples

	if actualAvg != expectedAvg {
		t.Errorf("Account trie average depth = %d, want %d", actualAvg, expectedAvg)
	}
}

func BenchmarkWitnessStatsAdd(b *testing.B) {
	// Create a realistic trie node structure
	nodes := make(map[string][]byte)
	for i := 0; i < 100; i++ {
		base := string(rune('a' + i%26))
		nodes[base] = []byte("data")
		for j := 0; j < 9; j++ {
			key := base + string(rune('0'+j))
			nodes[key] = []byte("data")
		}
	}

	stats := NewWitnessStats()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		stats.Add(nodes, common.Hash{})
	}
}
