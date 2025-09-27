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
)

var (
	minBlobGasPrice = big.NewInt(params.BlobTxMinBlobGasprice)
)

// BlobConfig contains the parameters for blob-related formulas.
// These can be adjusted in a fork.
type BlobConfig struct {
	Target         int
	Max            int
	UpdateFraction uint64
}

func (bc *BlobConfig) maxBlobGas() uint64 {
	return uint64(bc.Max) * params.BlobTxBlobGasPerBlob
}

// blobBaseFee computes the blob fee.
func (bc *BlobConfig) blobBaseFee(excessBlobGas uint64) *big.Int {
	return fakeExponential(minBlobGasPrice, new(big.Int).SetUint64(excessBlobGas), new(big.Int).SetUint64(bc.UpdateFraction))
}

// blobPrice returns the price of one blob in Wei.
func (bc *BlobConfig) blobPrice(excessBlobGas uint64) *big.Int {
	f := bc.blobBaseFee(excessBlobGas)
	return new(big.Int).Mul(f, big.NewInt(params.BlobTxBlobGasPerBlob))
}

func latestBlobConfig(cfg *params.ChainConfig, time uint64) *BlobConfig {
	if cfg.BlobScheduleConfig == nil {
		return nil
	}
	var (
		london = cfg.LondonBlock
		s      = cfg.BlobScheduleConfig
		bc     *params.BlobConfig
	)
	switch {
	case cfg.IsBPO5(london, time) && s.BPO5 != nil:
		bc = s.BPO5
	case cfg.IsBPO4(london, time) && s.BPO4 != nil:
		bc = s.BPO4
	case cfg.IsBPO3(london, time) && s.BPO3 != nil:
		bc = s.BPO3
	case cfg.IsBPO2(london, time) && s.BPO2 != nil:
		bc = s.BPO2
	case cfg.IsBPO1(london, time) && s.BPO1 != nil:
		bc = s.BPO1
	case cfg.IsOsaka(london, time) && s.Osaka != nil:
		bc = s.Osaka
	case cfg.IsPrague(london, time) && s.Prague != nil:
		bc = s.Prague
	case cfg.IsCancun(london, time) && s.Cancun != nil:
		bc = s.Cancun
	default:
		return nil
	}

	return &BlobConfig{
		Target:         bc.Target,
		Max:            bc.Max,
		UpdateFraction: bc.UpdateFraction,
	}
}

// VerifyEIP4844Header verifies the presence of the excessBlobGas field and that
// if the current block contains no transactions, the excessBlobGas is updated
// accordingly.
func VerifyEIP4844Header(config *params.ChainConfig, parent, header *types.Header) error {
	if header.Number.Uint64() != parent.Number.Uint64()+1 {
		panic("bad header pair")
	}

	bcfg := latestBlobConfig(config, header.Time)
	if bcfg == nil {
		panic("called before EIP-4844 is active")
	}

	if header.ExcessBlobGas == nil {
		return errors.New("header is missing excessBlobGas")
	}
	if header.BlobGasUsed == nil {
		return errors.New("header is missing blobGasUsed")
	}

	// Verify that the blob gas used remains within reasonable limits.
	if *header.BlobGasUsed > bcfg.maxBlobGas() {
		return fmt.Errorf("blob gas used %d exceeds maximum allowance %d", *header.BlobGasUsed, bcfg.maxBlobGas())
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
	isOsaka := config.IsOsaka(config.LondonBlock, headTimestamp)
	bcfg := latestBlobConfig(config, headTimestamp)
	return calcExcessBlobGas(isOsaka, bcfg, parent)
}

func calcExcessBlobGas(isOsaka bool, bcfg *BlobConfig, parent *types.Header) uint64 {
	var parentExcessBlobGas, parentBlobGasUsed uint64
	if parent.ExcessBlobGas != nil {
		parentExcessBlobGas = *parent.ExcessBlobGas
		parentBlobGasUsed = *parent.BlobGasUsed
	}

	var (
		excessBlobGas = parentExcessBlobGas + parentBlobGasUsed
		targetGas     = uint64(bcfg.Target) * params.BlobTxBlobGasPerBlob
	)
	if excessBlobGas < targetGas {
		return 0
	}

	// EIP-7918 (post-Osaka) introduces a different formula for computing excess,
	// in cases where the price is lower than a 'reserve price'.
	if isOsaka {
		var (
			baseCost     = big.NewInt(params.BlobBaseCost)
			reservePrice = baseCost.Mul(baseCost, parent.BaseFee)
			blobPrice    = bcfg.blobPrice(parentExcessBlobGas)
		)
		if reservePrice.Cmp(blobPrice) > 0 {
			scaledExcess := parentBlobGasUsed * uint64(bcfg.Max-bcfg.Target) / uint64(bcfg.Max)
			return parentExcessBlobGas + scaledExcess
		}
	}

	// Original EIP-4844 formula.
	return excessBlobGas - targetGas
}

// CalcBlobFee calculates the blobfee from the header's excess blob gas field.
func CalcBlobFee(config *params.ChainConfig, header *types.Header) *big.Int {
	blobConfig := latestBlobConfig(config, header.Time)
	if blobConfig == nil {
		panic("calculating blob fee on unsupported fork")
	}
	return blobConfig.blobBaseFee(*header.ExcessBlobGas)
}

// MaxBlobsPerBlock returns the max blobs per block for a block at the given timestamp.
func MaxBlobsPerBlock(cfg *params.ChainConfig, time uint64) int {
	blobConfig := latestBlobConfig(cfg, time)
	if blobConfig == nil {
		return 0
	}
	return blobConfig.Max
}

// MaxBlobGasPerBlock returns the maximum blob gas that can be spent in a block at the given timestamp.
func MaxBlobGasPerBlock(cfg *params.ChainConfig, time uint64) uint64 {
	return uint64(MaxBlobsPerBlock(cfg, time)) * params.BlobTxBlobGasPerBlob
}

// LatestMaxBlobsPerBlock returns the latest max blobs per block defined by the
// configuration, regardless of the currently active fork.
func LatestMaxBlobsPerBlock(cfg *params.ChainConfig) int {
	bcfg := latestBlobConfig(cfg, math.MaxUint64)
	if bcfg == nil {
		return 0
	}
	return bcfg.Max
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
