// Copyright 2020 The go-ethereum Authors
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

// Tests that setting the chain head backwards doesn't leave the database in some
// strange state with gaps in the chain, nor with block data dangling in the future.

package core

import (
	"io/ioutil"
	"math/big"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// setHeadTest is a test case for chain rollback upon user request.
type setHeadTest struct {
	canonicalBlocks int    // Number of blocks to generate for the canonical chain (heavier)
	sidechainBlocks int    // Number of blocks to generate for the side chain (lighter)
	freezeThreshold uint64 // Block number until which to move things into the freezer
	commitBlock     uint64 // Block number for which to commit the state to disk
	pivotBlock      uint64 // Pivot block number in case of fast sync

	setheadBlock       uint64 // Block number to set head back to
	expCanonicalBlocks int    // Number of canonical blocks expected to remain in the database
	expSidechainBlocks int    // Number of sidechain blocks expected to remain in the database
}

// Tests a sethead for a short canonical chain where a recent block was already
// comitted to disk and then the sethead called. In this case we expect the chain
// to be rolled back to the committed block, with everything afterwads deleted.
func TestShortSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		setheadBlock:       7,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a short canonical chain where the fast sync pivot point was
// already comitted, after which sethead was called. In this case we expect the
// chain to behave like in full sync mode, rolling back to the committed block,
// with everything afterwads deleted.
func TestShortFastSyncedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		setheadBlock:       7,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a short canonical chain where the fast sync pivot point was
// not yet comitted, but sethead was called. In this case we expect the chain to
// detect that it was fast syncing and delete everything from the new head, since
// we can just pick up fast syncing from there.
func TestShortFastSyncingSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		setheadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a short canonical chain and a shorter side chain, where a
// recent block was already comitted to disk and then sethead was called. In this
// test scenario the side chain is below the commited block. In this case we expect
// the canonical chain to be rolled back to the committed block, with everything
// afterwads deleted; but the side chain left alone as it was shorter.
func TestShortOldForkedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		setheadBlock:       7,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 3,
	})
}

// Tests a sethead for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was already comitted to disk and then sethead was
// called. In this test scenario the side chain is below the commited block. In
// this case we expect the canonical chain to be rolled back to the committed block,
// with everything afterwads deleted; but the side chain left alone as it was shorter.
func TestShortOldForkedFastSyncedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		setheadBlock:       7,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 3,
	})
}

// Tests a sethead for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was not yet comitted, but sethead was called. In this
// test scenario the side chain is below the commited block. In this case we expect
// the chain to detect that it was fast syncing and delete everything from the new
// head, since we can just pick up fast syncing from there.
func TestShortOldForkedFastSyncingSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		setheadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 3,
	})
}

// Tests a sethead for a short canonical chain and a shorter side chain, where a
// recent block was already comitted to disk and then sethead was called. In this
// test scenario the side chain reaches above the commited block. In this case we
// expect both canonical and side chains to be rolled back to the committed block,
// with everything afterwads deleted.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortNewlyForkedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    6,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		setheadBlock:       7,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 4,
	})
}

// Tests a sethead for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was already comitted to disk and then sethead was
// called. In this test scenario the side chain reaches above the commited block.
// In this case we expect both canonical and side chains to be rolled back to the
// committed block, with everything afterwads deleted.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortNewlyForkedFastSyncedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    6,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		setheadBlock:       7,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 4,
	})
}

// Tests a sethead for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was not yet comitted, but sethead was called. In
// this test scenario the side chain reaches above the commited block. In this
// case we expect the chain to detect that it was fast syncing and delete
// everything from the new head, since we can just pick up fast syncing from
// there.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortNewlyForkedFastSyncingSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    6,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		setheadBlock:       5,
		expCanonicalBlocks: 5,
		expSidechainBlocks: 5,
	})
}

// Tests a sethead for a short canonical chain and a longer side chain, where a
// recent block was already comitted to disk and then sethead was called. In this
// case we expect both canonical and side chains to be rolled back to the committed
// block, with everything afterwads deleted.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortReorgedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		setheadBlock:       7,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 4,
	})
}

// Tests a sethead for a short canonical chain and a longer side chain, where
// the fast sync pivot point was already comitted to disk and then sethead was
// called. In this case we expect both canonical and side chains to be rolled
// back to the committed block, with everything afterwads deleted.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortReorgedFastSyncedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		setheadBlock:       7,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 4,
	})
}

// Tests a sethead for a short canonical chain and a longer side chain, where
// the fast sync pivot point was not yet comitted, but sethead was called. In
// this case we expect the chain to detect that it was fast syncing and delete
// everything from the new head, since we can just pick up fast syncing from
// there.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortReorgedFastSyncingSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		setheadBlock:       7,
		expCanonicalBlocks: 7,
		expSidechainBlocks: 7,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks where a recent
// block - older than the ancient limit - was already comitted to disk and then
// sethead was called. In this case we expect the chain to be rolled back to the
// committed block, with everything afterwads deleted.
func TestLongSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		setheadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks where the fast
// sync pivot point - older than the ancient limit - was already comitted, after
// which sethead was called. In this case we expect the chain to behave like in
// full sync mode, rolling back to the committed block, with everything afterwads
// deleted.
func TestLongFastSyncedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		setheadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks where the fast
// sync pivot point - older than the ancient limit - was not yet comitted, but
// sethead was called. In this case we expect the chain to detect that it was fast
// syncing and delete everything from the new head, since we can just pick up fast
// syncing from there.
func TestLongFastSyncingSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		setheadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - older than the ancient limit - was already
// comitted to disk and then sethead was called. In this test scenario the side
// chain is below the commited block. In this case we expect the canonical chain
// to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongOldForkedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		setheadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already comitted to disk and then sethead was called. In this test scenario
// the side chain is below the commited block. In this case we expect the canonical
// chain to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongOldForkedFastSyncedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		setheadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet comitted, but sethead was called. In this test scenario the side
// chain is below the commited block. In this case we expect the chain to detect
// that it was fast syncing and delete everything from the new head, since we can
// just pick up fast syncing from there. The side chin is completely nuked by the
// freezer.
func TestLongOldForkedFastSyncingSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		setheadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - older than the ancient limit - was already
// comitted to disk and then sethead was called. In this test scenario the side
// chain is above the commited block. In this case we expect the canonical chain
// to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongNewerForkedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		setheadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already comitted to disk and then sethead was called. In this test scenario
// the side chain is above the commited block. In this case we expect the canonical
// chain to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongNewerForkedFastSyncedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		setheadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet comitted, but sethead was called. In this test scenario the side
// chain is above the commited block. In this case we expect the chain to detect
// that it was fast syncing and delete everything from the new head, since we can
// just pick up fast syncing from there. The side chain is completely nuked by
// the freezer.
func TestLongNewerForkedFastSyncingSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		setheadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks and a longer side
// chain, where a recent block - older than the ancient limit - was already comitted
// to disk and then sethead was called. In this case we expect the canonical chains
// to be rolled back to the committed block, with everything afterwads deleted. The
// side chain completely nuked by the freezer.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestLongReorgedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		setheadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already comitted to disk and then sethead was called. In this case we
// expect the canonical chains to be rolled back to the committed block, with
// everything afterwads deleted. The side chain completely nuked by the freezer.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestLongReorgedFastSyncedSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		setheadBlock:       6,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a sethead for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet comitted, but sethead was called. In this case we expect the
// chain to detect that it was fast syncing and delete everything from the new
// head, since we can just pick up fast syncing from there. The side chain is
// completely nuked by the freezer.
func TestLongReorgedFastSyncingSetHead(t *testing.T) {
	testSetHead(t, &setHeadTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		setheadBlock:       6,
		expCanonicalBlocks: 6,
		expSidechainBlocks: 0,
	})
}

func testSetHead(t *testing.T, tt *setHeadTest) {
	//log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))

	// Create a temporary persistent database
	datadir, err := ioutil.TempDir("", "")
	if err != nil {
		t.Fatalf("Failed to create temporary datadir: %v", err)
	}
	os.RemoveAll(datadir)

	db, err := rawdb.NewLevelDBDatabaseWithFreezer(datadir, 0, 0, datadir, "")
	if err != nil {
		t.Fatalf("Failed to create persistent database: %v", err)
	}
	defer db.Close()

	// Initialize a fresh chain
	var (
		genesis = new(Genesis).MustCommit(db)
		engine  = ethash.NewFullFaker()
	)
	chain, err := NewBlockChain(db, nil, params.AllEthashProtocolChanges, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to create chain: %v", err)
	}
	// If sidechain blocks are needed, make a light chain and import it
	var sideblocks types.Blocks
	if tt.sidechainBlocks > 0 {
		sideblocks, _ = GenerateChain(params.TestChainConfig, genesis, engine, rawdb.NewMemoryDatabase(), tt.sidechainBlocks, func(i int, b *BlockGen) {
			b.SetCoinbase(common.Address{0x01})
		})
		if _, err := chain.InsertChain(sideblocks); err != nil {
			t.Fatalf("Failed to import side chain: %v", err)
		}
	}
	canonblocks, _ := GenerateChain(params.TestChainConfig, genesis, engine, rawdb.NewMemoryDatabase(), tt.canonicalBlocks, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{0x02})
		b.SetDifficulty(big.NewInt(1000000))
	})
	if _, err := chain.InsertChain(canonblocks[:tt.commitBlock]); err != nil {
		t.Fatalf("Failed to import canonical chain start: %v", err)
	}
	if tt.commitBlock > 0 {
		chain.stateCache.TrieDB().Commit(canonblocks[tt.commitBlock-1].Root(), true, nil)
	}
	if _, err := chain.InsertChain(canonblocks[tt.commitBlock:]); err != nil {
		t.Fatalf("Failed to import canonical chain tail: %v", err)
	}
	// Force run a freeze cycle
	type freezer interface {
		Freeze(threshold uint64)
	}
	db.(freezer).Freeze(tt.freezeThreshold)

	// Set the head of the chain back to the requested number
	chain.SetHead(tt.setheadBlock)

	// Iterate over all the remaining blocks and ensure there are no gaps
	verifyNoGaps(t, chain, true, canonblocks)
	verifyNoGaps(t, chain, false, sideblocks)
	verifyCutoff(t, chain, true, canonblocks, tt.expCanonicalBlocks)
	verifyCutoff(t, chain, false, sideblocks, tt.expSidechainBlocks)
}
