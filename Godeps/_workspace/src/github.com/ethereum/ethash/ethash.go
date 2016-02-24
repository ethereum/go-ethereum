// Copyright 2015 The go-ethereum Authors
// Copyright 2015 Lefteris Karapetsas <lefteris@refu.co>
// Copyright 2015 Matthew Wampler-Doty <matthew.wampler.doty@gmail.com>
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

package ethash

/*
#include "src/libethash/internal.h"

int ethashGoCallback_cgo(unsigned);
*/
import "C"

import (
	"errors"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/pow"
)

var (
	maxUint256  = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
	sharedLight = new(Light)
)

const (
	epochLength         uint64     = 30000
	cacheSizeForTesting C.uint64_t = 1024
	dagSizeForTesting   C.uint64_t = 1024 * 32
)

var DefaultDir = defaultDir()

func defaultDir() string {
	home := os.Getenv("HOME")
	if user, err := user.Current(); err == nil {
		home = user.HomeDir
	}
	if runtime.GOOS == "windows" {
		return filepath.Join(home, "AppData", "Ethash")
	}
	return filepath.Join(home, ".ethash")
}

// cache wraps an ethash_light_t with some metadata
// and automatic memory management.
type cache struct {
	epoch uint64
	used  time.Time
	test  bool

	gen sync.Once // ensures cache is only generated once.
	ptr *C.struct_ethash_light
}

// generate creates the actual cache. it can be called from multiple
// goroutines. the first call will generate the cache, subsequent
// calls wait until it is generated.
func (cache *cache) generate() {
	cache.gen.Do(func() {
		started := time.Now()
		seedHash := makeSeedHash(cache.epoch)
		glog.V(logger.Debug).Infof("Generating cache for epoch %d (%x)", cache.epoch, seedHash)
		size := C.ethash_get_cachesize(C.uint64_t(cache.epoch * epochLength))
		if cache.test {
			size = cacheSizeForTesting
		}
		cache.ptr = C.ethash_light_new_internal(size, (*C.ethash_h256_t)(unsafe.Pointer(&seedHash[0])))
		runtime.SetFinalizer(cache, freeCache)
		glog.V(logger.Debug).Infof("Done generating cache for epoch %d, it took %v", cache.epoch, time.Since(started))
	})
}

func freeCache(cache *cache) {
	C.ethash_light_delete(cache.ptr)
	cache.ptr = nil
}

// Light implements the Verify half of the proof of work. It uses a few small
// in-memory caches to verify the nonces found by Full.
type Light struct {
	test      bool              // if set use a smaller cache size
	mu        sync.Mutex        // protects the per-epoch map of DAGs
	caches    map[uint64]*cache // currently cached verification DAGs
	NumCaches int               // Maximum number of DAGs to cache before eviction (only init, don't modify)
}

// Verify checks whether the block's nonce is valid.
func (l *Light) Verify(block pow.Block) bool {
	// TODO: do ethash_quick_verify before getCache in order
	// to prevent DOS attacks.
	blockNum := block.NumberU64()
	if blockNum >= epochLength*2048 {
		glog.V(logger.Debug).Infof("block number %d too high, limit is %d", epochLength*2048)
		return false
	}

	difficulty := block.Difficulty()
	/* Cannot happen if block header diff is validated prior to PoW, but can
		 happen if PoW is checked first due to parallel PoW checking.
		 We could check the minimum valid difficulty but for SoC we avoid (duplicating)
	   Ethereum protocol consensus rules here which are not in scope of Ethash
	*/
	if difficulty.Cmp(common.Big0) == 0 {
		glog.V(logger.Debug).Infof("invalid block difficulty")
		return false
	}

	cache := l.getCache(blockNum)
	dagSize := C.ethash_get_datasize(C.uint64_t(blockNum))

	if l.test {
		dagSize = dagSizeForTesting
	}
	// Recompute the hash using the cache.
	hash := hashToH256(block.HashNoNonce())
	ret := C.ethash_light_compute_internal(cache.ptr, dagSize, hash, C.uint64_t(block.Nonce()))
	if !ret.success {
		return false
	}

	// avoid mixdigest malleability as it's not included in a block's "hashNononce"
	if block.MixDigest() != h256ToHash(ret.mix_hash) {
		return false
	}

	// Make sure cache is live until after the C call.
	// This is important because a GC might happen and execute
	// the finalizer before the call completes.
	_ = cache
	// The actual check.
	target := new(big.Int).Div(maxUint256, difficulty)
	return h256ToHash(ret.result).Big().Cmp(target) <= 0
}

func h256ToHash(in C.ethash_h256_t) common.Hash {
	return *(*common.Hash)(unsafe.Pointer(&in.b))
}

func hashToH256(in common.Hash) C.ethash_h256_t {
	return C.ethash_h256_t{b: *(*[32]C.uint8_t)(unsafe.Pointer(&in[0]))}
}

func (l *Light) getCache(blockNum uint64) *cache {
	var c *cache
	epoch := blockNum / epochLength

	// If we have a PoW for that epoch, use that
	l.mu.Lock()
	if l.caches == nil {
		l.caches = make(map[uint64]*cache)
	}
	if l.NumCaches == 0 {
		l.NumCaches = 3
	}
	c = l.caches[epoch]
	if c == nil {
		// No cached DAG, evict the oldest if the cache limit was reached
		if len(l.caches) >= l.NumCaches {
			var evict *cache
			for _, cache := range l.caches {
				if evict == nil || evict.used.After(cache.used) {
					evict = cache
				}
			}
			glog.V(logger.Info).Infof("Evicting DAG for epoch %d in favour of epoch %d", evict.epoch, epoch)
			delete(l.caches, evict.epoch)
		}
		// Create and return a new DAG for the epoch
		c = &cache{epoch: epoch, test: l.test}
		l.caches[epoch] = c
	}
	c.used = time.Now()
	l.mu.Unlock()

	// Wait for generation finish and return the cache
	c.generate()
	return c
}

// dag wraps an ethash_full_t with some metadata
// and automatic memory management.
type dag struct {
	epoch uint64
	test  bool
	dir   string

	gen sync.Once // ensures DAG is only generated once.
	ptr *C.struct_ethash_full
}

// generate creates the actual DAG. it can be called from multiple
// goroutines. the first call will generate the DAG, subsequent
// calls wait until it is generated.
func (d *dag) generate() {
	d.gen.Do(func() {
		var (
			started   = time.Now()
			seedHash  = makeSeedHash(d.epoch)
			blockNum  = C.uint64_t(d.epoch * epochLength)
			cacheSize = C.ethash_get_cachesize(blockNum)
			dagSize   = C.ethash_get_datasize(blockNum)
		)
		if d.test {
			cacheSize = cacheSizeForTesting
			dagSize = dagSizeForTesting
		}
		if d.dir == "" {
			d.dir = DefaultDir
		}
		glog.V(logger.Info).Infof("Generating DAG for epoch %d (size %d) (%x)", d.epoch, dagSize, seedHash)
		// Generate a temporary cache.
		// TODO: this could share the cache with Light
		cache := C.ethash_light_new_internal(cacheSize, (*C.ethash_h256_t)(unsafe.Pointer(&seedHash[0])))
		defer C.ethash_light_delete(cache)
		// Generate the actual DAG.
		d.ptr = C.ethash_full_new_internal(
			C.CString(d.dir),
			hashToH256(seedHash),
			dagSize,
			cache,
			(C.ethash_callback_t)(unsafe.Pointer(C.ethashGoCallback_cgo)),
		)
		if d.ptr == nil {
			panic("ethash_full_new IO or memory error")
		}
		runtime.SetFinalizer(d, freeDAG)
		glog.V(logger.Info).Infof("Done generating DAG for epoch %d, it took %v", d.epoch, time.Since(started))
	})
}

func freeDAG(d *dag) {
	C.ethash_full_delete(d.ptr)
	d.ptr = nil
}

func (d *dag) Ptr() unsafe.Pointer {
	return unsafe.Pointer(d.ptr.data)
}

//export ethashGoCallback
func ethashGoCallback(percent C.unsigned) C.int {
	glog.V(logger.Info).Infof("Generating DAG: %d%%", percent)
	return 0
}

// MakeDAG pre-generates a DAG file for the given block number in the
// given directory. If dir is the empty string, the default directory
// is used.
func MakeDAG(blockNum uint64, dir string) error {
	d := &dag{epoch: blockNum / epochLength, dir: dir}
	if blockNum >= epochLength*2048 {
		return fmt.Errorf("block number too high, limit is %d", epochLength*2048)
	}
	d.generate()
	if d.ptr == nil {
		return errors.New("failed")
	}
	return nil
}

// Full implements the Search half of the proof of work.
type Full struct {
	Dir string // use this to specify a non-default DAG directory

	test     bool // if set use a smaller DAG size
	turbo    bool
	hashRate int32

	mu      sync.Mutex // protects dag
	current *dag       // current full DAG
}

func (pow *Full) getDAG(blockNum uint64) (d *dag) {
	epoch := blockNum / epochLength
	pow.mu.Lock()
	if pow.current != nil && pow.current.epoch == epoch {
		d = pow.current
	} else {
		d = &dag{epoch: epoch, test: pow.test, dir: pow.Dir}
		pow.current = d
	}
	pow.mu.Unlock()
	// wait for it to finish generating.
	d.generate()
	return d
}

func (pow *Full) Search(block pow.Block, stop <-chan struct{}, index int) (nonce uint64, mixDigest []byte) {
	dag := pow.getDAG(block.NumberU64())

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	diff := block.Difficulty()

	i := int64(0)
	starti := i
	start := time.Now().UnixNano()
	previousHashrate := int32(0)

	nonce = uint64(r.Int63())
	hash := hashToH256(block.HashNoNonce())
	target := new(big.Int).Div(maxUint256, diff)
	for {
		select {
		case <-stop:
			atomic.AddInt32(&pow.hashRate, -previousHashrate)
			return 0, nil
		default:
			i++

			// we don't have to update hash rate on every nonce, so update after
			// first nonce check and then after 2^X nonces
			if i == 2 || ((i % (1 << 16)) == 0) {
				elapsed := time.Now().UnixNano() - start
				hashes := (float64(1e9) / float64(elapsed)) * float64(i-starti)
				hashrateDiff := int32(hashes) - previousHashrate
				previousHashrate = int32(hashes)
				atomic.AddInt32(&pow.hashRate, hashrateDiff)
			}

			ret := C.ethash_full_compute(dag.ptr, hash, C.uint64_t(nonce))
			result := h256ToHash(ret.result).Big()

			// TODO: disagrees with the spec https://github.com/ethereum/wiki/wiki/Ethash#mining
			if ret.success && result.Cmp(target) <= 0 {
				mixDigest = C.GoBytes(unsafe.Pointer(&ret.mix_hash), C.int(32))
				atomic.AddInt32(&pow.hashRate, -previousHashrate)
				return nonce, mixDigest
			}
			nonce += 1
		}

		if !pow.turbo {
			time.Sleep(20 * time.Microsecond)
		}
	}
}

func (pow *Full) GetHashrate() int64 {
	return int64(atomic.LoadInt32(&pow.hashRate))
}

func (pow *Full) Turbo(on bool) {
	// TODO: this needs to use an atomic operation.
	pow.turbo = on
}

// Ethash combines block verification with Light and
// nonce searching with Full into a single proof of work.
type Ethash struct {
	*Light
	*Full
}

// New creates an instance of the proof of work.
func New() *Ethash {
	return &Ethash{new(Light), &Full{turbo: true}}
}

// NewShared creates an instance of the proof of work., where a single instance
// of the Light cache is shared across all instances created with NewShared.
func NewShared() *Ethash {
	return &Ethash{sharedLight, &Full{turbo: true}}
}

// NewForTesting creates a proof of work for use in unit tests.
// It uses a smaller DAG and cache size to keep test times low.
// DAG files are stored in a temporary directory.
//
// Nonces found by a testing instance are not verifiable with a
// regular-size cache.
func NewForTesting() (*Ethash, error) {
	dir, err := ioutil.TempDir("", "ethash-test")
	if err != nil {
		return nil, err
	}
	return &Ethash{&Light{test: true}, &Full{Dir: dir, test: true}}, nil
}

func GetSeedHash(blockNum uint64) ([]byte, error) {
	if blockNum >= epochLength*2048 {
		return nil, fmt.Errorf("block number too high, limit is %d", epochLength*2048)
	}
	sh := makeSeedHash(blockNum / epochLength)
	return sh[:], nil
}

func makeSeedHash(epoch uint64) (sh common.Hash) {
	for ; epoch > 0; epoch-- {
		sh = crypto.Sha3Hash(sh[:])
	}
	return sh
}
