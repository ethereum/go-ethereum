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
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

package dbconfig

import (
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/hashdb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
)

// HashDefaults represents a configuration for using a hash-based scheme with
// default settings. The default configuration is assumed immutable, so please
// deep-copy the configuration if any mutation is expected.
var HashDefaults = triedb.Config{
	Preimages: false,
	IsVerkle:  false,
	HashDB: &hashdb.Config{
		// Explicitly set clean cache size to zero as default to avoid
		// creating fastcache, otherwise database must be closed when
		// it's no longer needed to prevent memory leak.
		CleanCacheSize: 0,

		// Merkle trie resolver is used as the default node resolver.
		ChildResolver: trie.MerkleResolver,
	},
}

// PathDefaults represents a configuration for using a path-based scheme with
// default settings. The default configuration is assumed immutable, so please
// deep-copy the configuration if any mutation is expected.
var PathDefaults = triedb.Config{
	Preimages: false,
	IsVerkle:  false,
	PathDB: &pathdb.Config{
		StateHistory: params.FullImmutabilityThreshold,

		// Explicitly set clean cache size to zero as default to avoid
		// creating fastcache, otherwise database must be closed when
		// it's no longer needed to prevent memory leak.
		CleanCacheSize: 0,
		DirtyCacheSize: pathdb.DefaultBufferSize,

		// Merkle trie loader is used as the default trie loader.
		TrieOpener: trie.NewMerkleOpener,
	},
}
