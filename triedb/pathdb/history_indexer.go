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
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/sync/errgroup"
)

const (
	// The batch size for reading state histories
	historyReadBatch = 1000

	stateHistoryIndexV0         = uint8(0)               // initial version of state index structure
	stateHistoryIndexVersion    = stateHistoryIndexV0    // the current state index version
	trienodeHistoryIndexV0      = uint8(0)               // initial version of trienode index structure
	trienodeHistoryIndexVersion = trienodeHistoryIndexV0 // the current trienode index version

	// estimations for calculating the batch size for atomic database commit
	estimatedStateHistoryIndexSize    = 3  // The average size of each state history index entry is approximately 2–3 bytes
	estimatedTrienodeHistoryIndexSize = 3  // The average size of each trienode history index entry is approximately 2-3 bytes
	estimatedIndexBatchSizeFactor     = 32 // The factor counts for the write amplification for each entry
)

// indexVersion returns the latest index version for the given history type.
// It panics if the history type is unknown.
func indexVersion(typ historyType) uint8 {
	switch typ {
	case typeStateHistory:
		return stateHistoryIndexVersion
	case typeTrienodeHistory:
		return trienodeHistoryIndexVersion
	default:
		panic(fmt.Errorf("unknown history type: %d", typ))
	}
}

// indexMetadata describes the metadata of the historical data index.
type indexMetadata struct {
	Version uint8
	Last    uint64
}

// loadIndexMetadata reads the metadata of the specific history index.
func loadIndexMetadata(db ethdb.KeyValueReader, typ historyType) *indexMetadata {
	var blob []byte
	switch typ {
	case typeStateHistory:
		blob = rawdb.ReadStateHistoryIndexMetadata(db)
	case typeTrienodeHistory:
		blob = rawdb.ReadTrienodeHistoryIndexMetadata(db)
	default:
		panic(fmt.Errorf("unknown history type %d", typ))
	}
	if len(blob) == 0 {
		return nil
	}
	var m indexMetadata
	if err := rlp.DecodeBytes(blob, &m); err != nil {
		log.Error("Failed to decode index metadata", "err", err)
		return nil
	}
	return &m
}

// storeIndexMetadata stores the metadata of the specific history index.
func storeIndexMetadata(db ethdb.KeyValueWriter, typ historyType, last uint64) {
	m := indexMetadata{
		Version: indexVersion(typ),
		Last:    last,
	}
	blob, err := rlp.EncodeToBytes(m)
	if err != nil {
		panic(fmt.Errorf("fail to encode index metadata, %v", err))
	}
	switch typ {
	case typeStateHistory:
		rawdb.WriteStateHistoryIndexMetadata(db, blob)
	case typeTrienodeHistory:
		rawdb.WriteTrienodeHistoryIndexMetadata(db, blob)
	default:
		panic(fmt.Errorf("unknown history type %d", typ))
	}
	log.Debug("Written index metadata", "type", typ, "last", last)
}

// deleteIndexMetadata deletes the metadata of the specific history index.
func deleteIndexMetadata(db ethdb.KeyValueWriter, typ historyType) {
	switch typ {
	case typeStateHistory:
		rawdb.DeleteStateHistoryIndexMetadata(db)
	case typeTrienodeHistory:
		rawdb.DeleteTrienodeHistoryIndexMetadata(db)
	default:
		panic(fmt.Errorf("unknown history type %d", typ))
	}
	log.Debug("Deleted index metadata", "type", typ)
}

// batchIndexer is responsible for performing batch indexing or unindexing
// of historical data (e.g., state or trie node changes) atomically.
type batchIndexer struct {
	index   map[stateIdent][]uint64 // List of history IDs for tracked state entry
	pending int                     // Number of entries processed in the current batch.
	delete  bool                    // Operation mode: true for unindex, false for index.
	lastID  uint64                  // ID of the most recently processed history.
	typ     historyType             // Type of history being processed (e.g., state or trienode).
	db      ethdb.KeyValueStore     // Key-value database used to store or delete index data.
}

// newBatchIndexer constructs the batch indexer with the supplied mode.
func newBatchIndexer(db ethdb.KeyValueStore, delete bool, typ historyType) *batchIndexer {
	return &batchIndexer{
		index:  make(map[stateIdent][]uint64),
		delete: delete,
		typ:    typ,
		db:     db,
	}
}

// process traverses the state entries within the provided history and tracks the mutation
// records for them.
func (b *batchIndexer) process(h history, id uint64) error {
	for ident := range h.forEach() {
		b.index[ident] = append(b.index[ident], id)
		b.pending++
	}
	b.lastID = id

	return b.finish(false)
}

// makeBatch constructs a database batch based on the number of pending entries.
// The batch size is roughly estimated to minimize repeated resizing rounds,
// as accurately predicting the exact size is technically challenging.
func (b *batchIndexer) makeBatch() ethdb.Batch {
	var size int
	switch b.typ {
	case typeStateHistory:
		size = estimatedStateHistoryIndexSize
	case typeTrienodeHistory:
		size = estimatedTrienodeHistoryIndexSize
	default:
		panic(fmt.Sprintf("unknown history type %d", b.typ))
	}
	return b.db.NewBatchWithSize(size * estimatedIndexBatchSizeFactor * b.pending)
}

// finish writes the accumulated state indexes into the disk if either the
// memory limitation is reached or it's requested forcibly.
func (b *batchIndexer) finish(force bool) error {
	if b.pending == 0 {
		return nil
	}
	if !force && b.pending < historyIndexBatch {
		return nil
	}
	var (
		batch   = b.makeBatch()
		batchMu sync.RWMutex
		start   = time.Now()
		eg      errgroup.Group
	)
	eg.SetLimit(runtime.NumCPU())

	for ident, list := range b.index {
		eg.Go(func() error {
			if !b.delete {
				iw, err := newIndexWriter(b.db, ident)
				if err != nil {
					return err
				}
				for _, n := range list {
					if err := iw.append(n); err != nil {
						return err
					}
				}
				batchMu.Lock()
				iw.finish(batch)
				batchMu.Unlock()
			} else {
				id, err := newIndexDeleter(b.db, ident)
				if err != nil {
					return err
				}
				for _, n := range list {
					if err := id.pop(n); err != nil {
						return err
					}
				}
				batchMu.Lock()
				id.finish(batch)
				batchMu.Unlock()
			}
			return nil
		})
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	// Update the position of last indexed state history
	if !b.delete {
		storeIndexMetadata(batch, b.typ, b.lastID)
	} else {
		if b.lastID == 1 {
			deleteIndexMetadata(batch, b.typ)
		} else {
			storeIndexMetadata(batch, b.typ, b.lastID-1)
		}
	}
	if err := batch.Write(); err != nil {
		return err
	}
	log.Debug("Committed batch indexer", "type", b.typ, "entries", len(b.index), "records", b.pending, "elapsed", common.PrettyDuration(time.Since(start)))
	b.pending = 0
	b.index = make(map[stateIdent][]uint64)
	return nil
}

// indexSingle processes the state history with the specified ID for indexing.
func indexSingle(historyID uint64, db ethdb.KeyValueStore, freezer ethdb.AncientReader, typ historyType) error {
	start := time.Now()
	defer func() {
		if typ == typeStateHistory {
			stateIndexHistoryTimer.UpdateSince(start)
		} else if typ == typeTrienodeHistory {
			trienodeIndexHistoryTimer.UpdateSince(start)
		}
	}()

	metadata := loadIndexMetadata(db, typ)
	if metadata == nil || metadata.Last+1 != historyID {
		last := "null"
		if metadata != nil {
			last = fmt.Sprintf("%v", metadata.Last)
		}
		return fmt.Errorf("history indexing is out of order, last: %s, requested: %d", last, historyID)
	}
	var (
		err error
		h   history
		b   = newBatchIndexer(db, false, typ)
	)
	if typ == typeStateHistory {
		h, err = readStateHistory(freezer, historyID)
	} else {
		h, err = readTrienodeHistory(freezer, historyID)
	}
	if err != nil {
		return err
	}
	if err := b.process(h, historyID); err != nil {
		return err
	}
	if err := b.finish(true); err != nil {
		return err
	}
	log.Debug("Indexed history", "type", typ, "id", historyID, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// unindexSingle processes the state history with the specified ID for unindexing.
func unindexSingle(historyID uint64, db ethdb.KeyValueStore, freezer ethdb.AncientReader, typ historyType) error {
	start := time.Now()
	defer func() {
		if typ == typeStateHistory {
			stateUnindexHistoryTimer.UpdateSince(start)
		} else if typ == typeTrienodeHistory {
			trienodeUnindexHistoryTimer.UpdateSince(start)
		}
	}()

	metadata := loadIndexMetadata(db, typ)
	if metadata == nil || metadata.Last != historyID {
		last := "null"
		if metadata != nil {
			last = fmt.Sprintf("%v", metadata.Last)
		}
		return fmt.Errorf("history unindexing is out of order, last: %s, requested: %d", last, historyID)
	}
	var (
		err error
		h   history
	)
	b := newBatchIndexer(db, true, typ)
	if typ == typeStateHistory {
		h, err = readStateHistory(freezer, historyID)
	} else {
		h, err = readTrienodeHistory(freezer, historyID)
	}
	if err != nil {
		return err
	}
	if err := b.process(h, historyID); err != nil {
		return err
	}
	if err := b.finish(true); err != nil {
		return err
	}
	log.Debug("Unindexed history", "type", typ, "id", historyID, "elapsed", common.PrettyDuration(time.Since(start)))
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
	typ       historyType
	log       log.Logger // Contextual logger with the history type injected

	// indexing progress
	indexed atomic.Uint64 // the id of latest indexed state
	last    atomic.Uint64 // the id of the target state to be indexed

	wg sync.WaitGroup
}

func newIndexIniter(disk ethdb.KeyValueStore, freezer ethdb.AncientStore, typ historyType, lastID uint64) *indexIniter {
	initer := &indexIniter{
		disk:      disk,
		freezer:   freezer,
		interrupt: make(chan *interruptSignal),
		done:      make(chan struct{}),
		closed:    make(chan struct{}),
		typ:       typ,
		log:       log.New("type", typ.String()),
	}
	// Load indexing progress
	var recover bool
	initer.last.Store(lastID)
	metadata := loadIndexMetadata(disk, typ)
	if metadata != nil {
		initer.indexed.Store(metadata.Last)
		recover = metadata.Last > lastID
	}

	// Launch background indexer
	initer.wg.Add(1)
	if recover {
		log.Info("History indexer is recovering", "history", lastID, "indexed", metadata.Last)
		go initer.recover(lastID)
	} else {
		go initer.run(lastID)
	}
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

func (i *indexIniter) remain() uint64 {
	select {
	case <-i.closed:
		return 0
	case <-i.done:
		return 0
	default:
		last, indexed := i.last.Load(), i.indexed.Load()
		if last < indexed {
			i.log.Warn("State indexer is in recovery", "indexed", indexed, "last", last)
			return indexed - last
		}
		return last - indexed
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
			metadata := loadIndexMetadata(i.disk, i.typ)
			return metadata != nil && metadata.Last == lastID
		}
	)
	go i.index(done, interrupt, lastID)

	for {
		select {
		case signal := <-i.interrupt:
			// The indexing limit can only be extended or shortened continuously.
			newLastID := signal.newLastID
			if newLastID != lastID+1 && newLastID != lastID-1 {
				signal.result <- fmt.Errorf("invalid history id, last: %d, got: %d", lastID, newLastID)
				continue
			}
			i.last.Store(newLastID) // update indexing range

			// The index limit is extended by one, update the limit without
			// interrupting the current background process.
			if newLastID == lastID+1 {
				lastID = newLastID
				signal.result <- nil
				i.log.Debug("Extended history range", "last", lastID)
				continue
			}
			// The index limit is shortened by one, interrupt the current background
			// process and relaunch with new target.
			interrupt.Store(1)
			<-done

			// If all state histories, including the one to be reverted, have
			// been fully indexed, unindex it here and shut down the initializer.
			if checkDone() {
				i.log.Info("Truncate the extra history", "id", lastID)
				if err := unindexSingle(lastID, i.disk, i.freezer, i.typ); err != nil {
					signal.result <- err
					return
				}
				close(i.done)
				signal.result <- nil
				i.log.Info("Histories have been fully indexed", "last", lastID-1)
				return
			}
			// Adjust the indexing target and relaunch the process
			lastID = newLastID
			signal.result <- nil

			done, interrupt = make(chan struct{}), new(atomic.Int32)
			go i.index(done, interrupt, lastID)
			i.log.Debug("Shortened history range", "last", lastID)

		case <-done:
			if checkDone() {
				close(i.done)
				i.log.Info("Histories have been fully indexed", "last", lastID)
				return
			}
			// Relaunch the background runner if some tasks are left
			done, interrupt = make(chan struct{}), new(atomic.Int32)
			go i.index(done, interrupt, lastID)

		case <-i.closed:
			interrupt.Store(1)
			i.log.Info("Waiting background history index initer to exit")
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
	metadata := loadIndexMetadata(i.disk, i.typ)
	if metadata == nil {
		i.log.Debug("Initialize history indexing from scratch", "id", tailID)
		return tailID, nil
	}
	// Resume indexing from the last interrupted position
	if metadata.Last+1 >= tailID {
		i.log.Debug("Resume history indexing", "id", metadata.Last+1, "tail", tailID)
		return metadata.Last + 1, nil
	}
	// History has been shortened without indexing. Discard the gapped segment
	// in the history and shift to the first available element.
	//
	// The missing indexes corresponding to the gapped histories won't be visible.
	// It's fine to leave them unindexed.
	i.log.Info("History gap detected, discard old segment", "oldHead", metadata.Last, "newHead", tailID)
	return tailID, nil
}

func (i *indexIniter) index(done chan struct{}, interrupt *atomic.Int32, lastID uint64) {
	defer close(done)

	beginID, err := i.next()
	if err != nil {
		i.log.Error("Failed to find next history for indexing", "err", err)
		return
	}
	// All available state histories have been indexed, and the last indexed one
	// exceeds the most recent available state history. This situation may occur
	// when the state is reverted manually (chain.SetHead) or the deep reorg is
	// encountered. In such cases, no indexing should be scheduled.
	if beginID > lastID {
		if lastID == 0 && beginID == 1 {
			// Initialize the indexing flag if the state history is empty by
			// using zero as the disk layer ID. This is a common case that
			// can occur after snap sync.
			//
			// This step is essential to avoid spinning up indexing thread
			// endlessly until a history object is produced.
			storeIndexMetadata(i.disk, i.typ, 0)
			i.log.Info("Initialized history indexing flag")
		} else {
			i.log.Debug("History is fully indexed", "last", lastID)
		}
		return
	}
	i.log.Info("Start history indexing", "beginID", beginID, "lastID", lastID)

	var (
		current = beginID
		start   = time.Now()
		logged  = time.Now()
		batch   = newBatchIndexer(i.disk, false, i.typ)
	)
	for current <= lastID {
		count := lastID - current + 1
		if count > historyReadBatch {
			count = historyReadBatch
		}
		var histories []history
		if i.typ == typeStateHistory {
			histories, err = readStateHistories(i.freezer, current, count)
			if err != nil {
				// The history read might fall if the history is truncated from
				// head due to revert operation.
				i.log.Error("Failed to read history for indexing", "current", current, "count", count, "err", err)
				return
			}
		} else {
			histories, err = readTrienodeHistories(i.freezer, current, count)
			if err != nil {
				// The history read might fall if the history is truncated from
				// head due to revert operation.
				i.log.Error("Failed to read history for indexing", "current", current, "count", count, "err", err)
				return
			}
		}
		for _, h := range histories {
			if err := batch.process(h, current); err != nil {
				i.log.Error("Failed to index history", "err", err)
				return
			}
			current += 1

			// Occasionally report the indexing progress
			if time.Since(logged) > time.Second*8 {
				logged = time.Now()

				var (
					left = lastID - current + 1
					done = current - beginID
				)
				eta := common.CalculateETA(done, left, time.Since(start))
				i.log.Info("Indexing history", "processed", done, "left", left, "elapsed", common.PrettyDuration(time.Since(start)), "eta", common.PrettyDuration(eta))
			}
		}
		i.indexed.Store(current - 1) // update indexing progress

		// Check interruption signal and abort process if it's fired
		if interrupt != nil {
			if signal := interrupt.Load(); signal != 0 {
				if err := batch.finish(true); err != nil {
					i.log.Error("Failed to flush index", "err", err)
				}
				log.Info("State indexing interrupted")
				return
			}
		}
	}
	if err := batch.finish(true); err != nil {
		i.log.Error("Failed to flush index", "err", err)
	}
	i.log.Info("Indexed history", "from", beginID, "to", lastID, "elapsed", common.PrettyDuration(time.Since(start)))
}

// recover handles unclean shutdown recovery. After an unclean shutdown, any
// extra histories are typically truncated, while the corresponding history index
// entries may still have been written. Ideally, we would unindex these histories
// in reverse order, but there is no guarantee that the required histories will
// still be available.
//
// As a workaround, indexIniter waits until the missing histories are regenerated
// by chain recovery, under the assumption that the recovered histories will be
// identical to the lost ones. Fork-awareness should be added in the future to
// correctly handle histories affected by reorgs.
func (i *indexIniter) recover(lastID uint64) {
	defer i.wg.Done()

	for {
		select {
		case signal := <-i.interrupt:
			newLastID := signal.newLastID
			if newLastID != lastID+1 && newLastID != lastID-1 {
				signal.result <- fmt.Errorf("invalid history id, last: %d, got: %d", lastID, newLastID)
				continue
			}

			// Update the last indexed flag
			lastID = newLastID
			signal.result <- nil
			i.last.Store(newLastID)
			i.log.Debug("Updated history index flag", "last", lastID)

			// Terminate the recovery routine once the histories are fully aligned
			// with the index data, indicating that index initialization is complete.
			metadata := loadIndexMetadata(i.disk, i.typ)
			if metadata != nil && metadata.Last == lastID {
				close(i.done)
				i.log.Info("History indexer is recovered", "last", lastID)
				return
			}

		case <-i.closed:
			return
		}
	}
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
	typ     historyType
	disk    ethdb.KeyValueStore
	freezer ethdb.AncientStore
}

// checkVersion checks whether the index data in the database matches the version.
func checkVersion(disk ethdb.KeyValueStore, typ historyType) {
	var blob []byte
	if typ == typeStateHistory {
		blob = rawdb.ReadStateHistoryIndexMetadata(disk)
	} else if typ == typeTrienodeHistory {
		blob = rawdb.ReadTrienodeHistoryIndexMetadata(disk)
	} else {
		panic(fmt.Errorf("unknown history type: %v", typ))
	}
	// Short circuit if metadata is not found, re-index is required
	// from scratch.
	if len(blob) == 0 {
		return
	}
	// Short circuit if the metadata is found and the version is matched
	ver := stateHistoryIndexVersion
	if typ == typeTrienodeHistory {
		ver = trienodeHistoryIndexVersion
	}
	var m indexMetadata
	err := rlp.DecodeBytes(blob, &m)
	if err == nil && m.Version == ver {
		return
	}
	// Version is not matched, prune the existing data and re-index from scratch
	batch := disk.NewBatch()
	if typ == typeStateHistory {
		rawdb.DeleteStateHistoryIndexMetadata(batch)
		rawdb.DeleteStateHistoryIndexes(batch)
	} else {
		rawdb.DeleteTrienodeHistoryIndexMetadata(batch)
		rawdb.DeleteTrienodeHistoryIndexes(batch)
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to purge history index", "type", typ, "err", err)
	}
	version := "unknown"
	if err == nil {
		version = fmt.Sprintf("%d", m.Version)
	}
	log.Info("Cleaned up obsolete history index", "type", typ, "version", version, "want", version)
}

// newHistoryIndexer constructs the history indexer and launches the background
// initer to complete the indexing of any remaining state histories.
func newHistoryIndexer(disk ethdb.KeyValueStore, freezer ethdb.AncientStore, lastHistoryID uint64, typ historyType) *historyIndexer {
	checkVersion(disk, typ)
	return &historyIndexer{
		initer:  newIndexIniter(disk, freezer, typ, lastHistoryID),
		typ:     typ,
		disk:    disk,
		freezer: freezer,
	}
}

func (i *historyIndexer) close() {
	i.initer.close()
}

// inited returns a flag indicating whether the existing state histories
// have been fully indexed, in other words, whether they are available
// for external access.
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
		return indexSingle(historyID, i.disk, i.freezer, i.typ)
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
		return unindexSingle(historyID, i.disk, i.freezer, i.typ)
	case i.initer.interrupt <- signal:
		return <-signal.result
	}
}

// progress returns the indexing progress made so far. It provides the number
// of states that remain unindexed.
func (i *historyIndexer) progress() (uint64, error) {
	select {
	case <-i.initer.closed:
		return 0, errors.New("indexer is closed")
	default:
		return i.initer.remain(), nil
	}
}
