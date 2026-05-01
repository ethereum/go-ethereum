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

// Regression test for the partial-state restart gap bug: AdvancePartialHead
// must persist the canonical-hash entry for its currentHead (the snap-sync
// pivot), not only for the blocks above it. Without that entry, leveldb is
// missing H<pivot>n, which the freezer's gap-check at startup rejects with
// "gap in the chain between ancients ... and leveldb ...".

package core

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
)

// TestAdvancePartialHeadCoversPivot verifies that AdvancePartialHead writes
// the canonical-hash entry for its currentHead (the "pivot") and not only for
// the strictly newer blocks written by its backfill loop.
//
// Scenario:
//  1. Build an in-memory partial-state chain and insert a few blocks normally.
//  2. Simulate the bug's precondition by deleting the pivot's canonical hash
//     entry from leveldb and rewinding the in-memory head back to the pivot.
//     This mimics the state after the Engine API path persisted the pivot via
//     WriteBlockWithoutState (no canonical-hash key) while InsertReceiptChain
//     skipped writing one because HasBlock was already true.
//  3. Call AdvancePartialHead with a later block. With the fix, the pivot's
//     canonical hash is re-established; without the fix, it stays empty and
//     a subsequent freezer advance would crash on restart.
func TestAdvancePartialHeadCoversPivot(t *testing.T) {
	addr := common.HexToAddress("0xbeef")
	bc, gspec := newPartialBlockchain(t, rawdb.HashScheme, []common.Address{addr})
	defer bc.Stop()

	// Generate a 6-block canonical chain and insert it fully.
	_, blocks, _ := GenerateChainWithGenesis(gspec, ethash.NewFaker(), 6, func(i int, b *BlockGen) {})
	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatalf("failed to insert blocks: %v", err)
	}

	pivot := blocks[2]  // treat block #3 as the pivot
	target := blocks[5] // advance to block #6

	// Simulate the bug's precondition: pivot's canonical hash is missing
	// from leveldb, and the chain head is at the pivot.
	batch := bc.db.NewBatch()
	rawdb.DeleteCanonicalHash(batch, pivot.NumberU64())
	if err := batch.Write(); err != nil {
		t.Fatalf("failed to write batch: %v", err)
	}
	bc.currentBlock.Store(pivot.Header())
	bc.hc.SetCurrentHeader(pivot.Header())

	// Sanity: pivot's canonical hash is now absent.
	if got := rawdb.ReadCanonicalHash(bc.db, pivot.NumberU64()); got != (common.Hash{}) {
		t.Fatalf("setup failed: pivot canonical hash still present: %x", got)
	}

	// The actual call under test.
	if err := bc.AdvancePartialHead(target.Hash()); err != nil {
		t.Fatalf("AdvancePartialHead: %v", err)
	}

	// With the fix: the pivot's canonical hash has been written.
	if got := rawdb.ReadCanonicalHash(bc.db, pivot.NumberU64()); got != pivot.Hash() {
		t.Fatalf("pivot canonical hash not written after AdvancePartialHead: got %x, want %x",
			got, pivot.Hash())
	}
	// Existing behavior: blocks strictly above the pivot are also covered by
	// the backfill loop.
	mid := blocks[4]
	if got := rawdb.ReadCanonicalHash(bc.db, mid.NumberU64()); got != mid.Hash() {
		t.Fatalf("post-pivot canonical hash not written: got %x, want %x",
			got, mid.Hash())
	}
	// And the target itself (bc.CurrentBlock after advance).
	if got := rawdb.ReadCanonicalHash(bc.db, target.NumberU64()); got != target.Hash() {
		t.Fatalf("target canonical hash not written: got %x, want %x",
			got, target.Hash())
	}
	if head := bc.CurrentBlock(); head.Number.Uint64() != target.NumberU64() {
		t.Fatalf("current block not advanced: got %d, want %d", head.Number, target.NumberU64())
	}
}

// TestAdvancePartialHeadIdempotent verifies that repeating AdvancePartialHead
// with a target equal to the current head is a no-op (no error, no panic).
// This can happen if the Engine API re-requests an advance for a head we
// already caught up to; the single-line fix introduced a redundant write
// that must remain harmless.
func TestAdvancePartialHeadIdempotent(t *testing.T) {
	addr := common.HexToAddress("0xbeef")
	bc, gspec := newPartialBlockchain(t, rawdb.HashScheme, []common.Address{addr})
	defer bc.Stop()

	_, blocks, _ := GenerateChainWithGenesis(gspec, ethash.NewFaker(), 3, func(i int, b *BlockGen) {})
	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatalf("failed to insert blocks: %v", err)
	}
	head := blocks[2]

	// First advance (redundant — head is already at `head`). Expected: writes
	// head's canonical hash (already present, so it's a no-op rewrite), loop
	// does not execute.
	if err := bc.AdvancePartialHead(head.Hash()); err != nil {
		t.Fatalf("first AdvancePartialHead: %v", err)
	}
	if got := rawdb.ReadCanonicalHash(bc.db, head.NumberU64()); got != head.Hash() {
		t.Fatalf("head canonical hash lost: got %x, want %x", got, head.Hash())
	}
	// And a second call should remain successful.
	if err := bc.AdvancePartialHead(head.Hash()); err != nil {
		t.Fatalf("second AdvancePartialHead: %v", err)
	}
}

// TestPartialStateRestart_HeadBlock is a small integration check that a
// partial-state chain reopens cleanly and reports the same head block.
// The pebble+ancient persistence path is already covered by blockchain_snapshot_test.go;
// here we only want to confirm that partial-state-enabled config is not
// itself a blocker on restart.
func TestPartialStateRestart_HeadBlock(t *testing.T) {
	// Use the simplified in-memory path. The intent is to catch a regression
	// where AdvancePartialHead corrupts in-memory state such that a subsequent
	// CurrentBlock() read returns a stale value. The persistent-restart
	// scenario is exercised end-to-end via scripts/partial-sync/start_*.sh.
	addr := common.HexToAddress("0xbeef")
	bc, gspec := newPartialBlockchain(t, rawdb.HashScheme, []common.Address{addr})

	_, blocks, _ := GenerateChainWithGenesis(gspec, ethash.NewFaker(), 5, func(i int, b *BlockGen) {})
	if _, err := bc.InsertChain(blocks); err != nil {
		t.Fatalf("failed to insert blocks: %v", err)
	}
	want := blocks[4].Hash()

	if err := bc.AdvancePartialHead(blocks[4].Hash()); err != nil {
		t.Fatalf("AdvancePartialHead: %v", err)
	}
	if got := bc.CurrentBlock().Hash(); got != want {
		t.Fatalf("current block mismatch after advance: got %x, want %x", got, want)
	}

	// The canonical hash at the new head must be consistent (this is the
	// property the freezer's gap-check relies on).
	if got := rawdb.ReadCanonicalHash(bc.db, big.NewInt(5).Uint64()); got != want {
		t.Fatalf("canonical hash at head mismatch: got %x, want %x", got, want)
	}
	bc.Stop()
}
