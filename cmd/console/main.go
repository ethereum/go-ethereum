/*
	This file is part of go-ethereum

	go-ethereum is free software: you can redistribute it and/or modify
	it under the terms of the GNU General Public License as published by
	the Free Software Foundation, either version 3 of the License, or
	(at your option) any later version.

	go-ethereum is distributed in the hope that it will be useful,
	but WITHOUT ANY WARRANTY; without even the implied warranty of
	MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
	GNU General Public License for more details.

	You should have received a copy of the GNU General Public License
	along with go-ethereum.  If not, see <http://www.gnu.org/licenses/>.
*/
/**
 * @authors
 * 	Jeffrey Wilcke <i@jev.io>
 */
package main

import (
	"fmt"
	"io"
	"os"

	"github.com/mattn/go-colorable"
	"github.com/mattn/go-isatty"
	"github.com/codegangsta/cli"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/logger"
)

const (
	ClientIdentifier = "Geth console"
	Version          = "0.9.27"
)

var (
	gitCommit       string // set via linker flag
	nodeNameVersion string
	app           = utils.NewApp(Version, "the ether console")
)

func init() {
	if gitCommit == "" {
		nodeNameVersion = Version
	} else {
		nodeNameVersion = Version + "-" + gitCommit[:8]
	}

	app.Action = run
	app.Flags = []cli.Flag{
		utils.IPCDisabledFlag,
		utils.IPCPathFlag,
		utils.VerbosityFlag,
		utils.JSpathFlag,
	}

	app.Before = func(ctx *cli.Context) error {
		utils.SetupLogger(ctx)
		return nil
	}
}

func main() {
	// Wrap the standard output with a colorified stream (windows)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		if pr, pw, err := os.Pipe(); err == nil {
			go io.Copy(colorable.NewColorableStdout(), pr)
			os.Stdout = pw
		}
	}

	var interrupted = false
	utils.RegisterInterrupt(func(os.Signal) {
		interrupted = true
	})
	utils.HandleInterrupt()

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error: ", err)
	}

	// we need to run the interrupt callbacks in case gui is closed
	// this skips if we got here by actual interrupt stopping the GUI
	if !interrupted {
		utils.RunInterruptCallbacks(os.Interrupt)
	}
	logger.Flush()
}

func run(ctx *cli.Context) {
	jspath := ctx.GlobalString(utils.JSpathFlag.Name)
	ipcpath := ctx.GlobalString(utils.IPCPathFlag.Name)

	repl := newJSRE(jspath, ipcpath)
	repl.welcome()
	repl.interactive()
}
