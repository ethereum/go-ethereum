// Copyright 2021 The go-ethereum Authors
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

package rawdb

import (
	"io/ioutil"
	"math/rand"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/rlp"
)

func setupBatch(t *testing.T, name string, maxFilesize uint32, noCompression bool) (*freezerTable, func()) {
	t.Helper()
	dir, err := ioutil.TempDir("./", name)
	if err != nil {
		t.Fatal(err)
	}
	cleanup := func() {
		os.RemoveAll(dir)
	}

	f, err := newTable(dir, "tmp", metrics.NilMeter{}, metrics.NilMeter{},
		metrics.NilGauge{}, maxFilesize, noCompression)
	if err != nil {
		t.Fatal(err)
	}
	return f, cleanup
}

func BenchmarkBatchAppendBlob32(b *testing.B) {
	b.Run("raw", func(b *testing.B) { batchBench(b, 32, true) })
	b.Run("snappy", func(b *testing.B) { batchBench(b, 32, false) })
}

func BenchmarkBatchAppendBlob256(b *testing.B) {
	b.Run("raw", func(b *testing.B) { batchBench(b, 256, true) })
	b.Run("snappy", func(b *testing.B) { batchBench(b, 256, false) })
}
func BenchmarkBatchAppendBlob1024(b *testing.B) {
	b.Run("raw", func(b *testing.B) { batchBench(b, 1024, true) })
	b.Run("snappy", func(b *testing.B) { batchBench(b, 1024, false) })
}
func BenchmarkBatchAppendBlob4096(b *testing.B) {
	b.Run("raw", func(b *testing.B) { batchBench(b, 4096, true) })
	b.Run("snappy", func(b *testing.B) { batchBench(b, 4096, false) })
}

func BenchmarkBatchAppendRLP32(b *testing.B) {
	b.Run("raw", func(b *testing.B) { batchRlpBench(b, 32, true) })
	b.Run("snappy", func(b *testing.B) { batchRlpBench(b, 32, false) })
}
func BenchmarkBatchAppendRLP256(b *testing.B) {
	b.Run("raw", func(b *testing.B) { batchRlpBench(b, 256, true) })
	b.Run("snappy", func(b *testing.B) { batchRlpBench(b, 265, false) })
}
func BenchmarkBatchAppendRLP1024(b *testing.B) {
	b.Run("raw", func(b *testing.B) { batchRlpBench(b, 1024, true) })
	b.Run("snappy", func(b *testing.B) { batchRlpBench(b, 1024, false) })
}
func BenchmarkBatchAppendRLP4096(b *testing.B) {
	b.Run("raw", func(b *testing.B) { batchRlpBench(b, 4096, true) })
	b.Run("snappy", func(b *testing.B) { batchRlpBench(b, 4096, false) })
}

func batchBench(b *testing.B, nbytes int, noCompression bool) {
	dir, err := ioutil.TempDir("./", "freezer-batch-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	f, err := newTable(dir, "table", metrics.NilMeter{}, metrics.NilMeter{}, metrics.NilGauge{}, 20*1024*1024, noCompression)
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, nbytes)
	rand.Read(data)
	batch := f.newBatch()
	rlpData, _ := rlp.EncodeToBytes(data)
	b.SetBytes(int64(len(rlpData)))
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rlpData, _ := rlp.EncodeToBytes(data)
		if err := batch.Append(uint64(i), rlpData); err != nil {
			b.Fatal(err)
		}
	}
	if err := batch.commit(); err != nil {
		b.Fatal(err)
	}
	b.StopTimer() // Stop timer before deleting the files
}

func batchRlpBench(b *testing.B, nbytes int, noCompression bool) {
	dir, err := ioutil.TempDir("./", "freezer-rlp-batch-benchmark")
	if err != nil {
		b.Fatal(err)
	}
	defer os.RemoveAll(dir)

	f, err := newTable(dir, "table", metrics.NilMeter{}, metrics.NilMeter{}, metrics.NilGauge{}, 20*1024*1024, noCompression)
	if err != nil {
		b.Fatal(err)
	}
	data := make([]byte, nbytes)
	rand.Read(data)
	rlpData, _ := rlp.EncodeToBytes(data)
	b.SetBytes(int64(len(rlpData)))

	batch := f.newBatch()

	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if err := batch.Append(uint64(i), data); err != nil {
			b.Fatal(err)
		}
	}
	if err := batch.commit(); err != nil {
		b.Fatal(err)
	}
	b.StopTimer() // Stop timer before deleting the files
}
