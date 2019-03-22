// Copyright 2019 The go-ethereum Authors
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
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

// For EVM execution, we need around 256 items. We add a few more to allow reorgs.
// For mainnet, a couple more would suffice, but a few more added for
// testnets/private nets
// For LES, we use a larger buffer. 500K * 32 bytes = 16M
const hashBufferElems = 500000

var (
	hashHitCounter  = metrics.NewRegisteredGauge("chain/headerhash/hit", nil)
	hashMissCounter = metrics.NewRegisteredGauge("chain/headerhash/miss", nil)
	hashHeadGauge   = metrics.NewRegisteredGauge("chain/headerhash/head", nil)
	hashTailGauge   = metrics.NewRegisteredGauge("chain/headerhash/tail", nil)
)

// hashBuffer implements a storage for chains of hashes, intended to be used for quick lookup of block hashes.
// Internally, it uses an array of hashes in a circular buffer.
// It enforces that all hashes added have a contiguous parent-child relation, and supports rollbacks
// It is thread-safe.
type hashBuffer struct {
	// The data holds the hashes. The hashes are sequential, but also a
	// circular buffer.
	// The `head` points to the position of the latest hash.
	// The parent, if present, is located 32 bytes back.
	// [.., .., ..., head-2 , head-1, head, oldest, ... ]
	data [hashBufferElems]common.Hash

	head uint64 // index of hash for the head block

	headNumber uint64 // The block number for head (the most recent block)
	tailNumber uint64 // The block number for tail (the oldest block)

	mu sync.RWMutex
}

// newHashBuffer creates a new storage with a header in it.
// Since we take a header here, a hash storage can never be empty.
// This makes things easier later on (in Set)
func newHashBuffer(header *types.Header) *hashBuffer {
	return &hashBuffer{
		headNumber: header.Number.Uint64(),
		tailNumber: header.Number.Uint64(),
		data:       [hashBufferElems]common.Hash{header.Hash()},
	}
}

// Get locates the hash for the requested number
func (hs *hashBuffer) Get(number uint64) (common.Hash, bool) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	if !hs.has(number) {
		hashMissCounter.Inc(1)
		return common.Hash{}, false
	}
	hashHitCounter.Inc(1)
	distance := hs.headNumber - number
	index := (hs.head + hashBufferElems - distance) % hashBufferElems
	return hs.data[index], true
}

// has returns if the storage has a hash for the given number
func (hs *hashBuffer) has(number uint64) bool {
	return number <= hs.headNumber && number >= hs.tailNumber
}

// Contains checks if the hash at the given number matches the expected
func (hs *hashBuffer) Contains(number uint64, expected common.Hash) bool {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	if hs.contains(number, expected) {
		hashHitCounter.Inc(1)
		return true
	} else {
		hashMissCounter.Inc(1)
		return false
	}
}

// contains is the non-concurrency safe internal version of Contains
func (hs *hashBuffer) contains(number uint64, expected common.Hash) bool {
	if !hs.has(number) {
		return false
	}
	distance := hs.headNumber - number
	index := (hs.head + hashBufferElems - distance) % hashBufferElems
	return hs.data[index] == expected
}

// Newest returns the most recent (number, hash) stored
func (hs *hashBuffer) Newest() (uint64, common.Hash) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	return hs.headNumber, hs.data[hs.head]
}

// Oldest returns the oldest (number, hash) found
func (hs *hashBuffer) Oldest() (uint64, common.Hash) {
	hs.mu.RLock()
	defer hs.mu.RUnlock()
	distance := hs.headNumber - hs.tailNumber
	index := (hs.head + hashBufferElems - distance) % hashBufferElems
	return hs.tailNumber, hs.data[index]
}

// Set inserts a new header (hash) to the storage.
// If
// a) Header already exists, this is a no-op
// b) Number is occupied by other header, the new header replaces it, and also
// truncates any descendants
//
// If the new header does not have any ancestors, it replaces the entire storage.
func (hs *hashBuffer) Set(header *types.Header) {
	var (
		number = header.Number.Uint64()
		index  uint64
		hash   = header.Hash()
	)
	hs.mu.Lock()
	defer hs.mu.Unlock()
	if hs.contains(number-1, header.ParentHash) {
		if hs.headNumber >= number {
			distance := hs.headNumber - number
			index = (hs.head + hashBufferElems - distance) % hashBufferElems
			if hs.data[index] == hash {
				return
			}
			// Continue by replacing this number and wipe descendants
		} else {
			// head is parent of this new header - regular append
			index = (hs.head + 1) % hashBufferElems
		}
	} else {
		// This should not normally happen, and indicates a programming error
		log.Error("Hash storage wiping ancestors", "oldhead", hs.headNumber, "newhead", number, "oldtail", hs.tailNumber)
		// Wipe ancestors
		hs.tailNumber = number
	}
	hs.head = index
	hs.headNumber = number
	hs.data[hs.head] = hash

	if number-hs.tailNumber == hashBufferElems {
		// It's full, need to move the tail
		hs.tailNumber++
	}
	hashTailGauge.Update(int64(hs.tailNumber))
	hashHeadGauge.Update(int64(hs.headNumber))
}
