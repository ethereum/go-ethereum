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
//
// We cannot verify excessDataGas if there *are* transactions included as that
// would require the block body. However, it is nonetheless useful to verify the
// header in case of an empty body since certain code might skip body validations
// with no included transactions (e.g. snap sync).
func VerifyEIP4844Header(parent, header *types.Header) error {
	// Verify the header is not malformed
	if header.ExcessDataGas == nil {
		return errors.New("header is missing excessDataGas")
	}
	// Verify the excessDataGas is correct based on the parent header iff the
	// transaction list is empty. For non-empty blocks, validation needs to be
	// done later.
	if header.TxHash == types.EmptyTxsHash {
		expectedExcessDataGas := CalcExcessDataGas(parent.ExcessDataGas, 0)
		if header.ExcessDataGas.Cmp(expectedExcessDataGas) != 0 {
			return fmt.Errorf("invalid excessDataGas: have %s, want %s, parentExcessDataGas %s, blob txs %d",
				header.ExcessDataGas, expectedExcessDataGas, parent.ExcessDataGas, 0)
		}
	}
	return nil
}

// CalcExcessDataGas calculates the excess data gas after applying the set of
// blobs on top of the paren't excess data gas.
//
// Note, the excessDataGas is akin to gasUsed, in that it's calculated post-
// execution of the blob transactions not before. Hence, the blob fee used to
// pay for the blobs are actually derived from the parent data gas.
func CalcExcessDataGas(parentExcessDataGas *big.Int, blobs int) *big.Int {
	excessDataGas := new(big.Int)
	if parentExcessDataGas != nil {
		excessDataGas.Set(parentExcessDataGas)
	}
	consumed := big.NewInt(params.BlobTxDataGasPerBlob)
	consumed.Mul(consumed, big.NewInt(int64(blobs)))
	excessDataGas.Add(excessDataGas, consumed)

	targetGas := big.NewInt(params.BlobTxTargetDataGasPerBlock)
	if excessDataGas.Cmp(targetGas) < 0 {
		return new(big.Int)
	}
	return new(big.Int).Sub(excessDataGas, targetGas)
}

// CalcBlobFee calculates the blobfee from the header's excess data gas field.
//
// Note, the blob fee used to pay for blob transactions should be derived from
// the parent block's excess data gas, since it is a post-execution field akin
// to gas used.
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
