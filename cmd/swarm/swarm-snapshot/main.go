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
	"sort"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/log"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	gitCommit string // Git SHA1 commit hash of the release (set via linker flags)
)

var (
	topology  string
	services  string
	pivot     int
	nodes     int
	verbosity int
)

var app = utils.NewApp("", "Swarm Snapshot Util")
var discovery = true

func init() {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(verbosity), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))

	app.Name = "swarm-snapshot"
	app.Usage = ""

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "topology",
			Value:       "chain",
			Usage:       "the desired topology to connect the nodes in (star, ring, chain, full)",
			Destination: &topology,
		},
		cli.IntFlag{
			Name:        "pivot",
			Value:       0,
			Usage:       "pivot node zero-index",
			Destination: &pivot,
		},
		cli.IntFlag{
			Name:        "nodes",
			Value:       10,
			Usage:       "swarm nodes",
			Destination: &nodes,
		},
		cli.IntFlag{
			Name:        "verbosity",
			Value:       1,
			Usage:       "verbosity",
			Destination: &verbosity,
		},
		cli.StringFlag{
			Name:        "services",
			Value:       "",
			Usage:       "comma separated list of services to boot the nodes with",
			Destination: &services,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "create",
			Aliases: []string{"c"},
			Usage:   "create a swarm snapshot",
			Action:  create,
		},
		{
			Name:    "verify",
			Aliases: []string{"v"},
			Usage:   "verify a swarm snapshot",
			Action:  verify,
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))
}

func main() {
	err := app.Run(os.Args)
	if err != nil {
		log.Error(err.Error())

		os.Exit(1)
	}
	os.Exit(0)
}
