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

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

type (
	initPrepare func(*types.Block)              // The callback for customized prepare operation.
	initAction  func(ethdb.Batch, *types.Block) // The callback for customized initialisation action.
)

// iterateAncient iterates the specified range blocks from ancient database
// and then apply initialisation action.
func iterateAncient(db ethdb.Database, from uint64, typ string, prepare initPrepare, action initAction) error {
	// Short circuit if the init action is nil.
	if action == nil {
		return nil
	}
	// If we can't access the freezer or it's empty, abort
	frozen, err := db.Ancients()
	if err != nil || frozen == 0 {
		return err
	}
	// Spawn multi-routines, iterate over the specified blocks and invoke prepare
	// callback concurrently.
	var (
		number  uint64
		results = make(chan *types.Block, 4*runtime.NumCPU())
	)
	if from == 0 {
		number = ^uint64(0) // -1
	} else {
		number = from - 1
	}
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
				// number from the freezer).
				block := ReadBlock(db, common.Hash{}, n)
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
		queue = prque.New(nil)
		next  = int64(from)

		batch  = db.NewBatch()
		start  = time.Now()
		logged time.Time
	)
	for i := from; i < frozen; i++ {
		// Retrieve the next result and bail if it's nil
		block := <-results
		if block == nil {
			return errors.New("broken database")
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

			// Invoke action to inject specified data into key-value database.
			action(batch, block)

			// If enough data was accumulated in memory or we're at the last block, dump to disk
			if batch.ValueSize() > ethdb.IdealBatchSize || uint64(next) == frozen {
				if err := batch.Write(); err != nil {
					return err
				}
				batch.Reset()
			}
			// If we've spent too much time already, notify the user of what we're doing
			if time.Since(logged) > 8*time.Second {
				log.Info("Initializing chain from ancient data", "type", typ, "number", block.Number(), "hash", block.Hash(), "total", uint64(next)-from, "elapsed", common.PrettyDuration(time.Since(start)))
				logged = time.Now()
			}
		}
	}
	log.Info("Initialized chain from ancient data", "type", typ, "number", frozen-from, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// InitBlockIndexFromFreezer reinitializes an empty database from a previous batch
// of frozen ancient blocks. The method iterates over all the frozen blocks and
// injects into the database the block hash->number mappings and the transaction
// lookup entries.
func InitBlockIndexFromFreezer(db ethdb.Database) error {
	// If we can't access the freezer or it's empty, abort
	frozen, err := db.Ancients()
	if err != nil || frozen == 0 {
		return err
	}
	// hashBlock calculates block hash in advance using the multi-routine's concurrent
	// computing power.
	hashBlock := func(block *types.Block) { block.Hash() }

	// writeIndex injects hash <-> number mapping into the database.
	writeIndex := func(batch ethdb.Batch, block *types.Block) { WriteHeaderNumber(batch, block.Hash(), block.NumberU64()) }

	if err := iterateAncient(db, 0, "blocks", hashBlock, writeIndex); err != nil {
		return err
	}
	hash := ReadCanonicalHash(db, frozen-1)
	WriteHeadHeaderHash(db, hash)
	WriteHeadFastBlockHash(db, hash)
	return nil
}

// InitTxsLookupFromFreezer initializes txlookup indexes in the database.
func InitTxsLookupFromFreezer(db ethdb.Database, from uint64) error {
	// hashTxs calculates transaction hash in advance using the multi-routine's
	// concurrent computing power.
	hashTxs := func(block *types.Block) {
		for _, tx := range block.Transactions() {
			tx.Hash()
		}
	}
	// writeIndex injects txlookup indexes into the database.
	writeIndex := func(batch ethdb.Batch, block *types.Block) {
		WriteTxLookupEntries(batch, block)
		if block.NumberU64()%10000 == 0 {
			WriteAncientTxLookupProgress(batch, block.NumberU64())
		}
	}
	if err := iterateAncient(db, from, "txlookup", hashTxs, writeIndex); err != nil {
		return err
	}
	DeleteAncientTxLookupProgress(db) // Mark all txlookup indexes of ancient blocks have been inserted.
	return nil
}
