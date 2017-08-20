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
	typFlag = flag.String("type", "", "Struct name for the binding (default = package name)")

	linkFlag  = flag.String("link", "", "Library flag linker for name to addresses in the code")
	solFlag   = flag.String("sol", "", "Comma separated path(s) to Solidity source(s) to build and bind")
	pathsFlag = flag.String("paths", "", "Comma separated path(s) in the form of solidity remappings (advanced only)")
	solcFlag  = flag.String("compiler", "solc", "Compiler to use if source builds are requested (default = solc)")
	execFlag  = flag.String("exec", "", "Execute compiler with user generated command, advanced only, use commas to represent spaces in one string")

	pkgFlag  = flag.String("pkg", "", "Package name to generate the binding into")
	outFlag  = flag.String("out", "", "Output file for the generated binding (default = stdout)")
	langFlag = flag.String("lang", "go", "Destination language for the bindings (go, java, objc)")
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
	} else if (*binFlag == "" || *solFlag == "") && *linkFlag != "" {
		fmt.Printf("No contract bytecode created through (--bin) or (--sol) options to use with (--link) option\n")
		os.Exit(-1)
	}
	if *pkgFlag == "" {
		fmt.Printf("No destination package specified (--pkg)\n")
		os.Exit(-1)
	}
	var lang bind.Lang
	switch *langFlag {
	case "go":
		lang = bind.LangGo
	case "java":
		lang = bind.LangJava
	case "objc":
		lang = bind.LangObjC
	default:
		fmt.Printf("Unsupported destination language \"%s\" (--lang)\n", *langFlag)
		os.Exit(-1)
	}
	// If the entire solidity code was specified, build and bind based on that
	var (
		abis  []string
		bins  []string
		types []string
	)
	if *solFlag != "" {
		solc, err := compiler.InitSolc(*solcFlag)
		if err != nil {
			fmt.Printf("Failed to initialize Solidity compiler: %v\n", err)
			os.Exit(-1)
		}

		solc.Compile(compiler.FlagOpts{}, strings.Split(*solFlag, ","))
		// Gather all non-excluded contract for binding
		for name, contract := range contracts {
			abi, _ := json.Marshal(contract.Info.AbiDefinition) // Flatten the compiler parse
			abis = append(abis, string(abi))
			bins = append(bins, contract.Code)

			nameParts := strings.Split(name, ":")
			types = append(types, nameParts[len(nameParts)-1])
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
	code, err := bind.Bind(types, abis, bins, *pkgFlag, lang)
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
