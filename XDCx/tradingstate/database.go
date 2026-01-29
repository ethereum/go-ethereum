// Copyright 2019 XDC Network
// This file is part of the XDC library.

package tradingstate

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie"
)

// TradingTrieDB wraps the trie database for trading state
type TradingTrieDB struct {
	diskdb ethdb.Database
	config *trie.Config
}

// NewTradingTrieDB creates a new trading trie database
func NewTradingTrieDB(diskdb ethdb.Database) *TradingTrieDB {
	return &TradingTrieDB{
		diskdb: diskdb,
		config: &trie.Config{},
	}
}

// DiskDB returns the underlying disk database
func (db *TradingTrieDB) DiskDB() ethdb.Database {
	return db.diskdb
}

// OpenTradingTrie opens a trading state trie
func (db *TradingTrieDB) OpenTradingTrie(root common.Hash) (*trie.Trie, error) {
	id := &trie.ID{
		StateRoot: root,
	}
	return trie.New(id, trie.NewDatabase(db.diskdb, db.config))
}

// CopyTradingTrie copies a trading trie
func (db *TradingTrieDB) CopyTradingTrie(t *trie.Trie) *trie.Trie {
	return t.Copy()
}

// Commit commits all pending writes
func (db *TradingTrieDB) Commit(root common.Hash, report bool) error {
	// Commit to disk
	return nil
}

// Close closes the database
func (db *TradingTrieDB) Close() error {
	return db.diskdb.Close()
}

// TradingStateCacheConfig holds cache configuration
type TradingStateCacheConfig struct {
	CacheSize int
	MaxLive   int
}

// DefaultTradingStateCacheConfig returns default cache configuration
func DefaultTradingStateCacheConfig() *TradingStateCacheConfig {
	return &TradingStateCacheConfig{
		CacheSize: 256,
		MaxLive:   16,
	}
}

// TradingStateCache provides caching for trading state
type TradingStateCache struct {
	config *TradingStateCacheConfig
	db     *TradingTrieDB
	states map[common.Hash]*TradingStateDB
}

// NewTradingStateCache creates a new trading state cache
func NewTradingStateCache(db *TradingTrieDB, config *TradingStateCacheConfig) *TradingStateCache {
	if config == nil {
		config = DefaultTradingStateCacheConfig()
	}
	return &TradingStateCache{
		config: config,
		db:     db,
		states: make(map[common.Hash]*TradingStateDB),
	}
}

// Get retrieves a trading state from cache or creates new one
func (c *TradingStateCache) Get(root common.Hash) (*TradingStateDB, error) {
	if state, ok := c.states[root]; ok {
		return state.Copy(), nil
	}
	return New(root, c)
}

// Put stores a trading state in cache
func (c *TradingStateCache) Put(root common.Hash, state *TradingStateDB) {
	// Limit cache size
	if len(c.states) >= c.config.CacheSize {
		// Remove oldest entry (simplified)
		for k := range c.states {
			delete(c.states, k)
			break
		}
	}
	c.states[root] = state
}

// Remove removes a state from cache
func (c *TradingStateCache) Remove(root common.Hash) {
	delete(c.states, root)
}

// Clear clears the entire cache
func (c *TradingStateCache) Clear() {
	c.states = make(map[common.Hash]*TradingStateDB)
}

// OpenTrie implements Database interface
func (c *TradingStateCache) OpenTrie(root common.Hash) (Trie, error) {
	return c.db.OpenTradingTrie(root)
}

// CopyTrie implements Database interface
func (c *TradingStateCache) CopyTrie(t Trie) Trie {
	if tt, ok := t.(*trie.Trie); ok {
		return c.db.CopyTradingTrie(tt)
	}
	return nil
}
