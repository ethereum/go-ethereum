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

package blobpool

import (
	"context"
	"math/big"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/billy"
	"github.com/holiman/uint256"
)

type txSpec struct {
	blobs int
	tip   uint64
}

type testCache struct {
	*Cache
	clock   *mclock.Simulated
	iterCh  chan struct{}
	vhashes [][]common.Hash // vhashes in the pool
	offset  int             // next blob index to use when injecting more txs
}

// newTestCache creates a cache for test, with a pool that contains transactions
// specified in txConfig. The returned cache has the initial topK fetch already
// settled.
func newTestCache(t *testing.T, txConfig []txSpec) *testCache {
	storage := t.TempDir()
	if err := os.MkdirAll(filepath.Join(storage, pendingTransactionStore), 0700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	store, err := billy.Open(billy.Options{Path: filepath.Join(storage, pendingTransactionStore)}, newSlotterEIP7594(params.BlobTxMaxBlobs), nil)
	if err != nil {
		t.Fatalf("billy open: %v", err)
	}

	var (
		addrs   = make([]common.Address, 0, len(txConfig))
		vhashes = make([][]common.Hash, 0, len(txConfig))
		offset  int
	)
	for _, s := range txConfig {
		key, _ := crypto.GenerateKey()
		tx := makeMultiBlobTx(0, s.tip, 1_000_000, 1_000_000, s.blobs, offset, key)
		if _, err := store.Put(encodeForPool(tx)); err != nil {
			t.Fatalf("store put: %v", err)
		}
		addrs = append(addrs, crypto.PubkeyToAddress(key.PublicKey))
		vhashes = append(vhashes, tx.BlobHashes())
		offset += s.blobs
	}
	store.Close()

	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	for _, a := range addrs {
		statedb.AddBalance(a, uint256.NewInt(1_000_000_000_000), tracing.BalanceChangeUnspecified)
	}
	statedb.Commit(0, true, false)

	cancunTime := uint64(0)
	config := &params.ChainConfig{
		ChainID:     big.NewInt(1),
		LondonBlock: big.NewInt(0),
		BerlinBlock: big.NewInt(0),
		CancunTime:  &cancunTime,
		OsakaTime:   &cancunTime,
		BlobScheduleConfig: &params.BlobScheduleConfig{
			Osaka: &params.BlobConfig{
				Target:         1,
				Max:            1,
				UpdateFraction: params.DefaultCancunBlobConfig.UpdateFraction,
			},
		},
	}
	chain := &testBlockChain{
		config:  config,
		basefee: uint256.NewInt(1),
		blobfee: uint256.NewInt(1),
		statedb: statedb,
	}
	pool := New(Config{Datadir: storage}, chain, nil)
	if err := pool.Init(1, chain.CurrentBlock(), newReserver()); err != nil {
		t.Fatalf("init pool: %v", err)
	}
	t.Cleanup(func() { pool.Close() })

	clock := &mclock.Simulated{}
	iterCh := make(chan struct{}, 256)
	step := func() {
		select {
		case iterCh <- struct{}{}:
		default:
		}
	}
	cache := newCache(pool, clock, step)

	tc := &testCache{
		Cache:   cache,
		clock:   clock,
		iterCh:  iterCh,
		vhashes: vhashes,
		offset:  offset,
	}
	// The loop performs the initial topK update immediately on startup and then
	// arms the topK timer. Wait for the timer so we know the initial update has
	// been issued, then let it settle.
	clock.WaitForTimers(1)
	tc.wait(t, 0)
	return tc
}

// inject adds a tx with the given spec directly to the pool's index and store,
// bypassing the normal Add path. Returns the tx's blob versioned hashes.
func (tc *testCache) inject(t *testing.T, spec txSpec) []common.Hash {
	t.Helper()
	key, _ := crypto.GenerateKey()
	tx := makeMultiBlobTx(0, spec.tip, 1_000_000, 1_000_000, spec.blobs, tc.offset, key)
	tc.offset += spec.blobs

	ptx := newBlobTxForPool(tx)

	tc.blobpool.lock.Lock()
	defer tc.blobpool.lock.Unlock()

	id, err := tc.blobpool.store.Put(encodeForPool(tx))
	if err != nil {
		t.Fatalf("store put: %v", err)
	}
	meta := newBlobTxMeta(id, ptx.TxSize(), tc.blobpool.store.Size(id), ptx)
	addr := crypto.PubkeyToAddress(key.PublicKey)
	tc.blobpool.index[addr] = append(tc.blobpool.index[addr], meta)
	tc.blobpool.lookup.track(meta)

	return tx.BlobHashes()
}

// wait advances simulated time by d (if > 0) and then blocks until the cache
// loop and any inflight fetch goroutines have settled.
func (tc *testCache) wait(t *testing.T, d time.Duration) {
	t.Helper()
	if d > 0 {
		tc.clock.Run(d)
	}
	for {
		select {
		case <-tc.iterCh:
			tc.inflight.Wait()
		case <-time.After(50 * time.Millisecond):
			tc.inflight.Wait()
			return
		}
	}
}

func (tc *testCache) expectEntries(t *testing.T, want ...common.Hash) {
	t.Helper()
	wantSet := make(map[common.Hash]struct{}, len(want))
	for _, w := range want {
		wantSet[w] = struct{}{}
	}
	tc.mu.Lock()
	have := make(map[common.Hash]struct{}, len(tc.entries))
	for k := range tc.entries {
		have[k] = struct{}{}
	}
	tc.mu.Unlock()
	if !reflect.DeepEqual(have, wantSet) {
		t.Errorf("entries: got %s, want %s", hashSet(have), hashSet(wantSet))
	}
}

func hashSet(m map[common.Hash]struct{}) []string {
	out := make([]string, 0, len(m))
	for h := range m {
		out = append(out, h.Hex()[:10])
	}
	sort.Strings(out)
	return out
}

// TestCacheHasBlobsLoadsClaimedSet checks that a HasBlobs request loads
// exactly the txs whose vhashes the cache claimed available, regardless of
// whether the claim came from the cache itself or from the pool fallback.
func TestCacheHasBlobsLoadsClaimedSet(t *testing.T) {
	tc := newTestCache(t, []txSpec{
		{blobs: 2, tip: 100},
		{blobs: 2, tip: 200},
		{blobs: 2, tip: 300},
	})

	available := tc.HasBlobs(context.Background(), tc.vhashes[1])
	if !available[0] {
		t.Fatalf("expected vhash to be reported available")
	}
	tc.wait(t, 0)

	tc.expectEntries(t, tc.vhashes[1]...)
}

// TestCacheTopK exercises the initial topK update: after it settles in
// newTestCache, the cache entries equal the top-by-tip txs.
func TestCacheTopK(t *testing.T) {
	tc := newTestCache(t, []txSpec{
		{blobs: 1, tip: 100},
		{blobs: 1, tip: 200},
		{blobs: 1, tip: 300},
	})

	tc.expectEntries(t, tc.vhashes[2]...)
}

// TestCacheHbTimerFallsBackToTopK checks the fallback after a HasBlobs
// request: when hasBlobsTimeout elapses, a topK update replaces the entries
// with the topK set.
func TestCacheHbTimerFallsBackToTopK(t *testing.T) {
	tc := newTestCache(t, []txSpec{
		{blobs: 1, tip: 100},
		{blobs: 1, tip: 300},
	})

	tc.HasBlobs(context.Background(), tc.vhashes[0])
	tc.wait(t, 0)
	tc.expectEntries(t, tc.vhashes[0]...)

	tc.wait(t, hasBlobsTimeout)
	tc.expectEntries(t, tc.vhashes[1]...)
}

// TestCacheGetBlobs checks that GetBlobs returns the requested blobs and does
// not disturb the cached entries.
func TestCacheGetBlobs(t *testing.T) {
	tc := newTestCache(t, []txSpec{
		{blobs: 1, tip: 100},
		{blobs: 1, tip: 300},
	})
	tc.expectEntries(t, tc.vhashes[1]...)

	blobs, _, proofs, err := tc.GetBlobs(context.Background(), tc.vhashes[1], types.BlobSidecarVersion1)
	if err != nil {
		t.Fatalf("GetBlobs: %v", err)
	}
	for i := range blobs {
		if blobs[i] == nil {
			t.Errorf("blob %d missing in GetBlobs response", i)
		}
		if len(proofs[i]) == 0 {
			t.Errorf("proofs %d missing in GetBlobs response", i)
		}
	}
	tc.wait(t, 0)
	tc.expectEntries(t, tc.vhashes[1]...)
}

func TestCacheGetBlobsFallsBackOnVersionMismatch(t *testing.T) {
	tc := newTestCache(t, []txSpec{
		{blobs: 1, tip: 100},
	})
	vhash := tc.vhashes[0][0]
	tc.expectEntries(t, vhash)

	tc.mu.Lock()
	tc.entries[vhash].version = types.BlobSidecarVersion0
	tc.mu.Unlock()

	blobs, _, proofs, err := tc.GetBlobs(context.Background(), []common.Hash{vhash}, types.BlobSidecarVersion1)
	if err != nil {
		t.Fatalf("GetBlobs: %v", err)
	}
	if blobs[0] == nil {
		t.Fatal("blob missing in GetBlobs response")
	}
	if len(proofs[0]) == 0 {
		t.Fatal("proofs missing in GetBlobs response")
	}
}

// TestCacheTopKRefresh verifies that when a more profitable tx appears in the
// pool, the next topK tick replaces the cached entry with the better one.
func TestCacheTopKRefresh(t *testing.T) {
	tc := newTestCache(t, []txSpec{
		{blobs: 1, tip: 100},
		{blobs: 1, tip: 200},
		{blobs: 1, tip: 300},
	})
	tc.expectEntries(t, tc.vhashes[2]...)

	better := tc.inject(t, txSpec{blobs: 1, tip: 400})

	tc.wait(t, topKTimeout)
	tc.expectEntries(t, better...)
}
