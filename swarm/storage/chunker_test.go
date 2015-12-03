package storage

import (
	"bytes"
	// "fmt"
	"io"
	"testing"
	"time"
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
	errC := chunker.Split(key, data, chunkC, nil)
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
				// fmt.Printf("err %v", err)
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

func (self *chunkerTester) Join(chunker *TreeChunker, key Key, c int) SectionReader {
	// reset but not the chunks
	self.errors = nil
	self.timeout = false
	chunkC := make(chan *Chunk, 1000)

	reader := chunker.Join(key, chunkC)

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
				// dpaLogger.DebugDetailf("TESTER: chunk request %x", chunk.Key[:4])
				// this just mocks the behaviour of a chunk store retrieval
				var found bool
				for _, ch := range self.chunks {
					if bytes.Equal(chunk.Key, ch.Key) {
						found = true
						chunk.SData = ch.SData
						break
					}
				}
				if !found {
					// fmt.Printf("TESTER: chunk unknown for %x", chunk.Key[:4])
				}
				close(chunk.C)
				// dpaLogger.DebugDetailf("TESTER: chunk request served %x", chunk.Key[:4])
			}
		}
	}()
	return reader
}

func testRandomData(chunker *TreeChunker, tester *chunkerTester, n int, chunks int, t *testing.T) {
	key, input := tester.Split(chunker, n)

	t.Logf(" Key = %x\n", key)

	tester.checkChunks(t, chunks)
	time.Sleep(100 * time.Millisecond)

	reader := tester.Join(chunker, key, 0)
	output := make([]byte, n)
	r, err := reader.Read(output)
	if r != n || err != io.EOF {
		t.Errorf("read error  read: %v  n = %v  err = %v\n", r, n, err)
	}
	// t.Logf(" IN: %x\nOUT: %x\n", input, output)
	if !bytes.Equal(output, input) {
		t.Errorf("input and output mismatch\n IN: %x\nOUT: %x\n", input, output)
	}
}

func TestRandomData(t *testing.T) {
	chunker, tester := chunkerAndTester()
	testRandomData(chunker, tester, 60, 1, t)
	testRandomData(chunker, tester, 179, 5, t)
	testRandomData(chunker, tester, 253, 7, t)
	// t.Logf("chunks %v", tester.chunks)
}

func chunkerAndTester() (chunker *TreeChunker, tester *chunkerTester) {
	chunker = NewTreeChunker(&ChunkerParams{
		Branches:     2,
		Hash:         "SHA256",
		SplitTimeout: 10,
		JoinTimeout:  10,
	})
	tester = &chunkerTester{}
	return
}

func readAll(reader SectionReader, result []byte) {
	size := int64(len(result))

	var end int64
	for pos := int64(0); pos < size; pos += 1000 {
		if pos+1000 > size {
			end = size
		} else {
			end = pos + 1000
		}
		reader.ReadAt(result[pos:end], pos)
	}
}

func benchReadAll(reader SectionReader) {
	size := reader.Size()
	output := make([]byte, 1000)
	for pos := int64(0); pos < size; pos += 1000 {
		reader.ReadAt(output, pos)
	}
}

func benchmarkJoinRandomData(n int, chunks int, t *testing.B) {
	t.StopTimer()
	for i := 0; i < t.N; i++ {
		// fmt.Printf("round %v\n", i)
		chunker, tester := chunkerAndTester()
		key, _ := tester.Split(chunker, n)
		// fmt.Printf("split done %v, joining...\n", i)
		t.StartTimer()
		reader := tester.Join(chunker, key, i)
		// fmt.Printf("join done %v, reading...\n", i)
		benchReadAll(reader)
	}
}

func benchmarkSplitRandomData(n int, chunks int, t *testing.B) {
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
