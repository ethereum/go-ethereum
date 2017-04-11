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
	"errors"
	"fmt"
	"io"
	"math/big"
	mrand "math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/hashicorp/golang-lru"
)

var (
	blockInsertTimer = metrics.NewTimer("chain/inserts")

	ErrNoGenesis = errors.New("Genesis not found in chain")
)

const (
	bodyCacheLimit      = 256
	blockCacheLimit     = 256
	maxFutureBlocks     = 256
	maxTimeFutureBlocks = 30
	// must be bumped when consensus algorithm is changed, this forces the upgradedb
	// command to be run (forces the blocks to be imported again using the new algorithm)
	BlockChainVersion = 3
	badBlockLimit     = 10
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
	config *params.ChainConfig // chain & network configuration

	hc           *HeaderChain
	chainDb      ethdb.Database
	eventMux     *event.TypeMux
	genesisBlock *types.Block

	mu      sync.RWMutex // global mutex for locking chain operations
	chainmu sync.RWMutex // blockchain insertion lock
	procmu  sync.RWMutex // block processor lock

	checkpoint       int          // checkpoint counts towards the new checkpoint
	currentBlock     *types.Block // Current head of the block chain
	currentFastBlock *types.Block // Current head of the fast-sync chain (may be above the block chain!)

	stateCache   *state.StateDB // State database to reuse between imports (contains state cache)
	bodyCache    *lru.Cache     // Cache for the most recent block bodies
	bodyRLPCache *lru.Cache     // Cache for the most recent block bodies in RLP encoded format
	blockCache   *lru.Cache     // Cache for the most recent entire blocks
	futureBlocks *lru.Cache     // future blocks are blocks added for later processing

	quit    chan struct{} // blockchain quit channel
	running int32         // running must be called atomically
	// procInterrupt must be atomically called
	procInterrupt int32          // interrupt signaler for block processing
	wg            sync.WaitGroup // chain processing wait group for shutting down

	engine    consensus.Engine
	processor Processor // block processor interface
	validator Validator // block and state validator interface
	vmConfig  vm.Config

	badBlocks *lru.Cache // Bad block cache
}

// NewBlockChain returns a fully initialised block chain using information
// available in the database. It initialises the default Ethereum Validator and
// Processor.
func NewBlockChain(chainDb ethdb.Database, config *params.ChainConfig, engine consensus.Engine, mux *event.TypeMux, vmConfig vm.Config) (*BlockChain, error) {
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	futureBlocks, _ := lru.New(maxFutureBlocks)
	badBlocks, _ := lru.New(badBlockLimit)

	bc := &BlockChain{
		config:       config,
		chainDb:      chainDb,
		eventMux:     mux,
		quit:         make(chan struct{}),
		bodyCache:    bodyCache,
		bodyRLPCache: bodyRLPCache,
		blockCache:   blockCache,
		futureBlocks: futureBlocks,
		engine:       engine,
		vmConfig:     vmConfig,
		badBlocks:    badBlocks,
	}
	bc.SetValidator(NewBlockValidator(config, bc, engine))
	bc.SetProcessor(NewStateProcessor(config, bc, engine))

	var err error
	bc.hc, err = NewHeaderChain(chainDb, config, engine, bc.getProcInterrupt)
	if err != nil {
		return nil, err
	}
	bc.genesisBlock = bc.GetBlockByNumber(0)
	if bc.genesisBlock == nil {
		return nil, ErrNoGenesis
	}
	if err := bc.loadLastState(); err != nil {
		return nil, err
	}
	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for hash := range BadHashes {
		if header := bc.GetHeaderByHash(hash); header != nil {
			// get the canonical block corresponding to the offending header's number
			headerByNumber := bc.GetHeaderByNumber(header.Number.Uint64())
			// make sure the headerByNumber (if present) is in our current canonical chain
			if headerByNumber != nil && headerByNumber.Hash() == header.Hash() {
				log.Error("Found bad hash, rewinding chain", "number", header.Number, "hash", header.ParentHash)
				bc.SetHead(header.Number.Uint64() - 1)
				log.Error("Chain rewind was successful, resuming normal operation")
			}
		}
	}
	// Take ownership of this particular state
	go bc.update()
	return bc, nil
}

func (self *BlockChain) getProcInterrupt() bool {
	return atomic.LoadInt32(&self.procInterrupt) == 1
}

// loadLastState loads the last known chain state from the database. This method
// assumes that the chain manager mutex is held.
func (self *BlockChain) loadLastState() error {
	// Restore the last known head block
	head := GetHeadBlockHash(self.chainDb)
	if head == (common.Hash{}) {
		// Corrupt or empty database, init from scratch
		log.Warn("Empty database, resetting chain")
		return self.Reset()
	}
	// Make sure the entire head block is available
	currentBlock := self.GetBlockByHash(head)
	if currentBlock == nil {
		// Corrupt or empty database, init from scratch
		log.Warn("Head block missing, resetting chain", "hash", head)
		return self.Reset()
	}
	// Make sure the state associated with the block is available
	if _, err := state.New(currentBlock.Root(), self.chainDb); err != nil {
		// Dangling block without a state associated, init from scratch
		log.Warn("Head state missing, resetting chain", "number", currentBlock.Number(), "hash", currentBlock.Hash())
		return self.Reset()
	}
	// Everything seems to be fine, set as the head block
	self.currentBlock = currentBlock

	// Restore the last known head header
	currentHeader := self.currentBlock.Header()
	if head := GetHeadHeaderHash(self.chainDb); head != (common.Hash{}) {
		if header := self.GetHeaderByHash(head); header != nil {
			currentHeader = header
		}
	}
	self.hc.SetCurrentHeader(currentHeader)

	// Restore the last known head fast block
	self.currentFastBlock = self.currentBlock
	if head := GetHeadFastBlockHash(self.chainDb); head != (common.Hash{}) {
		if block := self.GetBlockByHash(head); block != nil {
			self.currentFastBlock = block
		}
	}
	// Initialize a statedb cache to ensure singleton account bloom filter generation
	statedb, err := state.New(self.currentBlock.Root(), self.chainDb)
	if err != nil {
		return err
	}
	self.stateCache = statedb

	// Issue a status log for the user
	headerTd := self.GetTd(currentHeader.Hash(), currentHeader.Number.Uint64())
	blockTd := self.GetTd(self.currentBlock.Hash(), self.currentBlock.NumberU64())
	fastTd := self.GetTd(self.currentFastBlock.Hash(), self.currentFastBlock.NumberU64())

	log.Info("Loaded most recent local header", "number", currentHeader.Number, "hash", currentHeader.Hash(), "td", headerTd)
	log.Info("Loaded most recent local full block", "number", self.currentBlock.Number(), "hash", self.currentBlock.Hash(), "td", blockTd)
	log.Info("Loaded most recent local fast block", "number", self.currentFastBlock.Number(), "hash", self.currentFastBlock.Hash(), "td", fastTd)

	return nil
}

// SetHead rewinds the local chain to a new head. In the case of headers, everything
// above the new head will be deleted and the new one set. In the case of blocks
// though, the head may be further rewound if block bodies are missing (non-archive
// nodes after a fast sync).
func (bc *BlockChain) SetHead(head uint64) error {
	log.Warn("Rewinding blockchain", "target", head)

	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Rewind the header chain, deleting all block bodies until then
	delFn := func(hash common.Hash, num uint64) {
		DeleteBody(bc.chainDb, hash, num)
	}
	bc.hc.SetHead(head, delFn)
	currentHeader := bc.hc.CurrentHeader()

	// Clear out any stale content from the caches
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.blockCache.Purge()
	bc.futureBlocks.Purge()

	// Rewind the block chain, ensuring we don't end up with a stateless head block
	if bc.currentBlock != nil && currentHeader.Number.Uint64() < bc.currentBlock.NumberU64() {
		bc.currentBlock = bc.GetBlock(currentHeader.Hash(), currentHeader.Number.Uint64())
	}
	if bc.currentBlock != nil {
		if _, err := state.New(bc.currentBlock.Root(), bc.chainDb); err != nil {
			// Rewound state missing, rolled back to before pivot, reset to genesis
			bc.currentBlock = nil
		}
	}
	// Rewind the fast block in a simpleton way to the target head
	if bc.currentFastBlock != nil && currentHeader.Number.Uint64() < bc.currentFastBlock.NumberU64() {
		bc.currentFastBlock = bc.GetBlock(currentHeader.Hash(), currentHeader.Number.Uint64())
	}
	// If either blocks reached nil, reset to the genesis state
	if bc.currentBlock == nil {
		bc.currentBlock = bc.genesisBlock
	}
	if bc.currentFastBlock == nil {
		bc.currentFastBlock = bc.genesisBlock
	}
	if err := WriteHeadBlockHash(bc.chainDb, bc.currentBlock.Hash()); err != nil {
		log.Crit("Failed to reset head full block", "err", err)
	}
	if err := WriteHeadFastBlockHash(bc.chainDb, bc.currentFastBlock.Hash()); err != nil {
		log.Crit("Failed to reset head fast block", "err", err)
	}
	return bc.loadLastState()
}

// FastSyncCommitHead sets the current head block to the one defined by the hash
// irrelevant what the chain contents were prior.
func (self *BlockChain) FastSyncCommitHead(hash common.Hash) error {
	// Make sure that both the block as well at its state trie exists
	block := self.GetBlockByHash(hash)
	if block == nil {
		return fmt.Errorf("non existent block [%x…]", hash[:4])
	}
	if _, err := trie.NewSecure(block.Root(), self.chainDb, 0); err != nil {
		return err
	}
	// If all checks out, manually set the head block
	self.mu.Lock()
	self.currentBlock = block
	self.mu.Unlock()

	log.Info("Committed new head block", "number", block.Number(), "hash", hash)
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

	return self.GetTd(self.currentBlock.Hash(), self.currentBlock.NumberU64()), self.currentBlock.Hash(), self.genesisBlock.Hash()
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

// State returns a new mutable state based on the current HEAD block.
func (self *BlockChain) State() (*state.StateDB, error) {
	return self.StateAt(self.CurrentBlock().Root())
}

// StateAt returns a new mutable state based on a particular point in time.
func (self *BlockChain) StateAt(root common.Hash) (*state.StateDB, error) {
	return self.stateCache.New(root)
}

// Reset purges the entire blockchain, restoring it to its genesis state.
func (bc *BlockChain) Reset() error {
	return bc.ResetWithGenesisBlock(bc.genesisBlock)
}

// ResetWithGenesisBlock purges the entire blockchain, restoring it to the
// specified genesis state.
func (bc *BlockChain) ResetWithGenesisBlock(genesis *types.Block) error {
	// Dump the entire block chain and purge the caches
	if err := bc.SetHead(0); err != nil {
		return err
	}
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Prepare the genesis block and reinitialise the chain
	if err := bc.hc.WriteTd(genesis.Hash(), genesis.NumberU64(), genesis.Difficulty()); err != nil {
		log.Crit("Failed to write genesis block TD", "err", err)
	}
	if err := WriteBlock(bc.chainDb, genesis); err != nil {
		log.Crit("Failed to write genesis block", "err", err)
	}
	bc.genesisBlock = genesis
	bc.insert(bc.genesisBlock)
	bc.currentBlock = bc.genesisBlock
	bc.hc.SetGenesis(bc.genesisBlock.Header())
	bc.hc.SetCurrentHeader(bc.genesisBlock.Header())
	bc.currentFastBlock = bc.genesisBlock

	return nil
}

// Export writes the active chain to the given writer.
func (self *BlockChain) Export(w io.Writer) error {
	return self.ExportN(w, uint64(0), self.currentBlock.NumberU64())
}

// ExportN writes a subset of the active chain to the given writer.
func (self *BlockChain) ExportN(w io.Writer, first uint64, last uint64) error {
	self.mu.RLock()
	defer self.mu.RUnlock()

	if first > last {
		return fmt.Errorf("export failed: first (%d) is greater than last (%d)", first, last)
	}
	log.Info("Exporting batch of blocks", "count", last-first+1)

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
		log.Crit("Failed to insert block number", "err", err)
	}
	if err := WriteHeadBlockHash(bc.chainDb, block.Hash()); err != nil {
		log.Crit("Failed to insert head block hash", "err", err)
	}
	bc.currentBlock = block

	// If the block is better than out head or is on a different chain, force update heads
	if updateHeads {
		bc.hc.SetCurrentHeader(block.Header())

		if err := WriteHeadFastBlockHash(bc.chainDb, block.Hash()); err != nil {
			log.Crit("Failed to insert head fast block hash", "err", err)
		}
		bc.currentFastBlock = block
	}
}

// Accessors
func (bc *BlockChain) Genesis() *types.Block {
	return bc.genesisBlock
}

// GetBody retrieves a block body (transactions and uncles) from the database by
// hash, caching it if found.
func (self *BlockChain) GetBody(hash common.Hash) *types.Body {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := self.bodyCache.Get(hash); ok {
		body := cached.(*types.Body)
		return body
	}
	body := GetBody(self.chainDb, hash, self.hc.GetBlockNumber(hash))
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
	body := GetBodyRLP(self.chainDb, hash, self.hc.GetBlockNumber(hash))
	if len(body) == 0 {
		return nil
	}
	// Cache the found body for next time and return
	self.bodyRLPCache.Add(hash, body)
	return body
}

// HasBlock checks if a block is fully present in the database or not, caching
// it if present.
func (bc *BlockChain) HasBlock(hash common.Hash) bool {
	return bc.GetBlockByHash(hash) != nil
}

// HasBlockAndState checks if a block and associated state trie is fully present
// in the database or not, caching it if present.
func (bc *BlockChain) HasBlockAndState(hash common.Hash) bool {
	// Check first that the block itself is known
	block := bc.GetBlockByHash(hash)
	if block == nil {
		return false
	}
	// Ensure the associated state is also present
	_, err := state.New(block.Root(), bc.chainDb)
	return err == nil
}

// GetBlock retrieves a block from the database by hash and number,
// caching it if found.
func (self *BlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := self.blockCache.Get(hash); ok {
		return block.(*types.Block)
	}
	block := GetBlock(self.chainDb, hash, number)
	if block == nil {
		return nil
	}
	// Cache the found block for next time and return
	self.blockCache.Add(block.Hash(), block)
	return block
}

// GetBlockByHash retrieves a block from the database by hash, caching it if found.
func (self *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	return self.GetBlock(hash, self.hc.GetBlockNumber(hash))
}

// GetBlockByNumber retrieves a block from the database by number, caching it
// (associated with its hash) if found.
func (self *BlockChain) GetBlockByNumber(number uint64) *types.Block {
	hash := GetCanonicalHash(self.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return self.GetBlock(hash, number)
}

// [deprecated by eth/62]
// GetBlocksFromHash returns the block corresponding to hash and up to n-1 ancestors.
func (self *BlockChain) GetBlocksFromHash(hash common.Hash, n int) (blocks []*types.Block) {
	number := self.hc.GetBlockNumber(hash)
	for i := 0; i < n; i++ {
		block := self.GetBlock(hash, number)
		if block == nil {
			break
		}
		blocks = append(blocks, block)
		hash = block.ParentHash()
		number--
	}
	return
}

// GetUnclesInChain retrieves all the uncles from a given block backwards until
// a specific distance is reached.
func (self *BlockChain) GetUnclesInChain(block *types.Block, length int) []*types.Header {
	uncles := []*types.Header{}
	for i := 0; block != nil && i < length; i++ {
		uncles = append(uncles, block.Uncles()...)
		block = self.GetBlock(block.ParentHash(), block.NumberU64()-1)
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
	log.Info("Blockchain manager stopped")
}

func (self *BlockChain) procFutureBlocks() {
	blocks := make([]*types.Block, 0, self.futureBlocks.Len())
	for _, hash := range self.futureBlocks.Keys() {
		if block, exist := self.futureBlocks.Peek(hash); exist {
			blocks = append(blocks, block.(*types.Block))
		}
	}
	if len(blocks) > 0 {
		types.BlockBy(types.Number).Sort(blocks)

		// Insert one by one as chain insertion needs contiguous ancestry between blocks
		for i := range blocks {
			self.InsertChain(blocks[i : i+1])
		}
	}
}

type WriteStatus byte

const (
	NonStatTy WriteStatus = iota
	CanonStatTy
	SplitStatTy
	SideStatTy
)

// Rollback is designed to remove a chain of links from the database that aren't
// certain enough to be valid.
func (self *BlockChain) Rollback(chain []common.Hash) {
	self.mu.Lock()
	defer self.mu.Unlock()

	for i := len(chain) - 1; i >= 0; i-- {
		hash := chain[i]

		currentHeader := self.hc.CurrentHeader()
		if currentHeader.Hash() == hash {
			self.hc.SetCurrentHeader(self.GetHeader(currentHeader.ParentHash, currentHeader.Number.Uint64()-1))
		}
		if self.currentFastBlock.Hash() == hash {
			self.currentFastBlock = self.GetBlock(self.currentFastBlock.ParentHash(), self.currentFastBlock.NumberU64()-1)
			WriteHeadFastBlockHash(self.chainDb, self.currentFastBlock.Hash())
		}
		if self.currentBlock.Hash() == hash {
			self.currentBlock = self.GetBlock(self.currentBlock.ParentHash(), self.currentBlock.NumberU64()-1)
			WriteHeadBlockHash(self.chainDb, self.currentBlock.Hash())
		}
	}
}

// SetReceiptsData computes all the non-consensus fields of the receipts
func SetReceiptsData(config *params.ChainConfig, block *types.Block, receipts types.Receipts) {
	signer := types.MakeSigner(config, block.Number())

	transactions, logIndex := block.Transactions(), uint(0)

	for j := 0; j < len(receipts); j++ {
		// The transaction hash can be retrieved from the transaction itself
		receipts[j].TxHash = transactions[j].Hash()

		tx, _ := transactions[j].AsMessage(signer)
		// The contract address can be derived from the transaction itself
		if MessageCreatesContract(tx) {
			receipts[j].ContractAddress = crypto.CreateAddress(tx.From(), tx.Nonce())
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
}

// InsertReceiptChain attempts to complete an already existing header chain with
// transaction and receipt data.
// XXX should this be moved to the test?
func (self *BlockChain) InsertReceiptChain(blockChain types.Blocks, receiptChain []types.Receipts) (int, error) {
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(blockChain); i++ {
		if blockChain[i].NumberU64() != blockChain[i-1].NumberU64()+1 || blockChain[i].ParentHash() != blockChain[i-1].Hash() {
			// Chain broke ancestry, log a messge (programming error) and skip insertion
			log.Error("Non contiguous receipt insert", "number", blockChain[i].Number(), "hash", blockChain[i].Hash(), "parent", blockChain[i].ParentHash(),
				"prevnumber", blockChain[i-1].Number(), "prevhash", blockChain[i-1].Hash())

			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x…], item %d is #%d [%x…] (parent [%x…])", i-1, blockChain[i-1].NumberU64(),
				blockChain[i-1].Hash().Bytes()[:4], i, blockChain[i].NumberU64(), blockChain[i].Hash().Bytes()[:4], blockChain[i].ParentHash().Bytes()[:4])
		}
	}
	// Pre-checks passed, start the block body and receipt imports
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
			SetReceiptsData(self.config, block, receipts)
			// Write all the data out into the database
			if err := WriteBody(self.chainDb, block.Hash(), block.NumberU64(), block.Body()); err != nil {
				errs[index] = fmt.Errorf("failed to write block body: %v", err)
				atomic.AddInt32(&failed, 1)
				log.Crit("Failed to write block body", "err", err)
				return
			}
			if err := WriteBlockReceipts(self.chainDb, block.Hash(), block.NumberU64(), receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write block receipts: %v", err)
				atomic.AddInt32(&failed, 1)
				log.Crit("Failed to write block receipts", "err", err)
				return
			}
			if err := WriteMipmapBloom(self.chainDb, block.NumberU64(), receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write log blooms: %v", err)
				atomic.AddInt32(&failed, 1)
				log.Crit("Failed to write log blooms", "err", err)
				return
			}
			if err := WriteTransactions(self.chainDb, block); err != nil {
				errs[index] = fmt.Errorf("failed to write individual transactions: %v", err)
				atomic.AddInt32(&failed, 1)
				log.Crit("Failed to write individual transactions", "err", err)
				return
			}
			if err := WriteReceipts(self.chainDb, receipts); err != nil {
				errs[index] = fmt.Errorf("failed to write individual receipts: %v", err)
				atomic.AddInt32(&failed, 1)
				log.Crit("Failed to write individual receipts", "err", err)
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
		log.Debug("Premature abort during receipts processing")
		return 0, nil
	}
	// Update the head fast sync block if better
	self.mu.Lock()

	head := blockChain[len(errs)-1]
	if td := self.GetTd(head.Hash(), head.NumberU64()); td != nil { // Rewind may have occurred, skip in that case
		if self.GetTd(self.currentFastBlock.Hash(), self.currentFastBlock.NumberU64()).Cmp(td) < 0 {
			if err := WriteHeadFastBlockHash(self.chainDb, head.Hash()); err != nil {
				log.Crit("Failed to update head fast block hash", "err", err)
			}
			self.currentFastBlock = head
		}
	}
	self.mu.Unlock()

	// Report some public statistics so the user has a clue what's going on
	last := blockChain[len(blockChain)-1]
	log.Info("Imported new block receipts", "count", stats.processed, "elapsed", common.PrettyDuration(time.Since(start)),
		"number", last.Number(), "hash", last.Hash(), "ignored", stats.ignored)

	return 0, nil
}

// WriteBlock writes the block to the chain.
func (self *BlockChain) WriteBlock(block *types.Block) (status WriteStatus, err error) {
	self.wg.Add(1)
	defer self.wg.Done()

	// Calculate the total difficulty of the block
	ptd := self.GetTd(block.ParentHash(), block.NumberU64()-1)
	if ptd == nil {
		return NonStatTy, consensus.ErrUnknownAncestor
	}
	// Make sure no inconsistent state is leaked during insertion
	self.mu.Lock()
	defer self.mu.Unlock()

	localTd := self.GetTd(self.currentBlock.Hash(), self.currentBlock.NumberU64())
	externTd := new(big.Int).Add(block.Difficulty(), ptd)

	// Irrelevant of the canonical status, write the block itself to the database
	if err := self.hc.WriteTd(block.Hash(), block.NumberU64(), externTd); err != nil {
		log.Crit("Failed to write block total difficulty", "err", err)
	}
	if err := WriteBlock(self.chainDb, block); err != nil {
		log.Crit("Failed to write block contents", "err", err)
	}

	// If the total difficulty is higher than our known, add it to the canonical chain
	// Second clause in the if statement reduces the vulnerability to selfish mining.
	// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
	if externTd.Cmp(localTd) > 0 || (externTd.Cmp(localTd) == 0 && mrand.Float64() < 0.5) {
		// Reorganise the chain if the parent is not the head block
		if block.ParentHash() != self.currentBlock.Hash() {
			if err := self.reorg(self.currentBlock, block); err != nil {
				return NonStatTy, err
			}
		}
		self.insert(block) // Insert the block as the new head of the chain
		status = CanonStatTy
	} else {
		status = SideStatTy
	}

	self.futureBlocks.Remove(block.Hash())

	return
}

// InsertChain will attempt to insert the given chain in to the canonical chain or, otherwise, create a fork. If an error is returned
// it will return the index number of the failing block as well an error describing what went wrong (for possible errors see core/errors.go).
func (self *BlockChain) InsertChain(chain types.Blocks) (int, error) {
	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		if chain[i].NumberU64() != chain[i-1].NumberU64()+1 || chain[i].ParentHash() != chain[i-1].Hash() {
			// Chain broke ancestry, log a messge (programming error) and skip insertion
			log.Error("Non contiguous block insert", "number", chain[i].Number(), "hash", chain[i].Hash(),
				"parent", chain[i].ParentHash(), "prevnumber", chain[i-1].Number(), "prevhash", chain[i-1].Hash())

			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x…], item %d is #%d [%x…] (parent [%x…])", i-1, chain[i-1].NumberU64(),
				chain[i-1].Hash().Bytes()[:4], i, chain[i].NumberU64(), chain[i].Hash().Bytes()[:4], chain[i].ParentHash().Bytes()[:4])
		}
	}
	// Pre-checks passed, start the full block imports
	self.wg.Add(1)
	defer self.wg.Done()

	self.chainmu.Lock()
	defer self.chainmu.Unlock()

	// A queued approach to delivering events. This is generally
	// faster than direct delivery and requires much less mutex
	// acquiring.
	var (
		stats         = insertStats{startTime: mclock.Now()}
		events        = make([]interface{}, 0, len(chain))
		coalescedLogs []*types.Log
	)
	// Start the parallel header verifier
	headers := make([]*types.Header, len(chain))
	seals := make([]bool, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
		seals[i] = true
	}
	abort, results := self.engine.VerifyHeaders(self, headers, seals)
	defer close(abort)

	// Iterate over the blocks and insert when the verifier permits
	for i, block := range chain {
		// If the chain is terminating, stop processing blocks
		if atomic.LoadInt32(&self.procInterrupt) == 1 {
			log.Debug("Premature abort during blocks processing")
			break
		}
		// If the header is a banned one, straight out abort
		if BadHashes[block.Hash()] {
			self.reportBlock(block, nil, ErrBlacklistedHash)
			return i, ErrBlacklistedHash
		}
		// Wait for the block's verification to complete
		bstart := time.Now()

		err := <-results
		if err == nil {
			err = self.Validator().ValidateBody(block)
		}
		if err != nil {
			if err == ErrKnownBlock {
				stats.ignored++
				continue
			}

			if err == consensus.ErrFutureBlock {
				// Allow up to MaxFuture second in the future blocks. If this limit
				// is exceeded the chain is discarded and processed at a later time
				// if given.
				max := big.NewInt(time.Now().Unix() + maxTimeFutureBlocks)
				if block.Time().Cmp(max) > 0 {
					return i, fmt.Errorf("future block: %v > %v", block.Time(), max)
				}
				self.futureBlocks.Add(block.Hash(), block)
				stats.queued++
				continue
			}

			if err == consensus.ErrUnknownAncestor && self.futureBlocks.Contains(block.ParentHash()) {
				self.futureBlocks.Add(block.Hash(), block)
				stats.queued++
				continue
			}

			self.reportBlock(block, nil, err)
			return i, err
		}
		// Create a new statedb using the parent block and report an
		// error if it fails.
		switch {
		case i == 0:
			err = self.stateCache.Reset(self.GetBlock(block.ParentHash(), block.NumberU64()-1).Root())
		default:
			err = self.stateCache.Reset(chain[i-1].Root())
		}
		if err != nil {
			self.reportBlock(block, nil, err)
			return i, err
		}
		// Process block using the parent state as reference point.
		receipts, logs, usedGas, err := self.processor.Process(block, self.stateCache, self.vmConfig)
		if err != nil {
			self.reportBlock(block, receipts, err)
			return i, err
		}
		// Validate the state using the default validator
		err = self.Validator().ValidateState(block, self.GetBlock(block.ParentHash(), block.NumberU64()-1), self.stateCache, receipts, usedGas)
		if err != nil {
			self.reportBlock(block, receipts, err)
			return i, err
		}
		// Write state changes to database
		_, err = self.stateCache.Commit(self.config.IsEIP158(block.Number()))
		if err != nil {
			return i, err
		}

		// coalesce logs for later processing
		coalescedLogs = append(coalescedLogs, logs...)

		if err = WriteBlockReceipts(self.chainDb, block.Hash(), block.NumberU64(), receipts); err != nil {
			return i, err
		}

		// write the block to the chain and get the status
		status, err := self.WriteBlock(block)
		if err != nil {
			return i, err
		}

		switch status {
		case CanonStatTy:
			log.Debug("Inserted new block", "number", block.Number(), "hash", block.Hash(), "uncles", len(block.Uncles()),
				"txs", len(block.Transactions()), "gas", block.GasUsed(), "elapsed", common.PrettyDuration(time.Since(bstart)))

			blockInsertTimer.UpdateSince(bstart)
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
			// Write hash preimages
			if err := WritePreimages(self.chainDb, block.NumberU64(), self.stateCache.Preimages()); err != nil {
				return i, err
			}
		case SideStatTy:
			log.Debug("Inserted forked block", "number", block.Number(), "hash", block.Hash(), "diff", block.Difficulty(), "elapsed",
				common.PrettyDuration(time.Since(bstart)), "txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()))

			blockInsertTimer.UpdateSince(bstart)
			events = append(events, ChainSideEvent{block})

		case SplitStatTy:
			events = append(events, ChainSplitEvent{block, logs})
		}
		stats.processed++
		stats.usedGas += usedGas.Uint64()
		stats.report(chain, i)
	}
	go self.postChainEvents(events, coalescedLogs)

	return 0, nil
}

// insertStats tracks and reports on block insertion.
type insertStats struct {
	queued, processed, ignored int
	usedGas                    uint64
	lastIndex                  int
	startTime                  mclock.AbsTime
}

// statsReportLimit is the time limit during import after which we always print
// out progress. This avoids the user wondering what's going on.
const statsReportLimit = 8 * time.Second

// report prints statistics if some number of blocks have been processed
// or more than a few seconds have passed since the last message.
func (st *insertStats) report(chain []*types.Block, index int) {
	// Fetch the timings for the batch
	var (
		now     = mclock.Now()
		elapsed = time.Duration(now) - time.Duration(st.startTime)
	)
	// If we're at the last block of the batch or report period reached, log
	if index == len(chain)-1 || elapsed >= statsReportLimit {
		var (
			end = chain[index]
			txs = countTransactions(chain[st.lastIndex : index+1])
		)
		context := []interface{}{
			"blocks", st.processed, "txs", txs, "mgas", float64(st.usedGas) / 1000000,
			"elapsed", common.PrettyDuration(elapsed), "mgasps", float64(st.usedGas) * 1000 / float64(elapsed),
			"number", end.Number(), "hash", end.Hash(),
		}
		if st.queued > 0 {
			context = append(context, []interface{}{"queued", st.queued}...)
		}
		if st.ignored > 0 {
			context = append(context, []interface{}{"ignored", st.ignored}...)
		}
		log.Info("Imported new chain segment", context...)

		*st = insertStats{startTime: now, lastIndex: index}
	}
}

func countTransactions(chain []*types.Block) (c int) {
	for _, b := range chain {
		c += len(b.Transactions())
	}
	return c
}

// reorgs takes two blocks, an old chain and a new chain and will reconstruct the blocks and inserts them
// to be part of the new canonical chain and accumulates potential missing transactions and post an
// event about them
func (self *BlockChain) reorg(oldBlock, newBlock *types.Block) error {
	var (
		newChain    types.Blocks
		oldChain    types.Blocks
		commonBlock *types.Block
		deletedTxs  types.Transactions
		deletedLogs []*types.Log
		// collectLogs collects the logs that were generated during the
		// processing of the block that corresponds with the given hash.
		// These logs are later announced as deleted.
		collectLogs = func(h common.Hash) {
			// Coalesce logs and set 'Removed'.
			receipts := GetBlockReceipts(self.chainDb, h, self.hc.GetBlockNumber(h))
			for _, receipt := range receipts {
				for _, log := range receipt.Logs {
					del := *log
					del.Removed = true
					deletedLogs = append(deletedLogs, &del)
				}
			}
		}
	)

	// first reduce whoever is higher bound
	if oldBlock.NumberU64() > newBlock.NumberU64() {
		// reduce old chain
		for ; oldBlock != nil && oldBlock.NumberU64() != newBlock.NumberU64(); oldBlock = self.GetBlock(oldBlock.ParentHash(), oldBlock.NumberU64()-1) {
			oldChain = append(oldChain, oldBlock)
			deletedTxs = append(deletedTxs, oldBlock.Transactions()...)

			collectLogs(oldBlock.Hash())
		}
	} else {
		// reduce new chain and append new chain blocks for inserting later on
		for ; newBlock != nil && newBlock.NumberU64() != oldBlock.NumberU64(); newBlock = self.GetBlock(newBlock.ParentHash(), newBlock.NumberU64()-1) {
			newChain = append(newChain, newBlock)
		}
	}
	if oldBlock == nil {
		return fmt.Errorf("Invalid old chain")
	}
	if newBlock == nil {
		return fmt.Errorf("Invalid new chain")
	}

	for {
		if oldBlock.Hash() == newBlock.Hash() {
			commonBlock = oldBlock
			break
		}

		oldChain = append(oldChain, oldBlock)
		newChain = append(newChain, newBlock)
		deletedTxs = append(deletedTxs, oldBlock.Transactions()...)
		collectLogs(oldBlock.Hash())

		oldBlock, newBlock = self.GetBlock(oldBlock.ParentHash(), oldBlock.NumberU64()-1), self.GetBlock(newBlock.ParentHash(), newBlock.NumberU64()-1)
		if oldBlock == nil {
			return fmt.Errorf("Invalid old chain")
		}
		if newBlock == nil {
			return fmt.Errorf("Invalid new chain")
		}
	}
	// Ensure the user sees large reorgs
	if len(oldChain) > 0 && len(newChain) > 0 {
		logFn := log.Debug
		if len(oldChain) > 63 {
			logFn = log.Warn
		}
		logFn("Chain split detected", "number", commonBlock.Number(), "hash", commonBlock.Hash(),
			"drop", len(oldChain), "dropfrom", oldChain[0].Hash(), "add", len(newChain), "addfrom", newChain[0].Hash())
	} else {
		log.Error("Impossible reorg, please file an issue", "oldnum", oldBlock.Number(), "oldhash", oldBlock.Hash(), "newnum", newBlock.Number(), "newhash", newBlock.Hash())
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
		receipts := GetBlockReceipts(self.chainDb, block.Hash(), block.NumberU64())
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
	if len(diff) > 0 {
		go self.eventMux.Post(RemovedTransactionEvent{diff})
	}
	if len(deletedLogs) > 0 {
		go self.eventMux.Post(RemovedLogsEvent{deletedLogs})
	}

	if len(oldChain) > 0 {
		go func() {
			for _, block := range oldChain {
				self.eventMux.Post(ChainSideEvent{Block: block})
			}
		}()
	}

	return nil
}

// postChainEvents iterates over the events generated by a chain insertion and
// posts them into the event mux.
func (self *BlockChain) postChainEvents(events []interface{}, logs []*types.Log) {
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

// BadBlockArgs represents the entries in the list returned when bad blocks are queried.
type BadBlockArgs struct {
	Hash   common.Hash   `json:"hash"`
	Header *types.Header `json:"header"`
}

// BadBlocks returns a list of the last 'bad blocks' that the client has seen on the network
func (bc *BlockChain) BadBlocks() ([]BadBlockArgs, error) {
	headers := make([]BadBlockArgs, 0, bc.badBlocks.Len())
	for _, hash := range bc.badBlocks.Keys() {
		if hdr, exist := bc.badBlocks.Peek(hash); exist {
			header := hdr.(*types.Header)
			headers = append(headers, BadBlockArgs{header.Hash(), header})
		}
	}
	return headers, nil
}

// addBadBlock adds a bad block to the bad-block LRU cache
func (bc *BlockChain) addBadBlock(block *types.Block) {
	bc.badBlocks.Add(block.Header().Hash(), block.Header())
}

// reportBlock logs a bad block error.
func (bc *BlockChain) reportBlock(block *types.Block, receipts types.Receipts, err error) {
	bc.addBadBlock(block)

	var receiptString string
	for _, receipt := range receipts {
		receiptString += fmt.Sprintf("\t%v\n", receipt)
	}
	log.Error(fmt.Sprintf(`
########## BAD BLOCK #########
Chain config: %v

Number: %v
Hash: 0x%x
%v

Error: %v
##############################
`, bc.config, block.Number(), block.Hash(), receiptString, err))
}

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify parameter can be used to fine tune whether nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verify nonces, as well as
// because nonces can be verified sparsely, not needing to check each.
func (self *BlockChain) InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error) {
	start := time.Now()
	if i, err := self.hc.ValidateHeaderChain(chain, checkFreq); err != nil {
		return i, err
	}

	// Make sure only one thread manipulates the chain at once
	self.chainmu.Lock()
	defer self.chainmu.Unlock()

	self.wg.Add(1)
	defer self.wg.Done()

	whFunc := func(header *types.Header) error {
		self.mu.Lock()
		defer self.mu.Unlock()

		_, err := self.hc.WriteHeader(header)
		return err
	}

	return self.hc.InsertHeaderChain(chain, whFunc, start)
}

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

	self.mu.Lock()
	defer self.mu.Unlock()

	_, err := self.hc.WriteHeader(header)
	return err
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (self *BlockChain) CurrentHeader() *types.Header {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.hc.CurrentHeader()
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash and number, caching it if found.
func (self *BlockChain) GetTd(hash common.Hash, number uint64) *big.Int {
	return self.hc.GetTd(hash, number)
}

// GetTdByHash retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (self *BlockChain) GetTdByHash(hash common.Hash) *big.Int {
	return self.hc.GetTdByHash(hash)
}

// GetHeader retrieves a block header from the database by hash and number,
// caching it if found.
func (self *BlockChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	return self.hc.GetHeader(hash, number)
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (self *BlockChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return self.hc.GetHeaderByHash(hash)
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (bc *BlockChain) HasHeader(hash common.Hash) bool {
	return bc.hc.HasHeader(hash)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (self *BlockChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	return self.hc.GetBlockHashesFromHash(hash, max)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (self *BlockChain) GetHeaderByNumber(number uint64) *types.Header {
	return self.hc.GetHeaderByNumber(number)
}

// Config retrieves the blockchain's chain configuration.
func (self *BlockChain) Config() *params.ChainConfig { return self.config }
