// Copyright 2016 The go-ethereum Authors
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

package vm

import (
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/params"
)

func TestInit(t *testing.T) {
	jumpTable := newJumpTable(&params.ChainConfig{HomesteadBlock: big.NewInt(1)}, big.NewInt(0))
	if jumpTable[DELEGATECALL].valid {
		t.Error("Expected DELEGATECALL not to be present")
	}

	for _, n := range []int64{1, 2, 100} {
		jumpTable := newJumpTable(&params.ChainConfig{HomesteadBlock: big.NewInt(1)}, big.NewInt(n))
		if !jumpTable[DELEGATECALL].valid {
			t.Error("Expected DELEGATECALL to be present for block", n)
		}
	}
}
