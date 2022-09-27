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
	"math"

	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// CalcExcessBlobTransactions calculates the number of blobs above the target
func CalcExcessBlobTransactions(parent *types.Header, blobs uint64) uint64 {
	var excessBlobs uint64
	if parent.ExcessBlobs != nil {
		excessBlobs = *parent.ExcessBlobs
	}
	adjusted := excessBlobs + blobs
	if adjusted < params.TargetBlobsPerBlock {
		return 0
	}
	return adjusted - params.TargetBlobsPerBlock
}

// FakeExponential approximates 2 ** (num / denom)
func FakeExponential(num uint64, denom uint64) uint64 {
	cofactor := uint64(math.Exp2(float64(num / denom)))
	fractional := num % denom
	return cofactor + (fractional*cofactor*2+
		(uint64(math.Pow(float64(fractional), 2))*cofactor)/denom)/(denom*3)
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
	if header.ExcessBlobs == nil {
		return fmt.Errorf("header is missing excessBlobs")
	}
	return nil
}
