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
	"math"
	"math/big"
	"math/rand"
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
	ErrNonceOutOfRange   = errors.New("nonce out of range")
	ErrInvalidDifficulty = errors.New("non-positive difficulty")
	ErrInvalidMixDigest  = errors.New("invalid mix digest")
	ErrInvalidPoW        = errors.New("pow difficulty invalid")
)

var (
	// maxUint256 is a big integer representing 2^256-1
	maxUint256 = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

	// sharedEthash is a full instance that can be shared between multiple users.
	sharedEthash = NewFullEthash("", 3, 0, "", 1, 0)

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

// generate ensures that the cache content is generated before use.
func (c *cache) generate(dir string, limit int, test bool) {
	c.once.Do(func() {
		// If we have a testing cache, generate and return
		if test {
			rawCache := generateCache(1024, seedHash(c.epoch*epochLength+1))
			c.cache = prepare(1024, bytes.NewReader(rawCache))
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

// dataset wraps an ethash dataset with some metadata to allow easier concurrent use.
type dataset struct {
	epoch   uint64     // Epoch for which this cache is relevant
	dataset []uint32   // The actual cache data content
	used    time.Time  // Timestamp of the last use for smarter eviction
	once    sync.Once  // Ensures the cache is generated only once
	lock    sync.Mutex // Ensures thread safety for updating the usage time
}

// generate ensures that the dataset content is generated before use.
func (d *dataset) generate(dir string, limit int, test bool, discard bool) {
	d.once.Do(func() {
		// If we have a testing dataset, generate and return
		if test {
			rawCache := generateCache(1024, seedHash(d.epoch*epochLength+1))
			intCache := prepare(1024, bytes.NewReader(rawCache))

			rawDataset := generateDataset(32*1024, intCache)
			d.dataset = prepare(32*1024, bytes.NewReader(rawDataset))

			return
		}
		// Full dataset generation is needed, check dataset dir for existing data
		csize := cacheSize(d.epoch*epochLength + 1)
		dsize := datasetSize(d.epoch*epochLength + 1)
		seed := seedHash(d.epoch*epochLength + 1)

		path := filepath.Join(dir, fmt.Sprintf("full-R%d-%x", algorithmRevision, seed))
		logger := log.New("seed", hexutil.Bytes(seed))

		if dir != "" {
			dump, err := os.Open(path)
			if err == nil {
				if !discard {
					logger.Info("Loading ethash DAG from disk")
					start := time.Now()
					d.dataset = prepare(dsize, bufio.NewReader(dump))
					logger.Info("Loaded ethash DAG from disk", "elapsed", common.PrettyDuration(time.Since(start)))
				}
				dump.Close()
				return
			}
		}
		// No previous disk dataset was available, generate on the fly
		rawCache := generateCache(csize, seed)
		intCache := prepare(csize, bytes.NewReader(rawCache))

		rawDataset := generateDataset(dsize, intCache)
		if !discard {
			d.dataset = prepare(dsize, bytes.NewReader(rawDataset))
		}
		// If a dataset directory is given, attempt to serialize for next time
		if dir != "" {
			// Store the ethash dataset to disk
			start := time.Now()
			if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
				logger.Error("Failed to create ethash DAG dir", "err", err)
			} else if err := ioutil.WriteFile(path, rawDataset, os.ModePerm); err != nil {
				logger.Error("Failed to write ethash DAG to disk", "err", err)
			} else {
				logger.Info("Stored ethash DAG to disk", "elapsed", common.PrettyDuration(time.Since(start)))
			}
			// Iterate over all previous instances and delete old ones
			for ep := int(d.epoch) - limit; ep >= 0; ep-- {
				seed := seedHash(uint64(ep)*epochLength + 1)
				path := filepath.Join(dir, fmt.Sprintf("full-R%d-%x", algorithmRevision, seed))
				os.Remove(path)
			}
		}
	})
}

// MakeCache generates a new ethash cache and optionally stores it to disk.
func MakeCache(block uint64, dir string) {
	c := cache{epoch: block/epochLength + 1}
	c.generate(dir, math.MaxInt32, false)
}

// MakeDataset generates a new ethash dataset and optionally stores it to disk.
func MakeDataset(block uint64, dir string) {
	d := dataset{epoch: block/epochLength + 1}
	d.generate(dir, math.MaxInt32, false, true)
}

// Ethash is a PoW data struture implementing the ethash algorithm.
type Ethash struct {
	cachedir     string // Data directory to store the verification caches
	cachesinmem  int    // Number of caches to keep in memory
	cachesondisk int    // Number of caches to keep on disk
	dagdir       string // Data directory to store full mining datasets
	dagsinmem    int    // Number of mining datasets to keep in memory
	dagsondisk   int    // Number of mining datasets to keep on disk

	caches   map[uint64]*cache   // In memory caches to avoid regenerating too often
	fcache   *cache              // Pre-generated cache for the estimated future epoch
	datasets map[uint64]*dataset // In memory datasets to avoid regenerating too often
	fdataset *dataset            // Pre-generated dataset for the estimated future epoch
	lock     sync.Mutex          // Ensures thread safety for the in-memory caches

	hashrate metrics.Meter // Meter tracking the average hashrate

	tester bool // Flag whether to use a smaller test dataset
}

// NewFullEthash creates a full sized ethash PoW scheme.
func NewFullEthash(cachedir string, cachesinmem, cachesondisk int, dagdir string, dagsinmem, dagsondisk int) PoW {
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
		dagsinmem:    dagsinmem,
		dagsondisk:   dagsondisk,
		caches:       make(map[uint64]*cache),
		datasets:     make(map[uint64]*dataset),
		hashrate:     metrics.NewMeter(),
	}
}

// NewTestEthash creates a small sized ethash PoW scheme useful only for testing
// purposes.
func NewTestEthash() PoW {
	return &Ethash{
		cachesinmem: 1,
		caches:      make(map[uint64]*cache),
		datasets:    make(map[uint64]*dataset),
		tester:      true,
		hashrate:    metrics.NewMeter(),
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
	// Sanity check that the block number is below the lookup table size (60M blocks)
	number := block.NumberU64()
	if number/epochLength >= uint64(len(cacheSizes)) {
		// Go < 1.7 cannot calculate new cache/dataset sizes (no fast prime check)
		return ErrNonceOutOfRange
	}
	// Ensure that we have a valid difficulty for the block
	difficulty := block.Difficulty()
	if difficulty.Sign() <= 0 {
		return ErrInvalidDifficulty
	}
	// Recompute the digest and PoW value and verify against the block
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
		if ethash.fcache != nil && ethash.fcache.epoch == epoch {
			log.Debug("Using pre-generated cache", "epoch", epoch)
			current, ethash.fcache = ethash.fcache, nil
		} else {
			log.Debug("Requiring new ethash cache", "epoch", epoch)
			current = &cache{epoch: epoch}
		}
		ethash.caches[epoch] = current

		// If we just used up the future cache, or need a refresh, regenerate
		if ethash.fcache == nil || ethash.fcache.epoch <= epoch {
			log.Debug("Requiring new future ethash cache", "epoch", epoch+1)
			future = &cache{epoch: epoch + 1}
			ethash.fcache = future
		}
	}
	current.used = time.Now()
	ethash.lock.Unlock()

	// Wait for generation finish, bump the timestamp and finalize the cache
	current.generate(ethash.cachedir, ethash.cachesondisk, ethash.tester)

	current.lock.Lock()
	current.used = time.Now()
	current.lock.Unlock()

	// If we exhausted the future cache, now's a good time to regenerate it
	if future != nil {
		go future.generate(ethash.cachedir, ethash.cachesondisk, ethash.tester)
	}
	return current.cache
}

// Search implements PoW, attempting to find a nonce that satisfies the block's
// difficulty requirements.
func (ethash *Ethash) Search(block Block, stop <-chan struct{}) (uint64, []byte) {
	// Extract some data from the block
	var (
		hash   = block.HashNoNonce().Bytes()
		diff   = block.Difficulty()
		target = new(big.Int).Div(maxUint256, diff)
	)
	// Retrieve the mining dataset
	dataset, size := ethash.dataset(block.NumberU64()), datasetSize(block.NumberU64())

	// Start generating random nonces until we abort or find a good one
	var (
		attempts int64

		rand  = rand.New(rand.NewSource(time.Now().UnixNano()))
		nonce = uint64(rand.Int63())
	)
	for {
		select {
		case <-stop:
			// Mining terminated, update stats and abort
			ethash.hashrate.Mark(attempts)
			return 0, nil

		default:
			// We don't have to update hash rate on every nonce, so update after after 2^X nonces
			attempts++
			if (attempts % (1 << 15)) == 0 {
				ethash.hashrate.Mark(attempts)
				attempts = 0
			}
			// Compute the PoW value of this nonce
			digest, result := hashimotoFull(size, dataset, hash, nonce)
			if new(big.Int).SetBytes(result).Cmp(target) <= 0 {
				return nonce, digest
			}
			nonce++
		}
	}
}

// dataset tries to retrieve a mining dataset for the specified block number
// by first checking against a list of in-memory datasets, then against DAGs
// stored on disk, and finally generating one if none can be found.
func (ethash *Ethash) dataset(block uint64) []uint32 {
	epoch := block / epochLength

	// If we have a PoW for that epoch, use that
	ethash.lock.Lock()

	current, future := ethash.datasets[epoch], (*dataset)(nil)
	if current == nil {
		// No in-memory dataset, evict the oldest if the dataset limit was reached
		for len(ethash.datasets) >= ethash.dagsinmem {
			var evict *dataset
			for _, dataset := range ethash.datasets {
				if evict == nil || evict.used.After(dataset.used) {
					evict = dataset
				}
			}
			delete(ethash.datasets, evict.epoch)

			log.Debug("Evicted ethash dataset", "epoch", evict.epoch, "used", evict.used)
		}
		// If we have the new cache pre-generated, use that, otherwise create a new one
		if ethash.fdataset != nil && ethash.fdataset.epoch == epoch {
			log.Debug("Using pre-generated dataset", "epoch", epoch)
			current = &dataset{epoch: ethash.fdataset.epoch} // Reload from disk
			ethash.fdataset = nil
		} else {
			log.Debug("Requiring new ethash dataset", "epoch", epoch)
			current = &dataset{epoch: epoch}
		}
		ethash.datasets[epoch] = current

		// If we just used up the future dataset, or need a refresh, regenerate
		if ethash.fdataset == nil || ethash.fdataset.epoch <= epoch {
			log.Debug("Requiring new future ethash dataset", "epoch", epoch+1)
			future = &dataset{epoch: epoch + 1}
			ethash.fdataset = future
		}
	}
	current.used = time.Now()
	ethash.lock.Unlock()

	// Wait for generation finish, bump the timestamp and finalize the cache
	current.generate(ethash.dagdir, ethash.dagsondisk, ethash.tester, false)

	current.lock.Lock()
	current.used = time.Now()
	current.lock.Unlock()

	// If we exhausted the future dataset, now's a good time to regenerate it
	if future != nil {
		go future.generate(ethash.dagdir, ethash.dagsondisk, ethash.tester, true) // Discard results from memorys
	}
	return current.dataset
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
