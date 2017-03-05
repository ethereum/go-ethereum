// Copyright 2017 The go-ethereum Authors
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

package pow

import (
	"bytes"
	"errors"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	metrics "github.com/rcrowley/go-metrics"
)

var (
	ErrInvalidDifficulty = errors.New("non-positive difficulty")
	ErrInvalidMixDigest  = errors.New("invalid mix digest")
	ErrInvalidPoW        = errors.New("pow difficulty invalid")
)

var (
	// maxUint256 is a big integer representing 2^256-1
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
)

// cache wraps an ethash cache with some metadata to allow easier concurrent use.
type cache struct {
	epoch uint64     // Epoch for which this cache is relevant
	cache []uint32   // The actual cache data content
	used  time.Time  // Timestamp of the last use for smarter eviction
	once  sync.Once  // Ensures the cache is generated only once
	lock  sync.Mutex // Ensures thread safety for updating the usage time
}

// generate ensures that the cache content is generates.
func (c *cache) generate(test bool) {
	c.once.Do(func() {
		cacheSize := cacheSize(c.epoch*epochLength + 1)
		if test {
			cacheSize = 1024
		}
		rawCache := generateCache(cacheSize, seedHash(c.epoch*epochLength+1))
		c.cache = prepare(uint64(len(rawCache)), bytes.NewReader(rawCache))
	})
}

// Ethash is a PoW data struture implementing the ethash algorithm.
type Ethash struct {
	cachedir string // Data directory to store the verification caches
	dagdir   string // Data directory to store full mining datasets

	caches map[uint64]*cache // In memory caches to avoid regenerating too often
	future *cache            // Pre-generated cache for the estimated future epoch
	lock   sync.Mutex        // Ensures thread safety for the in-memory caches

	hashrate *metrics.StandardMeter // Meter tracking the average hashrate

	tester bool // Flag whether to use a smaller test dataset
	shared bool // Flag whether to use a global chared dataset
}

// NewFullEthash creates a full sized ethash PoW scheme.
func NewFullEthash(cachedir, dagdir string) PoW {
	return &Ethash{
		cachedir: cachedir,
		dagdir:   dagdir,
		caches:   make(map[uint64]*cache),
	}
}

// NewTestEthash creates a small sized ethash PoW scheme useful only for testing
// purposes.
func NewTestEthash() PoW {
	return &Ethash{
		caches: make(map[uint64]*cache),
		tester: true,
	}
}

// NewSharedEthash creates a full sized ethash PoW shared between all requesters
// running in the same process.
func NewSharedEthash() PoW {
	return &Ethash{
		caches: make(map[uint64]*cache),
		shared: true,
	}
}

// Verify implements PoW, checking whether the given block satisfies the PoW
// difficulty requirements.
func (ethash *Ethash) Verify(block Block) error {
	// Ensure twe have a valid difficulty for the block
	difficulty := block.Difficulty()
	if difficulty.Sign() <= 0 {
		return ErrInvalidDifficulty
	}
	// Recompute the digest and PoW value and verify against the block
	number := block.NumberU64()
	cache := ethash.cache(number)

	size := datasetSize(number)
	if ethash.tester {
		size = 32 * 1024
	}
	digest, result := hashimotoLight(size, cache, block.HashNoNonce().Bytes(), block.Nonce())
	if !bytes.Equal(block.MixDigest().Bytes(), digest) {
		return ErrInvalidMixDigest
	}
	target := new(big.Int).Div(maxUint256, difficulty)
	if new(big.Int).SetBytes(result).Cmp(target) > 0 {
		return ErrInvalidPoW
	}
	return nil
}

// cache tries to retrieve a verification cache for the specified block number
// by first checking against a list of in-memory caches, then against caches
// stored on disk, and finally generating one if none can be found.
func (ethash *Ethash) cache(block uint64) []uint32 {
	epoch := block / epochLength

	// If we have a PoW for that epoch, use that
	ethash.lock.Lock()

	current, future := ethash.caches[epoch], (*cache)(nil)
	if current == nil {
		// No in-memory cache, evict the oldest if the cache limit was reached
		for len(ethash.caches) >= 3 {
			var evict *cache
			for _, cache := range ethash.caches {
				if evict == nil || evict.used.After(cache.used) {
					evict = cache
				}
			}
			delete(ethash.caches, evict.epoch)

			log.Debug("Evictinged ethash cache", "old", evict.epoch, "used", evict.used)
		}
		// If we have the new cache pre-generated, use that, otherwise create a new one
		if ethash.future != nil && ethash.future.epoch == epoch {
			log.Debug("Using pre-generated cache", "epoch", epoch)
			current, ethash.future = ethash.future, nil
		} else {
			log.Debug("Generating new ethash cache", "epoch", epoch)
			current = &cache{epoch: epoch}
		}
		ethash.caches[epoch] = current

		// If we just used up the future cache, or need a refresh, regenerate
		if ethash.future == nil || ethash.future.epoch <= epoch {
			log.Debug("Pre-generating cache for the future", "epoch", epoch+1)
			future = &cache{epoch: epoch + 1}
			ethash.future = future
		}
	}
	current.used = time.Now()
	ethash.lock.Unlock()

	// Wait for generation finish, bump the timestamp and finalize the cache
	current.once.Do(func() {
		current.generate(ethash.tester)
	})
	current.lock.Lock()
	current.used = time.Now()
	current.lock.Unlock()

	// If we exhusted the future cache, now's a goot time to regenerate it
	if future != nil {
		go future.generate(ethash.tester)
	}
	return current.cache
}

// Search implements PoW, attempting to find a nonce that satisfies the block's
// difficulty requirements.
func (ethash *Ethash) Search(block Block, stop <-chan struct{}) (uint64, []byte) {
	return 0, nil
}

// Hashrate implements PoW, returning the measured rate of the search invocations
// per second over the last minute.
func (ethash *Ethash) Hashrate() float64 {
	return ethash.hashrate.Rate1()
}

// EthashSeedHash is the seed to use for generating a vrification cache and the
// mining dataset.
func EthashSeedHash(block uint64) []byte {
	return seedHash(block)
}
