// Copyright 2017 The go-ethereum Authors
// This file is part of go-ethereum.
//
// go-ethereum is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-ethereum is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// along with go-ethereum. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"fmt"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/common"
)

func TestCalldataDecoding(t *testing.T) {

	// send(uint256)                              : a52c101e
	// compareAndApprove(address,uint256,uint256) : 751e1079
	// issue(address[],uint256)                   : 42958b54
	jsondata := `
[
	{"type":"function","name":"send","inputs":[{"name":"a","type":"uint256"}]},
	{"type":"function","name":"compareAndApprove","inputs":[{"name":"a","type":"address"},{"name":"a","type":"uint256"},{"name":"a","type":"uint256"}]},
	{"type":"function","name":"issue","inputs":[{"name":"a","type":"address[]"},{"name":"a","type":"uint256"}]},
	{"type":"function","name":"sam","inputs":[{"name":"a","type":"bytes"},{"name":"a","type":"bool"},{"name":"a","type":"uint256[]"}]}
]`
	//Expected failures
	for _, hexdata := range []string{
		"a52c101e00000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000042",
		"a52c101e000000000000000000000000000000000000000000000000000000000000001200",
		"a52c101e00000000000000000000000000000000000000000000000000000000000000",
		"a52c101e",
		"a52c10",
		"",
		// Too short
		"751e10790000000000000000000000000000000000000000000000000000000000000012",
		"751e1079FFffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		//Not valid multiple of 32
		"deadbeef00000000000000000000000000000000000000000000000000000000000000",
		//Too short 'issue'
		"42958b5400000000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000042",
		// Too short compareAndApprove
		"a52c101e00ff0000000000000000000000000000000000000000000000000000000000120000000000000000000000000000000000000000000000000000000000000042",
		// From https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI
		// contains a bool with illegal values
		"a5643bf20000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000001100000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000464617665000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003",
	} {
		_, err := parseCallData(common.Hex2Bytes(hexdata), jsondata)
		if err == nil {
			t.Errorf("Expected decoding to fail: %s", hexdata)
		}
	}

	//Expected success
	for _, hexdata := range []string{
		// From https://github.com/ethereum/wiki/wiki/Ethereum-Contract-ABI
		"a5643bf20000000000000000000000000000000000000000000000000000000000000060000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000a0000000000000000000000000000000000000000000000000000000000000000464617665000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000003000000000000000000000000000000000000000000000000000000000000000100000000000000000000000000000000000000000000000000000000000000020000000000000000000000000000000000000000000000000000000000000003",
		"a52c101e0000000000000000000000000000000000000000000000000000000000000012",
		"a52c101eFFffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff",
		"751e1079000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000000",
		"42958b54" +
			// start of dynamic type
			"0000000000000000000000000000000000000000000000000000000000000040" +
			//uint256
			"0000000000000000000000000000000000000000000000000000000000000001" +
			// length of  array
			"0000000000000000000000000000000000000000000000000000000000000002" +
			// array values
			"000000000000000000000000000000000000000000000000000000000000dead" +
			"000000000000000000000000000000000000000000000000000000000000beef",
	} {
		_, err := parseCallData(common.Hex2Bytes(hexdata), jsondata)
		if err != nil {
			t.Errorf("Unexpected failure on input %s:\n %v (%d bytes) ", hexdata, err, len(common.Hex2Bytes(hexdata)))
		}
	}
}

func TestSelectorUnmarshalling(t *testing.T) {
	var (
		db        *abiDb
		err       error
		abistring []byte
		abistruct abi.ABI
	)

	db, err = NewAbiDBFromFile("4byte.json")
	if err != nil {
		t.Fatal(err)
	}
	fmt.Printf("DB size %v\n", db.Size())
	for id, selector := range db.db {

		abistring, err = MethodSelectorToAbi(selector)
		if err != nil {
			t.Error(err)
			return
		}
		abistruct, err = abi.JSON(strings.NewReader(string(abistring)))
		if err != nil {
			t.Error(err)
			return
		}
		m := abistruct.MethodById(common.Hex2Bytes(id[2:]))
		if m.Sig() != selector {
			t.Errorf("Expected equality: %v != %v", m.Sig(), selector)
		}
	}

}
