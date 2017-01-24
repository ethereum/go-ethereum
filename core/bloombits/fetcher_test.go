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
	"testing"
	"time"
)

func testVector(b uint, s uint64) BitVector {
	r := make(BitVector, 10)
	binary.BigEndian.PutUint16(r[0:2], uint16(b))
	binary.BigEndian.PutUint64(r[2:10], s)
	return r
}

func TestFetcher(t *testing.T) {
	f := &fetcher{
		reqMap:  make(map[uint64]req),
		distChn: make(chan distReq, channelCap),
	}
	in := make(chan uint64, channelCap)
	stop := make(chan struct{})
	out := f.fetch(in, stop)

	for i := 0; i < 10; i++ {
		go func() {
			for {
				req, ok := <-f.distChn
				if !ok {
					return
				}
				time.Sleep(time.Duration(rand.Intn(1000000)))
				f.deliver([]uint64{req.sectionIdx}, []BitVector{testVector(req.bitIdx, req.sectionIdx)})
			}
		}()
	}

	go func() {
		for i := uint64(0); i < 10000; i++ {
			in <- i
		}
	}()

	for i := uint64(0); i < 10000; i++ {
		bv := <-out
		if !bytes.Equal(bv, testVector(0, i)) {
			if len(bv) != 10 {
				t.Errorf("Vector #%d length is %d, expected 10", i, len(bv))
			} else {
				j := binary.BigEndian.Uint64(bv[2:10])
				t.Errorf("Expected vector #%d, fetched #%d", i, j)
			}
		}
	}

	close(stop)
}
