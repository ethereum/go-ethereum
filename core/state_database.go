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

package core

import (
	"errors"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/utils"
	"github.com/ethereum/go-ethereum/triedb"
)

// stateDatabase is the central structure for managing state data,
// including both the Merkle-Patricia Trie and the Verkle tree (post-transition).
type stateDatabase struct {
	disk   ethdb.Database
	merkle *triedb.Database
	verkle *triedb.Database
	snap   *snapshot.Tree

	// Various caches
	codeCache     *lru.SizeConstrainedCache[common.Hash, []byte]
	codeSizeCache *lru.Cache[common.Hash, int]
	pointCache    *utils.PointCache
}

// newStateDatabase initializes a new stateDatabase instance, creating
// both Merkle and Verkle trie databases.
func newStateDatabase(disk ethdb.Database, cfg *BlockChainConfig) *stateDatabase {
	return &stateDatabase{
		disk:          disk,
		merkle:        triedb.NewDatabase(disk, cfg.triedbConfig(false)),
		verkle:        triedb.NewDatabase(disk, cfg.triedbConfig(true)),
		codeCache:     lru.NewSizeConstrainedCache[common.Hash, []byte](state.CodeCacheSize),
		codeSizeCache: lru.NewCache[common.Hash, int](state.CodeSizeCacheSize),
		pointCache:    utils.NewPointCache(state.PointCacheSize),
	}
}

// setSnapshot assigns a snapshot tree to the state database.
func (s *stateDatabase) setSnapshot(snap *snapshot.Tree) {
	s.snap = snap
}

// triedb returns the appropriate trie database based on the given flag.
func (s *stateDatabase) triedb(isVerkle bool) *triedb.Database {
	if isVerkle {
		return s.verkle
	}
	return s.merkle
}

// stateDB returns the appropriate state database based on the given flag.
func (s *stateDatabase) stateDB(isVerkle bool) *state.CachingDB {
	if isVerkle {
		// Verkle is compatible only with path mode; snapshot is integrated natively.
		return state.NewDatabaseWithCache(s.disk, s.verkle, nil, s.codeCache, s.codeSizeCache, s.pointCache)
	}
	return state.NewDatabaseWithCache(s.disk, s.merkle, s.snap, s.codeCache, s.codeSizeCache, s.pointCache)
}

// contractCode retrieves the contract code by its hash. If not present in the
// cache, it falls back to reading from disk using the contract database prefix.
func (s *stateDatabase) contractCode(address common.Address, codeHash common.Hash) []byte {
	code, _ := s.codeCache.Get(codeHash)
	if len(code) > 0 {
		return code
	}
	code = rawdb.ReadCodeWithPrefix(s.disk, codeHash)
	if len(code) > 0 {
		s.codeCache.Add(codeHash, code)
		s.codeSizeCache.Add(codeHash, len(code))
	}
	return code
}

// hasState checks whether a trie state exists for the given root hash
// in either the Merkle or Verkle database.
func (s *stateDatabase) hasState(root common.Hash) bool {
	_, err := s.merkle.NodeReader(root)
	if err == nil {
		return true
	}
	_, err = s.verkle.NodeReader(root)
	return err == nil
}

// close terminates the state database and closes two internal trie databases.
func (s *stateDatabase) close() error {
	var errs []error
	if err := s.merkle.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := s.verkle.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}
