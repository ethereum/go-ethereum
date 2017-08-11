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
	"fmt"
	"hash"
	"io"
	"math/rand"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/crypto/sha3"
)

const (
	maxproccnt = 8
)

func testDataReader(l int) (r io.Reader) {
	return io.LimitReader(crand.Reader, int64(l))
}

func TestHasherCorrectness(t *testing.T) {
	err := testHasher(testBaseHasher)
	if err != nil {
		t.Fatal(err)
	}
}

func testHasher(f func(BaseHasher, []byte, int, int) error) error {
	tdata := testDataReader(4128)
	data := make([]byte, 4128)
	tdata.Read(data)
	hasher := sha3.NewKeccak256
	size := hasher().Size()
	counts := []int{1, 2, 3, 4, 5, 8, 16, 32, 64, 128}

	var err error
	for _, count := range counts {
		max := count * size
		incr := max/size + 1
		for n := 0; n <= max+incr; n += incr {
			err = f(hasher, data, n, count)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func TestHasherReuseWithoutRelease(t *testing.T) {
	testHasherReuse(1, t)
}

func TestHasherReuseWithRelease(t *testing.T) {
	testHasherReuse(maxproccnt, t)
}

func testHasherReuse(i int, t *testing.T) {
	hasher := sha3.NewKeccak256
	pool := NewTreePool(hasher, 128, i)
	defer pool.Drain(0)
	bmt := New(pool)

	for i := 0; i < 500; i++ {
		n := rand.Intn(4096)
		tdata := testDataReader(n)
		data := make([]byte, n)
		tdata.Read(data)

		err := testHasherCorrectness(bmt, hasher, data, n, 128)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestHasherConcurrency(t *testing.T) {
	hasher := sha3.NewKeccak256
	pool := NewTreePool(hasher, 128, maxproccnt)
	defer pool.Drain(0)
	wg := sync.WaitGroup{}
	cycles := 100
	wg.Add(maxproccnt * cycles)
	errc := make(chan error)

	for p := 0; p < maxproccnt; p++ {
		for i := 0; i < cycles; i++ {
			go func() {
				bmt := New(pool)
				n := rand.Intn(4096)
				tdata := testDataReader(n)
				data := make([]byte, n)
				tdata.Read(data)
				err := testHasherCorrectness(bmt, hasher, data, n, 128)
				wg.Done()
				if err != nil {
					errc <- err
				}
			}()
		}
	}
	go func() {
		wg.Wait()
		close(errc)
	}()
	var err error
	select {
	case <-time.NewTimer(5 * time.Second).C:
		err = fmt.Errorf("timed out")
	case err = <-errc:
	}
	if err != nil {
		t.Fatal(err)
	}
}

func testBaseHasher(hasher BaseHasher, d []byte, n, count int) error {
	pool := NewTreePool(hasher, count, 1)
	defer pool.Drain(0)
	bmt := New(pool)
	return testHasherCorrectness(bmt, hasher, d, n, count)
}

func testHasherCorrectness(bmt hash.Hash, hasher BaseHasher, d []byte, n, count int) (err error) {
	data := d[:n]
	rbmt := NewRefHasher(hasher, count)
	exp := rbmt.Hash(data)
	timeout := time.NewTimer(time.Second)
	c := make(chan error)

	go func() {
		bmt.Reset()
		bmt.Write(data)
		got := bmt.Sum(nil)
		if !bytes.Equal(got, exp) {
			c <- fmt.Errorf("wrong hash: expected %x, got %x", exp, got)
		}
		close(c)
	}()
	select {
	case <-timeout.C:
		err = fmt.Errorf("BMT hash calculation timed out")
	case err = <-c:
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

func BenchmarkHasher_4k(t *testing.B)   { benchmarkHasher(4096, t) }
func BenchmarkHasher_2k(t *testing.B)   { benchmarkHasher(4096/2, t) }
func BenchmarkHasher_1k(t *testing.B)   { benchmarkHasher(4096/4, t) }
func BenchmarkHasher_512b(t *testing.B) { benchmarkHasher(4096/8, t) }
func BenchmarkHasher_256b(t *testing.B) { benchmarkHasher(4096/16, t) }
func BenchmarkHasher_128b(t *testing.B) { benchmarkHasher(4096/32, t) }

func BenchmarkHasherNoReuse_4k(t *testing.B)   { benchmarkHasherReuse(1, 4096, t) }
func BenchmarkHasherNoReuse_2k(t *testing.B)   { benchmarkHasherReuse(1, 4096/2, t) }
func BenchmarkHasherNoReuse_1k(t *testing.B)   { benchmarkHasherReuse(1, 4096/4, t) }
func BenchmarkHasherNoReuse_512b(t *testing.B) { benchmarkHasherReuse(1, 4096/8, t) }
func BenchmarkHasherNoReuse_256b(t *testing.B) { benchmarkHasherReuse(1, 4096/16, t) }
func BenchmarkHasherNoReuse_128b(t *testing.B) { benchmarkHasherReuse(1, 4096/32, t) }

func BenchmarkHasherReuse_4k(t *testing.B)   { benchmarkHasherReuse(16, 4096, t) }
func BenchmarkHasherReuse_2k(t *testing.B)   { benchmarkHasherReuse(16, 4096/2, t) }
func BenchmarkHasherReuse_1k(t *testing.B)   { benchmarkHasherReuse(16, 4096/4, t) }
func BenchmarkHasherReuse_512b(t *testing.B) { benchmarkHasherReuse(16, 4096/8, t) }
func BenchmarkHasherReuse_256b(t *testing.B) { benchmarkHasherReuse(16, 4096/16, t) }
func BenchmarkHasherReuse_128b(t *testing.B) { benchmarkHasherReuse(16, 4096/32, t) }

// benchmarks the minimum hashing time for a balanced (for simplicity) BMT
// by doing count/segmentsize parallel hashings of 2*segmentsize bytes
// doing it on n maxproccnt each reusing the base hasher
// the premise is that this is the minimum computation needed for a BMT
// therefore this serves as a theoretical optimum for concurrent implementations
func benchmarkBMTBaseline(n int, t *testing.B) {
	tdata := testDataReader(64)
	data := make([]byte, 64)
	tdata.Read(data)
	hasher := sha3.NewKeccak256

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		count := int32((n-1)/hasher().Size() + 1)
		wg := sync.WaitGroup{}
		wg.Add(maxproccnt)
		var i int32
		for j := 0; j < maxproccnt; j++ {
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

func benchmarkHasher(n int, t *testing.B) {
	tdata := testDataReader(n)
	data := make([]byte, n)
	tdata.Read(data)

	size := 1
	hasher := sha3.NewKeccak256
	segmentCount := 128
	pool := NewTreePool(hasher, segmentCount, size)
	bmt := New(pool)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		bmt.Reset()
		bmt.Write(data)
		bmt.Sum(nil)
	}
}

func benchmarkHasherReuse(poolsize, n int, t *testing.B) {
	tdata := testDataReader(n)
	data := make([]byte, n)
	tdata.Read(data)

	hasher := sha3.NewKeccak256
	segmentCount := 128
	pool := NewTreePool(hasher, segmentCount, poolsize)
	cycles := 200

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		wg := sync.WaitGroup{}
		wg.Add(cycles)
		for j := 0; j < cycles; j++ {
			bmt := New(pool)
			go func() {
				defer wg.Done()
				bmt.Reset()
				bmt.Write(data)
				bmt.Sum(nil)
			}()
		}
		wg.Wait()
	}
}

func benchmarkSHA3(n int, t *testing.B) {
	data := make([]byte, n)
	tdata := testDataReader(n)
	tdata.Read(data)
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

func benchmarkRefHasher(n int, t *testing.B) {
	data := make([]byte, n)
	tdata := testDataReader(n)
	tdata.Read(data)
	hasher := sha3.NewKeccak256
	rbmt := NewRefHasher(hasher, 128)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		rbmt.Hash(data)
	}
}
