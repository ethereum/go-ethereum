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
	"math"
	"math/big"
	"reflect"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
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
			stored:        &ChainConfig{ShanghaiTime: newUint64(10)},
			new:           &ChainConfig{ShanghaiTime: newUint64(20)},
			headTimestamp: 9,
			wantErr:       nil,
		},
		{
			stored:        &ChainConfig{ShanghaiTime: newUint64(10)},
			new:           &ChainConfig{ShanghaiTime: newUint64(20)},
			headTimestamp: 25,
			wantErr: &ConfigCompatError{
				What:         "Shanghai fork timestamp",
				StoredTime:   newUint64(10),
				NewTime:      newUint64(20),
				RewindToTime: 9,
			},
		},
	}

	for _, test := range tests {
		err := test.stored.CheckCompatible(test.new, test.headBlock, test.headTimestamp)
		if !reflect.DeepEqual(err, test.wantErr) {
			t.Errorf("error mismatch:\nstored: %v\nnew: %v\nheadBlock: %v\nheadTimestamp: %v\nerr: %v\nwant: %v", test.stored, test.new, test.headBlock, test.headTimestamp, err, test.wantErr)
		}
	}
}

func TestQuarkChainConfigEqual(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*QuarkChainConfig)
		want   bool
	}{
		{"unchanged", func(c *QuarkChainConfig) {}, true},
		{"shardId", func(c *QuarkChainConfig) { c.ShardID = 1 }, false},
		{"shardIds", func(c *QuarkChainConfig) { c.ShardIDs[1] = 2 }, false},
		{"posw", func(c *QuarkChainConfig) { c.PoSW.CoinbaseAmount.SetInt64(100) }, false},
		{"missing posw", func(c *QuarkChainConfig) { c.PoSW = nil }, false},
		{"rewardTaxRate", func(c *QuarkChainConfig) { c.RewardTaxRate.SetFrac64(1, 4) }, false},
		{"missing rewardTaxRate", func(c *QuarkChainConfig) { c.RewardTaxRate = nil }, false},
		{"maxBlocksPerRootBlock", func(c *QuarkChainConfig) { c.MaxBlocksPerRootBlock = 12 }, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			stored, next := NewTestQuarkChainConfig(0, 0, 1), NewTestQuarkChainConfig(0, 0, 1)
			test.change(next)
			if stored.Equal(next) != test.want || next.Equal(stored) != test.want {
				t.Fatalf("Equal returned the wrong result, want %v", test.want)
			}
		})
	}
	var missing *QuarkChainConfig
	if !missing.Equal(nil) || missing.Equal(&QuarkChainConfig{}) || (&QuarkChainConfig{}).Equal(missing) {
		t.Fatal("Equal did not distinguish nil and non-nil configurations")
	}
	// Values, rather than pointer identity or rational representation, determine equality.
	stored, next := validQuarkChainConfig(), validQuarkChainConfig()
	stored.RewardTaxRate = new(big.Rat)
	next.RewardTaxRate, _ = new(big.Rat).SetString("0.00")
	if !stored.Equal(next) {
		t.Fatal("equal consensus values compared unequal")
	}
}

// TestQuarkChainConfigTaxValidation checks the tax at initialization alongside
// PoSW parameters; a missing rate must not silently become zero tax.
func TestQuarkChainConfigTaxValidation(t *testing.T) {
	for _, test := range []struct {
		name string
		tax  *big.Rat
	}{
		{"missing", nil},
		{"negative", big.NewRat(-1, 2)},
		{"above one", big.NewRat(3, 2)},
		{"denominator", big.NewRat(1, 1000)},
		{"repeating decimal", big.NewRat(1, 3)},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := validQuarkChainConfig()
			config.RewardTaxRate = test.tax
			if config.Check() == nil || (&ChainConfig{QuarkChain: config}).CheckConfigForkOrder() == nil {
				t.Fatal("invalid tax accepted during initialization")
			}
		})
	}
	for _, tax := range []*big.Rat{new(big.Rat), big.NewRat(1, 8), big.NewRat(1, 2), big.NewRat(99, 100), big.NewRat(1, 1)} {
		config := validQuarkChainConfig()
		config.RewardTaxRate = tax
		if err := config.Check(); err != nil {
			t.Fatalf("valid tax %s rejected: %v", tax, err)
		}
	}
}

func TestConfigRules(t *testing.T) {
	c := &ChainConfig{
		LondonBlock:  new(big.Int),
		ShanghaiTime: newUint64(500),
	}
	var stamp uint64
	if r := c.Rules(big.NewInt(0), true, stamp); r.IsShanghai {
		t.Errorf("expected %v to not be shanghai", stamp)
	}
	stamp = 500
	if r := c.Rules(big.NewInt(0), true, stamp); !r.IsShanghai {
		t.Errorf("expected %v to be shanghai", stamp)
	}
	stamp = math.MaxInt64
	if r := c.Rules(big.NewInt(0), true, stamp); !r.IsShanghai {
		t.Errorf("expected %v to be shanghai", stamp)
	}
}

func TestTimestampCompatError(t *testing.T) {
	require.Equal(t, new(ConfigCompatError).Error(), "")

	errWhat := "Shanghai fork timestamp"
	require.Equal(t, newTimestampCompatError(errWhat, nil, newUint64(1681338455)).Error(),
		"mismatching Shanghai fork timestamp in database (have timestamp nil, want timestamp 1681338455, rewindto timestamp 1681338454)")

	require.Equal(t, newTimestampCompatError(errWhat, newUint64(1681338455), nil).Error(),
		"mismatching Shanghai fork timestamp in database (have timestamp 1681338455, want timestamp nil, rewindto timestamp 1681338454)")

	require.Equal(t, newTimestampCompatError(errWhat, newUint64(1681338455), newUint64(600624000)).Error(),
		"mismatching Shanghai fork timestamp in database (have timestamp 1681338455, want timestamp 600624000, rewindto timestamp 600623999)")

	require.Equal(t, newTimestampCompatError(errWhat, newUint64(0), newUint64(1681338455)).Error(),
		"mismatching Shanghai fork timestamp in database (have timestamp 0, want timestamp 1681338455, rewindto timestamp 0)")
}
