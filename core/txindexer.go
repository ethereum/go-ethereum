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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package core

import (
	"errors"
	"fmt"

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
	limit    uint64
	db       ethdb.Database
	progress chan chan TxIndexProgress
	term     chan chan struct{}
	closed   chan struct{}
}

// newTxIndexer initializes the transaction indexer.
func newTxIndexer(limit uint64, chain *BlockChain) *txIndexer {
	indexer := &txIndexer{
		limit:    limit,
		db:       chain.db,
		progress: make(chan chan TxIndexProgress),
		term:     make(chan chan struct{}),
		closed:   make(chan struct{}),
	}
	go indexer.loop(chain)

	var msg string
	if limit == 0 {
		msg = "entire chain"
	} else {
		msg = fmt.Sprintf("last %d blocks", limit)
	}
	log.Info("Initialized transaction indexer", "range", msg)

	return indexer
}

// run executes the scheduled indexing/unindexing task in a separate thread.
// If the stop channel is closed, the task should be terminated as soon as
// possible, the done channel will be closed once the task is finished.
func (indexer *txIndexer) run(tail *uint64, head uint64, stop chan struct{}, done chan struct{}) {
	defer func() { close(done) }()

	// Short circuit if chain is empty and nothing to index.
	if head == 0 {
		return
	}
	// The tail flag is not existent, it means the node is just initialized
	// and all blocks in the chain (part of them may from ancient store) are
	// not indexed yet, index the chain according to the configured limit.
	if tail == nil {
		from := uint64(0)
		if indexer.limit != 0 && head >= indexer.limit {
			from = head - indexer.limit + 1
		}
		rawdb.IndexTransactions(indexer.db, from, head+1, stop, true)
		return
	}
	// The tail flag is existent (which means indexes in [tail, head] should be
	// present), while the whole chain are requested for indexing.
	if indexer.limit == 0 || head < indexer.limit {
		if *tail > 0 {
			// It can happen when chain is rewound to a historical point which
			// is even lower than the indexes tail, recap the indexing target
			// to new head to avoid reading non-existent block bodies.
			end := *tail
			if end > head+1 {
				end = head + 1
			}
			rawdb.IndexTransactions(indexer.db, 0, end, stop, true)
		}
		return
	}
	// The tail flag is existent, adjust the index range according to configured
	// limit and the latest chain head.
	if head-indexer.limit+1 < *tail {
		// Reindex a part of missing indices and rewind index tail to HEAD-limit
		rawdb.IndexTransactions(indexer.db, head-indexer.limit+1, *tail, stop, true)
	} else {
		// Unindex a part of stale indices and forward index tail to HEAD-limit
		rawdb.UnindexTransactions(indexer.db, *tail, head-indexer.limit+1, stop, false)
	}
}

// loop is the scheduler of the indexer, assigning indexing/unindexing tasks depending
// on the received chain event.
func (indexer *txIndexer) loop(chain *BlockChain) {
	defer close(indexer.closed)

	// Listening to chain events and manipulate the transaction indexes.
	var (
		stop     chan struct{}                       // Non-nil if background routine is active.
		done     chan struct{}                       // Non-nil if background routine is active.
		lastHead uint64                              // The latest announced chain head (whose tx indexes are assumed created)
		lastTail = rawdb.ReadTxIndexTail(indexer.db) // The oldest indexed block, nil means nothing indexed

		headCh = make(chan ChainHeadEvent)
		sub    = chain.SubscribeChainHeadEvent(headCh)
	)
	defer sub.Unsubscribe()

	// Launch the initial processing if chain is not empty (head != genesis).
	// This step is useful in these scenarios that chain has no progress.
	if head := rawdb.ReadHeadBlock(indexer.db); head != nil && head.Number().Uint64() != 0 {
		stop = make(chan struct{})
		done = make(chan struct{})
		lastHead = head.Number().Uint64()
		go indexer.run(rawdb.ReadTxIndexTail(indexer.db), head.NumberU64(), stop, done)
	}
	for {
		select {
		case head := <-headCh:
			if done == nil {
				stop = make(chan struct{})
				done = make(chan struct{})
				go indexer.run(rawdb.ReadTxIndexTail(indexer.db), head.Block.NumberU64(), stop, done)
			}
			lastHead = head.Block.NumberU64()
		case <-done:
			stop = nil
			done = nil
			lastTail = rawdb.ReadTxIndexTail(indexer.db)
		case ch := <-indexer.progress:
			ch <- indexer.report(lastHead, lastTail)
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
	total := indexer.limit
	if indexer.limit == 0 || total > head {
		total = head + 1 // genesis included
	}
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

// txIndexProgress retrieves the tx indexing progress, or an error if the
// background tx indexer is already stopped.
func (indexer *txIndexer) txIndexProgress() (TxIndexProgress, error) {
	ch := make(chan TxIndexProgress, 1)
	select {
	case indexer.progress <- ch:
		return <-ch, nil
	case <-indexer.closed:
		return TxIndexProgress{}, errors.New("indexer is closed")
	}
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
