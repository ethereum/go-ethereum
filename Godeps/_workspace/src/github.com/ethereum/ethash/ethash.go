/*
###################################################################################
###################################################################################
####################                                           ####################
####################  EDIT AND YOU SHALL FEEL MY WRATH - jeff  ####################
####################                                           ####################
###################################################################################
###################################################################################
*/

package ethash

/*
#cgo CFLAGS: -std=gnu99 -Wall
#cgo LDFLAGS: -lm
#include "src/libethash/util.c"
#include "src/libethash/internal.c"
#include "src/libethash/sha3.c"
#include "src/libethash/io.c"
#ifdef _WIN32
#include "src/libethash/io_win32.c"
#include "src/libethash/mmap_win32.c"
#else
#include "src/libethash/io_posix.c"
#endif
*/
import "C"

import (
	"fmt"
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

const epochLength uint64 = 30000

var minDifficulty = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

var (
	DefaultDir string = defaultDir()
	TheLight   Light
)

const (
	cacheSizeForTesting = C.uint64_t(1024)
	dagSizeForTesting   = C.uint64_t(1024 * 32)
)

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

type cache struct {
	light *C.struct_ethash_light
}

func freeCache(h *cache) {
	C.ethash_light_delete(h.light)
}

func makeCache(blockNum uint64, test bool) *cache {
	seedHash, _ := GetSeedHash(blockNum)
	size := C.ethash_get_cachesize(C.uint64_t(blockNum))
	if test {
		size = cacheSizeForTesting
	}
	light := C.ethash_light_new(size, (*C.ethash_h256_t)(unsafe.Pointer(&seedHash[0])))
	cache := &cache{light}
	runtime.SetFinalizer(cache, freeCache)
	return cache
}

// Light wraps an ethash cache for light client verification
// of block nonces.
type Light struct {
	test bool // if set use a smaller cache size

	// This is a one-element cache of caches.
	// TODO: keep multiple caches for recent epochs.
	mu    sync.RWMutex
	epoch uint64
	cache *cache
}

func (l *Light) getCache(blockNum uint64) *cache {
	l.mu.RLock()
	epoch := blockNum / epochLength
	if l.epoch == epoch && l.cache != nil {
		l.mu.RUnlock()
		return l.cache
	}
	l.mu.RUnlock()
	// Create the actual cache for the epoch.
	// No lock is being held, so multiple goroutines
	// might perform the generation and fight for the lock below.
	cache := makeCache(blockNum, l.test)
	l.mu.Lock()
	if l.epoch != epoch || l.cache == nil {
		l.cache = cache
	}
	l.mu.Unlock()
	return cache
}

func (l *Light) Verify(block pow.Block) bool {
	return l.verify(block.HashNoNonce(), block.MixDigest(), block.Difficulty(), block.NumberU64(), block.Nonce())
}

func (l *Light) verify(hash common.Hash, mixDigest common.Hash, difficulty *big.Int, blockNum uint64, nonce uint64) bool {
	// Make sure the block num is valid
	if blockNum >= epochLength*2048 {
		glog.V(logger.Info).Infoln(fmt.Sprintf("Block number exceeds limit, invalid (value is %v, limit is %v)",
			blockNum, epochLength*2048))
		return false
	}

	// First check: make sure header, mixDigest, nonce are correct without hitting the cache
	// This is to prevent DOS attacks
	chash := (*C.ethash_h256_t)(unsafe.Pointer(&hash[0]))
	cnonce := C.uint64_t(nonce)
	target := new(big.Int).Div(minDifficulty, difficulty)

	cache := l.getCache(blockNum)
	size := C.ethash_get_datasize(C.uint64_t(blockNum))
	if l.test {
		size = dagSizeForTesting
	}
	var ret C.ethash_return_value_t
	C.ethash_light_compute(&ret, cache.light, size, chash, cnonce)
	result := common.Bytes2Big(C.GoBytes(unsafe.Pointer(&ret.result), C.int(32)))
	return result.Cmp(target) <= 0
}

type dag struct {
	mu   sync.Mutex // prevents double free
	full *C.struct_ethash_full
}

func freeDAG(h *dag) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.full != nil {
		C.ethash_full_delete(h.full)
		h.full = nil
	}
}

func makeDAG(blockNum uint64, test bool, dir string) *dag {
	if dir == "" {
		dir = DefaultDir
	}
	seedHash, _ := GetSeedHash(blockNum)
	cache := makeCache(blockNum, test)
	size := C.ethash_get_cachesize(C.uint64_t(blockNum))
	if test {
		size = dagSizeForTesting
	}
	full := C.ethash_full_new(
		C.CString(dir),
		(*C.ethash_h256_t)(unsafe.Pointer(&seedHash[0])),
		size,
		C.ethash_light_get_cache(cache.light),
		nil,
	)
	dag := &dag{full: full}
	runtime.SetFinalizer(dag, freeDAG)
	return dag
}

type Full struct {
	Dir string // use this to specify a non-default DAG directory

	test     bool // if set use a smaller DAG size
	turbo    bool
	hashRate int64

	mu    sync.Mutex // protects the fields below
	epoch uint64     // epoch number of current full dag
	dag   *dag       // current full DAG
}

func (pow *Full) getDAG(blockNum uint64) *dag {
	pow.mu.Lock()
	defer pow.mu.Unlock()
	epoch := blockNum / epochLength
	if pow.epoch == epoch && pow.dag != nil {
		return pow.dag // up to date
	}
	// Generate a new DAG.
	// This computation is very very expensive.
	// The lock should prevent more than one of them
	// to run at the same time.
	pow.dag = makeDAG(blockNum, pow.test, pow.Dir)
	return pow.dag
}

func (pow *Full) Search(block pow.Block, stop <-chan struct{}) (nonce uint64, mixDigest, seedHash []byte) {
	dag := pow.getDAG(block.NumberU64())

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	miningHash := block.HashNoNonce()
	diff := block.Difficulty()

	i := int64(0)
	starti := i
	start := time.Now().UnixNano()

	nonce = uint64(r.Int63())
	cMiningHash := (*C.ethash_h256_t)(unsafe.Pointer(&miningHash[0]))
	target := new(big.Int).Div(minDifficulty, diff)

	var ret C.ethash_return_value_t
	for {
		select {
		case <-stop:
			pow.hashRate = 0
			return 0, nil, nil
		default:
			i++

			elapsed := time.Now().UnixNano() - start
			hashes := ((float64(1e9) / float64(elapsed)) * float64(i-starti)) / 1000
			pow.hashRate = int64(hashes)

			C.ethash_full_compute(&ret, dag.full, cMiningHash, C.uint64_t(nonce))
			result := common.Bytes2Big(C.GoBytes(unsafe.Pointer(&ret.result), C.int(32)))

			// TODO: disagrees with the spec https://github.com/ethereum/wiki/wiki/Ethash#mining
			if result.Cmp(target) <= 0 {
				mixDigest = C.GoBytes(unsafe.Pointer(&ret.mix_hash), C.int(32))
				seedHash, _ = GetSeedHash(block.NumberU64()) // This seedhash is useless
				return nonce, mixDigest, seedHash
			}
			nonce += 1
		}

		if !pow.turbo {
			time.Sleep(20 * time.Microsecond)
		}
	}
}

func (pow *Full) GetHashrate() int64 {
	return pow.hashRate
}

func (pow *Full) Turbo(on bool) {
	pow.turbo = on
}

type Ethash struct {
	Light
	Full
}

func New() *Ethash {
	return &Ethash{Light: TheLight}
}

func NewForTesting() *Ethash {
	return &Ethash{
		Light{test: true},
		Full{test: true},
	}
}

func GetSeedHash(blockNum uint64) ([]byte, error) {
	if blockNum >= epochLength*2048 {
		return nil, fmt.Errorf("block number is out of bounds (value %v, limit is %v)", blockNum, epochLength*2048)
	}

	epoch := blockNum / epochLength
	seedHash := make([]byte, 32)
	var i uint64
	for i = 0; i < epoch; i++ {
		seedHash = crypto.Sha3(seedHash)
	}
	return seedHash, nil
}

func (pow *Ethash) Stop() {
	pow.Light.mu.Lock()
	pow.Full.mu.Lock()
	defer pow.Full.mu.Unlock()
	defer pow.Light.mu.Unlock()
	pow.Full.dag = nil
	pow.Light.cache = nil
}
