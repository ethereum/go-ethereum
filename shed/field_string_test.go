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

// TestStringField validates put and get operations
// of the StringField.
func TestStringField(t *testing.T) {
	db, cleanupFunc := newTestDB(t)
	defer cleanupFunc()

	simpleString, err := db.NewStringField("simple-string")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("get empty", func(t *testing.T) {
		got, err := simpleString.Get()
		if err != nil {
			t.Fatal(err)
		}
		want := ""
		if got != want {
			t.Errorf("got string %q, want %q", got, want)
		}
	})

	t.Run("put", func(t *testing.T) {
		want := "simple string value"
		err = simpleString.Put(want)
		if err != nil {
			t.Fatal(err)
		}
		got, err := simpleString.Get()
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("got string %q, want %q", got, want)
		}

		t.Run("overwrite", func(t *testing.T) {
			want := "overwritten string value"
			err = simpleString.Put(want)
			if err != nil {
				t.Fatal(err)
			}
			got, err := simpleString.Get()
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("got string %q, want %q", got, want)
			}
		})
	})

	t.Run("put in batch", func(t *testing.T) {
		batch := new(leveldb.Batch)
		want := "simple string batch value"
		simpleString.PutInBatch(batch, want)
		err = db.WriteBatch(batch)
		if err != nil {
			t.Fatal(err)
		}
		got, err := simpleString.Get()
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("got string %q, want %q", got, want)
		}

		t.Run("overwrite", func(t *testing.T) {
			batch := new(leveldb.Batch)
			want := "overwritten string batch value"
			simpleString.PutInBatch(batch, want)
			err = db.WriteBatch(batch)
			if err != nil {
				t.Fatal(err)
			}
			got, err := simpleString.Get()
			if err != nil {
				t.Fatal(err)
			}
			if got != want {
				t.Errorf("got string %q, want %q", got, want)
			}
		})
	})
}
