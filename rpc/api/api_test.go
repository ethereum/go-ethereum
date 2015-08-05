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

package api

import (
	"testing"

	"encoding/json"
	"strconv"

	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/rpc/codec"
	"github.com/ethereum/go-ethereum/rpc/shared"
	"github.com/ethereum/go-ethereum/xeth"
)

func TestParseApiString(t *testing.T) {
	apis, err := ParseApiString("", codec.JSON, nil, nil)
	if err == nil {
		t.Errorf("Expected an err from parsing empty API string but got nil")
	}

	if len(apis) != 0 {
		t.Errorf("Expected 0 apis from empty API string")
	}

	apis, err = ParseApiString("eth", codec.JSON, nil, nil)
	if err != nil {
		t.Errorf("Expected nil err from parsing empty API string but got %v", err)
	}

	if len(apis) != 1 {
		t.Errorf("Expected 1 apis but got %d - %v", apis, apis)
	}

	apis, err = ParseApiString("eth,eth", codec.JSON, nil, nil)
	if err != nil {
		t.Errorf("Expected nil err from parsing empty API string but got \"%v\"", err)
	}

	if len(apis) != 2 {
		t.Errorf("Expected 2 apis but got %d - %v", apis, apis)
	}

	apis, err = ParseApiString("eth,invalid", codec.JSON, nil, nil)
	if err == nil {
		t.Errorf("Expected an err but got no err")
	}

}

const solcVersion = "0.9.23"

func TestCompileSolidity(t *testing.T) {

	solc, err := compiler.New("")
	if solc == nil {
		t.Skip("no solc found: skip")
	} else if solc.Version() != solcVersion {
		t.Skip("WARNING: skipping test because of solc different version (%v, test written for %v, may need to update)", solc.Version(), solcVersion)
	}
	source := `contract test {\n` +
		"   /// @notice Will multiply `a` by 7." + `\n` +
		`   function multiply(uint a) returns(uint d) {\n` +
		`       return a * 7;\n` +
		`   }\n` +
		`}\n`

	jsonstr := `{"jsonrpc":"2.0","method":"eth_compileSolidity","params":["` + source + `"],"id":64}`

	expCode := "0x605880600c6000396000f3006000357c010000000000000000000000000000000000000000000000000000000090048063c6888fa114602e57005b603d6004803590602001506047565b8060005260206000f35b60006007820290506053565b91905056"
	expAbiDefinition := `[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}]`
	expUserDoc := `{"methods":{"multiply(uint256)":{"notice":"Will multiply ` + "`a`" + ` by 7."}}}`
	expDeveloperDoc := `{"methods":{}}`
	expCompilerVersion := solc.Version()
	expLanguage := "Solidity"
	expLanguageVersion := "0"
	expSource := source

	eth := &eth.Ethereum{}
	xeth := xeth.NewTest(eth, nil)
	api := NewEthApi(xeth, eth, codec.JSON)

	var rpcRequest shared.Request
	json.Unmarshal([]byte(jsonstr), &rpcRequest)

	response, err := api.CompileSolidity(&rpcRequest)
	if err != nil {
		t.Errorf("Execution failed, %v", err)
	}

	respjson, err := json.Marshal(response)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	var contracts = make(map[string]*compiler.Contract)
	err = json.Unmarshal(respjson, &contracts)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if len(contracts) != 1 {
		t.Errorf("expected one contract, got %v", len(contracts))
	}

	contract := contracts["test"]

	if contract.Code != expCode {
		t.Errorf("Expected \n%s got \n%s", expCode, contract.Code)
	}

	if strconv.Quote(contract.Info.Source) != `"`+expSource+`"` {
		t.Errorf("Expected \n'%s' got \n'%s'", expSource, strconv.Quote(contract.Info.Source))
	}

	if contract.Info.Language != expLanguage {
		t.Errorf("Expected %s got %s", expLanguage, contract.Info.Language)
	}

	if contract.Info.LanguageVersion != expLanguageVersion {
		t.Errorf("Expected %s got %s", expLanguageVersion, contract.Info.LanguageVersion)
	}

	if contract.Info.CompilerVersion != expCompilerVersion {
		t.Errorf("Expected %s got %s", expCompilerVersion, contract.Info.CompilerVersion)
	}

	userdoc, err := json.Marshal(contract.Info.UserDoc)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	devdoc, err := json.Marshal(contract.Info.DeveloperDoc)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	abidef, err := json.Marshal(contract.Info.AbiDefinition)
	if err != nil {
		t.Errorf("expected no error, got %v", err)
	}

	if string(abidef) != expAbiDefinition {
		t.Errorf("Expected \n'%s' got \n'%s'", expAbiDefinition, string(abidef))
	}

	if string(userdoc) != expUserDoc {
		t.Errorf("Expected \n'%s' got \n'%s'", expUserDoc, string(userdoc))
	}

	if string(devdoc) != expDeveloperDoc {
		t.Errorf("Expected %s got %s", expDeveloperDoc, string(devdoc))
	}
}
