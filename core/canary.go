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

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

var (
	jeff      = common.HexToAddress("959c33de5961820567930eccce51ea715c496f85")
	vitalik   = common.HexToAddress("c8158da0b567a8cc898991c2c2a073af67dc03a9")
	christoph = common.HexToAddress("7a19a893f91d5b6e2cdf941b6acbba2cbcf431ee")
	gav       = common.HexToAddress("539dd9aaf45c3feb03f9c004f4098bd3268fef6b")
)

// Canary will check the 0'd address of the 4 contracts above.
// If two or more are set to anything other than a 0 the canary
// dies a horrible death.
func Canary(statedb *state.StateDB) bool {
	var r int
	if (statedb.GetState(jeff, common.Hash{}).Big().Cmp(big.NewInt(0)) > 0) {
		r++
	}
	if (statedb.GetState(gav, common.Hash{}).Big().Cmp(big.NewInt(0)) > 0) {
		r++
	}
	if (statedb.GetState(christoph, common.Hash{}).Big().Cmp(big.NewInt(0)) > 0) {
		r++
	}
	if (statedb.GetState(vitalik, common.Hash{}).Big().Cmp(big.NewInt(0)) > 0) {
		r++
	}
	return r > 1
}
