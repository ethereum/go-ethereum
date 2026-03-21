// Copyright 2025 go-ethereum Authors
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
	"math/rand"
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

const (
	benchContracts   = 500                               // number of pre-populated contracts
	benchSlotsPerAcc = 1000                              // storage slots per contract
	benchTotalSlots  = benchContracts * benchSlotsPerAcc // 500K entries

	approveOpsPerBlock   = 2250  // cold SSTOREs per approve block
	approveGasPerOp      = 22100 // gas per cold SSTORE
	balanceOfOpsPerBlock = 24000 // cold SLOADs per balanceOf block
	balanceOfGasPerOp    = 2100  // gas per cold SLOAD
)

// setupPebbleStateDB creates a PebbleDB-backed StateDB pre-populated with
// 500K storage entries (500 contracts x 1000 slots). Returns the state database,
// current root, the pre-populated addresses, and a cleanup function.
func setupPebbleStateDB(b *testing.B) (*CachingDB, common.Hash, []common.Address, func()) {
	b.Helper()

	// Create PebbleDB in temp directory
	pdb, err := pebble.New(b.TempDir(), 128, 128, "", false)
	if err != nil {
		b.Fatalf("pebble.New: %v", err)
	}
	diskDB := rawdb.NewDatabase(pdb)

	tdb := triedb.NewDatabase(diskDB, &triedb.Config{
		IsVerkle: true,
		PathDB: &pathdb.Config{
			TrieCleanSize:   0, // cold reads, no fastcache
			StateCleanSize:  0,
			WriteBufferSize: 64 << 20,
			NoAsyncFlush:    true,
		},
	})
	cachingDB := NewDatabase(tdb, nil)

	// Generate deterministic addresses
	rng := rand.New(rand.NewSource(42))
	addresses := make([]common.Address, benchContracts)
	for i := range addresses {
		binary.BigEndian.PutUint64(addresses[i][12:], uint64(i))
		rng.Read(addresses[i][:12])
	}

	// Pre-populate in batches of 50 contracts to avoid excessive memory use
	root := types.EmptyBinaryHash
	batchSize := 50
	for batch := 0; batch < benchContracts; batch += batchSize {
		end := batch + batchSize
		if end > benchContracts {
			end = benchContracts
		}
		stateDB, err := New(root, cachingDB)
		if err != nil {
			b.Fatalf("New: %v", err)
		}
		for i := batch; i < end; i++ {
			addr := addresses[i]
			stateDB.SetNonce(addr, 1, tracing.NonceChangeUnspecified)
			stateDB.AddBalance(addr, uint256.NewInt(1_000_000), tracing.BalanceChangeUnspecified)
			for slot := 0; slot < benchSlotsPerAcc; slot++ {
				var key common.Hash
				binary.BigEndian.PutUint64(key[24:], uint64(slot+64))
				val := common.Hash{}
				binary.BigEndian.PutUint64(val[24:], uint64(i*benchSlotsPerAcc+slot+1))
				stateDB.SetState(addr, key, val)
			}
		}
		root = stateDB.IntermediateRoot(false)
		if _, err := stateDB.Commit(uint64(batch/batchSize+1), false, true); err != nil {
			b.Fatalf("Commit: %v", err)
		}
		if err := tdb.Commit(root, false); err != nil {
			b.Fatalf("tdb.Commit: %v", err)
		}
	}

	cleanup := func() {
		tdb.Close()
		diskDB.Close()
	}
	return cachingDB, root, addresses, cleanup
}

// BenchmarkBintrieApprove simulates an ERC-20 approve-heavy block:
// 2250 cold SSTOREs to unique addresses (not in pre-populated set).
// Each SetState internally does a GetCommittedState (cold SLOAD).
// Reports Mgas/s based on 2250 x 22100 = 49.725M gas per block.
func BenchmarkBintrieApprove(b *testing.B) {
	cachingDB, root, _, cleanup := setupPebbleStateDB(b)
	defer cleanup()

	totalGas := float64(approveOpsPerBlock) * float64(approveGasPerOp)
	currentRoot := root

	b.Run("full", func(b *testing.B) {
		r := currentRoot
		b.ResetTimer()
		b.ReportAllocs()
		for iter := 0; iter < b.N; iter++ {
			stateDB, err := New(r, cachingDB)
			if err != nil {
				b.Fatalf("New: %v", err)
			}
			rng := rand.New(rand.NewSource(int64(iter)*12345 + 1))
			for j := 0; j < approveOpsPerBlock; j++ {
				var addr common.Address
				rng.Read(addr[:])
				var key common.Hash
				binary.BigEndian.PutUint64(key[24:], uint64(j+64))
				var val common.Hash
				binary.BigEndian.PutUint64(val[24:], uint64(iter*approveOpsPerBlock+j+1))
				stateDB.SetState(addr, key, val)
			}
			r = stateDB.IntermediateRoot(false)
			if _, err := stateDB.Commit(uint64(iter+100), false, true); err != nil {
				b.Fatalf("Commit: %v", err)
			}
		}
		b.ReportMetric(totalGas/1e6/(b.Elapsed().Seconds()/float64(b.N)), "Mgas/s")
	})

	b.Run("state_ops", func(b *testing.B) {
		r := currentRoot
		b.ResetTimer()
		b.ReportAllocs()
		for iter := 0; iter < b.N; iter++ {
			stateDB, err := New(r, cachingDB)
			if err != nil {
				b.Fatalf("New: %v", err)
			}
			rng := rand.New(rand.NewSource(int64(iter)*12345 + 1))
			for j := 0; j < approveOpsPerBlock; j++ {
				var addr common.Address
				rng.Read(addr[:])
				var key common.Hash
				binary.BigEndian.PutUint64(key[24:], uint64(j+64))
				var val common.Hash
				binary.BigEndian.PutUint64(val[24:], uint64(iter*approveOpsPerBlock+j+1))
				stateDB.SetState(addr, key, val)
			}
			b.StopTimer()
			r = stateDB.IntermediateRoot(false)
			stateDB.Commit(uint64(iter+100), false, true)
			b.StartTimer()
		}
	})

	b.Run("intermediate_root", func(b *testing.B) {
		r := currentRoot
		b.ResetTimer()
		b.ReportAllocs()
		for iter := 0; iter < b.N; iter++ {
			b.StopTimer()
			stateDB, err := New(r, cachingDB)
			if err != nil {
				b.Fatalf("New: %v", err)
			}
			rng := rand.New(rand.NewSource(int64(iter)*12345 + 1))
			for j := 0; j < approveOpsPerBlock; j++ {
				var addr common.Address
				rng.Read(addr[:])
				var key common.Hash
				binary.BigEndian.PutUint64(key[24:], uint64(j+64))
				var val common.Hash
				binary.BigEndian.PutUint64(val[24:], uint64(iter*approveOpsPerBlock+j+1))
				stateDB.SetState(addr, key, val)
			}
			b.StartTimer()
			r = stateDB.IntermediateRoot(false)
			b.StopTimer()
			stateDB.Commit(uint64(iter+100), false, true)
			b.StartTimer()
		}
	})

	b.Run("commit", func(b *testing.B) {
		r := currentRoot
		b.ResetTimer()
		b.ReportAllocs()
		for iter := 0; iter < b.N; iter++ {
			stateDB, err := New(r, cachingDB)
			if err != nil {
				b.Fatalf("New: %v", err)
			}
			rng := rand.New(rand.NewSource(int64(iter)*12345 + 1))
			for j := 0; j < approveOpsPerBlock; j++ {
				var addr common.Address
				rng.Read(addr[:])
				var key common.Hash
				binary.BigEndian.PutUint64(key[24:], uint64(j+64))
				var val common.Hash
				binary.BigEndian.PutUint64(val[24:], uint64(iter*approveOpsPerBlock+j+1))
				stateDB.SetState(addr, key, val)
			}
			b.StopTimer()
			r = stateDB.IntermediateRoot(false)
			b.StartTimer()
			if _, err := stateDB.Commit(uint64(iter+100), false, true); err != nil {
				b.Fatalf("Commit: %v", err)
			}
		}
	})
}

// BenchmarkBintrieBalanceOf simulates an ERC-20 balanceOf-heavy block:
// 24000 cold SLOADs to unique addresses (non-existent slots).
// Reports Mgas/s based on 24000 x 2100 = 50.4M gas per block.
func BenchmarkBintrieBalanceOf(b *testing.B) {
	cachingDB, root, _, cleanup := setupPebbleStateDB(b)
	defer cleanup()

	totalGas := float64(balanceOfOpsPerBlock) * float64(balanceOfGasPerOp)
	currentRoot := root

	b.Run("full", func(b *testing.B) {
		r := currentRoot
		b.ResetTimer()
		b.ReportAllocs()
		for iter := 0; iter < b.N; iter++ {
			stateDB, err := New(r, cachingDB)
			if err != nil {
				b.Fatalf("New: %v", err)
			}
			rng := rand.New(rand.NewSource(int64(iter)*54321 + 1))
			for j := 0; j < balanceOfOpsPerBlock; j++ {
				var addr common.Address
				rng.Read(addr[:])
				var key common.Hash
				binary.BigEndian.PutUint64(key[24:], uint64(j+64))
				stateDB.GetState(addr, key)
			}
			r = stateDB.IntermediateRoot(false)
			if _, err := stateDB.Commit(uint64(iter+100), false, true); err != nil {
				b.Fatalf("Commit: %v", err)
			}
		}
		b.ReportMetric(totalGas/1e6/(b.Elapsed().Seconds()/float64(b.N)), "Mgas/s")
	})

	b.Run("state_ops", func(b *testing.B) {
		r := currentRoot
		b.ResetTimer()
		b.ReportAllocs()
		for iter := 0; iter < b.N; iter++ {
			stateDB, err := New(r, cachingDB)
			if err != nil {
				b.Fatalf("New: %v", err)
			}
			rng := rand.New(rand.NewSource(int64(iter)*54321 + 1))
			for j := 0; j < balanceOfOpsPerBlock; j++ {
				var addr common.Address
				rng.Read(addr[:])
				var key common.Hash
				binary.BigEndian.PutUint64(key[24:], uint64(j+64))
				stateDB.GetState(addr, key)
			}
			b.StopTimer()
			r = stateDB.IntermediateRoot(false)
			stateDB.Commit(uint64(iter+100), false, true)
			b.StartTimer()
		}
	})
}
