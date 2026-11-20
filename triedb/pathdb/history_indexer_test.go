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

package pathdb

import (
	"errors"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/internal/testrand"
)

// TestHistoryIndexerShortenDeadlock tests that a call to shorten does not
// deadlock when the indexer is active. This specifically targets the case where
// signal.result must be sent to unblock the caller.
func TestHistoryIndexerShortenDeadlock(t *testing.T) {
	// log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelDebug, true)))
	db := rawdb.NewMemoryDatabase()
	freezer, _ := rawdb.NewStateFreezer(t.TempDir(), false)
	defer freezer.Close()

	histories := makeStateHistories(100)
	for i, h := range histories {
		accountData, storageData, accountIndex, storageIndex := h.encode()
		rawdb.WriteStateHistory(freezer, uint64(i+1), h.meta.encode(), accountIndex, storageIndex, accountData, storageData)
	}
	// As a workaround, assign a future block to keep the initer running indefinitely
	indexer := newHistoryIndexer(db, freezer, 200, typeStateHistory, true)
	defer indexer.close()

	done := make(chan error, 1)
	go func() {
		done <- indexer.shorten(200)
	}()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("shorten returned an unexpected error: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("timed out waiting for shorten to complete, potential deadlock")
	}
}

// newPartlyIndexedHistories writes n state histories into a new freezer and
// indexes the first indexed of them.
func newPartlyIndexedHistories(t *testing.T, n, indexed int) (ethdb.Database, ethdb.ResettableAncientStore) {
	db := rawdb.NewMemoryDatabase()
	freezer, _ := rawdb.NewStateFreezer(t.TempDir(), false)
	t.Cleanup(func() { freezer.Close() })

	histories := makeStateHistories(n)
	for i, h := range histories {
		accountData, storageData, accountIndex, storageIndex := h.encode()
		rawdb.WriteStateHistory(freezer, uint64(i+1), h.meta.encode(), accountIndex, storageIndex, accountData, storageData)
	}
	b := newBatchIndexer(db, false, typeStateHistory)
	for i, h := range histories[:indexed] {
		if err := b.process(h, uint64(i+1)); err != nil {
			t.Fatalf("Failed to index history %d: %v", i+1, err)
		}
	}
	if err := b.finish(true); err != nil {
		t.Fatalf("Failed to index histories: %v", err)
	}
	return db, freezer
}

// shortenOrHang calls shorten, failing the test if it doesn't return.
func shortenOrHang(t *testing.T, indexer *historyIndexer, id uint64) error {
	result := make(chan error, 1)
	go func() { result <- indexer.shorten(id) }()
	select {
	case err := <-result:
		return err
	case <-time.After(5 * time.Second):
		t.Fatalf("Shorten of history %d is blocked", id)
		return nil
	}
}

// TestHistoryIndexerShortenAcrossIndexed tests a rollback below the indexed
// position while the initial indexing is unfinished. The histories above the
// position have nothing to unindex, and the ones below must be unindexed.
func TestHistoryIndexerShortenAcrossIndexed(t *testing.T) {
	db, freezer := newPartlyIndexedHistories(t, 20, 10)

	// The initer stays in the syncing state, leaving histories 11-20 unindexed
	indexer := newHistoryIndexer(db, freezer, 20, typeStateHistory, false)
	defer indexer.close()

	for id := uint64(20); id > 5; id-- {
		if err := shortenOrHang(t, indexer, id); err != nil {
			t.Errorf("Failed to shorten history %d: %v", id, err)
		}
	}
	if metadata := loadIndexMetadata(db, typeStateHistory); metadata == nil || metadata.Last != 5 {
		t.Fatalf("Unexpected index position after the rollback: %+v, want 5", metadata)
	}
}

// TestHistoryIndexerShortenFullyIndexed tests a rollback while all histories
// are indexed but the initer hasn't exited yet: the reverted history must be
// unindexed, and a failed unindexing must leave the indexer able to retry.
func TestHistoryIndexerShortenFullyIndexed(t *testing.T) {
	db, freezer := newPartlyIndexedHistories(t, 10, 10)

	// The initer stays in the syncing state and doesn't exit
	interrupted := true
	hdb := &hookedBatchDB{Database: db, hook: func() error {
		if interrupted {
			return errInterrupted
		}
		return nil
	}}
	indexer := newHistoryIndexer(hdb, freezer, 10, typeStateHistory, false)
	defer indexer.close()

	if err := shortenOrHang(t, indexer, 10); err == nil {
		t.Fatal("Shorten of history 10 succeeded without writing")
	}
	interrupted = false
	if err := shortenOrHang(t, indexer, 10); err != nil {
		t.Fatalf("Failed to shorten history 10: %v", err)
	}
	if metadata := loadIndexMetadata(db, typeStateHistory); metadata == nil || metadata.Last != 9 {
		t.Fatalf("Unexpected index position after the rollback: %+v, want 9", metadata)
	}
}

// TestHistoryIndexerShortenDuringIndexing tests a rollback that interrupts the
// initial indexing while it writes the index of the reverted history.
func TestHistoryIndexerShortenDuringIndexing(t *testing.T) {
	db, freezer := newPartlyIndexedHistories(t, 10, 5)

	synctest.Test(t, func(t *testing.T) {
		// Hold batch writes until the rollback interrupts the indexing
		writing, release := make(chan struct{}, 1), make(chan struct{})
		hdb := &hookedBatchDB{Database: db, hook: func() error {
			select {
			case writing <- struct{}{}:
			default:
			}
			<-release
			return nil
		}}
		indexer := newHistoryIndexer(hdb, freezer, 10, typeStateHistory, true)
		defer indexer.close()
		releaseWrites := sync.OnceFunc(func() { close(release) })
		defer releaseWrites()

		select {
		case <-writing:
		case <-time.After(time.Minute):
			t.Fatal("The indexing hasn't started")
		}
		// Send the rollback as shorten does, and let the initer reach its wait
		// for the interrupted indexing before releasing the write
		signal := &interruptSignal{newLastID: 9, result: make(chan error, 1)}
		select {
		case indexer.initer.interrupt <- signal:
		case <-time.After(time.Minute):
			t.Fatal("The rollback wasn't received")
		}
		synctest.Wait()
		releaseWrites()

		select {
		case err := <-signal.result:
			if err != nil {
				t.Fatalf("Failed to shorten history 10: %v", err)
			}
		case <-time.After(time.Minute):
			t.Fatal("Shorten of history 10 is blocked")
		}
		if metadata := loadIndexMetadata(db, typeStateHistory); metadata == nil || metadata.Last != 9 {
			t.Fatalf("Unexpected index position after the rollback: %+v, want 9", metadata)
		}
	})
}

var errInterrupted = errors.New("interrupted")

// hookedBatchDB calls hook before each batch write. An error from the hook
// fails the write, as if the process stopped there.
type hookedBatchDB struct {
	ethdb.Database
	hook func() error
}

type hookedBatch struct {
	ethdb.Batch
	hook func() error
}

func (db *hookedBatchDB) NewBatch() ethdb.Batch {
	return &hookedBatch{Batch: db.Database.NewBatch(), hook: db.hook}
}

func (db *hookedBatchDB) NewBatchWithSize(size int) ethdb.Batch {
	return &hookedBatch{Batch: db.Database.NewBatchWithSize(size), hook: db.hook}
}

func (b *hookedBatch) Write() error {
	if err := b.hook(); err != nil {
		return err
	}
	return b.Batch.Write()
}

// TestUnindexRetryAfterInterruption tests that an unindexing stopped at a batch
// write either has completed or can be retried.
func TestUnindexRetryAfterInterruption(t *testing.T) {
	db := rawdb.NewMemoryDatabase()
	freezer, _ := rawdb.NewStateFreezer(t.TempDir(), false)
	defer freezer.Close()

	// A history whose unindexing exceeds ethdb.IdealBatchSize
	accounts, storages := randomStateSet(500)
	h := newStateHistory(testrand.Hash(), types.EmptyRootHash, 1, accounts, storages, false)
	accountData, storageData, accountIndex, storageIndex := h.encode()
	rawdb.WriteStateHistory(freezer, 1, h.meta.encode(), accountIndex, storageIndex, accountData, storageData)

	b := newBatchIndexer(db, false, typeStateHistory)
	if err := b.process(h, 1); err != nil {
		t.Fatalf("Failed to index history: %v", err)
	}
	if err := b.finish(true); err != nil {
		t.Fatalf("Failed to index history: %v", err)
	}
	// Allow one batch write, and retry the unindexing if it failed
	allowed := 1
	hdb := &hookedBatchDB{Database: db, hook: func() error {
		if allowed == 0 {
			return errInterrupted
		}
		allowed--
		return nil
	}}
	if err := unindexSingle(1, hdb, freezer, typeStateHistory); err != nil {
		if err := unindexSingle(1, db, freezer, typeStateHistory); err != nil {
			t.Fatalf("Failed to retry the unindexing: %v", err)
		}
	}
	if metadata := loadIndexMetadata(db, typeStateHistory); metadata != nil {
		t.Fatalf("Unexpected index position: %d", metadata.Last)
	}
	it := db.NewIterator(rawdb.StateHistoryIndexPrefix, nil)
	defer it.Release()
	if it.Next() {
		t.Fatalf("Leftover history index data: %x", it.Key())
	}
}
