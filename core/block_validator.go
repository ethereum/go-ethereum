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
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/trie"
)

// BlockValidator is responsible for validating block headers, uncles and
// processed state.
//
// BlockValidator implements Validator.
type BlockValidator struct {
	config *params.ChainConfig // Chain configuration options
	bc     *BlockChain         // Canonical block chain
	engine consensus.Engine    // Consensus engine used for validating
}

// NewBlockValidator returns a new block validator which is safe for re-use
func NewBlockValidator(config *params.ChainConfig, blockchain *BlockChain, engine consensus.Engine) *BlockValidator {
	validator := &BlockValidator{
		config: config,
		engine: engine,
		bc:     blockchain,
	}
	return validator
}

// ValidateBody validates the given block's uncles and verifies the block
// header's transaction and uncle roots. The headers are assumed to be already
// validated at this point.
func (v *BlockValidator) ValidateBody(block *types.Block) error {
	// Check whether the block's known, and if not, that it's linkable
	if v.bc.HasBlockAndState(block.Hash(), block.NumberU64()) {
		return ErrKnownBlock
	}
	// Header validity is known at this point, check the uncles and transactions
	header := block.Header()
	if err := v.engine.VerifyUncles(v.bc, block); err != nil {
		return err
	}
	if hash := types.CalcUncleHash(block.Uncles()); hash != header.UncleHash {
		return fmt.Errorf("uncle root hash mismatch: have %x, want %x", hash, header.UncleHash)
	}
	if hash := types.DeriveSha(block.Transactions(), trie.NewStackTrie(nil)); hash != header.TxHash {
		return fmt.Errorf("transaction root hash mismatch: have %x, want %x", hash, header.TxHash)
	}
	if !v.bc.HasBlockAndState(block.ParentHash(), block.NumberU64()-1) {
		if !v.bc.HasBlock(block.ParentHash(), block.NumberU64()-1) {
			return consensus.ErrUnknownAncestor
		}
		return consensus.ErrPrunedAncestor
	}
	return nil
}

// ValidateState validates the various changes that happen after a state
// transition, such as amount of used gas, the receipt roots and the state root
// itself. ValidateState returns a database batch if the validation was a success
// otherwise nil and an error is returned.
func (v *BlockValidator) ValidateState(block *types.Block, statedb *state.StateDB, receipts types.Receipts, usedGas uint64) error {
	header := block.Header()
	if block.GasUsed() != usedGas {
		return fmt.Errorf("invalid gas used (remote: %d local: %d)", block.GasUsed(), usedGas)
	}
	// Validate the received block's bloom with the one derived from the generated receipts.
	// For valid blocks this should always validate to true.
	rbloom := types.CreateBloom(receipts)
	if rbloom != header.Bloom {
		return fmt.Errorf("invalid bloom (remote: %x  local: %x)", header.Bloom, rbloom)
	}
	// Tre receipt Trie's root (R = (Tr [[H1, R1], ... [Hn, Rn]]))
	receiptSha := types.DeriveSha(receipts, trie.NewStackTrie(nil))
	if receiptSha != header.ReceiptHash {
		return fmt.Errorf("invalid receipt root hash (remote: %x local: %x)", header.ReceiptHash, receiptSha)
	}
	// Validate the state root against the received state root and throw
	// an error if they don't match.
	if root := statedb.IntermediateRoot(v.config.IsEIP158(header.Number)); header.Root != root {
		return fmt.Errorf("invalid merkle root (remote: %x local: %x)", header.Root, root)
	}
	return nil
}

// CalcGasLimit computes the gas limit of the next block after parent. It aims
// to keep the baseline gas above the provided floor, and increase it towards the
// ceil if the blocks are full. If the ceil is exceeded, it will always decrease
// the gas allowance.
func CalcGasLimit(parent *types.Block, gasFloor, gasCeil uint64) uint64 {
	// contrib = (parentGasUsed * 3 / 2) / 1024
	contrib := (parent.GasUsed() + parent.GasUsed()/2) / params.GasLimitBoundDivisor

	// decay = parentGasLimit / 1024 -1
	decay := parent.GasLimit()/params.GasLimitBoundDivisor - 1

	/*
		strategy: gasLimit of block-to-mine is set based on parent's
		gasUsed value.  if parentGasUsed > parentGasLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentGasLimit * (2/3) parentGasUsed is.
	*/
	limit := parent.GasLimit() - decay + contrib
	if limit < params.MinGasLimit {
		limit = params.MinGasLimit
	}
	// If we're outside our allowed gas range, we try to hone towards them
	if limit < gasFloor {
		limit = parent.GasLimit() + decay
		if limit > gasFloor {
			limit = gasFloor
		}
	} else if limit > gasCeil {
		limit = parent.GasLimit() - decay
		if limit < gasCeil {
			limit = gasCeil
		}
	}
	return limit
}

func CalcGasLimitAndBaseFee(config *params.ChainConfig, parent *types.Block, gasFloor, gasCeil uint64) (uint64, *big.Int) {
	if !config.IsEIP1559(new(big.Int).Add(parent.Number(), common.Big1)) {
		return CalcGasLimit(parent, gasFloor, gasCeil), nil
	}
	return calcGasLimitAndBaseFee(config, parent)
}

// calcGasLimitAndBaseFee returns the EIP1559GasLimit and the BaseFee
// Start at 50 : 50 and then shift to 0 : 100
// The GasLimit for the legacy pool is (params.MaxGasEIP1559 - EIP1559GasLimit)
func calcGasLimitAndBaseFee(config *params.ChainConfig, parent *types.Block) (uint64, *big.Int) {
	height := new(big.Int).Add(parent.Number(), common.Big1)

	// If we are at the block of EIP1559 activation then the BaseFee is set to the initial value
	// and the GasLimit is split evenly between the two pools
	if config.EIP1559Block.Cmp(height) == 0 {
		return config.EIP1559.MaxGas / 2, new(big.Int).SetUint64(config.EIP1559.InitialBaseFee)
	}

	// Otherwise, calculate the BaseFee
	// As a default strategy, miners set BASEFEE as follows. Let delta = block.gas_used - TARGET_GASUSED (possibly negative).
	// Set BASEFEE = PARENT_BASEFEE + PARENT_BASEFEE * delta // TARGET_GASUSED // BASEFEE_MAX_CHANGE_DENOMINATOR,

	delta := new(big.Int).Sub(new(big.Int).SetUint64(parent.GasUsed()), new(big.Int).SetUint64(config.EIP1559.TargetGasUsed))
	mul := new(big.Int).Mul(parent.BaseFee(), delta)
	div := new(big.Int).Div(mul, new(big.Int).SetUint64(config.EIP1559.TargetGasUsed))
	div2 := new(big.Int).Div(div, new(big.Int).SetUint64(config.EIP1559.BaseFeeMaxChangeDenominator))
	baseFee := new(big.Int).Add(parent.BaseFee(), div2)

	// A valid BASEFEE is one such that abs(BASEFEE - PARENT_BASEFEE) <= max(1, PARENT_BASEFEE // BASEFEE_MAX_CHANGE_DENOMINATOR)
	diff := new(big.Int).Sub(baseFee, parent.BaseFee())
	neg := false
	if diff.Sign() < 0 {
		neg = true
		diff.Neg(diff)
	}
	max := new(big.Int).Div(parent.BaseFee(), new(big.Int).SetUint64(config.EIP1559.BaseFeeMaxChangeDenominator))
	if max.Cmp(common.Big1) < 0 {
		max = common.Big1
	}
	// If derived BaseFee is not valid, restrict it within the bounds
	if diff.Cmp(max) > 0 {
		if neg {
			max.Neg(max)
		}
		baseFee.Set(new(big.Int).Add(parent.BaseFee(), max))
	}

	// If EIP1559 is finalized, our limit for the EIP1559 pool is the entire max limit
	if config.IsEIP1559Finalized(new(big.Int).Add(parent.Number(), common.Big1)) {
		return params.MaxGasEIP1559, baseFee
	}

	// Otherwise calculate how much of the MaxGasEIP1559 serves as the limit for the EIP1559 pool
	// The GasLimit for the legacy pool is (params.MaxGasEIP1559 - eip1559GasLimit)
	numOfIncrements := new(big.Int).Sub(height, config.EIP1559Block).Uint64()
	eip1559GasLimit := (config.EIP1559.MaxGas / 2) + (numOfIncrements * config.EIP1559.GasIncrementAmount)
	return eip1559GasLimit, baseFee
}
