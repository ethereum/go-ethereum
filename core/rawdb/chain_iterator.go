// Copyright 2020 The go-ethereum Authors
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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// InitDatabaseFromFreezer reinitializes an empty database from a previous batch
// of frozen ancient blocks. The method iterates over all the frozen blocks and
// injects into the database the block hash->number mappings.
func InitDatabaseFromFreezer(db ethdb.Database) {
	// If we can't access the freezer or it's empty, abort
	frozen, err := db.Ancients()
	if err != nil || frozen == 0 {
		return
	}
	var (
		batch  = db.NewBatch()
		start  = time.Now()
		logged = start.Add(-7 * time.Second) // Unindex during import is fast, don't double log
		hash   common.Hash
	)
	for i := uint64(0); i < frozen; {
		// We read 100K hashes at a time, for a total of 3.2M
		count := uint64(100_000)
		if i+count > frozen {
			count = frozen - i
		}
		data, err := db.AncientRange(ChainFreezerHashTable, i, count, 32*count)
		if err != nil {
			log.Crit("Failed to init database from freezer", "err", err)
		}
		for j, h := range data {
			number := i + uint64(j)
			hash = common.BytesToHash(h)
			WriteHeaderNumber(batch, hash, number)
			// If enough data was accumulated in memory or we're at the last block, dump to disk
			if batch.ValueSize() > ethdb.IdealBatchSize {
				if err := batch.Write(); err != nil {
					log.Crit("Failed to write data to db", "err", err)
				}
				batch.Reset()
			}
		}
		i += uint64(len(data))
		// If we've spent too much time already, notify the user of what we're doing
		if time.Since(logged) > 8*time.Second {
			log.Info("Initializing database from freezer", "total", frozen, "number", i, "hash", hash, "elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write data to db", "err", err)
	}
	batch.Reset()

	WriteHeadHeaderHash(db, hash)
	WriteHeadFastBlockHash(db, hash)
	log.Info("Initialized database from freezer", "blocks", frozen, "elapsed", common.PrettyDuration(time.Since(start)))
}

// PruneTransactionIndex removes all tx index entries below a certain block number.
func PruneTransactionIndex(db ethdb.Database, pruneBlock uint64) {
	tail := ReadTxIndexTail(db)
	if tail == nil || *tail > pruneBlock {
		return // no index, or index ends above pruneBlock
	}
	// There are blocks below pruneBlock in the index. Iterate the entire index to remove
	// their entries. Note if this fails, the index is messed up, but tail still points to
	// the old tail.
	var count, removed int
	DeleteAllTxLookupEntries(db, func(txhash common.Hash, v []byte) bool {
		count++
		if count%10000000 == 0 {
			log.Info("Pruning tx index", "count", count, "removed", removed)
		}
		bn := DecodeTxLookupEntry(v, db)
		if *bn < pruneBlock {
			removed++
			return true
		}
		return false
	})
	WriteTxIndexTail(db, pruneBlock)
}
