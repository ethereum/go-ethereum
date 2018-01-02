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
	"encoding/binary"
	"fmt"
	"hash"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/sha3"
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

func mputChunks(store ChunkStore, processors int, n int, chunksize int, hash hash.Hash) (hs []Key) {
	f := func(int) *Chunk {
		data := make([]byte, chunksize)
		rand.Reader.Read(data)
		hash.Reset()
		hash.Write(data)
		h := hash.Sum(nil)
		chunk := NewChunk(Key(h), nil)
		chunk.SData = data
		return chunk
	}
	return mput(store, processors, n, f)
}

func mputRandomKey(store ChunkStore, processors int, n int, chunksize int) (hs []Key) {
	data := make([]byte, chunksize+8)
	binary.LittleEndian.PutUint64(data[0:8], uint64(chunksize))

	f := func(int) *Chunk {
		h := make([]byte, 32)
		rand.Reader.Read(h)
		chunk := NewChunk(Key(h), nil)
		chunk.SData = data
		return chunk
	}
	return mput(store, processors, n, f)
}

func mput(store ChunkStore, processors int, n int, f func(i int) *Chunk) (hs []Key) {
	wg := sync.WaitGroup{}
	wg.Add(processors)
	c := make(chan *Chunk)
	for i := 0; i < processors; i++ {
		go func() {
			defer wg.Done()
			for chunk := range c {
				store.Put(chunk)
			}
		}()
	}
	for i := 0; i < n; i++ {
		chunk := f(i)
		hs = append(hs, chunk.Key)
		c <- chunk
	}
	close(c)
	wg.Wait()
	return hs
}

func mget(store ChunkStore, hs []Key, f func(h Key, chunk *Chunk) error) error {
	wg := sync.WaitGroup{}
	wg.Add(len(hs))
	errc := make(chan error)

	for _, k := range hs {
		go func(h Key) {
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

func testDataReaderAndSlice(l int) (r io.Reader, slice []byte) {
	slice = make([]byte, l)
	if _, err := rand.Read(slice); err != nil {
		panic("rand error")
	}
	r = io.LimitReader(bytes.NewReader(slice), int64(l))
	return
}

func testStoreRandom(m ChunkStore, processors int, n int, chunksize int, t *testing.T) {
	hs := mputRandomKey(m, processors, n, chunksize)
	err := mget(m, hs, nil)
	if err != nil {
		t.Fatalf("testStore failed: %v", err)
	}
}

func testStoreCorrect(m ChunkStore, processors int, n int, chunksize int, t *testing.T) {
	hs := mputChunks(m, processors, n, chunksize, sha3.NewKeccak256())
	f := func(h Key, chunk *Chunk) error {
		if !bytes.Equal(h, chunk.Key) {
			return fmt.Errorf("key does not match retrieved chunk Key")
		}
		hasher := sha3.NewKeccak256()
		hasher.Write(chunk.SData)
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

func benchmarkStorePut(store ChunkStore, processors int, n int, chunksize int, b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mputRandomKey(store, processors, n, chunksize)
	}
}

func benchmarkStoreGet(store ChunkStore, processors int, n int, chunksize int, b *testing.B) {
	hs := mputRandomKey(store, processors, n, chunksize)
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := mget(store, hs, nil)
		if err != nil {
			b.Fatalf("mget failed: %v", err)
		}
	}
}
