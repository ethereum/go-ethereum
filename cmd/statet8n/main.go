// Copyright 2020 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/cmd/statet8n/machine"
	"github.com/ethereum/go-ethereum/cmd/utils"
	"gopkg.in/urfave/cli.v1"
)

// Git SHA1 commit hash of the release (set via linker flags)
var gitCommit = ""
var gitDate = ""

var (
	app = utils.NewApp(gitCommit, gitDate, "the vladvm command line interface")
)

func init() {
	app.Flags = []cli.Flag{
		machine.TraceFlag,
		machine.TraceDisableMemoryFlag,
		machine.TraceDisableStackFlag,
		machine.OutputAllocFlag,
		machine.OutputResultFlag,
		machine.InputAllocFlag,
		machine.InputEnvFlag,
		machine.InputTxsFlag,
		machine.ForknameFlag,
		machine.ChainIDFlag,
		machine.RewardFlag,
		machine.VerbosityFlag,
	}
	app.Action = machine.StateTransition
}

func main() {
	if err := app.Run(os.Args); err != nil {
		if numbered, ok := err.(*machine.NumberedError); ok {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(numbered.Code())
		}
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
