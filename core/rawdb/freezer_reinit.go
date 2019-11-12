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
	"runtime"
	"sync/atomic"
	"time"

	"github.com/maticnetwork/bor/common"
	"github.com/maticnetwork/bor/common/prque"
	"github.com/maticnetwork/bor/core/types"
	"github.com/maticnetwork/bor/ethdb"
	"github.com/maticnetwork/bor/log"
)

// InitDatabaseFromFreezer reinitializes an empty database from a previous batch
// of frozen ancient blocks. The method iterates over all the frozen blocks and
// injects into the database the block hash->number mappings and the transaction
// lookup entries.
func InitDatabaseFromFreezer(db ethdb.Database) error {
	// If we can't access the freezer or it's empty, abort
	frozen, err := db.Ancients()
	if err != nil || frozen == 0 {
		return err
	}
	// Blocks previously frozen, iterate over- and hash them concurrently
	var (
		number  = ^uint64(0) // -1
		results = make(chan *types.Block, 4*runtime.NumCPU())
	)
	abort := make(chan struct{})
	defer close(abort)

	for i := 0; i < runtime.NumCPU(); i++ {
		go func() {
			for {
				// Fetch the next task number, terminating if everything's done
				n := atomic.AddUint64(&number, 1)
				if n >= frozen {
					return
				}
				// Retrieve the block from the freezer (no need for the hash, we pull by
				// number from the freezer). If successful, pre-cache the block hash and
				// the individual transaction hashes for storing into the database.
				block := ReadBlock(db, common.Hash{}, n)
				if block != nil {
					block.Hash()
					for _, tx := range block.Transactions() {
						tx.Hash()
					}
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
	// Reassemble the blocks into a contiguous stream and push them out to disk
	var (
		queue = prque.New(nil)
		next  = int64(0)

		batch  = db.NewBatch()
		start  = time.Now()
		logged time.Time
	)
	for i := uint64(0); i < frozen; i++ {
		// Retrieve the next result and bail if it's nil
		block := <-results
		if block == nil {
			return errors.New("broken ancient database")
		}
		// Push the block into the import queue and process contiguous ranges
		queue.Push(block, -int64(block.NumberU64()))
		for !queue.Empty() {
			// If the next available item is gapped, return
			if _, priority := queue.Peek(); -priority != next {
				break
			}
			// Next block available, pop it off and index it
			block = queue.PopItem().(*types.Block)
			next++

			// Inject hash<->number mapping and txlookup indexes
			WriteHeaderNumber(batch, block.Hash(), block.NumberU64())
			WriteTxLookupEntries(batch, block)

			// If enough data was accumulated in memory or we're at the last block, dump to disk
			if batch.ValueSize() > ethdb.IdealBatchSize || uint64(next) == frozen {
				if err := batch.Write(); err != nil {
					return err
				}
				batch.Reset()
			}
			// If we've spent too much time already, notify the user of what we're doing
			if time.Since(logged) > 8*time.Second {
				log.Info("Initializing chain from ancient data", "number", block.Number(), "hash", block.Hash(), "total", frozen-1, "elapsed", common.PrettyDuration(time.Since(start)))
				logged = time.Now()
			}
		}
	}
	hash := ReadCanonicalHash(db, frozen-1)
	WriteHeadHeaderHash(db, hash)
	WriteHeadFastBlockHash(db, hash)

	log.Info("Initialized chain from ancient data", "number", frozen-1, "hash", hash, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}
