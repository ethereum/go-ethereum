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
	"reflect"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/stretchr/testify/require"
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

// TestEventMultiValueWithArrayUnpack verifies that array fields will be counted after parsing array.
func TestEventMultiValueWithArrayUnpack(t *testing.T) {
	definition := `[{"name": "test", "type": "event", "inputs": [{"indexed": false, "name":"value1", "type":"uint8[2]"},{"indexed": false, "name":"value2", "type":"uint8"}]}]`
	type testStruct struct {
		Value1 [2]uint8
		Value2 uint8
	}
	abi, err := JSON(strings.NewReader(definition))
	require.NoError(t, err)
	var b bytes.Buffer
	var i uint8 = 1
	for ; i <= 3; i++ {
		b.Write(packNum(reflect.ValueOf(i)))
	}
	var rst testStruct
	require.NoError(t, abi.Unpack(&rst, "test", b.Bytes()))
	require.Equal(t, [2]uint8{1, 2}, rst.Value1)
	require.Equal(t, uint8(3), rst.Value2)
}
