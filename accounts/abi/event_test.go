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

package abi

import (
	"bytes"
	"math/big"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/math"
	"github.com/ethereum/go-ethereum/crypto"
)

func TestEventId(t *testing.T) {
	var table = []struct {
		definition   string
		expectations map[string]common.Hash
	}{
		{
			definition: `[
			{ "type" : "event", "name" : "balance", "inputs": [{ "name" : "in", "type": "uint256" }] },
			{ "type" : "event", "name" : "check", "inputs": [{ "name" : "t", "type": "address" }, { "name": "b", "type": "uint256" }] }
			]`,
			expectations: map[string]common.Hash{
				"balance": crypto.Keccak256Hash([]byte("balance(uint256)")),
				"check":   crypto.Keccak256Hash([]byte("check(address,uint256)")),
			},
		},
	}

	for _, test := range table {
		abi, err := JSON(strings.NewReader(test.definition))
		if err != nil {
			t.Fatal(err)
		}

		for name, event := range abi.Events {
			if event.Id() != test.expectations[name] {
				t.Errorf("expected id to be %x, got %x", test.expectations[name], event.Id())
			}
		}
	}
}

type testResult struct {
	Value1 *big.Int
	Value2 *big.Int
}
type testCase struct {
	definition string
	want       testResult
}

func (tc testCase) encoded() []byte {
	var b bytes.Buffer
	if tc.want.Value1 != nil {
		b.Write(math.PaddedBigBytes(math.U256(tc.want.Value1), 32))
	}
	if tc.want.Value2 != nil {
		b.Write(math.PaddedBigBytes(math.U256(tc.want.Value2), 32))
	}
	return b.Bytes()
}

func TestEventUnpack(t *testing.T) {

	table := []testCase{
		{
			definition: `[{"anonymous":false,"inputs":[{"indexed":true,"name":"value1","type":"uint256"},{"indexed":false,"name":"value2","type":"uint256"}],"name":"transfer","type":"event"}]`,
			want:       testResult{Value2: big.NewInt(10)},
		},
		{
			definition: `[{"anonymous":false,"inputs":[{"indexed":false,"name":"value1","type":"uint256"},{"indexed":false,"name":"value2","type":"uint256"}],"name":"transfer","type":"event"}]`,
			want:       testResult{Value1: big.NewInt(100), Value2: big.NewInt(1)},
		},
		{
			definition: `[{"anonymous":false,"inputs":[{"indexed":false,"name":"value1","type":"uint256"},{"indexed":true,"name":"value2","type":"uint256"}],"name":"transfer","type":"event"}]`,
			want:       testResult{Value1: big.NewInt(100)},
		},
	}
	for i, row := range table {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			t.Logf("unpacking %b with expected %v", row.encoded(), row.want)
			abi, err := JSON(strings.NewReader(row.definition))
			if err != nil {
				t.Fatal(err)
			}
			var rst testResult
			if err := abi.Unpack(&rst, "transfer", row.encoded()); err != nil {
				t.Fatalf("error unpacking %s: %v", row.definition, err)
			}
			if row.want.Value1 != nil && rst.Value1.Cmp(row.want.Value1) != 0 {
				t.Errorf("result value1 %v is not equal to expected %v", rst.Value1, row.want.Value1)
			}

			if row.want.Value2 != nil && rst.Value2.Cmp(row.want.Value2) != 0 {
				t.Errorf("result value2 %v is not equal to expected %v", rst.Value2, row.want.Value2)
			}
		})
	}
}
