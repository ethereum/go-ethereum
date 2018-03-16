// Copyright 2017 The go-ethereum Authors
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

package ethash

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"runtime"
	"time"

	"github.com/TeamEGEM/go-egem/common"
	"github.com/TeamEGEM/go-egem/common/math"
	"github.com/TeamEGEM/go-egem/consensus"
	"github.com/TeamEGEM/go-egem/consensus/misc"
	"github.com/TeamEGEM/go-egem/core/state"
	"github.com/TeamEGEM/go-egem/core/types"
	"github.com/TeamEGEM/go-egem/params"
	set "gopkg.in/fatih/set.v0"
)

// Ethash proof-of-work protocol constants.
var (
	maxUncles                       = 2                 // Maximum number of uncles allowed in a single block
	allowedFutureBlockTime          = 15 * time.Second  // Max time from current time allowed for blocks, before they're considered future blocks
	FrontierBlockReward    					*big.Int = big.NewInt(5e+18) // Not used will be removed in furture EGEM update.
	ByzantiumBlockReward   					*big.Int = big.NewInt(3e+18) // Not used will be removed in furture EGEM update.
)

//  EGEM Variables
var (
	egem0BlockReward                *big.Int = big.NewInt(8e+18) //  8 EGEM Block reward in wei for successfully mining a block.
	egem1BlockReward                *big.Int = big.NewInt(4e+18) //  4 EGEM Block reward in wei for successfully mining a block.
	egem2BlockReward                *big.Int = big.NewInt(2e+18) //  2 EGEM Block reward in wei for successfully mining a block.
	egem0DevReward                  *big.Int = big.NewInt(1e+18) // Era0 Dev reward 1 EGEM per block.
	egem1DevReward                  *big.Int = big.NewInt(500000000000000000) // Era1 Dev reward 0.5 EGEM per block.
	egem2DevReward                  *big.Int = big.NewInt(100000000000000000) // Era2 Dev reward 0.1 EGEM per block.
	egemRewardSwitchBlockEra0       *big.Int = big.NewInt(5) //5K Block era transition
	egemRewardSwitchBlockEra1       *big.Int = big.NewInt(10) // 10M block era transition
	egemRewardSwitchBlockEra2       *big.Int = big.NewInt(20) // 20M block era transtiton
	devFund 												= common.HexToAddress("0xc393659c2918a64cdfb44d463de9c747aa4ce3f7") //r
)

// Various error messages to mark blocks invalid. These should be private to
// prevent engine specific errors from being referenced in the remainder of the
// codebase, inherently breaking if the engine is swapped out. Please put common
// error types into the consensus package.
var (
	errLargeBlockTime    = errors.New("timestamp too big")
	errZeroBlockTime     = errors.New("timestamp equals parent's")
	errTooManyUncles     = errors.New("too many uncles")
	errDuplicateUncle    = errors.New("duplicate uncle")
	errUncleIsAncestor   = errors.New("uncle is ancestor")
	errDanglingUncle     = errors.New("uncle's parent is not ancestor")
	errInvalidDifficulty = errors.New("non-positive difficulty")
	errInvalidMixDigest  = errors.New("invalid mix digest")
	errInvalidPoW        = errors.New("invalid proof-of-work")
)

// Author implements consensus.Engine, returning the header's coinbase as the
// proof-of-work verified author of the block.
func (ethash *Ethash) Author(header *types.Header) (common.Address, error) {
	return header.Coinbase, nil
}

// VerifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
func (ethash *Ethash) VerifyHeader(chain consensus.ChainReader, header *types.Header, seal bool) error {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake {
		return nil
	}
	// Short circuit if the header is known, or it's parent not
	number := header.Number.Uint64()
	if chain.GetHeader(header.Hash(), number) != nil {
		return nil
	}
	parent := chain.GetHeader(header.ParentHash, number-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	// Sanity checks passed, do a proper verification
	return ethash.verifyHeader(chain, header, parent, false, seal)
}

// VerifyHeaders is similar to VerifyHeader, but verifies a batch of headers
// concurrently. The method returns a quit channel to abort the operations and
// a results channel to retrieve the async verifications.
func (ethash *Ethash) VerifyHeaders(chain consensus.ChainReader, headers []*types.Header, seals []bool) (chan<- struct{}, <-chan error) {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake || len(headers) == 0 {
		abort, results := make(chan struct{}), make(chan error, len(headers))
		for i := 0; i < len(headers); i++ {
			results <- nil
		}
		return abort, results
	}

	// Spawn as many workers as allowed threads
	workers := runtime.GOMAXPROCS(0)
	if len(headers) < workers {
		workers = len(headers)
	}

	// Create a task channel and spawn the verifiers
	var (
		inputs = make(chan int)
		done   = make(chan int, workers)
		errors = make([]error, len(headers))
		abort  = make(chan struct{})
	)
	for i := 0; i < workers; i++ {
		go func() {
			for index := range inputs {
				errors[index] = ethash.verifyHeaderWorker(chain, headers, seals, index)
				done <- index
			}
		}()
	}

	errorsOut := make(chan error, len(headers))
	go func() {
		defer close(inputs)
		var (
			in, out = 0, 0
			checked = make([]bool, len(headers))
			inputs  = inputs
		)
		for {
			select {
			case inputs <- in:
				if in++; in == len(headers) {
					// Reached end of headers. Stop sending to workers.
					inputs = nil
				}
			case index := <-done:
				for checked[index] = true; checked[out]; out++ {
					errorsOut <- errors[out]
					if out == len(headers)-1 {
						return
					}
				}
			case <-abort:
				return
			}
		}
	}()
	return abort, errorsOut
}

func (ethash *Ethash) verifyHeaderWorker(chain consensus.ChainReader, headers []*types.Header, seals []bool, index int) error {
	var parent *types.Header
	if index == 0 {
		parent = chain.GetHeader(headers[0].ParentHash, headers[0].Number.Uint64()-1)
	} else if headers[index-1].Hash() == headers[index].ParentHash {
		parent = headers[index-1]
	}
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	if chain.GetHeader(headers[index].Hash(), headers[index].Number.Uint64()) != nil {
		return nil // known block
	}
	return ethash.verifyHeader(chain, headers[index], parent, false, seals[index])
}

// VerifyUncles verifies that the given block's uncles conform to the consensus
// rules of the stock Ethereum ethash engine.
func (ethash *Ethash) VerifyUncles(chain consensus.ChainReader, block *types.Block) error {
	// If we're running a full engine faking, accept any input as valid
	if ethash.config.PowMode == ModeFullFake {
		return nil
	}
	// Verify that there are at most 2 uncles included in this block
	if len(block.Uncles()) > maxUncles {
		return errTooManyUncles
	}
	// Gather the set of past uncles and ancestors
	uncles, ancestors := set.New(), make(map[common.Hash]*types.Header)

	number, parent := block.NumberU64()-1, block.ParentHash()
	for i := 0; i < 7; i++ {
		ancestor := chain.GetBlock(parent, number)
		if ancestor == nil {
			break
		}
		ancestors[ancestor.Hash()] = ancestor.Header()
		for _, uncle := range ancestor.Uncles() {
			uncles.Add(uncle.Hash())
		}
		parent, number = ancestor.ParentHash(), number-1
	}
	ancestors[block.Hash()] = block.Header()
	uncles.Add(block.Hash())

	// Verify each of the uncles that it's recent, but not an ancestor
	for _, uncle := range block.Uncles() {
		// Make sure every uncle is rewarded only once
		hash := uncle.Hash()
		if uncles.Has(hash) {
			return errDuplicateUncle
		}
		uncles.Add(hash)

		// Make sure the uncle has a valid ancestry
		if ancestors[hash] != nil {
			return errUncleIsAncestor
		}
		if ancestors[uncle.ParentHash] == nil || uncle.ParentHash == block.ParentHash() {
			return errDanglingUncle
		}
		if err := ethash.verifyHeader(chain, uncle, ancestors[uncle.ParentHash], true, true); err != nil {
			return err
		}
	}
	return nil
}

// verifyHeader checks whether a header conforms to the consensus rules of the
// stock Ethereum ethash engine.
// See YP section 4.3.4. "Block Header Validity"
func (ethash *Ethash) verifyHeader(chain consensus.ChainReader, header, parent *types.Header, uncle bool, seal bool) error {
	// Ensure that the header's extra-data section is of a reasonable size
	if uint64(len(header.Extra)) > params.MaximumExtraDataSize {
		return fmt.Errorf("extra-data too long: %d > %d", len(header.Extra), params.MaximumExtraDataSize)
	}
	// Verify the header's timestamp
	if uncle {
		if header.Time.Cmp(math.MaxBig256) > 0 {
			return errLargeBlockTime
		}
	} else {
		if header.Time.Cmp(big.NewInt(time.Now().Add(allowedFutureBlockTime).Unix())) > 0 {
			return consensus.ErrFutureBlock
		}
	}
	if header.Time.Cmp(parent.Time) <= 0 {
		return errZeroBlockTime
	}
	// Verify the block's difficulty based in it's timestamp and parent's difficulty
	expected := ethash.CalcDifficulty(chain, header.Time.Uint64(), parent)

	if expected.Cmp(header.Difficulty) != 0 {
		return fmt.Errorf("invalid difficulty: have %v, want %v", header.Difficulty, expected)
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
	// Verify the engine specific seal securing the block
	if seal {
		if err := ethash.VerifySeal(chain, header); err != nil {
			return err
		}
	}
	// If all checks passed, validate any special fields for hard forks
	if err := misc.VerifyDAOHeaderExtraData(chain.Config(), header); err != nil {
		return err
	}
	if err := misc.VerifyForkHashes(chain.Config(), header, uncle); err != nil {
		return err
	}
	return nil
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func (ethash *Ethash) CalcDifficulty(chain consensus.ChainReader, time uint64, parent *types.Header) *big.Int {
	return CalcDifficulty(chain.Config(), time, parent)
}

// CalcDifficulty is the difficulty adjustment algorithm. It returns
// the difficulty that a new block should have when created at time
// given the parent block's time and difficulty.
func CalcDifficulty(config *params.ChainConfig, time uint64, parent *types.Header) *big.Int {
	next := new(big.Int).Add(parent.Number, big1)
	switch {
	case config.IsByzantium(next):
		return calcDifficultyEGEM(time, parent)
	case config.IsHomestead(next):
		return calcDifficultyEGEM(time, parent)
	default:
		return calcDifficultyEGEM(time, parent)
	}
}

// Some weird constants to avoid constant memory allocs for them.
var (
	expDiffPeriod = big.NewInt(100000)
	big1          = big.NewInt(1)
	big2          = big.NewInt(2)
	big9          = big.NewInt(9)
	big10         = big.NewInt(10)
	big15					= big.NewInt(15)
	big20         = big.NewInt(20)
	big50         = big.NewInt(50)
	bigMinus99    = big.NewInt(-99)
	big2999999    = big.NewInt(2999999)
)

// EGEM Era Difficulty Algo

func calcDifficultyEGEM(time uint64, parent *types.Header) *big.Int {
	diff := new(big.Int)
	adjust0 := new(big.Int).Div(parent.Difficulty, big10)
	adjust1 := new(big.Int).Div(parent.Difficulty, big15)
	adjust2 := new(big.Int).Div(parent.Difficulty, big20)
	adjust3 := new(big.Int).Div(parent.Difficulty, big50)

	bigTime := new(big.Int)
	bigParentTime := new(big.Int)

	bigTime.SetUint64(time)
	bigParentTime.Set(parent.Time)

	if (parent.Number.Cmp(egemRewardSwitchBlockEra2) == 1) {
		if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimit) < 0 {
			diff.Add(parent.Difficulty, adjust0)
		} else {
			diff.Sub(parent.Difficulty, adjust0)
		}
		if diff.Cmp(params.MinimumDifficulty) < 0 {
			diff.Set(params.MinimumDifficulty)
		}
		return diff
	} else if (parent.Number.Cmp(egemRewardSwitchBlockEra1) == 1) {
		if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimit) < 0 {
			diff.Add(parent.Difficulty, adjust1)
		} else {
			diff.Sub(parent.Difficulty, adjust1)
		}
		if diff.Cmp(params.MinimumDifficulty) < 0 {
			diff.Set(params.MinimumDifficulty)
		}
		return diff
	} else if (parent.Number.Cmp(egemRewardSwitchBlockEra0) == 1) {
			if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimit) < 0 {
				diff.Add(parent.Difficulty, adjust2)
			} else {
				diff.Sub(parent.Difficulty, adjust2)
			}
			if diff.Cmp(params.MinimumDifficulty) < 0 {
				diff.Set(params.MinimumDifficulty)
			}
		return diff
	} else {
		if bigTime.Sub(bigTime, bigParentTime).Cmp(params.DurationLimit) < 0 {
			diff.Add(parent.Difficulty, adjust3)
		} else {
			diff.Sub(parent.Difficulty, adjust3)
		}
		if diff.Cmp(params.MinimumDifficulty) < 0 {
			diff.Set(params.MinimumDifficulty)
		}
		return diff
	}
}

// VerifySeal implements consensus.Engine, checking whether the given block satisfies
// the PoW difficulty requirements.
func (ethash *Ethash) VerifySeal(chain consensus.ChainReader, header *types.Header) error {
	// If we're running a fake PoW, accept any seal as valid
	if ethash.config.PowMode == ModeFake || ethash.config.PowMode == ModeFullFake {
		time.Sleep(ethash.fakeDelay)
		if ethash.fakeFail == header.Number.Uint64() {
			return errInvalidPoW
		}
		return nil
	}
	// If we're running a shared PoW, delegate verification to it
	if ethash.shared != nil {
		return ethash.shared.VerifySeal(chain, header)
	}
	// Ensure that we have a valid difficulty for the block
	if header.Difficulty.Sign() <= 0 {
		return errInvalidDifficulty
	}
	// Recompute the digest and PoW value and verify against the header
	number := header.Number.Uint64()

	cache := ethash.cache(number)
	size := datasetSize(number)
	if ethash.config.PowMode == ModeTest {
		size = 32 * 1024
	}
	digest, result := hashimotoLight(size, cache.cache, header.HashNoNonce().Bytes(), header.Nonce.Uint64())
	// Caches are unmapped in a finalizer. Ensure that the cache stays live
	// until after the call to hashimotoLight so it's not unmapped while being used.
	runtime.KeepAlive(cache)

	if !bytes.Equal(header.MixDigest[:], digest) {
		return errInvalidMixDigest
	}
	target := new(big.Int).Div(maxUint256, header.Difficulty)
	if new(big.Int).SetBytes(result).Cmp(target) > 0 {
		return errInvalidPoW
	}
	return nil
}

// Prepare implements consensus.Engine, initializing the difficulty field of a
// header to conform to the ethash protocol. The changes are done inline.
func (ethash *Ethash) Prepare(chain consensus.ChainReader, header *types.Header) error {
	parent := chain.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	header.Difficulty = ethash.CalcDifficulty(chain, header.Time.Uint64(), parent)
	return nil
}

// Finalize implements consensus.Engine, accumulating the block and uncle rewards,
// setting the final state and assembling the block.
func (ethash *Ethash) Finalize(chain consensus.ChainReader, header *types.Header, state *state.StateDB, txs []*types.Transaction, uncles []*types.Header, receipts []*types.Receipt) (*types.Block, error) {
	// Accumulate any block and uncle rewards and commit the final state root
	accumulateRewards(chain.Config(), state, header, uncles)
	header.Root = state.IntermediateRoot(chain.Config().IsEIP158(header.Number))

	// Header seems complete, assemble into a block and return
	return types.NewBlock(header, txs, uncles, receipts), nil
}

// Some weird constants to avoid constant memory allocs for them.
var (
	big8  = big.NewInt(8)
	big32 = big.NewInt(32)
)

// AccumulateRewards credits the coinbase of the given block with the mining
// reward. The total reward consists of the static block reward and rewards for
// included uncles. The coinbase of each uncle block is also rewarded.
func accumulateRewards(config *params.ChainConfig, state *state.StateDB, header *types.Header, uncles []*types.Header) {

	// Select the correct block reward based on chain progression
	block0Reward := egem0BlockReward
	block1Reward := egem1BlockReward
	block2Reward := egem2BlockReward
	d0Reward := egem0DevReward
	d1Reward := egem1DevReward
	d2Reward := egem2DevReward

	// Accumulate the rewards for the miner and any included uncles

	// if block.Number > 5000 dev rewards kick in 2/0.1.
	if (header.Number.Cmp(egemRewardSwitchBlockEra2) == 1) {
			reward := new(big.Int).Set(block2Reward)
			r := new(big.Int)
			for _, uncle := range uncles {
					r.Add(uncle.Number, big8)
					r.Sub(r, header.Number)
					r.Mul(r, reward)
					r.Div(r, big8)

					r.Div(reward, big32)
					reward.Add(reward, r)
			}
		state.AddBalance(header.Coinbase, reward)

		//dev rewards
		state.AddBalance(devFund, d2Reward) //ridz

	//Era2 Reward scheme Block and dev reward halving 4/0.5.
	} else if (header.Number.Cmp(egemRewardSwitchBlockEra1) == 1) {
		reward := new(big.Int).Set(block1Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
					r.Add(uncle.Number, big8)
					r.Sub(r, header.Number)
					r.Mul(r, reward)
					r.Div(r, big8)

					r.Div(reward, big32)
					reward.Add(reward, r)
			}
		state.AddBalance(header.Coinbase, reward)

		//dev rewards
		state.AddBalance(devFund, d1Reward) //ridz

  //Final form and final halving of block and dev rewards 8/1.
	} else if (header.Number.Cmp(egemRewardSwitchBlockEra0) == 1) {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
					r.Add(uncle.Number, big8)
					r.Sub(r, header.Number)
					r.Mul(r, reward)
					r.Div(r, big8)

					r.Div(reward, big32)
					reward.Add(reward, r)
			}
		state.AddBalance(header.Coinbase, reward)

		//dev rewards
		state.AddBalance(devFund, d0Reward) //ridz

	// Fair Launch upto block 5000 then dev fee system kicks in.
	} else  {
		reward := new(big.Int).Set(block0Reward)
		r := new(big.Int)
		for _, uncle := range uncles {
					r.Add(uncle.Number, big8)
					r.Sub(r, header.Number)
					r.Mul(r, reward)
					r.Div(r, big8)

					r.Div(reward, big32)
					reward.Add(reward, r)
			}

		state.AddBalance(header.Coinbase, reward)
	}

}
