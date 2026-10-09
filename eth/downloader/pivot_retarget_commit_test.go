// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
package downloader

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus/beacon"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

// TestSnapPivotRetargetReceiptCommitWritesCanonical checks the actual
// canonical-index side effect that the new snap/2 state-sync goroutine
// requires. The old pivot is available as a downloaded receipt result, but
// before inserting that result its canonical entry is absent.
func TestSnapPivotRetargetReceiptCommitWritesCanonical(t *testing.T) {
	genesis := &core.Genesis{Config: params.TestChainConfig}
	engine := beacon.New(ethash.NewFaker())
	_, blocks, receipts := core.GenerateChainWithGenesis(genesis, engine, 3, nil)
	tester := newTesterWithGenesis(t, SnapSync, nil, true, genesis, engine)
	defer tester.terminate()
	d := tester.downloader
	d.ancientLimit = 0 // Keep the fixture in the live canonical-index store.

	var pending []*fetchResult
	for i := 0; i < 2; i++ {
		block := blocks[i]
		result := newFetchResult(block.Header(), true, false)
		result.Uncles = block.Uncles()
		result.Transactions = block.Transactions()
		result.Withdrawals = block.Withdrawals()
		var err error
		result.Receipts, err = rlp.EncodeToBytes(receipts[i])
		if err != nil {
			t.Fatal(err)
		}
		pending = append(pending, result)
	}
	oldPivot, newPivot := blocks[0].Header(), blocks[2].Header()
	if got := rawdb.ReadCanonicalHash(tester.db, oldPivot.Number.Uint64()); got != (common.Hash{}) {
		t.Fatalf("fixture already had canonical old pivot %v", got)
	}
	p, before, after := splitAroundPivot(newPivot.Number.Uint64(), pending)
	if p != nil || len(before) != 2 || len(after) != 0 {
		t.Fatalf("wrong pending partition: pivot=%v before=%d after=%d", p, len(before), len(after))
	}
	// Commit under the old state-sync context, before starting a new one.
	// If the snap/2 goroutine started earlier, isPivotReorged would see
	// no indexed old pivot and conclude a false reorg.
	oldSync := newStateSync(d, oldPivot)
	if err := d.commitSnapSyncData(before, oldSync); err != nil {
		t.Fatal(err)
	}
	if got := rawdb.ReadCanonicalHash(tester.db, oldPivot.Number.Uint64()); got != oldPivot.Hash() {
		t.Fatalf("old pivot canonical hash after receipt commit = %v, want %v", got, oldPivot.Hash())
	}
	if got := rawdb.ReadCanonicalHash(tester.db, blocks[1].NumberU64()); got != blocks[1].Hash() {
		t.Fatalf("intermediate canonical hash after receipt commit = %v, want %v", got, blocks[1].Hash())
	}
}
