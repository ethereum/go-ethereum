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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/bitutil"
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	maxRequestLength = 16
	channelCap       = 100
)

// fetcher handles bit vector retrieval pipelines for a single bit index
type fetcher struct {
	bitIdx  uint
	reqMap  map[uint64]req
	reqLock sync.RWMutex
}

type req struct {
	data    []byte
	queued  bool
	fetched chan struct{}
}

type distReq struct {
	bitIdx     uint
	sectionIdx uint64
}

// fetch creates a retrieval pipeline, receiving section indexes from sectionCh and returning the results
// in the same order through the returned channel. Multiple fetch instances of the same fetcher are allowed
// to run in parallel, in case the same bit index appears multiple times in the filter structure. Each section
// is requested only once, requests are sent to the request distributor (part of Matcher) through distCh.
func (f *fetcher) fetch(sectionCh chan uint64, distCh chan distReq, stop chan struct{}, wg *sync.WaitGroup) chan []byte {
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
				f.reqLock.Lock()
				r := f.reqMap[idx]
				if r.data == nil {
					req = !r.queued
					r.queued = true
					if r.fetched == nil {
						r.fetched = make(chan struct{})
					}
					f.reqMap[idx] = r
				}
				f.reqLock.Unlock()
				if req {
					distCh <- distReq{bitIdx: f.bitIdx, sectionIdx: idx} // success is guaranteed, distibuteRequests shuts down after fetch
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

				f.reqLock.RLock()
				r := f.reqMap[idx]
				f.reqLock.RUnlock()

				if r.data == nil {
					select {
					case <-stop:
						return
					case <-r.fetched:
						f.reqLock.RLock()
						r = f.reqMap[idx]
						f.reqLock.RUnlock()
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
	f.reqLock.Lock()
	defer f.reqLock.Unlock()

	for i, idx := range sectionIdxList {
		r := f.reqMap[idx]
		if r.data != nil {
			panic("BloomBits section data delivered twice")
		}
		r.data = data[i]
		close(r.fetched)
		f.reqMap[idx] = r
	}
}

// Matcher is a pipelined structure of fetchers and logic matchers which perform
// binary AND/OR operations on the bitstreams, finally creating a stream of potential matches.
type Matcher struct {
	addresses   []types.BloomIndexList
	topics      [][]types.BloomIndexList
	fetchers    map[uint]*fetcher
	sectionSize uint64

	distCh       chan distReq
	reqs         map[uint][]uint64
	getNextReqCh chan chan nextRequests
	wg, distWg   sync.WaitGroup
}

// NewMatcher creates a new Matcher instance
func NewMatcher(sectionSize uint64) *Matcher {
	return &Matcher{fetchers: make(map[uint]*fetcher), reqs: make(map[uint][]uint64), distCh: make(chan distReq, channelCap), sectionSize: sectionSize}
}

// SetAddresses matches only logs that are generated from addresses that are included
// in the given addresses.
func (m *Matcher) SetAddresses(addr []common.Address) {
	m.addresses = make([]types.BloomIndexList, len(addr))
	for i, b := range addr {
		m.addresses[i] = types.BloomIndexes(b.Bytes())
	}

	for _, idxs := range m.addresses {
		for _, idx := range idxs {
			m.newFetcher(idx)
		}
	}
}

// SetTopics matches only logs that have topics matching the given topics.
func (m *Matcher) SetTopics(topics [][]common.Hash) {
	m.topics = nil
loop:
	for _, topicList := range topics {
		t := make([]types.BloomIndexList, len(topicList))
		for i, b := range topicList {
			if (b == common.Hash{}) {
				continue loop
			}
			t[i] = types.BloomIndexes(b.Bytes())
		}
		m.topics = append(m.topics, t)
	}

	for _, idxss := range m.topics {
		for _, idxs := range idxss {
			for _, idx := range idxs {
				m.newFetcher(idx)
			}
		}
	}
}

// match creates a daisy-chain of sub-matchers, one for the address set and one for each topic set, each
// sub-matcher receiving a section only if the previous ones have all found a potential match in one of
// the blocks of the section, then binary AND-ing its own matches and forwaring the result to the next one
func (m *Matcher) match(processCh chan partialMatches, stop chan struct{}) chan partialMatches {
	subIdx := m.topics
	if len(m.addresses) > 0 {
		subIdx = append([][]types.BloomIndexList{m.addresses}, subIdx...)
	}
	m.getNextReqCh = make(chan chan nextRequests) // should be a blocking channel
	m.distributeRequests(stop)

	for _, idx := range subIdx {
		processCh = m.subMatch(processCh, idx, stop)
	}
	return processCh
}

// partialMatches with a non-nil vector represents a section in which some sub-matchers have already
// found potential matches. Subsequent sub-matchers will binary AND their matches with this vector.
// If vector is nil, it represents a section to be processed by the first sub-matcher.
type partialMatches struct {
	sectionIdx uint64
	vector     []byte
}

// newFetcher adds a fetcher for the given bit index if it has not existed before
func (m *Matcher) newFetcher(idx uint) {
	if _, ok := m.fetchers[idx]; ok {
		return
	}
	f := &fetcher{
		bitIdx: idx,
		reqMap: make(map[uint64]req),
	}
	m.fetchers[idx] = f
}

// subMatch creates a sub-matcher that filters for a set of addresses or topics, binary OR-s those matches, then
// binary AND-s the result to the daisy-chain input (processCh) and forwards it to the daisy-chain output.
// The matches of each address/topic are calculated by fetching the given sections of the three bloom bit indexes belonging to
// that address/topic, and binary AND-ing those vectors together.
func (m *Matcher) subMatch(processCh chan partialMatches, idxs []types.BloomIndexList, stop chan struct{}) chan partialMatches {
	// set up fetchers
	fetchIdx := make([][3]chan uint64, len(idxs))
	fetchData := make([][3]chan []byte, len(idxs))
	for i, idx := range idxs {
		for j, ii := range idx {
			fetchIdx[i][j] = make(chan uint64, channelCap)
			fetchData[i][j] = m.fetchers[ii].fetch(fetchIdx[i][j], m.distCh, stop, &m.wg)
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
			case <-stop:
				return
			case s, ok := <-processCh:
				if !ok {
					close(fetchedCh)
					for _, ff := range fetchIdx {
						for _, f := range ff {
							close(f)
						}
					}
					return
				}

				for _, ff := range fetchIdx {
					for _, f := range ff {
						select {
						case <-stop:
							return
						case f <- s.sectionIdx:
						}
					}
				}
				select {
				case <-stop:
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
			case <-stop:
				return
			case s, ok := <-fetchedCh:
				if !ok {
					close(resultsCh)
					return
				}

				var orVector []byte
				for _, ff := range fetchData {
					var andVector []byte
					for _, f := range ff {
						var data []byte
						select {
						case <-stop:
							return
						case data = <-f:
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
					case <-stop:
						return
					case resultsCh <- partialMatches{s.sectionIdx, orVector}:
					}
				}
			}
		}
	}()

	return resultsCh
}

// GetMatches returns a stream of bloom matches in a given range of blocks.
// It returns a results channel immediately and stops if the stop channel is closed or
// there are no more matches in the range (in which case the results channel is closed).
// GetMatches can be called multiple times for different ranges, in which case already
// delivered bit vectors are not requested again.
func (m *Matcher) GetMatches(start, end uint64, stop chan struct{}) chan uint64 {
	m.distWg.Wait()

	processCh := make(chan partialMatches, channelCap)
	resultsCh := make(chan uint64, channelCap)

	res := m.match(processCh, stop)

	startSection := start / m.sectionSize
	endSection := end / m.sectionSize

	m.wg.Add(2)
	go func() {
		defer m.wg.Done()
		defer close(processCh)

		for i := startSection; i <= endSection; i++ {
			select {
			case processCh <- partialMatches{i, nil}:
			case <-stop:
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
				sectionStart := r.sectionIdx * m.sectionSize
				s := sectionStart
				if start > s {
					s = start
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
							case <-stop:
								return
							case resultsCh <- i:
							}
						}
					} else {
						i += bit
					}
				}

			case <-stop:
				return
			}
		}
	}()

	return resultsCh
}

type nextRequests struct {
	bitIdx         uint
	sectionIdxList []uint64
}

// distributeRequests receives requests from the fetchers and either queues them
// or immediately forwards them to one of the waiting NextRequest functions.
// Requests with a lower section idx are always prioritized.
func (m *Matcher) distributeRequests(stop chan struct{}) {
	m.distWg.Add(1)
	stopDist := make(chan struct{})
	go func() {
		<-stop
		m.wg.Wait()
		close(stopDist)
	}()

	go func() {
		defer m.distWg.Done()

		reqCount := 0
		for _, s := range m.reqs {
			reqCount += len(s)
		}

		storeReq := func(r distReq) {
			queue := m.reqs[r.bitIdx]
			i := 0
			for i < len(queue) && r.sectionIdx > queue[i] {
				i++
			}
			queue = append(queue, 0)
			copy(queue[i+1:], queue[i:len(queue)-1])
			queue[i] = r.sectionIdx

			m.reqs[r.bitIdx] = queue
			reqCount++
		}

		storeReqs := func(r distReq) {
			storeReq(r)
			timeout := time.After(time.Microsecond)
			for {
				select {
				case <-timeout:
					return
				case r := <-m.distCh:
					storeReq(r)
				case <-stopDist:
					return
				}
			}
		}

		for {
			if reqCount == 0 {
				select {
				case r := <-m.distCh:
					storeReqs(r)
				case <-stopDist:
					return
				}
			} else {
				select {
				case r := <-m.distCh:
					storeReqs(r)
				case <-stopDist:
					return
				case c := <-m.getNextReqCh:
					var (
						found       bool
						bestBit     uint
						bestSection uint64
					)

					for bitIdx, queue := range m.reqs {
						if len(queue) > 0 && (!found || queue[0] < bestSection) {
							found = true
							bestBit = bitIdx
							bestSection = queue[0]
						}
					}
					if !found {
						panic(nil)
					}

					bestQueue := m.reqs[bestBit]
					cnt := len(bestQueue)
					if cnt > maxRequestLength {
						cnt = maxRequestLength
					}
					res := nextRequests{bestBit, bestQueue[:cnt]}
					m.reqs[bestBit] = bestQueue[cnt:]
					reqCount -= cnt

					c <- res
				}
			}
		}
	}()
}

// NextRequest asks for a request from the request distributor, returns immediately if there
// was a queued one, otherwise waits until the distributor forwards one of the requests sent
// by one of the fetchers.
func (m *Matcher) NextRequest(stop chan struct{}) (bitIdx uint, sectionIdxList []uint64) {
	c := make(chan nextRequests)
	select {
	case m.getNextReqCh <- c:
		r := <-c
		return r.bitIdx, r.sectionIdxList
	case <-stop:
		return 0, nil
	}
}

// Deliver delivers a bit vector to the appropriate fetcher.
// It is possible to deliver data even after GetMatches has been stopped. Once a vector has been
// requested, the next call to GetMatches will keep waiting for delivery.
func (m *Matcher) Deliver(bitIdx uint, sectionIdxList []uint64, data [][]byte) {
	m.fetchers[bitIdx].deliver(sectionIdxList, data)
}
