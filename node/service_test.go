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
	ctx := &ServiceContext{Config: Config{Name: "unit-test", DataDir: dir}}
	db, err := ctx.OpenDatabase("persistent", 0, 0, "")
	if err != nil {
		t.Fatalf("failed to open persistent database: %v", err)
	}
	db.Close()

	if _, err := os.Stat(filepath.Join(dir, "unit-test", "persistent")); err != nil {
		t.Fatalf("persistent database doesn't exists: %v", err)
	}
	// Request th opening/creation of an ephemeral database and ensure it's not persisted
	ctx = &ServiceContext{Config: Config{DataDir: ""}}
	db, err = ctx.OpenDatabase("ephemeral", 0, 0, "")
	if err != nil {
		t.Fatalf("failed to open ephemeral database: %v", err)
	}
	db.Close()

	if _, err := os.Stat(filepath.Join(dir, "ephemeral")); err == nil {
		t.Fatalf("ephemeral database exists")
	}
}

// Tests that already constructed Lifecycles can be retrieved by later ones.
func TestContextLifecycles(t *testing.T) {
	stack, err := New(testNodeConfig())
	if err != nil {
		t.Fatalf("failed to create protocol stack: %v", err)
	}
	defer stack.Close()
	// Define a verifier that ensures a NoopA is before it and NoopB after

	noop := NewNoop()
	stack.RegisterLifecycle(noop)

	isC, err := NewInstrumentedService()
	if err != nil {
		t.Fatalf("could not create instrumented service %v", err)
	}

	isB, err := NewInstrumentedService()
	if err != nil {
		t.Fatalf("could not create instrumented service %v", err)

	}
	isB.startHook = func() {
		if err := stack.ServiceContext.Lifecycle(&noop); err != nil {
			t.Errorf("former service not found: %v", err)
		}
		if err := stack.ServiceContext.Lifecycle(&isC); err != ErrServiceUnknown {
			t.Errorf("latters lookup error mismatch: have %v, want %v", err, ErrServiceUnknown)
		}
	}
	stack.RegisterLifecycle(isB)

	// Start the protocol stack and ensure services are constructed in order
	if err := stack.Start(); err != nil {
		t.Fatalf("failed to start stack: %v", err)
	}

	defer stack.Stop()
}
