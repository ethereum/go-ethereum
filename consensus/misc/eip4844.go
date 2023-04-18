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

// CountBlobs returns the number of blob transactions in txs
func CountBlobs(txs []*types.Transaction) int {
	var count int
	for _, tx := range txs {
		count += len(tx.DataHashes())
	}
	return count
}

// VerifyEip4844Header verifies that the header is not malformed but does *not* check the value of excessDataGas.
// See VerifyExcessDataGas for the full check.
func VerifyEip4844Header(config *params.ChainConfig, parent, header *types.Header) error {
	if header.ExcessDataGas == nil {
		return fmt.Errorf("header is missing excessDataGas")
	}
	return nil
}

// VerifyExcessDataGas verifies the excess_data_gas in the block header
func VerifyExcessDataGas(chainReader ChainReader, block *types.Block) error {
	excessDataGas := block.ExcessDataGas()
	if !chainReader.Config().IsSharding(block.Time()) {
		if excessDataGas != nil {
			return fmt.Errorf("unexpected excessDataGas in header")
		}
		return nil

	}
	if excessDataGas == nil {
		return fmt.Errorf("header is missing excessDataGas")
	}

	number, parent := block.NumberU64()-1, block.ParentHash()
	parentBlock := chainReader.GetBlock(parent, number)
	if parentBlock == nil {
		return fmt.Errorf("parent block not found")
	}
	numBlobs := CountBlobs(block.Transactions())
	expectedEDG := CalcExcessDataGas(parentBlock.ExcessDataGas(), numBlobs)
	if excessDataGas.Cmp(expectedEDG) != 0 {
		return fmt.Errorf("invalid excessDataGas: have %s want %v", excessDataGas, expectedEDG)
	}
	return nil
}

// ChainReader defines a small collection of methods needed to access the local
// blockchain for EIP4844 block verifcation.
type ChainReader interface {
	Config() *params.ChainConfig
	// GetBlock retrieves a block from the database by hash and number.
	GetBlock(hash common.Hash, number uint64) *types.Block
}
