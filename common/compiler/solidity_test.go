package compiler

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

var (
	source = `
contract test {
   /// @notice Will multiply ` + "`a`" + ` by 7.
   function multiply(uint a) returns(uint d) {
       return a * 7;
   }
}
`
	code = "605280600c6000396000f3006000357c010000000000000000000000000000000000000000000000000000000090048063c6888fa114602e57005b60376004356041565b8060005260206000f35b6000600782029050604d565b91905056"
	info = `{"source":"\ncontract test {\n   /// @notice Will multiply ` + "`a`" + ` by 7.\n   function multiply(uint a) returns(uint d) {\n       return a * 7;\n   }\n}\n","language":"Solidity","languageVersion":"0","compilerVersion":"0.9.13","abiDefinition":[{"constant":false,"inputs":[{"name":"a","type":"uint256"}],"name":"multiply","outputs":[{"name":"d","type":"uint256"}],"type":"function"}],"userDoc":{"methods":{"multiply(uint256)":{"notice":"Will multiply ` + "`a`" + ` by 7."}}},"developerDoc":{"methods":{}}}`

	infohash = common.HexToHash("0xfdb031637e8a1c1891143f8d129ebc7f7c4e4b41ecad8c85abe1756190f74204")
)

func TestCompiler(t *testing.T) {
	sol, err := New("")
	if err != nil {
		t.Skip("no solc installed")
	}
	contract, err := sol.Compile(source)
	if err != nil {
		t.Errorf("error compiling source. result %v: %v", contract, err)
		return
	}
	if contract.Code != code {
		t.Errorf("wrong code, expected\n%s, got\n%s", code, contract.Code)
	}
}

func TestCompileError(t *testing.T) {
	sol, err := New("")
	if err != nil {
		t.Skip("no solc installed")
	}
	contract, err := sol.Compile(source[2:])
	if err == nil {
		t.Errorf("error expected compiling source. got none. result %v", contract)
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

func TestExtractInfo(t *testing.T) {
	var cinfo ContractInfo
	err := json.Unmarshal([]byte(info), &cinfo)
	if err != nil {
		t.Errorf("%v", err)
	}
	contract := &Contract{
		Code: "",
		Info: cinfo,
	}
	filename := "/tmp/solctest.info.json"
	os.Remove(filename)
	cinfohash, err := ExtractInfo(contract, filename)
	if err != nil {
		t.Errorf("%v", err)
	}
	got, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Errorf("%v", err)
	}
	if string(got) != info {
		t.Errorf("incorrect info.json extracted, expected:\n%s\ngot\n%s", info, string(got))
	}
	if cinfohash != infohash {
		t.Errorf("content hash for info is incorrect. expected %v, got %v", infohash.Hex(), cinfohash.Hex())
	}
}
