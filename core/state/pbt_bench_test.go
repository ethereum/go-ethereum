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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb/pebble"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
)

// State-level benchmarks comparing the merkle-patricia trie against the
// EIP-8297 binary tree through the same StateDB API.
//
// # This is not a CI check
//
// It builds a state large enough to be measured honestly, which takes minutes.
// Benchmarks do not run under plain `go test` — only `Test*` does, unless
// `-bench` is passed — so nothing here executes on a normal run, and no `Test*`
// belongs in this file. Run it by hand, occasionally:
//
//	go test ./core/state/ -run XXX -bench BenchmarkStateAccountRead -benchtime 2000x
//	go test ./core/state/ -run XXX -bench BenchmarkStateStorageRead -benchtime 2000x
//	go test ./core/state/ -run XXX -bench BenchmarkStateCommit      -benchtime 20x
//
// Correctness is not this file's job and must not lean on it. That is
// TestPBTFlatStateMatchesTrie, the attestation tests, and the reorg checks —
// all fast, all on every push.
//
// # Why a real database
//
// An earlier version of this ran against rawdb.NewMemoryDatabase(), a Go map.
// That hides everything that makes a key-value store behave like one: key-size
// effects on block layout, compression, bloom filters, block-cache behaviour,
// and on the write side the write-ahead log, memtable flush and compaction. The
// two tries use different key shapes — 34/66-byte tree keys against hex paths —
// so precisely the differences that should show up were the ones a map could not
// show. The commit benchmark was worst affected, omitting the entire write path
// that dominates a real commit.
//
// Two things follow from using pebble, and both are load-bearing:
//
//   - The working set must dwarf the block cache. minCache floors it at 16 MB
//     and there is no way to ask for less, so a small state would sit entirely
//     inside it and report cache hits dressed as disk reads. benchAccounts is
//     sized to put the state an order of magnitude past that.
//   - The data must be out of the memtable. Compact runs after building, or the
//     read benchmarks measure an in-memory skiplist with extra steps — the same
//     problem in a different costume.
const (
	// benchAccounts sizes the state. At roughly 110 bytes per account and 100
	// per slot in the flat tables, plus trie nodes, 1M accounts with two slots
	// each lands in the hundreds of MB — far clear of the 16 MB cache floor.
	// Lower it for a quick sanity run, but then say so alongside any number.
	benchAccounts = 1_000_000
	benchSlots    = 2 // one in the header stem, one in the overflow bucket
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

// buildBenchState populates a state on a real pebble database and returns it
// with the committed root, ready for measurement.
func buildBenchState(b *testing.B, pbt bool) (Database, common.Hash) {
	b.Helper()

	dir := b.TempDir()
	pdb, err := pebble.New(dir, 0, 0, "", false)
	if err != nil {
		b.Fatalf("failed to open the key-value store: %v", err)
	}
	disk, err := rawdb.Open(pdb, rawdb.OpenOptions{Ancient: dir + "/ancient"})
	if err != nil {
		b.Fatalf("failed to open the database: %v", err)
	}
	b.Cleanup(func() { disk.Close() })

	// Start from the production defaults and change only what the benchmark
	// needs. A bare &pathdb.Config{} is not "no options": zero means "keep the
	// entire chain" for both history settings, not "off", and the sanitiser
	// turns a zero checkpoint rate into full-value node records. Configured
	// that way this measures a node writing archive-class trienode history
	// every block - which is precisely the work the binary tree's larger node
	// records make expensive, and which no ordinary node does.
	pcfg := *pathdb.Defaults
	pcfg.NoAsyncFlush = true // determinism: flush inline rather than in the background

	tdb := triedb.NewDatabase(disk, &triedb.Config{
		IsPBT:  pbt,
		PathDB: &pcfg,
	})
	b.Cleanup(func() { tdb.Close() })
	db := NewDatabase(tdb, NewCodeDB(disk))

	root := types.EmptyRootHash
	if pbt {
		root = types.EmptyBinaryHash
	}
	// Build in batches so the write buffer flushes as it would during block
	// processing, rather than accumulating the whole state in memory.
	const perBatch = 50_000
	for start := 0; start < benchAccounts; start += perBatch {
		statedb, err := New(root, db)
		if err != nil {
			b.Fatal(err)
		}
		end := min(start+perBatch, benchAccounts)
		for i := start; i < end; i++ {
			addr := benchStateAddr(i)
			statedb.SetBalance(addr, uint256.NewInt(uint64(i)+1), tracing.BalanceChangeUnspecified)
			statedb.SetNonce(addr, uint64(i), tracing.NonceChangeUnspecified)
			for j := 0; j < benchSlots; j++ {
				slot := benchStateSlot(j)
				statedb.SetState(addr, slot, slot)
			}
		}
		// Cancun semantics: no storage wiping, matching how a live chain commits.
		root, err = statedb.Commit(uint64(start/perBatch), true, true)
		if err != nil {
			b.Fatal(err)
		}
	}
	if err := tdb.Commit(root, false); err != nil {
		b.Fatal(err)
	}
	// Drive the data out of the memtable, so reads below hit SSTables.
	if err := pdb.Compact(nil, nil); err != nil {
		b.Fatalf("failed to compact: %v", err)
	}
	return db, root
}

// benchBackends is the set of arms every benchmark below runs. Only two, and
// both with flat state on: that is the comparison that matters, and both come
// from this harness on this backend so the ratio means something.
var benchBackends = []struct {
	name string
	pbt  bool
}{
	{"MPT", false},
	{"PBT", true},
}

// BenchmarkStateAccountRead measures reading an account that is not already
// cached — the read the whole flat-state effort is about.
//
// The zero-balance check is not decoration: a flat store that answered "absent"
// for everything would be extremely fast, and would otherwise look like a win.
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

// BenchmarkStateCommit measures committing a block-sized batch of mutations.
//
// Each iteration builds on the previous one's root, the way blocks do. Opening
// every iteration from the same root instead would stack b.N sibling layers on
// one parent — a shape block processing never produces, and one that inflates
// later iterations as the layer tree grows. Chaining also lets the database cap
// and flatten on its normal schedule, so the flush cost lands in the
// measurement where it belongs.
//
// Note it carries the read cost too, since mutating an account loads it first.
// That is realistic — block processing reads before it writes — but it means
// the number is not purely commit-side.
func BenchmarkStateCommit(b *testing.B) {
	const mutations = 1000

	for _, backend := range benchBackends {
		b.Run(backend.name, func(b *testing.B) {
			db, root := buildBenchState(b, backend.pbt)
			b.ReportAllocs()
			b.ResetTimer()

			cur := root
			for i := 0; i < b.N; i++ {
				b.StopTimer()
				statedb, err := New(cur, db)
				if err != nil {
					b.Fatal(err)
				}
				b.StartTimer()

				for j := 0; j < mutations; j++ {
					addr := benchStateAddr((i*mutations + j) % benchAccounts)
					statedb.SetBalance(addr, uint256.NewInt(uint64(i*mutations+j)+2), tracing.BalanceChangeUnspecified)
				}
				cur, err = statedb.Commit(uint64(i+1), true, true)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
