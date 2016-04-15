// Copyright 2016 The go-ethereum Authors
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
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common/compiler"
)

var (
	abiFlag = flag.String("abi", "", "Path to the Ethereum contract ABI json to bind")
	binFlag = flag.String("bin", "", "Path to the Ethereum contract bytecode (generate deploy method)")
	typFlag = flag.String("type", "", "Go struct name for the binding (default = package name)")

	solFlag  = flag.String("sol", "", "Path to the Ethereum contract Solidity source to build and bind")
	solcFlag = flag.String("solc", "solc", "Solidity compiler to use if source builds are requested")
	excFlag  = flag.String("exc", "", "Comma separated types to exclude from binding")

	pkgFlag = flag.String("pkg", "", "Go package name to generate the binding into")
	outFlag = flag.String("out", "", "Output file for the generated binding (default = stdout)")
)

func main() {
	// Parse and ensure all needed inputs are specified
	flag.Parse()

	if *abiFlag == "" && *solFlag == "" {
		fmt.Printf("No contract ABI (--abi) or Solidity source (--sol) specified\n")
		os.Exit(-1)
	} else if (*abiFlag != "" || *binFlag != "" || *typFlag != "") && *solFlag != "" {
		fmt.Printf("Contract ABI (--abi), bytecode (--bin) and type (--type) flags are mutually exclusive with the Solidity source (--sol) flag\n")
		os.Exit(-1)
	}
	if *pkgFlag == "" {
		fmt.Printf("No destination Go package specified (--pkg)\n")
		os.Exit(-1)
	}
	// If the entire solidity code was specified, build and bind based on that
	var (
		abis  []string
		bins  []string
		types []string
	)
	if *solFlag != "" {
		// Generate the list of types to exclude from binding
		exclude := make(map[string]bool)
		for _, kind := range strings.Split(*excFlag, ",") {
			exclude[strings.ToLower(kind)] = true
		}
		// Build the Solidity source into bindable components
		solc, err := compiler.New(*solcFlag)
		if err != nil {
			fmt.Printf("Failed to locate Solidity compiler: %v\n", err)
			os.Exit(-1)
		}
		source, err := ioutil.ReadFile(*solFlag)
		if err != nil {
			fmt.Printf("Failed to read Soldity source code: %v\n", err)
			os.Exit(-1)
		}
		contracts, err := solc.Compile(string(source))
		if err != nil {
			fmt.Printf("Failed to build Solidity contract: %v\n", err)
			os.Exit(-1)
		}
		// Gather all non-excluded contract for binding
		for name, contract := range contracts {
			if exclude[strings.ToLower(name)] {
				continue
			}
			abi, _ := json.Marshal(contract.Info.AbiDefinition) // Flatten the compiler parse
			abis = append(abis, string(abi))
			bins = append(bins, contract.Code)
			types = append(types, name)
		}
	} else {
		// Otherwise load up the ABI, optional bytecode and type name from the parameters
		abi, err := ioutil.ReadFile(*abiFlag)
		if err != nil {
			fmt.Printf("Failed to read input ABI: %v\n", err)
			os.Exit(-1)
		}
		abis = append(abis, string(abi))

		bin := []byte{}
		if *binFlag != "" {
			if bin, err = ioutil.ReadFile(*binFlag); err != nil {
				fmt.Printf("Failed to read input bytecode: %v\n", err)
				os.Exit(-1)
			}
		}
		bins = append(bins, string(bin))

		kind := *typFlag
		if kind == "" {
			kind = *pkgFlag
		}
		types = append(types, kind)
	}
	// Generate the contract binding
	code, err := bind.Bind(types, abis, bins, *pkgFlag)
	if err != nil {
		fmt.Printf("Failed to generate ABI binding: %v\n", err)
		os.Exit(-1)
	}
	// Either flush it out to a file or display on the standard output
	if *outFlag == "" {
		fmt.Printf("%s\n", code)
		return
	}
	if err := ioutil.WriteFile(*outFlag, []byte(code), 0600); err != nil {
		fmt.Printf("Failed to write ABI binding: %v\n", err)
		os.Exit(-1)
	}
}
