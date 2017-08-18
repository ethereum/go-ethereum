// Copyright 2017 The go-ethereum Authors
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
package bloombits

import (
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bitutil"
	"github.com/ethereum/go-ethereum/core/types"
)

const channelCap = 100

// fetcher handles bit vector retrieval pipelines for a single bit index
type fetcher struct {
	bloomIndex  uint
	requestMap  map[uint64]fetchRequest
	requestLock sync.RWMutex
}

// fetchRequest represents the state of a bit vector requested from a fetcher. When a distRequest has been sent to the distributor but
// the data has not been delivered yet, queued is true. When delivered, it is stored in the data field and the delivered channel is closed.
type fetchRequest struct {
	data      []byte
	queued    bool
	delivered chan struct{}
}

// distRequest is sent by the fetcher to the distributor which groups and prioritizes these requests.
type distRequest struct {
	bloomIndex   uint
	sectionIndex uint64
}

// fetch creates a retrieval pipeline, receiving section indexes from sectionCh and returning the results
// in the same order through the returned channel. Multiple fetch instances of the same fetcher are allowed
// to run in parallel, in case the same bit index appears multiple times in the filter structure. Each section
// is requested only once, requests are sent to the request distributor (part of Matcher) through distCh.
func (f *fetcher) fetch(sectionCh chan uint64, distCh chan distRequest, stop chan struct{}, wg *sync.WaitGroup) chan []byte {
	dataCh := make(chan []byte, channelCap)
	returnCh := make(chan uint64, channelCap)
	wg.Add(2)

	go func() {
		defer wg.Done()
		defer close(returnCh)

		for {
			select {
			case <-stop:
				return
			case idx, ok := <-sectionCh:
				if !ok {
					return
				}

				req := false
				f.requestLock.Lock()
				r := f.requestMap[idx]
				if r.data == nil {
					req = !r.queued
					r.queued = true
					if r.delivered == nil {
						r.delivered = make(chan struct{})
					}
					f.requestMap[idx] = r
				}
				f.requestLock.Unlock()
				if req {
					distCh <- distRequest{bloomIndex: f.bloomIndex, sectionIndex: idx} // success is guaranteed, distibuteRequests shuts down after fetch
				}
				select {
				case <-stop:
					return
				case returnCh <- idx:
				}
			}
		}
	}()

	go func() {
		defer wg.Done()
		defer close(dataCh)

		for {
			select {
			case <-stop:
				return
			case idx, ok := <-returnCh:
				if !ok {
					return
				}

				f.requestLock.RLock()
				r := f.requestMap[idx]
				f.requestLock.RUnlock()

				if r.data == nil {
					select {
					case <-stop:
						return
					case <-r.delivered:
						f.requestLock.RLock()
						r = f.requestMap[idx]
						f.requestLock.RUnlock()
					}
				}
				select {
				case <-stop:
					return
				case dataCh <- r.data:
				}
			}
		}
	}()

	return dataCh
}

// deliver is called by the request distributor when a reply to a request has
// arrived
func (f *fetcher) deliver(sectionIdxList []uint64, data [][]byte) {
	f.requestLock.Lock()
	defer f.requestLock.Unlock()

	for i, sectionIdx := range sectionIdxList {
		r := f.requestMap[sectionIdx]
		if r.data != nil {
			panic("BloomBits section data delivered twice")
		}
		r.data = data[i]
		close(r.delivered)
		f.requestMap[sectionIdx] = r
	}
}

// Matcher is a pipelined structure of fetchers and logic matchers which perform
// binary AND/OR operations on the bitstreams, finally creating a stream of potential matches.
type Matcher struct {
	addresses   []types.BloomIndexList
	topics      [][]types.BloomIndexList
	fetchers    map[uint]*fetcher
	sectionSize uint64

	distCh     chan distRequest
	reqs       map[uint][]uint64
	freeQueues map[uint]struct{}
	allocQueue []chan uint
	running    bool
	stop       chan struct{}
	lock       sync.Mutex
	wg, distWg sync.WaitGroup
}

// NewMatcher creates a new Matcher instance
func NewMatcher(sectionSize uint64, addresses []common.Address, topics [][]common.Hash) *Matcher {
	m := &Matcher{
		fetchers:    make(map[uint]*fetcher),
		reqs:        make(map[uint][]uint64),
		freeQueues:  make(map[uint]struct{}),
		distCh:      make(chan distRequest, channelCap),
		sectionSize: sectionSize,
	}
	m.setAddresses(addresses)
	m.setTopics(topics)
	return m
}

// setAddresses matches only logs that are generated from addresses that are included
// in the given addresses.
func (m *Matcher) setAddresses(addresses []common.Address) {
	m.addresses = make([]types.BloomIndexList, len(addresses))
	for i, address := range addresses {
		m.addresses[i] = types.BloomIndexes(address.Bytes())
	}

	for _, bloomIndexList := range m.addresses {
		for _, bloomIndex := range bloomIndexList {
			m.newFetcher(bloomIndex)
		}
	}
}

// setTopics matches only logs that have topics matching the given topics.
func (m *Matcher) setTopics(topics [][]common.Hash) {
	m.topics = nil
loop:
	for _, topicList := range topics {
		t := make([]types.BloomIndexList, len(topicList))
		for i, topic := range topicList {
			if (topic == common.Hash{}) {
				continue loop
			}
			t[i] = types.BloomIndexes(topic.Bytes())
		}
		m.topics = append(m.topics, t)
	}

	for _, bloomIndexLists := range m.topics {
		for _, bloomIndexList := range bloomIndexLists {
			for _, bloomIndex := range bloomIndexList {
				m.newFetcher(bloomIndex)
			}
		}
	}
}

// match creates a daisy-chain of sub-matchers, one for the address set and one for each topic set, each
// sub-matcher receiving a section only if the previous ones have all found a potential match in one of
// the blocks of the section, then binary AND-ing its own matches and forwaring the result to the next one
func (m *Matcher) match(processCh chan partialMatches) chan partialMatches {
	indexLists := m.topics
	if len(m.addresses) > 0 {
		indexLists = append([][]types.BloomIndexList{m.addresses}, indexLists...)
	}
	m.distributeRequests()

	for _, subIndexList := range indexLists {
		processCh = m.subMatch(processCh, subIndexList)
	}
	return processCh
}

// partialMatches with a non-nil vector represents a section in which some sub-matchers have already
// found potential matches. Subsequent sub-matchers will binary AND their matches with this vector.
// If vector is nil, it represents a section to be processed by the first sub-matcher.
type partialMatches struct {
	sectionIndex uint64
	vector       []byte
}

// newFetcher adds a fetcher for the given bit index if it has not existed before
func (m *Matcher) newFetcher(idx uint) {
	if _, ok := m.fetchers[idx]; ok {
		return
	}
	f := &fetcher{
		bloomIndex: idx,
		requestMap: make(map[uint64]fetchRequest),
	}
	m.fetchers[idx] = f
}

// subMatch creates a sub-matcher that filters for a set of addresses or topics, binary OR-s those matches, then
// binary AND-s the result to the daisy-chain input (processCh) and forwards it to the daisy-chain output.
// The matches of each address/topic are calculated by fetching the given sections of the three bloom bit indexes belonging to
// that address/topic, and binary AND-ing those vectors together.
func (m *Matcher) subMatch(processCh chan partialMatches, bloomIndexLists []types.BloomIndexList) chan partialMatches {
	// set up fetchers
	fetchIndexChannels := make([][3]chan uint64, len(bloomIndexLists))
	fetchDataChannels := make([][3]chan []byte, len(bloomIndexLists))
	for i, bloomIndexList := range bloomIndexLists {
		for j, bloomIndex := range bloomIndexList {
			fetchIndexChannels[i][j] = make(chan uint64, channelCap)
			fetchDataChannels[i][j] = m.fetchers[bloomIndex].fetch(fetchIndexChannels[i][j], m.distCh, m.stop, &m.wg)
		}
	}

	fetchedCh := make(chan partialMatches, channelCap) // entries from processCh are forwarded here after fetches have been initiated
	resultsCh := make(chan partialMatches, channelCap)

	m.wg.Add(2)
	// goroutine for starting retrievals
	go func() {
		defer m.wg.Done()

		for {
			select {
			case <-m.stop:
				return
			case s, ok := <-processCh:
				if !ok {
					close(fetchedCh)
					for _, fetchIndexChs := range fetchIndexChannels {
						for _, fetchIndexCh := range fetchIndexChs {
							close(fetchIndexCh)
						}
					}
					return
				}

				for _, fetchIndexChs := range fetchIndexChannels {
					for _, fetchIndexCh := range fetchIndexChs {
						select {
						case <-m.stop:
							return
						case fetchIndexCh <- s.sectionIndex:
						}
					}
				}
				select {
				case <-m.stop:
					return
				case fetchedCh <- s:
				}
			}
		}
	}()

	// goroutine for processing retrieved data
	go func() {
		defer m.wg.Done()

		for {
			select {
			case <-m.stop:
				return
			case s, ok := <-fetchedCh:
				if !ok {
					close(resultsCh)
					return
				}

				var orVector []byte
				for _, fetchDataChs := range fetchDataChannels {
					var andVector []byte
					for _, fetchDataCh := range fetchDataChs {
						var data []byte
						select {
						case <-m.stop:
							return
						case data = <-fetchDataCh:
						}
						if andVector == nil {
							andVector = make([]byte, int(m.sectionSize/8))
							copy(andVector, data)
						} else {
							bitutil.ANDBytes(andVector, andVector, data)
						}
					}
					if orVector == nil {
						orVector = andVector
					} else {
						bitutil.ORBytes(orVector, orVector, andVector)
					}
				}

				if orVector == nil {
					orVector = make([]byte, int(m.sectionSize/8))
				}
				if s.vector != nil {
					bitutil.ANDBytes(orVector, orVector, s.vector)
				}
				if bitutil.TestBytes(orVector) {
					select {
					case <-m.stop:
						return
					case resultsCh <- partialMatches{s.sectionIndex, orVector}:
					}
				}
			}
		}
	}()

	return resultsCh
}

// Start starts the matching process and returns a stream of bloom matches in
// a given range of blocks.
// It returns a results channel immediately and stops if Stop is called or there
// are no more matches in the range (in which case the results channel is closed).
// Start/Stop can be called multiple times for different ranges, in which case already
// delivered bit vectors are not requested again.
func (m *Matcher) Start(begin, end uint64) chan uint64 {
	m.stop = make(chan struct{})
	processCh := make(chan partialMatches, channelCap)
	resultsCh := make(chan uint64, channelCap)

	res := m.match(processCh)

	startSection := begin / m.sectionSize
	endSection := end / m.sectionSize

	m.wg.Add(2)
	go func() {
		defer m.wg.Done()
		defer close(processCh)

		for i := startSection; i <= endSection; i++ {
			select {
			case processCh <- partialMatches{i, nil}:
			case <-m.stop:
				return
			}
		}
	}()

	go func() {
		defer m.wg.Done()
		defer close(resultsCh)

		for {
			select {
			case r, ok := <-res:
				if !ok {
					return
				}
				sectionStart := r.sectionIndex * m.sectionSize
				s := sectionStart
				if begin > s {
					s = begin
				}
				e := sectionStart + m.sectionSize - 1
				if end < e {
					e = end
				}
				for i := s; i <= e; i++ {
					b := r.vector[(i-sectionStart)/8]
					bit := 7 - i%8
					if b != 0 {
						if b&(1<<bit) != 0 {
							select {
							case <-m.stop:
								return
							case resultsCh <- i:
							}
						}
					} else {
						i += bit
					}
				}

			case <-m.stop:
				return
			}
		}
	}()

	return resultsCh
}

// Stop stops the matching process
func (m *Matcher) Stop() {
	close(m.stop)
	m.distWg.Wait()
}

// distributeRequests receives requests from the fetchers and either queues them
// or immediately forwards them to one of the waiting NextRequest functions.
// Requests with a lower section idx are always prioritized.
func (m *Matcher) distributeRequests() {
	m.distWg.Add(1)
	stopDist := make(chan struct{})
	go func() {
		<-m.stop
		m.wg.Wait()
		close(stopDist)
	}()

	m.running = true

	go func() {
		for {
			select {
			case r := <-m.distCh:
				m.lock.Lock()
				queue := m.reqs[r.bloomIndex]
				i := 0
				for i < len(queue) && r.sectionIndex > queue[i] {
					i++
				}
				queue = append(queue, 0)
				copy(queue[i+1:], queue[i:len(queue)-1])
				queue[i] = r.sectionIndex
				m.reqs[r.bloomIndex] = queue
				if len(queue) == 1 {
					m.freeQueue(r.bloomIndex)
				}
				m.lock.Unlock()
			case <-stopDist:
				m.lock.Lock()
				for _, ch := range m.allocQueue {
					close(ch)
				}
				m.allocQueue = nil
				m.running = false
				m.lock.Unlock()
				m.distWg.Done()
				return
			}
		}
	}()
}

// freeQueue marks a queue as free if there are no AllocSectionQueue functions
// waiting for allocation. If there is someone waiting, the queue is immediately
// allocated.
func (m *Matcher) freeQueue(bloomIndex uint) {
	if len(m.allocQueue) > 0 {
		m.allocQueue[0] <- bloomIndex
		m.allocQueue = m.allocQueue[1:]
	} else {
		m.freeQueues[bloomIndex] = struct{}{}
	}
}

// AllocSectionQueue allocates a queue of requested section indexes belonging to the same
// bloom bit index for a client process that can either immediately fetch the contents
// of the queue or wait a little while for more section indexes to be requested.
func (m *Matcher) AllocSectionQueue() (uint, bool) {
	m.lock.Lock()
	if !m.running {
		m.lock.Unlock()
		return 0, false
	}

	var allocCh chan uint
	if len(m.freeQueues) > 0 {
		var (
			found       bool
			bestSection uint64
			bestIndex   uint
		)
		for bloomIndex, _ := range m.freeQueues {
			if !found || m.reqs[bloomIndex][0] < bestSection {
				found = true
				bestIndex = bloomIndex
				bestSection = m.reqs[bloomIndex][0]
			}
		}
		delete(m.freeQueues, bestIndex)
		m.lock.Unlock()
		return bestIndex, true
	} else {
		allocCh = make(chan uint)
		m.allocQueue = append(m.allocQueue, allocCh)
	}
	m.lock.Unlock()

	bloomIndex, ok := <-allocCh
	return bloomIndex, ok
}

// SectionCount returns the length of the section index queue belonging to the given bloom bit index
func (m *Matcher) SectionCount(bloomIndex uint) int {
	m.lock.Lock()
	defer m.lock.Unlock()

	return len(m.reqs[bloomIndex])
}

// FetchSections fetches all or part of an already allocated queue and deallocates it
func (m *Matcher) FetchSections(bloomIndex uint, maxCount int) []uint64 {
	m.lock.Lock()
	defer m.lock.Unlock()

	queue := m.reqs[bloomIndex]
	if maxCount < len(queue) {
		// return only part of the existing queue, mark the rest as free
		m.reqs[bloomIndex] = queue[maxCount:]
		m.freeQueue(bloomIndex)
		return queue[:maxCount]
	} else {
		// return the entire queue
		delete(m.reqs, bloomIndex)
		return queue
	}
}

// Deliver delivers a bit vector to the appropriate fetcher.
// It is possible to deliver data even after Stop has been called. Once a vector has been
// requested, the matcher will keep waiting for delivery.
func (m *Matcher) Deliver(bloomIndex uint, sectionIdxList []uint64, data [][]byte) {
	m.fetchers[bloomIndex].deliver(sectionIdxList, data)
}
