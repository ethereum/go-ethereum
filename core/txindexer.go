// Copyright 2024 The go-ethereum Authors
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

package core

import (
	"fmt"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// TxIndexProgress is the struct describing the progress for transaction indexing.
type TxIndexProgress struct {
	Indexed   uint64 // number of blocks whose transactions are indexed
	Remaining uint64 // number of blocks whose transactions are not indexed yet
}

// Done returns an indicator if the transaction indexing is finished.
func (progress TxIndexProgress) Done() bool {
	return progress.Remaining == 0
}

// txIndexer is the module responsible for maintaining transaction indexes
// according to the configured indexing range by users.
type txIndexer struct {
	// limit is the maximum number of blocks from head whose tx indexes
	// are reserved:
	//  * 0: means the entire chain should be indexed
	//  * N: means the latest N blocks [HEAD-N+1, HEAD] should be indexed
	//       and all others shouldn't.
	limit uint64

	// The current head of blockchain for transaction indexing. This field
	// is accessed by both the indexer and the indexing progress queries.
	head atomic.Uint64

	// The current tail of the indexed transactions, null indicates
	// that no transactions have been indexed yet.
	//
	// This field is accessed by both the indexer and the indexing
	// progress queries.
	tail atomic.Pointer[uint64]

	// cutoff denotes the block number before which the chain segment should
	// be pruned and not available locally.
	cutoff uint64
	db     ethdb.Database
	term   chan chan struct{}
	closed chan struct{}
}

// newTxIndexer initializes the transaction indexer.
func newTxIndexer(limit uint64, chain *BlockChain) *txIndexer {
	cutoff, _ := chain.HistoryPruningCutoff()
	indexer := &txIndexer{
		limit:  limit,
		cutoff: cutoff,
		db:     chain.db,
		term:   make(chan chan struct{}),
		closed: make(chan struct{}),
	}
	indexer.head.Store(indexer.resolveHead())
	indexer.tail.Store(rawdb.ReadTxIndexTail(chain.db))

	go indexer.loop(chain)

	var msg string
	if limit == 0 {
		if indexer.cutoff == 0 {
			msg = "entire chain"
		} else {
			msg = fmt.Sprintf("blocks since #%d", indexer.cutoff)
		}
	} else {
		msg = fmt.Sprintf("last %d blocks", limit)
	}
	log.Info("Initialized transaction indexer", "range", msg)

	return indexer
}

// run executes the scheduled indexing/unindexing task in a separate thread.
// If the stop channel is closed, the task should terminate as soon as possible.
// The done channel will be closed once the task is complete.
//
// Existing transaction indexes are assumed to be valid, with both the head
// and tail above the configured cutoff.
func (indexer *txIndexer) run(head uint64, stop chan struct{}, done chan struct{}) {
	defer func() { close(done) }()

	// Short circuit if the chain is either empty, or entirely below the
	// cutoff point.
	if head == 0 || head < indexer.cutoff {
		return
	}
	// The tail flag is not existent, it means the node is just initialized
	// and all blocks in the chain (part of them may from ancient store) are
	// not indexed yet, index the chain according to the configured limit.
	tail := rawdb.ReadTxIndexTail(indexer.db)
	if tail == nil {
		// Determine the first block for transaction indexing, taking the
		// configured cutoff point into account.
		from := uint64(0)
		if indexer.limit != 0 && head >= indexer.limit {
			from = head - indexer.limit + 1
		}
		from = max(from, indexer.cutoff)
		rawdb.IndexTransactions(indexer.db, from, head+1, stop, true)
		return
	}
	// The tail flag is existent (which means indexes in [tail, head] should be
	// present), while the whole chain are requested for indexing.
	if indexer.limit == 0 || head < indexer.limit {
		if *tail > 0 {
			from := max(uint64(0), indexer.cutoff)
			rawdb.IndexTransactions(indexer.db, from, *tail, stop, true)
		}
		return
	}
	// The tail flag is existent, adjust the index range according to configured
	// limit and the latest chain head.
	from := head - indexer.limit + 1
	from = max(from, indexer.cutoff)
	if from < *tail {
		// Reindex a part of missing indices and rewind index tail to HEAD-limit
		rawdb.IndexTransactions(indexer.db, from, *tail, stop, true)
	} else {
		// Unindex a part of stale indices and forward index tail to HEAD-limit
		rawdb.UnindexTransactions(indexer.db, *tail, from, stop, false)
	}
}

// repair ensures that transaction indexes are in a valid state and invalidates
// them if they are not. The following cases are considered invalid:
// * The index tail is higher than the chain head.
// * The chain head is below the configured cutoff, but the index tail is not empty.
// * The index tail is below the configured cutoff, but it is not empty.
func (indexer *txIndexer) repair(head uint64) {
	// If the transactions haven't been indexed yet, nothing to repair
	tail := rawdb.ReadTxIndexTail(indexer.db)
	if tail == nil {
		return
	}
	// The transaction index tail is higher than the chain head, which may occur
	// when the chain is rewound to a historical height below the index tail.
	// Purge the transaction indexes from the database. **It's not a common case
	// to rewind the chain head below the index tail**.
	if *tail > head {
		// A crash may occur between the two delete operations,
		// potentially leaving dangling indexes in the database.
		// However, this is considered acceptable.
		indexer.tail.Store(nil)
		rawdb.DeleteTxIndexTail(indexer.db)
		rawdb.DeleteAllTxLookupEntries(indexer.db, nil)
		log.Warn("Purge transaction indexes", "head", head, "tail", *tail)
		return
	}

	// If the entire chain is below the configured cutoff point,
	// removing the tail of transaction indexing and purges the
	// transaction indexes. **It's not a common case, as the cutoff
	// is usually defined below the chain head**.
	if head < indexer.cutoff {
		// A crash may occur between the two delete operations,
		// potentially leaving dangling indexes in the database.
		// However, this is considered acceptable.
		//
		// The leftover indexes can't be unindexed by scanning
		// the blocks as they are not guaranteed to be available.
		// Traversing the database directly within the transaction
		// index namespace might be slow and expensive, but we
		// have no choice.
		indexer.tail.Store(nil)
		rawdb.DeleteTxIndexTail(indexer.db)
		rawdb.DeleteAllTxLookupEntries(indexer.db, nil)
		log.Warn("Purge transaction indexes", "head", head, "cutoff", indexer.cutoff)
		return
	}

	// The chain head is above the cutoff while the tail is below the
	// cutoff. Shift the tail to the cutoff point and remove the indexes
	// below.
	if *tail < indexer.cutoff {
		// A crash may occur between the two delete operations,
		// potentially leaving dangling indexes in the database.
		// However, this is considered acceptable.
		indexer.tail.Store(&indexer.cutoff)
		rawdb.WriteTxIndexTail(indexer.db, indexer.cutoff)
		rawdb.DeleteAllTxLookupEntries(indexer.db, func(txhash common.Hash, blob []byte) bool {
			n := rawdb.DecodeTxLookupEntry(blob, indexer.db)
			return n != nil && *n < indexer.cutoff
		})
		log.Warn("Purge transaction indexes below cutoff", "tail", *tail, "cutoff", indexer.cutoff)
	}
}

// resolveHead resolves the block number of the current chain head.
func (indexer *txIndexer) resolveHead() uint64 {
	headBlockHash := rawdb.ReadHeadBlockHash(indexer.db)
	if headBlockHash == (common.Hash{}) {
		return 0
	}
	headBlockNumber := rawdb.ReadHeaderNumber(indexer.db, headBlockHash)
	if headBlockNumber == nil {
		return 0
	}
	return *headBlockNumber
}

// loop is the scheduler of the indexer, assigning indexing/unindexing tasks depending
// on the received chain event.
func (indexer *txIndexer) loop(chain *BlockChain) {
	defer close(indexer.closed)

	// Listening to chain events and manipulate the transaction indexes.
	var (
		stop   chan struct{} // Non-nil if background routine is active
		done   chan struct{} // Non-nil if background routine is active
		headCh = make(chan ChainHeadEvent)
		sub    = chain.SubscribeChainHeadEvent(headCh)
	)
	defer sub.Unsubscribe()

	// Validate the transaction indexes and repair if necessary
	head := indexer.head.Load()
	indexer.repair(head)

	// Launch the initial processing if chain is not empty (head != genesis).
	// This step is useful in these scenarios that chain has no progress.
	if head != 0 {
		stop = make(chan struct{})
		done = make(chan struct{})
		go indexer.run(head, stop, done)
	}
	for {
		select {
		case h := <-headCh:
			indexer.head.Store(h.Header.Number.Uint64())
			if done == nil {
				stop = make(chan struct{})
				done = make(chan struct{})
				go indexer.run(h.Header.Number.Uint64(), stop, done)
			}

		case <-done:
			stop = nil
			done = nil
			indexer.tail.Store(rawdb.ReadTxIndexTail(indexer.db))

		case ch := <-indexer.term:
			if stop != nil {
				close(stop)
			}
			if done != nil {
				log.Info("Waiting background transaction indexer to exit")
				<-done
			}
			close(ch)
			return
		}
	}
}

// report returns the tx indexing progress.
func (indexer *txIndexer) report(head uint64, tail *uint64) TxIndexProgress {
	// Special case if the head is even below the cutoff,
	// nothing to index.
	if head < indexer.cutoff {
		return TxIndexProgress{
			Indexed:   0,
			Remaining: 0,
		}
	}
	// Compute how many blocks are supposed to be indexed
	total := indexer.limit
	if indexer.limit == 0 || total > head {
		total = head + 1 // genesis included
	}
	length := head - indexer.cutoff + 1 // all available chain for indexing
	if total > length {
		total = length
	}
	// Compute how many blocks have been indexed
	var indexed uint64
	if tail != nil {
		indexed = head - *tail + 1
	}
	// The value of indexed might be larger than total if some blocks need
	// to be unindexed, avoiding a negative remaining.
	var remaining uint64
	if indexed < total {
		remaining = total - indexed
	}
	return TxIndexProgress{
		Indexed:   indexed,
		Remaining: remaining,
	}
}

// txIndexProgress retrieves the transaction indexing progress. The reported
// progress may slightly lag behind the actual indexing state, as the tail is
// only updated at the end of each indexing operation. However, this delay is
// considered acceptable.
func (indexer *txIndexer) txIndexProgress() TxIndexProgress {
	return indexer.report(indexer.head.Load(), indexer.tail.Load())
}

// close shutdown the indexer. Safe to be called for multiple times.
func (indexer *txIndexer) close() {
	ch := make(chan struct{})
	select {
	case indexer.term <- ch:
		<-ch
	case <-indexer.closed:
	}
}
