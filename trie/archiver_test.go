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
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/archive"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb/database"
)

// rawNodeReader resolves trie nodes straight from the raw key-value store,
// without any hash verification. This mirrors the rawDBNodeReader used by
// the archival command: expired-node blobs (marker 0x00) do not hash to the
// value referenced by their parent, so hash-checking readers would drop them.
type rawNodeReader struct {
	db ethdb.Database
}

func (r *rawNodeReader) Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error) {
	var blob []byte
	if owner == (common.Hash{}) {
		blob = rawdb.ReadAccountTrieNode(r.db, path)
	} else {
		blob = rawdb.ReadStorageTrieNode(r.db, owner, path)
	}
	if len(blob) == 0 {
		return nil, &MissingNodeError{Owner: owner, Path: path, NodeHash: hash}
	}
	return blob, nil
}

// rawNodeDatabase is a minimal database.NodeDatabase backed by the raw
// key-value store, mirroring the archival command's setup.
type rawNodeDatabase struct {
	db ethdb.Database
}

func (d *rawNodeDatabase) NodeReader(stateRoot common.Hash) (database.NodeReader, error) {
	return &rawNodeReader{db: d.db}, nil
}

// TestArchiverProcessTrieAllSubtrees builds a perfectly balanced trie
// containing sixteen height-3 sibling subtrees and verifies that processTrie
// archives every single one of them. This is a regression test for the
// iterator skip logic: archiving a subtree must not cause the iterator to
// jump over the next sibling (which previously halved the archived count).
func TestArchiverProcessTrieAllSubtrees(t *testing.T) {
	// Build a trie from 256 two-byte keys whose nibbles (i,j,k,l) each range
	// over 0..3. The resulting structure is:
	//
	//   path []       fullNode  height 5
	//   path [i]      fullNode  height 4
	//   path [i,j]    fullNode  height 3   <- 16 archivable subtree roots
	//   path [i,j,k]  fullNode  height 2
	//   path [i,j,k,l] shortNode{16} -> valueNode (40-byte values, so no
	//                  node is embedded in its parent)
	values := make(map[string][]byte)
	tr := NewEmpty(nil)
	for i := range 4 {
		for j := range 4 {
			for k := range 4 {
				for l := range 4 {
					key := []byte{byte(i<<4 | j), byte(k<<4 | l)}
					val := bytes.Repeat([]byte(fmt.Sprintf("v%x", key)), 10) // 40 bytes
					tr.MustUpdate(key, val)
					values[string(key)] = val
				}
			}
		}
	}
	root, nodes := tr.Commit(false)

	// Persist all nodes to a raw path-scheme database.
	diskdb := rawdb.NewMemoryDatabase()
	for owner, set := range trienode.NewWithNodeSet(nodes).Sets {
		for path, n := range set.Nodes {
			rawdb.WriteTrieNode(diskdb, owner, []byte(path), n.Hash, n.Blob, rawdb.PathScheme)
		}
	}

	// Set up the archive file and point the global resolver at it.
	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "geth"), 0755); err != nil {
		t.Fatal(err)
	}
	writer, err := archive.NewArchiveWriter(filepath.Join(tmpDir, "geth", "nodearchive"))
	if err != nil {
		t.Fatal(err)
	}
	defer writer.Close()
	oldDir := archive.ArchiveDataDir
	archive.ArchiveDataDir = tmpDir
	defer func() { archive.ArchiveDataDir = oldDir }()

	// Run the archiver over the account trie.
	ndb := &rawNodeDatabase{db: diskdb}
	archTrie, err := New(TrieID(root), ndb)
	if err != nil {
		t.Fatalf("failed to open trie: %v", err)
	}
	archiver := NewArchiver(diskdb, ndb, writer, 0, false)
	if err := archiver.processTrie(common.Hash{}, archTrie); err != nil {
		t.Fatalf("processTrie failed: %v", err)
	}

	// Every height-3 subtree (all sixteen [i,j] roots) must be archived —
	// none may be skipped when its sibling is archived first.
	subtrees, leaves, _ := archiver.Stats()
	if subtrees != 16 {
		t.Errorf("archived subtree count: got %d, want 16", subtrees)
	}
	if leaves != 256 {
		t.Errorf("archived leaf count: got %d, want 256", leaves)
	}
	for i := range 4 {
		for j := range 4 {
			blob := rawdb.ReadAccountTrieNode(diskdb, []byte{byte(i), byte(j)})
			if len(blob) == 0 || blob[0] != expiredNodeMarker {
				t.Errorf("subtree root [%x,%x] not replaced by expired node (blob=%x)", i, j, blob)
			}
			// Interior nodes of archived subtrees must be deleted.
			for k := range 4 {
				if blob := rawdb.ReadAccountTrieNode(diskdb, []byte{byte(i), byte(j), byte(k)}); len(blob) != 0 {
					t.Errorf("interior node [%x,%x,%x] not deleted", i, j, k)
				}
			}
		}
	}

	// All values must remain readable through archive resolution.
	readTrie, err := New(TrieID(root), ndb)
	if err != nil {
		t.Fatalf("failed to reopen trie: %v", err)
	}
	for key, want := range values {
		got, err := readTrie.Get([]byte(key))
		if err != nil {
			t.Fatalf("Get(%x) failed after archival: %v", key, err)
		}
		if !bytes.Equal(got, want) {
			t.Errorf("Get(%x): got %q, want %q", key, got, want)
		}
	}

	// A second pass must find nothing left to archive.
	secondTrie, err := New(TrieID(root), ndb)
	if err != nil {
		t.Fatalf("failed to reopen trie: %v", err)
	}
	second := NewArchiver(diskdb, ndb, writer, 0, false)
	if err := second.processTrie(common.Hash{}, secondTrie); err != nil {
		t.Fatalf("second processTrie failed: %v", err)
	}
	if n, _, _ := second.Stats(); n != 0 {
		t.Errorf("second pass archived %d subtrees, want 0", n)
	}
}
