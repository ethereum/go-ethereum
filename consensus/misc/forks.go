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
	errInvalidInitialBaseFee           = fmt.Errorf("initial BaseFee must equal %d", params.EIP1559InitialBaseFee)
	errInvalidBaseFee                  = errors.New("invalid BaseFee")
	errMissingParentBaseFee            = errors.New("parent header is missing BaseFee")
	errMissingBaseFee                  = errors.New("current header is missing BaseFee")
	errHaveBaseFee                     = fmt.Errorf("BaseFee should not be set before block %d", params.EIP1559ForkBlockNumber)
	errInvalidEIP1559FinalizedGasLimit = fmt.Errorf("after EIP1559 finalization, GasLimit must equal %d", params.MaxGasEIP1559)
)

// VerifyForkHashes verifies that blocks conforming to network hard-forks do have
// the correct hashes, to avoid clients going off on different chains. This is an
// optional feature.
func VerifyForkHashes(config *params.ChainConfig, header *types.Header, uncle bool) error {
	// We don't care about uncles
	if uncle {
		return nil
	}
	// If the homestead reprice hash is set, validate it
	if config.EIP150Block != nil && config.EIP150Block.Cmp(header.Number) == 0 {
		if config.EIP150Hash != (common.Hash{}) && config.EIP150Hash != header.Hash() {
			return fmt.Errorf("homestead gas reprice fork: have 0x%x, want 0x%x", header.Hash(), config.EIP150Hash)
		}
	}
	// All ok, return
	return nil
}

// VerifyEIP1559BaseFee verifies that the EIP1559 BaseFee field is valid for the current block height
func VerifyEIP1559BaseFee(config *params.ChainConfig, header, parent *types.Header) error {
	// If we are at the EIP1559 fork block the BaseFee needs to be equal to params.EIP1559InitialBaseFee
	if config.EIP1559Block != nil && config.EIP1559Block.Cmp(header.Number) == 0 {
		if header.BaseFee == nil || header.BaseFee.Cmp(new(big.Int).SetUint64(params.EIP1559InitialBaseFee)) != 0 {
			return errInvalidInitialBaseFee
		}
		return nil
	}
	// If we are past the EIP1559 activation block verify the header's BaseFee is valid by deriving
	// it from the parent header and validating that they are the same
	if config.IsEIP1559(header.Number) {
		if parent.BaseFee == nil {
			return errMissingParentBaseFee
		}
		if header.BaseFee == nil {
			return errMissingBaseFee
		}
		delta := new(big.Int).Sub(new(big.Int).SetUint64(parent.GasUsed), new(big.Int).SetUint64(params.TargetGasUsed))
		mul := new(big.Int).Mul(parent.BaseFee, delta)
		div := new(big.Int).Div(mul, new(big.Int).SetUint64(params.TargetGasUsed))
		div2 := new(big.Int).Div(div, new(big.Int).SetUint64(params.BaseFeeMaxChangeDenominator))
		expectedBaseFee := new(big.Int).Add(parent.BaseFee, div2)
		diff := new(big.Int).Sub(expectedBaseFee, parent.BaseFee)
		neg := false
		if diff.Sign() < 0 {
			neg = true
			diff.Neg(diff)
		}
		max := new(big.Int).Div(parent.BaseFee, new(big.Int).SetUint64(params.BaseFeeMaxChangeDenominator))
		if max.Cmp(common.Big1) < 0 {
			max = common.Big1
		}
		if diff.Cmp(max) > 0 {
			if neg {
				max.Neg(max)
			}
			expectedBaseFee.Set(new(big.Int).Add(parent.BaseFee, max))
		}
		if expectedBaseFee.Cmp(header.BaseFee) != 0 {
			return errInvalidBaseFee
		}
		return nil
	}
	// If we are before the EIP1559 activation block the current and parent BaseFees should be nil
	if header.BaseFee != nil || parent.BaseFee != nil {
		return errHaveBaseFee
	}
	return nil
}

// VerifyEIP1559GasLimit verifies that the header.GasLimit field is valid for the current block height
// Only call this after activation has been confirmed (config.IsEIP1559(header.Number) == true)
func VerifyEIP1559GasLimit(config *params.ChainConfig, header *types.Header) error {
	// If EIP1559 has been finalized then header.GasLimit should be equal to the MaxGasEIP1559 (entire limit is in EIP1559 pool)
	if config.IsEIP1559Finalized(header.Number) {
		if header.GasLimit != params.MaxGasEIP1559 {
			return errInvalidEIP1559FinalizedGasLimit
		}
		return nil
	}
	// Else if we are between activation and finalization, header.GasLimit must be valid based on the decay function
	numOfIncrements := new(big.Int).Sub(header.Number, config.EIP1559Block).Uint64()
	expectedGasLimit := (params.MaxGasEIP1559 / 2) + (numOfIncrements * params.EIP1559GasIncrementAmount)
	if header.GasLimit != expectedGasLimit {
		return fmt.Errorf("invalid GasLimit: have %d, need %d", header.GasLimit, expectedGasLimit)
	}
	return nil
}
