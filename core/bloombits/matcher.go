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
	"github.com/ethereum/go-ethereum/core/types"
)

const (
	maxRequestLength = 16
	channelCap       = 100
)

// when received by fetcher:		data == nil, requested == false, fetched == chan struct{}
// when returned by NextRequest:	data == nil, requested == true, fetched == chan struct{}
// when data is delivered:			data == BitVector, requested == true, fetched == nil
type req struct {
	data      BitVector
	requested bool
	fetched   chan struct{}
}

type distReq struct {
	bitIdx     uint
	sectionIdx uint64
}

type fetcher struct {
	bitIdx  uint
	reqMap  map[uint64]req
	reqLock sync.RWMutex
}

func (f *fetcher) fetch(sectionChn chan uint64, distChn chan distReq, stop chan struct{}) chan BitVector {
	dataChn := make(chan BitVector, channelCap)
	returnChn := make(chan uint64, channelCap)

	go func() {
		defer close(returnChn)

		for {
			select {
			case <-stop:
				return
			case idx, ok := <-sectionChn:
				if !ok {
					return
				}

				f.reqLock.Lock()
				r := f.reqMap[idx]
				if r.data == nil {
					if r.fetched == nil {
						r.fetched = make(chan struct{})
					}
					if !r.requested {
						distChn <- distReq{bitIdx: f.bitIdx, sectionIdx: idx}
					}
					f.reqMap[idx] = r
				}
				f.reqLock.Unlock()
				returnChn <- idx
			}
		}
	}()

	go func() {
		defer close(dataChn)

		for {
			select {
			case <-stop:
				return
			case idx, ok := <-returnChn:
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
				dataChn <- r.data
			}
		}
	}()

	return dataChn
}

func (f *fetcher) requested(sectionIdxList []uint64) {
	//fmt.Println("requested", f.bitIdx, sectionIdxList)
	f.reqLock.Lock()
	defer f.reqLock.Unlock()

	for _, idx := range sectionIdxList {
		r := f.reqMap[idx]
		r.requested = true
		f.reqMap[idx] = r
	}
}

func (f *fetcher) deliver(sectionIdxList []uint64, data []BitVector) {
	//fmt.Println("deliver", f.bitIdx, sectionIdxList, data != nil)
	f.reqLock.Lock()
	defer f.reqLock.Unlock()

	for i, idx := range sectionIdxList {
		r := f.reqMap[idx]
		if data != nil {
			r.data = data[i]
			close(r.fetched)
			r.fetched = nil
		} else {
			r.requested = false
		}
		f.reqMap[idx] = r
	}
}

type Matcher struct {
	addresses []types.BloomIndexList
	topics    [][]types.BloomIndexList
	fetchers  map[uint]*fetcher

	distChn       chan distReq
	getNextReqChn chan chan nextRequests
}

func NewMatcher() *Matcher {
	return &Matcher{fetchers: make(map[uint]*fetcher)}
}

// SetAddresses matches only logs that are generated from addresses that are included
// in the given addresses.
func (m *Matcher) SetAddresses(addr []common.Address) {
	m.addresses = make([]types.BloomIndexList, len(addr))
	for i, b := range addr {
		m.addresses[i] = types.BloomIndexes(b.Bytes())
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
}

func (m *Matcher) match(sectionChn chan uint64, stop chan struct{}) (chan uint64, chan BitVector) {
	subIdx := m.topics
	if len(m.addresses) > 0 {
		subIdx = append([][]types.BloomIndexList{m.addresses}, subIdx...)
	}
	//fmt.Println("idx", subIdx)
	m.distChn = make(chan distReq, channelCap)
	m.getNextReqChn = make(chan chan nextRequests) // should be a blocking channel
	go m.distributeRequests(stop)

	s := sectionChn
	var bv chan BitVector
	for _, idx := range subIdx {
		s, bv = m.subMatch(s, bv, idx, stop)
	}
	return s, bv
}

func (m *Matcher) getOrNewFetcher(idx uint) *fetcher {
	if f, ok := m.fetchers[idx]; ok {
		return f
	}
	f := &fetcher{
		bitIdx: idx,
		reqMap: make(map[uint64]req),
	}
	m.fetchers[idx] = f
	return f
}

// andVector == nil
func (m *Matcher) subMatch(sectionChn chan uint64, andVectorChn chan BitVector, idxs []types.BloomIndexList, stop chan struct{}) (chan uint64, chan BitVector) {
	// set up fetchers
	fetchIdx := make([][3]chan uint64, len(idxs))
	fetchData := make([][3]chan BitVector, len(idxs))
	for i, idx := range idxs {
		for j, ii := range idx {
			fetchIdx[i][j] = make(chan uint64, channelCap)
			fetchData[i][j] = m.getOrNewFetcher(ii).fetch(fetchIdx[i][j], m.distChn, stop)
		}
	}

	processChn := make(chan uint64, channelCap)
	resIdxChn := make(chan uint64, channelCap)
	resDataChn := make(chan BitVector, channelCap)

	// goroutine for starting retrievals
	go func() {
		for {
			select {
			case <-stop:
				return
			case s, ok := <-sectionChn:
				if !ok {
					close(processChn)
					for _, ff := range fetchIdx {
						for _, f := range ff {
							close(f)
						}
					}
					return
				}

				processChn <- s
				for _, ff := range fetchIdx {
					for _, f := range ff {
						f <- s
					}
				}
			}
		}
	}()

	// goroutine for processing retrieved data
	go func() {
		for {
			select {
			case <-stop:
				return
			case s, ok := <-processChn:
				if !ok {
					close(resIdxChn)
					close(resDataChn)
					return
				}

				var orVector BitVector
				for _, ff := range fetchData {
					var andVector BitVector
					for _, f := range ff {
						data := <-f
						if andVector == nil {
							andVector = bvCopy(data)
						} else {
							bvAnd(andVector, data)
						}
					}
					if orVector == nil {
						orVector = andVector
					} else {
						bvOr(orVector, andVector)
					}
				}

				if orVector == nil {
					orVector = bvZero()
				}
				if andVectorChn != nil {
					bvAnd(orVector, <-andVectorChn)
				}
				if bvIsNonZero(orVector) {
					resIdxChn <- s
					resDataChn <- orVector
				}
			}
		}
	}()

	return resIdxChn, resDataChn
}

func (m *Matcher) GetMatches(start, end uint64, stop chan struct{}) chan uint64 {
	sectionChn := make(chan uint64, channelCap)
	resultsChn := make(chan uint64, channelCap)

	s, bv := m.match(sectionChn, stop)

	startSection := start / SectionSize
	endSection := end / SectionSize

	go func() {
		defer close(sectionChn)

		for i := startSection; i <= endSection; i++ {
			select {
			case sectionChn <- i:
			case <-stop:
				return
			}
		}
	}()

	go func() {
		defer close(resultsChn)

		for {
			select {
			case idx, ok := <-s:
				if !ok {
					return
				}
				match := <-bv //nil check
				sectionStart := idx * SectionSize
				s := sectionStart
				if start > s {
					s = start
				}
				e := sectionStart + SectionSize - 1
				if end < e {
					e = end
				}
				for i := s; i <= e; i++ {
					b := match[(i-sectionStart)/8]
					bit := 7 - i%8
					if b != 0 {
						if b&(1<<bit) != 0 {
							resultsChn <- i
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

	return resultsChn
}

type nextRequests struct {
	bitIdx         uint
	sectionIdxList []uint64
}

func (m *Matcher) distributeRequests(stop chan struct{}) {
	reqCnt := 0
	reqs := make(map[uint][]uint64)
	storeReq := func(r distReq) {
		queue := reqs[r.bitIdx]
		i := 0
		for i < len(queue) && r.sectionIdx > queue[i] {
			i++
		}
		reqs[r.bitIdx] = append(append(queue[:i], r.sectionIdx), queue[i:]...)
		reqCnt++
	}

	storeReqs := func(r distReq) {
		storeReq(r)
		timeout := time.After(time.Microsecond)
		for {
			select {
			case <-timeout:
				return
			case r := <-m.distChn:
				storeReq(r)
			}
		}
	}

	for {
		if reqCnt == 0 {
			select {
			case r := <-m.distChn:
				storeReqs(r)
			case <-stop:
				return
			}
		} else {
			select {
			case r := <-m.distChn:
				storeReqs(r)
			case <-stop:
				return
			case c := <-m.getNextReqChn:
				var (
					found       bool
					bestBit     uint
					bestSection uint64
				)

				for bitIdx, queue := range reqs {
					if len(queue) > 0 && (!found || queue[0] < bestSection) {
						found = true
						bestBit = bitIdx
						bestSection = queue[0]
					}
				}
				if !found {
					panic(nil)
				}

				bestQueue := reqs[bestBit]
				cnt := len(bestQueue)
				if cnt > maxRequestLength {
					cnt = maxRequestLength
				}
				res := nextRequests{bestBit, bestQueue[:cnt]}
				reqs[bestBit] = bestQueue[cnt:]
				reqCnt -= cnt

				c <- res
			}
		}
	}
}

func (m *Matcher) NextRequest(stop chan struct{}) (bitIdx uint, sectionIdxList []uint64) {
	c := make(chan nextRequests)
	select {
	case m.getNextReqChn <- c:
		r := <-c
		//fmt.Println("request", r.bitIdx, r.sectionIdxList)
		m.fetchers[r.bitIdx].requested(r.sectionIdxList)
		return r.bitIdx, r.sectionIdxList
	case <-stop:
		return 0, nil
	}
}

// It is possible to deliver data even after GetMatches has been stopped. Once a vector has been
// requested, the next call to GetMatches will keep waiting for delivery.
// If retrieval has been cancelled, call Deliver with data == nil. In this case the next call to
// GetMatches will re-request it.
func (m *Matcher) Deliver(bitIdx uint, sectionIdxList []uint64, data []BitVector) {
	m.fetchers[bitIdx].deliver(sectionIdxList, data)
}
