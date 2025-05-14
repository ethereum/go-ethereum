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

package eradb

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/log"
)

const openFileLimit = 64

var errClosed = errors.New("era store is closed")

// Store manages read access to a directory of era1 files.
// The getter methods are thread-safe.
type Store struct {
	datadir string
	mu      sync.Mutex
	lru     lru.BasicLRU[uint64, *fileCacheEntry]
	opening map[uint64]*fileCacheEntry
}

type fileCacheEntry struct {
	ref    atomic.Int32
	opened chan struct{}
	file   *era.Era
	err    error
}

// New opens the store directory.
func New(datadir string) (*Store, error) {
	// Ensure the datadir is not a symbolic link if it exists.
	if info, err := os.Lstat(datadir); !os.IsNotExist(err) {
		if info == nil {
			log.Warn("Could not Lstat the database", "path", datadir)
			return nil, errors.New("lstat failed")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			log.Warn("Symbolic link erastore is not supported", "path", datadir)
			return nil, errors.New("symbolic link datadir is not supported")
		}
	}
	db := &Store{
		datadir: datadir,
		lru:     lru.NewBasicLRU[uint64, *fileCacheEntry](openFileLimit),
		opening: make(map[uint64]*fileCacheEntry),
	}
	log.Info("Opened Era store", "datadir", datadir)
	return db, nil
}

// Close closes all open era1 files in the cache.
func (db *Store) Close() {
	db.mu.Lock()
	defer db.mu.Unlock()

	keys := db.lru.Keys()
	for _, epoch := range keys {
		entry, _ := db.lru.Get(epoch)
		entry.done(epoch)
	}
	db.opening = nil
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
	defer entry.done(epoch)
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
	defer entry.done(epoch)
	return entry.file.GetRawReceiptsByNumber(number)
}

// getEraByEpoch opens an era file or gets it from the cache. The caller can access
// entry.file and entry.err and must call entry.done when done reading the file.
func (db *Store) getEraByEpoch(epoch uint64) *fileCacheEntry {
	// Add the requested epoch to the cache.
	entry := db.getCacheEntry(epoch)
	if entry == nil {
		return &fileCacheEntry{err: errClosed}
	}

	// First goroutine to use the file has to open it.
	if entry.ref.Add(1) == 1 {
		e, err := db.openEraFile(epoch)
		if err != nil {
			db.fileFailedToOpen(epoch, entry, err)
		} else {
			db.fileOpened(epoch, entry, e)
		}
		close(entry.opened)
	}

	// Bump the refcount and wait for the file to be opened.
	entry.ref.Add(1)
	<-entry.opened
	return entry
}

// getCacheEntry gets an open era file from the cache.
func (db *Store) getCacheEntry(epoch uint64) *fileCacheEntry {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Check if this epoch is already being opened.
	if db.opening == nil {
		return nil
	}
	if entry, ok := db.opening[epoch]; ok {
		return entry
	}
	// Check if it's in the cache.
	if entry, ok := db.lru.Get(epoch); ok {
		return entry
	}
	// It's a new file, create an entry in the 'opening' table.
	entry := &fileCacheEntry{opened: make(chan struct{})}
	db.opening[epoch] = entry
	return entry
}

// fileOpened is called after an era file has been successfully opened.
func (db *Store) fileOpened(epoch uint64, entry *fileCacheEntry, file *era.Era) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// The database may have been closed while opening the file. When that happens,
	// db.opening will be set to nil, so we need to handle that here and ensure the caller
	// knows.
	if db.opening == nil {
		entry.err = errClosed
		return
	}

	// Remove from 'opening' table and add to the LRU.
	// This may evict an existing item, which we have to close.
	entry.file = file
	delete(db.opening, epoch)
	if _, evictedEntry, _ := db.lru.Add3(epoch, entry); evictedEntry != nil {
		evictedEntry.done(epoch)
	}
}

// fileFailedToOpen is called when an era file could not be opened.
func (db *Store) fileFailedToOpen(epoch uint64, entry *fileCacheEntry, err error) {
	entry.err = err

	db.mu.Lock()
	defer db.mu.Unlock()
	if db.opening != nil {
		delete(db.opening, epoch)
	}
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
		return nil, nil
	}
	filename := matches[0]

	e, err := era.Open(filename)
	if err != nil {
		return nil, err
	}
	// Assign an epoch to the table.
	if e.Count() != uint64(era.MaxEra1Size) {
		return nil, fmt.Errorf("pre-merge era1 files must be full. Want: %d, have: %d", era.MaxEra1Size, e.Count())
	}
	if e.Start()%uint64(era.MaxEra1Size) != 0 {
		return nil, fmt.Errorf("pre-merge era1 file has invalid boundary. %d %% %d != 0", e.Start(), era.MaxEra1Size)
	}
	return e, nil
}

// done signals that the caller has finished using a file.
// This decrements the refcount and ensures the file is closed by the last user.
func (f *fileCacheEntry) done(epoch uint64) {
	if f.err != nil {
		return
	}
	if f.ref.Add(-1) == 0 {
		err := f.file.Close()
		if err == nil {
			log.Debug("Closed era1 file", "epoch", epoch)
		} else {
			log.Warn("Error closing era1 file", "epoch", epoch, "err", err)
		}
	}
}
