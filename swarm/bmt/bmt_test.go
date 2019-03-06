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
	"encoding/binary"
	"fmt"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/swarm/testutil"
	"golang.org/x/crypto/sha3"
)

// the actual data length generated (could be longer than max datalength of the BMT)
const BufferSize = 4128

const (
	// segmentCount is the maximum number of segments of the underlying chunk
	// Should be equal to max-chunk-data-size / hash-size
	// Currently set to 128 == 4096 (default chunk size) / 32 (sha3.keccak256 size)
	segmentCount = 128
)

var counts = []int{1, 2, 3, 4, 5, 8, 9, 15, 16, 17, 32, 37, 42, 53, 63, 64, 65, 111, 127, 128}

// calculates the Keccak256 SHA3 hash of the data
func sha3hash(data ...[]byte) []byte {
	h := sha3.NewLegacyKeccak256()
	return doSum(h, nil, data...)
}

// TestRefHasher tests that the RefHasher computes the expected BMT hash for
// some small data lengths
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
	for i, x := range tests {
		for segmentCount := x.from; segmentCount <= x.to; segmentCount++ {
			for length := 1; length <= segmentCount*32; length++ {
				t.Run(fmt.Sprintf("%d_segments_%d_bytes", segmentCount, length), func(t *testing.T) {
					data := testutil.RandomBytes(i, length)
					expected := x.expected(data)
					actual := NewRefHasher(sha3.NewLegacyKeccak256, segmentCount).Hash(data)
					if !bytes.Equal(actual, expected) {
						t.Fatalf("expected %x, got %x", expected, actual)
					}
				})
			}
		}
	}
}

// tests if hasher responds with correct hash comparing the reference implementation return value
func TestHasherEmptyData(t *testing.T) {
	hasher := sha3.NewLegacyKeccak256
	var data []byte
	for _, count := range counts {
		t.Run(fmt.Sprintf("%d_segments", count), func(t *testing.T) {
			pool := NewTreePool(hasher, count, PoolSize)
			defer pool.Drain(0)
			bmt := New(pool)
			rbmt := NewRefHasher(hasher, count)
			refHash := rbmt.Hash(data)
			expHash := syncHash(bmt, nil, data)
			if !bytes.Equal(expHash, refHash) {
				t.Fatalf("hash mismatch with reference. expected %x, got %x", refHash, expHash)
			}
		})
	}
}

// tests sequential write with entire max size written in one go
func TestSyncHasherCorrectness(t *testing.T) {
	data := testutil.RandomBytes(1, BufferSize)
	hasher := sha3.NewLegacyKeccak256
	size := hasher().Size()

	var err error
	for _, count := range counts {
		t.Run(fmt.Sprintf("segments_%v", count), func(t *testing.T) {
			max := count * size
			var incr int
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

// tests order-neutral concurrent writes with entire max size written in one go
func TestAsyncCorrectness(t *testing.T) {
	data := testutil.RandomBytes(1, BufferSize)
	hasher := sha3.NewLegacyKeccak256
	size := hasher().Size()
	whs := []whenHash{first, last, random}

	for _, double := range []bool{false, true} {
		for _, wh := range whs {
			for _, count := range counts {
				t.Run(fmt.Sprintf("double_%v_hash_when_%v_segments_%v", double, wh, count), func(t *testing.T) {
					max := count * size
					var incr int
					capacity := 1
					pool := NewTreePool(hasher, count, capacity)
					defer pool.Drain(0)
					for n := 1; n <= max; n += incr {
						incr = 1 + rand.Intn(5)
						bmt := New(pool)
						d := data[:n]
						rbmt := NewRefHasher(hasher, count)
						exp := rbmt.Hash(d)
						got := syncHash(bmt, nil, d)
						if !bytes.Equal(got, exp) {
							t.Fatalf("wrong sync hash for datalength %v: expected %x (ref), got %x", n, exp, got)
						}
						sw := bmt.NewAsyncWriter(double)
						got = asyncHashRandom(sw, nil, d, wh)
						if !bytes.Equal(got, exp) {
							t.Fatalf("wrong async hash for datalength %v: expected %x, got %x", n, exp, got)
						}
					}
				})
			}
		}
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

// tests if bmt reuse is not corrupting result
func testHasherReuse(poolsize int, t *testing.T) {
	hasher := sha3.NewLegacyKeccak256
	pool := NewTreePool(hasher, segmentCount, poolsize)
	defer pool.Drain(0)
	bmt := New(pool)

	for i := 0; i < 100; i++ {
		data := testutil.RandomBytes(1, BufferSize)
		n := rand.Intn(bmt.Size())
		err := testHasherCorrectness(bmt, hasher, data, n, segmentCount)
		if err != nil {
			t.Fatal(err)
		}
	}
}

// Tests if pool can be cleanly reused even in concurrent use by several hasher
func TestBMTConcurrentUse(t *testing.T) {
	hasher := sha3.NewLegacyKeccak256
	pool := NewTreePool(hasher, segmentCount, PoolSize)
	defer pool.Drain(0)
	cycles := 100
	errc := make(chan error)

	for i := 0; i < cycles; i++ {
		go func() {
			bmt := New(pool)
			data := testutil.RandomBytes(1, BufferSize)
			n := rand.Intn(bmt.Size())
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
func TestBMTWriterBuffers(t *testing.T) {
	hasher := sha3.NewLegacyKeccak256

	for _, count := range counts {
		t.Run(fmt.Sprintf("%d_segments", count), func(t *testing.T) {
			errc := make(chan error)
			pool := NewTreePool(hasher, count, PoolSize)
			defer pool.Drain(0)
			n := count * 32
			bmt := New(pool)
			data := testutil.RandomBytes(1, n)
			rbmt := NewRefHasher(hasher, count)
			refHash := rbmt.Hash(data)
			expHash := syncHash(bmt, nil, data)
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
	got := syncHash(bmt, span, data)
	if !bytes.Equal(got, exp) {
		return fmt.Errorf("wrong hash: expected %x, got %x", exp, got)
	}
	return err
}

//
func BenchmarkBMT(t *testing.B) {
	for size := 4096; size >= 128; size /= 2 {
		t.Run(fmt.Sprintf("%v_size_%v", "SHA3", size), func(t *testing.B) {
			benchmarkSHA3(t, size)
		})
		t.Run(fmt.Sprintf("%v_size_%v", "Baseline", size), func(t *testing.B) {
			benchmarkBMTBaseline(t, size)
		})
		t.Run(fmt.Sprintf("%v_size_%v", "REF", size), func(t *testing.B) {
			benchmarkRefHasher(t, size)
		})
		t.Run(fmt.Sprintf("%v_size_%v", "BMT", size), func(t *testing.B) {
			benchmarkBMT(t, size)
		})
	}
}

type whenHash = int

const (
	first whenHash = iota
	last
	random
)

func BenchmarkBMTAsync(t *testing.B) {
	whs := []whenHash{first, last, random}
	for size := 4096; size >= 128; size /= 2 {
		for _, wh := range whs {
			for _, double := range []bool{false, true} {
				t.Run(fmt.Sprintf("double_%v_hash_when_%v_size_%v", double, wh, size), func(t *testing.B) {
					benchmarkBMTAsync(t, size, wh, double)
				})
			}
		}
	}
}

func BenchmarkPool(t *testing.B) {
	caps := []int{1, PoolSize}
	for size := 4096; size >= 128; size /= 2 {
		for _, c := range caps {
			t.Run(fmt.Sprintf("poolsize_%v_size_%v", c, size), func(t *testing.B) {
				benchmarkPool(t, c, size)
			})
		}
	}
}

// benchmarks simple sha3 hash on chunks
func benchmarkSHA3(t *testing.B, n int) {
	data := testutil.RandomBytes(1, n)
	hasher := sha3.NewLegacyKeccak256
	h := hasher()

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		doSum(h, nil, data)
	}
}

// benchmarks the minimum hashing time for a balanced (for simplicity) BMT
// by doing count/segmentsize parallel hashings of 2*segmentsize bytes
// doing it on n PoolSize each reusing the base hasher
// the premise is that this is the minimum computation needed for a BMT
// therefore this serves as a theoretical optimum for concurrent implementations
func benchmarkBMTBaseline(t *testing.B, n int) {
	hasher := sha3.NewLegacyKeccak256
	hashSize := hasher().Size()
	data := testutil.RandomBytes(1, hashSize)

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
					doSum(h, nil, data)
				}
			}()
		}
		wg.Wait()
	}
}

// benchmarks BMT Hasher
func benchmarkBMT(t *testing.B, n int) {
	data := testutil.RandomBytes(1, n)
	hasher := sha3.NewLegacyKeccak256
	pool := NewTreePool(hasher, segmentCount, PoolSize)
	bmt := New(pool)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		syncHash(bmt, nil, data)
	}
}

// benchmarks BMT hasher with asynchronous concurrent segment/section writes
func benchmarkBMTAsync(t *testing.B, n int, wh whenHash, double bool) {
	data := testutil.RandomBytes(1, n)
	hasher := sha3.NewLegacyKeccak256
	pool := NewTreePool(hasher, segmentCount, PoolSize)
	bmt := New(pool).NewAsyncWriter(double)
	idxs, segments := splitAndShuffle(bmt.SectionSize(), data)
	rand.Shuffle(len(idxs), func(i int, j int) {
		idxs[i], idxs[j] = idxs[j], idxs[i]
	})

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		asyncHash(bmt, nil, n, wh, idxs, segments)
	}
}

// benchmarks 100 concurrent bmt hashes with pool capacity
func benchmarkPool(t *testing.B, poolsize, n int) {
	data := testutil.RandomBytes(1, n)
	hasher := sha3.NewLegacyKeccak256
	pool := NewTreePool(hasher, segmentCount, poolsize)
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
				syncHash(bmt, nil, data)
			}()
		}
		wg.Wait()
	}
}

// benchmarks the reference hasher
func benchmarkRefHasher(t *testing.B, n int) {
	data := testutil.RandomBytes(1, n)
	hasher := sha3.NewLegacyKeccak256
	rbmt := NewRefHasher(hasher, 128)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		rbmt.Hash(data)
	}
}

// Hash hashes the data and the span using the bmt hasher
func syncHash(h *Hasher, span, data []byte) []byte {
	h.ResetWithLength(span)
	h.Write(data)
	return h.Sum(nil)
}

func splitAndShuffle(secsize int, data []byte) (idxs []int, segments [][]byte) {
	l := len(data)
	n := l / secsize
	if l%secsize > 0 {
		n++
	}
	for i := 0; i < n; i++ {
		idxs = append(idxs, i)
		end := (i + 1) * secsize
		if end > l {
			end = l
		}
		section := data[i*secsize : end]
		segments = append(segments, section)
	}
	rand.Shuffle(n, func(i int, j int) {
		idxs[i], idxs[j] = idxs[j], idxs[i]
	})
	return idxs, segments
}

// splits the input data performs a random shuffle to mock async section writes
func asyncHashRandom(bmt SectionWriter, span []byte, data []byte, wh whenHash) (s []byte) {
	idxs, segments := splitAndShuffle(bmt.SectionSize(), data)
	return asyncHash(bmt, span, len(data), wh, idxs, segments)
}

// mock for async section writes for BMT SectionWriter
// requires a permutation (a random shuffle) of list of all indexes of segments
// and writes them in order to the appropriate section
// the Sum function is called according to the wh parameter (first, last, random [relative to segment writes])
func asyncHash(bmt SectionWriter, span []byte, l int, wh whenHash, idxs []int, segments [][]byte) (s []byte) {
	bmt.Reset()
	if l == 0 {
		return bmt.Sum(nil, l, span)
	}
	c := make(chan []byte, 1)
	hashf := func() {
		c <- bmt.Sum(nil, l, span)
	}
	maxsize := len(idxs)
	var r int
	if wh == random {
		r = rand.Intn(maxsize)
	}
	for i, idx := range idxs {
		bmt.Write(idx, segments[idx])
		if (wh == first || wh == random) && i == r {
			go hashf()
		}
	}
	if wh == last {
		return bmt.Sum(nil, l, span)
	}
	return <-c
}
