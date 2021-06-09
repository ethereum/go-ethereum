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

// This test runs ModifyAncients and TruncateAncients concurrently with each other.
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
			truncateErr = f.TruncateAncients(0)
			wg.Done()
		}()
		wg.Wait()

		// Now check the outcome. If the truncate operation went through first, the append
		// fails, otherwise it succeeds. In either case, the freezer should be positioned
		// at zero after both operations are done.
		if truncateErr != nil {
			t.Fatal("concurrent truncate failed:", err)
		}
		if modifyErr != nil && modifyErr != errOutOrderInsertion {
			t.Fatal("wrong error from concurrent modify:", modifyErr)
		}
		if frozen, _ := f.Ancients(); frozen != 0 {
			t.Fatalf("Ancients returned %d, want 0", frozen)
		}
	}
}
