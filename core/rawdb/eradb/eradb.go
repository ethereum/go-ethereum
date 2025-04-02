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
	"path"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/era"
	"github.com/ethereum/go-ethereum/log"
)

type EraDatabase struct {
	datadir string
	table   map[uint64]*era.Era
	mu      sync.RWMutex
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
	log.Info("Opened erastore", "datadir", datadir)
	return &EraDatabase{datadir: datadir, table: make(map[uint64]*era.Era)}, nil
}

// scan returns a list of all era1 files in the datadir.
func (db *EraDatabase) scan() ([]string, error) {
	entries, err := os.ReadDir(db.datadir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, entry := range entries {
		if entry.IsDir() || entry.Type()&os.ModeSymlink != 0 {
			continue
		}
		// Skip files that are not era1 files.
		if path.Ext(entry.Name()) != ".era1" {
			continue
		}
		files = append(files, entry.Name())
	}
	return files, nil
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
	epoch := e.Start() / uint64(era.MaxEra1Size)
	db.mu.Lock()
	db.table[epoch] = e
	db.mu.Unlock()
	return e, nil
}

func (db *EraDatabase) Close() {
	db.mu.Lock()
	defer db.mu.Unlock()
	for _, e := range db.table {
		if err := e.Close(); err != nil {
			log.Warn("Failed to close era", "error", err)
		}
	}
	db.table = nil
}

func (db *EraDatabase) GetBlockByNumber(number uint64) (*types.Block, error) {
	files, err := db.scan()
	if err != nil {
		return nil, err
	}
	for _, file := range files {
		_, err := db.openEra(path.Join(db.datadir, file))
		if err != nil {
			return nil, err
		}
	}

	// Lookup the table by epoch.
	epoch := number / uint64(era.MaxEra1Size)
	db.mu.RLock()
	defer db.mu.RUnlock()
	if e, ok := db.table[epoch]; ok {
		block, err := e.GetBlockByNumber(number)
		if err == nil {
			return block, nil
		}
	}

	return nil, errors.New("block not found")
}
