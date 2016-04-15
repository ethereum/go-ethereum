// Copyright 2015 The go-ethereum Authors
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

package compiler

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const solcVersion = "0.1.1"

var (
	source = `
contract test {
   /// @notice Will multiply ` + "`a`" + ` by 7.
   function multiply(uint a) returns(uint d) {
       return a * 7;
   }
}
`
	code = "0x6060604052606d8060116000396000f30060606040526000357c010000000000000000000000000000000000000000000000000000000090048063c6888fa1146037576035565b005b6046600480359060200150605c565b6040518082815260200191505060405180910390f35b60006007820290506068565b91905056"
	info = `{"source":"\ncontract test {\n   /// @notice Will multiply ` + "`a`" + ` by 7.\n   function multiply(uint a) returns(uint d) {\n       return a * 7;\n   }\n}\n","language":"Solidity","languageVersion":"0.1.1","compilerVersion":"0.1.1","compilerOptions":"--binary file --json-abi file --natspec-user file --natspec-dev file --add-std 1","abiDefinition":[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}],"userDoc":{"methods":{"multiply(uint256)":{"notice":"Will multiply ` + "`a`" + ` by 7."}}},"developerDoc":{"methods":{}}}`

	infohash = common.HexToHash("0x9f3803735e7f16120c5a140ab3f02121fd3533a9655c69b33a10e78752cc49b0")
)

func TestCompiler(t *testing.T) {
	sol, err := New("")
	if err != nil {
		t.Skipf("solc not found: %v", err)
	} else if sol.Version() != solcVersion {
		t.Skipf("WARNING: a newer version of solc found (%v, expect %v)", sol.Version(), solcVersion)
	}
	contracts, err := sol.Compile(source)
	if err != nil {
		t.Errorf("error compiling source. result %v: %v", contracts, err)
		return
	}

	if len(contracts) != 1 {
		t.Errorf("one contract expected, got %d", len(contracts))
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
		t.Logf("solidity quits with error: %v", err)
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
	filename := path.Join(os.TempDir(), "solctest.info.json")
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
