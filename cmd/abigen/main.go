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
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/XinFinOrg/XDPoSChain/accounts/abi/bind"
	"github.com/XinFinOrg/XDPoSChain/common/compiler"
	"github.com/XinFinOrg/XDPoSChain/internal/flags"
	"github.com/XinFinOrg/XDPoSChain/log"
	"github.com/urfave/cli/v2"
)

var (
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""

	app *cli.App
)

var (
	// Flags needed by abigen
	abiFlag = &cli.StringFlag{
		Name:  "abi",
		Usage: "Path to the Ethereum contract ABI json to bind",
	}
	binFlag = &cli.StringFlag{
		Name:  "bin",
		Usage: "Path to the Ethereum contract bytecode (generate deploy method)",
	}
	typeFlag = &cli.StringFlag{
		Name:  "type",
		Usage: "Struct name for the binding (default = package name)",
	}
	solFlag = &cli.StringFlag{
		Name:  "sol",
		Usage: "Path to the Ethereum contract Solidity source to build and bind",
	}
	solcFlag = &cli.StringFlag{
		Name:  "solc",
		Usage: "Solidity compiler to use if source builds are requested",
		Value: "solc",
	}
	vyFlag = &cli.StringFlag{
		Name:  "vy",
		Usage: "Path to the Ethereum contract Vyper source to build and bind",
	}
	vyperFlag = &cli.StringFlag{
		Name:  "vyper",
		Usage: "Vyper compiler to use if source builds are requested",
		Value: "vyper",
	}
	excFlag = &cli.StringFlag{
		Name:  "exc",
		Usage: "Comma separated types to exclude from binding",
	}
	pkgFlag = &cli.StringFlag{
		Name:  "pkg",
		Usage: "Package name to generate the binding into",
	}
	outFlag = &cli.StringFlag{
		Name:  "out",
		Usage: "Output file for the generated binding (default = stdout)",
	}
	langFlag = &cli.StringFlag{
		Name:  "lang",
		Usage: "Destination language for the bindings (go)",
		Value: "go",
	}
)

func init() {
	app = flags.NewApp(gitCommit, "ethereum checkpoint helper tool")
	app.Name = "abigen"
	app.Flags = []cli.Flag{
		abiFlag,
		binFlag,
		typeFlag,
		solFlag,
		solcFlag,
		vyFlag,
		vyperFlag,
		excFlag,
		pkgFlag,
		outFlag,
		langFlag,
	}
	app.Action = abigen
}

func abigen(c *cli.Context) error {
	if c.String(abiFlag.Name) == "" && c.String(solFlag.Name) == "" && c.String(vyFlag.Name) == "" {
		fmt.Printf("No contract ABI (--abi), Solidity source (--sol), or Vyper source (--vy) specified\n")
		os.Exit(-1)
	} else if (c.String(abiFlag.Name) != "" || c.String(binFlag.Name) != "" || c.String(typeFlag.Name) != "") && (c.String(solFlag.Name) != "" || c.String(vyFlag.Name) != "") {
		fmt.Printf("Contract ABI (--abi), bytecode (--bin) and type (--type) flags are mutually exclusive with the Solidity (--sol) and Vyper (--vy) flags\n")
		os.Exit(-1)
	} else if c.String(solFlag.Name) != "" && c.String(vyFlag.Name) == "" {
		fmt.Printf("Solidity (--sol) and Vyper (--vy) flags are mutually exclusive\n")
		os.Exit(-1)
	}
	if c.String(pkgFlag.Name) == "" {
		fmt.Printf("No destination package specified (--pkg)\n")
		os.Exit(-1)
	}
	var lang bind.Lang
	switch c.String(langFlag.Name) {
	case "go":
		lang = bind.LangGo
	default:
		fmt.Printf("Unsupported destination language \"%s\" (--lang)\n", c.String(langFlag.Name))
		os.Exit(-1)
	}
	// If the entire solidity code was specified, build and bind based on that
	var (
		abis  []string
		bins  []string
		types []string
		sigs  []map[string]string
	)
	if c.String(solFlag.Name) != "" || c.String(vyFlag.Name) != "" || c.String(abiFlag.Name) == "-" {
		// Generate the list of types to exclude from binding
		exclude := make(map[string]bool)
		for _, kind := range strings.Split(c.String(excFlag.Name), ",") {
			exclude[strings.ToLower(kind)] = true
		}

		var contracts map[string]*compiler.Contract
		var err error

		switch {
		case c.String(solFlag.Name) != "":
			contracts, err = compiler.CompileSolidity(c.String(solcFlag.Name), c.String(solFlag.Name))
			if err != nil {
				fmt.Printf("Failed to build Solidity contract: %v\n", err)
				os.Exit(-1)
			}
		case c.String(vyFlag.Name) != "":
			contracts, err = compiler.CompileVyper(c.String(vyperFlag.Name), c.String(vyFlag.Name))
			if err != nil {
				fmt.Printf("Failed to build Vyper contract: %v\n", err)
				os.Exit(-1)
			}
		default:
			contracts, err = contractsFromStdin()
			if err != nil {
				fmt.Printf("Failed to read input ABIs from STDIN: %v\n", err)
				os.Exit(-1)
			}
		}

		// Gather all non-excluded contract for binding
		for name, contract := range contracts {
			if exclude[strings.ToLower(name)] {
				continue
			}
			abi, err := json.Marshal(contract.Info.AbiDefinition) // Flatten the compiler parse
			if err != nil {
				fmt.Printf("Failed to parse ABIs from compiler output: %v\n", err)
				os.Exit(-1)
			}
			abis = append(abis, string(abi))
			bins = append(bins, contract.Code)
			sigs = append(sigs, contract.Hashes)

			nameParts := strings.Split(name, ":")
			types = append(types, nameParts[len(nameParts)-1])
		}
	} else {
		// Otherwise load up the ABI, optional bytecode and type name from the parameters
		abi, err := os.ReadFile(c.String(abiFlag.Name))

		if err != nil {
			fmt.Printf("Failed to read input ABI: %v\n", err)
			os.Exit(-1)
		}
		abis = append(abis, string(abi))

		var bin []byte
		if c.String(binFlag.Name) != "" {
			if bin, err = os.ReadFile(c.String(binFlag.Name)); err != nil {
				fmt.Printf("Failed to read input bytecode: %v\n", err)
				os.Exit(-1)
			}
		}
		bins = append(bins, string(bin))

		kind := c.String(typeFlag.Name)
		if kind == "" {
			kind = c.String(pkgFlag.Name)
		}
		types = append(types, kind)
	}
	// Generate the contract binding
	code, err := bind.Bind(types, abis, bins, sigs, c.String(pkgFlag.Name), lang)
	if err != nil {
		fmt.Printf("Failed to generate ABI binding: %v\n", err)
		os.Exit(-1)
	}
	// Either flush it out to a file or display on the standard output
	if c.String(outFlag.Name) == "" {
		fmt.Printf("%s\n", code)
		return nil
	}
	if err := os.WriteFile(c.String(outFlag.Name), []byte(code), 0600); err != nil {
		fmt.Printf("Failed to write ABI binding: %v\n", err)
		os.Exit(-1)
	}
	return nil
}

func main() {
	log.SetDefault(log.NewLogger(log.NewTerminalHandlerWithLevel(os.Stderr, log.LevelInfo, true)))

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func contractsFromStdin() (map[string]*compiler.Contract, error) {
	bytes, err := io.ReadAll(os.Stdin)
	if err != nil {
		return nil, err
	}
	return compiler.ParseCombinedJSON(bytes, "", "", "", "")
}
