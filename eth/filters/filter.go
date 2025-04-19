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
	"time"

	"github.com/ethereum/go-ethereum/common"
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

	rangeLogsTestHook chan rangeLogsTestEvent
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

const (
	rangeLogsTestDone      = iota // zero range
	rangeLogsTestSync             // before sync; zero range
	rangeLogsTestSynced           // after sync; valid blocks range
	rangeLogsTestIndexed          // individual search range
	rangeLogsTestUnindexed        // individual search range
	rangeLogsTestResults          // results range after search iteration
	rangeLogsTestReorg            // results range trimmed by reorg
)

type rangeLogsTestEvent struct {
	event  int
	blocks common.Range[uint64]
}

// searchSession represents a single search session.
type searchSession struct {
	ctx       context.Context
	filter    *Filter
	mb        filtermaps.MatcherBackend
	syncRange filtermaps.SyncRange  // latest synchronized state with the matcher
	chainView *filtermaps.ChainView // can be more recent than the indexed view in syncRange
	// block ranges always refer to the current chainView
	firstBlock, lastBlock uint64               // specified search range; MaxUint64 means latest block
	searchRange           common.Range[uint64] // actual search range; end trimmed to latest head
	matchRange            common.Range[uint64] // range in which we have results (subset of searchRange)
	matches               []*types.Log         // valid set of matches in matchRange
	forceUnindexed        bool                 // revert to unindexed search
}

// newSearchSession returns a new searchSession.
func newSearchSession(ctx context.Context, filter *Filter, mb filtermaps.MatcherBackend, firstBlock, lastBlock uint64) (*searchSession, error) {
	s := &searchSession{
		ctx:        ctx,
		filter:     filter,
		mb:         mb,
		firstBlock: firstBlock,
		lastBlock:  lastBlock,
	}
	// enforce a consistent state before starting the search in order to be able
	// to determine valid range later
	var err error
	s.syncRange, err = s.mb.SyncLogIndex(s.ctx)
	if err != nil {
		return nil, err
	}
	if err := s.updateChainView(); err != nil {
		return nil, err
	}
	return s, nil
}

// updateChainView updates to the latest view of the underlying chain and sets
// searchRange by replacing MaxUint64 (meaning latest block) with actual head
// number in the specified search range.
// If the session already had an existing chain view and set of matches then
// it also trims part of the match set that a chain reorg might have invalidated.
func (s *searchSession) updateChainView() error {
	// update chain view based on current chain head (might be more recent than
	// the indexed view of syncRange as the indexer updates it asynchronously
	// with some delay
	newChainView := s.filter.sys.backend.CurrentView()
	if newChainView == nil {
		return errors.New("head block not available")
	}
	head := newChainView.HeadNumber()

	// update actual search range based on current head number
	firstBlock, lastBlock := s.firstBlock, s.lastBlock
	if firstBlock == math.MaxUint64 {
		firstBlock = head
	}
	if lastBlock == math.MaxUint64 {
		lastBlock = head
	}
	if firstBlock > lastBlock || lastBlock > head {
		return errInvalidBlockRange
	}
	s.searchRange = common.NewRange(firstBlock, lastBlock+1-firstBlock)

	// Trim existing match set in case a reorg may have invalidated some results
	if !s.matchRange.IsEmpty() {
		trimRange := newChainView.SharedRange(s.chainView).Intersection(s.searchRange)
		s.matchRange, s.matches = s.trimMatches(trimRange, s.matchRange, s.matches)
	}
	s.chainView = newChainView
	return nil
}

// trimMatches removes any entries from the specified set of matches that is
// outside the given range.
func (s *searchSession) trimMatches(trimRange, matchRange common.Range[uint64], matches []*types.Log) (common.Range[uint64], []*types.Log) {
	newRange := matchRange.Intersection(trimRange)
	if newRange == matchRange {
		return matchRange, matches
	}
	if newRange.IsEmpty() {
		return newRange, nil
	}
	for len(matches) > 0 && matches[0].BlockNumber < newRange.First() {
		matches = matches[1:]
	}
	for len(matches) > 0 && matches[len(matches)-1].BlockNumber > newRange.Last() {
		matches = matches[:len(matches)-1]
	}
	return newRange, matches
}

// searchInRange performs a single range search, either indexed or unindexed.
func (s *searchSession) searchInRange(r common.Range[uint64], indexed bool) (common.Range[uint64], []*types.Log, error) {
	if indexed {
		if s.filter.rangeLogsTestHook != nil {
			s.filter.rangeLogsTestHook <- rangeLogsTestEvent{rangeLogsTestIndexed, r}
		}
		results, err := s.filter.indexedLogs(s.ctx, s.mb, r.First(), r.Last())
		if err != nil && !errors.Is(err, filtermaps.ErrMatchAll) {
			return common.Range[uint64]{}, nil, err
		}
		if err == nil {
			// sync with filtermaps matcher
			if s.filter.rangeLogsTestHook != nil {
				s.filter.rangeLogsTestHook <- rangeLogsTestEvent{rangeLogsTestSync, common.Range[uint64]{}}
			}
			var syncErr error
			if s.syncRange, syncErr = s.mb.SyncLogIndex(s.ctx); syncErr != nil {
				return common.Range[uint64]{}, nil, syncErr
			}
			if s.filter.rangeLogsTestHook != nil {
				s.filter.rangeLogsTestHook <- rangeLogsTestEvent{rangeLogsTestSynced, s.syncRange.ValidBlocks}
			}
			// discard everything that might be invalid
			trimRange := s.syncRange.ValidBlocks.Intersection(s.chainView.SharedRange(s.syncRange.IndexedView))
			matchRange, matches := s.trimMatches(trimRange, r, results)
			return matchRange, matches, nil
		}
		// "match all" filters are not supported by filtermaps; fall back to
		// unindexed search which is the most efficient in this case
		s.forceUnindexed = true
		// fall through to unindexed case
	}
	if s.filter.rangeLogsTestHook != nil {
		s.filter.rangeLogsTestHook <- rangeLogsTestEvent{rangeLogsTestUnindexed, r}
	}
	matches, err := s.filter.unindexedLogs(s.ctx, s.chainView, r.First(), r.Last())
	if err != nil {
		return common.Range[uint64]{}, nil, err
	}
	return r, matches, nil
}

// doSearchIteration performs a search on a range missing from an incomplete set
// of results, adds the new section and removes invalidated entries.
func (s *searchSession) doSearchIteration() error {
	switch {
	case s.matchRange.IsEmpty():
		// no results yet; try search in entire range
		indexedSearchRange := s.searchRange.Intersection(s.syncRange.IndexedBlocks)
		if s.forceUnindexed = indexedSearchRange.IsEmpty(); !s.forceUnindexed {
			// indexed search on the intersection of indexed and searched range
			matchRange, matches, err := s.searchInRange(indexedSearchRange, true)
			if err != nil {
				return err
			}
			s.matchRange = matchRange
			s.matches = matches
			return nil
		} else {
			// no intersection of indexed and searched range; unindexed search on
			// the whole searched range
			matchRange, matches, err := s.searchInRange(s.searchRange, false)
			if err != nil {
				return err
			}
			s.matchRange = matchRange
			s.matches = matches
			return nil
		}

	case !s.matchRange.IsEmpty() && s.matchRange.First() > s.searchRange.First():
		// Results are available, but the tail section is missing. Perform an unindexed
		// search for the missing tail, while still allowing indexed search for the head.
		//
		// The unindexed search is necessary because the tail portion of the indexes
		// has been pruned.
		tailRange := common.NewRange(s.searchRange.First(), s.matchRange.First()-s.searchRange.First())
		_, tailMatches, err := s.searchInRange(tailRange, false)
		if err != nil {
			return err
		}
		s.matches = append(tailMatches, s.matches...)
		s.matchRange = tailRange.Union(s.matchRange)
		return nil

	case !s.matchRange.IsEmpty() && s.matchRange.First() == s.searchRange.First() && s.searchRange.AfterLast() > s.matchRange.AfterLast():
		// Results are available, but the head section is missing. Try to perform
		// the indexed search for the missing head, or fallback to unindexed search
		// if the tail portion of indexed range has been pruned.
		headRange := common.NewRange(s.matchRange.AfterLast(), s.searchRange.AfterLast()-s.matchRange.AfterLast())
		if !s.forceUnindexed {
			indexedHeadRange := headRange.Intersection(s.syncRange.IndexedBlocks)
			if !indexedHeadRange.IsEmpty() && indexedHeadRange.First() == headRange.First() {
				headRange = indexedHeadRange
			} else {
				// The tail portion of the indexes has been pruned, falling back
				// to unindexed search.
				s.forceUnindexed = true
			}
		}
		headMatchRange, headMatches, err := s.searchInRange(headRange, !s.forceUnindexed)
		if err != nil {
			return err
		}
		if headMatchRange.First() != s.matchRange.AfterLast() {
			// improbable corner case, first part of new head range invalidated by tail unindexing
			s.matches, s.matchRange = headMatches, headMatchRange
			return nil
		}
		s.matches = append(s.matches, headMatches...)
		s.matchRange = s.matchRange.Union(headMatchRange)
		return nil

	default:
		panic("invalid search session state")
	}
}

func (f *Filter) rangeLogs(ctx context.Context, firstBlock, lastBlock uint64) ([]*types.Log, error) {
	if f.rangeLogsTestHook != nil {
		defer func() {
			f.rangeLogsTestHook <- rangeLogsTestEvent{rangeLogsTestDone, common.Range[uint64]{}}
			close(f.rangeLogsTestHook)
		}()
	}

	if firstBlock > lastBlock {
		return nil, nil
	}
	mb := f.sys.backend.NewMatcherBackend()
	defer mb.Close()

	session, err := newSearchSession(ctx, f, mb, firstBlock, lastBlock)
	if err != nil {
		return nil, err
	}
	for session.searchRange != session.matchRange {
		if err := session.doSearchIteration(); err != nil {
			return nil, err
		}
		if f.rangeLogsTestHook != nil {
			f.rangeLogsTestHook <- rangeLogsTestEvent{rangeLogsTestResults, session.matchRange}
		}
		mr := session.matchRange
		if err := session.updateChainView(); err != nil {
			return nil, err
		}
		if f.rangeLogsTestHook != nil && session.matchRange != mr {
			f.rangeLogsTestHook <- rangeLogsTestEvent{rangeLogsTestReorg, session.matchRange}
		}
	}
	return session.matches, nil
}

func (f *Filter) indexedLogs(ctx context.Context, mb filtermaps.MatcherBackend, begin, end uint64) ([]*types.Log, error) {
	start := time.Now()
	potentialMatches, err := filtermaps.GetPotentialMatches(ctx, mb, begin, end, f.addresses, f.topics)
	matches := filterLogs(potentialMatches, nil, nil, f.addresses, f.topics)
	log.Trace("Performed indexed log search", "begin", begin, "end", end, "true matches", len(matches), "false positives", len(potentialMatches)-len(matches), "elapsed", common.PrettyDuration(time.Since(start)))
	return matches, err
}

// unindexedLogs returns the logs matching the filter criteria based on raw block
// iteration and bloom matching.
func (f *Filter) unindexedLogs(ctx context.Context, chainView *filtermaps.ChainView, begin, end uint64) ([]*types.Log, error) {
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
