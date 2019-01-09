// Copyright 2014 The go-ethereum Authors
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

// +build !js

package ethdb_test

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
)

var values = []string{"", "a", "1251", "\x00123\x00"}

func TestLDB_PutGet(t *testing.T) {
	ldb, remove := newTestLDB(t)
	defer remove()

	tests := []struct {
		name string
		db   ethdb.Database
	}{
		{"LDB", ldb},
		{"MemoryDB", ethdb.NewMemDatabase()},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			for _, k := range values {
				err := tc.db.Put([]byte(k), nil)
				if err != nil {
					t.Fatalf("tc.db.Put(%q, <nil>) = %v, want <nil>", k, err)
				}
			}

			for _, k := range values {
				data, err := tc.db.Get([]byte(k))
				if err != nil || len(data) != 0 {
					t.Fatalf("tc.db.Get(%q) = %q, %v, want \"\", <nil>", k, string(data), err)
				}
			}

			data, err := tc.db.Get([]byte("non-exist-key"))
			if err == nil {
				t.Fatalf("tc.db.Get(\"non-exist-key\") = %q, %v, want <nil>, <error: \"not found\">", string(data), err)
			}

			for _, v := range values {
				err := tc.db.Put([]byte(v), []byte(v))
				if err != nil {
					t.Fatalf("tc.db.Put(%q, %q) = %v, want <nil>", v, v, err)
				}
			}

			for _, v := range values {
				data, err := tc.db.Get([]byte(v))
				if err != nil || !bytes.Equal(data, []byte(v)) {
					t.Fatalf("tc.db.Get(%q) = %q, %v, want %q, <nil>", v, string(data), err, v)
				}
			}

			for _, v := range values {
				err := tc.db.Put([]byte(v), []byte("?"))
				if err != nil {
					t.Fatalf("tc.db.Put(%q, \"?\") = %v, want <nil>", v, err)
				}
			}

			for _, v := range values {
				data, err := tc.db.Get([]byte(v))
				if err != nil || !bytes.Equal(data, []byte("?")) {
					t.Fatalf("tc.db.Get(%q) = %q, %v, want \"?\", <nil>", v, string(data), err)

				}
			}

			for _, v := range values {
				orig, err := tc.db.Get([]byte(v))
				if err != nil || !bytes.Equal(orig, []byte("?")) {
					t.Fatalf("tc.db.Get(%q) = %q, %v, want \"?\", <nil>", v, string(orig), err)
				}
				// Mutate the original to ensure that the database is not returning the same value instance.
				orig[0] = byte(0xff)
				data, err := tc.db.Get([]byte(v))
				if err != nil || !bytes.Equal(data, []byte("?")) {
					t.Fatalf("tc.db.Get(%q) = %q, %v, want \"?\", <nil>", v, string(data), err)
				}
			}

			for _, v := range values {
				err := tc.db.Delete([]byte(v))
				if err != nil {
					t.Fatalf("tc.db.Delete(%q) = %v, want <nil>", v, err)
				}
			}

			for _, v := range values {
				data, err := tc.db.Get([]byte(v))
				if err == nil {
					t.Fatalf("tc.db.Get(%q) = %q, %v, want \"\", <error: \"not found\">", v, string(data), err)
				}
			}
		})
	}
}

func newTestLDB(t *testing.T) (*ethdb.LDBDatabase, func()) {
	t.Helper()

	dirname, err := ioutil.TempDir(os.TempDir(), "ethdb_test_")
	if err != nil {
		panic("failed to create test file: " + err.Error())
	}

	db, err := ethdb.NewLDBDatabase(dirname, 0, 0)
	if err != nil {
		panic("failed to create test database: " + err.Error())
	}

	return db, func() {
		db.Close()
		os.RemoveAll(dirname)
	}
}
