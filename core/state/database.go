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

package state

import (
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/trie"
	lru "github.com/hashicorp/golang-lru"
)

// Trie cache generation limit after which to evict trie nodes from memory.
var MaxTrieCacheGen = uint16(120)

const (
	// maxPastTries is the number of past tries to keep. This value is chosen such
	// that reasonable chain reorg depths will hit an existing trie.
	maxPastTries = 12

	// maxPastStorageTries is the number of past storage tries to keep.
	maxPastStorageTries = 4096

	// codeSizeCacheSize is the number of codehash->size associations to keep.
	codeSizeCacheSize = 100000
)

var (
	storageTrieMissCounter   = metrics.NewRegisteredCounter("state/storage/miss", nil)
	storageTrieUnloadCounter = metrics.NewRegisteredCounter("state/storage/unload", nil)
)

// StorageTrieMisses retrieves a global counter measuring the number of storage
// trie cache misses the state had since process startup. This isn't useful for
// anything apart from state debugging purposes.
func StorageTrieMisses() int64 {
	return storageTrieMissCounter.Count()
}

// StorageTrieUnloads retrieves a global counter measuring the number of storage
// trie cache unloads the state had since process startup. This isn't useful for
// anything apart from state debugging purposes.
func StorageTrieUnloads() int64 {
	return storageTrieUnloadCounter.Count()
}

// Database wraps access to tries and contract code.
type Database interface {
	// OpenTrie opens the main account trie.
	OpenTrie(root common.Hash) (Trie, error)

	// OpenStorageTrie opens the storage trie of an account.
	OpenStorageTrie(addrHash, root common.Hash) (Trie, error)

	// CopyTrie returns an independent copy of the given trie.
	CopyTrie(Trie) Trie

	// ContractCode retrieves a particular contract's code.
	ContractCode(addrHash, codeHash common.Hash) ([]byte, error)

	// ContractCodeSize retrieves a particular contracts code's size.
	ContractCodeSize(addrHash, codeHash common.Hash) (int, error)

	// TrieDB retrieves the low level trie database used for data storage.
	TrieDB() *trie.Database
}

// Trie is a Ethereum Merkle Trie.
type Trie interface {
	TryGet(key []byte) ([]byte, error)
	TryUpdate(key, value []byte) error
	TryDelete(key []byte) error
	Commit(onleaf trie.LeafCallback) (common.Hash, error)
	Hash() common.Hash
	NodeIterator(startKey []byte) trie.NodeIterator
	GetKey([]byte) []byte // TODO(fjl): remove this when SecureTrie is removed
	Prove(key []byte, fromLevel uint, proofDb ethdb.Putter) error
}

// NewDatabase creates a backing store for state. The returned database is safe for
// concurrent use and retains cached trie nodes in memory. The pool is an optional
// intermediate trie-node memory pool between the low level storage layer and the
// high level trie abstraction.
func NewDatabase(db ethdb.Database) Database {
	csc, _ := lru.New(codeSizeCacheSize)
	str, _ := lru.NewWithEvict(maxPastStorageTries, func(key interface{}, value interface{}) {
		storageTrieUnloadCounter.Inc(1)
	})

	return &cachingDB{
		db:               trie.NewDatabase(db),
		pastStorageTries: str,
		codeSizeCache:    csc,
	}
}

type cachingDB struct {
	db               *trie.Database
	mu               sync.Mutex
	pastTries        []*trie.SecureTrie
	pastStorageTries *lru.Cache
	codeSizeCache    *lru.Cache
}

// OpenTrie opens the main account trie.
func (db *cachingDB) OpenTrie(root common.Hash) (Trie, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	for i := len(db.pastTries) - 1; i >= 0; i-- {
		if db.pastTries[i].Hash() == root {
			return cachedTrie{db.pastTries[i].Copy(), db}, nil
		}
	}
	tr, err := trie.NewSecure(root, db.db, MaxTrieCacheGen)
	if err != nil {
		return nil, err
	}
	return cachedTrie{tr, db}, nil
}

func (db *cachingDB) pushTrie(t *trie.SecureTrie) {
	db.mu.Lock()
	defer db.mu.Unlock()

	if len(db.pastTries) >= maxPastTries {
		copy(db.pastTries, db.pastTries[1:])
		db.pastTries[len(db.pastTries)-1] = t
	} else {
		db.pastTries = append(db.pastTries, t)
	}
}

// OpenStorageTrie opens the storage trie of an account.
func (db *cachingDB) OpenStorageTrie(addrHash, root common.Hash) (Trie, error) {
	db.mu.Lock()
	defer db.mu.Unlock()

	// Retrieve a storage trie from the cache if available
	if t, ok := db.pastStorageTries.Get(root); ok {
		return &cachedStorageTrie{reader: t.(*syncedTrie), db: db}, nil
	}
	// Trie not cached, construct a brand new one and cache it if non-empty
	tr, err := trie.NewSecure(root, db.db, MaxTrieCacheGen)
	if err != nil {
		return nil, err
	}
	str := &syncedTrie{tr: tr}

	if root != emptyRoot {
		db.pastStorageTries.Add(root, str)
		storageTrieMissCounter.Inc(1)
	}
	return &cachedStorageTrie{reader: str, db: db}, nil
}

func (db *cachingDB) pushStorageTrie(t *syncedTrie) {
	// Refuse to cache the empty trie
	if t.Hash() == emptyRoot {
		return
	}
	db.mu.Lock()
	defer db.mu.Unlock()

	db.pastStorageTries.Add(t.Hash(), t)
}

// CopyTrie returns an independent copy of the given trie.
func (db *cachingDB) CopyTrie(t Trie) Trie {
	switch t := t.(type) {
	case cachedTrie:
		return cachedTrie{t.SecureTrie.Copy(), db}
	case *cachedStorageTrie:
		if t.writer != nil {
			return &cachedStorageTrie{reader: t.writer.Copy(), db: db}
		}
		return &cachedStorageTrie{reader: t.reader.Copy(), db: db}
	case *trie.SecureTrie:
		return t.Copy()
	default:
		panic(fmt.Errorf("unknown trie type %T", t))
	}
}

// ContractCode retrieves a particular contract's code.
func (db *cachingDB) ContractCode(addrHash, codeHash common.Hash) ([]byte, error) {
	code, err := db.db.Node(codeHash)
	if err == nil {
		db.codeSizeCache.Add(codeHash, len(code))
	}
	return code, err
}

// ContractCodeSize retrieves a particular contracts code's size.
func (db *cachingDB) ContractCodeSize(addrHash, codeHash common.Hash) (int, error) {
	if cached, ok := db.codeSizeCache.Get(codeHash); ok {
		return cached.(int), nil
	}
	code, err := db.ContractCode(addrHash, codeHash)
	return len(code), err
}

// TrieDB retrieves any intermediate trie-node caching layer.
func (db *cachingDB) TrieDB() *trie.Database {
	return db.db
}

// cachedTrie inserts its trie into a cachingDB on commit.
type cachedTrie struct {
	*trie.SecureTrie
	db *cachingDB
}

func (m cachedTrie) Commit(onleaf trie.LeafCallback) (common.Hash, error) {
	root, err := m.SecureTrie.Commit(onleaf)
	if err == nil {
		m.db.pushTrie(m.SecureTrie)
	}
	return root, err
}

func (m cachedTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.Putter) error {
	return m.SecureTrie.Prove(key, fromLevel, proofDb)
}

// syncedTrie is a synchronized wrapper around a trie to support multiple threads
// accessing and read-expanding the same trie (i.e. caching loaded trie nodes).
type syncedTrie struct {
	tr   *trie.SecureTrie
	lock sync.Mutex // Any operation on a trie can expand it (RWMutex is not enough)
}

func (s *syncedTrie) Copy() *syncedTrie {
	s.lock.Lock()
	defer s.lock.Unlock()

	return &syncedTrie{tr: s.tr.Copy()}
}

func (s *syncedTrie) TryGet(key []byte) ([]byte, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.tr.TryGet(key)
}

func (s *syncedTrie) TryUpdate(key, value []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.tr.TryUpdate(key, value)
}

func (s *syncedTrie) TryDelete(key []byte) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.tr.TryDelete(key)
}

func (s *syncedTrie) Commit(onleaf trie.LeafCallback) (common.Hash, error) {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.tr.Commit(onleaf)
}

func (s *syncedTrie) Hash() common.Hash {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.tr.Hash()
}

func (s *syncedTrie) NodeIterator(startKey []byte) trie.NodeIterator {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.tr.NodeIterator(startKey)
}

func (s *syncedTrie) GetKey(key []byte) []byte {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.tr.GetKey(key)
}

func (s *syncedTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.Putter) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.tr.Prove(key, fromLevel, proofDb)
}

// cachedStorageTrie is a wrapper around a read-only trie shared by possibly
// multiple goroutines and a write-enabled trie unique to each goroutine.
//
// Important! Reads always go through the expand-only disk-backed trie whereas
// writes go through the write-enabled one. This wrapper assumes that any trie
// item read and modified will not be read again, rather used from a higher cache.
type cachedStorageTrie struct {
	reader *syncedTrie // Original trie for expand-only operations
	writer *syncedTrie // Copy trie for update and delete operations

	db *cachingDB
}

func (c *cachedStorageTrie) TryGet(key []byte) ([]byte, error) {
	res, err := c.reader.TryGet(key)
	if c.writer != nil {
		return c.writer.TryGet(key)
	}
	return res, err
}

func (c *cachedStorageTrie) TryUpdate(key, value []byte) error {
	if c.writer == nil {
		c.writer = c.reader.Copy()
	}
	return c.writer.TryUpdate(key, value)
}

func (c *cachedStorageTrie) TryDelete(key []byte) error {
	if c.writer == nil {
		c.writer = c.reader.Copy()
	}
	return c.writer.TryDelete(key)
}

func (c *cachedStorageTrie) Hash() common.Hash {
	if c.writer != nil {
		return c.writer.Hash()
	}
	return c.reader.Hash()
}

func (c *cachedStorageTrie) NodeIterator(startKey []byte) trie.NodeIterator {
	if c.writer != nil {
		return c.writer.NodeIterator(startKey)
	}
	return c.reader.NodeIterator(startKey)
}

func (c *cachedStorageTrie) GetKey(key []byte) []byte {
	if c.writer != nil {
		return c.writer.GetKey(key)
	}
	return c.reader.GetKey(key)
}

func (c *cachedStorageTrie) Prove(key []byte, fromLevel uint, proofDb ethdb.Putter) error {
	if c.writer != nil {
		return c.writer.Prove(key, fromLevel, proofDb)
	}
	return c.reader.Prove(key, fromLevel, proofDb)
}

func (c *cachedStorageTrie) Commit(onleaf trie.LeafCallback) (common.Hash, error) {
	// Retrieve the hash of the read-only trie (should be free)
	origin, err := c.reader.Commit(nil)

	// If there have been modifications made, push the writer into the cache
	if c.writer != nil {
		updated, err := c.writer.Commit(onleaf)
		if err == nil && updated != origin {
			c.db.pushStorageTrie(c.writer)
		}
		return updated, err
	}
	return origin, err
}
