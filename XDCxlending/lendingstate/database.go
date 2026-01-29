// Copyright 2019 XDC Network
// This file is part of the XDC library.

package lendingstate

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
)

// LendingTrieDB wraps the trie database for lending state
type LendingTrieDB struct {
	diskdb ethdb.Database
	config *trie.Config
}

// NewLendingTrieDB creates a new lending trie database
func NewLendingTrieDB(diskdb ethdb.Database) *LendingTrieDB {
	return &LendingTrieDB{
		diskdb: diskdb,
		config: &trie.Config{},
	}
}

// DiskDB returns the underlying disk database
func (db *LendingTrieDB) DiskDB() ethdb.Database {
	return db.diskdb
}

// OpenLendingTrie opens a lending state trie
func (db *LendingTrieDB) OpenLendingTrie(root common.Hash) (*trie.Trie, error) {
	id := &trie.ID{
		StateRoot: root,
	}
	return trie.New(id, trie.NewDatabase(db.diskdb, db.config))
}

// CopyLendingTrie copies a lending trie
func (db *LendingTrieDB) CopyLendingTrie(t *trie.Trie) *trie.Trie {
	return t.Copy()
}

// Commit commits all pending writes
func (db *LendingTrieDB) Commit(root common.Hash, report bool) error {
	return nil
}

// Close closes the database
func (db *LendingTrieDB) Close() error {
	return db.diskdb.Close()
}

// LendingStateCacheConfig holds cache configuration
type LendingStateCacheConfig struct {
	CacheSize int
	MaxLive   int
}

// DefaultLendingStateCacheConfig returns default cache configuration
func DefaultLendingStateCacheConfig() *LendingStateCacheConfig {
	return &LendingStateCacheConfig{
		CacheSize: 256,
		MaxLive:   16,
	}
}

// LendingStateCache provides caching for lending state
type LendingStateCache struct {
	config *LendingStateCacheConfig
	db     *LendingTrieDB
	states map[common.Hash]*LendingStateDB
}

// NewLendingStateCache creates a new lending state cache
func NewLendingStateCache(db *LendingTrieDB, config *LendingStateCacheConfig) *LendingStateCache {
	if config == nil {
		config = DefaultLendingStateCacheConfig()
	}
	return &LendingStateCache{
		config: config,
		db:     db,
		states: make(map[common.Hash]*LendingStateDB),
	}
}

// Get retrieves a lending state from cache or creates new one
func (c *LendingStateCache) Get(root common.Hash) (*LendingStateDB, error) {
	if state, ok := c.states[root]; ok {
		return state.Copy(), nil
	}
	return New(root, c)
}

// Put stores a lending state in cache
func (c *LendingStateCache) Put(root common.Hash, state *LendingStateDB) {
	if len(c.states) >= c.config.CacheSize {
		for k := range c.states {
			delete(c.states, k)
			break
		}
	}
	c.states[root] = state
}

// Remove removes a state from cache
func (c *LendingStateCache) Remove(root common.Hash) {
	delete(c.states, root)
}

// Clear clears the entire cache
func (c *LendingStateCache) Clear() {
	c.states = make(map[common.Hash]*LendingStateDB)
}

// OpenTrie implements Database interface
func (c *LendingStateCache) OpenTrie(root common.Hash) (Trie, error) {
	return c.db.OpenLendingTrie(root)
}

// CopyTrie implements Database interface
func (c *LendingStateCache) CopyTrie(t Trie) Trie {
	if tt, ok := t.(*trie.Trie); ok {
		return c.db.CopyLendingTrie(tt)
	}
	return nil
}
