// Copyright 2025 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/ethclient"
	"github.com/ethereum/go-ethereum/internal/debug"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/urfave/cli/v2"
)

var app = flags.NewApp("go-ethereum workload test tool")

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
		runTestCommand,
		historyGenerateCommand,
		filterGenerateCommand,
		filterPerfCommand,
	}
}

func main() {
	exit(app.Run(os.Args))
}

type client struct {
	Eth *ethclient.Client
	RPC *rpc.Client
}

func makeClient(ctx *cli.Context) *client {
	if ctx.NArg() < 1 {
		exit("missing RPC endpoint URL as command-line argument")
	}
	url := ctx.Args().First()
	cl, err := rpc.Dial(url)
	if err != nil {
		exit(fmt.Errorf("Could not create RPC client at %s: %v", url, err))
	}
	return &client{
		RPC: cl,
		Eth: ethclient.NewClient(cl),
	}
}

func exit(err any) {
	if err == nil {
		os.Exit(0)
	}
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
