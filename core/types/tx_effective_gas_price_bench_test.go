// Copyright 2024 The go-ethereum Authors
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

package types

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

func BenchmarkBlobTxEffectiveGasPrice(b *testing.B) {
	gasTipCap := uint256.NewInt(2000000000) // 2 gwei
	gasFeeCap := uint256.NewInt(3000000000) // 3 gwei
	baseFee := big.NewInt(1000000000)       // 1 gwei

	tx := &BlobTx{
		ChainID:   uint256.NewInt(1),
		Nonce:     0,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       21000,
		To:        common.Address{},
		Value:     uint256.NewInt(0),
		Data:      nil,
	}

	b.Run("WithBaseFee", func(b *testing.B) {
		b.ReportAllocs()
		dst := new(big.Int)
		for b.Loop() {
			tx.effectiveGasPrice(dst, baseFee)
		}
	})
}

func BenchmarkDynamicFeeTxEffectiveGasPrice(b *testing.B) {
	gasTipCap := big.NewInt(2000000000) // 2 gwei
	gasFeeCap := big.NewInt(3000000000) // 3 gwei
	baseFee := big.NewInt(1000000000)   // 1 gwei

	tx := &DynamicFeeTx{
		ChainID:   big.NewInt(1),
		Nonce:     0,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       21000,
		To:        &common.Address{},
		Value:     big.NewInt(0),
		Data:      nil,
	}

	b.Run("WithBaseFee", func(b *testing.B) {
		b.ReportAllocs()
		dst := new(big.Int)
		for b.Loop() {
			tx.effectiveGasPrice(dst, baseFee)
		}
	})
}

func BenchmarkSetCodeTxEffectiveGasPrice(b *testing.B) {
	gasTipCap := uint256.NewInt(2000000000) // 2 gwei
	gasFeeCap := uint256.NewInt(3000000000) // 3 gwei
	baseFee := big.NewInt(1000000000)       // 1 gwei

	tx := &SetCodeTx{
		ChainID:   uint256.NewInt(1),
		Nonce:     0,
		GasTipCap: gasTipCap,
		GasFeeCap: gasFeeCap,
		Gas:       21000,
		To:        common.Address{},
		Value:     uint256.NewInt(0),
		Data:      nil,
	}

	b.Run("WithBaseFee", func(b *testing.B) {
		b.ReportAllocs()
		dst := new(big.Int)
		for b.Loop() {
			tx.effectiveGasPrice(dst, baseFee)
		}
	})
}
