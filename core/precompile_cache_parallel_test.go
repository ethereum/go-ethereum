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
	"context"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/trie"
)

// TestParallelPrecompileCache asserts two properties of the BAL-driven parallel
// processor once the shared precompile result cache is attached:
//
//   - It actually uses the cache. A cold run records misses (the cache is
//     consulted), a warm re-run of the same block records only hits and no
//     further misses (every precompile call is served from the cache). Both are
//     zero when the cache is not wired into the parallel EVMs, which is the bug
//     this guards against.
//
//   - Caching is transparent. The block-level output — state root, access list
//     hash, gas and receipts — is byte-for-byte identical on the cold (miss)
//     and warm (hit) runs, and both reproduce the committed block. A hit that
//     returned a wrong output, or skipped gas accounting or the access-list
//     touch, would diverge here.
func TestParallelPrecompileCache(t *testing.T) {
	// The number of ecrecover calls driven through the block, one per transaction.
	const precompileCalls = 4

	env := newBALTestEnv(nil)

	// Build a valid ecrecover (0x01) input; ecrecover is a cacheable precompile.
	msgHash := crypto.Keccak256Hash([]byte("parallel precompile cache test"))
	sig, err := crypto.Sign(msgHash.Bytes(), env.key)
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	input := make([]byte, 128)
	copy(input[:32], msgHash.Bytes())
	input[63] = sig[64] + 27
	copy(input[64:96], sig[:32])
	copy(input[96:128], sig[32:64])

	ecrecoverAddr := common.BytesToAddress([]byte{1})
	engine := beacon.New(ethash.NewFaker())
	_, blocks, _ := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, b *BlockGen) {
		for i := uint64(0); i < precompileCalls; i++ {
			b.AddTx(env.tx(i, &ecrecoverAddr, common.Big0, 100_000, 1, input))
		}
	})
	block := blocks[0]
	if block.AccessList() == nil {
		t.Fatal("expected non-nil block access list")
	}
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	// A cache is transparent: a hit and a miss return the same bytes, so the only
	// evidence that the parallel EVMs consult it at all is the main-pass hit/miss
	// meters. Mark is unconditional, so the counts move even with metrics
	// collection disabled. This test does not run in parallel with others in the
	// package, so nothing else marks these meters while it measures a delta.
	var (
		cache = vm.NewPrecompileCache()
		hits  = metrics.GetOrRegisterMeter("chain/cache/precompile/hit", nil)
		miss  = metrics.GetOrRegisterMeter("chain/cache/precompile/miss", nil)
	)
	process := func(cfg vm.Config) (*ProcessResult, common.Hash) {
		statedb, err := bc.State()
		if err != nil {
			t.Fatalf("state: %v", err)
		}
		res, err := NewStateProcessor(bc).Process(context.Background(), block, statedb, nil, cache, cfg, nil)
		if err != nil {
			t.Fatalf("process: %v", err)
		}
		return res, statedb.IntermediateRoot(env.cfg.IsEIP158(block.Number()))
	}
	// Cold parallel run: the cache is empty, so the parallel EVMs must consult it
	// and record misses. Zero misses means the cache was never wired in.
	missBefore := miss.Snapshot().Count()
	coldRes, coldRoot := process(vm.Config{})
	if coldMiss := miss.Snapshot().Count() - missBefore; coldMiss == 0 {
		t.Fatal("parallel execution never consulted the precompile cache")
	}
	// Warm parallel run: the same block re-executed. Every ecrecover call must be
	// served from the cache — all hits, no new misses.
	hitsBefore, missBefore := hits.Snapshot().Count(), miss.Snapshot().Count()
	warmRes, warmRoot := process(vm.Config{})
	if warmHits := hits.Snapshot().Count() - hitsBefore; warmHits < precompileCalls {
		t.Fatalf("warm parallel run served %d calls from the cache, want at least %d", warmHits, precompileCalls)
	}
	if warmMiss := miss.Snapshot().Count() - missBefore; warmMiss != 0 {
		t.Fatalf("warm parallel run recomputed %d cached precompile call(s), want 0", warmMiss)
	}
	// Transparency: cold (miss path) and warm (hit path) must agree on every
	// block-level output, and both must reproduce the committed block.
	assertResultsEqual(t, "cold vs warm parallel", coldRes, warmRes)
	if coldRoot != block.Root() || warmRoot != block.Root() {
		t.Fatalf("parallel state root: cold %x, warm %x, committed %x", coldRoot, warmRoot, block.Root())
	}
	// The warm cache built by the parallel path must also serve the sequential
	// path without diverging: this is the miner-warms / import-consumes shape,
	// where the two passes run different execution models over one cache.
	seqRes, seqRoot := process(vm.Config{DisableParallelExecution: true})
	assertResultsEqual(t, "parallel vs sequential over the warm cache", warmRes, seqRes)
	if seqRoot != block.Root() {
		t.Fatalf("warm sequential state root %x != committed %x", seqRoot, block.Root())
	}
}

// assertResultsEqual fails the test unless the two process results agree on
// every consensus-visible block output.
func assertResultsEqual(t *testing.T, ctx string, a, b *ProcessResult) {
	t.Helper()
	if a.GasUsed != b.GasUsed {
		t.Fatalf("%s: gas used %d != %d", ctx, a.GasUsed, b.GasUsed)
	}
	if ah, bh := a.Bal.ToEncodingObj().Hash(), b.Bal.ToEncodingObj().Hash(); ah != bh {
		t.Fatalf("%s: access list hash %x != %x", ctx, ah, bh)
	}
	if ah, bh := types.DeriveSha(a.Receipts, trie.NewStackTrie(nil)), types.DeriveSha(b.Receipts, trie.NewStackTrie(nil)); ah != bh {
		t.Fatalf("%s: receipt root %x != %x", ctx, ah, bh)
	}
	if ah, bh := types.CalcRequestsHash(a.Requests), types.CalcRequestsHash(b.Requests); ah != bh {
		t.Fatalf("%s: requests hash %x != %x", ctx, ah, bh)
	}
}

// BenchmarkParallelProcessPrecompiles measures parallel block execution of a
// MODEXP-heavy block without a precompile cache and with a warmed one, the
// state a block producer is in when its own payload returns for import.
func BenchmarkParallelProcessPrecompiles(b *testing.B) {
	env := newBALTestEnv(nil)
	env.gspec.GasLimit = 100_000_000

	// 2048-bit MODEXP with a full exponent, a distinct base per transaction.
	modexpInput := func(i byte) []byte {
		input := make([]byte, 96+3*256)
		input[30], input[62], input[94] = 1, 1, 1 // base_len = exp_len = mod_len = 256
		base := input[96 : 96+256]
		exp := input[96+256 : 96+512]
		mod := input[96+512:]
		base[0] = i + 1
		for j := range exp {
			exp[j] = 0xff
		}
		for j := range mod {
			mod[j] = 0xef
		}
		return input
	}
	modexpAddr := common.BytesToAddress([]byte{5})
	engine := beacon.New(ethash.NewFaker())
	_, blocks, receipts := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, g *BlockGen) {
		for i := uint64(0); i < 8; i++ {
			g.AddTx(env.tx(i, &modexpAddr, common.Big0, 9_000_000, 1, modexpInput(byte(i))))
		}
	})
	for _, r := range receipts[0] {
		if r.Status != types.ReceiptStatusSuccessful {
			b.Fatal("modexp transaction failed")
		}
	}
	block := blocks[0]
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
	if err != nil {
		b.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	processOnce := func(tb testing.TB, cache *vm.PrecompileCache, cfg vm.Config) {
		statedb, err := bc.State()
		if err != nil {
			tb.Fatalf("state: %v", err)
		}
		if _, err := NewStateProcessor(bc).Process(context.Background(), block, statedb, nil, cache, cfg, nil); err != nil {
			tb.Fatalf("process: %v", err)
		}
	}
	var (
		parallel   = vm.Config{}
		sequential = vm.Config{DisableParallelExecution: true}
	)
	uncached := func(cfg vm.Config) func(*testing.B) {
		return func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				processOnce(b, nil, cfg)
			}
		}
	}
	warm := func(cfg vm.Config) func(*testing.B) {
		return func(b *testing.B) {
			cache := vm.NewPrecompileCache()
			processOnce(b, cache, cfg) // warm outside the timer
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				processOnce(b, cache, cfg)
			}
		}
	}
	b.Run("parallel/uncached", uncached(parallel))
	b.Run("parallel/warm", warm(parallel))
	b.Run("sequential/uncached", uncached(sequential))
	b.Run("sequential/warm", warm(sequential))
}
