// Copyright 2019 The go-ethereum Authors
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
package vectordb

import (
	"encoding/hex"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

func TestOpenVersion(t *testing.T) {
	dir, rmdir := createTempDir(t)
	defer rmdir()

	vectorDB, err := Open("vectordb", dir)
	if err != nil {
		t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
	}
	defer vectorDB.Close()

	if version := vectorDB.Version(); version != currentVersion {
		t.Fatalf("vectorDB.Version() = %d, want %d", version, currentVersion)
	}
}

func TestVectorDB_Repair(t *testing.T) {
	tests := []struct {
		name              string
		rawIndex, rawData func() []byte
		blobs             [][]byte
	}{
		{
			"LastIndexEntryMissingAByte",
			func() []byte {
				index := marshalIndexEntries(&indexEntry{0, 4}, &indexEntry{4, 3})
				return index[:len(index)-1]
			},
			func() []byte {
				return []byte{1, 1, 1, 1, 2, 2, 2}
			},
			[][]byte{{1, 1, 1, 1}},
		},
		{
			"DanglingIndexEntry",
			func() []byte {
				index := marshalIndexEntries(&indexEntry{0, 4}, &indexEntry{4, 3})
				return index[:len(index)-1]
			},
			func() []byte {
				return []byte{1, 1, 1, 1}
			},
			[][]byte{{1, 1, 1, 1}},
		},
		{
			"LastDataEntryMissingAByte",
			func() []byte {
				return marshalIndexEntries(&indexEntry{0, 4}, &indexEntry{4, 3})
			},
			func() []byte {
				return []byte{1, 1, 1, 1, 2, 2}
			},
			[][]byte{{1, 1, 1, 1}},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, rmdir := createTempDir(t)
			defer rmdir()

			if err := os.MkdirAll(filepath.Join(dir, "vectordb"), dbDirPerm); err != nil {
				t.Fatalf("Error creating mock database directory: %v", err)
			}

			if err := ioutil.WriteFile(filepath.Join(dir, "vectordb", indexFile), tc.rawIndex(), indexFilePerm); err != nil {
				t.Fatalf("Error writing mock rawIndex file: %v", err)
			}
			if err := ioutil.WriteFile(filepath.Join(dir, "vectordb", dataFile), tc.rawData(), dataFilePerm); err != nil {
				t.Fatalf("Error writing mock rawData file: %v", err)
			}

			vectorDB, err := Open("vectordb", dir)
			if err != nil {
				t.Fatalf("Open(%q) = %v, got <nil>", dir, err)
			}
			defer vectorDB.Close()

			if got := vectorDB.Items(); got != uint64(len(tc.blobs)) {
				t.Fatalf("vectorDB.Items() = %d, want %d", got, uint64(len(tc.blobs)))
			}

			for i, want := range tc.blobs {
				got, err := vectorDB.Get(uint64(i))
				if err != nil {
					t.Errorf("vectorDB.Get(%d) = %s, %v, want %s, <nil>", uint64(i), hex.EncodeToString(got), err, hex.EncodeToString(want))
				}
			}
		})
	}
}

func marshalIndexEntries(entries ...*indexEntry) []byte {
	var b []byte
	for _, entry := range entries {
		b = append(b, entry.marshallBinary()...)
	}
	return b
}

func marshalDataBlobs(blobs ...[]byte) []byte {
	var b []byte
	for _, blob := range blobs {
		b = append(b, blob...)
	}
	return b
}

func TestOpen_DirectoryAlreadyExists_ReturnsError(t *testing.T) {
	dir, rmdir := createTempDir(t)
	defer rmdir()

	vectorDB, err := Open("vectordb", dir)
	if err != nil {
		t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
	}
	vectorDB.Close()

	vectorDB2, err := Open("vectordb", dir)
	if err != nil {
		t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
	}
	vectorDB2.Close()
}

func TestVectorDB_AppendGet(t *testing.T) {
	blobs := [][]byte{
		{1},
		{2, 2},
		{3, 3, 3},
	}

	dir, rmdir := createTempDir(t)
	defer rmdir()

	vectorDB, err := Open("vectordb", dir)
	if err != nil {
		t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
	}
	defer vectorDB.Close()

	for i, blob := range blobs {
		if err := vectorDB.Append(uint64(i), blob); err != nil {
			t.Errorf("vectorDB.Append(%d, %q) = %v, want <nil>", i, hex.EncodeToString(blob), err)
		}
		if vectorDB.Items() != uint64(i+1) {
			t.Errorf("vectorDB.Items() = %d, want %d", vectorDB.Items(), uint64(i+1))
		}
	}

	for i, want := range blobs {
		got, err := vectorDB.Get(uint64(i))
		if err != nil {
			t.Errorf("vectorDB.Get(%d) = %s, %v, want %s, <nil>", uint64(i), hex.EncodeToString(got), err, hex.EncodeToString(want))
		}
	}
}

func TestVectorDB_GetOnExistingDatabase(t *testing.T) {
	blobs := [][]byte{
		{1},
		{2, 2},
		{3, 3, 3},
	}

	dir, rmdir := createTempDir(t)
	defer rmdir()

	vectorDB, err := Open("vectordb", dir)
	if err != nil {
		t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
	}

	for i, blob := range blobs {
		if err := vectorDB.Append(uint64(i), blob); err != nil {
			t.Errorf("vectorDB.Append(%d, %q) = %v, want <nil>", i, hex.EncodeToString(blob), err)
		}
		if vectorDB.Items() != uint64(i+1) {
			t.Errorf("vectorDB.Items() = %d, want %d", vectorDB.Items(), uint64(i+1))
		}
	}

	if err := vectorDB.Sync(); err != nil {
		t.Fatalf("vectorDB.Sync() = %v, want <nil>", err)
	}
	vectorDB.Close()

	vectorDB2, err := Open("vectordb", dir)
	if err != nil {
		t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
	}
	defer vectorDB2.Close()

	for i, want := range blobs {
		got, err := vectorDB2.Get(uint64(i))
		if err != nil {
			t.Errorf("vectorDB2.Get(%d) = %s, %v, want %s, <nil>", uint64(i), hex.EncodeToString(got), err, hex.EncodeToString(want))
		}
	}
}

func TestAppend_PositionMismatch_ReturnsError(t *testing.T) {
	items := [][]byte{{1}, {2}, {3}}

	tests := []struct {
		name string
		pos  uint64
	}{
		{
			"Before",
			uint64(len(items) - 1),
		},
		{
			"After",
			uint64(len(items) + 1),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, rmdir := createTempDir(t)
			defer rmdir()

			vectorDB, err := Open("vectordb", dir)
			if err != nil {
				t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
			}
			defer vectorDB.Close()

			for i, item := range items {
				if err := vectorDB.Append(uint64(i), item); err != nil {
					t.Errorf("vectorDB.Append(%d, %q) = %v, want <nil>", i, hex.EncodeToString(item), err)
				}
			}

			if err := vectorDB.Append(tc.pos, []byte{0}); err == nil {
				t.Fatalf("vector.Append(%d, %s) = %v, want <error>", tc.pos, hex.EncodeToString([]byte{0}), err)
			}
		})
	}
}

func TestVectorDB_GetGreaterThanLen_ReturnsError(t *testing.T) {
	dir, rmdir := createTempDir(t)
	defer rmdir()

	vectorDB, err := Open("vectordb", dir)
	if err != nil {
		t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
	}
	defer vectorDB.Close()

	for i := 0; i < 3; i++ {
		vectorDB.Append(uint64(i), []byte{1, 2, 3})
	}

	if got, err := vectorDB.Get(3); err == nil {
		t.Errorf("vectorDB.Get(%d) = %s, %v, want \"\", <err>", uint64(3), hex.EncodeToString(got), err)
	}
}

func TestVectorDB_Truncate(t *testing.T) {
	const truncatedLen = 2
	blobs := [][]byte{
		{1},
		{2, 2},
		{3, 3, 3},
	}

	dir, rmdir := createTempDir(t)
	defer rmdir()

	vectorDB, err := Open("vectordb", dir)
	if err != nil {
		t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
	}
	defer vectorDB.Close()

	for i, blob := range blobs {
		if err := vectorDB.Append(uint64(i), blob); err != nil {
			t.Fatalf("vectorDB.Append(%d, %q) = %v, want <nil>", i, hex.EncodeToString(blob), err)
		}
		if vectorDB.Items() != uint64(i+1) {
			t.Fatalf("vectorDB.Items() = %d, want %d", vectorDB.Items(), uint64(i+1))
		}
	}

	if err := vectorDB.Truncate(truncatedLen); err != nil {
		t.Fatalf("vectorDB.Truncate(%d) = %v, want <nil>", truncatedLen, err)
	}

	for i, want := range blobs[:truncatedLen] {
		got, err := vectorDB.Get(uint64(i))
		if err != nil {
			t.Errorf("vectorDB.Get(%d) = %s, %v, want %s, <nil>", uint64(i), hex.EncodeToString(got), err, hex.EncodeToString(want))
		}
	}

	if got, err := vectorDB.Get(truncatedLen); err == nil {
		t.Errorf("vectorDB.Get(%d) = %s, %v, want \"\", <err>", truncatedLen, hex.EncodeToString(got), err)
	}
}

func TestVectorDB_TruncateGreaterThanLen_ReturnsError(t *testing.T) {
	dir, rmdir := createTempDir(t)
	defer rmdir()

	vectorDB, err := Open("vectordb", dir)
	if err != nil {
		t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
	}
	defer vectorDB.Close()

	for i := 0; i < 3; i++ {
		vectorDB.Append(uint64(i), []byte{1, 2, 3})
	}

	if err := vectorDB.Truncate(3); err == nil {
		t.Errorf("vectorDB.Truncate(%d) = %v, want <err>", uint64(3), err)
	}
}

func TestVectorDB_ReturnsErrrWhenClosed(t *testing.T) {
	tests := []struct {
		name string
		do   func(*VectorDB) error
	}{
		{
			"Append",
			func(db *VectorDB) error {
				return db.Append(uint64(0), []byte{1, 2, 3})
			},
		},
		{
			"Get",
			func(db *VectorDB) error {
				_, err := db.Get(0)
				return err
			},
		},
		{
			"Truncate",
			func(db *VectorDB) error {
				return db.Truncate(0)
			},
		},
		{
			"Sync",
			func(db *VectorDB) error {
				return db.Sync()
			},
		},
		{
			"Close",
			func(db *VectorDB) error {
				return db.Close()
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir, rmdir := createTempDir(t)
			defer rmdir()

			vectorDB, err := Open("vectordb", dir)
			if err != nil {
				t.Fatalf("Open(%q) = %v, want <nil>", dir, err)
			}

			vectorDB.Close()

			if err := tc.do(vectorDB); err != errClosed {
				t.Fatalf("vectorDB.%s = %v, want %v", tc.name, err, errClosed)
			}
		})
	}
}

func createTempDir(t *testing.T) (string, func()) {
	t.Helper()

	root, err := ioutil.TempDir(os.TempDir(), "vectordb_test_")
	if err != nil {
		t.Fatalf("Error creating test directory: %v", err)
	}
	return root, func() {
		os.RemoveAll(root)
	}
}
