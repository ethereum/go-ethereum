// Copyright 2018 The go-ethereum Authors
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
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

// errUnknownTable is returned if the user attempts to read from a table that is
// not tracked by the freezer.
var errUnknownTable = errors.New("unknown table")

const (
	// freezerRecheckInterval is the frequency to check the key-value database for
	// chain progression that might permit new blocks to be frozen into immutable
	// storage.
	freezerRecheckInterval = time.Minute

	// freezerBlockGraduation is the number of confirmations a block must achieve
	// before it becomes elligible for chain freezing. This must exceed any chain
	// reorg depth, since the freezer also deletes all block siblings.
	freezerBlockGraduation = 60000

	// freezerBatchLimit is the maximum number of blocks to freeze in one batch
	// before doing an fsync and deleting it from the key-value store.
	freezerBatchLimit = 30000
)

// freezer is an memory mapped append-only database to store immutable chain data
// into flat files:
//
// - The append only nature ensures that disk writes are minimized.
// - The memory mapping ensures we can max out system memory for caching without
//   reserving it for go-ethereum. This would also reduce the memory requirements
//   of Geth, and thus also GC overhead.
type freezer struct {
	tables map[string]*freezerTable // Data tables for storing everything
	frozen uint64                   // Number of blocks already frozen
}

// newFreezer creates a chain freezer that moves ancient chain data into
// append-only flat file containers.
func newFreezer(datadir string, namespace string) (*freezer, error) {
	// Create the initial freezer object
	var (
		readMeter  = metrics.NewRegisteredMeter(namespace+"ancient/read", nil)
		writeMeter = metrics.NewRegisteredMeter(namespace+"ancient/write", nil)
	)
	// Open all the supported data tables
	freezer := &freezer{
		tables: make(map[string]*freezerTable),
	}
	for _, name := range []string{"hashes", "headers", "bodies", "receipts", "diffs"} {
		table, err := newTable(datadir, name, readMeter, writeMeter)
		if err != nil {
			for _, table := range freezer.tables {
				table.Close()
			}
			return nil, err
		}
		freezer.tables[name] = table
	}
	// Truncate all data tables to the same length
	freezer.frozen = math.MaxUint64
	for _, table := range freezer.tables {
		if freezer.frozen > table.items {
			freezer.frozen = table.items
		}
	}
	for _, table := range freezer.tables {
		if err := table.truncate(freezer.frozen); err != nil {
			for _, table := range freezer.tables {
				table.Close()
			}
			return nil, err
		}
	}
	return freezer, nil
}

// Close terminates the chain freezer, unmapping all the data files.
func (f *freezer) Close() error {
	var errs []error
	for _, table := range f.tables {
		if err := table.Close(); err != nil {
			errs = append(errs, err)
		}
	}
	if errs != nil {
		return fmt.Errorf("%v", errs)
	}
	return nil
}

// sync flushes all data tables to disk.
func (f *freezer) sync() error {
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

// Ancient retrieves an ancient binary blob from the append-only immutable files.
func (f *freezer) Ancient(kind string, number uint64) ([]byte, error) {
	if table := f.tables[kind]; table != nil {
		return table.Retrieve(number)
	}
	return nil, errUnknownTable
}

// freeze is a background thread that periodically checks the blockchain for any
// import progress and moves ancient data from the fast database into the freezer.
//
// This functionality is deliberately broken off from block importing to avoid
// incurring additional data shuffling delays on block propagation.
func (f *freezer) freeze(db ethdb.KeyValueStore) {
	nfdb := &nofreezedb{KeyValueStore: db}

	for {
		// Retrieve the freezing threshold. In theory we're interested only in full
		// blocks post-sync, but that would keep the live database enormous during
		// dast sync. By picking the fast block, we still get to deep freeze all the
		// final immutable data without having to wait for sync to finish.
		hash := ReadHeadFastBlockHash(nfdb)
		if hash == (common.Hash{}) {
			log.Debug("Current fast block hash unavailable") // new chain, empty database
			time.Sleep(freezerRecheckInterval)
			continue
		}
		number := ReadHeaderNumber(nfdb, hash)
		switch {
		case number == nil:
			log.Error("Current fast block number unavailable", "hash", hash)
			time.Sleep(freezerRecheckInterval)
			continue

		case *number < freezerBlockGraduation:
			log.Debug("Current fast block not old enough", "number", *number, "hash", hash, "delay", freezerBlockGraduation)
			time.Sleep(freezerRecheckInterval)
			continue

		case *number-freezerBlockGraduation <= f.frozen:
			log.Debug("Ancient blocks frozen already", "number", *number, "hash", hash, "frozen", f.frozen)
			time.Sleep(freezerRecheckInterval)
			continue
		}
		head := ReadHeader(nfdb, hash, *number)
		if head == nil {
			log.Error("Current fast block unavailable", "number", *number, "hash", hash)
			time.Sleep(freezerRecheckInterval)
			continue
		}
		// Seems we have data ready to be frozen, process in usable batches
		limit := *number - freezerBlockGraduation
		if limit-f.frozen > freezerBatchLimit {
			limit = f.frozen + freezerBatchLimit
		}
		var (
			start    = time.Now()
			first    = f.frozen
			ancients = make([]common.Hash, 0, limit)
		)
		for f.frozen < limit {
			// Retrieves all the components of the canonical block
			hash := ReadCanonicalHash(nfdb, f.frozen)
			if hash == (common.Hash{}) {
				log.Error("Canonical hash missing, can't freeze", "number", f.frozen)
				break
			}
			header := ReadHeaderRLP(nfdb, hash, f.frozen)
			if len(header) == 0 {
				log.Error("Block header missing, can't freeze", "number", f.frozen, "hash", hash)
				break
			}
			body := ReadBodyRLP(nfdb, hash, f.frozen)
			if len(body) == 0 {
				log.Error("Block body missing, can't freeze", "number", f.frozen, "hash", hash)
				break
			}
			receipts := ReadReceiptsRLP(nfdb, hash, f.frozen)
			if len(receipts) == 0 {
				log.Error("Block receipts missing, can't freeze", "number", f.frozen, "hash", hash)
				break
			}
			td := ReadTdRLP(nfdb, hash, f.frozen)
			if len(td) == 0 {
				log.Error("Total difficulty missing, can't freeze", "number", f.frozen, "hash", hash)
				break
			}
			// Inject all the components into the relevant data tables
			if err := f.tables["hashes"].Append(f.frozen, hash[:]); err != nil {
				log.Error("Failed to deep freeze hash", "number", f.frozen, "hash", hash, "err", err)
				break
			}
			if err := f.tables["headers"].Append(f.frozen, header); err != nil {
				log.Error("Failed to deep freeze header", "number", f.frozen, "hash", hash, "err", err)
				break
			}
			if err := f.tables["bodies"].Append(f.frozen, body); err != nil {
				log.Error("Failed to deep freeze body", "number", f.frozen, "hash", hash, "err", err)
				break
			}
			if err := f.tables["receipts"].Append(f.frozen, receipts); err != nil {
				log.Error("Failed to deep freeze receipts", "number", f.frozen, "hash", hash, "err", err)
				break
			}
			if err := f.tables["diffs"].Append(f.frozen, td); err != nil {
				log.Error("Failed to deep freeze difficulty", "number", f.frozen, "hash", hash, "err", err)
				break
			}
			log.Trace("Deep froze ancient block", "number", f.frozen, "hash", hash)
			atomic.AddUint64(&f.frozen, 1) // Only modify atomically
			ancients = append(ancients, hash)
		}
		// Batch of blocks have been frozen, flush them before wiping from leveldb
		if err := f.sync(); err != nil {
			log.Crit("Failed to flush frozen tables", "err", err)
		}
		// Wipe out all data from the active database
		batch := db.NewBatch()
		for number := first; number < f.frozen; number++ {
			for _, hash := range readAllHashes(db, number) {
				if hash == ancients[number-first] {
					deleteBlockWithoutNumber(batch, hash, number)
				} else {
					DeleteBlock(batch, hash, number)
				}
			}
		}
		if err := batch.Write(); err != nil {
			log.Crit("Failed to delete frozen items", "err", err)
		}
		// Log something friendly for the user
		context := []interface{}{
			"blocks", f.frozen - first, "elapsed", common.PrettyDuration(time.Since(start)), "number", f.frozen - 1,
		}
		if n := len(ancients); n > 0 {
			context = append(context, []interface{}{"hash", ancients[n-1]}...)
		}
		log.Info("Deep froze chain segment", context...)

		// Avoid database thrashing with tiny writes
		if f.frozen-first < freezerBatchLimit {
			time.Sleep(freezerRecheckInterval)
		}
	}
}
