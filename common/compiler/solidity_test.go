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
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const solFile = `pragma solidity >= 0.0.0;
	contract main {
		uint a;
		function f() {
			a = 1;
		}
	}`

const faultySol = `pragma solidity >= 0.0.0;
	contract main {
		uint a;
		function f() {
			a = 1;
		}
	`
const solNoPragma = `contract main {
		uint a;
		function f() {
			a = 1;
		}
	}`
const simplyLibrarySol = `pragma solidity >=0.0.0;

library Set {
  struct Data { mapping(uint => bool) flags; }
  function insert(Data storage self, uint value)
      returns (bool)
  {
      if (self.flags[value])
          return false; // already there
      self.flags[value] = true;
      return true;
  }

  function remove(Data storage self, uint value)
      returns (bool)
  {
      if (!self.flags[value])
          return false; // not there
      self.flags[value] = false;
      return true;
  }

  function contains(Data storage self, uint value)
      returns (bool)
  {
      return self.flags[value];
  }
}

contract C {
    Set.Data knownValues;
    function register(uint value) {
        if (!Set.insert(knownValues, value))
            throw;
    }
}`

const solFile1 = `pragma solidity >=0.0.0;

library Set {
  struct Data { mapping(uint => bool) flags; }
  function insert(Data storage self, uint value)
      returns (bool)
  {
      if (self.flags[value])
          return false; // already there
      self.flags[value] = true;
      return true;
  }

  function remove(Data storage self, uint value)
      returns (bool)
  {
      if (!self.flags[value])
          return false; // not there
      self.flags[value] = false;
      return true;
  }

  function contains(Data storage self, uint value)
      returns (bool)
  {
      return self.flags[value];
  }
}`

const solFile2 = `pragma solidity >=0.0.0;
import "set.sol";

contract C {
    Set.Data knownValues;
    function register(uint value) {
        if (!Set.insert(knownValues, value))
            throw;
    }
}`

func TestSolcCompilerNormal(t *testing.T) {
	solc, err := InitCompiler("solc")
	if err != nil {
		t.Skip(err)
	}
	solc = solc.(*Solidity)
	file, err := os.Create("simpleContract.sol")
	defer os.Remove("simpleContract.sol")
	if err != nil {
		t.Fatal(err)
	}
	file.WriteString(solFile)
	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi"},
	}

	solReturn, err := solc.Compile([]string{"simpleContract.sol"}, FlagOpts{SolcFlagOpts: flags})
	if err != nil {
		t.Fatal(err)
	}

	if solReturn.Error != nil || solReturn.Warning != "" || len(solReturn.Contracts) != 1 {
		t.Fatalf("Expected no errors or warnings and expected contract items. Got %v for errors, %v for warnings, and %v for contract items", solReturn.Error, solReturn.Warning, solReturn.Contracts)
	}
}

func TestSolcCompilerError(t *testing.T) {
	solc, err := InitCompiler("solc")
	if err != nil {
		t.Skip(err)
	}
	solc = solc.(*Solidity)
	file, err := os.Create("faultyContract.sol")
	defer os.Remove("faultyContract.sol")
	if err != nil {
		t.Fatal(err)
	}
	file.WriteString(faultySol)
	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi"},
	}

	solReturn, err := solc.Compile([]string{"faultyContract.sol"}, FlagOpts{SolcFlagOpts: flags})
	if err != nil {
		t.Fatal(err)
	}
	if solReturn.Error == nil {
		t.Fatal("Expected an error, got nil.")
	}
}

func TestSolcCompilerWarning(t *testing.T) {
	solc, err := InitCompiler("solc")
	if err != nil {
		t.Skip(err)
	}
	solc = solc.(*Solidity)
	file, err := os.Create("simpleContract.sol")
	defer os.Remove("simpleContract.sol")
	if err != nil {
		t.Fatal(err)
	}
	file.WriteString(solNoPragma)
	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi"},
	}

	solReturn, err := solc.Compile([]string{"simpleContract.sol"}, FlagOpts{SolcFlagOpts: flags})
	if err != nil {
		t.Fatal(err)
	}
	if solReturn.Warning == "" {
		t.Fatal("Expected a warning.")
	}
}

func TestLinkingBinaries(t *testing.T) {
	solc, err := InitCompiler("solc")
	if err != nil {
		t.Skip(err)
	}
	solc = solc.(*Solidity)
	file, err := os.Create("simpleLibrary.sol")
	defer os.Remove("simpleLibrary.sol")
	if err != nil {
		t.Fatal(err)
	}
	file.WriteString(simplyLibrarySol)
	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin"},
	}

	solReturn, err := solc.Compile([]string{"simpleLibrary.sol"}, FlagOpts{SolcFlagOpts: flags})
	if err != nil {
		t.Fatal(err)
	}

	if solReturn.Error != nil || solReturn.Warning != "" || len(solReturn.Contracts) != 2 {
		t.Fatalf("Expected no errors or warnings and expected contract items. Got %v for errors, %v for warnings, and %v for contract items", solReturn.Error, solReturn.Warning, solReturn.Contracts)
	}
	// note: When solc upgrades to 0.4.10, will need to add "simpleLibrary.sol:" to beginning of this string
	flags.Libraries = map[string]common.Address{"simpleLibrary.sol:Set": common.StringToAddress("0x692a70d2e424a56d2c6c27aa97d1a86395877b3a")}
	binFile, err := os.Create("C.bin")
	defer os.Remove("C.bin")
	if err != nil {
		t.Fatal(err)
	}

	binFile.WriteString(solReturn.Contracts["simpleLibrary.sol:C"].Bin)
	_, err = solc.(*Solidity).Compile([]string{"./C.bin"}, FlagOpts{SolcFlagOpts: flags})
	if err != nil {
		t.Fatal(err)
	}
	output, err := ioutil.ReadFile("C.bin")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(output), "_") {
		t.Fatal("Expected binaries to link, but they did not")
	}
}

func TestLinkingBinariesAndNormalCompileMixed(t *testing.T) {
	solc, err := InitCompiler("solc")
	if err != nil {
		t.Skip(err)
	}
	solc = solc.(*Solidity)
	file, err := os.Create("simpleLibrary.sol")
	defer os.Remove("simpleLibrary.sol")
	if err != nil {
		t.Fatal(err)
	}
	file.WriteString(simplyLibrarySol)
	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin"},
	}

	solReturn, err := solc.Compile([]string{"simpleLibrary.sol"}, FlagOpts{SolcFlagOpts: flags})
	if err != nil {
		t.Fatal(err)
	}

	if solReturn.Error != nil || solReturn.Warning != "" || len(solReturn.Contracts) != 2 {
		t.Fatalf("Expected no errors or warnings and expected contract items. Got %v for errors, %v for warnings, and %v for contract items", solReturn.Error, solReturn.Warning, solReturn.Contracts)
	}
	// note: When solc upgrades to 0.4.10, will need to add "simpleLibrary.sol:" to beginning of this string
	flags.Libraries = map[string]common.Address{"simpleLibrary.sol:Set": common.StringToAddress("0x692a70d2e424a56d2c6c27aa97d1a86395877b3a")}
	binFile, err := os.Create("C.bin")
	defer os.Remove("C.bin")
	if err != nil {
		t.Fatal(err)
	}
	binFile.WriteString(solReturn.Contracts["simpleLibrary.sol:C"].Bin)

	solOutput, err := solc.Compile([]string{"./C.bin", "simpleLibrary.sol"}, FlagOpts{SolcFlagOpts: flags})
	if err != nil {
		t.Fatal(err)
	}
	binOutput, err := ioutil.ReadFile("C.bin")
	if err != nil {
		t.Fatal(err)
	}

	if strings.Contains(string(binOutput), "_") {
		t.Fatal("Expected binaries to link, but they did not")
	}

	if solOutput.Error != nil || solOutput.Warning != "" || len(solOutput.Contracts) != 2 {
		t.Fatalf("Expected no errors or warnings and expected contract items. Got %v for errors, %v for warnings, and %v for contract items", solReturn.Error, solReturn.Warning, solReturn.Contracts)
	}
}

func TestMultipleFilesCompiling(t *testing.T) {
	solc, err := InitCompiler("solc")
	if err != nil {
		t.Skip(err)
	}
	solc = solc.(*Solidity)
	set, err := os.Create("set.sol")
	defer os.Remove("set.sol")
	if err != nil {
		t.Fatal(err)
	}
	set.WriteString(solFile1)

	c, err := os.Create("C.sol")
	defer os.Remove("C.sol")
	if err != nil {
		t.Fatal(err)
	}
	c.WriteString(solFile2)
	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi"},
	}

	solReturn, err := solc.Compile([]string{"C.sol"}, FlagOpts{SolcFlagOpts: flags})
	if err != nil {
		t.Fatal(err)
	}

	if solReturn.Error != nil || solReturn.Warning != "" || len(solReturn.Contracts) != 2 {
		t.Fatalf("Expected no errors or warnings and expected contract items. Got %v for errors, %v for warnings, and %v for contract items", solReturn.Error, solReturn.Warning, solReturn.Contracts)
	}
}

func TestRemappings(t *testing.T) {
	solc, err := InitCompiler("solc")
	if err != nil {
		t.Skip(err)
	}
	solc = solc.(*Solidity)
	if err := os.MkdirAll("."+string(filepath.Separator)+"tempDir", 0777); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("." + string(filepath.Separator) + "tempDir")
	os.Chdir("tempDir")
	set, err := os.Create("set.sol")
	if err != nil {
		t.Fatal(err)
	}
	os.Chdir("..")
	set.WriteString(solFile1)

	c, err := os.Create("C.sol")
	defer os.Remove("C.sol")
	if err != nil {
		t.Fatal(err)
	}
	c.WriteString(solFile2)
	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi"},
		Remappings:     []string{`set.sol=./tempDir/set.sol`},
	}

	solReturn, err := solc.Compile([]string{"C.sol"}, FlagOpts{SolcFlagOpts: flags})
	if err != nil {
		t.Fatal(err)
	}

	if solReturn.Error != nil || solReturn.Warning != "" || len(solReturn.Contracts) != 2 {
		t.Fatalf("Expected no errors or warnings and expected contract items. Got %v for errors, %v for warnings, and %v for contract items", solReturn.Error, solReturn.Warning, solReturn.Contracts)
	}
}
