// Copyright 2017 The go-ethereum Authors
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

package bmt

import (
	"bytes"
	crand "crypto/rand"
	"encoding/binary"
	"fmt"
	"io"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/sha3"
)

// the actual data length generated (could be longer than max datalength of the BMT)
const BufferSize = 4128

var counts = []int{1, 2, 3, 4, 5, 8, 9, 15, 16, 17, 32, 37, 42, 53, 63, 64, 65, 111, 127, 128}

// calculates the Keccak256 SHA3 hash of the data
func sha3hash(data ...[]byte) []byte {
	h := sha3.NewKeccak256()
	return doHash(h, nil, data...)
}

// TestRefHasher tests that the RefHasher computes the expected BMT hash for
// all data lengths between 0 and 256 bytes
func TestRefHasher(t *testing.T) {

	// the test struct is used to specify the expected BMT hash for
	// segment counts between from and to and lengths from 1 to datalength
	type test struct {
		from     int
		to       int
		expected func([]byte) []byte
	}

	var tests []*test
	// all lengths in [0,64] should be:
	//
	//   sha3hash(data)
	//
	tests = append(tests, &test{
		from: 1,
		to:   2,
		expected: func(d []byte) []byte {
			data := make([]byte, 64)
			copy(data, d)
			return sha3hash(data)
		},
	})

	// all lengths in [3,4] should be:
	//
	//   sha3hash(
	//     sha3hash(data[:64])
	//     sha3hash(data[64:])
	//   )
	//
	tests = append(tests, &test{
		from: 3,
		to:   4,
		expected: func(d []byte) []byte {
			data := make([]byte, 128)
			copy(data, d)
			return sha3hash(sha3hash(data[:64]), sha3hash(data[64:]))
		},
	})

	// all segmentCounts in [5,8] should be:
	//
	//   sha3hash(
	//     sha3hash(
	//       sha3hash(data[:64])
	//       sha3hash(data[64:128])
	//     )
	//     sha3hash(
	//       sha3hash(data[128:192])
	//       sha3hash(data[192:])
	//     )
	//   )
	//
	tests = append(tests, &test{
		from: 5,
		to:   8,
		expected: func(d []byte) []byte {
			data := make([]byte, 256)
			copy(data, d)
			return sha3hash(sha3hash(sha3hash(data[:64]), sha3hash(data[64:128])), sha3hash(sha3hash(data[128:192]), sha3hash(data[192:])))
		},
	})

	// run the tests
	for _, x := range tests {
		for segmentCount := x.from; segmentCount <= x.to; segmentCount++ {
			for length := 1; length <= segmentCount*32; length++ {
				t.Run(fmt.Sprintf("%d_segments_%d_bytes", segmentCount, length), func(t *testing.T) {
					data := make([]byte, length)
					if _, err := io.ReadFull(crand.Reader, data); err != nil && err != io.EOF {
						t.Fatal(err)
					}
					expected := x.expected(data)
					actual := NewRefHasher(sha3.NewKeccak256, segmentCount).Hash(data)
					if !bytes.Equal(actual, expected) {
						t.Fatalf("expected %x, got %x", expected, actual)
					}
				})
			}
		}
	}
}

// tests if hasher responds with correct hash
func TestHasherEmptyData(t *testing.T) {
	hasher := sha3.NewKeccak256
	var data []byte
	for _, count := range counts {
		t.Run(fmt.Sprintf("%d_segments", count), func(t *testing.T) {
			pool := NewTreePool(hasher, count, PoolSize)
			defer pool.Drain(0)
			bmt := New(pool)
			rbmt := NewRefHasher(hasher, count)
			refHash := rbmt.Hash(data)
			expHash := Hash(bmt, nil, data)
			if !bytes.Equal(expHash, refHash) {
				t.Fatalf("hash mismatch with reference. expected %x, got %x", refHash, expHash)
			}
		})
	}
}

func TestHasherCorrectness(t *testing.T) {
	data := newData(BufferSize)
	hasher := sha3.NewKeccak256
	size := hasher().Size()

	var err error
	for _, count := range counts {
		t.Run(fmt.Sprintf("segments_%v", count), func(t *testing.T) {
			max := count * size
			incr := 1
			capacity := 1
			pool := NewTreePool(hasher, count, capacity)
			defer pool.Drain(0)
			for n := 0; n <= max; n += incr {
				incr = 1 + rand.Intn(5)
				bmt := New(pool)
				err = testHasherCorrectness(bmt, hasher, data, n, count)
				if err != nil {
					t.Fatal(err)
				}
			}
		})
	}
}

// Tests that the BMT hasher can be synchronously reused with poolsizes 1 and PoolSize
func TestHasherReuse(t *testing.T) {
	t.Run(fmt.Sprintf("poolsize_%d", 1), func(t *testing.T) {
		testHasherReuse(1, t)
	})
	t.Run(fmt.Sprintf("poolsize_%d", PoolSize), func(t *testing.T) {
		testHasherReuse(PoolSize, t)
	})
}

func testHasherReuse(poolsize int, t *testing.T) {
	hasher := sha3.NewKeccak256
	pool := NewTreePool(hasher, SegmentCount, poolsize)
	defer pool.Drain(0)
	bmt := New(pool)

	for i := 0; i < 100; i++ {
		data := newData(BufferSize)
		n := rand.Intn(bmt.DataLength())
		err := testHasherCorrectness(bmt, hasher, data, n, SegmentCount)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// Tests if pool can be cleanly reused even in concurrent use
func TestBMTHasherConcurrentUse(t *testing.T) {
	hasher := sha3.NewKeccak256
	pool := NewTreePool(hasher, SegmentCount, PoolSize)
	defer pool.Drain(0)
	cycles := 100
	errc := make(chan error)

	for i := 0; i < cycles; i++ {
		go func() {
			bmt := New(pool)
			data := newData(BufferSize)
			n := rand.Intn(bmt.DataLength())
			errc <- testHasherCorrectness(bmt, hasher, data, n, 128)
		}()
	}
LOOP:
	for {
		select {
		case <-time.NewTimer(5 * time.Second).C:
			t.Fatal("timed out")
		case err := <-errc:
			if err != nil {
				t.Fatal(err)
			}
			cycles--
			if cycles == 0 {
				break LOOP
			}
		}
	}
}

// Tests BMT Hasher io.Writer interface is working correctly
// even multiple short random write buffers
func TestBMTHasherWriterBuffers(t *testing.T) {
	hasher := sha3.NewKeccak256

	for _, count := range counts {
		t.Run(fmt.Sprintf("%d_segments", count), func(t *testing.T) {
			errc := make(chan error)
			pool := NewTreePool(hasher, count, PoolSize)
			defer pool.Drain(0)
			n := count * 32
			bmt := New(pool)
			data := newData(n)
			rbmt := NewRefHasher(hasher, count)
			refHash := rbmt.Hash(data)
			expHash := Hash(bmt, nil, data)
			if !bytes.Equal(expHash, refHash) {
				t.Fatalf("hash mismatch with reference. expected %x, got %x", refHash, expHash)
			}
			attempts := 10
			f := func() error {
				bmt := New(pool)
				bmt.Reset()
				var buflen int
				for offset := 0; offset < n; offset += buflen {
					buflen = rand.Intn(n-offset) + 1
					read, err := bmt.Write(data[offset : offset+buflen])
					if err != nil {
						return err
					}
					if read != buflen {
						return fmt.Errorf("incorrect read. expected %v bytes, got %v", buflen, read)
					}
				}
				hash := bmt.Sum(nil)
				if !bytes.Equal(hash, expHash) {
					return fmt.Errorf("hash mismatch. expected %x, got %x", hash, expHash)
				}
				return nil
			}

			for j := 0; j < attempts; j++ {
				go func() {
					errc <- f()
				}()
			}
			timeout := time.NewTimer(2 * time.Second)
			for {
				select {
				case err := <-errc:
					if err != nil {
						t.Fatal(err)
					}
					attempts--
					if attempts == 0 {
						return
					}
				case <-timeout.C:
					t.Fatalf("timeout")
				}
			}
		})
	}
}

// helper function that compares reference and optimised implementations on
// correctness
func testHasherCorrectness(bmt *Hasher, hasher BaseHasherFunc, d []byte, n, count int) (err error) {
	span := make([]byte, 8)
	if len(d) < n {
		n = len(d)
	}
	binary.BigEndian.PutUint64(span, uint64(n))
	data := d[:n]
	rbmt := NewRefHasher(hasher, count)
	exp := sha3hash(span, rbmt.Hash(data))
	got := Hash(bmt, span, data)
	if !bytes.Equal(got, exp) {
		return fmt.Errorf("wrong hash: expected %x, got %x", exp, got)
	}
	return err
}

func BenchmarkSHA3_4k(t *testing.B)   { benchmarkSHA3(4096, t) }
func BenchmarkSHA3_2k(t *testing.B)   { benchmarkSHA3(4096/2, t) }
func BenchmarkSHA3_1k(t *testing.B)   { benchmarkSHA3(4096/4, t) }
func BenchmarkSHA3_512b(t *testing.B) { benchmarkSHA3(4096/8, t) }
func BenchmarkSHA3_256b(t *testing.B) { benchmarkSHA3(4096/16, t) }
func BenchmarkSHA3_128b(t *testing.B) { benchmarkSHA3(4096/32, t) }

func BenchmarkBMTBaseline_4k(t *testing.B)   { benchmarkBMTBaseline(4096, t) }
func BenchmarkBMTBaseline_2k(t *testing.B)   { benchmarkBMTBaseline(4096/2, t) }
func BenchmarkBMTBaseline_1k(t *testing.B)   { benchmarkBMTBaseline(4096/4, t) }
func BenchmarkBMTBaseline_512b(t *testing.B) { benchmarkBMTBaseline(4096/8, t) }
func BenchmarkBMTBaseline_256b(t *testing.B) { benchmarkBMTBaseline(4096/16, t) }
func BenchmarkBMTBaseline_128b(t *testing.B) { benchmarkBMTBaseline(4096/32, t) }

func BenchmarkRefHasher_4k(t *testing.B)   { benchmarkRefHasher(4096, t) }
func BenchmarkRefHasher_2k(t *testing.B)   { benchmarkRefHasher(4096/2, t) }
func BenchmarkRefHasher_1k(t *testing.B)   { benchmarkRefHasher(4096/4, t) }
func BenchmarkRefHasher_512b(t *testing.B) { benchmarkRefHasher(4096/8, t) }
func BenchmarkRefHasher_256b(t *testing.B) { benchmarkRefHasher(4096/16, t) }
func BenchmarkRefHasher_128b(t *testing.B) { benchmarkRefHasher(4096/32, t) }

func BenchmarkBMTHasher_4k(t *testing.B)   { benchmarkBMTHasher(4096, t) }
func BenchmarkBMTHasher_2k(t *testing.B)   { benchmarkBMTHasher(4096/2, t) }
func BenchmarkBMTHasher_1k(t *testing.B)   { benchmarkBMTHasher(4096/4, t) }
func BenchmarkBMTHasher_512b(t *testing.B) { benchmarkBMTHasher(4096/8, t) }
func BenchmarkBMTHasher_256b(t *testing.B) { benchmarkBMTHasher(4096/16, t) }
func BenchmarkBMTHasher_128b(t *testing.B) { benchmarkBMTHasher(4096/32, t) }

func BenchmarkBMTHasherNoPool_4k(t *testing.B)   { benchmarkBMTHasherPool(1, 4096, t) }
func BenchmarkBMTHasherNoPool_2k(t *testing.B)   { benchmarkBMTHasherPool(1, 4096/2, t) }
func BenchmarkBMTHasherNoPool_1k(t *testing.B)   { benchmarkBMTHasherPool(1, 4096/4, t) }
func BenchmarkBMTHasherNoPool_512b(t *testing.B) { benchmarkBMTHasherPool(1, 4096/8, t) }
func BenchmarkBMTHasherNoPool_256b(t *testing.B) { benchmarkBMTHasherPool(1, 4096/16, t) }
func BenchmarkBMTHasherNoPool_128b(t *testing.B) { benchmarkBMTHasherPool(1, 4096/32, t) }

func BenchmarkBMTHasherPool_4k(t *testing.B)   { benchmarkBMTHasherPool(PoolSize, 4096, t) }
func BenchmarkBMTHasherPool_2k(t *testing.B)   { benchmarkBMTHasherPool(PoolSize, 4096/2, t) }
func BenchmarkBMTHasherPool_1k(t *testing.B)   { benchmarkBMTHasherPool(PoolSize, 4096/4, t) }
func BenchmarkBMTHasherPool_512b(t *testing.B) { benchmarkBMTHasherPool(PoolSize, 4096/8, t) }
func BenchmarkBMTHasherPool_256b(t *testing.B) { benchmarkBMTHasherPool(PoolSize, 4096/16, t) }
func BenchmarkBMTHasherPool_128b(t *testing.B) { benchmarkBMTHasherPool(PoolSize, 4096/32, t) }

// benchmarks simple sha3 hash on chunks
func benchmarkSHA3(n int, t *testing.B) {
	data := newData(n)
	hasher := sha3.NewKeccak256
	h := hasher()

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		h.Reset()
		h.Write(data)
		h.Sum(nil)
	}
}

// benchmarks the minimum hashing time for a balanced (for simplicity) BMT
// by doing count/segmentsize parallel hashings of 2*segmentsize bytes
// doing it on n PoolSize each reusing the base hasher
// the premise is that this is the minimum computation needed for a BMT
// therefore this serves as a theoretical optimum for concurrent implementations
func benchmarkBMTBaseline(n int, t *testing.B) {
	hasher := sha3.NewKeccak256
	hashSize := hasher().Size()
	data := newData(hashSize)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		count := int32((n-1)/hashSize + 1)
		wg := sync.WaitGroup{}
		wg.Add(PoolSize)
		var i int32
		for j := 0; j < PoolSize; j++ {
			go func() {
				defer wg.Done()
				h := hasher()
				for atomic.AddInt32(&i, 1) < count {
					h.Reset()
					h.Write(data)
					h.Sum(nil)
				}
			}()
		}
		wg.Wait()
	}
}

// benchmarks BMT Hasher
func benchmarkBMTHasher(n int, t *testing.B) {
	data := newData(n)
	hasher := sha3.NewKeccak256
	pool := NewTreePool(hasher, SegmentCount, PoolSize)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		bmt := New(pool)
		Hash(bmt, nil, data)
	}
}

// benchmarks 100 concurrent bmt hashes with pool capacity
func benchmarkBMTHasherPool(poolsize, n int, t *testing.B) {
	data := newData(n)
	hasher := sha3.NewKeccak256
	pool := NewTreePool(hasher, SegmentCount, poolsize)
	cycles := 100

	t.ReportAllocs()
	t.ResetTimer()
	wg := sync.WaitGroup{}
	for i := 0; i < t.N; i++ {
		wg.Add(cycles)
		for j := 0; j < cycles; j++ {
			go func() {
				defer wg.Done()
				bmt := New(pool)
				Hash(bmt, nil, data)
			}()
		}
		wg.Wait()
	}
}

// benchmarks the reference hasher
func benchmarkRefHasher(n int, t *testing.B) {
	data := newData(n)
	hasher := sha3.NewKeccak256
	rbmt := NewRefHasher(hasher, 128)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		rbmt.Hash(data)
	}
}

func newData(bufferSize int) []byte {
	data := make([]byte, bufferSize)
	_, err := io.ReadFull(crand.Reader, data)
	if err != nil {
		panic(err.Error())
	}
	return data
}
