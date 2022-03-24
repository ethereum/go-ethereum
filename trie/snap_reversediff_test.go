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

func makeDiffs(n int) []reverseDiff {
	var (
		parent = randomHash()
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
	dir := t.TempDir()
	db, err := rawdb.NewLevelDBDatabaseWithFreezer(dir, 16, 16, dir, "", false)
	if err != nil {
		panic("Failed to create database")
	}
	freezer, _ := openFreezer(path.Join(dir, "freezer"), false)

	var diffs = makeDiffs(10)
	for i := 0; i < len(diffs); i++ {
		blob, err := rlp.EncodeToBytes(diffs[i])
		if err != nil {
			t.Fatalf("Failed to encode reverse diff %v", err)
		}
		rawdb.WriteReverseDiff(freezer, uint64(i+1), blob, diffs[i].Parent)
		rawdb.WriteReverseDiffLookup(db, diffs[i].Parent, uint64(i+1))
	}
	for i := 0; i < len(diffs); i++ {
		diff, err := loadReverseDiff(freezer, uint64(i+1))
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

func assertReverseDiff(t *testing.T, freezer *rawdb.Freezer, db ethdb.Database, id uint64, exist bool) {
	blob := rawdb.ReadReverseDiff(freezer, id)
	if exist && len(blob) == 0 {
		t.Errorf("Failed to load reverse diff, %d", id)
	}
	if !exist && len(blob) != 0 {
		t.Errorf("Unexpected reverse diff, %d", id)
	}
	hash := rawdb.ReadReverseDiffHash(freezer, id)
	if exist && hash == (common.Hash{}) {
		t.Errorf("Failed to load reverse diff hash, %d", id)
	}
	if !exist && hash != (common.Hash{}) {
		t.Errorf("Unexpected reverse diff hash, %d", id)
	}
	diffid := rawdb.ReadReverseDiffLookup(db, hash)
	if exist && diffid == nil {
		t.Fatalf("Failed to load reverse diff lookup, %d", id)
	}
	if !exist && diffid != nil {
		t.Fatalf("Unexpected reverse diff lookup, %d", id)
	}
}

func assertReverseDiffInRange(t *testing.T, freezer *rawdb.Freezer, db ethdb.Database, from, to uint64, exist bool) {
	for i := from; i <= to; i++ {
		assertReverseDiff(t, freezer, db, i, exist)
	}
}

func TestTruncateHeadReverseDiff(t *testing.T) {
	dir := t.TempDir()
	db, err := rawdb.NewLevelDBDatabaseWithFreezer(dir, 16, 16, dir, "", false)
	if err != nil {
		panic("Failed to create database")
	}
	freezer, _ := openFreezer(path.Join(dir, "freezer"), false)

	var diffs = makeDiffs(10)
	for i := 0; i < len(diffs); i++ {
		blob, err := rlp.EncodeToBytes(diffs[i])
		if err != nil {
			t.Fatalf("Failed to encode reverse diff %v", err)
		}
		rawdb.WriteReverseDiff(freezer, uint64(i+1), blob, diffs[i].Parent)
		rawdb.WriteReverseDiffLookup(db, diffs[i].Parent, uint64(i+1))
	}
	for i := len(diffs); i > 0; i-- {
		pruned, err := truncateFromHead(freezer, db, uint64(i-1))
		if err != nil {
			t.Fatalf("Failed to truncate from head %v", err)
		}
		if i != 0 && pruned != 1 {
			t.Error("Unexpected pruned items", "want", 1, "got", pruned)
		}
		assertReverseDiffInRange(t, freezer, db, uint64(i), uint64(10), false)
		assertReverseDiffInRange(t, freezer, db, uint64(1), uint64(i-1), true)
	}
}

func TestTruncateTailReverseDiff(t *testing.T) {
	dir := t.TempDir()
	db, err := rawdb.NewLevelDBDatabaseWithFreezer(dir, 16, 16, dir, "", false)
	if err != nil {
		panic("Failed to create database")
	}
	freezer, _ := openFreezer(path.Join(dir, "freezer"), false)

	var diffs = makeDiffs(10)
	for i := 0; i < len(diffs); i++ {
		blob, err := rlp.EncodeToBytes(diffs[i])
		if err != nil {
			t.Fatalf("Failed to encode reverse diff %v", err)
		}
		rawdb.WriteReverseDiff(freezer, uint64(i+1), blob, diffs[i].Parent)
		rawdb.WriteReverseDiffLookup(db, diffs[i].Parent, uint64(i+1))

		pruned, _ := truncateFromTail(freezer, db, uint64(i))
		if i != 0 && pruned != 1 {
			t.Error("Unexpected pruned items", "want", 1, "got", pruned)
		}
		assertReverseDiffInRange(t, freezer, db, uint64(1), uint64(i), false)
		assertReverseDiff(t, freezer, db, uint64(i+1), true)
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
		dir := t.TempDir()
		db, err := rawdb.NewLevelDBDatabaseWithFreezer(dir, 16, 16, dir, "", false)
		if err != nil {
			panic("Failed to create database")
		}
		freezer, _ := openFreezer(path.Join(dir, "freezer"), false)

		var diffs = makeDiffs(10)
		for i := 0; i < len(diffs); i++ {
			blob, err := rlp.EncodeToBytes(diffs[i])
			if err != nil {
				t.Fatalf("Failed to encode reverse diff %v", err)
			}
			rawdb.WriteReverseDiff(freezer, uint64(i+1), blob, diffs[i].Parent)
			rawdb.WriteReverseDiffLookup(db, diffs[i].Parent, uint64(i+1))
		}

		pruned, _ := truncateFromTail(freezer, db, uint64(10)-c.limit)
		if pruned != c.expPruned {
			t.Error("Unexpected pruned items", "want", c.expPruned, "got", pruned)
		}
		if c.empty {
			assertReverseDiffInRange(t, freezer, db, uint64(1), uint64(10), false)
		} else {
			assertReverseDiffInRange(t, freezer, db, uint64(1), c.maxPruned, false)
			assertReverseDiff(t, freezer, db, c.minUnpruned, true)
		}
	}
}

// TestRepairReverseDiff tests the reverse diff history truncateDiffs. It simulates
// a few corner cases and checks if the database has the expected truncateDiffs behaviour.
func TestRepairReverseDiff(t *testing.T) {
	//log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	setup := func() (ethdb.Database, *rawdb.Freezer, []reverseDiff, func()) {
		dir := t.TempDir()
		db, err := rawdb.NewLevelDBDatabaseWithFreezer(dir, 16, 16, dir, "", false)
		if err != nil {
			panic("Failed to create database")
		}
		freezer, _ := openFreezer(path.Join(dir, "freezer"), false)

		var diffs = makeDiffs(10)
		for i := 0; i < len(diffs); i++ {
			blob, err := rlp.EncodeToBytes(diffs[i])
			if err != nil {
				t.Fatalf("Failed to encode reverse diff %v", err)
			}
			rawdb.WriteReverseDiff(freezer, uint64(i+1), blob, diffs[i].Parent)
			rawdb.WriteReverseDiffLookup(db, diffs[i].Parent, uint64(i+1))
		}
		return db, freezer, diffs, func() { os.RemoveAll(dir) }
	}

	// Scenario 1:
	// - head reverse diff in leveldb is lower than freezer, it can happen that
	//   reverse diff is persisted while corresponding state is not flushed.
	//   The extra reverse diff in freezer is expected to be truncated
	t.Run("Truncate-extra-rdiffs-match-root", func(t *testing.T) {
		t.Parallel()

		db, freezer, _, teardown := setup()
		defer teardown()

		// Block9's root.
		truncateDiffs(freezer, db, 9)
		assertReverseDiffInRange(t, freezer, db, uint64(1), uint64(9), true)
		assertReverseDiff(t, freezer, db, uint64(10), false)
	})

	// Scenario 2:
	// - head reverse diff in leveldb matches with the freezer
	t.Run("Aligned-reverse-diff-same-root", func(t *testing.T) {
		t.Parallel()

		db, freezer, _, teardown := setup()
		defer teardown()

		truncateDiffs(freezer, db, 10)
		assertReverseDiffInRange(t, freezer, db, uint64(1), uint64(10), true)
	})
}
