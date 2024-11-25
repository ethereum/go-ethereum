// Copyright 2014 The go-ethereum Authors
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

// evm executes EVM code snippets.
package main

import (
	"fmt"
	"math/big"
	"os"

	"github.com/XinFinOrg/XDPoSChain/internal/flags"
	"github.com/urfave/cli/v2"
)

var (
	gitCommit = "" // Git SHA1 commit hash of the release (set via linker flags)

	app = flags.NewApp(gitCommit, "the evm command line interface")
)

var (
	DebugFlag = &cli.BoolFlag{
		Name:     "debug",
		Usage:    "output full trace logs",
		Category: flags.VMCategory,
	}
	MemProfileFlag = &cli.StringFlag{
		Name:     "memprofile",
		Usage:    "creates a memory profile at the given path",
		Category: flags.VMCategory,
	}
	CPUProfileFlag = &cli.StringFlag{
		Name:     "cpuprofile",
		Usage:    "creates a CPU profile at the given path",
		Category: flags.VMCategory,
	}
	StatDumpFlag = &cli.BoolFlag{
		Name:     "statdump",
		Usage:    "displays stack and heap memory information",
		Category: flags.VMCategory,
	}
	CodeFlag = &cli.StringFlag{
		Name:     "code",
		Usage:    "EVM code",
		Category: flags.VMCategory,
	}
	CodeFileFlag = &cli.StringFlag{
		Name:     "codefile",
		Usage:    "File containing EVM code. If '-' is specified, code is read from stdin ",
		Category: flags.VMCategory,
	}
	GasFlag = &cli.Uint64Flag{
		Name:     "gas",
		Usage:    "gas limit for the evm",
		Value:    10000000000,
		Category: flags.VMCategory,
	}
	PriceFlag = &flags.BigFlag{
		Name:     "price",
		Usage:    "price set for the evm",
		Value:    new(big.Int),
		Category: flags.VMCategory,
	}
	ValueFlag = &flags.BigFlag{
		Name:     "value",
		Usage:    "value set for the evm",
		Value:    new(big.Int),
		Category: flags.VMCategory,
	}
	DumpFlag = &cli.BoolFlag{
		Name:     "dump",
		Usage:    "dumps the state after the run",
		Category: flags.VMCategory,
	}
	InputFlag = &cli.StringFlag{
		Name:     "input",
		Usage:    "input for the EVM",
		Category: flags.VMCategory,
	}
	VerbosityFlag = &cli.IntFlag{
		Name:     "verbosity",
		Usage:    "sets the verbosity level",
		Category: flags.VMCategory,
	}
	CreateFlag = &cli.BoolFlag{
		Name:     "create",
		Usage:    "indicates the action should be create rather than call",
		Category: flags.VMCategory,
	}
	GenesisFlag = &cli.StringFlag{
		Name:     "prestate",
		Usage:    "JSON file with prestate (genesis) config",
		Category: flags.VMCategory,
	}
	MachineFlag = &cli.BoolFlag{
		Name:     "json",
		Usage:    "output trace logs in machine readable format (json)",
		Category: flags.VMCategory,
	}
	SenderFlag = &cli.StringFlag{
		Name:     "sender",
		Usage:    "The transaction origin",
		Category: flags.VMCategory,
	}
	ReceiverFlag = &cli.StringFlag{
		Name:     "receiver",
		Usage:    "The transaction receiver (execution context)",
		Category: flags.VMCategory,
	}
	DisableMemoryFlag = &cli.BoolFlag{
		Name:     "nomemory",
		Value:    true,
		Usage:    "disable memory output",
		Category: flags.VMCategory,
	}
	DisableStackFlag = &cli.BoolFlag{
		Name:     "nostack",
		Usage:    "disable stack output",
		Category: flags.VMCategory,
	}
	DisableStorageFlag = &cli.BoolFlag{
		Name:     "nostorage",
		Usage:    "disable storage output",
		Category: flags.VMCategory,
	}
	DisableReturnDataFlag = &cli.BoolFlag{
		Name:     "noreturndata",
		Value:    true,
		Usage:    "enable return data output",
		Category: flags.VMCategory,
	}
)

func init() {
	app.Flags = []cli.Flag{
		CreateFlag,
		DebugFlag,
		VerbosityFlag,
		CodeFlag,
		CodeFileFlag,
		GasFlag,
		PriceFlag,
		ValueFlag,
		DumpFlag,
		InputFlag,
		MemProfileFlag,
		CPUProfileFlag,
		StatDumpFlag,
		GenesisFlag,
		MachineFlag,
		SenderFlag,
		ReceiverFlag,
		DisableMemoryFlag,
		DisableStackFlag,
	}
	app.Commands = []*cli.Command{
		compileCommand,
		disasmCommand,
		runCommand,
		stateTestCommand,
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
