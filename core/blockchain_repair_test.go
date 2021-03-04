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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
)

// Tests a recovery for a short canonical chain where a recent block was already
// committed to disk and then the process crashed. In this case we expect the full
// chain to be rolled back to the committed block, but the chain data itself left
// in the database for replaying.
func TestShortRepair(t *testing.T)              { testShortRepair(t, false) }
func TestShortRepairWithSnapshots(t *testing.T) { testShortRepair(t, true) }

func testShortRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 8,
		expSidechainBlocks: 0,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a short canonical chain where the fast sync pivot point was
// already committed, after which the process crashed. In this case we expect the full
// chain to be rolled back to the committed block, but the chain data itself left in
// the database for replaying.
func TestShortFastSyncedRepair(t *testing.T)              { testShortFastSyncedRepair(t, false) }
func TestShortFastSyncedRepairWithSnapshots(t *testing.T) { testShortFastSyncedRepair(t, true) }

func testShortFastSyncedRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 8,
		expSidechainBlocks: 0,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a short canonical chain where the fast sync pivot point was
// not yet committed, but the process crashed. In this case we expect the chain to
// detect that it was fast syncing and not delete anything, since we can just pick
// up directly where we left off.
func TestShortFastSyncingRepair(t *testing.T)              { testShortFastSyncingRepair(t, false) }
func TestShortFastSyncingRepairWithSnapshots(t *testing.T) { testShortFastSyncingRepair(t, true) }

func testShortFastSyncingRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//
	// Frozen: none
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 8,
		expSidechainBlocks: 0,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a short canonical chain and a shorter side chain, where a
// recent block was already committed to disk and then the process crashed. In this
// test scenario the side chain is below the committed block. In this case we expect
// the canonical chain to be rolled back to the committed block, but the chain data
// itself left in the database for replaying.
func TestShortOldForkedRepair(t *testing.T)              { testShortOldForkedRepair(t, false) }
func TestShortOldForkedRepairWithSnapshots(t *testing.T) { testShortOldForkedRepair(t, true) }

func testShortOldForkedRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//   └->S1->S2->S3
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 8,
		expSidechainBlocks: 3,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was already committed to disk and then the process
// crashed. In this test scenario the side chain is below the committed block. In
// this case we expect the canonical chain to be rolled back to the committed block,
// but the chain data itself left in the database for replaying.
func TestShortOldForkedFastSyncedRepair(t *testing.T) {
	testShortOldForkedFastSyncedRepair(t, false)
}
func TestShortOldForkedFastSyncedRepairWithSnapshots(t *testing.T) {
	testShortOldForkedFastSyncedRepair(t, true)
}

func testShortOldForkedFastSyncedRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//   └->S1->S2->S3
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 8,
		expSidechainBlocks: 3,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was not yet committed, but the process crashed. In this
// test scenario the side chain is below the committed block. In this case we expect
// the chain to detect that it was fast syncing and not delete anything, since we
// can just pick up directly where we left off.
func TestShortOldForkedFastSyncingRepair(t *testing.T) {
	testShortOldForkedFastSyncingRepair(t, false)
}
func TestShortOldForkedFastSyncingRepairWithSnapshots(t *testing.T) {
	testShortOldForkedFastSyncingRepair(t, true)
}

func testShortOldForkedFastSyncingRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen: none
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//   └->S1->S2->S3
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 8,
		expSidechainBlocks: 3,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a short canonical chain and a shorter side chain, where a
// recent block was already committed to disk and then the process crashed. In this
// test scenario the side chain reaches above the committed block. In this case we
// expect the canonical chain to be rolled back to the committed block, but the
// chain data itself left in the database for replaying.
func TestShortNewlyForkedRepair(t *testing.T)              { testShortNewlyForkedRepair(t, false) }
func TestShortNewlyForkedRepairWithSnapshots(t *testing.T) { testShortNewlyForkedRepair(t, true) }

func testShortNewlyForkedRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//   └->S1->S2->S3->S4->S5->S6
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    6,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 8,
		expSidechainBlocks: 6,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was already committed to disk and then the process
// crashed. In this test scenario the side chain reaches above the committed block.
// In this case we expect the canonical chain to be rolled back to the committed
// block, but the chain data itself left in the database for replaying.
func TestShortNewlyForkedFastSyncedRepair(t *testing.T) {
	testShortNewlyForkedFastSyncedRepair(t, false)
}
func TestShortNewlyForkedFastSyncedRepairWithSnapshots(t *testing.T) {
	testShortNewlyForkedFastSyncedRepair(t, true)
}

func testShortNewlyForkedFastSyncedRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//   └->S1->S2->S3->S4->S5->S6
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    6,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 8,
		expSidechainBlocks: 6,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a short canonical chain and a shorter side chain, where
// the fast sync pivot point was not yet committed, but the process crashed. In
// this test scenario the side chain reaches above the committed block. In this
// case we expect the chain to detect that it was fast syncing and not delete
// anything, since we can just pick up directly where we left off.
func TestShortNewlyForkedFastSyncingRepair(t *testing.T) {
	testShortNewlyForkedFastSyncingRepair(t, false)
}
func TestShortNewlyForkedFastSyncingRepairWithSnapshots(t *testing.T) {
	testShortNewlyForkedFastSyncingRepair(t, true)
}

func testShortNewlyForkedFastSyncingRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6
	//
	// Frozen: none
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//   └->S1->S2->S3->S4->S5->S6
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    6,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 8,
		expSidechainBlocks: 6,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a short canonical chain and a longer side chain, where a
// recent block was already committed to disk and then the process crashed. In this
// case we expect the canonical chain to be rolled back to the committed block, but
// the chain data itself left in the database for replaying.
func TestShortReorgedRepair(t *testing.T)              { testShortReorgedRepair(t, false) }
func TestShortReorgedRepairWithSnapshots(t *testing.T) { testShortReorgedRepair(t, true) }

func testShortReorgedRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 8,
		expSidechainBlocks: 10,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a short canonical chain and a longer side chain, where
// the fast sync pivot point was already committed to disk and then the process
// crashed. In this case we expect the canonical chain to be rolled back to the
// committed block, but the chain data itself left in the database for replaying.
func TestShortReorgedFastSyncedRepair(t *testing.T) {
	testShortReorgedFastSyncedRepair(t, false)
}
func TestShortReorgedFastSyncedRepairWithSnapshots(t *testing.T) {
	testShortReorgedFastSyncedRepair(t, true)
}

func testShortReorgedFastSyncedRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10
	//
	// Frozen: none
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 8,
		expSidechainBlocks: 10,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a short canonical chain and a longer side chain, where
// the fast sync pivot point was not yet committed, but the process crashed. In
// this case we expect the chain to detect that it was fast syncing and not delete
// anything, since we can just pick up directly where we left off.
func TestShortReorgedFastSyncingRepair(t *testing.T) {
	testShortReorgedFastSyncingRepair(t, false)
}
func TestShortReorgedFastSyncingRepairWithSnapshots(t *testing.T) {
	testShortReorgedFastSyncingRepair(t, true)
}

func testShortReorgedFastSyncingRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10
	//
	// Frozen: none
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in leveldb:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10
	//
	// Expected head header    : C8
	// Expected head fast block: C8
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    8,
		sidechainBlocks:    10,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 8,
		expSidechainBlocks: 10,
		expFrozen:          0,
		expHeadHeader:      8,
		expHeadFastBlock:   8,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks where a recent
// block - newer than the ancient limit - was already committed to disk and then
// the process crashed. In this case we expect the chain to be rolled back to the
// committed block, with everything afterwads kept as fast sync data.
func TestLongShallowRepair(t *testing.T)              { testLongShallowRepair(t, false) }
func TestLongShallowRepairWithSnapshots(t *testing.T) { testLongShallowRepair(t, true) }

func testLongShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks where a recent
// block - older than the ancient limit - was already committed to disk and then
// the process crashed. In this case we expect the chain to be rolled back to the
// committed block, with everything afterwads deleted.
func TestLongDeepRepair(t *testing.T)              { testLongDeepRepair(t, false) }
func TestLongDeepRepairWithSnapshots(t *testing.T) { testLongDeepRepair(t, true) }

func testLongDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks where the fast
// sync pivot point - newer than the ancient limit - was already committed, after
// which the process crashed. In this case we expect the chain to be rolled back
// to the committed block, with everything afterwads kept as fast sync data.
func TestLongFastSyncedShallowRepair(t *testing.T) {
	testLongFastSyncedShallowRepair(t, false)
}
func TestLongFastSyncedShallowRepairWithSnapshots(t *testing.T) {
	testLongFastSyncedShallowRepair(t, true)
}

func testLongFastSyncedShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks where the fast
// sync pivot point - older than the ancient limit - was already committed, after
// which the process crashed. In this case we expect the chain to be rolled back
// to the committed block, with everything afterwads deleted.
func TestLongFastSyncedDeepRepair(t *testing.T)              { testLongFastSyncedDeepRepair(t, false) }
func TestLongFastSyncedDeepRepairWithSnapshots(t *testing.T) { testLongFastSyncedDeepRepair(t, true) }

func testLongFastSyncedDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks where the fast
// sync pivot point - older than the ancient limit - was not yet committed, but the
// process crashed. In this case we expect the chain to detect that it was fast
// syncing and not delete anything, since we can just pick up directly where we
// left off.
func TestLongFastSyncingShallowRepair(t *testing.T) {
	testLongFastSyncingShallowRepair(t, false)
}
func TestLongFastSyncingShallowRepairWithSnapshots(t *testing.T) {
	testLongFastSyncingShallowRepair(t, true)
}

func testLongFastSyncingShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks where the fast
// sync pivot point - newer than the ancient limit - was not yet committed, but the
// process crashed. In this case we expect the chain to detect that it was fast
// syncing and not delete anything, since we can just pick up directly where we
// left off.
func TestLongFastSyncingDeepRepair(t *testing.T)              { testLongFastSyncingDeepRepair(t, false) }
func TestLongFastSyncingDeepRepairWithSnapshots(t *testing.T) { testLongFastSyncingDeepRepair(t, true) }

func testLongFastSyncingDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Expected in leveldb:
	//   C8)->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24
	//
	// Expected head header    : C24
	// Expected head fast block: C24
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    0,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 24,
		expSidechainBlocks: 0,
		expFrozen:          9,
		expHeadHeader:      24,
		expHeadFastBlock:   24,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - newer than the ancient limit - was already
// committed to disk and then the process crashed. In this test scenario the side
// chain is below the committed block. In this case we expect the chain to be
// rolled back to the committed block, with everything afterwads kept as fast
// sync data; the side chain completely nuked by the freezer.
func TestLongOldForkedShallowRepair(t *testing.T) {
	testLongOldForkedShallowRepair(t, false)
}
func TestLongOldForkedShallowRepairWithSnapshots(t *testing.T) {
	testLongOldForkedShallowRepair(t, true)
}

func testLongOldForkedShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - older than the ancient limit - was already
// committed to disk and then the process crashed. In this test scenario the side
// chain is below the committed block. In this case we expect the canonical chain
// to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongOldForkedDeepRepair(t *testing.T)              { testLongOldForkedDeepRepair(t, false) }
func TestLongOldForkedDeepRepairWithSnapshots(t *testing.T) { testLongOldForkedDeepRepair(t, true) }

func testLongOldForkedDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - newer than the ancient limit -
// was already committed to disk and then the process crashed. In this test scenario
// the side chain is below the committed block. In this case we expect the chain
// to be rolled back to the committed block, with everything afterwads kept as
// fast sync data; the side chain completely nuked by the freezer.
func TestLongOldForkedFastSyncedShallowRepair(t *testing.T) {
	testLongOldForkedFastSyncedShallowRepair(t, false)
}
func TestLongOldForkedFastSyncedShallowRepairWithSnapshots(t *testing.T) {
	testLongOldForkedFastSyncedShallowRepair(t, true)
}

func testLongOldForkedFastSyncedShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already committed to disk and then the process crashed. In this test scenario
// the side chain is below the committed block. In this case we expect the canonical
// chain to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongOldForkedFastSyncedDeepRepair(t *testing.T) {
	testLongOldForkedFastSyncedDeepRepair(t, false)
}
func TestLongOldForkedFastSyncedDeepRepairWithSnapshots(t *testing.T) {
	testLongOldForkedFastSyncedDeepRepair(t, true)
}

func testLongOldForkedFastSyncedDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet committed, but the process crashed. In this test scenario the side
// chain is below the committed block. In this case we expect the chain to detect
// that it was fast syncing and not delete anything. The side chain is completely
// nuked by the freezer.
func TestLongOldForkedFastSyncingShallowRepair(t *testing.T) {
	testLongOldForkedFastSyncingShallowRepair(t, false)
}
func TestLongOldForkedFastSyncingShallowRepairWithSnapshots(t *testing.T) {
	testLongOldForkedFastSyncingShallowRepair(t, true)
}

func testLongOldForkedFastSyncingShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet committed, but the process crashed. In this test scenario the side
// chain is below the committed block. In this case we expect the chain to detect
// that it was fast syncing and not delete anything. The side chain is completely
// nuked by the freezer.
func TestLongOldForkedFastSyncingDeepRepair(t *testing.T) {
	testLongOldForkedFastSyncingDeepRepair(t, false)
}
func TestLongOldForkedFastSyncingDeepRepairWithSnapshots(t *testing.T) {
	testLongOldForkedFastSyncingDeepRepair(t, true)
}

func testLongOldForkedFastSyncingDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Expected in leveldb:
	//   C8)->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24
	//
	// Expected head header    : C24
	// Expected head fast block: C24
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    3,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 24,
		expSidechainBlocks: 0,
		expFrozen:          9,
		expHeadHeader:      24,
		expHeadFastBlock:   24,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - newer than the ancient limit - was already
// committed to disk and then the process crashed. In this test scenario the side
// chain is above the committed block. In this case we expect the chain to be
// rolled back to the committed block, with everything afterwads kept as fast
// sync data; the side chain completely nuked by the freezer.
func TestLongNewerForkedShallowRepair(t *testing.T) {
	testLongNewerForkedShallowRepair(t, false)
}
func TestLongNewerForkedShallowRepairWithSnapshots(t *testing.T) {
	testLongNewerForkedShallowRepair(t, true)
}

func testLongNewerForkedShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where a recent block - older than the ancient limit - was already
// committed to disk and then the process crashed. In this test scenario the side
// chain is above the committed block. In this case we expect the canonical chain
// to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongNewerForkedDeepRepair(t *testing.T)              { testLongNewerForkedDeepRepair(t, false) }
func TestLongNewerForkedDeepRepairWithSnapshots(t *testing.T) { testLongNewerForkedDeepRepair(t, true) }

func testLongNewerForkedDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - newer than the ancient limit -
// was already committed to disk and then the process crashed. In this test scenario
// the side chain is above the committed block. In this case we expect the chain
// to be rolled back to the committed block, with everything afterwads kept as fast
// sync data; the side chain completely nuked by the freezer.
func TestLongNewerForkedFastSyncedShallowRepair(t *testing.T) {
	testLongNewerForkedFastSyncedShallowRepair(t, false)
}
func TestLongNewerForkedFastSyncedShallowRepairWithSnapshots(t *testing.T) {
	testLongNewerForkedFastSyncedShallowRepair(t, true)
}

func testLongNewerForkedFastSyncedShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already committed to disk and then the process crashed. In this test scenario
// the side chain is above the committed block. In this case we expect the canonical
// chain to be rolled back to the committed block, with everything afterwads deleted;
// the side chain completely nuked by the freezer.
func TestLongNewerForkedFastSyncedDeepRepair(t *testing.T) {
	testLongNewerForkedFastSyncedDeepRepair(t, false)
}
func TestLongNewerForkedFastSyncedDeepRepairWithSnapshots(t *testing.T) {
	testLongNewerForkedFastSyncedDeepRepair(t, true)
}

func testLongNewerForkedFastSyncedDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet committed, but the process crashed. In this test scenario the side
// chain is above the committed block. In this case we expect the chain to detect
// that it was fast syncing and not delete anything. The side chain is completely
// nuked by the freezer.
func TestLongNewerForkedFastSyncingShallowRepair(t *testing.T) {
	testLongNewerForkedFastSyncingShallowRepair(t, false)
}
func TestLongNewerForkedFastSyncingShallowRepairWithSnapshots(t *testing.T) {
	testLongNewerForkedFastSyncingShallowRepair(t, true)
}

func testLongNewerForkedFastSyncingShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a shorter
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet committed, but the process crashed. In this test scenario the side
// chain is above the committed block. In this case we expect the chain to detect
// that it was fast syncing and not delete anything. The side chain is completely
// nuked by the freezer.
func TestLongNewerForkedFastSyncingDeepRepair(t *testing.T) {
	testLongNewerForkedFastSyncingDeepRepair(t, false)
}
func TestLongNewerForkedFastSyncingDeepRepairWithSnapshots(t *testing.T) {
	testLongNewerForkedFastSyncingDeepRepair(t, true)
}

func testLongNewerForkedFastSyncingDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Expected in leveldb:
	//   C8)->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24
	//
	// Expected head header    : C24
	// Expected head fast block: C24
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    12,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 24,
		expSidechainBlocks: 0,
		expFrozen:          9,
		expHeadHeader:      24,
		expHeadFastBlock:   24,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a longer side
// chain, where a recent block - newer than the ancient limit - was already committed
// to disk and then the process crashed. In this case we expect the chain to be
// rolled back to the committed block, with everything afterwads kept as fast sync
// data. The side chain completely nuked by the freezer.
func TestLongReorgedShallowRepair(t *testing.T)              { testLongReorgedShallowRepair(t, false) }
func TestLongReorgedShallowRepairWithSnapshots(t *testing.T) { testLongReorgedShallowRepair(t, true) }

func testLongReorgedShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a longer side
// chain, where a recent block - older than the ancient limit - was already committed
// to disk and then the process crashed. In this case we expect the canonical chains
// to be rolled back to the committed block, with everything afterwads deleted. The
// side chain completely nuked by the freezer.
func TestLongReorgedDeepRepair(t *testing.T)              { testLongReorgedDeepRepair(t, false) }
func TestLongReorgedDeepRepairWithSnapshots(t *testing.T) { testLongReorgedDeepRepair(t, true) }

func testLongReorgedDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : none
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         nil,
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - newer than the ancient limit -
// was already committed to disk and then the process crashed. In this case we
// expect the chain to be rolled back to the committed block, with everything
// afterwads kept as fast sync data. The side chain completely nuked by the
// freezer.
func TestLongReorgedFastSyncedShallowRepair(t *testing.T) {
	testLongReorgedFastSyncedShallowRepair(t, false)
}
func TestLongReorgedFastSyncedShallowRepairWithSnapshots(t *testing.T) {
	testLongReorgedFastSyncedShallowRepair(t, true)
}

func testLongReorgedFastSyncedShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - older than the ancient limit -
// was already committed to disk and then the process crashed. In this case we
// expect the canonical chains to be rolled back to the committed block, with
// everything afterwads deleted. The side chain completely nuked by the freezer.
func TestLongReorgedFastSyncedDeepRepair(t *testing.T) {
	testLongReorgedFastSyncedDeepRepair(t, false)
}
func TestLongReorgedFastSyncedDeepRepairWithSnapshots(t *testing.T) {
	testLongReorgedFastSyncedDeepRepair(t, true)
}

func testLongReorgedFastSyncedDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G, C4
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4
	//
	// Expected in leveldb: none
	//
	// Expected head header    : C4
	// Expected head fast block: C4
	// Expected head block     : C4
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        4,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 4,
		expSidechainBlocks: 0,
		expFrozen:          5,
		expHeadHeader:      4,
		expHeadFastBlock:   4,
		expHeadBlock:       4,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - newer than the ancient limit -
// was not yet committed, but the process crashed. In this case we expect the
// chain to detect that it was fast syncing and not delete anything, since we
// can just pick up directly where we left off.
func TestLongReorgedFastSyncingShallowRepair(t *testing.T) {
	testLongReorgedFastSyncingShallowRepair(t, false)
}
func TestLongReorgedFastSyncingShallowRepairWithSnapshots(t *testing.T) {
	testLongReorgedFastSyncingShallowRepair(t, true)
}

func testLongReorgedFastSyncingShallowRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2
	//
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2
	//
	// Expected in leveldb:
	//   C2)->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18
	//
	// Expected head header    : C18
	// Expected head fast block: C18
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    18,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 18,
		expSidechainBlocks: 0,
		expFrozen:          3,
		expHeadHeader:      18,
		expHeadFastBlock:   18,
		expHeadBlock:       0,
	}, snapshots)
}

// Tests a recovery for a long canonical chain with frozen blocks and a longer
// side chain, where the fast sync pivot point - older than the ancient limit -
// was not yet committed, but the process crashed. In this case we expect the
// chain to detect that it was fast syncing and not delete anything, since we
// can just pick up directly where we left off.
func TestLongReorgedFastSyncingDeepRepair(t *testing.T) {
	testLongReorgedFastSyncingDeepRepair(t, false)
}
func TestLongReorgedFastSyncingDeepRepairWithSnapshots(t *testing.T) {
	testLongReorgedFastSyncingDeepRepair(t, true)
}

func testLongReorgedFastSyncingDeepRepair(t *testing.T, snapshots bool) {
	// Chain:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24 (HEAD)
	//   └->S1->S2->S3->S4->S5->S6->S7->S8->S9->S10->S11->S12->S13->S14->S15->S16->S17->S18->S19->S20->S21->S22->S23->S24->S25->S26
	//
	// Frozen:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Commit: G
	// Pivot : C4
	//
	// CRASH
	//
	// ------------------------------
	//
	// Expected in freezer:
	//   G->C1->C2->C3->C4->C5->C6->C7->C8
	//
	// Expected in leveldb:
	//   C8)->C9->C10->C11->C12->C13->C14->C15->C16->C17->C18->C19->C20->C21->C22->C23->C24
	//
	// Expected head header    : C24
	// Expected head fast block: C24
	// Expected head block     : G
	testRepair(t, &rewindTest{
		canonicalBlocks:    24,
		sidechainBlocks:    26,
		freezeThreshold:    16,
		commitBlock:        0,
		pivotBlock:         uint64ptr(4),
		expCanonicalBlocks: 24,
		expSidechainBlocks: 0,
		expFrozen:          9,
		expHeadHeader:      24,
		expHeadFastBlock:   24,
		expHeadBlock:       0,
	}, snapshots)
}

func testRepair(t *testing.T, tt *rewindTest, snapshots bool) {
	// It's hard to follow the test case, visualize the input
	//log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	// fmt.Println(tt.dump(true))

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
		config  = &CacheConfig{
			TrieCleanLimit: 256,
			TrieDirtyLimit: 256,
			TrieTimeLimit:  5 * time.Minute,
			SnapshotLimit:  0, // Disable snapshot by default
		}
	)
	if snapshots {
		config.SnapshotLimit = 256
		config.SnapshotWait = true
	}
	chain, err := NewBlockChain(db, config, params.AllEthashProtocolChanges, engine, vm.Config{}, nil, nil)
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
		if snapshots {
			if err := chain.snaps.Cap(canonblocks[tt.commitBlock-1].Root(), 0); err != nil {
				t.Fatalf("Failed to flatten snapshots: %v", err)
			}
		}
	}
	if _, err := chain.InsertChain(canonblocks[tt.commitBlock:]); err != nil {
		t.Fatalf("Failed to import canonical chain tail: %v", err)
	}
	// Force run a freeze cycle
	type freezer interface {
		Freeze(threshold uint64)
		Ancients() (uint64, error)
	}
	db.(freezer).Freeze(tt.freezeThreshold)

	// Set the simulated pivot block
	if tt.pivotBlock != nil {
		rawdb.WriteLastPivotNumber(db, *tt.pivotBlock)
	}
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

	if head := chain.CurrentHeader(); head.Number.Uint64() != tt.expHeadHeader {
		t.Errorf("Head header mismatch: have %d, want %d", head.Number, tt.expHeadHeader)
	}
	if head := chain.CurrentFastBlock(); head.NumberU64() != tt.expHeadFastBlock {
		t.Errorf("Head fast block mismatch: have %d, want %d", head.NumberU64(), tt.expHeadFastBlock)
	}
	if head := chain.CurrentBlock(); head.NumberU64() != tt.expHeadBlock {
		t.Errorf("Head block mismatch: have %d, want %d", head.NumberU64(), tt.expHeadBlock)
	}
	if frozen, err := db.(freezer).Ancients(); err != nil {
		t.Errorf("Failed to retrieve ancient count: %v\n", err)
	} else if int(frozen) != tt.expFrozen {
		t.Errorf("Frozen block count mismatch: have %d, want %d", frozen, tt.expFrozen)
	}
}
