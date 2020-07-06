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

package misc

import (
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

var (
	errInvalidBaseFee       = errors.New("invalid BaseFee")
	errMissingParentBaseFee = errors.New("parent header is missing BaseFee")
	errMissingBaseFee       = errors.New("current header is missing BaseFee")
	errHaveBaseFee          = fmt.Errorf("BaseFee should not be set before block %d", params.EIP1559ForkBlockNumber)
)

// VerifyEIP1559BaseFee verifies that the EIP1559 BaseFee field is valid for the current block height
func VerifyEIP1559BaseFee(config *params.ChainConfig, header, parent *types.Header) error {
	if config.IsEIP1559(parent.Number) {
		if parent.BaseFee == nil {
			return errMissingParentBaseFee
		}
	}
	expectedBaseFee := CalcBaseFee(config, parent)
	if header.BaseFee == nil {
		if expectedBaseFee != nil {
			return errMissingBaseFee
		}
		return nil
	}
	if expectedBaseFee == nil {
		if header.BaseFee != nil {
			return errHaveBaseFee
		}
		return nil
	}
	if header.BaseFee.Cmp(expectedBaseFee) != 0 {
		return errInvalidBaseFee
	}
	return nil
}

// CalcBaseFee returns the baseFee for the current block provided the parent header and config parameters
func CalcBaseFee(config *params.ChainConfig, parent *types.Header) *big.Int {
	height := new(big.Int).Add(parent.Number, common.Big1)
	// If we are before EIP1559 activation, the baseFee is nil
	if !config.IsEIP1559(height) {
		return nil
	}
	// If we are at the block of EIP1559 activation then the BaseFee is set to the initial value
	if config.EIP1559Block.Cmp(height) == 0 {
		return new(big.Int).SetUint64(config.EIP1559.InitialBaseFee)
	}

	// Otherwise,
	// BASEFEE = PARENT_BASEFEE + PARENT_BASEFEE * delta // PARENT_EIP1559_GAS_TARGET // BASEFEE_MAX_CHANGE_DENOMINATOR
	// Where delta = parent.GasUsed - PARENT_EIP1559_GAS_TARGET
	parentGasTarget := CalcEIP1559GasTarget(config, parent.Number, new(big.Int).SetUint64(parent.GasLimit))
	delta := new(big.Int).Sub(new(big.Int).SetUint64(parent.GasUsed), parentGasTarget)
	mul := new(big.Int).Mul(parent.BaseFee, delta)
	div := new(big.Int).Div(mul, parentGasTarget)
	div2 := new(big.Int).Div(div, new(big.Int).SetUint64(config.EIP1559.EIP1559BaseFeeMaxChangeDenominator))

	baseFee := new(big.Int).Add(parent.BaseFee, div2)

	// A valid BASEFEE is one such that abs(BASEFEE - PARENT_BASEFEE) <= max(1, PARENT_BASEFEE // BASEFEE_MAX_CHANGE_DENOMINATOR)
	diff := new(big.Int).Sub(baseFee, parent.BaseFee)
	neg := false
	if diff.Sign() < 0 {
		neg = true
		diff.Neg(diff)
	}
	max := new(big.Int).Div(parent.BaseFee, new(big.Int).SetUint64(config.EIP1559.EIP1559BaseFeeMaxChangeDenominator))
	if max.Cmp(common.Big1) < 0 {
		max = common.Big1
	}
	// If derived BaseFee is not valid, restrict it within the bounds
	if diff.Cmp(max) > 0 {
		if neg {
			max.Neg(max)
		}
		baseFee.Set(new(big.Int).Add(parent.BaseFee, max))
	}
	return baseFee
}

// CalcEIP1559GasTarget returns the EIP1559GasTarget at the current height and header.GasLimit
// This should only be called at or above the block height of EIP1559 activation and below finalization
func CalcEIP1559GasTarget(chainConfig *params.ChainConfig, height, gasLimit *big.Int) *big.Int {
	// After EIP1559 finalization the entire header.GasLimit field instead represents the EIP1559GasTarget
	if chainConfig.IsEIP1559Finalized(height) {
		return gasLimit
	} else if chainConfig.IsEIP1559(height) {
		// During transition,
		// EIP1559GasTarget = (header.GasLimit/2) + (header.GasLimit/2) * (blockNumber-initBlockNumber) / migrationBlockDuration
		// migrationBlockDuration cannot be 0 or IsEIP1559Finalized would be true
		halfLim := new(big.Int).Div(gasLimit, big.NewInt(2))
		blockDiff := new(big.Int).Sub(height, chainConfig.EIP1559Block)
		migrationBlockDuration := new(big.Int).SetUint64(chainConfig.EIP1559.MigrationBlockDuration)
		return new(big.Int).Add(halfLim, new(big.Int).Div(new(big.Int).Mul(halfLim, blockDiff), migrationBlockDuration))
	}
	// Before EIP1559 activation the target is 0
	return big.NewInt(0)
}
