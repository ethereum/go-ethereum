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
func MakeBlock(bman *BlockProcessor, parent *types.Block, i int, db common.Database, seed int) *types.Block {
	return types.NewBlock(makeHeader(parent, i, db, seed), nil, nil, nil)
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

// makeHeader creates the header for a new empty block, simulating
// what miner would do. We seed chains by the first byte of the coinbase.
func makeHeader(parent *types.Block, i int, db common.Database, seed int) *types.Header {
	var addr common.Address
	addr[0], addr[19] = byte(seed), byte(i) // 'random' coinbase
	time := parent.Time() + 10              // block time is fixed at 10 seconds

	// ensure that the block's coinbase has the block reward in the state.
	state := state.New(parent.Root(), db)
	cbase := state.GetOrNewStateObject(addr)
	cbase.SetGasLimit(CalcGasLimit(parent))
	cbase.AddBalance(BlockReward)
	state.Update()

	return &types.Header{
		Root:       state.Root(),
		ParentHash: parent.Hash(),
		Coinbase:   addr,
		Difficulty: CalcDifficulty(time, parent.Time(), parent.Difficulty()),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Time:       uint64(time),
		GasLimit:   CalcGasLimit(parent),
	}
}

// makeChain creates a valid chain of empty blocks.
func makeChain(bman *BlockProcessor, parent *types.Block, max int, db common.Database, seed int) types.Blocks {
	bman.bc.currentBlock = parent
	blocks := make(types.Blocks, max)
	for i := 0; i < max; i++ {
		block := types.NewBlock(makeHeader(parent, i, db, seed), nil, nil, nil)
		// Use ProcessWithParent to verify that we have produced a valid block.
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
	genesis := GenesisBlock(0, db)
	bc := &ChainManager{blockDb: db, stateDb: db, genesisBlock: genesis, eventMux: eventMux, pow: FakePow{}}
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
	bman := NewBlockProcessor(db, db, FakePow{}, chainMan, eventMux)
	return bman
}

// Make a new, deterministic canonical chain by running InsertChain
// on result of makeChain.
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
