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
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"
)

// Tests that databases are correctly created persistent or ephemeral based on
// the configured service context.
func TestContextDatabases(t *testing.T) {
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
	ctx := &ServiceContext{datadir: dir}
	db, err := ctx.Database("persistent", 0)
	if err != nil {
		t.Fatalf("failed to open persistent database: %v", err)
	}
	db.Close()

	if _, err := os.Stat(filepath.Join(dir, "persistent")); err != nil {
		t.Fatalf("persistent database doesn't exists: %v", err)
	}
	// Request th opening/creation of an ephemeral database and ensure it's not persisted
	ctx = &ServiceContext{datadir: ""}
	db, err = ctx.Database("ephemeral", 0)
	if err != nil {
		t.Fatalf("failed to open ephemeral database: %v", err)
	}
	db.Close()

	if _, err := os.Stat(filepath.Join(dir, "ephemeral")); err == nil {
		t.Fatalf("ephemeral database exists")
	}
}

// Tests that already constructed services can be retrieves by later ones.
func TestContextServices(t *testing.T) {
	stack, err := New(testNodeConfig)
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	// Define a set of services, constructed before/after a verifier
	formers := []string{"A", "B", "C", "D", "E", "F", "G", "H", "I", "J"}
	latters := []string{"0", "1", "2", "3", "4", "5", "6", "7", "8", "9"}

	verifier := func(ctx *ServiceContext) (Service, error) {
		for i, id := range formers {
			if ctx.Service(id) == nil {
				return nil, fmt.Errorf("former %d: service not found", i)
			}
		}
		for i, id := range latters {
			if ctx.Service(id) != nil {
				return nil, fmt.Errorf("latters %d: service found", i)
			}
		}
		return new(NoopService), nil
	}
	// Register the collection of services
	for i, id := range formers {
		if err := stack.Register(id, NewNoopService); err != nil {
			t.Fatalf("former #%d: failed to register service: %v", i, err)
		}
	}
	if err := stack.Register("verifier", verifier); err != nil {
		t.Fatalf("failed to register service verifier: %v", err)
	}
	for i, id := range latters {
		if err := stack.Register(id, NewNoopService); err != nil {
			t.Fatalf("latter #%d: failed to register service: %v", i, err)
		}
	}
	// Start the protocol stack and ensure services are constructed in order
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start stack: %v", err)
	}
	defer stack.Stop()
}
