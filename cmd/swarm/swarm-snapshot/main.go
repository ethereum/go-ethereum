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

package main

import (
	"os"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	cli "gopkg.in/urfave/cli.v1"
)

var gitCommit string // Git SHA1 commit hash of the release (set via linker flags)

// default value for "create" command --nodes flag
const defaultNodes = 8

func main() {
	err := newApp().Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

// newApp construct a new instance of Swarm Snapshot Utility.
// Method Run is called on it in the main function and in tests.
func newApp() (app *cli.App) {
	app = utils.NewApp(gitCommit, "Swarm Snapshot Utility")

	app.Name = "swarm-snapshot"
	app.Usage = ""

	// app flags (for all commands)
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "verbosity",
			Value: 1,
			Usage: "verbosity level",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "create a swarm snapshot",
			Action:  create,
			// Flags only for "create" command.
			// Allow app flags to be specified after the
			// command argument.
			Flags: append(app.Flags,
				cli.IntFlag{
					Name:  "nodes",
					Value: defaultNodes,
					Usage: "number of nodes",
				},
				cli.StringFlag{
					Name:  "services",
					Value: "bzz",
					Usage: "comma separated list of services to boot the nodes with",
				},
			),
		},
	}

	return app
}
