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
)

var (
	minBlobGasPrice            = big.NewInt(params.BlobTxMinBlobGasprice)
	blobGaspriceUpdateFraction = big.NewInt(params.BlobTxBlobGaspriceUpdateFraction)
)

// VerifyEIP4844Header verifies the presence of the excessBlobGas field and that
// if the current block contains no transactions, the excessBlobGas is updated
// accordingly.
// VerifyEIP4844Header는 excessBlobGas, BlobGasUsed 필드의 존재 등을 검증하고,
// 현재 블록이 트랜잭션을 담고 있지 않으면, 그에따라 excessBlobGas가 업데이트한다.
func VerifyEIP4844Header(parent, header *types.Header) error {
	// Verify the header is not malformed
	// 헤더의 형식 검증
	if header.ExcessBlobGas == nil {
		return errors.New("header is missing excessBlobGas")
	}
	if header.BlobGasUsed == nil {
		return errors.New("header is missing blobGasUsed")
	}
	// Verify that the blob gas used remains within reasonable limits.
	// 사용된 blob gas가 합리적인 한도 내에 있는지 검증한다.
	if *header.BlobGasUsed > params.MaxBlobGasPerBlock {
		return fmt.Errorf("blob gas used %d exceeds maximum allowance %d", *header.BlobGasUsed, params.MaxBlobGasPerBlock)
	}
	if *header.BlobGasUsed%params.BlobTxBlobGasPerBlob != 0 {
		return fmt.Errorf("blob gas used %d not a multiple of blob gas per blob %d", header.BlobGasUsed, params.BlobTxBlobGasPerBlob)
	}
	// Verify the excessBlobGas is correct based on the parent header
	// excessBlobGas가 parent header를 기준으로 올바른지 확인한다.
	var (
		parentExcessBlobGas uint64
		parentBlobGasUsed   uint64
	)
	if parent.ExcessBlobGas != nil {
		parentExcessBlobGas = *parent.ExcessBlobGas
		parentBlobGasUsed = *parent.BlobGasUsed
	}
	expectedExcessBlobGas := CalcExcessBlobGas(parentExcessBlobGas, parentBlobGasUsed)
	if *header.ExcessBlobGas != expectedExcessBlobGas {
		return fmt.Errorf("invalid excessBlobGas: have %d, want %d, parent excessBlobGas %d, parent blobDataUsed %d",
			*header.ExcessBlobGas, expectedExcessBlobGas, parentExcessBlobGas, parentBlobGasUsed)
	}
	return nil
}

// CalcExcessBlobGas calculates the excess blob gas after applying the set of
// blobs on top of the excess blob gas.
// CalcExcessBlobGas는 excess blob gas위에 blob set을 적용한 이후, excess blob gas를 계산한다.
func CalcExcessBlobGas(parentExcessBlobGas uint64, parentBlobGasUsed uint64) uint64 {
	excessBlobGas := parentExcessBlobGas + parentBlobGasUsed
	// 목표치 이하로 사용했으므로 excess blob gas = 0
	if excessBlobGas < params.BlobTxTargetBlobGasPerBlock {
		return 0
	}
	// excessBlobGas = (parentExcessBlobGas + parentBlobGasUsed) - BlobTxTargetBlobGasPerBlock
	return excessBlobGas - params.BlobTxTargetBlobGasPerBlock
}

// CalcBlobFee calculates the blobfee from the header's excess blob gas field.
// CalcBlobFee는 header의 excess blob gase field로 blobfee를 계산한다.
func CalcBlobFee(excessBlobGas uint64) *big.Int {
	return fakeExponential(minBlobGasPrice, new(big.Int).SetUint64(excessBlobGas), blobGaspriceUpdateFraction)
}

// fakeExponential approximates factor * e ** (numerator / denominator) using
// Taylor expansion.
// 테일러 급수 전개로 근사적으로 계산
// e^x = 1 + x + x^2/2! + x^3/3! + ...
func fakeExponential(factor, numerator, denominator *big.Int) *big.Int {
	var (
		output = new(big.Int)
		accum  = new(big.Int).Mul(factor, denominator)
		// ------ accoum = new(big.Int).Set(factor)
	)
	for i := 1; accum.Sign() > 0; i++ {
		output.Add(output, accum)

		accum.Mul(accum, numerator)
		accum.Div(accum, denominator)
		accum.Div(accum, big.NewInt(int64(i)))
	}
	return output.Div(output, denominator)
	// ------ return output
	// 왜 굳이 마지막에 output을 denominator로 나누는거지?
	// 내 생각엔 함수의 시작에서 accum을 정의할때 factor와 denominator의 곱을 저장하기 때문에
	// denominator로 나누어준 거 같은데 그냥 처음부터 accum을 factor로 정의해서
	// denominator를 곱하고 나누는 연산과정을 생략할 수 있지 않나?
}
