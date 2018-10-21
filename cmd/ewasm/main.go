// Copyright 2018 The go-ethereum Authors
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

// ewasm executes ewasm modules.
package main

import (
	"errors"
	"fmt"
	"math/big"
	"math/rand"
	"os"
	"strconv"

	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/params"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	coreVM "github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/ethdb"
	"gopkg.in/urfave/cli.v1"
)

var (
	gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)

	app = utils.NewApp(gitCommit, "the ewasm command line interface")
)

var runCommand = cli.Command{
	Action:      runCmd,
	Name:        "run",
	Usage:       "run and arbitrary ewasm module",
	ArgsUsage:   "<module name> <input> <gas>",
	Description: `The run command runs an arbitrary ewasm module.`,
}

func runCmd(ctx *cli.Context) error {
	args := ctx.Args()

	statedb, err := state.New(common.Hash{}, state.NewDatabase(ethdb.NewMemDatabase()))
	if err != nil {
		utils.Fatalf("Could not create the state database: %v", err)
	}

	if args.Present() {
		filename := args.First()
		inputStr := args.Get(1)
		gas, err := strconv.ParseUint(args.Get(2), 10, 64)
		if err != nil {
			return fmt.Errorf("Error parsing gas number: %v", err)
		}

		// Convert the input
		input := []byte{}
		if inputStr[0:2] != "0x" {
			return fmt.Errorf("Invalid input, it should be a hexadecimal number starting with 0x")
		}
		inputStr = inputStr[2:]
		if len(inputStr)%2 == 1 {
			inputStr = "0" + inputStr
		}
		for inputStr != "" {
			x, err := strconv.ParseUint(inputStr[0:2], 16, 8)
			if err != nil {
				return fmt.Errorf("Invalid byte in input")
			}
			input = append(input, byte(x))
			inputStr = inputStr[2:]
		}

		if fd, err := os.Open(filename); err == nil {

			fi, _ := fd.Stat()
			code := make([]byte, fi.Size())
			n, err := fd.Read(code)
			if n != len(code) || err != nil {
				return fmt.Errorf("Read %d bytes out of %d, err: %v", n, len(code), err)
			}

			randomContractAddress := make([]byte, common.HashLength)
			rand.Read(randomContractAddress)
			contractAddr := common.BytesToAddress(randomContractAddress)

			randomCallerAddress := make([]byte, common.HashLength)
			rand.Read(randomContractAddress)
			callerAddr := common.BytesToAddress(randomCallerAddress)

			contract := coreVM.NewContract(coreVM.AccountRef(callerAddr), coreVM.AccountRef(contractAddr), big.NewInt(100), gas)
			contract.Code = code
			contract.Input = input

			permissiveContext := coreVM.Context{
				CanTransfer: core.CanTransfer,
				Transfer:    core.Transfer,
			}

			evm := coreVM.NewEVM(permissiveContext, statedb, &params.ChainConfig{}, coreVM.Config{})

			evm.StateDB.SetCode(contractAddr, code)


			output, leftOver, err := evm.Call(coreVM.AccountRef(callerAddr), contractAddr, input, gas, big.NewInt(0))

			if err != nil {
				return err
			}

			fmt.Println("Output\n", output)
			fmt.Println("left over gas: ", leftOver)
			return nil
		}

		return fmt.Errorf("Error opening module file: %v", err)
	}

	return errors.New("You need to specify a module name")
}

func init() {
	app.Commands = []cli.Command{
		runCommand,
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
