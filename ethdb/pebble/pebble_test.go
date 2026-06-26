// Copyright 2019 The go-ethereum Authors
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

package pebble

import (
	"errors"
	"testing"

	"github.com/cockroachdb/pebble"
	pebblev1 "github.com/cockroachdb/pebble"
	pebblev2 "github.com/cockroachdb/pebble/v2"
	vfsv2 "github.com/cockroachdb/pebble/v2/vfs"
	"github.com/cockroachdb/pebble/vfs"
	vfsv1 "github.com/cockroachdb/pebble/vfs"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/dbtest"
)

func TestPebbleDB(t *testing.T) {
	t.Run("DatabaseSuite", func(t *testing.T) {
		dbtest.TestDatabaseSuite(t, func() ethdb.KeyValueStore {
			db, err := pebblev2.Open("", &pebblev2.Options{
				FS: vfsv2.NewMem(),
			})
			if err != nil {
				t.Fatal(err)
			}
			return &Database{
				db: db,
			}
		})
		dbtest.TestDatabaseSuite(t, func() ethdb.KeyValueStore {
			db, err := pebblev1.Open("", &pebblev1.Options{
				FS: vfsv1.NewMem(),
			})
			if err != nil {
				t.Fatal(err)
			}
			return &V1Database{
				db: db,
			}
		})
	})
}

func BenchmarkPebbleDB(b *testing.B) {
	dbtest.BenchDatabaseSuite(b, func() ethdb.KeyValueStore {
		db, err := pebblev2.Open("", &pebblev2.Options{
			FS: vfsv2.NewMem(),
		})
		if err != nil {
			b.Fatal(err)
		}
		return &Database{
			db: db,
		}
	})
	dbtest.BenchDatabaseSuite(b, func() ethdb.KeyValueStore {
		db, err := pebblev1.Open("", &pebblev1.Options{
			FS: vfsv1.NewMem(),
		})
		if err != nil {
			b.Fatal(err)
		}
		return &V1Database{
			db: db,
		}
	})
}

func TestPebbleLogDataV1(t *testing.T) {
	db, err := pebble.Open("", &pebble.Options{
		FS: vfs.NewMem(),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = db.Get(nil)
	if !errors.Is(err, pebble.ErrNotFound) {
		t.Fatal("Unknown database entry")
	}

	b := db.NewBatch()
	b.LogData(nil, nil)
	db.Apply(b, pebble.Sync)

	_, _, err = db.Get(nil)
	if !errors.Is(err, pebble.ErrNotFound) {
		t.Fatal("Unknown database entry")
	}
}

func TestPebbleLogDataV2(t *testing.T) {
	db, err := pebblev2.Open("", &pebblev2.Options{
		FS: vfsv2.NewMem(),
	})
	if err != nil {
		t.Fatal(err)
	}

	_, _, err = db.Get(nil)
	if !errors.Is(err, pebblev2.ErrNotFound) {
		t.Fatal("Unknown database entry")
	}

	b := db.NewBatch()
	b.LogData(nil, nil)
	db.Apply(b, pebblev2.Sync)

	_, _, err = db.Get(nil)
	if !errors.Is(err, pebblev2.ErrNotFound) {
		t.Fatal("Unknown database entry")
	}
}
