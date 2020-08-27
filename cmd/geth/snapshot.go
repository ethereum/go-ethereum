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
	"bytes"
	"fmt"
	"math/big"
	"os"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state/pruner"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
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
			{
				Name:      "traverse-state",
				Usage:     "Traverse the state with given root hash for verification",
				ArgsUsage: "<root>",
				Action:    utils.MigrateFlags(traverseState),
				Category:  "MISCELLANEOUS COMMANDS",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.RopstenFlag,
					utils.RinkebyFlag,
					utils.GoerliFlag,
					utils.LegacyTestnetFlag,
				},
				Description: `
geth snapshot traverse-state <state-root>
will traverse the whole trie from the given root and will abort if any referenced
node is missing. This command can be used for trie integrity verification.
`,
			},
			{
				Name:      "traverse-rawstate",
				Usage:     "Traverse the state with given root hash for verification",
				ArgsUsage: "<root>",
				Action:    utils.MigrateFlags(traverseRawState),
				Category:  "MISCELLANEOUS COMMANDS",
				Flags: []cli.Flag{
					utils.DataDirFlag,
					utils.RopstenFlag,
					utils.RinkebyFlag,
					utils.GoerliFlag,
					utils.LegacyTestnetFlag,
				},
				Description: `
geth snapshot traverse-rawstate <state-root>
will traverse the whole trie from the given root and will abort if any referenced
node/code is missing. This command can be used for trie integrity verification.
It's basically identical to traverse-state, but the check granularity is smaller.
`,
			},
		},
	}
)

func pruneState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
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
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chain, chaindb := utils.MakeChain(ctx, stack, true)
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

var (
	// emptyRoot is the known root hash of an empty trie.
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	// emptyCode is the known hash of the empty EVM bytecode.
	emptyCode = crypto.Keccak256(nil)
)

// traverseState is a helper function used for pruning verification.
// Basically it just iterates the trie, ensure all nodes and assoicated
// contract codes are present.
func traverseState(ctx *cli.Context) error {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	_, chaindb := utils.MakeChain(ctx, stack, true)
	defer chaindb.Close()

	if ctx.NArg() > 1 {
		log.Crit("Too many arguments given")
	}
	var root = rawdb.ReadSnapshotRoot(chaindb)
	if ctx.NArg() == 1 {
		root = common.HexToHash(ctx.Args()[0])
	}
	t, err := trie.NewSecure(root, trie.NewDatabase(chaindb))
	if err != nil {
		log.Crit("Failed to open trie", "root", root, "error", err)
	}
	var (
		accounts   int
		slots      int
		codes      int
		lastReport time.Time
		start      = time.Now()
	)
	accIter := trie.NewIterator(t.NodeIterator(nil))
	for accIter.Next() {
		accounts += 1
		var acc struct {
			Nonce    uint64
			Balance  *big.Int
			Root     common.Hash
			CodeHash []byte
		}
		if err := rlp.DecodeBytes(accIter.Value, &acc); err != nil {
			log.Crit("Invalid account encountered during traversal", "error", err)
		}
		if acc.Root != emptyRoot {
			storageTrie, err := trie.NewSecure(acc.Root, trie.NewDatabase(chaindb))
			if err != nil {
				log.Crit("Failed to open storage trie", "root", acc.Root, "error", err)
			}
			storageIter := trie.NewIterator(storageTrie.NodeIterator(nil))
			for storageIter.Next() {
				slots += 1
			}
			if storageIter.Err != nil {
				log.Crit("Failed to traverse storage trie", "root", acc.Root, "error", storageIter.Err)
			}
		}
		if !bytes.Equal(acc.CodeHash, emptyCode) {
			has, _ := chaindb.Has(acc.CodeHash)
			if !has {
				log.Crit("Code is missing", "account", common.BytesToHash(accIter.Key))
			}
			codes += 1
		}
		if time.Since(lastReport) > time.Second*8 {
			log.Info("Traversing state", "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
			lastReport = time.Now()
		}
	}
	if accIter.Err != nil {
		log.Crit("Failed to traverse state trie", "root", root, "error", accIter.Err)
	}
	log.Info("State is complete", "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// traverseRawState is a helper function used for pruning verification.
// Basically it just iterates the trie, ensure all nodes and assoicated
// contract codes are present. It's basically identical to traverseState
// but it will check each trie node.
func traverseRawState(ctx *cli.Context) error {
	glogger := log.NewGlogHandler(log.StreamHandler(os.Stderr, log.TerminalFormat(true)))
	glogger.Verbosity(log.LvlInfo)
	log.Root().SetHandler(glogger)

	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	_, chaindb := utils.MakeChain(ctx, stack, true)
	defer chaindb.Close()

	if ctx.NArg() > 1 {
		log.Crit("Too many arguments given")
	}
	var root = rawdb.ReadSnapshotRoot(chaindb)
	if ctx.NArg() == 1 {
		root = common.HexToHash(ctx.Args()[0])
	}
	t, err := trie.NewSecure(root, trie.NewDatabase(chaindb))
	if err != nil {
		log.Crit("Failed to open trie", "root", root, "error", err)
	}
	log.Info("Opened the state trie", "root", root)
	var (
		nodes      int
		accounts   int
		slots      int
		codes      int
		lastReport time.Time
		start      = time.Now()
	)
	accIter := t.NodeIterator(nil)
	for accIter.Next(true) {
		node := accIter.Hash()
		if node == (common.Hash{}) {
			continue
		}
		nodes += 1

		// If it's a leaf node, yes we are touching an account,
		// dig into the storage trie further.
		if accIter.Leaf() {
			accounts += 1
			var acc struct {
				Nonce    uint64
				Balance  *big.Int
				Root     common.Hash
				CodeHash []byte
			}
			if err := rlp.DecodeBytes(accIter.LeafBlob(), &acc); err != nil {
				log.Crit("Invalid account encountered during traversal", "error", err)
			}
			if acc.Root != emptyRoot {
				storageTrie, err := trie.NewSecure(acc.Root, trie.NewDatabase(chaindb))
				if err != nil {
					log.Crit("Failed to open storage trie", "root", acc.Root, "error", err)
				}
				storageIter := storageTrie.NodeIterator(nil)
				for storageIter.Next(true) {
					node := storageIter.Hash()
					if node == (common.Hash{}) {
						continue
					}
					nodes += 1
					if storageIter.Leaf() {
						slots += 1
					}
				}
				if storageIter.Error() != nil {
					log.Crit("Failed to traverse storage trie", "root", acc.Root, "error", storageIter.Error())
				}
			}
			if !bytes.Equal(acc.CodeHash, emptyCode) {
				code := rawdb.ReadCode(chaindb, common.BytesToHash(acc.CodeHash))
				if len(code) == 0 {
					log.Crit("Code is missing", "account", common.BytesToHash(accIter.LeafKey()))
				}
				codes += 1
			}
			if time.Since(lastReport) > time.Second*8 {
				log.Info("Traversing state", "nodes", nodes, "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
				lastReport = time.Now()
			}
		}
	}
	if accIter.Error() != nil {
		log.Crit("Failed to traverse state trie", "root", root, "error", accIter.Error())
	}
	log.Info("State is complete", "nodes", nodes, "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}
