package ethchain

import (
	"bytes"
	"github.com/ethereum/eth-go/ethlog"
	"github.com/ethereum/eth-go/ethutil"
	"github.com/ethereum/eth-go/ethwire"
	"math"
	"math/big"
)

var chainlogger = ethlog.NewLogger("CHAIN")

type BlockChain struct {
	Ethereum EthManager
	// The famous, the fabulous Mister GENESIIIIIIS (block)
	genesisBlock *Block
	// Last known total difficulty
	TD *big.Int

	LastBlockNumber uint64

	CurrentBlock  *Block
	LastBlockHash []byte
}

func NewBlockChain(ethereum EthManager) *BlockChain {
	bc := &BlockChain{}
	bc.genesisBlock = NewBlockFromBytes(ethutil.Encode(Genesis))
	bc.Ethereum = ethereum

	bc.setLastBlock()

	return bc
}

func (bc *BlockChain) Genesis() *Block {
	return bc.genesisBlock
}

func (bc *BlockChain) NewBlock(coinbase []byte) *Block {
	var root interface{}
	var lastBlockTime int64
	hash := ZeroHash256

	if bc.CurrentBlock != nil {
		root = bc.CurrentBlock.state.trie.Root
		hash = bc.LastBlockHash
		lastBlockTime = bc.CurrentBlock.Time
	}

	block := CreateBlock(
		root,
		hash,
		coinbase,
		ethutil.BigPow(2, 32),
		nil,
		"")

	block.MinGasPrice = big.NewInt(10000000000000)

	if bc.CurrentBlock != nil {
		var mul *big.Int
		if block.Time < lastBlockTime+42 {
			mul = big.NewInt(1)
		} else {
			mul = big.NewInt(-1)
		}

		diff := new(big.Int)
		diff.Add(diff, bc.CurrentBlock.Difficulty)
		diff.Div(diff, big.NewInt(1024))
		diff.Mul(diff, mul)
		diff.Add(diff, bc.CurrentBlock.Difficulty)
		block.Difficulty = diff

		block.Number = new(big.Int).Add(bc.CurrentBlock.Number, ethutil.Big1)

		block.GasLimit = block.CalcGasLimit(bc.CurrentBlock)
	}

	return block
}

func (bc *BlockChain) HasBlock(hash []byte) bool {
	data, _ := ethutil.Config.Db.Get(hash)
	return len(data) != 0
}

// TODO: At one point we might want to save a block by prevHash in the db to optimise this...
func (bc *BlockChain) HasBlockWithPrevHash(hash []byte) bool {
	block := bc.CurrentBlock

	for ; block != nil; block = bc.GetBlock(block.PrevHash) {
		if bytes.Compare(hash, block.PrevHash) == 0 {
			return true
		}
	}
	return false
}

func (bc *BlockChain) CalculateBlockTD(block *Block) *big.Int {
	blockDiff := new(big.Int)

	for _, uncle := range block.Uncles {
		blockDiff = blockDiff.Add(blockDiff, uncle.Difficulty)
	}
	blockDiff = blockDiff.Add(blockDiff, block.Difficulty)

	return blockDiff
}
func (bc *BlockChain) FindCanonicalChainFromMsg(msg *ethwire.Msg, commonBlockHash []byte) bool {
	var blocks []*Block
	for i := 0; i < (msg.Data.Len() - 1); i++ {
		block := NewBlockFromRlpValue(msg.Data.Get(i))
		blocks = append(blocks, block)
	}
	return bc.FindCanonicalChain(blocks, commonBlockHash)
}

// Is tasked by finding the CanonicalChain and resetting the chain if we are not the Conical one
// Return true if we are the using the canonical chain false if not
func (bc *BlockChain) FindCanonicalChain(blocks []*Block, commonBlockHash []byte) bool {
	// 1. Calculate TD of the current chain
	// 2. Calculate TD of the new chain
	// Reset state to the correct one

	chainDifficulty := new(big.Int)

	// Calculate the entire chain until the block we both have
	// Start with the newest block we got, all the way back to the common block we both know
	for _, block := range blocks {
		if bytes.Compare(block.Hash(), commonBlockHash) == 0 {
			chainlogger.Infoln("[CHAIN] We have found the common parent block, breaking")
			break
		}
		chainDifficulty.Add(chainDifficulty, bc.CalculateBlockTD(block))
	}

	chainlogger.Infoln("Incoming chain difficulty:", chainDifficulty)

	curChainDifficulty := new(big.Int)
	block := bc.CurrentBlock
	for i := 0; block != nil; block = bc.GetBlock(block.PrevHash) {
		i++
		if bytes.Compare(block.Hash(), commonBlockHash) == 0 {
			chainlogger.Infoln("We have found the common parent block, breaking")
			break
		}
		anOtherBlock := bc.GetBlock(block.PrevHash)
		if anOtherBlock == nil {
			// We do not want to count the genesis block for difficulty since that's not being sent
			chainlogger.Infoln("At genesis block, breaking")
			break
		}
		curChainDifficulty.Add(curChainDifficulty, bc.CalculateBlockTD(block))
	}

	chainlogger.Infoln("Current chain difficulty:", curChainDifficulty)
	if chainDifficulty.Cmp(curChainDifficulty) == 1 {
		chainlogger.Infof("The incoming Chain beat our asses, resetting to block: %x", commonBlockHash)
		bc.ResetTillBlockHash(commonBlockHash)
		return false
	} else {
		chainlogger.Infoln("Our chain showed the incoming chain who is boss. Ignoring.")
		return true
	}
}
func (bc *BlockChain) ResetTillBlockHash(hash []byte) error {
	lastBlock := bc.CurrentBlock
	var returnTo *Block
	// Reset to Genesis if that's all the origin there is.
	if bytes.Compare(hash, bc.genesisBlock.Hash()) == 0 {
		returnTo = bc.genesisBlock
		bc.CurrentBlock = bc.genesisBlock
		bc.LastBlockHash = bc.genesisBlock.Hash()
		bc.LastBlockNumber = 1
	} else {
		returnTo = bc.GetBlock(hash)
		bc.CurrentBlock = returnTo
		bc.LastBlockHash = returnTo.Hash()
		bc.LastBlockNumber = returnTo.Number.Uint64()
	}

	// Manually reset the last sync block
	err := ethutil.Config.Db.Delete(lastBlock.Hash())
	if err != nil {
		return err
	}

	var block *Block
	for ; block != nil; block = bc.GetBlock(block.PrevHash) {
		if bytes.Compare(block.Hash(), hash) == 0 {
			chainlogger.Infoln("We have arrived at the the common parent block, breaking")
			break
		}
		err = ethutil.Config.Db.Delete(block.Hash())
		if err != nil {
			return err
		}
	}
	chainlogger.Infoln("Split chain deleted and reverted to common parent block.")
	return nil
}

func (bc *BlockChain) GenesisBlock() *Block {
	return bc.genesisBlock
}

// Get chain return blocks from hash up to max in RLP format
func (bc *BlockChain) GetChainFromHash(hash []byte, max uint64) []interface{} {
	var chain []interface{}
	// Get the current hash to start with
	currentHash := bc.CurrentBlock.Hash()
	// Get the last number on the block chain
	lastNumber := bc.CurrentBlock.Number.Uint64()
	// Get the parents number
	parentNumber := bc.GetBlock(hash).Number.Uint64()
	// Get the min amount. We might not have max amount of blocks
	count := uint64(math.Min(float64(lastNumber-parentNumber), float64(max)))
	startNumber := parentNumber + count

	num := lastNumber
	for ; num > startNumber; currentHash = bc.GetBlock(currentHash).PrevHash {
		num--
	}
	for i := uint64(0); bytes.Compare(currentHash, hash) != 0 && num >= parentNumber && i < count; i++ {
		// Get the block of the chain
		block := bc.GetBlock(currentHash)
		if block == nil {
			chainlogger.Debugf("Unexpected error during GetChainFromHash: Unable to find %x\n", currentHash)
			break
		}

		currentHash = block.PrevHash

		chain = append(chain, block.Value().Val)

		num--
	}

	return chain
}

func (bc *BlockChain) GetChain(hash []byte, amount int) []*Block {
	genHash := bc.genesisBlock.Hash()

	block := bc.GetBlock(hash)
	var blocks []*Block

	for i := 0; i < amount && block != nil; block = bc.GetBlock(block.PrevHash) {
		blocks = append([]*Block{block}, blocks...)

		if bytes.Compare(genHash, block.Hash()) == 0 {
			break
		}
		i++
	}

	return blocks
}

func AddTestNetFunds(block *Block) {
	for _, addr := range []string{
		"51ba59315b3a95761d0863b05ccc7a7f54703d99",
		"e4157b34ea9615cfbde6b4fda419828124b70c78",
		"1e12515ce3e0f817a4ddef9ca55788a1d66bd2df",
		"6c386a4b26f73c802f34673f7248bb118f97424a",
		"cd2a3d9f938e13cd947ec05abc7fe734df8dd826",
		"2ef47100e0787b915105fd5e3f4ff6752079d5cb",
		"e6716f9544a56c530d868e4bfbacb172315bdead",
		"1a26338f0d905e295fccb71fa9ea849ffa12aaf4",
	} {
		codedAddr := ethutil.FromHex(addr)
		account := block.state.GetAccount(codedAddr)
		account.Amount = ethutil.Big("1606938044258990275541962092341162602522202993782792835301376") //ethutil.BigPow(2, 200)
		block.state.UpdateStateObject(account)
	}
}

func (bc *BlockChain) setLastBlock() {
	data, _ := ethutil.Config.Db.Get([]byte("LastBlock"))
	if len(data) != 0 {
		block := NewBlockFromBytes(data)
		//info := bc.BlockInfo(block)
		bc.CurrentBlock = block
		bc.LastBlockHash = block.Hash()
		bc.LastBlockNumber = block.Number.Uint64()

		chainlogger.Infof("Last known block height #%d\n", bc.LastBlockNumber)
	} else {
		AddTestNetFunds(bc.genesisBlock)

		bc.genesisBlock.state.trie.Sync()
		// Prepare the genesis block
		bc.Add(bc.genesisBlock)

		//chainlogger.Infof("root %x\n", bm.bc.genesisBlock.State().Root)
		//bm.bc.genesisBlock.PrintHash()
	}

	// Set the last know difficulty (might be 0x0 as initial value, Genesis)
	bc.TD = ethutil.BigD(ethutil.Config.Db.LastKnownTD())

	chainlogger.Infof("Last block: %x\n", bc.CurrentBlock.Hash())
}

func (bc *BlockChain) SetTotalDifficulty(td *big.Int) {
	ethutil.Config.Db.Put([]byte("LastKnownTotalDifficulty"), td.Bytes())
	bc.TD = td
}

// Add a block to the chain and record addition information
func (bc *BlockChain) Add(block *Block) {
	bc.writeBlockInfo(block)
	// Prepare the genesis block

	bc.CurrentBlock = block
	bc.LastBlockHash = block.Hash()

	encodedBlock := block.RlpEncode()
	ethutil.Config.Db.Put(block.Hash(), encodedBlock)
	ethutil.Config.Db.Put([]byte("LastBlock"), encodedBlock)
}

func (bc *BlockChain) GetBlock(hash []byte) *Block {
	data, _ := ethutil.Config.Db.Get(hash)
	if len(data) == 0 {
		return nil
	}

	return NewBlockFromBytes(data)
}

func (bc *BlockChain) BlockInfoByHash(hash []byte) BlockInfo {
	bi := BlockInfo{}
	data, _ := ethutil.Config.Db.Get(append(hash, []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

func (bc *BlockChain) BlockInfo(block *Block) BlockInfo {
	bi := BlockInfo{}
	data, _ := ethutil.Config.Db.Get(append(block.Hash(), []byte("Info")...))
	bi.RlpDecode(data)

	return bi
}

// Unexported method for writing extra non-essential block info to the db
func (bc *BlockChain) writeBlockInfo(block *Block) {
	bc.LastBlockNumber++
	bi := BlockInfo{Number: bc.LastBlockNumber, Hash: block.Hash(), Parent: block.PrevHash}

	// For now we use the block hash with the words "info" appended as key
	ethutil.Config.Db.Put(append(block.Hash(), []byte("Info")...), bi.RlpEncode())
}

func (bc *BlockChain) Stop() {
	if bc.CurrentBlock != nil {
		chainlogger.Infoln("Stopped")
	}
}
