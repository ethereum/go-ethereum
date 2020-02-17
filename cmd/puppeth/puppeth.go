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
	"math/rand"
	"os"
	"strings"
	"time"

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
		cli.StringFlag{
			Name:  "consensusType",
			Usage: "Which consensus engine to use? (default = null)\n \t\t1. Ethash - proof-of-work\n \t\t2. Clique - proof-of-authority",
		},
		cli.IntFlag{
			Name:  "blocksTime",
			Usage: "log level to emit to the screen",
		},
		cli.StringFlag{
			Name:  "sealAccounts",
			Usage: "Which accounts are allowed to seal? (mandatory at least one)",
		},
		cli.StringFlag{
			Name:  "preFundedAccounts",
			Usage: "Which accounts should be pre-funded? (advisable at least one)",
		},
		cli.StringFlag{
			Name:  "preCmpAddressWithOneWei",
			Usage: "Should the precompile-addresses (0x1 .. 0xff) be pre-funded with 1 wei?",
		},
		cli.Uint64Flag{
			Name:  "networkID",
			Value: 0,
			Usage: "Specify your chain/network ID if you want an explicit one (default = random)",
		},
	}
	app.Before = func(c *cli.Context) error {
		// Set up the logger to print everything and the random generator
		log.Root().SetHandler(log.LvlFilterHandler(log.Lvl(c.Int("loglevel")), log.StreamHandler(os.Stdout, log.TerminalFormat(true))))
		rand.Seed(time.Now().UnixNano())

		return nil
	}
	app.Action = runWizard
	app.Run(os.Args)
}

// runWizard start the wizard and relinquish control to it.
func runWizard(c *cli.Context) error {
	network := c.String("network")
	if strings.Contains(network, " ") || strings.Contains(network, "-") || strings.ToLower(network) != network {
		log.Crit("No spaces, hyphens or capital letters allowed in network name")
	}
	consensusType := c.String("consensusType")
	blocksTime := uint64(c.Int("blocksTime"))
	sealAccounts := c.String("sealAccounts")
	preFundedAccounts := c.String("preFundedAccounts")
	preCmpAddOneWei := c.String("preCmpAddressWithOneWei")
	networkID := c.Uint64("networkID")
	nonInteract := 0
	if network != "" && consensusType != "" && blocksTime > 0 && sealAccounts != "" && preFundedAccounts != "" && preCmpAddOneWei != "" &&  networkID > 0
	-------nonInteract = 1
	makeWizard(network, consensusType, blocksTime, sealAccounts, preFundedAccounts, preCmpAddOneWei, networkID, nonInteract).run()
	return nil
}
