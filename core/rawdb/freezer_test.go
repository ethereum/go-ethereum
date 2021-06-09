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
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
)

func newFreezerForTesting(t *testing.T) (*freezer, string) {
	t.Helper()

	dir, err := ioutil.TempDir("", "freezer")
	if err != nil {
		t.Fatal(err)
	}
	tables := map[string]bool{"test": true}
	// note: using low max table size here to ensure the tests actually
	// switch between multiple files.
	f, err := newFreezer(dir, "", false, 2049, tables)
	if err != nil {
		t.Fatal("can't open freezer", err)
	}
	return f, dir
}

// This test runs ModifyAncients and TruncateAncients concurrently with each other.
func TestFreezerConcurrentModifyRetrieve(t *testing.T) {
	f, dir := newFreezerForTesting(t)
	defer os.RemoveAll(dir)
	defer f.Close()

	var (
		numReaders     = 5
		writeBatchSize = uint64(50)
		written        = make(chan uint64, numReaders*6)
		wg             sync.WaitGroup
	)
	wg.Add(numReaders + 1)

	// Launch the writer. It appends 10000 items in batches.
	go func() {
		defer wg.Done()
		defer close(written)
		for item := uint64(0); item < 10000; item += writeBatchSize {
			_, err := f.ModifyAncients(func(op ethdb.AncientWriteOp) error {
				for i := uint64(0); i < writeBatchSize; i++ {
					item := item + i
					value := getChunk(32, int(item))
					if err := op.AppendRaw("test", item, value); err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				panic(err)
			}
			for i := 0; i < numReaders; i++ {
				written <- item + writeBatchSize
			}
		}
	}()

	// Launch the readers. They read random items from the freezer up to the
	// current frozen item count.
	for i := 0; i < numReaders; i++ {
		go func() {
			defer wg.Done()
			for frozen := range written {
				for rc := 0; rc < 50; rc++ {
					num := uint64(rand.Intn(int(frozen)))
					value, err := f.Ancient("test", num)
					if err != nil {
						panic(fmt.Errorf("error reading %d (frozen %d): %v", num, frozen, err))
					}
					if !bytes.Equal(value, getChunk(32, int(num))) {
						panic(fmt.Errorf("wrong value at %d", num))
					}
				}
			}
		}()
	}

	wg.Wait()
}

// This test runs ModifyAncients and TruncateAncients concurrently with each other.
func TestFreezerConcurrentModifyTruncate(t *testing.T) {
	f, dir := newFreezerForTesting(t)
	defer os.RemoveAll(dir)
	defer f.Close()

	var item = make([]byte, 256)

	for i := 0; i < 1000; i++ {
		// First reset, and write 100 items.
		if err := f.TruncateAncients(0); err != nil {
			t.Fatal("truncate failed:", err)
		}
		_, err := f.ModifyAncients(func(op ethdb.AncientWriteOp) error {
			for i := uint64(0); i < 100; i++ {
				if err := op.AppendRaw("test", i, item); err != nil {
					return err
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal("modify failed:", err)
		}
		if frozen, _ := f.Ancients(); frozen != 100 {
			t.Fatalf("wrong ancients count %d, want 100", frozen)
		}

		// Now append 100 more items and truncate concurrently.
		var (
			wg          sync.WaitGroup
			truncateErr error
			modifyErr   error
		)
		wg.Add(2)
		go func() {
			_, modifyErr = f.ModifyAncients(func(op ethdb.AncientWriteOp) error {
				for i := uint64(100); i < 200; i++ {
					if err := op.AppendRaw("test", i, item); err != nil {
						return err
					}
				}
				return nil
			})
			wg.Done()
		}()
		go func() {
			truncateErr = f.TruncateAncients(10)
			wg.Done()
		}()
		wg.Wait()

		// Now check the outcome. If the truncate operation went through first, the append
		// fails, otherwise it succeeds. In either case, the freezer should be positioned
		// at 10 after both operations are done.
		if truncateErr != nil {
			t.Fatal("concurrent truncate failed:", err)
		}
		if modifyErr != nil && modifyErr != errOutOrderInsertion {
			t.Fatal("wrong error from concurrent modify:", modifyErr)
		}
		if frozen, _ := f.Ancients(); frozen != 10 {
			t.Fatalf("Ancients returned %d, want 0", frozen)
		}
	}
}
