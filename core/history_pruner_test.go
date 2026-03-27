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
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/history"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/params"
)

// newTestChain generates a test chain of the given length and inserts it into a
// fresh database using InsertReceiptChain so the blocks end up in the freezer.
// Returns the database (still open), the genesis spec, and the generated blocks.
func newTestChain(t *testing.T, length int) (ethdb.Database, *Genesis, []*types.Block) {
	t.Helper()

	gspec := &Genesis{
		Config:  params.TestChainConfig,
		Alloc:   types.GenesisAlloc{common.HexToAddress("0x01"): {Balance: big.NewInt(1e18)}},
		BaseFee: big.NewInt(params.InitialBaseFee),
	}
	engine := beacon.New(ethash.NewFaker())
	_, blocks, receipts := GenerateChainWithGenesis(gspec, engine, length, nil)

	// Insert the chain into a KeepAll database so all blocks land in the freezer.
	db, _ := rawdb.Open(rawdb.NewMemoryDatabase(), rawdb.OpenOptions{})
	chain, err := NewBlockChain(db, gspec, engine, DefaultConfig().WithStateScheme(rawdb.HashScheme))
	if err != nil {
		t.Fatalf("failed to create chain: %v", err)
	}
	if _, err := chain.InsertReceiptChain(blocks, types.EncodeBlockReceiptLists(receipts), uint64(length)); err != nil {
		t.Fatalf("failed to insert receipt chain: %v", err)
	}
	chain.Stop()
	return db, gspec, blocks
}

// reopenChain reopens a BlockChain on the given database with the given history policy.
// Returns the chain and any error from NewBlockChain (including initializeHistoryPruning errors).
func reopenChain(db ethdb.Database, gspec *Genesis, policy history.HistoryPolicy) (*BlockChain, error) {
	cfg := DefaultConfig().WithStateScheme(rawdb.HashScheme)
	cfg.HistoryPolicy = policy
	return NewBlockChain(db, gspec, beacon.New(ethash.NewFaker()), cfg)
}

func TestInitHistoryPruningKeepAllPrunedDB(t *testing.T) {
	db, gspec, _ := newTestChain(t, 200)
	defer db.Close()

	// Pre-prune the freezer to simulate a previously pruned database.
	if _, err := db.TruncateTail(50); err != nil {
		t.Fatalf("failed to truncate tail: %v", err)
	}

	chain, err := reopenChain(db, gspec, history.HistoryPolicy{Mode: history.KeepAll})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer chain.Stop()

	cutoff, _ := chain.HistoryPruningCutoff()
	if cutoff != 50 {
		t.Errorf("prune point: got %d, want 50", cutoff)
	}
}

func TestInitHistoryPruningKeepRecentExpandedWindow(t *testing.T) {
	db, gspec, _ := newTestChain(t, 200)
	defer db.Close()

	// Pre-prune to block 100.
	if _, err := db.TruncateTail(100); err != nil {
		t.Fatalf("failed to truncate tail: %v", err)
	}

	// Reopen with a larger window — tail (100) > target (200-150=50).
	// KeepRecent should accept this (window was expanded).
	policy := history.HistoryPolicy{Mode: history.KeepRecent, Window: 150}
	chain, err := reopenChain(db, gspec, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer chain.Stop()

	cutoff, _ := chain.HistoryPruningCutoff()
	if cutoff != 100 {
		t.Errorf("should accept existing tail: got cutoff=%d, want 100", cutoff)
	}
}

func TestPruneChainHistory(t *testing.T) {
	db, gspec, _ := newTestChain(t, 200)
	defer db.Close()

	chain, err := reopenChain(db, gspec, history.HistoryPolicy{Mode: history.KeepAll})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer chain.Stop()

	// Prune to block 50 and verify the freezer tail and prune point advance.
	if err := chain.pruneChainHistory(50); err != nil {
		t.Fatalf("pruneChainHistory: %v", err)
	}
	tail, _ := db.Tail()
	if tail != 50 {
		t.Errorf("freezer tail: got %d, want 50", tail)
	}
	cutoff, _ := chain.HistoryPruningCutoff()
	if cutoff != 50 {
		t.Errorf("prune cutoff: got %d, want 50", cutoff)
	}

	// Prune again to a higher target.
	if err := chain.pruneChainHistory(100); err != nil {
		t.Fatalf("pruneChainHistory: %v", err)
	}
	tail, _ = db.Tail()
	if tail != 100 {
		t.Errorf("freezer tail after second prune: got %d, want 100", tail)
	}
	cutoff, _ = chain.HistoryPruningCutoff()
	if cutoff != 100 {
		t.Errorf("prune cutoff after second prune: got %d, want 100", cutoff)
	}

	// Prune to a lower target — should be a no-op.
	if err := chain.pruneChainHistory(50); err != nil {
		t.Fatalf("pruneChainHistory (no-op): %v", err)
	}
	tail, _ = db.Tail()
	if tail != 100 {
		t.Errorf("freezer tail after no-op prune: got %d, want 100", tail)
	}
}

func TestInitHistoryPruningStartupPrune(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping slow test")
	}
	db, gspec, blocks := newTestChain(t, 91000)
	defer db.Close()

	// Reopen with a static target at block 500. The chain is long enough
	// (91000 >= 500 + 90000) so initializeHistoryPruning should prune.
	policy := history.HistoryPolicy{
		Mode: history.KeepPostMerge,
		Target: &history.PrunePoint{
			BlockNumber: 500,
			BlockHash:   blocks[499].Hash(),
		},
	}
	chain, err := reopenChain(db, gspec, policy)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	defer chain.Stop()

	tail, _ := db.Tail()
	if tail != 500 {
		t.Errorf("freezer tail: got %d, want 500", tail)
	}
	cutoff, _ := chain.HistoryPruningCutoff()
	if cutoff != 500 {
		t.Errorf("prune cutoff: got %d, want 500", cutoff)
	}
}

func TestInitHistoryPruningStaticModeBeyondTarget(t *testing.T) {
	db, gspec, blocks := newTestChain(t, 200)
	defer db.Close()

	// Pre-prune to block 100.
	if _, err := db.TruncateTail(100); err != nil {
		t.Fatalf("failed to truncate tail: %v", err)
	}

	// Use a static policy with target at block 50 — tail (100) > target (50).
	// Static modes should error.
	policy := history.HistoryPolicy{
		Mode: history.KeepPostMerge,
		Target: &history.PrunePoint{
			BlockNumber: 50,
			BlockHash:   blocks[49].Hash(),
		},
	}
	_, err := reopenChain(db, gspec, policy)
	if err == nil {
		t.Fatal("expected 'pruned beyond' error for static mode, got nil")
	}
}
