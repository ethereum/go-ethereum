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

package misc

import (
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

func VerifyEip1559Header(config *params.ChainConfig, parent, header *types.Header) error {
	// Verify the header is not malformed
	if header.BaseFee == nil {
		return fmt.Errorf("missing baseFee after EIP-1559 fork block")
	}

	// Verify that the gasUsed is <= gasTarget*elasticityMultiplier
	if header.GasUsed > header.GasLimit*params.ElasticityMultiplier {
		return fmt.Errorf("exceeded elasticity multiplier: gasUsed %d, gasTarget*elasticityMultiplier %d", header.GasUsed, header.GasLimit*params.ElasticityMultiplier)
	}

	// Verify the baseFee is correct based on the parent header.
	expectedBaseFee := CalcBaseFee(config, parent)
	if header.BaseFee.Cmp(expectedBaseFee) != 0 {
		return fmt.Errorf("invalid baseFee: expected: %d, have %d, parentBaseFee: %v, parentGasUsed: %v", expectedBaseFee, header.BaseFee.Int64(), parent.BaseFee.Int64(), parent.GasUsed)
	}

	return nil
}

func CalcBaseFee(config *params.ChainConfig, parent *types.Header) *big.Int {
	// If the current block is the first EIP-1559 block, return the InitialBaseFee.
	if !config.IsLondon(parent.Number) {
		return new(big.Int).SetUint64(params.InitialBaseFee)
	}

	// If the parent gasUsed is the same as the target, the baseFee remains unchanged.
	if parent.GasUsed == parent.GasLimit {
		return new(big.Int).Set(parent.BaseFee)
	}

	var (
		gasLimit                 = new(big.Int).SetUint64(parent.GasLimit)
		baseFeeChangeDenominator = new(big.Int).SetUint64(params.BaseFeeChangeDenominator)
	)

	if parent.GasUsed > parent.GasLimit {
		// If the parent block used more gas than its target, the baseFee should increase.
		gasUsedDelta := new(big.Int).SetUint64(parent.GasUsed - parent.GasLimit)
		x := new(big.Int).Mul(parent.BaseFee, gasUsedDelta)
		y := x.Div(x, gasLimit)
		baseFeeDelta := math.BigMax(
			x.Div(y, baseFeeChangeDenominator),
			common.Big1,
		)

		return x.Add(parent.BaseFee, baseFeeDelta)
	} else {
		// Otherwise if the parent block used less gas than its target, the baseFee should decrease.
		gasUsedDelta := new(big.Int).SetUint64(parent.GasLimit - parent.GasUsed)
		x := new(big.Int).Mul(parent.BaseFee, gasUsedDelta)
		y := x.Div(x, gasLimit)
		baseFeeDelta := x.Div(y, baseFeeChangeDenominator)

		return math.BigMax(
			x.Sub(parent.BaseFee, baseFeeDelta),
			common.Big0,
		)
	}
}
