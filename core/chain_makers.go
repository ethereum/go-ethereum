package core

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/pow"
)

// So we can generate blocks easily
type FakePow struct{}

func (f FakePow) Search(block pow.Block, stop <-chan struct{}) (uint64, []byte) {
	return 0, nil
}
func (f FakePow) Verify(block pow.Block) bool { return true }
func (f FakePow) GetHashrate() int64          { return 0 }
func (f FakePow) Turbo(bool)                  {}

// So we can deterministically seed different blockchains
var (
	CanonicalSeed = 1
	ForkSeed      = 2
)

// Utility functions for making chains on the fly
// Exposed for sake of testing from other packages (eg. go-ethash)
func NewBlockFromParent(addr common.Address, parent *types.Block) *types.Block {
	return newBlockFromParent(addr, parent)
}

func MakeBlock(bman *BlockProcessor, parent *types.Block, i int, db common.Database, seed int) *types.Block {
	return makeBlock(bman, parent, i, db, seed)
}

func MakeChain(bman *BlockProcessor, parent *types.Block, max int, db common.Database, seed int) types.Blocks {
	return makeChain(bman, parent, max, db, seed)
}

func NewChainMan(block *types.Block, eventMux *event.TypeMux, db common.Database) *ChainManager {
	return newChainManager(block, eventMux, db)
}

func NewBlockProc(db common.Database, cman *ChainManager, eventMux *event.TypeMux) *BlockProcessor {
	return newBlockProcessor(db, cman, eventMux)
}

func NewCanonical(n int, db common.Database) (*BlockProcessor, error) {
	return newCanonical(n, db)
}

// block time is fixed at 10 seconds
func newBlockFromParent(addr common.Address, parent *types.Block) *types.Block {
	block := types.NewBlock(parent.Hash(), addr, parent.Root(), common.BigPow(2, 32), 0, nil)
	block.SetUncles(nil)
	block.SetTransactions(nil)
	block.SetReceipts(nil)

	header := block.Header()
	header.Difficulty = CalcDifficulty(block.Header(), parent.Header())
	header.Number = new(big.Int).Add(parent.Header().Number, common.Big1)
	header.Time = parent.Header().Time + 10
	header.GasLimit = CalcGasLimit(parent)

	block.Td = parent.Td

	return block
}

// Actually make a block by simulating what miner would do
// we seed chains by the first byte of the coinbase
func makeBlock(bman *BlockProcessor, parent *types.Block, i int, db common.Database, seed int) *types.Block {
	var addr common.Address
	addr[0], addr[19] = byte(seed), byte(i)
	block := newBlockFromParent(addr, parent)
	state := state.New(block.Root(), db)
	cbase := state.GetOrNewStateObject(addr)
	cbase.SetGasPool(CalcGasLimit(parent))
	cbase.AddBalance(BlockReward)
	state.Update()
	block.SetRoot(state.Root())
	return block
}

// Make a chain with real blocks
// Runs ProcessWithParent to get proper state roots
func makeChain(bman *BlockProcessor, parent *types.Block, max int, db common.Database, seed int) types.Blocks {
	bman.bc.currentBlock = parent
	blocks := make(types.Blocks, max)
	for i := 0; i < max; i++ {
		block := makeBlock(bman, parent, i, db, seed)
		_, err := bman.processWithParent(block, parent)
		if err != nil {
			fmt.Println("process with parent failed", err)
			panic(err)
		}
		block.Td = CalcTD(block, parent)
		blocks[i] = block
		parent = block
	}
	return blocks
}

// Create a new chain manager starting from given block
// Effectively a fork factory
func newChainManager(block *types.Block, eventMux *event.TypeMux, db common.Database) *ChainManager {
	genesis := GenesisBlock(db)
	bc := &ChainManager{blockDb: db, stateDb: db, genesisBlock: genesis, eventMux: eventMux}
	bc.txState = state.ManageState(state.New(genesis.Root(), db))
	bc.futureBlocks = NewBlockCache(1000)
	if block == nil {
		bc.Reset()
	} else {
		bc.currentBlock = block
		bc.td = block.Td
	}
	return bc
}

// block processor with fake pow
func newBlockProcessor(db common.Database, cman *ChainManager, eventMux *event.TypeMux) *BlockProcessor {
	chainMan := newChainManager(nil, eventMux, db)
	txpool := NewTxPool(eventMux, chainMan.State, chainMan.GasLimit)
	bman := NewBlockProcessor(db, db, FakePow{}, txpool, chainMan, eventMux)
	return bman
}

// Make a new, deterministic canonical chain by running InsertChain
// on result of makeChain
func newCanonical(n int, db common.Database) (*BlockProcessor, error) {
	eventMux := &event.TypeMux{}

	bman := newBlockProcessor(db, newChainManager(nil, eventMux, db), eventMux)
	bman.bc.SetProcessor(bman)
	parent := bman.bc.CurrentBlock()
	if n == 0 {
		return bman, nil
	}
	lchain := makeChain(bman, parent, n, db, CanonicalSeed)
	_, err := bman.bc.InsertChain(lchain)
	return bman, err
}
