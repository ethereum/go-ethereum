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
	"fmt"
	"math/big"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
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
	Values [2]*big.Int
	Value1 *big.Int
	Value2 *big.Int
}

type testCase struct {
	definition string
	want       testResult
}

func (tc testCase) encoded(intType, arrayType Type) []byte {
	var b bytes.Buffer
	if tc.want.Value1 != nil {
		val, _ := intType.pack(reflect.ValueOf(tc.want.Value1))
		b.Write(val)
	}

	if !reflect.DeepEqual(tc.want.Values, [2]*big.Int{nil, nil}) {
		val, _ := arrayType.pack(reflect.ValueOf(tc.want.Values))
		b.Write(val)
	}
	if tc.want.Value2 != nil {
		val, _ := intType.pack(reflect.ValueOf(tc.want.Value2))
		b.Write(val)
	}
	return b.Bytes()
}

func TestEventUnpack(t *testing.T) {
	intType, _ := NewType("uint256")
	arrayType, _ := NewType("uint256[2]")
	definitionTemplate := `[{"anonymous":false,"inputs":[
{"indexed":%t,"name":"value1","type":"%s"},
{"indexed":%t,"name":"values","type":"%s"},
{"indexed":%t,"name":"value2","type":"%s"}],
"name":"test","type":"event"}]`
	table := []testCase{
		{
			// value1 is indexed
			definition: fmt.Sprintf(definitionTemplate, true, intType, false, arrayType, false, intType),
			want:       testResult{Value2: big.NewInt(10), Values: [2]*big.Int{big.NewInt(10), big.NewInt(11)}},
		},
		{
			// only values field (array) is indexed
			definition: fmt.Sprintf(definitionTemplate, false, intType, true, arrayType, false, intType),
			want:       testResult{Value1: big.NewInt(100), Value2: big.NewInt(1)},
		},
		{
			// values and value2 are indexed
			definition: fmt.Sprintf(definitionTemplate, false, intType, true, arrayType, true, intType),
			want:       testResult{Value1: big.NewInt(100)},
		},
		{
			// value1 and values are not indexed
			definition: fmt.Sprintf(definitionTemplate, false, intType, false, arrayType, true, intType),
			want:       testResult{Value1: big.NewInt(10), Values: [2]*big.Int{big.NewInt(10), big.NewInt(11)}},
		},
	}
	for i, row := range table {
		t.Run(strconv.Itoa(i+1), func(t *testing.T) {
			encoded := row.encoded(intType, arrayType)
			t.Logf("unpacking definition %s, expected %v", row.definition, row.want)
			abi, err := JSON(strings.NewReader(row.definition))
			if err != nil {
				t.Fatal(err)
			}
			var rst testResult
			if err := abi.Unpack(&rst, "test", encoded); err != nil {
				t.Fatalf("error unpacking %s: %v", row.definition, err)
			}
			if row.want.Value1 != nil && rst.Value1.Cmp(row.want.Value1) != 0 {
				t.Errorf("result value1 %v is not equal to expected %v", rst.Value1, row.want.Value1)
			}
			if row.want.Value2 != nil && rst.Value2.Cmp(row.want.Value2) != 0 {
				t.Errorf("result value2 %v is not equal to expected %v", rst.Value2, row.want.Value2)
			}
			if len(row.want.Values) != len(rst.Values) {
				t.Errorf("result values %v are not equal to expected %v", rst.Values, row.want.Values)
			} else {
				for i, val := range rst.Values {
					if exp := row.want.Values[i]; exp != nil && exp.Cmp(val) != 0 {
						t.Errorf("value %d: %v is not equal to expected %v", i, val, exp)
					}
				}
			}
		})
	}
}
