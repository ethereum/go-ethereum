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

package rawdb

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
)

func setupRangeDB(b *testing.B, db ethdb.KeyValueStore, numEntries int) ([]byte, []byte) {
	b.Helper()
	var firstKey, lastKey []byte
	for i := 0; i < numEntries; i++ {
		key := make([]byte, 32)
		val := make([]byte, 64)
		if _, err := rand.Read(key); err != nil {
			b.Fatalf("Failed to generate random key: %v", err)
		}
		if _, err := rand.Read(val); err != nil {
			b.Fatalf("Failed to generate random value: %v", err)
		}
		hashKey := crypto.Keccak256Hash(key)
		if err := db.Put(hashKey.Bytes(), val); err != nil {
			b.Fatalf("Failed to put data: %v", err)
		}
		if i == 0 {
			firstKey = hashKey.Bytes()
		}
		if i == numEntries-1 {
			lastKey = hashKey.Bytes()
		}
	}
	if bytes.Compare(firstKey, lastKey) > 0 {
		firstKey, lastKey = lastKey, firstKey
	}
	return firstKey, append(lastKey, 0) // append 0 to make range exclusive for lastKey
}

func BenchmarkSafeDeleteRange(b *testing.B) {
	db := NewMemoryDatabase()
	defer db.Close()

	const numEntries = 10000

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		start, end := setupRangeDB(b, db, numEntries)
		if err := SafeDeleteRange(db, start, end, true, func(bool) bool { return false }); err != nil {
			b.Fatalf("Failed to delete range: %v", err)
		}
	}
}
