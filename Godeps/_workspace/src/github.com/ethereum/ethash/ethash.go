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
#include "src/libethash/util.c"
#include "src/libethash/internal.c"
#include "src/libethash/sha3.c"
*/
import "C"

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io/ioutil"
	"math/big"
	"math/rand"
	"os"
	"path"
	"sync"
	"time"
	"unsafe"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/pow"
)

var minDifficulty = new(big.Int).Exp(big.NewInt(2), big.NewInt(256), big.NewInt(0))

var powlogger = logger.NewLogger("POW")

type ParamsAndCache struct {
	params *C.ethash_params
	cache  *C.ethash_cache
	Epoch  uint64
}

type DAG struct {
	dag            unsafe.Pointer // full GB of memory for dag
	file           bool
	paramsAndCache *ParamsAndCache
}

type Ethash struct {
	turbo          bool
	HashRate       int64
	chainManager   pow.ChainManager
	dag            *DAG
	paramsAndCache *ParamsAndCache
	ret            *C.ethash_return_value
	dagMutex       *sync.RWMutex
	cacheMutex     *sync.RWMutex
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

func makeParamsAndCache(chainManager pow.ChainManager, blockNum uint64) (*ParamsAndCache, error) {
	if blockNum >= epochLength*2048 {
		return nil, fmt.Errorf("block number is out of bounds (value %v, limit is %v)", blockNum, epochLength*2048)
	}
	paramsAndCache := &ParamsAndCache{
		params: new(C.ethash_params),
		cache:  new(C.ethash_cache),
		Epoch:  blockNum / epochLength,
	}
	C.ethash_params_init(paramsAndCache.params, C.uint32_t(uint32(blockNum)))
	paramsAndCache.cache.mem = C.malloc(C.size_t(paramsAndCache.params.cache_size))

	seedHash, err := GetSeedHash(blockNum)
	if err != nil {
		return nil, err
	}

	powlogger.Infoln("Making Cache")
	start := time.Now()
	C.ethash_mkcache(paramsAndCache.cache, paramsAndCache.params, (*C.uint8_t)(unsafe.Pointer(&seedHash[0])))
	powlogger.Infoln("Took:", time.Since(start))

	return paramsAndCache, nil
}

func (pow *Ethash) UpdateCache(force bool) error {
	pow.cacheMutex.Lock()
	defer pow.cacheMutex.Unlock()

	thisEpoch := pow.chainManager.CurrentBlock().NumberU64() / epochLength
	if force || pow.paramsAndCache.Epoch != thisEpoch {
		var err error
		pow.paramsAndCache, err = makeParamsAndCache(pow.chainManager, pow.chainManager.CurrentBlock().NumberU64())
		if err != nil {
			panic(err)
		}
	}

	return nil
}

func makeDAG(p *ParamsAndCache) *DAG {
	d := &DAG{
		dag:            C.malloc(C.size_t(p.params.full_size)),
		file:           false,
		paramsAndCache: p,
	}

	donech := make(chan string)
	go func() {
		t := time.NewTicker(5 * time.Second)
		tstart := time.Now()
	done:
		for {
			select {
			case <-t.C:
				powlogger.Infof("... still generating DAG (%v) ...\n", time.Since(tstart).Seconds())
			case str := <-donech:
				powlogger.Infof("... %s ...\n", str)
				break done
			}
		}
	}()
	C.ethash_compute_full_data(d.dag, p.params, p.cache)
	donech <- "DAG generation completed"
	return d
}

func (pow *Ethash) writeDagToDisk(dag *DAG, epoch uint64) *os.File {
	if epoch > 2048 {
		panic(fmt.Errorf("Epoch must be less than 2048 (is %v)", epoch))
	}
	data := C.GoBytes(unsafe.Pointer(dag.dag), C.int(dag.paramsAndCache.params.full_size))
	file, err := os.Create("/tmp/dag")
	if err != nil {
		panic(err)
	}

	dataEpoch := make([]byte, 8)
	binary.BigEndian.PutUint64(dataEpoch, epoch)

	file.Write(dataEpoch)
	file.Write(data)

	return file
}

func (pow *Ethash) UpdateDAG() {
	blockNum := pow.chainManager.CurrentBlock().NumberU64()
	if blockNum >= epochLength*2048 {
		// This will crash in the 2030s or 2040s
		panic(fmt.Errorf("Current block number is out of bounds (value %v, limit is %v)", blockNum, epochLength*2048))
	}

	pow.dagMutex.Lock()
	defer pow.dagMutex.Unlock()
	thisEpoch := blockNum / epochLength
	if pow.dag == nil || pow.dag.paramsAndCache.Epoch != thisEpoch {
		if pow.dag != nil && pow.dag.dag != nil {
			C.free(pow.dag.dag)
			pow.dag.dag = nil
		}

		if pow.dag != nil && pow.dag.paramsAndCache.cache.mem != nil {
			C.free(pow.dag.paramsAndCache.cache.mem)
			pow.dag.paramsAndCache.cache.mem = nil
		}

		// Make the params and cache for the DAG
		paramsAndCache, err := makeParamsAndCache(pow.chainManager, blockNum)
		if err != nil {
			panic(err)
		}

		// TODO: On non-SSD disks, loading the DAG from disk takes longer than generating it in memory
		pow.paramsAndCache = paramsAndCache
		path := path.Join("/", "tmp", "dag")
		pow.dag = nil
		powlogger.Infoln("Retrieving DAG")
		start := time.Now()

		file, err := os.Open(path)
		if err != nil {
			powlogger.Infof("No DAG found. Generating new DAG in '%s' (this takes a while)...\n", path)
			pow.dag = makeDAG(paramsAndCache)
			file = pow.writeDagToDisk(pow.dag, thisEpoch)
			pow.dag.file = true
		} else {
			data, err := ioutil.ReadAll(file)
			if err != nil {
				powlogger.Infof("DAG load err: %v\n", err)
			}

			if len(data) < 8 {
				powlogger.Infof("DAG in '%s' is less than 8 bytes, it must be corrupted. Generating new DAG (this takes a while)...\n", path)
				pow.dag = makeDAG(paramsAndCache)
				file = pow.writeDagToDisk(pow.dag, thisEpoch)
				pow.dag.file = true
			} else {
				dataEpoch := binary.BigEndian.Uint64(data[0:8])
				if dataEpoch < thisEpoch {
					powlogger.Infof("DAG in '%s' is stale. Generating new DAG (this takes a while)...\n", path)
					pow.dag = makeDAG(paramsAndCache)
					file = pow.writeDagToDisk(pow.dag, thisEpoch)
					pow.dag.file = true
				} else if dataEpoch > thisEpoch {
					// FIXME
					panic(fmt.Errorf("Saved DAG in '%s' reports to be from future epoch %v (current epoch is %v)\n", path, dataEpoch, thisEpoch))
				} else if len(data) != (int(paramsAndCache.params.full_size) + 8) {
					powlogger.Infof("DAG in '%s' is corrupted. Generating new DAG (this takes a while)...\n", path)
					pow.dag = makeDAG(paramsAndCache)
					file = pow.writeDagToDisk(pow.dag, thisEpoch)
					pow.dag.file = true
				} else {
					data = data[8:]
					pow.dag = &DAG{
						dag:            unsafe.Pointer(&data[0]),
						file:           true,
						paramsAndCache: paramsAndCache,
					}
				}
			}
		}
		powlogger.Infoln("Took:", time.Since(start))

		file.Close()
	}
}

func New(chainManager pow.ChainManager) *Ethash {
	paramsAndCache, err := makeParamsAndCache(chainManager, chainManager.CurrentBlock().NumberU64())
	if err != nil {
		panic(err)
	}

	return &Ethash{
		turbo:          true,
		paramsAndCache: paramsAndCache,
		chainManager:   chainManager,
		dag:            nil,
		cacheMutex:     new(sync.RWMutex),
		dagMutex:       new(sync.RWMutex),
	}
}

func (pow *Ethash) DAGSize() uint64 {
	return uint64(pow.dag.paramsAndCache.params.full_size)
}

func (pow *Ethash) CacheSize() uint64 {
	return uint64(pow.paramsAndCache.params.cache_size)
}

func GetSeedHash(blockNum uint64) ([]byte, error) {
	if blockNum >= epochLength*2048 {
		return nil, fmt.Errorf("block number is out of bounds (value %v, limit is %v)", blockNum, epochLength*2048)
	}

	epoch := blockNum / epochLength
	seedHash := make([]byte, 32)
	var i uint64
	for i = 0; i < 32; i++ {
		seedHash[i] = 0
	}
	for i = 0; i < epoch; i++ {
		seedHash = crypto.Sha3(seedHash)
	}
	return seedHash, nil
}

func (pow *Ethash) Stop() {
	pow.cacheMutex.Lock()
	pow.dagMutex.Lock()
	defer pow.dagMutex.Unlock()
	defer pow.cacheMutex.Unlock()

	if pow.paramsAndCache.cache != nil {
		C.free(pow.paramsAndCache.cache.mem)
	}
	if pow.dag.dag != nil && !pow.dag.file {
		C.free(pow.dag.dag)
	}
	if pow.dag != nil && pow.dag.paramsAndCache != nil && pow.dag.paramsAndCache.cache.mem != nil {
		C.free(pow.dag.paramsAndCache.cache.mem)
		pow.dag.paramsAndCache.cache.mem = nil
	}
	pow.dag.dag = nil
}

func (pow *Ethash) Search(block pow.Block, stop <-chan struct{}) (uint64, []byte, []byte) {
	pow.UpdateDAG()

	pow.dagMutex.RLock()
	defer pow.dagMutex.RUnlock()

	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	miningHash := block.HashNoNonce()
	diff := block.Difficulty()

	i := int64(0)
	starti := i
	start := time.Now().UnixNano()

	nonce := uint64(r.Int63())
	cMiningHash := (*C.uint8_t)(unsafe.Pointer(&miningHash[0]))
	target := new(big.Int).Div(minDifficulty, diff)

	var ret C.ethash_return_value
	for {
		select {
		case <-stop:
			powlogger.Infoln("Breaking from mining")
			pow.HashRate = 0
			return 0, nil, nil
		default:
			i++

			elapsed := time.Now().UnixNano() - start
			hashes := ((float64(1e9) / float64(elapsed)) * float64(i-starti)) / 1000
			pow.HashRate = int64(hashes)

			C.ethash_full(&ret, pow.dag.dag, pow.dag.paramsAndCache.params, cMiningHash, C.uint64_t(nonce))
			result := common.Bytes2Big(C.GoBytes(unsafe.Pointer(&ret.result[0]), C.int(32)))

			// TODO: disagrees with the spec https://github.com/ethereum/wiki/wiki/Ethash#mining
			if result.Cmp(target) <= 0 {
				mixDigest := C.GoBytes(unsafe.Pointer(&ret.mix_hash[0]), C.int(32))
				seedHash, err := GetSeedHash(block.NumberU64()) // This seedhash is useless
				if err != nil {
					panic(err)
				}
				return nonce, mixDigest, seedHash
			}

			nonce += 1
		}

		if !pow.turbo {
			time.Sleep(20 * time.Microsecond)
		}
	}

}

func (pow *Ethash) Verify(block pow.Block) bool {
	return pow.verify(block.HashNoNonce().Bytes(), block.MixDigest().Bytes(), block.Difficulty(), block.NumberU64(), block.Nonce())
}

func (pow *Ethash) verify(hash []byte, mixDigest []byte, difficulty *big.Int, blockNum uint64, nonce uint64) bool {
	// Make sure the block num is valid
	if blockNum >= epochLength*2048 {
		powlogger.Infoln(fmt.Sprintf("Block number exceeds limit, invalid (value is %v, limit is %v)",
			blockNum, epochLength*2048))
		return false
	}

	// First check: make sure header, mixDigest, nonce are correct without hitting the cache
	// This is to prevent DOS attacks
	chash := (*C.uint8_t)(unsafe.Pointer(&hash[0]))
	cnonce := C.uint64_t(nonce)
	target := new(big.Int).Div(minDifficulty, difficulty)

	var pAc *ParamsAndCache
	// If its an old block (doesn't use the current cache)
	// get the cache for it but don't update (so we don't need the mutex)
	// Otherwise, it's the current block or a future block.
	// If current, updateCache will do nothing.
	if blockNum/epochLength < pow.paramsAndCache.Epoch {
		var err error
		// If we can't make the params for some reason, this block is invalid
		pAc, err = makeParamsAndCache(pow.chainManager, blockNum)
		if err != nil {
			powlogger.Infoln(err)
			return false
		}
	} else {
		pow.UpdateCache(false)
		pow.cacheMutex.RLock()
		defer pow.cacheMutex.RUnlock()
		pAc = pow.paramsAndCache
	}

	ret := new(C.ethash_return_value)

	C.ethash_light(ret, pAc.cache, pAc.params, chash, cnonce)

	result := common.Bytes2Big(C.GoBytes(unsafe.Pointer(&ret.result[0]), C.int(32)))
	return result.Cmp(target) <= 0
}

func (pow *Ethash) GetHashrate() int64 {
	return pow.HashRate
}

func (pow *Ethash) Turbo(on bool) {
	pow.turbo = on
}

func (pow *Ethash) FullHash(nonce uint64, miningHash []byte) []byte {
	pow.UpdateDAG()
	pow.dagMutex.Lock()
	defer pow.dagMutex.Unlock()
	cMiningHash := (*C.uint8_t)(unsafe.Pointer(&miningHash[0]))
	cnonce := C.uint64_t(nonce)
	ret := new(C.ethash_return_value)
	// pow.hash is the output/return of ethash_full
	C.ethash_full(ret, pow.dag.dag, pow.paramsAndCache.params, cMiningHash, cnonce)
	ghash_full := C.GoBytes(unsafe.Pointer(&ret.result), 32)
	return ghash_full
}

func (pow *Ethash) LightHash(nonce uint64, miningHash []byte) []byte {
	cMiningHash := (*C.uint8_t)(unsafe.Pointer(&miningHash[0]))
	cnonce := C.uint64_t(nonce)
	ret := new(C.ethash_return_value)
	C.ethash_light(ret, pow.paramsAndCache.cache, pow.paramsAndCache.params, cMiningHash, cnonce)
	ghash_light := C.GoBytes(unsafe.Pointer(&ret.result), 32)
	return ghash_light
}
