package core

import (
	"bytes"
	"fmt"
	"io"
	"math/big"
	"sync"

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

const blockCacheLimit = 10000

type StateQuery interface {
	GetAccount(addr []byte) *state.StateObject
}

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
	uncleDiff := new(big.Int)
	for _, uncle := range block.Uncles() {
		uncleDiff = uncleDiff.Add(uncleDiff, uncle.Difficulty)
	}

	// TD(genesis_block) = 0 and TD(B) = TD(B.parent) + sum(u.difficulty for u in B.uncles) + B.difficulty
	td := new(big.Int)
	td = td.Add(parent.Td, uncleDiff)
	td = td.Add(td, block.Header().Difficulty)

	return td
}

func CalcGasLimit(parent, block *types.Block) *big.Int {
	if block.Number().Cmp(big.NewInt(0)) == 0 {
		return common.BigPow(10, 6)
	}

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
	mu            sync.RWMutex
	tsmu          sync.RWMutex
	td            *big.Int
	currentBlock  *types.Block
	lastBlockHash common.Hash

	transState *state.StateDB
	txState    *state.ManagedState

	cache *BlockCache

	quit chan struct{}
}

func NewChainManager(blockDb, stateDb common.Database, mux *event.TypeMux) *ChainManager {
	bc := &ChainManager{blockDb: blockDb, stateDb: stateDb, genesisBlock: GenesisBlock(stateDb), eventMux: mux, quit: make(chan struct{}), cache: NewBlockCache(blockCacheLimit)}
	bc.setLastBlock()
	bc.transState = bc.State().Copy()
	// Take ownership of this particular state
	bc.txState = state.ManageState(bc.State().Copy())

	bc.makeCache()

	go bc.update()

	return bc
}

func (self *ChainManager) Td() *big.Int {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.td
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

func (bc *ChainManager) setLastBlock() {
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
		"")
	block.SetUncles(nil)
	block.SetTransactions(nil)
	block.SetReceipts(nil)

	parent := bc.currentBlock
	if parent != nil {
		header := block.Header()
		header.Difficulty = CalcDifficulty(block.Header(), parent.Header())
		header.Number = new(big.Int).Add(parent.Header().Number, common.Big1)
		header.GasLimit = CalcGasLimit(parent, block)

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
	bc.genesisBlock = gb
	bc.write(bc.genesisBlock)
	bc.insert(bc.genesisBlock)
	bc.currentBlock = bc.genesisBlock
	bc.makeCache()
}

// Export writes the active chain to the given writer.
func (self *ChainManager) Export(w io.Writer) error {
	self.mu.RLock()
	defer self.mu.RUnlock()
	glog.V(logger.Info).Infof("exporting %v blocks...\n", self.currentBlock.Header().Number)
	for block := self.currentBlock; block != nil; block = self.GetBlock(block.Header().ParentHash) {
		if err := block.EncodeRLP(w); err != nil {
			return err
		}
	}
	return nil
}

func (bc *ChainManager) insert(block *types.Block) {
	bc.blockDb.Put([]byte("LastBlock"), block.Hash().Bytes())
	bc.currentBlock = block
	bc.lastBlockHash = block.Hash()

	key := append(blockNumPre, block.Number().Bytes()...)
	bc.blockDb.Put(key, bc.lastBlockHash.Bytes())
	// Push block to cache
	bc.cache.Push(block)
}

func (bc *ChainManager) write(block *types.Block) {
	enc, _ := rlp.EncodeToBytes((*types.StorageBlock)(block))
	key := append(blockHashPre, block.Hash().Bytes()...)
	bc.blockDb.Put(key, enc)
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
		parentHash := block.Header().ParentHash
		block = self.GetBlock(parentHash)
		if block == nil {
			break
		}

		chain = append(chain, block.Hash())
		if block.Header().Number.Cmp(common.Big0) <= 0 {
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
		chainlogger.Errorf("invalid block RLP for hash %x: %v", hash, err)
		return nil
	}
	return (*types.Block)(&block)
}

func (self *ChainManager) GetBlockByNumber(num uint64) *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

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
}

type queueEvent struct {
	queue          []interface{}
	canonicalCount int
	sideCount      int
	splitCount     int
}

func (self *ChainManager) InsertChain(chain types.Blocks) error {
	//self.tsmu.Lock()
	//defer self.tsmu.Unlock()

	// A queued approach to delivering events. This is generally faster than direct delivery and requires much less mutex acquiring.
	var queue = make([]interface{}, len(chain))
	var queueEvent = queueEvent{queue: queue}
	for i, block := range chain {
		if block == nil {
			continue
		}
		// Call in to the block processor and check for errors. It's likely that if one block fails
		// all others will fail too (unless a known block is returned).
		td, logs, err := self.processor.Process(block)
		if err != nil {
			if IsKnownBlockErr(err) {
				continue
			}

			if err == BlockEqualTSErr {
				//queue[i] = ChainSideEvent{block, logs}
				// XXX silently discard it?
				continue
			}

			h := block.Header()
			chainlogger.Errorf("INVALID block #%v (%x)\n", h.Number, h.Hash().Bytes()[:4])
			chainlogger.Errorln(err)
			chainlogger.Debugln(block)
			return err
		}
		block.Td = td

		self.mu.Lock()
		cblock := self.currentBlock
		{
			// Write block to database. Eventually we'll have to improve on this and throw away blocks that are
			// not in the canonical chain.
			self.write(block)
			// Compare the TD of the last known block in the canonical chain to make sure it's greater.
			// At this point it's possible that a different chain (fork) becomes the new canonical chain.
			if td.Cmp(self.td) > 0 {
				if block.Header().Number.Cmp(new(big.Int).Add(cblock.Header().Number, common.Big1)) < 0 {
					chash := cblock.Hash()
					hash := block.Hash()

					if glog.V(logger.Info) {
						glog.Infof("Split detected. New head #%v (%x) TD=%v, was #%v (%x) TD=%v\n", block.Header().Number, hash[:4], td, cblock.Header().Number, chash[:4], self.td)
					}

					queue[i] = ChainSplitEvent{block, logs}
					queueEvent.splitCount++
				}

				self.setTotalDifficulty(td)
				self.insert(block)

				jsonlogger.LogJson(&logger.EthChainNewHead{
					BlockHash:     block.Hash().Hex(),
					BlockNumber:   block.Number(),
					ChainHeadHash: cblock.Hash().Hex(),
					BlockPrevHash: block.ParentHash().Hex(),
				})

				self.setTransState(state.New(block.Root(), self.stateDb))
				self.setTxState(state.New(block.Root(), self.stateDb))

				queue[i] = ChainEvent{block, logs}
				queueEvent.canonicalCount++

				if glog.V(logger.Debug) {
					glog.Infof("inserted block #%d (%d TXs %d UNCs) (%x...)\n", block.Number(), len(block.Transactions()), len(block.Uncles()), block.Hash().Bytes()[0:4])
				}
			} else {
				queue[i] = ChainSideEvent{block, logs}
				queueEvent.sideCount++
			}
		}
		self.mu.Unlock()

	}

	if len(chain) > 0 && glog.V(logger.Info) {
		start, end := chain[0], chain[len(chain)-1]
		glog.Infof("imported %d blocks #%v [%x / %x]\n", len(chain), end.Number(), start.Hash().Bytes()[:4], end.Hash().Bytes()[:4])
	}

	go self.eventMux.Post(queueEvent)

	return nil
}

func (self *ChainManager) update() {
	events := self.eventMux.Subscribe(queueEvent{})

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
		case <-self.quit:
			break out
		}
	}
}
