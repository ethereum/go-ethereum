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
	"io"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

type limitedReader struct {
	r    io.Reader
	off  int64
	size int64
}

func limitReader(r io.Reader, size int) *limitedReader {
	return &limitedReader{r, 0, int64(size)}
}

func (self *limitedReader) Read(buf []byte) (int, error) {
	limit := int64(len(buf))
	left := self.size - self.off
	if limit >= left {
		limit = left
	}
	n, err := self.r.Read(buf[:limit])
	if err == nil && limit == left {
		err = io.EOF
	}
	self.off += int64(n)
	return n, err
}

func testDataReader(l int) (r io.Reader) {
	return limitReader(rand.Reader, l)
}

func testDataReaderAndSlice(l int) (r io.Reader, slice []byte) {
	slice = make([]byte, l)
	if _, err := rand.Read(slice); err != nil {
		panic("rand error")
	}
	r = limitReader(bytes.NewReader(slice), l)
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
		Hash:     defaultHash,
	})
	swg := &sync.WaitGroup{}
	key, err := chunker.Split(rand.Reader, l, chunkC, swg, nil)
	swg.Wait()
	close(chunkC)
	chunkC = make(chan *Chunk)

	quit := make(chan bool)

	go func() {
		for ch := range chunkC {
			go func(chunk *Chunk) {
				storedChunk, err := m.Get(chunk.Key)
				if err == notFound {
					glog.V(logger.Detail).Infof("chunk '%v' not found", chunk.Key.Log())
				} else if err != nil {
					glog.V(logger.Detail).Infof("error retrieving chunk %v: %v", chunk.Key.Log(), err)
				} else {
					chunk.SData = storedChunk.SData
					chunk.Size = storedChunk.Size
				}
				glog.V(logger.Detail).Infof("chunk '%v' not found", chunk.Key.Log())
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
