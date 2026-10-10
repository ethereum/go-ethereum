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

	"github.com/ethereum/go-ethereum/trie/archive"
)

// BenchmarkResolveExpiredNode measures the full expired-node resolution path:
// archive read, record decode, subtree reconstruction, and hash verification.
// This is the hot path of every state access that touches an archived subtree.
func BenchmarkResolveExpiredNode(b *testing.B) {
	// A realistic height-3 subtree: 16 branches of 4 leaves each.
	var records []*archive.Record
	for i := range 16 {
		for l := range 4 {
			records = append(records, &archive.Record{
				Path:  []byte{byte(i), byte(l), 16},
				Value: bytes.Repeat([]byte(fmt.Sprintf("v%x%x", i, l)), 12),
			})
		}
	}
	tmpDir := b.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "geth"), 0755); err != nil {
		b.Fatal(err)
	}
	writer, err := archive.NewArchiveWriter(filepath.Join(tmpDir, "geth", "nodearchive"))
	if err != nil {
		b.Fatal(err)
	}
	offset, size, err := writer.WriteSubtree(records)
	if err != nil {
		b.Fatal(err)
	}
	writer.Close()
	oldDir := archive.ArchiveDataDir
	archive.ArchiveDataDir = tmpDir
	b.Cleanup(func() { archive.ArchiveDataDir = oldDir })

	// Expected subtree hash, as the parent reference would provide it.
	rebuilt, err := archiveRecordsToNode(records, nil)
	if err != nil {
		b.Fatal(err)
	}
	h := newHasher(false)
	expected := hashNode(h.hash(rebuilt, true))
	returnHasherToPool(h)

	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		n := &expiredNode{offset: offset, size: size, cachedHash: expected}
		if _, err := resolveExpiredNodeData(n); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkResolveExpiredNodeForKey measures the key-driven (pruned)
// resolution path used by get/insert/delete during block processing.
func BenchmarkResolveExpiredNodeForKey(b *testing.B) {
	var records []*archive.Record
	for i := range 16 {
		for l := range 4 {
			records = append(records, &archive.Record{
				Path:  []byte{byte(i), byte(l), 16},
				Value: bytes.Repeat([]byte(fmt.Sprintf("v%x%x", i, l)), 12),
			})
		}
	}
	tmpDir := b.TempDir()
	if err := os.MkdirAll(filepath.Join(tmpDir, "geth"), 0755); err != nil {
		b.Fatal(err)
	}
	writer, err := archive.NewArchiveWriter(filepath.Join(tmpDir, "geth", "nodearchive"))
	if err != nil {
		b.Fatal(err)
	}
	offset, size, err := writer.WriteSubtree(records)
	if err != nil {
		b.Fatal(err)
	}
	writer.Close()
	oldDir := archive.ArchiveDataDir
	archive.ArchiveDataDir = tmpDir
	b.Cleanup(func() { archive.ArchiveDataDir = oldDir })

	rebuilt, err := archiveRecordsToNode(records, nil)
	if err != nil {
		b.Fatal(err)
	}
	h := newHasher(false)
	expected := hashNode(h.hash(rebuilt, true))
	returnHasherToPool(h)

	relKey := []byte{0x05, 0x02, 16}
	b.ReportAllocs()
	b.ResetTimer()
	for range b.N {
		n := &expiredNode{offset: offset, size: size, cachedHash: expected}
		if _, err := resolveExpiredNodeForKey(n, relKey); err != nil {
			b.Fatal(err)
		}
	}
}
