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

package snapshot

import (
	"math/rand"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb/memorydb"
)

// Tests that given a database with random data content, all parts of a snapshot
// can be crrectly wiped without touching anything else.
func TestWipe(t *testing.T) {
	// Create a database with some random snapshot data
	db := memorydb.New()
	for i := 0; i < 128; i++ {
		rawdb.WriteAccountSnapshot(db, randomHash(), randomHash().Bytes())
	}
	// Add some random non-snapshot data too to make wiping harder
	for i := 0; i < 500; i++ {
		// Generate keys with wrong length for a state snapshot item
		keysuffix := make([]byte, 31)
		rand.Read(keysuffix)
		db.Put(append(rawdb.SnapshotAccountPrefix, keysuffix...), randomHash().Bytes())
		keysuffix = make([]byte, 33)
		rand.Read(keysuffix)
		db.Put(append(rawdb.SnapshotAccountPrefix, keysuffix...), randomHash().Bytes())
	}
	count := func() (items int) {
		it := db.NewIterator(rawdb.SnapshotAccountPrefix, nil)
		defer it.Release()
		for it.Next() {
			if len(it.Key()) == len(rawdb.SnapshotAccountPrefix)+common.HashLength {
				items++
			}
		}
		return items
	}
	// Sanity check that all the keys are present
	if items := count(); items != 128 {
		t.Fatalf("snapshot size mismatch: have %d, want %d", items, 128)
	}
	// Wipe the accounts
	if err := wipeKeyRange(db, "accounts", rawdb.SnapshotAccountPrefix, nil, nil,
		len(rawdb.SnapshotAccountPrefix)+common.HashLength, snapWipedAccountMeter, true); err != nil {
		t.Fatal(err)
	}
	// Iterate over the database end ensure no snapshot information remains
	if items := count(); items != 0 {
		t.Fatalf("snapshot size mismatch: have %d, want %d", items, 0)
	}
	// Iterate over the database and ensure miscellaneous items are present
	items := 0
	it := db.NewIterator(nil, nil)
	defer it.Release()
	for it.Next() {
		items++
	}
	if items != 1000 {
		t.Fatalf("misc item count mismatch: have %d, want %d", items, 1000)
	}
}
