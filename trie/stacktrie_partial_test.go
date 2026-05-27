// Copyright 2026 The go-ethereum Authors
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

package trie

import (
	"bytes"
	"sort"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/rlp"
)

// mkKey builds a 32-byte key from a leading hex string, right-padded with zeros
// (e.g. "3a" -> 0x3a000...0). The first nibble is prefixHex[0].
func mkKey(prefixHex string) []byte {
	return common.HexToHash(prefixHex + strings.Repeat("0", 64-len(prefixHex))).Bytes()
}

type nodeRec struct {
	hash common.Hash
	blob []byte
}

// collect builds a trie via the given updater and records every committed node
// keyed by its path.
func collect(update func(onNode OnTrieNode)) map[string]nodeRec {
	nodes := make(map[string]nodeRec)
	update(func(path []byte, hash common.Hash, blob []byte) {
		nodes[string(path)] = nodeRec{hash, common.CopyBytes(blob)}
	})
	return nodes
}

// nodeKind decodes a node blob into "branch", "extension" or "leaf".
func nodeKind(t *testing.T, blob []byte) string {
	t.Helper()
	elems, err := decodeNodeElements(blob)
	if err != nil {
		t.Fatalf("decode node: %v", err)
	}
	switch len(elems) {
	case 17:
		return "branch"
	case 2:
		key, _, err := rlp.SplitString(elems[0])
		if err != nil {
			t.Fatalf("split key: %v", err)
		}
		if hasTerm(compactToHex(key)) {
			return "leaf"
		}
		return "extension"
	default:
		t.Fatalf("unexpected element count %d", len(elems))
		return ""
	}
}

// TestPartialStackTrieMatchesFullSubtree proves that, for every shape the
// partition subtree root can take, the nodes emitted by a PartialStackTrie for
// partition n are byte-for-byte identical (path, hash, blob) to the [n]-subtree
// of the full trie built from the same keys.
func TestPartialStackTrieMatchesFullSubtree(t *testing.T) {
	const n = byte(3)

	// A single key in another partition (first nibble 9 > 3, so it sorts last)
	// forces the full trie's root to be a branch, giving a clean [n]-subtree.
	otherKey := mkKey("9")
	otherVal := bytes.Repeat([]byte{0xff}, 32)

	cases := []struct {
		name     string
		keys     []string // partition-n key prefixes (first nibble must be 3)
		wantRoot string   // expected shape of the partition subtree root
	}{
		{"single-leaf", []string{"3abc"}, "leaf"},
		{"branch-root", []string{"30", "37", "3a"}, "branch"},
		{"extension-root", []string{"3110", "3115", "311a"}, "extension"},
		{"mixed", []string{"30", "3105", "310a", "3f00", "3f0f"}, "branch"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Partition-n (key, value) pairs, sorted ascending as StackTrie requires.
			type kv struct{ k, v []byte }
			pairs := make([]kv, len(tc.keys))
			for i, p := range tc.keys {
				pairs[i] = kv{mkKey(p), bytes.Repeat([]byte{byte(i + 1)}, 32)}
			}
			sort.Slice(pairs, func(i, j int) bool { return bytes.Compare(pairs[i].k, pairs[j].k) < 0 })

			// Reference: full trie over the partition-n keys plus the other-partition key.
			full := collect(func(onNode OnTrieNode) {
				st := NewStackTrie(onNode)
				for _, p := range pairs {
					if err := st.Update(p.k, p.v); err != nil {
						t.Fatalf("full update: %v", err)
					}
				}
				if err := st.Update(otherKey, otherVal); err != nil {
					t.Fatalf("full update (other): %v", err)
				}
				st.Hash()
			})

			// Subject: PartialStackTrie over just the partition-n keys.
			var partRoot common.Hash
			part := collect(func(onNode OnTrieNode) {
				pst := NewPartialStackTrie(n, onNode)
				for _, p := range pairs {
					if err := pst.Update(p.k, p.v); err != nil {
						t.Fatalf("partial update: %v", err)
					}
				}
				partRoot = pst.Hash()
			})

			// The subtree root must live at path [n] in the full trie (i.e. it is
			// hash-referenced, not inlined) and its hash must match Hash().
			rootRec, ok := full[string([]byte{n})]
			if !ok {
				t.Fatalf("full trie has no node at path [%d]", n)
			}
			if rootRec.hash != partRoot {
				t.Fatalf("partition root %x != full subtree root %x", partRoot, rootRec.hash)
			}
			if got := nodeKind(t, rootRec.blob); got != tc.wantRoot {
				t.Fatalf("subtree root kind = %s, want %s", got, tc.wantRoot)
			}

			// Every full-trie node under [n] must equal the partition's node, and
			// the partition must emit no node outside [n].
			want := make(map[string]nodeRec)
			for p, rec := range full {
				if len(p) >= 1 && p[0] == n {
					want[p] = rec
				}
			}
			if len(want) != len(part) {
				t.Fatalf("node count: full subtree=%d, partition=%d", len(want), len(part))
			}
			for p, rec := range want {
				got, ok := part[p]
				if !ok {
					t.Fatalf("partition missing node at path %x", []byte(p))
				}
				if got.hash != rec.hash || !bytes.Equal(got.blob, rec.blob) {
					t.Fatalf("node mismatch at path %x", []byte(p))
				}
			}
		})
	}
}

// TestPartialStackTrieWrongNibble checks the guard that rejects a key whose
// leading nibble does not belong to the partition.
func TestPartialStackTrieWrongNibble(t *testing.T) {
	pst := NewPartialStackTrie(3, nil)
	if err := pst.Update(mkKey("4abc"), []byte{0x01}); err == nil {
		t.Fatal("expected error for key outside the partition, got nil")
	}
}
