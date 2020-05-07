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
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/trie"
	cli "gopkg.in/urfave/cli.v1"
)

var (
	snapshotCommand = cli.Command{
		Name:        "snapshot",
		Usage:       "A set of commands based on the snapshot",
		Category:    "MISCELLANEOUS COMMANDS",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:      "prune-state",
				Usage:     "Prune stale ethereum state data based on snapshot",
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
geth snapshot prune-state <state-root>
will prune historical state data with the help of state snapshot.
All trie nodes that do not belong to the specified version state
will be deleted from the database.
`,
			},
			{
				Name:      "verify-state",
				Usage:     "Recalculate state hash based on snapshot for verification",
				ArgsUsage: "<root>",
				Action:    utils.MigrateFlags(verifyState),
				Category:  "MISCELLANEOUS COMMANDS",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.RopstenFlag,
					utils.RinkebyFlag,
					utils.GoerliFlag,
					utils.LegacyTestnetFlag,
				},
				Description: `
geth snapshot verify-state <state-root>
will traverse the whole accounts and storages set based on the specified
snapshot and recalculate the root hash of state for verification.
`,
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

func verifyState(ctx *cli.Context) error {
	stack := makeFullNode(ctx)
	defer stack.Close()

	chain, chaindb := utils.MakeChain(ctx, stack)
	defer chaindb.Close()

	snaptree, err := snapshot.New(chaindb, trie.NewDatabase(chaindb), 256, chain.CurrentBlock().Root(), false, false)
	if err != nil {
		fmt.Println("Failed to open snapshot tree", "error", err)
		return nil
	}
	if ctx.NArg() > 1 {
		utils.Fatalf("too many arguments given")
	}
	var root = chain.CurrentBlock().Root()
	if ctx.NArg() == 1 {
		root = common.HexToHash(ctx.Args()[0])
	}
	if err := snapshot.VerifyState(snaptree, root); err != nil {
		fmt.Println("Failed to verify state", "error", err)
	} else {
		fmt.Println("Verified the state")
	}
	return nil
}
