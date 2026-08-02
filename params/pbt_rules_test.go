// Copyright 2026 The go-ethereum Authors
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
)

// TestPBTChangesNoExecutionRule is the load-bearing check that EIP-8297 is a
// state-commitment change and nothing more.
//
// It compares the whole Rules struct rather than named flags on purpose. Geth
// used to derive two execution rules from the binary tree - it activated
// EIP-4762's witness gas and switched EIP-2929 warm/cold access off - so PBT
// silently repriced execution. Neither has any basis in the EIP: the reference
// implementation's binary_tree fork is the underlying fork's VM verbatim.
//
// Comparing field-by-field means a future fork wiring anything new into PBT
// fails here rather than being discovered as a consensus split.
func TestPBTChangesNoExecutionRule(t *testing.T) {
	u64 := func(n uint64) *uint64 { return &n }

	base := &ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		TerminalTotalDifficulty: big.NewInt(0),
	}
	// Every fork the binary tree can sit on. A new fork belongs here.
	for _, tc := range []struct {
		name  string
		apply func(*ChainConfig)
	}{
		{"shanghai", func(c *ChainConfig) { c.ShanghaiTime = u64(0) }},
		{"cancun", func(c *ChainConfig) { c.ShanghaiTime, c.CancunTime = u64(0), u64(0) }},
		{"prague", func(c *ChainConfig) { c.ShanghaiTime, c.CancunTime, c.PragueTime = u64(0), u64(0), u64(0) }},
		{"osaka", func(c *ChainConfig) {
			c.ShanghaiTime, c.CancunTime, c.PragueTime, c.OsakaTime = u64(0), u64(0), u64(0), u64(0)
		}},
		{"amsterdam", func(c *ChainConfig) {
			c.ShanghaiTime, c.CancunTime, c.PragueTime = u64(0), u64(0), u64(0)
			c.OsakaTime, c.AmsterdamTime = u64(0), u64(0)
		}},
		{"bogota", func(c *ChainConfig) {
			c.ShanghaiTime, c.CancunTime, c.PragueTime = u64(0), u64(0), u64(0)
			c.OsakaTime, c.AmsterdamTime, c.BogotaTime = u64(0), u64(0), u64(0)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plainCfg := *base
			tc.apply(&plainCfg)
			treeCfg := plainCfg
			treeCfg.PBTTime = u64(0)

			num, time := big.NewInt(1), uint64(1)
			plain := plainCfg.Rules(num, true, time)
			tree := treeCfg.Rules(num, true, time)

			if !tree.IsPBT {
				t.Fatal("the binary tree is not active; this case proves nothing")
			}
			if plain.IsPBT {
				t.Fatal("the binary tree is active without PBTTime; the control is wrong")
			}
			// IsPBT is the one field allowed to differ - it selects the state
			// commitment, which is the whole of what the EIP changes.
			tree.IsPBT = false
			if !reflect.DeepEqual(plain, tree) {
				t.Fatalf("the binary tree changed execution rules\n plain: %+v\n  tree: %+v", plain, tree)
			}
		})
	}
}

// TestPBTKeepsEIP2929 calls out the specific rule that was wrong, so the reason
// survives even if the struct-wide comparison above is ever loosened.
//
// Berlin onwards has EIP-2929 warm/cold access pricing. Geth disabled it under
// the binary tree because EIP-4762 replaced it; with 4762 gone, dropping 2929
// too would leave PBT with no access pricing at all.
func TestPBTKeepsEIP2929(t *testing.T) {
	u64 := func(n uint64) *uint64 { return &n }
	cfg := &ChainConfig{
		ChainID:                 big.NewInt(1),
		HomesteadBlock:          big.NewInt(0),
		EIP150Block:             big.NewInt(0),
		EIP155Block:             big.NewInt(0),
		EIP158Block:             big.NewInt(0),
		ByzantiumBlock:          big.NewInt(0),
		ConstantinopleBlock:     big.NewInt(0),
		PetersburgBlock:         big.NewInt(0),
		IstanbulBlock:           big.NewInt(0),
		BerlinBlock:             big.NewInt(0),
		LondonBlock:             big.NewInt(0),
		MergeNetsplitBlock:      big.NewInt(0),
		TerminalTotalDifficulty: big.NewInt(0),
		ShanghaiTime:            u64(0),
		CancunTime:              u64(0),
		PragueTime:              u64(0),
		OsakaTime:               u64(0),
		AmsterdamTime:           u64(0),
		PBTTime:                 u64(0),
	}
	rules := cfg.Rules(big.NewInt(1), true, 1)
	if !rules.IsPBT {
		t.Fatal("the binary tree is not active; this proves nothing")
	}
	if !rules.IsEIP2929 {
		t.Fatal("EIP-2929 is off under the binary tree, leaving no access pricing at all")
	}
}
