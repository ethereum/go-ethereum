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
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
)

func TestFreezerConcurrentModifyTruncate(t *testing.T) {
	dir, err := ioutil.TempDir("", "freezer")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(dir)

	tables := map[string]bool{"test": true}
	f, err := newFreezer(dir, "", false, tables)
	if err != nil {
		t.Fatal("can't open freezer", err)
	}
	defer f.Close()

	var item = make([]byte, 256)

	for i := 0; i < 5000; i++ {
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

		// Now append 100 more items and truncate concurrently.
		var (
			wg          sync.WaitGroup
			truncateErr error
			modifyErr   error
		)
		wg.Add(2)
		go func() {
			truncateErr = f.TruncateAncients(0)
			wg.Done()
		}()
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
		wg.Wait()

		// Now check the outcome. If the truncate operation went through first,
		// the append fails, otherwise it succeeds. In either case, the freezer
		// should be positioned at zero.
		if truncateErr != nil {
			t.Fatal("concurrent truncate failed:", err)
		}
		if modifyErr != nil && modifyErr != errOutOfBounds {
			t.Fatal("wrong error from modify:", modifyErr)
		}
		index, err := f.Ancients()
		if err != nil {
			t.Fatal("error from Ancients:", err)
		}
		if index != 0 {
			t.Fatalf("Ancients returned %d, want 0", index)
		}
	}
}

// // TestAppendTruncateParallel is a test to check if the Append/truncate operations are
// // racy.
// //
// // The reason why it's not a regular fuzzer, within tests/fuzzers, is that it is dependent
// // on timing rather than 'clever' input -- there's no determinism.
// func TestAppendTruncateParallel(t *testing.T) {
// 	t.Skip()
//
// 	dir, err := ioutil.TempDir("", "freezer")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer os.RemoveAll(dir)
//
// 	f, err := newCustomTable(dir, "tmp", metrics.NilMeter{}, metrics.NilMeter{}, metrics.NilGauge{}, 8, true)
// 	if err != nil {
// 		t.Fatal(err)
// 	}
//
// 	fill := func(mark uint64) []byte {
// 		data := make([]byte, 8)
// 		binary.LittleEndian.PutUint64(data, mark)
// 		return data
// 	}
//
// 	for i := 0; i < 5000; i++ {
// 		require.NoError(t, f.truncate(0))
//
// 		var (
// 			data0 = fill(0)
// 			data1 = fill(1)
// 			batch = f.newBatch()
// 		)
// 		require.NoError(t, batch.AppendRaw(0, data0))
// 		require.NoError(t, batch.Commit())
//
// 		var wg sync.WaitGroup
// 		wg.Add(2)
// 		go func() {
// 			assert.NoError(t, f.truncate(0))
// 			wg.Done()
// 		}()
// 		go func() {
// 			batch := f.newBatch()
// 			assert.NoError(t, batch.AppendRaw(1, data1))
// 			assert.NoError(t, batch.Commit())
// 			wg.Done()
// 		}()
// 		wg.Wait()
//
// 		if have, err := f.Retrieve(0); err == nil {
// 			if !bytes.Equal(have, data0) {
// 				t.Fatalf("have %x want %x", have, data0)
// 			}
// 		}
// 	}
// }
