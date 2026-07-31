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

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// TestBALParallelBlockHashRace is a regression guard for a data race on the
// BLOCKHASH ancestor-hash cache in the parallel BAL executor.
//
// NewEVMBlockContext installs a GetHash closure returned by GetHashFn, which
// lazily caches ancestor hashes in an unsynchronized slice. processParallel
// builds one such BlockContext and hands it to a pool of worker goroutines; if
// they share the single closure, two transactions resolving BLOCKHASH on
// different workers append to and read that slice concurrently. Each worker
// must therefore be given its own GetHash instance.
//
// The block below fills every worker with transactions that all execute
// BLOCKHASH, so a shared closure trips the detector. The race is only
// observable under `go test -race`; without it the test simply exercises the
// parallel path.
func TestBALParallelBlockHashRace(t *testing.T) {
	// probe runtime code: PUSH1 0x00, BLOCKHASH, POP, STOP. Resolving the parent
	// hash on every call forces a read/append against the shared GetHashFn cache.
	probe := common.HexToAddress("0xb1a5")
	code := []byte{0x60, 0x00, 0x40, 0x50, 0x00}
	env := newBALTestEnv(types.GenesisAlloc{
		probe: {Code: code, Balance: common.Big0},
	})

	// Enough transactions to keep every parallel worker busy, so multiple of
	// them reach the shared closure concurrently.
	const numTxs = 64
	env.run(t, func(g *BlockGen) {
		for i := 0; i < numTxs; i++ {
			g.AddTx(env.tx(uint64(i), &probe, big.NewInt(0), 100_000, 0, nil))
		}
	})
}
