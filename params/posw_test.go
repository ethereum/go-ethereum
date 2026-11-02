// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package params

import (
	"encoding/json"
	"math/big"
	"reflect"
	"testing"
)

func validPoSWConfig() *PoSWConfig {
	return &PoSWConfig{
		AdjustmentCutoff: 7, AdjustmentFactor: 512, MinimumDifficulty: big.NewInt(1),
		CoinbaseAmount: big.NewInt(100), EpochInterval: 10, DecayFactor: big.NewRat(22, 25),
		StakingEnabled: true, WindowSize: 256, StakePerBlock: big.NewInt(100), DiffDivider: 20,
	}
}

func validQuarkChainConfig() *QuarkChainConfig {
	return &QuarkChainConfig{
		ChainIDs:              []*big.Int{big.NewInt(10001)},
		PoSW:                  validPoSWConfig(),
		RewardTaxRate:         big.NewRat(1, 2),
		MaxBlocksPerRootBlock: 100,
	}
}

func TestPoSWConfigEqual(t *testing.T) {
	for _, test := range []struct {
		name string
		a, b *PoSWConfig
		want bool
	}{
		{"nil", nil, nil, true},
		{"nil versus empty", nil, &PoSWConfig{}, false},
		{"empty", &PoSWConfig{}, &PoSWConfig{}, true},
		{"independent configs", validPoSWConfig(), validPoSWConfig(), true},
		{"missing fields", &PoSWConfig{}, validPoSWConfig(), false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if test.a.Equal(test.b) != test.want || test.b.Equal(test.a) != test.want {
				t.Fatalf("Equal returned the wrong result, want %v", test.want)
			}
		})
	}
}

func TestPoSWConfigEqualFields(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*PoSWConfig)
	}{
		{"adjustmentCutoff", func(c *PoSWConfig) { c.AdjustmentCutoff++ }},
		{"adjustmentFactor", func(c *PoSWConfig) { c.AdjustmentFactor++ }},
		{"minimumDifficulty", func(c *PoSWConfig) { c.MinimumDifficulty.SetInt64(2) }},
		{"coinbaseAmount", func(c *PoSWConfig) { c.CoinbaseAmount.SetInt64(200) }},
		{"epochInterval", func(c *PoSWConfig) { c.EpochInterval++ }},
		{"decayFactor", func(c *PoSWConfig) { c.DecayFactor.SetFrac64(1, 2) }},
		{"originalBlockNumber", func(c *PoSWConfig) { c.OriginalBlockNumber++ }},
		{"stakingEnabled", func(c *PoSWConfig) { c.StakingEnabled = false }},
		{"windowSize", func(c *PoSWConfig) { c.WindowSize++ }},
		{"diffDivider", func(c *PoSWConfig) { c.DiffDivider++ }},
		{"stakePerBlock", func(c *PoSWConfig) { c.StakePerBlock.SetInt64(200) }},
		{"poswStakingDecayEnabled", func(c *PoSWConfig) { c.PoSWStakingDecayEnabled = true }},
	} {
		t.Run(test.name, func(t *testing.T) {
			stored, next := validPoSWConfig(), validPoSWConfig()
			test.change(next)
			if stored.Equal(next) || next.Equal(stored) {
				t.Fatal("Equal ignored the changed field")
			}
		})
	}
}

func TestNewTestQuarkChainConfigIsolation(t *testing.T) {
	chainIDs := []int64{10001, 10002}
	first := NewTestQuarkChainConfig(10002, chainIDs...).QuarkChain
	second := NewTestQuarkChainConfig(10002, chainIDs...).QuarkChain
	if err := second.PoSW.Check(); err != nil {
		t.Fatal(err)
	}
	first.PoSW.MinimumDifficulty.SetInt64(9)
	first.PoSW.CoinbaseAmount.SetInt64(9)
	first.PoSW.DecayFactor.SetInt64(0)
	first.RewardTaxRate.SetInt64(0)
	first.ChainIDs[0].SetInt64(9)
	chainIDs[1] = 9
	if second.PoSW.MinimumDifficulty.Int64() != 1 || second.PoSW.CoinbaseAmount.Sign() != 0 ||
		second.PoSW.DecayFactor.Cmp(big.NewRat(22, 25)) != 0 || second.RewardTaxRate.Cmp(big.NewRat(1, 2)) != 0 ||
		!reflect.DeepEqual(second.ChainIDs, []*big.Int{big.NewInt(10001), big.NewInt(10002)}) || first.ChainIDs[1].Int64() != 10002 {
		t.Fatal("test configurations share mutable defaults or the caller's chain IDs")
	}
}

func TestPoSWConfigJSON(t *testing.T) {
	for _, test := range []struct {
		name  string
		input string
		valid bool
	}{
		{
			"numbers",
			`{"posw": {
				"adjustmentCutoff": 7,
				"adjustmentFactor": 512,
				"minimumDifficulty": 100,
				"coinbaseAmount": 6500000000000000000,
				"epochInterval": 3153600,
				"decayFactor": 0.88,
				"originalBlockNumber": 12345678,
				"stakingEnabled": true,
				"windowSize": 512,
				"stakePerBlock": 1000000000000000000000000,
				"poswStakingDecayEnabled": true,
				"diffDivider": 10000
			}}`,
			true,
		},
		{
			"quoted ratios",
			`{"posw": {
				"adjustmentCutoff": 7,
				"adjustmentFactor": 512,
				"minimumDifficulty": 100,
				"coinbaseAmount": 6500000000000000000,
				"epochInterval": 3153600,
				"decayFactor": "0.88",
				"originalBlockNumber": 12345678,
				"stakingEnabled": true,
				"windowSize": 512,
				"stakePerBlock": 1000000000000000000000000,
				"poswStakingDecayEnabled": true,
				"diffDivider": 10000
			}}`,
			true,
		},
		{
			"quoted",
			`{"posw": {
				"adjustmentCutoff": "7",
				"adjustmentFactor": "512",
				"minimumDifficulty": "100",
				"coinbaseAmount": "6500000000000000000",
				"epochInterval": "3153600",
				"decayFactor": "0.88",
				"originalBlockNumber": "12345678",
				"stakingEnabled": true,
				"windowSize": "512",
				"stakePerBlock": "1000000000000000000000000",
				"poswStakingDecayEnabled": true,
				"diffDivider": "10000"
			}}`,
			true,
		},
		{
			"hex strings",
			`{"posw": {
				"adjustmentCutoff": "0x07",
				"adjustmentFactor": "0x200",
				"minimumDifficulty": "0x64",
				"coinbaseAmount": "0x5a34a38fc00a0000",
				"epochInterval": "0x301ec0",
				"decayFactor": 0.88,
				"originalBlockNumber": "0xbc614e",
				"stakingEnabled": true,
				"windowSize": "0x200",
				"stakePerBlock": "0xd3c21bcecceda1000000",
				"poswStakingDecayEnabled": true,
				"diffDivider": "0x2710"
			}}`,
			true,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			var config QuarkChainConfig
			err := json.Unmarshal([]byte(test.input), &config)
			if err == nil {
				err = config.PoSW.Check()
			}
			if (err == nil) != test.valid {
				t.Fatalf("config validation %v, want valid %v", err, test.valid)
			}
			if err != nil {
				return
			}
			if config.PoSW.CoinbaseAmount.String() != "6500000000000000000" || config.PoSW.StakePerBlock.String() != "1000000000000000000000000" {
				t.Fatal("mainnet integer fields changed value")
			}
			encoded, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			var fields struct {
				PoSW map[string]json.RawMessage `json:"posw"`
			}
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			wantFields := map[string]string{
				"adjustmentCutoff": "7", "adjustmentFactor": "512",
				"minimumDifficulty": "100", "coinbaseAmount": "6500000000000000000",
				"epochInterval": "3153600", "decayFactor": "0.88",
				"originalBlockNumber": "12345678", "stakingEnabled": "true", "windowSize": "512",
				"stakePerBlock": "1000000000000000000000000", "poswStakingDecayEnabled": "true", "diffDivider": "10000",
			}
			if len(fields.PoSW) != len(wantFields) {
				t.Fatalf("unexpected JSON fields: %s", encoded)
			}
			for field, want := range wantFields {
				if string(fields.PoSW[field]) != want {
					t.Fatalf("%s: have %s, want JSON value %s", field, fields.PoSW[field], want)
				}
			}
			var roundtrip QuarkChainConfig
			if err := json.Unmarshal(encoded, &roundtrip); err != nil || !reflect.DeepEqual(config, roundtrip) {
				t.Fatalf("config JSON round trip: %s, %v", encoded, err)
			}
		})
	}
}

func TestPoSWConfigUint64JSON(t *testing.T) {
	for _, field := range []string{"adjustmentCutoff", "adjustmentFactor", "epochInterval", "originalBlockNumber", "windowSize", "diffDivider"} {
		for _, test := range []struct {
			input string
			valid bool
		}{
			{`7`, true}, {`"7"`, true}, {`"0x07"`, true},
			{`18446744073709551615`, true}, {`"0xffffffffffffffff"`, true},
			{`18446744073709551616`, false}, {`"0x10000000000000000"`, false},
			{`-1`, false}, {`"-1"`, false}, {`7.5`, false}, {`"7.5"`, false},
		} {
			config := validPoSWConfig()
			err := json.Unmarshal([]byte(`{"`+field+`":`+test.input+`}`), config)
			if (err == nil) != test.valid {
				t.Fatalf("%s=%s: decode %v, want valid %v", field, test.input, err, test.valid)
			}
		}
	}
}

func TestPoSWConfigJSONUpdates(t *testing.T) {
	config := validPoSWConfig()
	config.OriginalBlockNumber = 19
	config.PoSWStakingDecayEnabled = true
	if err := json.Unmarshal([]byte(`{"originalBlockNumber":0,"stakingEnabled":false,"poswStakingDecayEnabled":false}`), config); err != nil {
		t.Fatal(err)
	}
	if config.OriginalBlockNumber != 0 || config.StakingEnabled || config.PoSWStakingDecayEnabled || config.AdjustmentCutoff != 7 || config.CoinbaseAmount.Int64() != 100 {
		t.Fatal("partial decode must apply zero and false values and preserve omitted fields")
	}
	for _, field := range []string{"stakingEnabled", "poswStakingDecayEnabled"} {
		if err := json.Unmarshal([]byte(`{"`+field+`":"true"}`), config); err == nil {
			t.Fatalf("%s must reject quoted booleans", field)
		}
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(encoded, &fields); err != nil {
		t.Fatal(err)
	}
	if _, exists := fields["originalBlockNumber"]; exists {
		t.Fatal("zero originalBlockNumber must be omitted")
	}
}

func TestPoSWConfigRatiosJSON(t *testing.T) {
	for _, field := range []string{"decayFactor"} {
		for _, test := range []struct {
			input string
			valid bool
		}{
			{`0.88`, true}, {`"0.88"`, true}, {`"22/25"`, false},
			{`8.8e-1`, true}, {`"8.8e-1"`, true}, {`"bad"`, false},
			{`0.001`, false}, {`"0.001"`, false},
			// Values above one must not round down through float64.
			{`1.000000000000000000000000000000000001`, false},
			{`"1.000000000000000000000000000000000001"`, false},
		} {
			config := validPoSWConfig()
			err := json.Unmarshal([]byte(`{"`+field+`":`+test.input+`}`), config)
			if err == nil {
				err = config.Check()
			}
			if (err == nil) != test.valid {
				t.Fatalf("%s=%s: validation %v, want valid %v", field, test.input, err, test.valid)
			}
			if err != nil {
				continue
			}
			encoded, err := json.Marshal(config)
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields[field]) != "0.88" {
				t.Fatalf("%s=%s: have %s, want unquoted 0.88", field, test.input, fields[field])
			}
		}
	}
}

func TestPoSWConfigRatioMarshal(t *testing.T) {
	for _, field := range []string{"decayFactor"} {
		for _, test := range []struct {
			ratio *big.Rat
			want  string
		}{
			{big.NewRat(22, 25), "0.88"},
			{big.NewRat(1, 8), "0.125"},
			{big.NewRat(0, 1), "0"},
			{big.NewRat(1, 1), "1"},
			{nil, "null"},
			{big.NewRat(1, 3), ""}, // Never silently round a repeating decimal.
		} {
			config := validPoSWConfig()
			config.DecayFactor = test.ratio
			encoded, err := json.Marshal(config)
			if test.want == "" {
				if err == nil {
					t.Fatalf("%s: repeating decimal must fail marshaling", field)
				}
				continue
			}
			if err != nil {
				t.Fatal(err)
			}
			var fields map[string]json.RawMessage
			if err := json.Unmarshal(encoded, &fields); err != nil {
				t.Fatal(err)
			}
			if string(fields[field]) != test.want {
				t.Fatalf("%s: have %s, want %s", field, fields[field], test.want)
			}
		}
	}
}

func TestPoSWConfigValidation(t *testing.T) {
	for _, test := range []struct {
		name   string
		change func(*PoSWConfig)
	}{
		{"zero cutoff", func(c *PoSWConfig) { c.AdjustmentCutoff = 0 }},
		{"zero factor", func(c *PoSWConfig) { c.AdjustmentFactor = 0 }},
		{"missing minimum", func(c *PoSWConfig) { c.MinimumDifficulty = nil }},
		{"zero minimum", func(c *PoSWConfig) { c.MinimumDifficulty = new(big.Int) }},
		{"negative minimum", func(c *PoSWConfig) { c.MinimumDifficulty.SetInt64(-1) }},
		{"minimum overflow", func(c *PoSWConfig) { c.MinimumDifficulty.Lsh(big.NewInt(1), 256) }},
		{"missing subsidy", func(c *PoSWConfig) { c.CoinbaseAmount = nil }},
		{"negative subsidy", func(c *PoSWConfig) { c.CoinbaseAmount.SetInt64(-1) }},
		{"subsidy overflow", func(c *PoSWConfig) { c.CoinbaseAmount.Lsh(big.NewInt(1), 256) }},
		{"zero epoch", func(c *PoSWConfig) { c.EpochInterval = 0 }},
		{"missing decay", func(c *PoSWConfig) { c.DecayFactor = nil }},
		{"negative decay", func(c *PoSWConfig) { c.DecayFactor = big.NewRat(-1, 2) }},
		{"decay above one", func(c *PoSWConfig) { c.DecayFactor = big.NewRat(3, 2) }},
		{"decay denominator", func(c *PoSWConfig) { c.DecayFactor = big.NewRat(1, 1000) }},
		{"repeating decay", func(c *PoSWConfig) { c.DecayFactor = big.NewRat(1, 3) }},
		{"zero window", func(c *PoSWConfig) { c.WindowSize = 0 }},
		{"zero divider", func(c *PoSWConfig) { c.DiffDivider = 0 }},
		{"missing stake", func(c *PoSWConfig) { c.StakePerBlock = nil }},
		{"zero stake", func(c *PoSWConfig) { c.StakePerBlock.SetUint64(0) }},
		{"negative stake", func(c *PoSWConfig) { c.StakePerBlock.SetInt64(-1) }},
		{"stake overflow", func(c *PoSWConfig) { c.StakePerBlock.Lsh(big.NewInt(1), 256) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := validPoSWConfig()
			test.change(config)
			if err := config.Check(); err == nil {
				t.Fatal("invalid consensus config accepted")
			}
			if err := (&ChainConfig{QuarkChain: &QuarkChainConfig{PoSW: config, RewardTaxRate: big.NewRat(1, 2)}}).CheckConfigForkOrder(); err == nil {
				t.Fatal("invalid shard consensus config accepted")
			}
		})
	}
	for _, config := range []*PoSWConfig{nil, {}} {
		chainConfig := *MergedTestChainConfig
		chainConfig.QuarkChain = &QuarkChainConfig{PoSW: config, RewardTaxRate: big.NewRat(1, 2)}
		if config.Check() == nil || chainConfig.CheckConfigForkOrder() == nil {
			t.Fatal("missing consensus parameters accepted")
		}
	}
	for _, factor := range []*big.Rat{big.NewRat(0, 1), big.NewRat(1, 8), big.NewRat(1, 2), big.NewRat(22, 25), big.NewRat(99, 100), big.NewRat(1, 1)} {
		config := validPoSWConfig()
		config.DecayFactor = factor
		config.MinimumDifficulty.Sub(new(big.Int).Lsh(big.NewInt(1), 256), big.NewInt(1))
		config.CoinbaseAmount.SetUint64(0)
		config.StakingEnabled = false
		config.StakePerBlock, config.WindowSize, config.DiffDivider = nil, 0, 0
		if err := config.Check(); err != nil {
			t.Fatalf("zero subsidy, staking disabled, decay %s: %v", factor, err)
		}
		encoded, err := json.Marshal(config)
		if err != nil {
			t.Fatal(err)
		}
		var roundtrip PoSWConfig
		if err := json.Unmarshal(encoded, &roundtrip); err != nil {
			t.Fatal(err)
		}
		if err := roundtrip.Check(); err != nil {
			t.Fatal(err)
		}
		reencoded, err := json.Marshal(&roundtrip)
		if err != nil {
			t.Fatal(err)
		}
		if string(encoded) != string(reencoded) {
			t.Fatalf("validated config changed during JSON round trip: %s, have %s", encoded, reencoded)
		}
	}
}

func TestPoSWConfigAmountsJSON(t *testing.T) {
	const maxAmount = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
	for _, field := range []string{"minimumDifficulty", "coinbaseAmount", "stakePerBlock"} {
		for _, test := range []struct {
			input string
			valid bool
		}{
			{"100", true}, {maxAmount, true}, {`"100"`, true}, {`"0x64"`, true},
			{`"0xffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff"`, true},
			{"0.88", false}, {`"0.88"`, false}, {"-1", false}, {`"-1"`, false},
			{"115792089237316195423570985008687907853269984665640564039457584007913129639936", false},
		} {
			config := validPoSWConfig()
			err := json.Unmarshal([]byte(`{"`+field+`":`+test.input+`}`), config)
			if err == nil {
				err = config.Check()
			}
			if (err == nil) != test.valid {
				t.Fatalf("%s=%s: validation %v, want valid %v", field, test.input, err, test.valid)
			}
		}
	}
}

func TestPoSWConfigHeightQueries(t *testing.T) {
	for _, test := range []struct{ number, reward, stake uint64 }{
		{0, 0, 50}, {1, 25, 25}, {11, 12, 12}, {71, 0, 100},
	} {
		config := validPoSWConfig()
		config.DecayFactor = big.NewRat(1, 2)
		config.OriginalBlockNumber = 19
		config.PoSWStakingDecayEnabled = true
		before, _ := json.Marshal(config)
		number := new(big.Int).SetUint64(test.number)
		reward, stake := config.BlockRewardAt(number), config.StakePerBlockAt(number)
		if reward.Uint64() != test.reward || stake.Uint64() != test.stake {
			t.Fatalf("height %d: reward %s stake %s, want %d/%d", test.number, reward, stake, test.reward, test.stake)
		}
		reward.SetInt64(-1)
		stake.SetInt64(-1)
		after, _ := json.Marshal(config)
		if string(before) != string(after) || number.Uint64() != test.number {
			t.Fatal("height queries or their returned integers modified inputs")
		}
		config.PoSWStakingDecayEnabled = false
		fixed := config.StakePerBlockAt(number)
		fixed.SetInt64(0)
		if config.StakePerBlock.Uint64() != 100 || config.StakePerBlockAt(number).Uint64() != 100 {
			t.Fatal("fixed stake query modified or decayed the configured value")
		}
	}
	config := validPoSWConfig()
	config.DecayFactor = big.NewRat(1, 2)
	config.EpochInterval, config.OriginalBlockNumber = ^uint64(0), ^uint64(0)
	if got := config.BlockRewardAt(big.NewInt(1)); got.Uint64() != 50 {
		t.Fatalf("original height overflow: reward %s, want 50", got)
	}
}

func TestRegenesisRewards(t *testing.T) {
	for _, test := range []struct {
		number, offset uint64
		want           uint64
	}{
		{0, 19, 0}, {1, 18, 50}, {1, 19, 25}, {10, 19, 25}, {11, 19, 12}, {71, 19, 0},
	} {
		config := &PoSWConfig{
			CoinbaseAmount: big.NewInt(100), EpochInterval: 10, DecayFactor: big.NewRat(1, 2),
			OriginalBlockNumber: test.offset,
		}
		if got := config.BlockRewardAt(new(big.Int).SetUint64(test.number)); got.Cmp(new(big.Int).SetUint64(test.want)) != 0 {
			t.Fatalf("height %d offset %d: %s want %d", test.number, test.offset, got, test.want)
		}
	}
}

func TestRewardDecay(t *testing.T) {
	// Expected rewards follow pyquarkchain: floor the epoch decay first,
	// then multiply by the remaining reward share and floor again.
	for _, test := range []struct {
		name   string
		factor string
		tax    string
		amount uint64
		epoch  uint64
		want   uint64
	}{
		{"before decay", "0.88", "0.5", 100, 0, 50},
		{"first epoch", "0.88", "0.5", 100, 1, 44},
		{"second epoch", "0.88", "0.5", 100, 2, 38},
		{"round once after exponentiation", "0.88", "0.5", 100, 3, 34},
		{"slow decay", "0.88", "0.5", 100, 6, 23},
		{"decay before tax", "0.88", "0.25", 2, 1, 0},
		{"denominator limit", "0.99", "0.5", 100, 100, 18},
		{"fully decayed", "0.88", "0.5", 100, 100, 0},
		{"no decay", "1", "0.5", 100, 100, 50},
		{"zero factor before decay", "0", "0.5", 100, 0, 50},
		{"zero factor after decay", "0", "0.5", 100, 1, 0},
		{"zero subsidy", "0.88", "0.5", 0, 0, 0},
	} {
		t.Run(test.name, func(t *testing.T) {
			factor, ok := new(big.Rat).SetString(test.factor)
			if !ok {
				t.Fatal("invalid test factor", test.factor)
			}
			tax, ok := new(big.Rat).SetString(test.tax)
			if !ok {
				t.Fatal("invalid test tax rate", test.tax)
			}
			config := &QuarkChainConfig{
				ChainIDs:              []*big.Int{big.NewInt(10001)},
				RewardTaxRate:         new(big.Rat).Set(tax),
				MaxBlocksPerRootBlock: 100,
				PoSW: &PoSWConfig{
					AdjustmentCutoff: 7, AdjustmentFactor: 512, MinimumDifficulty: big.NewInt(1),
					CoinbaseAmount: new(big.Int).SetUint64(test.amount), EpochInterval: 1,
					DecayFactor:         new(big.Rat).Set(factor),
					OriginalBlockNumber: test.epoch,
				},
			}
			if err := config.Check(); err != nil {
				t.Fatal(err)
			}
			// A positive new block number with epoch zero is needed to exercise
			// the subsidy before its first decay without crediting genesis.
			if test.epoch == 0 {
				config.PoSW.EpochInterval = 2
			} else {
				config.PoSW.OriginalBlockNumber--
			}
			if got := config.AfterTax(config.PoSW.BlockRewardAt(big.NewInt(1))); got.Cmp(new(big.Int).SetUint64(test.want)) != 0 {
				t.Fatalf("reward %s, want %d", got, test.want)
			}
			if config.PoSW.DecayFactor.Cmp(factor) != 0 || config.RewardTaxRate.Cmp(tax) != 0 {
				t.Fatal("reward calculation modified the configured decay factor or tax rate")
			}
		})
	}
}

// TestAfterTax covers Python's floor of the retained share, including amounts
// whose intermediate product exceeds uint256. Inputs must remain unchanged.
func TestAfterTax(t *testing.T) {
	for _, test := range []struct {
		name   string
		tax    *big.Rat
		amount *big.Int
		want   *big.Int
	}{
		{"no tax", big.NewRat(0, 1), big.NewInt(3), big.NewInt(3)},
		{"full tax", big.NewRat(1, 1), big.NewInt(3), new(big.Int)},
		{"zero fee", big.NewRat(1, 2), new(big.Int), new(big.Int)},
		{"odd fee", big.NewRat(1, 2), big.NewInt(3), big.NewInt(1)},
		{"round retained share", big.NewRat(1, 2), big.NewInt(1), new(big.Int)},
		{"fractional rate", big.NewRat(1, 8), big.NewInt(7), big.NewInt(6)},
		{"large fee", big.NewRat(1, 4), new(big.Int).Lsh(big.NewInt(1), 255), new(big.Int).Lsh(big.NewInt(3), 253)},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := &QuarkChainConfig{RewardTaxRate: new(big.Rat).Set(test.tax)}
			amount := new(big.Int).Set(test.amount)
			if got := config.AfterTax(amount); got.Cmp(test.want) != 0 {
				t.Fatalf("after tax %s, want %s", got, test.want)
			}
			if amount.Cmp(test.amount) != 0 || config.RewardTaxRate.Cmp(test.tax) != 0 {
				t.Fatal("tax calculation modified the amount or tax rate")
			}
		})
	}
}
