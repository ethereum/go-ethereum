package chain

import (
	"bytes"
	"container/list"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/chain/types"
	"github.com/ethereum/go-ethereum/ethutil"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/state"
)

var chainlogger = logger.NewLogger("CHAIN")

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

func CalcDifficulty(block, parent *types.Block) *big.Int {
	diff := new(big.Int)

	adjust := new(big.Int).Rsh(parent.Difficulty, 10)
	if block.Time >= parent.Time+5 {
		diff.Sub(parent.Difficulty, adjust)
	} else {
		diff.Add(parent.Difficulty, adjust)
	}

	return diff
}

type ChainManager struct {
	//eth          EthManager
	processor    types.BlockProcessor
	genesisBlock *types.Block
	// Last known total difficulty
	TD *big.Int

	LastBlockNumber uint64

	CurrentBlock  *types.Block
	LastBlockHash []byte

	workingChain *BlockChain
}

func NewChainManager() *ChainManager {
	bc := &ChainManager{}
	bc.genesisBlock = types.NewBlockFromBytes(ethutil.Encode(Genesis))
	//bc.eth = ethereum

	bc.setLastBlock()

	return bc
}

func (self *ChainManager) SetProcessor(proc types.BlockProcessor) {
	self.processor = proc
}

func (bc *ChainManager) setLastBlock() {
	data, _ := ethutil.Config.Db.Get([]byte("LastBlock"))
	if len(data) != 0 {
		// Prep genesis
		AddTestNetFunds(bc.genesisBlock)

		block := types.NewBlockFromBytes(data)
		bc.CurrentBlock = block
		bc.LastBlockHash = block.Hash()
		bc.LastBlockNumber = block.Number.Uint64()

		// Set the last know difficulty (might be 0x0 as initial value, Genesis)
		bc.TD = ethutil.BigD(ethutil.Config.Db.LastKnownTD())
	} else {
		bc.Reset()
	}

	chainlogger.Infof("Last block (#%d) %x\n", bc.LastBlockNumber, bc.CurrentBlock.Hash())
}

// Block creation & chain handling
func (bc *ChainManager) NewBlock(coinbase []byte) *types.Block {
	var root interface{}
	hash := ZeroHash256

	if bc.CurrentBlock != nil {
		root = bc.CurrentBlock.Root()
		hash = bc.LastBlockHash
	}

	block := types.CreateBlock(
		root,
		hash,
		coinbase,
		ethutil.BigPow(2, 32),
		nil,
		"")

	block.MinGasPrice = big.NewInt(10000000000000)

	parent := bc.CurrentBlock
	if parent != nil {
		block.Difficulty = CalcDifficulty(block, parent)
		block.Number = new(big.Int).Add(bc.CurrentBlock.Number, ethutil.Big1)
		block.GasLimit = block.CalcGasLimit(bc.CurrentBlock)

	}

	return block
}

func (bc *ChainManager) Reset() {
	AddTestNetFunds(bc.genesisBlock)

	bc.genesisBlock.Trie().Sync()
	// Prepare the genesis block
	bc.add(bc.genesisBlock)
	bc.CurrentBlock = bc.genesisBlock

	bc.SetTotalDifficulty(ethutil.Big("0"))

	// Set the last know difficulty (might be 0x0 as initial value, Genesis)
	bc.TD = ethutil.BigD(ethutil.Config.Db.LastKnownTD())
}

// Add a block to the chain and record addition information
func (bc *ChainManager) add(block *types.Block) {
	bc.writeBlockInfo(block)

	bc.CurrentBlock = block
	bc.LastBlockHash = block.Hash()

	encodedBlock := block.RlpEncode()
	ethutil.Config.Db.Put(block.Hash(), encodedBlock)
	ethutil.Config.Db.Put([]byte("LastBlock"), encodedBlock)

	//chainlogger.Infof("Imported block #%d (%x...)\n", block.Number, block.Hash()[0:4])
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

func (self *ChainManager) GetChainHashesFromHash(hash []byte, max uint64) (chain [][]byte) {
	block := self.GetBlock(hash)
	if block == nil {
		return
	}

	// XXX Could be optimised by using a different database which only holds hashes (i.e., linked list)
	for i := uint64(0); i < max; i++ {

		chain = append(chain, block.Hash())

		if block.Number.Cmp(ethutil.Big0) <= 0 {
			break
		}

		block = self.GetBlock(block.PrevHash)
	}

	return
}

func (self *ChainManager) GetBlock(hash []byte) *types.Block {
	data, _ := ethutil.Config.Db.Get(hash)
	if len(data) == 0 {
		if self.workingChain != nil {
			// Check the temp chain
			for e := self.workingChain.Front(); e != nil; e = e.Next() {
				if bytes.Compare(e.Value.(*link).block.Hash(), hash) == 0 {
					return e.Value.(*link).block
				}
			}
		}

		return nil
	}

	return types.NewBlockFromBytes(data)
}

func (self *ChainManager) GetBlockByNumber(num uint64) *types.Block {
	block := self.CurrentBlock
	for ; block != nil; block = self.GetBlock(block.PrevHash) {
		if block.Number.Uint64() == num {
			break
		}
	}

	if block != nil && block.Number.Uint64() == 0 && num != 0 {
		return nil
	}

	return block
}

func (bc *ChainManager) SetTotalDifficulty(td *big.Int) {
	ethutil.Config.Db.Put([]byte("LTD"), td.Bytes())
	bc.TD = td
}

func (self *ChainManager) CalcTotalDiff(block *types.Block) (*big.Int, error) {
	parent := self.GetBlock(block.PrevHash)
	if parent == nil {
		return nil, fmt.Errorf("Unable to calculate total diff without known parent %x", block.PrevHash)
	}

	parentTd := parent.BlockInfo().TD

	uncleDiff := new(big.Int)
	for _, uncle := range block.Uncles {
		uncleDiff = uncleDiff.Add(uncleDiff, uncle.Difficulty)
	}

	td := new(big.Int)
	td = td.Add(parentTd, uncleDiff)
	td = td.Add(td, block.Difficulty)

	return td, nil
}

func (bc *ChainManager) BlockInfo(block *types.Block) types.BlockInfo {
	bi := types.BlockInfo{}
	data, _ := ethutil.Config.Db.Get(append(block.Hash(), []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

// Unexported method for writing extra non-essential block info to the db
func (bc *ChainManager) writeBlockInfo(block *types.Block) {
	bc.LastBlockNumber++
	bi := types.BlockInfo{Number: bc.LastBlockNumber, Hash: block.Hash(), Parent: block.PrevHash, TD: bc.TD}

	// For now we use the block hash with the words "info" appended as key
	ethutil.Config.Db.Put(append(block.Hash(), []byte("Info")...), bi.RlpEncode())
}

func (bc *ChainManager) Stop() {
	if bc.CurrentBlock != nil {
		chainlogger.Infoln("Stopped")
	}
}

func (self *ChainManager) NewIterator(startHash []byte) *ChainIterator {
	return &ChainIterator{self, self.GetBlock(startHash)}
}

// This function assumes you've done your checking. No checking is done at this stage anymore
func (self *ChainManager) InsertChain(chain *BlockChain) {
	for e := chain.Front(); e != nil; e = e.Next() {
		link := e.Value.(*link)

		self.add(link.block)
		self.SetTotalDifficulty(link.td)
		//self.eth.EventMux().Post(NewBlockEvent{link.block})
		//self.eth.EventMux().Post(link.messages)
	}

	b, e := chain.Front(), chain.Back()
	if b != nil && e != nil {
		front, back := b.Value.(*link).block, e.Value.(*link).block
		chainlogger.Infof("Imported %d blocks. #%v (%x) / %#v (%x)", chain.Len(), front.Number, front.Hash()[0:4], back.Number, back.Hash()[0:4])
	}
}

func (self *ChainManager) TestChain(chain *BlockChain) (td *big.Int, err error) {
	self.workingChain = chain
	defer func() { self.workingChain = nil }()

	for e := chain.Front(); e != nil; e = e.Next() {
		var (
			l      = e.Value.(*link)
			block  = l.block
			parent = self.GetBlock(block.PrevHash)
		)

		if parent == nil {
			err = fmt.Errorf("incoming chain broken on hash %x\n", block.PrevHash[0:4])
			return
		}

		var messages state.Messages
		td, messages, err = self.processor.ProcessWithParent(block, parent) //self.eth.BlockManager().ProcessWithParent(block, parent)
		if err != nil {
			chainlogger.Infoln(err)
			chainlogger.Debugf("Block #%v failed (%x...)\n", block.Number, block.Hash()[0:4])
			chainlogger.Debugln(block)

			err = fmt.Errorf("incoming chain failed %v\n", err)
			return
		}
		l.td = td
		l.messages = messages
	}

	if td.Cmp(self.TD) <= 0 {
		err = &TDError{td, self.TD}
		return
	}

	self.workingChain = nil

	return
}

type link struct {
	block    *types.Block
	messages state.Messages
	td       *big.Int
}

type BlockChain struct {
	*list.List
}

func NewChain(blocks types.Blocks) *BlockChain {
	chain := &BlockChain{list.New()}

	for _, block := range blocks {
		chain.PushBack(&link{block, nil, nil})
	}

	return chain
}

func (self *BlockChain) RlpEncode() []byte {
	dat := make([]interface{}, 0)
	for e := self.Front(); e != nil; e = e.Next() {
		dat = append(dat, e.Value.(*link).block.RlpData())
	}

	return ethutil.Encode(dat)
}

type ChainIterator struct {
	cm    *ChainManager
	block *types.Block // current block in the iterator
}

func (self *ChainIterator) Prev() *types.Block {
	self.block = self.cm.GetBlock(self.block.PrevHash)
	return self.block
}
