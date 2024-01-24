// Copyright 2024 The go-ethereum Authors
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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>

package pathdb

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/ethereum/go-ethereum/triedb/state"
)

// Config contains the settings for database.
type Config struct {
	StateHistory   uint64                                          // Number of recent blocks to maintain state history for
	CleanCacheSize int                                             // Maximum memory allowance (in bytes) for caching clean nodes
	DirtyCacheSize int                                             // Maximum memory allowance (in bytes) for caching dirty nodes
	ReadOnly       bool                                            // Flag whether the database is opened in read only mode.
	TrieOpener     func(db database.NodeDatabase) state.TrieOpener // Function to create trie loader for trie state transition
}

// sanitize checks the provided user configurations and changes anything that's
// unreasonable or unworkable.
func (c *Config) sanitize() (*Config, error) {
	if c == nil {
		return nil, errors.New("pathdb config is nil")
	}
	if c.TrieOpener == nil {
		return nil, errors.New("trie opener is not configured")
	}
	conf := *c
	if conf.DirtyCacheSize > maxBufferSize {
		log.Warn("Sanitizing invalid node buffer size", "provided", common.StorageSize(conf.DirtyCacheSize), "updated", common.StorageSize(maxBufferSize))
		conf.DirtyCacheSize = maxBufferSize
	}
	return &conf, nil
}

// Copy returns a deep copied config object.
func (c *Config) Copy() *Config {
	return &Config{
		StateHistory:   c.StateHistory,
		CleanCacheSize: c.CleanCacheSize,
		DirtyCacheSize: c.DirtyCacheSize,
		ReadOnly:       c.ReadOnly,
		TrieOpener:     c.TrieOpener,
	}
}
