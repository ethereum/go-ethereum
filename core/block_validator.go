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
	"time"

	"github.com/scroll-tech/go-ethereum/consensus"
	"github.com/scroll-tech/go-ethereum/core/rawdb"
	"github.com/scroll-tech/go-ethereum/core/state"
	"github.com/scroll-tech/go-ethereum/core/types"
	"github.com/scroll-tech/go-ethereum/log"
	"github.com/scroll-tech/go-ethereum/metrics"
	"github.com/scroll-tech/go-ethereum/params"
	"github.com/scroll-tech/go-ethereum/trie"
)

var (
	validateL1MessagesTimer = metrics.NewRegisteredTimer("validator/l1msg", nil)
	asyncValidatorTimer     = metrics.NewRegisteredTimer("validator/async", nil)
)

// BlockValidator is responsible for validating block headers, uncles and
// processed state.
//
// BlockValidator implements Validator.
type BlockValidator struct {
	config         *params.ChainConfig      // Chain configuration options
	bc             *BlockChain              // Canonical block chain
	engine         consensus.Engine         // Consensus engine used for validating
	asyncValidator func(*types.Block) error // Asynchronously run a validation task
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

// WithAsyncValidator sets up an async validator to be triggered on each new block
func (v *BlockValidator) WithAsyncValidator(asyncValidator func(*types.Block) error) Validator {
	v.asyncValidator = asyncValidator
	return v
}

// ValidateBody validates the given block's uncles and verifies the block
// header's transaction and uncle roots. The headers are assumed to be already
// validated at this point.
func (v *BlockValidator) ValidateBody(block *types.Block) error {
	// Check whether the block's known, and if not, that it's linkable
	if v.bc.HasBlockAndState(block.Hash(), block.NumberU64()) {
		return ErrKnownBlock
	}
	if !v.config.Scroll.IsValidTxCount(len(block.Transactions())) {
		return consensus.ErrInvalidTxCount
	}
	// Check if block payload size is smaller than the max size
	if !v.config.Scroll.IsValidBlockSize(block.PayloadSize()) {
		return ErrInvalidBlockPayloadSize
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
	if err := v.ValidateL1Messages(block); err != nil {
		return err
	}

	if v.asyncValidator != nil {
		asyncStart := time.Now()
		if err := v.asyncValidator(block); err != nil {
			return err
		}
		asyncValidatorTimer.UpdateSince(asyncStart)
	}
	return nil
}

// ValidateL1Messages validates L1 messages contained in a block.
// We check the following conditions:
// - L1 messages are in a contiguous section at the front of the block.
// - The first L1 message's QueueIndex is right after the last L1 message included in the chain.
// - L1 messages follow the QueueIndex order.
// - The L1 messages included in the block match the node's view of the L1 ledger.
func (v *BlockValidator) ValidateL1Messages(block *types.Block) error {
	defer func(t0 time.Time) {
		validateL1MessagesTimer.Update(time.Since(t0))
	}(time.Now())

	// skip DB read if the block contains no L1 messages
	if !block.ContainsL1Messages() {
		return nil
	}

	blockHash := block.Hash()

	if v.config.Scroll.L1Config == nil {
		// TODO: should we allow follower nodes to skip L1 message verification?
		panic("Running on L1Message-enabled network but no l1Config was provided")
	}

	nextQueueIndex := rawdb.ReadFirstQueueIndexNotInL2Block(v.bc.db, block.ParentHash())
	if nextQueueIndex == nil {
		// we'll reprocess this block at a later time
		return consensus.ErrMissingL1MessageData
	}
	queueIndex := *nextQueueIndex

	L1SectionOver := false
	it := rawdb.IterateL1MessagesFrom(v.bc.db, queueIndex)

	for _, tx := range block.Transactions() {
		if !tx.IsL1MessageTx() {
			L1SectionOver = true
			continue // we do not verify L2 transactions here
		}

		// check that L1 messages are before L2 transactions
		if L1SectionOver {
			return consensus.ErrInvalidL1MessageOrder
		}

		// queue index cannot decrease
		txQueueIndex := tx.AsL1MessageTx().QueueIndex

		if txQueueIndex < queueIndex {
			return consensus.ErrInvalidL1MessageOrder
		}

		// skipped messages
		// TODO: consider verifying that skipped messages overflow
		for index := queueIndex; index < txQueueIndex; index++ {
			if exists := it.Next(); !exists {
				if err := it.Error(); err != nil {
					log.Error("Unexpected DB error in ValidateL1Messages", "err", err, "queueIndex", queueIndex)
				}
				// the message in this block is not available in our local db.
				// we'll reprocess this block at a later time.
				return consensus.ErrMissingL1MessageData
			}

			l1msg := it.L1Message()
			skippedTx := types.NewTx(&l1msg)
			log.Debug("Skipped L1 message", "queueIndex", index, "tx", skippedTx.Hash().String(), "block", blockHash.String())
			rawdb.WriteSkippedTransaction(v.bc.db, skippedTx, nil, "unknown", block.NumberU64(), &blockHash)
		}

		queueIndex = txQueueIndex + 1

		if exists := it.Next(); !exists {
			if err := it.Error(); err != nil {
				log.Error("Unexpected DB error in ValidateL1Messages", "err", err, "queueIndex", txQueueIndex)
			}
			// the message in this block is not available in our local db.
			// we'll reprocess this block at a later time.
			return consensus.ErrMissingL1MessageData
		}

		// check that the L1 message in the block is the same that we collected from L1
		msg := it.L1Message()
		expectedHash := types.NewTx(&msg).Hash()

		if tx.Hash() != expectedHash {
			return consensus.ErrUnknownL1Message
		}
	}

	// TODO: consider adding a rule to enforce L1Config.NumL1MessagesPerBlock.
	// If there are L1 messages available, sequencer nodes should include them.
	// However, this is hard to enforce as different nodes might have different views of L1.

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
	shouldValidateStateRoot := v.config.Scroll.UseZktrie != v.config.IsEuclid(header.Time)
	if root := statedb.IntermediateRoot(v.config.IsEIP158(header.Number)); shouldValidateStateRoot && header.Root != root {
		return fmt.Errorf("invalid merkle root (remote: %x local: %x)", header.Root, root)
	}
	return nil
}

// CalcGasLimit computes the gas limit of the next block after parent. It aims
// to keep the baseline gas close to the provided target, and increase it towards
// the target if the baseline gas is lower.
func CalcGasLimit(parentGasLimit, desiredLimit uint64) uint64 {
	delta := parentGasLimit/params.GasLimitBoundDivisor - 1
	limit := parentGasLimit
	if desiredLimit < params.MinGasLimit {
		desiredLimit = params.MinGasLimit
	}
	// If we're outside our allowed gas range, we try to hone towards them
	if limit < desiredLimit {
		limit = parentGasLimit + delta
		if limit > desiredLimit {
			limit = desiredLimit
		}
		return limit
	}
	if limit > desiredLimit {
		limit = parentGasLimit - delta
		if limit < desiredLimit {
			limit = desiredLimit
		}
	}
	return limit
}
