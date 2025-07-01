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
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/sync/errgroup"
)

const (
	// The batch size for reading state histories
	historyReadBatch = 1000

	stateIndexV0      = uint8(0)     // initial version of state index structure
	stateIndexVersion = stateIndexV0 // the current state index version
)

type indexMetadata struct {
	Version uint8
	Last    uint64
}

func loadIndexMetadata(db ethdb.KeyValueReader) *indexMetadata {
	blob := rawdb.ReadStateHistoryIndexMetadata(db)
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

func storeIndexMetadata(db ethdb.KeyValueWriter, last uint64) {
	var m indexMetadata
	m.Version = stateIndexVersion
	m.Last = last
	blob, err := rlp.EncodeToBytes(m)
	if err != nil {
		log.Crit("Failed to encode index metadata", "err", err)
	}
	rawdb.WriteStateHistoryIndexMetadata(db, blob)
}

// batchIndexer is a structure designed to perform batch indexing or unindexing
// of state histories atomically.
type batchIndexer struct {
	accounts map[common.Hash][]uint64                 // History ID list, Keyed by the hash of account address
	storages map[common.Hash]map[common.Hash][]uint64 // History ID list, Keyed by the hash of account address and the hash of raw storage key
	counter  int                                      // The counter of processed states
	delete   bool                                     // Index or unindex mode
	lastID   uint64                                   // The ID of latest processed history
	db       ethdb.KeyValueStore
}

// newBatchIndexer constructs the batch indexer with the supplied mode.
func newBatchIndexer(db ethdb.KeyValueStore, delete bool) *batchIndexer {
	return &batchIndexer{
		accounts: make(map[common.Hash][]uint64),
		storages: make(map[common.Hash]map[common.Hash][]uint64),
		delete:   delete,
		db:       db,
	}
}

// process iterates through the accounts and their associated storage slots in the
// state history, tracking the mapping between state and history IDs.
func (b *batchIndexer) process(h *history, historyID uint64) error {
	for _, address := range h.accountList {
		addrHash := crypto.Keccak256Hash(address.Bytes())
		b.counter += 1
		b.accounts[addrHash] = append(b.accounts[addrHash], historyID)

		for _, slotKey := range h.storageList[address] {
			b.counter += 1
			if _, ok := b.storages[addrHash]; !ok {
				b.storages[addrHash] = make(map[common.Hash][]uint64)
			}
			// The hash of the storage slot key is used as the identifier because the
			// legacy history does not include the raw storage key, therefore, the
			// conversion from storage key to hash is necessary for non-v0 histories.
			slotHash := slotKey
			if h.meta.version != stateHistoryV0 {
				slotHash = crypto.Keccak256Hash(slotKey.Bytes())
			}
			b.storages[addrHash][slotHash] = append(b.storages[addrHash][slotHash], historyID)
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
	var (
		batch    = b.db.NewBatch()
		batchMu  sync.RWMutex
		storages int
		start    = time.Now()
		eg       errgroup.Group
	)
	eg.SetLimit(runtime.NumCPU())

	for addrHash, idList := range b.accounts {
		eg.Go(func() error {
			if !b.delete {
				iw, err := newIndexWriter(b.db, newAccountIdent(addrHash))
				if err != nil {
					return err
				}
				for _, n := range idList {
					if err := iw.append(n); err != nil {
						return err
					}
				}
				batchMu.Lock()
				iw.finish(batch)
				batchMu.Unlock()
			} else {
				id, err := newIndexDeleter(b.db, newAccountIdent(addrHash))
				if err != nil {
					return err
				}
				for _, n := range idList {
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
	for addrHash, slots := range b.storages {
		storages += len(slots)
		for storageHash, idList := range slots {
			eg.Go(func() error {
				if !b.delete {
					iw, err := newIndexWriter(b.db, newStorageIdent(addrHash, storageHash))
					if err != nil {
						return err
					}
					for _, n := range idList {
						if err := iw.append(n); err != nil {
							return err
						}
					}
					batchMu.Lock()
					iw.finish(batch)
					batchMu.Unlock()
				} else {
					id, err := newIndexDeleter(b.db, newStorageIdent(addrHash, storageHash))
					if err != nil {
						return err
					}
					for _, n := range idList {
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
	}
	if err := eg.Wait(); err != nil {
		return err
	}
	// Update the position of last indexed state history
	if !b.delete {
		storeIndexMetadata(batch, b.lastID)
	} else {
		if b.lastID == 1 {
			rawdb.DeleteStateHistoryIndexMetadata(batch)
		} else {
			storeIndexMetadata(batch, b.lastID-1)
		}
	}
	if err := batch.Write(); err != nil {
		return err
	}
	log.Debug("Committed batch indexer", "accounts", len(b.accounts), "storages", storages, "records", b.counter, "elapsed", common.PrettyDuration(time.Since(start)))
	b.counter = 0
	b.accounts = make(map[common.Hash][]uint64)
	b.storages = make(map[common.Hash]map[common.Hash][]uint64)
	return nil
}

// indexSingle processes the state history with the specified ID for indexing.
func indexSingle(historyID uint64, db ethdb.KeyValueStore, freezer ethdb.AncientReader) error {
	start := time.Now()
	defer func() {
		indexHistoryTimer.UpdateSince(start)
	}()

	metadata := loadIndexMetadata(db)
	if metadata == nil || metadata.Last+1 != historyID {
		last := "null"
		if metadata != nil {
			last = fmt.Sprintf("%v", metadata.Last)
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
	log.Debug("Indexed state history", "id", historyID, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// unindexSingle processes the state history with the specified ID for unindexing.
func unindexSingle(historyID uint64, db ethdb.KeyValueStore, freezer ethdb.AncientReader) error {
	start := time.Now()
	defer func() {
		unindexHistoryTimer.UpdateSince(start)
	}()

	metadata := loadIndexMetadata(db)
	if metadata == nil || metadata.Last != historyID {
		last := "null"
		if metadata != nil {
			last = fmt.Sprintf("%v", metadata.Last)
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
	log.Debug("Unindexed state history", "id", historyID, "elapsed", common.PrettyDuration(time.Since(start)))
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

	// indexing progress
	indexed atomic.Uint64 // the id of latest indexed state
	last    atomic.Uint64 // the id of the target state to be indexed

	wg sync.WaitGroup
}

func newIndexIniter(disk ethdb.KeyValueStore, freezer ethdb.AncientStore, lastID uint64) *indexIniter {
	initer := &indexIniter{
		disk:      disk,
		freezer:   freezer,
		interrupt: make(chan *interruptSignal),
		done:      make(chan struct{}),
		closed:    make(chan struct{}),
	}
	// Load indexing progress
	initer.last.Store(lastID)
	metadata := loadIndexMetadata(disk)
	if metadata != nil {
		initer.indexed.Store(metadata.Last)
	}

	// Launch background indexer
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

func (i *indexIniter) remain() uint64 {
	select {
	case <-i.closed:
		return 0
	case <-i.done:
		return 0
	default:
		last, indexed := i.last.Load(), i.indexed.Load()
		if last < indexed {
			log.Error("Invalid state indexing range", "last", last, "indexed", indexed)
			return 0
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
			metadata := loadIndexMetadata(i.disk)
			return metadata != nil && metadata.Last == lastID
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
			i.last.Store(signal.newLastID) // update indexing range

			// The index limit is extended by one, update the limit without
			// interrupting the current background process.
			if signal.newLastID == lastID+1 {
				lastID = signal.newLastID
				signal.result <- nil
				log.Debug("Extended state history range", "last", lastID)
				continue
			}
			// The index limit is shortened by one, interrupt the current background
			// process and relaunch with new target.
			interrupt.Store(1)
			<-done

			// If all state histories, including the one to be reverted, have
			// been fully indexed, unindex it here and shut down the initializer.
			if checkDone() {
				log.Info("Truncate the extra history", "id", lastID)
				if err := unindexSingle(lastID, i.disk, i.freezer); err != nil {
					signal.result <- err
					return
				}
				close(i.done)
				signal.result <- nil
				log.Info("State histories have been fully indexed", "last", lastID-1)
				return
			}
			// Adjust the indexing target and relaunch the process
			lastID = signal.newLastID
			done, interrupt = make(chan struct{}), new(atomic.Int32)
			go i.index(done, interrupt, lastID)
			log.Debug("Shortened state history range", "last", lastID)

		case <-done:
			if checkDone() {
				close(i.done)
				log.Info("State histories have been fully indexed", "last", lastID)
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
	metadata := loadIndexMetadata(i.disk)
	if metadata == nil {
		log.Debug("Initialize state history indexing from scratch", "id", tailID)
		return tailID, nil
	}
	// Resume indexing from the last interrupted position
	if metadata.Last+1 >= tailID {
		log.Debug("Resume state history indexing", "id", metadata.Last+1, "tail", tailID)
		return metadata.Last + 1, nil
	}
	// History has been shortened without indexing. Discard the gapped segment
	// in the history and shift to the first available element.
	//
	// The missing indexes corresponding to the gapped histories won't be visible.
	// It's fine to leave them unindexed.
	log.Info("History gap detected, discard old segment", "oldHead", metadata.Last, "newHead", tailID)
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
		if lastID == 0 && beginID == 1 {
			// Initialize the indexing flag if the state history is empty by
			// using zero as the disk layer ID. This is a common case that
			// can occur after snap sync.
			//
			// This step is essential to avoid spinning up indexing thread
			// endlessly until a history object is produced.
			storeIndexMetadata(i.disk, 0)
			log.Info("Initialized history indexing flag")
		} else {
			log.Debug("State history is fully indexed", "last", lastID)
		}
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
				log.Info("Indexing state history", "processed", done, "left", left, "elapsed", common.PrettyDuration(time.Since(start)), "eta", common.PrettyDuration(eta))
			}
		}
		i.indexed.Store(current - 1) // update indexing progress

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

// checkVersion checks whether the index data in the database matches the version.
func checkVersion(disk ethdb.KeyValueStore) {
	blob := rawdb.ReadStateHistoryIndexMetadata(disk)
	if len(blob) == 0 {
		return
	}
	var m indexMetadata
	err := rlp.DecodeBytes(blob, &m)
	if err == nil && m.Version == stateIndexVersion {
		return
	}
	// TODO(rjl493456442) would be better to group them into a batch.
	rawdb.DeleteStateHistoryIndexMetadata(disk)
	rawdb.DeleteStateHistoryIndex(disk)

	version := "unknown"
	if err == nil {
		version = fmt.Sprintf("%d", m.Version)
	}
	log.Info("Cleaned up obsolete state history index", "version", version, "want", stateIndexVersion)
}

// newHistoryIndexer constructs the history indexer and launches the background
// initer to complete the indexing of any remaining state histories.
func newHistoryIndexer(disk ethdb.KeyValueStore, freezer ethdb.AncientStore, lastHistoryID uint64) *historyIndexer {
	checkVersion(disk)
	return &historyIndexer{
		initer:  newIndexIniter(disk, freezer, lastHistoryID),
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
