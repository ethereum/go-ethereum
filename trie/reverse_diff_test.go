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

func TestTruncateHeadReverseDiff(t *testing.T) {
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
	for i := len(diffs); i > 0; i-- {
		pruned, err := truncateFromHead(db, uint64(i-1))
		if err != nil {
			t.Fatalf("Failed to truncate from head %v", err)
		}
		if i != 0 && pruned != 1 {
			t.Error("Unexpected pruned items", "want", 1, "got", pruned)
		}
		assertReverseDiffInRange(t, db, uint64(i), uint64(10), false)
		assertReverseDiffInRange(t, db, uint64(1), uint64(i-1), true)
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

		pruned, _ := truncateFromTail(db, uint64(i))
		if i != 0 && pruned != 1 {
			t.Error("Unexpected pruned items", "want", 1, "got", pruned)
		}
		assertReverseDiffInRange(t, db, uint64(1), uint64(i), false)
		assertReverseDiff(t, db, uint64(i+1), true)
	}
}

func TestTruncateTailReverseDiffs(t *testing.T) {
	var cases = []struct {
		limit       uint64
		expPruned   int
		maxPruned   uint64
		minUnpruned uint64
		empty       bool
	}{
		{
			1, 9, 9, 10, false,
		},
		{
			0, 10, 10, 0 /* no meaning */, true,
		},
		{
			10, 0, 0, 1, false,
		},
	}
	for _, c := range cases {
		dir, err := ioutil.TempDir(os.TempDir(), "")
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

		pruned, _ := truncateFromTail(db, uint64(10)-c.limit)
		if pruned != c.expPruned {
			t.Error("Unexpected pruned items", "want", c.expPruned, "got", pruned)
		}
		if c.empty {
			assertReverseDiffInRange(t, db, uint64(1), uint64(10), false)
		} else {
			assertReverseDiffInRange(t, db, uint64(1), c.maxPruned, false)
			assertReverseDiff(t, db, c.minUnpruned, true)
		}
	}
}

func TestPurgeReverseDiffs(t *testing.T) {
	var cases = []struct {
		prePruned int
		expHead   uint64
	}{
		{0, 0},
		{1, 1},
		{10, 10},
	}
	for _, c := range cases {
		dir, err := ioutil.TempDir(os.TempDir(), "")
		if err != nil {
			panic("Failed to allocate tempdir")
		}
		db, err := rawdb.NewLevelDBDatabaseWithFreezer(dir, 16, 16, path.Join(dir, "test-fr"), "", false)
		if err != nil {
			panic("Failed to create database")
		}
		defer os.RemoveAll(dir)

		var diffs = genDiffs(15)
		for i := 0; i < 10; i++ {
			blob, err := rlp.EncodeToBytes(diffs[i])
			if err != nil {
				t.Fatalf("Failed to encode reverse diff %v", err)
			}
			rawdb.WriteReverseDiff(db, uint64(i+1), blob, diffs[i].Parent)
			rawdb.WriteReverseDiffLookup(db, diffs[i].Parent, uint64(i+1))
		}

		// Deleted the items in the tail if it's required
		if c.prePruned != 0 {
			pruned, err := truncateFromTail(db, uint64(c.prePruned))
			if err != nil {
				t.Fatalf("Failed to truncate reverse diff %v", err)
			}
			if pruned != c.prePruned {
				t.Fatalf("Unexpected pruned items %d - %d", pruned, c.prePruned)
			}
		}

		// Purge all the reverse diffs stored, ensure nothing left
		newHead, err := purgeReverseDiffs(db)
		if err != nil {
			t.Fatalf("Failed to purgeReverseDiffs reverse diff %v", err)
		}
		if newHead != c.expHead {
			t.Fatalf("Unexpected new head %d - %d", newHead, c.expHead)
		}
		assertReverseDiffInRange(t, db, uint64(1), uint64(10), false)

		// Push new reverse diffs on top, ensure everything flushed is accessible
		for i := c.expHead + 1; i <= 15; i++ {
			blob, err := rlp.EncodeToBytes(diffs[i-1])
			if err != nil {
				t.Fatalf("Failed to encode reverse diff %v", err)
			}
			rawdb.WriteReverseDiff(db, i, blob, diffs[i-1].Parent)
			rawdb.WriteReverseDiffLookup(db, diffs[i-1].Parent, i)
		}
		assertReverseDiffInRange(t, db, c.expHead+1, uint64(15), true)
	}
}

// TestRepairReverseDiff tests the reverse diff history repairReverseDiffs. It simulates
// a few corner cases and checks if the database has the expected repairReverseDiffs behaviour.
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

	// Scenario 1:
	// - head reverse diff in leveldb is lower than freezer, it can happen that
	//   reverse diff is persisted while corresponding state is not flushed.
	//   The extra reverse diff in freezer is expected to be truncated
	t.Run("Truncate-extra-rdiffs-match-root", func(t *testing.T) {
		t.Parallel()

		db, _, teardown := setup()
		defer teardown()

		// Block9's root.
		repairReverseDiffs(db, 9)
		assertReverseDiffInRange(t, db, uint64(1), uint64(9), true)
		assertReverseDiff(t, db, uint64(10), false)
	})

	// Scenario 2:
	// - head reverse diff in leveldb matches with the freezer
	t.Run("Aligned-reverse-diff-same-root", func(t *testing.T) {
		t.Parallel()

		db, _, teardown := setup()
		defer teardown()

		repairReverseDiffs(db, 10)
		assertReverseDiffInRange(t, db, uint64(1), uint64(10), true)
	})
}
