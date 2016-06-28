package storage

import (
	"bytes"
	"fmt"
	"io"
	"runtime"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

func init() {
	glog.SetV(logger.Info)
	glog.SetToStderr(true)
}

/*
Tests TreeChunker by splitting and joining a random byte slice
*/

type test interface {
	Fatalf(string, ...interface{})
}

type chunkerTester struct {
	chunks []*Chunk
	t      test
}

func (self *chunkerTester) checkChunks(t *testing.T, want int) {
	l := len(self.chunks)
	if l != want {
		t.Errorf("expected %v chunks, got %v", want, l)
	}
}

func (self *chunkerTester) Split(chunker Splitter, data io.Reader, size int64, chunkC chan *Chunk, swg *sync.WaitGroup) (key Key) {
	// reset
	self.chunks = nil
	quitC := make(chan bool)
	timeout := time.After(600 * time.Second)
	if chunkC != nil {
		go func() {
			for {
				select {
				case <-timeout:
					self.t.Fatalf("Join timeout error")

				case chunk, ok := <-chunkC:
					if !ok {
						// glog.V(logger.Info).Infof("chunkC closed quitting")
						close(quitC)
						return
					}
					// glog.V(logger.Info).Infof("chunk %v received", len(self.chunks))
					self.chunks = append(self.chunks, chunk)
					if chunk.wg != nil {
						chunk.wg.Done()
					}
				}
			}
		}()
	}
	key, err := chunker.Split(data, size, chunkC, swg, nil)
	if err != nil {
		self.t.Fatalf("Split error: %v", err)
	}
	if chunkC != nil {
		if swg != nil {
			// glog.V(logger.Info).Infof("Waiting for storage to finish")
			swg.Wait()
			// glog.V(logger.Info).Infof("St	orage finished")
		}
		close(chunkC)
	}
	if chunkC != nil {
		<-quitC
	}
	return
}

func (self *chunkerTester) Join(chunker *TreeChunker, key Key, c int, chunkC chan *Chunk, quitC chan bool) LazySectionReader {
	// reset but not the chunks

	reader := chunker.Join(key, chunkC)

	timeout := time.After(600 * time.Second)
	i := 0
	go func() {
		for {
			select {
			case <-timeout:
				self.t.Fatalf("Join timeout error")

			case chunk, ok := <-chunkC:
				if !ok {
					close(quitC)
					return
				}
				i++
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
					self.t.Fatalf("not found	")
				}
				close(chunk.C)
			}
		}
	}()
	return reader
}

func testRandomData(n int, chunks int, t *testing.T) {
	chunker := NewTreeChunker(&ChunkerParams{
		Branches: 128,
		Hash:     "SHA3",
	})
	tester := &chunkerTester{t: t}
	data, input := testDataReaderAndSlice(n)

	chunkC := make(chan *Chunk, 1000)
	swg := &sync.WaitGroup{}

	splitter := chunker
	key := tester.Split(splitter, data, int64(n), chunkC, swg)

	// t.Logf(" Key = %v\n", key)

	// tester.checkChunks(t, chunks)
	chunkC = make(chan *Chunk, 1000)
	quitC := make(chan bool)

	reader := tester.Join(chunker, key, 0, chunkC, quitC)
	output := make([]byte, n)
	r, err := reader.Read(output)
	if r != n || err != io.EOF {
		t.Fatalf("read error  read: %v  n = %v  err = %v\n", r, n, err)
	}
	if input != nil {
		if !bytes.Equal(output, input) {
			t.Fatalf("input and output mismatch\n IN: %v\nOUT: %v\n", input, output)
		}
	}
	close(chunkC)
	<-quitC
}

func TestRandomData(t *testing.T) {
	testRandomData(60, 1, t)
	testRandomData(83, 3, t)
	testRandomData(179, 5, t)
	testRandomData(253, 7, t)
}

func readAll(reader LazySectionReader, result []byte) {
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

func benchReadAll(reader LazySectionReader) {
	size, _ := reader.Size(nil)
	output := make([]byte, 1000)
	for pos := int64(0); pos < size; pos += 1000 {
		reader.ReadAt(output, pos)
	}
}

func benchmarkJoin(n int, t *testing.B) {
	for i := 0; i < t.N; i++ {
		chunker := NewTreeChunker(&ChunkerParams{
			Branches: 128,
			Hash:     "SHA3",
		})
		tester := &chunkerTester{t: t}
		data := testDataReader(n)

		chunkC := make(chan *Chunk, 1000)
		swg := &sync.WaitGroup{}

		key := tester.Split(chunker, data, int64(n), chunkC, swg)
		t.StartTimer()
		chunkC = make(chan *Chunk, 1000)
		quitC := make(chan bool)
		reader := tester.Join(chunker, key, i, chunkC, quitC)
		t.StopTimer()
		benchReadAll(reader)
		close(chunkC)
		<-quitC
	}
}

func benchmarkSplitTree(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		chunker := NewTreeChunker(&ChunkerParams{
			Branches: 128,
			Hash:     "SHA3",
		})
		tester := &chunkerTester{t: t}
		data := testDataReader(n)
		// glog.V(logger.Info).Infof("splitting data of length %v", n)
		tester.Split(chunker, data, int64(n), nil, nil)
	}
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	fmt.Println(stats.Sys)
}

func benchmarkSplitPyramid(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		splitter := NewPyramidChunker(&ChunkerParams{
			Branches: 128,
			Hash:     "SHA3",
		})
		tester := &chunkerTester{t: t}
		data := testDataReader(n)
		// glog.V(logger.Info).Infof("splitting data of length %v", n)
		tester.Split(splitter, data, int64(n), nil, nil)
	}
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	fmt.Println(stats.Sys)
}

func BenchmarkJoin_100_2(t *testing.B)     { benchmarkJoin(100, t) }
func BenchmarkJoin_1000_2(t *testing.B)    { benchmarkJoin(1000, t) }
func BenchmarkJoin_10000_2(t *testing.B)   { benchmarkJoin(10000, t) }
func BenchmarkJoin_100000_2(t *testing.B)  { benchmarkJoin(100000, t) }
func BenchmarkJoin_1000000_2(t *testing.B) { benchmarkJoin(1000000, t) }

func BenchmarkSplitTree_2(t *testing.B)  { benchmarkSplitTree(100, t) }
func BenchmarkSplitTree_2h(t *testing.B) { benchmarkSplitTree(500, t) }
func BenchmarkSplitTree_3(t *testing.B)  { benchmarkSplitTree(1000, t) }
func BenchmarkSplitTree_3h(t *testing.B) { benchmarkSplitTree(5000, t) }
func BenchmarkSplitTree_4(t *testing.B)  { benchmarkSplitTree(10000, t) }
func BenchmarkSplitTree_4h(t *testing.B) { benchmarkSplitTree(50000, t) }
func BenchmarkSplitTree_5(t *testing.B)  { benchmarkSplitTree(100000, t) }
func BenchmarkSplitTree_6(t *testing.B)  { benchmarkSplitTree(1000000, t) }
func BenchmarkSplitTree_7(t *testing.B)  { benchmarkSplitTree(10000000, t) }
func BenchmarkSplitTree_8(t *testing.B)  { benchmarkSplitTree(100000000, t) }

func BenchmarkSplitPyramid_2(t *testing.B)  { benchmarkSplitPyramid(100, t) }
func BenchmarkSplitPyramid_2h(t *testing.B) { benchmarkSplitPyramid(500, t) }
func BenchmarkSplitPyramid_3(t *testing.B)  { benchmarkSplitPyramid(1000, t) }
func BenchmarkSplitPyramid_3h(t *testing.B) { benchmarkSplitPyramid(5000, t) }
func BenchmarkSplitPyramid_4(t *testing.B)  { benchmarkSplitPyramid(10000, t) }
func BenchmarkSplitPyramid_4h(t *testing.B) { benchmarkSplitPyramid(50000, t) }
func BenchmarkSplitPyramid_5(t *testing.B)  { benchmarkSplitPyramid(100000, t) }
func BenchmarkSplitPyramid_6(t *testing.B)  { benchmarkSplitPyramid(1000000, t) }
func BenchmarkSplitPyramid_7(t *testing.B)  { benchmarkSplitPyramid(10000000, t) }
func BenchmarkSplitPyramid_8(t *testing.B)  { benchmarkSplitPyramid(100000000, t) }

// godep go test -bench ./swarm/storage -cpuprofile cpu.out -memprofile mem.out
