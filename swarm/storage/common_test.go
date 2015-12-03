package storage

import (
	"crypto/rand"
	"io"
	"sync"
	"testing"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

func testDataReader(l int) (r *ChunkReader, slice []byte) {
	slice = make([]byte, l)
	if _, err := rand.Read(slice); err != nil {
		panic("rand error")
	}
	r = NewChunkReaderFromBytes(slice)
	return
}

func randomChunks(l int64, branches int64, chunkC chan *Chunk) (key Key, errC chan error) {
	chunker := NewTreeChunker(&ChunkerParams{
		Branches:     branches,
		Hash:         defaultHash,
		SplitTimeout: splitTimeout,
	})
	key = make([]byte, 32)
	b := make([]byte, l)
	_, err := rand.Read(b)
	if err != nil {
		panic("no rand")
	}
	wg := &sync.WaitGroup{}
	errC = chunker.Split(key, NewChunkReaderFromBytes(b), chunkC, wg)
	wg.Wait()
	return
}

func testStore(m ChunkStore, l int64, branches int64, t *testing.T) {

	chunkC := make(chan *Chunk)
	key, errC := randomChunks(l, branches, chunkC)

SPLIT:
	for {
		select {
		case chunk := <-chunkC:
			m.Put(chunk)
		case err, ok := <-errC:
			if err != nil {
				t.Errorf("Chunker error: %v", err)
				return
			}
			if !ok {
				break SPLIT
			}
		}
	}
	chunker := NewTreeChunker(&ChunkerParams{
		Branches:     branches,
		Hash:         defaultHash,
		SplitTimeout: splitTimeout,
	})
	chunkC = make(chan *Chunk)
	var r SectionReader
	r = chunker.Join(key, chunkC)

	quit := make(chan bool)

	go func() {
		for ch := range chunkC {
			go func(chunk *Chunk) {
				storedChunk, err := m.Get(chunk.Key)
				if err == notFound {
					glog.V(logger.Detail).Infof("[BZZ] chunk '%x' not found", chunk.Key)
				} else if err != nil {
					glog.V(logger.Detail).Infof("[BZZ] error retrieving chunk %x: %v", chunk.Key, err)
				} else {
					chunk.SData = storedChunk.SData
				}
				glog.V(logger.Detail).Infof("[BZZ] chunk '%x' not found", chunk.Key[:4])
				close(chunk.C)
			}(ch)
		}
	}()

	b := make([]byte, l)
	n, err := r.ReadAt(b, 0)
	if err != io.EOF {
		t.Errorf("read error (%v/%v) %v", n, l, err)
		close(quit)
	}
}
