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

package beacon

import (
	"errors"
	"fmt"
	"math/big"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
)

// Proof-of-stake protocol constants.
var (
	beaconDifficulty = common.Big1          // The default block difficulty in the beacon consensus
	beaconNonce      = types.EncodeNonce(0) // The default block nonce in the beacon consensus
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
	errInvalidUncleHash  = errors.New("invalid uncle hash")
)

// Beacon is a consensus engine combines the ethereum 1 consensus and proof-of-stake
// algorithm. There is a special flag inside to decide whether to use legacy consensus
// rules or new rules. The transition rule is described in the eth1/2 merge spec.
// https://hackmd.io/@n0ble/ethereum_consensus_upgrade_mainnet_perspective#Transition-process
//
// The beacon here is a half-functional consensus engine with partial functions which
// is only used for necessary consensus checks. The legacy consensus engine can be any
// engine implements the consensus interface(except the beacon itself).
type Beacon struct {
	ethone consensus.Engine // Classic consensus engine used in the eth1, ethash or clique

	// transitioned is the flag whether the transition has been triggered.
	// It's triggered by receiving the first "NewHead" message from the
	// external consensus engine.
	transitioned bool
	lock         sync.RWMutex
}

// New creates a consensus engine with the given embedded ethereum 1 engine.
func New(ethone consensus.Engine, transitioned bool) *Beacon {
	if _, ok := ethone.(*Beacon); ok {
		panic("nested consensus engine")
	}
	return &Beacon{
		ethone:       ethone,
		transitioned: transitioned,
	}
}

// Author implements consensus.Engine, returning the header's coinbase as the
// verified author of the block.
func (beacon *Beacon) Author(header *types.Header) (common.Address, error) {
	if !beacon.IsPostMergeHeader(header) {
		return beacon.ethone.Author(header)
	}
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum consensus engine.
func (beacon *Beacon) VerifyHeader(chain consensus.ChainHeaderReader, header *types.Header, seal bool) error {
	if !beacon.IsPostMergeHeader(header) {
		return beacon.ethone.VerifyHeader(chain, header, seal)
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
	return beacon.verifyHeader(chain, header, parent)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (beacon *Beacon) VerifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	if !beacon.IsPostMergeHeader(headers[len(headers)-1]) {
		return beacon.ethone.VerifyHeaders(chain, headers, seals)
	}
	var (
		preHeaders  []*types.Header
		postHeaders []*types.Header
		preSeals    []bool
	)
	for index, header := range headers {
		if beacon.IsPostMergeHeader(header) {
			preHeaders = headers[:index]
			postHeaders = headers[index:]
			preSeals = seals[:index]
			break
		}
	}
	// All the headers have passed the transition point, use new rules.
	if len(preHeaders) == 0 {
		return beacon.verifyHeaders(chain, headers, nil)
	}
	// The transition point exists in the middle, separate the headers
	// into two batches and apply different verification rules for them.
	var (
		abort   = make(chan struct{})
		results = make(chan error, len(headers))
	)
	go func() {
		var (
			old, new, out      = 0, len(preHeaders), 0
			errors             = make([]error, len(headers))
			done               = make([]bool, len(headers))
			oldDone, oldResult = beacon.ethone.VerifyHeaders(chain, preHeaders, preSeals)
			newDone, newResult = beacon.verifyHeaders(chain, postHeaders, preHeaders[len(preHeaders)-1])
		)
		for {
			for ; done[out]; out++ {
				results <- errors[out]
				if out == len(headers)-1 {
					return
				}
			}
			select {
			case err := <-oldResult:
				errors[old], done[old] = err, true
				old++
			case err := <-newResult:
				errors[new], done[new] = err, true
				new++
			case <-abort:
				close(oldDone)
				close(newDone)
				return
			}
		}
	}()
	return abort, results
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock Ethereum consensus engine.
func (beacon *Beacon) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	if !beacon.IsPostMergeHeader(block.Header()) {
		return beacon.ethone.VerifyUncles(chain, block)
	}
	// Verify that there is no uncle block. It's explicitly disabled in the beacon
	if len(block.Uncles()) > 0 {
		return errTooManyUncles
	}
	return nil
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum consensus engine. The difference between the beacon and ethash is
// (a) the difficulty, mixhash, nonce, extradata and unclehash are expected
//     to be the desired constants
// (b) the timestamp is not verified anymore
func (beacon *Beacon) verifyHeader(chain consensus.ChainHeaderReader, header, parent *types.Header) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if len(header.Extra) != 0 {
		return fmt.Errorf("non-empty extra-data(%d)", len(header.Extra))
	}
	// Verify the block's difficulty to ensure it's the default constant
	if beaconDifficulty.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, beaconDifficulty)
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
	// Verify the seal parts. Ensure the mixhash, nonce and uncle hash are the expected value.
	if header.MixDigest != (common.Hash{}) {
		return errInvalidMixDigest
	}
	if header.Nonce != beaconNonce {
		return errInvalidNonce
	}
	if header.UncleHash != types.EmptyUncleHash {
		return errInvalidUncleHash
	}
	return nil
}

// verifyHeaders is similar to verifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications. An additional parent
// header will be passed if the relevant header is not in the database yet.
func (beacon *Beacon) verifyHeaders(chain consensus.ChainHeaderReader, headers []*types.Header, ancestor *types.Header) (chan<- struct{}, <-chan error) {
	var (
		abort   = make(chan struct{})
		results = make(chan error, len(headers))
	)
	go func() {
		for i, header := range headers {
			var parent *types.Header
			if i == 0 {
				if ancestor != nil {
					parent = ancestor
				} else {
					parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
				}
			} else if headers[i-1].Hash() == headers[i].ParentHash {
				parent = headers[i-1]
			}
			if parent == nil {
				select {
				case <-abort:
					return
				case results <- consensus.ErrUnknownAncestor:
				}
				continue
			}
			err := beacon.verifyHeader(chain, header, parent)
			select {
			case <-abort:
				return
			case results <- err:
			}
		}
	}()
	return abort, results
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the beacon protocol. The changes are done inline.
func (beacon *Beacon) Prepare(chain consensus.ChainHeaderReader, header *types.Header) error {
	beacon.lock.RLock()
	defer beacon.lock.RUnlock()

	// Transition isn't triggered yet, use the legacy rules for preparation.
	if !beacon.transitioned {
		return beacon.ethone.Prepare(chain, header)
	}
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Difficulty = beaconDifficulty
	return nil
}

// Finalize implements consensus.Engine, setting the final state on the header
func (beacon *Beacon) Finalize(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header) {
	beacon.lock.RLock()
	defer beacon.lock.RUnlock()

	// Finalize is different with Prepare, it can be used in both block generation
	// and verification. So determine the consensus rules by header type.
	if !beacon.IsPostMergeHeader(header) {
		beacon.ethone.Finalize(chain, header, state, txs, uncles)
		return
	}
	// The block reward is no longer handled here. It's done by the
	// external consensus engine.
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))
}

// FinalizeAndAssemble implements consensus.Engine, setting the final state and
// assembling the block.
func (beacon *Beacon) FinalizeAndAssemble(chain consensus.ChainHeaderReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	beacon.lock.RLock()
	defer beacon.lock.RUnlock()

	// FinalizeAndAssemble is different with Prepare, it can be used in both block
	// generation and verification. So determine the consensus rules by header type.
	if !beacon.IsPostMergeHeader(header) {
		return beacon.ethone.FinalizeAndAssemble(chain, header, state, txs, uncles, receipts)
	}
	// Finalize and assemble the block
	beacon.Finalize(chain, header, state, txs, uncles)
	return types.NewBlock(header, txs, uncles, receipts, trie.NewStackTrie(nil)), nil
}

// Seal generates a new sealing request for the given input block and pushes
// the result into the given channel.
//
// Note, the method returns immediately and will send the result async. More
// than one result may also be returned depending on the consensus algorithm.
func (beacon *Beacon) Seal(chain consensus.ChainHeaderReader, block *types.Block, results chan<- *types.Block, stop <-chan struct{}) error {
	beacon.lock.RLock()
	defer beacon.lock.RUnlock()

	if !beacon.IsPostMergeHeader(block.Header()) {
		return beacon.ethone.Seal(chain, block, results, stop)
	}
	// The seal verification is done by the external consensus engine,
	// return directly without pushing any block back. In another word
	// beacon won't return any result by `results` channel which may
	// blocks the receiver logic forever.
	return nil
}

// SealHash returns the hash of a block prior to it being sealed.
func (beacon *Beacon) SealHash(header *types.Header) common.Hash {
	return beacon.ethone.SealHash(header)
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func (beacon *Beacon) CalcDifficulty(chain consensus.ChainHeaderReader, time uint64, parent *types.Header) *big.Int {
	beacon.lock.RLock()
	defer beacon.lock.RUnlock()

	// Transition isn't triggered yet, use the legacy rules for calculation
	if !beacon.transitioned {
		return beacon.ethone.CalcDifficulty(chain, time, parent)
	}
	return beaconDifficulty
}

// APIs implements consensus.Engine, returning the user facing RPC APIs.
func (beacon *Beacon) APIs(chain consensus.ChainHeaderReader) []rpc.API {
	return beacon.ethone.APIs(chain)
}

// Close shutdowns the consensus engine
func (beacon *Beacon) Close() error {
	return beacon.ethone.Close()
}

// IsPostMergeHeader reports the header belongs to the PoS-stage with some special fields.
// This function is not suitable for a part of APIs like Prepare or CalcDifficulty because
// the header difficulty is not set yet.
func (beacon *Beacon) IsPostMergeHeader(header *types.Header) bool {
	// These fields can be used to filter out ethash block
	if header.Difficulty.Cmp(beaconDifficulty) != 0 {
		return false
	}
	if header.MixDigest != (common.Hash{}) {
		return false
	}
	if header.Nonce != beaconNonce {
		return false
	}
	// Extra field can be used to filter out clique block
	if len(header.Extra) != 0 {
		return false
	}
	return true
}

// MarkTransitioned sets the transitioned flag.
func (beacon *Beacon) MarkTransitioned() {
	beacon.lock.Lock()
	defer beacon.lock.Unlock()

	beacon.transitioned = true
}

// InnerEngine returns the embedded eth1 consensus engine.
func (beacon *Beacon) InnerEngine() consensus.Engine {
	return beacon.ethone
}

// SetThreads updates the mining threads. Delegate the call
// to the eth1 engine if it's threaded.
func (beacon *Beacon) SetThreads(threads int) {
	type threaded interface {
		SetThreads(threads int)
	}
	if th, ok := beacon.ethone.(threaded); ok {
		th.SetThreads(threads)
	}
}
