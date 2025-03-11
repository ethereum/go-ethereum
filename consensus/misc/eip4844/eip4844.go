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
	"github.com/ethereum/go-ethereum/params/forks"
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
	excessBlobGas := parentExcessBlobGas + parentBlobGasUsed
	targetGas := uint64(targetBlobsPerBlock(config, headTimestamp)) * params.BlobTxBlobGasPerBlob
	if excessBlobGas < targetGas {
		return 0
	}
	return excessBlobGas - targetGas
}

// CalcBlobFee calculates the blobfee from the header's excess blob gas field.
func CalcBlobFee(config *params.ChainConfig, header *types.Header) *big.Int {
	var frac uint64
	switch config.LatestFork(header.Time) {
	case forks.Osaka:
		frac = config.BlobScheduleConfig.Osaka.UpdateFraction
	case forks.Prague:
		frac = config.BlobScheduleConfig.Prague.UpdateFraction
	case forks.Cancun:
		frac = config.BlobScheduleConfig.Cancun.UpdateFraction
	default:
		panic("calculating blob fee on unsupported fork")
	}
	return fakeExponential(minBlobGasPrice, new(big.Int).SetUint64(*header.ExcessBlobGas), new(big.Int).SetUint64(frac))
}

// MaxBlobsPerBlock returns the max blobs per block for a block at the given timestamp.
func MaxBlobsPerBlock(cfg *params.ChainConfig, time uint64) int {
	if cfg.BlobScheduleConfig == nil {
		return 0
	}
	var (
		london = cfg.LondonBlock
		s      = cfg.BlobScheduleConfig
	)
	switch {
	case cfg.IsOsaka(london, time) && s.Osaka != nil:
		return s.Osaka.Max
	case cfg.IsPrague(london, time) && s.Prague != nil:
		return s.Prague.Max
	case cfg.IsCancun(london, time) && s.Cancun != nil:
		return s.Cancun.Max
	default:
		return 0
	}
}

// MaxBlobsPerBlock returns the maximum blob gas that can be spent in a block at the given timestamp.
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
	if cfg.BlobScheduleConfig == nil {
		return 0
	}
	var (
		london = cfg.LondonBlock
		s      = cfg.BlobScheduleConfig
	)
	switch {
	case cfg.IsOsaka(london, time) && s.Osaka != nil:
		return s.Osaka.Target
	case cfg.IsPrague(london, time) && s.Prague != nil:
		return s.Prague.Target
	case cfg.IsCancun(london, time) && s.Cancun != nil:
		return s.Cancun.Target
	default:
		return 0
	}
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
