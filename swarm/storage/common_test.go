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
	"flag"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	colorable "github.com/mattn/go-colorable"
)

var (
	loglevel = flag.Int("loglevel", 3, "verbosity of logs")
)

func init() {
	flag.Parse()
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(*loglevel), log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))
}

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

func mputRandomChunks(store ChunkStore, processors int, n int, chunksize int64) (hs []Address) {
	return mput(store, processors, n, GenerateRandomChunk)
}

func mput(store ChunkStore, processors int, n int, f func(i int64) *Chunk) (hs []Address) {
	wg := sync.WaitGroup{}
	wg.Add(processors)
	c := make(chan *Chunk)
	for i := 0; i < processors; i++ {
		go func() {
			defer wg.Done()
			for chunk := range c {
				wg.Add(1)
				chunk := chunk
				store.Put(chunk)
				go func() {
					defer wg.Done()
					<-chunk.dbStoredC
				}()
			}
		}()
	}
	fa := f
	if _, ok := store.(*MemStore); ok {
		fa = func(i int64) *Chunk {
			chunk := f(i)
			chunk.markAsStored()
			return chunk
		}
	}
	for i := 0; i < n; i++ {
		chunk := fa(int64(i))
		hs = append(hs, chunk.Addr)
		c <- chunk
	}
	close(c)
	wg.Wait()
	return hs
}

func mget(store ChunkStore, hs []Address, f func(h Address, chunk *Chunk) error) error {
	wg := sync.WaitGroup{}
	wg.Add(len(hs))
	errc := make(chan error)

	for _, k := range hs {
		go func(h Address) {
			defer wg.Done()
			chunk, err := store.Get(h)
			if err != nil {
				errc <- err
				return
			}
			if f != nil {
				err = f(h, chunk)
				if err != nil {
					errc <- err
					return
				}
			}
		}(k)
	}
	go func() {
		wg.Wait()
		close(errc)
	}()
	var err error
	select {
	case err = <-errc:
	case <-time.NewTimer(5 * time.Second).C:
		err = fmt.Errorf("timed out after 5 seconds")
	}
	return err
}

func testDataReader(l int) (r io.Reader) {
	return io.LimitReader(rand.Reader, int64(l))
}

func (r *brokenLimitedReader) Read(buf []byte) (int, error) {
	if r.off+len(buf) > r.errAt {
		return 0, fmt.Errorf("Broken reader")
	}
	r.off += len(buf)
	return r.lr.Read(buf)
}

func generateRandomData(l int) (r io.Reader, slice []byte) {
	slice = make([]byte, l)
	if _, err := rand.Read(slice); err != nil {
		panic("rand error")
	}
	r = io.LimitReader(bytes.NewReader(slice), int64(l))
	return
}

func testStoreRandom(m ChunkStore, processors int, n int, chunksize int64, t *testing.T) {
	hs := mputRandomChunks(m, processors, n, chunksize)
	err := mget(m, hs, nil)
	if err != nil {
		t.Fatalf("testStore failed: %v", err)
	}
}

func testStoreCorrect(m ChunkStore, processors int, n int, chunksize int64, t *testing.T) {
	hs := mputRandomChunks(m, processors, n, chunksize)
	f := func(h Address, chunk *Chunk) error {
		if !bytes.Equal(h, chunk.Addr) {
			return fmt.Errorf("key does not match retrieved chunk Key")
		}
		hasher := MakeHashFunc(DefaultHash)()
		hasher.ResetWithLength(chunk.SData[:8])
		hasher.Write(chunk.SData[8:])
		exp := hasher.Sum(nil)
		if !bytes.Equal(h, exp) {
			return fmt.Errorf("key is not hash of chunk data")
		}
		return nil
	}
	err := mget(m, hs, f)
	if err != nil {
		t.Fatalf("testStore failed: %v", err)
	}
}

func benchmarkStorePut(store ChunkStore, processors int, n int, chunksize int64, b *testing.B) {
	chunks := make([]*Chunk, n)
	i := 0
	f := func(dataSize int64) *Chunk {
		chunk := GenerateRandomChunk(dataSize)
		chunks[i] = chunk
		i++
		return chunk
	}

	mput(store, processors, n, f)

	f = func(dataSize int64) *Chunk {
		chunk := chunks[i]
		i++
		return chunk
	}

	b.ReportAllocs()
	b.ResetTimer()

	for j := 0; j < b.N; j++ {
		i = 0
		mput(store, processors, n, f)
	}
}

func benchmarkStoreGet(store ChunkStore, processors int, n int, chunksize int64, b *testing.B) {
	hs := mputRandomChunks(store, processors, n, chunksize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := mget(store, hs, nil)
		if err != nil {
			b.Fatalf("mget failed: %v", err)
		}
	}
}
