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

func testDataReader(l int) (r io.Reader) {
	return io.LimitReader(rand.Reader, int64(l))
}

func testDataReaderAndSlice(l int) (r io.Reader, slice []byte) {
	slice = make([]byte, l)
	if _, err := rand.Read(slice); err != nil {
		panic("rand error")
	}
	r = bytes.NewReader(slice)
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
	r := chunker.Join(key, chunkC)

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
		close(quit)
	}()

	b := make([]byte, l)
	n, err := r.ReadAt(b, 0)
	if err != io.EOF {
		t.Fatalf("read error (%v/%v) %v", n, l, err)
	}
	close(chunkC)
	<-quit
}
