// Copyright 2024 The go-ethereum Authors
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
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/urfave/cli/v2"

	// Force-load the tracer engines to trigger registration
	_ "github.com/ethereum/go-ethereum/eth/tracers/js"
	_ "github.com/ethereum/go-ethereum/eth/tracers/native"
)

var app = flags.NewApp("go-ethereum t8n tool")

func init() {
	app.Flags = append(app.Flags, debug.Flags...)
	app.Before = func(ctx *cli.Context) error {
		flags.MigrateGlobalFlags(ctx)
		return debug.Setup(ctx)
	}
	app.After = func(ctx *cli.Context) error {
		debug.Exit()
		return nil
	}
	app.CommandNotFound = func(ctx *cli.Context, cmd string) {
		fmt.Fprintf(os.Stderr, "No such command: %s\n", cmd)
		os.Exit(1)
	}

	// Add subcommands.
	app.Commands = []*cli.Command{
		stateTransitionCommand,
		transactionCommand,
		blockBuilderCommand,
	}
}

var (
	stateTransitionCommand = &cli.Command{
		Name:    "transition",
		Aliases: []string{"t8n", "run"},
		Usage:   "Executes a full state transition",
		Action:  Transition,
		Flags: []cli.Flag{
			TraceFlag,
			TraceTracerFlag,
			TraceTracerConfigFlag,
			TraceEnableMemoryFlag,
			TraceDisableStackFlag,
			TraceEnableReturnDataFlag,
			TraceEnableCallFramesFlag,
			OutputBasedir,
			OutputAllocFlag,
			OutputResultFlag,
			OutputBodyFlag,
			InputAllocFlag,
			InputEnvFlag,
			InputTxsFlag,
			ForknameFlag,
			ChainIDFlag,
			RewardFlag,
		},
	}
	transactionCommand = &cli.Command{
		Name:    "transaction",
		Aliases: []string{"t9n"},
		Usage:   "Performs transaction validation",
		Action:  Transaction,
		Flags: []cli.Flag{
			InputTxsFlag,
			ChainIDFlag,
			ForknameFlag,
		},
	}
	blockBuilderCommand = &cli.Command{
		Name:    "block-builder",
		Aliases: []string{"b11r"},
		Usage:   "Builds a block",
		Action:  BuildBlock,
		Flags: []cli.Flag{
			OutputBasedir,
			OutputBlockFlag,
			InputHeaderFlag,
			InputOmmersFlag,
			InputWithdrawalsFlag,
			InputTxsRlpFlag,
			SealCliqueFlag,
		},
	}
)

func main() {
	if err := app.Run(os.Args); err != nil {
		code := 1
		if ec, ok := err.(*NumberedError); ok {
			code = ec.ExitCode()
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(code)
	}
}
