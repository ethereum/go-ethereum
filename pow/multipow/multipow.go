// Copyright 2016 The go-ethereum Authors
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

// Package multipow is a cached PoW using multiple instances internally.
package multipow

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/pow"
)

// singlePoW is a proof-of-work instance with an associated time value of when it
// was last used for verification.
type singlePoW struct {
	pow  pow.PoW
	used time.Time
}

// multiPoW is a proof-of-work implementation that uses multiple underlying PoW
// calculators to verify and mine blocks. This allows a higher level caching of
// PoW data structures, enabling reusing implementations that otherwise do not
// support multiple epoch caching.
//
// This is particularly important because ethash can only work with a single
// verification DAG, and will generate a new one whenever a PoW from a different
// epoch is requested. This causes huge cache trashes during epoch transitions
// as block/uncle is in different epochs, whilst parallel PoW verifications yet
// further wrosten the situation.
type multiPoW struct {
	miner     pow.PoW            // PoW implementation used solely for mining
	verifiers map[int]*singlePoW // Set of PoW verifiers by epoch cache
	lock      sync.RWMutex       // Mutex protecting the verifier cache
}

// New creates a new proof-of-work caching layer, which creates and maintains
// mutliple PoW implementations for handling epoch transitions gracefully.
func New(newpow func() pow.PoW, epochs int) pow.PoW {
	mp := &multiPoW{
		miner:     newpow(),
		verifiers: make(map[int]*singlePoW),
	}
	for i := 0; i < epochs; i++ {
		mp.verifiers[i] = &singlePoW{
			pow:  newpow(),
			used: time.Now(),
		}
	}
	return mp
}

// Verify delegates a verification request to the PoW with the appropriate
// epoch cached. If no such exists, the smallest is used.
func (mp *multiPoW) Verify(block pow.Block) bool {
	// Calculate the epoch required
	epoch := int(block.NumberU64() / params.EpochDuration.Uint64())

	// If we have a PoW for that epoch, use that
	mp.lock.Lock()

	cached := mp.verifiers[epoch]
	if cached == nil {
		// No cached verifier, reallocate the least recently used
		evict := 0
		for old, verifier := range mp.verifiers {
			if cached == nil || cached.used.After(verifier.used) {
				evict, cached = old, verifier
			}
		}
		glog.V(logger.Debug).Infof("Replacing epoch %d proof-of-work with epoch %d", evict, epoch)
		delete(mp.verifiers, evict)
		mp.verifiers[epoch] = cached
	}
	cached.used = time.Now()

	// Release the cache lock and verify the block
	mp.lock.Unlock()

	return cached.pow.Verify(block)
}

// Mining related operations simply delegate to the underlying miner PoW.
func (mp *multiPoW) Search(block pow.Block, stop <-chan struct{}, index int) (uint64, []byte) {
	return mp.miner.Search(block, stop, index)
}
func (mp *multiPoW) GetHashrate() int64 { return mp.miner.GetHashrate() }
func (mp *multiPoW) Turbo(enable bool)  { mp.miner.Turbo(enable) }
