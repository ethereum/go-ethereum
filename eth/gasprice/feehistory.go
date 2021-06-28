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

package gasprice

import (
	"context"
	"errors"
	"math/big"
	"sort"
	"sync/atomic"

	"github.com/ethereum/go-ethereum/consensus/misc"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	errInvalidPercentiles = errors.New("Invalid reward percentiles")
	errRequestBeyondHead  = errors.New("Request beyond head block")
)

const maxBlockCount = 1024 // number of blocks retrievable with a single query

// blockFees represents a single block for processing
type blockFees struct {
	// set by the caller
	blockNumber rpc.BlockNumber
	header      *types.Header
	block       *types.Block // only set if reward percentiles are requested
	receipts    types.Receipts
	// filled by processBlock
	reward               []*big.Int
	baseFee, nextBaseFee *big.Int
	gasUsedRatio         float64
	err                  error
}

// txGasAndReward is sorted in ascending order based on reward
type (
	txGasAndReward struct {
		gasUsed uint64
		reward  *big.Int
	}
	sortGasAndReward []txGasAndReward
)

func (s sortGasAndReward) Len() int { return len(s) }
func (s sortGasAndReward) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
func (s sortGasAndReward) Less(i, j int) bool {
	return s[i].reward.Cmp(s[j].reward) < 0
}

// processBlock takes a blockFees structure with the blockNumber, the header and optionally
// the block field filled in, retrieves the block from the backend if not present yet and
// fills in the rest of the fields.
func (oracle *Oracle) processBlock(bf *blockFees, percentiles []float64) {
	chainconfig := oracle.backend.ChainConfig()
	if bf.baseFee = bf.header.BaseFee; bf.baseFee == nil {
		bf.baseFee = new(big.Int)
	}
	if chainconfig.IsLondon(big.NewInt(int64(bf.blockNumber + 1))) {
		bf.nextBaseFee = misc.CalcBaseFee(chainconfig, bf.header)
	} else {
		bf.nextBaseFee = new(big.Int)
	}
	bf.gasUsedRatio = float64(bf.header.GasUsed) / float64(bf.header.GasLimit)
	if len(percentiles) == 0 {
		// rewards were not requested, return null
		return
	}
	if bf.block == nil || (bf.receipts == nil && len(bf.block.Transactions()) != 0) {
		log.Error("Block or receipts are missing while reward percentiles are requested")
		return
	}

	bf.reward = make([]*big.Int, len(percentiles))
	if len(bf.block.Transactions()) == 0 {
		// return an all zero row if there are no transactions to gather data from
		for i := range bf.reward {
			bf.reward[i] = new(big.Int)
		}
		return
	}

	sorter := make(sortGasAndReward, len(bf.block.Transactions()))
	for i, tx := range bf.block.Transactions() {
		reward, _ := tx.EffectiveGasTip(bf.block.BaseFee())
		sorter[i] = txGasAndReward{gasUsed: bf.receipts[i].GasUsed, reward: reward}
	}
	sort.Sort(sorter)

	var txIndex int
	sumGasUsed := sorter[0].gasUsed

	for i, p := range percentiles {
		thresholdGasUsed := uint64(float64(bf.block.GasUsed()) * p / 100)
		for sumGasUsed < thresholdGasUsed && txIndex < len(bf.block.Transactions())-1 {
			txIndex++
			sumGasUsed += sorter[txIndex].gasUsed
		}
		bf.reward[i] = sorter[txIndex].reward
	}
}

// resolveBlockRange resolves the specified block range to absolute block numbers while also
// enforcing backend specific limitations. The pending block and corresponding receipts are
// also returned if requested and available.
// Note: an error is only returned if retrieving the head header has failed. If there are no
// retrievable blocks in the specified range then zero block count is returned with no error.
func (oracle *Oracle) resolveBlockRange(ctx context.Context, lastBlockNumber rpc.BlockNumber, blockCount, maxHistory int) (*types.Block, types.Receipts, rpc.BlockNumber, int, error) {
	var (
		headBlockNumber rpc.BlockNumber
		pendingBlock    *types.Block
		pendingReceipts types.Receipts
	)

	// query either pending block or head header and set headBlockNumber
	if lastBlockNumber == rpc.PendingBlockNumber {
		if pendingBlock, pendingReceipts = oracle.backend.PendingBlockAndReceipts(); pendingBlock != nil {
			lastBlockNumber = rpc.BlockNumber(pendingBlock.NumberU64())
			headBlockNumber = lastBlockNumber - 1
		} else {
			// pending block not supported by backend, process until latest block
			lastBlockNumber = rpc.LatestBlockNumber
			blockCount--
			if blockCount == 0 {
				return nil, nil, 0, 0, nil
			}
		}
	}
	if pendingBlock == nil {
		// if pending block is not fetched then we retrieve the head header to get the head block number
		if latestHeader, err := oracle.backend.HeaderByNumber(ctx, rpc.LatestBlockNumber); err == nil {
			headBlockNumber = rpc.BlockNumber(latestHeader.Number.Uint64())
		} else {
			return nil, nil, 0, 0, err
		}
	}
	if lastBlockNumber == rpc.LatestBlockNumber {
		lastBlockNumber = headBlockNumber
	} else if pendingBlock == nil && lastBlockNumber > headBlockNumber {
		return nil, nil, 0, 0, errRequestBeyondHead
	}
	if maxHistory != 0 {
		// limit retrieval to the given number of latest blocks
		if tooOldCount := int64(headBlockNumber) - int64(maxHistory) - int64(lastBlockNumber) + int64(blockCount); tooOldCount > 0 {
			// tooOldCount is the number of requested blocks that are too old to be served
			if int64(blockCount) > tooOldCount {
				blockCount -= int(tooOldCount)
			} else {
				return nil, nil, 0, 0, nil
			}
		}
	}
	// ensure not trying to retrieve before genesis
	if rpc.BlockNumber(blockCount) > lastBlockNumber+1 {
		blockCount = int(lastBlockNumber + 1)
	}
	return pendingBlock, pendingReceipts, lastBlockNumber, blockCount, nil
}

// FeeHistory returns data relevant for fee estimation based on the specified range of blocks.
// The range can be specified either with absolute block numbers or ending with the latest
// or pending block. Backends may or may not support gathering data from the pending block
// or blocks older than a certain age (specified in maxHistory). The first block of the
// actually processed range is returned to avoid ambiguity when parts of the requested range
// are not available or when the head has changed during processing this request.
// Three arrays are returned based on the processed blocks:
// - reward: the requested percentiles of effective priority fees per gas of transactions in each
//   block, sorted in ascending order and weighted by gas used.
// - baseFee: base fee per gas in the given block
// - gasUsedRatio: gasUsed/gasLimit in the given block
// Note: baseFee includes the next block after the newest of the returned range, because this
// value can be derived from the newest block.
func (oracle *Oracle) FeeHistory(ctx context.Context, blockCount int, lastBlockNumber rpc.BlockNumber, rewardPercentiles []float64) (firstBlockNumber rpc.BlockNumber, reward [][]*big.Int, baseFee []*big.Int, gasUsedRatio []float64, err error) {
	if blockCount < 1 {
		// returning with no data and no error means there are no retrievable blocks
		return
	}
	if blockCount > maxBlockCount {
		blockCount = maxBlockCount
	}
	for i, p := range rewardPercentiles {
		if p < 0 || p > 100 || (i > 0 && p < rewardPercentiles[i-1]) {
			return 0, nil, nil, nil, errInvalidPercentiles
		}
	}

	processBlocks := len(rewardPercentiles) != 0
	// limit retrieval to maxHistory if set
	var maxHistory int
	if processBlocks {
		maxHistory = oracle.maxBlockHistory
	} else {
		maxHistory = oracle.maxHeaderHistory
	}

	var (
		pendingBlock    *types.Block
		pendingReceipts types.Receipts
	)
	if pendingBlock, pendingReceipts, lastBlockNumber, blockCount, err = oracle.resolveBlockRange(ctx, lastBlockNumber, blockCount, maxHistory); err != nil || blockCount == 0 {
		return
	}
	firstBlockNumber = lastBlockNumber + 1 - rpc.BlockNumber(blockCount)

	processNext := int64(firstBlockNumber)
	resultCh := make(chan *blockFees, blockCount)
	threadCount := 4
	if blockCount < threadCount {
		threadCount = blockCount
	}
	for i := 0; i < threadCount; i++ {
		go func() {
			for {
				blockNumber := rpc.BlockNumber(atomic.AddInt64(&processNext, 1) - 1)
				if blockNumber > lastBlockNumber {
					return
				}

				bf := &blockFees{blockNumber: blockNumber}
				if pendingBlock != nil && blockNumber >= rpc.BlockNumber(pendingBlock.NumberU64()) {
					bf.block, bf.receipts = pendingBlock, pendingReceipts
				} else {
					if processBlocks {
						bf.block, bf.err = oracle.backend.BlockByNumber(ctx, blockNumber)
						if bf.block != nil {
							bf.receipts, bf.err = oracle.backend.GetReceipts(ctx, bf.block.Hash())
						}
					} else {
						bf.header, bf.err = oracle.backend.HeaderByNumber(ctx, blockNumber)
					}
				}
				if bf.block != nil {
					bf.header = bf.block.Header()
				}
				if bf.header != nil {
					oracle.processBlock(bf, rewardPercentiles)
				}
				// send to resultCh even if empty to guarantee that blockCount items are sent in total
				resultCh <- bf
			}
		}()
	}

	reward = make([][]*big.Int, blockCount)
	baseFee = make([]*big.Int, blockCount+1)
	gasUsedRatio = make([]float64, blockCount)
	firstMissing := blockCount

	for ; blockCount > 0; blockCount-- {
		bf := <-resultCh
		if bf.err != nil {
			return 0, nil, nil, nil, bf.err
		}
		i := int(bf.blockNumber - firstBlockNumber)
		if bf.header != nil {
			reward[i], baseFee[i], baseFee[i+1], gasUsedRatio[i] = bf.reward, bf.baseFee, bf.nextBaseFee, bf.gasUsedRatio
		} else {
			// getting no block and no error means we are requesting into the future (might happen because of a reorg)
			if i < firstMissing {
				firstMissing = i
			}
		}
	}
	if firstMissing == 0 {
		return 0, nil, nil, nil, nil
	}
	if processBlocks {
		reward = reward[:firstMissing]
	} else {
		reward = nil
	}
	baseFee, gasUsedRatio = baseFee[:firstMissing+1], gasUsedRatio[:firstMissing]
	return
}
