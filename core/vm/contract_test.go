// Copyright 2017 The go-ethereum Authors
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
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/holiman/uint256"
)

func TestValidJumpdest(t *testing.T) {
	cField := []byte{11, 12}
	c := Contract{Code: cField}
	destField := [4]uint64{2, 0, 0, 0}
	dest := uint256.Int(destField)
	ret := c.validJumpdest(&dest)
	if ret != false {
		t.Errorf("validJumpdest not successful")
	}

	c = Contract{}
	destField = [4]uint64{0, 1, 0, 0}
	dest = uint256.Int(destField)
	ret = c.validJumpdest(&dest)
	if ret != false {
		t.Errorf("validJumpdest not successful")
	}

	c = Contract{Code: cField}
	destField = [4]uint64{1, 0, 0, 0}
	dest = uint256.Int(destField)
	ret = c.validJumpdest(&dest)
	if ret != false {
		t.Errorf("validJumpdest not successful")
	}

	cField[1] = byte(JUMPDEST)
	c = Contract{Code: cField}
	ret = c.validJumpdest(&dest)
	if ret != true {
		t.Errorf("validJumpdest not successful")
	}
}

func TestIsCode(t *testing.T) {
	destField := [4]uint64{1, 0, 0, 0}
	cField := []byte{11, byte(JUMPDEST)}
	c := Contract{Code: cField}
	ret := c.isCode(destField[0])
	if ret != true || c.analysis == nil {
		t.Errorf("isCode not successful")
	}

	destField = [4]uint64{1, 0, 0, 0}
	aField := []byte{0, 0, 0, 0, 0}
	c = Contract{analysis: aField}
	ret = c.isCode(destField[0])
	if ret != true {
		t.Errorf("isCode not successful")
	}

	destField = [4]uint64{1, 0, 0, 0}
	jd := make(map[common.Hash]bitvec)
	c = Contract{CodeHash: common.Hash{}, jumpdests: jd}
	c.CodeHash[0] = 1
	ret = c.isCode(destField[0])
	if ret != true || c.analysis == nil {
		t.Errorf("isCode not successful")
	}
}
