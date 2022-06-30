// Copyright 2022 The go-ethereum Authors
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
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// throwerRecheckInterval is the frequency to check the key-value database for
	// chain progression that might permit new blocks to be frozen into immutable
	// storage.
	throwerRecheckInterval = time.Minute

	// throwerBatchLimit is the maximum number of blocks to freeze in one batch
	// before doing an fsync and deleting it from the key-value store.
	throwerBatchLimit = 30000
)

// chainThrower is a wrapper of freezer with additional chain freezing feature.
// The background thread will keep moving ancient chain segments from key-value
// database to flat files for saving space on live database.
type chainThrower struct {
	throwdb

	// WARNING: The `threshold` field is accessed atomically. On 32 bit platforms, only
	// 64-bit aligned fields can be atomic. The struct is guaranteed to be so aligned,
	// so take advantage of that (https://golang.org/pkg/sync/atomic/#pkg-note-BUG).
	threshold uint64 // Number of recent blocks not to freeze (params.FullImmutabilityThreshold apart from tests)

	quit    chan struct{}
	wg      sync.WaitGroup
	trigger chan chan struct{} // Manual blocking freeze trigger, test determinism
}

// throwdb is a database wrapper that disables freezer data retrievals.
type throwdb struct {
	ethdb.KeyValueStore
}

// HasAncient always returns false as `throwdb` has thrown everything written to it
func (db *throwdb) HasAncient(kind string, number uint64) (bool, error) {
	return false, nil
}

// Ancient returns an empty result as `throwdb` has thrown everything written to it
func (db *throwdb) Ancient(kind string, number uint64) ([]byte, error) {
	return []byte{}, nil
}

// AncientRange returns an empty result as `throwdb` has thrown everything written to it
func (db *throwdb) AncientRange(kind string, start, max, maxByteSize uint64) ([][]byte, error) {
	return [][]byte{}, nil
}

// Ancients returns 0 as we don't have a backing chain freezer.
func (db *throwdb) Ancients() (uint64, error) {
	return 0, nil
}

// Tail returns 0 as we don't have a backing chain freezer.
func (db *throwdb) Tail() (uint64, error) {
	return 0, errNotSupported
}

// AncientSize returns an error as we don't have a backing chain freezer.
func (db *throwdb) AncientSize(kind string) (uint64, error) {
	return 0, nil
}

// ModifyAncients is not supported.
func (db *throwdb) ModifyAncients(func(ethdb.AncientWriteOp) error) (int64, error) {
	return 0, nil
}

// TruncateHead returns an error as we don't have a backing chain freezer.
func (db *throwdb) TruncateHead(items uint64) error {
	return nil
}

// TruncateTail returns an error as we don't have a backing chain freezer.
func (db *throwdb) TruncateTail(items uint64) error {
	return nil
}

// Sync returns an error as we don't have a backing chain freezer.
func (db *throwdb) Sync() error {
	return nil
}

func (db *throwdb) ReadAncients(fn func(reader ethdb.AncientReaderOp) error) (err error) {
	// Unlike other ancient-related methods, this method does not return
	// errNotSupported when invoked.
	// The reason for this is that the caller might want to do several things:
	// 1. Check if something is in freezer,
	// 2. If not, check leveldb.
	//
	// This will work, since the ancient-checks inside 'fn' will return errors,
	// and the leveldb work will continue.
	//
	// If we instead were to return errNotSupported here, then the caller would
	// have to explicitly check for that, having an extra clause to do the
	// non-ancient operations.
	return fn(db)
}

// MigrateTable processes the entries in a given table in sequence
// converting them to a new format if they're of an old format.
func (db *throwdb) MigrateTable(kind string, convert convertLegacyFn) error {
	return errNotSupported
}

// AncientDatadir returns an error as we don't have a backing chain freezer.
func (db *throwdb) AncientDatadir() (string, error) {
	return "", errNotSupported
}

// newChainFreezer initializes the freezer for ancient chain data.
func newChainThrower(datadir string, namespace string, readonly bool) (*chainThrower, error) {
	return &chainThrower{
		threshold: params.FullImmutabilityThreshold,
		quit:      make(chan struct{}),
		trigger:   make(chan chan struct{}),
	}, nil
}

// Close closes the chain freezer instance and terminates the background thread.
func (f *chainThrower) Close() error {
	select {
	case <-f.quit:
	default:
		close(f.quit)
	}
	f.wg.Wait()
	return nil
}

// freeze is a background thread that periodically checks the blockchain for any
// import progress and moves ancient data from the fast database into the freezer.
//
// This functionality is deliberately broken off from block importing to avoid
// incurring additional data shuffling delays on block propagation.
func (f *chainThrower) throw(db ethdb.KeyValueStore) {
	nfdb := &throwdb{KeyValueStore: db}

	var (
		backoff   bool
		triggered chan struct{} // Used in tests
	)
	for {
		select {
		case <-f.quit:
			log.Info("Thrower shutting down")
			return
		default:
		}
		if backoff {
			// If we were doing a manual trigger, notify it
			if triggered != nil {
				triggered <- struct{}{}
				triggered = nil
			}
			select {
			case <-time.NewTimer(throwerRecheckInterval).C:
				backoff = false
			case triggered = <-f.trigger:
				backoff = false
			case <-f.quit:
				return
			}
		}
		// Retrieve the freezing threshold.
		hash := ReadHeadBlockHash(nfdb)
		if hash == (common.Hash{}) {
			log.Debug("Current full block hash unavailable") // new chain, empty database
			backoff = true
			continue
		}
		number := ReadHeaderNumber(nfdb, hash)
		threshold := atomic.LoadUint64(&f.threshold)
		last := ReadAncientThrowLastBlockNumber(db)
		if last == nil {
			last = new(uint64)
		}
		switch {
		case number == nil:
			log.Error("Current full block number unavailable", "hash", hash)
			backoff = true
			continue

		case *number < threshold:
			log.Debug("Current full block not old enough", "number", *number, "hash", hash, "delay", threshold)
			backoff = true
			continue

		case *number-threshold <= *last:
			log.Debug("Ancient blocks frozen already", "number", *number, "hash", hash, "last", last)
			backoff = true
			continue
		}

		storedSections := ReadStoredBloomSections(nfdb)
		if storedSections*params.BloomBitsBlocks-1 < *last {
			log.Warn("Attempt to prune the ancient blocks that bloom filter haven't finished yet, postpone to next round", "storedSections", storedSections, "pruneTo", *last)
			return
		}

		head := ReadHeader(nfdb, hash, *number)
		if head == nil {
			log.Error("Current full block unavailable", "number", *number, "hash", hash)
			backoff = true
			continue
		}

		// Seems we have data ready to be frozen, process in usable batches
		var (
			start = time.Now()
			first = *last
			limit = *number - threshold
		)
		if limit-first > freezerBatchLimit {
			limit = first + freezerBatchLimit
		}
		log.Info("schedule throwing blocks", "from", first, "to", limit)
		ancients, err := f.throwRange(nfdb, first, limit)
		if err != nil {
			log.Error("Error in block freeze operation", "err", err)
			backoff = true
			continue
		}

		// Wipe out all data from the active database
		batch := db.NewBatch()
		for i := 0; i < len(ancients); i++ {
			// Always keep the genesis block in active database
			if first+uint64(i) != 0 {
				DeleteBlockWithoutNumber(batch, ancients[i], first+uint64(i))
				DeleteCanonicalHash(batch, first+uint64(i))
			}
		}
		if err := batch.Write(); err != nil {
			log.Crit("Failed to throw ancient blocks", "err", err)
		}
		batch.Reset()

		// record
		WriteAncientThrowLastBlockNumber(db, limit)

		// Wipe out side chains also and track dangling side chains
		var dangling []common.Hash
		for number := first; number < limit; number++ {
			// Always keep the genesis block in active database
			if number != 0 {
				dangling = ReadAllHashes(db, number)
				for _, hash := range dangling {
					log.Trace("Deleting side chain", "number", number, "hash", hash)
					DeleteBlock(batch, hash, number)
				}
			}
		}
		if err := batch.Write(); err != nil {
			log.Crit("Failed to delete frozen side blocks", "err", err)
		}
		batch.Reset()

		// Step into the future and delete and dangling side chains
		if limit > 0 {
			tip := limit
			for len(dangling) > 0 {
				drop := make(map[common.Hash]struct{})
				for _, hash := range dangling {
					log.Debug("Dangling parent from Freezer", "number", tip-1, "hash", hash)
					drop[hash] = struct{}{}
				}
				children := ReadAllHashes(db, tip)
				for i := 0; i < len(children); i++ {
					// Dig up the child and ensure it's dangling
					child := ReadHeader(nfdb, children[i], tip)
					if child == nil {
						log.Error("Missing dangling header", "number", tip, "hash", children[i])
						continue
					}
					if _, ok := drop[child.ParentHash]; !ok {
						children = append(children[:i], children[i+1:]...)
						i--
						continue
					}
					// Delete all block data associated with the child
					log.Debug("Deleting dangling block", "number", tip, "hash", children[i], "parent", child.ParentHash)
					DeleteBlock(batch, children[i], tip)
				}
				dangling = children
				tip++
			}
			if err := batch.Write(); err != nil {
				log.Crit("Failed to delete dangling side blocks", "err", err)
			}
		}

		// Log something friendly for the user
		context := []interface{}{
			"blocks", limit - first, "elapsed", common.PrettyDuration(time.Since(start)), "number", limit - 1,
		}
		if n := len(ancients); n > 0 {
			context = append(context, []interface{}{"hash", ancients[n-1]}...)
		}
		log.Info("Deep throw chain segment", context...)

		// Avoid database thrashing with tiny writes
		if limit-first < freezerBatchLimit {
			backoff = true
		}
	}
}

func (f *chainThrower) throwRange(nfdb *throwdb, number, limit uint64) (hashes []common.Hash, err error) {
	hashes = make([]common.Hash, 0, limit-number)

	for ; number <= limit; number++ {
		// Retrieve all the components of the canonical block.
		hash := ReadCanonicalHash(nfdb, number)
		if hash == (common.Hash{}) {
			log.Error("canonical hash missing, can't freeze", "block %d", number)
			continue
		}
		header := ReadHeaderRLP(nfdb, hash, number)
		if len(header) == 0 {
			log.Error("block header missing, can't freeze", "block %d", number)
			continue
		}
		body := ReadBodyRLP(nfdb, hash, number)
		if len(body) == 0 {
			log.Error("block body missing, can't freeze", "block %d", number)
			continue
		}
		receipts := ReadReceiptsRLP(nfdb, hash, number)
		if len(receipts) == 0 {
			log.Error("block receipts missing, can't freeze", "block %d", number)
			continue
		}
		td := ReadTdRLP(nfdb, hash, number)
		if len(td) == 0 {
			log.Error("total difficulty missing, can't freeze", "block %d", number)
			continue
		}

		hashes = append(hashes, hash)
	}

	return hashes, err
}
