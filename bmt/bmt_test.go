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

func TestBMTHasherCorrectness(t *testing.T) {
	err := testHasher(testBMTHasher)
	if err != nil {
		t.Fatal(err)
	}
}

func testHasher(f func(Hasher, []byte, int, int) error) error {
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
		fmt.Println("max=", max, "incr=", incr)
		for n := 0; n <= max+incr; n += incr {
			fmt.Println("     ", "datalen=", len(data), "n=", n, "count=", count)
			err = f(hasher, data, n, count)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func TestBMTHasherReuseWithoutRelease(t *testing.T) {
	testBMTHasherReuse(t)
}

func TestBMTHasherReuseWithRelease(t *testing.T) {
	testBMTHasherReuse(t)
}

func testBMTHasherReuse(t *testing.T) {
	hasher := sha3.NewKeccak256
	pool := NewBMTreePool(hasher, 128, 1)
	defer pool.Drain(0)
	bmt := NewBMTHasher(pool)

	for i := 0; i < 500; i++ {
		n := rand.Intn(4096)
		tdata := testDataReader(n)
		data := make([]byte, n)
		tdata.Read(data)

		err := testBMTHasherCorrectness(bmt, hasher, data, n, 128)
		if err != nil {
			t.Fatal(err)
		}
	}
}

func TestBMTHasherConcurrency(t *testing.T) {
	hasher := sha3.NewKeccak256
	pool := NewBMTreePool(hasher, 128, maxproccnt)
	defer pool.Drain(0)
	wg := sync.WaitGroup{}
	cycles := 100
	wg.Add(maxproccnt * cycles)
	errc := make(chan error)

	for p := 0; p < maxproccnt; p++ {
		bmt := NewBMTHasher(pool)
		go func() {
			for i := 0; i < cycles; i++ {
				n := rand.Intn(4096)
				tdata := testDataReader(n)
				data := make([]byte, n)
				tdata.Read(data)
				err := testBMTHasherCorrectness(bmt, hasher, data, n, 128)
				wg.Done()
				if err != nil {
					errc <- err
				}
			}
		}()
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
func testBMTHasher(hasher Hasher, d []byte, n, count int) error {
	pool := NewBMTreePool(hasher, count, 1)
	defer pool.Drain(0)
	bmt := NewBMTHasher(pool)
	return testBMTHasherCorrectness(bmt, hasher, d, n, count)
}

func testBMTHasherCorrectness(bmt hash.Hash, hasher Hasher, d []byte, n, count int) (err error) {
	data := d[:n]
	rbmt := NewRBMTHasher(hasher, count)
	exp := rbmt.Hash(data)
	timeout := time.NewTimer(time.Second)
	c := make(chan error)
	go func() {
		bmt.Reset()
		bmt.Write(data)
		got := bmt.Sum(nil)
		fmt.Printf("     result %x\n", got)
		if !bytes.Equal(got, exp) {
			var t string
			node, ok := bmt.(*BMTHasher)
			if ok && node.bmt != nil {
				d := depth(n)
				t = node.bmt.Draw(got, d)
			}
			c <- fmt.Errorf("wrong hash. expected %x, got %x\n%s\n", exp, got, t)
		}

		close(c)

	}()
	select {
	case _, ok := <-timeout.C:
		if ok {
			err = fmt.Errorf("BMT hash calculation timed out")
		}
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

func BenchmarkRBMTHasher_4k(t *testing.B)   { benchmarkRBMTHasher(4096, t) }
func BenchmarkRBMTHasher_2k(t *testing.B)   { benchmarkRBMTHasher(4096/2, t) }
func BenchmarkRBMTHasher_1k(t *testing.B)   { benchmarkRBMTHasher(4096/4, t) }
func BenchmarkRBMTHasher_512b(t *testing.B) { benchmarkRBMTHasher(4096/8, t) }
func BenchmarkRBMTHasher_256b(t *testing.B) { benchmarkRBMTHasher(4096/16, t) }
func BenchmarkRBMTHasher_128b(t *testing.B) { benchmarkRBMTHasher(4096/32, t) }

func BenchmarkBMTHasher_4k(t *testing.B)   { benchmarkBMTHasher(4096, t) }
func BenchmarkBMTHasher_2k(t *testing.B)   { benchmarkBMTHasher(4096/2, t) }
func BenchmarkBMTHasher_1k(t *testing.B)   { benchmarkBMTHasher(4096/4, t) }
func BenchmarkBMTHasher_512b(t *testing.B) { benchmarkBMTHasher(4096/8, t) }
func BenchmarkBMTHasher_256b(t *testing.B) { benchmarkBMTHasher(4096/16, t) }
func BenchmarkBMTHasher_128b(t *testing.B) { benchmarkBMTHasher(4096/32, t) }

func BenchmarkBMTHasherReuse_4k(t *testing.B)   { benchmarkBMTHasherReuse(4096, t) }
func BenchmarkBMTHasherReuse_2k(t *testing.B)   { benchmarkBMTHasherReuse(4096/2, t) }
func BenchmarkBMTHasherReuse_1k(t *testing.B)   { benchmarkBMTHasherReuse(4096/4, t) }
func BenchmarkBMTHasherReuse_512b(t *testing.B) { benchmarkBMTHasherReuse(4096/8, t) }
func BenchmarkBMTHasherReuse_256b(t *testing.B) { benchmarkBMTHasherReuse(4096/16, t) }
func BenchmarkBMTHasherReuse_128b(t *testing.B) { benchmarkBMTHasherReuse(4096/32, t) }

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

func benchmarkBMTHasher(n int, t *testing.B) {
	tdata := testDataReader(n)
	data := make([]byte, n)
	tdata.Read(data)

	size := 1
	hasher := sha3.NewKeccak256
	segmentCount := 128
	pool := NewBMTreePool(hasher, segmentCount, size)
	bmt := NewBMTHasher(pool)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		bmt.Reset()
		bmt.Write(data)
		bmt.Sum(nil)
	}
}

func benchmarkBMTHasherReuse(n int, t *testing.B) {
	tdata := testDataReader(n)
	data := make([]byte, n)
	tdata.Read(data)

	size := 2
	hasher := sha3.NewKeccak256
	segmentCount := 128
	pool := NewBMTreePool(hasher, segmentCount, size)
	bmt := NewBMTHasher(pool)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		bmt.Reset()
		bmt.Write(data)
		bmt.Sum(nil)
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

func benchmarkRBMTHasher(n int, t *testing.B) {
	data := make([]byte, n)
	tdata := testDataReader(n)
	tdata.Read(data)
	hasher := sha3.NewKeccak256
	rbmt := NewRBMTHasher(hasher, 128)

	t.ReportAllocs()
	t.ResetTimer()
	for i := 0; i < t.N; i++ {
		rbmt.Hash(data)
	}
}
