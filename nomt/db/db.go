// Package db provides the NOMT trie database combining PebbleDB page
// storage with the PageWalker merkle engine.
//
// Trie pages are stored as 4KB blobs in geth's ethdb under key prefix 0x04.
// Flat key-value storage (accounts, storage slots) stays on geth's PebbleDB
// under separate prefixes managed by triedb/nomtdb.
package db

import (
	"bytes"
	"fmt"
	"runtime"
	"sort"
	"sync"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/nomt/core"
	"github.com/ethereum/go-ethereum/nomt/merkle"
)

const (
	// nomtPagePrefix is the ethdb key prefix for NOMT trie pages.
	// Key format: 0x04 || PageID.Encode()[32] → RawPage[4032]
	nomtPagePrefix byte = 0x04

	// nomtMetaPrefix is the ethdb key prefix for NOMT metadata.
	nomtMetaPrefix byte = 0x05
)

// nomtMetaRootKey is the ethdb key for the persisted page tree root.
var nomtMetaRootKey = []byte{nomtMetaPrefix, 'r', 'o', 'o', 't'}

// Config holds configuration for the NOMT database.
type Config struct {
	// NumWorkers is the number of parallel goroutines for trie updates.
	// Defaults to runtime.NumCPU() if zero.
	NumWorkers int
}

// DefaultConfig returns a default configuration.
func DefaultConfig() Config {
	return Config{}
}

// DB is the NOMT trie database.
type DB struct {
	diskdb     ethdb.Database
	root       core.Node
	numWorkers int
	mu         sync.RWMutex
}

// New creates or opens a NOMT trie database backed by the given ethdb.
// The page tree root is loaded from persisted metadata if available.
func New(diskdb ethdb.Database, config Config) (*DB, error) {
	numWorkers := config.NumWorkers
	if numWorkers <= 0 {
		numWorkers = runtime.NumCPU()
	}

	db := &DB{
		diskdb:     diskdb,
		root:       core.Terminator,
		numWorkers: numWorkers,
	}

	// Load persisted root.
	if data, err := diskdb.Get(nomtMetaRootKey); err == nil && len(data) == 32 {
		copy(db.root[:], data)
	}

	return db, nil
}

// Root returns the current trie root hash.
func (db *DB) Root() core.Node {
	db.mu.RLock()
	defer db.mu.RUnlock()
	return db.root
}

// Update applies a batch of stem key-value pairs to the trie.
// The pairs are sorted internally before processing.
func (db *DB) Update(ops []core.StemKeyValue) (core.Node, error) {
	sort.Slice(ops, func(i, j int) bool {
		return stemLess(&ops[i].Stem, &ops[j].Stem)
	})
	return db.UpdateSorted(ops)
}

// UpdateSorted applies a pre-sorted batch of stem key-value pairs to the trie.
// The caller must ensure ops are sorted by stem path.
func (db *DB) UpdateSorted(ops []core.StemKeyValue) (core.Node, error) {
	if len(ops) == 0 {
		return db.Root(), nil
	}

	db.mu.Lock()
	defer db.mu.Unlock()

	pageSetFactory := func() merkle.PageSet {
		return newPebblePageSet(db.diskdb)
	}
	out := merkle.ParallelUpdate(db.root, ops, db.numWorkers, pageSetFactory)

	// Persist updated pages via atomic batch write.
	batch := db.diskdb.NewBatch()
	for _, up := range out.Pages {
		key := nomtPageKey(up.PageID)
		if up.Diff.IsCleared() {
			if err := batch.Delete(key); err != nil {
				return core.Terminator, fmt.Errorf("nomt/db: delete page: %w", err)
			}
		} else {
			if err := batch.Put(key, up.Page[:]); err != nil {
				return core.Terminator, fmt.Errorf("nomt/db: put page: %w", err)
			}
		}
	}
	// Persist root.
	if err := batch.Put(nomtMetaRootKey, out.Root[:]); err != nil {
		return core.Terminator, fmt.Errorf("nomt/db: put root: %w", err)
	}
	if err := batch.Write(); err != nil {
		return core.Terminator, fmt.Errorf("nomt/db: batch write: %w", err)
	}

	db.root = out.Root
	return out.Root, nil
}

// LoadPage loads a page from ethdb storage by its PageID.
func (db *DB) LoadPage(pageID core.PageID) (*core.RawPage, error) {
	data, err := db.diskdb.Get(nomtPageKey(pageID))
	if err != nil {
		return nil, nil // Not found.
	}
	if len(data) != core.PageSize {
		return nil, fmt.Errorf("nomt/db: page size mismatch: got %d, want %d", len(data), core.PageSize)
	}
	page := new(core.RawPage)
	copy(page[:], data)
	return page, nil
}

// Close is a no-op — the ethdb lifecycle is managed by the caller.
func (db *DB) Close() error {
	return nil
}

// --- PebblePageSet ---

// pebblePageSet implements merkle.PageSet backed by ethdb (PebbleDB).
type pebblePageSet struct {
	diskdb ethdb.Database
	cache  map[string]*core.RawPage
}

func newPebblePageSet(diskdb ethdb.Database) *pebblePageSet {
	return &pebblePageSet{
		diskdb: diskdb,
		cache:  make(map[string]*core.RawPage, 16),
	}
}

func (ps *pebblePageSet) Get(pageID core.PageID) (
	*core.RawPage, merkle.PageOrigin, bool,
) {
	key := pageIDCacheKey(pageID)
	if cached, ok := ps.cache[key]; ok {
		// Return a copy so the walker can mutate freely.
		pageCopy := new(core.RawPage)
		*pageCopy = *cached
		return pageCopy, merkle.PageOrigin{
			Kind: merkle.PageOriginPersisted,
		}, true
	}

	data, err := ps.diskdb.Get(nomtPageKey(pageID))
	if err != nil || len(data) != core.PageSize {
		// Return a fresh page if not found — this handles the case
		// where the trie is being built from scratch or expanded
		// into new regions.
		fresh := new(core.RawPage)
		return fresh, merkle.PageOrigin{Kind: merkle.PageOriginFresh}, true
	}

	page := new(core.RawPage)
	copy(page[:], data)
	ps.cache[key] = page

	// Return a copy so the walker can mutate freely.
	pageCopy := new(core.RawPage)
	*pageCopy = *page
	return pageCopy, merkle.PageOrigin{
		Kind: merkle.PageOriginPersisted,
	}, true
}

func (ps *pebblePageSet) Contains(pageID core.PageID) bool {
	key := pageIDCacheKey(pageID)
	if _, ok := ps.cache[key]; ok {
		return true
	}
	has, _ := ps.diskdb.Has(nomtPageKey(pageID))
	return has
}

func (ps *pebblePageSet) Fresh(pageID core.PageID) *core.RawPage {
	return new(core.RawPage)
}

func (ps *pebblePageSet) Insert(
	pageID core.PageID, page *core.RawPage, origin merkle.PageOrigin,
) {
	ps.cache[pageIDCacheKey(pageID)] = page
}

// nomtPageKey builds the ethdb key for a NOMT trie page.
func nomtPageKey(id core.PageID) []byte {
	encoded := id.Encode()
	key := make([]byte, 1+len(encoded))
	key[0] = nomtPagePrefix
	copy(key[1:], encoded[:])
	return key
}

// pageIDCacheKey returns a string key for the in-memory cache.
func pageIDCacheKey(id core.PageID) string {
	encoded := id.Encode()
	return string(encoded[:])
}

func stemLess(a, b *core.StemPath) bool {
	return bytes.Compare(a[:], b[:]) < 0
}
