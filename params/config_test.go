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

package params

import (
	"math/big"
	"reflect"
	"testing"
	"time"

	"gotest.tools/assert"

	"github.com/ethereum/go-ethereum/common/math"
)

func TestCheckCompatible(t *testing.T) {
	type test struct {
		stored, new   *ChainConfig
		headBlock     uint64
		headTimestamp uint64
		wantErr       *ConfigCompatError
	}

	tests := []test{
		{stored: AllEthashProtocolChanges, new: AllEthashProtocolChanges, headBlock: 0, headTimestamp: 0, wantErr: nil},
		{stored: AllEthashProtocolChanges, new: AllEthashProtocolChanges, headBlock: 0, headTimestamp: uint64(time.Now().Unix()), wantErr: nil},
		{stored: AllEthashProtocolChanges, new: AllEthashProtocolChanges, headBlock: 100, wantErr: nil},
		{
			stored:    &ChainConfig{EIP150Block: big.NewInt(10)},
			new:       &ChainConfig{EIP150Block: big.NewInt(20)},
			headBlock: 9,
			wantErr:   nil,
		},
		{
			stored:    AllEthashProtocolChanges,
			new:       &ChainConfig{HomesteadBlock: nil},
			headBlock: 3,
			wantErr: &ConfigCompatError{
				What:          "Homestead fork block",
				StoredBlock:   big.NewInt(0),
				NewBlock:      nil,
				RewindToBlock: 0,
			},
		},
		{
			stored:    AllEthashProtocolChanges,
			new:       &ChainConfig{HomesteadBlock: big.NewInt(1)},
			headBlock: 3,
			wantErr: &ConfigCompatError{
				What:          "Homestead fork block",
				StoredBlock:   big.NewInt(0),
				NewBlock:      big.NewInt(1),
				RewindToBlock: 0,
			},
		},
		{
			stored:    &ChainConfig{HomesteadBlock: big.NewInt(30), EIP150Block: big.NewInt(10)},
			new:       &ChainConfig{HomesteadBlock: big.NewInt(25), EIP150Block: big.NewInt(20)},
			headBlock: 25,
			wantErr: &ConfigCompatError{
				What:          "EIP150 fork block",
				StoredBlock:   big.NewInt(10),
				NewBlock:      big.NewInt(20),
				RewindToBlock: 9,
			},
		},
		{
			stored:    &ChainConfig{ConstantinopleBlock: big.NewInt(30)},
			new:       &ChainConfig{ConstantinopleBlock: big.NewInt(30), PetersburgBlock: big.NewInt(30)},
			headBlock: 40,
			wantErr:   nil,
		},
		{
			stored:    &ChainConfig{ConstantinopleBlock: big.NewInt(30)},
			new:       &ChainConfig{ConstantinopleBlock: big.NewInt(30), PetersburgBlock: big.NewInt(31)},
			headBlock: 40,
			wantErr: &ConfigCompatError{
				What:          "Petersburg fork block",
				StoredBlock:   nil,
				NewBlock:      big.NewInt(31),
				RewindToBlock: 30,
			},
		},
		{
			stored:        &ChainConfig{ShanghaiBlock: big.NewInt(30)},
			new:           &ChainConfig{ShanghaiBlock: big.NewInt(30)},
			headTimestamp: 9,
			wantErr:       nil,
		},
	}

	for _, test := range tests {
		err := test.stored.CheckCompatible(test.new, test.headBlock, test.headTimestamp)
		if !reflect.DeepEqual(err, test.wantErr) {
			t.Errorf("error mismatch:\nstored: %v\nnew: %v\nheadBlock: %v\nheadTimestamp: %v\nerr: %v\nwant: %v", test.stored, test.new, test.headBlock, test.headTimestamp, err, test.wantErr)
		}
	}
}

func TestConfigRules(t *testing.T) {
	t.Parallel()

	c := &ChainConfig{
		LondonBlock:   new(big.Int),
		ShanghaiBlock: big.NewInt(10),
		CancunBlock:   big.NewInt(20),
		PragueBlock:   big.NewInt(30),
		VerkleBlock:   big.NewInt(40),
	}

	block := new(big.Int)

	if r := c.Rules(block, true, 0); r.IsShanghai {
		t.Errorf("expected %v to not be shanghai", 0)
	}

	block.SetInt64(10)

	if r := c.Rules(block, true, 0); !r.IsShanghai {
		t.Errorf("expected %v to be shanghai", 10)
	}

	block.SetInt64(20)

	if r := c.Rules(block, true, 0); !r.IsCancun {
		t.Errorf("expected %v to be cancun", 20)
	}

	block.SetInt64(30)

	if r := c.Rules(block, true, 0); !r.IsPrague {
		t.Errorf("expected %v to be prague", 30)
	}

	block = block.SetInt64(math.MaxInt64)

	if r := c.Rules(block, true, 0); !r.IsShanghai {
		t.Errorf("expected %v to be shanghai", 0)
	}
}

func TestBorKeyValueConfigHelper(t *testing.T) {
	t.Parallel()

	backupMultiplier := map[string]uint64{
		"0":        2,
		"25275000": 5,
		"29638656": 2,
	}
	assert.Equal(t, borKeyValueConfigHelper(backupMultiplier, 0), uint64(2))
	assert.Equal(t, borKeyValueConfigHelper(backupMultiplier, 1), uint64(2))
	assert.Equal(t, borKeyValueConfigHelper(backupMultiplier, 25275000-1), uint64(2))
	assert.Equal(t, borKeyValueConfigHelper(backupMultiplier, 25275000), uint64(5))
	assert.Equal(t, borKeyValueConfigHelper(backupMultiplier, 25275000+1), uint64(5))
	assert.Equal(t, borKeyValueConfigHelper(backupMultiplier, 29638656-1), uint64(5))
	assert.Equal(t, borKeyValueConfigHelper(backupMultiplier, 29638656), uint64(2))
	assert.Equal(t, borKeyValueConfigHelper(backupMultiplier, 29638656+1), uint64(2))

	config := map[string]uint64{
		"0":         1,
		"90000000":  2,
		"100000000": 3,
	}
	assert.Equal(t, borKeyValueConfigHelper(config, 0), uint64(1))
	assert.Equal(t, borKeyValueConfigHelper(config, 1), uint64(1))
	assert.Equal(t, borKeyValueConfigHelper(config, 90000000-1), uint64(1))
	assert.Equal(t, borKeyValueConfigHelper(config, 90000000), uint64(2))
	assert.Equal(t, borKeyValueConfigHelper(config, 90000000+1), uint64(2))
	assert.Equal(t, borKeyValueConfigHelper(config, 100000000-1), uint64(2))
	assert.Equal(t, borKeyValueConfigHelper(config, 100000000), uint64(3))
	assert.Equal(t, borKeyValueConfigHelper(config, 100000000+1), uint64(3))

	burntContract := map[string]string{
		"22640000": "0x70bcA57F4579f58670aB2d18Ef16e02C17553C38",
		"41824608": "0x617b94CCCC2511808A3C9478ebb96f455CF167aA",
	}
	assert.Equal(t, borKeyValueConfigHelper(burntContract, 22640000), "0x70bcA57F4579f58670aB2d18Ef16e02C17553C38")
	assert.Equal(t, borKeyValueConfigHelper(burntContract, 22640000+1), "0x70bcA57F4579f58670aB2d18Ef16e02C17553C38")
	assert.Equal(t, borKeyValueConfigHelper(burntContract, 41824608-1), "0x70bcA57F4579f58670aB2d18Ef16e02C17553C38")
	assert.Equal(t, borKeyValueConfigHelper(burntContract, 41824608), "0x617b94CCCC2511808A3C9478ebb96f455CF167aA")
	assert.Equal(t, borKeyValueConfigHelper(burntContract, 41824608+1), "0x617b94CCCC2511808A3C9478ebb96f455CF167aA")
}
