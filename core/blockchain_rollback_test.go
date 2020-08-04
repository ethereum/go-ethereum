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

package core

import (
	"fmt"
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

// Tests that abnormal program termination (i.e.crash) and restart doesn't leave
// the database in some strange state with gaps in the chain, nor with block data
// dangling in the future.
//
// The expected behavior in case of missing head state is to delete all blocks and
// related data up until the first block for which we do have the state, or if we
// exceed the fast sync pivot point to stop there and reenable fast sync.
//
// Note, the trigger condition needs to be the current full block, not the current
// head header, as fast sync is allowed to go further with headers.

// Tests a recovery for a short canonical chain where a recent block was already
// comitted to disk and then the process crashed. In this case we expect the chain
// to be rolled back to the committed block, with everything afterwads deleted.
func TestShortRepair(t *testing.T) {
	testRepair(t, 8, 0, 16, 4, 0, 4, 0)
}

// Tests a recovery for a short canonical chain where the fast sync pivot point was
// already comitted, after which the process crashed. In this case we expect the
// chain to behave like in full sync mode, rolling back to the committed block,
// with everything afterwads deleted.
func TestShortFastSyncedRepair(t *testing.T) {
	testRepair(t, 8, 0, 16, 4, 4, 4, 0)
}

// Tests a recovery for a short canonical chain where the fast sync pivot point was
// not yet comitted, but the process crashed. In this case we expect the chain to
// detect that it was fast syncing and not delete anything, since we can just pick
// up directly where we left off.
func TestShortFastSyncingRepair(t *testing.T) {
	testRepair(t, 8, 0, 16, 0, 4, 8, 0)
}

// Tests a recovery for a short canonical chain and a shorter side chain, where a
// recent block was already comitted to disk and then the process crashed. In this
// test scenario the side chain is below the commited block. In this case we expect
// the canonical chain to be rolled back to the committed block, with everything
// afterwads deleted; but the side chain left alone as it was shorter.
func TestShortOldForkedRepair(t *testing.T) {
	testRepair(t, 8, 3, 16, 4, 0, 4, 3)
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was already comitted to disk and then the process
// crashed. In this test scenario the side chain is below the commited block. In
// this case we expect the canonical chain to be rolled back to the committed block,
// with everything afterwads deleted; but the side chain left alone as it was shorter.
func TestShortOldForkedFastSyncedRepair(t *testing.T) {
	testRepair(t, 8, 3, 16, 4, 4, 4, 3)
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was not yet comitted, but the process crashed. In this
// test scenario the side chain is below the commited block. In this case we expect
// the chain to detect that it was fast syncing and not delete anything, since we
// can just pick up directly where we left off.
func TestShortOldForkedFastSyncingRepair(t *testing.T) {
	testRepair(t, 8, 3, 16, 0, 4, 8, 3)
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
	testRepair(t, 8, 6, 16, 4, 0, 4, 4)
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
	testRepair(t, 8, 6, 16, 4, 4, 4, 4)
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was not yet comitted, but the process crashed. In
// this test scenario the side chain reaches above the commited block. In this
// case we expect the chain to detect that it was fast syncing and not delete
// anything, since we can just pick up directly where we left off.
func TestShortNewlyForkedFastSyncingRepair(t *testing.T) {
	testRepair(t, 8, 6, 16, 0, 4, 8, 6)
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
	testRepair(t, 8, 10, 16, 4, 0, 4, 4)
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
	testRepair(t, 8, 10, 16, 4, 4, 4, 4)
}

// Tests a recovery for a short canonical chain and a longer side chain, where
// the fast sync pivot point was not yet comitted, but the process crashed. In
// this case we expect the chain to detect that it was fast syncing and not delete
// anything, since we can just pick up directly where we left off.
func TestShortReorgedFastSyncingRepair(t *testing.T) {
	testRepair(t, 8, 10, 16, 0, 4, 8, 10)
}

// Tests a recovery for a long canonical chain with frozen blocks where a recent
// block - older than the ancient limit - was already comitted to disk and then
// the process crashed. In this case we expect the chain to be rolled back to the
// committed block, with everything afterwads deleted.
func TestLongRepair(t *testing.T) {
	testRepair(t, 24, 0, 16, 4, 0, 4, 0)
}

// Tests a recovery for a long canonical chain with frozen blocks where the fast
// sync pivot point - older than the ancient limit - was already comitted, after
// which the process crashed. In this case we expect the chain to behave like in
// full sync mode, rolling back to the committed block, with everything afterwads
// deleted.
func TestLongFastSyncedRepair(t *testing.T) {
	testRepair(t, 24, 0, 16, 4, 4, 4, 0)
}

// Tests a recovery for a long canonical chain with frozen blocks where the fast
// sync pivot point - older than the ancient limit - was not yet comitted, but the
// process crashed. In this case we expect the chain to detect that it was fast
// syncing and not delete anything, since we can just pick up directly where we
// left off.
func TestLongFastSyncingRepair(t *testing.T) {
	testRepair(t, 24, 0, 16, 0, 4, 24, 0)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - older than the ancient limit - was already
// comitted to disk and then the process crashed. In this test scenario the side
// chain is below the commited block. In this case we expect the canonical chain
// to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongOldForkedRepair(t *testing.T) {
	testRepair(t, 24, 3, 16, 4, 0, 4, 0)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already comitted to disk and then the process crashed. In this test scenario
// the side chain is below the commited block. In this case we expect the canonical
// chain to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongOldForkedFastSyncedRepair(t *testing.T) {
	testRepair(t, 24, 3, 16, 4, 4, 4, 0)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet comitted, but the process crashed. In this test scenario the side
// chain is below the commited block. In this case we expect the chain to detect
// that it was fast syncing and not delete anything. The side chin is completely
// nuked by the freezer.
func TestLongOldForkedFastSyncingRepair(t *testing.T) {
	testRepair(t, 24, 3, 16, 0, 4, 24, 0)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - older than the ancient limit - was already
// comitted to disk and then the process crashed. In this test scenario the side
// chain is abo e the commited block. In this case we expect the canonical chain
// to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongNewerForkedRepair(t *testing.T) {
	testRepair(t, 24, 12, 16, 4, 0, 4, 0)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already comitted to disk and then the process crashed. In this test scenario
// the side chain is abo e the commited block. In this case we expect the canonical
// chain to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongNewerForkedFastSyncedRepair(t *testing.T) {
	testRepair(t, 24, 12, 16, 4, 4, 4, 0)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet comitted, but the process crashed. In this test scenario the side
// chain is abo e the commited block. In this case we expect the chain to detect
// that it was fast syncing and not delete anything. The side chain is completely
// nuked by the freezer.
func TestLongNewerForkedFastSyncingRepair(t *testing.T) {
	testRepair(t, 24, 12, 16, 0, 4, 24, 0)
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
	testRepair(t, 24, 26, 16, 4, 0, 4, 0)
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
	testRepair(t, 24, 26, 16, 4, 4, 4, 0)
}

// Tests a recovery for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet comitted, but the process crashed. In this case we expect the
// chain to detect that it was fast syncing and not delete anything, since we
// can just pick up directly where we left off.
func TestLongReorgedFastSyncingRepair(t *testing.T) {
	testRepair(t, 24, 26, 16, 0, 4, 24, 26)
}

func testRepair(t *testing.T, canonchain int, sidechain int, freeze uint64, commit int, pivot int, canonretain int, sideretain int) {
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
	if sidechain > 0 {
		sideblocks, _ = GenerateChain(params.TestChainConfig, genesis, engine, rawdb.NewMemoryDatabase(), sidechain, func(i int, b *BlockGen) {
			b.SetCoinbase(common.Address{0x01})
		})
		if _, err := chain.InsertChain(sideblocks); err != nil {
			t.Fatalf("Failed to import side chain: %v", err)
		}
	}
	canonblocks, _ := GenerateChain(params.TestChainConfig, genesis, engine, rawdb.NewMemoryDatabase(), canonchain, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{0x02})
		b.SetDifficulty(big.NewInt(1000000))
	})
	if _, err := chain.InsertChain(canonblocks[:commit]); err != nil {
		t.Fatalf("Failed to import canonical chain start: %v", err)
	}
	if commit > 0 {
		chain.stateCache.TrieDB().Commit(canonblocks[commit-1].Root(), true, nil)
	}
	if _, err := chain.InsertChain(canonblocks[commit:]); err != nil {
		t.Fatalf("Failed to import canonical chain tail: %v", err)
	}
	if sidechain > 0 {
		fmt.Println(sideblocks[sidechain-1].Hash(), canonblocks[canonchain-1].Hash())
		fmt.Println(chain.CurrentBlock().Hash())
	}
	// Force run a freeze cycle
	type freezer interface {
		Freeze(threshold uint64)
	}
	db.(freezer).Freeze(freeze)

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
	verifyNoGaps(t, chain, canonblocks)
	verifyNoGaps(t, chain, sideblocks)
	verifyCutoff(t, chain, canonblocks, canonretain)
	verifyCutoff(t, chain, sideblocks, sideretain)
}

// verifyNoGaps checks that there are no gaps after the initial set of blocks in
// the database and errors if found.
func verifyNoGaps(t *testing.T, chain *BlockChain, inserted types.Blocks) {
	t.Helper()

	var end uint64
	for i := uint64(0); i <= uint64(len(inserted)); i++ {
		header := chain.GetHeaderByNumber(i)
		if header == nil && end == 0 {
			end = i
		}
		if header != nil && end > 0 {
			t.Errorf("Header gap between #%d-#%d", end, i-1)
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
			t.Errorf("Block gap between #%d-#%d", end, i-1)
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
			t.Errorf("Receipt gap between #%d-#%d", end, i-1)
			end = 0 // Reset for further gap detection
		}
	}
}

// verifyCutoff checks that there are no chain data available in the chain after
// the specified limit, but that it is available before.
func verifyCutoff(t *testing.T, chain *BlockChain, inserted types.Blocks, head int) {
	t.Helper()

	for i := 1; i <= len(inserted); i++ {
		if i <= head {
			if header := chain.GetHeader(inserted[i-1].Hash(), uint64(i)); header == nil {
				t.Errorf("Header   #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
			}
			if block := chain.GetBlock(inserted[i-1].Hash(), uint64(i)); block == nil {
				t.Errorf("Block    #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
			}
			if receipts := chain.GetReceiptsByHash(inserted[i-1].Hash()); receipts == nil {
				t.Errorf("Receipts #%2d [%x...] missing before cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
			}
		} else {
			if header := chain.GetHeader(inserted[i-1].Hash(), uint64(i)); header != nil {
				t.Errorf("Header   #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
			}
			if block := chain.GetBlock(inserted[i-1].Hash(), uint64(i)); block != nil {
				t.Errorf("Block    #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
			}
			if receipts := chain.GetReceiptsByHash(inserted[i-1].Hash()); receipts != nil {
				t.Errorf("Receipts #%2d [%x...] present after cap %d", inserted[i-1].Number(), inserted[i-1].Hash().Bytes()[:3], head)
			}
		}
	}
}
