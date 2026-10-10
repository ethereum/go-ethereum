// Copyright 2026 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package utils

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
)

// errorAfterIterator yields `limit` successful Next() calls then stops with err.
type errorAfterIterator struct {
	ethdb.Iterator
	limit int
	count int
	err   error
}

func (it *errorAfterIterator) Next() bool {
	if it.count >= it.limit {
		return false
	}
	if !it.Iterator.Next() {
		return false
	}
	it.count++
	return true
}

func (it *errorAfterIterator) Error() error {
	if it.count >= it.limit {
		return it.err
	}
	return it.Iterator.Error()
}

type iteratorErrorDB struct {
	ethdb.Database
	limit int
	err   error
}

func (db *iteratorErrorDB) NewIterator(prefix []byte, start []byte) ethdb.Iterator {
	return &errorAfterIterator{
		Iterator: db.Database.NewIterator(prefix, start),
		limit:    db.limit,
		err:      db.err,
	}
}

func TestExportPreimagesIteratorError(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	rawdb.WritePreimages(db, map[common.Hash][]byte{
		common.Hash{1}: []byte("one"),
		common.Hash{2}: []byte("two"),
		common.Hash{3}: []byte("three"),
	})

	want := errors.New("iterator boom")
	wrapped := &iteratorErrorDB{Database: db, limit: 2, err: want}
	path := filepath.Join(t.TempDir(), "preimages")
	err := ExportPreimages(wrapped, path)
	if !errors.Is(err, want) {
		t.Fatalf("ExportPreimages() error = %v, want %v", err, want)
	}
}
