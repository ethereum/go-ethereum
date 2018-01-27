// Copyright 2018 The go-ethereum Authors
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

// Package chainstats implements some chain utilities for sync-free blockchain info lookup

package chainstats

import (
	"github.com/ethereum/go-ethereum/core/types"
	"math/big"
	"sync/atomic"
)

type Chainstats struct {
	currentBlockNumber     atomic.Value
	currentFastBlockNumber atomic.Value
	currentTd              atomic.Value
	currentFastTd          atomic.Value
}

func NewChainstats() *Chainstats {
	return &Chainstats{}
}
func (stats *Chainstats) GetNumber() uint64 {
	return stats.currentBlockNumber.Load().(*big.Int).Uint64()
}
func (stats *Chainstats) UpdateNumbers(currentBlock, currentFastBlock *types.Block) {
	stats.currentBlockNumber.Store(currentBlock.Number())
	stats.currentFastBlockNumber.Store(currentFastBlock.Number())
}
func (stats *Chainstats) SetNumber(number *big.Int) {
	stats.currentBlockNumber.Store(number)
}
func (stats *Chainstats) GetFastNumber() uint64 {
	return stats.currentFastBlockNumber.Load().(*big.Int).Uint64()
}
func (stats *Chainstats) GetNumbers() (uint64, uint64) {
	return stats.currentBlockNumber.Load().(*big.Int).Uint64(),
		stats.currentFastBlockNumber.Load().(*big.Int).Uint64()
}
func (stats *Chainstats) SetFastNumber(number *big.Int) {
	stats.currentFastBlockNumber.Store(number)
}
func (stats *Chainstats) GetTotalDifficulty() *big.Int {
	return new(big.Int).Set(stats.currentTd.Load().(*big.Int))
}
func (stats *Chainstats) SetTotalDifficulty(newTd *big.Int) {
	stats.currentTd.Store(newTd)
}
func (stats *Chainstats) SetTotalFastDifficulty(newTd *big.Int) {
	stats.currentFastTd.Store(newTd)
}
