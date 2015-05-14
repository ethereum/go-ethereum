package core

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	chainlogger = logger.NewLogger("CHAIN")
	jsonlogger  = logger.NewJsonLogger()

	blockHashPre = []byte("block-hash-")
	blockNumPre  = []byte("block-num-")
)

const (
	blockCacheLimit = 10000
	maxFutureBlocks = 256
)

func CalcDifficulty(block, parent *types.Header) *big.Int {
	diff := new(big.Int)

	adjust := new(big.Int).Div(parent.Difficulty, params.DifficultyBoundDivisor)
	if big.NewInt(int64(block.Time)-int64(parent.Time)).Cmp(params.DurationLimit) < 0 {
		diff.Add(parent.Difficulty, adjust)
	} else {
		diff.Sub(parent.Difficulty, adjust)
	}

	if diff.Cmp(params.MinimumDifficulty) < 0 {
		return params.MinimumDifficulty
	}

	return diff
}

func CalculateTD(block, parent *types.Block) *big.Int {
	if parent == nil {
		return block.Difficulty()
	}

	td := new(big.Int).Add(parent.Td, block.Header().Difficulty)

	return td
}

func CalcGasLimit(parent *types.Block) *big.Int {
	// ((1024-1) * parent.gasLimit + (gasUsed * 6 / 5)) / 1024
	previous := new(big.Int).Mul(big.NewInt(1024-1), parent.GasLimit())
	current := new(big.Rat).Mul(new(big.Rat).SetInt(parent.GasUsed()), big.NewRat(6, 5))
	curInt := new(big.Int).Div(current.Num(), current.Denom())

	result := new(big.Int).Add(previous, curInt)
	result.Div(result, big.NewInt(1024))
	return common.BigMax(params.GenesisGasLimit, result)
}

type ChainManager struct {
	//eth          EthManager
	blockDb      common.Database
	stateDb      common.Database
	processor    types.BlockProcessor
	eventMux     *event.TypeMux
	genesisBlock *types.Block
	// Last known total difficulty
	mu   sync.RWMutex
	tsmu sync.RWMutex

	td              *big.Int
	currentBlock    *types.Block
	lastBlockHash   common.Hash
	currentGasLimit *big.Int

	transState *state.StateDB
	txState    *state.ManagedState

	cache        *BlockCache
	futureBlocks *BlockCache

	quit chan struct{}
	wg   sync.WaitGroup
}

func NewChainManager(blockDb, stateDb common.Database, mux *event.TypeMux) *ChainManager {
	bc := &ChainManager{
		blockDb:      blockDb,
		stateDb:      stateDb,
		genesisBlock: GenesisBlock(stateDb),
		eventMux:     mux,
		quit:         make(chan struct{}),
		cache:        NewBlockCache(blockCacheLimit),
	}
	bc.setLastState()

	// Check the current state of the block hashes and make sure that we do not have any of the bad blocks in our chain
	for _, hash := range badHashes {
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

	return bc
}

func (bc *ChainManager) SetHead(head *types.Block) {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for block := bc.currentBlock; block != nil && block.Hash() != head.Hash(); block = bc.GetBlock(block.Header().ParentHash) {
		bc.removeBlock(block)
	}

	bc.cache = NewBlockCache(blockCacheLimit)
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

	return self.td
}

func (self *ChainManager) GasLimit() *big.Int {
	// return self.currentGasLimit
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

	return self.td, self.currentBlock.Hash(), self.genesisBlock.Hash()
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

func (self *ChainManager) TxState() *state.ManagedState {
	self.tsmu.RLock()
	defer self.tsmu.RUnlock()

	return self.txState
}

func (self *ChainManager) setTxState(statedb *state.StateDB) {
	self.tsmu.Lock()
	defer self.tsmu.Unlock()
	self.txState = state.ManageState(statedb)
}

func (self *ChainManager) setTransState(statedb *state.StateDB) {
	self.transState = statedb
}

func (bc *ChainManager) setLastState() {
	data, _ := bc.blockDb.Get([]byte("LastBlock"))
	if len(data) != 0 {
		block := bc.GetBlock(common.BytesToHash(data))
		bc.currentBlock = block
		bc.lastBlockHash = block.Hash()

		// Set the last know difficulty (might be 0x0 as initial value, Genesis)
		bc.td = common.BigD(bc.blockDb.LastKnownTD())
	} else {
		bc.Reset()
	}
	bc.currentGasLimit = CalcGasLimit(bc.currentBlock)

	if glog.V(logger.Info) {
		glog.Infof("Last block (#%v) %x TD=%v\n", bc.currentBlock.Number(), bc.currentBlock.Hash(), bc.td)
	}
}

func (bc *ChainManager) makeCache() {
	if bc.cache == nil {
		bc.cache = NewBlockCache(blockCacheLimit)
	}
	// load in last `blockCacheLimit` - 1 blocks. Last block is the current.
	ancestors := bc.GetAncestors(bc.currentBlock, blockCacheLimit-1)
	ancestors = append(ancestors, bc.currentBlock)
	for _, block := range ancestors {
		bc.cache.Push(block)
	}
}

// Block creation & chain handling
func (bc *ChainManager) NewBlock(coinbase common.Address) *types.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	var (
		root       common.Hash
		parentHash common.Hash
	)

	if bc.currentBlock != nil {
		root = bc.currentBlock.Header().Root
		parentHash = bc.lastBlockHash
	}

	block := types.NewBlock(
		parentHash,
		coinbase,
		root,
		common.BigPow(2, 32),
		0,
		nil)
	block.SetUncles(nil)
	block.SetTransactions(nil)
	block.SetReceipts(nil)

	parent := bc.currentBlock
	if parent != nil {
		header := block.Header()
		header.Difficulty = CalcDifficulty(block.Header(), parent.Header())
		header.Number = new(big.Int).Add(parent.Header().Number, common.Big1)
		header.GasLimit = CalcGasLimit(parent)
	}

	return block
}

func (bc *ChainManager) Reset() {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for block := bc.currentBlock; block != nil; block = bc.GetBlock(block.Header().ParentHash) {
		bc.removeBlock(block)
	}

	if bc.cache == nil {
		bc.cache = NewBlockCache(blockCacheLimit)
	}

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

	for block := bc.currentBlock; block != nil; block = bc.GetBlock(block.Header().ParentHash) {
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
	self.mu.RLock()
	defer self.mu.RUnlock()
	glog.V(logger.Info).Infof("exporting %v blocks...\n", self.currentBlock.Header().Number)

	last := self.currentBlock.NumberU64()

	for nr := uint64(0); nr <= last; nr++ {
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

func (bc *ChainManager) insert(block *types.Block) {
	key := append(blockNumPre, block.Number().Bytes()...)
	bc.blockDb.Put(key, block.Hash().Bytes())

	bc.blockDb.Put([]byte("LastBlock"), block.Hash().Bytes())
	bc.currentBlock = block
	bc.lastBlockHash = block.Hash()
}

func (bc *ChainManager) write(block *types.Block) {
	enc, _ := rlp.EncodeToBytes((*types.StorageBlock)(block))
	key := append(blockHashPre, block.Hash().Bytes()...)
	bc.blockDb.Put(key, enc)
	// Push block to cache
	bc.cache.Push(block)
}

// Accessors
func (bc *ChainManager) Genesis() *types.Block {
	return bc.genesisBlock
}

// Block fetching methods
func (bc *ChainManager) HasBlock(hash common.Hash) bool {
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
	if block := self.cache.Get(hash); block != nil {
		return block
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
	return (*types.Block)(&block)
}

func (self *ChainManager) GetBlockByNumber(num uint64) *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.getBlockByNumber(num)

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

func (self *ChainManager) GetAncestors(block *types.Block, length int) (blocks []*types.Block) {
	for i := 0; i < length; i++ {
		block = self.GetBlock(block.ParentHash())
		if block == nil {
			break
		}

		blocks = append(blocks, block)
	}

	return
}

func (bc *ChainManager) setTotalDifficulty(td *big.Int) {
	bc.blockDb.Put([]byte("LTD"), td.Bytes())
	bc.td = td
}

func (self *ChainManager) CalcTotalDiff(block *types.Block) (*big.Int, error) {
	parent := self.GetBlock(block.Header().ParentHash)
	if parent == nil {
		return nil, fmt.Errorf("Unable to calculate total diff without known parent %x", block.Header().ParentHash)
	}

	parentTd := parent.Td

	uncleDiff := new(big.Int)
	for _, uncle := range block.Uncles() {
		uncleDiff = uncleDiff.Add(uncleDiff, uncle.Difficulty)
	}

	td := new(big.Int)
	td = td.Add(parentTd, uncleDiff)
	td = td.Add(td, block.Header().Difficulty)

	return td, nil
}

func (bc *ChainManager) Stop() {
	close(bc.quit)

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
	blocks := make([]*types.Block, len(self.futureBlocks.blocks))
	self.futureBlocks.Each(func(i int, block *types.Block) {
		blocks[i] = block
	})

	types.BlockBy(types.Number).Sort(blocks)
	self.InsertChain(blocks)
}

// InsertChain will attempt to insert the given chain in to the canonical chain or, otherwise, create a fork. It an error is returned
// it will return the index number of the failing block as well an error describing what went wrong (for possible errors see core/errors.go).
func (self *ChainManager) InsertChain(chain types.Blocks) (int, error) {
	self.wg.Add(1)
	defer self.wg.Done()

	// A queued approach to delivering events. This is generally faster than direct delivery and requires much less mutex acquiring.
	var (
		queue      = make([]interface{}, len(chain))
		queueEvent = queueEvent{queue: queue}
		stats      struct{ queued, processed, ignored int }
		tstart     = time.Now()
	)
	for i, block := range chain {
		if block == nil {
			continue
		}
		// Setting block.Td regardless of error (known for example) prevents errors down the line
		// in the protocol handler
		block.Td = new(big.Int).Set(CalculateTD(block, self.GetBlock(block.ParentHash())))

		// Call in to the block processor and check for errors. It's likely that if one block fails
		// all others will fail too (unless a known block is returned).
		logs, err := self.processor.Process(block)
		if err != nil {
			if IsKnownBlockErr(err) {
				stats.ignored++
				continue
			}

			block.Td = new(big.Int)
			// Do not penelise on future block. We'll need a block queue eventually that will queue
			// future block for future use
			if err == BlockFutureErr {
				block.SetQueued(true)
				self.futureBlocks.Push(block)
				stats.queued++
				continue
			}

			if IsParentErr(err) && self.futureBlocks.Has(block.ParentHash()) {
				block.SetQueued(true)
				self.futureBlocks.Push(block)
				stats.queued++
				continue
			}

			h := block.Header()

			glog.V(logger.Error).Infof("INVALID block #%v (%x)\n", h.Number, h.Hash().Bytes())
			glog.V(logger.Error).Infoln(err)
			glog.V(logger.Debug).Infoln(block)

			return i, err
		}

		self.mu.Lock()
		{
			cblock := self.currentBlock
			// Write block to database. Eventually we'll have to improve on this and throw away blocks that are
			// not in the canonical chain.
			self.write(block)
			// Compare the TD of the last known block in the canonical chain to make sure it's greater.
			// At this point it's possible that a different chain (fork) becomes the new canonical chain.
			if block.Td.Cmp(self.td) > 0 {
				// Check for chain forks. If H(block.num - 1) != block.parent, we're on a fork and need to do some merging
				if previous := self.getBlockByNumber(block.NumberU64() - 1); previous.Hash() != block.ParentHash() {
					// during split we merge two different chains and create the new canonical chain
					self.merge(previous, block)

					queue[i] = ChainSplitEvent{block, logs}
					queueEvent.splitCount++
				}

				self.setTotalDifficulty(block.Td)
				self.insert(block)

				jsonlogger.LogJson(&logger.EthChainNewHead{
					BlockHash:     block.Hash().Hex(),
					BlockNumber:   block.Number(),
					ChainHeadHash: cblock.Hash().Hex(),
					BlockPrevHash: block.ParentHash().Hex(),
				})

				self.setTransState(state.New(block.Root(), self.stateDb))
				self.txState.SetState(state.New(block.Root(), self.stateDb))

				queue[i] = ChainEvent{block, logs}
				queueEvent.canonicalCount++

				if glog.V(logger.Debug) {
					glog.Infof("[%v] inserted block #%d (%d TXs %d UNCs) (%x...)\n", time.Now().UnixNano(), block.Number(), len(block.Transactions()), len(block.Uncles()), block.Hash().Bytes()[0:4])
				}
			} else {
				if glog.V(logger.Detail) {
					glog.Infof("inserted forked block #%d (TD=%v) (%d TXs %d UNCs) (%x...)\n", block.Number(), block.Difficulty(), len(block.Transactions()), len(block.Uncles()), block.Hash().Bytes()[0:4])
				}

				queue[i] = ChainSideEvent{block, logs}
				queueEvent.sideCount++
			}
			self.futureBlocks.Delete(block.Hash())
		}
		self.mu.Unlock()

		stats.processed++

	}

	if (stats.queued > 0 || stats.processed > 0 || stats.ignored > 0) && bool(glog.V(logger.Info)) {
		tend := time.Since(tstart)
		start, end := chain[0], chain[len(chain)-1]
		glog.Infof("imported %d block(s) (%d queued %d ignored) in %v. #%v [%x / %x]\n", stats.processed, stats.queued, stats.ignored, tend, end.Number(), start.Hash().Bytes()[:4], end.Hash().Bytes()[:4])
	}

	go self.eventMux.Post(queueEvent)

	return 0, nil
}

// diff takes two blocks, an old chain and a new chain and will reconstruct the blocks and inserts them
// to be part of the new canonical chain.
func (self *ChainManager) diff(oldBlock, newBlock *types.Block) types.Blocks {
	var (
		newChain    types.Blocks
		commonBlock *types.Block
		oldStart    = oldBlock
		newStart    = newBlock
	)
	// first find common number
	for newBlock = newBlock; newBlock.NumberU64() != oldBlock.NumberU64(); newBlock = self.GetBlock(newBlock.ParentHash()) {
		newChain = append(newChain, newBlock)
	}

	numSplit := newBlock.Number()
	for {
		if oldBlock.Hash() == newBlock.Hash() {
			commonBlock = oldBlock
			break
		}
		newChain = append(newChain, newBlock)

		oldBlock, newBlock = self.GetBlock(oldBlock.ParentHash()), self.GetBlock(newBlock.ParentHash())
	}

	if glog.V(logger.Info) {
		commonHash := commonBlock.Hash()
		glog.Infof("Fork detected @ %x. Reorganising chain from #%v %x to %x", commonHash[:4], numSplit, oldStart.Hash().Bytes()[:4], newStart.Hash().Bytes()[:4])
	}

	return newChain
}

// merge merges two different chain to the new canonical chain
func (self *ChainManager) merge(oldBlock, newBlock *types.Block) {
	newChain := self.diff(oldBlock, newBlock)

	// insert blocks
	for _, block := range newChain {
		self.insert(block)
	}
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
				for i, event := range ev.queue {
					switch event := event.(type) {
					case ChainEvent:
						// We need some control over the mining operation. Acquiring locks and waiting for the miner to create new block takes too long
						// and in most cases isn't even necessary.
						if i+1 == ev.canonicalCount {
							self.currentGasLimit = CalcGasLimit(event.Block)
							self.eventMux.Post(ChainHeadEvent{event.Block})
						}
					case ChainSplitEvent:
						// On chain splits we need to reset the transaction state. We can't be sure whether the actual
						// state of the accounts are still valid.
						if i == ev.splitCount {
							self.setTxState(state.New(event.Block.Root(), self.stateDb))
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
