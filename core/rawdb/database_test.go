// Copyright 2026 The go-ethereum Authors
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
	"errors"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
)

type errorIteratorDatabase struct {
	ethdb.KeyValueStore
	err error
}

func (db *errorIteratorDatabase) NewIterator(prefix, start []byte) ethdb.Iterator {
	return &errorIterator{Iterator: db.KeyValueStore.NewIterator(prefix, start), err: db.err}
}

type errorIterator struct {
	ethdb.Iterator
	err    error
	failed bool
}

func (it *errorIterator) Next() bool {
	if it.Iterator.Next() {
		return true
	}
	it.failed = true
	return false
}

func (it *errorIterator) Error() error {
	if !it.failed {
		return it.Iterator.Error()
	}
	return it.err
}

func TestSafeDeleteRangeIteratorError(t *testing.T) {
	base := NewMemoryDatabase()
	if err := base.Put([]byte("a"), []byte("value")); err != nil {
		t.Fatal(err)
	}
	want := errors.New("test iterator error")
	db := &errorIteratorDatabase{KeyValueStore: base, err: want}
	if err := SafeDeleteRange(db, []byte("a"), []byte("b"), true, func(bool) bool { return false }); !errors.Is(err, want) {
		t.Fatalf("error = %v, want %v", err, want)
	}
	if has, err := base.Has([]byte("a")); err != nil {
		t.Fatal(err)
	} else if !has {
		t.Fatal("entry deleted despite iterator error")
	}
}
