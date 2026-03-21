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
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/archive"
)

// setupTestArchive creates a temporary archive directory with an archive file
// containing the given records, and configures archive.ArchiveDataDir to point
// to it. It returns the offset and size of the written data, and a cleanup function.
func setupTestArchive(t *testing.T, records []*archive.Record) (offset, size uint64, cleanup func()) {
	t.Helper()
	tmpDir := t.TempDir()
	gethDir := filepath.Join(tmpDir, "geth")
	if err := os.MkdirAll(gethDir, 0755); err != nil {
		t.Fatal(err)
	}

	writer, err := archive.NewArchiveWriter(filepath.Join(gethDir, "nodearchive"))
	if err != nil {
		t.Fatal(err)
	}

	offset, size, err = writer.WriteSubtree(records)
	if err != nil {
		writer.Close()
		t.Fatal(err)
	}
	writer.Close()

	oldDir := archive.ArchiveDataDir
	archive.ArchiveDataDir = tmpDir

	return offset, size, func() {
		archive.ArchiveDataDir = oldDir
	}
}

func TestExpiredNodeEncodeDecode(t *testing.T) {
	testCases := []struct {
		offset uint64
		size   uint64
	}{
		{0, 0},
		{1, 100},
		{255, 1024},
		{256, 4096},
		{1 << 16, 1 << 20},
		{1 << 32, 1 << 32},
		{1<<64 - 1, 1<<64 - 1},
	}

	for _, tc := range testCases {
		original := &expiredNode{offset: tc.offset, size: tc.size}

		w := rlp.NewEncoderBuffer(nil)
		original.encode(w)
		encoded := w.ToBytes()
		w.Flush()

		decoded, err := decodeNodeUnsafe(nil, encoded)
		if err != nil {
			t.Fatalf("failed to decode expired node with offset %d, size %d: %v", tc.offset, tc.size, err)
		}

		expNode, ok := decoded.(*expiredNode)
		if !ok {
			t.Fatalf("decoded node is not an expired node, got %T", decoded)
		}

		if expNode.offset != original.offset {
			t.Errorf("offset mismatch: got %d, want %d", expNode.offset, original.offset)
		}
		if expNode.size != original.size {
			t.Errorf("size mismatch: got %d, want %d", expNode.size, original.size)
		}
	}
}

func TestExpiredNodeEncodedFormat(t *testing.T) {
	node := &expiredNode{offset: 0x0102030405060708, size: 0x1112131415161718}

	w := rlp.NewEncoderBuffer(nil)
	node.encode(w)
	encoded := w.ToBytes()
	w.Flush()

	expected := []byte{
		0x00,
		0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
		0x11, 0x12, 0x13, 0x14, 0x15, 0x16, 0x17, 0x18,
	}
	if !bytes.Equal(encoded, expected) {
		t.Errorf("encoded format mismatch: got %x, want %x", encoded, expected)
	}
}

func TestExpiredNodeFstring(t *testing.T) {
	node := &expiredNode{offset: 12345, size: 6789}
	s := node.fstring("")
	if s != "<expired: offset=12345, size=6789> " {
		t.Errorf("fstring mismatch: got %q", s)
	}
}

func TestExpiredNodeCache(t *testing.T) {
	node := &expiredNode{offset: 100}
	hash, dirty := node.cache()
	if hash != nil {
		t.Error("expected nil hash from expired node cache")
	}
	if !dirty {
		t.Error("expected dirty=true from expired node cache")
	}
}

func TestExpiredNodeInvalidLength(t *testing.T) {
	invalidCases := [][]byte{
		{0x00},
		{0x00, 0x01},
		{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08},
		{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f},
		{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0a, 0x0b, 0x0c, 0x0d, 0x0e, 0x0f, 0x10, 0x11},
	}

	for _, buf := range invalidCases {
		_, err := decodeNodeUnsafe(nil, buf)
		if err == nil {
			t.Errorf("expected error for buffer length %d, got nil", len(buf))
		}
	}
}

func TestExpiredNodeNoArchiveFile(t *testing.T) {
	// When no archive file exists, Get should return an error
	tmpDir := t.TempDir()
	gethDir := filepath.Join(tmpDir, "geth")
	if err := os.MkdirAll(gethDir, 0755); err != nil {
		t.Fatal(err)
	}

	oldDir := archive.ArchiveDataDir
	archive.ArchiveDataDir = tmpDir
	defer func() { archive.ArchiveDataDir = oldDir }()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: 100, size: 50}

	_, err := tr.Get([]byte("key"))
	if err == nil {
		t.Error("expected error when archive file doesn't exist")
	}
}

func TestExpiredNodeWithResolver(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("testvalue")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	val, err := tr.Get([]byte{0x12})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val) != "testvalue" {
		t.Errorf("value mismatch: got %q, want %q", val, "testvalue")
	}
}

func TestExpiredNodeCopy(t *testing.T) {
	original := &expiredNode{
		offset:          12345,
		size:            6789,
		archiveResolver: archive.ArchivedNodeResolver,
	}

	copied := copyNode(original)
	copiedExp, ok := copied.(*expiredNode)
	if !ok {
		t.Fatalf("copied node is not an expired node, got %T", copied)
	}

	if copiedExp.offset != original.offset {
		t.Errorf("offset mismatch: got %d, want %d", copiedExp.offset, original.offset)
	}

	if copiedExp.size != original.size {
		t.Errorf("size mismatch: got %d, want %d", copiedExp.size, original.size)
	}

	if copiedExp.archiveResolver == nil {
		t.Error("archive resolver was not copied")
	}
}

func TestArchiveRecordsToNodeEmpty(t *testing.T) {
	_, err := archiveRecordsToNode([]*archive.Record{})
	if !errors.Is(err, archive.EmptyArchiveRecord) {
		t.Errorf("expected EmptyArchiveRecord error, got %v", err)
	}

	_, err = archiveRecordsToNode(nil)
	if !errors.Is(err, archive.EmptyArchiveRecord) {
		t.Errorf("expected EmptyArchiveRecord error for nil slice, got %v", err)
	}
}

func TestArchiveRecordsToNodeMultiple(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 16}, Value: []byte("value1")},
		{Path: []byte{0x02, 16}, Value: []byte("value2")},
	}

	node, err := archiveRecordsToNode(records)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fn, ok := node.(*fullNode)
	if !ok {
		t.Fatalf("expected fullNode, got %T", node)
	}

	if fn.Children[0x01] == nil {
		t.Error("expected child at index 0x01")
	}
	if fn.Children[0x02] == nil {
		t.Error("expected child at index 0x02")
	}
}

func TestExpiredNodeGetMultipleRecords(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("value1")},
		{Path: []byte{0x04, 0x05, 16}, Value: []byte("value2")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	val, err := tr.Get([]byte{0x12})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val) != "value1" {
		t.Errorf("value mismatch: got %q, want %q", val, "value1")
	}

	tr2 := NewEmpty(nil)
	tr2.root = &expiredNode{offset: offset, size: size}

	val2, err := tr2.Get([]byte{0x45})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val2) != "value2" {
		t.Errorf("value mismatch: got %q, want %q", val2, "value2")
	}
}

func TestExpiredNodeGetKeyNotFound(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("value1")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	val, err := tr.Get([]byte{0xff, 0xff})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil value for non-existent key, got %q", val)
	}
}

func TestExpiredNodeGetPathMismatch(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("testvalue")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	val, err := tr.Get([]byte{0x19})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil value when leaf key doesn't match, got %q", val)
	}
}

func TestExpiredNodeInsert(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("existing")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	err := tr.Update([]byte{0x45}, []byte("newvalue"))
	if err != nil {
		t.Fatalf("unexpected error on insert: %v", err)
	}

	val, err := tr.Get([]byte{0x45})
	if err != nil {
		t.Fatalf("unexpected error on get: %v", err)
	}
	if string(val) != "newvalue" {
		t.Errorf("value mismatch: got %q, want %q", val, "newvalue")
	}
}

func TestExpiredNodeUpdate(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("oldvalue")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	err := tr.Update([]byte{0x12}, []byte("newvalue"))
	if err != nil {
		t.Fatalf("unexpected error on update: %v", err)
	}

	val, err := tr.Get([]byte{0x12})
	if err != nil {
		t.Fatalf("unexpected error on get: %v", err)
	}
	if string(val) != "newvalue" {
		t.Errorf("value mismatch: got %q, want %q", val, "newvalue")
	}
}

func TestExpiredNodeDelete(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("value1")},
		{Path: []byte{0x04, 0x05, 16}, Value: []byte("value2")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	err := tr.Delete([]byte{0x12})
	if err != nil {
		t.Fatalf("unexpected error on delete: %v", err)
	}

	val, err := tr.Get([]byte{0x12})
	if err != nil {
		t.Fatalf("unexpected error on get after delete: %v", err)
	}
	if val != nil {
		t.Errorf("expected nil after delete, got %q", val)
	}

	val2, err := tr.Get([]byte{0x45})
	if err != nil {
		t.Fatalf("unexpected error getting other key: %v", err)
	}
	if string(val2) != "value2" {
		t.Errorf("other value should still exist: got %q, want %q", val2, "value2")
	}
}

func TestTrieCopyPreservesArchiveResolver(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("testvalue")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	trCopy := tr.Copy()

	val, err := trCopy.Get([]byte{0x12})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if string(val) != "testvalue" {
		t.Errorf("value mismatch: got %q, want %q", val, "testvalue")
	}
}

func TestWalkWithExpiredNodes(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("value1")},
		{Path: []byte{0x04, 0x05, 16}, Value: []byte("value2")},
		{Path: []byte{0x07, 0x08, 16}, Value: []byte("value3")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	var leaves []string
	stats, err := tr.Walk(func(path []byte, value []byte) error {
		leaves = append(leaves, string(value))
		return nil
	})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if stats.Leaves != 3 {
		t.Errorf("expected 3 leaves, got %d", stats.Leaves)
	}
	if stats.ExpiredResolved != 1 {
		t.Errorf("expected 1 expired resolved, got %d", stats.ExpiredResolved)
	}
	// Verify all values were visited
	expected := map[string]bool{"value1": true, "value2": true, "value3": true}
	for _, leaf := range leaves {
		if !expected[leaf] {
			t.Errorf("unexpected leaf value: %q", leaf)
		}
		delete(expected, leaf)
	}
	if len(expected) > 0 {
		t.Errorf("missing leaves: %v", expected)
	}
}

func TestWalkEmptyTrie(t *testing.T) {
	tr := NewEmpty(nil)
	stats, err := tr.Walk(func(path []byte, value []byte) error {
		t.Error("callback should not be called for empty trie")
		return nil
	})
	if err != nil {
		t.Fatalf("Walk failed: %v", err)
	}
	if stats.Leaves != 0 || stats.ExpiredResolved != 0 {
		t.Errorf("expected zero stats for empty trie, got leaves=%d expired=%d", stats.Leaves, stats.ExpiredResolved)
	}
}

func TestWalkCallbackError(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("value1")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	testErr := errors.New("test error")
	_, err := tr.Walk(func(path []byte, value []byte) error {
		return testErr
	})
	if !errors.Is(err, testErr) {
		t.Fatalf("expected test error, got %v", err)
	}
}

// TestExpiredNodeResolvedSubtreeDirty verifies that when an expired node is
// resolved and a sibling leaf is modified, the commit captures ALL resolved
// nodes (not just the modified path). Without this fix, resolved-but-unmodified
// nodes would be lost: not in the diff layer (clean) and not in the raw DB
// (deleted by archiver).
func TestExpiredNodeResolvedSubtreeDirty(t *testing.T) {
	// Use large values (>32 bytes) so leaf nodes are NOT embedded in
	// their parent. This matches production storage tries where
	// intermediate nodes are large enough to be stored independently.
	bigVal1 := bytes.Repeat([]byte("A"), 40)
	bigVal2 := bytes.Repeat([]byte("B"), 40)

	// Create an archive with records under different branches.
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: bigVal1},
		{Path: []byte{0x04, 0x05, 16}, Value: bigVal2},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	// Insert a value that goes through one branch of the resolved subtree.
	// This modifies path [1, ...] but leaves path [4, ...] unmodified.
	if err := tr.Update([]byte{0x12}, bytes.Repeat([]byte("C"), 40)); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	// Commit the trie. The NodeSet should be non-nil because we modified data.
	_, nodes := tr.Commit(false)
	if nodes == nil {
		t.Fatal("expected non-nil NodeSet after modifying expired subtree")
	}

	// The resolved-but-unmodified sibling (path [4, 5]) should also be
	// captured in the NodeSet, because markSubtreeDirty ensures all resolved
	// nodes are dirty. Count the nodes to verify.
	nodeCount := len(nodes.Nodes)
	// We expect at least 3 nodes: the root, the modified branch, and the
	// sibling branch. The exact count depends on trie structure.
	if nodeCount < 3 {
		t.Errorf("expected at least 3 nodes in NodeSet (root + modified + sibling), got %d", nodeCount)
	}
}

// TestMarkSubtreeDirty verifies that markSubtreeDirty correctly sets the dirty
// flag on all nodes in a subtree while preserving cached hashes.
func TestMarkSubtreeDirty(t *testing.T) {
	// Build a small trie structure
	leaf1 := &shortNode{Key: []byte{1, 16}, Val: valueNode("v1")}
	leaf2 := &shortNode{Key: []byte{2, 16}, Val: valueNode("v2")}
	branch := &fullNode{}
	branch.Children[1] = leaf1
	branch.Children[2] = leaf2

	// Set hash but not dirty (as if loaded from DB)
	branch.flags = nodeFlag{hash: hashNode("testhash"), dirty: false}
	leaf1.flags = nodeFlag{hash: hashNode("hash1"), dirty: false}
	leaf2.flags = nodeFlag{hash: hashNode("hash2"), dirty: false}

	markSubtreeDirty(branch)

	// All nodes should be dirty
	if !branch.flags.dirty {
		t.Error("branch should be dirty")
	}
	if !leaf1.flags.dirty {
		t.Error("leaf1 should be dirty")
	}
	if !leaf2.flags.dirty {
		t.Error("leaf2 should be dirty")
	}

	// Hashes should be preserved
	if !bytes.Equal(branch.flags.hash, hashNode("testhash")) {
		t.Error("branch hash should be preserved")
	}
	if !bytes.Equal(leaf1.flags.hash, hashNode("hash1")) {
		t.Error("leaf1 hash should be preserved")
	}
	if !bytes.Equal(leaf2.flags.hash, hashNode("hash2")) {
		t.Error("leaf2 hash should be preserved")
	}
}

func TestExpiredNodeGetNode(t *testing.T) {
	records := []*archive.Record{
		{Path: []byte{0x01, 0x02, 16}, Value: []byte("testvalue")},
	}
	offset, size, cleanup := setupTestArchive(t, records)
	defer cleanup()

	tr := NewEmpty(nil)
	tr.root = &expiredNode{offset: offset, size: size}

	_, _, err := tr.GetNode(hexToCompact([]byte{0x01, 0x02}))
	if err != nil && err.Error() != "non-consensus node" {
		t.Fatalf("unexpected error: %v", err)
	}
}
