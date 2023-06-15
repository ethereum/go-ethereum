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
	"errors"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

var (
	minDataGasPrice            = big.NewInt(params.BlobTxMinDataGasprice)
	dataGaspriceUpdateFraction = big.NewInt(params.BlobTxDataGaspriceUpdateFraction)
)

// VerifyEIP4844Header verifies the presence of the excessDataGas field and that
// if the current block contains no transactions, the excessDataGas is updated
// accordingly.
func VerifyEIP4844Header(parent, header *types.Header) error {
	// Verify the header is not malformed
	if header.ExcessDataGas == nil {
		return errors.New("header is missing excessDataGas")
	}
	if header.DataGasUsed == nil {
		return errors.New("header is missing dataGasUsed")
	}
	// Verify that the data gas used remains within reasonable limits.
	if *header.DataGasUsed > params.BlobTxMaxDataGasPerBlock {
		return fmt.Errorf("data gas used %d exceeds maximum allowance %d", *header.DataGasUsed, params.BlobTxMaxDataGasPerBlock)
	}
	if *header.DataGasUsed%params.BlobTxDataGasPerBlob != 0 {
		return fmt.Errorf("data gas used %d not a multiple of data gas per blob %d", header.DataGasUsed, params.BlobTxDataGasPerBlob)
	}
	// Verify the excessDataGas is correct based on the parent header
	var (
		parentExcessDataGas uint64
		parentDataGasUsed   uint64
	)
	if parent.ExcessDataGas != nil {
		parentExcessDataGas = *parent.ExcessDataGas
		parentDataGasUsed = *parent.DataGasUsed
	}
	expectedExcessDataGas := CalcExcessDataGas(parentExcessDataGas, parentDataGasUsed)
	if *header.ExcessDataGas != expectedExcessDataGas {
		return fmt.Errorf("invalid excessDataGas: have %d, want %d, parent excessDataGas %d, parent blobDataUsed %d",
			*header.ExcessDataGas, expectedExcessDataGas, parentExcessDataGas, parentDataGasUsed)
	}
	return nil
}

// CalcExcessDataGas calculates the excess data gas after applying the set of
// blobs on top of the excess data gas.
func CalcExcessDataGas(parentExcessDataGas uint64, parentDataGasUsed uint64) uint64 {
	excessDataGas := parentExcessDataGas + parentDataGasUsed
	if excessDataGas < params.BlobTxTargetDataGasPerBlock {
		return 0
	}
	return excessDataGas - params.BlobTxTargetDataGasPerBlock
}

// CalcBlobFee calculates the blobfee from the header's excess data gas field.
func CalcBlobFee(excessDataGas uint64) *big.Int {
	return fakeExponential(minDataGasPrice, new(big.Int).SetUint64(excessDataGas), dataGaspriceUpdateFraction)
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
