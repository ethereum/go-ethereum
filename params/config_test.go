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

	"github.com/ethereum/go-ethereum/common"
)

// Test HF::EIPs boolean logic
func TestIsByzantiumAndAssociatedEIPFFns(t *testing.T) {
	blocksWantsAroundFork := func(forkBlock *big.Int) (blocks []*big.Int, wants []bool) {
		blocks, wants = append(blocks, forkBlock), append(wants, forkBlock != nil)
		if forkBlock == nil {
			blocks, wants = append(blocks, big.NewInt(0)), append(wants, false)
			blocks, wants = append(blocks, big.NewInt(42)), append(wants, false)
			return
		}
		blocks, wants = append(blocks, new(big.Int).Sub(forkBlock, common.Big1)), append(wants, false)
		blocks, wants = append(blocks, new(big.Int).Add(forkBlock, common.Big1)), append(wants, true)
		return
	}

	c := &ChainConfig{}
	*c = *MainnetChainConfig
	blocks, wants := blocksWantsAroundFork(c.ByzantiumBlock)
	for i, b := range blocks {
		if c.IsByzantium(b) != wants[i] {
			t.Errorf("i: %d, b: %v, got: %v, want: %v", i, b, c.IsByzantium(b), wants[i])
		}
		// Show that Byzantium's EIP<N>F block methods imply Byzantium block presence
		for j, fn := range []func(*big.Int) bool{
			c.IsEIP100F,
			c.IsEIP140F,
			c.IsEIP198F,
			c.IsEIP211F,
			c.IsEIP212F,
			c.IsEIP213F,
			c.IsEIP214F,
			c.IsEIP649F,
			c.IsEIP658F,
		} {
			if fn(b) != c.IsByzantium(b) {
				t.Errorf("j: %d, b: %v, got: %v, want: %v", j, b, fn(b), c.IsByzantium(b))
			}
		}
	}

	// Show that presence of all Byzantium's EIP<N>F blocks alone satisfy IsByzantium fn
	c.EIP100FBlock = new(big.Int).Set(c.ByzantiumBlock)
	c.EIP140FBlock = new(big.Int).Set(c.ByzantiumBlock)
	c.EIP198FBlock = new(big.Int).Set(c.ByzantiumBlock)
	c.EIP211FBlock = new(big.Int).Set(c.ByzantiumBlock)
	c.EIP212FBlock = new(big.Int).Set(c.ByzantiumBlock)
	c.EIP213FBlock = new(big.Int).Set(c.ByzantiumBlock)
	c.EIP214FBlock = new(big.Int).Set(c.ByzantiumBlock)
	c.EIP649FBlock = new(big.Int).Set(c.ByzantiumBlock)
	c.EIP658FBlock = new(big.Int).Set(c.ByzantiumBlock)
	c.ByzantiumBlock = nil
	for i, b := range blocks {
		if c.IsByzantium(b) != wants[i] {
			t.Errorf("i: %d, b: %v, got: %v, want: %v", i, b, c.IsByzantium(b), wants[i])
		}
		for j, fn := range []func(*big.Int) bool{
			c.IsEIP100F,
			c.IsEIP140F,
			c.IsEIP198F,
			c.IsEIP211F,
			c.IsEIP212F,
			c.IsEIP213F,
			c.IsEIP214F,
			c.IsEIP649F,
			c.IsEIP658F,
		} {
			if fn(b) != c.IsByzantium(b) {
				t.Errorf("j: %d, b: %v, got: %v, want: %v", j, b, fn(b), c.IsByzantium(b))
			}
		}
	}

	// Show that ALL EIP<N>F blocks must be set in order to be sufficiently "Byzantium"
	c.EIP658FBlock = nil
	for i, b := range blocks {
		if c.IsByzantium(b) {
			t.Errorf("i: %d, b: %v, got: %v, want: %v", i, b, c.IsByzantium(b), wants[i])
		}
		for j, fn := range []func(*big.Int) bool{
			c.IsEIP100F,
			c.IsEIP140F,
			c.IsEIP198F,
			c.IsEIP211F,
			c.IsEIP212F,
			c.IsEIP213F,
			c.IsEIP214F,
			c.IsEIP649F,
		} {
			if fn(b) != wants[i] {
				t.Errorf("j: %d, b: %v, got: %v, want: %v", j, b, fn(b), wants[i])
			}
		}
		if c.IsEIP658F(b) {
			t.Errorf("got: %v, want: %v", c.IsEIP658F(b), false)
		}
	}
}

func TestCheckCompatible(t *testing.T) {
	type test struct {
		stored, new *ChainConfig
		head        uint64
		wantErr     *ConfigCompatError
	}
	tests := []test{
		{stored: AllEthashProtocolChanges, new: AllEthashProtocolChanges, head: 0, wantErr: nil},
		{stored: AllEthashProtocolChanges, new: AllEthashProtocolChanges, head: 100, wantErr: nil},
		{
			stored:  &ChainConfig{EIP150Block: big.NewInt(10)},
			new:     &ChainConfig{EIP150Block: big.NewInt(20)},
			head:    9,
			wantErr: nil,
		},
		{
			stored: AllEthashProtocolChanges,
			new:    &ChainConfig{HomesteadBlock: nil},
			head:   3,
			wantErr: &ConfigCompatError{
				What:         "Homestead fork block",
				StoredConfig: big.NewInt(0),
				NewConfig:    nil,
				RewindTo:     0,
			},
		},
		{
			stored: AllEthashProtocolChanges,
			new:    &ChainConfig{HomesteadBlock: big.NewInt(1)},
			head:   3,
			wantErr: &ConfigCompatError{
				What:         "Homestead fork block",
				StoredConfig: big.NewInt(0),
				NewConfig:    big.NewInt(1),
				RewindTo:     0,
			},
		},
		{
			stored: &ChainConfig{HomesteadBlock: big.NewInt(30), EIP150Block: big.NewInt(10)},
			new:    &ChainConfig{HomesteadBlock: big.NewInt(25), EIP150Block: big.NewInt(20)},
			head:   25,
			wantErr: &ConfigCompatError{
				What:         "EIP150 fork block",
				StoredConfig: big.NewInt(10),
				NewConfig:    big.NewInt(20),
				RewindTo:     9,
			},
		},
		{
			stored: &ChainConfig{EIP100FBlock: big.NewInt(30), EIP649FBlock: big.NewInt(31)},
			new:    &ChainConfig{EIP100FBlock: big.NewInt(30), EIP649FBlock: big.NewInt(31)},
			head:   25,
			wantErr: &ConfigCompatError{
				What:         "EIP100F/EIP649F not equal",
				StoredConfig: big.NewInt(30),
				NewConfig:    big.NewInt(31),
				RewindTo:     29,
			},
		},
		{
			stored: &ChainConfig{EIP100FBlock: big.NewInt(30), EIP649FBlock: big.NewInt(30)},
			new:    &ChainConfig{EIP100FBlock: big.NewInt(24), EIP649FBlock: big.NewInt(24)},
			head:   25,
			wantErr: &ConfigCompatError{
				What:         "EIP100F fork block",
				StoredConfig: big.NewInt(30),
				NewConfig:    big.NewInt(24),
				RewindTo:     23,
			},
		},
		{
			stored:  &ChainConfig{ByzantiumBlock: big.NewInt(30)},
			new:     &ChainConfig{EIP211FBlock: big.NewInt(26)},
			head:    25,
			wantErr: nil,
		},
		{
			stored: &ChainConfig{ByzantiumBlock: big.NewInt(30)},
			new:    &ChainConfig{EIP100FBlock: big.NewInt(26)}, // err: EIP649 must also be set
			head:   25,
			wantErr: &ConfigCompatError{
				What:         "EIP100F/EIP649F not equal",
				StoredConfig: big.NewInt(26), // this yields a weird-looking error (correctly, though), b/c ConfigCompatError not set up for these kinds of strange cases
				NewConfig:    nil,
				RewindTo:     25,
			},
		},
		{
			stored:  &ChainConfig{ByzantiumBlock: big.NewInt(30)},
			new:     &ChainConfig{EIP100FBlock: big.NewInt(26), EIP649FBlock: big.NewInt(26)},
			head:    25,
			wantErr: nil,
		},
		{
			stored: MainnetChainConfig,
			new: func() *ChainConfig {
				c := &ChainConfig{}
				*c = *MainnetChainConfig
				c.DAOForkSupport = !MainnetChainConfig.DAOForkSupport
				return c
			}(),
			head: MainnetChainConfig.DAOForkBlock.Uint64(),
			wantErr: &ConfigCompatError{
				What:         "DAO fork support flag",
				StoredConfig: MainnetChainConfig.DAOForkBlock,
				NewConfig:    MainnetChainConfig.DAOForkBlock,
				RewindTo:     new(big.Int).Sub(MainnetChainConfig.DAOForkBlock, common.Big1).Uint64(),
			},
		},
		{
			stored: MainnetChainConfig,
			new: func() *ChainConfig {
				c := &ChainConfig{}
				*c = *MainnetChainConfig
				c.ChainID = new(big.Int).Sub(MainnetChainConfig.EIP155Block, common.Big1)
				return c
			}(),
			head: MainnetChainConfig.EIP158Block.Uint64(),
			wantErr: &ConfigCompatError{
				What:         "EIP155 chain ID",
				StoredConfig: MainnetChainConfig.EIP155Block,
				NewConfig:    MainnetChainConfig.EIP155Block,
				RewindTo:     new(big.Int).Sub(MainnetChainConfig.EIP158Block, common.Big1).Uint64(),
			},
		},
	}

	for _, test := range tests {
		err := test.stored.CheckCompatible(test.new, test.head)
		if !reflect.DeepEqual(err, test.wantErr) {
			t.Errorf("error mismatch:\nstored: %v\nnew: %v\nhead: %v\nerr: %v\nwant: %v", test.stored, test.new, test.head, err, test.wantErr)
		}
	}
}
