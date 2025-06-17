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
	"math/big"
	"runtime"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/consensus/misc/eip4844"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
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
	config *params.ChainConfig
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
		config: chain.chainConfig,
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
		indexer.index(from, head+1, stop, nil, true)
		return
	}
	// The tail flag is existent (which means indexes in [tail, head] should be
	// present), while the whole chain are requested for indexing.
	if indexer.limit == 0 || head < indexer.limit {
		if *tail > 0 {
			from := max(uint64(0), indexer.cutoff)
			indexer.index(from, *tail, stop, nil, true)
		}
		return
	}
	// The tail flag is existent, adjust the index range according to configured
	// limit and the latest chain head.
	from := head - indexer.limit + 1
	from = max(from, indexer.cutoff)
	if from < *tail {
		// Reindex a part of missing indices and rewind index tail to HEAD-limit
		indexer.index(from, *tail, stop, nil, true)
	} else {
		// Unindex a part of stale indices and forward index tail to HEAD-limit
		indexer.unindex(*tail, from, stop, nil, false)
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

// generateTxIndex generates the data that will be stored alongside the transaction index
func (indexer *txIndexer) generateTxIndex(txs types.Transactions, header *types.Header, receipts types.Receipts) []rawdb.TxIndex {
	var (
		logIndex              = 0
		prevCumulativeGasUsed = uint64(0)
		signer                = types.MakeSigner(indexer.config, header.Number, header.Time)
		indexes               = make([]rawdb.TxIndex, len(txs))
	)

	var blobGasPrice *big.Int
	if header.ExcessBlobGas != nil {
		blobGasPrice = eip4844.CalcBlobFee(indexer.config, header)
	}

	for blockIndex, tx := range txs {
		from, _ := signer.Sender(tx)
		indexes[blockIndex] = rawdb.TxIndex{
			Type:  tx.Type(),
			Nonce: tx.Nonce(),
			To:    tx.To(),

			BlockNumber:       header.Number.Uint64(),
			BlockHash:         header.Hash(),
			BlockTime:         header.Time,
			BaseFee:           header.BaseFee,
			TxIndex:           uint32(blockIndex),
			Sender:            from,
			EffectiveGasPrice: tx.EffectiveGasPrice(header.BaseFee),
			GasUsed:           receipts[blockIndex].CumulativeGasUsed - prevCumulativeGasUsed,
			LogIndex:          uint32(logIndex),
			BlobGas:           tx.BlobGas(),
			BlobGasPrice:      blobGasPrice,
		}
		prevCumulativeGasUsed = receipts[blockIndex].CumulativeGasUsed
		logIndex += len(receipts[blockIndex].Logs)
	}
	return indexes
}

type blockIndexingContext struct {
	number   uint64
	txHashes []common.Hash
	indexes  []rawdb.TxIndex
}

// iterate iterates over all transactions in the (canon) block
// number(s) given, and yields the hashes on a channel. If there is a signal
// received from interrupt channel, the iteration will be aborted and result
// channel will be closed.
func (indexer *txIndexer) iterate(from uint64, to uint64, reverse, indexing bool, interrupt chan struct{}) chan *blockIndexingContext {
	// One thread sequentially reads data from db
	type numberRlp struct {
		number      uint64
		headerRLP   rlp.RawValue
		bodyRLP     rlp.RawValue
		receiptsRLP rlp.RawValue
	}
	if to == from {
		return nil
	}
	threads := to - from
	if cpus := runtime.NumCPU(); threads > uint64(cpus) {
		threads = uint64(cpus)
	}
	var (
		rlpCh      = make(chan *numberRlp, threads*2)            // we send raw rlp over this channel
		contextsCh = make(chan *blockIndexingContext, threads*2) // send indexing context over contextsCh
	)
	// lookup runs in one instance
	lookup := func() {
		n, end := from, to
		if reverse {
			n, end = to-1, from-1
		}
		defer close(rlpCh)
		for n != end {
			blockHash := rawdb.ReadCanonicalHash(indexer.db, n)
			rlps := &numberRlp{
				number:  n,
				bodyRLP: rawdb.ReadBodyRLP(indexer.db, blockHash, n),
			}

			if indexing {
				// We are indexing, read more to build the index
				rlps.headerRLP = rawdb.ReadHeaderRLP(indexer.db, blockHash, n)
				rlps.receiptsRLP = rawdb.ReadReceiptsRLP(indexer.db, blockHash, n)
			}
			// Feed the block to the aggregator, or abort on interrupt
			select {
			case rlpCh <- rlps:
			case <-interrupt:
				return
			}
			if reverse {
				n--
			} else {
				n++
			}
		}
	}
	// process runs in parallel
	var nThreadsAlive atomic.Int32
	nThreadsAlive.Store(int32(threads))
	process := func() {
		defer func() {
			// Last processor closes the result channel
			if nThreadsAlive.Add(-1) == 0 {
				close(contextsCh)
			}
		}()
		for data := range rlpCh {
			var body types.Body
			if err := rlp.DecodeBytes(data.bodyRLP, &body); err != nil {
				log.Error("Failed to decode block body", "block", data.number, "error", err)
				return
			}
			txHashes := make([]common.Hash, 0, len(body.Transactions))
			for _, tx := range body.Transactions {
				txHashes = append(txHashes, tx.Hash())
			}

			result := &blockIndexingContext{
				number:   data.number,
				txHashes: txHashes,
			}

			if indexing {
				var header types.Header
				if err := rlp.DecodeBytes(data.headerRLP, &header); err != nil {
					log.Error("failed to decode header", "block", data.number, "err", err)
					return
				}

				var storageReceipts []*types.ReceiptForStorage
				if err := rlp.DecodeBytes(data.receiptsRLP, &storageReceipts); err != nil {
					log.Error("failed to decode receipts", "block", data.number, "err", err)
					return
				}

				receipts := make([]*types.Receipt, len(storageReceipts))
				for i, receipt := range storageReceipts {
					receipts[i] = (*types.Receipt)(receipt)
				}
				result.indexes = indexer.generateTxIndex(body.Transactions, &header, receipts)
			}
			// Feed the block to the aggregator, or abort on interrupt
			select {
			case contextsCh <- result:
			case <-interrupt:
				return
			}
		}
	}
	go lookup() // start the sequential db accessor
	for i := 0; i < int(threads); i++ {
		go process()
	}
	return contextsCh
}

// index creates txlookup indices of the specified block range.
//
// This function iterates canonical chain in reverse order, it has one main advantage:
// We can write tx index tail flag periodically even without the whole indexing
// procedure is finished. So that we can resume indexing procedure next time quickly.
//
// There is a passed channel, the whole procedure will be interrupted if any
// signal received.
func (indexer *txIndexer) index(from uint64, to uint64, interrupt chan struct{}, hook func(uint64) bool, report bool) {
	// short circuit for invalid range
	if from >= to {
		return
	}
	var (
		contextsCh = indexer.iterate(from, to, true, true, interrupt)
		batch      = indexer.db.NewBatch()
		start      = time.Now()
		logged     = start.Add(-7 * time.Second)

		// Since we iterate in reverse, we expect the first number to come
		// in to be [to-1]. Therefore, setting lastNum to means that the
		// queue gap-evaluation will work correctly
		lastNum     = to
		queue       = prque.New[int64, *blockIndexingContext](nil)
		blocks, txs = 0, 0 // for stats reporting
	)
	for chanDelivery := range contextsCh {
		// Push the delivery into the queue and process contiguous ranges.
		// Since we iterate in reverse, so lower numbers have lower prio, and
		// we can use the number directly as prio marker
		queue.Push(chanDelivery, int64(chanDelivery.number))
		for !queue.Empty() {
			// If the next available item is gapped, return
			if _, priority := queue.Peek(); priority != int64(lastNum-1) {
				break
			}
			// For testing
			if hook != nil && !hook(lastNum-1) {
				break
			}
			// Next block available, pop it off and index it
			delivery := queue.PopItem()
			lastNum = delivery.number
			rawdb.WriteTxLookupEntries(batch, delivery.txHashes, delivery.indexes)
			blocks++
			txs += len(delivery.txHashes)
			// If enough data was accumulated in memory or we're at the last block, dump to disk
			if batch.ValueSize() > ethdb.IdealBatchSize {
				rawdb.WriteTxIndexTail(batch, lastNum) // Also write the tail here
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
	// Flush the new indexing tail and the last committed data. It can also happen
	// that the last batch is empty because nothing to index, but the tail has to
	// be flushed anyway.
	rawdb.WriteTxIndexTail(batch, lastNum)
	if err := batch.Write(); err != nil {
		log.Crit("Failed writing batch to db", "error", err)
		return
	}
	logger := log.Debug
	if report {
		logger = log.Info
	}
	select {
	case <-interrupt:
		logger("Transaction indexing interrupted", "blocks", blocks, "txs", txs, "tail", lastNum, "elapsed", common.PrettyDuration(time.Since(start)))
	default:
		logger("Indexed transactions", "blocks", blocks, "txs", txs, "tail", lastNum, "elapsed", common.PrettyDuration(time.Since(start)))
	}
}

// unindex removes txlookup indices of the specified block range.
//
// There is a passed channel, the whole procedure will be interrupted if any
// signal received.
func (indexer *txIndexer) unindex(from uint64, to uint64, interrupt chan struct{}, hook func(uint64) bool, report bool) {
	// short circuit for invalid range
	if from >= to {
		return
	}
	var (
		contextsCh = indexer.iterate(from, to, false, false, interrupt)
		batch      = indexer.db.NewBatch()
		start      = time.Now()
		logged     = start.Add(-7 * time.Second)

		// we expect the first number to come in to be [from]. Therefore, setting
		// nextNum to from means that the queue gap-evaluation will work correctly
		nextNum     = from
		queue       = prque.New[int64, *blockIndexingContext](nil)
		blocks, txs = 0, 0 // for stats reporting
	)
	// Otherwise spin up the concurrent iterator and unindexer
	for delivery := range contextsCh {
		// Push the delivery into the queue and process contiguous ranges.
		queue.Push(delivery, -int64(delivery.number))
		for !queue.Empty() {
			// If the next available item is gapped, return
			if _, priority := queue.Peek(); -priority != int64(nextNum) {
				break
			}
			// For testing
			if hook != nil && !hook(nextNum) {
				break
			}
			delivery := queue.PopItem()
			nextNum = delivery.number + 1
			rawdb.DeleteTxLookupEntries(batch, delivery.txHashes)
			txs += len(delivery.txHashes)
			blocks++

			// If enough data was accumulated in memory or we're at the last block, dump to disk
			// A batch counts the size of deletion as '1', so we need to flush more
			// often than that.
			if blocks%1000 == 0 {
				rawdb.WriteTxIndexTail(batch, nextNum)
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
	}
	// Flush the new indexing tail and the last committed data. It can also happen
	// that the last batch is empty because nothing to unindex, but the tail has to
	// be flushed anyway.
	rawdb.WriteTxIndexTail(batch, nextNum)
	if err := batch.Write(); err != nil {
		log.Crit("Failed writing batch to db", "error", err)
		return
	}
	logger := log.Debug
	if report {
		logger = log.Info
	}
	select {
	case <-interrupt:
		logger("Transaction unindexing interrupted", "blocks", blocks, "txs", txs, "tail", to, "elapsed", common.PrettyDuration(time.Since(start)))
	default:
		logger("Unindexed transactions", "blocks", blocks, "txs", txs, "tail", to, "elapsed", common.PrettyDuration(time.Since(start)))
	}
}

// indexHead indexes given head block synchronously
func (indexer *txIndexer) indexHead(db ethdb.KeyValueWriter, block *types.Block, receipts types.Receipts) {
	indexes := indexer.generateTxIndex(block.Transactions(), block.Header(), receipts)
	hashes := make([]common.Hash, len(block.Transactions()))
	for i, tx := range block.Transactions() {
		hashes[i] = tx.Hash()
	}
	rawdb.WriteTxLookupEntries(db, hashes, indexes)
}
