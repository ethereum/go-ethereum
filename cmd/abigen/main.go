// Copyright 2016 The go-ethereum Authors
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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ethereum/go-ethereum/accounts/abi/bind"
)

var (
	abiFlag = flag.String("abi", "", "Path to the Ethereum contract ABI json to bind")
	binFlag = flag.String("bin", "", "Path to the Ethereum contract bytecode (generate deploy method)")
	pkgFlag = flag.String("pkg", "", "Go package name to generate the binding into")
	typFlag = flag.String("type", "", "Go struct name for the binding (default = package name)")
	outFlag = flag.String("out", "", "Output path for the generated binding (default = stdout)")
)

func main() {
	// Parse and validate the command line flags
	flag.Parse()

	if *abiFlag == "" {
		fmt.Printf("No contract ABI path specified (--abi)\n")
		os.Exit(-1)
	}
	if *pkgFlag == "" {
		fmt.Printf("No destination Go package specified (--pkg)\n")
		os.Exit(-1)
	}
	// Read the ABI json from disk and optionally the contract bytecode too
	abi, err := ioutil.ReadFile(*abiFlag)
	if err != nil {
		fmt.Printf("Failed to read input ABI: %v\n", err)
		os.Exit(-1)
	}
	bin := []byte{}
	if *binFlag != "" {
		if bin, err = ioutil.ReadFile(*binFlag); err != nil {
			fmt.Printf("Failed to read input bytecode: %v\n", err)
			os.Exit(-1)
		}
	}
	// Generate the contract binding
	kind := *typFlag
	if kind == "" {
		kind = *pkgFlag
	}
	code, err := bind.Bind(string(abi), string(bin), *pkgFlag, kind)
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
