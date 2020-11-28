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
	"github.com/ethereum/go-ethereum/cmd/utils"
	"gopkg.in/urfave/cli.v1"
)

var (
	catalystCommand = cli.Command{
		Name:  "catalyst",
		Usage: "Set geth into eth1 engine mode",
		Subcommands: []cli.Command{
			{
				Name:     "init",
				Usage:    "initialize geth in eth1 engine mode",
				Action:   initCatalyst,
				Category: "CATALYST COMMANDS",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.KeyStoreDirFlag,
				},
				Description: `
				Initialize geth in eth1 engine mode`,
			},
		},
		Action: catalyst,
	}
)

func initCatalyst(ctx *cli.Context) error {
	return nil
}

func catalyst(ctx *cli.Context) error {
	ctx.GlobalSet(utils.LegacyRPCApiFlag.Name, "eth2,eth")
	ctx.GlobalSet(utils.NoDiscoverFlag.Name, "true")

	// TODO check etherbase is set

	prepare(ctx)
	stack, backend := makeFullNode(ctx)
	defer stack.Close()

	startNode(ctx, stack, backend)
	stack.Wait()
	return nil
}
