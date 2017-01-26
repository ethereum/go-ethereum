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
	"os/exec"
	"path"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const (
	testSource = `
contract test {
   /// @notice Will multiply ` + "`a`" + ` by 7.
   function multiply(uint a) returns(uint d) {
       return a * 7;
   }
}
`
	testInfo = `{"source":"\ncontract test {\n   /// @notice Will multiply ` + "`a`" + ` by 7.\n   function multiply(uint a) returns(uint d) {\n       return a * 7;\n   }\n}\n","language":"Solidity","languageVersion":"0.1.1","compilerVersion":"0.1.1","compilerOptions":"--binary file --json-abi file --add-std 1","abiDefinition":[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}],"userDoc":{"methods":{"multiply(uint256)":{"notice":"Will multiply ` + "`a`" + ` by 7."}}},"developerDoc":{"methods":{}}}`
)

func skipWithoutSolc(t *testing.T) {
	if _, err := exec.LookPath("solc"); err != nil {
		t.Skip(err)
	}
}

func TestCompiler(t *testing.T) {
	skipWithoutSolc(t)

	contracts, err := CompileSolidityString("", testSource)
	if err != nil {
		t.Fatalf("error compiling source. result %v: %v", contracts, err)
	}
	if len(contracts) != 1 {
		t.Errorf("one contract expected, got %d", len(contracts))
	}
	c, ok := contracts["test"]
	if !ok {
		t.Fatal("info for contract 'test' not present in result")
	}
	if c.Code == "" {
		t.Error("empty code")
	}
	if c.Info.Source != testSource {
		t.Error("wrong source")
	}
	if c.Info.CompilerVersion == "" {
		t.Error("empty version")
	}
}

func TestCompileError(t *testing.T) {
	skipWithoutSolc(t)

	contracts, err := CompileSolidityString("", testSource[4:])
	if err == nil {
		t.Errorf("error expected compiling source. got none. result %v", contracts)
	}
	t.Logf("error: %v", err)
}

func TestSaveInfo(t *testing.T) {
	var cinfo ContractInfo
	err := json.Unmarshal([]byte(testInfo), &cinfo)
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
	if string(got) != testInfo {
		t.Errorf("incorrect info.json extracted, expected:\n%s\ngot\n%s", testInfo, string(got))
	}
	wantHash := common.HexToHash("0x22450a77f0c3ff7a395948d07bc1456881226a1b6325f4189cb5f1254a824080")
	if cinfohash != wantHash {
		t.Errorf("content hash for info is incorrect. expected %v, got %v", wantHash.Hex(), cinfohash.Hex())
	}
}
