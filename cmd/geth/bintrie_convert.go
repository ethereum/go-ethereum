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
	"runtime"
	"runtime/debug"
	"slices"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/trie/trienode"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/urfave/cli/v2"
)

var (
	deleteSourceFlag = &cli.BoolFlag{
		Name:  "delete-source",
		Usage: "Delete MPT trie nodes after conversion",
	}
	memoryLimitFlag = &cli.Uint64Flag{
		Name:  "memory-limit",
		Usage: "Max heap allocation in MB before forcing a commit cycle",
		Value: 16384,
	}

	bintrieCommand = &cli.Command{
		Name:        "bintrie",
		Usage:       "A set of commands for binary trie operations",
		Description: "",
		Subcommands: []*cli.Command{
			{
				Name:      "convert",
				Usage:     "Convert MPT state to binary trie",
				ArgsUsage: "[state-root]",
				Action:    convertToBinaryTrie,
				Flags: slices.Concat([]cli.Flag{
					deleteSourceFlag,
					memoryLimitFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth bintrie convert [--delete-source] [--memory-limit MB] [state-root]

Reads all state from the Merkle Patricia Trie and writes it into a Binary Trie,
operating offline. Memory-safe via periodic commit-and-reload cycles.

The optional state-root argument specifies which state root to convert.
If omitted, the head block's state root is used.

Flags:
  --delete-source    Delete MPT trie nodes after successful conversion
  --memory-limit     Max heap allocation in MB before forcing a commit (default: 16384)
`,
			},
		},
	}
)

type conversionStats struct {
	accounts   uint64
	slots      uint64
	codes      uint64
	commits    uint64
	start      time.Time
	lastReport time.Time
	lastMemChk time.Time
}

func (s *conversionStats) report(force bool) {
	if !force && time.Since(s.lastReport) < 8*time.Second {
		return
	}
	elapsed := time.Since(s.start).Seconds()
	acctRate := float64(0)
	if elapsed > 0 {
		acctRate = float64(s.accounts) / elapsed
	}
	log.Info("Conversion progress",
		"accounts", s.accounts,
		"slots", s.slots,
		"codes", s.codes,
		"commits", s.commits,
		"accounts/sec", fmt.Sprintf("%.0f", acctRate),
		"elapsed", common.PrettyDuration(time.Since(s.start)),
	)
	s.lastReport = time.Now()
}

func convertToBinaryTrie(ctx *cli.Context) error {
	if ctx.NArg() > 1 {
		return errors.New("too many arguments")
	}
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, false)
	defer chaindb.Close()

	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		return errors.New("no head block found")
	}
	var (
		root common.Hash
		err  error
	)
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			return fmt.Errorf("invalid state root: %w", err)
		}
	} else {
		root = headBlock.Root()
	}
	log.Info("Starting MPT to binary trie conversion", "root", root, "block", headBlock.NumberU64())

	srcTriedb := utils.MakeTrieDatabase(ctx, stack, chaindb, true, true, false)
	defer srcTriedb.Close()

	destTriedb := triedb.NewDatabase(chaindb, &triedb.Config{
		IsUBT: true,
		PathDB: &pathdb.Config{
			JournalDirectory: stack.ResolvePath("triedb-bintrie"),
		},
	})
	defer destTriedb.Close()

	binTrie, err := bintrie.NewBinaryTrie(types.EmptyBinaryHash, destTriedb)
	if err != nil {
		return fmt.Errorf("failed to create binary trie: %w", err)
	}
	memLimit := ctx.Uint64(memoryLimitFlag.Name) * 1024 * 1024

	currentRoot, err := runConversionLoop(chaindb, srcTriedb, destTriedb, binTrie, root, memLimit)
	if err != nil {
		return err
	}
	log.Info("Conversion complete", "binaryRoot", currentRoot)

	if ctx.Bool(deleteSourceFlag.Name) {
		log.Info("Deleting source MPT data")
		if err := deleteMPTData(chaindb, srcTriedb, root); err != nil {
			return fmt.Errorf("MPT deletion failed: %w", err)
		}
		log.Info("Source MPT data deleted")
	}
	return nil
}

func runConversionLoop(chaindb ethdb.Database, srcTriedb *triedb.Database, destTriedb *triedb.Database, binTrie *bintrie.BinaryTrie, root common.Hash, memLimit uint64) (common.Hash, error) {
	currentRoot := types.EmptyBinaryHash
	stats := &conversionStats{
		start:      time.Now(),
		lastReport: time.Now(),
		lastMemChk: time.Now(),
	}

	srcTrie, err := trie.NewStateTrie(trie.StateTrieID(root), srcTriedb)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to open source trie: %w", err)
	}
	acctIt, err := srcTrie.NodeIterator(nil)
	if err != nil {
		return common.Hash{}, fmt.Errorf("failed to create account iterator: %w", err)
	}
	accIter := trie.NewIterator(acctIt)

	for accIter.Next() {
		var acc types.StateAccount
		if err := rlp.DecodeBytes(accIter.Value, &acc); err != nil {
			return common.Hash{}, fmt.Errorf("invalid account RLP: %w", err)
		}
		addrBytes := srcTrie.GetKey(accIter.Key)
		if addrBytes == nil {
			return common.Hash{}, fmt.Errorf("missing preimage for account hash %x (run with --cache.preimages)", accIter.Key)
		}
		addr := common.BytesToAddress(addrBytes)

		var code []byte
		codeHash := common.BytesToHash(acc.CodeHash)
		if codeHash != types.EmptyCodeHash {
			code = rawdb.ReadCode(chaindb, codeHash)
			if code == nil {
				return common.Hash{}, fmt.Errorf("missing code for hash %x (account %x)", codeHash, addr)
			}
			stats.codes++
		}

		if err := binTrie.UpdateAccount(addr, &acc, len(code)); err != nil {
			return common.Hash{}, fmt.Errorf("failed to update account %x: %w", addr, err)
		}
		if len(code) > 0 {
			if err := binTrie.UpdateContractCode(addr, codeHash, code); err != nil {
				return common.Hash{}, fmt.Errorf("failed to update code for %x: %w", addr, err)
			}
		}

		if acc.Root != types.EmptyRootHash {
			addrHash := common.BytesToHash(accIter.Key)
			storageTrie, err := trie.NewStateTrie(trie.StorageTrieID(root, addrHash, acc.Root), srcTriedb)
			if err != nil {
				return common.Hash{}, fmt.Errorf("failed to open storage trie for %x: %w", addr, err)
			}
			storageNodeIt, err := storageTrie.NodeIterator(nil)
			if err != nil {
				return common.Hash{}, fmt.Errorf("failed to create storage iterator for %x: %w", addr, err)
			}
			storageIter := trie.NewIterator(storageNodeIt)

			slotCount := uint64(0)
			for storageIter.Next() {
				slotKey := storageTrie.GetKey(storageIter.Key)
				if slotKey == nil {
					return common.Hash{}, fmt.Errorf("missing preimage for storage key %x (account %x)", storageIter.Key, addr)
				}
				_, content, _, err := rlp.Split(storageIter.Value)
				if err != nil {
					return common.Hash{}, fmt.Errorf("invalid storage RLP for key %x (account %x): %w", slotKey, addr, err)
				}
				if err := binTrie.UpdateStorage(addr, slotKey, content); err != nil {
					return common.Hash{}, fmt.Errorf("failed to update storage %x/%x: %w", addr, slotKey, err)
				}
				stats.slots++
				slotCount++

				if slotCount%10000 == 0 {
					binTrie, currentRoot, err = maybeCommit(binTrie, currentRoot, destTriedb, memLimit, stats)
					if err != nil {
						return common.Hash{}, err
					}
				}
			}
			if storageIter.Err != nil {
				return common.Hash{}, fmt.Errorf("storage iteration error for %x: %w", addr, storageIter.Err)
			}
		}
		stats.accounts++
		stats.report(false)

		if stats.accounts%1000 == 0 {
			binTrie, currentRoot, err = maybeCommit(binTrie, currentRoot, destTriedb, memLimit, stats)
			if err != nil {
				return common.Hash{}, err
			}
		}
	}
	if accIter.Err != nil {
		return common.Hash{}, fmt.Errorf("account iteration error: %w", accIter.Err)
	}

	_, currentRoot, err = commitBinaryTrie(binTrie, currentRoot, destTriedb)
	if err != nil {
		return common.Hash{}, fmt.Errorf("final commit failed: %w", err)
	}
	stats.commits++
	stats.report(true)
	return currentRoot, nil
}

func maybeCommit(bt *bintrie.BinaryTrie, currentRoot common.Hash, destDB *triedb.Database, memLimit uint64, stats *conversionStats) (*bintrie.BinaryTrie, common.Hash, error) {
	if time.Since(stats.lastMemChk) < 5*time.Second {
		return bt, currentRoot, nil
	}
	stats.lastMemChk = time.Now()

	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	if m.Alloc < memLimit {
		return bt, currentRoot, nil
	}
	log.Info("Memory limit reached, committing", "alloc", common.StorageSize(m.Alloc), "limit", common.StorageSize(memLimit))

	bt, currentRoot, err := commitBinaryTrie(bt, currentRoot, destDB)
	if err != nil {
		return nil, common.Hash{}, err
	}
	stats.commits++
	stats.report(true)
	return bt, currentRoot, nil
}

func commitBinaryTrie(bt *bintrie.BinaryTrie, currentRoot common.Hash, destDB *triedb.Database) (*bintrie.BinaryTrie, common.Hash, error) {
	newRoot, nodeSet := bt.Commit(false)
	if nodeSet != nil {
		merged := trienode.NewWithNodeSet(nodeSet)
		if err := destDB.Update(newRoot, currentRoot, 0, merged, triedb.NewStateSet()); err != nil {
			return nil, common.Hash{}, fmt.Errorf("triedb update failed: %w", err)
		}
		if err := destDB.Commit(newRoot, false); err != nil {
			return nil, common.Hash{}, fmt.Errorf("triedb commit failed: %w", err)
		}
	}
	runtime.GC()
	debug.FreeOSMemory()

	bt, err := bintrie.NewBinaryTrie(newRoot, destDB)
	if err != nil {
		return nil, common.Hash{}, fmt.Errorf("failed to reload binary trie: %w", err)
	}
	return bt, newRoot, nil
}

func deleteMPTData(chaindb ethdb.Database, srcTriedb *triedb.Database, root common.Hash) error {
	isPathDB := srcTriedb.Scheme() == rawdb.PathScheme

	srcTrie, err := trie.NewStateTrie(trie.StateTrieID(root), srcTriedb)
	if err != nil {
		return fmt.Errorf("failed to open source trie for deletion: %w", err)
	}
	acctIt, err := srcTrie.NodeIterator(nil)
	if err != nil {
		return fmt.Errorf("failed to create account iterator for deletion: %w", err)
	}
	batch := chaindb.NewBatch()
	deleted := 0

	for acctIt.Next(true) {
		if isPathDB {
			rawdb.DeleteAccountTrieNode(batch, acctIt.Path())
		} else {
			node := acctIt.Hash()
			if node != (common.Hash{}) {
				rawdb.DeleteLegacyTrieNode(batch, node)
			}
		}
		deleted++

		if acctIt.Leaf() {
			var acc types.StateAccount
			if err := rlp.DecodeBytes(acctIt.LeafBlob(), &acc); err != nil {
				return fmt.Errorf("invalid account during deletion: %w", err)
			}
			if acc.Root != types.EmptyRootHash {
				addrHash := common.BytesToHash(acctIt.LeafKey())
				storageTrie, err := trie.NewStateTrie(trie.StorageTrieID(root, addrHash, acc.Root), srcTriedb)
				if err != nil {
					return fmt.Errorf("failed to open storage trie for deletion: %w", err)
				}
				storageIt, err := storageTrie.NodeIterator(nil)
				if err != nil {
					return fmt.Errorf("failed to create storage iterator for deletion: %w", err)
				}
				for storageIt.Next(true) {
					if isPathDB {
						rawdb.DeleteStorageTrieNode(batch, addrHash, storageIt.Path())
					} else {
						node := storageIt.Hash()
						if node != (common.Hash{}) {
							rawdb.DeleteLegacyTrieNode(batch, node)
						}
					}
					deleted++
					if batch.ValueSize() >= ethdb.IdealBatchSize {
						if err := batch.Write(); err != nil {
							return fmt.Errorf("batch write failed: %w", err)
						}
						batch.Reset()
					}
				}
				if storageIt.Error() != nil {
					return fmt.Errorf("storage deletion iterator error: %w", storageIt.Error())
				}
			}
		}
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return fmt.Errorf("batch write failed: %w", err)
			}
			batch.Reset()
		}
	}
	if acctIt.Error() != nil {
		return fmt.Errorf("account deletion iterator error: %w", acctIt.Error())
	}
	if batch.ValueSize() > 0 {
		if err := batch.Write(); err != nil {
			return fmt.Errorf("final batch write failed: %w", err)
		}
	}
	log.Info("MPT deletion complete", "nodesDeleted", deleted)
	return nil
}
