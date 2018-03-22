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
	"errors"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/sha3"
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

func (self *chunkerTester) Split(chunker Splitter, data io.Reader, size int64, chunkC chan *Chunk, swg *sync.WaitGroup, expectedError error) (key Key, err error) {
	// reset
	self.chunks = make(map[string]*Chunk)

	if self.inputs == nil {
		self.inputs = make(map[uint64][]byte)
	}

	quitC := make(chan bool)
	timeout := time.After(600 * time.Second)
	if chunkC != nil {
		go func() error {
			for {
				select {
				case <-timeout:
					return errors.New("Split timeout error")
				case <-quitC:
					return nil
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

	key, err = chunker.Split(data, size, chunkC, swg, nil)
	if err != nil && expectedError == nil {
		err = fmt.Errorf("Split error: %v", err)
	}

	if chunkC != nil {
		if swg != nil {
			swg.Wait()
		}
		close(quitC)
	}
	return key, err
}

func (self *chunkerTester) Append(chunker Splitter, rootKey Key, data io.Reader, chunkC chan *Chunk, swg *sync.WaitGroup, expectedError error) (key Key, err error) {
	quitC := make(chan bool)
	timeout := time.After(60 * time.Second)
	if chunkC != nil {
		go func() error {
			for {
				select {
				case <-timeout:
					return errors.New("Append timeout error")
				case <-quitC:
					return nil
				case chunk := <-chunkC:
					if chunk != nil {
						stored, success := self.chunks[chunk.Key.String()]
						if !success {
							// Requesting data
							self.chunks[chunk.Key.String()] = chunk
							if chunk.wg != nil {
								chunk.wg.Done()
							}
						} else {
							// getting data
							chunk.SData = stored.SData
							chunk.Size = int64(binary.LittleEndian.Uint64(chunk.SData[0:8]))
							close(chunk.C)
						}
					}
				}
			}
		}()
	}

	key, err = chunker.Append(rootKey, data, chunkC, swg, nil)
	if err != nil && expectedError == nil {
		err = fmt.Errorf("Append error: %v", err)
	}

	if chunkC != nil {
		if swg != nil {
			swg.Wait()
		}
		close(quitC)
	}
	return key, err
}

func (self *chunkerTester) Join(chunker Chunker, key Key, c int, chunkC chan *Chunk, quitC chan bool) LazySectionReader {
	// reset but not the chunks

	reader := chunker.Join(key, chunkC)

	timeout := time.After(600 * time.Second)
	i := 0
	go func() error {
		for {
			select {
			case <-timeout:
				return errors.New("Join timeout error")
			case chunk, ok := <-chunkC:
				if !ok {
					close(quitC)
					return nil
				}
				// this just mocks the behaviour of a chunk store retrieval
				stored, success := self.chunks[chunk.Key.String()]
				if !success {
					return errors.New("Not found")
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

	expectedError := fmt.Errorf("Broken reader")
	key, err := tester.Split(splitter, brokendata, int64(n), chunkC, swg, expectedError)
	if err == nil || err.Error() != expectedError.Error() {
		tester.t.Fatalf("Not receiving the correct error! Expected %v, received %v", expectedError, err)
	}
	tester.t.Logf(" Key = %v\n", key)
}

func testRandomData(splitter Splitter, n int, tester *chunkerTester) Key {
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

	key, err := tester.Split(splitter, data, int64(n), chunkC, swg, nil)
	if err != nil {
		tester.t.Fatalf(err.Error())
	}
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

	return key
}

func testRandomDataAppend(splitter Splitter, n, m int, tester *chunkerTester) {
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

	key, err := tester.Split(splitter, data, int64(n), chunkC, swg, nil)
	if err != nil {
		tester.t.Fatalf(err.Error())
	}
	tester.t.Logf(" Key = %v\n", key)

	//create a append data stream
	appendInput, found := tester.inputs[uint64(m)]
	var appendData io.Reader
	if !found {
		appendData, appendInput = testDataReaderAndSlice(m)
		tester.inputs[uint64(m)] = appendInput
	} else {
		appendData = io.LimitReader(bytes.NewReader(appendInput), int64(m))
	}

	chunkC = make(chan *Chunk, 1000)
	swg = &sync.WaitGroup{}

	newKey, err := tester.Append(splitter, key, appendData, chunkC, swg, nil)
	if err != nil {
		tester.t.Fatalf(err.Error())
	}
	tester.t.Logf(" NewKey = %v\n", newKey)

	chunkC = make(chan *Chunk, 1000)
	quitC := make(chan bool)

	chunker := NewTreeChunker(NewChunkerParams())
	reader := tester.Join(chunker, newKey, 0, chunkC, quitC)
	newOutput := make([]byte, n+m)
	r, err := reader.Read(newOutput)
	if r != (n + m) {
		tester.t.Fatalf("read error  read: %v  n = %v  err = %v\n", r, n, err)
	}

	newInput := append(input, appendInput...)
	if !bytes.Equal(newOutput, newInput) {
		tester.t.Fatalf("input and output mismatch\n IN: %v\nOUT: %v\n", newInput, newOutput)
	}

	close(chunkC)
}

func TestSha3ForCorrectness(t *testing.T) {
	tester := &chunkerTester{t: t}

	size := 4096
	input := make([]byte, size+8)
	binary.LittleEndian.PutUint64(input[:8], uint64(size))

	io.LimitReader(bytes.NewReader(input[8:]), int64(size))

	rawSha3 := sha3.NewKeccak256()
	rawSha3.Reset()
	rawSha3.Write(input)
	rawSha3Output := rawSha3.Sum(nil)

	sha3FromMakeFunc := MakeHashFunc(SHA3Hash)()
	sha3FromMakeFunc.ResetWithLength(input[:8])
	sha3FromMakeFunc.Write(input[8:])
	sha3FromMakeFuncOutput := sha3FromMakeFunc.Sum(nil)

	if len(rawSha3Output) != len(sha3FromMakeFuncOutput) {
		tester.t.Fatalf("Original SHA3 and abstracted Sha3 has different length %v:%v\n", len(rawSha3Output), len(sha3FromMakeFuncOutput))
	}

	if !bytes.Equal(rawSha3Output, sha3FromMakeFuncOutput) {
		tester.t.Fatalf("Original SHA3 and abstracted Sha3 mismatch %v:%v\n", rawSha3Output, sha3FromMakeFuncOutput)
	}

}

func TestDataAppend(t *testing.T) {
	sizes := []int{1, 1, 1, 4095, 4096, 4097, 1, 1, 1, 123456, 2345678, 2345678}
	appendSizes := []int{4095, 4096, 4097, 1, 1, 1, 8191, 8192, 8193, 9000, 3000, 5000}

	tester := &chunkerTester{t: t}
	chunker := NewPyramidChunker(NewChunkerParams())
	for i, s := range sizes {
		testRandomDataAppend(chunker, s, appendSizes[i], tester)

	}
}

func TestRandomData(t *testing.T) {
	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 123456, 2345678}
	tester := &chunkerTester{t: t}

	chunker := NewTreeChunker(NewChunkerParams())
	pyramid := NewPyramidChunker(NewChunkerParams())
	for _, s := range sizes {
		treeChunkerKey := testRandomData(chunker, s, tester)
		pyramidChunkerKey := testRandomData(pyramid, s, tester)
		if treeChunkerKey.String() != pyramidChunkerKey.String() {
			tester.t.Fatalf("tree chunker and pyramid chunker key mismatch for size %v\n TC: %v\n PC: %v\n", s, treeChunkerKey.String(), pyramidChunkerKey.String())
		}
	}

	cp := NewChunkerParams()
	cp.Hash = BMTHash
	chunker = NewTreeChunker(cp)
	pyramid = NewPyramidChunker(cp)
	for _, s := range sizes {
		treeChunkerKey := testRandomData(chunker, s, tester)
		pyramidChunkerKey := testRandomData(pyramid, s, tester)
		if treeChunkerKey.String() != pyramidChunkerKey.String() {
			tester.t.Fatalf("tree chunker BMT and pyramid chunker BMT key mismatch for size %v \n TC: %v\n PC: %v\n", s, treeChunkerKey.String(), pyramidChunkerKey.String())
		}
	}

}

func XTestRandomBrokenData(t *testing.T) {
	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 123456, 2345678}
	tester := &chunkerTester{t: t}
	chunker := NewTreeChunker(NewChunkerParams())
	for _, s := range sizes {
		testRandomBrokenData(chunker, s, tester)
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

		key, err := tester.Split(chunker, data, int64(n), chunkC, swg, nil)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}
		chunkC = make(chan *Chunk, 1000)
		quitC := make(chan bool)
		reader := tester.Join(chunker, key, i, chunkC, quitC)
		benchReadAll(reader)
		close(chunkC)
		<-quitC
	}
}

func benchmarkSplitTreeSHA3(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		chunker := NewTreeChunker(NewChunkerParams())
		tester := &chunkerTester{t: t}
		data := testDataReader(n)
		_, err := tester.Split(chunker, data, int64(n), nil, nil, nil)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}
	}
}

func benchmarkSplitTreeBMT(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		cp := NewChunkerParams()
		cp.Hash = BMTHash
		chunker := NewTreeChunker(cp)
		tester := &chunkerTester{t: t}
		data := testDataReader(n)
		_, err := tester.Split(chunker, data, int64(n), nil, nil, nil)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}
	}
}

func benchmarkSplitPyramidSHA3(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		splitter := NewPyramidChunker(NewChunkerParams())
		tester := &chunkerTester{t: t}
		data := testDataReader(n)
		_, err := tester.Split(splitter, data, int64(n), nil, nil, nil)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}
	}
}

func benchmarkSplitPyramidBMT(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		cp := NewChunkerParams()
		cp.Hash = BMTHash
		splitter := NewPyramidChunker(cp)
		tester := &chunkerTester{t: t}
		data := testDataReader(n)
		_, err := tester.Split(splitter, data, int64(n), nil, nil, nil)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}
	}
}

func benchmarkAppendPyramid(n, m int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		chunker := NewPyramidChunker(NewChunkerParams())
		tester := &chunkerTester{t: t}
		data := testDataReader(n)
		data1 := testDataReader(m)

		chunkC := make(chan *Chunk, 1000)
		swg := &sync.WaitGroup{}
		key, err := tester.Split(chunker, data, int64(n), chunkC, swg, nil)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}

		chunkC = make(chan *Chunk, 1000)
		swg = &sync.WaitGroup{}

		_, err = tester.Append(chunker, key, data1, chunkC, swg, nil)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}

		close(chunkC)
	}
}

func BenchmarkJoin_2(t *testing.B) { benchmarkJoin(100, t) }
func BenchmarkJoin_3(t *testing.B) { benchmarkJoin(1000, t) }
func BenchmarkJoin_4(t *testing.B) { benchmarkJoin(10000, t) }
func BenchmarkJoin_5(t *testing.B) { benchmarkJoin(100000, t) }
func BenchmarkJoin_6(t *testing.B) { benchmarkJoin(1000000, t) }
func BenchmarkJoin_7(t *testing.B) { benchmarkJoin(10000000, t) }
func BenchmarkJoin_8(t *testing.B) { benchmarkJoin(100000000, t) }

func BenchmarkSplitTreeSHA3_2(t *testing.B)  { benchmarkSplitTreeSHA3(100, t) }
func BenchmarkSplitTreeSHA3_2h(t *testing.B) { benchmarkSplitTreeSHA3(500, t) }
func BenchmarkSplitTreeSHA3_3(t *testing.B)  { benchmarkSplitTreeSHA3(1000, t) }
func BenchmarkSplitTreeSHA3_3h(t *testing.B) { benchmarkSplitTreeSHA3(5000, t) }
func BenchmarkSplitTreeSHA3_4(t *testing.B)  { benchmarkSplitTreeSHA3(10000, t) }
func BenchmarkSplitTreeSHA3_4h(t *testing.B) { benchmarkSplitTreeSHA3(50000, t) }
func BenchmarkSplitTreeSHA3_5(t *testing.B)  { benchmarkSplitTreeSHA3(100000, t) }
func BenchmarkSplitTreeSHA3_6(t *testing.B)  { benchmarkSplitTreeSHA3(1000000, t) }
func BenchmarkSplitTreeSHA3_7(t *testing.B)  { benchmarkSplitTreeSHA3(10000000, t) }
func BenchmarkSplitTreeSHA3_8(t *testing.B)  { benchmarkSplitTreeSHA3(100000000, t) }

func BenchmarkSplitTreeBMT_2(t *testing.B)  { benchmarkSplitTreeBMT(100, t) }
func BenchmarkSplitTreeBMT_2h(t *testing.B) { benchmarkSplitTreeBMT(500, t) }
func BenchmarkSplitTreeBMT_3(t *testing.B)  { benchmarkSplitTreeBMT(1000, t) }
func BenchmarkSplitTreeBMT_3h(t *testing.B) { benchmarkSplitTreeBMT(5000, t) }
func BenchmarkSplitTreeBMT_4(t *testing.B)  { benchmarkSplitTreeBMT(10000, t) }
func BenchmarkSplitTreeBMT_4h(t *testing.B) { benchmarkSplitTreeBMT(50000, t) }
func BenchmarkSplitTreeBMT_5(t *testing.B)  { benchmarkSplitTreeBMT(100000, t) }
func BenchmarkSplitTreeBMT_6(t *testing.B)  { benchmarkSplitTreeBMT(1000000, t) }
func BenchmarkSplitTreeBMT_7(t *testing.B)  { benchmarkSplitTreeBMT(10000000, t) }
func BenchmarkSplitTreeBMT_8(t *testing.B)  { benchmarkSplitTreeBMT(100000000, t) }

func BenchmarkSplitPyramidSHA3_2(t *testing.B)  { benchmarkSplitPyramidSHA3(100, t) }
func BenchmarkSplitPyramidSHA3_2h(t *testing.B) { benchmarkSplitPyramidSHA3(500, t) }
func BenchmarkSplitPyramidSHA3_3(t *testing.B)  { benchmarkSplitPyramidSHA3(1000, t) }
func BenchmarkSplitPyramidSHA3_3h(t *testing.B) { benchmarkSplitPyramidSHA3(5000, t) }
func BenchmarkSplitPyramidSHA3_4(t *testing.B)  { benchmarkSplitPyramidSHA3(10000, t) }
func BenchmarkSplitPyramidSHA3_4h(t *testing.B) { benchmarkSplitPyramidSHA3(50000, t) }
func BenchmarkSplitPyramidSHA3_5(t *testing.B)  { benchmarkSplitPyramidSHA3(100000, t) }
func BenchmarkSplitPyramidSHA3_6(t *testing.B)  { benchmarkSplitPyramidSHA3(1000000, t) }
func BenchmarkSplitPyramidSHA3_7(t *testing.B)  { benchmarkSplitPyramidSHA3(10000000, t) }
func BenchmarkSplitPyramidSHA3_8(t *testing.B)  { benchmarkSplitPyramidSHA3(100000000, t) }

func BenchmarkSplitPyramidBMT_2(t *testing.B)  { benchmarkSplitPyramidBMT(100, t) }
func BenchmarkSplitPyramidBMT_2h(t *testing.B) { benchmarkSplitPyramidBMT(500, t) }
func BenchmarkSplitPyramidBMT_3(t *testing.B)  { benchmarkSplitPyramidBMT(1000, t) }
func BenchmarkSplitPyramidBMT_3h(t *testing.B) { benchmarkSplitPyramidBMT(5000, t) }
func BenchmarkSplitPyramidBMT_4(t *testing.B)  { benchmarkSplitPyramidBMT(10000, t) }
func BenchmarkSplitPyramidBMT_4h(t *testing.B) { benchmarkSplitPyramidBMT(50000, t) }
func BenchmarkSplitPyramidBMT_5(t *testing.B)  { benchmarkSplitPyramidBMT(100000, t) }
func BenchmarkSplitPyramidBMT_6(t *testing.B)  { benchmarkSplitPyramidBMT(1000000, t) }
func BenchmarkSplitPyramidBMT_7(t *testing.B)  { benchmarkSplitPyramidBMT(10000000, t) }
func BenchmarkSplitPyramidBMT_8(t *testing.B)  { benchmarkSplitPyramidBMT(100000000, t) }

func BenchmarkAppendPyramid_2(t *testing.B)  { benchmarkAppendPyramid(100, 1000, t) }
func BenchmarkAppendPyramid_2h(t *testing.B) { benchmarkAppendPyramid(500, 1000, t) }
func BenchmarkAppendPyramid_3(t *testing.B)  { benchmarkAppendPyramid(1000, 1000, t) }
func BenchmarkAppendPyramid_4(t *testing.B)  { benchmarkAppendPyramid(10000, 1000, t) }
func BenchmarkAppendPyramid_4h(t *testing.B) { benchmarkAppendPyramid(50000, 1000, t) }
func BenchmarkAppendPyramid_5(t *testing.B)  { benchmarkAppendPyramid(1000000, 1000, t) }
func BenchmarkAppendPyramid_6(t *testing.B)  { benchmarkAppendPyramid(1000000, 1000, t) }
func BenchmarkAppendPyramid_7(t *testing.B)  { benchmarkAppendPyramid(10000000, 1000, t) }
func BenchmarkAppendPyramid_8(t *testing.B)  { benchmarkAppendPyramid(100000000, 1000, t) }

// go test -timeout 20m -cpu 4 -bench=./swarm/storage -run no
// If you dont add the timeout argument above .. the benchmark will timeout and dump
