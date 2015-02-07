package bzz

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/bzz/test"
)

/*
Tests TreeChunker by splitting and joining a random byte slice
*/

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
	chunkC := make(chan *Chunk, 1000)
	errC := chunker.Split(key, data, chunkC)
	quitC := make(chan bool)
	timeout := time.After(600 * time.Second)

	go func() {
	LOOP:
		for {
			select {
			case <-timeout:
				self.timeout = true
				break LOOP

			case chunk := <-chunkC:
				if chunk != nil {
					self.chunks = append(self.chunks, chunk)
				} else {
					break LOOP
				}

			case err, ok := <-errC:
				if err != nil {
					self.errors = append(self.errors, err)
				}
				if !ok {
					close(chunkC)
					errC = nil
				}
			}
		}
		close(quitC)
	}()
	<-quitC // waiting for it to finish
	return
}

func (self *chunkerTester) Join(chunker *TreeChunker, key Key, c int) (LazySectionReader, chan bool) {
	// reset but not the chunks
	self.errors = nil
	self.timeout = false
	chunkC := make(chan *Chunk, 1000)

	reader, errC := chunker.Join(key, chunkC)

	quitC := make(chan bool)
	timeout := time.After(600 * time.Second)
	i := 0
	go func() {
	LOOP:
		for {
			select {
			case <-quitC:
				break LOOP

			case <-timeout:
				self.timeout = true
				break LOOP

			case chunk := <-chunkC:
				i++
				// this just mocks the behaviour of a chunk store retrieval
				var found bool
				for _, ch := range self.chunks {
					if bytes.Compare(chunk.Key, ch.Key) == 0 {
						found = true
						chunk.Reader = ch.Reader
						chunk.Size = ch.Size
						close(chunk.C)
						break
					}
				}
				if !found {
					fmt.Printf("chunk request unknown for %x", chunk.Key[:4])
				}
			case err, ok := <-errC:
				if err != nil {
					fmt.Printf("error %v", err)
					self.errors = append(self.errors, err)
				}
				if !ok {
					close(chunkC)
					errC = nil
					break LOOP
				}
			}
		}
	}()
	return reader, quitC
}

func testRandomData(chunker *TreeChunker, tester *chunkerTester, n int, chunks int, t *testing.T) {
	key, input := tester.Split(chunker, n)
	tester.checkChunks(t, chunks)
	t.Logf("chunks: %v", tester.chunks)
	reader, quitC := tester.Join(chunker, key, 0)
	output := make([]byte, reader.Size())
	_, err := reader.Read(output)
	if err != nil {
		t.Errorf("read error %v\n", err)
	}
	t.Logf(" IN: %x\nOUT: %x\n", input, output)
	if bytes.Compare(output, input) != 0 {
		t.Errorf("input and output mismatch\n IN: %x\nOUT: %x\n", input, output)
	}
	close(quitC)
}

func TestRandomData(t *testing.T) {
	defer test.Testlog(t).Detach()
	chunker := &TreeChunker{
		Branches:     2,
		SplitTimeout: 10 * time.Second,
		JoinTimeout:  10 * time.Second,
	}
	chunker.Init()
	tester := &chunkerTester{}
	testRandomData(chunker, tester, 70, 3, t)
	testRandomData(chunker, tester, 179, 5, t)
	testRandomData(chunker, tester, 253, 7, t)
	t.Logf("chunks %v", tester.chunks)
}

func chunkerAndTester() (chunker *TreeChunker, tester *chunkerTester) {
	chunker = &TreeChunker{
		Branches:     2,
		SplitTimeout: 10 * time.Second,
		JoinTimeout:  10 * time.Second,
	}
	chunker.Init()
	tester = &chunkerTester{}
	return
}

func readAll(reader SectionReader) {
	size := reader.Size()
	output := make([]byte, 1000)
	for pos := int64(0); pos < size; pos += 1000 {
		reader.ReadAt(output, pos)
	}
}

func benchmarkJoinRandomData(n int, chunks int, t *testing.B) {
	for i := 0; i < t.N; i++ {
		t.StopTimer()
		chunker, tester := chunkerAndTester()
		key, _ := tester.Split(chunker, n)
		t.StartTimer()
		reader, quitC := tester.Join(chunker, key, i)
		readAll(reader)
		close(quitC)
	}
}

func benchmarkSplitRandomData(n int, chunks int, t *testing.B) {
	defer test.Benchlog(t).Detach()
	for i := 0; i < t.N; i++ {
		chunker, tester := chunkerAndTester()
		tester.Split(chunker, n)
	}
}

func BenchmarkJoinRandomData_100_2(t *testing.B)     { benchmarkJoinRandomData(100, 3, t) }
func BenchmarkJoinRandomData_1000_2(t *testing.B)    { benchmarkJoinRandomData(1000, 3, t) }
func BenchmarkJoinRandomData_10000_2(t *testing.B)   { benchmarkJoinRandomData(10000, 3, t) }
func BenchmarkJoinRandomData_100000_2(t *testing.B)  { benchmarkJoinRandomData(100000, 3, t) }
func BenchmarkJoinRandomData_1000000_2(t *testing.B) { benchmarkJoinRandomData(1000000, 3, t) }

func BenchmarkSplitRandomData_100_2(t *testing.B)      { benchmarkSplitRandomData(100, 3, t) }
func BenchmarkSplitRandomData_1000_2(t *testing.B)     { benchmarkSplitRandomData(1000, 3, t) }
func BenchmarkSplitRandomData_10000_2(t *testing.B)    { benchmarkSplitRandomData(10000, 3, t) }
func BenchmarkSplitRandomData_100000_2(t *testing.B)   { benchmarkSplitRandomData(100000, 3, t) }
func BenchmarkSplitRandomData_1000000_2(t *testing.B)  { benchmarkSplitRandomData(1000000, 3, t) }
func BenchmarkSplitRandomData_10000000_2(t *testing.B) { benchmarkSplitRandomData(10000000, 3, t) }

// go test -bench ./bzz -cpuprofile cpu.out -memprofile mem.out
