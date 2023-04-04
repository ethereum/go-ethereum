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

package misc

import (
	"math/big"

	"github.com/ethereum/go-ethereum/params"
)

var (
	minDataGasPrice            = big.NewInt(params.BlobTxMinDataGasprice)
	dataGaspriceUpdateFraction = big.NewInt(params.BlobTxDataGaspriceUpdateFraction)
)

// CalcBlobFee calculates the blobfee from the header's excess data gas field.
func CalcBlobFee(excessDataGas *big.Int) *big.Int {
	// If this block does not yet have EIP-4844 enabled, return the starting fee
	if excessDataGas == nil {
		return big.NewInt(params.BlobTxMinDataGasprice)
	}
	return fakeExponential(minDataGasPrice, excessDataGas, dataGaspriceUpdateFraction)
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
