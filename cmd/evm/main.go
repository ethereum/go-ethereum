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
	"os"

	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/urfave/cli/v2"
)

const traceCategory = "TRACING"

var (
	// Debug flags.
	DumpFlag = &cli.BoolFlag{
		Name:  "dump",
		Usage: "dumps the state after the run",
	}

	StatDumpFlag = &cli.BoolFlag{
		Name:  "statdump",
		Usage: "displays stack and heap memory information",
	}

	// Tracing flags.
	DebugFlag = &cli.BoolFlag{
		Name:     "debug",
		Usage:    "output full trace logs",
		Category: traceCategory,
	}
	MachineFlag = &cli.BoolFlag{
		Name:     "json",
		Usage:    "output trace logs in machine readable format (json)",
		Category: traceCategory,
	}

	DisableMemoryFlag = &cli.BoolFlag{
		Name:     "nomemory",
		Value:    true,
		Usage:    "disable memory output",
		Category: traceCategory,
	}
	DisableStackFlag = &cli.BoolFlag{
		Name:     "nostack",
		Usage:    "disable stack output",
		Category: traceCategory,
	}
	DisableStorageFlag = &cli.BoolFlag{
		Name:     "nostorage",
		Usage:    "disable storage output",
		Category: traceCategory,
	}
	DisableReturnDataFlag = &cli.BoolFlag{
		Name:     "noreturndata",
		Value:    true,
		Usage:    "enable return data output",
		Category: traceCategory,
	}
)

// traceFlags contains flags that configure tracing output.
var traceFlags = []cli.Flag{
	DebugFlag,
	DumpFlag,
	MachineFlag,
	StatDumpFlag,
	DisableMemoryFlag,
	DisableStackFlag,
	DisableStorageFlag,
	DisableReturnDataFlag,
}

var app = flags.NewApp("the evm command line interface")

func init() {
	app.Flags = flags.Merge(traceFlags, debug.Flags)
	app.Commands = []*cli.Command{
		runCommand,
		blockTestCommand,
		stateTestCommand,
	}
	app.Before = func(ctx *cli.Context) error {
		flags.MigrateGlobalFlags(ctx)
		return debug.Setup(ctx)
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		return nil
	}
}

func main() {
	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
