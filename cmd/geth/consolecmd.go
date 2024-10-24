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
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/console"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/urfave/cli/v2"
)

var (
	consoleFlags = []cli.Flag{utils.JSpathFlag, utils.ExecFlag, utils.PreloadJSFlag}

	consoleCommand = &cli.Command{
		Action: localConsole,
		Name:   "console",
		Usage:  "Start an interactive JavaScript environment",
		Flags:  flags.Merge(nodeFlags, rpcFlags, consoleFlags),
		Description: `
The Geth console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://geth.ethereum.org/docs/interacting-with-geth/javascript-console.`,
	}

	attachCommand = &cli.Command{
		Action:    remoteConsole,
		Name:      "attach",
		Usage:     "Start an interactive JavaScript environment (connect to node)",
		ArgsUsage: "[endpoint]",
		Flags:     flags.Merge([]cli.Flag{utils.DataDirFlag, utils.HttpHeaderFlag}, consoleFlags),
		Description: `
The Geth console is an interactive shell for the JavaScript runtime environment
which exposes a node admin interface as well as the Ðapp JavaScript API.
See https://geth.ethereum.org/docs/interacting-with-geth/javascript-console.
This command allows to open a console on a running geth node.`,
	}
)

// localConsole starts a new geth node, attaching a JavaScript console to it at the
// same time.
func localConsole(ctx *cli.Context) error {
	// Create and start the node based on the CLI flags
	prepare(ctx)
	stack := makeFullNode(ctx)
	startNode(ctx, stack, true)
	defer stack.Close()

	// Attach to the newly started node and create the JavaScript console.
	return console.RunInProc(stack.IPCEndpoint())
}

// remoteConsole will connect to a remote geth instance, attaching a JavaScript
// console to it.
func remoteConsole(ctx *cli.Context) error {
	return console.RunAsProc(ctx.Args().First())
}
