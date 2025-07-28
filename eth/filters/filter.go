// Copyright 2014 The go-ethereum Authors
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

package filters

import (
	"context"
	"errors"
	"fmt"
	"math"
	"math/big"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/filtermaps"
	"github.com/ethereum/go-ethereum/core/history"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// Filter can be used to retrieve and filter logs.
type Filter struct {
	sys *FilterSystem

	addresses []common.Address
	topics    [][]common.Hash

	block      *common.Hash // Block hash if filtering a single block
	begin, end int64        // Range interval if filtering multiple blocks

	testFilterRanges []testFilterRange
}

type testFilterRange struct {
	begin, end uint64
	indexed    bool
}

// NewRangeFilter creates a new filter which uses a bloom filter on blocks to
// figure out whether a particular block is interesting or not.
func (sys *FilterSystem) NewRangeFilter(begin, end int64, addresses []common.Address, topics [][]common.Hash) *Filter {
	// Create a generic filter and convert it into a range filter
	filter := newFilter(sys, addresses, topics)
	filter.begin = begin
	filter.end = end

	return filter
}

// NewBlockFilter creates a new filter which directly inspects the contents of
// a block to figure out whether it is interesting or not.
func (sys *FilterSystem) NewBlockFilter(block common.Hash, addresses []common.Address, topics [][]common.Hash) *Filter {
	// Create a generic filter and convert it into a block filter
	filter := newFilter(sys, addresses, topics)
	filter.block = &block
	return filter
}

// newFilter creates a generic filter that can either filter based on a block hash,
// or based on range queries. The search criteria needs to be explicitly set.
func newFilter(sys *FilterSystem, addresses []common.Address, topics [][]common.Hash) *Filter {
	return &Filter{
		sys:       sys,
		addresses: addresses,
		topics:    topics,
	}
}

// Logs searches the blockchain for matching log entries, returning all from the
// first block that contains matches, updating the start of the filter accordingly.
func (f *Filter) Logs(ctx context.Context) ([]*types.Log, error) {
	// If we're doing singleton block filtering, execute and return
	if f.block != nil {
		header, err := f.sys.backend.HeaderByHash(ctx, *f.block)
		if err != nil {
			return nil, err
		}
		if header == nil {
			return nil, errUnknownBlock
		}
		if header.Number.Uint64() < f.sys.backend.HistoryPruningCutoff() {
			return nil, &history.PrunedHistoryError{}
		}
		return f.blockLogs(ctx, header)
	}

	// Disallow pending logs.
	if f.begin == rpc.PendingBlockNumber.Int64() || f.end == rpc.PendingBlockNumber.Int64() {
		return nil, errPendingLogsUnsupported
	}

	resolveSpecial := func(number int64) (uint64, error) {
		switch number {
		case rpc.LatestBlockNumber.Int64():
			// when searching from and/or until the current head, we resolve it
			// to MaxUint64 which is translated by rangeLogs to the actual head
			// in each iteration, ensuring that the head block will be searched
			// even if the chain is updated during search.
			return math.MaxUint64, nil
		case rpc.FinalizedBlockNumber.Int64():
			hdr, _ := f.sys.backend.HeaderByNumber(ctx, rpc.FinalizedBlockNumber)
			if hdr == nil {
				return 0, errors.New("finalized header not found")
			}
			return hdr.Number.Uint64(), nil
		case rpc.SafeBlockNumber.Int64():
			hdr, _ := f.sys.backend.HeaderByNumber(ctx, rpc.SafeBlockNumber)
			if hdr == nil {
				return 0, errors.New("safe header not found")
			}
			return hdr.Number.Uint64(), nil
		case rpc.EarliestBlockNumber.Int64():
			earliest := f.sys.backend.HistoryPruningCutoff()
			hdr, _ := f.sys.backend.HeaderByNumber(ctx, rpc.BlockNumber(earliest))
			if hdr == nil {
				return 0, errors.New("earliest header not found")
			}
			return hdr.Number.Uint64(), nil
		default:
			if number < 0 {
				return 0, errors.New("negative block number")
			}
			return uint64(number), nil
		}
	}

	// range query need to resolve the special begin/end block number
	begin, err := resolveSpecial(f.begin)
	if err != nil {
		return nil, err
	}
	end, err := resolveSpecial(f.end)
	if err != nil {
		return nil, err
	}
	return f.rangeLogs(ctx, begin, end)
}

func (f *Filter) rangeLogs(ctx context.Context, firstBlock, lastBlock uint64) ([]*types.Log, error) {
	if firstBlock > lastBlock {
		return nil, nil
	}
	chainView := f.sys.backend.CurrentChainView()

	if firstBlock > chainView.HeadNumber() {
		firstBlock = chainView.HeadNumber()
	}
	if lastBlock > chainView.HeadNumber() {
		lastBlock = chainView.HeadNumber()
	}

	indexView := f.sys.backend.GetIndexView(chainView.BlockHash(chainView.HeadNumber()))
	if indexView == nil {
		return f.unindexedLogs(ctx, chainView, firstBlock, lastBlock)
	}
	searchRange := common.NewRange[uint64](firstBlock, lastBlock+1-firstBlock)
	indexedRange := indexView.BlockRange().Intersection(searchRange)
	if indexedRange.IsEmpty() {
		indexView.Release()
		return f.unindexedLogs(ctx, chainView, firstBlock, lastBlock)
	}
	var (
		res1, res2, res3 []*types.Log
		err              error
	)
	res2, err = f.indexedLogs(ctx, chainView, indexView, indexedRange.First(), indexedRange.Last())
	indexView.Release()
	if err == filtermaps.ErrMatchAll {
		return f.unindexedLogs(ctx, chainView, firstBlock, lastBlock)
	}
	if err != nil {
		return nil, err
	}
	if searchRange.First() < indexedRange.First() {
		res1, err = f.unindexedLogs(ctx, chainView, searchRange.First(), indexedRange.First()-1)
		if err != nil {
			return nil, err
		}
	}
	if indexedRange.Last() < searchRange.Last() {
		res3, err = f.unindexedLogs(ctx, chainView, indexedRange.AfterLast(), searchRange.Last())
		if err != nil {
			return nil, err
		}
	}
	return append(append(res1, res2...), res3...), nil
}

func (f *Filter) indexedLogs(ctx context.Context, chainView *filtermaps.ChainView, indexView *filtermaps.IndexView, begin, end uint64) ([]*types.Log, error) {
	if f.testFilterRanges != nil {
		f.testFilterRanges = append(f.testFilterRanges, testFilterRange{begin: begin, end: end, indexed: true})
	}
	start := time.Now()
	potentialMatches, err := filtermaps.GetPotentialMatches(ctx, indexView, begin, end, f.addresses, f.topics)
	potentialLogs := make([]*types.Log, len(potentialMatches))
	indexCh := make(chan int)
	errCh := make(chan error, 1)
	var wg sync.WaitGroup

	worker := func() {
		defer wg.Done()
		for i := range indexCh {
			logPosition := potentialMatches[i]
			receipts := chainView.Receipts(logPosition.BlockNumber)
			if receipts == nil {
				select {
				case errCh <- fmt.Errorf("receipts for block #%d not found", logPosition.BlockNumber):
				default:
				}
				return
			}
			log, err := logPosition.GetLog(receipts)
			if err != nil {
				select {
				case errCh <- err:
				default:
				}
				return
			}
			potentialLogs[i] = log
		}
	}

	for range 4 { //TODO
		wg.Add(1)
		go worker()
	}
	for i := range potentialMatches {
		indexCh <- i
	}
	close(indexCh)
	wg.Wait()
	matches := filterLogs(potentialLogs, nil, nil, f.addresses, f.topics)
	log.Trace("Performed indexed log search", "begin", begin, "end", end, "true matches", len(matches), "false positives", len(potentialMatches)-len(matches), "elapsed", common.PrettyDuration(time.Since(start)))
	return matches, err
}

// unindexedLogs returns the logs matching the filter criteria based on raw block
// iteration and bloom matching.
func (f *Filter) unindexedLogs(ctx context.Context, chainView *filtermaps.ChainView, begin, end uint64) ([]*types.Log, error) {
	if f.testFilterRanges != nil {
		f.testFilterRanges = append(f.testFilterRanges, testFilterRange{begin: begin, end: end, indexed: false})
	}
	start := time.Now()
	log.Debug("Performing unindexed log search", "begin", begin, "end", end)
	var matches []*types.Log
	for blockNumber := begin; blockNumber <= end; blockNumber++ {
		select {
		case <-ctx.Done():
			return matches, ctx.Err()
		default:
		}
		if blockNumber > chainView.HeadNumber() {
			// check here so that we can return matches up until head along with
			// the error
			return matches, errInvalidBlockRange
		}
		header := chainView.Header(blockNumber)
		if header == nil {
			return matches, errors.New("header not found")
		}
		found, err := f.blockLogs(ctx, header)
		if err != nil {
			return matches, err
		}
		matches = append(matches, found...)
	}
	log.Debug("Performed unindexed log search", "begin", begin, "end", end, "matches", len(matches), "elapsed", common.PrettyDuration(time.Since(start)))
	return matches, nil
}

// blockLogs returns the logs matching the filter criteria within a single block.
func (f *Filter) blockLogs(ctx context.Context, header *types.Header) ([]*types.Log, error) {
	if bloomFilter(header.Bloom, f.addresses, f.topics) {
		return f.checkMatches(ctx, header)
	}
	return nil, nil
}

// checkMatches checks if the receipts belonging to the given header contain any log events that
// match the filter criteria. This function is called when the bloom filter signals a potential match.
func (f *Filter) checkMatches(ctx context.Context, header *types.Header) ([]*types.Log, error) {
	hash := header.Hash()
	// Logs in cache are partially filled with context data
	// such as tx index, block hash, etc.
	// Notably tx hash is NOT filled in because it needs
	// access to block body data.
	cached, err := f.sys.cachedLogElem(ctx, hash, header.Number.Uint64(), header.Time)
	if err != nil {
		return nil, err
	}
	logs := filterLogs(cached.logs, nil, nil, f.addresses, f.topics)
	if len(logs) == 0 {
		return nil, nil
	}
	// Most backends will deliver un-derived logs, but check nevertheless.
	if len(logs) > 0 && logs[0].TxHash != (common.Hash{}) {
		return logs, nil
	}

	body, err := f.sys.cachedGetBody(ctx, cached, hash, header.Number.Uint64())
	if err != nil {
		return nil, err
	}
	for i, log := range logs {
		// Copy log not to modify cache elements
		logcopy := *log
		logcopy.TxHash = body.Transactions[logcopy.TxIndex].Hash()
		logs[i] = &logcopy
	}
	return logs, nil
}

// filterLogs creates a slice of logs matching the given criteria.
func filterLogs(logs []*types.Log, fromBlock, toBlock *big.Int, addresses []common.Address, topics [][]common.Hash) []*types.Log {
	var check = func(log *types.Log) bool {
		if fromBlock != nil && fromBlock.Int64() >= 0 && fromBlock.Uint64() > log.BlockNumber {
			return false
		}
		if toBlock != nil && toBlock.Int64() >= 0 && toBlock.Uint64() < log.BlockNumber {
			return false
		}
		if len(addresses) > 0 && !slices.Contains(addresses, log.Address) {
			return false
		}
		// If the to filtered topics is greater than the amount of topics in logs, skip.
		if len(topics) > len(log.Topics) {
			return false
		}
		for i, sub := range topics {
			if len(sub) == 0 {
				continue // empty rule set == wildcard
			}
			if !slices.Contains(sub, log.Topics[i]) {
				return false
			}
		}
		return true
	}
	var ret []*types.Log
	for _, log := range logs {
		if log != nil && check(log) {
			ret = append(ret, log)
		}
	}
	return ret
}

func bloomFilter(bloom types.Bloom, addresses []common.Address, topics [][]common.Hash) bool {
	if len(addresses) > 0 {
		var included bool
		for _, addr := range addresses {
			if types.BloomLookup(bloom, addr) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}

	for _, sub := range topics {
		included := len(sub) == 0 // empty rule set == wildcard
		for _, topic := range sub {
			if types.BloomLookup(bloom, topic) {
				included = true
				break
			}
		}
		if !included {
			return false
		}
	}
	return true
}

// ReceiptWithTx contains a receipt and its corresponding transaction
type ReceiptWithTx struct {
	Receipt     *types.Receipt
	Transaction *types.Transaction
}

// filterReceipts returns the receipts matching the given criteria
// In addition to returning receipts, it also returns the corresponding transactions.
// This is because receipts only contain low-level data, while user-facing data
// may require additional information from the Transaction.
func filterReceipts(txHashes map[common.Hash]bool, ev core.ChainEvent) []*ReceiptWithTx {
	var ret []*ReceiptWithTx

	receipts := ev.Receipts
	txs := ev.Transactions

	if len(receipts) != len(txs) {
		log.Warn("Receipts and transactions length mismatch", "receipts", len(receipts), "transactions", len(txs))
		return ret
	}

	if len(txHashes) == 0 {
		// No filter, send all receipts with their transactions.
		ret = make([]*ReceiptWithTx, len(receipts))
		for i, receipt := range receipts {
			ret[i] = &ReceiptWithTx{
				Receipt:     receipt,
				Transaction: txs[i],
			}
		}
	} else {
		for i, receipt := range receipts {
			if txHashes[receipt.TxHash] {
				ret = append(ret, &ReceiptWithTx{
					Receipt:     receipt,
					Transaction: txs[i],
				})

				// Early exit if all receipts are found
				if len(ret) == len(txHashes) {
					break
				}
			}
		}
	}

	return ret
}
