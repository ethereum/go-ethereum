// Copyright 2024 The go-ethereum Authors
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

package ethapi

import (
	"encoding/json"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/internal/ethapi/override"
)

func TestSimulateSanitizeBlockOrder(t *testing.T) {
	type result struct {
		number    uint64
		timestamp uint64
	}
	for i, tc := range []struct {
		baseNumber    int
		baseTimestamp uint64
		blocks        []simBlock
		expected      []result
		err           string
	}{
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{}, {}, {}},
			expected:      []result{{number: 11, timestamp: 62}, {number: 12, timestamp: 74}, {number: 13, timestamp: 86}},
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &override.BlockOverrides{Number: newInt(13), Time: newUint64(80)}}, {}},
			expected:      []result{{number: 11, timestamp: 62}, {number: 12, timestamp: 74}, {number: 13, timestamp: 80}, {number: 14, timestamp: 92}},
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &override.BlockOverrides{Number: newInt(11)}}, {BlockOverrides: &override.BlockOverrides{Number: newInt(14)}}, {}},
			expected:      []result{{number: 11, timestamp: 62}, {number: 12, timestamp: 74}, {number: 13, timestamp: 86}, {number: 14, timestamp: 98}, {number: 15, timestamp: 110}},
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &override.BlockOverrides{Number: newInt(13)}}, {BlockOverrides: &override.BlockOverrides{Number: newInt(12)}}},
			err:           "block numbers must be in order: 12 <= 13",
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &override.BlockOverrides{Number: newInt(13), Time: newUint64(74)}}},
			err:           "block timestamps must be in order: 74 <= 74",
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &override.BlockOverrides{Number: newInt(11), Time: newUint64(60)}}, {BlockOverrides: &override.BlockOverrides{Number: newInt(12), Time: newUint64(55)}}},
			err:           "block timestamps must be in order: 55 <= 60",
		},
		{
			baseNumber:    10,
			baseTimestamp: 50,
			blocks:        []simBlock{{BlockOverrides: &override.BlockOverrides{Number: newInt(11), Time: newUint64(60)}}, {BlockOverrides: &override.BlockOverrides{Number: newInt(13), Time: newUint64(72)}}},
			err:           "block timestamps must be in order: 72 <= 72",
		},
	} {
		sim := &simulator{
			base:   &types.Header{Number: big.NewInt(int64(tc.baseNumber)), Time: tc.baseTimestamp},
			budget: newGasBudget(0),
		}
		res, err := sim.sanitizeChain(tc.blocks)
		if err != nil {
			if err.Error() == tc.err {
				continue
			} else {
				t.Fatalf("testcase %d: error mismatch. Want '%s', have '%s'", i, tc.err, err.Error())
			}
		}
		if err == nil && tc.err != "" {
			t.Fatalf("testcase %d: expected err", i)
		}
		if len(res) != len(tc.expected) {
			t.Errorf("testcase %d: mismatch number of blocks. Want %d, have %d", i, len(tc.expected), len(res))
		}
		for bi, b := range res {
			if b.BlockOverrides == nil {
				t.Fatalf("testcase %d: block overrides nil", i)
			}
			if b.BlockOverrides.Number == nil {
				t.Fatalf("testcase %d: block number not set", i)
			}
			if b.BlockOverrides.Time == nil {
				t.Fatalf("testcase %d: block time not set", i)
			}
			if uint64(*b.BlockOverrides.Time) != tc.expected[bi].timestamp {
				t.Errorf("testcase %d: block timestamp mismatch. Want %d, have %d", i, tc.expected[bi].timestamp, uint64(*b.BlockOverrides.Time))
			}
			have := b.BlockOverrides.Number.ToInt().Uint64()
			if have != tc.expected[bi].number {
				t.Errorf("testcase %d: block number mismatch. Want %d, have %d", i, tc.expected[bi].number, have)
			}
		}
	}
}

func newInt(n int64) *hexutil.Big {
	return (*hexutil.Big)(big.NewInt(n))
}

func TestSimBlockUnmarshalOverrideKeys(t *testing.T) {
	for _, tc := range []struct {
		input string
		err   string
	}{
		{input: `{"blockOverrides":{"number":"0x1","beaconRoot":"0x0000000000000000000000000000000000000000000000000000000000000001"},"stateOverrides":{"0xc000000000000000000000000000000000000000":{"balance":"0x1","movePrecompileToAddress":"0xc100000000000000000000000000000000000000"}}}`},
		{input: `{"blockOverrides":null,"stateOverrides":null}`},
		{input: `{"blockOverrides":{"bogusField":"0x1"}}`, err: `unknown field "bogusField" in block override`},
		{input: `{"blockOverrides":{"Number":"0x1"}}`},
		{input: `{"blockOverrides":{"number":"0x1","Number":"0x2"}}`, err: `keys "Number" and "number" in block override name the same field`},
		{input: `{"blockOverrides":{},"blockOverrides":{"Bogus":"0x1"}}`, err: `unknown field "Bogus" in block override`},
		{input: `{"stateOverrides":{"0xc000000000000000000000000000000000000000":{"bogusField":"0x1"}}}`, err: `unknown field "bogusField" in state override for 0xC000000000000000000000000000000000000000`},
		{input: `{"stateOverrides":{"0xc000000000000000000000000000000000000000":{"Code":"0x00"}}}`},
		{input: `{"stateOverrides":{"0xc000000000000000000000000000000000000000":{"code":"0x00","Code":"0x01"}}}`, err: `keys "Code" and "code" in state override for 0xC000000000000000000000000000000000000000 name the same field`},
	} {
		var b simBlock
		err := json.Unmarshal([]byte(tc.input), &b)
		switch {
		case tc.err == "" && err != nil:
			t.Errorf("%s: unexpected error: %v", tc.input, err)
		case tc.err != "" && (err == nil || err.Error() != tc.err):
			t.Errorf("%s: error mismatch: have %v, want %s", tc.input, err, tc.err)
		}
	}
}
