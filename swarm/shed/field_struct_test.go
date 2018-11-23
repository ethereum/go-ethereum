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
	"testing"

	"github.com/syndtr/goleveldb/leveldb"
)

// TestStructField validates put and get operations
// of the StructField.
func TestStructField(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	complexField, err := db.NewStructField("complex-field")
	if err != nil {
		t.Fatal(err)
	}

	type complexStructure struct {
		A string
	}

	t.Run("get empty", func(t *testing.T) {
		var s complexStructure
		err := complexField.Get(&s)
		if err != leveldb.ErrNotFound {
			t.Fatalf("got error %v, want %v", err, leveldb.ErrNotFound)
		}
		want := ""
		if s.A != want {
			t.Errorf("got string %q, want %q", s.A, want)
		}
	})

	t.Run("put", func(t *testing.T) {
		want := complexStructure{
			A: "simple string value",
		}
		err = complexField.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		var got complexStructure
		err = complexField.Get(&got)
		if err != nil {
			t.Fatal(err)
		}
		if got.A != want.A {
			t.Errorf("got string %q, want %q", got.A, want.A)
		}

		t.Run("overwrite", func(t *testing.T) {
			want := complexStructure{
				A: "overwritten string value",
			}
			err = complexField.Put(want)
			if err != nil {
				t.Fatal(err)
			}
			var got complexStructure
			err = complexField.Get(&got)
			if err != nil {
				t.Fatal(err)
			}
			if got.A != want.A {
				t.Errorf("got string %q, want %q", got.A, want.A)
			}
		})
	})

	t.Run("put in batch", func(t *testing.T) {
		batch := new(leveldb.Batch)
		want := complexStructure{
			A: "simple string batch value",
		}
		complexField.PutInBatch(batch, want)
		err = db.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}
		var got complexStructure
		err := complexField.Get(&got)
		if err != nil {
			t.Fatal(err)
		}
		if got.A != want.A {
			t.Errorf("got string %q, want %q", got, want)
		}

		t.Run("overwrite", func(t *testing.T) {
			batch := new(leveldb.Batch)
			want := complexStructure{
				A: "overwritten string batch value",
			}
			complexField.PutInBatch(batch, want)
			err = db.WriteBatch(batch)
			if err != nil {
				t.Fatal(err)
			}
			var got complexStructure
			err := complexField.Get(&got)
			if err != nil {
				t.Fatal(err)
			}
			if got.A != want.A {
				t.Errorf("got string %q, want %q", got, want)
			}
		})
	})
}
