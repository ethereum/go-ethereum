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
	"reflect"
	"testing"

	"github.com/holiman/uint256"
)

func TestPoSWConfig(t *testing.T) {
	var config *PoSWConfig
	if err := config.Check(); err != nil {
		t.Fatal(err)
	}
	config = &PoSWConfig{}
	if err := config.Check(); err != nil {
		t.Fatalf("disabled PoSW: %v", err)
	}
	config = &PoSWConfig{
		Enabled: true, WindowSize: 256, StakePerBlock: uint256.MustFromDecimal("1000000000000000000000000000000"),
		DiffDivider: 20,
	}
	if err := config.Check(); err != nil {
		t.Fatalf("valid PoSW: %v", err)
	}
	chainConfig := &ChainConfig{QuarkChain: &QuarkChainConfig{PoSW: config}}
	encoded, err := json.Marshal(chainConfig)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ChainConfig
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(chainConfig, &decoded) {
		t.Fatalf("configuration changed on JSON round trip: %s", encoded)
	}
}

func TestPoSWConfigValidation(t *testing.T) {
	valid := func() *PoSWConfig {
		return &PoSWConfig{Enabled: true, WindowSize: 256, StakePerBlock: uint256.NewInt(100), DiffDivider: 20}
	}
	for _, test := range []struct {
		name   string
		change func(*PoSWConfig)
	}{
		{"zero window", func(c *PoSWConfig) { c.WindowSize = 0 }},
		{"zero divider", func(c *PoSWConfig) { c.DiffDivider = 0 }},
		{"missing stake", func(c *PoSWConfig) { c.StakePerBlock = nil }},
		{"zero stake", func(c *PoSWConfig) { c.StakePerBlock = new(uint256.Int) }},
	} {
		t.Run(test.name, func(t *testing.T) {
			config := valid()
			test.change(config)
			if err := config.Check(); err == nil {
				t.Fatal("invalid PoSW config accepted")
			}
			if err := (&ChainConfig{QuarkChain: &QuarkChainConfig{PoSW: config}}).CheckConfigForkOrder(); err == nil {
				t.Fatal("invalid shard PoSW config accepted")
			}
		})
	}
}

func TestPoSWStakeJSON(t *testing.T) {
	const maxStake = "115792089237316195423570985008687907853269984665640564039457584007913129639935"
	for _, test := range []struct {
		input   string
		want    string
		wantErr bool
	}{
		{"100", "100", false},
		{`"100"`, "100", false},
		{`"0x64"`, "100", false},
		{maxStake, maxStake, false},
		{"-1", "", true},
		{`"-1"`, "", true},
		{"115792089237316195423570985008687907853269984665640564039457584007913129639936", "", true},
	} {
		var config PoSWConfig
		err := json.Unmarshal([]byte(`{"stakePerBlock":`+test.input+`}`), &config)
		if (err != nil) != test.wantErr {
			t.Fatalf("decode %s: have %v, want error %v", test.input, err, test.wantErr)
		}
		if test.wantErr {
			continue
		}
		if config.StakePerBlock == nil || config.StakePerBlock.Dec() != test.want {
			t.Fatalf("decoded stake: have %v, want %s", config.StakePerBlock, test.want)
		}
		encoded, err := json.Marshal(config)
		if err != nil {
			t.Fatal(err)
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(encoded, &fields); err != nil {
			t.Fatal(err)
		}
		if string(fields["stakePerBlock"]) != `"`+test.want+`"` {
			t.Fatalf("stake was not encoded as a decimal string: %s", encoded)
		}
	}
}
