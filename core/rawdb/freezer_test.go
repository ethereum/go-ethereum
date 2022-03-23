// Copyright 2021 The go-ethereum Authors
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

package rawdb

import (
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/dbtest"
	"github.com/stretchr/testify/require"
)

func TestFreezer(t *testing.T) {
	t.Run("AncientStoreSuite", func(t *testing.T) {
		dbtest.TestAncientStoreSuite(t, func(tables map[string]bool) (store ethdb.AncientStore, closeAndCleanup func()) {
			f, dir := newFreezerForTesting(t, tables)
			return f, func() {
				if err := f.Close(); err != nil {
					t.Errorf("failed to close freezer: %v", err)
				}
				if err := os.RemoveAll(dir); err != nil {
					t.Errorf("failed to cleanup after freezer close: %v", err)
				}
			}
		})
	})
}

func TestFreezerReadonlyValidate(t *testing.T) {
	tables := map[string]bool{"a": true, "b": true}
	dir, err := ioutil.TempDir("", "freezer")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)
	// Open non-readonly freezer and fill individual tables
	// with different amount of data.
	f, err := newFreezer(dir, "", false, 2049, tables)
	if err != nil {
		t.Fatal("can't open freezer", err)
	}
	var item = make([]byte, 1024)
	aBatch := f.tables["a"].newBatch()
	require.NoError(t, aBatch.AppendRaw(0, item))
	require.NoError(t, aBatch.AppendRaw(1, item))
	require.NoError(t, aBatch.AppendRaw(2, item))
	require.NoError(t, aBatch.commit())
	bBatch := f.tables["b"].newBatch()
	require.NoError(t, bBatch.AppendRaw(0, item))
	require.NoError(t, bBatch.commit())
	if f.tables["a"].items != 3 {
		t.Fatalf("unexpected number of items in table")
	}
	if f.tables["b"].items != 1 {
		t.Fatalf("unexpected number of items in table")
	}
	require.NoError(t, f.Close())

	// Re-openening as readonly should fail when validating
	// table lengths.
	f, err = newFreezer(dir, "", true, 2049, tables)
	if err == nil {
		t.Fatal("readonly freezer should fail with differing table lengths")
	}
}

func newFreezerForTesting(t *testing.T, tables map[string]bool) (*freezer, string) {
	t.Helper()

	dir, err := ioutil.TempDir("", "freezer")
	if err != nil {
		t.Fatal(err)
	}
	// note: using low max table size here to ensure the tests actually
	// switch between multiple files.
	f, err := newFreezer(dir, "", false, 2049, tables)
	if err != nil {
		t.Fatal("can't open freezer", err)
	}
	return f, dir
}
