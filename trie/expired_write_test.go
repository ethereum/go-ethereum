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
	"github.com/ethereum/go-ethereum/trie/archive"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// buildArchivedTrie constructs the balanced 256-key trie from
// TestArchiverProcessTrieAllSubtrees, archives all sixteen height-3 subtrees,
// and returns the root plus a NodeDatabase over the raw disk DB. A reference
// un-archived copy of the same trie content is returned as well.
func buildArchivedTrie(t *testing.T) (common.Hash, *rawNodeDatabase, map[string][]byte) {
	t.Helper()
	values := make(map[string][]byte)
	tr := NewEmpty(nil)
	for i := range 4 {
		for j := range 4 {
			for k := range 4 {
				for l := range 4 {
					key := []byte{byte(i<<4 | j), byte(k<<4 | l)}
					val := bytes.Repeat([]byte(fmt.Sprintf("v%x", key)), 10)
					tr.MustUpdate(key, val)
					values[string(key)] = val
				}
			}
		}
	}
	root, nodes := tr.Commit(false)

	diskdb := rawdb.NewMemoryDatabase()
	for owner, set := range trienode.NewWithNodeSet(nodes).Sets {
		for path, n := range set.Nodes {
			rawdb.WriteTrieNode(diskdb, owner, []byte(path), n.Hash, n.Blob, rawdb.PathScheme)
		}
	}

	tmpDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "geth"), 0755); err != nil {
		t.Fatal(err)
	}
	writer, err := archive.NewArchiveWriter(filepath.Join(tmpDir, "geth", "nodearchive"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { writer.Close() })
	oldDir := archive.ArchiveDataDir
	archive.ArchiveDataDir = tmpDir
	t.Cleanup(func() { archive.ArchiveDataDir = oldDir })

	ndb := &rawNodeDatabase{db: diskdb}
	archTrie, err := New(TrieID(root), ndb)
	if err != nil {
		t.Fatal(err)
	}
	archiver := NewArchiver(diskdb, ndb, writer, 0, false)
	if err := archiver.processTrie(common.Hash{}, archTrie); err != nil {
		t.Fatal(err)
	}
	if n, _, _ := archiver.Stats(); n != 16 {
		t.Fatalf("expected 16 archived subtrees, got %d", n)
	}
	return root, ndb, values
}

// applyOps applies the given (key, value) ops in order; empty value = delete.
type writeOp struct {
	key []byte
	val []byte // nil => delete
}

func applyOps(t *testing.T, tr *Trie, ops []writeOp) common.Hash {
	t.Helper()
	for _, op := range ops {
		var err error
		if op.val == nil {
			err = tr.Delete(op.key)
		} else {
			err = tr.Update(op.key, op.val)
		}
		if err != nil {
			t.Fatalf("op on %x failed: %v", op.key, err)
		}
	}
	return tr.Hash()
}

// referenceRoot computes the expected root by replaying the full content plus
// ops into a fresh in-memory trie that never saw archival.
func referenceRoot(t *testing.T, values map[string][]byte, ops []writeOp) common.Hash {
	t.Helper()
	content := make(map[string][]byte, len(values))
	for k, v := range values {
		content[k] = v
	}
	for _, op := range ops {
		if op.val == nil {
			delete(content, string(op.key))
		} else {
			content[string(op.key)] = op.val
		}
	}
	ref := NewEmpty(nil)
	for k, v := range content {
		ref.MustUpdate([]byte(k), v)
	}
	return ref.Hash()
}

// TestExpiredWriteOrderIndependence verifies that updates and deletes applied
// to a trie containing expired (archived) subtrees produce the canonical root
// regardless of operation order. Order-dependent or non-canonical results
// reproduce the mainnet block-import state-root mismatch.
func TestExpiredWriteOrderIndependence(t *testing.T) {
	root, ndb, values := buildArchivedTrie(t)

	newVal := bytes.Repeat([]byte("XY"), 20)
	cases := []struct {
		name string
		ops  []writeOp
	}{
		{"single update", []writeOp{{[]byte{0x00, 0x00}, newVal}}},
		{"single delete", []writeOp{{[]byte{0x00, 0x00}, nil}}},
		{"update then delete same subtree", []writeOp{
			{[]byte{0x00, 0x00}, newVal}, {[]byte{0x00, 0x11}, nil}}},
		{"delete then update same subtree", []writeOp{
			{[]byte{0x00, 0x11}, nil}, {[]byte{0x00, 0x00}, newVal}}},
		{"update then delete across subtrees", []writeOp{
			{[]byte{0x00, 0x00}, newVal}, {[]byte{0x11, 0x11}, nil}}},
		{"delete then update across subtrees", []writeOp{
			{[]byte{0x11, 0x11}, nil}, {[]byte{0x00, 0x00}, newVal}}},
		{"delete whole subtree then update sibling", []writeOp{
			{[]byte{0x00, 0x00}, nil}, {[]byte{0x00, 0x01}, nil}, {[]byte{0x00, 0x02}, nil}, {[]byte{0x00, 0x03}, nil},
			{[]byte{0x00, 0x10}, nil}, {[]byte{0x00, 0x11}, nil}, {[]byte{0x00, 0x12}, nil}, {[]byte{0x00, 0x13}, nil},
			{[]byte{0x00, 0x20}, nil}, {[]byte{0x00, 0x21}, nil}, {[]byte{0x00, 0x22}, nil}, {[]byte{0x00, 0x23}, nil},
			{[]byte{0x00, 0x30}, nil}, {[]byte{0x00, 0x31}, nil}, {[]byte{0x00, 0x32}, nil}, {[]byte{0x00, 0x33}, nil},
			{[]byte{0x01, 0x00}, newVal}}},
		{"insert new key extending expired subtree", []writeOp{
			{[]byte{0x00, 0x00, 0xaa}, newVal}}},
		{"read then update", []writeOp{ // mimic get-resolution before write
			{[]byte{0x22, 0x22}, newVal}}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tr, err := New(TrieID(root), ndb)
			if err != nil {
				t.Fatal(err)
			}
			// Force get-resolution first for the "read then update" case,
			// mirroring block processing where reads precede writes.
			for _, op := range tc.ops {
				tr.MustGet(op.key)
			}
			got := applyOps(t, tr, tc.ops)
			want := referenceRoot(t, values, tc.ops)
			if got != want {
				t.Errorf("root mismatch:\n got %x\nwant %x", got, want)
			}
		})
	}
}

// TestExpiredCollapseMergesShortNode reproduces the mainnet block-import
// state-root mismatch: deleting the last sibling of an expired subtree whose
// root is a shortNode must merge the extension keys through the archive.
// Before the resolve() fix, the collapse type-checked the raw expiredNode,
// skipped the merge, and produced a non-canonical shortNode chain.
func TestExpiredCollapseMergesShortNode(t *testing.T) {
	values := make(map[string][]byte)
	tr := NewEmpty(nil)
	put := func(key []byte) {
		val := bytes.Repeat([]byte(fmt.Sprintf("v%x", key)), 10)
		tr.MustUpdate(key, val)
		values[string(key)] = val
	}
	// Height-3 subtree rooted at path [0,0] with a shortNode root:
	// all its leaves share nibble 'a' at position 3.
	for l := range 4 {
		put([]byte{0x00, byte(0xa0 | l)})
	}
	// Sibling leaf under [0,1] whose deletion collapses the fullNode at [0].
	put([]byte{0x01, 0x23})
	// Ballast so the trie root is a fullNode with other children.
	put([]byte{0x10, 0x00})
	put([]byte{0x20, 0x00})
	root, nodes := tr.Commit(false)

	diskdb := rawdb.NewMemoryDatabase()
	for owner, set := range trienode.NewWithNodeSet(nodes).Sets {
		for path, n := range set.Nodes {
			rawdb.WriteTrieNode(diskdb, owner, []byte(path), n.Hash, n.Blob, rawdb.PathScheme)
		}
	}
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

	ndb := &rawNodeDatabase{db: diskdb}
	archTrie, err := New(TrieID(root), ndb)
	if err != nil {
		t.Fatal(err)
	}
	archiver := NewArchiver(diskdb, ndb, writer, 0, false)
	if err := archiver.processTrie(common.Hash{}, archTrie); err != nil {
		t.Fatal(err)
	}
	if n, _, _ := archiver.Stats(); n != 1 {
		t.Fatalf("expected exactly 1 archived subtree, got %d", n)
	}
	if blob := rawdb.ReadAccountTrieNode(diskdb, []byte{0, 0}); len(blob) == 0 || blob[0] != expiredNodeMarker {
		t.Fatal("subtree at [0,0] not archived")
	}

	// Delete the sibling: the fullNode at [0] collapses onto the expired
	// subtree, requiring an extension-key merge through the archive.
	ops := []writeOp{{[]byte{0x01, 0x23}, nil}}
	live, err := New(TrieID(root), ndb)
	if err != nil {
		t.Fatal(err)
	}
	got := applyOps(t, live, ops)
	want := referenceRoot(t, values, ops)
	if got != want {
		t.Errorf("root mismatch after collapse:\n got %x\nwant %x", got, want)
	}
}

// TestExpiredRepeatResolutionCommit covers the interaction between the
// verified-offset cache and the committer: the verification hash walk is
// what stamps flags.hash on interior subtree nodes, and the committer
// treats hash==nil as "embedded small node". A repeat resolution (skipped
// verification) must therefore not stamp the subtree root, otherwise the
// main hasher short-circuits and the committer embeds oversized unhashed
// leaves inline into parent blobs, corrupting the database.
func TestExpiredRepeatResolutionCommit(t *testing.T) {
	root, ndb, values := buildArchivedTrie(t)

	// First resolution: full verification, populates the verified cache.
	tr1, err := New(TrieID(root), ndb)
	if err != nil {
		t.Fatal(err)
	}
	tr1.MustGet([]byte{0x00, 0x00})

	// Second resolution of the same subtree in a fresh trie, via a READ:
	// verification is skipped and (pre-fix) the subtree root got stamped
	// while its interiors stayed unhashed. Then modify a DIFFERENT
	// subtree, so the read-resolved one is committed (markSubtreeDirty)
	// without any insert resetting its stamped root.
	tr2, err := New(TrieID(root), ndb)
	if err != nil {
		t.Fatal(err)
	}
	tr2.MustGet([]byte{0x00, 0x00})
	// Write into the NEIGHBOURING subtree [0,1]: it shares the fullNode
	// ancestor at path [0] with the read-resolved subtree [0,0], so the
	// commit traversal reaches the stamped subtree through the dirtied
	// ancestor chain.
	newVal := bytes.Repeat([]byte("Z"), 40)
	tr2.MustUpdate([]byte{0x01, 0x23}, newVal)
	newRoot, nodes := tr2.Commit(false)

	// The committed root must match a never-archived reference.
	want := referenceRoot(t, values, []writeOp{{[]byte{0x01, 0x23}, newVal}})
	if newRoot != want {
		t.Errorf("committed root mismatch: got %x want %x", newRoot, want)
	}
	// Every committed blob must round-trip decode: oversized inline
	// children make the parent blob undecodable.
	for _, set := range trienode.NewWithNodeSet(nodes).Sets {
		for path, n := range set.Nodes {
			if n.IsDeleted() {
				continue
			}
			if _, err := decodeNode(n.Hash[:], n.Blob); err != nil {
				t.Errorf("committed blob at path %x is undecodable: %v", path, err)
			}
		}
	}
}
