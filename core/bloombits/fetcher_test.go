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
	"bytes"
	"encoding/binary"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

const testFetcherReqCount = 5000

func fetcherTestVector(b uint, s uint64) []byte {
	r := make([]byte, 10)
	binary.BigEndian.PutUint16(r[0:2], uint16(b))
	binary.BigEndian.PutUint64(r[2:10], s)
	return r
}

func TestFetcher(t *testing.T) {
	testFetcher(t, 1)
}

func TestFetcherMultipleReaders(t *testing.T) {
	testFetcher(t, 10)
}

func testFetcher(t *testing.T, cnt int) {
	f := &fetcher{
		requestMap: make(map[uint64]fetchRequest),
	}
	distCh := make(chan distRequest, channelCap)
	stop := make(chan struct{})
	var reqCount uint32

	for i := 0; i < 10; i++ {
		go func() {
			for {
				req, ok := <-distCh
				if !ok {
					return
				}
				time.Sleep(time.Duration(rand.Intn(100000)))
				atomic.AddUint32(&reqCount, 1)
				f.deliver([]uint64{req.sectionIndex}, [][]byte{fetcherTestVector(req.bloomIndex, req.sectionIndex)})
			}
		}()
	}

	var wg, wg2 sync.WaitGroup
	for cc := 0; cc < cnt; cc++ {
		wg.Add(1)
		in := make(chan uint64, channelCap)
		out := f.fetch(in, distCh, stop, &wg2)

		time.Sleep(time.Millisecond * 10 * time.Duration(cc))
		go func() {
			for i := uint64(0); i < testFetcherReqCount; i++ {
				in <- i
			}
		}()

		go func() {
			for i := uint64(0); i < testFetcherReqCount; i++ {
				bv := <-out
				if !bytes.Equal(bv, fetcherTestVector(0, i)) {
					if len(bv) != 10 {
						t.Errorf("Vector #%d length is %d, expected 10", i, len(bv))
					} else {
						j := binary.BigEndian.Uint64(bv[2:10])
						t.Errorf("Expected vector #%d, fetched #%d", i, j)
					}
				}
			}
			wg.Done()
		}()
	}

	wg.Wait()
	close(stop)
	if reqCount != testFetcherReqCount {
		t.Errorf("Request count mismatch: expected %v, got %v", testFetcherReqCount, reqCount)
	}
}
