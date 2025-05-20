// Copyright 2024 The go-ethereum Authors
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

package bind_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ethereum/go-ethereum/accounts/abi/abigen"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/compiler"
	"github.com/ethereum/go-ethereum/crypto"
)

// Run go generate to recreate the test bindings.
//
//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen -v2 -combined-json internal/contracts/db/combined-abi.json -type DBStats -pkg db -out internal/contracts/db/bindings.go
//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen -v2 -combined-json internal/contracts/events/combined-abi.json -type C -pkg events -out internal/contracts/events/bindings.go
//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen -v2 -combined-json internal/contracts/nested_libraries/combined-abi.json -type C1 -pkg nested_libraries -out internal/contracts/nested_libraries/bindings.go
//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen -v2 -combined-json internal/contracts/solc_errors/combined-abi.json -type C -pkg solc_errors -out internal/contracts/solc_errors/bindings.go
//go:generate go run github.com/ethereum/go-ethereum/cmd/abigen -v2 -combined-json internal/contracts/uint256arrayreturn/combined-abi.json -type C -pkg uint256arrayreturn -out internal/contracts/uint256arrayreturn/bindings.go

// TestBindingGeneration tests that re-running generation of bindings does not result in
// mutations to the binding code.
func TestBindingGeneration(t *testing.T) {
	matches, _ := filepath.Glob("internal/contracts/*")
	var dirs []string
	for _, match := range matches {
		f, _ := os.Stat(match)
		if f.IsDir() {
			dirs = append(dirs, f.Name())
		}
	}

	for _, dir := range dirs {
		var (
			abis  []string
			bins  []string
			types []string
			libs  = make(map[string]string)
		)
		basePath := filepath.Join("internal/contracts", dir)
		combinedJsonPath := filepath.Join(basePath, "combined-abi.json")
		abiBytes, err := os.ReadFile(combinedJsonPath)
		if err != nil {
			t.Fatalf("error trying to read file %s: %v", combinedJsonPath, err)
		}
		contracts, err := compiler.ParseCombinedJSON(abiBytes, "", "", "", "")
		if err != nil {
			t.Fatalf("Failed to read contract information from json output: %v", err)
		}

		for name, contract := range contracts {
			// fully qualified name is of the form <solFilePath>:<type>
			nameParts := strings.Split(name, ":")
			typeName := nameParts[len(nameParts)-1]
			abi, err := json.Marshal(contract.Info.AbiDefinition) // Flatten the compiler parse
			if err != nil {
				utils.Fatalf("Failed to parse ABIs from compiler output: %v", err)
			}
			abis = append(abis, string(abi))
			bins = append(bins, contract.Code)
			types = append(types, typeName)

			// Derive the library placeholder which is a 34 character prefix of the
			// hex encoding of the keccak256 hash of the fully qualified library name.
			// Note that the fully qualified library name is the path of its source
			// file and the library name separated by ":".
			libPattern := crypto.Keccak256Hash([]byte(name)).String()[2:36] // the first 2 chars are 0x
			libs[libPattern] = typeName
		}
		code, err := abigen.BindV2(types, abis, bins, dir, libs, make(map[string]string))
		if err != nil {
			t.Fatalf("error creating bindings for package %s: %v", dir, err)
		}

		existingBindings, err := os.ReadFile(filepath.Join(basePath, "bindings.go"))
		if err != nil {
			t.Fatalf("ReadFile returned error: %v", err)
		}
		if code != string(existingBindings) {
			t.Fatalf("code mismatch for %s", dir)
		}
	}
}
