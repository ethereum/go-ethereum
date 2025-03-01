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
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/XinFinOrg/XDPoSChain/XDCx/tradingstate"
	"github.com/XinFinOrg/XDPoSChain/XDCxlending/lendingstate"
	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common"
	"github.com/XinFinOrg/XDPoSChain/common/lru"
	"github.com/XinFinOrg/XDPoSChain/common/mclock"
	"github.com/XinFinOrg/XDPoSChain/common/prque"
	"github.com/XinFinOrg/XDPoSChain/common/sort"
	"github.com/XinFinOrg/XDPoSChain/consensus"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS"
	"github.com/XinFinOrg/XDPoSChain/consensus/XDPoS/utils"
	contractValidator "github.com/XinFinOrg/XDPoSChain/contracts/validator/contract"
	"github.com/XinFinOrg/XDPoSChain/core/rawdb"
	"github.com/XinFinOrg/XDPoSChain/core/state"
	"github.com/XinFinOrg/XDPoSChain/core/types"
	"github.com/XinFinOrg/XDPoSChain/core/vm"
	"github.com/XinFinOrg/XDPoSChain/crypto"
	"github.com/XinFinOrg/XDPoSChain/ethclient"
	"github.com/XinFinOrg/XDPoSChain/ethdb"
	"github.com/XinFinOrg/XDPoSChain/event"
	"github.com/XinFinOrg/XDPoSChain/internal/syncx"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/XinFinOrg/XDPoSChain/metrics"
	"github.com/XinFinOrg/XDPoSChain/params"
	"github.com/XinFinOrg/XDPoSChain/rlp"
	"github.com/XinFinOrg/XDPoSChain/trie"
)

var (
	headBlockGauge     = metrics.NewRegisteredGauge("chain/head/block", nil)
	headHeaderGauge    = metrics.NewRegisteredGauge("chain/head/header", nil)
	headFastBlockGauge = metrics.NewRegisteredGauge("chain/head/receipt", nil)

	chainInfoGauge = metrics.NewRegisteredGaugeInfo("chain/info", nil)

	accountReadTimer   = metrics.NewRegisteredResettingTimer("chain/account/reads", nil)
	accountHashTimer   = metrics.NewRegisteredResettingTimer("chain/account/hashes", nil)
	accountUpdateTimer = metrics.NewRegisteredResettingTimer("chain/account/updates", nil)
	accountCommitTimer = metrics.NewRegisteredResettingTimer("chain/account/commits", nil)

	storageReadTimer   = metrics.NewRegisteredResettingTimer("chain/storage/reads", nil)
	storageHashTimer   = metrics.NewRegisteredResettingTimer("chain/storage/hashes", nil)
	storageUpdateTimer = metrics.NewRegisteredResettingTimer("chain/storage/updates", nil)
	storageCommitTimer = metrics.NewRegisteredResettingTimer("chain/storage/commits", nil)

	blockInsertTimer     = metrics.NewRegisteredResettingTimer("chain/inserts", nil)
	blockValidationTimer = metrics.NewRegisteredResettingTimer("chain/validation", nil)
	blockExecutionTimer  = metrics.NewRegisteredResettingTimer("chain/execution", nil)
	blockWriteTimer      = metrics.NewRegisteredResettingTimer("chain/write", nil)

	blockReorgMeter     = metrics.NewRegisteredMeter("chain/reorg/executes", nil)
	blockReorgAddMeter  = metrics.NewRegisteredMeter("chain/reorg/add", nil)
	blockReorgDropMeter = metrics.NewRegisteredMeter("chain/reorg/drop", nil)

	errInsertionInterrupted = errors.New("insertion is interrupted")
	errChainStopped         = errors.New("blockchain is stopped")
	errInvalidOldChain      = errors.New("invalid old chain")
	errInvalidNewChain      = errors.New("invalid new chain")

	CheckpointCh = make(chan int)
)

const (
	bodyCacheLimit      = 256
	blockCacheLimit     = 256
	receiptsCacheLimit  = 32
	maxFutureBlocks     = 256
	maxTimeFutureBlocks = 30
	badBlockLimit       = 10
	triesInMemory       = 128

	// BlockChainVersion ensures that an incompatible database forces a resync from scratch.
	//
	// Changelog:
	//
	// - Version 4
	//   The following incompatible database changes were added:
	//   * the `BlockNumber`, `TxHash`, `TxIndex`, `BlockHash` and `Index` fields of log are deleted
	//   * the `Bloom` field of receipt is deleted
	//   * the `BlockIndex` and `TxIndex` fields of txlookup are deleted
	// - Version 5
	//  The following incompatible database changes were added:
	//    * the `TxHash`, `GasCost`, and `ContractAddress` fields are no longer stored for a receipt
	//    * the `TxHash`, `GasCost`, and `ContractAddress` fields are computed by looking up the
	//      receipts' corresponding block
	// - Version 6
	//  The following incompatible database changes were added:
	//    * Transaction lookup information stores the corresponding block number instead of block hash
	BlockChainVersion uint64 = 6

	// Maximum length of chain to cache by block's number
	blocksHashCacheLimit = 900
)

// CacheConfig contains the configuration values for the trie caching/pruning
// that's resident in a blockchain.
type CacheConfig struct {
	Disabled      bool          // Whether to disable trie write caching (archive node)
	TrieNodeLimit int           // Memory limit (MB) at which to flush the current in-memory trie to disk
	TrieTimeLimit time.Duration // Time limit after which to flush the current in-memory trie to disk
}
type ResultProcessBlock struct {
	logs         []*types.Log
	receipts     []*types.Receipt
	state        *state.StateDB
	tradingState *tradingstate.TradingStateDB
	lendingState *lendingstate.LendingStateDB
	proctime     time.Duration
	usedGas      uint64
}

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
	chainConfig *params.ChainConfig // Chain & network configuration
	cacheConfig *CacheConfig        // Cache configuration for pruning

	db     ethdb.Database // Low level persistent database to store final content in
	XDCxDb ethdb.XDCxDatabase
	triegc *prque.Prque[int64, common.Hash] // Priority queue mapping block numbers to tries to gc
	gcproc time.Duration                    // Accumulates canonical block processing for trie dumping

	hc            *HeaderChain
	rmLogsFeed    event.Feed
	chainFeed     event.Feed
	chainSideFeed event.Feed
	chainHeadFeed event.Feed
	logsFeed      event.Feed
	scope         event.SubscriptionScope
	genesisBlock  *types.Block

	// This mutex synchronizes chain write operations.
	// Readers don't need to take it, they can just read the database.
	chainmu *syncx.ClosableMutex

	procmu sync.RWMutex // block processor lock

	currentBlock     atomic.Value // Current head of the block chain
	currentFastBlock atomic.Value // Current head of the fast-sync chain (may be above the block chain!)

	stateCache state.Database // State database to reuse between imports (contains state cache)

	bodyCache        *lru.Cache[common.Hash, *types.Body]         // Cache for the most recent block bodies
	bodyRLPCache     *lru.Cache[common.Hash, rlp.RawValue]        // Cache for the most recent block bodies in RLP encoded format
	receiptsCache    *lru.Cache[common.Hash, types.Receipts]      // Cache for the most recent block receipts
	blockCache       *lru.Cache[common.Hash, *types.Block]        // Cache for the most recent entire blocks
	resultProcess    *lru.Cache[common.Hash, *ResultProcessBlock] // Cache for processed blocks
	calculatingBlock *lru.Cache[common.Hash, *CalculatedBlock]    // Cache for processing blocks
	downloadingBlock *lru.Cache[common.Hash, struct{}]            // Cache for downloading blocks (avoid duplication from fetcher)
	badBlocks        *lru.Cache[common.Hash, *types.Header]       // Bad block cache

	// future blocks are blocks added for later processing
	futureBlocks *lru.Cache[common.Hash, *types.Block]

	wg            sync.WaitGroup
	quit          chan struct{} // shutdown signal, closed in Stop.
	running       int32         // 0 if chain is running, 1 when stopped
	procInterrupt int32         // interrupt signaler for block processing

	engine    consensus.Engine
	processor Processor // block processor interface
	validator Validator // block and state validator interface
	vmConfig  vm.Config

	IPCEndpoint string
	Client      bind.ContractBackend // Global ipc client instance.

	// Blocks hash array by block number
	// cache field for tracking finality purpose, can't use for tracking block vs block relationship
	blocksHashCache *lru.Cache[uint64, []common.Hash]

	resultTrade         *lru.Cache[common.Hash, interface{}] // trades result: key - takerOrderHash, value: trades corresponding to takerOrder
	rejectedOrders      *lru.Cache[common.Hash, interface{}] // rejected orders: key - takerOrderHash, value: rejected orders corresponding to takerOrder
	resultLendingTrade  *lru.Cache[common.Hash, interface{}]
	rejectedLendingItem *lru.Cache[common.Hash, interface{}]
	finalizedTrade      *lru.Cache[common.Hash, interface{}] // include both trades which force update to closed/liquidated by the protocol
}

// NewBlockChain returns a fully initialised block chain using information
// available in the database. It initialises the default Ethereum Validator and
// Processor.
func NewBlockChain(db ethdb.Database, cacheConfig *CacheConfig, chainConfig *params.ChainConfig, engine consensus.Engine, vmConfig vm.Config) (*BlockChain, error) {
	if cacheConfig == nil {
		cacheConfig = &CacheConfig{
			TrieNodeLimit: 256 * 1024 * 1024,
			TrieTimeLimit: 5 * time.Minute,
		}
	}

	bc := &BlockChain{
		chainConfig:         chainConfig,
		cacheConfig:         cacheConfig,
		db:                  db,
		triegc:              prque.New[int64, common.Hash](nil),
		stateCache:          state.NewDatabase(db),
		quit:                make(chan struct{}),
		chainmu:             syncx.NewClosableMutex(),
		bodyCache:           lru.NewCache[common.Hash, *types.Body](bodyCacheLimit),
		bodyRLPCache:        lru.NewCache[common.Hash, rlp.RawValue](bodyCacheLimit),
		receiptsCache:       lru.NewCache[common.Hash, types.Receipts](receiptsCacheLimit),
		blockCache:          lru.NewCache[common.Hash, *types.Block](blockCacheLimit),
		futureBlocks:        lru.NewCache[common.Hash, *types.Block](maxFutureBlocks),
		resultProcess:       lru.NewCache[common.Hash, *ResultProcessBlock](blockCacheLimit),
		calculatingBlock:    lru.NewCache[common.Hash, *CalculatedBlock](blockCacheLimit),
		downloadingBlock:    lru.NewCache[common.Hash, struct{}](blockCacheLimit),
		engine:              engine,
		vmConfig:            vmConfig,
		badBlocks:           lru.NewCache[common.Hash, *types.Header](badBlockLimit),
		blocksHashCache:     lru.NewCache[uint64, []common.Hash](blocksHashCacheLimit),
		resultTrade:         lru.NewCache[common.Hash, interface{}](tradingstate.OrderCacheLimit),
		rejectedOrders:      lru.NewCache[common.Hash, interface{}](tradingstate.OrderCacheLimit),
		resultLendingTrade:  lru.NewCache[common.Hash, interface{}](tradingstate.OrderCacheLimit),
		rejectedLendingItem: lru.NewCache[common.Hash, interface{}](tradingstate.OrderCacheLimit),
		finalizedTrade:      lru.NewCache[common.Hash, interface{}](tradingstate.OrderCacheLimit),
	}
	bc.SetValidator(NewBlockValidator(chainConfig, bc, engine))
	bc.SetProcessor(NewStateProcessor(chainConfig, bc, engine))

	var err error
	bc.hc, err = NewHeaderChain(db, chainConfig, engine, bc.insertStopped)
	if err != nil {
		return nil, err
	}
	bc.genesisBlock = bc.GetBlockByNumber(0)
	if bc.genesisBlock == nil {
		return nil, ErrNoGenesis
	}

	// Update chain info data metrics
	chainInfoGauge.Update(metrics.GaugeInfoValue{"chain_id": bc.chainConfig.ChainId.String()})

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

	// Start future block processor.
	bc.wg.Add(1)
	go bc.futureBlocksLoop()

	return bc, nil
}

// GetVMConfig returns the block chain VM config.
func (bc *BlockChain) GetVMConfig() *vm.Config {
	return &bc.vmConfig
}

// NewBlockChainEx extend old blockchain, add order state db
func NewBlockChainEx(db ethdb.Database, XDCxDb ethdb.XDCxDatabase, cacheConfig *CacheConfig, chainConfig *params.ChainConfig, engine consensus.Engine, vmConfig vm.Config) (*BlockChain, error) {
	blockchain, err := NewBlockChain(db, cacheConfig, chainConfig, engine, vmConfig)
	if err != nil {
		return nil, err
	}
	if blockchain != nil {
		blockchain.addXDCxDb(XDCxDb)
	}
	return blockchain, nil
}

func (bc *BlockChain) addXDCxDb(XDCxDb ethdb.XDCxDatabase) {
	bc.XDCxDb = XDCxDb
}

// loadLastState loads the last known chain state from the database. This method
// assumes that the chain manager mutex is held.
func (bc *BlockChain) loadLastState() error {
	// Restore the last known head block
	head := rawdb.ReadHeadBlockHash(bc.db)
	if head == (common.Hash{}) {
		// Corrupt or empty database, init from scratch
		log.Warn("Empty database, resetting chain")
		return bc.Reset()
	}
	// Make sure the entire head block is available
	currentBlock := bc.GetBlockByHash(head)
	if currentBlock == nil {
		// Corrupt or empty database, init from scratch
		log.Warn("Head block missing, resetting chain", "hash", head)
		return bc.Reset()
	}
	// Make sure the state associated with the block is available
	repair := false
	_, err := state.New(currentBlock.Root(), bc.stateCache)
	if err != nil {
		repair = true
	} else {
		engine, ok := bc.Engine().(*XDPoS.XDPoS)
		if ok {
			tradingService := engine.GetXDCXService()
			lendingService := engine.GetLendingService()
			if bc.Config().IsTIPXDCX(currentBlock.Number()) && bc.chainConfig.XDPoS != nil && currentBlock.NumberU64() > bc.chainConfig.XDPoS.Epoch && tradingService != nil && lendingService != nil {
				author, _ := bc.Engine().Author(currentBlock.Header())
				tradingRoot, err := tradingService.GetTradingStateRoot(currentBlock, author)
				if err != nil {
					repair = true
				} else {
					if tradingService.GetStateCache() != nil {
						_, err = tradingstate.New(tradingRoot, tradingService.GetStateCache())
						if err != nil {
							repair = true
						}
					}
				}

				if !repair {
					lendingRoot, err := lendingService.GetLendingStateRoot(currentBlock, author)
					if err != nil {
						repair = true
					} else {
						if lendingService.GetStateCache() != nil {
							_, err = lendingstate.New(lendingRoot, lendingService.GetStateCache())
							if err != nil {
								repair = true
							}
						}
					}
				}
			}
		}
	}
	if repair {
		// Dangling block without a state associated, init from scratch
		log.Warn("Head state missing, repairing chain", "number", currentBlock.Number(), "hash", currentBlock.Hash())
		if err := bc.repair(&currentBlock); err != nil {
			return err
		}
	}
	// Everything seems to be fine, set as the head block
	bc.currentBlock.Store(currentBlock)
	headBlockGauge.Update(int64(currentBlock.NumberU64()))

	// Restore the last known head header
	currentHeader := currentBlock.Header()
	if head := rawdb.ReadHeadHeaderHash(bc.db); head != (common.Hash{}) {
		if header := bc.GetHeaderByHash(head); header != nil {
			currentHeader = header
		}
	}
	bc.hc.SetCurrentHeader(currentHeader)

	// Restore the last known head fast block
	bc.currentFastBlock.Store(currentBlock)
	headFastBlockGauge.Update(int64(currentBlock.NumberU64()))

	if head := rawdb.ReadHeadFastBlockHash(bc.db); head != (common.Hash{}) {
		if block := bc.GetBlockByHash(head); block != nil {
			bc.currentFastBlock.Store(block)
			headFastBlockGauge.Update(int64(block.NumberU64()))
		}
	}

	// Issue a status log for the user
	currentFastBlock := bc.CurrentFastBlock()

	headerTd := bc.GetTd(currentHeader.Hash(), currentHeader.Number.Uint64())
	blockTd := bc.GetTd(currentBlock.Hash(), currentBlock.NumberU64())
	fastTd := bc.GetTd(currentFastBlock.Hash(), currentFastBlock.NumberU64())

	log.Info("Loaded most recent local header", "number", currentHeader.Number, "hash", currentHeader.Hash(), "td", headerTd)
	log.Info("Loaded most recent local full block", "number", currentBlock.Number(), "hash", currentBlock.Hash(), "td", blockTd)
	log.Info("Loaded most recent local fast block", "number", currentFastBlock.Number(), "hash", currentFastBlock.Hash(), "td", fastTd)

	return nil
}

// SetHead rewinds the local chain to a new head. In the case of headers, everything
// above the new head will be deleted and the new one set. In the case of blocks
// though, the head may be further rewound if block bodies are missing (non-archive
// nodes after a fast sync).
func (bc *BlockChain) SetHead(head uint64) error {
	if !bc.chainmu.TryLock() {
		return errChainStopped
	}
	defer bc.chainmu.Unlock()

	updateFn := func(db ethdb.KeyValueWriter, header *types.Header) {
		// Rewind the block chain, ensuring we don't end up with a stateless head block
		if currentBlock := bc.CurrentBlock(); currentBlock != nil && header.Number.Uint64() < currentBlock.NumberU64() {
			newHeadBlock := bc.GetBlock(header.Hash(), header.Number.Uint64())
			if newHeadBlock == nil {
				newHeadBlock = bc.genesisBlock
			} else {
				if _, err := state.New(newHeadBlock.Root(), bc.stateCache); err != nil {
					// Rewound state missing, rolled back to before pivot, reset to genesis
					newHeadBlock = bc.genesisBlock
				}
			}
			rawdb.WriteHeadBlockHash(db, newHeadBlock.Hash())

			// Degrade the chain markers if they are explicitly reverted.
			// In theory we should update all in-memory markers in the
			// last step, however the direction of SetHead is from high
			// to low, so it's safe the update in-memory markers directly.
			bc.currentBlock.Store(newHeadBlock)
			headBlockGauge.Update(int64(newHeadBlock.NumberU64()))
		}

		// Rewind the fast block in a simpleton way to the target head
		if currentFastBlock := bc.CurrentFastBlock(); currentFastBlock != nil && header.Number.Uint64() < currentFastBlock.NumberU64() {
			newHeadFastBlock := bc.GetBlock(header.Hash(), header.Number.Uint64())
			// If either blocks reached nil, reset to the genesis state
			if newHeadFastBlock == nil {
				newHeadFastBlock = bc.genesisBlock
			}
			rawdb.WriteHeadFastBlockHash(db, newHeadFastBlock.Hash())

			// Degrade the chain markers if they are explicitly reverted.
			// In theory we should update all in-memory markers in the
			// last step, however the direction of SetHead is from high
			// to low, so it's safe the update in-memory markers directly.
			bc.currentFastBlock.Store(newHeadFastBlock)
			headFastBlockGauge.Update(int64(newHeadFastBlock.NumberU64()))
		}
	}

	// Rewind the header chain, deleting all block bodies until then
	delFn := func(db ethdb.KeyValueWriter, hash common.Hash, num uint64) {
		// Ignore the error here since light client won't hit this path
		frozen, _ := bc.db.Ancients()
		if num+1 <= frozen {
			// Truncate all relative data(header, total difficulty, body, receipt
			// and canonical hash) from ancient store.
			if err := bc.db.TruncateAncients(num + 1); err != nil {
				log.Crit("Failed to truncate ancient data", "number", num, "err", err)
			}

			// Remove the hash <-> number mapping from the active store.
			rawdb.DeleteHeaderNumber(db, hash)
		} else {
			// Remove relative body and receipts from the active store.
			// The header, total difficulty and canonical hash will be
			// removed in the hc.SetHead function.
			rawdb.DeleteBody(db, hash, num)
			rawdb.DeleteReceipts(db, hash, num)
		}
		// Todo(rjl493456442) txlookup, bloombits, etc
	}
	bc.hc.SetHead(head, updateFn, delFn)

	// Clear out any stale content from the caches
	bc.bodyCache.Purge()
	bc.bodyRLPCache.Purge()
	bc.receiptsCache.Purge()
	bc.blockCache.Purge()
	bc.futureBlocks.Purge()
	bc.blocksHashCache.Purge()

	return bc.loadLastState()
}

// FastSyncCommitHead sets the current head block to the one defined by the hash
// irrelevant what the chain contents were prior.
func (bc *BlockChain) FastSyncCommitHead(hash common.Hash) error {
	// Make sure that both the block as well at its state trie exists
	block := bc.GetBlockByHash(hash)
	if block == nil {
		return fmt.Errorf("non existent block [%x..]", hash[:4])
	}
	if _, err := trie.NewSecure(block.Root(), bc.stateCache.TrieDB()); err != nil {
		return err
	}

	// If all checks out, manually set the head block.
	if !bc.chainmu.TryLock() {
		return errChainStopped
	}
	bc.currentBlock.Store(block)
	headBlockGauge.Update(int64(block.NumberU64()))
	bc.chainmu.Unlock()

	log.Info("Committed new head block", "number", block.Number(), "hash", hash)
	return nil
}

// GasLimit returns the gas limit of the current HEAD block.
func (bc *BlockChain) GasLimit() uint64 {
	return bc.CurrentBlock().GasLimit()
}

// CurrentBlock retrieves the current head block of the canonical chain. The
// block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) CurrentBlock() *types.Block {
	return bc.currentBlock.Load().(*types.Block)
}

// CurrentFastBlock retrieves the current fast-sync head block of the canonical
// chain. The block is retrieved from the blockchain's internal cache.
func (bc *BlockChain) CurrentFastBlock() *types.Block {
	return bc.currentFastBlock.Load().(*types.Block)
}

// SetProcessor sets the processor required for making state modifications.
func (bc *BlockChain) SetProcessor(processor Processor) {
	bc.procmu.Lock()
	defer bc.procmu.Unlock()
	bc.processor = processor
}

// SetValidator sets the validator which is used to validate incoming blocks.
func (bc *BlockChain) SetValidator(validator Validator) {
	bc.procmu.Lock()
	defer bc.procmu.Unlock()
	bc.validator = validator
}

// Validator returns the current validator.
func (bc *BlockChain) Validator() Validator {
	bc.procmu.RLock()
	defer bc.procmu.RUnlock()
	return bc.validator
}

// Processor returns the current processor.
func (bc *BlockChain) Processor() Processor {
	bc.procmu.RLock()
	defer bc.procmu.RUnlock()
	return bc.processor
}

// State returns a new mutable state based on the current HEAD block.
func (bc *BlockChain) State() (*state.StateDB, error) {
	return bc.StateAt(bc.CurrentBlock().Root())
}

// StateAt returns a new mutable state based on a particular point in time.
func (bc *BlockChain) StateAt(root common.Hash) (*state.StateDB, error) {
	return state.New(root, bc.stateCache)
}

// OrderStateAt returns a new mutable state based on a particular point in time.
func (bc *BlockChain) OrderStateAt(block *types.Block) (*tradingstate.TradingStateDB, error) {
	engine, ok := bc.Engine().(*XDPoS.XDPoS)
	if ok {
		XDCXService := engine.GetXDCXService()
		if bc.Config().IsTIPXDCX(block.Number()) && bc.chainConfig.XDPoS != nil && block.NumberU64() > bc.chainConfig.XDPoS.Epoch && XDCXService != nil {
			author, _ := bc.Engine().Author(block.Header())
			log.Debug("OrderStateAt", "blocknumber", block.Header().Number)
			XDCxState, err := XDCXService.GetTradingState(block, author)
			if err == nil {
				return XDCxState, nil
			} else {
				return nil, err
			}
		} else {
			XDCxState, err := XDCXService.GetEmptyTradingState()
			if err == nil {
				return XDCxState, nil
			} else {
				return nil, err
			}
		}
	}
	return nil, errors.New("Get XDCx state fail")

}

// LendingStateAt returns a new mutable state based on a particular point in time.
func (bc *BlockChain) LendingStateAt(block *types.Block) (*lendingstate.LendingStateDB, error) {
	engine, ok := bc.Engine().(*XDPoS.XDPoS)
	if ok {
		lendingService := engine.GetLendingService()
		if bc.Config().IsTIPXDCX(block.Number()) && bc.chainConfig.XDPoS != nil && block.NumberU64() > bc.chainConfig.XDPoS.Epoch && lendingService != nil {
			author, _ := bc.Engine().Author(block.Header())
			log.Debug("LendingStateAt", "blocknumber", block.Header().Number)
			lendingState, err := lendingService.GetLendingState(block, author)
			if err == nil {
				return lendingState, nil
			}
			return nil, err
		}
	}
	return nil, errors.New("Get XDCx state fail")

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
	if !bc.chainmu.TryLock() {
		return errChainStopped
	}
	defer bc.chainmu.Unlock()

	// Prepare the genesis block and reinitialise the chain
	batch := bc.db.NewBatch()
	rawdb.WriteTd(batch, genesis.Hash(), genesis.NumberU64(), genesis.Difficulty())
	rawdb.WriteBlock(batch, genesis)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write genesis block", "err", err)
	}
	bc.writeHeadBlock(genesis, false)

	// Last update all in-memory chain markers
	bc.genesisBlock = genesis
	bc.currentBlock.Store(bc.genesisBlock)
	headBlockGauge.Update(int64(bc.genesisBlock.NumberU64()))
	bc.hc.SetGenesis(bc.genesisBlock.Header())
	bc.hc.SetCurrentHeader(bc.genesisBlock.Header())
	bc.currentFastBlock.Store(bc.genesisBlock)
	headFastBlockGauge.Update(int64(bc.genesisBlock.NumberU64()))
	return nil
}

// repair tries to repair the current blockchain by rolling back the current block
// until one with associated state is found. This is needed to fix incomplete db
// writes caused either by crashes/power outages, or simply non-committed tries.
//
// This method only rolls back the current block. The current header and current
// fast block are left intact.
func (bc *BlockChain) repair(head **types.Block) error {
	for {
		// Abort if we've rewound to a head block that does have associated state
		if (common.RollbackNumber == 0) || ((*head).Number().Uint64() < common.RollbackNumber) {
			if _, err := state.New((*head).Root(), bc.stateCache); err == nil {
				log.Info("Rewound blockchain to past state", "number", (*head).Number(), "hash", (*head).Hash())
				engine, ok := bc.Engine().(*XDPoS.XDPoS)
				if ok {
					tradingService := engine.GetXDCXService()
					lendingService := engine.GetLendingService()
					if bc.Config().IsTIPXDCXReceiver((*head).Number()) && bc.chainConfig.XDPoS != nil && (*head).NumberU64() > bc.chainConfig.XDPoS.Epoch && tradingService != nil && lendingService != nil {
						author, _ := bc.Engine().Author((*head).Header())
						tradingRoot, err := tradingService.GetTradingStateRoot(*head, author)
						if err == nil {
							_, err = tradingstate.New(tradingRoot, tradingService.GetStateCache())
						}
						if err == nil {
							lendingRoot, err := lendingService.GetLendingStateRoot(*head, author)
							if err == nil {
								_, err = lendingstate.New(lendingRoot, lendingService.GetStateCache())
								if err == nil {
									return nil
								}
							}
						}
					} else {
						return nil
					}
				} else {
					return nil
				}
			}
		} else {
			log.Info("Rewound blockchain to past state", "number", (*head).Number(), "hash", (*head).Hash())
		}
		// Otherwise rewind one block and recheck state availability there
		(*head) = bc.GetBlock((*head).ParentHash(), (*head).NumberU64()-1)
	}
}

// Export writes the active chain to the given writer.
func (bc *BlockChain) Export(w io.Writer) error {
	return bc.ExportN(w, uint64(0), bc.CurrentBlock().NumberU64())
}

// ExportN writes a subset of the active chain to the given writer.
func (bc *BlockChain) ExportN(w io.Writer, first uint64, last uint64) error {
	if !bc.chainmu.TryLock() {
		return errChainStopped
	}
	defer bc.chainmu.Unlock()

	if first > last {
		return fmt.Errorf("export failed: first (%d) is greater than last (%d)", first, last)
	}
	log.Info("Exporting batch of blocks", "count", last-first+1)

	for nr := first; nr <= last; nr++ {
		block := bc.GetBlockByNumber(nr)
		if block == nil {
			return fmt.Errorf("export failed on #%d: not found", nr)
		}

		if err := block.EncodeRLP(w); err != nil {
			return err
		}
	}
	return nil
}

// writeHeadBlock injects a new head block into the current block chain. This method
// assumes that the block is indeed a true head. It will also reset the head
// header and the head fast sync block to this very same block if they are older
// or if they are on a different side chain.
//
// Note, this function assumes that the `mu` mutex is held!
func (bc *BlockChain) writeHeadBlock(block *types.Block, writeBlock bool) {
	blockHash := block.Hash()
	blockNumberU64 := block.NumberU64()

	// Add the block to the canonical chain number scheme and mark as the head
	batch := bc.db.NewBatch()
	rawdb.WriteHeadHeaderHash(batch, blockHash)
	rawdb.WriteHeadFastBlockHash(batch, blockHash)
	rawdb.WriteCanonicalHash(batch, blockHash, blockNumberU64)
	rawdb.WriteTxLookupEntriesByBlock(batch, block)
	rawdb.WriteHeadBlockHash(batch, blockHash)
	if writeBlock {
		rawdb.WriteBlock(batch, block)
	}

	// Flush the whole batch into the disk, exit the node if failed
	if err := batch.Write(); err != nil {
		log.Crit("Failed to update chain indexes and markers", "err", err)
	}

	// Update all in-memory chain markers in the last step
	bc.hc.SetCurrentHeader(block.Header())

	bc.currentFastBlock.Store(block)
	headFastBlockGauge.Update(int64(blockNumberU64))

	bc.currentBlock.Store(block)
	headBlockGauge.Update(int64(block.NumberU64()))

	// save cache BlockSigners
	if bc.chainConfig.XDPoS != nil && !bc.chainConfig.IsTIPSigning(block.Number()) {
		engine, ok := bc.Engine().(*XDPoS.XDPoS)
		if ok {
			engine.CacheNoneTIPSigningTxs(block.Header(), block.Transactions(), bc.GetReceiptsByHash(blockHash))
		}
	}
}

// Genesis retrieves the chain's genesis block.
func (bc *BlockChain) Genesis() *types.Block {
	return bc.genesisBlock
}

// GetBody retrieves a block body (transactions and uncles) from the database by
// hash, caching it if found.
func (bc *BlockChain) GetBody(hash common.Hash) *types.Body {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := bc.bodyCache.Get(hash); ok {
		return cached
	}
	number := bc.hc.GetBlockNumber(hash)
	if number == nil {
		return nil
	}
	body := rawdb.ReadBody(bc.db, hash, *number)
	if body == nil {
		return nil
	}
	// Cache the found body for next time and return
	bc.bodyCache.Add(hash, body)
	return body
}

// GetBodyRLP retrieves a block body in RLP encoding from the database by hash,
// caching it if found.
func (bc *BlockChain) GetBodyRLP(hash common.Hash) rlp.RawValue {
	// Short circuit if the body's already in the cache, retrieve otherwise
	if cached, ok := bc.bodyRLPCache.Get(hash); ok {
		return cached
	}
	number := bc.hc.GetBlockNumber(hash)
	if number == nil {
		return nil
	}
	body := rawdb.ReadBodyRLP(bc.db, hash, *number)
	if len(body) == 0 {
		return nil
	}
	// Cache the found body for next time and return
	bc.bodyRLPCache.Add(hash, body)
	return body
}

// HasBlock checks if a block is fully present in the database or not.
func (bc *BlockChain) HasBlock(hash common.Hash, number uint64) bool {
	if bc.blockCache.Contains(hash) {
		return true
	}
	if !bc.HasHeader(hash, number) {
		return false
	}
	return rawdb.HasBody(bc.db, hash, number)
}

// HasFastBlock checks if a fast block is fully present in the database or not.
func (bc *BlockChain) HasFastBlock(hash common.Hash, number uint64) bool {
	if !bc.HasBlock(hash, number) {
		return false
	}
	if bc.receiptsCache.Contains(hash) {
		return true
	}
	return rawdb.HasReceipts(bc.db, hash, number)
}

// HasFullState checks if state trie is fully present in the database or not.
func (bc *BlockChain) HasFullState(block *types.Block) bool {
	_, err := bc.stateCache.OpenTrie(block.Root())
	if err != nil {
		return false
	}
	engine, _ := bc.Engine().(*XDPoS.XDPoS)
	if bc.Config().IsTIPXDCX(block.Number()) && bc.chainConfig.XDPoS != nil && engine != nil && block.NumberU64() > bc.chainConfig.XDPoS.Epoch {
		tradingService := engine.GetXDCXService()
		lendingService := engine.GetLendingService()
		author, _ := bc.Engine().Author(block.Header())
		if tradingService != nil && !tradingService.HasTradingState(block, author) {
			return false
		}
		if lendingService != nil && !lendingService.HasLendingState(block, author) {
			return false
		}
	}
	return true
}

// HasBlockAndFullState checks if a block and associated state trie is fully present
// in the database or not, caching it if present.
func (bc *BlockChain) HasBlockAndFullState(hash common.Hash, number uint64) bool {
	// Check first that the block itself is known
	block := bc.GetBlock(hash, number)
	if block == nil {
		return false
	}
	return bc.HasFullState(block)
}

// GetBlock retrieves a block from the database by hash and number,
// caching it if found.
func (bc *BlockChain) GetBlock(hash common.Hash, number uint64) *types.Block {
	// Short circuit if the block's already in the cache, retrieve otherwise
	if block, ok := bc.blockCache.Get(hash); ok {
		return block
	}
	block := rawdb.ReadBlock(bc.db, hash, number)
	if block == nil {
		return nil
	}
	// Cache the found block for next time and return
	bc.blockCache.Add(block.Hash(), block)
	return block
}

// GetBlockByHash retrieves a block from the database by hash, caching it if found.
func (bc *BlockChain) GetBlockByHash(hash common.Hash) *types.Block {
	number := bc.hc.GetBlockNumber(hash)
	if number == nil {
		return nil
	}
	return bc.GetBlock(hash, *number)
}

// GetBlockByNumber retrieves a block from the database by number, caching it
// (associated with its hash) if found.
func (bc *BlockChain) GetBlockByNumber(number uint64) *types.Block {
	hash := rawdb.ReadCanonicalHash(bc.db, number)
	if hash == (common.Hash{}) {
		return nil
	}
	return bc.GetBlock(hash, number)
}

// GetReceiptsByHash retrieves the receipts for all transactions in a given block.
func (bc *BlockChain) GetReceiptsByHash(hash common.Hash) types.Receipts {
	if receipts, ok := bc.receiptsCache.Get(hash); ok {
		return receipts
	}
	number := rawdb.ReadHeaderNumber(bc.db, hash)
	if number == nil {
		return nil
	}
	receipts := rawdb.ReadReceipts(bc.db, hash, *number, bc.chainConfig)
	if receipts == nil {
		return nil
	}
	bc.receiptsCache.Add(hash, receipts)
	return receipts
}

// GetBlocksFromHash returns the block corresponding to hash and up to n-1 ancestors.
// [deprecated by eth/62]
func (bc *BlockChain) GetBlocksFromHash(hash common.Hash, n int) (blocks []*types.Block) {
	number := bc.hc.GetBlockNumber(hash)
	if number == nil {
		return nil
	}
	for i := 0; i < n; i++ {
		block := bc.GetBlock(hash, *number)
		if block == nil {
			break
		}
		blocks = append(blocks, block)
		hash = block.ParentHash()
		*number--
	}
	return
}

// GetBlocksHashCache get all block's hashes with same level
// just work with latest blocksHashCacheLimit
func (bc *BlockChain) GetBlocksHashCache(number uint64) []common.Hash {
	cached, ok := bc.blocksHashCache.Get(number)

	if ok {
		return cached
	}
	return nil
}

// AreTwoBlockSamePath check if two blocks are same path
// Assume block 1 is ahead block 2 so we need to check parentHash
func (bc *BlockChain) AreTwoBlockSamePath(bh1 common.Hash, bh2 common.Hash) bool {
	bl1 := bc.GetBlockByHash(bh1)
	bl2 := bc.GetBlockByHash(bh2)
	toBlockLevel := bl2.Number().Uint64()

	for bl1.Number().Uint64() > toBlockLevel {
		bl1 = bc.GetBlockByHash(bl1.ParentHash())
	}

	return (bl1.Hash() == bl2.Hash())
}

// GetUnclesInChain retrieves all the uncles from a given block backwards until
// a specific distance is reached.
func (bc *BlockChain) GetUnclesInChain(block *types.Block, length int) []*types.Header {
	uncles := []*types.Header{}
	for i := 0; block != nil && i < length; i++ {
		uncles = append(uncles, block.Uncles()...)
		block = bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
	}
	return uncles
}

// TrieNode retrieves a blob of data associated with a trie node (or code hash)
// either from ephemeral in-memory cache, or from persistent storage.
func (bc *BlockChain) TrieNode(hash common.Hash) ([]byte, error) {
	return bc.stateCache.TrieDB().Node(hash)
}

func (bc *BlockChain) saveData() {
	// Ensure the state of a recent block is also stored to disk before exiting.
	// We're writing three different states to catch different restart scenarios:
	//  - HEAD:     So we don't need to reprocess any blocks in the general case
	//  - HEAD-1:   So we don't do large reorgs if our HEAD becomes an uncle
	//  - HEAD-127: So we have a hard limit on the number of blocks reexecuted
	if !bc.cacheConfig.Disabled {
		var tradingTriedb *trie.Database
		var lendingTriedb *trie.Database
		engine, _ := bc.Engine().(*XDPoS.XDPoS)
		triedb := bc.stateCache.TrieDB()
		var tradingService utils.TradingService
		var lendingService utils.LendingService
		if bc.Config().IsTIPXDCX(bc.CurrentBlock().Number()) && bc.chainConfig.XDPoS != nil && bc.CurrentBlock().NumberU64() > bc.chainConfig.XDPoS.Epoch && engine != nil {
			tradingService = engine.GetXDCXService()
			if tradingService != nil && tradingService.GetStateCache() != nil {
				tradingTriedb = tradingService.GetStateCache().TrieDB()
			}
			lendingService = engine.GetLendingService()
			if lendingService != nil && lendingService.GetStateCache() != nil {
				lendingTriedb = lendingService.GetStateCache().TrieDB()
			}
		}
		for _, offset := range []uint64{0, 1, triesInMemory - 1} {
			if number := bc.CurrentBlock().NumberU64(); number > offset {
				recent := bc.GetBlockByNumber(number - offset)

				log.Info("Writing cached state to disk", "block", recent.Number(), "hash", recent.Hash(), "root", recent.Root())
				if err := triedb.Commit(recent.Root(), true); err != nil {
					log.Error("Failed to commit recent state trie", "err", err)
				}
				if bc.Config().IsTIPXDCXReceiver(recent.Number()) && bc.chainConfig.XDPoS != nil && recent.NumberU64() > bc.chainConfig.XDPoS.Epoch && engine != nil {
					author, _ := bc.Engine().Author(recent.Header())
					if tradingService != nil {
						tradingRoot, _ := tradingService.GetTradingStateRoot(recent, author)
						if !tradingRoot.IsZero() && tradingTriedb != nil {
							if err := tradingTriedb.Commit(tradingRoot, true); err != nil {
								log.Error("Failed to commit trading state recent state trie", "err", err)
							}
						}
					}
					if lendingService != nil {
						lendingRoot, _ := lendingService.GetLendingStateRoot(recent, author)
						if !lendingRoot.IsZero() && lendingTriedb != nil {
							if err := lendingTriedb.Commit(lendingRoot, true); err != nil {
								log.Error("Failed to commit lending state recent state trie", "err", err)
							}
						}
					}
				}
			}
		}
		for !bc.triegc.Empty() {
			triedb.Dereference(bc.triegc.PopItem())
		}
		if tradingTriedb != nil && lendingTriedb != nil {
			if tradingService.GetTriegc() != nil {
				for !tradingService.GetTriegc().Empty() {
					tradingTriedb.Dereference(tradingService.GetTriegc().PopItem())
				}
			}
			if lendingService.GetTriegc() != nil {
				for !lendingService.GetTriegc().Empty() {
					lendingTriedb.Dereference(lendingService.GetTriegc().PopItem())
				}
			}
		}
		if size, _ := triedb.Size(); size != 0 {
			log.Error("Dangling trie nodes after full cleanup")
		}
	}
}

// Stop stops the blockchain service. If any imports are currently in progress
// it will abort them using the procInterrupt.
func (bc *BlockChain) Stop() {
	if !atomic.CompareAndSwapInt32(&bc.running, 0, 1) {
		return
	}

	// Unsubscribe all subscriptions registered from blockchain.
	bc.scope.Close()

	// Signal shutdown to all goroutines.
	close(bc.quit)
	bc.StopInsert()

	// Now wait for all chain modifications to end and persistent goroutines to exit.
	//
	// Note: Close waits for the mutex to become available, i.e. any running chain
	// modification will have exited when Close returns. Since we also called StopInsert,
	// the mutex should become available quickly. It cannot be taken again after Close has
	// returned.
	bc.chainmu.Close()
	bc.wg.Wait()
	bc.saveData()
	log.Info("Blockchain manager stopped")
}

// StopInsert interrupts all insertion methods, causing them to return
// errInsertionInterrupted as soon as possible. Insertion is permanently disabled after
// calling this method.
func (bc *BlockChain) StopInsert() {
	atomic.StoreInt32(&bc.procInterrupt, 1)
}

// insertStopped returns true after StopInsert has been called.
func (bc *BlockChain) insertStopped() bool {
	return atomic.LoadInt32(&bc.procInterrupt) == 1
}

func (bc *BlockChain) procFutureBlocks() {
	blocks := make([]*types.Block, 0, bc.futureBlocks.Len())
	for _, hash := range bc.futureBlocks.Keys() {
		if block, exist := bc.futureBlocks.Peek(hash); exist {
			blocks = append(blocks, block)
		}
	}
	if len(blocks) > 0 {
		types.BlockBy(types.Number).Sort(blocks)

		// Insert one by one as chain insertion needs contiguous ancestry between blocks
		for i := range blocks {
			_, err := bc.InsertChain(blocks[i : i+1])
			// let consensus engine handle the last block (e.g. for voting)
			if i == len(blocks)-1 && err == nil {
				engine, ok := bc.Engine().(*XDPoS.XDPoS)
				if ok {
					j := i
					go func() {
						header := blocks[j].Header()
						err = engine.HandleProposedBlock(bc, header)
						if err != nil {
							log.Info("[procFutureBlocks] handle proposed block has error", "err", err, "block hash", header.Hash(), "number", header.Number)
						}
					}()
				}
			}
		}
	}
}

// WriteStatus status of write
type WriteStatus byte

const (
	NonStatTy WriteStatus = iota
	CanonStatTy
	SideStatTy
)

// Rollback is designed to remove a chain of links from the database that aren't
// certain enough to be valid.
func (bc *BlockChain) Rollback(chain []common.Hash) {
	if !bc.chainmu.TryLock() {
		return
	}
	defer bc.chainmu.Unlock()

	batch := bc.db.NewBatch()
	for i := len(chain) - 1; i >= 0; i-- {
		hash := chain[i]

		// Degrade the chain markers if they are explicitly reverted.
		// In theory we should update all in-memory markers in the
		// last step, however the direction of rollback is from high
		// to low, so it's safe the update in-memory markers directly.
		currentHeader := bc.hc.CurrentHeader()
		if currentHeader.Hash() == hash {
			newHeadHeader := bc.GetHeader(currentHeader.ParentHash, currentHeader.Number.Uint64()-1)
			rawdb.WriteHeadHeaderHash(batch, currentHeader.ParentHash)
			bc.hc.SetCurrentHeader(newHeadHeader)
		}
		if currentFastBlock := bc.CurrentFastBlock(); currentFastBlock.Hash() == hash {
			newFastBlock := bc.GetBlock(currentFastBlock.ParentHash(), currentFastBlock.NumberU64()-1)
			rawdb.WriteHeadFastBlockHash(batch, currentFastBlock.ParentHash())
			bc.currentFastBlock.Store(newFastBlock)
			headFastBlockGauge.Update(int64(newFastBlock.NumberU64()))
		}
		if currentBlock := bc.CurrentBlock(); currentBlock.Hash() == hash {
			newBlock := bc.GetBlock(currentBlock.ParentHash(), currentBlock.NumberU64()-1)
			rawdb.WriteHeadBlockHash(batch, currentBlock.ParentHash())
			bc.currentBlock.Store(newBlock)
			headBlockGauge.Update(int64(newBlock.NumberU64()))
		}
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to rollback chain markers", "err", err)
	}
	// TODO: Truncate ancient data which exceeds the current header.
}

// InsertReceiptChain attempts to complete an already existing header chain with
// transaction and receipt data.
func (bc *BlockChain) InsertReceiptChain(blockChain types.Blocks, receiptChain []types.Receipts) (int, error) {
	// We don't require the chainMu here since we want to maximize the
	// concurrency of header insertion and receipt insertion.
	bc.wg.Add(1)
	defer bc.wg.Done()

	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(blockChain); i++ {
		if blockChain[i].NumberU64() != blockChain[i-1].NumberU64()+1 || blockChain[i].ParentHash() != blockChain[i-1].Hash() {
			log.Error("Non contiguous receipt insert", "number", blockChain[i].Number(), "hash", blockChain[i].Hash(), "parent", blockChain[i].ParentHash(),
				"prevnumber", blockChain[i-1].Number(), "prevhash", blockChain[i-1].Hash())
			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x..], item %d is #%d [%x..] (parent [%x..])", i-1, blockChain[i-1].NumberU64(),
				blockChain[i-1].Hash().Bytes()[:4], i, blockChain[i].NumberU64(), blockChain[i].Hash().Bytes()[:4], blockChain[i].ParentHash().Bytes()[:4])
		}
	}

	var (
		stats = struct{ processed, ignored int32 }{}
		start = time.Now()
		bytes = 0
		batch = bc.db.NewBatch()
	)
	for i, block := range blockChain {
		receipts := receiptChain[i]
		// Short circuit insertion if shutting down or processing failed
		if atomic.LoadInt32(&bc.procInterrupt) == 1 {
			return 0, nil
		}
		blockHash, blockNumber := block.Hash(), block.NumberU64()
		// Short circuit if the owner header is unknown
		if !bc.HasHeader(blockHash, blockNumber) {
			return i, fmt.Errorf("containing header #%d [%x..] unknown", blockNumber, blockHash.Bytes()[:4])
		}
		// Skip if the entire data is already known
		if bc.HasBlock(blockHash, blockNumber) {
			stats.ignored++
			continue
		}
		// Compute all the non-consensus fields of the receipts
		if err := receipts.DeriveFields(bc.chainConfig, blockHash, blockNumber, block.BaseFee(), block.Transactions()); err != nil {
			return i, fmt.Errorf("failed to derive receipts data: %v", err)
		}
		// Write all the data out into the database
		rawdb.WriteBody(batch, blockHash, blockNumber, block.Body())
		rawdb.WriteReceipts(batch, blockHash, blockNumber, receipts)
		rawdb.WriteTxLookupEntriesByBlock(batch, block)

		// Write everything belongs to the blocks into the database. So that
		// we can ensure all components of body is completed(body, receipts,
		// tx indexes)
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return 0, err
			}
			bytes += batch.ValueSize()
			batch.Reset()
		}
		stats.processed++
	}
	// Write everything belongs to the blocks into the database. So that
	// we can ensure all components of body is completed(body, receipts,
	// tx indexes)
	if batch.ValueSize() > 0 {
		bytes += batch.ValueSize()
		if err := batch.Write(); err != nil {
			return 0, err
		}
	}

	// Update the head fast sync block if better
	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	head := blockChain[len(blockChain)-1]
	if td := bc.GetTd(head.Hash(), head.NumberU64()); td != nil { // Rewind may have occurred, skip in that case
		currentFastBlock := bc.CurrentFastBlock()
		if bc.GetTd(currentFastBlock.Hash(), currentFastBlock.NumberU64()).Cmp(td) < 0 {
			rawdb.WriteHeadFastBlockHash(bc.db, head.Hash())
			bc.currentFastBlock.Store(head)
			headFastBlockGauge.Update(int64(head.NumberU64()))
		}
	}
	bc.chainmu.Unlock()

	log.Info("Imported new block receipts",
		"count", stats.processed,
		"elapsed", common.PrettyDuration(time.Since(start)),
		"number", head.Number(),
		"hash", head.Hash(),
		"size", common.StorageSize(bytes),
		"ignored", stats.ignored)
	return 0, nil
}

var lastWrite uint64

// writeBlockWithoutState writes only the block and its metadata to the database,
// but does not write any state. This is used to construct competing side forks
// up to the point where they exceed the canonical total difficulty.
func (bc *BlockChain) writeBlockWithoutState(block *types.Block, td *big.Int) (err error) {
	if bc.insertStopped() {
		return errInsertionInterrupted
	}

	batch := bc.db.NewBatch()
	rawdb.WriteTd(batch, block.Hash(), block.NumberU64(), td)
	rawdb.WriteBlock(batch, block)
	if err := batch.Write(); err != nil {
		log.Crit("Failed to write block into disk", "err", err)
	}
	return nil
}

// WriteBlockWithState writes the block and all associated state to the database.
func (bc *BlockChain) WriteBlockWithState(block *types.Block, receipts []*types.Receipt, state *state.StateDB, tradingState *tradingstate.TradingStateDB, lendingState *lendingstate.LendingStateDB) (status WriteStatus, err error) {
	if !bc.chainmu.TryLock() {
		return NonStatTy, errInsertionInterrupted
	}
	defer bc.chainmu.Unlock()
	return bc.writeBlockWithState(block, receipts, state, tradingState, lendingState)
}

// writeBlockWithState writes the block and all associated state to the database,
// but is expects the chain mutex to be held.
func (bc *BlockChain) writeBlockWithState(block *types.Block, receipts []*types.Receipt, state *state.StateDB, tradingState *tradingstate.TradingStateDB, lendingState *lendingstate.LendingStateDB) (status WriteStatus, err error) {
	if bc.insertStopped() {
		return NonStatTy, errInsertionInterrupted
	}

	// Calculate the total difficulty of the block
	ptd := bc.GetTd(block.ParentHash(), block.NumberU64()-1)
	if ptd == nil {
		return NonStatTy, consensus.ErrUnknownAncestor
	}
	// Make sure no inconsistent state is leaked during insertion
	currentBlock := bc.CurrentBlock()
	localTd := bc.GetTd(currentBlock.Hash(), currentBlock.NumberU64())
	externTd := new(big.Int).Add(block.Difficulty(), ptd)

	// Irrelevant of the canonical status, write the block itself to the database.
	//
	// Note all the components of block(td, hash->number map, header, body, receipts)
	// should be written atomically. BlockBatch is used for containing all components.
	blockBatch := bc.db.NewBatch()
	rawdb.WriteTd(blockBatch, block.Hash(), block.NumberU64(), externTd)
	rawdb.WriteBlock(blockBatch, block)
	rawdb.WriteReceipts(blockBatch, block.Hash(), block.NumberU64(), receipts)
	rawdb.WritePreimages(blockBatch, state.Preimages())
	if err := blockBatch.Write(); err != nil {
		log.Crit("Failed to write block into disk", "err", err)
	}
	// Commit all cached state changes into underlying memory database.
	root, err := state.Commit(bc.chainConfig.IsEIP158(block.Number()))
	if err != nil {
		return NonStatTy, err
	}
	triedb := bc.stateCache.TrieDB()

	tradingRoot := common.Hash{}
	if tradingState != nil {
		tradingRoot, err = tradingState.Commit()
		if err != nil {
			return NonStatTy, err
		}
	}
	lendingRoot := common.Hash{}
	if lendingState != nil {
		lendingRoot, err = lendingState.Commit()
		if err != nil {
			return NonStatTy, err
		}
	}

	engine, _ := bc.Engine().(*XDPoS.XDPoS)
	var tradingTrieDb *trie.Database
	var tradingService utils.TradingService
	var lendingTrieDb *trie.Database
	var lendingService utils.LendingService
	if bc.Config().IsTIPXDCXReceiver(block.Number()) && bc.chainConfig.XDPoS != nil && block.NumberU64() > bc.chainConfig.XDPoS.Epoch && engine != nil {
		tradingService = engine.GetXDCXService()
		if tradingService != nil {
			tradingTrieDb = tradingService.GetStateCache().TrieDB()
		}
		lendingService = engine.GetLendingService()
		if lendingService != nil {
			lendingTrieDb = lendingService.GetStateCache().TrieDB()
		}
	}

	// If we're running an archive node, always flush
	if bc.cacheConfig.Disabled {
		if err := triedb.Commit(root, false); err != nil {
			return NonStatTy, err
		}
		if tradingTrieDb != nil {
			if err := tradingTrieDb.Commit(tradingRoot, false); err != nil {
				return NonStatTy, err
			}
		}
		if lendingTrieDb != nil {
			if err := lendingTrieDb.Commit(lendingRoot, false); err != nil {
				return NonStatTy, err
			}
		}
	} else {
		// Full but not archive node, do proper garbage collection
		triedb.Reference(root, common.Hash{}) // metadata reference to keep trie alive
		bc.triegc.Push(root, -int64(block.NumberU64()))
		if tradingTrieDb != nil {
			tradingTrieDb.Reference(tradingRoot, common.Hash{})
		}
		if tradingService != nil {
			tradingService.GetTriegc().Push(tradingRoot, -int64(block.NumberU64()))
		}
		if lendingTrieDb != nil {
			lendingTrieDb.Reference(lendingRoot, common.Hash{})
		}
		if lendingService != nil {
			lendingService.GetTriegc().Push(lendingRoot, -int64(block.NumberU64()))
		}
		if current := block.NumberU64(); current > triesInMemory {
			// Find the next state trie we need to commit
			chosen := current - triesInMemory
			// Only write to disk if we exceeded our memory allowance *and* also have at
			// least a given number of tries gapped.
			//
			//if tradingTrieDb != nil {
			//	size = size + tradingTrieDb.Size()
			//}
			//if lendingTrieDb != nil {
			//	size = size + lendingTrieDb.Size()
			//}
			var (
				nodes, imgs = triedb.Size()
				limit       = common.StorageSize(bc.cacheConfig.TrieNodeLimit) * 1024 * 1024
			)
			if nodes > limit || imgs > 4*1024*1024 {
				triedb.Cap(limit - ethdb.IdealBatchSize)
			}
			if bc.gcproc > bc.cacheConfig.TrieTimeLimit || chosen > lastWrite+triesInMemory {
				// If the header is missing (canonical chain behind), we're reorging a low
				// diff sidechain. Suspend committing until this operation is completed.
				header := bc.GetHeaderByNumber(chosen)
				if header == nil {
					log.Warn("Reorg in progress, trie commit postponed", "number", chosen)
				} else {
					// If we're exceeding limits but haven't reached a large enough memory gap,
					// warn the user that the system is becoming unstable.
					if chosen < lastWrite+triesInMemory && bc.gcproc >= 2*bc.cacheConfig.TrieTimeLimit {
						log.Info("State in memory for too long, committing", "time", bc.gcproc, "allowance", bc.cacheConfig.TrieTimeLimit, "optimum", float64(chosen-lastWrite)/triesInMemory)
					}
					// Flush an entire trie and restart the counters
					triedb.Commit(header.Root, true)
					lastWrite = chosen
					bc.gcproc = 0
					if tradingTrieDb != nil && lendingTrieDb != nil {
						b := bc.GetBlock(header.Hash(), current-triesInMemory)
						author, _ := bc.Engine().Author(b.Header())
						oldTradingRoot, _ := tradingService.GetTradingStateRoot(b, author)
						oldLendingRoot, _ := lendingService.GetLendingStateRoot(b, author)
						tradingTrieDb.Commit(oldTradingRoot, true)
						lendingTrieDb.Commit(oldLendingRoot, true)
					}
				}
			}
			// Garbage collect anything below our required write retention
			for !bc.triegc.Empty() {
				root, number := bc.triegc.Pop()
				if uint64(-number) > chosen {
					bc.triegc.Push(root, number)
					break
				}
				triedb.Dereference(root)
			}
			if tradingService != nil {
				for !tradingService.GetTriegc().Empty() {
					tradingRoot, number := tradingService.GetTriegc().Pop()
					if uint64(-number) > chosen {
						tradingService.GetTriegc().Push(tradingRoot, number)
						break
					}
					tradingTrieDb.Dereference(tradingRoot)
				}
			}
			if lendingService != nil {
				for !lendingService.GetTriegc().Empty() {
					lendingRoot, number := lendingService.GetTriegc().Pop()
					if uint64(-number) > chosen {
						lendingService.GetTriegc().Push(lendingRoot, number)
						break
					}
					lendingTrieDb.Dereference(lendingRoot)
				}
			}
		}
	}

	// If the total difficulty is higher than our known, add it to the canonical chain
	// Second clause in the if statement reduces the vulnerability to selfish mining.
	// Please refer to http://www.cs.cornell.edu/~ie53/publications/btcProcFC.pdf
	reorg := externTd.Cmp(localTd) > 0
	currentBlock = bc.CurrentBlock()
	if !reorg && externTd.Cmp(localTd) == 0 {
		// Split same-difficulty blocks by number
		reorg = block.NumberU64() > currentBlock.NumberU64()
	}
	if reorg {
		// Reorganise the chain if the parent is not the head block
		if block.ParentHash() != currentBlock.Hash() {
			if err := bc.reorg(currentBlock.Header(), block.Header()); err != nil {
				return NonStatTy, err
			}
		}
		status = CanonStatTy
	} else {
		status = SideStatTy
	}

	// Set new head.
	if status == CanonStatTy {
		// WriteBlock has already been called, no need to write again
		bc.writeHeadBlock(block, false)
		// prepare set of masternodes for the next epoch
		if bc.chainConfig.XDPoS != nil && ((block.NumberU64() % bc.chainConfig.XDPoS.Epoch) == (bc.chainConfig.XDPoS.Epoch - bc.chainConfig.XDPoS.Gap)) {
			if err := bc.UpdateM1(); err != nil {
				log.Crit("Fail to update masternodes during writeBlockWithState", "number", block.Number, "hash", block.Hash().Hex(), "err", err)
			}
		}
	}
	// save cache BlockSigners
	if bc.chainConfig.XDPoS != nil && bc.chainConfig.IsTIPSigning(block.Number()) {
		engine, ok := bc.Engine().(*XDPoS.XDPoS)
		if ok {
			engine.CacheSigningTxs(block.Header().Hash(), block.Transactions())
		}
	}
	bc.futureBlocks.Remove(block.Hash())
	return status, nil
}

// InsertChain attempts to insert the given batch of blocks in to the canonical
// chain or, otherwise, create a fork. If an error is returned it will return
// the index number of the failing block as well an error describing what went
// wrong.
//
// After insertion is done, all accumulated events will be fired.
func (bc *BlockChain) InsertChain(chain types.Blocks) (int, error) {
	// Sanity check that we have something meaningful to import
	if len(chain) == 0 {
		return 0, nil
	}

	// Do a sanity check that the provided chain is actually ordered and linked
	for i := 1; i < len(chain); i++ {
		block, prev := chain[i], chain[i-1]
		if block.NumberU64() != chain[i-1].NumberU64()+1 || block.ParentHash() != chain[i-1].Hash() {
			// Chain broke ancestry, log a messge (programming error) and skip insertion
			log.Error("Non contiguous block insert",
				"number", block.Number(),
				"hash", block.Hash(),
				"parent", block.ParentHash(),
				"prevnumber", prev.Number(),
				"prevhash", prev.Hash())

			return 0, fmt.Errorf("non contiguous insert: item %d is #%d [%x..], item %d is #%d [%x..] (parent [%x..])", i-1, prev.NumberU64(),
				prev.Hash().Bytes()[:4], i, block.NumberU64(), block.Hash().Bytes()[:4], block.ParentHash().Bytes()[:4])
		}
	}

	// Pre-check passed, start the full block imports.
	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()
	n, events, logs, err := bc.insertChain(chain, true)
	bc.PostChainEvents(events, logs)
	return n, err
}

// insertChain is the internal implementation of InsertChain, which assumes that
// 1) chains are contiguous, and 2) The chain mutex is held.
//
// This method is split out so that import batches that require re-injecting
// historical blocks can do so without releasing the lock, which could lead to
// racey behaviour. If a sidechain import is in progress, and the historic state
// is imported, but then new canon-head is added before the actual sidechain
// completes, then the historic state could be pruned again
func (bc *BlockChain) insertChain(chain types.Blocks, verifySeals bool) (int, []interface{}, []*types.Log, error) {
	// If the chain is terminating, don't even bother starting up.
	if bc.insertStopped() {
		return 0, nil, nil, nil
	}

	// A queued approach to delivering events. This is generally
	// faster than direct delivery and requires much less mutex
	// acquiring.
	var (
		stats         = insertStats{startTime: mclock.Now()}
		events        = make([]interface{}, 0, len(chain))
		lastCanon     *types.Block
		coalescedLogs []*types.Log
	)
	// Start the parallel header verifier
	headers := make([]*types.Header, len(chain))
	seals := make([]bool, len(chain))

	for i, block := range chain {
		headers[i] = block.Header()
		seals[i] = verifySeals
		bc.downloadingBlock.Add(block.Hash(), struct{}{})
	}
	abort, results := bc.engine.VerifyHeaders(bc, headers, seals)
	defer close(abort)

	// Start a parallel signature recovery (signer will fluke on fork transition, minimal perf loss)
	SenderCacher.RecoverFromBlocks(types.MakeSigner(bc.chainConfig, chain[0].Number()), chain)

	// Iterate over the blocks and insert when the verifier permits
	for i, block := range chain {
		// If the chain is terminating, stop processing blocks
		if atomic.LoadInt32(&bc.procInterrupt) == 1 {
			log.Debug("Premature abort during blocks processing")
			break
		}
		// If the header is a banned one, straight out abort
		if BadHashes[block.Hash()] {
			bc.reportBlock(block, nil, ErrBlacklistedHash)
			return i, events, coalescedLogs, ErrBlacklistedHash
		}
		// Wait for the block's verification to complete
		bstart := time.Now()

		err := <-results
		if err == nil {
			err = bc.Validator().ValidateBody(block)
		}
		switch {
		case err == ErrKnownBlock:
			// Block and state both already known. However if the current block is below
			// this number we did a rollback and we should reimport it nonetheless.
			if bc.CurrentBlock().NumberU64() >= block.NumberU64() {
				stats.ignored++
				continue
			}

		case err == consensus.ErrFutureBlock:
			// Allow up to MaxFuture second in the future blocks. If this limit is exceeded
			// the chain is discarded and processed at a later time if given.
			max := big.NewInt(time.Now().Unix() + maxTimeFutureBlocks)
			if block.Time().Cmp(max) > 0 {
				return i, events, coalescedLogs, fmt.Errorf("future block: %v > %v", block.Time(), max)
			}
			bc.futureBlocks.Add(block.Hash(), block)
			stats.queued++
			continue

		case err == consensus.ErrUnknownAncestor && bc.futureBlocks.Contains(block.ParentHash()):
			bc.futureBlocks.Add(block.Hash(), block)
			stats.queued++
			continue

		case err == consensus.ErrPrunedAncestor:
			// Block competing with the canonical chain, store in the db, but don't process
			// until the competitor TD goes above the canonical TD
			currentBlock := bc.CurrentBlock()
			localTd := bc.GetTd(currentBlock.Hash(), currentBlock.NumberU64())
			externTd := new(big.Int).Add(bc.GetTd(block.ParentHash(), block.NumberU64()-1), block.Difficulty())
			if localTd.Cmp(externTd) > 0 {
				if err = bc.writeBlockWithoutState(block, externTd); err != nil {
					return i, events, coalescedLogs, err
				}
				continue
			}
			// Competitor chain beat canonical, gather all blocks from the common ancestor
			var winner []*types.Block

			parent := bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
			for !bc.HasFullState(parent) {
				winner = append(winner, parent)
				parent = bc.GetBlock(parent.ParentHash(), parent.NumberU64()-1)
			}
			for j := 0; j < len(winner)/2; j++ {
				winner[j], winner[len(winner)-1-j] = winner[len(winner)-1-j], winner[j]
			}
			log.Debug("Number block need calculated again", "number", block.NumberU64(), "hash", block.Hash().Hex(), "winners", len(winner))
			// Import all the pruned blocks to make the state available
			// During reorg, we use verifySeals=false
			_, evs, logs, err := bc.insertChain(winner, false)
			events, coalescedLogs = evs, logs

			if err != nil {
				return i, events, coalescedLogs, err
			}

		case err != nil:
			bc.reportBlock(block, nil, err)
			return i, events, coalescedLogs, err
		}
		// Create a new statedb using the parent block and report an
		// error if it fails.
		var parent *types.Block
		if i == 0 {
			parent = bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
		} else {
			parent = chain[i-1]
		}
		statedb, err := state.New(parent.Root(), bc.stateCache)
		if err != nil {
			return i, events, coalescedLogs, err
		}
		// clear the previous dry-run cache
		var tradingState *tradingstate.TradingStateDB
		var lendingState *lendingstate.LendingStateDB
		var tradingService utils.TradingService
		var lendingService utils.LendingService
		isSDKNode := false
		engine, _ := bc.Engine().(*XDPoS.XDPoS)
		if bc.Config().IsTIPXDCXReceiver(block.Number()) && bc.chainConfig.XDPoS != nil && engine != nil && block.NumberU64() > bc.chainConfig.XDPoS.Epoch {
			author, err := bc.Engine().Author(block.Header()) // Ignore error, we're past header validation
			if err != nil {
				bc.reportBlock(block, nil, err)
				return i, events, coalescedLogs, err
			}
			parentAuthor, _ := bc.Engine().Author(parent.Header())
			tradingService = engine.GetXDCXService()
			lendingService = engine.GetLendingService()
			if tradingService != nil && lendingService != nil {
				isSDKNode = tradingService.IsSDKNode()
				txMatchBatchData, err := ExtractTradingTransactions(block.Transactions())
				if err != nil {
					bc.reportBlock(block, nil, err)
					return i, events, coalescedLogs, err
				}
				tradingState, err = tradingService.GetTradingState(parent, parentAuthor)
				if err != nil {
					bc.reportBlock(block, nil, err)
					return i, events, coalescedLogs, err
				}
				lendingState, err = lendingService.GetLendingState(parent, parentAuthor)
				if err != nil {
					bc.reportBlock(block, nil, err)
					return i, events, coalescedLogs, err
				}
				isEpochSwithBlock, epochNumber, err := engine.IsEpochSwitch(block.Header())
				if err != nil {
					log.Error("[insertChain] Error while checking if the incoming block is epoch switch block", "Hash", block.Hash(), "Number", block.Number())
					bc.reportBlock(block, nil, err)
				}
				if isEpochSwithBlock {
					if err := tradingService.UpdateMediumPriceBeforeEpoch(epochNumber, tradingState, statedb); err != nil {
						return i, events, coalescedLogs, err
					}
				} else {
					for _, txMatchBatch := range txMatchBatchData {
						log.Debug("Verify matching transaction", "txHash", txMatchBatch.TxHash.Hex())
						err := bc.Validator().ValidateTradingOrder(statedb, tradingState, txMatchBatch, author, block.Header())
						if err != nil {
							bc.reportBlock(block, nil, err)
							return i, events, coalescedLogs, err
						}
					}
					//
					batches, err := ExtractLendingTransactions(block.Transactions())
					if err != nil {
						bc.reportBlock(block, nil, err)
						return i, events, coalescedLogs, err
					}
					for _, batch := range batches {
						log.Debug("Verify matching transaction", "txHash", batch.TxHash.Hex())
						err := bc.Validator().ValidateLendingOrder(statedb, lendingState, tradingState, batch, author, block.Header())
						if err != nil {
							bc.reportBlock(block, nil, err)
							return i, events, coalescedLogs, err
						}
					}
					// liquidate / finalize open lendingTrades
					if block.Number().Uint64()%bc.chainConfig.XDPoS.Epoch == common.LiquidateLendingTradeBlock {
						finalizedTrades, _, _, _, _, err := lendingService.ProcessLiquidationData(block.Header(), bc, statedb, tradingState, lendingState)
						if err != nil {
							return i, events, coalescedLogs, fmt.Errorf("failed to ProcessLiquidationData. Err: %v", err)
						}
						if isSDKNode {
							finalizedTx := lendingstate.FinalizedResult{}
							if finalizedTx, err = ExtractLendingFinalizedTradeTransactions(block.Transactions()); err != nil {
								return i, events, coalescedLogs, err
							}
							bc.AddFinalizedTrades(finalizedTx.TxHash, finalizedTrades)
						}
					}
				}
				//check
				if tradingState != nil {
					gotRoot := tradingState.IntermediateRoot()
					expectRoot, _ := tradingService.GetTradingStateRoot(block, author)
					parentRoot, _ := tradingService.GetTradingStateRoot(parent, parentAuthor)
					if gotRoot != expectRoot {
						err = fmt.Errorf("invalid XDCx trading state merke trie got : %s , expect : %s ,parent : %s", gotRoot.Hex(), expectRoot.Hex(), parentRoot.Hex())
						bc.reportBlock(block, nil, err)
						return i, events, coalescedLogs, err
					}
					log.Debug("XDCX Trading State Root", "number", block.NumberU64(), "parent", parentRoot.Hex(), "nextRoot", expectRoot.Hex())
				}
				if lendingState != nil && tradingState != nil {
					gotRoot := lendingState.IntermediateRoot()
					expectRoot, _ := lendingService.GetLendingStateRoot(block, author)
					parentRoot, _ := lendingService.GetLendingStateRoot(parent, parentAuthor)
					if gotRoot != expectRoot {
						err = fmt.Errorf("invalid lending state merke trie got: %s, expect: %s, parent: %s", gotRoot.Hex(), expectRoot.Hex(), parentRoot.Hex())
						bc.reportBlock(block, nil, err)
						return i, events, coalescedLogs, err
					}
					log.Debug("XDCX Lending State Root", "number", block.NumberU64(), "parent", parentRoot.Hex(), "nextRoot", expectRoot.Hex())
				}
			}
		}
		feeCapacity := state.GetTRC21FeeCapacityFromStateWithCache(parent.Root(), statedb)
		// Process block using the parent state as reference point.
		t0 := time.Now()
		receipts, logs, usedGas, err := bc.processor.Process(block, statedb, tradingState, bc.vmConfig, feeCapacity)
		t1 := time.Now()
		if err != nil {
			bc.reportBlock(block, receipts, err)
			return i, events, coalescedLogs, err
		}
		// Validate the state using the default validator
		err = bc.Validator().ValidateState(block, parent, statedb, receipts, usedGas)
		if err != nil {
			bc.reportBlock(block, receipts, err)
			return i, events, coalescedLogs, err
		}
		t2 := time.Now()
		proctime := time.Since(bstart)

		// Write the block to the chain and get the status.
		status, err := bc.writeBlockWithState(block, receipts, statedb, tradingState, lendingState)
		t3 := time.Now()
		if err != nil {
			return i, events, coalescedLogs, err
		}

		// Update the metrics subsystem with all the measurements
		accountReadTimer.Update(statedb.AccountReads)
		accountHashTimer.Update(statedb.AccountHashes)
		accountUpdateTimer.Update(statedb.AccountUpdates)
		accountCommitTimer.Update(statedb.AccountCommits)

		storageReadTimer.Update(statedb.StorageReads)
		storageHashTimer.Update(statedb.StorageHashes)
		storageUpdateTimer.Update(statedb.StorageUpdates)
		storageCommitTimer.Update(statedb.StorageCommits)

		trieAccess := statedb.AccountReads + statedb.AccountHashes + statedb.AccountUpdates + statedb.AccountCommits
		trieAccess += statedb.StorageReads + statedb.StorageHashes + statedb.StorageUpdates + statedb.StorageCommits

		blockInsertTimer.UpdateSince(bstart)
		blockExecutionTimer.Update(t1.Sub(t0) - trieAccess)
		blockValidationTimer.Update(t2.Sub(t1))
		blockWriteTimer.Update(t3.Sub(t2))

		switch status {
		case CanonStatTy:
			log.Debug("Inserted new block from downloader", "number", block.Number(), "hash", block.Hash(), "uncles", len(block.Uncles()),
				"txs", len(block.Transactions()), "gas", block.GasUsed(), "elapsed", common.PrettyDuration(time.Since(bstart)))

			coalescedLogs = append(coalescedLogs, logs...)
			events = append(events, ChainEvent{block, block.Hash(), logs})
			lastCanon = block

			// Only count canonical blocks for GC processing time
			bc.gcproc += proctime
			bc.UpdateBlocksHashCache(block)
			if bc.chainConfig.IsTIPXDCX(block.Number()) && bc.chainConfig.XDPoS != nil && block.NumberU64() > bc.chainConfig.XDPoS.Epoch {
				bc.logExchangeData(block)
				bc.logLendingData(block)
			}
		case SideStatTy:
			log.Debug("Inserted forked block from downloader", "number", block.Number(), "hash", block.Hash(), "diff", block.Difficulty(), "elapsed",
				common.PrettyDuration(time.Since(bstart)), "txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()))
			events = append(events, ChainSideEvent{block})
			bc.UpdateBlocksHashCache(block)
		}
		stats.processed++
		stats.usedGas += usedGas
		dirty, _ := bc.stateCache.TrieDB().Size()
		stats.report(chain, i, dirty)
		if bc.chainConfig.XDPoS != nil {
			// epoch block
			isEpochSwithBlock, _, err := engine.IsEpochSwitch(chain[i].Header())
			if err != nil {
				log.Error("[insertChain] Error while checking and notifying channel CheckpointCh if the incoming block is epoch switch block", "Hash", block.Hash(), "Number", block.Number())
				bc.reportBlock(block, nil, err)
			}
			if isEpochSwithBlock {
				CheckpointCh <- 1
			}
		}
	}
	// Append a single chain head event if we've progressed the chain
	if lastCanon != nil && bc.CurrentBlock().Hash() == lastCanon.Hash() {
		log.Debug("New ChainHeadEvent ", "number", lastCanon.NumberU64(), "hash", lastCanon.Hash())
		events = append(events, ChainHeadEvent{lastCanon})
	}
	return 0, events, coalescedLogs, nil
}

func (bc *BlockChain) InsertBlock(block *types.Block) error {
	events, logs, err := bc.insertBlock(block)
	bc.PostChainEvents(events, logs)
	return err
}

func (bc *BlockChain) PrepareBlock(block *types.Block) (err error) {
	defer log.Debug("Done prepare block ", "number", block.NumberU64(), "hash", block.Hash(), "validator", block.Header().Validator, "err", err)
	if _, check := bc.resultProcess.Get(block.Hash()); check {
		log.Debug("Stop prepare a block because the result cached", "number", block.NumberU64(), "hash", block.Hash(), "validator", block.Header().Validator)
		return nil
	}
	if _, check := bc.calculatingBlock.Get(block.Hash()); check {
		log.Debug("Stop prepare a block because inserting", "number", block.NumberU64(), "hash", block.Hash(), "validator", block.Header().Validator)
		return nil
	}
	err = bc.engine.VerifyHeader(bc, block.Header(), false)
	if err != nil {
		return err
	}
	result, err := bc.getResultBlock(block, false)
	if err == nil {
		bc.resultProcess.Add(block.Hash(), result)
		return nil
	} else if err == ErrKnownBlock {
		return nil
	} else if err == ErrStopPreparingBlock {
		log.Debug("Stop prepare a block because calculating", "number", block.NumberU64(), "hash", block.Hash(), "validator", block.Header().Validator)
		return nil
	}
	return err
}

func (bc *BlockChain) getResultBlock(block *types.Block, verifiedM2 bool) (*ResultProcessBlock, error) {
	var calculatedBlock *CalculatedBlock
	if verifiedM2 {
		if result, check := bc.resultProcess.Get(block.HashNoValidator()); check {
			log.Debug("Get result block from cache ", "number", block.NumberU64(), "hash", block.Hash(), "hash no validator", block.HashNoValidator())
			return result, nil
		}
		log.Debug("Not found cache prepare block ", "number", block.NumberU64(), "hash", block.Hash(), "validator", block.HashNoValidator())
		if calculatedBlock, _ := bc.calculatingBlock.Get(block.HashNoValidator()); calculatedBlock != nil {
			calculatedBlock.stop = true
		}
	}
	calculatedBlock = &CalculatedBlock{block, false}
	bc.calculatingBlock.Add(block.HashNoValidator(), calculatedBlock)
	// Start the parallel header verifier
	// If the chain is terminating, stop processing blocks
	if atomic.LoadInt32(&bc.procInterrupt) == 1 {
		log.Debug("Premature abort during blocks processing")
		return nil, ErrBlacklistedHash
	}
	// If the header is a banned one, straight out abort
	if BadHashes[block.Hash()] {
		bc.reportBlock(block, nil, ErrBlacklistedHash)
		return nil, ErrBlacklistedHash
	}
	// Wait for the block's verification to complete
	bstart := time.Now()
	err := bc.Validator().ValidateBody(block)
	switch {
	case err == ErrKnownBlock:
		// Block and state both already known. However if the current block is below
		// this number we did a rollback and we should reimport it nonetheless.
		if bc.CurrentBlock().NumberU64() >= block.NumberU64() {
			return nil, ErrKnownBlock
		}
	case err == consensus.ErrPrunedAncestor:
		// Block competing with the canonical chain, store in the db, but don't process
		// until the competitor TD goes above the canonical TD
		currentBlock := bc.CurrentBlock()
		localTd := bc.GetTd(currentBlock.Hash(), currentBlock.NumberU64())
		externTd := new(big.Int).Add(bc.GetTd(block.ParentHash(), block.NumberU64()-1), block.Difficulty())
		if localTd.Cmp(externTd) > 0 {
			return nil, err
		}
		// Competitor chain beat canonical, gather all blocks from the common ancestor
		var winner []*types.Block

		parent := bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
		for !bc.HasFullState(parent) {
			winner = append(winner, parent)
			parent = bc.GetBlock(parent.ParentHash(), parent.NumberU64()-1)
		}
		for j := 0; j < len(winner)/2; j++ {
			winner[j], winner[len(winner)-1-j] = winner[len(winner)-1-j], winner[j]
		}
		log.Debug("Number block need calculated again", "number", block.NumberU64(), "hash", block.Hash().Hex(), "winners", len(winner))
		// Import all the pruned blocks to make the state available
		// During reorg, we use verifySeals=false
		_, _, _, err := bc.insertChain(winner, false)
		if err != nil {
			return nil, err
		}
	case err != nil:
		bc.reportBlock(block, nil, err)
		return nil, err
	}
	// Create a new statedb using the parent block and report an
	// error if it fails.
	var parent = bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
	statedb, err := state.New(parent.Root(), bc.stateCache)
	if err != nil {
		return nil, err
	}
	engine, _ := bc.Engine().(*XDPoS.XDPoS)
	author, err := bc.Engine().Author(block.Header()) // Ignore error, we're past header validation
	if err != nil {
		bc.reportBlock(block, nil, err)
		return nil, err
	}
	parentAuthor, _ := bc.Engine().Author(parent.Header())

	var tradingState *tradingstate.TradingStateDB
	var lendingState *lendingstate.LendingStateDB
	var tradingService utils.TradingService
	var lendingService utils.LendingService
	isSDKNode := false
	if bc.Config().IsTIPXDCX(block.Number()) && bc.chainConfig.XDPoS != nil && engine != nil && block.NumberU64() > bc.chainConfig.XDPoS.Epoch {
		tradingService = engine.GetXDCXService()
		lendingService = engine.GetLendingService()
		if tradingService != nil && lendingService != nil {
			isSDKNode = tradingService.IsSDKNode()
			tradingState, err = tradingService.GetTradingState(parent, parentAuthor)
			if err != nil {
				bc.reportBlock(block, nil, err)
				return nil, err
			}
			lendingState, err = lendingService.GetLendingState(parent, parentAuthor)
			if err != nil {
				bc.reportBlock(block, nil, err)
				return nil, err
			}

			isEpochSwithBlock, epochNumber, err := engine.IsEpochSwitch(block.Header())
			if err != nil {
				log.Error("[getResultBlock] Error while checking block is epoch switch block", "Hash", block.Hash(), "Number", block.Number())
				bc.reportBlock(block, nil, err)
			}

			if isEpochSwithBlock {
				if err := tradingService.UpdateMediumPriceBeforeEpoch(epochNumber, tradingState, statedb); err != nil {
					return nil, err
				}
			} else {
				txMatchBatchData, err := ExtractTradingTransactions(block.Transactions())
				if err != nil {
					bc.reportBlock(block, nil, err)
					return nil, err
				}
				for _, txMatchBatch := range txMatchBatchData {
					log.Debug("Verify matching transaction", "txHash", txMatchBatch.TxHash.Hex())
					err := bc.Validator().ValidateTradingOrder(statedb, tradingState, txMatchBatch, author, block.Header())
					if err != nil {
						bc.reportBlock(block, nil, err)
						return nil, err
					}
				}
				batches, err := ExtractLendingTransactions(block.Transactions())
				if err != nil {
					bc.reportBlock(block, nil, err)
					return nil, err
				}
				for _, batch := range batches {
					log.Debug("Lending Verify matching transaction", "txHash", batch.TxHash.Hex())
					err := bc.Validator().ValidateLendingOrder(statedb, lendingState, tradingState, batch, author, block.Header())
					if err != nil {
						bc.reportBlock(block, nil, err)
						return nil, err
					}
				}
				// liquidate / finalize open lendingTrades
				if block.Number().Uint64()%bc.chainConfig.XDPoS.Epoch == common.LiquidateLendingTradeBlock {
					finalizedTrades, _, _, _, _, err := lendingService.ProcessLiquidationData(block.Header(), bc, statedb, tradingState, lendingState)
					if err != nil {
						return nil, fmt.Errorf("failed to ProcessLiquidationData. Err: %v", err)
					}
					if isSDKNode {
						finalizedTx := lendingstate.FinalizedResult{}
						if finalizedTx, err = ExtractLendingFinalizedTradeTransactions(block.Transactions()); err != nil {
							return nil, err
						}
						bc.AddFinalizedTrades(finalizedTx.TxHash, finalizedTrades)
					}
				}
			}
			if tradingState != nil {
				gotRoot := tradingState.IntermediateRoot()
				expectRoot, _ := tradingService.GetTradingStateRoot(block, author)
				parentRoot, _ := tradingService.GetTradingStateRoot(parent, parentAuthor)
				if gotRoot != expectRoot {
					err = fmt.Errorf("invalid XDCx trading state merke trie got : %s , expect : %s ,parent : %s", gotRoot.Hex(), expectRoot.Hex(), parentRoot.Hex())
					bc.reportBlock(block, nil, err)
					return nil, err
				}
				log.Debug("XDCX Trading State Root", "number", block.NumberU64(), "parent", parentRoot.Hex(), "nextRoot", expectRoot.Hex())
			}
			if lendingState != nil && tradingState != nil {
				gotRoot := lendingState.IntermediateRoot()
				expectRoot, _ := lendingService.GetLendingStateRoot(block, author)
				parentRoot, _ := lendingService.GetLendingStateRoot(parent, parentAuthor)
				if gotRoot != expectRoot {
					err = fmt.Errorf("invalid lending state merke trie got: %s , expect : %s , parent : %s", gotRoot.Hex(), expectRoot.Hex(), parentRoot.Hex())
					bc.reportBlock(block, nil, err)
					return nil, err
				}
				log.Debug("XDCX Lending State Root", "number", block.NumberU64(), "parent", parentRoot.Hex(), "nextRoot", expectRoot.Hex())
			}
		}
	}
	feeCapacity := state.GetTRC21FeeCapacityFromStateWithCache(parent.Root(), statedb)
	// Process block using the parent state as reference point.
	receipts, logs, usedGas, err := bc.processor.ProcessBlockNoValidator(calculatedBlock, statedb, tradingState, bc.vmConfig, feeCapacity)
	process := time.Since(bstart)
	if err != nil {
		if err != ErrStopPreparingBlock {
			bc.reportBlock(block, receipts, err)
		}
		return nil, err
	}
	// Validate the state using the default validator
	err = bc.Validator().ValidateState(block, parent, statedb, receipts, usedGas)
	if err != nil {
		bc.reportBlock(block, receipts, err)
		return nil, err
	}
	proctime := time.Since(bstart)
	log.Debug("Calculate new block", "number", block.Number(), "hash", block.Hash(), "uncles", len(block.Uncles()),
		"txs", len(block.Transactions()), "gas", block.GasUsed(), "elapsed", common.PrettyDuration(time.Since(bstart)), "process", process)
	return &ResultProcessBlock{receipts: receipts, logs: logs, state: statedb, tradingState: tradingState, lendingState: lendingState, proctime: proctime, usedGas: usedGas}, nil
}

// UpdateBlocksHashCache update BlocksHashCache by block number
func (bc *BlockChain) UpdateBlocksHashCache(block *types.Block) []common.Hash {
	var hashArr []common.Hash
	blockNumber := block.Number().Uint64()
	cached, ok := bc.blocksHashCache.Get(blockNumber)

	if ok {
		hashArr := cached
		hashArr = append(hashArr, block.Hash())
		bc.blocksHashCache.Remove(blockNumber)
		bc.blocksHashCache.Add(blockNumber, hashArr)
		return hashArr
	}

	hashArr = []common.Hash{
		block.Hash(),
	}
	bc.blocksHashCache.Add(blockNumber, hashArr)
	return hashArr
}

// insertChain will execute the actual chain insertion and event aggregation. The
// only reason this method exists as a separate one is to make locking cleaner
// with deferred statements.
func (bc *BlockChain) insertBlock(block *types.Block) ([]interface{}, []*types.Log, error) {
	var (
		stats         = insertStats{startTime: mclock.Now()}
		events        = make([]interface{}, 0, 1)
		coalescedLogs []*types.Log
	)
	if _, check := bc.downloadingBlock.Get(block.Hash()); check {
		log.Debug("Stop fetcher a block because downloading", "number", block.NumberU64(), "hash", block.Hash())
		return events, coalescedLogs, nil
	}
	result, err := bc.getResultBlock(block, true)
	if err != nil {
		return events, coalescedLogs, err
	}
	defer bc.resultProcess.Remove(block.HashNoValidator())
	bc.wg.Add(1)
	defer bc.wg.Done()
	// Write the block to the chain and get the status.
	if !bc.chainmu.TryLock() {
		return nil, nil, errChainStopped
	}
	defer bc.chainmu.Unlock()
	if bc.HasBlockAndFullState(block.Hash(), block.NumberU64()) {
		return events, coalescedLogs, nil
	}
	status, err := bc.writeBlockWithState(block, result.receipts, result.state, result.tradingState, result.lendingState)

	if err != nil {
		return events, coalescedLogs, err
	}
	switch status {
	case CanonStatTy:
		log.Debug("Inserted new block from fetcher", "number", block.Number(), "hash", block.Hash(), "uncles", len(block.Uncles()),
			"txs", len(block.Transactions()), "gas", block.GasUsed(), "elapsed", common.PrettyDuration(time.Since(block.ReceivedAt)))
		coalescedLogs = append(coalescedLogs, result.logs...)
		events = append(events, ChainEvent{block, block.Hash(), result.logs})
		// Only count canonical blocks for GC processing time
		bc.gcproc += result.proctime
		bc.UpdateBlocksHashCache(block)
		if bc.chainConfig.IsTIPXDCXReceiver(block.Number()) && bc.chainConfig.XDPoS != nil && block.NumberU64() > bc.chainConfig.XDPoS.Epoch {
			bc.logExchangeData(block)
			bc.logLendingData(block)
		}
	case SideStatTy:
		log.Debug("Inserted forked block from fetcher", "number", block.Number(), "hash", block.Hash(), "diff", block.Difficulty(), "elapsed",
			common.PrettyDuration(time.Since(block.ReceivedAt)), "txs", len(block.Transactions()), "gas", block.GasUsed(), "uncles", len(block.Uncles()))
		blockInsertTimer.Update(result.proctime)
		events = append(events, ChainSideEvent{block})
		bc.UpdateBlocksHashCache(block)
	}
	stats.processed++
	stats.usedGas += result.usedGas
	dirty, _ := bc.stateCache.TrieDB().Size()
	stats.report(types.Blocks{block}, 0, dirty)
	if bc.chainConfig.XDPoS != nil {
		// epoch block
		isEpochSwithBlock, _, err := bc.Engine().(*XDPoS.XDPoS).IsEpochSwitch(block.Header())
		if err != nil {
			log.Error("[insertBlock] Error while checking if the incoming block is epoch switch block", "Hash", block.Hash(), "Number", block.Number())
			bc.reportBlock(block, nil, err)
		}
		if isEpochSwithBlock {
			CheckpointCh <- 1
		}
	}
	// Append a single chain head event if we've progressed the chain
	if status == CanonStatTy && bc.CurrentBlock().Hash() == block.Hash() {
		events = append(events, ChainHeadEvent{block})
		log.Debug("New ChainHeadEvent from fetcher ", "number", block.NumberU64(), "hash", block.Hash())
	}
	return events, coalescedLogs, nil
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
func (st *insertStats) report(chain []*types.Block, index int, dirty common.StorageSize) {
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
			"number", end.Number(), "hash", end.Hash(), "dirty", dirty,
		}
		if st.queued > 0 {
			context = append(context, []interface{}{"queued", st.queued}...)
		}
		if st.ignored > 0 {
			context = append(context, []interface{}{"ignored", st.ignored}...)
		}
		log.Info("Imported new chain segment", context...)
		*st = insertStats{startTime: now, lastIndex: index + 1}
	}
}

func countTransactions(chain []*types.Block) (c int) {
	for _, b := range chain {
		c += len(b.Transactions())
	}
	return c
}

// collectLogs collects the logs that were generated or removed during
// the processing of a block. These logs are later announced as deleted or reborn.
func (bc *BlockChain) collectLogs(b *types.Block, removed bool) []*types.Log {
	receipts := rawdb.ReadRawReceipts(bc.db, b.Hash(), b.NumberU64())
	if err := receipts.DeriveFields(bc.chainConfig, b.Hash(), b.NumberU64(), b.BaseFee(), b.Transactions()); err != nil {
		log.Error("Failed to derive block receipts fields", "hash", b.Hash(), "number", b.NumberU64(), "err", err)
	}

	var logs []*types.Log
	for _, receipt := range receipts {
		for _, log := range receipt.Logs {
			if removed {
				log.Removed = true
			}
			logs = append(logs, log)
		}
	}
	return logs
}

// reorg takes two blocks, an old chain and a new chain and will reconstruct the
// blocks and inserts them to be part of the new canonical chain and accumulates
// potential missing transactions and post an event about them.
func (bc *BlockChain) reorg(oldHead, newHead *types.Header) error {
	log.Warn("Reorg", "OldHash", oldHead.Hash().Hex(), "OldNum", oldHead.Number, "NewHash", newHead.Hash().Hex(), "NewNum", newHead.Number)

	var (
		newChain    []*types.Header
		oldChain    []*types.Header
		commonBlock *types.Header
	)

	// Reduce the longer chain to the same number as the shorter one
	if oldHead.Number.Uint64() > newHead.Number.Uint64() {
		// Old chain is longer, gather all transactions and logs as deleted ones
		for ; oldHead != nil && oldHead.Number.Uint64() != newHead.Number.Uint64(); oldHead = bc.GetHeader(oldHead.ParentHash, oldHead.Number.Uint64()-1) {
			oldChain = append(oldChain, oldHead)
		}
	} else {
		// New chain is longer, stash all blocks away for subsequent insertion
		for ; newHead != nil && newHead.Number.Uint64() != oldHead.Number.Uint64(); newHead = bc.GetHeader(newHead.ParentHash, newHead.Number.Uint64()-1) {
			newChain = append(newChain, newHead)
		}
	}
	if oldHead == nil {
		return errInvalidOldChain
	}
	if newHead == nil {
		return errInvalidNewChain
	}

	// Both sides of the reorg are at the same number, reduce both until the common
	// ancestor is found
	for {
		// If the common ancestor was found, bail out
		if oldHead.Hash() == newHead.Hash() {
			commonBlock = oldHead
			break
		}
		// Remove an old block as well as stash away a new block
		oldChain = append(oldChain, oldHead)
		newChain = append(newChain, newHead)

		// Step back with both chains
		oldHead = bc.GetHeader(oldHead.ParentHash, oldHead.Number.Uint64()-1)
		if oldHead == nil {
			return errInvalidOldChain
		}
		newHead = bc.GetHeader(newHead.ParentHash, newHead.Number.Uint64()-1)
		if newHead == nil {
			return errInvalidNewChain
		}
	}

	// Ensure XDPoS engine committed block will be not reverted
	if xdpos, ok := bc.Engine().(*XDPoS.XDPoS); ok {
		latestCommittedBlock := xdpos.EngineV2.GetLatestCommittedBlockInfo()
		if latestCommittedBlock != nil {
			cmp := commonBlock.Number.Cmp(latestCommittedBlock.Number)
			if cmp < 0 {
				for _, oldBlock := range oldChain {
					if oldBlock.Number.Cmp(latestCommittedBlock.Number) == 0 {
						if oldBlock.Hash() != latestCommittedBlock.Hash {
							log.Error("Impossible reorg, please file an issue", "OldNum", oldBlock.Number, "OldHash", oldBlock.Hash().Hex(), "LatestCommittedHash", latestCommittedBlock.Hash.Hex())
						} else {
							log.Warn("Stop reorg, blockchain is under forking attack", "OldCommittedNum", oldBlock.Number, "OldCommittedHash", oldBlock.Hash().Hex())
							return fmt.Errorf("stop reorg, blockchain is under forking attack. OldCommitted num %d, hash %s", oldBlock.Number, oldBlock.Hash().Hex())
						}
					}
				}
			} else if cmp == 0 {
				if commonBlock.Hash() != latestCommittedBlock.Hash {
					log.Error("Impossible reorg, please file an issue", "OldNum", commonBlock.Number.Uint64(), "OldHash", commonBlock.Hash().Hex(), "LatestCommittedHash", latestCommittedBlock.Hash.Hex())
				}
			}
		}
	}

	// Ensure the user sees large reorgs
	if len(oldChain) > 0 && len(newChain) > 0 {
		logFn := log.Info
		msg := "Chain reorg detected"
		if len(oldChain) > 63 {
			msg = "Large chain reorg detected"
			logFn = log.Warn
		}
		logFn(msg, "number", commonBlock.Number, "hash", commonBlock.Hash().Hex(),
			"drop", len(oldChain), "dropfrom", oldChain[0].Hash().Hex(), "add", len(newChain), "addfrom", newChain[0].Hash().Hex())
		blockReorgAddMeter.Mark(int64(len(newChain)))
		blockReorgDropMeter.Mark(int64(len(oldChain)))
		blockReorgMeter.Mark(1)
	} else if len(newChain) > 0 {
		// Special case happens in the post merge stage that current head is
		// the ancestor of new head while these two blocks are not consecutive
		log.Info("Extend chain", "add", len(newChain), "number", newChain[0].Number, "hash", newChain[0].Hash())
		blockReorgAddMeter.Mark(int64(len(newChain)))
	} else {
		// len(newChain) == 0 && len(oldChain) > 0
		// rewind the canonical chain to a lower point.
		log.Error("Impossible reorg, please file an issue", "oldnum", oldHead.Number, "oldhash", oldHead.Hash(), "oldblocks", len(oldChain), "newnum", newHead.Number, "newhash", newHead.Hash(), "newblocks", len(newChain))
	}

	// Acquire the tx-lookup lock before mutation. This step is essential
	// as the txlookups should be changed atomically, and all subsequent
	// reads should be blocked until the mutation is complete.
	// bc.txLookupLock.Lock()

	// Reorg can be executed, start reducing the chain's old blocks and appending
	// the new blocks
	var (
		deletedTxs []common.Hash
		rebirthTxs []common.Hash

		deletedLogs []*types.Log
		rebirthLogs []*types.Log
	)

	// Deleted log emission on the API uses forward order, which is borked, but
	// we'll leave it in for legacy reasons.
	//
	// TODO(karalabe): This should be nuked out, no idea how, deprecate some APIs?
	{
		for i := len(oldChain) - 1; i >= 0; i-- {
			block := bc.GetBlock(oldChain[i].Hash(), oldChain[i].Number.Uint64())
			if block == nil {
				return errInvalidOldChain // Corrupt database, mostly here to avoid weird panics
			}
			if logs := bc.collectLogs(block, true); len(logs) > 0 {
				deletedLogs = append(deletedLogs, logs...)
			}
			if len(deletedLogs) > 512 {
				go bc.rmLogsFeed.Send(RemovedLogsEvent{deletedLogs})
				deletedLogs = nil
			}
			// TODO(daniel): remove chainSideFeed, reference PR #30601
			// Also send event for blocks removed from the canon chain.
			// bc.chainSideFeed.Send(ChainSideEvent{Block: block})
		}
		if len(deletedLogs) > 0 {
			go bc.rmLogsFeed.Send(RemovedLogsEvent{deletedLogs})
		}
	}

	// Undo old blocks in reverse order
	for i := 0; i < len(oldChain); i++ {
		// Collect all the deleted transactions
		block := bc.GetBlock(oldChain[i].Hash(), oldChain[i].Number.Uint64())
		if block == nil {
			return errInvalidOldChain // Corrupt database, mostly here to avoid weird panics
		}
		for _, tx := range block.Transactions() {
			deletedTxs = append(deletedTxs, tx.Hash())
		}
		// Collect deleted logs and emit them for new integrations
		// if logs := bc.collectLogs(block, true); len(logs) > 0 {
		// 	slices.Reverse(logs) // Emit revertals latest first, older then
		// }
	}

	// Apply new blocks in forward order
	for i := len(newChain) - 1; i >= 0; i-- {
		// Collect all the included transactions
		block := bc.GetBlock(newChain[i].Hash(), newChain[i].Number.Uint64())
		if block == nil {
			return errInvalidNewChain // Corrupt database, mostly here to avoid weird panics
		}
		for _, tx := range block.Transactions() {
			rebirthTxs = append(rebirthTxs, tx.Hash())
		}
		// Collect inserted logs and emit them
		if logs := bc.collectLogs(block, false); len(logs) > 0 {
			rebirthLogs = append(rebirthLogs, logs...)
		}
		if len(rebirthLogs) > 512 {
			bc.logsFeed.Send(rebirthLogs)
			rebirthLogs = nil
		}
		// Update the head block
		bc.writeHeadBlock(block, true)
		// prepare set of masternodes for the next epoch
		if bc.chainConfig.XDPoS != nil && ((block.NumberU64() % bc.chainConfig.XDPoS.Epoch) == (bc.chainConfig.XDPoS.Epoch - bc.chainConfig.XDPoS.Gap)) {
			if err := bc.UpdateM1(); err != nil {
				log.Crit("Fail to update masternodes during reorg", "number", block.Number, "hash", block.Hash().Hex(), "err", err)
			}
		}
	}
	if len(rebirthLogs) > 0 {
		bc.logsFeed.Send(rebirthLogs)
	}

	// Delete useless indexes right now which includes the non-canonical
	// transaction indexes, canonical chain indexes which above the head.
	batch := bc.db.NewBatch()
	for _, tx := range types.HashDifference(deletedTxs, rebirthTxs) {
		rawdb.DeleteTxLookupEntry(batch, tx)
	}
	// Delete all hash markers that are not part of the new canonical chain.
	// Because the reorg function handles new chain head, all hash
	// markers greater than new chain head should be deleted.
	number := commonBlock.Number
	if len(newChain) > 0 {
		number = newChain[0].Number
	}
	for i := number.Uint64() + 1; ; i++ {
		hash := rawdb.ReadCanonicalHash(bc.db, i)
		if hash == (common.Hash{}) {
			break
		}
		rawdb.DeleteCanonicalHash(batch, i)
	}
	if err := batch.Write(); err != nil {
		log.Crit("Failed to delete useless indexes", "err", err)
	}

	// Reset the tx lookup cache to clear stale txlookup cache.
	// bc.txLookupCache.Purge()

	// Release the tx-lookup lock after mutation.
	// bc.txLookupLock.Unlock()

	return nil
}

// PostChainEvents iterates over the events generated by a chain insertion and
// posts them into the event feed.
// TODO: Should not expose PostChainEvents. The chain events should be posted in WriteBlock.
func (bc *BlockChain) PostChainEvents(events []interface{}, logs []*types.Log) {
	// post event logs for further processing
	if logs != nil {
		bc.logsFeed.Send(logs)
	}
	for _, event := range events {
		switch ev := event.(type) {
		case ChainEvent:
			bc.chainFeed.Send(ev)

		case ChainHeadEvent:
			bc.chainHeadFeed.Send(ev)

		case ChainSideEvent:
			bc.chainSideFeed.Send(ev)
		}
	}
}

// futureBlocksLoop processes the 'future block' queue.
func (bc *BlockChain) futureBlocksLoop() {
	defer bc.wg.Done()

	futureTimer := time.NewTicker(10 * time.Millisecond)
	defer futureTimer.Stop()
	for {
		select {
		case <-futureTimer.C:
			bc.procFutureBlocks()
		case <-bc.quit:
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
		if header, exist := bc.badBlocks.Peek(hash); exist {
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

	var roundNumber = types.Round(0)
	engine, ok := bc.Engine().(*XDPoS.XDPoS)
	if ok {
		var err error
		roundNumber, err = engine.EngineV2.GetRoundNumber(block.Header())
		if err != nil {
			log.Error("reportBlock", "GetRoundNumber", err)
		}
	}

	var receiptString string
	for i, receipt := range receipts {
		receiptString += fmt.Sprintf("\n  %d: cumulative: %v gas: %v contract: %v status: %v tx: %v logs: %v bloom: %x state: %x",
			i, receipt.CumulativeGasUsed, receipt.GasUsed, receipt.ContractAddress.Hex(),
			receipt.Status, receipt.TxHash.Hex(), receipt.Logs, receipt.Bloom, receipt.PostState)
	}
	log.Error(fmt.Sprintf(`
########## BAD BLOCK #########
Number: %v
Hash: %#x
Round: %v
Error: %v
%s
Receipts: %v
##############################
`, block.Number(), block.Hash(), roundNumber, err, bc.chainConfig.Description(), receiptString))
}

// InsertHeaderChain attempts to insert the given header chain in to the local
// chain, possibly creating a reorg. If an error is returned, it will return the
// index number of the failing header as well an error describing what went wrong.
//
// The verify parameter can be used to fine tune whether nonce verification
// should be done or not. The reason behind the optional check is because some
// of the header retrieval mechanisms already need to verify nonces, as well as
// because nonces can be verified sparsely, not needing to check each.
func (bc *BlockChain) InsertHeaderChain(chain []*types.Header, checkFreq int) (int, error) {
	start := time.Now()
	if i, err := bc.hc.ValidateHeaderChain(chain, checkFreq); err != nil {
		return i, err
	}

	if !bc.chainmu.TryLock() {
		return 0, errChainStopped
	}
	defer bc.chainmu.Unlock()

	whFunc := func(header *types.Header) error {
		_, err := bc.hc.WriteHeader(header)
		return err
	}

	return bc.hc.InsertHeaderChain(chain, whFunc, start)
}

// CurrentHeader retrieves the current head header of the canonical chain. The
// header is retrieved from the HeaderChain's internal cache.
func (bc *BlockChain) CurrentHeader() *types.Header {
	return bc.hc.CurrentHeader()
}

// GetTd retrieves a block's total difficulty in the canonical chain from the
// database by hash and number, caching it if found.
func (bc *BlockChain) GetTd(hash common.Hash, number uint64) *big.Int {
	return bc.hc.GetTd(hash, number)
}

// GetTdByHash retrieves a block's total difficulty in the canonical chain from the
// database by hash, caching it if found.
func (bc *BlockChain) GetTdByHash(hash common.Hash) *big.Int {
	return bc.hc.GetTdByHash(hash)
}

// GetHeader retrieves a block header from the database by hash and number,
// caching it if found.
func (bc *BlockChain) GetHeader(hash common.Hash, number uint64) *types.Header {
	return bc.hc.GetHeader(hash, number)
}

// GetHeaderByHash retrieves a block header from the database by hash, caching it if
// found.
func (bc *BlockChain) GetHeaderByHash(hash common.Hash) *types.Header {
	return bc.hc.GetHeaderByHash(hash)
}

// HasHeader checks if a block header is present in the database or not, caching
// it if present.
func (bc *BlockChain) HasHeader(hash common.Hash, number uint64) bool {
	return bc.hc.HasHeader(hash, number)
}

// GetCanonicalHash returns the canonical hash for a given block number
func (bc *BlockChain) GetCanonicalHash(number uint64) common.Hash {
	return bc.hc.GetCanonicalHash(number)
}

// GetBlockHashesFromHash retrieves a number of block hashes starting at a given
// hash, fetching towards the genesis block.
func (bc *BlockChain) GetBlockHashesFromHash(hash common.Hash, max uint64) []common.Hash {
	return bc.hc.GetBlockHashesFromHash(hash, max)
}

// GetHeaderByNumber retrieves a block header from the database by number,
// caching it (associated with its hash) if found.
func (bc *BlockChain) GetHeaderByNumber(number uint64) *types.Header {
	return bc.hc.GetHeaderByNumber(number)
}

// Set config for testing purpose function
func (bc *BlockChain) SetConfig(config *params.ChainConfig) {
	bc.chainConfig = config
}

// Config retrieves the blockchain's chain configuration.
func (bc *BlockChain) Config() *params.ChainConfig { return bc.chainConfig }

// Engine retrieves the blockchain's consensus engine.
func (bc *BlockChain) Engine() consensus.Engine { return bc.engine }

// SubscribeRemovedLogsEvent registers a subscription of RemovedLogsEvent.
func (bc *BlockChain) SubscribeRemovedLogsEvent(ch chan<- RemovedLogsEvent) event.Subscription {
	return bc.scope.Track(bc.rmLogsFeed.Subscribe(ch))
}

// SubscribeChainEvent registers a subscription of ChainEvent.
func (bc *BlockChain) SubscribeChainEvent(ch chan<- ChainEvent) event.Subscription {
	return bc.scope.Track(bc.chainFeed.Subscribe(ch))
}

// SubscribeChainHeadEvent registers a subscription of ChainHeadEvent.
func (bc *BlockChain) SubscribeChainHeadEvent(ch chan<- ChainHeadEvent) event.Subscription {
	return bc.scope.Track(bc.chainHeadFeed.Subscribe(ch))
}

// SubscribeChainSideEvent registers a subscription of ChainSideEvent.
func (bc *BlockChain) SubscribeChainSideEvent(ch chan<- ChainSideEvent) event.Subscription {
	return bc.scope.Track(bc.chainSideFeed.Subscribe(ch))
}

// SubscribeLogsEvent registers a subscription of []*types.Log.
func (bc *BlockChain) SubscribeLogsEvent(ch chan<- []*types.Log) event.Subscription {
	return bc.scope.Track(bc.logsFeed.Subscribe(ch))
}

// Get current IPC Client.
func (bc *BlockChain) GetClient() (bind.ContractBackend, error) {
	if bc.Client == nil {
		// Inject ipc client global instance.
		client, err := ethclient.Dial(bc.IPCEndpoint)
		if err != nil {
			log.Error("Fail to connect IPC", "error", err)
			return nil, err
		}
		bc.Client = client
	}

	return bc.Client, nil
}

func (bc *BlockChain) UpdateM1() error {
	engine, ok := bc.Engine().(*XDPoS.XDPoS)
	if bc.Config().XDPoS == nil || !ok {
		return ErrNotXDPoS
	}
	log.Info("It's time to update new set of masternodes for the next epoch...")
	// get masternodes information from smart contract
	client, err := bc.GetClient()
	if err != nil {
		return err
	}
	addr := common.MasternodeVotingSMCBinary
	validator, err := contractValidator.NewXDCValidator(addr, client)
	if err != nil {
		return err
	}
	opts := new(bind.CallOpts)

	var candidates []common.Address
	// get candidates from slot of stateDB
	// if can't get anything, request from contracts
	stateDB, err := bc.State()
	if err != nil {
		candidates, err = validator.GetCandidates(opts)
		if err != nil {
			return err
		}
	} else if stateDB == nil {
		return errors.New("nil stateDB in UpdateM1")
	} else {
		candidates = state.GetCandidates(stateDB)
	}

	var ms []utils.Masternode
	for _, candidate := range candidates {
		v, err := validator.GetCandidateCap(opts, candidate)
		if err != nil {
			return err
		}
		// TODO: smart contract shouldn't return "0x0000000000000000000000000000000000000000"
		if !candidate.IsZero() {
			ms = append(ms, utils.Masternode{Address: candidate, Stake: v})
		}
	}
	if len(ms) == 0 {
		log.Error("No masternode found. Stopping node")
		os.Exit(1)
	} else {
		sort.Slice(ms, func(i, j int) bool {
			return ms[i].Stake.Cmp(ms[j].Stake) >= 0
		})
		log.Info("Ordered list of masternode candidates")
		for _, m := range ms {
			log.Info("", "address", m.Address.String(), "stake", m.Stake)
		}
		// update masternodes

		log.Info("Updating new set of masternodes")
		// get block header
		header := bc.CurrentHeader()
		err = engine.UpdateMasternodes(bc, header, ms)
		if err != nil {
			return err
		}
		log.Info("Masternodes are ready for the next epoch")
	}
	return nil
}

func (bc *BlockChain) logExchangeData(block *types.Block) {
	engine, ok := bc.Engine().(*XDPoS.XDPoS)
	if !ok || engine == nil {
		return
	}
	XDCXService := engine.GetXDCXService()
	if XDCXService == nil || !XDCXService.IsSDKNode() {
		return
	}
	txMatchBatchData, err := ExtractTradingTransactions(block.Transactions())
	if err != nil {
		log.Crit("failed to extract matching transaction", "err", err)
		return
	}
	if len(txMatchBatchData) == 0 {
		return
	}
	currentState, err := bc.State()
	if err != nil {
		log.Crit("logExchangeData: failed to get current state", "err", err)
		return
	}
	start := time.Now()
	defer func() {
		//The deferred call's arguments are evaluated immediately, but the function call is not executed until the surrounding function returns
		// That's why we should put this log statement in an anonymous function
		log.Debug("logExchangeData takes", "time", common.PrettyDuration(time.Since(start)), "blockNumber", block.NumberU64())
	}()

	for _, txMatchBatch := range txMatchBatchData {
		dirtyOrderCount := uint64(0)
		for _, txMatch := range txMatchBatch.Data {
			var (
				takerOrderInTx *tradingstate.OrderItem
				trades         []map[string]string
				rejectedOrders []*tradingstate.OrderItem
			)

			if takerOrderInTx, err = txMatch.DecodeOrder(); err != nil {
				log.Crit("SDK node decode takerOrderInTx failed", "txDataMatch", txMatch)
				return
			}
			cacheKey := crypto.Keccak256Hash(txMatchBatch.TxHash.Bytes(), tradingstate.GetMatchingResultCacheKey(takerOrderInTx).Bytes())
			// getTrades from cache
			resultTrades, ok := bc.resultTrade.Get(cacheKey)
			if ok && resultTrades != nil {
				trades = resultTrades.([]map[string]string)
			}

			// getRejectedOrder from cache
			rejected, ok := bc.rejectedOrders.Get(cacheKey)
			if ok && rejected != nil {
				rejectedOrders = rejected.([]*tradingstate.OrderItem)
			}

			txMatchTime := time.Unix(block.Header().Time.Int64(), 0).UTC()
			if err := XDCXService.SyncDataToSDKNode(takerOrderInTx, txMatchBatch.TxHash, txMatchTime, currentState, trades, rejectedOrders, &dirtyOrderCount); err != nil {
				log.Crit("failed to SyncDataToSDKNode ", "blockNumber", block.Number(), "err", err)
				return
			}
		}
	}
}

func (bc *BlockChain) logLendingData(block *types.Block) {
	engine, ok := bc.Engine().(*XDPoS.XDPoS)
	if !ok || engine == nil {
		return
	}
	XDCXService := engine.GetXDCXService()
	if XDCXService == nil || !XDCXService.IsSDKNode() {
		return
	}
	lendingService := engine.GetLendingService()
	if lendingService == nil {
		return
	}
	batches, err := ExtractLendingTransactions(block.Transactions())
	if err != nil {
		log.Crit("failed to extract lending transaction", "err", err)
	}
	start := time.Now()
	defer func() {
		//The deferred call's arguments are evaluated immediately, but the function call is not executed until the surrounding function returns
		// That's why we should put this log statement in an anonymous function
		log.Debug("logLendingData takes", "time", common.PrettyDuration(time.Since(start)), "blockNumber", block.NumberU64())
	}()

	for _, batch := range batches {

		dirtyOrderCount := uint64(0)
		for _, item := range batch.Data {
			var (
				trades         []*lendingstate.LendingTrade
				rejectedOrders []*lendingstate.LendingItem
			)
			// getTrades from cache
			resultLendingTrades, ok := bc.resultLendingTrade.Get(crypto.Keccak256Hash(batch.TxHash.Bytes(), lendingstate.GetLendingCacheKey(item).Bytes()))

			if ok && resultLendingTrades != nil {
				trades = resultLendingTrades.([]*lendingstate.LendingTrade)
			}

			// getRejectedOrder from cache
			rejected, ok := bc.rejectedLendingItem.Get(crypto.Keccak256Hash(batch.TxHash.Bytes(), lendingstate.GetLendingCacheKey(item).Bytes()))
			if ok && rejected != nil {
				rejectedOrders = rejected.([]*lendingstate.LendingItem)
			}

			txMatchTime := time.Unix(block.Header().Time.Int64(), 0).UTC()
			statedb, _ := bc.State()

			if err := lendingService.SyncDataToSDKNode(bc, statedb.Copy(), block, item, batch.TxHash, txMatchTime, trades, rejectedOrders, &dirtyOrderCount); err != nil {
				log.Crit("lending: failed to SyncDataToSDKNode ", "blockNumber", block.Number(), "err", err)
			}
		}
	}

	// update finalizedTrades
	if block.Number().Uint64()%bc.chainConfig.XDPoS.Epoch == common.LiquidateLendingTradeBlock {
		finalizedTx, err := ExtractLendingFinalizedTradeTransactions(block.Transactions())
		if err != nil {
			log.Crit("failed to extract finalizedTrades transaction", "err", err)
		}
		finalizedTrades := map[common.Hash]*lendingstate.LendingTrade{}
		finalizedData, ok := bc.finalizedTrade.Get(finalizedTx.TxHash)
		if ok && finalizedData != nil {
			finalizedTrades = finalizedData.(map[common.Hash]*lendingstate.LendingTrade)
		}
		if len(finalizedTrades) > 0 {
			if err := lendingService.UpdateLiquidatedTrade(block.Time().Uint64(), finalizedTx, finalizedTrades); err != nil {
				log.Crit("lending: failed to UpdateLiquidatedTrade ", "blockNumber", block.Number(), "err", err)
			}
		}
	}
}

func (bc *BlockChain) AddMatchingResult(txHash common.Hash, matchingResults map[common.Hash]tradingstate.MatchingResult) {
	for hash, result := range matchingResults {
		cacheKey := crypto.Keccak256Hash(txHash.Bytes(), hash.Bytes())
		bc.resultTrade.Add(cacheKey, result.Trades)
		bc.rejectedOrders.Add(cacheKey, result.Rejects)
	}
}

func (bc *BlockChain) AddLendingResult(txHash common.Hash, lendingResults map[common.Hash]lendingstate.MatchingResult) {
	for hash, result := range lendingResults {
		bc.resultLendingTrade.Add(crypto.Keccak256Hash(txHash.Bytes(), hash.Bytes()), result.Trades)
		bc.rejectedLendingItem.Add(crypto.Keccak256Hash(txHash.Bytes(), hash.Bytes()), result.Rejects)
	}
}

func (bc *BlockChain) AddFinalizedTrades(txHash common.Hash, trades map[common.Hash]*lendingstate.LendingTrade) {
	bc.finalizedTrade.Add(txHash, trades)
}
