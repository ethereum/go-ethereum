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

// Tests that abnormal program termination (i.e.crash) and restart doesn't leave
// the database in some strange state with gaps in the chain, nor with block data
// dangling in the future.

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

// repairTest is a test case for chain reparation after a crash.
type repairTest struct {
	canonicalBlocks int    // Number of blocks to generate for the canonical chain (heavier)
	sidechainBlocks int    // Number of blocks to generate for the side chain (lighter)
	freezeThreshold uint64 // Block number until which to move things into the freezer
	commitBlock     uint64 // Block number for which to commit the state to disk
	pivotBlock      uint64 // Pivot block number in case of fast sync

	expCanonicalBlocks int // Number of canonical blocks expected to remain in the database
	expSidechainBlocks int // Number of sidechain blocks expected to remain in the database
}

// Tests a recovery for a short canonical chain where a recent block was already
// comitted to disk and then the process crashed. In this case we expect the chain
// to be rolled back to the committed block, with everything afterwads deleted.
func TestShortRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a short canonical chain where the fast sync pivot point was
// already comitted, after which the process crashed. In this case we expect the
// chain to behave like in full sync mode, rolling back to the committed block,
// with everything afterwads deleted.
func TestShortFastSyncedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a short canonical chain where the fast sync pivot point was
// not yet comitted, but the process crashed. In this case we expect the chain to
// detect that it was fast syncing and not delete anything, since we can just pick
// up directly where we left off.
func TestShortFastSyncingRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		expCanonicalBlocks: 8,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a short canonical chain and a shorter side chain, where a
// recent block was already comitted to disk and then the process crashed. In this
// test scenario the side chain is below the commited block. In this case we expect
// the canonical chain to be rolled back to the committed block, with everything
// afterwads deleted; but the side chain left alone as it was shorter.
func TestShortOldForkedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 3,
	})
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was already comitted to disk and then the process
// crashed. In this test scenario the side chain is below the commited block. In
// this case we expect the canonical chain to be rolled back to the committed block,
// with everything afterwads deleted; but the side chain left alone as it was shorter.
func TestShortOldForkedFastSyncedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 3,
	})
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was not yet comitted, but the process crashed. In this
// test scenario the side chain is below the commited block. In this case we expect
// the chain to detect that it was fast syncing and not delete anything, since we
// can just pick up directly where we left off.
func TestShortOldForkedFastSyncingRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		expCanonicalBlocks: 8,
		expSidechainBlocks: 3,
	})
}

// Tests a recovery for a short canonical chain and a shorter side chain, where a
// recent block was already comitted to disk and then the process crashed. In this
// test scenario the side chain reaches above the commited block. In this case we
// expect both canonical and side chains to be rolled back to the committed block,
// with everything afterwads deleted.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortNewlyForkedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    6,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 4,
	})
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was already comitted to disk and then the process
// crashed. In this test scenario the side chain reaches above the commited block.
// In this case we expect both canonical and side chains to be rolled back to the
// committed block, with everything afterwads deleted.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortNewlyForkedFastSyncedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    6,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 4,
	})
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was not yet comitted, but the process crashed. In
// this test scenario the side chain reaches above the commited block. In this
// case we expect the chain to detect that it was fast syncing and not delete
// anything, since we can just pick up directly where we left off.
func TestShortNewlyForkedFastSyncingRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    6,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		expCanonicalBlocks: 8,
		expSidechainBlocks: 6,
	})
}

// Tests a recovery for a short canonical chain and a longer side chain, where a
// recent block was already comitted to disk and then the process crashed. In this
// case we expect both canonical and side chains to be rolled back to the committed
// block, with everything afterwads deleted.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortReorgedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 4,
	})
}

// Tests a recovery for a short canonical chain and a longer side chain, where
// the fast sync pivot point was already comitted to disk and then the process
// crashed. In this case we expect both canonical and side chains to be rolled
// back to the committed block, with everything afterwads deleted.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestShortReorgedFastSyncedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 4,
	})
}

// Tests a recovery for a short canonical chain and a longer side chain, where
// the fast sync pivot point was not yet comitted, but the process crashed. In
// this case we expect the chain to detect that it was fast syncing and not delete
// anything, since we can just pick up directly where we left off.
func TestShortReorgedFastSyncingRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		expCanonicalBlocks: 8,
		expSidechainBlocks: 10,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks where a recent
// block - older than the ancient limit - was already comitted to disk and then
// the process crashed. In this case we expect the chain to be rolled back to the
// committed block, with everything afterwads deleted.
func TestLongRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks where the fast
// sync pivot point - older than the ancient limit - was already comitted, after
// which the process crashed. In this case we expect the chain to behave like in
// full sync mode, rolling back to the committed block, with everything afterwads
// deleted.
func TestLongFastSyncedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks where the fast
// sync pivot point - older than the ancient limit - was not yet comitted, but the
// process crashed. In this case we expect the chain to detect that it was fast
// syncing and not delete anything, since we can just pick up directly where we
// left off.
func TestLongFastSyncingRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		expCanonicalBlocks: 24,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - older than the ancient limit - was already
// comitted to disk and then the process crashed. In this test scenario the side
// chain is below the commited block. In this case we expect the canonical chain
// to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongOldForkedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already comitted to disk and then the process crashed. In this test scenario
// the side chain is below the commited block. In this case we expect the canonical
// chain to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongOldForkedFastSyncedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet comitted, but the process crashed. In this test scenario the side
// chain is below the commited block. In this case we expect the chain to detect
// that it was fast syncing and not delete anything. The side chin is completely
// nuked by the freezer.
func TestLongOldForkedFastSyncingRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		expCanonicalBlocks: 24,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - older than the ancient limit - was already
// comitted to disk and then the process crashed. In this test scenario the side
// chain is above the commited block. In this case we expect the canonical chain
// to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongNewerForkedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already comitted to disk and then the process crashed. In this test scenario
// the side chain is above the commited block. In this case we expect the canonical
// chain to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongNewerForkedFastSyncedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet comitted, but the process crashed. In this test scenario the side
// chain is above the commited block. In this case we expect the chain to detect
// that it was fast syncing and not delete anything. The side chain is completely
// nuked by the freezer.
func TestLongNewerForkedFastSyncingRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		expCanonicalBlocks: 24,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks and a longer side
// chain, where a recent block - older than the ancient limit - was already comitted
// to disk and then the process crashed. In this case we expect the canonical chains
// to be rolled back to the committed block, with everything afterwads deleted. The
// side chain completely nuked by the freezer.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestLongReorgedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         0,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already comitted to disk and then the process crashed. In this case we
// expect the canonical chains to be rolled back to the committed block, with
// everything afterwads deleted. The side chain completely nuked by the freezer.
//
// The side chain could be left to be if the fork point was before the new head
// we are deleting to, but it would be exceedingly hard to detect that case and
// properly handle it, so we'll trade extra work in exchange for simpler code.
func TestLongReorgedFastSyncedRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         4,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
	})
}

// Tests a recovery for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet comitted, but the process crashed. In this case we expect the
// chain to detect that it was fast syncing and not delete anything, since we
// can just pick up directly where we left off.
func TestLongReorgedFastSyncingRepair(t *testing.T) {
	testRepair(t, &repairTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         4,
		expCanonicalBlocks: 24,
		expSidechainBlocks: 26,
	})
}

func testRepair(t *testing.T, tt *repairTest) {
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
	defer db.Close() // Might double close, should be fine

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

	// Pull the plug on the database, simulating a hard crash
	db.Close()

	// Start a new blockchain back up and see where the repait leads us
	db, err = rawdb.NewLevelDBDatabaseWithFreezer(datadir, 0, 0, datadir, "")
	if err != nil {
		t.Fatalf("Failed to reopen persistent database: %v", err)
	}
	defer db.Close()

	chain, err = NewBlockChain(db, nil, params.AllEthashProtocolChanges, engine, vm.Config{}, nil, nil)
	if err != nil {
		t.Fatalf("Failed to recreate chain: %v", err)
	}
	defer chain.Stop()

	// Iterate over all the remaining blocks and ensure there are no gaps
	verifyNoGaps(t, chain, true, canonblocks)
	verifyNoGaps(t, chain, false, sideblocks)
	verifyCutoff(t, chain, true, canonblocks, tt.expCanonicalBlocks)
	verifyCutoff(t, chain, false, sideblocks, tt.expSidechainBlocks)
}

// verifyNoGaps checks that there are no gaps after the initial set of blocks in
// the database and errors if found.
func verifyNoGaps(t *testing.T, chain *BlockChain, canonical bool, inserted types.Blocks) {
	t.Helper()

	var end uint64
	for i := uint64(0); i <= uint64(len(inserted)); i++ {
		header := chain.GetHeaderByNumber(i)
		if header == nil && end == 0 {
			end = i
		}
		if header != nil && end > 0 {
			if canonical {
				t.Errorf("Canonical header gap between #%d-#%d", end, i-1)
			} else {
				t.Errorf("Sidechain header gap between #%d-#%d", end, i-1)
			}
			end = 0 // Reset for further gap detection
		}
	}
	end = 0
	for i := uint64(0); i <= uint64(len(inserted)); i++ {
		block := chain.GetBlockByNumber(i)
		if block == nil && end == 0 {
			end = i
		}
		if block != nil && end > 0 {
			if canonical {
				t.Errorf("Canonical block gap between #%d-#%d", end, i-1)
			} else {
				t.Errorf("Sidechain block gap between #%d-#%d", end, i-1)
			}
			end = 0 // Reset for further gap detection
		}
	}
	end = 0
	for i := uint64(1); i <= uint64(len(inserted)); i++ {
		receipts := chain.GetReceiptsByHash(inserted[i-1].Hash())
		if receipts == nil && end == 0 {
			end = i
		}
		if receipts != nil && end > 0 {
			if canonical {
				t.Errorf("Canonical receipt gap between #%d-#%d", end, i-1)
			} else {
				t.Errorf("Sidechain receipt gap between #%d-#%d", end, i-1)
			}
			end = 0 // Reset for further gap detection
		}
	}
}

// verifyCutoff checks that there are no chain data available in the chain after
// the specified limit, but that it is available before.
func verifyCutoff(t *testing.T, chain *BlockChain, canonical bool, inserted types.Blocks, head int) {
	t.Helper()

	for i := 1; i <= len(inserted); i++ {
		if i <= head {
			if header := chain.GetHeader(inserted[i-1].Hash(), uint64(i)); header == nil {
				if canonical {
					t.Errorf("Canonical header   #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain header   #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
			if block := chain.GetBlock(inserted[i-1].Hash(), uint64(i)); block == nil {
				if canonical {
					t.Errorf("Canonical block    #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain block    #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
			if receipts := chain.GetReceiptsByHash(inserted[i-1].Hash()); receipts == nil {
				if canonical {
					t.Errorf("Canonical receipts #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain receipts #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
		} else {
			if header := chain.GetHeader(inserted[i-1].Hash(), uint64(i)); header != nil {
				if canonical {
					t.Errorf("Canonical header   #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain header   #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
			if block := chain.GetBlock(inserted[i-1].Hash(), uint64(i)); block != nil {
				if canonical {
					t.Errorf("Canonical block    #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain block    #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
			if receipts := chain.GetReceiptsByHash(inserted[i-1].Hash()); receipts != nil {
				if canonical {
					t.Errorf("Canonical receipts #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				} else {
					t.Errorf("Sidechain receipts #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
				}
			}
		}
	}
}
