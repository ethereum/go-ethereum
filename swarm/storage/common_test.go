// Copyright 2016 The go-ethereum Authors
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

package storage

import (
	"bytes"
	"crypto/rand"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/log"
)

type brokenLimitedReader struct {
	lr    io.Reader
	errAt int
	off   int
	size  int
}

func brokenLimitReader(data io.Reader, size int, errAt int) *brokenLimitedReader {
	return &brokenLimitedReader{
		lr:    data,
		errAt: errAt,
		size:  size,
	}
}

func testDataReader(l int) (r io.Reader) {
	return io.LimitReader(rand.Reader, int64(l))
}

func (self *brokenLimitedReader) Read(buf []byte) (int, error) {
	if self.off+len(buf) > self.errAt {
		return 0, fmt.Errorf("Broken reader")
	}
	self.off += len(buf)
	return self.lr.Read(buf)
}

func testDataReaderAndSlice(l int) (r io.Reader, slice []byte) {
	slice = make([]byte, l)
	if _, err := rand.Read(slice); err != nil {
		panic("rand error")
	}
	r = io.LimitReader(bytes.NewReader(slice), int64(l))
	return
}

func testStore(m ChunkStore, l int64, branches int64, t *testing.T) {

	chunkC := make(chan *Chunk)
	go func() {
		for chunk := range chunkC {
			m.Put(chunk)
			if chunk.wg != nil {
				chunk.wg.Done()
			}
		}
	}()
	chunker := NewTreeChunker(&ChunkerParams{
		Branches: branches,
		Hash:     SHA3Hash,
	})
	swg := &sync.WaitGroup{}
	key, _ := chunker.Split(rand.Reader, l, chunkC, swg, nil)
	swg.Wait()
	close(chunkC)
	chunkC = make(chan *Chunk)

	quit := make(chan bool)

	go func() {
		for ch := range chunkC {
			go func(chunk *Chunk) {
				storedChunk, err := m.Get(chunk.Key)
				if err == notFound {
					log.Trace(fmt.Sprintf("chunk '%v' not found", chunk.Key.Log()))
				} else if err != nil {
					log.Trace(fmt.Sprintf("error retrieving chunk %v: %v", chunk.Key.Log(), err))
				} else {
					chunk.SData = storedChunk.SData
					chunk.Size = storedChunk.Size
				}
				log.Trace(fmt.Sprintf("chunk '%v' not found", chunk.Key.Log()))
				close(chunk.C)
			}(ch)
		}
		close(quit)
	}()
	r := chunker.Join(key, chunkC)

	b := make([]byte, l)
	n, err := r.ReadAt(b, 0)
	if err != io.EOF {
		t.Fatalf("read error (%v/%v) %v", n, l, err)
	}
	close(chunkC)
	<-quit
}
