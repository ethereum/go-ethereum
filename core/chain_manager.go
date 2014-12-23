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

/*
func AddTestNetFunds(block *types.Block) {
	for _, addr := range []string{
		"51ba59315b3a95761d0863b05ccc7a7f54703d99",
		"e4157b34ea9615cfbde6b4fda419828124b70c78",
		"b9c015918bdaba24b4ff057a92a3873d6eb201be",
		"6c386a4b26f73c802f34673f7248bb118f97424a",
		"cd2a3d9f938e13cd947ec05abc7fe734df8dd826",
		"2ef47100e0787b915105fd5e3f4ff6752079d5cb",
		"e6716f9544a56c530d868e4bfbacb172315bdead",
		"1a26338f0d905e295fccb71fa9ea849ffa12aaf4",
	} {
		codedAddr := ethutil.Hex2Bytes(addr)
		account := block.State().GetAccount(codedAddr)
		account.SetBalance(ethutil.Big("1606938044258990275541962092341162602522202993782792835301376")) //ethutil.BigPow(2, 200)
		block.State().UpdateStateObject(account)
	}
}
*/

func CalcDifficulty(block, parent *types.Block) *big.Int {
	diff := new(big.Int)

	bh, ph := block.Header(), parent.Header()
	adjust := new(big.Int).Rsh(ph.Difficulty, 10)
	if bh.Time >= ph.Time+5 {
		diff.Sub(ph.Difficulty, adjust)
	} else {
		diff.Add(ph.Difficulty, adjust)
	}

	return diff
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
	processor    types.BlockProcessor
	eventMux     *event.TypeMux
	genesisBlock *types.Block
	// Last known total difficulty
	mu              sync.RWMutex
	td              *big.Int
	lastBlockNumber uint64
	currentBlock    *types.Block
	lastBlockHash   []byte

	transState *state.StateDB
}

func (self *ChainManager) Td() *big.Int {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.td
}

func (self *ChainManager) LastBlockNumber() uint64 {
	self.mu.RLock()
	defer self.mu.RUnlock()

	return self.lastBlockNumber
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

func NewChainManager(mux *event.TypeMux) *ChainManager {
	bc := &ChainManager{}
	bc.genesisBlock = GenesisBlock()
	bc.eventMux = mux

	bc.setLastBlock()

	bc.transState = bc.State().Copy()

	return bc
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
	return state.New(self.CurrentBlock().Trie())
}

func (self *ChainManager) TransState() *state.StateDB {
	return self.transState
}

func (bc *ChainManager) setLastBlock() {
	data, _ := ethutil.Config.Db.Get([]byte("LastBlock"))
	if len(data) != 0 {
		var block types.Block
		rlp.Decode(bytes.NewReader(data), &block)
		bc.currentBlock = &block
		bc.lastBlockHash = block.Hash()
		bc.lastBlockNumber = block.Header().Number.Uint64()

		// Set the last know difficulty (might be 0x0 as initial value, Genesis)
		bc.td = ethutil.BigD(ethutil.Config.Db.LastKnownTD())
	} else {
		bc.Reset()
	}

	chainlogger.Infof("Last block (#%d) %x\n", bc.lastBlockNumber, bc.currentBlock.Hash())
}

// Block creation & chain handling
func (bc *ChainManager) NewBlock(coinbase []byte) *types.Block {
	bc.mu.RLock()
	defer bc.mu.RUnlock()

	var root []byte
	parentHash := ZeroHash256

	if bc.CurrentBlock != nil {
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

	// Prepare the genesis block
	bc.write(bc.genesisBlock)
	bc.insert(bc.genesisBlock)
	bc.currentBlock = bc.genesisBlock

	bc.setTotalDifficulty(ethutil.Big("0"))

	// Set the last know difficulty (might be 0x0 as initial value, Genesis)
	bc.td = ethutil.BigD(ethutil.Config.Db.LastKnownTD())
}

func (self *ChainManager) Export() []byte {
	self.mu.RLock()
	defer self.mu.RUnlock()

	chainlogger.Infof("exporting %v blocks...\n", self.currentBlock.Header().Number)

	blocks := make([]*types.Block, int(self.currentBlock.NumberU64())+1)
	for block := self.currentBlock; block != nil; block = self.GetBlock(block.Header().ParentHash) {
		blocks[block.NumberU64()] = block
	}
	//fmt.Println(blocks)

	return nil
	return ethutil.Encode(blocks)
}

func (bc *ChainManager) insert(block *types.Block) {
	encodedBlock := ethutil.Encode(block)
	ethutil.Config.Db.Put([]byte("LastBlock"), encodedBlock)
	bc.currentBlock = block
	bc.lastBlockHash = block.Hash()
}

func (bc *ChainManager) write(block *types.Block) {
	bc.writeBlockInfo(block)

	encodedBlock := ethutil.Encode(block)
	ethutil.Config.Db.Put(block.Hash(), encodedBlock)
}

// Accessors
func (bc *ChainManager) Genesis() *types.Block {
	return bc.genesisBlock
}

// Block fetching methods
func (bc *ChainManager) HasBlock(hash []byte) bool {
	data, _ := ethutil.Config.Db.Get(hash)
	return len(data) != 0
}

func (self *ChainManager) GetBlockHashesFromHash(hash []byte, max uint64) (chain [][]byte) {
	block := self.GetBlock(hash)
	if block == nil {
		return
	}

	// XXX Could be optimised by using a different database which only holds hashes (i.e., linked list)
	for i := uint64(0); i < max; i++ {
		chain = append(chain, block.Hash())

		if block.Header().Number.Cmp(ethutil.Big0) <= 0 {
			break
		}

		block = self.GetBlock(block.Header().ParentHash)
	}

	return
}

func (self *ChainManager) GetBlock(hash []byte) *types.Block {
	data, _ := ethutil.Config.Db.Get(hash)
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

func (self *ChainManager) GetBlockByNumber(num uint64) *types.Block {
	self.mu.RLock()
	defer self.mu.RUnlock()

	block := self.currentBlock
	for ; block != nil; block = self.GetBlock(block.Header().ParentHash) {
		if block.Header().Number.Uint64() == num {
			break
		}
	}

	if block != nil && block.Header().Number.Uint64() == 0 && num != 0 {
		return nil
	}

	return block
}

func (bc *ChainManager) setTotalDifficulty(td *big.Int) {
	ethutil.Config.Db.Put([]byte("LTD"), td.Bytes())
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

// Unexported method for writing extra non-essential block info to the db
func (bc *ChainManager) writeBlockInfo(block *types.Block) {
	bc.lastBlockNumber++
}

func (bc *ChainManager) Stop() {
	if bc.CurrentBlock != nil {
		chainlogger.Infoln("Stopped")
	}
}

func (self *ChainManager) InsertChain(chain types.Blocks) error {
	for _, block := range chain {
		td, messages, err := self.processor.Process(block)
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

		self.mu.Lock()
		{
			self.write(block)
			cblock := self.currentBlock
			if td.Cmp(self.td) > 0 {
				if block.Header().Number.Cmp(new(big.Int).Add(cblock.Header().Number, ethutil.Big1)) < 0 {
					chainlogger.Infof("Split detected. New head #%v (%x), was #%v (%x)\n", block.Header().Number, block.Hash()[:4], cblock.Header().Number, cblock.Hash()[:4])
				}

				self.setTotalDifficulty(td)
				self.insert(block)
				self.transState = state.New(cblock.Trie().Copy())
			}

		}
		self.mu.Unlock()

		self.eventMux.Post(NewBlockEvent{block})
		self.eventMux.Post(messages)
	}

	return nil
}
