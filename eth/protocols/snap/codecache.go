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
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

// codeCacheLimit caps the size of cache, budgeting ~64 bytes per entry
// (32-byte hash plus map overhead) within 256MB. On overflow the cache
// degrades to partial mode.
var codeCacheLimit = 256 * 1024 * 1024 / 64

// codeCache tracks which bytecodes are already present in the database.
//
// The cache is bounded: once the entry cap is hit it degrades to partial
// mode, where hits remain authoritative but misses consult the database.
type codeCache struct {
	db      ethdb.KeyValueStore
	seen    map[common.Hash]struct{}
	partial bool
}

// newCodeCache constructs an unloaded bytecode cache.
func newCodeCache(db ethdb.KeyValueStore) *codeCache {
	return &codeCache{db: db, partial: true}
}

// loaded reports whether the cache has been warmed up from disk.
func (c *codeCache) loaded() bool {
	return c.seen != nil
}

// load warms the cache up from the codes already on disk. The scan stops
// once the memory budget is hit, degrading the cache to partial mode.
func (c *codeCache) load() {
	start := time.Now()
	c.seen = make(map[common.Hash]struct{})
	c.partial = false

	it := c.db.NewIterator(rawdb.CodePrefix, nil)
	defer it.Release()

	prefix := len(rawdb.CodePrefix)
	for it.Next() {
		if key := it.Key(); len(key) == prefix+common.HashLength {
			if len(c.seen) >= codeCacheLimit {
				c.partial = true
				break
			}
			c.seen[common.BytesToHash(key[prefix:])] = struct{}{}
		}
	}
	log.Info("Loaded bytecode presence cache", "codes", len(c.seen), "partial", c.partial, "elapsed", common.PrettyDuration(time.Since(start)))
}

// mark registers a persisted bytecode in the cache, degrading the cache to
// partial mode instead of outgrowing its memory budget.
func (c *codeCache) mark(hash common.Hash) {
	if c.seen == nil {
		return
	}
	if _, ok := c.seen[hash]; ok {
		return
	}
	if c.partial {
		return
	}
	c.seen[hash] = struct{}{}
	c.partial = len(c.seen) >= codeCacheLimit
}

// has reports whether the bytecode is already stored locally. A cache hit
// is always authoritative; a miss is only trusted if the cache is complete,
// otherwise the database is consulted.
func (c *codeCache) has(hash common.Hash) bool {
	if _, ok := c.seen[hash]; ok {
		return true
	}
	return c.partial && rawdb.HasCodeWithPrefix(c.db, hash)
}

// release drops the cached entries once the account phase no longer needs
// them, reverting the cache to a pure database pass-through.
func (c *codeCache) release() {
	c.seen = nil
	c.partial = true
}
