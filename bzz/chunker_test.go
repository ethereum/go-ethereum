package bzz

import (
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/logger"
)

/*
Tests TreeChunker by splitting and joining a random byte slice
*/

type testLogger struct{ t *testing.T }

func testlog(t *testing.T) testLogger {
	logger.Reset()
	l := testLogger{t}
	logger.AddLogSystem(l)
	return l
}

func (testLogger) GetLogLevel() logger.LogLevel { return logger.DebugDetailLevel }
func (testLogger) SetLogLevel(logger.LogLevel)  {}

func (l testLogger) LogPrint(level logger.LogLevel, msg string) {
	l.t.Logf("%s", msg)
}

func (testLogger) detach() {
	logger.Flush()
	logger.Reset()
}

func randomByteSlice(l int) (b []byte) {

	r := rand.New(rand.NewSource(int64(l)))

	b = make([]byte, l)
	for i := 0; i < l; i++ {
		b[i] = byte(r.Intn(256))
	}

	return
}

func testDataReader(l int) (r *ChunkReader, slice []byte) {
	slice = randomByteSlice(l)
	r = NewChunkReaderFromBytes(slice)
	return
}

type chunkerTester struct {
	errors  []error
	chunks  []*Chunk
	timeout bool
}

func (self *chunkerTester) checkChunks(t *testing.T, want int) {
	l := len(self.chunks)
	if l != want {
		t.Errorf("expected %v chunks, got %v", want, l)
	}
}

func (self *chunkerTester) Split(chunker *TreeChunker, l int) (key Key, input []byte) {
	// reset
	self.errors = nil
	self.chunks = nil
	self.timeout = false

	data, slice := testDataReader(l)
	input = slice
	key = make([]byte, 32)
	chunkC, errC := chunker.Split(key, data)
	quitC := make(chan bool)
	timeout := time.After(60 * time.Second)

	go func() {
	LOOP:
		for {
			select {
			case <-timeout:
				self.timeout = true
				break LOOP

			case chunk, ok := <-chunkC:
				if chunk != nil {
					self.chunks = append(self.chunks, chunk)
				}
				if !ok { // game over but need to continue to see errc still
					chunkC = nil // make it block so no infinite loop
				}

			case err, ok := <-errC:
				if err != nil {
					self.errors = append(self.errors, err)
				}
				if !ok {
					break LOOP
				}
			}
		}
		close(quitC)
	}()
	<-quitC // waiting for it to finish
	return
}

func (self *chunkerTester) Join(t *testing.T, chunker *TreeChunker, key Key) LazySectionReader {
	// reset but not the chunks
	self.errors = nil
	self.timeout = false

	reader, chunkC, errC := chunker.Join(key)

	quitC := make(chan bool)
	timeout := time.After(60 * time.Second)

	go func() {
	LOOP:
		for {
			t.Logf("waiting to mock Chunk Store")
			select {
			case <-timeout:
				self.timeout = true
				break LOOP

			case chunk, ok := <-chunkC:
				if chunk != nil {
					// this just mocks the behaviour of a chunk store retrieval
					var found bool
					for _, ch := range self.chunks {
						if bytes.Compare(chunk.Key, ch.Key) == 0 {
							found = true
							// ch.Data.Seek(0, 0) // the reader has to be reset
							chunk.Data = ch.Data
							chunk.Size = ch.Size
							close(chunk.C)
							break
						}
					}
					if !found {
						t.Errorf("chunk request unknown for %x", chunk.Key[:4])
					}
				}
				if !ok { // game over but need to continue to see errc still
					chunkC = nil // make it block so no infinite loop
				}

			case err, ok := <-errC:
				if err != nil {
					self.errors = append(self.errors, err)
				}
				if !ok {
					break LOOP
				}
			}
		}
		close(quitC)
	}()
	<-quitC // waiting for it to finish
	return reader
}

func testRandomData(chunker *TreeChunker, tester *chunkerTester, n int, chunks int, t *testing.T) {
	key, input := tester.Split(chunker, n)
	tester.checkChunks(t, chunks)
	t.Logf("chunks: %v", tester.chunks)
	reader := tester.Join(t, chunker, key)
	output := make([]byte, reader.Size())
	_, err := reader.Read(output)
	if err != nil {
		t.Errorf("read error %v\n", err)
	}
	t.Logf(" IN: %x\nOUT: %x\n", input, output)
	if bytes.Compare(output, input) != 0 {
		t.Errorf("input and output mismatch\n IN: %x\nOUT: %x\n", input, output)
	}
}

func TestRandomData(t *testing.T) {
	defer testlog(t).detach()
	chunker := &TreeChunker{
		Branches:     2,
		SplitTimeout: 10 * time.Second,
		JoinTimeout:  10 * time.Second,
	}
	chunker.Init()
	tester := &chunkerTester{}
	testRandomData(chunker, tester, 70, 3, t)
	// testRandomData(chunker, tester, 179, 5, t)
	// testRandomData(chunker, tester, 253, 7, t)
	t.Logf("chunks %v", tester.chunks)
}
