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
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"testing"

	"github.com/ethereum/go-ethereum/swarm/testutil"
	"golang.org/x/crypto/sha3"
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
	t      test
}

func newTestHasherStore(store ChunkStore, hash string) *hasherStore {
	return NewHasherStore(store, MakeHashFunc(hash), false)
}

func testRandomBrokenData(n int, tester *chunkerTester) {
	data := testutil.RandomReader(1, n)
	brokendata := brokenLimitReader(data, n, n/2)

	buf := make([]byte, n)
	_, err := brokendata.Read(buf)
	if err == nil || err.Error() != "Broken reader" {
		tester.t.Fatalf("Broken reader is not broken, hence broken. Returns: %v", err)
	}

	data = testutil.RandomReader(2, n)
	brokendata = brokenLimitReader(data, n, n/2)

	putGetter := newTestHasherStore(NewMapChunkStore(), SHA3Hash)

	expectedError := fmt.Errorf("Broken reader")
	ctx := context.Background()
	key, _, err := TreeSplit(ctx, brokendata, int64(n), putGetter)
	if err == nil || err.Error() != expectedError.Error() {
		tester.t.Fatalf("Not receiving the correct error! Expected %v, received %v", expectedError, err)
	}
	tester.t.Logf(" Address = %v\n", key)
}

func testRandomData(usePyramid bool, hash string, n int, tester *chunkerTester) Address {
	if tester.inputs == nil {
		tester.inputs = make(map[uint64][]byte)
	}
	input, found := tester.inputs[uint64(n)]
	var data io.Reader
	if !found {
		input = testutil.RandomBytes(1, n)
		data = bytes.NewReader(input)
		tester.inputs[uint64(n)] = input
	} else {
		data = io.LimitReader(bytes.NewReader(input), int64(n))
	}

	putGetter := newTestHasherStore(NewMapChunkStore(), hash)

	var addr Address
	var wait func(context.Context) error
	var err error
	ctx := context.TODO()
	if usePyramid {
		addr, wait, err = PyramidSplit(ctx, data, putGetter, putGetter)
	} else {
		addr, wait, err = TreeSplit(ctx, data, int64(n), putGetter)
	}
	if err != nil {
		tester.t.Fatalf(err.Error())
	}
	tester.t.Logf(" Address = %v\n", addr)
	err = wait(ctx)
	if err != nil {
		tester.t.Fatalf(err.Error())
	}

	reader := TreeJoin(ctx, addr, putGetter, 0)
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

	// testing partial read
	for i := 1; i < n; i += 10000 {
		readableLength := n - i
		r, err := reader.ReadAt(output, int64(i))
		if r != readableLength || err != io.EOF {
			tester.t.Fatalf("readAt error with offset %v read: %v  n = %v  err = %v\n", i, r, readableLength, err)
		}
		if input != nil {
			if !bytes.Equal(output[:readableLength], input[i:]) {
				tester.t.Fatalf("input and output mismatch\n IN: %v\nOUT: %v\n", input[i:], output[:readableLength])
			}
		}
	}

	return addr
}

func TestSha3ForCorrectness(t *testing.T) {
	tester := &chunkerTester{t: t}

	size := 4096
	input := make([]byte, size+8)
	binary.LittleEndian.PutUint64(input[:8], uint64(size))

	io.LimitReader(bytes.NewReader(input[8:]), int64(size))

	rawSha3 := sha3.NewLegacyKeccak256()
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
	for i := range sizes {
		n := sizes[i]
		m := appendSizes[i]

		if tester.inputs == nil {
			tester.inputs = make(map[uint64][]byte)
		}
		input, found := tester.inputs[uint64(n)]
		var data io.Reader
		if !found {
			input = testutil.RandomBytes(i, n)
			data = bytes.NewReader(input)
			tester.inputs[uint64(n)] = input
		} else {
			data = io.LimitReader(bytes.NewReader(input), int64(n))
		}

		store := NewMapChunkStore()
		putGetter := newTestHasherStore(store, SHA3Hash)

		ctx := context.TODO()
		addr, wait, err := PyramidSplit(ctx, data, putGetter, putGetter)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}
		err = wait(ctx)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}
		//create a append data stream
		appendInput, found := tester.inputs[uint64(m)]
		var appendData io.Reader
		if !found {
			appendInput = testutil.RandomBytes(i, m)
			appendData = bytes.NewReader(appendInput)
			tester.inputs[uint64(m)] = appendInput
		} else {
			appendData = io.LimitReader(bytes.NewReader(appendInput), int64(m))
		}

		putGetter = newTestHasherStore(store, SHA3Hash)
		newAddr, wait, err := PyramidAppend(ctx, addr, appendData, putGetter, putGetter)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}
		err = wait(ctx)
		if err != nil {
			tester.t.Fatalf(err.Error())
		}

		reader := TreeJoin(ctx, newAddr, putGetter, 0)
		newOutput := make([]byte, n+m)
		r, err := reader.Read(newOutput)
		if r != (n + m) {
			tester.t.Fatalf("read error  read: %v  n = %v  m = %v  err = %v\n", r, n, m, err)
		}

		newInput := append(input, appendInput...)
		if !bytes.Equal(newOutput, newInput) {
			tester.t.Fatalf("input and output mismatch\n IN: %v\nOUT: %v\n", newInput, newOutput)
		}
	}
}

func TestRandomData(t *testing.T) {
	// This test can validate files up to a relatively short length, as tree chunker slows down drastically.
	// Validation of longer files is done by TestLocalStoreAndRetrieve in swarm package.
	//sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 524288, 524288 + 1, 524288 + 4097, 7 * 524288, 7*524288 + 1, 7*524288 + 4097}
	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4097, 8191, 8192, 12288, 12289, 524288}
	tester := &chunkerTester{t: t}

	for _, s := range sizes {
		treeChunkerAddress := testRandomData(false, SHA3Hash, s, tester)
		pyramidChunkerAddress := testRandomData(true, SHA3Hash, s, tester)
		if treeChunkerAddress.String() != pyramidChunkerAddress.String() {
			tester.t.Fatalf("tree chunker and pyramid chunker key mismatch for size %v\n TC: %v\n PC: %v\n", s, treeChunkerAddress.String(), pyramidChunkerAddress.String())
		}
	}

	for _, s := range sizes {
		treeChunkerAddress := testRandomData(false, BMTHash, s, tester)
		pyramidChunkerAddress := testRandomData(true, BMTHash, s, tester)
		if treeChunkerAddress.String() != pyramidChunkerAddress.String() {
			tester.t.Fatalf("tree chunker and pyramid chunker key mismatch for size %v\n TC: %v\n PC: %v\n", s, treeChunkerAddress.String(), pyramidChunkerAddress.String())
		}
	}
}

func TestRandomBrokenData(t *testing.T) {
	sizes := []int{1, 60, 83, 179, 253, 1024, 4095, 4096, 4097, 8191, 8192, 8193, 12287, 12288, 12289, 123456, 2345678}
	tester := &chunkerTester{t: t}
	for _, s := range sizes {
		testRandomBrokenData(s, tester)
	}
}

func benchReadAll(reader LazySectionReader) {
	size, _ := reader.Size(context.TODO(), nil)
	output := make([]byte, 1000)
	for pos := int64(0); pos < size; pos += 1000 {
		reader.ReadAt(output, pos)
	}
}

func benchmarkSplitJoin(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		data := testutil.RandomReader(i, n)

		putGetter := newTestHasherStore(NewMapChunkStore(), SHA3Hash)
		ctx := context.TODO()
		key, wait, err := PyramidSplit(ctx, data, putGetter, putGetter)
		if err != nil {
			t.Fatalf(err.Error())
		}
		err = wait(ctx)
		if err != nil {
			t.Fatalf(err.Error())
		}
		reader := TreeJoin(ctx, key, putGetter, 0)
		benchReadAll(reader)
	}
}

func benchmarkSplitTreeSHA3(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		data := testutil.RandomReader(i, n)
		putGetter := newTestHasherStore(&FakeChunkStore{}, SHA3Hash)

		ctx := context.Background()
		_, wait, err := TreeSplit(ctx, data, int64(n), putGetter)
		if err != nil {
			t.Fatalf(err.Error())
		}
		err = wait(ctx)
		if err != nil {
			t.Fatalf(err.Error())
		}

	}
}

func benchmarkSplitTreeBMT(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		data := testutil.RandomReader(i, n)
		putGetter := newTestHasherStore(&FakeChunkStore{}, BMTHash)

		ctx := context.Background()
		_, wait, err := TreeSplit(ctx, data, int64(n), putGetter)
		if err != nil {
			t.Fatalf(err.Error())
		}
		err = wait(ctx)
		if err != nil {
			t.Fatalf(err.Error())
		}
	}
}

func benchmarkSplitPyramidBMT(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		data := testutil.RandomReader(i, n)
		putGetter := newTestHasherStore(&FakeChunkStore{}, BMTHash)

		ctx := context.Background()
		_, wait, err := PyramidSplit(ctx, data, putGetter, putGetter)
		if err != nil {
			t.Fatalf(err.Error())
		}
		err = wait(ctx)
		if err != nil {
			t.Fatalf(err.Error())
		}
	}
}

func benchmarkSplitPyramidSHA3(n int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		data := testutil.RandomReader(i, n)
		putGetter := newTestHasherStore(&FakeChunkStore{}, SHA3Hash)

		ctx := context.Background()
		_, wait, err := PyramidSplit(ctx, data, putGetter, putGetter)
		if err != nil {
			t.Fatalf(err.Error())
		}
		err = wait(ctx)
		if err != nil {
			t.Fatalf(err.Error())
		}
	}
}

func benchmarkSplitAppendPyramid(n, m int, t *testing.B) {
	t.ReportAllocs()
	for i := 0; i < t.N; i++ {
		data := testutil.RandomReader(i, n)
		data1 := testutil.RandomReader(t.N+i, m)

		store := NewMapChunkStore()
		putGetter := newTestHasherStore(store, SHA3Hash)

		ctx := context.Background()
		key, wait, err := PyramidSplit(ctx, data, putGetter, putGetter)
		if err != nil {
			t.Fatalf(err.Error())
		}
		err = wait(ctx)
		if err != nil {
			t.Fatalf(err.Error())
		}

		putGetter = newTestHasherStore(store, SHA3Hash)
		_, wait, err = PyramidAppend(ctx, key, data1, putGetter, putGetter)
		if err != nil {
			t.Fatalf(err.Error())
		}
		err = wait(ctx)
		if err != nil {
			t.Fatalf(err.Error())
		}
	}
}

func BenchmarkSplitJoin_2(t *testing.B) { benchmarkSplitJoin(100, t) }
func BenchmarkSplitJoin_3(t *testing.B) { benchmarkSplitJoin(1000, t) }
func BenchmarkSplitJoin_4(t *testing.B) { benchmarkSplitJoin(10000, t) }
func BenchmarkSplitJoin_5(t *testing.B) { benchmarkSplitJoin(100000, t) }
func BenchmarkSplitJoin_6(t *testing.B) { benchmarkSplitJoin(1000000, t) }
func BenchmarkSplitJoin_7(t *testing.B) { benchmarkSplitJoin(10000000, t) }

// func BenchmarkSplitJoin_8(t *testing.B) { benchmarkJoin(100000000, t) }

func BenchmarkSplitTreeSHA3_2(t *testing.B)  { benchmarkSplitTreeSHA3(100, t) }
func BenchmarkSplitTreeSHA3_2h(t *testing.B) { benchmarkSplitTreeSHA3(500, t) }
func BenchmarkSplitTreeSHA3_3(t *testing.B)  { benchmarkSplitTreeSHA3(1000, t) }
func BenchmarkSplitTreeSHA3_3h(t *testing.B) { benchmarkSplitTreeSHA3(5000, t) }
func BenchmarkSplitTreeSHA3_4(t *testing.B)  { benchmarkSplitTreeSHA3(10000, t) }
func BenchmarkSplitTreeSHA3_4h(t *testing.B) { benchmarkSplitTreeSHA3(50000, t) }
func BenchmarkSplitTreeSHA3_5(t *testing.B)  { benchmarkSplitTreeSHA3(100000, t) }
func BenchmarkSplitTreeSHA3_6(t *testing.B)  { benchmarkSplitTreeSHA3(1000000, t) }
func BenchmarkSplitTreeSHA3_7(t *testing.B)  { benchmarkSplitTreeSHA3(10000000, t) }

// func BenchmarkSplitTreeSHA3_8(t *testing.B)  { benchmarkSplitTreeSHA3(100000000, t) }

func BenchmarkSplitTreeBMT_2(t *testing.B)  { benchmarkSplitTreeBMT(100, t) }
func BenchmarkSplitTreeBMT_2h(t *testing.B) { benchmarkSplitTreeBMT(500, t) }
func BenchmarkSplitTreeBMT_3(t *testing.B)  { benchmarkSplitTreeBMT(1000, t) }
func BenchmarkSplitTreeBMT_3h(t *testing.B) { benchmarkSplitTreeBMT(5000, t) }
func BenchmarkSplitTreeBMT_4(t *testing.B)  { benchmarkSplitTreeBMT(10000, t) }
func BenchmarkSplitTreeBMT_4h(t *testing.B) { benchmarkSplitTreeBMT(50000, t) }
func BenchmarkSplitTreeBMT_5(t *testing.B)  { benchmarkSplitTreeBMT(100000, t) }
func BenchmarkSplitTreeBMT_6(t *testing.B)  { benchmarkSplitTreeBMT(1000000, t) }
func BenchmarkSplitTreeBMT_7(t *testing.B)  { benchmarkSplitTreeBMT(10000000, t) }

// func BenchmarkSplitTreeBMT_8(t *testing.B)  { benchmarkSplitTreeBMT(100000000, t) }

func BenchmarkSplitPyramidSHA3_2(t *testing.B)  { benchmarkSplitPyramidSHA3(100, t) }
func BenchmarkSplitPyramidSHA3_2h(t *testing.B) { benchmarkSplitPyramidSHA3(500, t) }
func BenchmarkSplitPyramidSHA3_3(t *testing.B)  { benchmarkSplitPyramidSHA3(1000, t) }
func BenchmarkSplitPyramidSHA3_3h(t *testing.B) { benchmarkSplitPyramidSHA3(5000, t) }
func BenchmarkSplitPyramidSHA3_4(t *testing.B)  { benchmarkSplitPyramidSHA3(10000, t) }
func BenchmarkSplitPyramidSHA3_4h(t *testing.B) { benchmarkSplitPyramidSHA3(50000, t) }
func BenchmarkSplitPyramidSHA3_5(t *testing.B)  { benchmarkSplitPyramidSHA3(100000, t) }
func BenchmarkSplitPyramidSHA3_6(t *testing.B)  { benchmarkSplitPyramidSHA3(1000000, t) }
func BenchmarkSplitPyramidSHA3_7(t *testing.B)  { benchmarkSplitPyramidSHA3(10000000, t) }

// func BenchmarkSplitPyramidSHA3_8(t *testing.B)  { benchmarkSplitPyramidSHA3(100000000, t) }

func BenchmarkSplitPyramidBMT_2(t *testing.B)  { benchmarkSplitPyramidBMT(100, t) }
func BenchmarkSplitPyramidBMT_2h(t *testing.B) { benchmarkSplitPyramidBMT(500, t) }
func BenchmarkSplitPyramidBMT_3(t *testing.B)  { benchmarkSplitPyramidBMT(1000, t) }
func BenchmarkSplitPyramidBMT_3h(t *testing.B) { benchmarkSplitPyramidBMT(5000, t) }
func BenchmarkSplitPyramidBMT_4(t *testing.B)  { benchmarkSplitPyramidBMT(10000, t) }
func BenchmarkSplitPyramidBMT_4h(t *testing.B) { benchmarkSplitPyramidBMT(50000, t) }
func BenchmarkSplitPyramidBMT_5(t *testing.B)  { benchmarkSplitPyramidBMT(100000, t) }
func BenchmarkSplitPyramidBMT_6(t *testing.B)  { benchmarkSplitPyramidBMT(1000000, t) }
func BenchmarkSplitPyramidBMT_7(t *testing.B)  { benchmarkSplitPyramidBMT(10000000, t) }

// func BenchmarkSplitPyramidBMT_8(t *testing.B)  { benchmarkSplitPyramidBMT(100000000, t) }

func BenchmarkSplitAppendPyramid_2(t *testing.B)  { benchmarkSplitAppendPyramid(100, 1000, t) }
func BenchmarkSplitAppendPyramid_2h(t *testing.B) { benchmarkSplitAppendPyramid(500, 1000, t) }
func BenchmarkSplitAppendPyramid_3(t *testing.B)  { benchmarkSplitAppendPyramid(1000, 1000, t) }
func BenchmarkSplitAppendPyramid_4(t *testing.B)  { benchmarkSplitAppendPyramid(10000, 1000, t) }
func BenchmarkSplitAppendPyramid_4h(t *testing.B) { benchmarkSplitAppendPyramid(50000, 1000, t) }
func BenchmarkSplitAppendPyramid_5(t *testing.B)  { benchmarkSplitAppendPyramid(1000000, 1000, t) }
func BenchmarkSplitAppendPyramid_6(t *testing.B)  { benchmarkSplitAppendPyramid(1000000, 1000, t) }
func BenchmarkSplitAppendPyramid_7(t *testing.B)  { benchmarkSplitAppendPyramid(10000000, 1000, t) }

// func BenchmarkAppendPyramid_8(t *testing.B)  { benchmarkAppendPyramid(100000000, 1000, t) }

// go test -timeout 20m -cpu 4 -bench=./swarm/storage -run no
// If you dont add the timeout argument above .. the benchmark will timeout and dump
