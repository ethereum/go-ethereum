// Copyright 2018 The go-ethereum Authors
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

package memorydb

import (
	"encoding/binary"
	"testing"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/ethdb/dbtest"
)

func TestMemoryDB(t *testing.T) {
	t.Run("DatabaseSuite", func(t *testing.T) {
		dbtest.TestDatabaseSuite(t, func() ethdb.KeyValueStore {
			return New()
		})
	})
}

// BenchmarkBatchAllocs measures the time/allocs for storing 120 kB of data
func BenchmarkBatchAllocs(b *testing.B) {
	b.ReportAllocs()
	var key = make([]byte, 20)
	var val = make([]byte, 100)
	// 120 * 1_000 -> 120_000 == 120kB
	for i := 0; i < b.N; i++ {
		batch := New().NewBatch()
		for j := uint64(0); j < 1000; j++ {
			binary.BigEndian.PutUint64(key, j)
			binary.BigEndian.PutUint64(val, j)
			batch.Put(key, val)
		}
		batch.Write()
	}
}
