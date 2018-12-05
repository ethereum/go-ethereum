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

package localstore

import (
	"crypto/rand"
	"io/ioutil"
	"os"
	"testing"

	ch "github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// TestDB_useRetrievalCompositeIndex checks if optional argument
// WithRetrievalCompositeIndex to New constructor is setting the
// correct state.
func TestDB_useRetrievalCompositeIndex(t *testing.T) {
	t.Run("set true", func(t *testing.T) {
		db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: true})
		defer cleanupFunc()

		if !db.useRetrievalCompositeIndex {
			t.Error("useRetrievalCompositeIndex is not set to true")
		}
	})
	t.Run("set false", func(t *testing.T) {
		db, cleanupFunc := newTestDB(t, &Options{UseRetrievalCompositeIndex: false})
		defer cleanupFunc()

		if db.useRetrievalCompositeIndex {
			t.Error("useRetrievalCompositeIndex is not set to false")
		}
	})
	t.Run("unset", func(t *testing.T) {
		db, cleanupFunc := newTestDB(t, nil)
		defer cleanupFunc()

		if db.useRetrievalCompositeIndex {
			t.Error("useRetrievalCompositeIndex is not set to false")
		}
	})
}

// newTestDB is a helper function that constructs a
// temporary database and returns a cleanup function that must
// be called to remove the data.
func newTestDB(t *testing.T, o *Options) (db *DB, cleanupFunc func()) {
	t.Helper()

	dir, err := ioutil.TempDir("", "shed-test")
	if err != nil {
		t.Fatal(err)
	}
	cleanupFunc = func() { os.RemoveAll(dir) }
	baseKey := make([]byte, 32)
	if _, err := rand.Read(baseKey); err != nil {
		t.Fatal(err)
	}
	db, err = New(dir, baseKey, o)
	if err != nil {
		cleanupFunc()
		t.Fatal(err)
	}
	cleanupFunc = func() {
		err := db.Close()
		if err != nil {
			t.Error(err)
		}
		os.RemoveAll(dir)
	}
	return db, cleanupFunc
}

func generateRandomChunk() storage.Chunk {
	return storage.GenerateRandomChunk(ch.DefaultSize)
}
