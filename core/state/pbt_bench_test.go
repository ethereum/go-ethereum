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

package state

import (
	"encoding/binary"
	"fmt"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// State-level benchmarks comparing the merkle-patricia trie against the
// EIP-8297 binary tree through the same StateDB API.
//
// What these measure: the shape and count of lookups on the read path. Both
// arms run over an in-memory key-value store, so the numbers are CPU and
// lookup-count, not device I/O - a real disk would widen any gap rather than
// narrow it. The clean caches are deliberately tiny so reads reach the trie
// instead of being served from RAM, which is the whole point of the
// comparison.
//
// The arms to record, in order:
//
//	MPT          merkle-patricia trie, flat state as usual
//	PBT-flat-off binary tree today: flat state built, then discarded
//	PBT-flat-on  binary tree once the generation marker no longer blocks it
//
// The middle arm stops existing the moment flat state is unlocked, so it has
// to be captured first. Run with:
//
//	go test ./core/state/ -run XXX -bench 'BenchmarkState' -benchtime 2000x

const (
	benchAccounts = 20000 // number of accounts populated
	benchSlots    = 2     // storage slots per account: one header, one overflow
)

// benchStateAddr derives the address of the i'th benchmark account.
func benchStateAddr(i int) common.Address {
	var a common.Address
	binary.BigEndian.PutUint64(a[12:], uint64(i)+1)
	return a
}

// benchStateSlot derives the j'th storage slot key. Slot 1 lives in the binary
// tree's header stem, slot 100 in its overflow bucket, so both storage homes
// are exercised.
func benchStateSlot(j int) common.Hash {
	var slot common.Hash
	if j == 0 {
		slot[31] = 1
		return slot
	}
	slot[31] = 100
	return slot
}

// newBenchDatabase opens a state database over a fresh in-memory store, on the
// path scheme, with caches small enough that reads are not absorbed by them.
func newBenchDatabase(pbt bool) *CachingDB {
	disk := rawdb.NewMemoryDatabase()
	config := &triedb.Config{
		IsPBT: pbt,
		PathDB: &pathdb.Config{
			TrieCleanSize:  1024,
			StateCleanSize: 1024,
			NoAsyncFlush:   true,
		},
	}
	return NewDatabase(triedb.NewDatabase(disk, config), NewCodeDB(disk))
}

// buildBenchState populates a state with benchAccounts accounts, each holding
// benchSlots storage slots, and returns the database and the committed root.
// The state is written through the ordinary StateDB commit path so that any
// flat-state accumulation happens exactly as it would during block processing.
func buildBenchState(b *testing.B, pbt bool) (*CachingDB, common.Hash) {
	b.Helper()

	db := newBenchDatabase(pbt)
	root := types.EmptyRootHash
	if pbt {
		root = types.EmptyBinaryHash
	}
	statedb, err := New(root, db)
	if err != nil {
		b.Fatal(err)
	}
	for i := 0; i < benchAccounts; i++ {
		addr := benchStateAddr(i)
		statedb.SetBalance(addr, uint256.NewInt(uint64(i)+1), tracing.BalanceChangeUnspecified)
		statedb.SetNonce(addr, uint64(i), tracing.NonceChangeUnspecified)
		for j := 0; j < benchSlots; j++ {
			slot := benchStateSlot(j)
			statedb.SetState(addr, slot, slot)
		}
	}
	// Cancun semantics: no storage wiping, matching how a live chain commits.
	root, err = statedb.Commit(0, true, true)
	if err != nil {
		b.Fatal(err)
	}
	if err := db.TrieDB().Commit(root, false); err != nil {
		b.Fatal(err)
	}
	return db, root
}

// benchBackends is the set of arms every benchmark below runs.
var benchBackends = []struct {
	name string
	pbt  bool
}{
	{"MPT", false},
	{"PBT", true},
}

// BenchmarkStateAccountRead measures reading an account that is not already
// cached. This is the read the whole flat-state effort is about: the binary
// tree currently resolves it by walking the trie, where the merkle-patricia
// trie resolves it from flat state.
func BenchmarkStateAccountRead(b *testing.B) {
	for _, backend := range benchBackends {
		b.Run(backend.name, func(b *testing.B) {
			db, root := buildBenchState(b, backend.pbt)

			statedb, err := New(root, db)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				// Stride by a large odd number so consecutive reads land far
				// apart in the key space and are unlikely to share trie nodes.
				addr := benchStateAddr((i * 7919) % benchAccounts)
				if statedb.GetBalance(addr).IsZero() {
					b.Fatalf("account %x is empty", addr)
				}
				// A fresh reader every so often, so the per-StateDB object
				// cache does not turn this into a memory benchmark.
				if i%256 == 255 {
					b.StopTimer()
					statedb, err = New(root, db)
					if err != nil {
						b.Fatal(err)
					}
					b.StartTimer()
				}
			}
		})
	}
}

// BenchmarkStateStorageRead measures reading a storage slot, alternating
// between the two homes the binary tree keeps storage in.
func BenchmarkStateStorageRead(b *testing.B) {
	for _, backend := range benchBackends {
		b.Run(backend.name, func(b *testing.B) {
			db, root := buildBenchState(b, backend.pbt)

			statedb, err := New(root, db)
			if err != nil {
				b.Fatal(err)
			}
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				addr := benchStateAddr((i * 7919) % benchAccounts)
				slot := benchStateSlot(i % benchSlots)
				if statedb.GetState(addr, slot) == (common.Hash{}) {
					b.Fatalf("slot %x of %x is empty", slot, addr)
				}
				if i%256 == 255 {
					b.StopTimer()
					statedb, err = New(root, db)
					if err != nil {
						b.Fatal(err)
					}
					b.StartTimer()
				}
			}
		})
	}
}

// BenchmarkStateCommit measures committing a block-sized batch of mutations,
// which is the side of the ledger flat state is expected to cost rather than
// save. The gate is that the binary tree stays within 1.5x of the merkle
// trie here while winning on reads.
func BenchmarkStateCommit(b *testing.B) {
	const mutations = 1000

	for _, backend := range benchBackends {
		b.Run(backend.name, func(b *testing.B) {
			db, root := buildBenchState(b, backend.pbt)
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				b.StopTimer()
				statedb, err := New(root, db)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()

				for j := 0; j < mutations; j++ {
					addr := benchStateAddr((i*mutations + j) % benchAccounts)
					statedb.SetBalance(addr, uint256.NewInt(uint64(i*mutations+j)+2), tracing.BalanceChangeUnspecified)
				}
				if _, err := statedb.Commit(uint64(i+1), true, true); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestBenchStateBuilds guards the harness itself: a benchmark that silently
// built an empty state would report a very fast lookup of nothing.
func TestBenchStateBuilds(t *testing.T) {
	for _, backend := range benchBackends {
		t.Run(backend.name, func(t *testing.T) {
			db, root := buildBenchState(&testing.B{}, backend.pbt)

			statedb, err := New(root, db)
			if err != nil {
				t.Fatal(err)
			}
			for _, i := range []int{0, benchAccounts / 2, benchAccounts - 1} {
				addr := benchStateAddr(i)
				if got := statedb.GetBalance(addr).Uint64(); got != uint64(i)+1 {
					t.Fatalf("account %d balance is %d, want %d", i, got, i+1)
				}
				for j := 0; j < benchSlots; j++ {
					slot := benchStateSlot(j)
					if got := statedb.GetState(addr, slot); got != slot {
						t.Fatalf("account %d slot %x is %x, want %x", i, slot, got, slot)
					}
				}
			}
			fmt.Fprintf(testWriter{t}, "%s state root %x\n", backend.name, root)
		})
	}
}

// testWriter adapts *testing.T to io.Writer for diagnostic output.
type testWriter struct{ t *testing.T }

func (w testWriter) Write(p []byte) (int, error) {
	w.t.Logf("%s", p)
	return len(p), nil
}
