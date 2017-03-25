// Copyright 2014 The go-ethereum Authors
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

package pow

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
)

type Block interface {
	Difficulty() *big.Int
	HashNoNonce() common.Hash
	Nonce() uint64
	MixDigest() common.Hash
	NumberU64() uint64
}

type ChainManager interface {
	GetBlockByNumber(uint64) *types.Block
	CurrentBlock() *types.Block
}

type PoW interface {
	Verify(block Block) error
	Search(block Block, stop <-chan struct{}) (uint64, []byte)
	Hashrate() float64
}

// FakePow is a non-validating proof of work implementation.
// It returns true from Verify for any block.
type FakePow struct{}

// Verify implements PoW, returning a success for an input.
func (pow FakePow) Verify(block Block) error { return nil }

// Search implements PoW, returning the nonce 0 for any call.
func (pow FakePow) Search(block Block, stop <-chan struct{}) (uint64, []byte) {
	return 0, nil
}

// Hashrate implements PoW, returning 0.
func (pow FakePow) Hashrate() float64 { return 0 }
