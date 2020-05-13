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
	"runtime"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"golang.org/x/crypto/sha3"
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
	for i := uint64(0); i < frozen; i++ {
		// Since the freezer has all data in sequential order on a file,
		// it would be 'neat' to read more data in one go, and let the
		// freezerdb return N items (e.g up to 1000 items per go)
		// That would require an API change in Ancients though
		if h, err := db.Ancient(freezerHashTable, i); err != nil {
			log.Crit("Failed to init database from freezer", "err", err)
		} else {
			hash = common.BytesToHash(h)
		}
		WriteHeaderNumber(batch, hash, i)
		// If enough data was accumulated in memory or we're at the last block, dump to disk
		if batch.ValueSize() > ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				log.Crit("Failed to write data to db", "err", err)
			}
			batch.Reset()
		}
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

type blockTxHashes struct {
	number uint64
	hashes []common.Hash
}

// iterateTransactions iterates over all transactions in the (canon) block
// number(s) given, and yields the hashes on a channel
func iterateTransactions(db ethdb.Database, from uint64, to uint64, reverse bool) (chan *blockTxHashes, chan struct{}) {
	// One thread sequentially reads data from db
	type numberRlp struct {
		number uint64
		rlp    rlp.RawValue
	}
	if to == from {
		return nil, nil
	}
	threads := to - from
	if cpus := runtime.NumCPU(); threads > uint64(cpus) {
		threads = uint64(cpus)
	}
	var (
		rlpCh    = make(chan *numberRlp, threads*2)     // we send raw rlp over this channel
		hashesCh = make(chan *blockTxHashes, threads*2) // send hashes over hashesCh
		abortCh  = make(chan struct{})
	)
	// lookup runs in one instance
	lookup := func() {
		n, end := from, to
		if reverse {
			n, end = to-1, from-1
		}
		defer close(rlpCh)
		for n != end {
			data := ReadCanonicalBodyRLP(db, n)
			// Feed the block to the aggregator, or abort on interrupt
			select {
			case rlpCh <- &numberRlp{n, data}:
			case <-abortCh:
				return
			}
			if reverse {
				n--
			} else {
				n++
			}
		}
	}
	// process runs in parallell
	nThreadsAlive := int32(threads)
	process := func() {
		defer func() {
			// Last processor closes the result channel
			if atomic.AddInt32(&nThreadsAlive, -1) == 0 {
				close(hashesCh)
			}
		}()

		var hasher = sha3.NewLegacyKeccak256()
		for data := range rlpCh {
			it, err := rlp.NewListIterator(data.rlp)
			if err != nil {
				log.Warn("tx iteration error", "error", err)
				return
			}
			it.Next()
			txs := it.Value()
			txIt, err := rlp.NewListIterator(txs)
			if err != nil {
				log.Warn("tx iteration error", "error", err)
				return
			}
			var hashes []common.Hash
			for txIt.Next() {
				if err := txIt.Err(); err != nil {
					log.Warn("tx iteration error", "error", err)
					return
				}
				var txHash common.Hash
				hasher.Reset()
				hasher.Write(txIt.Value())
				hasher.Sum(txHash[:0])
				hashes = append(hashes, txHash)
			}
			result := &blockTxHashes{
				hashes: hashes,
				number: data.number,
			}
			// Feed the block to the aggregator, or abort on interrupt
			select {
			case hashesCh <- result:
			case <-abortCh:
				return
			}
		}
	}
	go lookup() // start the sequential db accessor
	for i := 0; i < int(threads); i++ {
		go process()
	}
	return hashesCh, abortCh
}

// IndexTransactions creates txlookup indices of the specified block range.
//
// This function iterates canonical chain in reverse order, it has one main advantage:
// We can write tx index tail flag periodically even without the whole indexing
// procedure is finished. So that we can resume indexing procedure next time quickly.
func IndexTransactions(db ethdb.Database, from uint64, to uint64) {
	// short circuit for invalid range
	if from >= to {
		return
	}
	var (
		hashesCh, abortCh = iterateTransactions(db, from, to, true)
		batch             = db.NewBatch()
		start             = time.Now()
		logged            = start.Add(-7 * time.Second)
		//  Since we iterate in reverse, we expect the first number to come
		// in to be [to-1]. Therefore, setting lastNum to means that the
		// prqueue gap-evaluation will work correctly
		lastNum = to
		queue   = prque.New(nil)
		// for stats reporting
		blocks, txs = 0, 0
	)
	defer close(abortCh)

	for chanDelivery := range hashesCh {
		// Push the delivery into the queue and process contiguous ranges.
		// Since we iterate in reverse, so lower numbers have lower prio, and
		// we can use the number directly as prio marker
		queue.Push(chanDelivery, int64(chanDelivery.number))
		for !queue.Empty() {
			// If the next available item is gapped, return
			if _, priority := queue.Peek(); priority != int64(lastNum-1) {
				break
			}
			// Next block available, pop it off and index it
			delivery := queue.PopItem().(*blockTxHashes)
			lastNum = delivery.number
			WriteTxLookupEntriesByHash(batch, delivery.number, delivery.hashes)
			blocks++
			txs += len(delivery.hashes)
			// If enough data was accumulated in memory or we're at the last block, dump to disk
			if batch.ValueSize() > ethdb.IdealBatchSize {
				// Also write the tail there
				WriteTxIndexTail(batch, lastNum)
				if err := batch.Write(); err != nil {
					log.Crit("Failed writing batch to db", "error", err)
					return
				}
				batch.Reset()
			}
			// If we've spent too much time already, notify the user of what we're doing
			if time.Since(logged) > 8*time.Second {
				log.Info("Indexing transactions", "blocks", blocks, "txs", txs, "tail", lastNum, "total", to-from, "elapsed", common.PrettyDuration(time.Since(start)))
				logged = time.Now()
			}
		}
	}
	if lastNum < to {
		WriteTxIndexTail(batch, lastNum)
		// No need to write the batch if we never entered the loop above...
		if err := batch.Write(); err != nil {
			log.Crit("Failed writing batch to db", "error", err)
			return
		}
	}
	log.Info("Indexed transactions", "blocks", blocks, "txs", txs, "tail", lastNum, "elapsed", common.PrettyDuration(time.Since(start)))
}

// UnindexTransactions removes txlookup indices of the specified block range.
func UnindexTransactions(db ethdb.Database, from uint64, to uint64) {
	// short circuit for invalid range
	if from >= to {
		return
	}
	// Write flag first and then unindex the transaction indices. Some indices
	// will be left in the database if crash happens but it's fine.
	WriteTxIndexTail(db, to)
	// If only one block is unindexed, do it directly
	//if from+1 == to {
	//	data := ReadCanonicalBodyRLP(db, uint64(from))
	//	DeleteTxLookupEntries(db, ReadBlock(db, ReadCanonicalHash(db, from), from))
	//	log.Info("Unindexed transactions", "blocks", 1, "tail", to)
	//	return
	//}
	// TODO @holiman, add this back (if we want it)
	var (
		hashesCh, abortCh = iterateTransactions(db, from, to, false)
		batch             = db.NewBatch()
		start             = time.Now()
		logged            = start.Add(-7 * time.Second)
	)
	defer close(abortCh)
	// Otherwise spin up the concurrent iterator and unindexer
	blocks, txs := 0, 0
	for delivery := range hashesCh {
		DeleteTxLookupEntriesByHash(batch, delivery.hashes)
		txs += len(delivery.hashes)
		blocks++

		// If enough data was accumulated in memory or we're at the last block, dump to disk
		// A batch counts the size of deletion as '1', so we need to flush more
		// often than that.
		if blocks%1000 == 0 {
			if err := batch.Write(); err != nil {
				log.Crit("Failed writing batch to db", "error", err)
				return
			}
			batch.Reset()
		}
		// If we've spent too much time already, notify the user of what we're doing
		if time.Since(logged) > 8*time.Second {
			log.Info("Unindexing transactions", "blocks", blocks, "txs", txs, "total", to-from, "elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed writing batch to db", "error", err)
		return
	}
	log.Info("Unindexed transactions", "blocks", blocks, "txs", txs, "tail", to, "elapsed", common.PrettyDuration(time.Since(start)))
}
