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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/

package pathdb

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// The batch size for reading state histories
const historyReadBatch = 1000

// batchIndexer is a structure designed to perform batch indexing or unindexing
// of state histories atomically.
type batchIndexer struct {
	accounts map[common.Address][]uint64                 // History ID list, Keyed by account address
	storages map[common.Address]map[common.Hash][]uint64 // History ID list, Keyed by account address and the hash of raw storage key
	counter  int                                         // The counter of processed states
	delete   bool                                        // Index or unindex mode
	lastID   uint64                                      // The ID of latest processed history
	db       ethdb.KeyValueStore
}

// newBatchIndexer constructs the batch indexer with the supplied mode.
func newBatchIndexer(db ethdb.KeyValueStore, delete bool) *batchIndexer {
	return &batchIndexer{
		accounts: make(map[common.Address][]uint64),
		storages: make(map[common.Address]map[common.Hash][]uint64),
		delete:   delete,
		db:       db,
	}
}

// process iterates through the accounts and their associated storage slots in the
// state history, tracking the mapping between state and history IDs.
func (b *batchIndexer) process(h *history, historyID uint64) error {
	for _, address := range h.accountList {
		b.counter += 1
		b.accounts[address] = append(b.accounts[address], historyID)

		for _, slotKey := range h.storageList[address] {
			b.counter += 1
			if _, ok := b.storages[address]; !ok {
				b.storages[address] = make(map[common.Hash][]uint64)
			}
			// The hash of the storage slot key is used as the identifier because the
			// legacy history does not include the raw storage key, therefore, the
			// conversion from storage key to hash is necessary for non-v0 histories.
			slotHash := slotKey
			if h.meta.version != stateHistoryV0 {
				slotHash = crypto.Keccak256Hash(slotKey.Bytes())
			}
			b.storages[address][slotHash] = append(b.storages[address][slotHash], historyID)
		}
	}
	b.lastID = historyID
	return b.finish(false)
}

// finish writes the accumulated state indexes into the disk if either the
// memory limitation is reached or it's requested forcibly.
func (b *batchIndexer) finish(force bool) error {
	if b.counter == 0 {
		return nil
	}
	if !force && b.counter < historyIndexBatch {
		return nil
	}
	batch := b.db.NewBatch()
	for address, idList := range b.accounts {
		if !b.delete {
			iw, err := newIndexWriter(b.db, newAccountIdent(address))
			if err != nil {
				return err
			}
			for _, n := range idList {
				if err := iw.append(n); err != nil {
					return err
				}
			}
			iw.finish(batch)
		} else {
			id, err := newIndexDeleter(b.db, newAccountIdent(address))
			if err != nil {
				return err
			}
			for _, n := range idList {
				if err := id.pop(n); err != nil {
					return err
				}
			}
			id.finish(batch)
		}
	}
	for address, slots := range b.storages {
		for storageHash, idList := range slots {
			if !b.delete {
				iw, err := newIndexWriter(b.db, newStorageIdent(address, storageHash))
				if err != nil {
					return err
				}
				for _, n := range idList {
					if err := iw.append(n); err != nil {
						return err
					}
				}
				iw.finish(batch)
			} else {
				id, err := newIndexDeleter(b.db, newStorageIdent(address, storageHash))
				if err != nil {
					return err
				}
				for _, n := range idList {
					if err := id.pop(n); err != nil {
						return err
					}
				}
				id.finish(batch)
			}
		}
	}
	// Update the position of last indexed state history
	if !b.delete {
		rawdb.WriteLastStateHistoryIndex(batch, b.lastID)
	} else {
		if b.lastID == 1 {
			rawdb.DeleteLastStateHistoryIndex(batch)
		} else {
			rawdb.WriteLastStateHistoryIndex(batch, b.lastID-1)
		}
	}
	if err := batch.Write(); err != nil {
		return err
	}
	b.counter = 0
	b.accounts = make(map[common.Address][]uint64)
	b.storages = make(map[common.Address]map[common.Hash][]uint64)
	return nil
}

// indexSingle processes the state history with the specified ID for indexing.
func indexSingle(historyID uint64, db ethdb.KeyValueStore, freezer ethdb.AncientReader) error {
	defer func(start time.Time) {
		indexHistoryTimer.UpdateSince(start)
	}(time.Now())

	indexed := rawdb.ReadLastStateHistoryIndex(db)
	if indexed == nil || *indexed+1 != historyID {
		last := "null"
		if indexed != nil {
			last = fmt.Sprintf("%v", *indexed)
		}
		return fmt.Errorf("history indexing is out of order, last: %s, requested: %d", last, historyID)
	}
	h, err := readHistory(freezer, historyID)
	if err != nil {
		return err
	}
	b := newBatchIndexer(db, false)
	if err := b.process(h, historyID); err != nil {
		return err
	}
	if err := b.finish(true); err != nil {
		return err
	}
	log.Debug("Indexed state history", "id", historyID)
	return nil
}

// unindexSingle processes the state history with the specified ID for unindexing.
func unindexSingle(historyID uint64, db ethdb.KeyValueStore, freezer ethdb.AncientReader) error {
	defer func(start time.Time) {
		unindexHistoryTimer.UpdateSince(start)
	}(time.Now())

	indexed := rawdb.ReadLastStateHistoryIndex(db)
	if indexed == nil || *indexed != historyID {
		last := "null"
		if indexed != nil {
			last = fmt.Sprintf("%v", *indexed)
		}
		return fmt.Errorf("history unindexing is out of order, last: %s, requested: %d", last, historyID)
	}
	h, err := readHistory(freezer, historyID)
	if err != nil {
		return err
	}
	b := newBatchIndexer(db, true)
	if err := b.process(h, historyID); err != nil {
		return err
	}
	if err := b.finish(true); err != nil {
		return err
	}
	log.Debug("Unindexed state history", "id", historyID)
	return nil
}

type interruptSignal struct {
	newLastID uint64
	result    chan error
}

// indexIniter is responsible for completing the indexing of remaining state
// histories in batch. It runs as a one-time background thread and terminates
// once all available state histories are indexed.
//
// Afterward, new state histories should be indexed synchronously alongside
// the state data itself, ensuring both the history and its index are available.
// If a state history is removed due to a rollback, the associated indexes should
// be unmarked accordingly.
type indexIniter struct {
	disk      ethdb.KeyValueStore
	freezer   ethdb.AncientStore
	interrupt chan *interruptSignal
	done      chan struct{}
	closed    chan struct{}
	wg        sync.WaitGroup
}

func newIndexIniter(disk ethdb.KeyValueStore, freezer ethdb.AncientStore, lastID uint64) *indexIniter {
	initer := &indexIniter{
		disk:      disk,
		freezer:   freezer,
		interrupt: make(chan *interruptSignal),
		done:      make(chan struct{}),
		closed:    make(chan struct{}),
	}
	initer.wg.Add(1)
	go initer.run(lastID)
	return initer
}

func (i *indexIniter) close() {
	select {
	case <-i.closed:
		return
	default:
		close(i.closed)
		i.wg.Wait()
	}
}

func (i *indexIniter) inited() bool {
	select {
	case <-i.closed:
		return false
	case <-i.done:
		return true
	default:
		return false
	}
}

func (i *indexIniter) run(lastID uint64) {
	defer i.wg.Done()

	// Launch background indexing thread
	var (
		done      = make(chan struct{})
		interrupt = new(atomic.Int32)

		// checkDone indicates whether all requested state histories
		// have been fully indexed.
		checkDone = func() bool {
			indexed := rawdb.ReadLastStateHistoryIndex(i.disk)
			return indexed != nil && *indexed == lastID
		}
	)
	go i.index(done, interrupt, lastID)

	for {
		select {
		case signal := <-i.interrupt:
			// The indexing limit can only be extended or shortened continuously.
			if signal.newLastID != lastID+1 && signal.newLastID != lastID-1 {
				signal.result <- fmt.Errorf("invalid history id, last: %d, got: %d", lastID, signal.newLastID)
				continue
			}
			// The index limit is extended by one, update the limit without
			// interrupting the current background process.
			if signal.newLastID == lastID+1 {
				lastID = signal.newLastID
				signal.result <- nil
				continue
			}
			// The index limit is shortened by one, interrupt the current background
			// process and relaunch with new target.
			interrupt.Store(1)
			<-done

			// If all state histories, including the one to be reverted, have
			// been fully indexed, unindex it here and shut down the initializer.
			if checkDone() {
				if err := unindexSingle(lastID, i.disk, i.freezer); err != nil {
					signal.result <- err
					return
				}
				close(i.done)
				signal.result <- nil
				return
			}
			// Adjust the indexing target and relaunch the process
			lastID = signal.newLastID
			done, interrupt = make(chan struct{}), new(atomic.Int32)
			go i.index(done, interrupt, lastID)

		case <-done:
			if checkDone() {
				close(i.done)
				return
			}
			// Relaunch the background runner if some tasks are left
			done, interrupt = make(chan struct{}), new(atomic.Int32)
			go i.index(done, interrupt, lastID)

		case <-i.closed:
			interrupt.Store(1)
			log.Info("Waiting background history index initer to exit")
			<-done

			if checkDone() {
				close(i.done)
			}
			return
		}
	}
}

// next returns the ID of the next state history to be indexed.
func (i *indexIniter) next() (uint64, error) {
	tail, err := i.freezer.Tail()
	if err != nil {
		return 0, err
	}
	tailID := tail + 1 // compute the id of the oldest history

	// Start indexing from scratch if nothing has been indexed
	lastIndexed := rawdb.ReadLastStateHistoryIndex(i.disk)
	if lastIndexed == nil {
		return tailID, nil
	}
	// Resume indexing from the last interrupted position
	if *lastIndexed+1 >= tailID {
		return *lastIndexed + 1, nil
	}
	// History has been shortened without indexing. Discard the gapped segment
	// in the history and shift to the first available element.
	//
	// The missing indexes corresponding to the gapped histories won't be visible.
	// It's fine to leave them unindexed.
	log.Info("History gap detected, discard old segment", "oldHead", *lastIndexed, "newHead", tailID)
	return tailID, nil
}

func (i *indexIniter) index(done chan struct{}, interrupt *atomic.Int32, lastID uint64) {
	defer close(done)

	beginID, err := i.next()
	if err != nil {
		log.Error("Failed to find next state history for indexing", "err", err)
		return
	}
	// All available state histories have been indexed, and the last indexed one
	// exceeds the most recent available state history. This situation may occur
	// when the state is reverted manually (chain.SetHead) or the deep reorg is
	// encountered. In such cases, no indexing should be scheduled.
	if beginID > lastID {
		return
	}
	log.Info("Start history indexing", "beginID", beginID, "lastID", lastID)

	var (
		current = beginID
		start   = time.Now()
		logged  = time.Now()
		batch   = newBatchIndexer(i.disk, false)
	)
	for current <= lastID {
		count := lastID - current + 1
		if count > historyReadBatch {
			count = historyReadBatch
		}
		histories, err := readHistories(i.freezer, current, count)
		if err != nil {
			// The history read might fall if the history is truncated from
			// head due to revert operation.
			log.Error("Failed to read history for indexing", "current", current, "count", count, "err", err)
			return
		}
		for _, h := range histories {
			if err := batch.process(h, current); err != nil {
				log.Error("Failed to index history", "err", err)
				return
			}
			current += 1

			// Occasionally report the indexing progress
			if time.Since(logged) > time.Second*8 {
				logged = time.Now()

				var (
					left  = lastID - current + 1
					done  = current - beginID
					speed = done/uint64(time.Since(start)/time.Millisecond+1) + 1 // +1s to avoid division by zero
				)
				// Override the ETA if larger than the largest until now
				eta := time.Duration(left/speed) * time.Millisecond
				log.Info("Indexing state history", "processed", done, "left", left, "eta", common.PrettyDuration(eta))
			}
		}
		// Check interruption signal and abort process if it's fired
		if interrupt != nil {
			if signal := interrupt.Load(); signal != 0 {
				if err := batch.finish(true); err != nil {
					log.Error("Failed to flush index", "err", err)
				}
				log.Info("State indexing interrupted")
				return
			}
		}
	}
	if err := batch.finish(true); err != nil {
		log.Error("Failed to flush index", "err", err)
	}
	log.Info("Indexed state history", "from", beginID, "to", lastID, "elapsed", common.PrettyDuration(time.Since(start)))
}

// historyIndexer manages the indexing and unindexing of state histories,
// providing access to historical states.
//
// Upon initialization, historyIndexer starts a one-time background process
// to complete the indexing of any remaining state histories. Once this
// process is finished, all state histories are marked as fully indexed,
// enabling handling of requests for historical states. Thereafter, any new
// state histories must be indexed or unindexed synchronously, ensuring that
// the history index is created or removed along with the corresponding
// state history.
type historyIndexer struct {
	initer  *indexIniter
	disk    ethdb.KeyValueStore
	freezer ethdb.AncientStore
}

// newHistoryIndexer constructs the history indexer and launches the background
// initer to complete the indexing of any remaining state histories.
func newHistoryIndexer(disk ethdb.KeyValueStore, freezer ethdb.AncientStore, lastHistoryID uint64) *historyIndexer {
	return &historyIndexer{
		initer:  newIndexIniter(disk, freezer, lastHistoryID),
		disk:    disk,
		freezer: freezer,
	}
}

func (i *historyIndexer) close() {
	i.initer.close()
}

func (i *historyIndexer) inited() bool {
	return i.initer.inited()
}

// extend sends the notification that new state history with specified ID
// has been written into the database and is ready for indexing.
func (i *historyIndexer) extend(historyID uint64) error {
	signal := &interruptSignal{
		newLastID: historyID,
		result:    make(chan error, 1),
	}
	select {
	case <-i.initer.closed:
		return errors.New("indexer is closed")
	case <-i.initer.done:
		return indexSingle(historyID, i.disk, i.freezer)
	case i.initer.interrupt <- signal:
		return <-signal.result
	}
}

// shorten sends the notification that state history with specified ID
// is about to be deleted from the database and should be unindexed.
func (i *historyIndexer) shorten(historyID uint64) error {
	signal := &interruptSignal{
		newLastID: historyID - 1,
		result:    make(chan error, 1),
	}
	select {
	case <-i.initer.closed:
		return errors.New("indexer is closed")
	case <-i.initer.done:
		return unindexSingle(historyID, i.disk, i.freezer)
	case i.initer.interrupt <- signal:
		return <-signal.result
	}
}
