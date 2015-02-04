package bzz

import (
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/bzz/test"
)

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

func testMemStore(l int64, branches int64, t *testing.T) {
	m := newMemStore()
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
						t.Errorf("Chunk not found: %v", err)
						return
					}
					if err != nil {
						t.Errorf("GET error: %v", err)
						return
					}
					chunk.Reader = NewChunkReaderFromBytes(storedChunk.Data)
					chunk.Size = storedChunk.Size
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

func TestMemStore128_10000(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testMemStore(10000, 128, t)
}

func TestMemStore128_1000(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testMemStore(1000, 128, t)
}

func TestMemStore128_100(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testMemStore(100, 128, t)
}

func TestMemStore2_100(t *testing.T) {
	// defer test.Testlog(t).Detach()
	test.LogInit()
	testMemStore(100, 2, t)
}
