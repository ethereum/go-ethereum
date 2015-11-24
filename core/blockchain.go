// Copyright 2014 The go-ethereum Authors
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

// Package core implements the Ethereum consensus protocol.
package core

import (
	crand "crypto/rand"
	"errors"
	"fmt"
	"io"
	"math"
	"math/big"
	mrand "math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/hashicorp/golang-lru"
)

var (
	chainlogger = logger.NewLogger("CHAIN")
	jsonlogger  = logger.NewJsonLogger()

	blockInsertTimer = metrics.NewTimer("chain/inserts")

	ErrNoGenesis = errors.New("Genesis not found in chain")
)

const (
	headerCacheLimit    = 512
	bodyCacheLimit      = 256
	tdCacheLimit        = 1024
	blockCacheLimit     = 256
	maxFutureBlocks     = 256
	maxTimeFutureBlocks = 30
	// must be bumped when consensus algorithm is changed, this forces the upgradedb
	// command to be run (forces the blocks to be imported again using the new algorithm)
	BlockChainVersion = 3
)

// BlockChain represents the canonical chain given a database with a genesis
// block. The Blockchain manages chain imports, reverts, chain reorganisations.
//
// Importing blocks in to the block chain happens according to the set of rules
// defined by the two stage Validator. Processing of blocks is done using the
// Processor which processes the included transaction. The validation of the state
// is done in the second part of the Validator. Failing results in aborting of
// the import.
//
// The BlockChain also helps in returning blocks from **any** chain included
// in the database as well as blocks that represents the canonical chain. It's
// important to note that GetBlock can return any block and does not need to be
// included in the canonical one where as GetBlockByNumber always represents the
// canonical chain.
type BlockChain struct {
	chainDb      ethdb.Database
	eventMux     *event.TypeMux
	genesisBlock *types.Block
	// Last known total difficulty
	mu      sync.RWMutex
	chainmu sync.RWMutex
	tsmu    sync.RWMutex
	procmu  sync.RWMutex

	checkpoint       int           // checkpoint counts towards the new checkpoint
	currentHeader    *types.Header // Current head of the header chain (may be above the block chain!)
	currentBlock     *types.Block  // Current head of the block chain
	currentFastBlock *types.Block  // Current head of the fast-sync chain (may be above the block chain!)

	headerCache  *lru.Cache // Cache for the most recent block headers
	bodyCache    *lru.Cache // Cache for the most recent block bodies
	bodyRLPCache *lru.Cache // Cache for the most recent block bodies in RLP encoded format
	tdCache      *lru.Cache // Cache for the most recent block total difficulties
	blockCache   *lru.Cache // Cache for the most recent entire blocks
	futureBlocks *lru.Cache // future blocks are blocks added for later processing

	quit    chan struct{}
	running int32 // running must be called automically
	// procInterrupt must be atomically called
	procInterrupt int32 // interrupt signaler for block processing
	wg            sync.WaitGroup

	pow       pow.PoW
	rand      *mrand.Rand
	processor Processor
	validator Validator
}

// NewBlockChain returns a fully initialised block chain using information
// available in the database. It initialiser the default Ethereum Validator and
// Processor.
func NewBlockChain(chainDb ethdb.Database, pow pow.PoW, mux *event.TypeMux) (*BlockChain, error) {
	headerCache, _ := lru.New(headerCacheLimit)
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	tdCache, _ := lru.New(tdCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	futureBlocks, _ := lru.New(maxFutureBlocks)

	bc := &BlockChain{
		chainDb:      chainDb,
		eventMux:     mux,
		quit:         make(chan struct{}),
		headerCache:  headerCache,
		bodyCache:    bodyCache,
		bodyRLPCache: bodyRLPCache,
		tdCache:      tdCache,
		blockCache:   blockCache,
		futureBlocks: futureBlocks,
		pow:          pow,
	}
	// Seed a fast but crypto originating random generator
	seed, err := crand.Int(crand.Reader, big.NewInt(math.MaxInt64))
	if err != nil {
		return nil, err
	}
	bc.rand = mrand.New(mrand.NewSource(seed.Int64()))
	bc.SetValidator(NewBlockValidator(bc, pow))
	bc.SetProcessor(NewStateProcessor(bc))

	bc.genesisBlock = bc.GetBlockByNumber(0)
	if bc.genesisBlock == nil {
		reader, err := NewDefaultGenesisReader()
		if err != nil {
			return nil, err
		}
		bc.genesisBlock, err = WriteGenesisBlock(chainDb, reader)
		if err != nil {
			return nil, err
		}
		glog.V(logger.Info).Infoln("WARNING: Wrote default ethereum genesis block")
	}
	if err := bc.loadLastState(); err != nil {
		return nil, err
	}
	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for hash, _ := range BadHashes {
		if header := bc.GetHeader(hash); header != nil {
			glog.V(logger.Error).Infof("Found bad hash, rewinding chain to block #%d [%x…]", header.Number, header.ParentHash[:4])
			bc.SetHead(header.Number.Uint64() - 1)
			glog.V(logger.Error).Infoln("Chain rewind was successful, resuming normal operation")
		}
	}
	// Take ownership of this particular state
	go bc.update()
	return bc, nil
}

// loadLastState loads the last known chain state from the database. This method
// assumes that the chain manager mutex is held.
func (self *BlockChain) loadLastState() error {
	// Restore the last known head block
	head := GetHeadBlockHash(self.chainDb)
	if head == (common.Hash{}) {
		// Corrupt or empty database, init from scratch
		self.Reset()
	} else {
		if block := self.GetBlock(head); block != nil {
			// Block found, set as the current head
			self.currentBlock = block
		} else {
			// Corrupt or empty database, init from scratch
			self.Reset()
		}
	}
	// Restore the last known head header
	self.currentHeader = self.currentBlock.Header()
	if head := GetHeadHeaderHash(self.chainDb); head != (common.Hash{}) {
		if header := self.GetHeader(head); header != nil {
			self.currentHeader = header
		}
	}
	// Restore the last known head fast block
	self.currentFastBlock = self.currentBlock
	if head := GetHeadFastBlockHash(self.chainDb); head != (common.Hash{}) {
		if block := self.GetBlock(head); block != nil {
			self.currentFastBlock = block
		}
	}
	// Issue a status log and return
	headerTd := self.GetTd(self.currentHeader.Hash())
	blockTd := self.GetTd(self.currentBlock.Hash())
	fastTd := self.GetTd(self.currentFastBlock.Hash())

	glog.V(logger.Info).Infof("Last header: #%d [%x…] TD=%v", self.currentHeader.Number, self.currentHeader.Hash().Bytes()[:4], headerTd)
	glog.V(logger.Info).Infof("Last block: #%d [%x…] TD=%v", self.currentBlock.Number(), self.currentBlock.Hash().Bytes()[:4], blockTd)
	glog.V(logger.Info).Infof("Fast block: #%d [%x…] TD=%v", self.currentFastBlock.Number(), self.currentFastBlock.Hash().Bytes()[:4], fastTd)

	return nil
}

// SetHead rewinds the local chain to a new head. In the case of headers, everything
// above the new head will be deleted and the new one set. In the case of blocks
// though, the head may be further rewound if block bodies are missing (non-archive
// nodes after a fast sync).
func (bc *BlockChain) SetHead(head uint64) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Figure out the highest known canonical headers and/or blocks
	height := uint64(0)
	if bc.currentHeader != nil {
		if hh := bc.currentHeader.Number.Uint64(); hh > height {
			height = hh
		}
	}
	if bc.currentBlock != nil {
		if bh := bc.currentBlock.NumberU64(); bh > height {
			height = bh
		}
	}
	if bc.currentFastBlock != nil {
		if fbh := bc.currentFastBlock.NumberU64(); fbh > height {
			height = fbh
		}
	}
	// Gather all the hashes that need deletion
	drop := make(map[common.Hash]struct{})

	for bc.currentHeader != nil && bc.currentHeader.Number.Uint64() > head {
		drop[bc.currentHeader.Hash()] = struct{}{}
		bc.currentHeader = bc.GetHeader(bc.currentHeader.ParentHash)
	}
	for bc.currentBlock != nil && bc.currentBlock.NumberU64() > head {
		drop[bc.currentBlock.Hash()] = struct{}{}
		bc.currentBlock = bc.GetBlock(bc.currentBlock.ParentHash())
	}
	for bc.currentFastBlock != nil && bc.currentFastBlock.NumberU64() > head {
		drop[bc.currentFastBlock.Hash()] = struct{}{}
		bc.currentFastBlock = bc.GetBlock(bc.currentFastBlock.ParentHash())
	}
	// Roll back the canonical chain numbering
	for i := height; i > head; i-- {
		DeleteCanonicalHash(bc.chainDb, i)
	}
	// Delete everything found by the above rewind
	for hash, _ := range drop {
		DeleteHeader(bc.chainDb, hash)
		DeleteBody(bc.chainDb, hash)
		DeleteTd(bc.chainDb, hash)
	}
	// Clear out any stale content from the caches
	bc.headerCache.Purge()
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.blockCache.Purge()
	bc.futureBlocks.Purge()

	// Update all computed fields to the new head
	if bc.currentBlock == nil {
		bc.currentBlock = bc.genesisBlock
	}
	if bc.currentHeader == nil {
		bc.currentHeader = bc.genesisBlock.Header()
	}
	if bc.currentFastBlock == nil {
		bc.currentFastBlock = bc.genesisBlock
	}
	if err := WriteHeadBlockHash(bc.chainDb, bc.currentBlock.Hash()); err != nil {
		glog.Fatalf("failed to reset head block hash: %v", err)
	}
	if err := WriteHeadHeaderHash(bc.chainDb, bc.currentHeader.Hash()); err != nil {
		glog.Fatalf("failed to reset head header hash: %v", err)
	}
	if err := WriteHeadFastBlockHash(bc.chainDb, bc.currentFastBlock.Hash()); err != nil {
		glog.Fatalf("failed to reset head fast block hash: %v", err)
	}
	bc.loadLastState()
}

// FastSyncCommitHead sets the current head block to the one defined by the hash
// irrelevant what the chain contents were prior.
func (self *BlockChain) FastSyncCommitHead(hash common.Hash) error {
	// Make sure that both the block as well at its state trie exists
	block := self.GetBlock(hash)
	if block == nil {
		return fmt.Errorf("non existent block [%x…]", hash[:4])
	}
	if _, err := trie.NewSecure(block.Root(), self.chainDb); err != nil {
		return err
	}
	// If all checks out, manually set the head block
	self.mu.Lock()
	self.currentBlock = block
	self.mu.Unlock()

	glog.V(logger.Info).Infof("committed block #%d [%x…] as new head", block.Number(), hash[:4])
	return nil
}

// GasLimit returns the gas limit of the current HEAD block.
func (self *BlockChain) GasLimit() *big.Int {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentBlock.GasLimit()
}

// LastBlockHash return the hash of the HEAD block.
func (self *BlockChain) LastBlockHash() common.Hash {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentBlock.Hash()
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the blockchain's internal cache.
func (self *BlockChain) CurrentHeader() *types.Header {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentHeader
}

// CurrentBlock retrieves the current head block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (self *BlockChain) CurrentBlock() *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentBlock
}

// CurrentFastBlock retrieves the current fast-sync head block of the canonical
// chain. The block is retrieved from the blockchain's internal cache.
func (self *BlockChain) CurrentFastBlock() *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentFastBlock
}

// Status returns status information about the current chain such as the HEAD Td,
// the HEAD hash and the hash of the genesis block.
func (self *BlockChain) Status() (td *big.Int, currentBlock common.Hash, genesisBlock common.Hash) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.GetTd(self.currentBlock.Hash()), self.currentBlock.Hash(), self.genesisBlock.Hash()
}

// SetProcessor sets the processor required for making state modifications.
func (self *BlockChain) SetProcessor(processor Processor) {
	self.procmu.Lock()
	defer self.procmu.Unlock()
	self.processor = processor
}

// SetValidator sets the validator which is used to validate incoming blocks.
func (self *BlockChain) SetValidator(validator Validator) {
	self.procmu.Lock()
	defer self.procmu.Unlock()
	self.validator = validator
}

// Validator returns the current validator.
func (self *BlockChain) Validator() Validator {
	self.procmu.RLock()
	defer self.procmu.RUnlock()
	return self.validator
}

// Processor returns the current processor.
func (self *BlockChain) Processor() Processor {
	self.procmu.RLock()
	defer self.procmu.RUnlock()
	return self.processor
}

// AuxValidator returns the auxiliary validator (Proof of work atm)
func (self *BlockChain) AuxValidator() pow.PoW { return self.pow }

// State returns a new mutable state based on the current HEAD block.
func (self *BlockChain) State() (*state.StateDB, error) {
	return state.New(self.CurrentBlock().Root(), self.chainDb)
}

// Reset purges the entire blockchain, restoring it to its genesis state.
func (bc *BlockChain) Reset() {
	bc.ResetWithGenesisBlock(bc.genesisBlock)
}

// ResetWithGenesisBlock purges the entire blockchain, restoring it to the
// specified genesis state.
func (bc *BlockChain) ResetWithGenesisBlock(genesis *types.Block) {
	// Dump the entire block chain and purge the caches
	bc.SetHead(0)

	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Prepare the genesis block and reinitialise the chain
	if err := WriteTd(bc.chainDb, genesis.Hash(), genesis.Difficulty()); err != nil {
		glog.Fatalf("failed to write genesis block TD: %v", err)
	}
	if err := WriteBlock(bc.chainDb, genesis); err != nil {
		glog.Fatalf("failed to write genesis block: %v", err)
	}
	bc.genesisBlock = genesis
	bc.insert(bc.genesisBlock)
	bc.currentBlock = bc.genesisBlock
	bc.currentHeader = bc.genesisBlock.Header()
	bc.currentFastBlock = bc.genesisBlock
}

// Export writes the active chain to the given writer.
func (self *BlockChain) Export(w io.Writer) error {
	if err := self.ExportN(w, uint64(0), self.currentBlock.NumberU64()); err != nil {
		return err
	}
	return nil
}

// ExportN writes a subset of the active chain to the given writer.
func (self *BlockChain) ExportN(w io.Writer, first uint64, last uint64) error {
	self.mu.RLock()
	defer self.mu.RUnlock()

	if first > last {
		return fmt.Errorf("export failed: first (%d) is greater than last (%d)", first, last)
	}

	glog.V(logger.Info).Infof("exporting %d blocks...\n", last-first+1)

	for nr := first; nr <= last; nr++ {
		block := self.GetBlockByNumber(nr)
		if block == nil {
			return fmt.Errorf("export failed on #%d: not found", nr)
		}

		if err := block.EncodeRLP(w); err != nil {
			return err
		}
	}

	return nil
}

// insert injects a new head block into the current block chain. This method
// assumes that the block is indeed a true head. It will also reset the head
// header and the head fast sync block to this very same block if they are older
// or if they are on a different side chain.
//
// Note, this function assumes that the `mu` mutex is held!
func (bc *BlockChain) insert(block *types.Block) {
	// If the block is on a side chain or an unknown one, force other heads onto it too
	updateHeads := GetCanonicalHash(bc.chainDb, block.NumberU64()) != block.Hash()

	// Add the block to the canonical chain number scheme and mark as the head
	if err := WriteCanonicalHash(bc.chainDb, block.Hash(), block.NumberU64()); err != nil {
		glog.Fatalf("failed to insert block number: %v", err)
	}
	if err := WriteHeadBlockHash(bc.chainDb, block.Hash()); err != nil {
		glog.Fatalf("failed to insert head block hash: %v", err)
	}
	bc.currentBlock = block

	// If the block is better than out head or is on a different chain, force update heads
	if updateHeads {
		if err := WriteHeadHeaderHash(bc.chainDb, block.Hash()); err != nil {
			glog.Fatalf("failed to insert head header hash: %v", err)
		}
		bc.currentHeader = block.Header()

		if err := WriteHeadFastBlockHash(bc.chainDb, block.Hash()); err != nil {
			glog.Fatalf("failed to insert head fast block hash: %v", err)
		}
		bc.currentFastBlock = block
	}
}

// Accessors
func (bc *BlockChain) Genesis() *types.Block {
	return bc.genesisBlock
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (bc *BlockChain) HasHeader(hash common.Hash) bool {
	return bc.GetHeader(hash) != nil
}

// GetHeader retrieves a block header from the database by hash, caching it if
// found.
func (self *BlockChain) GetHeader(hash common.Hash) *types.Header {
	// Short circuit if the header's already in the cache, retrieve otherwise
	if header, ok := self.headerCache.Get(hash); ok {
		return header.(*types.Header)
	}
	header := GetHeader(self.chainDb, hash)
	if header == nil {
		return nil
	}
	// Cache the found header for next time and return
	self.headerCache.Add(header.Hash(), header)
	return header
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (self *BlockChain) GetHeaderByNumber(number uint64) *types.Header {
	hash := GetCanonicalHash(self.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return self.GetHeader(hash)
}

// GetBody retrieves a block body (transactions and uncles) from the database by
// hash, caching it if found.
func (self *BlockChain) GetBody(hash common.Hash) *types.Body {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := self.bodyCache.Get(hash); ok {
		body := cached.(*types.Body)
		return body
	}
	body := GetBody(self.chainDb, hash)
	if body == nil {
		return nil
	}
	// Cache the found body for next time and return
	self.bodyCache.Add(hash, body)
	return body
}

// GetBodyRLP retrieves a block body in RLP encoding from the database by hash,
// caching it if found.
func (self *BlockChain) GetBodyRLP(hash common.Hash) rlp.RawValue {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := self.bodyRLPCache.Get(hash); ok {
		return cached.(rlp.RawValue)
	}
	body := GetBodyRLP(self.chainDb, hash)
	if len(body) == 0 {
		return nil
	}
	// Cache the found body for next time and return
	self.bodyRLPCache.Add(hash, body)
	return body
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (self *BlockChain) GetTd(hash common.Hash) *big.Int {
	// Short circuit if the td's already in the cache, retrieve otherwise
	if cached, ok := self.tdCache.Get(hash); ok {
		return cached.(*big.Int)
	}
	td := GetTd(self.chainDb, hash)
	if td == nil {
		return nil
	}
	// Cache the found body for next time and return
	self.tdCache.Add(hash, td)
	return td
}

// HasBlock checks if a block is fully present in the database or not, caching
// it if present.
func (bc *BlockChain) HasBlock(hash common.Hash) bool {
	return bc.GetBlock(hash) != nil
}

// GetBlock retrieves a block from the database by hash, caching it if found.
func (self *BlockChain) GetBlock(hash common.Hash) *types.Block {
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := self.blockCache.Get(hash); ok {
		return block.(*types.Block)
	}
	block := GetBlock(self.chainDb, hash)
	if block == nil {
		return nil
	}
	// Cache the found block for next time and return
	self.blockCache.Add(block.Hash(), block)
	return block
}

// GetBlockByNumber retrieves a block from the database by number, caching it
// (associated with its hash) if found.
func (self *BlockChain) GetBlockByNumber(number uint64) *types.Block {
	hash := GetCanonicalHash(self.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return self.GetBlock(hash)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (self *BlockChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	// Get the origin header from which to fetch
	header := self.GetHeader(hash)
	if header == nil {
		return nil
	}
	// Iterate the headers until enough is collected or the genesis reached
	chain := make([]common.Hash, 0, max)
	for i := uint64(0); i < max; i++ {
		if header = self.GetHeader(header.ParentHash); header == nil {
			break
		}
		chain = append(chain, header.Hash())
		if header.Number.Cmp(common.Big0) == 0 {
			break
		}
	}
	return chain
}

// [deprecated by eth/62]
// GetBlocksFromHash returns the block corresponding to hash and up to n-1 ancestors.
func (self *BlockChain) GetBlocksFromHash(hash common.Hash, n int) (blocks []*types.Block) {
	for i := 0; i < n; i++ {
		block := self.GetBlock(hash)
		if block == nil {
			break
		}
		blocks = append(blocks, block)
		hash = block.ParentHash()
	}
	return
}

// GetUnclesInChain retrieves all the uncles from a given block backwards until
// a specific distance is reached.
func (self *BlockChain) GetUnclesInChain(block *types.Block, length int) []*types.Header {
	uncles := []*types.Header{}
	for i := 0; block != nil && i < length; i++ {
		uncles = append(uncles, block.Uncles()...)
		block = self.GetBlock(block.ParentHash())
	}
	return uncles
}

// Stop stops the blockchain service. If any imports are currently in progress
// it will abort them using the procInterrupt.
func (bc *BlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}
	close(bc.quit)
	atomic.StoreInt32(&bc.procInterrupt, 1)

	bc.wg.Wait()

	glog.V(logger.Info).Infoln("Chain manager stopped")
}

func (self *BlockChain) procFutureBlocks() {
	blocks := make([]*types.Block, self.futureBlocks.Len())
	for i, hash := range self.futureBlocks.Keys() {
		block, _ := self.futureBlocks.Get(hash)
		blocks[i] = block.(*types.Block)
	}
	if len(blocks) > 0 {
		types.BlockBy(types.Number).Sort(blocks)
		self.InsertChain(blocks)
	}
}

type writeStatus byte

const (
	NonStatTy writeStatus = iota
	CanonStatTy
	SplitStatTy
	SideStatTy
)

// writeHeader writes a header into the local chain, given that its parent is
// already known. If the total difficulty of the newly inserted header becomes
// greater than the current known TD, the canonical chain is re-routed.
//
// Note: This method is not concurrent-safe with inserting blocks simultaneously
// into the chain, as side effects caused by reorganisations cannot be emulated
// without the real blocks. Hence, writing headers directly should only be done
// in two scenarios: pure-header mode of operation (light clients), or properly
// separated header/block phases (non-archive clients).
func (self *BlockChain) writeHeader(header *types.Header) error {
	self.wg.Add(1)
	defer self.wg.Done()

	// Calculate the total difficulty of the header
	ptd := self.GetTd(header.ParentHash)
	if ptd == nil {
		return ParentError(header.ParentHash)
	}
	td := new(big.Int).Add(header.Difficulty, ptd)

	// Make sure no inconsistent state is leaked during insertion
	self.mu.Lock()
	defer self.mu.Unlock()

	// If the total difficulty is higher than our known, add it to the canonical chain
	if td.Cmp(self.GetTd(self.currentHeader.Hash())) > 0 {
		// Delete any canonical number assignments above the new head
		for i := header.Number.Uint64() + 1; GetCanonicalHash(self.chainDb, i) != (common.Hash{}); i++ {
			DeleteCanonicalHash(self.chainDb, i)
		}
		// Overwrite any stale canonical number assignments
		head := self.GetHeader(header.ParentHash)
		for GetCanonicalHash(self.chainDb, head.Number.Uint64()) != head.Hash() {
			WriteCanonicalHash(self.chainDb, head.Hash(), head.Number.Uint64())
			head = self.GetHeader(head.ParentHash)
		}
		// Extend the canonical chain with the new header
		if err := WriteCanonicalHash(self.chainDb, header.Hash(), header.Number.Uint64()); err != nil {
			glog.Fatalf("failed to insert header number: %v", err)
		}
		if err := WriteHeadHeaderHash(self.chainDb, header.Hash()); err != nil {
			glog.Fatalf("failed to insert head header hash: %v", err)
		}
		self.currentHeader = types.CopyHeader(header)
	}
	// Irrelevant of the canonical status, write the header itself to the database
	if err := WriteTd(self.chainDb, header.Hash(), td); err != nil {
		glog.Fatalf("failed to write header total difficulty: %v", err)
	}
	if err := WriteHeader(self.chainDb, header); err != nil {
		glog.Fatalf("filed to write header contents: %v", err)
	}
	return nil
}

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify parameter can be used to fine tune whether nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verfy nonces, as well as
// because nonces can be verified sparsely, not needing to check each.
func (self *BlockChain) InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error) {
	self.wg.Add(1)
	defer self.wg.Done()

	// Make sure only one thread manipulates the chain at once
	self.chainmu.Lock()
	defer self.chainmu.Unlock()

	// Collect some import statistics to report on
	stats := struct{ processed, ignored int }{}
	start := time.Now()

	// Generate the list of headers that should be POW verified
	verify := make([]bool, len(chain))
	for i := 0; i < len(verify)/checkFreq; i++ {
		index := i*checkFreq + self.rand.Intn(checkFreq)
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
			if atomic.LoadInt32(&self.procInterrupt) == 1 {
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
			if self.HasHeader(hash) {
				continue
			}
			// Verify that the header honors the chain parameters
			checkPow := verify[index]

			var err error
			if index == 0 {
				err = self.Validator().ValidateHeader(header, self.GetHeader(header.ParentHash), checkPow)
			} else {
				err = self.Validator().ValidateHeader(header, chain[index-1], checkPow)
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
		if atomic.LoadInt32(&self.procInterrupt) == 1 {
			glog.V(logger.Debug).Infoln("premature abort during header chain processing")
			break
		}
		hash := header.Hash()

		// If the header's already known, skip it, otherwise store
		if self.HasHeader(hash) {
			stats.ignored++
			continue
		}
		if err := self.writeHeader(header); err != nil {
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

// Rollback is designed to remove a chain of links from the database that aren't
// certain enough to be valid.
func (self *BlockChain) Rollback(chain []common.Hash) {
	self.mu.Lock()
	defer self.mu.Unlock()

	for i := len(chain) - 1; i >= 0; i-- {
		hash := chain[i]

		if self.currentHeader.Hash() == hash {
			self.currentHeader = self.GetHeader(self.currentHeader.ParentHash)
			WriteHeadHeaderHash(self.chainDb, self.currentHeader.Hash())
		}
		if self.currentFastBlock.Hash() == hash {
			self.currentFastBlock = self.GetBlock(self.currentFastBlock.ParentHash())
			WriteHeadFastBlockHash(self.chainDb, self.currentFastBlock.Hash())
		}
		if self.currentBlock.Hash() == hash {
			self.currentBlock = self.GetBlock(self.currentBlock.ParentHash())
			WriteHeadBlockHash(self.chainDb, self.currentBlock.Hash())
		}
	}
}

// InsertReceiptChain attempts to complete an already existing header chain with
// transaction and receipt data.
func (self *BlockChain) InsertReceiptChain(blockChain types.Blocks, receiptChain []types.Receipts) (int, error) {
	self.wg.Add(1)
	defer self.wg.Done()

	// Collect some import statistics to report on
	stats := struct{ processed, ignored int32 }{}
	start := time.Now()

	// Create the block importing task queue and worker functions
	tasks := make(chan int, len(blockChain))
	for i := 0; i < len(blockChain) && i < len(receiptChain); i++ {
		tasks <- i
	}
	close(tasks)

	errs, failed := make([]error, len(tasks)), int32(0)
	process := func(worker int) {
		for index := range tasks {
			block, receipts := blockChain[index], receiptChain[index]

			// Short circuit insertion if shutting down or processing failed
			if atomic.LoadInt32(&self.procInterrupt) == 1 {
				return
			}
			if atomic.LoadInt32(&failed) > 0 {
				return
			}
			// Short circuit if the owner header is unknown
			if !self.HasHeader(block.Hash()) {
				errs[index] = fmt.Errorf("containing header #%d [%x…] unknown", block.Number(), block.Hash().Bytes()[:4])
				atomic.AddInt32(&failed, 1)
				return
			}
			// Skip if the entire data is already known
			if self.HasBlock(block.Hash()) {
				atomic.AddInt32(&stats.ignored, 1)
				continue
			}
			// Compute all the non-consensus fields of the receipts
			transactions, logIndex := block.Transactions(), uint(0)
			for j := 0; j < len(receipts); j++ {
				// The transaction hash can be retrieved from the transaction itself
				receipts[j].TxHash = transactions[j].Hash()

				// The contract address can be derived from the transaction itself
				if MessageCreatesContract(transactions[j]) {
					from, _ := transactions[j].From()
					receipts[j].ContractAddress = crypto.CreateAddress(from, transactions[j].Nonce())
				}
				// The used gas can be calculated based on previous receipts
				if j == 0 {
					receipts[j].GasUsed = new(big.Int).Set(receipts[j].CumulativeGasUsed)
				} else {
					receipts[j].GasUsed = new(big.Int).Sub(receipts[j].CumulativeGasUsed, receipts[j-1].CumulativeGasUsed)
				}
				// The derived log fields can simply be set from the block and transaction
				for k := 0; k < len(receipts[j].Logs); k++ {
					receipts[j].Logs[k].BlockNumber = block.NumberU64()
					receipts[j].Logs[k].BlockHash = block.Hash()
					receipts[j].Logs[k].TxHash = receipts[j].TxHash
					receipts[j].Logs[k].TxIndex = uint(j)
					receipts[j].Logs[k].Index = logIndex
					logIndex++
				}
			}
			// Write all the data out into the database
			if err := WriteBody(self.chainDb, block.Hash(), &types.Body{block.Transactions(), block.Uncles()}); err != nil {
				errs[index] = fmt.Errorf("failed to write block body: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			if err := WriteBlockReceipts(self.chainDb, block.Hash(), receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write block receipts: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			if err := WriteMipmapBloom(self.chainDb, block.NumberU64(), receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write log blooms: %v", err)
				atomic.AddInt32(&failed, 1)
				glog.Fatal(errs[index])
				return
			}
			atomic.AddInt32(&stats.processed, 1)
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
	if atomic.LoadInt32(&self.procInterrupt) == 1 {
		glog.V(logger.Debug).Infoln("premature abort during receipt chain processing")
		return 0, nil
	}
	// Update the head fast sync block if better
	self.mu.Lock()
	head := blockChain[len(errs)-1]
	if self.GetTd(self.currentFastBlock.Hash()).Cmp(self.GetTd(head.Hash())) < 0 {
		if err := WriteHeadFastBlockHash(self.chainDb, head.Hash()); err != nil {
			glog.Fatalf("failed to update head fast block hash: %v", err)
		}
		self.currentFastBlock = head
	}
	self.mu.Unlock()

	// Report some public statistics so the user has a clue what's going on
	first, last := blockChain[0], blockChain[len(blockChain)-1]
	glog.V(logger.Info).Infof("imported %d receipt(s) (%d ignored) in %v. #%d [%x… / %x…]", stats.processed, stats.ignored,
		time.Since(start), last.Number(), first.Hash().Bytes()[:4], last.Hash().Bytes()[:4])

	return 0, nil
}

// WriteBlock writes the block to the chain.
func (self *BlockChain) WriteBlock(block *types.Block) (status writeStatus, err error) {
	self.wg.Add(1)
	defer self.wg.Done()

	// Calculate the total difficulty of the block
	ptd := self.GetTd(block.ParentHash())
	if ptd == nil {
		return NonStatTy, ParentError(block.ParentHash())
	}
	td := new(big.Int).Add(block.Difficulty(), ptd)

	// Make sure no inconsistent state is leaked during insertion
	self.mu.Lock()
	defer self.mu.Unlock()

	// If the total difficulty is higher than our known, add it to the canonical chain
	if td.Cmp(self.GetTd(self.currentBlock.Hash())) > 0 {
		// Reorganize the chain if the parent is not the head block
		if block.ParentHash() != self.currentBlock.Hash() {
			if err := self.reorg(self.currentBlock, block); err != nil {
				return NonStatTy, err
			}
		}
		// Insert the block as the new head of the chain
		self.insert(block)
		status = CanonStatTy
	} else {
		status = SideStatTy
	}
	// Irrelevant of the canonical status, write the block itself to the database
	if err := WriteTd(self.chainDb, block.Hash(), td); err != nil {
		glog.Fatalf("failed to write block total difficulty: %v", err)
	}
	if err := WriteBlock(self.chainDb, block); err != nil {
		glog.Fatalf("filed to write block contents: %v", err)
	}
	self.futureBlocks.Remove(block.Hash())

	return
}

// InsertChain will attempt to insert the given chain in to the canonical chain or, otherwise, create a fork. It an error is returned
// it will return the index number of the failing block as well an error describing what went wrong (for possible errors see core/errors.go).
func (self *BlockChain) InsertChain(chain types.Blocks) (int, error) {
	self.wg.Add(1)
	defer self.wg.Done()

	self.chainmu.Lock()
	defer self.chainmu.Unlock()

	// A queued approach to delivering events. This is generally
	// faster than direct delivery and requires much less mutex
	// acquiring.
	var (
		stats         struct{ queued, processed, ignored int }
		events        = make([]interface{}, 0, len(chain))
		coalescedLogs vm.Logs
		tstart        = time.Now()

		nonceChecked = make([]bool, len(chain))
	)

	// Start the parallel nonce verifier.
	nonceAbort, nonceResults := verifyNoncesFromBlocks(self.pow, chain)
	defer close(nonceAbort)

	txcount := 0
	for i, block := range chain {
		if atomic.LoadInt32(&self.procInterrupt) == 1 {
			glog.V(logger.Debug).Infoln("Premature abort during block chain processing")
			break
		}

		bstart := time.Now()
		// Wait for block i's nonce to be verified before processing
		// its state transition.
		for !nonceChecked[i] {
			r := <-nonceResults
			nonceChecked[r.index] = true
			if !r.valid {
				block := chain[r.index]
				return r.index, &BlockNonceErr{Hash: block.Hash(), Number: block.Number(), Nonce: block.Nonce()}
			}
		}

		if BadHashes[block.Hash()] {
			err := BadHashError(block.Hash())
			reportBlock(block, err)
			return i, err
		}
		// Stage 1 validation of the block using the chain's validator
		// interface.
		err := self.Validator().ValidateBlock(block)
		if err != nil {
			if IsKnownBlockErr(err) {
				stats.ignored++
				continue
			}

			if err == BlockFutureErr {
				// Allow up to MaxFuture second in the future blocks. If this limit
				// is exceeded the chain is discarded and processed at a later time
				// if given.
				max := big.NewInt(time.Now().Unix() + maxTimeFutureBlocks)
				if block.Time().Cmp(max) == 1 {
					return i, fmt.Errorf("%v: BlockFutureErr, %v > %v", BlockFutureErr, block.Time(), max)
				}

				self.futureBlocks.Add(block.Hash(), block)
				stats.queued++
				continue
			}

			if IsParentErr(err) && self.futureBlocks.Contains(block.ParentHash()) {
				self.futureBlocks.Add(block.Hash(), block)
				stats.queued++
				continue
			}

			reportBlock(block, err)

			return i, err
		}

		// Create a new statedb using the parent block and report an
		// error if it fails.
		statedb, err := state.New(self.GetBlock(block.ParentHash()).Root(), self.chainDb)
		if err != nil {
			reportBlock(block, err)
			return i, err
		}
		// Process block using the parent state as reference point.
		receipts, logs, usedGas, err := self.processor.Process(block, statedb)
		if err != nil {
			reportBlock(block, err)
			return i, err
		}
		// Validate the state using the default validator
		err = self.Validator().ValidateState(block, self.GetBlock(block.ParentHash()), statedb, receipts, usedGas)
		if err != nil {
			reportBlock(block, err)
			return i, err
		}
		// Write state changes to database
		_, err = statedb.Commit()
		if err != nil {
			return i, err
		}

		// coalesce logs for later processing
		coalescedLogs = append(coalescedLogs, logs...)

		if err := WriteBlockReceipts(self.chainDb, block.Hash(), receipts); err != nil {
			return i, err
		}

		txcount += len(block.Transactions())
		// write the block to the chain and get the status
		status, err := self.WriteBlock(block)
		if err != nil {
			return i, err
		}

		switch status {
		case CanonStatTy:
			if glog.V(logger.Debug) {
				glog.Infof("[%v] inserted block #%d (%d TXs %v G %d UNCs) (%x...). Took %v\n", time.Now().UnixNano(), block.Number(), len(block.Transactions()), block.GasUsed(), len(block.Uncles()), block.Hash().Bytes()[0:4], time.Since(bstart))
			}
			events = append(events, ChainEvent{block, block.Hash(), logs})

			// This puts transactions in a extra db for rpc
			if err := WriteTransactions(self.chainDb, block); err != nil {
				return i, err
			}
			// store the receipts
			if err := WriteReceipts(self.chainDb, receipts); err != nil {
				return i, err
			}
			// Write map map bloom filters
			if err := WriteMipmapBloom(self.chainDb, block.NumberU64(), receipts); err != nil {
				return i, err
			}
		case SideStatTy:
			if glog.V(logger.Detail) {
				glog.Infof("inserted forked block #%d (TD=%v) (%d TXs %d UNCs) (%x...). Took %v\n", block.Number(), block.Difficulty(), len(block.Transactions()), len(block.Uncles()), block.Hash().Bytes()[0:4], time.Since(bstart))
			}
			events = append(events, ChainSideEvent{block, logs})

		case SplitStatTy:
			events = append(events, ChainSplitEvent{block, logs})
		}
		stats.processed++
	}

	if (stats.queued > 0 || stats.processed > 0 || stats.ignored > 0) && bool(glog.V(logger.Info)) {
		tend := time.Since(tstart)
		start, end := chain[0], chain[len(chain)-1]
		glog.Infof("imported %d block(s) (%d queued %d ignored) including %d txs in %v. #%v [%x / %x]\n", stats.processed, stats.queued, stats.ignored, txcount, tend, end.Number(), start.Hash().Bytes()[:4], end.Hash().Bytes()[:4])
	}
	go self.postChainEvents(events, coalescedLogs)

	return 0, nil
}

// reorgs takes two blocks, an old chain and a new chain and will reconstruct the blocks and inserts them
// to be part of the new canonical chain and accumulates potential missing transactions and post an
// event about them
func (self *BlockChain) reorg(oldBlock, newBlock *types.Block) error {
	var (
		newChain    types.Blocks
		commonBlock *types.Block
		oldStart    = oldBlock
		newStart    = newBlock
		deletedTxs  types.Transactions
	)

	// first reduce whoever is higher bound
	if oldBlock.NumberU64() > newBlock.NumberU64() {
		// reduce old chain
		for oldBlock = oldBlock; oldBlock != nil && oldBlock.NumberU64() != newBlock.NumberU64(); oldBlock = self.GetBlock(oldBlock.ParentHash()) {
			deletedTxs = append(deletedTxs, oldBlock.Transactions()...)
		}
	} else {
		// reduce new chain and append new chain blocks for inserting later on
		for newBlock = newBlock; newBlock != nil && newBlock.NumberU64() != oldBlock.NumberU64(); newBlock = self.GetBlock(newBlock.ParentHash()) {
			newChain = append(newChain, newBlock)
		}
	}
	if oldBlock == nil {
		return fmt.Errorf("Invalid old chain")
	}
	if newBlock == nil {
		return fmt.Errorf("Invalid new chain")
	}

	numSplit := newBlock.Number()
	for {
		if oldBlock.Hash() == newBlock.Hash() {
			commonBlock = oldBlock
			break
		}
		newChain = append(newChain, newBlock)
		deletedTxs = append(deletedTxs, oldBlock.Transactions()...)

		oldBlock, newBlock = self.GetBlock(oldBlock.ParentHash()), self.GetBlock(newBlock.ParentHash())
		if oldBlock == nil {
			return fmt.Errorf("Invalid old chain")
		}
		if newBlock == nil {
			return fmt.Errorf("Invalid new chain")
		}
	}

	if glog.V(logger.Debug) {
		commonHash := commonBlock.Hash()
		glog.Infof("Chain split detected @ %x. Reorganising chain from #%v %x to %x", commonHash[:4], numSplit, oldStart.Hash().Bytes()[:4], newStart.Hash().Bytes()[:4])
	}

	var addedTxs types.Transactions
	// insert blocks. Order does not matter. Last block will be written in ImportChain itself which creates the new head properly
	for _, block := range newChain {
		// insert the block in the canonical way, re-writing history
		self.insert(block)
		// write canonical receipts and transactions
		if err := WriteTransactions(self.chainDb, block); err != nil {
			return err
		}
		receipts := GetBlockReceipts(self.chainDb, block.Hash())
		// write receipts
		if err := WriteReceipts(self.chainDb, receipts); err != nil {
			return err
		}
		// Write map map bloom filters
		if err := WriteMipmapBloom(self.chainDb, block.NumberU64(), receipts); err != nil {
			return err
		}

		addedTxs = append(addedTxs, block.Transactions()...)
	}

	// calculate the difference between deleted and added transactions
	diff := types.TxDifference(deletedTxs, addedTxs)
	// When transactions get deleted from the database that means the
	// receipts that were created in the fork must also be deleted
	for _, tx := range diff {
		DeleteReceipt(self.chainDb, tx.Hash())
		DeleteTransaction(self.chainDb, tx.Hash())
	}
	// Must be posted in a goroutine because of the transaction pool trying
	// to acquire the chain manager lock
	go self.eventMux.Post(RemovedTransactionEvent{diff})

	return nil
}

// postChainEvents iterates over the events generated by a chain insertion and
// posts them into the event mux.
func (self *BlockChain) postChainEvents(events []interface{}, logs vm.Logs) {
	// post event logs for further processing
	self.eventMux.Post(logs)
	for _, event := range events {
		if event, ok := event.(ChainEvent); ok {
			// We need some control over the mining operation. Acquiring locks and waiting for the miner to create new block takes too long
			// and in most cases isn't even necessary.
			if self.LastBlockHash() == event.Hash {
				self.eventMux.Post(ChainHeadEvent{event.Block})
			}
		}
		// Fire the insertion events individually too
		self.eventMux.Post(event)
	}
}

func (self *BlockChain) update() {
	futureTimer := time.Tick(5 * time.Second)
	for {
		select {
		case <-futureTimer:
			self.procFutureBlocks()
		case <-self.quit:
			return
		}
	}
}

// reportBlock reports the given block and error using the canonical block
// reporting tool. Reporting the block to the service is handled in a separate
// goroutine.
func reportBlock(block *types.Block, err error) {
	if glog.V(logger.Error) {
		glog.Errorf("Bad block #%v (%s)\n", block.Number(), block.Hash().Hex())
		glog.Errorf("    %v", err)
	}
	go ReportBlock(block, err)
}
