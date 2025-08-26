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

// Package eradb implements a history backend using era1 files.
package eradb

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"path/filepath"
	"sync"

	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

const openFileLimit = 64

var errClosed = errors.New("era store is closed")

// Store manages read access to a directory of era1 files.
// The getter methods are thread-safe.
type Store struct {
	datadir string

	// The mutex protects all remaining fields.
	mu      sync.Mutex
	cond    *sync.Cond
	lru     lru.BasicLRU[uint64, *fileCacheEntry]
	opening map[uint64]*fileCacheEntry
	closing bool
}

type fileCacheEntry struct {
	refcount int           // reference count. This is protected by Store.mu!
	opened   chan struct{} // signals opening of file has completed
	file     *era.Era      // the file
	err      error         // error from opening the file
}

type fileCacheStatus byte

const (
	storeClosing fileCacheStatus = iota
	fileIsNew
	fileIsOpening
	fileIsCached
)

// New opens the store directory.
func New(datadir string) (*Store, error) {
	db := &Store{
		datadir: datadir,
		lru:     lru.NewBasicLRU[uint64, *fileCacheEntry](openFileLimit),
		opening: make(map[uint64]*fileCacheEntry),
	}
	db.cond = sync.NewCond(&db.mu)
	log.Info("Opened Era store", "datadir", datadir)
	return db, nil
}

// Close closes all open era1 files in the cache.
func (db *Store) Close() {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Prevent new cache additions.
	db.closing = true

	// Deref all active files. Since inactive files have a refcount of one, they will be
	// closed right here and now after decrementing. Files which are currently being used
	// have a refcount > 1 and will hit zero when their access finishes.
	for _, epoch := range db.lru.Keys() {
		entry, _ := db.lru.Peek(epoch)
		if entry.derefAndClose(epoch) {
			db.lru.Remove(epoch)
		}
	}

	// Wait for all store access to finish.
	for db.lru.Len() > 0 || len(db.opening) > 0 {
		db.cond.Wait()
	}
}

// GetRawBody returns the raw body for a given block number.
func (db *Store) GetRawBody(number uint64) ([]byte, error) {
	epoch := number / uint64(era.MaxEra1Size)
	entry := db.getEraByEpoch(epoch)
	if entry.err != nil {
		if errors.Is(entry.err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, entry.err
	}
	defer db.doneWithFile(epoch, entry)

	return entry.file.GetRawBodyByNumber(number)
}

// GetRawReceipts returns the raw receipts for a given block number.
func (db *Store) GetRawReceipts(number uint64) ([]byte, error) {
	epoch := number / uint64(era.MaxEra1Size)
	entry := db.getEraByEpoch(epoch)
	if entry.err != nil {
		if errors.Is(entry.err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, entry.err
	}
	defer db.doneWithFile(epoch, entry)

	data, err := entry.file.GetRawReceiptsByNumber(number)
	if err != nil {
		return nil, err
	}
	return convertReceipts(data)
}

// convertReceipts transforms an encoded block receipts list from the format
// used by era1 into the 'storage' format used by the go-ethereum ancients database.
func convertReceipts(input []byte) ([]byte, error) {
	var (
		out bytes.Buffer
		enc = rlp.NewEncoderBuffer(&out)
	)
	blockListIter, err := rlp.NewListIterator(input)
	if err != nil {
		return nil, fmt.Errorf("invalid block receipts list: %v", err)
	}
	outerList := enc.List()
	for i := 0; blockListIter.Next(); i++ {
		kind, content, _, err := rlp.Split(blockListIter.Value())
		if err != nil {
			return nil, fmt.Errorf("receipt %d invalid: %v", i, err)
		}
		var receiptData []byte
		switch kind {
		case rlp.Byte:
			return nil, fmt.Errorf("receipt %d is single byte", i)
		case rlp.String:
			// Typed receipt - skip type.
			receiptData = content[1:]
		case rlp.List:
			// Legacy receipt
			receiptData = blockListIter.Value()
		}
		// Convert data list.
		// Input is  [status, gas-used, bloom, logs]
		// Output is [status, gas-used, logs], i.e. we need to skip the bloom.
		dataIter, err := rlp.NewListIterator(receiptData)
		if err != nil {
			return nil, fmt.Errorf("receipt %d has invalid data: %v", i, err)
		}
		innerList := enc.List()
		for field := 0; dataIter.Next(); field++ {
			if field == 2 {
				continue // skip bloom
			}
			enc.Write(dataIter.Value())
		}
		enc.ListEnd(innerList)
		if dataIter.Err() != nil {
			return nil, fmt.Errorf("receipt %d iterator error: %v", i, dataIter.Err())
		}
	}
	enc.ListEnd(outerList)
	if blockListIter.Err() != nil {
		return nil, fmt.Errorf("block receipt list iterator error: %v", blockListIter.Err())
	}
	enc.Flush()
	return out.Bytes(), nil
}

// getEraByEpoch opens an era file or gets it from the cache.
// The caller can freely access the returned entry's .file and .err
// db.doneWithFile must be called when it is done reading the file.
func (db *Store) getEraByEpoch(epoch uint64) *fileCacheEntry {
	stat, entry := db.getCacheEntry(epoch)

	switch stat {
	case storeClosing:
		return &fileCacheEntry{err: errClosed}

	case fileIsNew:
		// Open the file and put it into the cache.
		e, err := db.openEraFile(epoch)
		if err != nil {
			db.fileFailedToOpen(epoch, entry, err)
		} else {
			db.fileOpened(epoch, entry, e)
		}
		close(entry.opened)

	case fileIsOpening:
		// Wait for open to finish.
		<-entry.opened

	case fileIsCached:
		// Nothing to do.

	default:
		panic(fmt.Sprintf("invalid file state %d", stat))
	}
	return entry
}

// getCacheEntry gets an open era file from the cache.
func (db *Store) getCacheEntry(epoch uint64) (stat fileCacheStatus, entry *fileCacheEntry) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if db.closing {
		return storeClosing, nil
	}
	if entry = db.opening[epoch]; entry != nil {
		stat = fileIsOpening
	} else if entry, _ = db.lru.Get(epoch); entry != nil {
		stat = fileIsCached
	} else {
		// It's a new file, create an entry in the opening table. Note the entry is
		// created with an initial refcount of one. We increment the count once more
		// before returning, but the count will return to one when the file has been
		// accessed. When the store is closed or the file gets evicted from the cache,
		// refcount will be decreased by one, thus allowing it to hit zero.
		entry = &fileCacheEntry{refcount: 1, opened: make(chan struct{})}
		db.opening[epoch] = entry
		stat = fileIsNew
	}
	entry.refcount++
	return stat, entry
}

// fileOpened is called after an era file has been successfully opened.
func (db *Store) fileOpened(epoch uint64, entry *fileCacheEntry, file *era.Era) {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.opening, epoch)
	db.cond.Signal() // db.opening was modified

	// The database may have been closed while opening the file. When that happens, we
	// need to close the file here, since it isn't tracked by the LRU yet.
	if db.closing {
		entry.err = errClosed
		file.Close()
		return
	}

	// Add it to the LRU. This may evict an existing item, which we have to close.
	entry.file = file
	evictedEpoch, evictedEntry, _ := db.lru.Add3(epoch, entry)
	if evictedEntry != nil {
		evictedEntry.derefAndClose(evictedEpoch)
	}
}

// fileFailedToOpen is called when an era file could not be opened.
func (db *Store) fileFailedToOpen(epoch uint64, entry *fileCacheEntry, err error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	delete(db.opening, epoch)
	db.cond.Signal() // db.opening was modified
	entry.err = err
}

func (db *Store) openEraFile(epoch uint64) (*era.Era, error) {
	// File name scheme is <network>-<epoch>-<root>.
	glob := fmt.Sprintf("*-%05d-*.era1", epoch)
	matches, err := filepath.Glob(filepath.Join(db.datadir, glob))
	if err != nil {
		return nil, err
	}
	if len(matches) > 1 {
		return nil, fmt.Errorf("multiple era1 files found for epoch %d", epoch)
	}
	if len(matches) == 0 {
		return nil, fs.ErrNotExist
	}
	filename := matches[0]

	e, err := era.Open(filename)
	if err != nil {
		return nil, err
	}
	// Sanity-check start block.
	if e.Start()%uint64(era.MaxEra1Size) != 0 {
		return nil, fmt.Errorf("pre-merge era1 file has invalid boundary. %d %% %d != 0", e.Start(), era.MaxEra1Size)
	}
	log.Debug("Opened era1 file", "epoch", epoch)
	return e, nil
}

// doneWithFile signals that the caller has finished using a file.
// This decrements the refcount and ensures the file is closed by the last user.
func (db *Store) doneWithFile(epoch uint64, entry *fileCacheEntry) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if entry.err != nil {
		return
	}
	if entry.derefAndClose(epoch) {
		// Delete closed entry from LRU if it is still present.
		if e, _ := db.lru.Peek(epoch); e == entry {
			db.lru.Remove(epoch)
			db.cond.Signal() // db.lru was modified
		}
	}
}

// derefAndClose decrements the reference counter and closes the file
// when it hits zero.
func (entry *fileCacheEntry) derefAndClose(epoch uint64) (closed bool) {
	entry.refcount--
	if entry.refcount > 0 {
		return false
	}

	closeErr := entry.file.Close()
	if closeErr == nil {
		log.Debug("Closed era1 file", "epoch", epoch)
	} else {
		log.Warn("Error closing era1 file", "epoch", epoch, "err", closeErr)
	}
	return true
}
