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
	"encoding/json"
	"math/big"
	"reflect"
	"strings"
	"testing"
)

// pbtRulesBase is a chain with every block-numbered fork at genesis, so the
// cases below only have to add the timestamp forks they care about.
func pbtRulesBase() *ChainConfig {
	return &ChainConfig{
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
		// Cancun and Prague carry blobs, so CheckConfigForkOrder demands a
		// schedule entry for each once their timestamps are set below.
		BlobScheduleConfig: &BlobScheduleConfig{
			Cancun: DefaultCancunBlobConfig,
			Prague: DefaultPragueBlobConfig,
		},
	}
}

func u64ptr(n uint64) *uint64 { return &n }

// TestPBTChangesNoExecutionRule is the load-bearing check that EIP-8297 is a
// state-commitment change and nothing more.
//
// It compares the whole Rules struct rather than named flags on purpose. Geth
// used to derive two execution rules from the binary tree - it activated
// EIP-4762's witness gas and switched EIP-2929 warm/cold access off - so PBT
// silently repriced execution. Neither has any basis in the EIP: the reference
// implementation's binary_tree fork is the underlying fork's VM verbatim.
//
// The tree is not represented in Rules at all, so the two structs must be
// exactly equal - there is no field the commitment is permitted to move, and no
// exception to carve out. A future fork wiring anything into PBT fails here
// rather than being discovered as a consensus split.
func TestPBTChangesNoExecutionRule(t *testing.T) {
	// The forks the binary tree can sit on. It requires Amsterdam, so those are
	// Amsterdam and everything after it; a new fork belongs here.
	for _, tc := range []struct {
		name  string
		apply func(*ChainConfig)
	}{
		{"amsterdam", func(c *ChainConfig) {
			c.ShanghaiTime, c.CancunTime, c.PragueTime = u64ptr(0), u64ptr(0), u64ptr(0)
			c.OsakaTime, c.AmsterdamTime = u64ptr(0), u64ptr(0)
		}},
		{"bogota", func(c *ChainConfig) {
			c.ShanghaiTime, c.CancunTime, c.PragueTime = u64ptr(0), u64ptr(0), u64ptr(0)
			c.OsakaTime, c.AmsterdamTime, c.BogotaTime = u64ptr(0), u64ptr(0), u64ptr(0)
		}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			plainCfg := pbtRulesBase()
			tc.apply(plainCfg)
			treeCfg := *plainCfg
			treeCfg.BinaryTrieTime = u64ptr(0)

			// Both configurations must be valid: the tree is optional on these
			// forks, so the plain one has to stay legal.
			if err := plainCfg.CheckConfigForkOrder(); err != nil {
				t.Fatalf("the merkle-patricia configuration is rejected: %v", err)
			}
			if err := treeCfg.CheckConfigForkOrder(); err != nil {
				t.Fatalf("the binary tree configuration is rejected: %v", err)
			}

			num, time := big.NewInt(1), uint64(1)
			plain := plainCfg.Rules(num, true, time)
			tree := treeCfg.Rules(num, true, time)

			if !treeCfg.IsPBT() {
				t.Fatal("the binary tree is not active; this case proves nothing")
			}
			if plainCfg.IsPBT() {
				t.Fatal("the binary tree is active without being enabled; the control is wrong")
			}
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
	cfg := pbtRulesBase()
	cfg.ShanghaiTime, cfg.CancunTime, cfg.PragueTime = u64ptr(0), u64ptr(0), u64ptr(0)
	cfg.OsakaTime, cfg.AmsterdamTime = u64ptr(0), u64ptr(0)
	cfg.BinaryTrieTime = u64ptr(0)

	if !cfg.IsPBT() {
		t.Fatal("the binary tree is not active; this proves nothing")
	}
	if rules := cfg.Rules(big.NewInt(1), true, 1); !rules.IsEIP2929 {
		t.Fatal("EIP-2929 is off under the binary tree, leaving no access pricing at all")
	}
}

// TestPBTRequiresAmsterdam pins the ordering rules the binary tree fork has:
// it is only defined from Amsterdam onwards, so it cannot be scheduled on a
// chain that never reaches Amsterdam, nor before Amsterdam's own time.
func TestPBTRequiresAmsterdam(t *testing.T) {
	withoutAmsterdam := pbtRulesBase()
	withoutAmsterdam.ShanghaiTime, withoutAmsterdam.CancunTime = u64ptr(0), u64ptr(0)
	withoutAmsterdam.PragueTime, withoutAmsterdam.OsakaTime = u64ptr(0), u64ptr(0)
	withoutAmsterdam.BinaryTrieTime = u64ptr(0)

	if err := withoutAmsterdam.CheckConfigForkOrder(); err == nil {
		t.Fatal("the binary tree was accepted on a chain that never schedules Amsterdam")
	}

	// The same chain is fine without the tree - Osaka on the merkle-patricia
	// trie is an ordinary configuration, and the new rule must not reject it.
	withoutAmsterdam.BinaryTrieTime = nil
	if err := withoutAmsterdam.CheckConfigForkOrder(); err != nil {
		t.Fatalf("a merkle-patricia chain stopping at Osaka is rejected: %v", err)
	}

	// With Amsterdam scheduled, the tree may activate with it or after it, but
	// never before it.
	withAmsterdam := pbtRulesBase()
	withAmsterdam.ShanghaiTime, withAmsterdam.CancunTime = u64ptr(0), u64ptr(0)
	withAmsterdam.PragueTime, withAmsterdam.OsakaTime = u64ptr(0), u64ptr(0)
	withAmsterdam.AmsterdamTime = u64ptr(100)

	for _, tc := range []struct {
		name string
		time *uint64
		ok   bool
	}{
		{"before amsterdam", u64ptr(99), false},
		{"at amsterdam", u64ptr(100), true},
		{"after amsterdam", u64ptr(101), true},
		{"unscheduled", nil, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cfg := *withAmsterdam
			cfg.BinaryTrieTime = tc.time
			err := cfg.CheckConfigForkOrder()
			if tc.ok && err != nil {
				t.Fatalf("a legal schedule is rejected: %v", err)
			}
			if !tc.ok && err == nil {
				t.Fatal("a binary tree scheduled before Amsterdam was accepted")
			}
		})
	}

	// Bogota does not bound the tree: folding it into the ordered fork list
	// would reject this legal schedule.
	trailing := *withAmsterdam
	trailing.BogotaTime = u64ptr(150)
	trailing.BinaryTrieTime = u64ptr(200)
	if err := trailing.CheckConfigForkOrder(); err != nil {
		t.Fatalf("bogota bounded the binary tree: %v", err)
	}
}

// TestBinaryTrieTimeJSONKey pins the genesis key the fork is configured with.
// The key is shared with besu, which reads it case-insensitively as
// "binarytrietime", so both spellings have to decode; the retired "pbt"
// boolean must be ignored.
func TestBinaryTrieTimeJSONKey(t *testing.T) {
	cfg := pbtRulesBase()
	cfg.BinaryTrieTime = u64ptr(0)
	blob, err := json.Marshal(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(blob), `"binaryTrieTime":0`) {
		t.Fatalf("the config does not serialize the binaryTrieTime key: %s", blob)
	}
	if strings.Contains(string(blob), `"pbt"`) {
		t.Fatalf("the retired pbt key is serialized: %s", blob)
	}

	for _, tc := range []struct {
		name string
		blob string
		want *uint64
	}{
		{"geth spelling", `{"binaryTrieTime":7}`, u64ptr(7)},
		{"besu spelling", `{"binarytrietime":7}`, u64ptr(7)},
		{"absent", `{}`, nil},
		{"retired pbt boolean", `{"pbt":true}`, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var decoded ChainConfig
			if err := json.Unmarshal([]byte(tc.blob), &decoded); err != nil {
				t.Fatal(err)
			}
			switch {
			case tc.want == nil && decoded.BinaryTrieTime != nil:
				t.Fatalf("binaryTrieTime decoded as %d, want unset", *decoded.BinaryTrieTime)
			case tc.want != nil && (decoded.BinaryTrieTime == nil || *decoded.BinaryTrieTime != *tc.want):
				t.Fatalf("binaryTrieTime decoded as %v, want %d", decoded.BinaryTrieTime, *tc.want)
			}
		})
	}
}
