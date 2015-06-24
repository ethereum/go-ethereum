package core

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/compression/rle"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/pow"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/hashicorp/golang-lru"
	"github.com/rcrowley/go-metrics"
	"github.com/syndtr/goleveldb/leveldb"
)

var (
	chainlogger = logger.NewLogger("CHAIN")
	jsonlogger  = logger.NewJsonLogger()

	blockHashPre = []byte("block-hash-")
	blockNumPre  = []byte("block-num-")

	blockInsertTimer = metrics.GetOrRegisterTimer("core/BlockInsertions", metrics.DefaultRegistry)
)

const (
	blockCacheLimit     = 256
	maxFutureBlocks     = 256
	maxTimeFutureBlocks = 30
)

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block b should have when created at time
// given the parent block's time and difficulty.
func CalcDifficulty(time int64, parentTime int64, parentDiff *big.Int) *big.Int {
	diff := new(big.Int)
	adjust := new(big.Int).Div(parentDiff, params.DifficultyBoundDivisor)
	if big.NewInt(time-parentTime).Cmp(params.DurationLimit) < 0 {
		diff.Add(parentDiff, adjust)
	} else {
		diff.Sub(parentDiff, adjust)
	}
	if diff.Cmp(params.MinimumDifficulty) < 0 {
		return params.MinimumDifficulty
	}
	return diff
}

// CalcTD computes the total difficulty of block.
func CalcTD(block, parent *types.Block) *big.Int {
	if parent == nil {
		return block.Difficulty()
	}
	d := block.Difficulty()
	d.Add(d, parent.Td)
	return d
}

// CalcGasLimit computes the gas limit of the next block after parent.
// The result may be modified by the caller.
func CalcGasLimit(parent *types.Block) *big.Int {
	decay := new(big.Int).Div(parent.GasLimit(), params.GasLimitBoundDivisor)
	contrib := new(big.Int).Mul(parent.GasUsed(), big.NewInt(3))
	contrib = contrib.Div(contrib, big.NewInt(2))
	contrib = contrib.Div(contrib, params.GasLimitBoundDivisor)

	gl := new(big.Int).Sub(parent.GasLimit(), decay)
	gl = gl.Add(gl, contrib)
	gl = gl.Add(gl, big.NewInt(1))
	gl.Set(common.BigMax(gl, params.MinGasLimit))

	if gl.Cmp(params.GenesisGasLimit) < 0 {
		gl.Add(parent.GasLimit(), decay)
		gl.Set(common.BigMin(gl, params.GenesisGasLimit))
	}
	return gl
}

type ChainManager struct {
	//eth          EthManager
	blockDb      common.Database
	stateDb      common.Database
	processor    types.BlockProcessor
	eventMux     *event.TypeMux
	genesisBlock *types.Block
	// Last known total difficulty
	mu      sync.RWMutex
	chainmu sync.RWMutex
	tsmu    sync.RWMutex

	td              *big.Int
	currentBlock    *types.Block
	lastBlockHash   common.Hash
	currentGasLimit *big.Int

	transState *state.StateDB
	txState    *state.ManagedState

	cache         *lru.Cache  // cache is the LRU caching
	futureBlocks  *BlockCache // future blocks are blocks added for later processing
	pendingBlocks *BlockCache // pending blocks contain blocks not yet written to the db

	quit chan struct{}
	// procInterrupt must be atomically called
	procInterrupt int32 // interrupt signaler for block processing
	wg            sync.WaitGroup

	pow pow.PoW
}

func NewChainManager(genesis *types.Block, blockDb, stateDb common.Database, pow pow.PoW, mux *event.TypeMux) (*ChainManager, error) {
	cache, _ := lru.New(blockCacheLimit)
	bc := &ChainManager{
		blockDb:      blockDb,
		stateDb:      stateDb,
		genesisBlock: GenesisBlock(42, stateDb),
		eventMux:     mux,
		quit:         make(chan struct{}),
		cache:        cache,
		pow:          pow,
	}
	// Check the genesis block given to the chain manager. If the genesis block mismatches block number 0
	// throw an error. If no block or the same block's found continue.
	if g := bc.GetBlockByNumber(0); g != nil && g.Hash() != genesis.Hash() {
		return nil, fmt.Errorf("Genesis mismatch. Maybe different nonce (%d vs %d)? %x / %x", g.Nonce(), genesis.Nonce(), g.Hash().Bytes()[:4], genesis.Hash().Bytes()[:4])
	}
	bc.genesisBlock = genesis
	bc.setLastState()

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

	bc.transState = bc.State().Copy()
	// Take ownership of this particular state
	bc.txState = state.ManageState(bc.State().Copy())

	bc.futureBlocks = NewBlockCache(maxFutureBlocks)
	bc.makeCache()

	go bc.update()

	return bc, nil
}

func (bc *ChainManager) SetHead(head *types.Block) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for block := bc.currentBlock; block != nil && block.Hash() != head.Hash(); block = bc.GetBlock(block.ParentHash()) {
		bc.removeBlock(block)
	}

	bc.cache, _ = lru.New(blockCacheLimit)
	bc.currentBlock = head
	bc.makeCache()

	statedb := state.New(head.Root(), bc.stateDb)
	bc.txState = state.ManageState(statedb)
	bc.transState = statedb.Copy()
	bc.setTotalDifficulty(head.Td)
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

	return self.lastBlockHash
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
	return state.New(self.CurrentBlock().Root(), self.stateDb)
}

func (self *ChainManager) TransState() *state.StateDB {
	self.tsmu.RLock()
	defer self.tsmu.RUnlock()

	return self.transState
}

func (self *ChainManager) setTransState(statedb *state.StateDB) {
	self.transState = statedb
}

func (bc *ChainManager) setLastState() {
	data, _ := bc.blockDb.Get([]byte("LastBlock"))
	if len(data) != 0 {
		block := bc.GetBlock(common.BytesToHash(data))
		if block != nil {
			bc.currentBlock = block
			bc.lastBlockHash = block.Hash()
		} else {
			glog.Fatalf("Fatal. LastBlock not found. Please run removedb and resync")
		}
	} else {
		bc.Reset()
	}
	bc.td = bc.currentBlock.Td
	bc.currentGasLimit = CalcGasLimit(bc.currentBlock)

	if glog.V(logger.Info) {
		glog.Infof("Last block (#%v) %x TD=%v\n", bc.currentBlock.Number(), bc.currentBlock.Hash(), bc.td)
	}
}

func (bc *ChainManager) makeCache() {
	bc.cache, _ = lru.New(blockCacheLimit)
	// load in last `blockCacheLimit` - 1 blocks. Last block is the current.
	for _, block := range bc.GetBlocksFromHash(bc.currentBlock.Hash(), blockCacheLimit) {
		bc.cache.Add(block.Hash(), block)
	}
}

func (bc *ChainManager) Reset() {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for block := bc.currentBlock; block != nil; block = bc.GetBlock(block.ParentHash()) {
		bc.removeBlock(block)
	}

	bc.cache, _ = lru.New(blockCacheLimit)

	// Prepare the genesis block
	bc.write(bc.genesisBlock)
	bc.insert(bc.genesisBlock)
	bc.currentBlock = bc.genesisBlock
	bc.makeCache()

	bc.setTotalDifficulty(common.Big("0"))
}

func (bc *ChainManager) removeBlock(block *types.Block) {
	bc.blockDb.Delete(append(blockHashPre, block.Hash().Bytes()...))
}

func (bc *ChainManager) ResetWithGenesisBlock(gb *types.Block) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for block := bc.currentBlock; block != nil; block = bc.GetBlock(block.ParentHash()) {
		bc.removeBlock(block)
	}

	// Prepare the genesis block
	gb.Td = gb.Difficulty()
	bc.genesisBlock = gb
	bc.write(bc.genesisBlock)
	bc.insert(bc.genesisBlock)
	bc.currentBlock = bc.genesisBlock
	bc.makeCache()
	bc.td = gb.Difficulty()
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
	key := append(blockNumPre, block.Number().Bytes()...)
	err := bc.blockDb.Put(key, block.Hash().Bytes())
	if err != nil {
		glog.Fatal("db write fail:", err)
	}
	err = bc.blockDb.Put([]byte("LastBlock"), block.Hash().Bytes())
	if err != nil {
		glog.Fatal("db write fail:", err)
	}

	bc.currentBlock = block
	bc.lastBlockHash = block.Hash()
}

func (bc *ChainManager) write(block *types.Block) {
	tstart := time.Now()

	go func() {
		enc, _ := rlp.EncodeToBytes((*types.StorageBlock)(block))
		key := append(blockHashPre, block.Hash().Bytes()...)
		err := bc.blockDb.Put(key, enc)
		if err != nil {
			glog.Fatal("db write fail:", err)
		}
	}()

	if glog.V(logger.Debug) {
		glog.Infof("wrote block #%v %s. Took %v\n", block.Number(), common.PP(block.Hash().Bytes()), time.Since(tstart))
	}
}

// Accessors
func (bc *ChainManager) Genesis() *types.Block {
	return bc.genesisBlock
}

// Block fetching methods
func (bc *ChainManager) HasBlock(hash common.Hash) bool {
	if bc.cache.Contains(hash) {
		return true
	}

	if bc.pendingBlocks != nil {
		if block := bc.pendingBlocks.Get(hash); block != nil {
			return true
		}
	}

	data, _ := bc.blockDb.Get(append(blockHashPre, hash[:]...))
	return len(data) != 0
}

func (self *ChainManager) GetBlockHashesFromHash(hash common.Hash, max uint64) (chain []common.Hash) {
	block := self.GetBlock(hash)
	if block == nil {
		return
	}
	// XXX Could be optimised by using a different database which only holds hashes (i.e., linked list)
	for i := uint64(0); i < max; i++ {
		block = self.GetBlock(block.ParentHash())
		if block == nil {
			break
		}

		chain = append(chain, block.Hash())
		if block.Number().Cmp(common.Big0) <= 0 {
			break
		}
	}

	return
}

func (self *ChainManager) GetBlock(hash common.Hash) *types.Block {
	if block, ok := self.cache.Get(hash); ok {
		return block.(*types.Block)
	}

	if self.pendingBlocks != nil {
		if block := self.pendingBlocks.Get(hash); block != nil {
			return block
		}
	}

	data, _ := self.blockDb.Get(append(blockHashPre, hash[:]...))
	if len(data) == 0 {
		return nil
	}
	var block types.StorageBlock
	if err := rlp.Decode(bytes.NewReader(data), &block); err != nil {
		glog.V(logger.Error).Infof("invalid block RLP for hash %x: %v", hash, err)
		return nil
	}

	// Add the block to the cache
	self.cache.Add(hash, (*types.Block)(&block))

	return (*types.Block)(&block)
}

func (self *ChainManager) GetBlockByNumber(num uint64) *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.getBlockByNumber(num)

}

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

// non blocking version
func (self *ChainManager) getBlockByNumber(num uint64) *types.Block {
	key, _ := self.blockDb.Get(append(blockNumPre, big.NewInt(int64(num)).Bytes()...))
	if len(key) == 0 {
		return nil
	}

	return self.GetBlock(common.BytesToHash(key))
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
	var blocks []*types.Block
	self.futureBlocks.Each(func(i int, block *types.Block) {
		blocks = append(blocks, block)
	})
	if len(blocks) > 0 {
		types.BlockBy(types.Number).Sort(blocks)
		self.InsertChain(blocks)
	}
}

func (self *ChainManager) enqueueForWrite(block *types.Block) {
	self.pendingBlocks.Push(block)
}

func (self *ChainManager) flushQueuedBlocks() {
	db, batchWrite := self.blockDb.(*ethdb.LDBDatabase)
	batch := new(leveldb.Batch)
	self.pendingBlocks.Each(func(i int, block *types.Block) {
		enc, _ := rlp.EncodeToBytes((*types.StorageBlock)(block))
		key := append(blockHashPre, block.Hash().Bytes()...)
		if batchWrite {
			batch.Put(key, rle.Compress(enc))
		} else {
			self.blockDb.Put(key, enc)
		}
	})
	if batchWrite {
		db.LDB().Write(batch, nil)
	}
}

// InsertChain will attempt to insert the given chain in to the canonical chain or, otherwise, create a fork. It an error is returned
// it will return the index number of the failing block as well an error describing what went wrong (for possible errors see core/errors.go).
func (self *ChainManager) InsertChain(chain types.Blocks) (int, error) {
	self.wg.Add(1)
	defer self.wg.Done()

	self.chainmu.Lock()
	defer self.chainmu.Unlock()

	self.pendingBlocks = NewBlockCache(len(chain))

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
	defer self.flushQueuedBlocks()

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

		// Setting block.Td regardless of error (known for example) prevents errors down the line
		// in the protocol handler
		block.Td = new(big.Int).Set(CalcTD(block, self.GetBlock(block.ParentHash())))

		// Call in to the block processor and check for errors. It's likely that if one block fails
		// all others will fail too (unless a known block is returned).
		logs, err := self.processor.Process(block)
		if err != nil {
			if IsKnownBlockErr(err) {
				stats.ignored++
				continue
			}

			if err == BlockFutureErr {
				// Allow up to MaxFuture second in the future blocks. If this limit
				// is exceeded the chain is discarded and processed at a later time
				// if given.
				if max := time.Now().Unix() + maxTimeFutureBlocks; block.Time() > max {
					return i, fmt.Errorf("%v: BlockFutureErr, %v > %v", BlockFutureErr, block.Time(), max)
				}

				self.futureBlocks.Push(block)
				stats.queued++
				continue
			}

			if IsParentErr(err) && self.futureBlocks.Has(block.ParentHash()) {
				self.futureBlocks.Push(block)
				stats.queued++
				continue
			}

			blockErr(block, err)

			return i, err
		}

		txcount += len(block.Transactions())

		cblock := self.currentBlock
		// Compare the TD of the last known block in the canonical chain to make sure it's greater.
		// At this point it's possible that a different chain (fork) becomes the new canonical chain.
		if block.Td.Cmp(self.Td()) > 0 {
			// chain fork
			if block.ParentHash() != cblock.Hash() {
				// during split we merge two different chains and create the new canonical chain
				err := self.merge(cblock, block)
				if err != nil {
					return i, err
				}

				queue[i] = ChainSplitEvent{block, logs}
				queueEvent.splitCount++
			}

			self.mu.Lock()
			self.setTotalDifficulty(block.Td)
			self.insert(block)
			self.mu.Unlock()

			jsonlogger.LogJson(&logger.EthChainNewHead{
				BlockHash:     block.Hash().Hex(),
				BlockNumber:   block.Number(),
				ChainHeadHash: cblock.Hash().Hex(),
				BlockPrevHash: block.ParentHash().Hex(),
			})

			self.setTransState(state.New(block.Root(), self.stateDb))
			self.txState.SetState(state.New(block.Root(), self.stateDb))

			queue[i] = ChainEvent{block, block.Hash(), logs}
			queueEvent.canonicalCount++

			if glog.V(logger.Debug) {
				glog.Infof("[%v] inserted block #%d (%d TXs %d UNCs) (%x...). Took %v\n", time.Now().UnixNano(), block.Number(), len(block.Transactions()), len(block.Uncles()), block.Hash().Bytes()[0:4], time.Since(bstart))
			}
		} else {
			if glog.V(logger.Detail) {
				glog.Infof("inserted forked block #%d (TD=%v) (%d TXs %d UNCs) (%x...). Took %v\n", block.Number(), block.Difficulty(), len(block.Transactions()), len(block.Uncles()), block.Hash().Bytes()[0:4], time.Since(bstart))
			}

			queue[i] = ChainSideEvent{block, logs}
			queueEvent.sideCount++
		}
		self.enqueueForWrite(block)
		// Delete from future blocks
		self.futureBlocks.Delete(block.Hash())

		stats.processed++
		blockInsertTimer.UpdateSince(bstart)
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

	if glog.V(logger.Info) {
		commonHash := commonBlock.Hash()
		glog.Infof("Fork detected @ %x. Reorganising chain from #%v %x to %x", commonHash[:4], numSplit, oldStart.Hash().Bytes()[:4], newStart.Hash().Bytes()[:4])
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
		self.insert(block)
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
						if self.lastBlockHash == event.Hash {
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
