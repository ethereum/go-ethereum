package ethash

/*
#cgo CFLAGS: -std=gnu99 -Wall
#include "libethash/ethash.h"
#include "libethash/util.c"
#include "libethash/internal.c"
#include "libethash/sha3.c"
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"log"
	"math/big"
	"math/rand"
	"sync"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow"
)

var powlogger = logger.NewLogger("POW")

type DAG struct {
	SeedBlockNum uint64
	dag          unsafe.Pointer // full GB of memory for dag
}

type ParamsAndCache struct {
	params       *C.ethash_params
	cache        *C.ethash_cache
	SeedBlockNum uint64
}

type Ethash struct {
	turbo          bool
	HashRate       int64
	chainManager   pow.ChainManager
	dag            *DAG
	paramsAndCache *ParamsAndCache
	nextdag        unsafe.Pointer
	ret            *C.ethash_return_value
	dagMutex       *sync.Mutex
	cacheMutex     *sync.Mutex
}

func parseNonce(nonce []byte) (uint64, error) {
	nonceBuf := bytes.NewBuffer(nonce)
	nonceInt, err := binary.ReadUvarint(nonceBuf)
	if err != nil {
		return 0, err
	}
	return nonceInt, nil
}

const epochLength uint64 = 30000

func getSeedBlockNum(blockNum uint64) uint64 {
	var seedBlockNum uint64 = 0
	if blockNum >= 2*epochLength {
		seedBlockNum = ((blockNum / epochLength) - 1) * epochLength
	}
	return seedBlockNum
}

func makeParamsAndCache(chainManager pow.ChainManager, blockNum uint64) *ParamsAndCache {
	seedBlockNum := getSeedBlockNum(blockNum)
	paramsAndCache := &ParamsAndCache{
		params:       new(C.ethash_params),
		cache:        new(C.ethash_cache),
		SeedBlockNum: seedBlockNum,
	}
	C.ethash_params_init(paramsAndCache.params, C.uint32_t(seedBlockNum))
	paramsAndCache.cache.mem = C.malloc(paramsAndCache.params.cache_size)
	seedHash := chainManager.GetBlockByNumber(seedBlockNum).Header().Hash()
	log.Println("Params", paramsAndCache.params)

	log.Println("Making Cache")
	start := time.Now()
	C.ethash_mkcache(paramsAndCache.cache, paramsAndCache.params, (*C.uint8_t)((unsafe.Pointer)(&seedHash[0])))
	log.Println("Took:", time.Since(start))
	return paramsAndCache
}

func (pow *Ethash) updateCache() {
	pow.cacheMutex.Lock()
	seedNum := getSeedBlockNum(pow.chainManager.CurrentBlock().NumberU64())
	if pow.paramsAndCache.SeedBlockNum != seedNum {
		pow.paramsAndCache = makeParamsAndCache(pow.chainManager, pow.chainManager.CurrentBlock().NumberU64())
	}
	pow.cacheMutex.Unlock()
}

func makeDAG(p *ParamsAndCache) *DAG {
	d := &DAG{
		dag:          C.malloc(p.params.full_size),
		SeedBlockNum: p.SeedBlockNum,
	}
	C.ethash_compute_full_data(d.dag, p.params, p.cache)
	return d
}

func (pow *Ethash) updateDAG() {
	pow.cacheMutex.Lock()
	pow.dagMutex.Lock()

	seedNum := getSeedBlockNum(pow.chainManager.CurrentBlock().NumberU64())
	if pow.dag == nil || pow.dag.SeedBlockNum != seedNum {
		pow.dag = nil
		log.Println("Making Dag")
		start := time.Now()
		pow.dag = makeDAG(pow.paramsAndCache)
		log.Println("Took:", time.Since(start))
	}

	pow.dagMutex.Unlock()
	pow.cacheMutex.Unlock()
}

func New(chainManager pow.ChainManager) *Ethash {
	return &Ethash{
		turbo:          false,
		paramsAndCache: makeParamsAndCache(chainManager, chainManager.CurrentBlock().NumberU64()),
		chainManager:   chainManager,
		dag:            nil,
		ret:            new(C.ethash_return_value),
		cacheMutex:     new(sync.Mutex),
		dagMutex:       new(sync.Mutex),
	}
}

func (pow *Ethash) DAGSize() uint64 {
	return uint64(pow.paramsAndCache.params.full_size)
}

func (pow *Ethash) CacheSize() uint64 {
	return uint64(pow.paramsAndCache.params.cache_size)
}

func (pow *Ethash) GetSeedHash(blockNum uint64) []byte {
	return pow.chainManager.GetBlockByNumber(getSeedBlockNum(blockNum)).Header().Hash()
}

func (pow *Ethash) Stop() {
	pow.cacheMutex.Lock()
	pow.dagMutex.Lock()
	if pow.paramsAndCache.cache != nil {
		C.free(pow.paramsAndCache.cache.mem)
	}
	if pow.dag != nil {
		C.free(pow.dag.dag)
	}
	pow.dagMutex.Unlock()
	pow.cacheMutex.Unlock()
}

func (pow *Ethash) Search(block pow.Block, stop <-chan struct{}) ([]byte, []byte, []byte) {
	pow.updateDAG()

	// Not very elegant, multiple mining instances are not supported
	pow.dagMutex.Lock()
	pow.cacheMutex.Lock()
	defer pow.cacheMutex.Unlock()
	defer pow.dagMutex.Unlock()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	miningHash := block.HashNoNonce()
	diff := block.Difficulty()
	log.Println("difficulty", diff)
	i := int64(0)
	start := time.Now().UnixNano()
	t := time.Now()

	nonce := uint64(r.Int63())

	for {
		select {
		case <-stop:
			powlogger.Infoln("Breaking from mining")
			pow.HashRate = 0
			pow.dagMutex.Unlock()
			return nil, nil, nil
		default:
			i++

			if time.Since(t) > (1 * time.Second) {
				elapsed := time.Now().UnixNano() - start
				hashes := ((float64(1e9) / float64(elapsed)) * float64(i)) / 1000
				pow.HashRate = int64(hashes)
				powlogger.Infoln("Hashing @", pow.HashRate, "khash")

				t = time.Now()
			}

			cMiningHash := (*C.uint8_t)(unsafe.Pointer(&miningHash))
			cnonce := C.uint64_t(nonce)
			log.Printf("seed hash, nonce: %x %x\n", miningHash, nonce)
			// pow.hash is the output/return of ethash_full
			C.ethash_full(pow.ret, pow.dag.dag, pow.paramsAndCache.params, cMiningHash, cnonce)
			res := C.ethash_check_difficulty((*C.uint8_t)(&pow.ret.result[0]), (*C.uint8_t)(unsafe.Pointer(&diff.Bytes()[0])))
			if res == 1 {
				mixDigest := C.GoBytes(unsafe.Pointer(&pow.ret.mix_hash[0]), 32)
				// We don't really nead 32 bytes here
				buf := make([]byte, 32)
				binary.PutUvarint(buf, nonce)
				return buf, mixDigest, pow.GetSeedHash(block.NumberU64())
			}
			nonce += 1
		}

		if !pow.turbo {
			time.Sleep(20 * time.Microsecond)
		}
	}
}

func (pow *Ethash) Verify(block pow.Block) bool {
	// Make sure the SeedHash is set correctly
	if bytes.Compare(block.SeedHash(), pow.GetSeedHash(block.NumberU64())) != 0 {
		log.Println("Block had wrong SeedHash")
		log.Println("Expected: ", pow.GetSeedHash(block.NumberU64()))
		log.Println("Actual: ", block.SeedHash())
		return false
	}

	nonceInt, err := parseNonce(block.Nonce())
	if err != nil {
		log.Println("nonce to int err:", err)
		return false
	}
	return pow.verify(block.HashNoNonce(), block.MixDigest(), block.Difficulty(), block.NumberU64(), nonceInt)
}

func (pow *Ethash) verify(hash []byte, mixDigest []byte, difficulty *big.Int, blockNum uint64, nonce uint64) bool {
	// First check: make sure header, mixDigest, nonce are correct without hitting the DAG
	// This is to prevent DOS attacks
	chash := (*C.uint8_t)(unsafe.Pointer(&hash))
	cnonce := C.uint64_t(nonce)
	cmixDigest := (*C.uint8_t)(unsafe.Pointer(&mixDigest))
	cdifficulty := (*C.uint8_t)(unsafe.Pointer(&difficulty.Bytes()[0]))
	if C.ethash_quick_check_difficulty(chash, cnonce, cmixDigest, cdifficulty) != 1 {
		log.Println("Failed to pass quick check.  Are you sure that the mix digest is correct?")
		return false
	}

	var pAc *ParamsAndCache
	// If its an old block (doesn't use the current cache)
	// get the cache for it but don't update (so we don't need the mutex)
	// Otherwise, it's the current block or a future.
	// If current, updateCache will do nothing.
	if getSeedBlockNum(blockNum) < pow.paramsAndCache.SeedBlockNum {
		pAc = makeParamsAndCache(pow.chainManager, blockNum)
	} else {
		pow.updateCache()
		pow.cacheMutex.Lock()
		defer pow.cacheMutex.Unlock()
		pAc = pow.paramsAndCache
	}

	C.ethash_light(pow.ret, pAc.cache, pAc.params, chash, cnonce)
	res := C.ethash_check_difficulty((*C.uint8_t)(unsafe.Pointer(&pow.ret.result[0])), cdifficulty)
	return res == 1
}

func (pow *Ethash) GetHashrate() int64 {
	return pow.HashRate
}

func (pow *Ethash) Turbo(on bool) {
	pow.turbo = on
}

func (pow *Ethash) FullHash(nonce uint64, miningHash []byte) []byte {
	pow.updateDAG()
	pow.dagMutex.Lock()
	defer pow.dagMutex.Unlock()
	cMiningHash := (*C.uint8_t)(unsafe.Pointer(&miningHash))
	cnonce := C.uint64_t(nonce)
	log.Println("seed hash, nonce:", miningHash, nonce)
	// pow.hash is the output/return of ethash_full
	C.ethash_full(pow.ret, pow.dag.dag, pow.paramsAndCache.params, cMiningHash, cnonce)
	ghash_full := C.GoBytes(unsafe.Pointer(&pow.ret.result[0]), 32)
	return ghash_full
}

func (pow *Ethash) LightHash(nonce uint64, miningHash []byte) []byte {
	cMiningHash := (*C.uint8_t)(unsafe.Pointer(&miningHash))
	cnonce := C.uint64_t(nonce)
	C.ethash_light(pow.ret, pow.paramsAndCache.cache, pow.paramsAndCache.params, cMiningHash, cnonce)
	ghash_light := C.GoBytes(unsafe.Pointer(&pow.ret.result[0]), 32)
	return ghash_light
}
