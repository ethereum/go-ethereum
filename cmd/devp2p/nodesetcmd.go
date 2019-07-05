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
	"fmt"

	"gopkg.in/urfave/cli.v1"
)

var (
	nodesetCommand = cli.Command{
		Name:  "nodeset",
		Usage: "Node set tools",
		Subcommands: []cli.Command{
			nodesetInfoCommand,
		},
	}
	nodesetInfoCommand = cli.Command{
		Name:      "info",
		Usage:     "Shows statistics about a node set",
		Action:    nodesetInfo,
		ArgsUsage: "<nodes.json>",
	}
)

func nodesetInfo(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		return fmt.Errorf("need nodes file as argument")
	}

	ns := loadNodesJSON(ctx.Args().First())
	fmt.Printf("Set contains %d nodes.\n", len(ns))
	return nil
}
