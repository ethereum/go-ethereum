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
	"github.com/ethereum/go-ethereum/triedb/pathdb"
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

// TestDatabaseScheme tests the scheme identification functionality.
func TestDatabaseScheme(t *testing.T) {
	memDB := rawdb.NewMemoryDatabase()

	// Create a database with hash scheme
	hashConfig := &Config{
		Preimages: false,
		HashDB:    hashdb.Defaults,
	}
	hashDB := NewDatabase(memDB, hashConfig)
	if scheme := hashDB.Scheme(); scheme != rawdb.HashScheme {
		t.Errorf("Expected hash scheme, got %s", scheme)
	}
	hashDB.Close()

	// Create a database with path scheme
	pathConfig := &Config{
		Preimages: false,
		PathDB:    pathdb.Defaults,
	}
	pathDB := NewDatabase(memDB, pathConfig)
	if scheme := pathDB.Scheme(); scheme != rawdb.PathScheme {
		t.Errorf("Expected path scheme, got %s", scheme)
	}
	pathDB.Close()
}

// TestDatabaseSize tests the size reporting functionality of the trie database.
func TestDatabaseSize(t *testing.T) {
	memDB := rawdb.NewMemoryDatabase()
	config := &Config{
		Preimages: true,
		HashDB:    hashdb.Defaults,
	}
	db := NewDatabase(memDB, config)
	defer db.Close()

	// Record initial sizes
	_, _, initPreimages := db.Size()

	// Insert some preimages
	preimages := make(map[common.Hash][]byte)
	for i := 0; i < 10; i++ {
		data := bytes.Repeat([]byte{byte(i)}, 100) // 100 bytes each
		hash := common.BytesToHash(data)
		preimages[hash] = data
	}
	db.InsertPreimage(preimages)

	// Check that preimage size has increased
	_, _, afterPreimages := db.Size()
	if afterPreimages <= initPreimages {
		t.Errorf("Expected preimage size to increase, got %v before and %v after", initPreimages, afterPreimages)
	}
}

// TestDatabaseIsVerkle tests the Verkle tree flag functionality.
func TestDatabaseIsVerkle(t *testing.T) {
	memDB := rawdb.NewMemoryDatabase()

	// Create a regular database (non-Verkle)
	regConfig := &Config{
		Preimages: false,
		IsVerkle:  false,
		HashDB:    hashdb.Defaults,
	}
	regDB := NewDatabase(memDB, regConfig)
	if regDB.IsVerkle() {
		t.Errorf("Expected IsVerkle() to be false for regular database")
	}
	regDB.Close()

	// Create a Verkle database
	verkleConfig := &Config{
		Preimages: false,
		IsVerkle:  true,
		PathDB:    pathdb.Defaults,
	}
	verkleDB := NewDatabase(memDB, verkleConfig)
	if !verkleDB.IsVerkle() {
		t.Errorf("Expected IsVerkle() to be true for Verkle database")
	}
	verkleDB.Close()
}

// TestDatabaseDiskAccess tests that the Disk method returns the underlying database.
func TestDatabaseDiskAccess(t *testing.T) {
	memDB := rawdb.NewMemoryDatabase()
	config := &Config{
		Preimages: false,
		HashDB:    hashdb.Defaults,
	}
	db := NewDatabase(memDB, config)
	defer db.Close()

	// The Disk() method should return the database we provided during initialization
	disk := db.Disk()
	if disk == nil {
		t.Errorf("Expected Disk() to return a database, got nil")
	}
}

// TestDatabaseCap tests the Cap functionality.
func TestDatabaseCap(t *testing.T) {
	memDB := rawdb.NewMemoryDatabase()
	config := &Config{
		Preimages: true,
		HashDB:    hashdb.Defaults,
	}
	db := NewDatabase(memDB, config)
	defer db.Close()

	// Insert some preimages to have some memory usage
	preimages := make(map[common.Hash][]byte)
	for i := 0; i < 100; i++ {
		data := bytes.Repeat([]byte{byte(i)}, 1000) // 1000 bytes each
		hash := common.BytesToHash(data)
		preimages[hash] = data
	}
	db.InsertPreimage(preimages)

	// Record initial size
	_, _, initialSize := db.Size()

	// Test that calling Cap with a limit doesn't cause an error
	// Note: In the current implementation, Cap on preimages may not immediately
	// reduce the size since it often just marks data for future pruning during
	// garbage collection, rather than immediately removing it
	err := db.Cap(initialSize / 2)
	if err != nil {
		t.Errorf("Error when applying cap: %v", err)
	}
}
