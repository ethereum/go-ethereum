// Copyright 2019 The go-ethereum Authors
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

package rawdb

import (
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/gofrs/flock"
)

var (
	// errReadOnly is returned if the freezer is opened in read only mode. All the
	// mutations are disallowed.
	errReadOnly = errors.New("read only")

	// errUnknownTable is returned if the user attempts to read from a table that is
	// not tracked by the freezer.
	errUnknownTable = errors.New("unknown table")

	// errOutOrderInsertion is returned if the user attempts to inject out-of-order
	// binary blobs into the freezer.
	errOutOrderInsertion = errors.New("the append operation is out-order")

	// errSymlinkDatadir is returned if the ancient directory specified by user
	// is a symbolic link.
	errSymlinkDatadir = errors.New("symbolic link datadir is not supported")
)

// freezerTableSize defines the maximum size of freezer data files.
const freezerTableSize = 2 * 1000 * 1000 * 1000

// Freezer is an append-only database to store immutable ordered data into
// flat files:
//
// - The append-only nature ensures that disk writes are minimized.
// - The in-order data ensures that disk reads are always optimized.
type Freezer struct {
	datadir string
	frozen  atomic.Uint64 // Number of items already frozen
	tail    atomic.Uint64 // Number of the first stored item in the freezer

	// This lock synchronizes writers and the truncate operation, as well as
	// the "atomic" (batched) read operations.
	writeLock  sync.RWMutex
	writeBatch *freezerBatch

	// Used during ancient db pruning
	offset atomic.Uint64 // Starting block number in current freezer

	readonly     bool
	tables       map[string]*freezerTable // Data tables for storing everything
	instanceLock *flock.Flock             // File-system lock to prevent double opens
	closeOnce    sync.Once
}

// NewFreezer creates a freezer instance for maintaining immutable ordered
// data according to the given parameters.
//
// The 'tables' argument defines the data tables. If the value of a map
// entry is true, snappy compression is disabled for the table.
func NewFreezer(datadir string, namespace string, readonly bool, offset uint64, maxTableSize uint32, tables map[string]freezerTableConfig) (*Freezer, error) {
	// Create the initial freezer object
	var (
		readMeter  = metrics.NewRegisteredMeter(namespace+"ancient/read", nil)
		writeMeter = metrics.NewRegisteredMeter(namespace+"ancient/write", nil)
		sizeGauge  = metrics.NewRegisteredGauge(namespace+"ancient/size", nil)
	)
	// Ensure the datadir is not a symbolic link if it exists.
	if info, err := os.Lstat(datadir); !os.IsNotExist(err) {
		if info == nil {
			log.Warn("Could not Lstat the database", "path", datadir)
			return nil, errors.New("lstat failed")
		}
		if info.Mode()&os.ModeSymlink != 0 {
			log.Warn("Symbolic link ancient database is not supported", "path", datadir)
			return nil, errSymlinkDatadir
		}
	}
	flockFile := filepath.Join(datadir, "FLOCK")
	if err := os.MkdirAll(filepath.Dir(flockFile), 0755); err != nil {
		return nil, err
	}
	// Leveldb uses LOCK as the filelock filename. To prevent the
	// name collision, we use FLOCK as the lock name.
	lock := flock.New(flockFile)
	tryLock := lock.TryLock
	if readonly {
		tryLock = lock.TryRLock
	}
	if locked, err := tryLock(); err != nil {
		return nil, err
	} else if !locked {
		return nil, errors.New("locking failed")
	}
	// Open all the supported data tables
	freezer := &Freezer{
		datadir:      datadir,
		readonly:     readonly,
		tables:       make(map[string]*freezerTable),
		instanceLock: lock,
	}
	freezer.offset.Store(offset)

	// Create the tables.
	for name, config := range tables {
		table, err := newTable(datadir, name, readMeter, writeMeter, sizeGauge, maxTableSize, config, readonly)
		if err != nil {
			for _, table := range freezer.tables {
				table.Close()
			}
			lock.Unlock()
			return nil, err
		}
		freezer.tables[name] = table
	}
	var err error
	if freezer.readonly {
		// In readonly mode only validate, don't truncate.
		// validate also sets `freezer.frozen`.
		err = freezer.validate()
	} else {
		// Truncate all tables to common length.
		err = freezer.repair()
	}
	if err != nil {
		for _, table := range freezer.tables {
			table.Close()
		}
		lock.Unlock()
		return nil, err
	}

	// Some blocks in ancientDB may have already been frozen and been pruned, so adding the offset to
	// represent the absolute number of blocks already frozen.
	freezer.frozen.Add(offset)

	// Create the write batch.
	freezer.writeBatch = newFreezerBatch(freezer)

	log.Info("Opened ancient database", "database", datadir, "readonly", readonly, "frozen", freezer.frozen.Load(), "offset", freezer.offset.Load())
	return freezer, nil
}

// Close terminates the chain freezer, closing all the data files.
func (f *Freezer) Close() error {
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	var errs []error
	f.closeOnce.Do(func() {
		for _, table := range f.tables {
			if err := table.Close(); err != nil {
				errs = append(errs, err)
			}
		}
		if err := f.instanceLock.Unlock(); err != nil {
			errs = append(errs, err)
		}
	})
	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

// AncientDatadir returns the path of the ancient store.
func (f *Freezer) AncientDatadir() (string, error) {
	return f.datadir, nil
}

// HasAncient returns an indicator whether the specified ancient data exists
// in the freezer.
func (f *Freezer) HasAncient(kind string, number uint64) (bool, error) {
	if table := f.tables[kind]; table != nil {
		return table.has(number - f.offset.Load()), nil
	}
	return false, nil
}

// Ancient retrieves an ancient binary blob from the append-only immutable files.
func (f *Freezer) Ancient(kind string, number uint64) ([]byte, error) {
	if table := f.tables[kind]; table != nil {
		return table.Retrieve(number - f.offset.Load())
	}
	return nil, errUnknownTable
}

// AncientRange retrieves multiple items in sequence, starting from the index 'start'.
// It will return
//   - at most 'count' items,
//   - if maxBytes is specified: at least 1 item (even if exceeding the maxByteSize),
//     but will otherwise return as many items as fit into maxByteSize.
//   - if maxBytes is not specified, 'count' items will be returned if they are present.
func (f *Freezer) AncientRange(kind string, start, count, maxBytes uint64) ([][]byte, error) {
	if table := f.tables[kind]; table != nil {
		return table.RetrieveItems(start, count, maxBytes)
	}
	return nil, errUnknownTable
}

// Ancients returns the length of the frozen items.
func (f *Freezer) Ancients() (uint64, error) {
	return f.frozen.Load(), nil
}

// ItemAmountInAncient returns the actual length of current ancientDB.
func (f *Freezer) ItemAmountInAncient() (uint64, error) {
	return f.frozen.Load() - f.offset.Load(), nil
}

// AncientOffSet returns the offset of current ancientDB.
func (f *Freezer) AncientOffSet() uint64 {
	return f.offset.Load()
}

// Tail returns the number of first stored item in the freezer.
func (f *Freezer) Tail() (uint64, error) {
	return f.tail.Load(), nil
}

// AncientSize returns the ancient size of the specified category.
func (f *Freezer) AncientSize(kind string) (uint64, error) {
	// This needs the write lock to avoid data races on table fields.
	// Speed doesn't matter here, AncientSize is for debugging.
	f.writeLock.RLock()
	defer f.writeLock.RUnlock()

	if table := f.tables[kind]; table != nil {
		return table.size()
	}
	return 0, errUnknownTable
}

// ReadAncients runs the given read operation while ensuring that no writes take place
// on the underlying freezer.
func (f *Freezer) ReadAncients(fn func(ethdb.AncientReaderOp) error) (err error) {
	f.writeLock.RLock()
	defer f.writeLock.RUnlock()

	return fn(f)
}

// ModifyAncients runs the given write operation.
func (f *Freezer) ModifyAncients(fn func(ethdb.AncientWriteOp) error) (writeSize int64, err error) {
	if f.readonly {
		return 0, errReadOnly
	}
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	// Roll back all tables to the starting position in case of error.
	prevItem := f.frozen.Load()
	defer func() {
		if err != nil {
			// The write operation has failed. Go back to the previous item position.
			for name, table := range f.tables {
				err := table.truncateHead(prevItem)
				if err != nil {
					log.Error("Freezer table roll-back failed", "table", name, "index", prevItem, "err", err)
				}
			}
		}
	}()

	f.writeBatch.reset()
	if err := fn(f.writeBatch); err != nil {
		return 0, err
	}
	item, writeSize, err := f.writeBatch.commit()
	if err != nil {
		return 0, err
	}
	f.frozen.Store(item)
	return writeSize, nil
}

// TruncateHead discards any recent data above the provided threshold number.
// It returns the previous head number.
func (f *Freezer) TruncateHead(items uint64) (uint64, error) {
	if f.readonly {
		return 0, errReadOnly
	}
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	oitems := f.frozen.Load()
	if oitems <= items {
		return oitems, nil
	}
	for _, table := range f.tables {
		if err := table.truncateHead(items - f.offset.Load()); err != nil {
			return 0, err
		}
	}
	f.frozen.Store(items)
	return oitems, nil
}

// TruncateTail discards all data below the specified threshold. Note that only
// 'prunable' tables will be truncated.
func (f *Freezer) TruncateTail(tail uint64) (uint64, error) {
	if f.readonly {
		return 0, errReadOnly
	}
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	old := f.tail.Load()
	if old >= tail {
		return old, nil
	}
	for _, table := range f.tables {
		if table.config.prunable {
			if err := table.truncateTail(tail - f.offset.Load()); err != nil {
				return 0, err
			}
		}
	}
	f.tail.Store(tail)
	return old, nil
}

// Sync flushes all data tables to disk.
func (f *Freezer) Sync() error {
	var errs []error
	for _, table := range f.tables {
		if err := table.Sync(); err != nil {
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

// validate checks that every table has the same boundary.
// Used instead of `repair` in readonly mode.
func (f *Freezer) validate() error {
	if len(f.tables) == 0 {
		return nil
	}
	var (
		head       uint64
		prunedTail *uint64
	)
	// get any head value
	for _, table := range f.tables {
		head = table.items.Load()
		break
	}
	for kind, table := range f.tables {
		// all tables have to have the same head
		if head != table.items.Load() {
			return fmt.Errorf("freezer table %s has a differing head: %d != %d", kind, table.items.Load(), head)
		}
		if !table.config.prunable {
			// non-prunable tables have to start at 0
			if table.itemHidden.Load() != 0 {
				return fmt.Errorf("non-prunable freezer table '%s' has a non-zero tail: %d", kind, table.itemHidden.Load())
			}
		} else {
			// prunable tables have to have the same length
			if prunedTail == nil {
				tmp := table.itemHidden.Load()
				prunedTail = &tmp
			}
			if *prunedTail != table.itemHidden.Load() {
				return fmt.Errorf("freezer table %s has differing tail: %d != %d", kind, table.itemHidden.Load(), *prunedTail)
			}
		}
	}

	if prunedTail == nil {
		tmp := uint64(0)
		prunedTail = &tmp
	}

	f.frozen.Store(head)
	f.tail.Store(*prunedTail)
	return nil
}

// repair truncates all data tables to the same length.
func (f *Freezer) repair() error {
	var (
		head       = uint64(math.MaxUint64)
		prunedTail = uint64(0)
	)
	// get the minimal head and the maximum tail
	for _, table := range f.tables {
		head = min(head, table.items.Load())
		prunedTail = max(prunedTail, table.itemHidden.Load())
	}
	// apply the pruning
	for kind, table := range f.tables {
		// all tables need to have the same head
		if err := table.truncateHead(head); err != nil {
			return err
		}
		if !table.config.prunable {
			// non-prunable tables have to start at 0
			if table.itemHidden.Load() != 0 {
				panic(fmt.Sprintf("non-prunable freezer table %s has non-zero tail: %v", kind, table.itemHidden.Load()))
			}
		} else {
			// prunable tables have to have the same length
			if err := table.truncateTail(prunedTail); err != nil {
				return err
			}
		}
	}

	f.frozen.Store(head)
	f.tail.Store(prunedTail)
	return nil
}

// convertLegacyFn takes a raw freezer entry in an older format and
// returns it in the new format.
type convertLegacyFn = func([]byte) ([]byte, error)

// MigrateTable processes the entries in a given table in sequence
// converting them to a new format if they're of an old format.
func (f *Freezer) MigrateTable(kind string, convert convertLegacyFn) error {
	if f.readonly {
		return errReadOnly
	}
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	table, ok := f.tables[kind]
	if !ok {
		return errUnknownTable
	}
	// forEach iterates every entry in the table serially and in order, calling `fn`
	// with the item as argument. If `fn` returns an error the iteration stops
	// and that error will be returned.
	forEach := func(t *freezerTable, offset uint64, fn func(uint64, []byte) error) error {
		var (
			items     = t.items.Load()
			batchSize = uint64(1024)
			maxBytes  = uint64(1024 * 1024)
		)
		for i := offset; i < items; {
			if i+batchSize > items {
				batchSize = items - i
			}
			data, err := t.RetrieveItems(i, batchSize, maxBytes)
			if err != nil {
				return err
			}
			for j, item := range data {
				if err := fn(i+uint64(j), item); err != nil {
					return err
				}
			}
			i += uint64(len(data))
		}
		return nil
	}
	// TODO(s1na): This is a sanity-check since as of now no process does tail-deletion. But the migration
	// process assumes no deletion at tail and needs to be modified to account for that.
	if table.itemOffset.Load() > 0 || table.itemHidden.Load() > 0 {
		return errors.New("migration not supported for tail-deleted freezers")
	}
	ancientsPath := filepath.Dir(table.index.Name())
	// Set up new dir for the migrated table, the content of which
	// we'll at the end move over to the ancients dir.
	migrationPath := filepath.Join(ancientsPath, "migration")
	newTable, err := newFreezerTable(migrationPath, kind, table.config, false)
	if err != nil {
		return err
	}
	var (
		batch  = newTable.newBatch(f.offset.Load())
		out    []byte
		start  = time.Now()
		logged = time.Now()
		offset = newTable.items.Load()
	)
	if offset > 0 {
		log.Info("found previous migration attempt", "migrated", offset)
	}
	// Iterate through entries and transform them
	if err := forEach(table, offset, func(i uint64, blob []byte) error {
		if i%10000 == 0 && time.Since(logged) > 16*time.Second {
			log.Info("Processing legacy elements", "count", i, "elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
		out, err = convert(blob)
		if err != nil {
			return err
		}
		if err := batch.AppendRaw(i, out); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	if err := batch.commit(); err != nil {
		return err
	}
	log.Info("Replacing old table files with migrated ones", "elapsed", common.PrettyDuration(time.Since(start)))
	// Release and delete old table files. Note this won't
	// delete the index file.
	table.releaseFilesAfter(0, true)

	if err := newTable.Close(); err != nil {
		return err
	}
	files, err := os.ReadDir(migrationPath)
	if err != nil {
		return err
	}
	// Move migrated files to ancients dir.
	for _, f := range files {
		// This will replace the old index file as a side-effect.
		if err := os.Rename(filepath.Join(migrationPath, f.Name()), filepath.Join(ancientsPath, f.Name())); err != nil {
			return err
		}
	}
	// Delete by now empty dir.
	if err := os.Remove(migrationPath); err != nil {
		return err
	}
	return nil
}
