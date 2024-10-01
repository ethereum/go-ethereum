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
	"math"
	"math/big"
	"slices"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/filtermaps"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

// Filter can be used to retrieve and filter logs.
type Filter struct {
	sys *FilterSystem

	addresses []common.Address
	topics    [][]common.Hash

	block        *common.Hash // Block hash if filtering a single block
	begin, end   int64        // Range interval if filtering multiple blocks
	bbMatchCount uint64
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
			return nil, errors.New("unknown block")
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
		}
		if number < 0 {
			return 0, errors.New("negative block number")
		}
		return uint64(number), nil
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
		return nil, errors.New("invalid search range")
	}
	mb := f.sys.backend.NewMatcherBackend()
	defer mb.Close()

	// enforce a consistent state before starting the search in order to be able
	// to determine valid range later
	syncRange, err := mb.SyncLogIndex(ctx)
	if err != nil {
		return nil, err
	}
	headBlock := syncRange.Head.Number.Uint64() // Head is guaranteed != nil
	// if haveMatches == true then matches correspond to the block number range
	// between matchFirst and matchLast
	var (
		matches                     []*types.Log
		haveMatches, forceUnindexed bool
		matchFirst, matchLast       uint64
	)
	trimMatches := func(trimFirst, trimLast uint64) {
		if !haveMatches {
			return
		}
		if trimLast < matchFirst || trimFirst > matchLast {
			matches, haveMatches = nil, false
			return
		}
		if trimFirst > matchFirst {
			for len(matches) > 0 && matches[0].BlockNumber < trimFirst {
				matches = matches[1:]
			}
			matchFirst = trimFirst
		}
		if trimLast < matchLast {
			for len(matches) > 0 && matches[len(matches)-1].BlockNumber > trimLast {
				matches = matches[:len(matches)-1]
			}
			matchLast = trimLast
		}
	}

	for {
		// determine range to be searched; for simplicity we only extend the most
		// recent end of the existing match set by matching between searchFirst
		// and searchLast.
		searchFirst, searchLast := firstBlock, lastBlock
		if searchFirst > headBlock {
			searchFirst = headBlock
		}
		if searchLast > headBlock {
			searchLast = headBlock
		}
		trimMatches(searchFirst, searchLast)
		if haveMatches && matchFirst == searchFirst && matchLast == searchLast {
			return matches, nil
		}
		var trimTailIfNotValid uint64
		if haveMatches && matchFirst > searchFirst {
			// missing tail section; do unindexed search
			tailMatches, err := f.unindexedLogs(ctx, searchFirst, matchFirst-1)
			if err != nil {
				return matches, err
			}
			matches = append(tailMatches, matches...)
			matchFirst = searchFirst
			// unindexed results are not affected by valid tail; do not trim tail
			trimTailIfNotValid = math.MaxUint64
		}
		// now if we have matches, they start at searchFirst
		if haveMatches {
			searchFirst = matchLast + 1
			if !syncRange.Indexed || syncRange.FirstIndexed > searchFirst {
				forceUnindexed = true
			}
		}
		var newMatches []*types.Log
		if !syncRange.Indexed || syncRange.FirstIndexed > searchLast || syncRange.LastIndexed < searchFirst {
			forceUnindexed = true
		}
		if !forceUnindexed {
			if syncRange.FirstIndexed > searchFirst {
				searchFirst = syncRange.FirstIndexed
			}
			if syncRange.LastIndexed > searchLast {
				searchLast = syncRange.LastIndexed
			}
			newMatches, err = f.indexedLogs(ctx, mb, searchFirst, searchLast)
			// trim tail if it affects the indexed search range
			trimTailIfNotValid = searchFirst
			if err == filtermaps.ErrMatchAll {
				// "match all" filters are not supported by filtermaps; fall back
				// to unindexed search which is the most efficient in this case
				forceUnindexed = true
			}
		}
		if forceUnindexed {
			newMatches, err = f.unindexedLogs(ctx, searchFirst, searchLast)
			// unindexed results are not affected by valid tail; do not trim tail
			trimTailIfNotValid = math.MaxUint64
		}
		if err != nil {
			return matches, err
		}
		if matches == nil {
			matches = newMatches
			haveMatches, matchFirst, matchLast = true, searchFirst, searchLast
		} else {
			matches = append(matches, newMatches...)
			matchLast = searchLast
		}

		syncRange, err = mb.SyncLogIndex(ctx)
		if err != nil {
			return matches, err
		}
		if !syncRange.Valid {
			matches, haveMatches = nil, false
		} else {
			if syncRange.FirstValid > trimTailIfNotValid {
				trimMatches(syncRange.FirstValid, syncRange.LastValid)
			} else {
				trimMatches(0, syncRange.LastValid)
			}
		}
	}
}

func (f *Filter) indexedLogs(ctx context.Context, mb filtermaps.MatcherBackend, begin, end uint64) ([]*types.Log, error) {
	logs, err := filtermaps.GetPotentialMatches(ctx, mb, begin, end, f.addresses, f.topics)
	logs = filterLogs(logs, nil, nil, f.addresses, f.topics)
	return logs, err
}

// unindexedLogs returns the logs matching the filter criteria based on raw block
// iteration and bloom matching.
func (f *Filter) unindexedLogs(ctx context.Context, begin, end uint64) ([]*types.Log, error) {
	log.Warn("Performing unindexed log search", "begin", begin, "end", end)
	var logs []*types.Log
	for blockNumber := begin; blockNumber <= end; blockNumber++ {
		select {
		case <-ctx.Done():
			return logs, ctx.Err()
		default:
		}
		header, err := f.sys.backend.HeaderByNumber(ctx, rpc.BlockNumber(blockNumber))
		if header == nil || err != nil {
			return logs, err
		}
		found, err := f.blockLogs(ctx, header)
		if err != nil {
			return logs, err
		}
		logs = append(logs, found...)
	}
	return logs, nil
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
// skipFilter signals all logs of the given block are requested.
func (f *Filter) checkMatches(ctx context.Context, header *types.Header) ([]*types.Log, error) {
	hash := header.Hash()
	// Logs in cache are partially filled with context data
	// such as tx index, block hash, etc.
	// Notably tx hash is NOT filled in because it needs
	// access to block body data.
	cached, err := f.sys.cachedLogElem(ctx, hash, header.Number.Uint64())
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
		if check(log) {
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
