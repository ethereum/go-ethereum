// Copyright 2026 go-ethereum Authors
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
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/trie/archive"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// TestExpiredWriteDifferential builds random tries, archives their height-3
// subtrees, applies random write batches, and compares the resulting root
// against a reference trie that was never archived. Any divergence is a
// consensus bug (reproduces the mainnet block-import root mismatch).
func fillBytes(rng *rand.Rand, b []byte) {
	for i := range b {
		b[i] = byte(rng.Uint64())
	}
}

func TestExpiredWriteDifferential(t *testing.T) {
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "geth"), 0755); err != nil {
		t.Fatal(err)
	}
	oldDir := archive.ArchiveDataDir
	archive.ArchiveDataDir = tmpDir
	defer func() { archive.ArchiveDataDir = oldDir }()

	for trial := range 300 {
		seed := int64(trial) + 1
		rng := rand.New(rand.NewPCG(uint64(seed), 0))

		// Build a random trie: hashed-style 32-byte keys drawn from a narrow
		// prefix space so keys share prefixes (creating extensions), plus
		// random values large enough to avoid embedding sometimes and small
		// enough to embed other times.
		nkeys := 50 + rng.IntN(400)
		values := make(map[string][]byte)
		tr := NewEmpty(nil)
		keys := make([][]byte, 0, nkeys)
		for range nkeys {
			key := make([]byte, 32)
			fillBytes(rng, key)
			// Narrow the keyspace to force shared prefixes/extension nodes.
			key[0] &= 0x33
			key[1] &= 0x11
			val := make([]byte, 1+rng.IntN(60))
			fillBytes(rng, val)
			tr.MustUpdate(key, val)
			values[string(key)] = val
			keys = append(keys, key)
		}
		root, nodes := tr.Commit(false)

		diskdb := rawdb.NewMemoryDatabase()
		for owner, set := range trienode.NewWithNodeSet(nodes).Sets {
			for path, n := range set.Nodes {
				rawdb.WriteTrieNode(diskdb, owner, []byte(path), n.Hash, n.Blob, rawdb.PathScheme)
			}
		}
		writer, err := archive.NewArchiveWriter(filepath.Join(tmpDir, "geth", "nodearchive"))
		if err != nil {
			t.Fatal(err)
		}
		ndb := &rawNodeDatabase{db: diskdb}
		archTrie, err := New(TrieID(root), ndb)
		if err != nil {
			t.Fatal(err)
		}
		archiver := NewArchiver(diskdb, ndb, writer, 0, false)
		if err := archiver.processTrie(common.Hash{}, archTrie); err != nil {
			writer.Close()
			t.Fatalf("seed %d: archival failed: %v", seed, err)
		}
		writer.Close()
		subtrees, _, _ := archiver.Stats()
		if subtrees == 0 {
			continue // nothing archived, trivial trial
		}

		// Random op batch: mix of updates to existing keys, deletes of
		// existing keys, and inserts of fresh keys (some sharing prefixes
		// with existing keys to force splits inside resolved subtrees).
		nops := 1 + rng.IntN(30)
		ops := make([]writeOp, 0, nops)
		for range nops {
			switch rng.IntN(4) {
			case 0: // update existing
				k := keys[rng.IntN(len(keys))]
				v := make([]byte, 1+rng.IntN(60))
				fillBytes(rng, v)
				ops = append(ops, writeOp{k, v})
			case 1: // delete existing
				k := keys[rng.IntN(len(keys))]
				ops = append(ops, writeOp{k, nil})
			case 2: // insert fresh key near an existing one (prefix split)
				k := append([]byte{}, keys[rng.IntN(len(keys))]...)
				k[31] ^= byte(1 + rng.IntN(255))
				v := make([]byte, 1+rng.IntN(60))
				fillBytes(rng, v)
				ops = append(ops, writeOp{k, v})
			case 3: // insert totally fresh key
				k := make([]byte, 32)
				fillBytes(rng, k)
				k[0] &= 0x33
				k[1] &= 0x11
				v := make([]byte, 1+rng.IntN(60))
				fillBytes(rng, v)
				ops = append(ops, writeOp{k, v})
			}
		}

		live, err := New(TrieID(root), ndb)
		if err != nil {
			t.Fatal(err)
		}
		// Mimic block processing: reads precede writes.
		for _, op := range ops {
			live.MustGet(op.key)
		}
		got := applyOps(t, live, ops)
		want := referenceRoot(t, values, ops)
		if got != want {
			var desc string
			for _, op := range ops {
				if op.val == nil {
					desc += fmt.Sprintf("  del %x\n", op.key)
				} else {
					desc += fmt.Sprintf("  put %x (%d bytes)\n", op.key, len(op.val))
				}
			}
			t.Fatalf("seed %d: root mismatch (subtrees=%d nkeys=%d):\n got %x\nwant %x\nops:\n%s",
				seed, subtrees, nkeys, got, want, desc)
		}
		// Commit the mutated trie and check every persisted blob decodes:
		// this catches committer-level corruption (e.g. oversized nodes
		// embedded inline) that pure root comparison cannot see.
		if _, nodes := live.Commit(false); nodes != nil {
			for _, set := range trienode.NewWithNodeSet(nodes).Sets {
				for path, n := range set.Nodes {
					if n.IsDeleted() {
						continue
					}
					if _, err := decodeNode(n.Hash[:], n.Blob); err != nil {
						t.Fatalf("seed %d: committed blob at %x is undecodable: %v", seed, path, err)
					}
				}
			}
		}
	}
}
