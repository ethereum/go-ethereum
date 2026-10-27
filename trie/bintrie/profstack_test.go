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

package bintrie_test

import (
	"math/rand/v2"
	"runtime"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/triedb"
)

// TestRecursionKeepsProfileStacksBounded checks that the tree's recursive walks
// reach both children through one call site. Go's heap profiler keeps a record
// per distinct allocation stack for the life of the process. A walk that
// descends left and right from different lines makes each stack spell out its
// key's path, so the records grow with every new key written: on a mainnet-sized
// tree that held over 1 GiB after a long access-list replay.
func TestRecursionKeepsProfileStacksBounded(t *testing.T) {
	db := triedb.NewDatabase(rawdb.NewMemoryDatabase(), triedb.PBTDefaults)
	defer db.Close()

	rng := rand.NewChaCha8([32]byte{1})
	write := func(root common.Hash, block uint64) common.Hash {
		tr := openTrie(t, db, root)
		for range 2000 {
			var (
				addr        common.Address
				slot, value [32]byte
			)
			rng.Read(addr[:])
			rng.Read(slot[:])
			rng.Read(value[:])
			value[0] |= 1 // a zero value deletes
			if err := tr.UpdateStorage(addr, slot[:], value[:]); err != nil {
				t.Fatal(err)
			}
		}
		return commitTrie(t, db, tr, root, block)
	}
	root := types.EmptyBinaryHash
	for block := uint64(1); block <= 10; block++ {
		root = write(root, block)
	}
	// count brings the profile up to date, which lags by up to two collections,
	// then counts every bucket, freed or not.
	count := func() int {
		runtime.GC()
		runtime.GC()
		n, _ := runtime.MemProfile(nil, true)
		return n
	}
	defer func(rate int) { runtime.MemProfileRate = rate }(runtime.MemProfileRate)
	runtime.MemProfileRate = 1

	// Two blocks meet the walks' bounded set of stacks; the third, written from
	// the same line, must add little.
	var grown int
	prev := count()
	for block := uint64(11); block <= 13; block++ {
		root = write(root, block)
		n := count()
		grown, prev = n-prev, n
	}
	if grown > 5000 { // 350 with one call site per walk, 70k without
		t.Fatalf("heap profile grew by %d buckets over 2000 writes: a tree walk encodes key paths in its call stacks", grown)
	}
}
