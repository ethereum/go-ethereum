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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/pow"
)

// FakePow is a non-validating proof of work implementation.
// It returns true from Verify for any block.
type FakePow struct{}

func (f FakePow) Search(block pow.Block, stop <-chan struct{}, index int) (uint64, []byte) {
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
	root := b.statedb.IntermediateRoot()
	b.header.GasUsed.Add(b.header.GasUsed, gas)
	receipt := types.NewReceipt(root.Bytes(), b.header.GasUsed)
	logs := b.statedb.GetLogs(tx.Hash())
	receipt.Logs = logs
	receipt.Bloom = types.CreateBloom(types.Receipts{receipt})
	b.txs = append(b.txs, tx)
	b.receipts = append(b.receipts, receipt)
}

// AddUncheckedReceipts forcefully adds a receipts to the block without a
// backing transaction.
//
// AddUncheckedReceipts will cause consensus failures when used during real
// chain processing. This is best used in conjuction with raw block insertion.
func (b *BlockGen) AddUncheckedReceipt(receipt *types.Receipt) {
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

// OffsetTime modifies the time instance of a block, implicitly changing its
// associated difficulty. It's useful to test scenarios where forking is not
// tied to chain length directly.
func (b *BlockGen) OffsetTime(seconds int64) {
	b.header.Time.Add(b.header.Time, new(big.Int).SetInt64(seconds))
	if b.header.Time.Cmp(b.parent.Header().Time) <= 0 {
		panic("block time out of range")
	}
	b.header.Difficulty = CalcDifficulty(b.header.Time.Uint64(), b.parent.Time().Uint64(), b.parent.Number(), b.parent.Difficulty())
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
// values. Inserting them into BlockChain requires use of FakePow or
// a similar non-validating proof of work implementation.
func GenerateChain(parent *types.Block, db ethdb.Database, n int, gen func(int, *BlockGen)) ([]*types.Block, []types.Receipts) {
	statedb, err := state.New(parent.Root(), db)
	if err != nil {
		panic(err)
	}
	blocks, receipts := make(types.Blocks, n), make([]types.Receipts, n)
	genblock := func(i int, h *types.Header) (*types.Block, types.Receipts) {
		b := &BlockGen{parent: parent, i: i, chain: blocks, header: h, statedb: statedb}
		if gen != nil {
			gen(i, b)
		}
		AccumulateRewards(statedb, h, b.uncles)
		root, err := statedb.Commit()
		if err != nil {
			panic(fmt.Sprintf("state write error: %v", err))
		}
		h.Root = root
		return types.NewBlock(h, b.txs, b.uncles, b.receipts), b.receipts
	}
	for i := 0; i < n; i++ {
		header := makeHeader(parent, statedb)
		block, receipt := genblock(i, header)
		blocks[i] = block
		receipts[i] = receipt
		parent = block
	}
	return blocks, receipts
}

func makeHeader(parent *types.Block, state *state.StateDB) *types.Header {
	var time *big.Int
	if parent.Time() == nil {
		time = big.NewInt(10)
	} else {
		time = new(big.Int).Add(parent.Time(), big.NewInt(10)) // block time is fixed at 10 seconds
	}
	return &types.Header{
		Root:       state.IntermediateRoot(),
		ParentHash: parent.Hash(),
		Coinbase:   parent.Coinbase(),
		Difficulty: CalcDifficulty(time.Uint64(), new(big.Int).Sub(time, big.NewInt(10)).Uint64(), parent.Number(), parent.Difficulty()),
		GasLimit:   CalcGasLimit(parent),
		GasUsed:    new(big.Int),
		Number:     new(big.Int).Add(parent.Number(), common.Big1),
		Time:       time,
	}
}

// newCanonical creates a chain database, and injects a deterministic canonical
// chain. Depending on the full flag, if creates either a full block chain or a
// header only chain.
func newCanonical(n int, full bool) (ethdb.Database, *BlockProcessor, error) {
	// Create te new chain database
	db, _ := ethdb.NewMemDatabase()
	evmux := &event.TypeMux{}

	// Initialize a fresh chain with only a genesis block
	genesis, _ := WriteTestNetGenesisBlock(db, 0)

	blockchain, _ := NewBlockChain(db, FakePow{}, evmux)
	processor := NewBlockProcessor(db, FakePow{}, blockchain, evmux)
	processor.bc.SetProcessor(processor)

	// Create and inject the requested chain
	if n == 0 {
		return db, processor, nil
	}
	if full {
		// Full block-chain requested
		blocks := makeBlockChain(genesis, n, db, canonicalSeed)
		_, err := blockchain.InsertChain(blocks)
		return db, processor, err
	}
	// Header-only chain requested
	headers := makeHeaderChain(genesis.Header(), n, db, canonicalSeed)
	_, err := blockchain.InsertHeaderChain(headers, 1)
	return db, processor, err
}

// makeHeaderChain creates a deterministic chain of headers rooted at parent.
func makeHeaderChain(parent *types.Header, n int, db ethdb.Database, seed int) []*types.Header {
	blocks := makeBlockChain(types.NewBlockWithHeader(parent), n, db, seed)
	headers := make([]*types.Header, len(blocks))
	for i, block := range blocks {
		headers[i] = block.Header()
	}
	return headers
}

// makeBlockChain creates a deterministic chain of blocks rooted at parent.
func makeBlockChain(parent *types.Block, n int, db ethdb.Database, seed int) []*types.Block {
	blocks, _ := GenerateChain(parent, db, n, func(i int, b *BlockGen) {
		b.SetCoinbase(common.Address{0: byte(seed), 19: byte(i)})
	})
	return blocks
}
