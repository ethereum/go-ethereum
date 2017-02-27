// Copyright 2015 The go-ethereum Authors
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
	crand "crypto/rand"
	"fmt"
	"math"
	"math/big"
	mrand "math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/hashicorp/golang-lru"
)

const (
	headerCacheLimit = 512
	tdCacheLimit     = 1024
	numberCacheLimit = 2048
)

// HeaderChain implements the basic block header chain logic that is shared by
// core.BlockChain and light.LightChain. It is not usable in itself, only as
// a part of either structure.
// It is not thread safe either, the encapsulating chain structures should do
// the necessary mutex locking/unlocking.
type HeaderChain struct {
	config *params.ChainConfig

	chainDb       ethdb.Database
	genesisHeader *types.Header

	currentHeader     *types.Header // Current head of the header chain (may be above the block chain!)
	currentHeaderHash common.Hash   // Hash of the current head of the header chain (prevent recomputing all the time)

	headerCache *lru.Cache // Cache for the most recent block headers
	tdCache     *lru.Cache // Cache for the most recent block total difficulties
	numberCache *lru.Cache // Cache for the most recent block numbers

	procInterrupt func() bool

	rand         *mrand.Rand
	getValidator getHeaderValidatorFn
}

// getHeaderValidatorFn returns a HeaderValidator interface
type getHeaderValidatorFn func() HeaderValidator

// NewHeaderChain creates a new HeaderChain structure.
//  getValidator should return the parent's validator
//  procInterrupt points to the parent's interrupt semaphore
//  wg points to the parent's shutdown wait group
func NewHeaderChain(chainDb ethdb.Database, config *params.ChainConfig, getValidator getHeaderValidatorFn, procInterrupt func() bool) (*HeaderChain, error) {
	headerCache, _ := lru.New(headerCacheLimit)
	tdCache, _ := lru.New(tdCacheLimit)
	numberCache, _ := lru.New(numberCacheLimit)

	// Seed a fast but crypto originating random generator
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	hc := &HeaderChain{
		config:        config,
		chainDb:       chainDb,
		headerCache:   headerCache,
		tdCache:       tdCache,
		numberCache:   numberCache,
		procInterrupt: procInterrupt,
		rand:          mrand.New(mrand.NewSource(seed.Int64())),
		getValidator:  getValidator,
	}

	hc.genesisHeader = hc.GetHeaderByNumber(0)
	if hc.genesisHeader == nil {
		genesisBlock, err := WriteDefaultGenesisBlock(chainDb)
		if err != nil {
			return nil, err
		}
		log.Info(fmt.Sprint("WARNING: Wrote default ethereum genesis block"))
		hc.genesisHeader = genesisBlock.Header()
	}

	hc.currentHeader = hc.genesisHeader
	if head := GetHeadBlockHash(chainDb); head != (common.Hash{}) {
		if chead := hc.GetHeaderByHash(head); chead != nil {
			hc.currentHeader = chead
		}
	}
	hc.currentHeaderHash = hc.currentHeader.Hash()

	return hc, nil
}

// GetBlockNumber retrieves the block number belonging to the given hash
// from the cache or database
func (hc *HeaderChain) GetBlockNumber(hash common.Hash) uint64 {
	if cached, ok := hc.numberCache.Get(hash); ok {
		return cached.(uint64)
	}
	number := GetBlockNumber(hc.chainDb, hash)
	if number != missingNumber {
		hc.numberCache.Add(hash, number)
	}
	return number
}

// WriteHeader writes a header into the local chain, given that its parent is
// already known. If the total difficulty of the newly inserted header becomes
// greater than the current known TD, the canonical chain is re-routed.
//
// Note: This method is not concurrent-safe with inserting blocks simultaneously
// into the chain, as side effects caused by reorganisations cannot be emulated
// without the real blocks. Hence, writing headers directly should only be done
// in two scenarios: pure-header mode of operation (light clients), or properly
// separated header/block phases (non-archive clients).
func (hc *HeaderChain) WriteHeader(header *types.Header) (status WriteStatus, err error) {
	// Cache some values to prevent constant recalculation
	var (
		hash   = header.Hash()
		number = header.Number.Uint64()
	)
	// Calculate the total difficulty of the header
	ptd := hc.GetTd(header.ParentHash, number-1)
	if ptd == nil {
		return NonStatTy, ParentError(header.ParentHash)
	}
	localTd := hc.GetTd(hc.currentHeaderHash, hc.currentHeader.Number.Uint64())
	externTd := new(big.Int).Add(header.Difficulty, ptd)

	// Irrelevant of the canonical status, write the td and header to the database
	if err := hc.WriteTd(hash, number, externTd); err != nil {
		log.Crit(fmt.Sprintf("failed to write header total difficulty: %v", err))
	}
	if err := WriteHeader(hc.chainDb, header); err != nil {
		log.Crit(fmt.Sprintf("failed to write header contents: %v", err))
	}

	// If the total difficulty is higher than our known, add it to the canonical chain
	// Second clause in the if statement reduces the vulnerability to selfish mining.
	// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
	if externTd.Cmp(localTd) > 0 || (externTd.Cmp(localTd) == 0 && mrand.Float64() < 0.5) {
		// Delete any canonical number assignments above the new head
		for i := number + 1; ; i++ {
			hash := GetCanonicalHash(hc.chainDb, i)
			if hash == (common.Hash{}) {
				break
			}
			DeleteCanonicalHash(hc.chainDb, i)
		}
		// Overwrite any stale canonical number assignments
		var (
			headHash   = header.ParentHash
			headNumber = header.Number.Uint64() - 1
			headHeader = hc.GetHeader(headHash, headNumber)
		)
		for GetCanonicalHash(hc.chainDb, headNumber) != headHash {
			WriteCanonicalHash(hc.chainDb, headHash, headNumber)

			headHash = headHeader.ParentHash
			headNumber = headHeader.Number.Uint64() - 1
			headHeader = hc.GetHeader(headHash, headNumber)
		}

		// Extend the canonical chain with the new header
		if err := WriteCanonicalHash(hc.chainDb, hash, number); err != nil {
			log.Crit(fmt.Sprintf("failed to insert header number: %v", err))
		}
		if err := WriteHeadHeaderHash(hc.chainDb, hash); err != nil {
			log.Crit(fmt.Sprintf("failed to insert head header hash: %v", err))
		}

		hc.currentHeaderHash, hc.currentHeader = hash, types.CopyHeader(header)

		status = CanonStatTy
	} else {
		status = SideStatTy
	}

	hc.headerCache.Add(hash, header)
	hc.numberCache.Add(hash, number)

	return
}

// WhCallback is a callback function for inserting individual headers.
// A callback is used for two reasons: first, in a LightChain, status should be
// processed and light chain events sent, while in a BlockChain this is not
// necessary since chain events are sent after inserting blocks. Second, the
// header writes should be protected by the parent chain mutex individually.
type WhCallback func(*types.Header) error

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify parameter can be used to fine tune whether nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verfy nonces, as well as
// because nonces can be verified sparsely, not needing to check each.
func (hc *HeaderChain) InsertHeaderChain(chain []*types.Header, checkFreq int, writeHeader WhCallback) (int, error) {
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		if chain[i].Number.Uint64() != chain[i-1].Number.Uint64()+1 || chain[i].ParentHash != chain[i-1].Hash() {
			// Chain broke ancestry, log a messge (programming error) and skip insertion
			failure := fmt.Errorf("non contiguous insert: item %d is #%d [%x…], item %d is #%d [%x…] (parent [%x…])",
				i-1, chain[i-1].Number.Uint64(), chain[i-1].Hash().Bytes()[:4], i, chain[i].Number.Uint64(), chain[i].Hash().Bytes()[:4], chain[i].ParentHash.Bytes()[:4])

			log.Error(fmt.Sprint(failure.Error()))
			return 0, failure
		}
	}
	// Collect some import statistics to report on
	stats := struct{ processed, ignored int }{}
	start := time.Now()

	// Generate the list of headers that should be POW verified
	verify := make([]bool, len(chain))
	for i := 0; i < len(verify)/checkFreq; i++ {
		index := i*checkFreq + hc.rand.Intn(checkFreq)
		if index >= len(verify) {
			index = len(verify) - 1
		}
		verify[index] = true
	}
	verify[len(verify)-1] = true // Last should always be verified to avoid junk

	// Create the header verification task queue and worker functions
	tasks := make(chan int, len(chain))
	for i := 0; i < len(chain); i++ {
		tasks <- i
	}
	close(tasks)

	errs, failed := make([]error, len(tasks)), int32(0)
	process := func(worker int) {
		for index := range tasks {
			header, hash := chain[index], chain[index].Hash()

			// Short circuit insertion if shutting down or processing failed
			if hc.procInterrupt() {
				return
			}
			if atomic.LoadInt32(&failed) > 0 {
				return
			}
			// Short circuit if the header is bad or already known
			if BadHashes[hash] {
				errs[index] = BadHashError(hash)
				atomic.AddInt32(&failed, 1)
				return
			}
			if hc.HasHeader(hash) {
				continue
			}
			// Verify that the header honors the chain parameters
			checkPow := verify[index]

			var err error
			if index == 0 {
				err = hc.getValidator().ValidateHeader(header, hc.GetHeader(header.ParentHash, header.Number.Uint64()-1), checkPow)
			} else {
				err = hc.getValidator().ValidateHeader(header, chain[index-1], checkPow)
			}
			if err != nil {
				errs[index] = err
				atomic.AddInt32(&failed, 1)
				return
			}
		}
	}
	// Start as many worker threads as goroutines allowed
	pending := new(sync.WaitGroup)
	for i := 0; i < runtime.GOMAXPROCS(0); i++ {
		pending.Add(1)
		go func(id int) {
			defer pending.Done()
			process(id)
		}(i)
	}
	pending.Wait()

	// If anything failed, report
	if failed > 0 {
		for i, err := range errs {
			if err != nil {
				return i, err
			}
		}
	}
	// All headers passed verification, import them into the database
	for i, header := range chain {
		// Short circuit insertion if shutting down
		if hc.procInterrupt() {
			log.Debug(fmt.Sprint("premature abort during header chain processing"))
			break
		}
		hash := header.Hash()

		// If the header's already known, skip it, otherwise store
		if hc.HasHeader(hash) {
			stats.ignored++
			continue
		}
		if err := writeHeader(header); err != nil {
			return i, err
		}
		stats.processed++
	}
	// Report some public statistics so the user has a clue what's going on
	first, last := chain[0], chain[len(chain)-1]

	ignored := ""
	if stats.ignored > 0 {
		ignored = fmt.Sprintf(" (%d ignored)", stats.ignored)
	}
	log.Info(fmt.Sprintf("imported %4d headers%s in %9v. #%v [%x… / %x…]", stats.processed, ignored, common.PrettyDuration(time.Since(start)), last.Number, first.Hash().Bytes()[:4], last.Hash().Bytes()[:4]))

	return 0, nil
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (hc *HeaderChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	// Get the origin header from which to fetch
	header := hc.GetHeaderByHash(hash)
	if header == nil {
		return nil
	}
	// Iterate the headers until enough is collected or the genesis reached
	chain := make([]common.Hash, 0, max)
	for i := uint64(0); i < max; i++ {
		next := header.ParentHash
		if header = hc.GetHeader(next, header.Number.Uint64()-1); header == nil {
			break
		}
		chain = append(chain, next)
		if header.Number.Sign() == 0 {
			break
		}
	}
	return chain
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash and number, caching it if found.
func (hc *HeaderChain) GetTd(hash common.Hash, number uint64) *big.Int {
	// Short circuit if the td's already in the cache, retrieve otherwise
	if cached, ok := hc.tdCache.Get(hash); ok {
		return cached.(*big.Int)
	}
	td := GetTd(hc.chainDb, hash, number)
	if td == nil {
		return nil
	}
	// Cache the found body for next time and return
	hc.tdCache.Add(hash, td)
	return td
}

// GetTdByHash retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (hc *HeaderChain) GetTdByHash(hash common.Hash) *big.Int {
	return hc.GetTd(hash, hc.GetBlockNumber(hash))
}

// WriteTd stores a block's total difficulty into the database, also caching it
// along the way.
func (hc *HeaderChain) WriteTd(hash common.Hash, number uint64, td *big.Int) error {
	if err := WriteTd(hc.chainDb, hash, number, td); err != nil {
		return err
	}
	hc.tdCache.Add(hash, new(big.Int).Set(td))
	return nil
}

// GetHeader retrieves a block header from the database by hash and number,
// caching it if found.
func (hc *HeaderChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	// Short circuit if the header's already in the cache, retrieve otherwise
	if header, ok := hc.headerCache.Get(hash); ok {
		return header.(*types.Header)
	}
	header := GetHeader(hc.chainDb, hash, number)
	if header == nil {
		return nil
	}
	// Cache the found header for next time and return
	hc.headerCache.Add(hash, header)
	return header
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (hc *HeaderChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return hc.GetHeader(hash, hc.GetBlockNumber(hash))
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (hc *HeaderChain) HasHeader(hash common.Hash) bool {
	return hc.GetHeaderByHash(hash) != nil
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (hc *HeaderChain) GetHeaderByNumber(number uint64) *types.Header {
	hash := GetCanonicalHash(hc.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return hc.GetHeader(hash, number)
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (hc *HeaderChain) CurrentHeader() *types.Header {
	return hc.currentHeader
}

// SetCurrentHeader sets the current head header of the canonical chain.
func (hc *HeaderChain) SetCurrentHeader(head *types.Header) {
	if err := WriteHeadHeaderHash(hc.chainDb, head.Hash()); err != nil {
		log.Crit(fmt.Sprintf("failed to insert head header hash: %v", err))
	}
	hc.currentHeader = head
	hc.currentHeaderHash = head.Hash()
}

// DeleteCallback is a callback function that is called by SetHead before
// each header is deleted.
type DeleteCallback func(common.Hash, uint64)

// SetHead rewinds the local chain to a new head. Everything above the new head
// will be deleted and the new one set.
func (hc *HeaderChain) SetHead(head uint64, delFn DeleteCallback) {
	height := uint64(0)
	if hc.currentHeader != nil {
		height = hc.currentHeader.Number.Uint64()
	}

	for hc.currentHeader != nil && hc.currentHeader.Number.Uint64() > head {
		hash := hc.currentHeader.Hash()
		num := hc.currentHeader.Number.Uint64()
		if delFn != nil {
			delFn(hash, num)
		}
		DeleteHeader(hc.chainDb, hash, num)
		DeleteTd(hc.chainDb, hash, num)
		hc.currentHeader = hc.GetHeader(hc.currentHeader.ParentHash, hc.currentHeader.Number.Uint64()-1)
	}
	// Roll back the canonical chain numbering
	for i := height; i > head; i-- {
		DeleteCanonicalHash(hc.chainDb, i)
	}
	// Clear out any stale content from the caches
	hc.headerCache.Purge()
	hc.tdCache.Purge()
	hc.numberCache.Purge()

	if hc.currentHeader == nil {
		hc.currentHeader = hc.genesisHeader
	}
	hc.currentHeaderHash = hc.currentHeader.Hash()

	if err := WriteHeadHeaderHash(hc.chainDb, hc.currentHeaderHash); err != nil {
		log.Crit(fmt.Sprintf("failed to reset head header hash: %v", err))
	}
}

// SetGenesis sets a new genesis block header for the chain
func (hc *HeaderChain) SetGenesis(head *types.Header) {
	hc.genesisHeader = head
}

// headerValidator is responsible for validating block headers
//
// headerValidator implements HeaderValidator.
type headerValidator struct {
	config *params.ChainConfig
	hc     *HeaderChain // Canonical header chain
	Pow    pow.PoW      // Proof of work used for validating
}

// NewBlockValidator returns a new block validator which is safe for re-use
func NewHeaderValidator(config *params.ChainConfig, chain *HeaderChain, pow pow.PoW) HeaderValidator {
	return &headerValidator{
		config: config,
		Pow:    pow,
		hc:     chain,
	}
}

// ValidateHeader validates the given header and, depending on the pow arg,
// checks the proof of work of the given header. Returns an error if the
// validation failed.
func (v *headerValidator) ValidateHeader(header, parent *types.Header, checkPow bool) error {
	// Short circuit if the parent is missing.
	if parent == nil {
		return ParentError(header.ParentHash)
	}
	// Short circuit if the header's already known or its parent missing
	if v.hc.HasHeader(header.Hash()) {
		return nil
	}
	return ValidateHeader(v.config, v.Pow, header, parent, checkPow, false)
}
