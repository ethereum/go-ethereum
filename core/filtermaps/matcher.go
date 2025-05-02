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
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

const doRuntimeStats = false

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
	GetFilterMapRow(ctx context.Context, mapIndex, rowIndex uint32, baseLayerOnly bool) (FilterRow, error)
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
	IndexedView *ChainView
	// block range where the index has not changed since the last matcher sync
	// and therefore the set of matches found in this region is guaranteed to
	// be valid and complete.
	ValidBlocks common.Range[uint64]
	// block range indexed according to the given chain head.
	IndexedBlocks common.Range[uint64]
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
		return nil, fmt.Errorf("failed to retrieve log value pointer for first block %d: %v", firstBlock, err)
	}
	lastIndex, err := backend.GetBlockLvPointer(ctx, lastBlock+1)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve log value pointer after last block %d: %v", lastBlock, err)
	}
	if lastIndex > 0 {
		lastIndex--
	}

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

	m := &matcherEnv{
		ctx:        ctx,
		backend:    backend,
		params:     params,
		matcher:    matcher,
		firstIndex: firstIndex,
		lastIndex:  lastIndex,
		firstMap:   uint32(firstIndex >> params.logValuesPerMap),
		lastMap:    uint32(lastIndex >> params.logValuesPerMap),
	}

	start := time.Now()
	res, err := m.process()
	matchRequestTimer.Update(time.Since(start))

	if doRuntimeStats {
		log.Info("Log search finished", "elapsed", time.Since(start))
		for i, ma := range matchers {
			for j, m := range ma.(matchAny) {
				log.Info("Single matcher stats", "matchSequence", i, "matchAny", j)
				m.(*singleMatcher).stats.print()
			}
		}
		log.Info("Get log stats")
		m.getLogStats.print()
	}
	return res, err
}

type matcherEnv struct {
	getLogStats           runtimeStats // 64 bit aligned
	ctx                   context.Context
	backend               MatcherBackend
	params                *Params
	matcher               matcher
	firstIndex, lastIndex uint64
	firstMap, lastMap     uint32
}

func (m *matcherEnv) process() ([]*types.Log, error) {
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
			task.logs, task.err = m.processEpoch(task.epochIndex)
			close(task.done)
		}
		wg.Done()
	}

	for range 4 {
		wg.Add(1)
		go worker()
	}

	firstEpoch, lastEpoch := m.firstMap>>m.params.logMapsPerEpoch, m.lastMap>>m.params.logMapsPerEpoch
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
				if err == ErrMatchAll {
					matchAllMeter.Mark(1)
					return logs, err
				}
				return logs, fmt.Errorf("failed to process log index epoch %d: %v", waitEpoch, err)
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

// processEpoch returns the potentially matching logs from the given epoch.
func (m *matcherEnv) processEpoch(epochIndex uint32) ([]*types.Log, error) {
	start := time.Now()
	var logs []*types.Log
	// create a list of map indices to process
	fm, lm := epochIndex<<m.params.logMapsPerEpoch, (epochIndex+1)<<m.params.logMapsPerEpoch-1
	if fm < m.firstMap {
		fm = m.firstMap
	}
	if lm > m.lastMap {
		lm = m.lastMap
	}
	//
	mapIndices := make([]uint32, lm+1-fm)
	for i := range mapIndices {
		mapIndices[i] = fm + uint32(i)
	}
	// find potential matches
	matches, err := m.getAllMatches(mapIndices)
	if err != nil {
		return logs, err
	}
	// get the actual logs located at the matching log value indices
	var st int
	m.getLogStats.setState(&st, stGetLog)
	defer m.getLogStats.setState(&st, stNone)
	for _, match := range matches {
		if match == nil {
			return nil, ErrMatchAll
		}
		mlogs, err := m.getLogsFromMatches(match)
		if err != nil {
			return logs, err
		}
		logs = append(logs, mlogs...)
	}
	m.getLogStats.addAmount(st, int64(len(logs)))
	matchEpochTimer.Update(time.Since(start))
	return logs, nil
}

// getLogsFromMatches returns the list of potentially matching logs located at
// the given list of matching log indices. Matches outside the firstIndex to
// lastIndex range are not returned.
func (m *matcherEnv) getLogsFromMatches(matches potentialMatches) ([]*types.Log, error) {
	var logs []*types.Log
	for _, match := range matches {
		if match < m.firstIndex || match > m.lastIndex {
			continue
		}
		log, err := m.backend.GetLogByLvIndex(m.ctx, match)
		if err != nil {
			return logs, fmt.Errorf("failed to retrieve log at index %d: %v", match, err)
		}
		if log != nil {
			logs = append(logs, log)
		}
		matchLogLookup.Mark(1)
	}
	return logs, nil
}

// getAllMatches creates an instance for a given matcher and set of map indices,
// iterates through mapping layers and collects all results, then returns all
// results in the same order as the map indices were specified.
func (m *matcherEnv) getAllMatches(mapIndices []uint32) ([]potentialMatches, error) {
	instance := m.matcher.newInstance(mapIndices)
	resultsMap := make(map[uint32]potentialMatches)
	for layerIndex := uint32(0); len(resultsMap) < len(mapIndices); layerIndex++ {
		results, err := instance.getMatchesForLayer(m.ctx, layerIndex)
		if err != nil {
			return nil, err
		}
		for _, result := range results {
			resultsMap[result.mapIndex] = result.matches
		}
	}
	matches := make([]potentialMatches, len(mapIndices))
	for i, mapIndex := range mapIndices {
		matches[i] = resultsMap[mapIndex]
	}
	return matches, nil
}

// matcher defines a general abstraction for any matcher configuration that
// can instantiate a matcherInstance.
type matcher interface {
	newInstance(mapIndices []uint32) matcherInstance
}

// matcherInstance defines a general abstraction for a matcher configuration
// working on a specific set of map indices and eventually returning a list of
// potentially matching log value indices.
// Note that processing happens per mapping layer, each call returning a set
// of results for the maps where the processing has been finished at the given
// layer. Map indices can also be dropped before a result is returned for them
// in case the result is no longer interesting. Dropping indices twice or after
// a result has been returned has no effect. Exactly one matcherResult is
// returned per requested map index unless dropped.
type matcherInstance interface {
	getMatchesForLayer(ctx context.Context, layerIndex uint32) ([]matcherResult, error)
	dropIndices(mapIndices []uint32)
}

// matcherResult contains the list of potentially matching log value indices
// for a given map index.
type matcherResult struct {
	mapIndex uint32
	matches  potentialMatches
}

// singleMatcher implements matcher by returning matches for a single log value hash.
type singleMatcher struct {
	backend MatcherBackend
	value   common.Hash
	stats   runtimeStats
}

// singleMatcherInstance is an instance of singleMatcher.
type singleMatcherInstance struct {
	*singleMatcher
	mapIndices []uint32
	filterRows map[uint32][]FilterRow
}

// newInstance creates a new instance of singleMatcher.
func (m *singleMatcher) newInstance(mapIndices []uint32) matcherInstance {
	filterRows := make(map[uint32][]FilterRow)
	for _, idx := range mapIndices {
		filterRows[idx] = []FilterRow{}
	}
	copiedIndices := make([]uint32, len(mapIndices))
	copy(copiedIndices, mapIndices)
	return &singleMatcherInstance{
		singleMatcher: m,
		mapIndices:    copiedIndices,
		filterRows:    filterRows,
	}
}

// getMatchesForLayer implements matcherInstance.
func (m *singleMatcherInstance) getMatchesForLayer(ctx context.Context, layerIndex uint32) (results []matcherResult, err error) {
	var st int
	m.stats.setState(&st, stOther)
	params := m.backend.GetParams()
	maskedMapIndex, rowIndex := uint32(math.MaxUint32), uint32(0)
	for _, mapIndex := range m.mapIndices {
		filterRows, ok := m.filterRows[mapIndex]
		if !ok {
			continue
		}
		if mm := params.maskedMapIndex(mapIndex, layerIndex); mm != maskedMapIndex {
			// only recalculate rowIndex when necessary
			maskedMapIndex = mm
			rowIndex = params.rowIndex(mapIndex, layerIndex, m.value)
		}
		if layerIndex == 0 {
			m.stats.setState(&st, stFetchFirst)
		} else {
			m.stats.setState(&st, stFetchMore)
		}
		filterRow, err := m.backend.GetFilterMapRow(ctx, mapIndex, rowIndex, layerIndex == 0)
		if err != nil {
			m.stats.setState(&st, stNone)
			return nil, fmt.Errorf("failed to retrieve filter map %d row %d: %v", mapIndex, rowIndex, err)
		}
		if layerIndex == 0 {
			matchBaseRowAccessMeter.Mark(1)
			matchBaseRowSizeMeter.Mark(int64(len(filterRow)))
		} else {
			matchExtRowAccessMeter.Mark(1)
			matchExtRowSizeMeter.Mark(int64(len(filterRow)))
		}
		m.stats.addAmount(st, int64(len(filterRow)))
		m.stats.setState(&st, stOther)
		filterRows = append(filterRows, filterRow)
		if uint32(len(filterRow)) < params.maxRowLength(layerIndex) {
			m.stats.setState(&st, stProcess)
			matches := params.potentialMatches(filterRows, mapIndex, m.value)
			m.stats.addAmount(st, int64(len(matches)))
			results = append(results, matcherResult{
				mapIndex: mapIndex,
				matches:  matches,
			})
			m.stats.setState(&st, stOther)
			delete(m.filterRows, mapIndex)
		} else {
			m.filterRows[mapIndex] = filterRows
		}
	}
	m.cleanMapIndices()
	m.stats.setState(&st, stNone)
	return results, nil
}

// dropIndices implements matcherInstance.
func (m *singleMatcherInstance) dropIndices(dropIndices []uint32) {
	for _, mapIndex := range dropIndices {
		delete(m.filterRows, mapIndex)
	}
	m.cleanMapIndices()
}

// cleanMapIndices removes map indices from the list if there is no matching
// filterRows entry because a result has been returned or the index has been
// dropped.
func (m *singleMatcherInstance) cleanMapIndices() {
	var j int
	for i, mapIndex := range m.mapIndices {
		if _, ok := m.filterRows[mapIndex]; ok {
			if i != j {
				m.mapIndices[j] = mapIndex
			}
			j++
		}
	}
	m.mapIndices = m.mapIndices[:j]
}

// matchAny combinines a set of matchers and returns a match for every position
// where any of the underlying matchers signaled a match. A zero-length matchAny
// acts as a "wild card" that signals a potential match at every position.
type matchAny []matcher

// matchAnyInstance is an instance of matchAny.
type matchAnyInstance struct {
	matchAny
	childInstances []matcherInstance
	childResults   map[uint32]matchAnyResults
}

// matchAnyResults is used by matchAnyInstance to collect results from all
// child matchers for a specific map index. Once all results has been received
// a merged result is returned for the given map and this structure is discarded.
type matchAnyResults struct {
	matches  []potentialMatches
	done     []bool
	needMore int
}

// newInstance creates a new instance of matchAny.
func (m matchAny) newInstance(mapIndices []uint32) matcherInstance {
	if len(m) == 1 {
		return m[0].newInstance(mapIndices)
	}
	childResults := make(map[uint32]matchAnyResults)
	for _, idx := range mapIndices {
		childResults[idx] = matchAnyResults{
			matches:  make([]potentialMatches, len(m)),
			done:     make([]bool, len(m)),
			needMore: len(m),
		}
	}
	childInstances := make([]matcherInstance, len(m))
	for i, matcher := range m {
		childInstances[i] = matcher.newInstance(mapIndices)
	}
	return &matchAnyInstance{
		matchAny:       m,
		childInstances: childInstances,
		childResults:   childResults,
	}
}

// getMatchesForLayer implements matcherInstance.
func (m *matchAnyInstance) getMatchesForLayer(ctx context.Context, layerIndex uint32) (mergedResults []matcherResult, err error) {
	if len(m.matchAny) == 0 {
		// return "wild card" results (potentialMatches(nil) is interpreted as a
		// potential match at every log value index of the map).
		mergedResults = make([]matcherResult, len(m.childResults))
		var i int
		for mapIndex := range m.childResults {
			mergedResults[i] = matcherResult{mapIndex: mapIndex, matches: nil}
			i++
		}
		return mergedResults, nil
	}
	for i, childInstance := range m.childInstances {
		results, err := childInstance.getMatchesForLayer(ctx, layerIndex)
		if err != nil {
			return nil, fmt.Errorf("failed to evaluate child matcher on layer %d: %v", layerIndex, err)
		}
		for _, result := range results {
			mr, ok := m.childResults[result.mapIndex]
			if !ok || mr.done[i] {
				continue
			}
			mr.done[i] = true
			mr.matches[i] = result.matches
			mr.needMore--
			if mr.needMore == 0 || result.matches == nil {
				mergedResults = append(mergedResults, matcherResult{
					mapIndex: result.mapIndex,
					matches:  mergeResults(mr.matches),
				})
				delete(m.childResults, result.mapIndex)
			} else {
				m.childResults[result.mapIndex] = mr
			}
		}
	}
	return mergedResults, nil
}

// dropIndices implements matcherInstance.
func (m *matchAnyInstance) dropIndices(dropIndices []uint32) {
	for _, childInstance := range m.childInstances {
		childInstance.dropIndices(dropIndices)
	}
	for _, mapIndex := range dropIndices {
		delete(m.childResults, mapIndex)
	}
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
	params               *Params
	base, next           matcher
	offset               uint64
	statsLock            sync.Mutex
	baseStats, nextStats matchOrderStats
}

// newInstance creates a new instance of matchSequence.
func (m *matchSequence) newInstance(mapIndices []uint32) matcherInstance {
	// determine set of indices to request from next matcher
	needMatched := make(map[uint32]struct{})
	baseRequested := make(map[uint32]struct{})
	nextRequested := make(map[uint32]struct{})
	for _, mapIndex := range mapIndices {
		needMatched[mapIndex] = struct{}{}
		baseRequested[mapIndex] = struct{}{}
		nextRequested[mapIndex] = struct{}{}
	}
	return &matchSequenceInstance{
		matchSequence: m,
		baseInstance:  m.base.newInstance(mapIndices),
		nextInstance:  m.next.newInstance(mapIndices),
		needMatched:   needMatched,
		baseRequested: baseRequested,
		nextRequested: nextRequested,
		baseResults:   make(map[uint32]potentialMatches),
		nextResults:   make(map[uint32]potentialMatches),
	}
}

// matchOrderStats collects statistics about the evaluating cost and the
// occurrence of empty result sets from both base and next child matchers.
// This allows the optimization of the evaluation order by evaluating the
// child first that is cheaper and/or gives empty results more often and not
// evaluating the other child in most cases.
// Note that matchOrderStats is specific to matchSequence and the results are
// carried over to future instances as the results are mostly useful when
// evaluating layer zero of each instance. For this reason it should be used
// in a thread safe way as is may be accessed from multiple worker goroutines.
type matchOrderStats struct {
	totalCount, nonEmptyCount, totalCost uint64
}

// add collects statistics after a child has been evaluated for a certain layer.
func (ms *matchOrderStats) add(empty bool, layerIndex uint32) {
	if empty && layerIndex != 0 {
		// matchers may be evaluated for higher layers after all results have
		// been returned. Also, empty results are not relevant when previous
		// layers yielded matches already, so these cases can be ignored.
		return
	}
	ms.totalCount++
	if !empty {
		ms.nonEmptyCount++
	}
	ms.totalCost += uint64(layerIndex + 1)
}

// mergeStats merges two sets of matchOrderStats.
func (ms *matchOrderStats) mergeStats(add matchOrderStats) {
	ms.totalCount += add.totalCount
	ms.nonEmptyCount += add.nonEmptyCount
	ms.totalCost += add.totalCost
}

// baseFirst returns true if the base child matcher should be evaluated first.
func (m *matchSequence) baseFirst() bool {
	m.statsLock.Lock()
	bf := float64(m.baseStats.totalCost)*float64(m.nextStats.totalCount)+
		float64(m.baseStats.nonEmptyCount)*float64(m.nextStats.totalCost) <
		float64(m.baseStats.totalCost)*float64(m.nextStats.nonEmptyCount)+
			float64(m.nextStats.totalCost)*float64(m.baseStats.totalCount)
	m.statsLock.Unlock()
	return bf
}

// mergeBaseStats merges a set of matchOrderStats into the base matcher stats.
func (m *matchSequence) mergeBaseStats(stats matchOrderStats) {
	m.statsLock.Lock()
	m.baseStats.mergeStats(stats)
	m.statsLock.Unlock()
}

// mergeNextStats merges a set of matchOrderStats into the next matcher stats.
func (m *matchSequence) mergeNextStats(stats matchOrderStats) {
	m.statsLock.Lock()
	m.nextStats.mergeStats(stats)
	m.statsLock.Unlock()
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

// matchSequenceInstance is an instance of matchSequence.
type matchSequenceInstance struct {
	*matchSequence
	baseInstance, nextInstance                matcherInstance
	baseRequested, nextRequested, needMatched map[uint32]struct{}
	baseResults, nextResults                  map[uint32]potentialMatches
}

// getMatchesForLayer implements matcherInstance.
func (m *matchSequenceInstance) getMatchesForLayer(ctx context.Context, layerIndex uint32) (matchedResults []matcherResult, err error) {
	// decide whether to evaluate base or next matcher first
	baseFirst := m.baseFirst()
	if baseFirst {
		if err := m.evalBase(ctx, layerIndex); err != nil {
			return nil, err
		}
	}
	if err := m.evalNext(ctx, layerIndex); err != nil {
		return nil, err
	}
	if !baseFirst {
		if err := m.evalBase(ctx, layerIndex); err != nil {
			return nil, err
		}
	}
	// evaluate and return matched results where possible
	for mapIndex := range m.needMatched {
		if _, ok := m.baseRequested[mapIndex]; ok {
			continue
		}
		if _, ok := m.nextRequested[mapIndex]; ok {
			continue
		}
		matchedResults = append(matchedResults, matcherResult{
			mapIndex: mapIndex,
			matches:  m.params.matchResults(mapIndex, m.offset, m.baseResults[mapIndex], m.nextResults[mapIndex]),
		})
		delete(m.needMatched, mapIndex)
	}
	return matchedResults, nil
}

// dropIndices implements matcherInstance.
func (m *matchSequenceInstance) dropIndices(dropIndices []uint32) {
	for _, mapIndex := range dropIndices {
		delete(m.needMatched, mapIndex)
	}
	var dropBase, dropNext []uint32
	for _, mapIndex := range dropIndices {
		if m.dropBase(mapIndex) {
			dropBase = append(dropBase, mapIndex)
		}
	}
	m.baseInstance.dropIndices(dropBase)
	for _, mapIndex := range dropIndices {
		if m.dropNext(mapIndex) {
			dropNext = append(dropNext, mapIndex)
		}
	}
	m.nextInstance.dropIndices(dropNext)
}

// evalBase evaluates the base child matcher and drops map indices from the
// next matcher if possible.
func (m *matchSequenceInstance) evalBase(ctx context.Context, layerIndex uint32) error {
	results, err := m.baseInstance.getMatchesForLayer(ctx, layerIndex)
	if err != nil {
		return fmt.Errorf("failed to evaluate base matcher on layer %d: %v", layerIndex, err)
	}
	var (
		dropIndices []uint32
		stats       matchOrderStats
	)
	for _, r := range results {
		m.baseResults[r.mapIndex] = r.matches
		delete(m.baseRequested, r.mapIndex)
		stats.add(r.matches != nil && len(r.matches) == 0, layerIndex)
	}
	m.mergeBaseStats(stats)
	for _, r := range results {
		if m.dropNext(r.mapIndex) {
			dropIndices = append(dropIndices, r.mapIndex)
		}
	}
	if len(dropIndices) > 0 {
		m.nextInstance.dropIndices(dropIndices)
	}
	return nil
}

// evalNext evaluates the next child matcher and drops map indices from the
// base matcher if possible.
func (m *matchSequenceInstance) evalNext(ctx context.Context, layerIndex uint32) error {
	results, err := m.nextInstance.getMatchesForLayer(ctx, layerIndex)
	if err != nil {
		return fmt.Errorf("failed to evaluate next matcher on layer %d: %v", layerIndex, err)
	}
	var (
		dropIndices []uint32
		stats       matchOrderStats
	)
	for _, r := range results {
		m.nextResults[r.mapIndex] = r.matches
		delete(m.nextRequested, r.mapIndex)
		stats.add(r.matches != nil && len(r.matches) == 0, layerIndex)
	}
	m.mergeNextStats(stats)
	for _, r := range results {
		if m.dropBase(r.mapIndex) {
			dropIndices = append(dropIndices, r.mapIndex)
		}
	}
	if len(dropIndices) > 0 {
		m.baseInstance.dropIndices(dropIndices)
	}
	return nil
}

// dropBase checks whether the given map index can be dropped from the base
// matcher based on the known results from the next matcher and removes it
// from the internal requested set and returns true if possible.
func (m *matchSequenceInstance) dropBase(mapIndex uint32) bool {
	if _, ok := m.baseRequested[mapIndex]; !ok {
		return false
	}
	if _, ok := m.needMatched[mapIndex]; ok {
		if next := m.nextResults[mapIndex]; next == nil || len(next) > 0 {
			return false
		}
	}
	delete(m.baseRequested, mapIndex)
	return true
}

// dropNext checks whether the given map index can be dropped from the next
// matcher based on the known results from the base matcher and removes it
// from the internal requested set and returns true if possible.
func (m *matchSequenceInstance) dropNext(mapIndex uint32) bool {
	if _, ok := m.nextRequested[mapIndex]; !ok {
		return false
	}
	if _, ok := m.needMatched[mapIndex]; ok {
		if base := m.baseResults[mapIndex]; base == nil || len(base) > 0 {
			return false
		}
	}
	delete(m.nextRequested, mapIndex)
	return true
}

// matchResults returns a list of sequence matches for the given mapIndex and
// offset based on the base matcher's results at mapIndex and the next matcher's
// results at mapIndex and mapIndex+1. Note that acquiring nextNextRes may be
// skipped and it can be substituted with an empty list if baseRes has no potential
// matches that could be sequence matched with anything that could be in nextNextRes.
func (params *Params) matchResults(mapIndex uint32, offset uint64, baseRes, nextRes potentialMatches) potentialMatches {
	if nextRes == nil || (baseRes != nil && len(baseRes) == 0) {
		// if nextRes is a wild card or baseRes is empty then the sequence matcher
		// result equals baseRes.
		return baseRes
	}
	if baseRes == nil || len(nextRes) == 0 {
		// if baseRes is a wild card or nextRes is empty then the sequence matcher
		// result is the items of nextRes with a negative offset applied.
		result := make(potentialMatches, 0, len(nextRes))
		min := (uint64(mapIndex) << params.logValuesPerMap) + offset
		for _, v := range nextRes {
			if v >= min {
				result = append(result, v-offset)
			}
		}
		return result
	}
	// iterate through baseRes and nextRes in parallel and collect matching results.
	maxLen := len(baseRes)
	if l := len(nextRes); l < maxLen {
		maxLen = l
	}
	matchedRes := make(potentialMatches, 0, maxLen)
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
	return matchedRes
}

// runtimeStats collects processing time statistics while searching in the log
// index. Used only when the doRuntimeStats global flag is true.
type runtimeStats struct {
	dt, cnt, amount [stCount]int64
}

const (
	stNone = iota
	stFetchFirst
	stFetchMore
	stProcess
	stGetLog
	stOther
	stCount
)

var stNames = []string{"", "fetchFirst", "fetchMore", "process", "getLog", "other"}

// set sets the processing state to one of the pre-defined constants.
// Processing time spent in each state is measured separately.
func (ts *runtimeStats) setState(state *int, newState int) {
	if !doRuntimeStats || newState == *state {
		return
	}
	now := int64(mclock.Now())
	atomic.AddInt64(&ts.dt[*state], now)
	atomic.AddInt64(&ts.dt[newState], -now)
	atomic.AddInt64(&ts.cnt[newState], 1)
	*state = newState
}

func (ts *runtimeStats) addAmount(state int, amount int64) {
	atomic.AddInt64(&ts.amount[state], amount)
}

// print prints the collected statistics.
func (ts *runtimeStats) print() {
	for i := 1; i < stCount; i++ {
		log.Info("Matcher stats", "name", stNames[i], "dt", time.Duration(ts.dt[i]), "count", ts.cnt[i], "amount", ts.amount[i])
	}
}
