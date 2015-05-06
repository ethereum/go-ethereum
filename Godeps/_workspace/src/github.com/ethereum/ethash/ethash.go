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
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/pow"
)

var (
	minDifficulty = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))
	sharedLight   = new(Light)
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

// Light implements the Verify half of the proof of work.
// It uses a small in-memory cache to verify the nonces
// found by Full.
type Light struct {
	test    bool       // if set use a smaller cache size
	mu      sync.Mutex // protects current
	current *cache     // last cache which was generated.
	// TODO: keep multiple caches.
}

// Verify checks whether the block's nonce is valid.
func (l *Light) Verify(block pow.Block) bool {
	// TODO: do ethash_quick_verify before getCache in order
	// to prevent DOS attacks.
	var (
		blockNum   = block.NumberU64()
		difficulty = block.Difficulty()
		cache      = l.getCache(blockNum)
		dagSize    = C.ethash_get_datasize(C.uint64_t(blockNum))
	)
	if l.test {
		dagSize = dagSizeForTesting
	}
	if blockNum >= epochLength*2048 {
		glog.V(logger.Debug).Infof("block number %d too high, limit is %d", epochLength*2048)
		return false
	}
	// Recompute the hash using the cache.
	hash := hashToH256(block.HashNoNonce())
	ret := C.ethash_light_compute_internal(cache.ptr, dagSize, hash, C.uint64_t(block.Nonce()))
	if !ret.success {
		return false
	}
	// Make sure cache is live until after the C call.
	// This is important because a GC might happen and execute
	// the finalizer before the call completes.
	_ = cache
	// The actual check.
	target := new(big.Int).Div(minDifficulty, difficulty)
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
	// Update or reuse the last cache.
	l.mu.Lock()
	if l.current != nil && l.current.epoch == epoch {
		c = l.current
	} else {
		c = &cache{epoch: epoch, test: l.test}
		l.current = c
	}
	l.mu.Unlock()
	// Wait for the cache to finish generating.
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
		glog.V(logger.Info).Infof("Generating DAG for epoch %d (%x)", d.epoch, seedHash)
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

func freeDAG(h *dag) {
	C.ethash_full_delete(h.ptr)
	h.ptr = nil
}

//export ethashGoCallback
func ethashGoCallback(percent C.unsigned) C.int {
	glog.V(logger.Info).Infof("Still generating DAG: %d%%", percent)
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
	hashRate int64

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

func (pow *Full) Search(block pow.Block, stop <-chan struct{}) (nonce uint64, mixDigest []byte) {
	dag := pow.getDAG(block.NumberU64())

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	diff := block.Difficulty()

	i := int64(0)
	starti := i
	start := time.Now().UnixNano()

	nonce = uint64(r.Int63())
	hash := hashToH256(block.HashNoNonce())
	target := new(big.Int).Div(minDifficulty, diff)
	for {
		select {
		case <-stop:
			pow.hashRate = 0
			return 0, nil
		default:
			i++

			elapsed := time.Now().UnixNano() - start
			hashes := ((float64(1e9) / float64(elapsed)) * float64(i-starti)) / 1000
			pow.hashRate = int64(hashes)

			ret := C.ethash_full_compute(dag.ptr, hash, C.uint64_t(nonce))
			result := h256ToHash(ret.result).Big()

			// TODO: disagrees with the spec https://github.com/ethereum/wiki/wiki/Ethash#mining
			if ret.success && result.Cmp(target) <= 0 {
				mixDigest = C.GoBytes(unsafe.Pointer(&ret.mix_hash), C.int(32))
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
	// TODO: this needs to use an atomic operation.
	return pow.hashRate
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
// A single instance of Light is shared across all instances
// created with New.
func New() *Ethash {
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
