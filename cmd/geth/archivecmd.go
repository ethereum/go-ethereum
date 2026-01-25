// Copyright 2026 The go-ethereum Authors
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
	"errors"
	"fmt"
	"path/filepath"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/archive"
	"github.com/urfave/cli/v2"
)

var (
	// Flags for the archive command
	archiveOutputFlag = &cli.StringFlag{
		Name:  "output",
		Usage: "Path to archive output file",
		Value: "", // Default: <datadir>/nodearchive
	}
	archiveCompactionIntervalFlag = &cli.Uint64Flag{
		Name:  "compaction-interval",
		Usage: "Run compaction after this many subtrees (0 = disable)",
		Value: 1000,
	}
	archiveDryRunFlag = &cli.BoolFlag{
		Name:  "dry-run",
		Usage: "Simulate without modifying database",
	}

	// Commands
	archiveCommand = &cli.Command{
		Name:  "archive",
		Usage: "Archive state trie nodes to reduce database size",
		Subcommands: []*cli.Command{
			archiveGenerateCmd,
		},
	}

	archiveGenerateCmd = &cli.Command{
		Name:      "generate",
		Usage:     "Generate archive files from height-3 subtrees",
		ArgsUsage: "[state-root]",
		Action:    archiveGenerate,
		Flags: slices.Concat([]cli.Flag{
			archiveOutputFlag,
			archiveCompactionIntervalFlag,
			archiveDryRunFlag,
		}, utils.NetworkFlags, utils.DatabaseFlags),
		Description: `
Walks the state trie of the specified root (or head block) and archives
subtrees at height 3. Each archived subtree is replaced with an expiredNode
that references the archive file offset and size.

Height is measured from leaves: leaves=0, parents=1, etc. A height-3 node
has leaves at most 3 levels below it.

Examples:
  # Archive from head state
  geth archive generate --datadir /path/to/datadir

  # Dry run to see what would be archived
  geth archive generate --dry-run --datadir /path/to/datadir

  # Archive from a specific state root
  geth archive generate 0x1234...abcd --datadir /path/to/datadir

  # Custom output and compaction interval
  geth archive generate --output /path/to/archive --compaction-interval 500
`,
	}
)

func archiveGenerate(ctx *cli.Context) error {
	// 1. Setup node and databases
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	// Open database in write mode (readOnly=false) unless dry-run
	dryRun := ctx.Bool(archiveDryRunFlag.Name)
	chaindb := utils.MakeChainDatabase(ctx, stack, dryRun)
	defer chaindb.Close()

	// Check state scheme - we only support PathDB
	scheme := cycleCheckScheme(ctx, chaindb)
	if scheme != rawdb.PathScheme {
		return fmt.Errorf("archive generation requires path-based state scheme, got: %s", scheme)
	}

	triedb := utils.MakeTrieDatabase(ctx, stack, chaindb, false, false, false)
	defer triedb.Close()

	// 2. Determine state root
	var root common.Hash
	if ctx.NArg() > 0 {
		root = common.HexToHash(ctx.Args().First())
		log.Info("Using specified state root", "root", root)
	} else {
		headBlock := rawdb.ReadHeadBlock(chaindb)
		if headBlock == nil {
			return errors.New("no head block found - specify a state root or sync the chain first")
		}
		root = headBlock.Root()
		log.Info("Using head block state", "number", headBlock.NumberU64(), "root", root)
	}

	// Verify the state exists
	if !rawdb.HasAccountTrieNode(chaindb, nil) {
		return errors.New("state trie not found in database")
	}

	// 3. Open archive writer (unless dry-run)
	var writer *archive.ArchiveWriter
	archivePath := ctx.String(archiveOutputFlag.Name)
	if archivePath == "" {
		archivePath = filepath.Join(stack.ResolvePath(""), "nodearchive")
	}

	if !dryRun {
		var err error
		writer, err = archive.NewArchiveWriter(archivePath)
		if err != nil {
			return fmt.Errorf("failed to open archive file %s: %w", archivePath, err)
		}
		defer writer.Close()
		log.Info("Opened archive file", "path", archivePath)
	} else {
		log.Info("Dry run mode - no changes will be made")
	}

	// 4. Create and run archiver
	archiver := trie.NewArchiver(
		chaindb,
		triedb,
		writer,
		ctx.Uint64(archiveCompactionIntervalFlag.Name),
		dryRun,
	)

	start := time.Now()
	if err := archiver.ProcessState(root); err != nil {
		return fmt.Errorf("archive generation failed: %w", err)
	}

	// 5. Get stats and optionally run final compaction
	subtrees, leaves, bytesDeleted := archiver.Stats()

	if !dryRun && subtrees > 0 {
		log.Info("Running final database compaction")
		if err := chaindb.Compact(nil, nil); err != nil {
			log.Warn("Final compaction failed", "err", err)
		}
	}

	// 6. Print summary
	var archiveSize uint64
	if writer != nil {
		archiveSize = writer.Offset()
	}

	log.Info("Archive generation complete",
		"subtrees", subtrees,
		"leaves", leaves,
		"bytesDeleted", bytesDeleted,
		"archiveSize", archiveSize,
		"elapsed", common.PrettyDuration(time.Since(start)))

	if dryRun {
		log.Info("This was a dry run - no changes were made to the database")
	}

	return nil
}

// cycleCheckScheme returns the state scheme for the database.
// It's a helper to check what scheme is in use.
func cycleCheckScheme(ctx *cli.Context, db ethdb.Database) string {
	return rawdb.ReadStateScheme(db)
}
