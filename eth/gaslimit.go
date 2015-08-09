// Copyright 2015 The go-ethereum Authors
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

package eth

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

// CalcGasLimit computes the gas limit of the next block after parent.
// The result may be modified by the caller.
func (eth *Ethereum) CalcGasLimit(parent *types.Block) *big.Int {

	var limit, step, min, max *big.Int
	step = parent.GasLimit()
	step = step.Div(step, params.GasLimitBoundDivisor)
	step = step.Sub(step, big.NewInt(1))
	min = parent.GasLimit()
	min = min.Sub(min, step)
	max = parent.GasLimit()
	max = max.Add(max, step)

	switch eth.Gls {
	case "target":
		limit = eth.GlsTarget
	case "blkutil":
		limit = eth.CalcGlsBlkUtil(parent)
	default:
		limit = parent.GasLimit()
	}

	if limit.Cmp(min) < 0 {
		limit = min
	}
	if limit.Cmp(max) > 0 {
		limit = max
	}
	return limit
}

func (eth *Ethereum) CalcGlsBlkUtil(parent *types.Block) *big.Int {
	// contrib = (parentGasUsed * 100 / GlsBlkUtil) / 1024
	contrib := new(big.Int).Mul(parent.GasUsed(), big.NewInt(100))
	contrib = contrib.Div(contrib, big.NewInt(int64(eth.GlsBlkUtil)))
	contrib = contrib.Div(contrib, params.GasLimitBoundDivisor)

	// decay = parentGasLimit / 1024 -1
	decay := new(big.Int).Div(parent.GasLimit(), params.GasLimitBoundDivisor)
	decay.Sub(decay, big.NewInt(1))

	/*
		strategy: gasLimit of block-to-mine is set based on parent's
		gasUsed value.  if parentGasUsed > parentGasLimit * (2/3) then we
		increase it, otherwise lower it (or leave it unchanged if it's right
		at that usage) the amount increased/decreased depends on how far away
		from parentGasLimit * (2/3) parentGasUsed is.
	*/
	gl := new(big.Int).Sub(parent.GasLimit(), decay)
	gl = gl.Add(gl, contrib)
	gl.Set(common.BigMax(gl, params.MinGasLimit))

	// however, if we're now below the target (GenesisGasLimit) we increase the
	// limit as much as we can (parentGasLimit / 1024 -1)
	if gl.Cmp(params.GenesisGasLimit) < 0 {
		gl.Add(parent.GasLimit(), decay)
		gl.Set(common.BigMin(gl, params.GenesisGasLimit))
	}
	return gl
}
