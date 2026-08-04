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

package downloader

import (
	"math/big"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/ethdb"
)

// TestPivotHeaderReadRace exercises the unsynchronized d.pivotHeader read in
// reportSnapSyncProgress concurrently with a pivot move as performed by
// fetchHeaders (beaconsync.go), which updates d.pivotHeader under pivotLock.
// Both run as sibling fetcher goroutines under spawnSync in production. Run
// with -race: without the pivotLock read guard in reportSnapSyncProgress this
// reports a data race.
func TestPivotHeaderReadRace(t *testing.T) {
	success := make(chan struct{})
	tester := newTesterWithNotification(t, SnapSync, func() { close(success) })
	defer tester.terminate()

	// Sync a small chain for real so the progress-report gates (snap block
	// beyond the sync start point) are satisfied.
	chain := testChainBase.shorten(blockCacheMaxItems - 15)
	tester.newPeer("peer", eth.ETH69, chain.blocks[1:])

	if err := tester.downloader.BeaconSync(chain.blocks[len(chain.blocks)-1].Header(), nil); err != nil {
		t.Fatalf("failed to beacon-sync chain: %v", err)
	}
	select {
	case <-success:
	case <-time.After(3 * time.Second):
		t.Fatalf("failed to sync chain in three seconds")
	}
	d := tester.downloader

	// Drop the skeleton sync status so skeleton.Bounds() fails and the
	// progress report falls back to the pivot header (the "cheat for
	// non-merged networks" path).
	rawdb.DeleteSkeletonSyncStatus(tester.db)
	if _, _, _, err := d.skeleton.Bounds(); err == nil {
		t.Fatalf("skeleton bounds still available, pivot fallback unreachable")
	}
	// Stuff a byte into the ancient store so the report doesn't short-circuit
	// on syncedBytes == 0 (the short test chain freezes nothing on its own).
	if _, err := tester.db.ModifyAncients(func(op ethdb.AncientWriteOp) error {
		for _, table := range []string{
			rawdb.ChainFreezerHeaderTable, rawdb.ChainFreezerHashTable,
			rawdb.ChainFreezerBodiesTable, rawdb.ChainFreezerReceiptTable,
			rawdb.ChainFreezerBALTable,
		} {
			if err := op.AppendRaw(table, 0, []byte{0xde, 0xad}); err != nil {
				return err
			}
		}
		return nil
	}); err != nil {
		t.Fatalf("failed to write ancient data: %v", err)
	}
	// Remaining report gates: cutoff disabled and progress beyond the start.
	if d.chainCutoffNumber != 0 {
		t.Fatalf("unexpected chain cutoff: %d", d.chainCutoffNumber)
	}
	if head := d.blockchain.CurrentSnapBlock().Number.Uint64(); head <= d.syncStartBlock {
		t.Fatalf("no snap progress: head %d, start %d", head, d.syncStartBlock)
	}
	// Race the report (reader) against a pivot move (writer). The writer is
	// the faithful stand-in for beaconsync.go's stale-pivot move: a plain
	// pointer store under pivotLock.
	pivot := &types.Header{
		Number: new(big.Int).Add(chain.blocks[len(chain.blocks)-1].Number(), big.NewInt(1000)),
	}
	var (
		wg    sync.WaitGroup
		start = make(chan struct{})
	)
	wg.Add(2)
	go func() {
		defer wg.Done()
		<-start
		for range 200 {
			d.pivotLock.Lock()
			d.pivotHeader = pivot
			d.pivotLock.Unlock()
		}
	}()
	go func() {
		defer wg.Done()
		<-start
		for range 200 {
			d.reportSnapSyncProgress(true)
		}
	}()
	close(start)
	wg.Wait()
}
