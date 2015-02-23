package core

import (
	"bytes"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/state"
)

var chainlogger = logger.NewLogger("CHAIN")

type ChainEvent struct {
	Block *types.Block
	Td    *big.Int
}

type StateQuery interface {
	GetAccount(addr []byte) *state.StateObject
}

func CalcDifficulty(block, parent *types.Block) *big.Int {
	diff := new(big.Int)

	adjust := new(big.Int).Rsh(parent.Difficulty(), 10)
	if block.Time() >= parent.Time()+8 {
		diff.Sub(parent.Difficulty(), adjust)
	} else {
		diff.Add(parent.Difficulty(), adjust)
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
		return ethutil.BigPow(10, 6)
	}

	// ((1024-1) * parent.gasLimit + (gasUsed * 6 / 5)) / 1024

	previous := new(big.Int).Mul(big.NewInt(1024-1), parent.GasLimit())
	current := new(big.Rat).Mul(new(big.Rat).SetInt(parent.GasUsed()), big.NewRat(6, 5))
	curInt := new(big.Int).Div(current.Num(), current.Denom())

	result := new(big.Int).Add(previous, curInt)
	result.Div(result, big.NewInt(1024))

	min := big.NewInt(125000)

	return ethutil.BigMax(min, result)
}

type ChainManager struct {
	//eth          EthManager
	db           ethutil.Database
	processor    types.BlockProcessor
	eventMux     *event.TypeMux
	genesisBlock *types.Block
	// Last known total difficulty
	mu            sync.RWMutex
	tsmu          sync.RWMutex
	td            *big.Int
	currentBlock  *types.Block
	lastBlockHash []byte

	transState *state.StateDB
	txState    *state.StateDB
}

func NewChainManager(db ethutil.Database, mux *event.TypeMux) *ChainManager {
	bc := &ChainManager{db: db, genesisBlock: GenesisBlock(db), eventMux: mux}
	bc.setLastBlock()
	bc.transState = bc.State().Copy()
	bc.txState = bc.State().Copy()

	return bc
}

func (self *ChainManager) Td() *big.Int {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.td
}

func (self *ChainManager) LastBlockHash() []byte {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.lastBlockHash
}

func (self *ChainManager) CurrentBlock() *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.currentBlock
}

func (self *ChainManager) Status() (td *big.Int, currentBlock []byte, genesisBlock []byte) {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.td, self.currentBlock.Hash(), self.Genesis().Hash()
}

func (self *ChainManager) SetProcessor(proc types.BlockProcessor) {
	self.processor = proc
}

func (self *ChainManager) State() *state.StateDB {
	return state.New(self.CurrentBlock().Root(), self.db)
}

func (self *ChainManager) TransState() *state.StateDB {
	self.tsmu.RLock()
	defer self.tsmu.RUnlock()

	return self.transState
}

func (self *ChainManager) TxState() *state.StateDB {
	self.tsmu.RLock()
	defer self.tsmu.RUnlock()

	return self.txState
}

func (self *ChainManager) setTxState(state *state.StateDB) {
	self.tsmu.Lock()
	defer self.tsmu.Unlock()
	self.txState = state
}

func (self *ChainManager) setTransState(statedb *state.StateDB) {
	self.transState = statedb
}

func (bc *ChainManager) setLastBlock() {
	data, _ := bc.db.Get([]byte("LastBlock"))
	if len(data) != 0 {
		var block types.Block
		rlp.Decode(bytes.NewReader(data), &block)
		bc.currentBlock = &block
		bc.lastBlockHash = block.Hash()

		// Set the last know difficulty (might be 0x0 as initial value, Genesis)
		bc.td = ethutil.BigD(bc.db.LastKnownTD())
	} else {
		bc.Reset()
	}

	chainlogger.Infof("Last block (#%v) %x TD=%v\n", bc.currentBlock.Number(), bc.currentBlock.Hash(), bc.td)
}

// Block creation & chain handling
func (bc *ChainManager) NewBlock(coinbase []byte) *types.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	var root []byte
	parentHash := ZeroHash256

	if bc.currentBlock != nil {
		root = bc.currentBlock.Header().Root
		parentHash = bc.lastBlockHash
	}

	block := types.NewBlock(
		parentHash,
		coinbase,
		root,
		ethutil.BigPow(2, 32),
		nil,
		"")
	block.SetUncles(nil)
	block.SetTransactions(nil)
	block.SetReceipts(nil)

	parent := bc.currentBlock
	if parent != nil {
		header := block.Header()
		header.Difficulty = CalcDifficulty(block, parent)
		header.Number = new(big.Int).Add(parent.Header().Number, ethutil.Big1)
		header.GasLimit = CalcGasLimit(parent, block)

	}

	return block
}

func (bc *ChainManager) Reset() {
	bc.mu.Lock()
	defer bc.mu.Unlock()

	for block := bc.currentBlock; block != nil; block = bc.GetBlock(block.Header().ParentHash) {
		bc.db.Delete(block.Hash())
	}

	// Prepare the genesis block
	bc.write(bc.genesisBlock)
	bc.insert(bc.genesisBlock)
	bc.currentBlock = bc.genesisBlock

	bc.setTotalDifficulty(ethutil.Big("0"))
}

func (self *ChainManager) Export() []byte {
	self.mu.RLock()
	defer self.mu.RUnlock()

	chainlogger.Infof("exporting %v blocks...\n", self.currentBlock.Header().Number)

	blocks := make([]*types.Block, int(self.currentBlock.NumberU64())+1)
	for block := self.currentBlock; block != nil; block = self.GetBlock(block.Header().ParentHash) {
		blocks[block.NumberU64()] = block
	}

	return ethutil.Encode(blocks)
}

func (bc *ChainManager) insert(block *types.Block) {
	encodedBlock := ethutil.Encode(block)
	bc.db.Put([]byte("LastBlock"), encodedBlock)
	bc.currentBlock = block
	bc.lastBlockHash = block.Hash()
}

func (bc *ChainManager) write(block *types.Block) {
	encodedBlock := ethutil.Encode(block.RlpDataForStorage())
	bc.db.Put(block.Hash(), encodedBlock)
}

// Accessors
func (bc *ChainManager) Genesis() *types.Block {
	return bc.genesisBlock
}

// Block fetching methods
func (bc *ChainManager) HasBlock(hash []byte) bool {
	data, _ := bc.db.Get(hash)
	return len(data) != 0
}

func (self *ChainManager) GetBlockHashesFromHash(hash []byte, max uint64) (chain [][]byte) {
	block := self.GetBlock(hash)
	if block == nil {
		return
	}

	// XXX Could be optimised by using a different database which only holds hashes (i.e., linked list)
	for i := uint64(0); i < max; i++ {
		parentHash := block.Header().ParentHash
		block = self.GetBlock(parentHash)
		if block == nil {
			chainlogger.Infof("GetBlockHashesFromHash Parent UNKNOWN %x\n", parentHash)
			break
		}

		chain = append(chain, block.Hash())
		if block.Header().Number.Cmp(ethutil.Big0) <= 0 {
			break
		}
	}

	return
}

func (self *ChainManager) GetBlock(hash []byte) *types.Block {
	data, _ := self.db.Get(hash)
	if len(data) == 0 {
		return nil
	}
	var block types.Block
	if err := rlp.Decode(bytes.NewReader(data), &block); err != nil {
		fmt.Println(err)
		return nil
	}

	return &block
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

func (self *ChainManager) GetBlockByNumber(num uint64) *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

	var block *types.Block

	if num <= self.currentBlock.Number().Uint64() {
		block = self.currentBlock
		for ; block != nil; block = self.GetBlock(block.Header().ParentHash) {
			if block.Header().Number.Uint64() == num {
				break
			}
		}
	}

	return block
}

func (bc *ChainManager) setTotalDifficulty(td *big.Int) {
	bc.db.Put([]byte("LTD"), td.Bytes())
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
	if bc.CurrentBlock != nil {
		chainlogger.Infoln("Stopped")
	}
}

func (self *ChainManager) InsertChain(chain types.Blocks) error {
	self.tsmu.Lock()
	defer self.tsmu.Unlock()

	for _, block := range chain {
		// Call in to the block processor and check for errors. It's likely that if one block fails
		// all others will fail too (unless a known block is returned).
		td, err := self.processor.Process(block)
		if err != nil {
			if IsKnownBlockErr(err) {
				continue
			}

			h := block.Header()
			chainlogger.Infof("block #%v process failed (%x)\n", h.Number, h.Hash()[:4])
			chainlogger.Infoln(block)
			chainlogger.Infoln(err)
			return err
		}
		block.Td = td

		var canonical, split bool
		self.mu.Lock()
		{
			// Write block to database. Eventually we'll have to improve on this and throw away blocks that are
			// not in the canonical chain.
			self.write(block)
			cblock := self.currentBlock
			// Compare the TD of the last known block in the canonical chain to make sure it's greater.
			// At this point it's possible that a different chain (fork) becomes the new canonical chain.
			if td.Cmp(self.td) > 0 {
				if block.Header().Number.Cmp(new(big.Int).Add(cblock.Header().Number, ethutil.Big1)) < 0 {
					chainlogger.Infof("Split detected. New head #%v (%x) TD=%v, was #%v (%x) TD=%v\n", block.Header().Number, block.Hash()[:4], td, cblock.Header().Number, cblock.Hash()[:4], self.td)
					split = true
				}

				self.setTotalDifficulty(td)
				self.insert(block)

				canonical = true
			}
		}
		self.mu.Unlock()

		if canonical {
			self.setTransState(state.New(block.Root(), self.db))
			self.eventMux.Post(ChainEvent{block, td})
		}

		if split {
			self.setTxState(state.New(block.Root(), self.db))
			self.eventMux.Post(ChainSplitEvent{block})
		}
	}

	return nil
}

// Satisfy state query interface
func (self *ChainManager) GetAccount(addr []byte) *state.StateObject {
	return self.State().GetAccount(addr)
}
