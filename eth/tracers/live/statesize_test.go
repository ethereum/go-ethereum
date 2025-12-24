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

package live

import (
	"testing"

	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

func TestCalculateDepthCountsByType(t *testing.T) {
	tests := []struct {
		name            string
		paths           []string
		expectedCreated [65]int64
	}{
		{
			name:            "empty map",
			paths:           []string{},
			expectedCreated: [65]int64{},
		},
		{
			name:  "root only",
			paths: []string{""},
			expectedCreated: func() [65]int64 {
				var c [65]int64
				c[0] = 1
				return c
			}(),
		},
		{
			name:  "linear chain - all branch nodes",
			paths: []string{"", "0", "0a", "0a3"},
			expectedCreated: func() [65]int64 {
				var c [65]int64
				c[0] = 1 // ""
				c[1] = 1 // "0"
				c[2] = 1 // "0a"
				c[3] = 1 // "0a3"
				return c
			}(),
		},
		{
			name:  "extension node - path jump",
			paths: []string{"", "0a3"}, // extension from root to "0a3"
			expectedCreated: func() [65]int64 {
				var c [65]int64
				c[0] = 1 // ""
				c[1] = 1 // "0a3" (child of root)
				return c
			}(),
		},
		{
			name:  "branching at root",
			paths: []string{"", "0", "1", "2"},
			expectedCreated: func() [65]int64 {
				var c [65]int64
				c[0] = 1 // ""
				c[1] = 3 // "0", "1", "2"
				return c
			}(),
		},
		{
			name:  "two branches from root",
			paths: []string{"", "0", "0a", "1", "1b"},
			expectedCreated: func() [65]int64 {
				var c [65]int64
				c[0] = 1 // ""
				c[1] = 2 // "0", "1"
				c[2] = 2 // "0a", "1b"
				return c
			}(),
		},
		{
			name:  "mixed extension and branch",
			paths: []string{"", "0", "0a", "0a3", "0b"},
			expectedCreated: func() [65]int64 {
				var c [65]int64
				c[0] = 1 // ""
				c[1] = 1 // "0"
				c[2] = 2 // "0a", "0b"
				c[3] = 1 // "0a3"
				return c
			}(),
		},
		{
			name:  "deep path with extensions",
			paths: []string{"", "abc", "abcdef"},
			expectedCreated: func() [65]int64 {
				var c [65]int64
				c[0] = 1 // ""
				c[1] = 1 // "abc"
				c[2] = 1 // "abcdef"
				return c
			}(),
		},
		{
			name:  "siblings at various depths",
			paths: []string{"", "0", "00", "01", "1", "10", "11"},
			expectedCreated: func() [65]int64 {
				var c [65]int64
				c[0] = 1 // ""
				c[1] = 2 // "0", "1"
				c[2] = 4 // "00", "01", "10", "11"
				return c
			}(),
		},
		{
			name:  "complex tree",
			paths: []string{"", "a", "ab", "abc", "abd", "b", "bc"},
			expectedCreated: func() [65]int64 {
				var c [65]int64
				c[0] = 1 // ""
				c[1] = 2 // "a", "b"
				c[2] = 2 // "ab", "bc"
				c[3] = 2 // "abc", "abd"
				return c
			}(),
		},
		{
			name:  "max depth path",
			paths: []string{"", "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"},
			expectedCreated: func() [65]int64 {
				var c [65]int64
				c[0] = 1 // ""
				c[1] = 1 // 64-nibble path (child of root via extension)
				return c
			}(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Build pathMap with New blob data (simulates created nodes)
			pathMap := make(map[string]*tracing.TrieNodeChange, len(tt.paths))
			for _, p := range tt.paths {
				pathMap[p] = &tracing.TrieNodeChange{
					New: &trienode.Node{Blob: []byte{0x01}}, // Non-empty blob marks as created
				}
			}

			created, deleted := calculateDepthCountsByType(pathMap)

			if created != tt.expectedCreated {
				t.Errorf("calculateDepthCountsByType() created mismatch")
				for i := 0; i < 65; i++ {
					if created[i] != tt.expectedCreated[i] {
						t.Errorf("  depth %d: got %d, want %d", i, created[i], tt.expectedCreated[i])
					}
				}
			}

			// All nodes have New data, so deleted should be all zeros
			var expectedDeleted [65]int64
			if deleted != expectedDeleted {
				t.Errorf("calculateDepthCountsByType() deleted should be all zeros for created nodes")
			}
		})
	}
}

func TestCalculateDepthCountsByType_Deletion(t *testing.T) {
	// Test that deleted nodes are counted correctly
	pathMap := map[string]*tracing.TrieNodeChange{
		"":   {Prev: &trienode.Node{Blob: []byte{0x01}}},                                          // deleted at depth 0
		"0":  {Prev: &trienode.Node{Blob: []byte{0x01}}, New: &trienode.Node{Blob: []byte{}}},     // deleted at depth 1
		"0a": {Prev: &trienode.Node{Blob: []byte{0x01}}},                                          // deleted at depth 2
		"1":  {Prev: &trienode.Node{Blob: []byte{0x01}}, New: &trienode.Node{Blob: []byte{0x02}}}, // modified at depth 1 (not deleted)
	}

	created, deleted := calculateDepthCountsByType(pathMap)

	// Expected deleted: depth 0 = 1, depth 1 = 1, depth 2 = 1
	expectedDeleted := [65]int64{1, 1, 1}
	for i := 0; i < 65; i++ {
		if deleted[i] != expectedDeleted[i] {
			t.Errorf("deleted depth %d: got %d, want %d", i, deleted[i], expectedDeleted[i])
		}
	}

	// Expected created: only "1" is modified (has New data), depth 1 = 1
	expectedCreated := [65]int64{0, 1}
	for i := 0; i < 65; i++ {
		if created[i] != expectedCreated[i] {
			t.Errorf("created depth %d: got %d, want %d", i, created[i], expectedCreated[i])
		}
	}
}

func TestCalculateDepthCountsByType_Mixed(t *testing.T) {
	// Test mixed scenario: some nodes created, some modified, some deleted
	pathMap := map[string]*tracing.TrieNodeChange{
		"": {
			Prev: &trienode.Node{Blob: []byte{0x01}},
			New:  &trienode.Node{Blob: []byte{0x02}},
		}, // modified at depth 0
		"0": {
			New: &trienode.Node{Blob: []byte{0x01}},
		}, // created at depth 1
		"1": {
			Prev: &trienode.Node{Blob: []byte{0x01}},
		}, // deleted at depth 1
		"0a": {
			Prev: &trienode.Node{Blob: []byte{0x01}},
			New:  &trienode.Node{Blob: []byte{0x02}},
		}, // modified at depth 2
		"1b": {
			Prev: &trienode.Node{Blob: []byte{0x01}},
		}, // deleted at depth 2
	}

	created, deleted := calculateDepthCountsByType(pathMap)

	// Created/Modified: "" (depth 0), "0" (depth 1), "0a" (depth 2)
	expectedCreated := [65]int64{1, 1, 1}
	for i := 0; i < 65; i++ {
		if created[i] != expectedCreated[i] {
			t.Errorf("created depth %d: got %d, want %d", i, created[i], expectedCreated[i])
		}
	}

	// Deleted: "1" (depth 1), "1b" (depth 2)
	expectedDeleted := [65]int64{0, 1, 1}
	for i := 0; i < 65; i++ {
		if deleted[i] != expectedDeleted[i] {
			t.Errorf("deleted depth %d: got %d, want %d", i, deleted[i], expectedDeleted[i])
		}
	}
}
