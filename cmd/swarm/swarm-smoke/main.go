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
	"fmt"
	"os"
	"sort"

	"github.com/ethereum/go-ethereum/log"
	colorable "github.com/mattn/go-colorable"

	cli "gopkg.in/urfave/cli.v1"
)

var (
	endpoints []string
	endpoint  string
	filesize  int
)

func init() {
	for port := 8501; port <= 8512; port++ {
		endpoints = append(endpoints, fmt.Sprintf("http://%v.testing.swarm-gateways.net", port))
	}
}

func main() {
	log.PrintOrigins(true)
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlTrace, log.StreamHandler(colorable.NewColorableStderr(), log.TerminalFormat(true))))

	app := cli.NewApp()
	app.Name = "smoke-test"
	app.Usage = ""

	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:        "bzz-api",
			Value:       endpoints[0],
			Usage:       "upload node endpoint",
			Destination: &endpoint,
		},
		cli.IntFlag{
			Name:        "filesize",
			Value:       1,
			Usage:       "file size for generated random file in MB",
			Destination: &filesize,
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "upload_and_sync",
			Aliases: []string{"c"},
			Usage:   "upload and sync",
			Action:  cliUploadAndSync,
		},
	}

	sort.Sort(cli.FlagsByName(app.Flags))
	sort.Sort(cli.CommandsByName(app.Commands))

	err := app.Run(os.Args)
	if err != nil {
		log.Error(err.Error())
	}
}
