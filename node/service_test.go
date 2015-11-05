// Copyright 2015 The go-ethereum Authors
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

package node

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// Tests that service context methods work properly.
func TestServiceContext(t *testing.T) {
	// Create a temporary folder and ensure no database is contained within
	dir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("failed to create temporary data directory: %v", err)
	}
	defer os.RemoveAll(dir)

	if _, err := os.Stat(filepath.Join(dir, "database")); err == nil {
		t.Fatalf("non-created database already exists")
	}
	// Request the opening/creation of a database and ensure it persists to disk
	ctx := &ServiceContext{dataDir: dir}
	db, err := ctx.Database("persistent", 0)
	if err != nil {
		t.Fatalf("failed to open persistent database: %v", err)
	}
	db.Close()

	if _, err := os.Stat(filepath.Join(dir, "persistent")); err != nil {
		t.Fatalf("persistent database doesn't exists: %v", err)
	}
	// Request th opening/creation of an ephemeral database and ensure it's not persisted
	ctx = &ServiceContext{dataDir: ""}
	db, err = ctx.Database("ephemeral", 0)
	if err != nil {
		t.Fatalf("failed to open ephemeral database: %v", err)
	}
	db.Close()

	if _, err := os.Stat(filepath.Join(dir, "ephemeral")); err == nil {
		t.Fatalf("ephemeral database exists")
	}
}
