// Copyright 2015 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.

package core

import (
	"math/big"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
)

var (
	jeff      = common.HexToAddress("a8edb1ac2c86d3d9d78f96cd18001f60df29e52c")
	vitalik   = common.HexToAddress("1baf27b88c48dd02b744999cf3522766929d2b2a")
	christoph = common.HexToAddress("60d11b58744784dc97f878f7e3749c0f1381a004")
	gav       = common.HexToAddress("4bb7e8ae99b645c2b7860b8f3a2328aae28bd80a")
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
