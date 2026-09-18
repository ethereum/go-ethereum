// Copyright 2025 The go-ethereum Authors
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
	"os"
	"path/filepath"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/holiman/billy"
	"github.com/holiman/uint256"
)

// TestContentFrom verifies that ContentFrom returns pending blob transactions
// for a given address with blob sidecars stripped.
func TestContentFrom(t *testing.T) {
	storage := t.TempDir()
	os.MkdirAll(filepath.Join(storage, pendingTransactionStore), 0700)
	store, _ := billy.Open(billy.Options{Path: filepath.Join(storage, pendingTransactionStore)}, newSlotter(testMaxBlobsPerBlock), nil)

	// Create two accounts: one with transactions, one without
	key1, _ := crypto.GenerateKey()
	addr1 := crypto.PubkeyToAddress(key1.PublicKey)
	key2, _ := crypto.GenerateKey()
	addr2 := crypto.PubkeyToAddress(key2.PublicKey)

	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	statedb.AddBalance(addr1, new(uint256.Int).SetUint64(10_000_000), tracing.BalanceChangeUnspecified)
	statedb.AddBalance(addr2, new(uint256.Int).SetUint64(10_000_000), tracing.BalanceChangeUnspecified)

	// Seed two contiguous transactions for addr1
	tx0 := types.MustSignNewTx(key1, types.LatestSigner(params.MainnetChainConfig), makeUnsignedTxWithTestBlob(0, 1, 1, 1, 0))
	tx1 := types.MustSignNewTx(key1, types.LatestSigner(params.MainnetChainConfig), makeUnsignedTxWithTestBlob(1, 1, 1, 1, 1))

	blob0, _ := rlp.EncodeToBytes(tx0)
	blob1, _ := rlp.EncodeToBytes(tx1)
	store.Put(blob0)
	store.Put(blob1)

	statedb.Commit(0, true, false)
	store.Close()

	chain := &testBlockChain{
		config:  params.MainnetChainConfig,
		basefee: uint256.NewInt(1),
		blobfee: uint256.NewInt(1),
		statedb: statedb,
	}
	pool := New(Config{Datadir: storage}, chain, nil)
	if err := pool.Init(1, chain.CurrentBlock(), newReserver()); err != nil {
		t.Fatalf("failed to create blob pool: %v", err)
	}
	defer pool.Close()

	// Verify both transactions are indexed
	if !pool.Has(tx0.Hash()) || !pool.Has(tx1.Hash()) {
		t.Fatal("seeded transactions not found in pool")
	}

	// ContentFrom for addr1 should return both pending transactions
	pending, queued := pool.ContentFrom(addr1)
	if len(pending) != 2 {
		t.Fatalf("expected 2 pending transactions, got %d", len(pending))
	}
	if len(queued) != 0 {
		t.Fatalf("expected 0 queued transactions, got %d", len(queued))
	}

	// Verify nonce ordering
	if pending[0].Nonce() != 0 || pending[1].Nonce() != 1 {
		t.Fatalf("unexpected nonces: got %d, %d; want 0, 1", pending[0].Nonce(), pending[1].Nonce())
	}

	// Verify blob sidecars are stripped
	for i, tx := range pending {
		if tx.BlobTxSidecar() != nil {
			t.Errorf("pending tx %d: blob sidecar should be stripped", i)
		}
	}

	// Verify blob versioned hashes are preserved
	for i, tx := range pending {
		if len(tx.BlobHashes()) == 0 {
			t.Errorf("pending tx %d: blob versioned hashes should be preserved", i)
		}
	}

	// ContentFrom for addr2 (no transactions) should return empty
	pending2, queued2 := pool.ContentFrom(addr2)
	if len(pending2) != 0 || len(queued2) != 0 {
		t.Fatalf("expected empty results for addr2, got pending=%d queued=%d", len(pending2), len(queued2))
	}

	// ContentFrom for unknown address should return empty
	pending3, queued3 := pool.ContentFrom(common.Address{0x99})
	if len(pending3) != 0 || len(queued3) != 0 {
		t.Fatalf("expected empty results for unknown address, got pending=%d queued=%d", len(pending3), len(queued3))
	}
}

// TestContentFromGapped verifies that ContentFrom returns gapped (queued)
// transactions separately from pending ones.
func TestContentFromGapped(t *testing.T) {
	storage := t.TempDir()
	os.MkdirAll(filepath.Join(storage, pendingTransactionStore), 0700)
	store, _ := billy.Open(billy.Options{Path: filepath.Join(storage, pendingTransactionStore)}, newSlotter(testMaxBlobsPerBlock), nil)

	key, _ := crypto.GenerateKey()
	addr := crypto.PubkeyToAddress(key.PublicKey)

	statedb, _ := state.New(types.EmptyRootHash, state.NewDatabaseForTesting())
	statedb.AddBalance(addr, new(uint256.Int).SetUint64(10_000_000), tracing.BalanceChangeUnspecified)
	// Set nonce to 10 so gapped allowance permits queued txs (log10(10+1)=1)
	statedb.SetNonce(addr, 10, tracing.NonceChangeUnspecified)

	// Seed a pending transaction at nonce 10
	tx0 := types.MustSignNewTx(key, types.LatestSigner(params.MainnetChainConfig), makeUnsignedTxWithTestBlob(10, 1, 1, 1, 0))
	blob0, _ := rlp.EncodeToBytes(tx0)
	store.Put(blob0)

	statedb.Commit(0, true, false)
	store.Close()

	chain := &testBlockChain{
		config:  params.MainnetChainConfig,
		basefee: uint256.NewInt(1),
		blobfee: uint256.NewInt(1),
		statedb: statedb,
	}
	pool := New(Config{Datadir: storage}, chain, nil)
	if err := pool.Init(1, chain.CurrentBlock(), newReserver()); err != nil {
		t.Fatalf("failed to create blob pool: %v", err)
	}
	defer pool.Close()

	// Add a gapped transaction at nonce 12 (skipping nonce 11)
	gappedTx := types.MustSignNewTx(key, types.LatestSigner(params.MainnetChainConfig), makeUnsignedTxWithTestBlob(12, 1, 1, 1, 2))
	pool.Add([]*types.Transaction{gappedTx}, true)

	// Verify the gapped tx is queued
	if pool.Status(gappedTx.Hash()) != 1 { // TxStatusQueued
		t.Fatalf("gapped transaction should be queued, got status %d", pool.Status(gappedTx.Hash()))
	}

	// ContentFrom should return pending and queued separately
	pending, queued := pool.ContentFrom(addr)
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending transaction, got %d", len(pending))
	}
	if len(queued) != 1 {
		t.Fatalf("expected 1 queued transaction, got %d", len(queued))
	}

	// Verify nonces
	if pending[0].Nonce() != 10 {
		t.Fatalf("pending tx nonce: got %d, want 10", pending[0].Nonce())
	}
	if queued[0].Nonce() != 12 {
		t.Fatalf("queued tx nonce: got %d, want 12", queued[0].Nonce())
	}

	// Verify blob sidecars are stripped on both
	if pending[0].BlobTxSidecar() != nil {
		t.Error("pending tx: blob sidecar should be stripped")
	}
	if queued[0].BlobTxSidecar() != nil {
		t.Error("queued tx: blob sidecar should be stripped")
	}
}
