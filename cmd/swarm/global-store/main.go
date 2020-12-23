// Copyright 2019 The go-ethereum Authors
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

func main() {
	err := newApp().Run(os.Args)
	if err != nil {
		log.Error(err.Error())
		os.Exit(1)
	}
}

// newApp construct a new instance of Swarm Global Store.
// Method Run is called on it in the main function and in tests.
func newApp() (app *cli.App) {
	app = utils.NewApp(gitCommit, "Swarm Global Store")

	app.Name = "global-store"

	// app flags (for all commands)
	app.Flags = []cli.Flag{
		cli.IntFlag{
			Name:  "verbosity",
			Value: 3,
			Usage: "verbosity level",
		},
	}

	app.Commands = []cli.Command{
		{
			Name:    "http",
			Aliases: []string{"h"},
			Usage:   "start swarm global store with http server",
			Action:  startHTTP,
			// Flags only for "start" command.
			// Allow app flags to be specified after the
			// command argument.
			Flags: append(app.Flags,
				cli.StringFlag{
					Name:  "dir",
					Value: "",
					Usage: "data directory",
				},
				cli.StringFlag{
					Name:  "addr",
					Value: "0.0.0.0:3033",
					Usage: "address to listen for http connection",
				},
			),
		},
		{
			Name:    "websocket",
			Aliases: []string{"ws"},
			Usage:   "start swarm global store with websocket server",
			Action:  startWS,
			// Flags only for "start" command.
			// Allow app flags to be specified after the
			// command argument.
			Flags: append(app.Flags,
				cli.StringFlag{
					Name:  "dir",
					Value: "",
					Usage: "data directory",
				},
				cli.StringFlag{
					Name:  "addr",
					Value: "0.0.0.0:3033",
					Usage: "address to listen for websocket connection",
				},
				cli.StringSliceFlag{
					Name:  "origins",
					Value: &cli.StringSlice{"*"},
					Usage: "websocket origins",
				},
			),
		},
	}

	return app
}
