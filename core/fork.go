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
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
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
	td       *big.Int           // total difficulty of the block
	header   types.Header       // block's header
	txs      types.Transactions // block's transactions
	receipts types.Receipts     // block's receipts after processing transactions
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
	db     ethdb.Database    // backing database
	tx     ethdb.Transaction // current operating transaction
	reader BlockReader       // block reader utility interface

	origin       common.Hash  // origin notes the start of the fork
	currentBlock *types.Block // the current block within this transaction

	changes []Changes // changes
}

// Fork returns a new blockchain with the given database as backing layer
// for the localised blockchain transaction.
func Fork(blockReader BlockReader, origin common.Hash) (*ChainFork, error) {
	fork := &ChainFork{
		db:     blockReader.Db(),
		reader: blockReader,
		origin: origin,
	}
	// open a new leveldb transaction
	tx, err := fork.db.OpenTransaction()
	if err != nil {
		return nil, err
	}
	fork.tx = tx

	// get the origin block from which this fork originates
	if block := blockReader.GetBlock(origin); block != nil {
		// Block found, set as the current head
		fork.currentBlock = block
	} else {
		return nil, fmt.Errorf("core/fork: no block found with hash: %x", origin)
	}

	return fork, nil
}

// CommitChanges commits the changes to the fork and takes care of the writing
// to the tx (e.g. blocks, block receipts, transactions, etc.).
func (fork *ChainFork) CommitChanges(td *big.Int, header types.Header, transactions types.Transactions, receipts types.Receipts) error {
	hash := header.Hash()

	fork.changes = append(fork.changes, Changes{
		td:       td,
		header:   header,
		txs:      transactions,
		receipts: receipts,
	})

	if err := WriteHeader(fork.tx, &header); err != nil {
		return err
	}
	if err := WriteTd(fork.tx, hash, td); err != nil {
		return err
	}
	if len(receipts) > 0 {
		if err := WriteBlockReceipts(fork.tx, hash, receipts); err != nil {
			return err
		}
	}

	if len(transactions) > 0 {
		if err := WriteBlockTransactions(fork.tx, header, transactions); err != nil {
			return err
		}
	}

	return nil
}

// WriteBlockBody writes the gives block to the database transaction.
func (fork *ChainFork) WriteBlockBody(hash common.Hash, body types.Body) error {
	// Store the body first to retain database consistency
	if err := WriteBody(fork.tx, hash, &body); err != nil {
		return err
	}

	return nil
}

// CommitTo commits the current changes to the database
func (fork *ChainFork) CommitToDb() error {
	// write mips to the datatabase transaction
	cachedMips := make(map[string]types.Bloom)
	for _, level := range MIPMapLevels {
		for _, change := range fork.changes {
			var (
				key    = mipmapKey(change.header.Number.Uint64(), level)
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
	}
	// write out mip maps
	for key, mipmap := range cachedMips {
		fork.tx.Put([]byte(key), mipmap.Bytes())
	}
	return fork.tx.Commit()
}

// ApplyTo applies the fork to chain and uses the resolver to write and resolve
// any chain reorganisations.
func (fork *ChainFork) ApplyTo(resolver ChainResolver) error {
	return resolver.Resolve(fork.tx, fork.changes)
}
