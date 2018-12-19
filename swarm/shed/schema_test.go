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
	"bytes"
	"testing"
)

// TestDB_schemaFieldKey validates correctness of schemaFieldKey.
func TestDB_schemaFieldKey(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	t.Run("empty name or type", func(t *testing.T) {
		_, err := db.schemaFieldKey("", "")
		if err == nil {
			t.Errorf("error not returned, but expected")
		}
		_, err = db.schemaFieldKey("", "type")
		if err == nil {
			t.Errorf("error not returned, but expected")
		}

		_, err = db.schemaFieldKey("test", "")
		if err == nil {
			t.Errorf("error not returned, but expected")
		}
	})

	t.Run("same field", func(t *testing.T) {
		key1, err := db.schemaFieldKey("test", "undefined")
		if err != nil {
			t.Fatal(err)
		}

		key2, err := db.schemaFieldKey("test", "undefined")
		if err != nil {
			t.Fatal(err)
		}

		if !bytes.Equal(key1, key2) {
			t.Errorf("schema keys for the same field name are not the same: %q, %q", string(key1), string(key2))
		}
	})

	t.Run("different fields", func(t *testing.T) {
		key1, err := db.schemaFieldKey("test1", "undefined")
		if err != nil {
			t.Fatal(err)
		}

		key2, err := db.schemaFieldKey("test2", "undefined")
		if err != nil {
			t.Fatal(err)
		}

		if bytes.Equal(key1, key2) {
			t.Error("schema keys for the same field name are the same, but must not be")
		}
	})

	t.Run("same field name different types", func(t *testing.T) {
		_, err := db.schemaFieldKey("the-field", "one-type")
		if err != nil {
			t.Fatal(err)
		}

		_, err = db.schemaFieldKey("the-field", "another-type")
		if err == nil {
			t.Errorf("error not returned, but expected")
		}
	})
}

// TestDB_schemaIndexPrefix validates correctness of schemaIndexPrefix.
func TestDB_schemaIndexPrefix(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	t.Run("same name", func(t *testing.T) {
		id1, err := db.schemaIndexPrefix("test")
		if err != nil {
			t.Fatal(err)
		}

		id2, err := db.schemaIndexPrefix("test")
		if err != nil {
			t.Fatal(err)
		}

		if id1 != id2 {
			t.Errorf("schema keys for the same field name are not the same: %v, %v", id1, id2)
		}
	})

	t.Run("different names", func(t *testing.T) {
		id1, err := db.schemaIndexPrefix("test1")
		if err != nil {
			t.Fatal(err)
		}

		id2, err := db.schemaIndexPrefix("test2")
		if err != nil {
			t.Fatal(err)
		}

		if id1 == id2 {
			t.Error("schema ids for the same index name are the same, but must not be")
		}
	})
}
