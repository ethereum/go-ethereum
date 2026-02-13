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
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/archive"
	"github.com/ethereum/go-ethereum/triedb/database"
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
	archiveCheckNodeFlag = &cli.StringFlag{
		Name:  "owner",
		Usage: "Owner hash (hex) for the trie node to check",
	}
	archiveCheckPathFlag = &cli.StringFlag{
		Name:  "path",
		Usage: "Path (hex nibbles) of the trie node to check",
	}

	archiveCommand = &cli.Command{
		Name:  "archive",
		Usage: "Archive state trie nodes to reduce database size",
		Subcommands: []*cli.Command{
			archiveGenerateCmd,
			archiveVerifyCmd,
			archiveDeleteJournalCmd,
			archiveCheckNodeCmd,
		},
	}

	archiveCheckNodeCmd = &cli.Command{
		Name:   "check-node",
		Usage:  "Check if a specific trie node exists in the raw DB",
		Action: archiveCheckNode,
		Flags: slices.Concat([]cli.Flag{
			archiveCheckNodeFlag,
			archiveCheckPathFlag,
		}, utils.NetworkFlags, utils.DatabaseFlags),
	}

	archiveDeleteJournalCmd = &cli.Command{
		Name:   "delete-journal",
		Usage:  "Delete the pathdb journal to force a clean restart",
		Action: archiveDeleteJournal,
		Flags:  slices.Concat(utils.NetworkFlags, utils.DatabaseFlags),
		Description: `
Deletes the pathdb journal (TrieJournal key and merkle.journal file) from the
database. This forces geth to restart with a bare disk layer, discarding any
in-memory diff layers that may be inconsistent with archived state.

Use this after running 'archive generate' if geth was started in between and
recreated the journal.

Examples:
  geth archive delete-journal --datadir /path/to/datadir
  geth archive delete-journal --hoodi
`,
	}

	archiveVerifyCmd = &cli.Command{
		Name:   "verify",
		Usage:  "Verify all archived nodes can be correctly resurrected",
		Action: archiveVerify,
		Flags:  slices.Concat(utils.NetworkFlags, utils.DatabaseFlags),
		Description: `
Walks the entire state trie, resolving every expired node from the archive
file and verifying that the reconstructed subtree hash matches the original.
Also walks all storage tries referenced by accounts.

The database is opened read-only. No modifications are made.

Examples:
  geth archive verify --datadir /path/to/datadir
  geth archive verify --hoodi
`,
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

The archiver reads trie nodes directly from the persistent database layer,
bypassing any in-memory diff layers. This ensures consistency between the
data it reads and the data it modifies.

Examples:
  # Archive from the persistent disk state
  geth archive generate --datadir /path/to/datadir

  # Dry run to see what would be archived
  geth archive generate --dry-run --datadir /path/to/datadir

  # Custom output and compaction interval
  geth archive generate --output /path/to/archive --compaction-interval 500
`,
	}
)

// rawDBNodeReader implements database.NodeReader by reading trie nodes directly
// from the raw key-value database, bypassing pathdb's in-memory diff layers.
// This ensures the archiver sees the same trie state it modifies.
type rawDBNodeReader struct {
	db ethdb.KeyValueReader
}

func (r *rawDBNodeReader) Node(owner common.Hash, path []byte, hash common.Hash) ([]byte, error) {
	var blob []byte
	if owner == (common.Hash{}) {
		blob = rawdb.ReadAccountTrieNode(r.db, path)
	} else {
		blob = rawdb.ReadStorageTrieNode(r.db, owner, path)
	}
	// Skip hash verification: the raw DB may contain expiredNode markers
	// (blob[0] == 0x00) which have different hashes than the original nodes.
	return blob, nil
}

// rawDBNodeDatabase implements database.NodeDatabase using direct raw DB reads.
type rawDBNodeDatabase struct {
	db   ethdb.KeyValueReader
	root common.Hash
}

func (d *rawDBNodeDatabase) NodeReader(stateRoot common.Hash) (database.NodeReader, error) {
	// Only allow reading the persistent disk root state
	if stateRoot != d.root {
		return nil, fmt.Errorf("raw DB reader only supports disk root %x, got %x", d.root, stateRoot)
	}
	return &rawDBNodeReader{db: d.db}, nil
}

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

	// 2. Determine the persistent disk state root.
	//
	// The archiver reads and writes directly to the raw key-value database,
	// bypassing pathdb's in-memory diff layers. This avoids the inconsistency
	// where diff layers shadow expiredNode markers written to disk.
	//
	// The disk root is computed by hashing the account trie root node stored
	// in the raw database. This root corresponds to the last state that was
	// fully persisted (i.e., PersistentStateID), which matches the canonical
	// chain head.
	rootBlob := rawdb.ReadAccountTrieNode(chaindb, nil)
	if len(rootBlob) == 0 {
		return errors.New("state trie not found in database")
	}
	root := crypto.Keccak256Hash(rootBlob)
	log.Info("Using persistent disk state root", "root", root)

	// Create a raw DB node reader that bypasses pathdb layers
	nodeDB := &rawDBNodeDatabase{db: chaindb, root: root}

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
		nodeDB,
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

	if !dryRun {
		// Delete the pathdb journal. The archiver modified the raw DB
		// underneath the diff layers, so the journal's buffered state is
		// inconsistent. Deleting forces geth to restart with a bare disk
		// layer and rewind the chain head to the disk state.
		if err := chaindb.Delete([]byte("TrieJournal")); err != nil {
			log.Warn("Failed to delete pathdb journal key", "err", err)
		}
		log.Info("Deleted pathdb journal to force clean restart")

		// Delete journal file(s) - check both legacy and current locations
		for _, dir := range []string{"triedb", ""} {
			for _, name := range []string{"merkle.journal", "verkle.journal"} {
				journalFile := filepath.Join(stack.ResolvePath(dir), name)
				if err := os.Remove(journalFile); err == nil {
					log.Info("Deleted journal file", "path", journalFile)
				}
			}
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

func archiveVerify(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	// Open database read-only
	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()

	scheme := cycleCheckScheme(ctx, chaindb)
	if scheme != rawdb.PathScheme {
		return fmt.Errorf("archive verify requires path-based state scheme, got: %s", scheme)
	}

	// Set archive data dir so ArchivedNodeResolver can find the file
	// ResolvePath("") returns the node's data directory (e.g. .ethereum/hoodi/geth),
	// but ArchivedNodeResolver expects the instance directory (.ethereum/hoodi)
	// since it appends "geth/nodearchive" itself.
	archive.ArchiveDataDir = filepath.Dir(stack.ResolvePath(""))

	// Compute disk root
	rootBlob := rawdb.ReadAccountTrieNode(chaindb, nil)
	if len(rootBlob) == 0 {
		return errors.New("state trie not found in database")
	}
	root := crypto.Keccak256Hash(rootBlob)
	log.Info("Verifying archived nodes", "root", root)

	nodeDB := &rawDBNodeDatabase{db: chaindb, root: root}

	// Open account trie
	accountTrie, err := trie.New(trie.StateTrieID(root), nodeDB)
	if err != nil {
		return fmt.Errorf("failed to open account trie: %w", err)
	}

	var (
		totalAccounts     int
		totalStorageTries int
		totalLeaves       int
		totalExpired      int
		totalErrors       int
		start             = time.Now()
		lastLog           = time.Now()
	)

	// Walk the account trie — this resolves all expired nodes and verifies hashes
	accountStats, err := accountTrie.Walk(func(path []byte, value []byte) error {
		totalAccounts++
		if time.Since(lastLog) > 30*time.Second {
			log.Info("Verification progress",
				"accounts", totalAccounts,
				"storageTries", totalStorageTries,
				"leaves", totalLeaves,
				"expired", totalExpired,
				"errors", totalErrors)
			lastLog = time.Now()
		}

		// Decode account to check for storage trie
		var acc types.StateAccount
		if err := rlp.DecodeBytes(value, &acc); err != nil {
			log.Warn("Failed to decode account", "err", err)
			totalErrors++
			return nil // continue walking
		}
		if acc.Root == types.EmptyRootHash {
			return nil
		}

		// Open and walk storage trie.
		// path is hex-nibble encoded (with a 16 terminator from the trie key),
		// so convert nibble pairs back to the 32-byte account hash.
		nibbles := path
		if len(nibbles) > 0 && nibbles[len(nibbles)-1] == 16 {
			nibbles = nibbles[:len(nibbles)-1]
		}
		keyBytes := make([]byte, len(nibbles)/2)
		for i := 0; i < len(nibbles); i += 2 {
			keyBytes[i/2] = nibbles[i]<<4 | nibbles[i+1]
		}
		accountHash := common.BytesToHash(keyBytes)
		storageID := trie.StorageTrieID(root, accountHash, acc.Root)
		storageTrie, err := trie.New(storageID, nodeDB)
		if err != nil {
			log.Warn("Failed to open storage trie", "account", accountHash, "err", err)
			totalErrors++
			return nil
		}

		storageStats, err := storageTrie.Walk(func(spath []byte, svalue []byte) error {
			return nil
		})
		if err != nil {
			log.Warn("Storage trie walk failed", "account", accountHash, "err", err)
			totalErrors++
			return nil
		}
		totalStorageTries++
		totalLeaves += storageStats.Leaves
		totalExpired += storageStats.ExpiredResolved
		return nil
	})
	if err != nil {
		return fmt.Errorf("account trie walk failed: %w", err)
	}

	totalLeaves += accountStats.Leaves
	totalExpired += accountStats.ExpiredResolved

	log.Info("Archive verification complete",
		"accounts", totalAccounts,
		"storageTries", totalStorageTries,
		"totalLeaves", totalLeaves,
		"expiredResolved", totalExpired,
		"errors", totalErrors,
		"elapsed", common.PrettyDuration(time.Since(start)))

	if totalErrors > 0 {
		return fmt.Errorf("verification completed with %d errors", totalErrors)
	}
	return nil
}

func archiveDeleteJournal(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, false)
	defer chaindb.Close()

	// Delete the pathdb journal KV key
	if err := chaindb.Delete([]byte("TrieJournal")); err != nil {
		log.Warn("Failed to delete pathdb journal key", "err", err)
	} else {
		log.Info("Deleted pathdb journal key (TrieJournal)")
	}

	// Delete the journal file(s) - check both legacy and current locations
	for _, dir := range []string{"triedb", ""} {
		for _, name := range []string{"merkle.journal", "verkle.journal"} {
			journalFile := filepath.Join(stack.ResolvePath(dir), name)
			if err := os.Remove(journalFile); err == nil {
				log.Info("Deleted journal file", "path", journalFile)
			} else if !os.IsNotExist(err) {
				log.Warn("Failed to delete journal file", "path", journalFile, "err", err)
			}
		}
	}

	return nil
}

func archiveCheckNode(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()

	ownerHex := ctx.String(archiveCheckNodeFlag.Name)
	pathHex := ctx.String(archiveCheckPathFlag.Name)

	if ownerHex == "" {
		return errors.New("--owner flag is required")
	}

	owner := common.HexToHash(ownerHex)

	// Parse path: hex nibbles like "08" → []byte{0, 8}
	var path []byte
	for _, c := range pathHex {
		var nibble byte
		switch {
		case c >= '0' && c <= '9':
			nibble = byte(c - '0')
		case c >= 'a' && c <= 'f':
			nibble = byte(c-'a') + 10
		case c >= 'A' && c <= 'F':
			nibble = byte(c-'A') + 10
		default:
			return fmt.Errorf("invalid hex char in path: %c", c)
		}
		path = append(path, nibble)
	}

	log.Info("Checking node in raw DB", "owner", owner, "path", fmt.Sprintf("%x", path))

	// Read the node directly from the raw DB
	isAccount := owner == (common.Hash{})

	// Check the target path and all prefixes up to root
	for i := len(path); i >= 0; i-- {
		subpath := path[:i]
		var blob []byte
		if isAccount {
			blob = rawdb.ReadAccountTrieNode(chaindb, subpath)
		} else {
			blob = rawdb.ReadStorageTrieNode(chaindb, owner, subpath)
		}

		status := "MISSING"
		details := ""
		if len(blob) > 0 {
			if blob[0] == 0x00 {
				status = "EXPIRED"
				if len(blob) == 17 {
					offset := binary.BigEndian.Uint64(blob[1:9])
					size := binary.BigEndian.Uint64(blob[9:17])
					details = fmt.Sprintf("offset=%d size=%d", offset, size)
				}
			} else {
				status = fmt.Sprintf("PRESENT (%d bytes, first=0x%02x)", len(blob), blob[0])
			}
		}
		label := "prefix"
		if i == len(path) {
			label = "TARGET"
		}
		if i == 0 {
			label = "ROOT"
		}
		log.Info("Node check",
			"label", label,
			"path", fmt.Sprintf("%x", subpath),
			"pathLen", i,
			"status", status,
			"details", details)
	}

	// Also check a few child paths to see what's below the target
	for nibble := byte(0); nibble < 16; nibble++ {
		childPath := append(append([]byte{}, path...), nibble)
		var blob []byte
		if isAccount {
			blob = rawdb.ReadAccountTrieNode(chaindb, childPath)
		} else {
			blob = rawdb.ReadStorageTrieNode(chaindb, owner, childPath)
		}
		if len(blob) > 0 {
			status := fmt.Sprintf("PRESENT (%d bytes, first=0x%02x)", len(blob), blob[0])
			if blob[0] == 0x00 && len(blob) == 17 {
				offset := binary.BigEndian.Uint64(blob[1:9])
				size := binary.BigEndian.Uint64(blob[9:17])
				status = fmt.Sprintf("EXPIRED offset=%d size=%d", offset, size)
			}
			log.Info("Child node", "path", fmt.Sprintf("%x", childPath), "status", status)
		}
	}

	return nil
}

// cycleCheckScheme returns the state scheme for the database.
// It's a helper to check what scheme is in use.
func cycleCheckScheme(ctx *cli.Context, db ethdb.Database) string {
	return rawdb.ReadStateScheme(db)
}
