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

const (
	rangeLogsTestSync = iota
	rangeLogsTestTrimmed
	rangeLogsTestIndexed
	rangeLogsTestUnindexed
	rangeLogsTestDone
)

type rangeLogsTestEvent struct {
	event      int
	begin, end uint64
}

// searchSession represents a single search session.
type searchSession struct {
	ctx                   context.Context
	filter                *Filter
	mb                    filtermaps.MatcherBackend
	syncRange             filtermaps.SyncRange // latest synchronized state with the matcher
	firstBlock, lastBlock uint64               // specified search range; each can be MaxUint64
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
	if err := s.syncMatcher(0); err != nil {
		return nil, err
	}
	return s, nil
}

// syncMatcher performs a synchronization step with the matcher. The resulting
// syncRange structure holds information about the latest range of indexed blocks
// and the guaranteed valid blocks whose log index have not been changed since
// the previous synchronization.
// The function also performs trimming of the match set in order to always keep
// it consistent with the synced matcher state.
// Tail trimming is only performed if the first block of the valid log index range
// is higher than trimTailThreshold. This is useful because unindexed log search
// is not affected by the valid tail (on the other hand, valid head is taken into
// account in order to provide reorg safety, even though the log index is not used).
// In case of indexed search the tail is only trimmed if the first part of the
// recently obtained results might be invalid. If guaranteed valid new results
// have been added at the head of previously validated results then there is no
// need to discard those even if the index tail have been unindexed since that.
func (s *searchSession) syncMatcher(trimTailThreshold uint64) error {
	if s.filter.rangeLogsTestHook != nil && !s.matchRange.IsEmpty() {
		s.filter.rangeLogsTestHook <- rangeLogsTestEvent{event: rangeLogsTestSync, begin: s.matchRange.First(), end: s.matchRange.Last()}
	}
	var err error
	s.syncRange, err = s.mb.SyncLogIndex(s.ctx)
	if err != nil {
		return err
	}
	// update actual search range based on current head number
	first := min(s.firstBlock, s.syncRange.HeadNumber)
	last := min(s.lastBlock, s.syncRange.HeadNumber)
	s.searchRange = common.NewRange(first, last+1-first)
	// discard everything that is not needed or might be invalid
	trimRange := s.syncRange.ValidBlocks
	if trimRange.First() <= trimTailThreshold {
		// everything before this point is already known to be valid; if this is
		// valid then keep everything before
		trimRange.SetFirst(0)
	}
	trimRange = trimRange.Intersection(s.searchRange)
	s.trimMatches(trimRange)
	if s.filter.rangeLogsTestHook != nil {
		if !s.matchRange.IsEmpty() {
			s.filter.rangeLogsTestHook <- rangeLogsTestEvent{event: rangeLogsTestTrimmed, begin: s.matchRange.First(), end: s.matchRange.Last()}
		} else {
			s.filter.rangeLogsTestHook <- rangeLogsTestEvent{event: rangeLogsTestTrimmed, begin: 0, end: 0}
		}
	}
	return nil
}

// trimMatches removes any entries from the current set of matches that is outside
// the given range.
func (s *searchSession) trimMatches(trimRange common.Range[uint64]) {
	s.matchRange = s.matchRange.Intersection(trimRange)
	if s.matchRange.IsEmpty() {
		s.matches = nil
		return
	}
	for len(s.matches) > 0 && s.matches[0].BlockNumber < s.matchRange.First() {
		s.matches = s.matches[1:]
	}
	for len(s.matches) > 0 && s.matches[len(s.matches)-1].BlockNumber > s.matchRange.Last() {
		s.matches = s.matches[:len(s.matches)-1]
	}
}

// searchInRange performs a single range search, either indexed or unindexed.
func (s *searchSession) searchInRange(r common.Range[uint64], indexed bool) ([]*types.Log, error) {
	first, last := r.First(), r.Last()
	if indexed {
		if s.filter.rangeLogsTestHook != nil {
			s.filter.rangeLogsTestHook <- rangeLogsTestEvent{rangeLogsTestIndexed, first, last}
		}
		results, err := s.filter.indexedLogs(s.ctx, s.mb, first, last)
		if err != filtermaps.ErrMatchAll {
			return results, err
		}
		// "match all" filters are not supported by filtermaps; fall back to
		// unindexed search which is the most efficient in this case
		s.forceUnindexed = true
		// fall through to unindexed case
	}
	if s.filter.rangeLogsTestHook != nil {
		s.filter.rangeLogsTestHook <- rangeLogsTestEvent{rangeLogsTestUnindexed, first, last}
	}
	return s.filter.unindexedLogs(s.ctx, first, last)
}

// doSearchIteration performs a search on a range missing from an incomplete set
// of results, adds the new section and removes invalidated entries.
func (s *searchSession) doSearchIteration() error {
	switch {
	case s.syncRange.IndexedBlocks.IsEmpty():
		// indexer is not ready; fallback to completely unindexed search, do not check valid range
		var err error
		s.matchRange = s.searchRange
		s.matches, err = s.searchInRange(s.searchRange, false)
		return err

	case s.matchRange.IsEmpty():
		// no results yet; try search in entire range
		indexedSearchRange := s.searchRange.Intersection(s.syncRange.IndexedBlocks)
		var err error
		if s.forceUnindexed = indexedSearchRange.IsEmpty(); !s.forceUnindexed {
			// indexed search on the intersection of indexed and searched range
			s.matchRange = indexedSearchRange
			s.matches, err = s.searchInRange(indexedSearchRange, true)
			if err != nil {
				return err
			}
			return s.syncMatcher(0) // trim everything that the matcher considers potentially invalid
		} else {
			// no intersection of indexed and searched range; unindexed search on the whole searched range
			s.matchRange = s.searchRange
			s.matches, err = s.searchInRange(s.searchRange, false)
			if err != nil {
				return err
			}
			return s.syncMatcher(math.MaxUint64) // unindexed search is not affected by the tail of valid range
		}

	case !s.matchRange.IsEmpty() && s.matchRange.First() > s.searchRange.First():
		// we have results but tail section is missing; do unindexed search for
		// the tail part but still allow indexed search for missing head section
		tailRange := common.NewRange(s.searchRange.First(), s.matchRange.First()-s.searchRange.First())
		tailMatches, err := s.searchInRange(tailRange, false)
		if err != nil {
			return err
		}
		s.matches = append(tailMatches, s.matches...)
		s.matchRange = tailRange.Union(s.matchRange)
		return s.syncMatcher(math.MaxUint64) // unindexed search is not affected by the tail of valid range

	case !s.matchRange.IsEmpty() && s.matchRange.First() == s.searchRange.First() && s.searchRange.AfterLast() > s.matchRange.AfterLast():
		// we have results but head section is missing
		headRange := common.NewRange(s.matchRange.AfterLast(), s.searchRange.AfterLast()-s.matchRange.AfterLast())
		if !s.forceUnindexed {
			indexedHeadRange := headRange.Intersection(s.syncRange.IndexedBlocks)
			if !indexedHeadRange.IsEmpty() && indexedHeadRange.First() == headRange.First() {
				// indexed head range search is possible
				headRange = indexedHeadRange
			} else {
				s.forceUnindexed = true
			}
		}
		headMatches, err := s.searchInRange(headRange, !s.forceUnindexed)
		if err != nil {
			return err
		}
		s.matches = append(s.matches, headMatches...)
		s.matchRange = s.matchRange.Union(headRange)
		if s.forceUnindexed {
			return s.syncMatcher(math.MaxUint64) // unindexed search is not affected by the tail of valid range
		} else {
			return s.syncMatcher(headRange.First()) // trim if the tail of latest head search results might be invalid
		}

	default:
		panic("invalid search session state")
	}
}

func (f *Filter) rangeLogs(ctx context.Context, firstBlock, lastBlock uint64) ([]*types.Log, error) {
	if f.rangeLogsTestHook != nil {
		defer func() {
			f.rangeLogsTestHook <- rangeLogsTestEvent{rangeLogsTestDone, 0, 0}
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
			return session.matches, err
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
func (f *Filter) unindexedLogs(ctx context.Context, begin, end uint64) ([]*types.Log, error) {
	start := time.Now()
	log.Debug("Performing unindexed log search", "begin", begin, "end", end)
	var matches []*types.Log
	for blockNumber := begin; blockNumber <= end; blockNumber++ {
		select {
		case <-ctx.Done():
			return matches, ctx.Err()
		default:
		}
		header, err := f.sys.backend.HeaderByNumber(ctx, rpc.BlockNumber(blockNumber))
		if header == nil || err != nil {
			return matches, err
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
