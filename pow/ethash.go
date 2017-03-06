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
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
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

	// sharedEthash is a full instance that can be shared between multiple users.
	sharedEthash = NewFullEthash("", 3, 0, "", 0)

	// algorithmRevision is the data structure version used for file naming.
	algorithmRevision = 23

	// dumpMagic is a dataset dump header to sanity check a data dump.
	dumpMagic = hexutil.MustDecode("0xfee1deadbaddcafe")
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
func (c *cache) generate(dir string, limit int, test bool) {
	c.once.Do(func() {
		// If we have a testing cache, generate and return
		if test {
			rawCache := generateCache(1024, seedHash(c.epoch*epochLength+1))
			c.cache = prepare(uint64(len(rawCache)), bytes.NewReader(rawCache))
			return
		}
		// Full cache generation is needed, check cache dir for existing data
		size := cacheSize(c.epoch*epochLength + 1)
		seed := seedHash(c.epoch*epochLength + 1)

		path := filepath.Join(dir, fmt.Sprintf("cache-R%d-%x", algorithmRevision, seed))
		logger := log.New("seed", hexutil.Bytes(seed))

		if dir != "" {
			dump, err := os.Open(path)
			if err == nil {
				logger.Info("Loading ethash cache from disk")
				start := time.Now()
				c.cache = prepare(size, bufio.NewReader(dump))
				logger.Info("Loaded ethash cache from disk", "elapsed", common.PrettyDuration(time.Since(start)))

				dump.Close()
				return
			}
		}
		// No previous disk cache was available, generate on the fly
		rawCache := generateCache(size, seed)
		c.cache = prepare(size, bytes.NewReader(rawCache))

		// If a cache directory is given, attempt to serialize for next time
		if dir != "" {
			// Store the ethash cache to disk
			start := time.Now()
			if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
				logger.Error("Failed to create ethash cache dir", "err", err)
			} else if err := ioutil.WriteFile(path, rawCache, os.ModePerm); err != nil {
				logger.Error("Failed to write ethash cache to disk", "err", err)
			} else {
				logger.Info("Stored ethash cache to disk", "elapsed", common.PrettyDuration(time.Since(start)))
			}
			// Iterate over all previous instances and delete old ones
			for ep := int(c.epoch) - limit; ep >= 0; ep-- {
				seed := seedHash(uint64(ep)*epochLength + 1)
				path := filepath.Join(dir, fmt.Sprintf("cache-R%d-%x", algorithmRevision, seed))
				os.Remove(path)
			}
		}
	})
}

// Ethash is a PoW data struture implementing the ethash algorithm.
type Ethash struct {
	cachedir     string // Data directory to store the verification caches
	cachesinmem  int    // Number of caches to keep in memory
	cachesondisk int    // Number of caches to keep on disk
	dagdir       string // Data directory to store full mining datasets
	dagsondisk   int    // Number of mining datasets to keep on disk

	caches map[uint64]*cache // In memory caches to avoid regenerating too often
	future *cache            // Pre-generated cache for the estimated future epoch
	lock   sync.Mutex        // Ensures thread safety for the in-memory caches

	hashrate *metrics.StandardMeter // Meter tracking the average hashrate

	tester bool // Flag whether to use a smaller test dataset
}

// NewFullEthash creates a full sized ethash PoW scheme.
func NewFullEthash(cachedir string, cachesinmem, cachesondisk int, dagdir string, dagsondisk int) PoW {
	if cachesinmem <= 0 {
		log.Warn("One ethash cache must alwast be in memory", "requested", cachesinmem)
		cachesinmem = 1
	}
	if cachedir != "" && cachesondisk > 0 {
		log.Info("Disk storage enabled for ethash caches", "dir", cachedir, "count", cachesondisk)
	}
	if dagdir != "" && dagsondisk > 0 {
		log.Info("Disk storage enabled for ethash DAGs", "dir", dagdir, "count", dagsondisk)
	}
	return &Ethash{
		cachedir:     cachedir,
		cachesinmem:  cachesinmem,
		cachesondisk: cachesondisk,
		dagdir:       dagdir,
		dagsondisk:   dagsondisk,
		caches:       make(map[uint64]*cache),
	}
}

// NewTestEthash creates a small sized ethash PoW scheme useful only for testing
// purposes.
func NewTestEthash() PoW {
	return &Ethash{
		cachesinmem: 1,
		caches:      make(map[uint64]*cache),
		tester:      true,
	}
}

// NewSharedEthash creates a full sized ethash PoW shared between all requesters
// running in the same process.
func NewSharedEthash() PoW {
	return sharedEthash
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
		for len(ethash.caches) >= ethash.cachesinmem {
			var evict *cache
			for _, cache := range ethash.caches {
				if evict == nil || evict.used.After(cache.used) {
					evict = cache
				}
			}
			delete(ethash.caches, evict.epoch)

			log.Debug("Evicted ethash cache", "epoch", evict.epoch, "used", evict.used)
		}
		// If we have the new cache pre-generated, use that, otherwise create a new one
		if ethash.future != nil && ethash.future.epoch == epoch {
			log.Debug("Using pre-generated cache", "epoch", epoch)
			current, ethash.future = ethash.future, nil
		} else {
			log.Debug("Requiring new ethash cache", "epoch", epoch)
			current = &cache{epoch: epoch}
		}
		ethash.caches[epoch] = current

		// If we just used up the future cache, or need a refresh, regenerate
		if ethash.future == nil || ethash.future.epoch <= epoch {
			log.Debug("Requiring new future ethash cache", "epoch", epoch+1)
			future = &cache{epoch: epoch + 1}
			ethash.future = future
		}
	}
	current.used = time.Now()
	ethash.lock.Unlock()

	// Wait for generation finish, bump the timestamp and finalize the cache
	current.generate(ethash.cachedir, ethash.cachesondisk, ethash.tester)

	current.lock.Lock()
	current.used = time.Now()
	current.lock.Unlock()

	// If we exhusted the future cache, now's a goot time to regenerate it
	if future != nil {
		go future.generate(ethash.cachedir, ethash.cachesondisk, ethash.tester)
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
