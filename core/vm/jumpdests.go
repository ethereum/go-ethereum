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

package vm

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/metrics"
)

var jumpDestLocalHitMeter = metrics.NewRegisteredMeter("chain/cache/jumpdest/local/hit", nil)

// JumpDestCache represents the cache of jumpdest analysis results.
type JumpDestCache interface {
	// Load retrieves the cached jumpdest analysis for the given code hash.
	// Returns the BitVec and true if found, or nil and false if not cached.
	Load(codeHash common.Hash) (BitVec, bool)

	// Store saves the jumpdest analysis for the given code hash.
	Store(codeHash common.Hash, vec BitVec)
}

// mapJumpDests is the default implementation of JumpDests using a map.
// This implementation is not thread-safe and is meant to be used per EVM instance.
type mapJumpDests map[common.Hash]BitVec

// newMapJumpDests creates a new map-based JumpDests implementation.
func newMapJumpDests() JumpDestCache {
	return make(mapJumpDests)
}

func (j mapJumpDests) Load(codeHash common.Hash) (BitVec, bool) {
	vec, ok := j[codeHash]
	return vec, ok
}

func (j mapJumpDests) Store(codeHash common.Hash, vec BitVec) {
	j[codeHash] = vec
}

// layeredJumpDests sits in front of a shared JumpDestCache.
// During a single tx, an eviction from the shared cache can never cause the
// same code to be analyzed twice. The local layer is reset on every new tx.
// Thus the jumpdest analysis gets charged correctly, once per cold account load.
type layeredJumpDests struct {
	local  mapJumpDests
	shared JumpDestCache
}

// newLayeredJumpDests creates a per-transaction layer on top of the shared cache.
func newLayeredJumpDests(shared JumpDestCache) *layeredJumpDests {
	return &layeredJumpDests{local: make(mapJumpDests), shared: shared}
}

// reset drops the analyses retained by the local layer.
func (j *layeredJumpDests) reset() {
	clear(j.local)
}

func (j *layeredJumpDests) Load(codeHash common.Hash) (BitVec, bool) {
	if vec, ok := j.local[codeHash]; ok {
		jumpDestLocalHitMeter.Mark(1)
		return vec, true
	}
	vec, ok := j.shared.Load(codeHash)
	if ok {
		j.local[codeHash] = vec
	}
	return vec, ok
}

func (j *layeredJumpDests) Store(codeHash common.Hash, vec BitVec) {
	j.local[codeHash] = vec
	j.shared.Store(codeHash, vec)
}
