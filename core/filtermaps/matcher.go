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

package filtermaps

import (
	"context"
	"errors"
	"math"
	"sync"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

// ErrMatchAll is returned when the specified filter matches everything.
// Handling this case in filtermaps would require an extra special case and
// would actually be slower than reverting to legacy filter.
var ErrMatchAll = errors.New("match all patterns not supported")

// MatcherBackend defines the functions required for searching in the log index
// data structure. It is currently implemented by FilterMapsMatcherBackend but
// once EIP-7745 is implemented and active, these functions can also be trustlessly
// served by a remote prover.
type MatcherBackend interface {
	GetParams() *Params
	GetBlockLvPointer(ctx context.Context, blockNumber uint64) (uint64, error)
	GetFilterMapRow(ctx context.Context, mapIndex, rowIndex uint32) (FilterRow, error)
	GetLogByLvIndex(ctx context.Context, lvIndex uint64) (*types.Log, error)
	SyncLogIndex(ctx context.Context) (SyncRange, error)
	Close()
}

// SyncRange is returned by MatcherBackend.SyncLogIndex. It contains the latest
// chain head, the indexed range that is currently consistent with the chain
// and the valid range that has not been changed and has been consistent with
// all states of the chain since the previous SyncLogIndex or the creation of
// the matcher backend.
type SyncRange struct {
	Head *types.Header
	// block range where the index has not changed since the last matcher sync
	// and therefore the set of matches found in this region is guaranteed to
	// be valid and complete.
	Valid                 bool
	FirstValid, LastValid uint64
	// block range indexed according to the given chain head.
	Indexed                   bool
	FirstIndexed, LastIndexed uint64
}

// GetPotentialMatches returns a list of logs that are potential matches for the
// given filter criteria. If parts of the log index in the searched range are
// missing or changed during the search process then the resulting logs belonging
// to that block range might be missing or incorrect.
// Also note that the returned list may contain false positives.
func GetPotentialMatches(ctx context.Context, backend MatcherBackend, firstBlock, lastBlock uint64, addresses []common.Address, topics [][]common.Hash) ([]*types.Log, error) {
	params := backend.GetParams()
	// find the log value index range to search
	firstIndex, err := backend.GetBlockLvPointer(ctx, firstBlock)
	if err != nil {
		return nil, err
	}
	lastIndex, err := backend.GetBlockLvPointer(ctx, lastBlock+1)
	if err != nil {
		return nil, err
	}
	if lastIndex > 0 {
		lastIndex--
	}
	firstMap, lastMap := uint32(firstIndex>>params.logValuesPerMap), uint32(lastIndex>>params.logValuesPerMap)
	firstEpoch, lastEpoch := firstMap>>params.logMapsPerEpoch, lastMap>>params.logMapsPerEpoch

	// build matcher according to the given filter criteria
	matchers := make([]matcher, len(topics)+1)
	// matchAddress signals a match when there is a match for any of the given
	// addresses.
	// If the list of addresses is empty then it creates a "wild card" matcher
	// that signals every index as a potential match.
	matchAddress := make(matchAny, len(addresses))
	for i, address := range addresses {
		matchAddress[i] = &singleMatcher{backend: backend, value: addressValue(address)}
	}
	matchers[0] = matchAddress
	for i, topicList := range topics {
		// matchTopic signals a match when there is a match for any of the topics
		// specified for the given position (topicList).
		// If topicList is empty then it creates a "wild card" matcher that signals
		// every index as a potential match.
		matchTopic := make(matchAny, len(topicList))
		for j, topic := range topicList {
			matchTopic[j] = &singleMatcher{backend: backend, value: topicValue(topic)}
		}
		matchers[i+1] = matchTopic
	}
	// matcher is the final sequence matcher that signals a match when all underlying
	// matchers signal a match for consecutive log value indices.
	matcher := newMatchSequence(params, matchers)

	// processEpoch returns the potentially matching logs from the given epoch.
	processEpoch := func(epochIndex uint32) ([]*types.Log, error) {
		var logs []*types.Log
		// create a list of map indices to process
		fm, lm := epochIndex<<params.logMapsPerEpoch, (epochIndex+1)<<params.logMapsPerEpoch-1
		if fm < firstMap {
			fm = firstMap
		}
		if lm > lastMap {
			lm = lastMap
		}
		//
		mapIndices := make([]uint32, lm+1-fm)
		for i := range mapIndices {
			mapIndices[i] = fm + uint32(i)
		}
		// find potential matches
		matches, err := matcher.getMatches(ctx, mapIndices)
		if err != nil {
			return logs, err
		}
		// get the actual logs located at the matching log value indices
		for _, m := range matches {
			if m == nil {
				return nil, ErrMatchAll
			}
			mlogs, err := getLogsFromMatches(ctx, backend, firstIndex, lastIndex, m)
			if err != nil {
				return logs, err
			}
			logs = append(logs, mlogs...)
		}
		return logs, nil
	}

	type task struct {
		epochIndex uint32
		logs       []*types.Log
		err        error
		done       chan struct{}
	}

	taskCh := make(chan *task)
	var wg sync.WaitGroup
	defer func() {
		close(taskCh)
		wg.Wait()
	}()

	worker := func() {
		for task := range taskCh {
			if task == nil {
				break
			}
			task.logs, task.err = processEpoch(task.epochIndex)
			close(task.done)
		}
		wg.Done()
		return
	}

	for i := 0; i < 4; i++ {
		wg.Add(1)
		go worker()
	}

	var logs []*types.Log
	// startEpoch is the next task to send whenever a worker can accept it.
	// waitEpoch is the next task we are waiting for to finish in order to append
	// results in the correct order.
	startEpoch, waitEpoch := firstEpoch, firstEpoch
	tasks := make(map[uint32]*task)
	tasks[startEpoch] = &task{epochIndex: startEpoch, done: make(chan struct{})}
	for waitEpoch <= lastEpoch {
		select {
		case taskCh <- tasks[startEpoch]:
			startEpoch++
			if startEpoch <= lastEpoch {
				if tasks[startEpoch] == nil {
					tasks[startEpoch] = &task{epochIndex: startEpoch, done: make(chan struct{})}
				}
			}
		case <-tasks[waitEpoch].done:
			logs = append(logs, tasks[waitEpoch].logs...)
			if err := tasks[waitEpoch].err; err != nil {
				return logs, err
			}
			delete(tasks, waitEpoch)
			waitEpoch++
			if waitEpoch <= lastEpoch {
				if tasks[waitEpoch] == nil {
					tasks[waitEpoch] = &task{epochIndex: waitEpoch, done: make(chan struct{})}
				}
			}
		}
	}
	return logs, nil
}

// getLogsFromMatches returns the list of potentially matching logs located at
// the given list of matching log indices. Matches outside the firstIndex to
// lastIndex range are not returned.
func getLogsFromMatches(ctx context.Context, backend MatcherBackend, firstIndex, lastIndex uint64, matches potentialMatches) ([]*types.Log, error) {
	var logs []*types.Log
	for _, match := range matches {
		if match < firstIndex || match > lastIndex {
			continue
		}
		log, err := backend.GetLogByLvIndex(ctx, match)
		if err != nil {
			return logs, err
		}
		if log != nil {
			logs = append(logs, log)
		}
	}
	return logs, nil
}

// matcher interface is defined so that individual address/topic matchers can be
// combined into a pattern matcher (see matchAny and matchSequence).
type matcher interface {
	// getMatches takes a list of map indices and returns an equal number of
	// potentialMatches, one for each corresponding map index.
	// Note that the map index list is typically a list of the potentially
	// interesting maps from an epoch, plus sometimes the first map of the next
	// epoch if it is required for sequence matching.
	getMatches(ctx context.Context, mapIndices []uint32) ([]potentialMatches, error)
}

// singleMatcher implements matcher by returning matches for a single log value hash.
type singleMatcher struct {
	backend MatcherBackend
	value   common.Hash
}

// getMatches implements matcher
func (s *singleMatcher) getMatches(ctx context.Context, mapIndices []uint32) ([]potentialMatches, error) {
	params := s.backend.GetParams()
	results := make([]potentialMatches, len(mapIndices))
	for i, mapIndex := range mapIndices {
		filterRow, err := s.backend.GetFilterMapRow(ctx, mapIndex, params.rowIndex(mapIndex>>params.logMapsPerEpoch, s.value))
		if err != nil {
			return nil, err
		}
		results[i] = params.potentialMatches(filterRow, mapIndex, s.value)
	}
	return results, nil
}

// matchAny combinines a set of matchers and returns a match for every position
// where any of the underlying matchers signaled a match. A zero-length matchAny
// acts as a "wild card" that signals a potential match at every position.
type matchAny []matcher

// getMatches implements matcher
func (m matchAny) getMatches(ctx context.Context, mapIndices []uint32) ([]potentialMatches, error) {
	if len(m) == 0 {
		// return "wild card" results (potentialMatches(nil) is interpreted as a
		// potential match at every log value index of the map).
		return make([]potentialMatches, len(mapIndices)), nil
	}
	if len(m) == 1 {
		return m[0].getMatches(ctx, mapIndices)
	}
	matches := make([][]potentialMatches, len(m))
	for i, matcher := range m {
		var err error
		if matches[i], err = matcher.getMatches(ctx, mapIndices); err != nil {
			return nil, err
		}
	}
	results := make([]potentialMatches, len(mapIndices))
	merge := make([]potentialMatches, len(m))
	for i := range results {
		for j := range merge {
			merge[j] = matches[j][i]
		}
		results[i] = mergeResults(merge)
	}
	return results, nil
}

// mergeResults merges multiple lists of matches into a single one, preserving
// ascending order and filtering out any duplicates.
func mergeResults(results []potentialMatches) potentialMatches {
	if len(results) == 0 {
		return nil
	}
	var sumLen int
	for _, res := range results {
		if res == nil {
			// nil is a wild card; all indices in map range are potential matches
			return nil
		}
		sumLen += len(res)
	}
	merged := make(potentialMatches, 0, sumLen)
	for {
		best := -1
		for i, res := range results {
			if len(res) == 0 {
				continue
			}
			if best < 0 || res[0] < results[best][0] {
				best = i
			}
		}
		if best < 0 {
			return merged
		}
		if len(merged) == 0 || results[best][0] > merged[len(merged)-1] {
			merged = append(merged, results[best][0])
		}
		results[best] = results[best][1:]
	}
}

// matchSequence combines two matchers, a "base" and a "next" matcher with a
// positive integer offset so that the resulting matcher signals a match at log
// value index X when the base matcher returns a match at X and the next matcher
// gives a match at X+offset. Note that matchSequence can be used recursively to
// detect any log value sequence.
type matchSequence struct {
	baseEmptyRate, nextEmptyRate uint64 // first in struct to ensure 8 byte alignment
	params                       *Params
	base, next                   matcher
	offset                       uint64
	// *EmptyRate == totalCount << 32 + emptyCount (atomically accessed)
}

// newMatchSequence creates a recursive sequence matcher from a list of underlying
// matchers. The resulting matcher signals a match at log value index X when each
// underlying matcher matchers[i] returns a match at X+i.
func newMatchSequence(params *Params, matchers []matcher) matcher {
	if len(matchers) == 0 {
		panic("zero length sequence matchers are not allowed")
	}
	if len(matchers) == 1 {
		return matchers[0]
	}
	return &matchSequence{
		params: params,
		base:   newMatchSequence(params, matchers[:len(matchers)-1]),
		next:   matchers[len(matchers)-1],
		offset: uint64(len(matchers) - 1),
	}
}

// getMatches implements matcher
func (m *matchSequence) getMatches(ctx context.Context, mapIndices []uint32) ([]potentialMatches, error) {
	// decide whether to evaluate base or next matcher first
	baseEmptyRate := atomic.LoadUint64(&m.baseEmptyRate)
	nextEmptyRate := atomic.LoadUint64(&m.nextEmptyRate)
	baseTotal, baseEmpty := baseEmptyRate>>32, uint64(uint32(baseEmptyRate))
	nextTotal, nextEmpty := nextEmptyRate>>32, uint64(uint32(nextEmptyRate))
	baseFirst := baseEmpty*nextTotal >= nextEmpty*baseTotal/2

	var (
		baseRes, nextRes []potentialMatches
		baseIndices      []uint32
	)
	if baseFirst {
		// base first mode; request base matcher
		baseIndices = mapIndices
		var err error
		baseRes, err = m.base.getMatches(ctx, baseIndices)
		if err != nil {
			return nil, err
		}
	}

	// determine set of indices to request from next matcher
	nextIndices := make([]uint32, 0, len(mapIndices)*3/2)
	lastAdded := uint32(math.MaxUint32)
	for i, mapIndex := range mapIndices {
		if baseFirst && baseRes[i] != nil && len(baseRes[i]) == 0 {
			// do not request map index from next matcher if no results from base matcher
			continue
		}
		if lastAdded != mapIndex {
			nextIndices = append(nextIndices, mapIndex)
			lastAdded = mapIndex
		}
		if !baseFirst || baseRes[i] == nil || baseRes[i][len(baseRes[i])-1] >= (uint64(mapIndex+1)<<m.params.logValuesPerMap)-m.offset {
			nextIndices = append(nextIndices, mapIndex+1)
			lastAdded = mapIndex + 1
		}
	}

	if len(nextIndices) != 0 {
		// request next matcher
		var err error
		nextRes, err = m.next.getMatches(ctx, nextIndices)
		if err != nil {
			return nil, err
		}
	}

	if !baseFirst {
		// next first mode; determine set of indices to request from base matcher
		baseIndices = make([]uint32, 0, len(mapIndices))
		var nextPtr int
		for _, mapIndex := range mapIndices {
			// find corresponding results in nextRes
			for nextPtr+1 < len(nextIndices) && nextIndices[nextPtr] < mapIndex {
				nextPtr++
			}
			if nextPtr+1 >= len(nextIndices) {
				break
			}
			if nextIndices[nextPtr] != mapIndex || nextIndices[nextPtr+1] != mapIndex+1 {
				panic("invalid nextIndices")
			}
			next1, next2 := nextRes[nextPtr], nextRes[nextPtr+1]
			if next1 == nil || (len(next1) > 0 && next1[len(next1)-1] >= (uint64(mapIndex)<<m.params.logValuesPerMap)+m.offset) ||
				next2 == nil || (len(next2) > 0 && next2[0] < (uint64(mapIndex+1)<<m.params.logValuesPerMap)+m.offset) {
				baseIndices = append(baseIndices, mapIndex)
			}
		}
		if len(baseIndices) != 0 {
			// request base matcher
			var err error
			baseRes, err = m.base.getMatches(ctx, baseIndices)
			if err != nil {
				return nil, err
			}
		}
	}

	// all potential matches of base and next matchers obtained, update empty rates
	for _, res := range baseRes {
		if res != nil && len(res) == 0 {
			atomic.AddUint64(&m.baseEmptyRate, 0x100000001)
		} else {
			atomic.AddUint64(&m.baseEmptyRate, 0x100000000)
		}
	}
	for _, res := range nextRes {
		if res != nil && len(res) == 0 {
			atomic.AddUint64(&m.nextEmptyRate, 0x100000001)
		} else {
			atomic.AddUint64(&m.nextEmptyRate, 0x100000000)
		}
	}

	// define iterator functions to find base/next matcher results by map index
	var basePtr int
	baseResult := func(mapIndex uint32) potentialMatches {
		for basePtr < len(baseIndices) && baseIndices[basePtr] <= mapIndex {
			if baseIndices[basePtr] == mapIndex {
				return baseRes[basePtr]
			}
			basePtr++
		}
		return noMatches
	}
	var nextPtr int
	nextResult := func(mapIndex uint32) potentialMatches {
		for nextPtr < len(nextIndices) && nextIndices[nextPtr] <= mapIndex {
			if nextIndices[nextPtr] == mapIndex {
				return nextRes[nextPtr]
			}
			nextPtr++
		}
		return noMatches
	}

	// match corresponding base and next matcher results
	results := make([]potentialMatches, len(mapIndices))
	for i, mapIndex := range mapIndices {
		results[i] = m.matchResults(mapIndex, m.offset, baseResult(mapIndex), nextResult(mapIndex), nextResult(mapIndex+1))
	}
	return results, nil
}

// matchResults returns a list of sequence matches for the given mapIndex and
// offset based on the base matcher's results at mapIndex and the next matcher's
// results at mapIndex and mapIndex+1. Note that acquiring nextNextRes may be
// skipped and it can be substituted with an empty list if baseRes has no potential
// matches that could be sequence matched with anything that could be in nextNextRes.
func (m *matchSequence) matchResults(mapIndex uint32, offset uint64, baseRes, nextRes, nextNextRes potentialMatches) potentialMatches {
	if nextRes == nil || (baseRes != nil && len(baseRes) == 0) {
		// if nextRes is a wild card or baseRes is empty then the sequence matcher
		// result equals baseRes.
		return baseRes
	}
	if len(nextRes) > 0 {
		// discard items from nextRes whose corresponding base matcher results
		// with the negative offset applied would be located at mapIndex-1.
		start := 0
		for start < len(nextRes) && nextRes[start] < uint64(mapIndex)<<m.params.logValuesPerMap+offset {
			start++
		}
		nextRes = nextRes[start:]
	}
	if len(nextNextRes) > 0 {
		// discard items from nextNextRes whose corresponding base matcher results
		// with the negative offset applied would still be located at mapIndex+1.
		stop := 0
		for stop < len(nextNextRes) && nextNextRes[stop] < uint64(mapIndex+1)<<m.params.logValuesPerMap+offset {
			stop++
		}
		nextNextRes = nextNextRes[:stop]
	}
	maxLen := len(nextRes) + len(nextNextRes)
	if maxLen == 0 {
		return nextRes
	}
	if len(baseRes) < maxLen {
		maxLen = len(baseRes)
	}
	// iterate through baseRes, nextRes and nextNextRes and collect matching results.
	matchedRes := make(potentialMatches, 0, maxLen)
	for _, nextRes := range []potentialMatches{nextRes, nextNextRes} {
		if baseRes != nil {
			for len(nextRes) > 0 && len(baseRes) > 0 {
				if nextRes[0] > baseRes[0]+offset {
					baseRes = baseRes[1:]
				} else if nextRes[0] < baseRes[0]+offset {
					nextRes = nextRes[1:]
				} else {
					matchedRes = append(matchedRes, baseRes[0])
					baseRes = baseRes[1:]
					nextRes = nextRes[1:]
				}
			}
		} else {
			// baseRes is a wild card so just return next matcher results with
			// negative offset.
			for len(nextRes) > 0 {
				matchedRes = append(matchedRes, nextRes[0]-offset)
				nextRes = nextRes[1:]
			}
		}
	}
	return matchedRes
}
