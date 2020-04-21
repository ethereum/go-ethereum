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
	"fmt"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state/pruner"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	pruningCommand = cli.Command{
		Name:        "prune",
		Usage:       "Prune ethereum historical or stale data",
		ArgsUsage:   "",
		Category:    "MISCELLANEOUS COMMANDS",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:      "state",
				Usage:     "Prune stale ethereum state data",
				ArgsUsage: "<root>",
				Action:    utils.MigrateFlags(pruneState),
				Category:  "MISCELLANEOUS COMMANDS",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.RopstenFlag,
					utils.RinkebyFlag,
					utils.GoerliFlag,
					utils.LegacyTestnetFlag,
				},
				Description: `
geth prune state <state-root>
will prune historical state data with the help of state snapshot.
All trie nodes which not belong to the state snapshot will be delete
from the database.
`,
			},
			{
				Name:      "chain",
				Usage:     "Prune historical ethereum chain data",
				ArgsUsage: "",
				Action:    utils.MigrateFlags(pruneChain),
				Category:  "MISCELLANEOUS COMMANDS",
				Flags: []cli.Flag{
					utils.DataDirFlag,
				},
				Description: ``,
			},
		},
	}
)

func pruneState(ctx *cli.Context) error {
	stack, _ := makeFullNode(ctx)
	defer stack.Close()

	chain, chaindb := utils.MakeChain(ctx, stack, true)
	defer chaindb.Close()

	pruner, err := pruner.NewPruner(chaindb, chain.CurrentBlock().Root(), stack.ResolvePath(""))
	fmt.Println(stack.ResolvePath(""))
	if err != nil {
		utils.Fatalf("Failed to open snapshot tree %v", err)
	}
	if ctx.NArg() > 1 {
		utils.Fatalf("too many arguments given")
	}
	var root common.Hash
	if ctx.NArg() == 1 {
		root = common.HexToHash(ctx.Args()[0])
	}
	err = pruner.Prune(root)
	if err != nil {
		utils.Fatalf("Failed to prune state", "error", err)
	}
	return nil
}

func pruneChain(ctx *cli.Context) error {
	return nil
}
