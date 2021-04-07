// Copyright 2021 The go-ethereum Authors
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

package hybrid

import (
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/consensus/ethash"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

// Casper proof-of-stake protocol constants.
var (
	casperDifficulty = common.Big1          // The default block difficulty in the casper
	casperNonce      = types.EncodeNonce(1) // The default block nonce in the casper
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errTooManyUncles     = errors.New("too many uncles")
	errInvalidDifficulty = errors.New("invalid difficulty")
	errInvalidMixDigest  = errors.New("invalid mix digest")
	errInvalidNonce      = errors.New("invalid nonce")
)

// Hybrid is a consensus engine combines the proof-of-work and proof-of-stake
// implementing the ethash and casper algorithm. There is a special flag inside
// to decide whether to use ethash rules or capser rules. The transition rule
// is described in the eth1/2 merge spec.
// https://hackmd.io/@n0ble/ethereum_consensus_upgrade_mainnet_perspective#Transition-process
// The casper here is a tailored consensus engine with partial functions which
// is only used for necessary consensus checks.
type Hybrid struct {
	ethash *ethash.Ethash

	// transitioned is the flag whether the chain has finished the ethash->casper
	// transition. It's triggered by receiving the first "FinaliseBlock" message
	// from the external consensus engine.
	transitioned uint32
}

// NewHybrid creates a hybrid consensus engine.
func NewHybrid(config ethash.Config, notify []string, noverify bool, transitioned bool) *Hybrid {
	engine := &Hybrid{ethash: ethash.New(config, notify, noverify)}
	if transitioned {
		engine.SetTransitioned()
	}
	return engine
}

// Author implements consensus.Engine, returning the header's coinbase as the
// verified author of the block. It's same in both ethash and casper.
func (hybrid *Hybrid) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum hybrid engine.
func (hybrid *Hybrid) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	// Delegate the verification to the ethash if the transition is
	// still not finished yet.
	if !hybrid.IsTransitioned() {
		return hybrid.ethash.VerifyHeader(chain, header, seal)
	}
	// Short circuit if the header is known, or its parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	return hybrid.verifyHeader(chain, header, parent)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (hybrid *Hybrid) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	// Delegate the verification to the ethash if the transition is
	// still not finished yet.
	if !hybrid.IsTransitioned() {
		return hybrid.ethash.VerifyHeaders(chain, headers, seals)
	}
	var (
		abort   = make(chan struct{})
		results = make(chan error, len(headers))
	)
	go func() {
		for i, header := range headers {
			var parent *types.Header
			if i == 0 {
				parent = chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
				select {
				case <-abort:
					return
				case results <- consensus.ErrUnknownAncestor:
				}
				continue
			} else {
				parent = headers[i-1]
			}
			err := hybrid.verifyHeader(chain, header, parent)
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock Ethereum hybrid engine.
func (hybrid *Hybrid) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	// Delegate the verification to the ethash if the transition is
	// still not finished yet.
	if !hybrid.IsTransitioned() {
		return hybrid.ethash.VerifyUncles(chain, block)
	}
	// Verify that there is no uncle block. It's explicitly disabled in the casper
	if len(block.Uncles()) > 0 {
		return errTooManyUncles
	}
	return nil
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum capser engine. The difference between the casper and ethash is
// (a) the difficulty, mixhash, nonce, extradata and unclehash are expected
//     to be the desired constants
// (b) the timestamp is not verified anymore
func (hybrid *Hybrid) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if len(header.Extra) != 0 {
		return fmt.Errorf("non-empty extra-data(%d)", len(header.Extra))
	}
	// Verify the block's difficulty to ensure it's the default constant
	if casperDifficulty.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, casperDifficulty)
	}
	// Verify that the gas limit is <= 2^63-1
	cap := uint64(0x7fffffffffffffff)
	if header.GasLimit > cap {
		return fmt.Errorf("invalid gasLimit: have %v, max %v", header.GasLimit, cap)
	}
	// Verify that the gasUsed is <= gasLimit
	if header.GasUsed > header.GasLimit {
		return fmt.Errorf("invalid gasUsed: have %d, gasLimit %d", header.GasUsed, header.GasLimit)
	}
	// Verify that the gas limit remains within allowed bounds
	diff := int64(parent.GasLimit) - int64(header.GasLimit)
	if diff < 0 {
		diff *= -1
	}
	limit := parent.GasLimit / params.GasLimitBoundDivisor
	if uint64(diff) >= limit || header.GasLimit < params.MinGasLimit {
		return fmt.Errorf("invalid gas limit: have %d, want %d += %d", header.GasLimit, parent.GasLimit, limit)
	}
	// Verify that the block number is parent's +1
	if diff := new(big.Int).Sub(header.Number, parent.Number); diff.Cmp(big.NewInt(1)) != 0 {
		return consensus.ErrInvalidNumber
	}
	// Verify the seal parts. Ensure the mixhash and nonce is the expected value.
	if header.MixDigest != (common.Hash{}) {
		return errInvalidMixDigest
	}
	if header.Nonce != casperNonce {
		return errInvalidNonce
	}
	return nil
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the hybrid protocol. The changes are done inline.
func (hybrid *Hybrid) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	// Delegate the preparation to the ethash if the transition is
	// still not finished yet.
	if !hybrid.IsTransitioned() {
		return hybrid.ethash.Prepare(chain, header)
	}
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Difficulty = casperDifficulty
	return nil
}

// Finalize implements consensus.Engine, setting the final state on the header
func (hybrid *Hybrid) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	// Delegate the finalization to the ethash if the transition is
	// still not finished yet.
	if !hybrid.IsTransitioned() {
		hybrid.ethash.Finalize(chain, header, state, txs, uncles)
		return
	}
	// The block reward is no longer handled here. It's done by the
	// external consensus engine.
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
}

// FinalizeAndAssemble implements consensus.Engine, setting the final state and
// assembling the block.
func (hybrid *Hybrid) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// Delegate the finalization to the ethash if the transition is
	// still not finished yet.
	if !hybrid.IsTransitioned() {
		return hybrid.ethash.FinalizeAndAssemble(chain, header, state, txs, uncles, receipts)
	}
	// Finalize and assemble the block
	hybrid.Finalize(chain, header, state, txs, uncles)
	return types.NewBlock(header, txs, uncles, receipts, trie.NewStackTrie(nil)), nil
}

// Seal generates a new sealing request for the given input block and pushes
// the result into the given channel.
//
// Note, the method returns immediately and will send the result async. More
// than one result may also be returned depending on the consensus algorithm.
func (hybrid *Hybrid) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	// Delegate the sealing to the ethash if the transition is
	// still not finished yet.
	if !hybrid.IsTransitioned() {
		return hybrid.ethash.Seal(chain, block, results, stop)
	}
	// The seal verification is done by the external consensus engine,
	// Return directly without pushing any block back.
	return nil
}

// SealHash returns the hash of a block prior to it being sealed. It's same in
// both ethash and casper.
func (hybrid *Hybrid) SealHash(header *types.Header) (hash common.Hash) {
	return hybrid.ethash.SealHash(header)
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func (hybrid *Hybrid) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	// Delegate the calculation to the ethash if the transition is
	// still not finished yet.
	if !hybrid.IsTransitioned() {
		return hybrid.ethash.CalcDifficulty(chain, time, parent)
	}
	return casperDifficulty
}

// APIs implements consensus.Engine, returning the user facing RPC APIs.
func (hybrid *Hybrid) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return hybrid.ethash.APIs(chain)
}

// Close shutdowns the consensus engine
func (hybrid *Hybrid) Close() error {
	return hybrid.ethash.Close()
}

// SetThreads updates the number of mining threads currently enabled. Calling
// this method does not start mining, only sets the thread count. If zero is
// specified, the miner will use all cores of the machine. Setting a thread
// count below zero is allowed and will cause the miner to idle, without any
// work being done.
func (hybrid *Hybrid) SetThreads(threads int) {
	if !hybrid.IsTransitioned() {
		hybrid.ethash.SetThreads(threads)
	}
}

// SetTransitioned marks the transition has been done.
func (hybrid *Hybrid) SetTransitioned() {
	atomic.StoreUint32(&hybrid.transitioned, 1)
}

// IsTransitioned reports whether the transition has finished.
func (hybrid *Hybrid) IsTransitioned() bool {
	return atomic.LoadUint32(&hybrid.transitioned) == 1
}
