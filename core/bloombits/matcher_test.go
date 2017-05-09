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
	"math/rand"
	"sync/atomic"
	"testing"

	"github.com/ethereum/go-ethereum/core/types"
)

const testSectionSize = 4096

func matcherTestVector(b uint, s uint64) []byte {
	r := make([]byte, testSectionSize/8)
	for i, _ := range r {
		var bb byte
		for bit := 0; bit < 8; bit++ {
			blockIdx := s*testSectionSize + uint64(i*8+bit)
			bb += bb
			if (blockIdx % uint64(b)) == 0 {
				bb++
			}
		}
		r[i] = bb
	}
	return r
}

func expMatch1(idxs types.BloomIndexList, i uint64) bool {
	for _, ii := range idxs {
		if (i % uint64(ii)) != 0 {
			return false
		}
	}
	return true
}

func expMatch2(idxs []types.BloomIndexList, i uint64) bool {
	for _, ii := range idxs {
		if expMatch1(ii, i) {
			return true
		}
	}
	return false
}

func expMatch3(idxs [][]types.BloomIndexList, i uint64) bool {
	for _, ii := range idxs {
		if !expMatch2(ii, i) {
			return false
		}
	}
	return true
}

func testServeMatcher(m *Matcher, stop chan struct{}, cnt *uint32) {
	// serve matcher with test vectors
	for i := 0; i < 10; i++ {
		go func() {
			for {
				select {
				case <-stop:
					return
				default:
				}
				b, s := m.NextRequest(stop)
				if s == nil {
					return
				}
				res := make([][]byte, len(s))
				for i, ss := range s {
					res[i] = matcherTestVector(b, ss)
					atomic.AddUint32(cnt, 1)
				}
				m.Deliver(b, s, res)
			}
		}()
	}
}

func testMatcher(t *testing.T, idxs [][]types.BloomIndexList, cnt uint64, stopOnMatches bool, expCount uint32) uint32 {
	m := NewMatcher(testSectionSize, nil, nil)

	for _, idxss := range idxs {
		for _, idxs := range idxss {
			for _, idx := range idxs {
				m.newFetcher(idx)
			}
		}
	}

	m.addresses = idxs[0]
	m.topics = idxs[1:]
	var reqCount uint32

	stop := make(chan struct{})
	chn := m.GetMatches(0, cnt-1, stop)
	testServeMatcher(m, stop, &reqCount)

	for i := uint64(0); i < cnt; i++ {
		if expMatch3(idxs, i) {
			match, ok := <-chn
			if !ok {
				t.Errorf("Error matching idxs = %v  count = %v  stopOnMatches = %v: expected #%v, results channel closed", idxs, cnt, stopOnMatches, i)
				return 0
			}
			if match != i {
				t.Errorf("Error matching idxs = %v  count = %v  stopOnMatches = %v: expected #%v, got #%v", idxs, cnt, stopOnMatches, i, match)
			}
			if stopOnMatches {
				close(stop)
				stop = make(chan struct{})
				chn = m.GetMatches(i+1, cnt-1, stop)
				testServeMatcher(m, stop, &reqCount)
			}
		}
	}
	match, ok := <-chn
	if ok {
		t.Errorf("Error matching idxs = %v  count = %v  stopOnMatches = %v: expected closed channel, got #%v", idxs, cnt, stopOnMatches, match)
	}
	close(stop)

	if expCount != 0 && expCount != reqCount {
		t.Errorf("Error matching idxs = %v  count = %v  stopOnMatches = %v: request count mismatch, expected #%v, got #%v", idxs, cnt, stopOnMatches, expCount, reqCount)
	}

	return reqCount
}

func testRandomIdxs(l []int, max int) [][]types.BloomIndexList {
	res := make([][]types.BloomIndexList, len(l))
	for i, ll := range l {
		res[i] = make([]types.BloomIndexList, ll)
		for j, _ := range res[i] {
			for k, _ := range res[i][j] {
				res[i][j][k] = uint(rand.Intn(max-1) + 2)
			}
		}
	}
	return res
}

func TestMatcher(t *testing.T) {
	testMatcher(t, [][]types.BloomIndexList{{{10, 20, 30}}}, 1000000, false, 735)
	testMatcher(t, [][]types.BloomIndexList{{{32, 3125, 100}}, {{40, 50, 10}}}, 1000000, false, 768)
	testMatcher(t, [][]types.BloomIndexList{{{4, 8, 11}, {7, 8, 17}}, {{9, 9, 12}, {15, 20, 13}}, {{18, 15, 15}, {12, 10, 4}}}, 100000, false, 300)
}

func TestMatcherStopOnMatches(t *testing.T) {
	testMatcher(t, [][]types.BloomIndexList{{{10, 20, 30}}}, 1000000, true, 735)
	testMatcher(t, [][]types.BloomIndexList{{{4, 8, 11}, {7, 8, 17}}, {{9, 9, 12}, {15, 20, 13}}, {{18, 15, 15}, {12, 10, 4}}}, 100000, true, 300)
}

func TestMatcherRandom(t *testing.T) {
	for i := 0; i < 20; i++ {
		testMatcher(t, testRandomIdxs([]int{1}, 50), 1000000, false, 0)
		testMatcher(t, testRandomIdxs([]int{3}, 50), 1000000, false, 0)
		testMatcher(t, testRandomIdxs([]int{2, 2, 2}, 20), 1000000, false, 0)
		testMatcher(t, testRandomIdxs([]int{5, 5, 5}, 50), 1000000, false, 0)
		idxs := testRandomIdxs([]int{2, 2, 2}, 20)
		reqCount := testMatcher(t, idxs, 100000, false, 0)
		testMatcher(t, idxs, 100000, true, reqCount)
	}
}
