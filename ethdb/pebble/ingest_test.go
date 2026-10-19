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

package pebble

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

func ingestKey(i int) []byte   { return []byte(fmt.Sprintf("key-%04d", i)) }
func ingestValue(i int) []byte { return bytes.Repeat([]byte{byte(i)}, 64) }

// Tests that the ingest batch commits the same content as a normal batch, with
// the latest operation of a key taking precedence.
func TestIngestBatch(t *testing.T) {
	db, err := New(t.TempDir(), 16, 16, "", false)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	// Split the ingested content into tiny sstables
	db.ingestOpts.TargetFileSizes[0] = 1024

	expect := make(map[string][]byte)
	for i := 0; i < 100; i++ {
		db.Put(ingestKey(i), ingestValue(i))
		expect[string(ingestKey(i))] = ingestValue(i)
	}
	var (
		batch = db.NewIngestBatch(0)
		ref   = memorydb.New()
		put   = func(k, v []byte) {
			batch.Put(k, v)
			expect[string(k)] = v
		}
		del = func(k []byte) {
			batch.Delete(k)
			delete(expect, string(k))
		}
	)
	// Overwrite and delete the existing keys, and insert the new ones
	for i := 0; i < 50; i++ {
		put(ingestKey(i), ingestValue(i+1))
	}
	for i := 50; i < 100; i++ {
		del(ingestKey(i))
	}
	for i := 300; i >= 100; i-- {
		put(ingestKey(i), ingestValue(i))
	}
	// Operate on the same keys multiple times
	put(ingestKey(400), ingestValue(1))
	del(ingestKey(400))
	del(ingestKey(401))
	put(ingestKey(401), ingestValue(2))
	put(ingestKey(402), ingestValue(3))
	put(ingestKey(402), ingestValue(4))

	if err := batch.DeleteRange(ingestKey(0), ingestKey(1)); !errors.Is(err, errIngestRangeDeletion) {
		t.Fatalf("Unexpected range deletion error: %v", err)
	}
	if err := batch.Replay(ref); err != nil {
		t.Fatalf("Failed to replay batch: %v", err)
	}
	if err := batch.Write(); err != nil {
		t.Fatalf("Failed to write batch: %v", err)
	}
	batch.Close()

	for i := 0; i <= 402; i++ {
		key := ingestKey(i)
		have, _ := db.Get(key)
		want := expect[string(key)]
		if !bytes.Equal(have, want) {
			t.Fatalf("Unexpected value of %s, want %x, got %x", key, want, have)
		}
		replayed, _ := ref.Get(key)
		if !bytes.Equal(replayed, want) {
			t.Fatalf("Unexpected replayed value of %s, want %x, got %x", key, want, replayed)
		}
	}
	// The ingested sstables must be removed from the ingest directory
	entries, err := os.ReadDir(db.ingestDir)
	if err != nil {
		t.Fatalf("Failed to read ingest directory: %v", err)
	}
	if len(entries) != 0 {
		t.Fatalf("Unexpected leftover sstables: %d", len(entries))
	}
}

// Tests that the leftover sstables of an interrupted ingestion are removed when
// the database is opened.
func TestIngestLeftoverCleanup(t *testing.T) {
	dir := t.TempDir()
	db, err := New(dir, 16, 16, "", false)
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	db.Close()

	leftover := filepath.Join(dir, ingestDirName, "1.sst")
	if err := os.MkdirAll(filepath.Dir(leftover), 0755); err != nil {
		t.Fatalf("Failed to create ingest directory: %v", err)
	}
	if err := os.WriteFile(leftover, []byte{0x1}, 0644); err != nil {
		t.Fatalf("Failed to write leftover sstable: %v", err)
	}
	db, err = New(dir, 16, 16, "", false)
	if err != nil {
		t.Fatalf("Failed to reopen database: %v", err)
	}
	defer db.Close()

	if _, err := os.Stat(leftover); !os.IsNotExist(err) {
		t.Fatalf("Leftover sstable is not removed: %v", err)
	}
}
