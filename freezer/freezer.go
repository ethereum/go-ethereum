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

// Package freezer implements an append-only immutable mmap chain database.
package freezer

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

// Freezer is an memory mapped append-only database to store immutable chain data
// into flat files:
//
// - The append only nature ensures that disk writes are minimized.
// - The memory mapping ensures we can max out system memory for caching without
//   reserving it for go-ethereum. This would also reduce the memory requirements
//   of Geth, and thus also GC overhead.
type Freezer struct {
	frozen uint64 // Number of blocks already frozen

	headers  *table // Data table for storing the block headers
	bodies   *table // Data table for storing the block bodies
	receipts *table // Data table for storing the block receipts
	diffs    *table // Data table for storing the block tds

	logger log.Logger // Contextual logger for the freezer database
}

// New creates a chain freezer that moves ancient chain data into immutable flat
// file containers.
func New(datadir string) (*Freezer, error) {
	// Create the initial freezer object
	var (
		freezer = &Freezer{
			logger: log.New("path", datadir),
		}
		readMeter  = metrics.NewRegisteredMeter("eth/db/freezer/read", nil)
		writeMeter = metrics.NewRegisteredMeter("eth/db/freezer/write", nil)
		err        error
	)
	// Open all the supported data tables
	if freezer.headers, err = newTable(datadir, "headers", readMeter, writeMeter); err != nil {
		return nil, err
	}
	if freezer.bodies, err = newTable(datadir, "bodies", readMeter, writeMeter); err != nil {
		freezer.headers.Close()
		return nil, err
	}
	if freezer.receipts, err = newTable(datadir, "receipts", readMeter, writeMeter); err != nil {
		freezer.bodies.Close()
		freezer.headers.Close()
		return nil, err
	}
	if freezer.diffs, err = newTable(datadir, "diffs", readMeter, writeMeter); err != nil {
		freezer.receipts.Close()
		freezer.bodies.Close()
		freezer.headers.Close()
		return nil, err
	}
	return freezer, nil
}

// Close terminates the chain freezer, unmapping all the data files.
func (f *Freezer) Close() error {
	f.diffs.Close()
	f.receipts.Close()
	f.bodies.Close()
	f.headers.Close()

	return nil
}

// Freeze is a background thread that periodically checks the blockchain for any
// import progress and moves ancient data from the fast database into the freezer.
//
// This functionality is deliberately broken off from block importing to avoid
// incurring additional data shuffling delays on block propagation.
func (f *Freezer) Freeze(chain *core.BlockChain, db ethdb.Database, recheck time.Duration, delay uint64) {
	for {
		// Retrieve the freezing threshold. In theory we're interested only in full
		// blocks post-sync, but that would keep the live database enormous during
		// dast sync. By picking the fast block, we still get to deep freeze all the
		// final immutable data without having to wait for sync to finish.
		head := chain.CurrentFastBlock()
		if head == nil {
			log.Error("Current fast block is nil")
			time.Sleep(recheck)
			continue
		}
		if head.NumberU64() < delay {
			log.Debug("Current block not old enough", "number", head.Number(), "hash", head.Hash(), "age", common.PrettyAge(time.Unix(head.Time().Int64(), 0)), "delay", delay)
			time.Sleep(recheck)
			continue
		}
		limit := head.NumberU64() - delay
		if limit <= f.frozen {
			log.Debug("Ancient blocks frozen already")
			time.Sleep(recheck)
			continue
		}
		// Seems we have data ready to be frozen, process in usable batches
		if limit-f.frozen > 30000 {
			limit = f.frozen + 30000
		}
		var (
			start = time.Now()
			first = f.frozen
			last  *types.Block
		)
		for f.frozen < limit {
			// Deep freeze the next canonical block if it's available
			if block := chain.GetBlockByNumber(f.frozen); block != nil {
				// Deep freeze the block header and body
				blob, _ := rlp.EncodeToBytes(block.Header())
				if err := f.headers.Append(f.frozen, blob); err != nil {
					log.Error("Failed to deep freeze header", "number", block.Number(), "hash", block.Hash(), "age", common.PrettyAge(time.Unix(block.Time().Int64(), 0)), "err", err)
					break
				}
				blob, _ = rlp.EncodeToBytes(block.Body())
				if err := f.bodies.Append(f.frozen, blob); err != nil {
					log.Error("Failed to deep freeze body", "number", block.Number(), "hash", block.Hash(), "age", common.PrettyAge(time.Unix(block.Time().Int64(), 0)), "err", err)
					break
				}
				// Deep freeze the block receipts and total difficulty
				if receipts := chain.GetReceiptsByHash(block.Hash()); receipts != nil {
					blob, _ = rlp.EncodeToBytes(receipts)
					if err := f.receipts.Append(f.frozen, blob); err != nil {
						log.Error("Failed to deep freeze receipts", "number", block.Number(), "hash", block.Hash(), "age", common.PrettyAge(time.Unix(block.Time().Int64(), 0)), "err", err)
						break
					}
				}
				if td := chain.GetTd(block.Hash(), block.NumberU64()); td != nil {
					blob, _ = rlp.EncodeToBytes(td)
					if err := f.diffs.Append(f.frozen, blob); err != nil {
						log.Error("Failed to deep freeze difficulty", "number", block.Number(), "hash", block.Hash(), "age", common.PrettyAge(time.Unix(block.Time().Int64(), 0)), "err", err)
						break
					}
				}
				log.Trace("Deep froze ancient block", "number", block.Number(), "hash", block.Hash(), "age", common.PrettyAge(time.Unix(block.Time().Int64(), 0)))
				f.frozen++

				// If it's the last block, save for reporting
				if f.frozen == limit-1 {
					last = block
				}
			}
		}
		// Batch of blocks have been frozen, flush them before wiping from leveldb
		if err := f.headers.Flush(); err != nil {
			f.logger.Error("Failed to flush frozen headers", "err", err)
			time.Sleep(recheck)
			continue
		}
		if err := f.bodies.Flush(); err != nil {
			f.logger.Error("Failed to flush frozen bodies", "err", err)
			time.Sleep(recheck)
			continue
		}
		if err := f.receipts.Flush(); err != nil {
			f.logger.Error("Failed to flush frozen receipts", "err", err)
			time.Sleep(recheck)
			continue
		}
		if err := f.diffs.Flush(); err != nil {
			f.logger.Error("Failed to flush frozen diffs", "err", err)
			time.Sleep(recheck)
			continue
		}
		// Wipe out all data from the active database
		for number := first; number < f.frozen; number++ {
			if number == 0 {
				// Skip deleting the genesis for the PoC
				continue
			}
			rawdb.DeleteBlock(db, rawdb.ReadCanonicalHash(db, number), number)
		}
		// Log something friendly for the user
		context := []interface{}{
			"count", f.frozen - first, "elapsed", common.PrettyDuration(time.Since(start)), "number", f.frozen - 1,
		}
		if last != nil {
			context = append(context, []interface{}{"hash", last.Hash(), "age", common.PrettyAge(time.Unix(last.Time().Int64(), 0))}...)
		}
		log.Info("Deep froze chain segment", context...)

		// Avoid database thrashing with tiny writes
		if f.frozen-first < 30000 {
			time.Sleep(recheck)
		}
	}
}
