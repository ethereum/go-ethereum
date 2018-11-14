// Copyright 2018 The go-ethereum Authors
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

package shed

import (
	"io/ioutil"
	"os"
	"testing"
)

// TestNewDB constructs a new DB
// and validates if the schema is initialized properly.
func TestNewDB(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	s, err := db.getSchema()
	if err != nil {
		t.Fatal(err)
	}
	if s.Fields == nil {
		t.Error("schema fields are empty")
	}
	if len(s.Fields) != 0 {
		t.Errorf("got schema fields length %v, want %v", len(s.Fields), 0)
	}
	if s.Indexes == nil {
		t.Error("schema indexes are empty")
	}
	if len(s.Indexes) != 0 {
		t.Errorf("got schema indexes length %v, want %v", len(s.Indexes), 0)
	}
}

// TestDB_persistence creates one DB, saves a field and closes that DB.
// Then, it constructs another DB and trues to retrieve the saved value.
func TestDB_persistence(t *testing.T) {
	dir, err := ioutil.TempDir("", "shed-test-persistence")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	db, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	stringField, err := db.NewStringField("preserve-me")
	if err != nil {
		t.Fatal(err)
	}
	want := "persistent value"
	err = stringField.Put(want)
	if err != nil {
		t.Fatal(err)
	}
	err = db.Close()
	if err != nil {
		t.Fatal(err)
	}

	db2, err := NewDB(dir)
	if err != nil {
		t.Fatal(err)
	}
	stringField2, err := db2.NewStringField("preserve-me")
	if err != nil {
		t.Fatal(err)
	}
	got, err := stringField2.Get()
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Errorf("got string %q, want %q", got, want)
	}
}

// newTestDB is a helper function that constructs a
// temporary database and returns a cleanup function that must
// be called to remove the data.
func newTestDB(t *testing.T) (db *DB, cleanupFunc func()) {
	t.Helper()

	dir, err := ioutil.TempDir("", "shed-test")
	if err != nil {
		t.Fatal(err)
	}
	cleanupFunc = func() { os.RemoveAll(dir) }
	db, err = NewDB(dir)
	if err != nil {
		cleanupFunc()
		t.Fatal(err)
	}
	return db, cleanupFunc
}
