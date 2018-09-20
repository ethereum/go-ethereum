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

// registrar is a utility that can be used to query checkpoint information
// and register stable checkpoint into the contract.
package main

import (
	"fmt"
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common/fdlimit"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
)

const (
	commandHelperTemplate = `{{.Name}}{{if .Subcommands}} command{{end}}{{if .Flags}} [command options]{{end}} [arguments...]
{{if .Description}}{{.Description}}
{{end}}{{if .Subcommands}}
SUBCOMMANDS:
	{{range .Subcommands}}{{.Name}}{{with .ShortName}}, {{.}}{{end}}{{ "\t" }}{{.Usage}}
	{{end}}{{end}}{{if .Flags}}
OPTIONS:
{{range $.Flags}}{{"\t"}}{{.}}
{{end}}
{{end}}`
)

// Git SHA1 commit hash of the release (set via linker flags)
var gitCommit = ""

var app *cli.App

func init() {
	app = utils.NewApp(gitCommit, "ethereum checkpoint helper tool")
	app.Commands = []cli.Command{
		commandQueryAdmin,
		commandQueryCheckpoint,
		commandPendingProposal,
		commandDeployContract,
		commandRegisterCheckpoint,
	}
	app.Flags = []cli.Flag{
		checkpointIndexFlag,
		sigThresholdFlag,
		keyFileFlag,
		clientURLFlag,
		trustedSignerFlag,
		utils.PasswordFileFlag,
	}
	cli.CommandHelpTemplate = commandHelperTemplate
}

// Commonly used command line flags.
var (
	checkpointIndexFlag = cli.Int64Flag{
		Name:  "index",
		Usage: "The index of checkpoint, use the latest index if not specified",
	}
	sigThresholdFlag = cli.Int64Flag{
		Name:  "threshold",
		Usage: "The minimal signature required to approve a checkpoint",
	}
	keyFileFlag = cli.StringFlag{
		Name:  "keyfile",
		Usage: "The private key file",
	}
	clientURLFlag = cli.StringFlag{
		Name:  "rpc",
		Value: "http://localhost:8545",
		Usage: "The rpc endpoint of a local or remote geth node",
	}
	trustedSignerFlag = cli.StringFlag{
		Name:  "signer",
		Usage: "Comma separated accounts to treat as trusted checkpoint signer",
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
