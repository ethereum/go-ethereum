// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package core

import (
	"errors"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// ErrInvalidEpoch is returned when epoch validation fails
	ErrInvalidEpoch = errors.New("invalid epoch")
	
	// ErrNotMasternode is returned when signer is not a masternode
	ErrNotMasternode = errors.New("signer is not a masternode")
	
	// ErrMasternodePenalized is returned when masternode is penalized
	ErrMasternodePenalized = errors.New("masternode is penalized")
)

// CheckpointCh is a channel for checkpoint notifications
var CheckpointCh = make(chan int)

// BlockChainHooks defines hooks for XDPoS consensus integration
type BlockChainHooks struct {
	// HookReward is called to distribute block rewards
	HookReward func(chain *BlockChain, statedb *state.StateDB, parentState *state.StateDB, header *types.Header, txs []*types.Transaction, receipts []*types.Receipt) ([]*types.Receipt, error)
	
	// HookPenalty is called to handle validator penalties
	HookPenalty func(chain *BlockChain, number *big.Int, parentHash common.Hash, coinbase common.Address) ([]common.Address, error)
	
	// HookValidator is called to validate block signer
	HookValidator func(header *types.Header, signers []common.Address) (common.Address, error)
	
	// HookVerifyMasterNodes is called to verify masternode list
	HookVerifyMasterNodes func(header *types.Header, signers []common.Address) error
	
	// HookGetSignersFromContract gets signers from smart contract
	HookGetSignersFromContract func(chain *BlockChain, block *types.Block) ([]common.Address, error)
	
	// HookRandomizeSigners randomizes signer order for a round
	HookRandomizeSigners func(masternodes []common.Address, round uint64) []common.Address
}

// XDCBlockchainContext provides XDC-specific blockchain context
type XDCBlockchainContext struct {
	// IPCEndpoint is the IPC endpoint for contract calls
	IPCEndpoint string
	
	// Database for XDCx trading state
	XDCxDb ethdb.Database
	
	// Hooks for XDPoS consensus
	Hooks *BlockChainHooks
}

// GetMasternodes returns the masternode list for the given epoch
func (bc *BlockChain) GetMasternodes(epoch uint64) []common.Address {
	return rawdb.ReadMasternodeList(bc.db, epoch)
}

// SetMasternodes stores the masternode list for the given epoch
func (bc *BlockChain) SetMasternodes(epoch uint64, masternodes []common.Address) {
	rawdb.WriteMasternodeList(bc.db, epoch, masternodes)
}

// GetPenalizedValidators returns the penalized validators for the given epoch
func (bc *BlockChain) GetPenalizedValidators(epoch uint64) []common.Address {
	return rawdb.ReadPenalizedList(bc.db, epoch)
}

// SetPenalizedValidators stores the penalized validators for the given epoch
func (bc *BlockChain) SetPenalizedValidators(epoch uint64, penalized []common.Address) {
	rawdb.WritePenalizedList(bc.db, epoch, penalized)
}

// GetBlockSigner returns the signer of a block
func (bc *BlockChain) GetBlockSigner(number uint64) common.Address {
	return rawdb.ReadBlockSigner(bc.db, number)
}

// SetBlockSigner stores the signer of a block
func (bc *BlockChain) SetBlockSigner(number uint64, signer common.Address) {
	rawdb.WriteBlockSigner(bc.db, number, signer)
}

// IsCheckpoint returns true if the block number is a checkpoint (epoch switch)
func (bc *BlockChain) IsCheckpoint(number uint64) bool {
	config := bc.Config()
	if config.XDPoS == nil {
		return false
	}
	return number%config.XDPoS.Epoch == 0
}

// GetEpochNumber returns the epoch number for a given block number
func (bc *BlockChain) GetEpochNumber(blockNumber uint64) uint64 {
	config := bc.Config()
	if config.XDPoS == nil || config.XDPoS.Epoch == 0 {
		return 0
	}
	return blockNumber / config.XDPoS.Epoch
}

// IsGapBlock returns true if the block is a gap block (epoch - gap)
func (bc *BlockChain) IsGapBlock(number uint64) bool {
	config := bc.Config()
	if config.XDPoS == nil {
		return false
	}
	gap := config.XDPoS.Gap
	epoch := config.XDPoS.Epoch
	return number%epoch == epoch-gap
}

// UpdateM1 updates masternode list for the next epoch
// Called at gap block (Epoch - Gap)
func (bc *BlockChain) UpdateM1() error {
	currentBlock := bc.CurrentBlock()
	if currentBlock == nil {
		return errors.New("current block is nil")
	}
	
	config := bc.Config()
	if config.XDPoS == nil {
		return nil
	}
	
	number := currentBlock.NumberU64()
	if !bc.IsGapBlock(number) {
		return nil
	}
	
	epoch := bc.GetEpochNumber(number) + 1 // Next epoch
	
	log.Info("Updating masternode list for next epoch", "currentBlock", number, "nextEpoch", epoch)
	
	// In a full implementation, this would:
	// 1. Get candidates from validator contract
	// 2. Sort by stake
	// 3. Select top N as masternodes
	// 4. Store in database
	
	return nil
}

// GetTradingStateRoot gets the trading state root for a block
func (bc *BlockChain) GetTradingStateRoot(blockHash common.Hash) common.Hash {
	return rawdb.ReadTradingStateRoot(bc.db, blockHash)
}

// SetTradingStateRoot sets the trading state root for a block
func (bc *BlockChain) SetTradingStateRoot(blockHash common.Hash, root common.Hash) {
	rawdb.WriteTradingStateRoot(bc.db, blockHash, root)
}

// GetLendingStateRoot gets the lending state root for a block
func (bc *BlockChain) GetLendingStateRoot(blockHash common.Hash) common.Hash {
	return rawdb.ReadLendingStateRoot(bc.db, blockHash)
}

// SetLendingStateRoot sets the lending state root for a block
func (bc *BlockChain) SetLendingStateRoot(blockHash common.Hash, root common.Hash) {
	rawdb.WriteLendingStateRoot(bc.db, blockHash, root)
}

// IsTIPXDCX returns whether XDCX trading is enabled at the given block
func (c *params.ChainConfig) IsTIPXDCX(num *big.Int) bool {
	// XDCX is enabled after a certain block
	// For mainnet, this is block 0 (always enabled)
	return true
}

// IsTIPXDCXReceiver returns whether XDCX receiver is enabled at the given block
func (c *params.ChainConfig) IsTIPXDCXReceiver(num *big.Int) bool {
	return c.IsTIPXDCX(num)
}

// IsTIPSigning returns whether the new signing scheme is enabled
func (c *params.ChainConfig) IsTIPSigning(num *big.Int) bool {
	// New signing scheme enabled after XDPoS 2.0
	if c.XDPoS == nil {
		return false
	}
	return num.Uint64() >= c.XDPoS.V2.SwitchBlock.Uint64()
}

// GetBlocksHashCache gets cached block hashes at a given height
// Used for fork tracking and finality
func (bc *BlockChain) GetBlocksHashCache(number uint64) []common.Hash {
	// This is a placeholder - actual implementation would use LRU cache
	block := bc.GetBlockByNumber(number)
	if block == nil {
		return nil
	}
	return []common.Hash{block.Hash()}
}

// UpdateBlocksHashCache updates the block hash cache
func (bc *BlockChain) UpdateBlocksHashCache(block *types.Block) {
	// Placeholder for block hash cache update
	// In production, this maintains a cache of block hashes per height
	// for tracking forks
}

// AreTwoBlockSamePath checks if two blocks are on the same chain path
func (bc *BlockChain) AreTwoBlockSamePath(hash1, hash2 common.Hash) bool {
	block1 := bc.GetBlockByHash(hash1)
	block2 := bc.GetBlockByHash(hash2)
	
	if block1 == nil || block2 == nil {
		return false
	}
	
	// Check if one is ancestor of the other
	if block1.NumberU64() > block2.NumberU64() {
		// Walk back block1 to block2's height
		for block1.NumberU64() > block2.NumberU64() {
			block1 = bc.GetBlock(block1.ParentHash(), block1.NumberU64()-1)
			if block1 == nil {
				return false
			}
		}
		return block1.Hash() == block2.Hash()
	}
	
	// Walk back block2 to block1's height
	for block2.NumberU64() > block1.NumberU64() {
		block2 = bc.GetBlock(block2.ParentHash(), block2.NumberU64()-1)
		if block2 == nil {
			return false
		}
	}
	return block1.Hash() == block2.Hash()
}
