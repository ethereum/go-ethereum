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

// lespay is the package used to offer auxiliary functionalities relevant
// with les payment protocol
package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/internal/flags"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
)

var (
	// Git SHA1 commit hash of the release (set via linker flags)
	gitCommit = ""
	gitDate   = ""
)

var app *cli.App

func init() {
	app = flags.NewApp(gitCommit, gitDate, "ethereum lottery payment tool")
	app.Commands = []cli.Command{
		commandDeploy,
		// todo channel shutdown command
	}
	app.Flags = []cli.Flag{
		nodeURLFlag,
		clefURLFlag,
		signerFlag,
	}
	cli.CommandHelpTemplate = flags.OriginCommandHelpTemplate
}

// Commonly used command line flags.
var (
	nodeURLFlag = cli.StringFlag{
		Name:  "rpc",
		Value: "http://localhost:8545",
		Usage: "The rpc endpoint of a local or remote geth node",
	}
	clefURLFlag = cli.StringFlag{
		Name:  "clef",
		Value: "http://localhost:8550",
		Usage: "The rpc endpoint of clef",
	}
	signerFlag = cli.StringFlag{
		Name:  "signer",
		Usage: "Signer address for clef signing",
	}
)

func main() {
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stderr, log.TerminalFormat(true))))
	fdlimit.Raise(2048)

	if err := app.Run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
