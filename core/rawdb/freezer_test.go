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
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/stretchr/testify/require"
)

var freezerTestTableDef = map[string]bool{"test": true}

func TestFreezerModify(t *testing.T) {
	t.Parallel()

	// Create test data.
	var valuesRaw [][]byte
	var valuesRLP []*big.Int
	for x := 0; x < 100; x++ {
		v := getChunk(256, x)
		valuesRaw = append(valuesRaw, v)
		iv := big.NewInt(int64(x))
		iv = iv.Exp(iv, iv, nil)
		valuesRLP = append(valuesRLP, iv)
	}

	tables := map[string]bool{"raw": true, "rlp": false}
	f, dir := newFreezerForTesting(t, tables)
	defer os.RemoveAll(dir)
	defer f.Close()

	// Commit test data.
	_, err := f.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for i := range valuesRaw {
			if err := op.AppendRaw("raw", uint64(i), valuesRaw[i]); err != nil {
				return err
			}
			if err := op.Append("rlp", uint64(i), valuesRLP[i]); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal("ModifyAncients failed:", err)
	}

	// Dump indexes.
	for _, table := range f.tables {
		t.Log(table.name, "index:", table.dumpIndexString(0, int64(len(valuesRaw))))
	}

	// Read back test data.
	checkAncientCount(t, f, "raw", uint64(len(valuesRaw)))
	checkAncientCount(t, f, "rlp", uint64(len(valuesRLP)))
	for i := range valuesRaw {
		v, _ := f.Ancient("raw", uint64(i))
		if !bytes.Equal(v, valuesRaw[i]) {
			t.Fatalf("wrong raw value at %d: %x", i, v)
		}
		ivEnc, _ := f.Ancient("rlp", uint64(i))
		want, _ := rlp.EncodeToBytes(valuesRLP[i])
		if !bytes.Equal(ivEnc, want) {
			t.Fatalf("wrong RLP value at %d: %x", i, ivEnc)
		}
	}
}

// This checks that ModifyAncients rolls back freezer updates
// when the function passed to it returns an error.
func TestFreezerModifyRollback(t *testing.T) {
	t.Parallel()

	f, dir := newFreezerForTesting(t, freezerTestTableDef)
	defer os.RemoveAll(dir)

	theError := errors.New("oops")
	_, err := f.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		// Append three items. This creates two files immediately,
		// because the table size limit of the test freezer is 2048.
		require.NoError(t, op.AppendRaw("test", 0, make([]byte, 2048)))
		require.NoError(t, op.AppendRaw("test", 1, make([]byte, 2048)))
		require.NoError(t, op.AppendRaw("test", 2, make([]byte, 2048)))
		return theError
	})
	if err != theError {
		t.Errorf("ModifyAncients returned wrong error %q", err)
	}
	checkAncientCount(t, f, "test", 0)
	f.Close()

	// Reopen and check that the rolled-back data doesn't reappear.
	tables := map[string]bool{"test": true}
	f2, err := newFreezer(dir, "", false, 2049, tables)
	if err != nil {
		t.Fatalf("can't reopen freezer after failed ModifyAncients: %v", err)
	}
	defer f2.Close()
	checkAncientCount(t, f2, "test", 0)
}

// This test runs ModifyAncients and Ancient concurrently with each other.
func TestFreezerConcurrentModifyRetrieve(t *testing.T) {
	t.Parallel()

	f, dir := newFreezerForTesting(t, freezerTestTableDef)
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
				for rc := 0; rc < 80; rc++ {
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
	f, dir := newFreezerForTesting(t, freezerTestTableDef)
	defer os.RemoveAll(dir)
	defer f.Close()

	var item = make([]byte, 256)

	for i := 0; i < 1000; i++ {
		// First reset and write 100 items.
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
		checkAncientCount(t, f, "test", 100)

		// Now append 100 more items and truncate concurrently.
		var (
			wg          sync.WaitGroup
			truncateErr error
			modifyErr   error
		)
		wg.Add(3)
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
		go func() {
			f.AncientSize("test")
			wg.Done()
		}()
		wg.Wait()

		// Now check the outcome. If the truncate operation went through first, the append
		// fails, otherwise it succeeds. In either case, the freezer should be positioned
		// at 10 after both operations are done.
		if truncateErr != nil {
			t.Fatal("concurrent truncate failed:", err)
		}
		if !(errors.Is(modifyErr, nil) || errors.Is(modifyErr, errOutOrderInsertion)) {
			t.Fatal("wrong error from concurrent modify:", modifyErr)
		}
		checkAncientCount(t, f, "test", 10)
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

// checkAncientCount verifies that the freezer contains n items.
func checkAncientCount(t *testing.T, f *freezer, kind string, n uint64) {
	t.Helper()

	if frozen, _ := f.Ancients(); frozen != n {
		t.Fatalf("Ancients() returned %d, want %d", frozen, n)
	}

	// Check at index n-1.
	if n > 0 {
		index := n - 1
		if ok, _ := f.HasAncient(kind, index); !ok {
			t.Errorf("HasAncient(%q, %d) returned false unexpectedly", kind, index)
		}
		if _, err := f.Ancient(kind, index); err != nil {
			t.Errorf("Ancient(%q, %d) returned unexpected error %q", kind, index, err)
		}
	}

	// Check at index n.
	index := n
	if ok, _ := f.HasAncient(kind, index); ok {
		t.Errorf("HasAncient(%q, %d) returned true unexpectedly", kind, index)
	}
	if _, err := f.Ancient(kind, index); err == nil {
		t.Errorf("Ancient(%q, %d) didn't return expected error", kind, index)
	} else if err != errOutOfBounds {
		t.Errorf("Ancient(%q, %d) returned unexpected error %q", kind, index, err)
	}
}
