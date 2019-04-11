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
	"errors"
	"fmt"
	"math"
	"math/big"
	mrand "math/rand"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
	lru "github.com/hashicorp/golang-lru"
)

const (
	headerCacheLimit = 512
	tdCacheLimit     = 1024
	numberCacheLimit = 4096
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

	currentHeader     atomic.Value // Current head of the header chain (may be above the block chain!)
	currentHeaderHash common.Hash  // Hash of the current head of the header chain (prevent recomputing all the time)

	headerCache *lru.Cache // Cache for the most recent block headers
	tdCache     *lru.Cache // Cache for the most recent block total difficulties
	numberCache *lru.Cache // Cache for the most recent block numbers

	procInterrupt func() bool

	rand   *mrand.Rand
	engine consensus.Engine
}

// NewHeaderChain creates a new HeaderChain structure.
//  getValidator should return the parent's validator
//  procInterrupt points to the parent's interrupt semaphore
//  wg points to the parent's shutdown wait group
func NewHeaderChain(chainDb ethdb.Database, config *params.ChainConfig, engine consensus.Engine, procInterrupt func() bool) (*HeaderChain, error) {
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
		engine:        engine,
	}

	hc.genesisHeader = hc.GetHeaderByNumber(0)
	if hc.genesisHeader == nil {
		return nil, ErrNoGenesis
	}

	hc.currentHeader.Store(hc.genesisHeader)
	if head := rawdb.ReadHeadBlockHash(chainDb); head != (common.Hash{}) {
		if chead := hc.GetHeaderByHash(head); chead != nil {
			hc.currentHeader.Store(chead)
		}
	}
	hc.currentHeaderHash = hc.CurrentHeader().Hash()
	headHeaderGauge.Update(hc.CurrentHeader().Number.Int64())

	return hc, nil
}

// GetBlockNumber retrieves the block number belonging to the given hash
// from the cache or database
func (hc *HeaderChain) GetBlockNumber(hash common.Hash) *uint64 {
	if cached, ok := hc.numberCache.Get(hash); ok {
		number := cached.(uint64)
		return &number
	}
	number := rawdb.ReadHeaderNumber(hc.chainDb, hash)
	if number != nil {
		hc.numberCache.Add(hash, *number)
	}
	return number
}

// WriteHeaders writes a chain of headers into the local chain, given that the parents
// are already known. If the total difficulty of the newly inserted chain becomes
// greater than the current known TD, the canonical chain is reorged.
//
// Note: This method is not concurrent-safe with inserting blocks simultaneously
// into the chain, as side effects caused by reorganisations cannot be emulated
// without the real blocks. Hence, writing headers directly should only be done
// in two scenarios: pure-header mode of operation (light clients), or properly
// separated header/block phases (non-archive clients).
func (hc *HeaderChain) WriteHeaders(headers []*types.Header, pwCallbackFn PWCallback) (ignored, imported int, status WriteStatus, lastHash common.Hash, lastHeader *types.Header, err error) {
	if len(headers) == 0 {
		return ignored, imported, NonStatTy, lastHash, nil, err
	}
	ptd := hc.GetTd(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	if ptd == nil {
		return ignored, imported, NonStatTy, lastHash, nil, consensus.ErrUnknownAncestor
	}
	type numberHash struct {
		number uint64
		hash   common.Hash
	}
	var (
		lastNumber uint64       // Last successfully imported number
		externTd   *big.Int     // TD of successfully imported chain
		inserted   []numberHash // Quick lookup of number/hash for the chain
	)
	lastHash, lastNumber = headers[0].ParentHash, headers[0].Number.Uint64()-1 // Already validated above
	batch := hc.chainDb.NewBatch()
	for _, header := range headers {
		hash, number := header.Hash(), header.Number.Uint64()
		if header.ParentHash != lastHash {
			log.Warn("Non-contiguous header insertion", "header.parent", header.ParentHash, "expected", hash, "number", number)
			break
		}
		externTd = new(big.Int).Add(header.Difficulty, ptd)
		// If the header's already known, skip it, otherwise store
		if !hc.HasHeader(hash, number) {
			// Irrelevant of the canonical status, write the td and header to the database
			rawdb.WriteTd(batch, hash, number, externTd)
			hc.tdCache.Add(hash, new(big.Int).Set(externTd))

			rawdb.WriteHeader(batch, header)
			inserted = append(inserted, numberHash{number, hash})
			hc.headerCache.Add(hash, header)
			hc.numberCache.Add(hash, number)
		}
		lastHeader, lastHash, lastNumber, ptd = header, hash, number, externTd
	}
	batch.Write()
	batch.Reset()
	localTd := hc.GetTd(hc.currentHeaderHash, hc.CurrentHeader().Number.Uint64())
	// If the total difficulty is higher than our known, add it to the canonical chain
	// Second clause in the if statement reduces the vulnerability to selfish mining.
	// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
	if externTd.Cmp(localTd) > 0 || (externTd.Cmp(localTd) == 0 && mrand.Float64() < 0.5) {
		// Delete any canonical number assignments above the new head
		for i := lastNumber + 1; ; i++ {
			hash := rawdb.ReadCanonicalHash(hc.chainDb, i)
			if hash == (common.Hash{}) {
				break
			}
			rawdb.DeleteCanonicalHash(batch, i)
		}
		batch.Write()
		batch.Reset()

		// Overwrite any stale canonical number assignments, work backwards
		// from the first inserted header
		var (
			headHash   = headers[0].ParentHash
			headNumber = headers[0].Number.Uint64() - 1
		)
		for rawdb.ReadCanonicalHash(hc.chainDb, headNumber) != headHash {
			rawdb.WriteCanonicalHash(batch, headHash, headNumber)
			headHeader := hc.GetHeader(headHash, headNumber)
			headHash = headHeader.ParentHash
			headNumber = headHeader.Number.Uint64() - 1
		}
		// Now write the current chain in a batch
		for _, hdr := range inserted {
			rawdb.WriteCanonicalHash(batch, hdr.hash, hdr.number)
		}
		rawdb.WriteHeadHeaderHash(batch, lastHash)
		batch.Write()
		batch.Reset()
		hc.currentHeaderHash = lastHash
		hc.currentHeader.Store(types.CopyHeader(lastHeader))
		headHeaderGauge.Update(lastHeader.Number.Int64())
		status = CanonStatTy
	} else {
		status = SideStatTy
	}
	// Execute any post-write callback function
	if pwCallbackFn != nil {
		for _, header := range headers {
			if header.Number.Uint64() > lastNumber {
				break
			}
			pwCallbackFn(header, status)
		}
	}
	imported = len(inserted)
	ignored = len(headers) - imported
	return ignored, imported, status, lastHash, lastHeader, nil
}

// PWCallback (Post-Write Callback) is a callback function for inserting
// that is called after each header is inserted.
// A callback is used for two reasons: first, in a LightChain, status should be
// processed and light chain events sent, while in a BlockChain this is not
// necessary since chain events are sent after inserting blocks.
type PWCallback func(header *types.Header, status WriteStatus)

func (hc *HeaderChain) ValidateHeaderChain(chain []*types.Header, checkFreq int) (int, error) {
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		if chain[i].Number.Uint64() != chain[i-1].Number.Uint64()+1 || chain[i].ParentHash != chain[i-1].Hash() {
			// Chain broke ancestry, log a message (programming error) and skip insertion
			log.Error("Non contiguous header insert", "number", chain[i].Number, "hash", chain[i].Hash(),
				"parent", chain[i].ParentHash, "prevnumber", chain[i-1].Number, "prevhash", chain[i-1].Hash())

			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x…], item %d is #%d [%x…] (parent [%x…])", i-1, chain[i-1].Number,
				chain[i-1].Hash().Bytes()[:4], i, chain[i].Number, chain[i].Hash().Bytes()[:4], chain[i].ParentHash[:4])
		}
	}

	// Generate the list of seal verification requests, and start the parallel verifier
	seals := make([]bool, len(chain))
	if checkFreq != 0 {
		// In case of checkFreq == 0 all seals are left false.
		for i := 0; i < len(seals)/checkFreq; i++ {
			index := i*checkFreq + hc.rand.Intn(checkFreq)
			if index >= len(seals) {
				index = len(seals) - 1
			}
			seals[index] = true
		}
		// Last should always be verified to avoid junk.
		seals[len(seals)-1] = true
	}

	abort, results := hc.engine.VerifyHeaders(hc, chain, seals)
	defer close(abort)

	// Iterate over the headers and ensure they all check out
	for i, header := range chain {
		// If the chain is terminating, stop processing blocks
		if hc.procInterrupt() {
			log.Debug("Premature abort during headers verification")
			return 0, errors.New("aborted")
		}
		// If the header is a banned one, straight out abort
		if BadHashes[header.Hash()] {
			return i, ErrBlacklistedHash
		}
		// Otherwise wait for headers checks and ensure they pass
		if err := <-results; err != nil {
			return i, err
		}
	}

	return 0, nil
}

// InsertHeaderChain inserts the given headers, and returns the
// index at which something went wrong, the status for the last imported block, and
// the error (if any)
// The caller should hold applicable mutexes
func (hc *HeaderChain) InsertHeaderChain(chain []*types.Header, start time.Time, pwCallBackFn PWCallback) (int, WriteStatus, error) {
	// Collect some import statistics to report on
	ignored, imported, status, lastHash, last, err := hc.WriteHeaders(chain, pwCallBackFn)
	context := []interface{}{
		"count", imported,
		"elapsed", common.PrettyDuration(time.Since(start)),
	}
	if err != nil {
		context = append(context, "err", err)
	}
	if last != nil {
		context = append(context, "number", last.Number, "hash", lastHash)
		if timestamp := time.Unix(int64(last.Time), 0); time.Since(timestamp) > time.Minute {
			context = append(context, []interface{}{"age", common.PrettyAge(timestamp)}...)
		}
	}
	if ignored > 0 {
		context = append(context, []interface{}{"ignored", ignored}...)
	}
	log.Info("Imported new block headers", context...)
	return 0, status, nil
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

// GetAncestor retrieves the Nth ancestor of a given block. It assumes that either the given block or
// a close ancestor of it is canonical. maxNonCanonical points to a downwards counter limiting the
// number of blocks to be individually checked before we reach the canonical chain.
//
// Note: ancestor == 0 returns the same block, 1 returns its parent and so on.
func (hc *HeaderChain) GetAncestor(hash common.Hash, number, ancestor uint64, maxNonCanonical *uint64) (common.Hash, uint64) {
	if ancestor > number {
		return common.Hash{}, 0
	}
	if ancestor == 1 {
		// in this case it is cheaper to just read the header
		if header := hc.GetHeader(hash, number); header != nil {
			return header.ParentHash, number - 1
		} else {
			return common.Hash{}, 0
		}
	}
	for ancestor != 0 {
		if rawdb.ReadCanonicalHash(hc.chainDb, number) == hash {
			ancestorHash := rawdb.ReadCanonicalHash(hc.chainDb, number-ancestor)
			if rawdb.ReadCanonicalHash(hc.chainDb, number) == hash {
				number -= ancestor
				return ancestorHash, number
			}
		}
		if *maxNonCanonical == 0 {
			return common.Hash{}, 0
		}
		*maxNonCanonical--
		ancestor--
		header := hc.GetHeader(hash, number)
		if header == nil {
			return common.Hash{}, 0
		}
		hash = header.ParentHash
		number--
	}
	return hash, number
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash and number, caching it if found.
func (hc *HeaderChain) GetTd(hash common.Hash, number uint64) *big.Int {
	// Short circuit if the td's already in the cache, retrieve otherwise
	if cached, ok := hc.tdCache.Get(hash); ok {
		return cached.(*big.Int)
	}
	td := rawdb.ReadTd(hc.chainDb, hash, number)
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
	number := hc.GetBlockNumber(hash)
	if number == nil {
		return nil
	}
	return hc.GetTd(hash, *number)
}

// WriteTd stores a block's total difficulty into the database, also caching it
// along the way.
func (hc *HeaderChain) WriteTd(hash common.Hash, number uint64, td *big.Int) error {
	rawdb.WriteTd(hc.chainDb, hash, number, td)
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
	header := rawdb.ReadHeader(hc.chainDb, hash, number)
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
	number := hc.GetBlockNumber(hash)
	if number == nil {
		return nil
	}
	return hc.GetHeader(hash, *number)
}

// HasHeader checks if a block header is present in the database or not.
func (hc *HeaderChain) HasHeader(hash common.Hash, number uint64) bool {
	if hc.numberCache.Contains(hash) || hc.headerCache.Contains(hash) {
		return true
	}
	return rawdb.HasHeader(hc.chainDb, hash, number)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (hc *HeaderChain) GetHeaderByNumber(number uint64) *types.Header {
	hash := rawdb.ReadCanonicalHash(hc.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return hc.GetHeader(hash, number)
}

func (hc *HeaderChain) GetCanonicalHash(number uint64) common.Hash {
	return rawdb.ReadCanonicalHash(hc.chainDb, number)
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (hc *HeaderChain) CurrentHeader() *types.Header {
	return hc.currentHeader.Load().(*types.Header)
}

// SetCurrentHeader sets the current head header of the canonical chain.
func (hc *HeaderChain) SetCurrentHeader(head *types.Header) {
	rawdb.WriteHeadHeaderHash(hc.chainDb, head.Hash())

	hc.currentHeader.Store(head)
	hc.currentHeaderHash = head.Hash()
	headHeaderGauge.Update(head.Number.Int64())
}

type (
	// UpdateHeadBlocksCallback is a callback function that is called by SetHead
	// before head header is updated.
	UpdateHeadBlocksCallback func(ethdb.KeyValueWriter, *types.Header)

	// DeleteBlockContentCallback is a callback function that is called by SetHead
	// before each header is deleted.
	DeleteBlockContentCallback func(ethdb.KeyValueWriter, common.Hash, uint64)
)

// SetHead rewinds the local chain to a new head. Everything above the new head
// will be deleted and the new one set.
func (hc *HeaderChain) SetHead(head uint64, updateFn UpdateHeadBlocksCallback, delFn DeleteBlockContentCallback) {
	var (
		parentHash common.Hash
		batch      = hc.chainDb.NewBatch()
	)
	for hdr := hc.CurrentHeader(); hdr != nil && hdr.Number.Uint64() > head; hdr = hc.CurrentHeader() {
		hash, num := hdr.Hash(), hdr.Number.Uint64()

		// Rewind block chain to new head.
		parent := hc.GetHeader(hdr.ParentHash, num-1)
		if parent == nil {
			parent = hc.genesisHeader
		}
		parentHash = hdr.ParentHash
		// Notably, since geth has the possibility for setting the head to a low
		// height which is even lower than ancient head.
		// In order to ensure that the head is always no higher than the data in
		// the database(ancient store or active store), we need to update head
		// first then remove the relative data from the database.
		//
		// Update head first(head fast block, head full block) before deleting the data.
		if updateFn != nil {
			updateFn(hc.chainDb, parent)
		}
		// Update head header then.
		rawdb.WriteHeadHeaderHash(hc.chainDb, parentHash)

		// Remove the relative data from the database.
		if delFn != nil {
			delFn(batch, hash, num)
		}
		// Rewind header chain to new head.
		rawdb.DeleteHeader(batch, hash, num)
		rawdb.DeleteTd(batch, hash, num)
		rawdb.DeleteCanonicalHash(batch, num)

		hc.currentHeader.Store(parent)
		hc.currentHeaderHash = parentHash
		headHeaderGauge.Update(parent.Number.Int64())
	}
	batch.Write()

	// Clear out any stale content from the caches
	hc.headerCache.Purge()
	hc.tdCache.Purge()
	hc.numberCache.Purge()
}

// SetGenesis sets a new genesis block header for the chain
func (hc *HeaderChain) SetGenesis(head *types.Header) {
	hc.genesisHeader = head
}

// Config retrieves the header chain's chain configuration.
func (hc *HeaderChain) Config() *params.ChainConfig { return hc.config }

// Engine retrieves the header chain's consensus engine.
func (hc *HeaderChain) Engine() consensus.Engine { return hc.engine }

// GetBlock implements consensus.ChainReader, and returns nil for every input as
// a header chain does not have blocks available for retrieval.
func (hc *HeaderChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	return nil
}
