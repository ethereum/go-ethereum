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
	"io"
	"runtime"
	"sync"
	"testing"
	"time"
)

/*
Tests TreeChunker by splitting and joining a random byte slice
*/

type test interface {
	Fatalf(string, ...interface{})
	Logf(string, ...interface{})
}

type chunkerTester struct {
	inputs map[uint64][]byte
	chunks map[string]*Chunk
	t      test
}

func (self *chunkerTester) checkChunks(t *testing.T, want int) {
	l := len(self.chunks)
	if l != want {
		t.Errorf("expected %v chunks, got %v", want, l)
	}
}

func (self *chunkerTester) Split(chunker Splitter, data io.Reader, size int64, chunkC chan *Chunk, swg *sync.WaitGroup, expectedError error) (key Key) {
	// reset
	self.chunks = make(map[string]*Chunk)

	if self.inputs == nil {
		self.inputs = make(map[uint64][]byte)
	}

	quitC := make(chan bool)
	timeout := time.After(600 * time.Second)
	if chunkC != nil {
		go func() {
			for {
				select {
				case <-timeout:
					self.t.Fatalf("Join timeout error")
				case <-quitC:
					return
				case chunk := <-chunkC:
					// self.chunks = append(self.chunks, chunk)
					self.chunks[chunk.Key.String()] = chunk
					if chunk.wg != nil {
						chunk.wg.Done()
					}
				}
			}
		}()
	}
	key, err := chunker.Split(data, size, chunkC, swg, nil)
	if err != nil && expectedError == nil {
		self.t.Fatalf("Split error: %v", err)
	} else if expectedError != nil && (err == nil || err.Error() != expectedError.Error()) {
		self.t.Fatalf("Not receiving the correct error! Expected %v, received %v", expectedError, err)
	}
	if chunkC != nil {
		if swg != nil {
			swg.Wait()
		}
		close(quitC)
	}
	return
}

func (self *chunkerTester) Join(chunker Chunker, key Key, c int, chunkC chan *Chunk, quitC chan bool) LazySectionReader {
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
				// this just mocks the behaviour of a chunk store retrieval
				stored, success := self.chunks[chunk.Key.String()]
				if !success {
					self.t.Fatalf("not found")
					return
				}
				chunk.SData = stored.SData
				chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))
				close(chunk.C)
				i++
			}
		}
	}()
	return reader
}

func testRandomBrokenData(splitter Splitter, n int, tester *chunkerTester) {
	data := io.LimitReader(rand.Reader, int64(n))
	brokendata := brokenLimitReader(data, n, n/2)

	buf := make([]byte, n)
	_, err := brokendata.Read(buf)
	if err == nil || err.Error() != "Broken reader" {
		tester.t.Fatalf("Broken reader is not broken, hence broken. Returns: %v", err)
	}

	data = io.LimitReader(rand.Reader, int64(n))
	brokendata = brokenLimitReader(data, n, n/2)

	chunkC := make(chan *Chunk, 1000)
	swg := &sync.WaitGroup{}

	key := tester.Split(splitter, brokendata, int64(n), chunkC, swg, fmt.Errorf("Broken reader"))
	tester.t.Logf(" Key = %v\n", key)
}

func testRandomData(splitter Splitter, n int, tester *chunkerTester) {
	if tester.inputs == nil {
		tester.inputs = make(map[uint64][]byte)
	}
	input, found := tester.inputs[uint64(n)]
	var data io.Reader
	if !found {
		data, input = testDataReaderAndSlice(n)
		tester.inputs[uint64(n)] = input
	} else {
		data = io.LimitReader(bytes.NewReader(input), int64(n))
	}

	chunkC := make(chan *Chunk, 1000)
	swg := &sync.WaitGroup{}

	key := tester.Split(splitter, data, int64(n), chunkC, swg, nil)
	tester.t.Logf(" Key = %v\n", key)

	chunkC = make(chan *Chunk, 1000)
	quitC := make(chan bool)

	chunker := NewTreeChunker(NewChunkerParams())
	reader := tester.Join(chunker, key, 0, chunkC, quitC)
	output := make([]byte, n)
	r, err := reader.Read(output)
	if r != n || err != io.EOF {
		tester.t.Fatalf("read error  read: %v  n = %v  err = %v\n", r, n, err)
	}
	if input != nil {
		if !bytes.Equal(output, input) {
			tester.t.Fatalf("input and output mismatch\n IN: %v\nOUT: %v\n", input, output)
		}
	}
	close(chunkC)
	<-quitC
}

func TestRandomData(t *testing.T) {
	// sizes := []int{123456}
	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 123456, 2345678}
	tester := &chunkerTester{t: t}
	chunker := NewTreeChunker(NewChunkerParams())
	for _, s := range sizes {
		testRandomData(chunker, s, tester)
	}
	pyramid := NewPyramidChunker(NewChunkerParams())
	for _, s := range sizes {
		testRandomData(pyramid, s, tester)
	}
}

func TestRandomBrokenData(t *testing.T) {
	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 123456, 2345678}
	tester := &chunkerTester{t: t}
	chunker := NewTreeChunker(NewChunkerParams())
	for _, s := range sizes {
		testRandomBrokenData(chunker, s, tester)
		t.Logf("done size: %v", s)
	}
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
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		chunker := NewTreeChunker(NewChunkerParams())
		tester := &chunkerTester{t: t}
		data := testDataReader(n)

		chunkC := make(chan *Chunk, 1000)
		swg := &sync.WaitGroup{}

		key := tester.Split(chunker, data, int64(n), chunkC, swg, nil)
		// t.StartTimer()
		chunkC = make(chan *Chunk, 1000)
		quitC := make(chan bool)
		reader := tester.Join(chunker, key, i, chunkC, quitC)
		benchReadAll(reader)
		close(chunkC)
		<-quitC
		// t.StopTimer()
	}
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	fmt.Println(stats.Sys)
}

func benchmarkSplitTree(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		chunker := NewTreeChunker(NewChunkerParams())
		tester := &chunkerTester{t: t}
		data := testDataReader(n)
		tester.Split(chunker, data, int64(n), nil, nil, nil)
	}
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	fmt.Println(stats.Sys)
}

func benchmarkSplitPyramid(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		splitter := NewPyramidChunker(NewChunkerParams())
		tester := &chunkerTester{t: t}
		data := testDataReader(n)
		tester.Split(splitter, data, int64(n), nil, nil, nil)
	}
	stats := new(runtime.MemStats)
	runtime.ReadMemStats(stats)
	fmt.Println(stats.Sys)
}

func BenchmarkJoin_2(t *testing.B) { benchmarkJoin(100, t) }
func BenchmarkJoin_3(t *testing.B) { benchmarkJoin(1000, t) }
func BenchmarkJoin_4(t *testing.B) { benchmarkJoin(10000, t) }
func BenchmarkJoin_5(t *testing.B) { benchmarkJoin(100000, t) }
func BenchmarkJoin_6(t *testing.B) { benchmarkJoin(1000000, t) }
func BenchmarkJoin_7(t *testing.B) { benchmarkJoin(10000000, t) }
func BenchmarkJoin_8(t *testing.B) { benchmarkJoin(100000000, t) }

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
