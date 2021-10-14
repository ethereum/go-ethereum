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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package trie

import (
	"fmt"
	"io/ioutil"
	"math/rand"
	"os"
	"path"
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/rlp"
)

func genDiffs(n int) []reverseDiff {
	var (
		parent common.Hash
		ret    []reverseDiff
	)
	for i := 0; i < n; i++ {
		var (
			root   = randomHash()
			states []stateDiff
		)
		for j := 0; j < 10; j++ {
			if rand.Intn(2) == 0 {
				states = append(states, stateDiff{
					Key: randBytes(30),
					Val: randBytes(30),
				})
			} else {
				states = append(states, stateDiff{
					Key: randBytes(30),
					Val: []byte{},
				})
			}
		}
		ret = append(ret, reverseDiff{
			Version: reverseDiffVersion,
			Parent:  parent,
			Root:    root,
			States:  states,
		})
		parent = root
	}
	return ret
}

func TestLoadStoreReverseDiff(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "testing")
	if err != nil {
		panic("Failed to allocate tempdir")
	}
	db, err := rawdb.NewLevelDBDatabaseWithFreezer(dir, 16, 16, path.Join(dir, "test-fr"), "", false)
	if err != nil {
		panic("Failed to create database")
	}
	defer os.RemoveAll(dir)

	var diffs = genDiffs(10)
	for i := 0; i < len(diffs); i++ {
		blob, err := rlp.EncodeToBytes(diffs[i])
		if err != nil {
			t.Fatalf("Failed to encode reverse diff %v", err)
		}
		rawdb.WriteReverseDiff(db, uint64(i+1), blob, diffs[i].Parent)
		rawdb.WriteReverseDiffLookup(db, diffs[i].Parent, uint64(i+1))
	}
	for i := 0; i < len(diffs); i++ {
		diff, err := loadReverseDiff(db, uint64(i+1))
		if err != nil {
			t.Fatalf("Failed to load reverse diff %v", err)
		}
		if diff.Version != reverseDiffVersion {
			t.Fatalf("Unexpected version want %d got %d", reverseDiffVersion, diff.Version)
		}
		if diff.Root != diffs[i].Root {
			t.Fatalf("Unexpected root want %x got %x", diffs[i].Root, diff.Root)
		}
		if diff.Parent != diffs[i].Parent {
			t.Fatalf("Unexpected parent want %x got %x", diffs[i].Parent, diff.Parent)
		}
		if !reflect.DeepEqual(diff.States, diffs[i].States) {
			t.Fatal("Unexpected states")
		}
	}
}

func assertReverseDiff(t *testing.T, db ethdb.Database, id uint64, exist bool) {
	blob := rawdb.ReadReverseDiff(db, id)
	if exist && len(blob) == 0 {
		t.Fatalf("Failed to load reverse diff, %d", id)
	}
	if !exist && len(blob) != 0 {
		t.Fatalf("Unexpected reverse diff, %d", id)
	}
}

func assertReverseDiffInRange(t *testing.T, db ethdb.Database, from, to uint64, exist bool) {
	for i := from; i <= to; i++ {
		assertReverseDiff(t, db, i, exist)
	}
}

func TestTruncateTailReverseDiff(t *testing.T) {
	dir, err := ioutil.TempDir(os.TempDir(), "testing")
	if err != nil {
		panic("Failed to allocate tempdir")
	}
	db, err := rawdb.NewLevelDBDatabaseWithFreezer(dir, 16, 16, path.Join(dir, "test-fr"), "", false)
	if err != nil {
		panic("Failed to create database")
	}
	defer os.RemoveAll(dir)

	var diffs = genDiffs(10)
	for i := 0; i < len(diffs); i++ {
		blob, err := rlp.EncodeToBytes(diffs[i])
		if err != nil {
			t.Fatalf("Failed to encode reverse diff %v", err)
		}
		rawdb.WriteReverseDiff(db, uint64(i+1), blob, diffs[i].Parent)
		rawdb.WriteReverseDiffLookup(db, diffs[i].Parent, uint64(i+1))

		// Only keep the latest reverse diff in disk, ensure all
		// older are evicted.
		pruned, _ := truncateFromTail(db, uint64(i+1), 1)
		if i != 0 && pruned != 1 {
			t.Error("Unexpected pruned items", "want", 1, "got", pruned)
		}
		assertReverseDiffInRange(t, db, 0, uint64(i), false)
		assertReverseDiff(t, db, uint64(i+1), true)
	}
}

// TestRepairReverseDiff tests the reverse diff history repair. It simulates
// a few corner cases and checks if the database has the expected repair behaviour.
func TestRepairReverseDiff(t *testing.T) {
	//log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	setup := func() (ethdb.Database, []reverseDiff, func()) {
		dir, err := ioutil.TempDir(os.TempDir(), fmt.Sprintf("testing-%d", rand.Uint64()))
		if err != nil {
			panic("Failed to allocate tempdir")
		}
		db, err := rawdb.NewLevelDBDatabaseWithFreezer(dir, 16, 16, path.Join(dir, "test-fr"), "", false)
		if err != nil {
			panic("Failed to create database")
		}
		var diffs = genDiffs(10)
		for i := 0; i < len(diffs); i++ {
			blob, err := rlp.EncodeToBytes(diffs[i])
			if err != nil {
				t.Fatalf("Failed to encode reverse diff %v", err)
			}
			rawdb.WriteReverseDiff(db, uint64(i+1), blob, diffs[i].Parent)
			rawdb.WriteReverseDiffLookup(db, diffs[i].Parent, uint64(i+1))
		}
		return db, diffs, func() {
			os.RemoveAll(dir)
		}
	}

	// Scenario a:
	// - head reverse diff in leveldb is lower than freezer, it can happen that
	//   reverse diff is persisted while the head flag is not updated yet.
	//   The extra reverse diff in freezer is expected to be truncated
	t.Run("Truncate-extra-rdiffs-match-root", func(t *testing.T) {
		t.Parallel()

		db, diffs, teardown := setup()
		defer teardown()

		rawdb.WriteReverseDiffHead(db, uint64(9)) // Head reverse diff in ldb: 9
		repairReverseDiff(db, diffs[len(diffs)-2].Root)

		assertReverseDiffInRange(t, db, uint64(1), uint64(9), true)
		assertReverseDiff(t, db, uint64(10), false)
		if head := rawdb.ReadReverseDiffHead(db); head != uint64(9) {
			t.Fatalf("Unexpected reverse diff head %d", head)
		}
	})
	t.Run("Truncate-extra-rdiffs-unmatch-root", func(t *testing.T) {
		t.Parallel()

		db, _, teardown := setup()
		defer teardown()

		rawdb.WriteReverseDiffHead(db, uint64(9))
		repairReverseDiff(db, randomHash())

		assertReverseDiffInRange(t, db, uint64(1), uint64(10), false)
		if head := rawdb.ReadReverseDiffHead(db); head != uint64(0) {
			t.Fatalf("Unexpected reverse diff head %d", head)
		}
	})

	// Scenario b:
	// - head reverse diff in leveldb is higher than freezer, it's not supposed
	//   to be occurred.
	//   In this case all the existent reverse diffs should all be dropped.
	t.Run("Truncate-unknown-rdiffs-zero-tail", func(t *testing.T) {
		t.Parallel()

		db, _, teardown := setup()
		defer teardown()

		rawdb.WriteReverseDiffHead(db, uint64(11))
		repairReverseDiff(db, randomHash())

		assertReverseDiffInRange(t, db, uint64(1), uint64(10), false)
		if head := rawdb.ReadReverseDiffHead(db); head != uint64(0) {
			t.Fatalf("Unexpected reverse diff head %d", head)
		}
	})
	t.Run("Truncate-unknown-rdiffs-non-zero-tail", func(t *testing.T) {
		t.Parallel()

		db, _, teardown := setup()
		defer teardown()

		truncateFromTail(db, uint64(10), uint64(1)) // Stored rdiffs: [rdiff-10, tail = 9]

		rawdb.WriteReverseDiffHead(db, uint64(11))
		repairReverseDiff(db, randomHash())

		assertReverseDiffInRange(t, db, uint64(1), uint64(10), false)
		if head := rawdb.ReadReverseDiffHead(db); head != uint64(9) {
			t.Fatalf("Unexpected reverse diff head %d", head)
		}
	})

	// Scenario c:
	// - head reverse diff in leveldb matches with the freezer
	t.Run("Aligned-reverse-diff-same-root", func(t *testing.T) {
		t.Parallel()

		db, diffs, teardown := setup()
		defer teardown()

		rawdb.WriteReverseDiffHead(db, uint64(10))
		repairReverseDiff(db, diffs[len(diffs)-1].Root)

		assertReverseDiffInRange(t, db, uint64(1), uint64(10), true)
		if head := rawdb.ReadReverseDiffHead(db); head != uint64(10) {
			t.Fatalf("Unexpected reverse diff head %d", head)
		}
	})
	t.Run("Aligned-reverse-diff-non-matched-root", func(t *testing.T) {
		t.Parallel()

		db, _, teardown := setup()
		defer teardown()

		rawdb.WriteReverseDiffHead(db, uint64(10))
		repairReverseDiff(db, randomHash())

		assertReverseDiffInRange(t, db, uint64(1), uint64(10), false)
		if head := rawdb.ReadReverseDiffHead(db); head != uint64(0) {
			t.Fatalf("Unexpected reverse diff head %d", head)
		}
	})
}
