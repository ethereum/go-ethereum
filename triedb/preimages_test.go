// Copyright 2023 The go-ethereum Authors
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

package triedb

import (
	"bytes"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
)

// TestDatabasePreimages tests the preimage functionality of the trie database.
func TestDatabasePreimages(t *testing.T) {
	// Create a database with preimages enabled
	memDB := rawdb.NewMemoryDatabase()
	config := &Config{
		Preimages: true,
		HashDB:    hashdb.Defaults,
	}
	db := NewDatabase(memDB, config)
	defer db.Close()

	// Test inserting and retrieving preimages
	preimages := make(map[common.Hash][]byte)
	for i := 0; i < 10; i++ {
		data := []byte{byte(i), byte(i + 1), byte(i + 2)}
		hash := common.BytesToHash(data)
		preimages[hash] = data
	}

	// Insert preimages into the database
	db.InsertPreimage(preimages)

	// Verify all preimages are retrievable
	for hash, data := range preimages {
		retrieved := db.Preimage(hash)
		if retrieved == nil {
			t.Errorf("Preimage for %x not found", hash)
		}
		if !bytes.Equal(retrieved, data) {
			t.Errorf("Preimage data mismatch: got %x want %x", retrieved, data)
		}
	}

	// Test non-existent preimage
	nonExistentHash := common.HexToHash("deadbeef")
	if data := db.Preimage(nonExistentHash); data != nil {
		t.Errorf("Unexpected preimage data for non-existent hash: %x", data)
	}

	// Force preimage commit and verify again
	db.WritePreimages()
	for hash, data := range preimages {
		retrieved := db.Preimage(hash)
		if retrieved == nil {
			t.Errorf("Preimage for %x not found after forced commit", hash)
		}
		if !bytes.Equal(retrieved, data) {
			t.Errorf("Preimage data mismatch after forced commit: got %x want %x", retrieved, data)
		}
	}
}
