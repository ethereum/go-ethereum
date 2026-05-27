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
	head    atomic.Uint64             // Number of items stored (including items removed from tail)
	tails   map[string]*atomic.Uint64 // Per-group tail cache, keyed by tail group name

	// This lock synchronizes writers and the truncate operation, as well as
	// the "atomic" (batched) read operations.
	writeLock  sync.RWMutex
	writeBatch *freezerBatch

	readonly     bool
	tables       map[string]*freezerTable // Data tables for storing everything
	instanceLock *flock.Flock             // File-system lock to prevent double opens
	closeOnce    sync.Once
}

// NewFreezer creates a freezer instance for maintaining immutable ordered
// data according to the given parameters.
//
// The 'tables' argument defines the freezer tables and their configuration.
// Each value is a freezerTableConfig describing whether Snappy compression
// is disabled (noSnappy) and which tail group the table belongs to.
func NewFreezer(datadir string, namespace string, readonly bool, maxTableSize uint32, tables map[string]freezerTableConfig) (*Freezer, error) {
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
	// Leveldb/Pebble uses LOCK as the filelock filename. To prevent the
	// name collision, we use FLOCK as the lock name.
	flockFile := filepath.Join(datadir, "FLOCK")
	if err := os.MkdirAll(filepath.Dir(flockFile), 0755); err != nil {
		return nil, err
	}
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
		tails:        make(map[string]*atomic.Uint64),
		instanceLock: lock,
	}

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

	// Create the write batch.
	freezer.writeBatch = newFreezerBatch(freezer)

	log.Info("Opened ancient database", "database", datadir, "readonly", readonly)
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
	return errors.Join(errs...)
}

// AncientDatadir returns the path of the ancient store.
func (f *Freezer) AncientDatadir() (string, error) {
	return f.datadir, nil
}

// Ancient retrieves an ancient binary blob from the append-only immutable files.
func (f *Freezer) Ancient(kind string, number uint64) ([]byte, error) {
	if table := f.tables[kind]; table != nil {
		return table.Retrieve(number)
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

// AncientBytes retrieves the value segment of the element specified by the id
// and value offsets.
func (f *Freezer) AncientBytes(kind string, id, offset, length uint64) ([]byte, error) {
	if table := f.tables[kind]; table != nil {
		return table.RetrieveBytes(id, offset, length)
	}
	return nil, errUnknownTable
}

// Ancients returns the length of the frozen items.
func (f *Freezer) Ancients() (uint64, error) {
	return f.head.Load(), nil
}

// Tail returns the lowest accessible item index for the given tail group.
// All tables sharing this group agree on the tail; an empty group name
// refers to non-prunable tables and always returns 0. Unknown groups return
// an error.
func (f *Freezer) Tail(group string) (uint64, error) {
	if group == "" {
		return 0, nil
	}
	tail, ok := f.tails[group]
	if !ok {
		return 0, fmt.Errorf("unknown tail group: %q", group)
	}
	return tail.Load(), nil
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
	prevItem := f.head.Load()
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
	f.head.Store(item)
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

	oitems := f.head.Load()
	if oitems <= items {
		return oitems, nil
	}
	for _, table := range f.tables {
		if err := table.truncateHead(items); err != nil {
			return 0, err
		}
	}
	f.head.Store(items)
	return oitems, nil
}

// TruncateTail discards all data below the specified threshold across every
// table that belongs to the named tail group. Tables that are already past
// the threshold are left untouched. The previous tail of the group is
// returned. An empty group name or an unknown group name returns an error.
func (f *Freezer) TruncateTail(group string, tail uint64) (uint64, error) {
	if f.readonly {
		return 0, errReadOnly
	}
	if group == "" {
		return 0, errors.New("empty tail group")
	}
	cached, ok := f.tails[group]
	if !ok {
		return 0, fmt.Errorf("unknown tail group: %q", group)
	}
	f.writeLock.Lock()
	defer f.writeLock.Unlock()

	prev := cached.Load()
	if prev >= tail {
		return prev, nil
	}
	for _, table := range f.tables {
		if table.config.tailGroup != group {
			continue
		}
		if err := table.truncateTail(tail); err != nil {
			return 0, err
		}
	}
	cached.Store(tail)

	// Update the head if the requested tail exceeds the current head.
	if f.head.Load() < tail {
		f.head.Store(tail)
	}
	return prev, nil
}

// SyncAncient flushes all data tables to disk.
func (f *Freezer) SyncAncient() error {
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

// validate checks that every table has the same head and that tables sharing
// a tail group also share a tail. Used instead of `repair` in readonly mode.
func (f *Freezer) validate() error {
	if len(f.tables) == 0 {
		return nil
	}
	var (
		head    uint64
		headSet bool
		tails   = make(map[string]uint64)
	)
	for kind, table := range f.tables {
		// A freshly added table is empty and has not yet been aligned to the
		// common head, skip the error here.
		//
		// Tradeoff:
		// It loosens corruption detection slightly: a table that lost its data
		// and now reports items == 0 would be treated as "freshly added" rather
		// than flagged. It's the tradeoff we accept.
		items := table.items.Load()
		if items == 0 {
			continue
		}
		// Validate the table head
		if !headSet {
			head = items
			headSet = true
		} else if items != head {
			return fmt.Errorf("freezer table %s has a differing head: %d != %d", kind, items, head)
		}
		// Validate the table tail
		if table.config.tailGroup == "" {
			if table.itemHidden.Load() != 0 {
				return fmt.Errorf("non-prunable freezer table '%s' has a non-zero tail: %d", kind, table.itemHidden.Load())
			}
			continue
		}
		hidden := table.itemHidden.Load()
		if t, ok := tails[table.config.tailGroup]; ok {
			if t != hidden {
				return fmt.Errorf("freezer table %s has differing tail in group %q: %d != %d", kind, table.config.tailGroup, hidden, t)
			}
		} else {
			tails[table.config.tailGroup] = hidden
		}
	}
	f.head.Store(head)

	for group, tail := range tails {
		counter := new(atomic.Uint64)
		counter.Store(tail)
		f.tails[group] = counter
	}
	return nil
}

// repair brings every table into a consistent state. The common head is taken
// as the minimum item count among non-empty tables; freshly added empty tables
// are fast-forwarded to that head via tail truncation. Within each tail group
// the maximum tail wins, and prunable tables are truncated to it.
func (f *Freezer) repair() error {
	// Determine the common head from non-empty tables. Empty tables are
	// excluded so that a freshly added table cannot drag the existing head
	// down to zero on first cold-start.
	var (
		hasNonEmpty bool
		head        uint64 = math.MaxUint64
	)
	for _, table := range f.tables {
		if table.items.Load() == 0 {
			continue
		}
		if items := table.items.Load(); items < head {
			head = items
		}
		hasNonEmpty = true
	}
	if !hasNonEmpty {
		head = 0
	}
	// Align newly added empty tables to the common head. truncateTail
	// internally calls resetTo when the requested tail exceeds the current
	// head, which is exactly what we need here.
	if head > 0 {
		for _, table := range f.tables {
			if table.items.Load() == 0 {
				if err := table.truncateTail(head); err != nil {
					return err
				}
			}
		}
	}
	// Truncate every table to the common head.
	for _, table := range f.tables {
		if err := table.truncateHead(head); err != nil {
			return err
		}
	}
	// Per-group tail alignment: take the maximum tail in each group and apply
	// it to all members. Non-prunable tables must remain at tail 0.
	tails := make(map[string]uint64)
	for kind, table := range f.tables {
		if table.config.tailGroup == "" {
			if table.itemHidden.Load() != 0 {
				panic(fmt.Sprintf("non-prunable freezer table %s has non-zero tail: %v", kind, table.itemHidden.Load()))
			}
			continue
		}
		hidden := table.itemHidden.Load()
		if t, ok := tails[table.config.tailGroup]; !ok || hidden > t {
			tails[table.config.tailGroup] = hidden
		}
	}
	for _, table := range f.tables {
		if table.config.tailGroup == "" {
			continue
		}
		if err := table.truncateTail(tails[table.config.tailGroup]); err != nil {
			return err
		}
	}
	f.head.Store(head)

	for group, tail := range tails {
		counter := new(atomic.Uint64)
		counter.Store(tail)
		f.tails[group] = counter
	}
	return nil
}
