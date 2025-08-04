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
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

var (
	minBlobGasPrice = big.NewInt(params.BlobTxMinBlobGasprice)
)

// VerifyEIP4844Header verifies the presence of the excessBlobGas field and that
// if the current block contains no transactions, the excessBlobGas is updated
// accordingly.
func VerifyEIP4844Header(config *params.ChainConfig, parent, header *types.Header) error {
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
	// Verify that the blob gas used remains within reasonable limits.
	maxBlobGas := MaxBlobGasPerBlock(config, header.Time)
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
func CalcExcessBlobGas(config *params.ChainConfig, parent *types.Header, headTimestamp uint64) uint64 {
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
	if !config.IsOsaka(config.LondonBlock, headTimestamp) {
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
func CalcBlobFee(config *params.ChainConfig, header *types.Header) *big.Int {
	blobConfig := latestBlobConfig(config, header.Time)
	if blobConfig == nil {
		panic("calculating blob fee on unsupported fork")
	}
	return fakeExponential(minBlobGasPrice, new(big.Int).SetUint64(*header.ExcessBlobGas), new(big.Int).SetUint64(blobConfig.UpdateFraction))
}

// MaxBlobsPerBlock returns the max blobs per block for a block at the given timestamp.
func MaxBlobsPerBlock(cfg *params.ChainConfig, time uint64) int {
	blobConfig := latestBlobConfig(cfg, time)
	if blobConfig == nil {
		return 0
	}
	return blobConfig.Max
}

func latestBlobConfig(cfg *params.ChainConfig, time uint64) *params.BlobConfig {
	if cfg.BlobScheduleConfig == nil {
		return nil
	}
	var (
		london = cfg.LondonBlock
		s      = cfg.BlobScheduleConfig
	)
	switch {
	case cfg.IsBPO5(london, time) && s.BPO5 != nil:
		return s.BPO5
	case cfg.IsBPO4(london, time) && s.BPO4 != nil:
		return s.BPO4
	case cfg.IsBPO3(london, time) && s.BPO3 != nil:
		return s.BPO3
	case cfg.IsBPO2(london, time) && s.BPO2 != nil:
		return s.BPO2
	case cfg.IsBPO1(london, time) && s.BPO1 != nil:
		return s.BPO1
	case cfg.IsOsaka(london, time) && s.Osaka != nil:
		return s.Osaka
	case cfg.IsPrague(london, time) && s.Prague != nil:
		return s.Prague
	case cfg.IsCancun(london, time) && s.Cancun != nil:
		return s.Cancun
	default:
		return nil
	}
}

// MaxBlobGasPerBlock returns the maximum blob gas that can be spent in a block at the given timestamp.
func MaxBlobGasPerBlock(cfg *params.ChainConfig, time uint64) uint64 {
	return uint64(MaxBlobsPerBlock(cfg, time)) * params.BlobTxBlobGasPerBlob
}

// LatestMaxBlobsPerBlock returns the latest max blobs per block defined by the
// configuration, regardless of the currently active fork.
func LatestMaxBlobsPerBlock(cfg *params.ChainConfig) int {
	s := cfg.BlobScheduleConfig
	if s == nil {
		return 0
	}
	switch {
	case s.BPO5 != nil:
		return s.BPO5.Max
	case s.BPO4 != nil:
		return s.BPO4.Max
	case s.BPO3 != nil:
		return s.BPO3.Max
	case s.BPO2 != nil:
		return s.BPO2.Max
	case s.BPO1 != nil:
		return s.BPO1.Max
	case s.Osaka != nil:
		return s.Osaka.Max
	case s.Prague != nil:
		return s.Prague.Max
	case s.Cancun != nil:
		return s.Cancun.Max
	default:
		return 0
	}
}

// targetBlobsPerBlock returns the target number of blobs in a block at the given timestamp.
func targetBlobsPerBlock(cfg *params.ChainConfig, time uint64) int {
	blobConfig := latestBlobConfig(cfg, time)
	if blobConfig == nil {
		return 0
	}
	return blobConfig.Target
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
func calcBlobPrice(config *params.ChainConfig, header *types.Header) *big.Int {
	blobBaseFee := CalcBlobFee(config, header)
	return new(big.Int).Mul(blobBaseFee, big.NewInt(params.BlobTxBlobGasPerBlob))
}
