// Copyright 2023 The XDC Network Authors
// This file is part of the XDC Network library.
//
// The XDC Network library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/params"
)

// XDCStateProcessor extends StateProcessor with XDPoS-specific processing
type XDCStateProcessor struct {
	*StateProcessor
	config *params.ChainConfig
}

// NewXDCStateProcessor creates a new XDC state processor
func NewXDCStateProcessor(config *params.ChainConfig, bc *BlockChain, engine consensus.Engine) *XDCStateProcessor {
	return &XDCStateProcessor{
		StateProcessor: NewStateProcessor(config, bc, engine),
		config:         config,
	}
}

// ProcessXDC processes XDPoS-specific state transitions
func (p *XDCStateProcessor) ProcessXDC(
	block *types.Block,
	statedb *state.StateDB,
	cfg vm.Config,
	feeCapacity state.FeeCapacity,
) ([]*types.Receipt, []*types.Log, uint64, error) {
	// Call base processor first
	receipts, logs, usedGas, err := p.Process(block, statedb, cfg, feeCapacity)
	if err != nil {
		return nil, nil, 0, err
	}

	// Apply XDPoS-specific state changes
	header := block.Header()

	// Process rewards at epoch switch
	if p.isEpochSwitch(header.Number.Uint64()) {
		if err := p.processEpochRewards(statedb, header); err != nil {
			log.Error("Failed to process epoch rewards", "err", err)
		}
	}

	// Process penalties
	if err := p.processPenalties(statedb, header); err != nil {
		log.Error("Failed to process penalties", "err", err)
	}

	return receipts, logs, usedGas, nil
}

// isEpochSwitch checks if block number is an epoch switch
func (p *XDCStateProcessor) isEpochSwitch(number uint64) bool {
	if p.config.XDPoS == nil || p.config.XDPoS.Epoch == 0 {
		return false
	}
	return number%p.config.XDPoS.Epoch == 0
}

// processEpochRewards processes rewards at epoch boundaries
func (p *XDCStateProcessor) processEpochRewards(statedb *state.StateDB, header *types.Header) error {
	if p.config.XDPoS == nil {
		return nil
	}

	// Calculate total rewards for the epoch
	// Standard block reward: 250 XDC
	blockReward := new(big.Int).Mul(big.NewInt(250), big.NewInt(1e18))

	// In XDPoS, rewards go to:
	// - Block signer (coinbase)
	// - Voters who voted for the signer

	// For now, just add to coinbase
	statedb.AddBalance(header.Coinbase, blockReward, 0)

	log.Debug("Processed epoch rewards",
		"block", header.Number,
		"signer", header.Coinbase,
		"reward", blockReward,
	)

	return nil
}

// processPenalties handles validator penalties
func (p *XDCStateProcessor) processPenalties(statedb *state.StateDB, header *types.Header) error {
	// Penalties would be applied based on:
	// - Missing blocks
	// - Invalid votes
	// - Double signing

	// This is a placeholder - actual implementation would:
	// 1. Check missed block count for validators
	// 2. Apply slashing if threshold exceeded
	// 3. Update validator state

	return nil
}

// ApplyXDCTransaction applies a transaction with XDPoS-specific handling
func ApplyXDCTransaction(
	config *params.ChainConfig,
	bc ChainContext,
	author *common.Address,
	gp *GasPool,
	statedb *state.StateDB,
	header *types.Header,
	tx *types.Transaction,
	usedGas *uint64,
	cfg vm.Config,
	feeCapacity state.FeeCapacity,
) (*types.Receipt, error) {
	// Check if this is a special XDPoS transaction
	if isXDPoSSpecialTx(tx) {
		return applyXDPoSSpecialTx(config, bc, author, gp, statedb, header, tx, usedGas, cfg)
	}

	// Apply as normal transaction
	return ApplyTransaction(config, bc, author, gp, statedb, header, tx, usedGas, cfg)
}

// isXDPoSSpecialTx checks if transaction is XDPoS-specific
func isXDPoSSpecialTx(tx *types.Transaction) bool {
	to := tx.To()
	if to == nil {
		return false
	}

	// Check for special contract addresses
	specialAddresses := []common.Address{
		common.HexToAddress("0x0000000000000000000000000000000000000088"), // Validator contract
		common.HexToAddress("0x0000000000000000000000000000000000000089"), // Block signer contract
		common.HexToAddress("0x0000000000000000000000000000000000000090"), // Randomize contract
	}

	for _, addr := range specialAddresses {
		if *to == addr {
			return true
		}
	}

	return false
}

// applyXDPoSSpecialTx applies XDPoS-specific transactions
func applyXDPoSSpecialTx(
	config *params.ChainConfig,
	bc ChainContext,
	author *common.Address,
	gp *GasPool,
	statedb *state.StateDB,
	header *types.Header,
	tx *types.Transaction,
	usedGas *uint64,
	cfg vm.Config,
) (*types.Receipt, error) {
	// Special handling for XDPoS contract interactions
	// This would handle:
	// - Validator registration/resignation
	// - Vote casting
	// - Reward claims
	// - etc.

	// For now, apply as normal transaction
	return ApplyTransaction(config, bc, author, gp, statedb, header, tx, usedGas, cfg)
}

// CalculateXDCReward calculates the block reward for XDPoS
func CalculateXDCReward(blockNumber *big.Int, config *params.XDPoSConfig) *big.Int {
	if config == nil {
		return big.NewInt(0)
	}

	// Base reward: 250 XDC per block
	baseReward := new(big.Int).Mul(big.NewInt(250), big.NewInt(1e18))

	// Could add halvings or other adjustments here based on block number

	return baseReward
}

// DistributeRewards distributes rewards to signer and voters
func DistributeRewards(
	statedb *state.StateDB,
	header *types.Header,
	reward *big.Int,
	voterRewardPercent int,
) {
	if reward.Sign() <= 0 {
		return
	}

	// Calculate voter reward portion (e.g., 40%)
	voterPortion := new(big.Int).Mul(reward, big.NewInt(int64(voterRewardPercent)))
	voterPortion.Div(voterPortion, big.NewInt(100))

	// Signer gets remaining
	signerPortion := new(big.Int).Sub(reward, voterPortion)

	// Add signer portion to coinbase
	statedb.AddBalance(header.Coinbase, signerPortion, 0)

	log.Debug("Distributed rewards",
		"block", header.Number,
		"signer", header.Coinbase,
		"signerReward", signerPortion,
		"voterPortion", voterPortion,
	)

	// Voter distribution would require:
	// 1. Getting voter list from validator contract
	// 2. Calculating each voter's share based on stake
	// 3. Distributing proportionally
}

// Import required types
import (
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/vm"
)
