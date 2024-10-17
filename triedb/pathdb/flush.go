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

package pathdb

import (
	"github.com/VictoriaMetrics/fastcache"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/trie/trienode"
)

// nodeCacheKey constructs the unique key of clean cache. The assumption is held
// that zero address does not have any associated storage slots.
func nodeCacheKey(owner common.Hash, path []byte) []byte {
	if owner == (common.Hash{}) {
		return path
	}
	return append(owner.Bytes(), path...)
}

// writeNodes writes the trie nodes into the provided database batch.
// Note this function will also inject all the newly written nodes
// into clean cache.
func writeNodes(batch ethdb.Batch, nodes map[common.Hash]map[string]*trienode.Node, clean *fastcache.Cache) (total int) {
	for owner, subset := range nodes {
		for path, n := range subset {
			if n.IsDeleted() {
				if owner == (common.Hash{}) {
					rawdb.DeleteAccountTrieNode(batch, []byte(path))
				} else {
					rawdb.DeleteStorageTrieNode(batch, owner, []byte(path))
				}
				if clean != nil {
					clean.Del(nodeCacheKey(owner, []byte(path)))
				}
			} else {
				if owner == (common.Hash{}) {
					rawdb.WriteAccountTrieNode(batch, []byte(path), n.Blob)
				} else {
					rawdb.WriteStorageTrieNode(batch, owner, []byte(path), n.Blob)
				}
				if clean != nil {
					clean.Set(nodeCacheKey(owner, []byte(path)), n.Blob)
				}
			}
		}
		total += len(subset)
	}
	return total
}
