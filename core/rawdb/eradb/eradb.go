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
	"os"
	"path/filepath"

	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/log"
)

/**
* TODO:
* - FD leak possible on cache eviction.
* - FD leak possible on concurrent access to GetRaw*.
 */

const (
	openFileLimit = 64
)

// EraDatabase manages read access to a directory of era1 files.
// The getter methods are thread-safe.
type EraDatabase struct {
	datadir string
	cache   *lru.Cache[uint64, *era.Era]
}

// New creates a new EraDatabase instance.
func New(datadir string) (*EraDatabase, error) {
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
	if err := os.MkdirAll(datadir, 0755); err != nil {
		return nil, err
	}
	db := &EraDatabase{datadir: datadir, cache: lru.NewCache[uint64, *era.Era](openFileLimit)}
	db.cache.OnEvicted(func(key uint64, value *era.Era) {
		if value == nil {
			log.Warn("Era1 cache evicted nil value", "epoch", key)
			return
		}
		// Close the era1 file when it is evicted from the cache
		// to avoid leaks.
		if err := value.Close(); err != nil {
			log.Warn("Error closing era1 file", "epoch", key, "err", err)
		}
	})
	log.Info("Opened erastore", "datadir", datadir)
	return db, nil
}

// Close closes all open era1 files in the cache.
func (db *EraDatabase) Close() error {
	// Close all open era1 files in the cache.
	keys := db.cache.Keys()
	errs := make([]error, len(keys))
	for _, key := range keys {
		if e, ok := db.cache.Get(key); ok {
			if err := e.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}
	return errors.Join(errs...)
}

// GetRawBody returns the raw body for a given block number.
func (db *EraDatabase) GetRawBody(number uint64) ([]byte, error) {
	// Lookup the table by epoch.
	epoch := number / uint64(era.MaxEra1Size)
	e, err := db.getEraByEpoch(epoch)
	if err != nil {
		return nil, err
	}
	// The era1 file for given epoch may not exist.
	if e == nil {
		return nil, nil
	}
	return e.GetRawBodyByNumber(number)
}

// GetRawReceipts returns the raw receipts for a given block number.
func (db *EraDatabase) GetRawReceipts(number uint64) ([]byte, error) {
	epoch := number / uint64(era.MaxEra1Size)
	e, err := db.getEraByEpoch(epoch)
	if err != nil {
		return nil, err
	}
	// The era1 file for given epoch may not exist.
	if e == nil {
		return nil, nil
	}
	return e.GetRawReceiptsByNumber(number)
}

func (db *EraDatabase) openEra(name string) (*era.Era, error) {
	e, err := era.Open(name)
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

func (db *EraDatabase) getEraByEpoch(epoch uint64) (*era.Era, error) {
	// Check the cache first.
	if e, ok := db.cache.Get(epoch); ok {
		return e, nil
	}
	// file name scheme is <network>-<epoch>-<root>.
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
	e, err := db.openEra(filename)
	if err != nil {
		return nil, err
	}
	// Add the era to the cache.
	db.cache.Add(epoch, e)
	return e, nil
}
