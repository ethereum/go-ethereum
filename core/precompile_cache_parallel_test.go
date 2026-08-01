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
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
)

// processBlock executes the block once against the chain's genesis state with
// the given cache and config, and returns the result alongside the post-state.
// The state root is computed here so that timed callers measure the same work
// on the parallel path (which hashes state inside Process) and the sequential
// one (which defers hashing to validation).
func processBlock(tb testing.TB, bc *BlockChain, block *types.Block, cache *vm.PrecompileCache, cfg vm.Config) (*ProcessResult, *state.StateDB, common.Hash) {
	tb.Helper()
	statedb, err := bc.State()
	if err != nil {
		tb.Fatalf("state: %v", err)
	}
	res, err := NewStateProcessor(bc).Process(context.Background(), block, statedb, nil, cache, cfg, nil)
	if err != nil {
		tb.Fatalf("process: %v", err)
	}
	return res, statedb, statedb.IntermediateRoot(bc.chainConfig.IsEIP158(block.Number()))
}

// TestParallelPrecompileCache asserts three properties of the BAL-driven
// parallel processor once the shared precompile result cache is attached:
//
//   - It actually uses the cache. A cold run records misses (the cache is
//     consulted), a warm re-run of the same block records only hits and no
//     further misses. Both are zero when the cache is not wired into the
//     parallel EVMs, which is the bug this guards against.
//
//   - Hits return the right bytes. Every transaction routes a distinct
//     ecrecover call through a contract that stores the returned word. On the
//     parallel path the canonical post-state is replayed from the committed
//     access list, so re-execution outputs surface in the receipts and the
//     rebuilt access list instead: the test pins the stored words there, and
//     in the state for the sequential run. A hit serving wrong or stale bytes
//     fails those checks (and shifts gas for the zero-word case).
//
//   - Caching is otherwise transparent. The cold run is anchored field by
//     field to the committed block (built without any cache), and the warm
//     parallel and sequential runs must reproduce it exactly.
func TestParallelPrecompileCache(t *testing.T) {
	// The number of ecrecover calls driven through the block, one distinctly
	// keyed call per transaction.
	const precompileCalls = 4

	// A contract that forwards its calldata to ecrecover (0x01) and stores the
	// returned word at the slot named by the first calldata word (the signed
	// hash), making the precompile output part of the state root:
	//
	// CALLDATASIZE PUSH1 0 PUSH1 0 CALLDATACOPY
	// PUSH1 32 (retSize) PUSH1 128 (retOff) CALLDATASIZE (argsSize) PUSH1 0 (argsOff)
	// PUSH1 1 GAS STATICCALL POP
	// PUSH1 128 MLOAD PUSH1 0 CALLDATALOAD SSTORE STOP
	caller := common.HexToAddress("0xc0de")
	callerCode := []byte{
		0x36, 0x60, 0x00, 0x60, 0x00, 0x37,
		0x60, 0x20, 0x60, 0x80, 0x36, 0x60, 0x00,
		0x60, 0x01, 0x5a, 0xfa, 0x50,
		0x60, 0x80, 0x51, 0x60, 0x00, 0x35, 0x55, 0x00,
	}
	env := newBALTestEnv(types.GenesisAlloc{caller: {Code: callerCode, Balance: common.Big0}})

	// Sign a distinct message per transaction so the block exercises one cache
	// key per call; ecrecover is a cacheable precompile.
	var (
		inputs    = make([][]byte, precompileCalls)
		msgHashes = make([]common.Hash, precompileCalls)
	)
	for i := range inputs {
		msgHash := crypto.Keccak256Hash([]byte{byte(i)}, []byte("parallel precompile cache test"))
		sig, err := crypto.Sign(msgHash.Bytes(), env.key)
		if err != nil {
			t.Fatalf("sign: %v", err)
		}
		input := make([]byte, 128)
		copy(input[:32], msgHash.Bytes())
		input[63] = sig[64] + 27
		copy(input[64:96], sig[:32])
		copy(input[96:128], sig[32:64])
		inputs[i], msgHashes[i] = input, msgHash
	}
	engine := beacon.New(ethash.NewFaker())
	_, blocks, receipts := GenerateChainWithGenesis(env.gspec, engine, 1, func(_ int, b *BlockGen) {
		for i := uint64(0); i < precompileCalls; i++ {
			b.AddTx(env.tx(i, &caller, common.Big0, 200_000, 1, inputs[i]))
		}
	})
	for _, r := range receipts[0] {
		if r.Status != types.ReceiptStatusSuccessful {
			t.Fatal("ecrecover transaction failed")
		}
	}
	block := blocks[0]
	if !supportsParallelExecution(block, env.cfg, false, false, false) {
		t.Fatal("generated block is not eligible for parallel execution")
	}
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
	if err != nil {
		t.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	// checkBALOutputs asserts that the rebuilt access list records the signer's
	// address as the post-value of every result slot. On the parallel path the
	// rebuilt access list is where re-execution outputs land — the canonical
	// post-state is replayed from the committed one — so this pins the bytes a
	// cache hit returns, not just the cache traffic.
	checkBALOutputs := func(desc string, res *ProcessResult) {
		t.Helper()
		list := res.Bal.ToEncodingObj()
		aa := findAccount(list, caller)
		if aa == nil {
			t.Fatalf("%s: caller missing from rebuilt access list", desc)
		}
		want := new(uint256.Int).SetBytes(env.from.Bytes())
		for _, h := range msgHashes {
			var (
				slot  = new(uint256.Int).SetBytes(h[:])
				found = false
			)
			for i := range aa.StorageChanges {
				sc := &aa.StorageChanges[i]
				if sc.Slot.Cmp(slot) != 0 {
					continue
				}
				found = true
				if len(sc.SlotChanges) != 1 || sc.SlotChanges[0].PostValue.Cmp(want) != 0 {
					t.Fatalf("%s: stored ecrecover output for slot %x = %v, want %v", desc, h, sc.SlotChanges, want)
				}
			}
			if !found {
				t.Fatalf("%s: result slot %x missing from rebuilt access list", desc, h)
			}
		}
	}
	// checkStateOutputs asserts the same on a post-state whose storage derives
	// from execution directly, which holds only for the sequential run.
	checkStateOutputs := func(desc string, statedb *state.StateDB) {
		t.Helper()
		want := common.BytesToHash(env.from.Bytes())
		for _, h := range msgHashes {
			if got := statedb.GetState(caller, h); got != want {
				t.Fatalf("%s: stored ecrecover output for %x = %x, want %x", desc, h, got, want)
			}
		}
	}
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
	// Cold parallel run: the cache is empty, so the parallel EVMs must consult it
	// and record misses. Zero misses means the cache was never wired in. The
	// result is anchored to the committed block, which was built with no cache.
	missBefore := miss.Snapshot().Count()
	coldRes, _, coldRoot := processBlock(t, bc, block, cache, vm.Config{})
	if coldMiss := miss.Snapshot().Count() - missBefore; coldMiss == 0 {
		t.Fatal("parallel execution never consulted the precompile cache")
	}
	checkBALOutputs("cold parallel", coldRes)
	assertMatchesBlock(t, "cold parallel", coldRes, coldRoot, block)

	// Warm parallel run: the same block re-executed. Every ecrecover call must
	// be served from the cache — all hits, no new misses — and the stored
	// outputs must still be correct.
	hitsBefore, missBefore := hits.Snapshot().Count(), miss.Snapshot().Count()
	warmRes, _, warmRoot := processBlock(t, bc, block, cache, vm.Config{})
	if warmHits := hits.Snapshot().Count() - hitsBefore; warmHits < precompileCalls {
		t.Fatalf("warm parallel run served %d calls from the cache, want at least %d", warmHits, precompileCalls)
	}
	if warmMiss := miss.Snapshot().Count() - missBefore; warmMiss != 0 {
		t.Fatalf("warm parallel run recomputed %d cached precompile call(s), want 0", warmMiss)
	}
	checkBALOutputs("warm parallel", warmRes)
	assertResultsEqual(t, "cold vs warm parallel", coldRes, warmRes)
	if warmRoot != block.Root() {
		t.Fatalf("warm parallel state root %x != committed %x", warmRoot, block.Root())
	}
	// The warm cache built by the parallel path must also serve the sequential
	// path without diverging: this is the miner-warms / import-consumes shape,
	// where the two passes run different execution models over one cache.
	seqRes, seqState, seqRoot := processBlock(t, bc, block, cache, vm.Config{DisableParallelExecution: true})
	checkStateOutputs("warm sequential", seqState)
	assertResultsEqual(t, "parallel vs sequential over the warm cache", warmRes, seqRes)
	if seqRoot != block.Root() {
		t.Fatalf("warm sequential state root %x != committed %x", seqRoot, block.Root())
	}
}

// assertMatchesBlock fails the test unless the process result reproduces every
// consensus-visible output of the committed block.
func assertMatchesBlock(t *testing.T, desc string, res *ProcessResult, root common.Hash, block *types.Block) {
	t.Helper()
	if root != block.Root() {
		t.Fatalf("%s: state root %x != committed %x", desc, root, block.Root())
	}
	if res.GasUsed != block.GasUsed() {
		t.Fatalf("%s: gas used %d != committed %d", desc, res.GasUsed, block.GasUsed())
	}
	if got := types.DeriveSha(res.Receipts, trie.NewStackTrie(nil)); got != block.ReceiptHash() {
		t.Fatalf("%s: receipt root %x != committed %x", desc, got, block.ReceiptHash())
	}
	if res.Bal == nil {
		t.Fatalf("%s: missing rebuilt access list", desc)
	}
	if got := res.Bal.ToEncodingObj().Hash(); got != *block.BlockAccessListHash() {
		t.Fatalf("%s: access list hash %x != committed %x", desc, got, *block.BlockAccessListHash())
	}
	if res.Requests == nil {
		t.Fatalf("%s: missing requests", desc)
	}
	if got := types.CalcRequestsHash(res.Requests); got != *block.RequestsHash() {
		t.Fatalf("%s: requests hash %x != committed %x", desc, got, *block.RequestsHash())
	}
}

// assertResultsEqual fails the test unless the two process results agree on
// every consensus-visible block output.
func assertResultsEqual(t *testing.T, desc string, a, b *ProcessResult) {
	t.Helper()
	if a.GasUsed != b.GasUsed {
		t.Fatalf("%s: gas used %d != %d", desc, a.GasUsed, b.GasUsed)
	}
	if (a.Bal == nil) != (b.Bal == nil) {
		t.Fatalf("%s: access list nil-ness mismatch", desc)
	}
	if a.Bal != nil {
		if ah, bh := a.Bal.ToEncodingObj().Hash(), b.Bal.ToEncodingObj().Hash(); ah != bh {
			t.Fatalf("%s: access list hash %x != %x", desc, ah, bh)
		}
	}
	if ah, bh := types.DeriveSha(a.Receipts, trie.NewStackTrie(nil)), types.DeriveSha(b.Receipts, trie.NewStackTrie(nil)); ah != bh {
		t.Fatalf("%s: receipt root %x != %x", desc, ah, bh)
	}
	if (a.Requests == nil) != (b.Requests == nil) {
		t.Fatalf("%s: requests nil-ness mismatch", desc)
	}
	if a.Requests != nil {
		if ah, bh := types.CalcRequestsHash(a.Requests), types.CalcRequestsHash(b.Requests); ah != bh {
			t.Fatalf("%s: requests hash %x != %x", desc, ah, bh)
		}
	}
}

// BenchmarkParallelProcessPrecompiles measures block execution of a
// MODEXP-heavy block across cache states. parallel/cold is the shape of a
// non-producing node importing a block first-hand (cache attached, all
// misses); parallel/warm is a block producer re-importing its own payload
// (all hits). The uncached arms pass no cache at all, a configuration real
// nodes no longer run, as a floor. processBlock hashes the post-state in
// every arm so the parallel and sequential numbers are comparable.
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
	if !supportsParallelExecution(block, env.cfg, false, false, false) {
		b.Fatal("generated block is not eligible for parallel execution")
	}
	bc, err := NewBlockChain(rawdb.NewMemoryDatabase(), env.gspec, engine, nil)
	if err != nil {
		b.Fatalf("new blockchain: %v", err)
	}
	defer bc.Stop()

	var (
		parallel   = vm.Config{}
		sequential = vm.Config{DisableParallelExecution: true}
	)
	uncached := func(cfg vm.Config) func(*testing.B) {
		return func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				processBlock(b, bc, block, nil, cfg)
			}
		}
	}
	cold := func(cfg vm.Config) func(*testing.B) {
		return func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				processBlock(b, bc, block, vm.NewPrecompileCache(), cfg)
			}
		}
	}
	warm := func(cfg vm.Config) func(*testing.B) {
		return func(b *testing.B) {
			cache := vm.NewPrecompileCache()
			processBlock(b, bc, block, cache, cfg) // warm outside the timer
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				processBlock(b, bc, block, cache, cfg)
			}
		}
	}
	b.Run("parallel/uncached", uncached(parallel))
	b.Run("parallel/cold", cold(parallel))
	b.Run("parallel/warm", warm(parallel))
	b.Run("sequential/uncached", uncached(sequential))
	b.Run("sequential/warm", warm(sequential))
}
