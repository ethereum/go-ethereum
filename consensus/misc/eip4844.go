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

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// CalcExcessDataGas implements calc_excess_data_gas from EIP-4844
func CalcExcessDataGas(parentExcessDataGas *big.Int, newBlobs int) *big.Int {
	excessDataGas := new(big.Int)
	if parentExcessDataGas != nil {
		excessDataGas.Set(parentExcessDataGas)
	}
	consumedGas := big.NewInt(params.DataGasPerBlob)
	consumedGas.Mul(consumedGas, big.NewInt(int64(newBlobs)))

	excessDataGas.Add(excessDataGas, consumedGas)
	targetGas := big.NewInt(params.TargetDataGasPerBlock)
	if excessDataGas.Cmp(targetGas) < 0 {
		return new(big.Int)
	}
	return new(big.Int).Set(excessDataGas.Sub(excessDataGas, targetGas))
}

// FakeExponential approximates factor * e ** (num / denom) using a taylor expansion
// as described in the EIP-4844 spec.
func FakeExponential(factor, num, denom *big.Int) *big.Int {
	output := new(big.Int)
	numAccum := new(big.Int).Mul(factor, denom)
	for i := 1; numAccum.Sign() > 0; i++ {
		output.Add(output, numAccum)
		numAccum.Mul(numAccum, num)
		iBig := big.NewInt(int64(i))
		numAccum.Div(numAccum, iBig.Mul(iBig, denom))
	}
	return output.Div(output, denom)
}

// CountBlobs returns the number of blob transactions in txs
func CountBlobs(txs []*types.Transaction) int {
	var count int
	for _, tx := range txs {
		count += len(tx.DataHashes())
	}
	return count
}

// VerifyEip4844Header verifies that the header is not malformed
func VerifyEip4844Header(config *params.ChainConfig, parent, header *types.Header) error {
	if header.ExcessDataGas == nil {
		return fmt.Errorf("header is missing excessDataGas")
	}
	return nil
}
