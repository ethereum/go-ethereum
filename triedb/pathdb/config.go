// Copyright 2025 The go-ethereum Authors
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

package pathdb

import (
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

const (
	// defaultTrieCleanSize is the default memory allowance of clean trie cache.
	defaultTrieCleanSize = 16 * 1024 * 1024

	// defaultStateCleanSize is the default memory allowance of clean state cache.
	defaultStateCleanSize = 16 * 1024 * 1024

	// maxBufferSize is the maximum memory allowance of node buffer.
	// Too large buffer will cause the system to pause for a long
	// time when write happens. Also, the largest batch that pebble can
	// support is 4GB, node will panic if batch size exceeds this limit.
	maxBufferSize = 256 * 1024 * 1024

	// defaultBufferSize is the default memory allowance of node buffer
	// that aggregates the writes from above until it's flushed into the
	// disk. It's meant to be used once the initial sync is finished.
	// Do not increase the buffer size arbitrarily, otherwise the system
	// pause time will increase when the database writes happen.
	defaultBufferSize = 64 * 1024 * 1024
)

var (
	// maxDiffLayers is the maximum diff layers allowed in the layer tree.
	maxDiffLayers = 128
)

// Defaults contains default settings for Ethereum mainnet.
var Defaults = &Config{
	StateHistory:        params.FullImmutabilityThreshold,
	EnableStateIndexing: false,
	TrieCleanSize:       defaultTrieCleanSize,
	StateCleanSize:      defaultStateCleanSize,
	WriteBufferSize:     defaultBufferSize,
}

// ReadOnly is the config in order to open database in read only mode.
var ReadOnly = &Config{
	ReadOnly:       true,
	TrieCleanSize:  defaultTrieCleanSize,
	StateCleanSize: defaultStateCleanSize,
}

// Config contains the settings for database.
type Config struct {
	StateHistory        uint64 // Number of recent blocks to maintain state history for, 0: full chain
	EnableStateIndexing bool   // Whether to enable state history indexing for external state access
	TrieCleanSize       int    // Maximum memory allowance (in bytes) for caching clean trie data
	StateCleanSize      int    // Maximum memory allowance (in bytes) for caching clean state data
	WriteBufferSize     int    // Maximum memory allowance (in bytes) for write buffer
	ReadOnly            bool   // Flag whether the database is opened in read only mode
	JournalDirectory    string // Absolute path of journal directory (null means the journal data is persisted in key-value store)

	// Testing configurations
	SnapshotNoBuild   bool // Flag Whether the state generation is disabled
	NoAsyncFlush      bool // Flag whether the background buffer flushing is disabled
	NoAsyncGeneration bool // Flag whether the background generation is disabled
}

// sanitize checks the provided user configurations and changes anything that's
// unreasonable or unworkable.
func (c *Config) sanitize() *Config {
	conf := *c
	if conf.WriteBufferSize > maxBufferSize {
		log.Warn("Sanitizing invalid node buffer size", "provided", common.StorageSize(conf.WriteBufferSize), "updated", common.StorageSize(maxBufferSize))
		conf.WriteBufferSize = maxBufferSize
	}
	return &conf
}

// fields returns a list of attributes of config for printing.
func (c *Config) fields() []interface{} {
	var list []interface{}
	if c.ReadOnly {
		list = append(list, "readonly", true)
	}
	list = append(list, "triecache", common.StorageSize(c.TrieCleanSize))
	list = append(list, "statecache", common.StorageSize(c.StateCleanSize))
	list = append(list, "buffer", common.StorageSize(c.WriteBufferSize))

	if c.StateHistory == 0 {
		list = append(list, "state-history", "entire chain")
	} else {
		list = append(list, "state-history", fmt.Sprintf("last %d blocks", c.StateHistory))
	}
	if c.EnableStateIndexing {
		list = append(list, "index-history", true)
	}
	if c.JournalDirectory != "" {
		list = append(list, "journal-dir", c.JournalDirectory)
	}
	return list
}
