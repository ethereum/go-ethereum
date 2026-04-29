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

package keystore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteKeyFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "keyfile")
	content := []byte(`{"address":"deadbeef"}`)

	if err := writeKeyFile(path, content); err != nil {
		t.Fatalf("writeKeyFile: %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("content mismatch: got %q want %q", got, content)
	}
	// Verify no temp file leaked into the directory.
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("readdir: %v", err)
	}
	if len(entries) != 1 || entries[0].Name() != "keyfile" {
		var names []string
		for _, e := range entries {
			names = append(names, e.Name())
		}
		t.Fatalf("unexpected directory contents: %v", names)
	}
}
