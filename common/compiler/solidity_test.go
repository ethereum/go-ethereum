// Copyright 2015 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// go-ethereum is free software: you can redistribute it and/or modify
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

package compiler

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const solcVersion = "0.9.23"

var (
	source = `
contract test {
   /// @notice Will multiply ` + "`a`" + ` by 7.
   function multiply(uint a) returns(uint d) {
       return a * 7;
   }
}
`
	code = "0x605880600c6000396000f3006000357c010000000000000000000000000000000000000000000000000000000090048063c6888fa114602e57005b603d6004803590602001506047565b8060005260206000f35b60006007820290506053565b91905056"
	info = `{"source":"\ncontract test {\n   /// @notice Will multiply ` + "`a`" + ` by 7.\n   function multiply(uint a) returns(uint d) {\n       return a * 7;\n   }\n}\n","language":"Solidity","languageVersion":"0","compilerVersion":"0.9.23","abiDefinition":[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}],"userDoc":{"methods":{"multiply(uint256)":{"notice":"Will multiply ` + "`a`" + ` by 7."}}},"developerDoc":{"methods":{}}}`

	infohash = common.HexToHash("0xea782f674eb898e477c20e8a7cf11c2c28b09fa68b5278732104f7a101aed255")
)

func TestCompiler(t *testing.T) {
	sol, err := New("")
	if err != nil {
		t.Skip("solc not found: skip")
	} else if sol.Version() != solcVersion {
		t.Skip("WARNING: skipping due to a newer version of solc found (%v, expect %v)", sol.Version(), solcVersion)
	}
	contracts, err := sol.Compile(source)
	if err != nil {
		t.Errorf("error compiling source. result %v: %v", contracts, err)
		return
	}

	if len(contracts) != 1 {
		t.Errorf("one contract expected, got\n%s", len(contracts))
	}

	if contracts["test"].Code != code {
		t.Errorf("wrong code, expected\n%s, got\n%s", code, contracts["test"].Code)
	}

}

func TestCompileError(t *testing.T) {
	sol, err := New("")
	if err != nil || sol.version != solcVersion {
		t.Skip("solc not found: skip")
	} else if sol.Version() != solcVersion {
		t.Skip("WARNING: skipping due to a newer version of solc found (%v, expect %v)", sol.Version(), solcVersion)
	}
	contracts, err := sol.Compile(source[2:])
	if err == nil {
		t.Errorf("error expected compiling source. got none. result %v", contracts)
		return
	}
}

func TestNoCompiler(t *testing.T) {
	_, err := New("/path/to/solc")
	if err != nil {
		t.Log("solidity quits with error: %v", err)
	} else {
		t.Errorf("no solc installed, but got no error")
	}
}

func TestSaveInfo(t *testing.T) {
	var cinfo ContractInfo
	err := json.Unmarshal([]byte(info), &cinfo)
	if err != nil {
		t.Errorf("%v", err)
	}
	filename := "/tmp/solctest.info.json"
	os.Remove(filename)
	cinfohash, err := SaveInfo(&cinfo, filename)
	if err != nil {
		t.Errorf("error extracting info: %v", err)
	}
	got, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("error reading '%v': %v", filename, err)
	}
	if string(got) != info {
		t.Errorf("incorrect info.json extracted, expected:\n%s\ngot\n%s", info, string(got))
	}
	if cinfohash != infohash {
		t.Errorf("content hash for info is incorrect. expected %v, got %v", infohash.Hex(), cinfohash.Hex())
	}
}
