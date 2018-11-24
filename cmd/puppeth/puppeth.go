// Copyright 2017 The go-ethereum Authors
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

// puppeth is a command to assemble and maintain private networks.
package main

import (
	"encoding/json"
	"io/ioutil"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/log"
	"gopkg.in/urfave/cli.v1"
)

// main is just a boring entry point to set up the CLI app.
func main() {
	app := cli.NewApp()
	app.Name = "puppeth"
	app.Usage = "assemble and maintain private Ethereum networks"
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "network",
			Usage: "name of the network to administer (no spaces or hyphens, please)",
		},
		cli.IntFlag{
			Name:  "loglevel",
			Value: 3,
			Usage: "log level to emit to the screen",
		},
	}
	app.Commands = []cli.Command{
		cli.Command{
			Action:    utils.MigrateFlags(convert),
			Name:      "convert",
			Usage:     "Convert from geth genesis into chainspecs for other nodes.",
			ArgsUsage: "<geth-genesis.json>",
		},
	}
	app.Action = func(c *cli.Context) error {
		// Set up the logger to print everything and the random generator
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(c.Int("loglevel")), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
		rand.Seed(time.Now().UnixNano())

		network := c.String("network")
		if strings.Contains(network, " ") || strings.Contains(network, "-") || strings.ToLower(network) != network {
			log.Crit("No spaces, hyphens or capital letters allowed in network name")
		}
		// Start the wizard and relinquish control
		makeWizard(c.String("network")).run()
		return nil
	}
	app.Run(os.Args)
}

func convert(ctx *cli.Context) error {
	// Ensure we have a source genesis
	log.Root().SetHandler(log.LvlFilterHandler(log.LvlInfo, log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
	if len(ctx.Args()) != 1 {
		utils.Fatalf("No geth genesis provided")
	}
	blob, err := ioutil.ReadFile(ctx.Args().First())
	if err != nil {
		utils.Fatalf("Could not read file: %v", err)
	}

	var genesis core.Genesis
	if err := json.Unmarshal(blob, &genesis); err != nil {
		utils.Fatalf("Failed parsing genesis: %v", err)
	}
	basename := strings.TrimRight(ctx.Args().First(), ".json")
	convertGenesis(&genesis, basename, basename, []string{})
	return nil
}
