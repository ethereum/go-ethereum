// Copyright 2023 The XDC Network Authors
// XDPoS-specific state processing extensions

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/params"
	"github.com/holiman/uint256"
)

// XDCStateProcessor handles XDPoS-specific state transitions
type XDCStateProcessor struct {
	config *params.ChainConfig
	bc     *BlockChain
}

// NewXDCStateProcessor creates a new XDC state processor
func NewXDCStateProcessor(config *params.ChainConfig, bc *BlockChain) *XDCStateProcessor {
	return &XDCStateProcessor{
		config: config,
		bc:     bc,
	}
}

// ProcessXDCReward processes masternode rewards at epoch boundaries
func (p *XDCStateProcessor) ProcessXDCReward(
	header *types.Header,
	statedb *state.StateDB,
	masternodes []common.Address,
) error {
	if len(masternodes) == 0 {
		return nil
	}

	// Default block reward: 5000 XDC in wei
	blockReward := new(big.Int).Mul(big.NewInt(5000), big.NewInt(1e18))
	perMasternode := new(big.Int).Div(blockReward, big.NewInt(int64(len(masternodes))))

	// Convert to uint256 for state operations
	reward, _ := uint256.FromBig(perMasternode)

	// Distribute rewards
	for _, mn := range masternodes {
		statedb.AddBalance(mn, reward, 0)
	}

	return nil
}

// ProcessXDCPenalty processes penalties for misbehaving masternodes
func (p *XDCStateProcessor) ProcessXDCPenalty(
	header *types.Header,
	statedb *state.StateDB,
	penalties []common.Address,
) error {
	// Penalty logic - typically handled by smart contract
	return nil
}

// ProcessXDCxTrade processes XDCx trade transactions
func (p *XDCStateProcessor) ProcessXDCxTrade(
	tx *types.Transaction,
	statedb *state.StateDB,
	cfg vm.Config,
) error {
	// XDCx trade processing - stub for integration
	return nil
}

// ProcessLendingTrade processes lending trade transactions
func (p *XDCStateProcessor) ProcessLendingTrade(
	tx *types.Transaction,
	statedb *state.StateDB,
	cfg vm.Config,
) error {
	// Lending trade processing - stub for integration
	return nil
}
