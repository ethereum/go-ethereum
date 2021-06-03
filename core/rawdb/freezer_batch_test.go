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
	"bytes"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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

	f, err := newCustomTable(dir, "tmp", metrics.NilMeter{}, metrics.NilMeter{},
		metrics.NilGauge{}, maxFilesize, noCompression)
	if err != nil {
		t.Fatal(err)
	}
	return f, cleanup
}

func TestBatch(t *testing.T) {
	f, cleanup := setupBatch(t, "freezer", 31, true)
	defer cleanup()
	batch := f.newBatch()
	// Write 15 bytes 30 times
	for x := 0; x < 30; x++ {
		data := getChunk(15, x)
		batch.AppendRaw(uint64(x), data)
	}
	if err := batch.Commit(); err != nil {
		t.Fatal(err)
	}
	t.Log(f.dumpIndexString(0, 30))

	if got, err := f.Retrieve(29); err != nil {
		t.Fatal(err)
	} else if exp := getChunk(15, 29); !bytes.Equal(got, exp) {
		t.Fatalf("expected %x got %x", exp, got)
	}
}

func TestBatchRLP(t *testing.T) {
	f, cleanup := setupBatch(t, "freezer-rlp", 31, true)
	defer cleanup()
	batch := f.newBatch()
	// Write 15 bytes 30 times
	for x := 0; x < 30; x++ {
		data := big.NewInt(int64(x))
		data = data.Exp(data, data, nil)
		batch.Append(uint64(x), data)
	}
	if err := batch.Commit(); err != nil {
		t.Fatal(err)
	}
	t.Log(f.dumpIndexString(0, 30))

	if got, err := f.Retrieve(29); err != nil {
		t.Fatal(err)
	} else if exp := common.FromHex("921d79c05d04235e8807c34cbc36a8b48a4c0d"); !bytes.Equal(got, exp) {
		t.Fatalf("expected %x got %x", exp, got)
	}
}

func TestBatchSequence(t *testing.T) {
	f, cleanup := setupBatch(t, "freezer-batchid", 31, true)
	defer cleanup()
	batch := f.newBatch()
	if err := batch.AppendRaw(2, []byte{0}); err != nil {
		// The validity of the first ID is not checked until later
		t.Fatalf("expected no error, got %v", err)
	}
	// We've written '2', writing below that should error
	if err := batch.AppendRaw(0, []byte{0}); err == nil {
		t.Fatal("expected error")
	}
	if err := batch.AppendRaw(2, []byte{0}); err == nil {
		t.Fatal("expected error")
	}
	if err := batch.Commit(); err == nil {
		t.Fatal("expected error")
	}
	if have, want := batch.headBytes, uint32(0); have != want {
		t.Fatalf("have %d, want %d", have, want)
	}
	// Add some dummy data that we then clear out
	if err := batch.AppendRaw(1000, []byte{1, 1, 1, 1, 1}); err == nil {
		t.Fatal("expected error")
	}
	batch.Reset() // Clear it out
	if err := batch.AppendRaw(0, []byte{0}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := batch.AppendRaw(1, []byte{0}); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if err := batch.Commit(); err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
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

	f, err := newCustomTable(dir, "table", metrics.NilMeter{}, metrics.NilMeter{}, metrics.NilGauge{}, 20*1024*1024, noCompression)
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
	if err := batch.Commit(); err != nil {
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

	f, err := newCustomTable(dir, "table", metrics.NilMeter{}, metrics.NilMeter{}, metrics.NilGauge{}, 20*1024*1024, noCompression)
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
	if err := batch.Commit(); err != nil {
		b.Fatal(err)
	}
	b.StopTimer() // Stop timer before deleting the files
}
