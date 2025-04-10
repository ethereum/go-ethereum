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
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/log"
)

type EraDatabase struct {
	datadir string
	// TODO: should take into account configured number of fd handles.
	cache *lru.Cache[uint64, *era.Era]
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
	db := &EraDatabase{datadir: datadir, cache: lru.NewCache[uint64, *era.Era](50)}
	log.Info("Opened erastore", "datadir", datadir)
	return db, nil
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

func (db *EraDatabase) Close() {
	// Close all open era1 files in the cache.
	keys := db.cache.Keys()
	for _, key := range keys {
		if e, ok := db.cache.Get(key); ok {
			e.Close()
		}
	}
}

// TODO: do we need this method? we do have headers in the freezer.
func (db *EraDatabase) GetHeaderByNumber(number uint64) (*types.Header, error) {
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
	return e.GetHeaderByNumber(number)
}

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

func (db *EraDatabase) GetBlockByNumber(number uint64) (*types.Block, error) {
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
	return e.GetBlockByNumber(number)
}

func (db *EraDatabase) GetReceiptsByNumber(number uint64) (types.Receipts, error) {
	epoch := number / uint64(era.MaxEra1Size)
	e, err := db.getEraByEpoch(epoch)
	if err != nil {
		return nil, err
	}
	// The era1 file for given epoch may not exist.
	if e == nil {
		return nil, nil
	}
	return e.GetReceiptsByNumber(number)
}
