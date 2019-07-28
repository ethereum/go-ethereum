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
	"math"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

type (
	prepareCallback func(*types.Block)              // The callback for customized prepare operation.
	actionCallback  func(ethdb.Batch, *types.Block) // The callback for customized action.
)

// iterateCanonicalChain iterates the specified range canonical chain and then
// apply the given action callback.
// Note both for forward and backward iteration, the range is [from, to).
func iterateCanonicalChain(db ethdb.Database, from uint64, to uint64, typ string, prepare prepareCallback, action actionCallback, reverse bool, report bool) error {
	// Short circuit if the action is nil.
	if action == nil {
		return nil
	}
	// Short circuit if the iteration range is invalid.
	if from >= to {
		return nil
	}
	// Spawn multi-routines, iterate over the specified blocks and invoke prepare
	// callback concurrently.
	var (
		number  int64
		results = make(chan *types.Block, 4*runtime.NumCPU())
	)
	if !reverse {
		number = int64(from - 1)
	} else {
		number = int64(to)
	}
	abort := make(chan struct{})
	defer close(abort)

	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for {
				// Fetch the next task number, terminating if everything's done
				var n int64
				if !reverse {
					n = atomic.AddInt64(&number, 1)
					if n >= int64(to) {
						return
					}
				} else {
					n = atomic.AddInt64(&number, -1)
					if n < int64(from) {
						return
					}
				}
				block := ReadBlock(db, ReadCanonicalHash(db, uint64(n)), uint64(n))
				if prepare != nil && block != nil {
					prepare(block)
				}
				// Feed the block to the aggregator, or abort on interrupt
				select {
				case results <- block:
				case <-abort:
					return
				}
			}
		}()
	}
	// Reassemble the blocks into a contiguous stream and apply the action callback.
	var (
		next, first, last int64
		queue             = prque.New(nil)

		batch  = db.NewBatch()
		start  = time.Now()
		logged time.Time
	)
	if !reverse {
		next, first, last = int64(from), int64(from), int64(to)
	} else {
		next, first, last = int64(to-1), int64(to-1), int64(from-1)
	}
	logFn := log.Debug
	if report {
		logFn = log.Info
	}
	for i := from; i < to; i++ {
		// Retrieve the next result and bail if it's nil
		block := <-results
		if block == nil {
			return errors.New("broken database")
		}
		// Push the block into the import queue and process contiguous ranges
		priority := -int64(block.NumberU64())
		if reverse {
			priority = int64(block.NumberU64())
		}
		queue.Push(block, priority)
		for !queue.Empty() {
			// If the next available item is gapped, return
			if _, priority := queue.Peek(); !reverse && -priority != next || reverse && priority != next {
				break
			}
			// Next block available, pop it off and index it
			block = queue.PopItem().(*types.Block)

			if !reverse {
				next++
			} else {
				next--
			}
			// Invoke action to inject specified data into key-value database.
			action(batch, block)

			// If enough data was accumulated in memory or we're at the last block, dump to disk
			if batch.ValueSize() > ethdb.IdealBatchSize || next == last {
				if err := batch.Write(); err != nil {
					return err
				}
				batch.Reset()
			}
			// If we've spent too much time already, notify the user of what we're doing
			if time.Since(logged) > 8*time.Second {
				logFn("Iterating canonical chain", "type", typ, "reserve", reverse, "number", block.Number(), "hash", block.Hash(), "total", int64(math.Abs(float64(next-first))), "elapsed", common.PrettyDuration(time.Since(start)))
				logged = time.Now()
			}
		}
	}
	logFn("Iterated canonical chain", "type", typ, "reverse", reverse, "total", to-from, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// InitBlockIndexFromFreezer reinitializes an empty database from a previous batch
// of frozen ancient blocks. The method iterates over all the frozen blocks and
// injects into the database the block hash->number mappings.
func InitBlockIndexFromFreezer(db ethdb.Database) {
	// If we can't access the freezer or it's empty, abort
	frozen, err := db.Ancients()
	if err != nil || frozen == 0 {
		return
	}
	// hashBlock calculates block hash in advance using the multi-routine's concurrent
	// computing power.
	hashBlock := func(block *types.Block) { block.Hash() }

	// writeIndex injects hash <-> number mapping into the database.
	writeIndex := func(batch ethdb.Batch, block *types.Block) { WriteHeaderNumber(batch, block.Hash(), block.NumberU64()) }

	if err := iterateCanonicalChain(db, 0, frozen, "blocks", hashBlock, writeIndex, false, true); err != nil {
		log.Crit("Failed to iterate canonical chain", "err", err)
	}
	hash := ReadCanonicalHash(db, frozen-1)
	WriteHeadHeaderHash(db, hash)
	WriteHeadFastBlockHash(db, hash)
	log.Info("Initialized chain from ancient data", "number", frozen-1, "hash", hash)
}

// IndexTxLookup initializes txlookup indices of the specified range blocks into
// the database.
//
// This function iterates canonical chain in reverse order, it has one main advantage:
// We can write tx index tail flag periodically even without the whole indexing
// procedure is finished. So that we can resume indexing procedure next time quickly.
func IndexTxLookup(db ethdb.Database, from uint64, to uint64) {
	// hashTxs calculates transaction hash in advance using the multi-routine's
	// concurrent computing power.
	hashTxs := func(block *types.Block) {
		for _, tx := range block.Transactions() {
			tx.Hash()
		}
	}
	// writeIndices injects txlookup indices into the database.
	writeIndices := func(batch ethdb.Batch, block *types.Block) {
		WriteTxLookupEntries(batch, block)
		if block.NumberU64() == to-1 || block.NumberU64()%10000 == 0 {
			WriteTxIndexTail(batch, block.NumberU64())
		}
	}
	start := time.Now()
	if err := iterateCanonicalChain(db, from, to, "txlookup", hashTxs, writeIndices, true, true); err != nil {
		log.Crit("Failed to iterate canonical chain", "err", err)
	}
	WriteTxIndexTail(db, from)
	log.Info("Constructed transaction indices", "from", from, "to", to, "count", to-from, "elapsed", common.PrettyDuration(time.Since(start)))
}

// RemoveTxsLookup removes txlookup indices of the specified range blocks.
func RemoveTxsLookup(db ethdb.Database, from uint64, to uint64) {
	// Write flag first and then unindex the transaction indices. Some indices
	// will be left in the database if crash happens but it's fine.
	WriteTxIndexTail(db, to)

	if from+1 == to {
		hash := ReadCanonicalHash(db, from)
		DeleteTxLookupEntries(db, ReadBlock(db, hash, from))
		log.Debug("Removed transaction indices", "number", from, "hash", hash)
	} else {
		deleteIndices := func(batch ethdb.Batch, block *types.Block) { DeleteTxLookupEntries(batch, block) }
		if err := iterateCanonicalChain(db, from, to, "txlookup", nil, deleteIndices, false, false); err != nil {
			log.Crit("Failed to iterate canonical chain", "err", err)
		}
		log.Debug("Removed transaction indices", "from", from, "to", to, "count", to-from)
	}
}
