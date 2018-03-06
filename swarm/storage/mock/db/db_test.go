// +build go1.8
//
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

package db

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/storage/mock/test"
)

// TestDBStore is running a test.MockStore tests
// using test.MockStore function.
func TestDBStore(t *testing.T) {
	dir, err := ioutil.TempDir("", "mock_"+t.Name())
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir)

	store, err := NewGlobalStore(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()

	test.MockStore(t, store, 100)
}

// TestImportExport is running a test.ImportExport tests
// using test.MockStore function.
func TestImportExport(t *testing.T) {
	dir1, err := ioutil.TempDir("", "mock_"+t.Name()+"_exporter")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir1)

	store1, err := NewGlobalStore(dir1)
	if err != nil {
		t.Fatal(err)
	}
	defer store1.Close()

	dir2, err := ioutil.TempDir("", "mock_"+t.Name()+"_importer")
	if err != nil {
		panic(err)
	}
	defer os.RemoveAll(dir2)

	store2, err := NewGlobalStore(dir2)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.Close()

	test.ImportExport(t, store1, store2, 100)
}
