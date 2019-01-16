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
	"context"
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/log"
	ch "github.com/ethereum/go-ethereum/swarm/chunk"
	"github.com/mattn/go-colorable"
)

var (
	loglevel   = flag.Int("loglevel", 3, "verbosity of logs")
	getTimeout = 30 * time.Second
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

func newLDBStore(t *testing.T) (*LDBStore, func()) {
	dir, err := ioutil.TempDir("", "bzz-storage-test")
	if err != nil {
		t.Fatal(err)
	}
	log.Trace("memstore.tempdir", "dir", dir)

	ldbparams := NewLDBStoreParams(NewDefaultStoreParams(), dir)
	db, err := NewLDBStore(ldbparams)
	if err != nil {
		t.Fatal(err)
	}

	cleanup := func() {
		db.Close()
		err := os.RemoveAll(dir)
		if err != nil {
			t.Fatal(err)
		}
	}

	return db, cleanup
}

func mputRandomChunks(store ChunkStore, n int, chunksize int64) ([]Chunk, error) {
	return mput(store, n, GenerateRandomChunk)
}

func mput(store ChunkStore, n int, f func(i int64) Chunk) (hs []Chunk, err error) {
	// put to localstore and wait for stored channel
	// does not check delivery error state
	errc := make(chan error)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for i := int64(0); i < int64(n); i++ {
		chunk := f(ch.DefaultSize)
		go func() {
			select {
			case errc <- store.Put(ctx, chunk):
			case <-ctx.Done():
			}
		}()
		hs = append(hs, chunk)
	}

	// wait for all chunks to be stored
	for i := 0; i < n; i++ {
		err := <-errc
		if err != nil {
			return nil, err
		}
	}
	return hs, nil
}

func mget(store ChunkStore, hs []Address, f func(h Address, chunk Chunk) error) error {
	wg := sync.WaitGroup{}
	wg.Add(len(hs))
	errc := make(chan error)

	for _, k := range hs {
		go func(h Address) {
			defer wg.Done()
			// TODO: write timeout with context
			chunk, err := store.Get(context.TODO(), h)
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

func (r *brokenLimitedReader) Read(buf []byte) (int, error) {
	if r.off+len(buf) > r.errAt {
		return 0, fmt.Errorf("Broken reader")
	}
	r.off += len(buf)
	return r.lr.Read(buf)
}

func testStoreRandom(m ChunkStore, n int, chunksize int64, t *testing.T) {
	chunks, err := mputRandomChunks(m, n, chunksize)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	err = mget(m, chunkAddresses(chunks), nil)
	if err != nil {
		t.Fatalf("testStore failed: %v", err)
	}
}

func testStoreCorrect(m ChunkStore, n int, chunksize int64, t *testing.T) {
	chunks, err := mputRandomChunks(m, n, chunksize)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	f := func(h Address, chunk Chunk) error {
		if !bytes.Equal(h, chunk.Address()) {
			return fmt.Errorf("key does not match retrieved chunk Address")
		}
		hasher := MakeHashFunc(DefaultHash)()
		data := chunk.Data()
		hasher.ResetWithLength(data[:8])
		hasher.Write(data[8:])
		exp := hasher.Sum(nil)
		if !bytes.Equal(h, exp) {
			return fmt.Errorf("key is not hash of chunk data")
		}
		return nil
	}
	err = mget(m, chunkAddresses(chunks), f)
	if err != nil {
		t.Fatalf("testStore failed: %v", err)
	}
}

func benchmarkStorePut(store ChunkStore, n int, chunksize int64, b *testing.B) {
	chunks := make([]Chunk, n)
	i := 0
	f := func(dataSize int64) Chunk {
		chunk := GenerateRandomChunk(dataSize)
		chunks[i] = chunk
		i++
		return chunk
	}

	mput(store, n, f)

	f = func(dataSize int64) Chunk {
		chunk := chunks[i]
		i++
		return chunk
	}

	b.ReportAllocs()
	b.ResetTimer()

	for j := 0; j < b.N; j++ {
		i = 0
		mput(store, n, f)
	}
}

func benchmarkStoreGet(store ChunkStore, n int, chunksize int64, b *testing.B) {
	chunks, err := mputRandomChunks(store, n, chunksize)
	if err != nil {
		b.Fatalf("expected no error, got %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()
	addrs := chunkAddresses(chunks)
	for i := 0; i < b.N; i++ {
		err := mget(store, addrs, nil)
		if err != nil {
			b.Fatalf("mget failed: %v", err)
		}
	}
}

// MapChunkStore is a very simple ChunkStore implementation to store chunks in a map in memory.
type MapChunkStore struct {
	chunks map[string]Chunk
	mu     sync.RWMutex
}

func NewMapChunkStore() *MapChunkStore {
	return &MapChunkStore{
		chunks: make(map[string]Chunk),
	}
}

func (m *MapChunkStore) Put(_ context.Context, ch Chunk) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.chunks[ch.Address().Hex()] = ch
	return nil
}

func (m *MapChunkStore) Get(_ context.Context, ref Address) (Chunk, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	chunk := m.chunks[ref.Hex()]
	if chunk == nil {
		return nil, ErrChunkNotFound
	}
	return chunk, nil
}

func (m *MapChunkStore) Close() {
}

func chunkAddresses(chunks []Chunk) []Address {
	addrs := make([]Address, len(chunks))
	for i, ch := range chunks {
		addrs[i] = ch.Address()
	}
	return addrs
}
