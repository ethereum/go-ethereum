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
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/hashicorp/golang-lru"
)

// HeaderChain implements the basic block header chain logic that is shared by
// core.BlockChain and light.LightChain. It is not usable in itself, only as
// a part of either structure.
// It is not thread safe either, the encapsulating chain structures should do
// the necessary mutex locking/unlocking.
type HeaderChain struct {
	chainDb       ethdb.Database
	genesisHeader *types.Header

	currentHeader *types.Header // Current head of the header chain (may be above the block chain!)
	headerCache   *lru.Cache    // Cache for the most recent block headers
	tdCache       *lru.Cache    // Cache for the most recent block total difficulties

	procInterrupt  func() bool

	rand         *mrand.Rand
	getValidator getHeaderValidatorFn
}

// getHeaderValidatorFn returns a HeaderValidator interface
type getHeaderValidatorFn func() HeaderValidator

// NewHeaderChain creates a new HeaderChain structure.
//  getValidator should return the parent's validator
//  procInterrupt points to the parent's interrupt semaphore
//  wg points to the parent's shutdown wait group
func NewHeaderChain(chainDb ethdb.Database, getValidator getHeaderValidatorFn, procInterrupt func() bool) (*HeaderChain, error) {
	headerCache, _ := lru.New(headerCacheLimit)
	tdCache, _ := lru.New(tdCacheLimit)

	// Seed a fast but crypto originating random generator
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}

	hc := &HeaderChain{
		chainDb:       chainDb,
		headerCache:   headerCache,
		tdCache:       tdCache,
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
		glog.V(logger.Info).Infoln("WARNING: Wrote default ethereum genesis block")
		hc.genesisHeader = genesisBlock.Header()
	}

	hc.currentHeader = hc.genesisHeader
	if head := GetHeadBlockHash(chainDb); head != (common.Hash{}) {
		if chead := hc.GetHeader(head); chead != nil {
			hc.currentHeader = chead
		}
	}

	return hc, nil
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
	// Calculate the total difficulty of the header
	ptd := hc.GetTd(header.ParentHash)
	if ptd == nil {
		return NonStatTy, ParentError(header.ParentHash)
	}
	localTd := hc.GetTd(hc.currentHeader.Hash())
	externTd := new(big.Int).Add(header.Difficulty, ptd)

	// If the total difficulty is higher than our known, add it to the canonical chain
	// Second clause in the if statement reduces the vulnerability to selfish mining.
	// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
	if externTd.Cmp(localTd) > 0 || (externTd.Cmp(localTd) == 0 && mrand.Float64() < 0.5) {
		// Delete any canonical number assignments above the new head
		for i := header.Number.Uint64() + 1; GetCanonicalHash(hc.chainDb, i) != (common.Hash{}); i++ {
			DeleteCanonicalHash(hc.chainDb, i)
		}
		// Overwrite any stale canonical number assignments
		head := hc.GetHeader(header.ParentHash)
		for GetCanonicalHash(hc.chainDb, head.Number.Uint64()) != head.Hash() {
			WriteCanonicalHash(hc.chainDb, head.Hash(), head.Number.Uint64())
			head = hc.GetHeader(head.ParentHash)
		}
		// Extend the canonical chain with the new header
		if err := WriteCanonicalHash(hc.chainDb, header.Hash(), header.Number.Uint64()); err != nil {
			glog.Fatalf("failed to insert header number: %v", err)
		}
		if err := WriteHeadHeaderHash(hc.chainDb, header.Hash()); err != nil {
			glog.Fatalf("failed to insert head header hash: %v", err)
		}
		hc.currentHeader = types.CopyHeader(header)
		status = CanonStatTy
	} else {
		status = SideStatTy
	}
	// Irrelevant of the canonical status, write the header itself to the database
	if err := WriteTd(hc.chainDb, header.Hash(), externTd); err != nil {
		glog.Fatalf("failed to write header total difficulty: %v", err)
	}
	if err := WriteHeader(hc.chainDb, header); err != nil {
		glog.Fatalf("failed to write header contents: %v", err)
	}
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
				err = hc.getValidator().ValidateHeader(header, hc.GetHeader(header.ParentHash), checkPow)
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
			glog.V(logger.Debug).Infoln("premature abort during header chain processing")
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
	glog.V(logger.Info).Infof("imported %d header(s) (%d ignored) in %v. #%v [%x… / %x…]", stats.processed, stats.ignored,
		time.Since(start), last.Number, first.Hash().Bytes()[:4], last.Hash().Bytes()[:4])

	return 0, nil
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (hc *HeaderChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	// Get the origin header from which to fetch
	header := hc.GetHeader(hash)
	if header == nil {
		return nil
	}
	// Iterate the headers until enough is collected or the genesis reached
	chain := make([]common.Hash, 0, max)
	for i := uint64(0); i < max; i++ {
		if header = hc.GetHeader(header.ParentHash); header == nil {
			break
		}
		chain = append(chain, header.Hash())
		if header.Number.Cmp(common.Big0) == 0 {
			break
		}
	}
	return chain
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (hc *HeaderChain) GetTd(hash common.Hash) *big.Int {
	// Short circuit if the td's already in the cache, retrieve otherwise
	if cached, ok := hc.tdCache.Get(hash); ok {
		return cached.(*big.Int)
	}
	td := GetTd(hc.chainDb, hash)
	if td == nil {
		return nil
	}
	// Cache the found body for next time and return
	hc.tdCache.Add(hash, td)
	return td
}

// GetHeader retrieves a block header from the database by hash, caching it if
// found.
func (hc *HeaderChain) GetHeader(hash common.Hash) *types.Header {
	// Short circuit if the header's already in the cache, retrieve otherwise
	if header, ok := hc.headerCache.Get(hash); ok {
		return header.(*types.Header)
	}
	header := GetHeader(hc.chainDb, hash)
	if header == nil {
		return nil
	}
	// Cache the found header for next time and return
	hc.headerCache.Add(header.Hash(), header)
	return header
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (hc *HeaderChain) HasHeader(hash common.Hash) bool {
	return hc.GetHeader(hash) != nil
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (hc *HeaderChain) GetHeaderByNumber(number uint64) *types.Header {
	hash := GetCanonicalHash(hc.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return hc.GetHeader(hash)
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (hc *HeaderChain) CurrentHeader() *types.Header {
	return hc.currentHeader
}

// SetCurrentHeader sets the current head header of the canonical chain.
func (hc *HeaderChain) SetCurrentHeader(head *types.Header) {
	if err := WriteHeadHeaderHash(hc.chainDb, head.Hash()); err != nil {
		glog.Fatalf("failed to insert head header hash: %v", err)
	}
	hc.currentHeader = head
}

// DeleteCallback is a callback function that is called by SetHead before
// each header is deleted.
type DeleteCallback func(common.Hash)

// SetHead rewinds the local chain to a new head. Everything above the new head
// will be deleted and the new one set.
func (hc *HeaderChain) SetHead(head uint64, delFn DeleteCallback) {
	height := uint64(0)
	if hc.currentHeader != nil {
		height = hc.currentHeader.Number.Uint64()
	}

	for hc.currentHeader != nil && hc.currentHeader.Number.Uint64() > head {
		hash := hc.currentHeader.Hash()
		if delFn != nil {
			delFn(hash)
		}
		DeleteHeader(hc.chainDb, hash)
		DeleteTd(hc.chainDb, hash)
		hc.currentHeader = hc.GetHeader(hc.currentHeader.ParentHash)
	}
	// Roll back the canonical chain numbering
	for i := height; i > head; i-- {
		DeleteCanonicalHash(hc.chainDb, i)
	}
	// Clear out any stale content from the caches
	hc.headerCache.Purge()
	hc.tdCache.Purge()

	if hc.currentHeader == nil {
		hc.currentHeader = hc.genesisHeader
	}
	if err := WriteHeadHeaderHash(hc.chainDb, hc.currentHeader.Hash()); err != nil {
		glog.Fatalf("failed to reset head header hash: %v", err)
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
	hc  *HeaderChain // Canonical header chain
	Pow pow.PoW      // Proof of work used for validating
}

// NewBlockValidator returns a new block validator which is safe for re-use
func NewHeaderValidator(chain *HeaderChain, pow pow.PoW) HeaderValidator {
	return &headerValidator{
		Pow: pow,
		hc:  chain,
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
	return ValidateHeader(v.Pow, header, parent, checkPow, false)
}
