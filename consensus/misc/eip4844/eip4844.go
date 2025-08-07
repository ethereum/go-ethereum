// Copyright 2023 The go-ethereum Authors
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

package eip4844

import (
	"errors"
	"fmt"
	"math"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/params/forks"
)

var (
	minBlobGasPrice = big.NewInt(params.BlobTxMinBlobGasprice)
)

// VerifyEIP4844Header verifies the presence of the excessBlobGas field and that
// if the current block contains no transactions, the excessBlobGas is updated
// accordingly.
func VerifyEIP4844Header(config *params.Config2, parent, header *types.Header) error {
	if header.Number.Uint64() != parent.Number.Uint64()+1 {
		panic("bad header pair")
	}

	// Verify the header is not malformed
	if header.ExcessBlobGas == nil {
		return errors.New("header is missing excessBlobGas")
	}
	if header.BlobGasUsed == nil {
		return errors.New("header is missing blobGasUsed")
	}

	blobcfg := scheduleAtTime(config, header.Time)
	if blobcfg == nil {
		return fmt.Errorf("blob schedule is undefined at time %d", header.Time)
	}

	// Verify that the blob gas used remains within reasonable limits.
	maxBlobGas := uint64(blobcfg.Max) * params.BlobTxBlobGasPerBlob
	if *header.BlobGasUsed > maxBlobGas {
		return fmt.Errorf("blob gas used %d exceeds maximum allowance %d", *header.BlobGasUsed, maxBlobGas)
	}
	if *header.BlobGasUsed%params.BlobTxBlobGasPerBlob != 0 {
		return fmt.Errorf("blob gas used %d not a multiple of blob gas per blob %d", header.BlobGasUsed, params.BlobTxBlobGasPerBlob)
	}
	// Verify the excessBlobGas is correct based on the parent header
	expectedExcessBlobGas := CalcExcessBlobGas(config, parent, header.Time)
	if *header.ExcessBlobGas != expectedExcessBlobGas {
		return fmt.Errorf("invalid excessBlobGas: have %d, want %d", *header.ExcessBlobGas, expectedExcessBlobGas)
	}
	return nil
}

// CalcExcessBlobGas calculates the excess blob gas after applying the set of
// blobs on top of the excess blob gas.
func CalcExcessBlobGas(config *params.Config2, parent *types.Header, headTimestamp uint64) uint64 {
	var (
		parentExcessBlobGas uint64
		parentBlobGasUsed   uint64
	)
	if parent.ExcessBlobGas != nil {
		parentExcessBlobGas = *parent.ExcessBlobGas
		parentBlobGasUsed = *parent.BlobGasUsed
	}
	var (
		excessBlobGas = parentExcessBlobGas + parentBlobGasUsed
		target        = targetBlobsPerBlock(config, headTimestamp)
		targetGas     = uint64(target) * params.BlobTxBlobGasPerBlob
	)
	if excessBlobGas < targetGas {
		return 0
	}
	if !config.Active(forks.Osaka, parent.Number.Uint64()+1, headTimestamp) {
		// Pre-Osaka, we use the formula defined by EIP-4844.
		return excessBlobGas - targetGas
	}

	// EIP-7918 (post-Osaka) introduces a different formula for computing excess.
	var (
		baseCost     = big.NewInt(params.BlobBaseCost)
		reservePrice = baseCost.Mul(baseCost, parent.BaseFee)
		blobPrice    = calcBlobPrice(config, parent)
	)
	if reservePrice.Cmp(blobPrice) > 0 {
		max := MaxBlobsPerBlock(config, headTimestamp)
		scaledExcess := parentBlobGasUsed * uint64(max-target) / uint64(max)
		return parentExcessBlobGas + scaledExcess
	}
	return excessBlobGas - targetGas
}

// CalcBlobFee calculates the blobfee from the header's excess blob gas field.
func CalcBlobFee(config *params.Config2, header *types.Header) *big.Int {
	blobcfg := scheduleAtTime(config, header.Time)
	if blobcfg == nil {
		return new(big.Int)
	}
	frac := blobcfg.UpdateFraction
	return fakeExponential(minBlobGasPrice, new(big.Int).SetUint64(*header.ExcessBlobGas), new(big.Int).SetUint64(frac))
}

// MaxBlobsPerBlock returns the max blobs per block for a block at the given timestamp.
func MaxBlobsPerBlock(cfg *params.Config2, time uint64) int {
	blobcfg := scheduleAtTime(cfg, time)
	if blobcfg == nil {
		return 0
	}
	return blobcfg.Max
}

// MaxBlobsPerBlock returns the maximum blob gas that can be spent in a block at the given timestamp.
func MaxBlobGasPerBlock(cfg *params.Config2, time uint64) uint64 {
	return uint64(MaxBlobsPerBlock(cfg, time)) * params.BlobTxBlobGasPerBlob
}

// LatestMaxBlobsPerBlock returns the latest max blobs per block defined by the
// configuration, regardless of the currently active fork.
func LatestMaxBlobsPerBlock(cfg *params.Config2) int {
	blobcfg := scheduleAtTime(cfg, math.MaxUint64)
	return blobcfg.Max
}

// targetBlobsPerBlock returns the target number of blobs in a block at the given timestamp.
func targetBlobsPerBlock(cfg *params.Config2, time uint64) int {
	return scheduleAtTime(cfg, time).Target
}

// scheduleAtTime resolves the blob schedule at the given timestamp.
func scheduleAtTime(cfg *params.Config2, time uint64) *params.BlobConfig {
	schedule := params.BlobSchedule.Get(cfg)
	if schedule == nil {
		return nil
	}

	// Find the latest fork defined by the schedule.
	forkList := make([]forks.Fork, 0, len(schedule))
	for f := range schedule {
		act, ok := cfg.Activation(f)
		if ok && act <= time {
			forkList = append(forkList, f)
		}
	}
	forkList = forks.DependencyOrder(forkList)

	// Return the blob config of the last available fork.
	if len(forkList) == 0 {
		return nil
	}
	blobcfg := schedule[forkList[len(forkList)-1]]
	return &blobcfg
}

// fakeExponential approximates factor * e ** (numerator / denominator) using
// Taylor expansion.
func fakeExponential(factor, numerator, denominator *big.Int) *big.Int {
	var (
		output = new(big.Int)
		accum  = new(big.Int).Mul(factor, denominator)
	)
	for i := 1; accum.Sign() > 0; i++ {
		output.Add(output, accum)

		accum.Mul(accum, numerator)
		accum.Div(accum, denominator)
		accum.Div(accum, big.NewInt(int64(i)))
	}
	return output.Div(output, denominator)
}

// calcBlobPrice calculates the blob price for a block.
func calcBlobPrice(config *params.Config2, header *types.Header) *big.Int {
	blobBaseFee := CalcBlobFee(config, header)
	return new(big.Int).Mul(blobBaseFee, big.NewInt(params.BlobTxBlobGasPerBlob))
}
