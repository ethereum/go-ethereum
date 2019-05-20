// Copyright 2015 The go-ethereum Authors
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

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/pow"
)

// FakePow is a non-validating proof of work implementation.
// It returns true from Verify for any block.
type FakePow struct{}

func (f FakePow) Search(block pow.Block, stop <-chan struct{}) (uint64, []byte) {
	return 0, nil
}
func (f FakePow) Verify(block pow.Block) bool { return true }
func (f FakePow) GetHashrate() int64          { return 0 }
func (f FakePow) Turbo(bool)                  {}

// So we can deterministically seed different blockchains
var (
	canonicalSeed = 1
	forkSeed      = 2
)

// BlockGen creates blocks for testing.
// See GenerateChain for a detailed explanation.
type BlockGen struct {
	i       int
	parent  *types.Block
	chain   []*types.Block
	header  *types.Header
	statedb *state.StateDB

	coinbase *state.StateObject
	txs      []*types.Transaction
	receipts []*types.Receipt
	uncles   []*types.Header
}

// SetCoinbase sets the coinbase of the generated block.
// It can be called at most once.
func (b *BlockGen) SetCoinbase(addr common.Address) {
	if b.coinbase != nil {
		if len(b.txs) > 0 {
			panic("coinbase must be set before adding transactions")
		}
		panic("coinbase can only be set once")
	}
	b.header.Coinbase = addr
	b.coinbase = b.statedb.GetOrNewStateObject(addr)
	b.coinbase.SetGasLimit(b.header.GasLimit)
}

// SetExtra sets the extra data field of the generated block.
func (b *BlockGen) SetExtra(data []byte) {
	b.header.Extra = data
}

// AddTx adds a transaction to the generated block. If no coinbase has
// been set, the block's coinbase is set to the zero address.
//
// AddTx panics if the transaction cannot be executed. In addition to
// the protocol-imposed limitations (gas limit, etc.), there are some
// further limitations on the content of transactions that can be
// added. Notably, contract code relying on the BLOCKHASH instruction
// will panic during execution.
func (b *BlockGen) AddTx(tx *types.Transaction) {
	if b.coinbase == nil {
		b.SetCoinbase(common.Address{})
	}
	_, gas, err := ApplyMessage(NewEnv(b.statedb, nil, tx, b.header), tx, b.coinbase)
	if err != nil {
		panic(err)
	}
	b.statedb.SyncIntermediate()
	b.header.GasUsed.Add(b.header.GasUsed, gas)
	receipt := types.NewReceipt(b.statedb.Root().Bytes(), b.header.GasUsed)
	logs := b.statedb.GetLogs(tx.Hash())
	receipt.SetLogs(logs)
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	b.txs = append(b.txs, tx)
	b.receipts = append(b.receipts, receipt)
}

// TxNonce returns the next valid transaction nonce for the
// account at addr. It panics if the account does not exist.
func (b *BlockGen) TxNonce(addr common.Address) uint64 {
	if !b.statedb.HasAccount(addr) {
		panic("account does not exist")
	}
	return b.statedb.GetNonce(addr)
}

// AddUncle adds an uncle header to the generated block.
func (b *BlockGen) AddUncle(h *types.Header) {
	b.uncles = append(b.uncles, h)
}

// PrevBlock returns a previously generated block by number. It panics if
// num is greater or equal to the number of the block being generated.
// For index -1, PrevBlock returns the parent block given to GenerateChain.
func (b *BlockGen) PrevBlock(index int) *types.Block {
	if index >= b.i {
		panic("block index out of range")
	}
	if index == -1 {
		return b.parent
	}
	return b.chain[index]
}

// GenerateChain creates a chain of n blocks. The first block's
// parent will be the provided parent. db is used to store
// intermediate states and should contain the parent's state trie.
//
// The generator function is called with a new block generator for
// every block. Any transactions and uncles added to the generator
// become part of the block. If gen is nil, the blocks will be empty
// and their coinbase will be the zero address.
//
// Blocks created by GenerateChain do not contain valid proof of work
// values. Inserting them into ChainManager requires use of FakePow or
// a similar non-validating proof of work implementation.
func GenerateChain(parent *types.Block, db common.Database, n int, gen func(int, *BlockGen)) []*types.Block {
	statedb := state.New(parent.Root(), db)
	blocks := make(types.Blocks, n)
	genblock := func(i int, h *types.Header) *types.Block {
		b := &BlockGen{parent: parent, i: i, chain: blocks, header: h, statedb: statedb}
		if gen != nil {
			gen(i, b)
		}
		AccumulateRewards(statedb, h, b.uncles)
		statedb.SyncIntermediate()
		h.Root = statedb.Root()
		return types.NewBlock(h, b.txs, b.uncles, b.receipts)
	}
	for i := 0; i < n; i++ {
		header := makeHeader(parent, statedb)
		block := genblock(i, header)
		block.Td = CalcTD(block, parent)
		blocks[i] = block
		parent = block
	}
	return blocks
}

func makeHeader(parent *types.Block, state *state.StateDB) *types.Header {
	time := parent.Time() + 10 // block time is fixed at 10 seconds
	return &types.Header{
		Root:       state.Root(),
		ParentHash: parent.Hash(),
		Coinbase:   parent.Coinbase(),
		Difficulty: CalcDifficulty(time, parent.Time(), parent.Difficulty()),
		GasLimit:   CalcGasLimit(parent),
		GasUsed:    new(big.Int),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Time:       uint64(time),
	}
}

// newCanonical creates a new deterministic canonical chain by running
// InsertChain on the result of makeChain.
func newCanonical(n int, db common.Database) (*BlockProcessor, error) {
	evmux := &event.TypeMux{}

	WriteTestNetGenesisBlock(db, db, 0)
	chainman, _ := NewChainManager(db, db, db, FakePow{}, evmux)
	bman := NewBlockProcessor(db, db, FakePow{}, chainman, evmux)
	bman.bc.SetProcessor(bman)
	parent := bman.bc.CurrentBlock()
	if n == 0 {
		return bman, nil
	}
	lchain := makeChain(parent, n, db, canonicalSeed)
	_, err := bman.bc.InsertChain(lchain)
	return bman, err
}

func makeChain(parent *types.Block, n int, db common.Database, seed int) []*types.Block {
	return GenerateChain(parent, db, n, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{0: byte(seed), 19: byte(i)})
	})
}
