// Copyright 2026 The go-ethereum Authors
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

package snap

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/crypto"
)

// Tests the bytecode presence cache in complete mode: after warming it up
// from disk, hits and misses are both authoritative and no database lookups
// happen.
func TestCodeCacheComplete(t *testing.T) {
	db := rawdb.NewMemoryDatabase()

	var hashes []common.Hash
	for i := 0; i < 4; i++ {
		code := []byte{byte(i + 1)}
		rawdb.WriteCode(db, crypto.Keccak256Hash(code), code)
		hashes = append(hashes, crypto.Keccak256Hash(code))
	}
	c := newCodeCache(db)

	// An unloaded cache passes every lookup through to the database
	if !c.has(hashes[0]) {
		t.Fatalf("persisted code %x invisible before load", hashes[0])
	}
	if c.has(common.Hash{0x01}) {
		t.Fatalf("absent code reported present before load")
	}
	c.load()

	if c.partial {
		t.Fatalf("presence cache unexpectedly partial")
	}
	for _, hash := range hashes {
		if !c.has(hash) {
			t.Fatalf("preloaded code %x reported missing", hash)
		}
	}
	if c.has(common.Hash{0x01}) {
		t.Fatalf("absent code reported present")
	}
	// A complete cache trusts misses: codes written behind its back (there
	// are no such writes in the syncer) are deliberately not visible.
	sneaked := crypto.Keccak256Hash([]byte{0xff})
	rawdb.WriteCode(db, sneaked, []byte{0xff})
	if c.has(sneaked) {
		t.Fatalf("unmarked code visible in complete mode")
	}
	// Delivered codes are visible the moment they are marked
	c.mark(sneaked)
	if !c.has(sneaked) {
		t.Fatalf("marked code reported missing")
	}
	// A released cache reverts to a pure database pass-through
	c.release()
	for _, hash := range hashes {
		if !c.has(hash) {
			t.Fatalf("persisted code %x invisible after release", hash)
		}
	}
	if c.has(common.Hash{0x01}) {
		t.Fatalf("absent code reported present after release")
	}
}

// Tests the bytecode presence cache in partial mode: once the entry cap is
// hit the cache stops growing and misses fall back to database lookups.
func TestCodeCachePartial(t *testing.T) {
	defer func(limit int) { codeCacheLimit = limit }(codeCacheLimit)
	codeCacheLimit = 2

	db := rawdb.NewMemoryDatabase()

	var hashes []common.Hash
	for i := 0; i < 4; i++ {
		code := []byte{byte(i + 1)}
		rawdb.WriteCode(db, crypto.Keccak256Hash(code), code)
		hashes = append(hashes, crypto.Keccak256Hash(code))
	}
	c := newCodeCache(db)
	c.load()

	if !c.partial {
		t.Fatalf("presence cache unexpectedly complete")
	}
	if len(c.seen) != codeCacheLimit {
		t.Fatalf("cache size mismatch: have %d, want %d", len(c.seen), codeCacheLimit)
	}
	// All persisted codes must be reported present, cached or not
	for _, hash := range hashes {
		if !c.has(hash) {
			t.Fatalf("persisted code %x reported missing", hash)
		}
	}
	if c.has(common.Hash{0x01}) {
		t.Fatalf("absent code reported present")
	}
	// Delivered codes must not outgrow the cap, yet stay visible via the
	// database fallback once persisted
	delivered := crypto.Keccak256Hash([]byte{0xff})
	rawdb.WriteCode(db, delivered, []byte{0xff})
	c.mark(delivered)

	if len(c.seen) != codeCacheLimit {
		t.Fatalf("cache outgrew its cap: have %d, want %d", len(c.seen), codeCacheLimit)
	}
	if !c.has(delivered) {
		t.Fatalf("delivered code reported missing")
	}
}
