package bzz

import (
	"crypto/rand"
	"testing"
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
	chunker := &TreeChunker{
		Branches: branches,
	}
	chunker.Init()
	key = make([]byte, 32)
	b := make([]byte, l)
	_, err := rand.Read(b)
	if err != nil {
		panic("no rand")
	}
	errC = chunker.Split(key, NewChunkReaderFromBytes(b), chunkC)
	return
}

func testStore(m ChunkStore, l int64, branches int64, t *testing.T) {

	chunkC := make(chan *Chunk)
	key, errC := randomChunks(l, branches, chunkC)

SPLIT:
	for {
		select {
		case chunk := <-chunkC:
			chunk.Data = make([]byte, chunk.Reader.Size())
			chunk.Reader.ReadAt(chunk.Data, 0)
			m.Put(chunk)

		case err, ok := <-errC:
			if err != nil {
				t.Errorf("Chunker error: %v", err)
				return
			}
			if !ok {
				t.Logf("quitting SPLIT loop\n")
				break SPLIT
			}
		}
	}

	chunker := &TreeChunker{
		Branches: branches,
	}
	chunker.Init()
	chunkC = make(chan *Chunk)
	var r LazySectionReader
	r, errC = chunker.Join(key, chunkC)

	quit := make(chan bool)

	go func() {
	JOIN:
		for {
			select {
			case chunk := <-chunkC:
				go func() {
					storedChunk, err := m.Get(chunk.Key)
					if err == notFound {
						dpaLogger.DebugDetailf("chunk '%x' not found", chunk.Key)
					} else if err != nil {
						dpaLogger.DebugDetailf("error retrieving chunk %x: %v", chunk.Key, err)
					} else {
						chunk.Reader = NewChunkReaderFromBytes(storedChunk.Data)
						chunk.Size = storedChunk.Size
					}
					close(chunk.C)
				}()
			case err, ok := <-errC:
				if err != nil {
					t.Errorf("Chunker error: %v", err)
					return
				}
				if !ok {
					break JOIN
				}
			case <-quit:
				break JOIN
			}
		}
	}()

	b := make([]byte, l)
	n, err := r.ReadAt(b, 0)
	if err != nil {
		t.Errorf("read error (%v/%v) %v", n, l, err)
		close(quit)
	}
}
