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
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

const regularSolFile = `pragma solidity >= 0.0.0;
	contract main {

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
import "/somedir/set.sol";

contract C {
    Set.Data knownValues;
    function register(uint value) {
        if (!Set.insert(knownValues, value))
            throw;
    }
}`

const solFile2 = `pragma solidity >=0.0.0;

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

func skipWithoutSolc(t *testing.T) {
	if _, err := exec.LookPath("solc"); err != nil {
		t.Skip(err)
	}
}

func writeToTempFile(tmpfile *os.File, content []byte) error {

	if _, err := tmpfile.Write(content); err != nil {
		return err
	}
	if err := tmpfile.Close(); err != nil {
		return err
	}
	return nil
}

func TestSolcCompilerNormal(t *testing.T) {

	skipWithoutSolc(t)

	solc, err := InitSolc("solc")
	if err != nil {
		t.Fatalf("Could not initialize solc: %v", err)
	}

	content := []byte(regularSolFile)
	tmpfile, err := ioutil.TempFile("", "simpleContract.sol")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	err = writeToTempFile(tmpfile, content)
	if err != nil {
		t.Fatal(err)
	}

	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi"},
	}

	solReturn, err := solc.Compile(FlagOpts{flags}, tmpfile.Name())
	if err != nil {
		t.Errorf("Expected no errors: %v", err)
	}

	if solReturn.Warning != "" || len(solReturn.Contracts) != 1 {
		t.Fatalf("Expected no warnings and expected 1 contract item. Got %v for warnings, and %v for contract items", solReturn.Warning, len(solReturn.Contracts))
	}
}

func TestSolcCompilerError(t *testing.T) {

	skipWithoutSolc(t)

	solc, err := InitSolc("solc")
	if err != nil {
		t.Fatalf("Could not initialize solc: %v", err)
	}
	content := []byte(faultySol)
	tmpfile, err := ioutil.TempFile("", "faultyContract.sol")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	err = writeToTempFile(tmpfile, content)
	if err != nil {
		t.Fatal(err)
	}

	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi"},
	}

	_, err = solc.Compile(FlagOpts{SolcFlagOpts: flags}, tmpfile.Name())

	if err == nil {
		t.Fatal("Expected an error, got nil.")
	} else if !strings.Contains(err.Error(), "solc") {
		t.Fatalf("Expected error to come directly from compiler, got err from elsewhere: %v", err)
	}
}

func TestSolcCompilerWarning(t *testing.T) {

	skipWithoutSolc(t)

	solc, err := InitSolc("solc")
	if err != nil {
		t.Fatalf("Could not initialize solc: %v", err)
	}
	content := []byte(solNoPragma)
	tmpfile, err := ioutil.TempFile("", "warningContract.sol")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	err = writeToTempFile(tmpfile, content)
	if err != nil {
		t.Fatal(err)
	}

	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi"},
	}

	solReturn, err := solc.Compile(FlagOpts{SolcFlagOpts: flags}, tmpfile.Name())
	if err != nil {
		t.Error(err)
	}
	if solReturn.Warning == "" {
		t.Error("Expected a warning, got none.")
	}
}

func TestLinkingBinaries(t *testing.T) {

	skipWithoutSolc(t)

	solc, err := InitSolc("solc")
	if err != nil {
		t.Fatalf("Could not initialize solc: %v", err)
	}
	content := []byte(simplyLibrarySol)
	tmpfile, err := ioutil.TempFile("", "libraryContracts.sol")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name()) // clean up

	err = writeToTempFile(tmpfile, content)
	if err != nil {
		t.Fatal(err)
	}

	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin"},
	}

	solReturn, err := solc.Compile(FlagOpts{SolcFlagOpts: flags}, tmpfile.Name())
	if err != nil {
		t.Fatal(err)
	}

	if solReturn.Warning != "" || len(solReturn.Contracts) != 2 {
		t.Fatalf("Expected no errors or warnings and expected contract items. Got %v for warnings, and %v for contract items", solReturn.Warning, solReturn.Contracts)
	}

	output, err := solc.(*Solidity).Link(map[string]common.Address{"Set": common.StringToAddress("0x692a70d2e424a56d2c6c27aa97d1a86395877b3a")}, solReturn.Contracts["C"].Bin)
	if err != nil {
		t.Error(err)
	}
	if strings.Contains(output, "_") {
		t.Errorf("Expected binaries to link, but they did not")
	}
}

func TestRemappings(t *testing.T) {

	skipWithoutSolc(t)

	solc, err := InitSolc("solc")
	if err != nil {
		t.Fatalf("Could not initialize solc: %v", err)
	}

	tmpfile1, err := ioutil.TempFile("", "C.sol")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile1.Name()) // clean up

	err = writeToTempFile(tmpfile1, []byte(solFile1))
	if err != nil {
		t.Fatal(err)
	}

	dir, err := ioutil.TempDir("", "tempDir")
	if err != nil {
		t.Fatal(err)
	}

	defer os.RemoveAll(dir) // clean up

	tmpfn := filepath.Join(dir, "set.sol")
	if err := ioutil.WriteFile(tmpfn, []byte(solFile2), 0666); err != nil {
		t.Fatal(err)
	}

	tmpfile2, err := ioutil.TempFile("", "main.sol")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile2.Name()) // clean up

	err = writeToTempFile(tmpfile2, []byte(regularSolFile))
	if err != nil {
		t.Fatal(err)
	}
	flags := SolcFlagOpts{
		CombinedOutput: []string{"bin", "abi"},
		Remappings:     []string{`/somedir/=` + dir + "/"},
	}

	solReturn, err := solc.Compile(FlagOpts{SolcFlagOpts: flags}, tmpfile1.Name(), tmpfile2.Name())
	if err != nil {
		t.Fatal(err)
	}

	if solReturn.Warning != "" || len(solReturn.Contracts) != 3 {
		t.Fatalf("Expected no warnings and expected %v contract items. Got %v for warnings, and %v for contract items", 3, solReturn.Warning, len(solReturn.Contracts))
	}
}
