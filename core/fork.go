// Copyright 2016 The go-ethereum Authors
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
	"errors"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
)

// ChainResolver should implement chain resolving and should be capable
// handling reorganisations.
type ChainResolver interface {
	Resolve(ethdb.ReadWriter, []Changes) error
}

// Changes are changes that have previously been applied to the forked blockchain.
type Changes struct {
	td       *big.Int       // total difficulty of the block
	block    *types.Block   // the block
	receipts types.Receipts // block's receipts after processing transactions
}

// BlockReader is the basic chain reader interface for reading blocks of the blockchain
type BlockReader interface {
	GetBlock(common.Hash) *types.Block // GetBlock returns the block that corresponds to the given hash
	Db() ethdb.Database                // XXX this doesn't really belong here.
}

// receipt keeps a log of changes for a specific block number which can
// later be used to write out the result.
type receipt struct {
	blockNumber uint64
	blockHash   common.Hash
	receipts    types.Receipts
}

// ChainFork is a temporary fork of the blockchain to which all block changes
// must be applied before being written out to the database.
//
// Example chain processing
//
// 	fork := Fork(chain, blocks[0].ParentHash)
//	for _, block := range blocks {
//		// State is an in-memory StateDB used throughout the fork
//		err := ValidateBlock(block, fork.State())
//		if err != nil {
//			return err
//		}
//		fork.CommitBlock(block)
//	}
//	fork.ApplyTo(blockchain)
//	fork.CommitToDb()
//
// Example block generation
//
//	fork := Fork(chain, hash)
//
//	// Create a new unsealed block
//	block := fork.NewUnsealedBlock()
//	// Apply transactions and uncles
//	appliedTxs := block.ApplyTransactions(txpool.Transactions())
// 	appliedUncles := block.ApplyUncles(someUncles)
//
//  	// Seal the block, making it a valid sealed and signed block.
//	block := Seal(powSealer, block)
//
//	// Commit the block back to the fork.
//	fork.CommitBlock(block)
//
type ChainFork struct {
	db     ethdb.Database // backing database
	reader BlockReader    // block reader utility interface
	state  *state.StateDB // the current state database

	originN      uint64       // the origin block number
	origin       *types.Block // origin notes the start of the fork
	currentBlock *types.Block // the current block within this transaction

	changes []Changes // changes
}

// Fork returns a new blockchain with the given database as backing layer
// for the localised blockchain transaction.
func Fork(blockReader BlockReader, origin common.Hash) (*ChainFork, error) {
	fork := &ChainFork{
		db:     blockReader.Db(),
		reader: blockReader,
	}

	// get the origin block from which this fork originates
	if fork.origin = blockReader.GetBlock(origin); fork.origin == nil {
		return nil, fmt.Errorf("core/fork: no block found with hash: %x", origin)
	}
	fork.originN = fork.origin.NumberU64()

	statedb, err := state.New(fork.origin.Root(), fork.db)
	if err != nil {
		return nil, fmt.Errorf("core/fork: enable to create state: %v", err)
	}
	fork.state = statedb

	return fork, nil
}

// GetNumHash returns the hash of the block that corresponds to the block number
// in our current fork
func (fork *ChainFork) GetNumHash(n uint64) common.Hash {
	// Short circuit if the number is larger than our chain.
	if n > uint64(len(fork.changes))+fork.originN {
		return common.Hash{}
	}

	// check if we should have it cached
	if n > fork.originN {
		return fork.changes[n-fork.originN].block.Hash()
	}

	// otherwise search in the database
	for block := fork.reader.GetBlock(fork.origin.Hash()); block != nil; block = fork.reader.GetBlock(block.ParentHash()) {
		if block.NumberU64() == n {
			return block.Hash()
		}
	}
	return common.Hash{}
}

func (fork *ChainFork) State() *state.StateDB {
	return fork.state
}

var errUnboundedParent = errors.New("core/fork: parent hash does not match last block")

// CommitBlock commits a new block to the fork. The block that's being commited their parent hash must
// match the previously committed block or the origin if the fork is empty.
func (fork *ChainFork) CommitBlock(td *big.Int, block *types.Block, receipts types.Receipts) error {
	// Check and make sure that the block being applied is valid and can be applied
	// on the last block.
	if len(fork.changes) == 0 && block.ParentHash() != fork.origin.Hash() {
		return errUnboundedParent
	} else if len(fork.changes) > 0 && block.ParentHash() != fork.changes[len(fork.changes)-1].block.Hash() {
		return errUnboundedParent
	}

	fork.changes = append(fork.changes, Changes{
		td:       td,
		block:    block,
		receipts: receipts,
	})
	return nil
}

// CommitTo commits the current changes to the database
func (fork *ChainFork) CommitToDb() error {
	tx, err := fork.db.OpenTransaction()
	if err != nil {
		return fmt.Errorf("core/fork: unable to create db transaction: %v", err)
	}

	// write mips to the datatabase transaction
	cachedMips := make(map[string]types.Bloom)
	for _, change := range fork.changes {
		for _, level := range MIPMapLevels {
			var (
				key    = mipmapKey(change.block.NumberU64(), level)
				mipmap types.Bloom
				ok     bool
			)
			if mipmap, ok = cachedMips[string(key)]; !ok {
				bloomDat, _ := fork.db.Get(key)
				mipmap = types.BytesToBloom(bloomDat)
			}

			for _, receipt := range change.receipts {
				for _, log := range receipt.Logs {
					mipmap.Add(log.Address.Big())
				}
			}
			cachedMips[string(key)] = mipmap
		}

		hash := change.block.Hash()
		if err := WriteHeader(tx, change.block.Header()); err != nil {
			return err
		}
		if err := WriteTd(tx, hash, change.td); err != nil {
			return err
		}

		if len(change.receipts) > 0 {
			if err := WriteBlockReceipts(tx, hash, change.receipts); err != nil {
				return err
			}
		}

		txs := change.block.Transactions()
		if len(txs) > 0 {
			if err := WriteBlockTransactions(tx, change.block.Header(), txs); err != nil {
				return err
			}
		}
	}

	// write out mip maps
	for key, mipmap := range cachedMips {
		tx.Put([]byte(key), mipmap.Bytes())
	}
	return tx.Commit()
}

// ApplyTo applies the fork to chain and uses the resolver to write and resolve
// any chain reorganisations.
func (fork *ChainFork) ApplyTo(resolver ChainResolver) error {
	tx, err := fork.db.OpenTransaction()
	if err != nil {
		return err
	}
	return resolver.Resolve(tx, fork.changes)
}

// NewUnsealedBlock creates a new unsealed block using the last block in the fork
// as its parent.
func (fork *ChainFork) NewUnsealedBlock(coinbase common.Address, extra []byte) *UnsealedBlock {
	var (
		parent        *types.Header
		unsealedBlock = new(UnsealedBlock)
	)

	if len(fork.changes) == 0 {
		parent = fork.origin.Header()
	} else {
		parent = fork.changes[len(fork.changes)-1].block.Header()
	}

	tstamp := time.Now().Unix()
	if parent.Time.Cmp(new(big.Int).SetInt64(tstamp)) >= 0 {
		tstamp = parent.Time.Int64() + 1
	}

	unsealedBlock.header = types.Header{
		Root:       fork.state.IntermediateRoot(),
		ParentHash: parent.Hash(),
		Number:     new(big.Int).Add(parent.Number, common.Big1),
		Difficulty: CalcDifficulty(nil, uint64(tstamp), parent.Time.Uint64(), parent.Number, parent.Difficulty),
		GasLimit:   CalcGasLimit(types.NewBlockWithHeader(parent)),
		GasUsed:    new(big.Int),
		Coinbase:   coinbase,
		Extra:      extra,
		Time:       big.NewInt(tstamp),
	}

	return unsealedBlock
}
