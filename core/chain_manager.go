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
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/rlp"
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
	checkpointLimit     = 200
)

type ChainManager struct {
	//eth          EthManager
	chainDb      common.Database
	processor    types.BlockProcessor
	eventMux     *event.TypeMux
	genesisBlock *types.Block
	// Last known total difficulty
	mu      sync.RWMutex
	chainmu sync.RWMutex
	tsmu    sync.RWMutex

	checkpoint      int // checkpoint counts towards the new checkpoint
	td              *big.Int
	currentBlock    *types.Block
	currentGasLimit *big.Int

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

	pow pow.PoW
}

func NewChainManager(chainDb common.Database, pow pow.PoW, mux *event.TypeMux) (*ChainManager, error) {
	headerCache, _ := lru.New(headerCacheLimit)
	bodyCache, _ := lru.New(bodyCacheLimit)
	bodyRLPCache, _ := lru.New(bodyCacheLimit)
	tdCache, _ := lru.New(tdCacheLimit)
	blockCache, _ := lru.New(blockCacheLimit)
	futureBlocks, _ := lru.New(maxFutureBlocks)

	bc := &ChainManager{
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
	if err := bc.setLastState(); err != nil {
		return nil, err
	}
	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for hash, _ := range BadHashes {
		if block := bc.GetBlock(hash); block != nil {
			glog.V(logger.Error).Infof("Found bad hash. Reorganising chain to state %x\n", block.ParentHash().Bytes()[:4])
			block = bc.GetBlock(block.ParentHash())
			if block == nil {
				glog.Fatal("Unable to complete. Parent block not found. Corrupted DB?")
			}
			bc.SetHead(block)

			glog.V(logger.Error).Infoln("Chain reorg was successfull. Resuming normal operation")
		}
	}
	// Take ownership of this particular state
	go bc.update()
	return bc, nil
}

func (bc *ChainManager) SetHead(head *types.Block) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for block := bc.currentBlock; block != nil && block.Hash() != head.Hash(); block = bc.GetBlock(block.ParentHash()) {
		DeleteBlock(bc.chainDb, block.Hash())
	}
	bc.headerCache.Purge()
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.blockCache.Purge()
	bc.futureBlocks.Purge()

	bc.currentBlock = head
	bc.setTotalDifficulty(bc.GetTd(head.Hash()))
	bc.insert(head)
	bc.setLastState()
}

func (self *ChainManager) Td() *big.Int {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return new(big.Int).Set(self.td)
}

func (self *ChainManager) GasLimit() *big.Int {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentBlock.GasLimit()
}

func (self *ChainManager) LastBlockHash() common.Hash {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentBlock.Hash()
}

func (self *ChainManager) CurrentBlock() *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentBlock
}

func (self *ChainManager) Status() (td *big.Int, currentBlock common.Hash, genesisBlock common.Hash) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return new(big.Int).Set(self.td), self.currentBlock.Hash(), self.genesisBlock.Hash()
}

func (self *ChainManager) SetProcessor(proc types.BlockProcessor) {
	self.processor = proc
}

func (self *ChainManager) State() *state.StateDB {
	return state.New(self.CurrentBlock().Root(), self.chainDb)
}

func (bc *ChainManager) recover() bool {
	data, _ := bc.chainDb.Get([]byte("checkpoint"))
	if len(data) != 0 {
		block := bc.GetBlock(common.BytesToHash(data))
		if block != nil {
			if err := WriteCanonicalHash(bc.chainDb, block.Hash(), block.NumberU64()); err != nil {
				glog.Fatalf("failed to write database head number: %v", err)
			}
			if err := WriteHeadBlockHash(bc.chainDb, block.Hash()); err != nil {
				glog.Fatalf("failed to write database head hash: %v", err)
			}
			bc.currentBlock = block
			return true
		}
	}
	return false
}

func (bc *ChainManager) setLastState() error {
	head := GetHeadBlockHash(bc.chainDb)
	if head != (common.Hash{}) {
		block := bc.GetBlock(head)
		if block != nil {
			bc.currentBlock = block
		} else {
			glog.Infof("LastBlock (%x) not found. Recovering...\n", head)
			if bc.recover() {
				glog.Infof("Recover successful")
			} else {
				glog.Fatalf("Recover failed. Please report")
			}
		}
	} else {
		bc.Reset()
	}
	bc.td = bc.GetTd(bc.currentBlock.Hash())
	bc.currentGasLimit = CalcGasLimit(bc.currentBlock)

	if glog.V(logger.Info) {
		glog.Infof("Last block (#%v) %x TD=%v\n", bc.currentBlock.Number(), bc.currentBlock.Hash(), bc.td)
	}

	return nil
}

// Reset purges the entire blockchain, restoring it to its genesis state.
func (bc *ChainManager) Reset() {
	bc.ResetWithGenesisBlock(bc.genesisBlock)
}

// ResetWithGenesisBlock purges the entire blockchain, restoring it to the
// specified genesis state.
func (bc *ChainManager) ResetWithGenesisBlock(genesis *types.Block) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	// Dump the entire block chain and purge the caches
	for block := bc.currentBlock; block != nil; block = bc.GetBlock(block.ParentHash()) {
		DeleteBlock(bc.chainDb, block.Hash())
	}
	bc.headerCache.Purge()
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.blockCache.Purge()
	bc.futureBlocks.Purge()

	// Prepare the genesis block and reinitialize the chain
	if err := WriteTd(bc.chainDb, genesis.Hash(), genesis.Difficulty()); err != nil {
		glog.Fatalf("failed to write genesis block TD: %v", err)
	}
	if err := WriteBlock(bc.chainDb, genesis); err != nil {
		glog.Fatalf("failed to write genesis block: %v", err)
	}
	bc.insert(bc.genesisBlock)
	bc.currentBlock = bc.genesisBlock
	bc.setTotalDifficulty(genesis.Difficulty())
}

// Export writes the active chain to the given writer.
func (self *ChainManager) Export(w io.Writer) error {
	if err := self.ExportN(w, uint64(0), self.currentBlock.NumberU64()); err != nil {
		return err
	}
	return nil
}

// ExportN writes a subset of the active chain to the given writer.
func (self *ChainManager) ExportN(w io.Writer, first uint64, last uint64) error {
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

// insert injects a block into the current chain block chain. Note, this function
// assumes that the `mu` mutex is held!
func (bc *ChainManager) insert(block *types.Block) {
	// Add the block to the canonical chain number scheme and mark as the head
	if err := WriteCanonicalHash(bc.chainDb, block.Hash(), block.NumberU64()); err != nil {
		glog.Fatalf("failed to insert block number: %v", err)
	}
	if err := WriteHeadBlockHash(bc.chainDb, block.Hash()); err != nil {
		glog.Fatalf("failed to insert block number: %v", err)
	}
	// Add a new restore point if we reached some limit
	bc.checkpoint++
	if bc.checkpoint > checkpointLimit {
		if err := bc.chainDb.Put([]byte("checkpoint"), block.Hash().Bytes()); err != nil {
			glog.Fatalf("failed to create checkpoint: %v", err)
		}
		bc.checkpoint = 0
	}
	// Update the internal internal state with the head block
	bc.currentBlock = block
}

// Accessors
func (bc *ChainManager) Genesis() *types.Block {
	return bc.genesisBlock
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (bc *ChainManager) HasHeader(hash common.Hash) bool {
	return bc.GetHeader(hash) != nil
}

// GetHeader retrieves a block header from the database by hash, caching it if
// found.
func (self *ChainManager) GetHeader(hash common.Hash) *types.Header {
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
func (self *ChainManager) GetHeaderByNumber(number uint64) *types.Header {
	hash := GetCanonicalHash(self.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return self.GetHeader(hash)
}

// GetBody retrieves a block body (transactions and uncles) from the database by
// hash, caching it if found.
func (self *ChainManager) GetBody(hash common.Hash) *types.Body {
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
func (self *ChainManager) GetBodyRLP(hash common.Hash) rlp.RawValue {
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
func (self *ChainManager) GetTd(hash common.Hash) *big.Int {
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
func (bc *ChainManager) HasBlock(hash common.Hash) bool {
	return bc.GetBlock(hash) != nil
}

// GetBlock retrieves a block from the database by hash, caching it if found.
func (self *ChainManager) GetBlock(hash common.Hash) *types.Block {
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
func (self *ChainManager) GetBlockByNumber(number uint64) *types.Block {
	hash := GetCanonicalHash(self.chainDb, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return self.GetBlock(hash)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (self *ChainManager) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
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
func (self *ChainManager) GetBlocksFromHash(hash common.Hash, n int) (blocks []*types.Block) {
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

func (self *ChainManager) GetUnclesInChain(block *types.Block, length int) (uncles []*types.Header) {
	for i := 0; block != nil && i < length; i++ {
		uncles = append(uncles, block.Uncles()...)
		block = self.GetBlock(block.ParentHash())
	}

	return
}

// setTotalDifficulty updates the TD of the chain manager. Note, this function
// assumes that the `mu` mutex is held!
func (bc *ChainManager) setTotalDifficulty(td *big.Int) {
	bc.td = new(big.Int).Set(td)
}

func (bc *ChainManager) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}
	close(bc.quit)
	atomic.StoreInt32(&bc.procInterrupt, 1)

	bc.wg.Wait()

	glog.V(logger.Info).Infoln("Chain manager stopped")
}

type queueEvent struct {
	queue          []interface{}
	canonicalCount int
	sideCount      int
	splitCount     int
}

func (self *ChainManager) procFutureBlocks() {
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

// WriteBlock writes the block to the chain.
func (self *ChainManager) WriteBlock(block *types.Block) (status writeStatus, err error) {
	self.wg.Add(1)
	defer self.wg.Done()

	// Calculate the total difficulty of the block
	ptd := self.GetTd(block.ParentHash())
	if ptd == nil {
		return NonStatTy, ParentError(block.ParentHash())
	}
	td := new(big.Int).Add(block.Difficulty(), ptd)

	self.mu.RLock()
	cblock := self.currentBlock
	self.mu.RUnlock()

	// Compare the TD of the last known block in the canonical chain to make sure it's greater.
	// At this point it's possible that a different chain (fork) becomes the new canonical chain.
	if td.Cmp(self.Td()) > 0 {
		// chain fork
		if block.ParentHash() != cblock.Hash() {
			// during split we merge two different chains and create the new canonical chain
			err := self.merge(cblock, block)
			if err != nil {
				return NonStatTy, err
			}
			status = SplitStatTy
		}
		self.mu.Lock()
		self.setTotalDifficulty(td)
		self.insert(block)
		self.mu.Unlock()

		status = CanonStatTy
	} else {
		status = SideStatTy
	}

	if err := WriteTd(self.chainDb, block.Hash(), td); err != nil {
		glog.Fatalf("failed to write block total difficulty: %v", err)
	}
	if err := WriteBlock(self.chainDb, block); err != nil {
		glog.Fatalf("filed to write block contents: %v", err)
	}
	// Delete from future blocks
	self.futureBlocks.Remove(block.Hash())

	return
}

// InsertChain will attempt to insert the given chain in to the canonical chain or, otherwise, create a fork. It an error is returned
// it will return the index number of the failing block as well an error describing what went wrong (for possible errors see core/errors.go).
func (self *ChainManager) InsertChain(chain types.Blocks) (int, error) {
	self.wg.Add(1)
	defer self.wg.Done()

	self.chainmu.Lock()
	defer self.chainmu.Unlock()

	// A queued approach to delivering events. This is generally
	// faster than direct delivery and requires much less mutex
	// acquiring.
	var (
		queue      = make([]interface{}, len(chain))
		queueEvent = queueEvent{queue: queue}
		stats      struct{ queued, processed, ignored int }
		tstart     = time.Now()

		nonceDone    = make(chan nonceResult, len(chain))
		nonceQuit    = make(chan struct{})
		nonceChecked = make([]bool, len(chain))
	)

	// Start the parallel nonce verifier.
	go verifyNonces(self.pow, chain, nonceQuit, nonceDone)
	defer close(nonceQuit)

	txcount := 0
	for i, block := range chain {
		if atomic.LoadInt32(&self.procInterrupt) == 1 {
			glog.V(logger.Debug).Infoln("Premature abort during chain processing")
			break
		}

		bstart := time.Now()
		// Wait for block i's nonce to be verified before processing
		// its state transition.
		for !nonceChecked[i] {
			r := <-nonceDone
			nonceChecked[r.i] = true
			if !r.valid {
				block := chain[r.i]
				return r.i, &BlockNonceErr{Hash: block.Hash(), Number: block.Number(), Nonce: block.Nonce()}
			}
		}

		if BadHashes[block.Hash()] {
			err := fmt.Errorf("Found known bad hash in chain %x", block.Hash())
			blockErr(block, err)
			return i, err
		}
		// Call in to the block processor and check for errors. It's likely that if one block fails
		// all others will fail too (unless a known block is returned).
		logs, receipts, err := self.processor.Process(block)
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

			blockErr(block, err)

			go ReportBlock(block, err)

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
			queue[i] = ChainEvent{block, block.Hash(), logs}
			queueEvent.canonicalCount++

			// This puts transactions in a extra db for rpc
			PutTransactions(self.chainDb, block, block.Transactions())
			// store the receipts
			PutReceipts(self.chainDb, receipts)
		case SideStatTy:
			if glog.V(logger.Detail) {
				glog.Infof("inserted forked block #%d (TD=%v) (%d TXs %d UNCs) (%x...). Took %v\n", block.Number(), block.Difficulty(), len(block.Transactions()), len(block.Uncles()), block.Hash().Bytes()[0:4], time.Since(bstart))
			}
			queue[i] = ChainSideEvent{block, logs}
			queueEvent.sideCount++
		case SplitStatTy:
			queue[i] = ChainSplitEvent{block, logs}
			queueEvent.splitCount++
		}
		if err := PutBlockReceipts(self.chainDb, block, receipts); err != nil {
			glog.V(logger.Warn).Infoln("error writing block receipts:", err)
		}

		stats.processed++
	}

	if (stats.queued > 0 || stats.processed > 0 || stats.ignored > 0) && bool(glog.V(logger.Info)) {
		tend := time.Since(tstart)
		start, end := chain[0], chain[len(chain)-1]
		glog.Infof("imported %d block(s) (%d queued %d ignored) including %d txs in %v. #%v [%x / %x]\n", stats.processed, stats.queued, stats.ignored, txcount, tend, end.Number(), start.Hash().Bytes()[:4], end.Hash().Bytes()[:4])
	}

	go self.eventMux.Post(queueEvent)

	return 0, nil
}

// diff takes two blocks, an old chain and a new chain and will reconstruct the blocks and inserts them
// to be part of the new canonical chain.
func (self *ChainManager) diff(oldBlock, newBlock *types.Block) (types.Blocks, error) {
	var (
		newChain    types.Blocks
		commonBlock *types.Block
		oldStart    = oldBlock
		newStart    = newBlock
	)

	// first reduce whoever is higher bound
	if oldBlock.NumberU64() > newBlock.NumberU64() {
		// reduce old chain
		for oldBlock = oldBlock; oldBlock != nil && oldBlock.NumberU64() != newBlock.NumberU64(); oldBlock = self.GetBlock(oldBlock.ParentHash()) {
		}
	} else {
		// reduce new chain and append new chain blocks for inserting later on
		for newBlock = newBlock; newBlock != nil && newBlock.NumberU64() != oldBlock.NumberU64(); newBlock = self.GetBlock(newBlock.ParentHash()) {
			newChain = append(newChain, newBlock)
		}
	}
	if oldBlock == nil {
		return nil, fmt.Errorf("Invalid old chain")
	}
	if newBlock == nil {
		return nil, fmt.Errorf("Invalid new chain")
	}

	numSplit := newBlock.Number()
	for {
		if oldBlock.Hash() == newBlock.Hash() {
			commonBlock = oldBlock
			break
		}
		newChain = append(newChain, newBlock)

		oldBlock, newBlock = self.GetBlock(oldBlock.ParentHash()), self.GetBlock(newBlock.ParentHash())
		if oldBlock == nil {
			return nil, fmt.Errorf("Invalid old chain")
		}
		if newBlock == nil {
			return nil, fmt.Errorf("Invalid new chain")
		}
	}

	if glog.V(logger.Debug) {
		commonHash := commonBlock.Hash()
		glog.Infof("Chain split detected @ %x. Reorganising chain from #%v %x to %x", commonHash[:4], numSplit, oldStart.Hash().Bytes()[:4], newStart.Hash().Bytes()[:4])
	}

	return newChain, nil
}

// merge merges two different chain to the new canonical chain
func (self *ChainManager) merge(oldBlock, newBlock *types.Block) error {
	newChain, err := self.diff(oldBlock, newBlock)
	if err != nil {
		return fmt.Errorf("chain reorg failed: %v", err)
	}

	// insert blocks. Order does not matter. Last block will be written in ImportChain itself which creates the new head properly
	self.mu.Lock()
	for _, block := range newChain {
		// insert the block in the canonical way, re-writing history
		self.insert(block)
		// write canonical receipts and transactions
		PutTransactions(self.chainDb, block, block.Transactions())
		PutReceipts(self.chainDb, GetBlockReceipts(self.chainDb, block.Hash()))

	}
	self.mu.Unlock()

	return nil
}

func (self *ChainManager) update() {
	events := self.eventMux.Subscribe(queueEvent{})
	futureTimer := time.Tick(5 * time.Second)
out:
	for {
		select {
		case ev := <-events.Chan():
			switch ev := ev.(type) {
			case queueEvent:
				for _, event := range ev.queue {
					switch event := event.(type) {
					case ChainEvent:
						// We need some control over the mining operation. Acquiring locks and waiting for the miner to create new block takes too long
						// and in most cases isn't even necessary.
						if self.currentBlock.Hash() == event.Hash {
							self.currentGasLimit = CalcGasLimit(event.Block)
							self.eventMux.Post(ChainHeadEvent{event.Block})
						}
					}
					self.eventMux.Post(event)
				}
			}
		case <-futureTimer:
			self.procFutureBlocks()
		case <-self.quit:
			break out
		}
	}
}

func blockErr(block *types.Block, err error) {
	h := block.Header()
	glog.V(logger.Error).Infof("Bad block #%v (%x)\n", h.Number, h.Hash().Bytes())
	glog.V(logger.Error).Infoln(err)
	glog.V(logger.Debug).Infoln(verifyNonces)
}

type nonceResult struct {
	i     int
	valid bool
}

// block verifies nonces of the given blocks in parallel and returns
// an error if one of the blocks nonce verifications failed.
func verifyNonces(pow pow.PoW, blocks []*types.Block, quit <-chan struct{}, done chan<- nonceResult) {
	// Spawn a few workers. They listen for blocks on the in channel
	// and send results on done. The workers will exit in the
	// background when in is closed.
	var (
		in       = make(chan int)
		nworkers = runtime.GOMAXPROCS(0)
	)
	defer close(in)
	if len(blocks) < nworkers {
		nworkers = len(blocks)
	}
	for i := 0; i < nworkers; i++ {
		go func() {
			for i := range in {
				done <- nonceResult{i: i, valid: pow.Verify(blocks[i])}
			}
		}()
	}
	// Feed block indices to the workers.
	for i := range blocks {
		select {
		case in <- i:
			continue
		case <-quit:
			return
		}
	}
}
