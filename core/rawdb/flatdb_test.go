// Copyright 2020 The go-ethereum Authors
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
	"bytes"
	"crypto/rand"
	"io/ioutil"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
)

type flatDBTester struct {
	dir string
	db  *FlatDatabase
}

func newFlatDBTester(read bool) *flatDBTester {
	dir, _ := ioutil.TempDir("", "")
	db, err := NewFlatDatabase(dir, read)
	if err != nil {
		return nil
	}
	return &flatDBTester{
		dir: dir,
		db:  db,
	}
}

func (tester *flatDBTester) teardown() {
	if tester.dir != "" {
		os.RemoveAll(tester.dir)
	}
}

func (tester *flatDBTester) Put(key, value []byte) {
	tester.db.Put(key, value)
}

func (tester *flatDBTester) Iterate() ethdb.Iterator {
	return tester.db.NewIterator(nil, nil)
}

func (tester *flatDBTester) Commit() {
	tester.db.Commit()
}

func (tester *flatDBTester) checkIteration(t *testing.T, keys [][]byte, vals [][]byte) {
	iter := tester.Iterate()
	var index int
	for iter.Next() {
		if index >= len(keys) {
			t.Fatalf("Extra entry found")
		}
		if index >= len(vals) {
			t.Fatalf("Extra entry found")
		}
		if !bytes.Equal(iter.Key(), keys[index]) {
			t.Fatalf("Entry key mismatch %v -> %v", keys[index], iter.Key())
		}
		if !bytes.Equal(iter.Value(), vals[index]) {
			t.Fatalf("Entry value mismatch %v -> %v", vals[index], iter.Value())
		}
		index += 1
	}
	if iter.Error() != nil {
		t.Fatalf("Iteration error %v", iter.Error())
	}
	iter.Release()

	if index != len(keys) {
		t.Fatalf("Missing entries, want %d, got %d", len(keys), index)
	}
}

func newTestCases(size int) ([][]byte, [][]byte) {
	var (
		keys [][]byte
		vals [][]byte
		kbuf [20]byte
		vbuf [32]byte
	)
	for i := 0; i < size; i++ {
		rand.Read(kbuf[:])
		keys = append(keys, common.CopyBytes(kbuf[:]))

		rand.Read(vbuf[:])
		vals = append(vals, common.CopyBytes(vbuf[:]))
	}
	return keys, vals
}

func TestReadNonExistentDB(t *testing.T) {
	tester := newFlatDBTester(true)
	if tester != nil {
		t.Fatalf("Expect the error for opening the non-existent db")
	}
}

func TestReadConcurrently(t *testing.T) {
	tester := newFlatDBTester(false)
	if tester == nil {
		t.Fatalf("Failed to init tester")
	}
	defer tester.teardown()

	iter := tester.Iterate()
	if iter == nil {
		t.Fatalf("Failed to obtain iterator")
	}
	if iter := tester.Iterate(); iter != nil {
		t.Fatalf("Concurrent iteration is not allowed")
	}
	iter.Release()
	if iter := tester.Iterate(); iter == nil {
		t.Fatalf("Failed to obtain iterator")
	}
}

func TestFlatDatabase(t *testing.T) {
	tester := newFlatDBTester(false)
	if tester == nil {
		t.Fatalf("Failed to init tester")
	}
	defer tester.teardown()

	keys, vals := newTestCases(1024 * 1024)
	for i := 0; i < len(keys); i++ {
		tester.Put(keys[i], vals[i])
	}
	tester.Commit()
	tester.checkIteration(t, keys, vals)
	tester.checkIteration(t, keys, vals) // Check twice
}

func TestFlatDatabaseBatchWrite(t *testing.T) {
	tester := newFlatDBTester(false)
	if tester == nil {
		t.Fatalf("Failed to init tester")
	}
	defer tester.teardown()

	keys, vals := newTestCases(1024 * 1024)
	batch := tester.db.NewBatch()
	for i := 0; i < len(keys); i++ {
		batch.Put(keys[i], vals[i])
		if batch.ValueSize() > 1024 {
			batch.Write()
			batch.Reset()
		}
	}
	batch.Write()

	tester.Commit()
	tester.checkIteration(t, keys, vals)
}

func TestFlatDatabaseConcurrentWrite(t *testing.T) {
	tester := newFlatDBTester(false)
	if tester == nil {
		t.Fatalf("Failed to init tester")
	}
	defer tester.teardown()

	var wg sync.WaitGroup
	writer := func() {
		defer wg.Done()
		keys, vals := newTestCases(1024 * 1024)
		batch := tester.db.NewBatch()
		for i := 0; i < len(keys); i++ {
			batch.Put(keys[i], vals[i])
			if batch.ValueSize() > 1024 {
				batch.Write()
				batch.Reset()
			}
		}
		batch.Write()
	}
	wg.Add(2)
	go writer()
	go writer()

	wg.Wait()
	tester.Commit()
}
