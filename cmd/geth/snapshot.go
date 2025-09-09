// Copyright 2021 The go-ethereum Authors
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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sync/atomic"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/pruner"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/urfave/cli/v2"
)

var (
	snapshotCommand = &cli.Command{
		Name:        "snapshot",
		Usage:       "A set of commands based on the snapshot",
		Description: "",
		Subcommands: []*cli.Command{
			{
				Name:      "prune-state",
				Usage:     "Prune stale ethereum state data based on the snapshot",
				ArgsUsage: "<root>",
				Action:    pruneState,
				Flags: slices.Concat([]cli.Flag{
					utils.BloomFilterSizeFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot prune-state <state-root>
will prune historical state data with the help of the state snapshot.
All trie nodes and contract codes that do not belong to the specified
version state will be deleted from the database. After pruning, only
two version states are available: genesis and the specific one.

The default pruning target is the HEAD-127 state.

WARNING: it's only supported in hash mode(--state.scheme=hash)".
`,
			},
			{
				Name:      "verify-state",
				Usage:     "Recalculate state hash based on the snapshot for verification",
				ArgsUsage: "<root>",
				Action:    verifyState,
				Flags:     slices.Concat(utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot verify-state <state-root>
will traverse the whole accounts and storages set based on the specified
snapshot and recalculate the root hash of state for verification.
In other words, this command does the snapshot to trie conversion.
`,
			},
			{
				Name:      "check-dangling-storage",
				Usage:     "Check that there is no 'dangling' snap storage",
				ArgsUsage: "<root>",
				Action:    checkDanglingStorage,
				Flags:     slices.Concat(utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot check-dangling-storage <state-root> traverses the snap storage
data, and verifies that all snapshot storage data has a corresponding account.
`,
			},
			{
				Name:      "inspect-account",
				Usage:     "Check all snapshot layers for the specific account",
				ArgsUsage: "<address | hash>",
				Action:    checkAccount,
				Flags:     slices.Concat(utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot inspect-account <address | hash> checks all snapshot layers and prints out
information about the specified address.
`,
			},
			{
				Name:      "traverse-state",
				Usage:     "Traverse the state with given root hash and perform quick verification",
				ArgsUsage: "<root>",
				Action:    traverseState,
				Flags:     slices.Concat(utils.TraverseStateFlags, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot traverse-state [--account <account>] [--start <key>] [--limit <key>] <state-root>

1. Traverse the whole state from the given state root:
- --start: starting account key (64/66 chars hex) [optional]
- --limit: ending account key (64/66 chars hex) [optional]

2. Traverse a specific account's storage:
- --account: account address (40/42 chars) or hash (64/66 chars) [required]
- --start: starting storage key (64/66 chars hex) [optional]
- --limit: ending storage key (64/66 chars hex) [optional]

The default checking state root is the HEAD state if not specified.
The command will abort if any referenced trie node or contract code is missing.
This can be used for state integrity verification. The default target is HEAD state.

It's also usable without snapshot enabled.
`,
			},
			{
				Name:      "traverse-rawstate",
				Usage:     "Traverse the state with given root hash and perform detailed verification",
				ArgsUsage: "<root>",
				Action:    traverseRawState,
				Flags:     slices.Concat(utils.TraverseStateFlags, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot traverse-rawstate [--account <account>] [--start <key>] [--limit <key>] <state-root>

Similar to traverse-state but with more detailed verification at the trie node level.

1. Traverse the whole state from the given state root:
- --start: starting account key (64/66 chars hex) [optional]
- --limit: ending account key (64/66 chars hex) [optional]

2. Traverse a specific account's storage:
- --account: account address (40/42 chars) or hash (64/66 chars) [required]
- --start: starting storage key (64/66 chars hex) [optional]
- --limit: ending storage key (64/66 chars hex) [optional]

The default checking state root is the HEAD state if not specified.
The command will abort if any referenced trie node or contract code is missing.
This can be used for state integrity verification. The default target is HEAD state.

It's also usable without snapshot enabled.
`,
			},
			{
				Name:      "dump",
				Usage:     "Dump a specific block from storage (same as 'geth dump' but using snapshots)",
				ArgsUsage: "[? <blockHash> | <blockNum>]",
				Action:    dumpState,
				Flags: slices.Concat([]cli.Flag{
					utils.ExcludeCodeFlag,
					utils.ExcludeStorageFlag,
					utils.StartKeyFlag,
					utils.DumpLimitFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
This command is semantically equivalent to 'geth dump', but uses the snapshots
as the backend data source, making this command a lot faster.

The argument is interpreted as block number or hash. If none is provided, the latest
block is used.
`,
			},
			{
				Action:    snapshotExportPreimages,
				Name:      "export-preimages",
				Usage:     "Export the preimage in snapshot enumeration order",
				ArgsUsage: "<dumpfile> [<root>]",
				Flags:     utils.DatabaseFlags,
				Description: `
The export-preimages command exports hash preimages to a flat file, in exactly
the expected order for the overlay tree migration.
`,
			},
		},
	}
)

// Deprecation: this command should be deprecated once the hash-based
// scheme is deprecated.
func pruneState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, false)
	defer chaindb.Close()

	if rawdb.ReadStateScheme(chaindb) != rawdb.HashScheme {
		log.Crit("Offline pruning is not required for path scheme")
	}
	prunerconfig := pruner.Config{
		Datadir:   stack.ResolvePath(""),
		BloomSize: ctx.Uint64(utils.BloomFilterSizeFlag.Name),
	}
	pruner, err := pruner.NewPruner(chaindb, prunerconfig)
	if err != nil {
		log.Error("Failed to open snapshot tree", "err", err)
		return err
	}
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var targetRoot common.Hash
	if ctx.NArg() == 1 {
		targetRoot, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "err", err)
			return err
		}
	}
	if err = pruner.Prune(targetRoot); err != nil {
		log.Error("Failed to prune state", "err", err)
		return err
	}
	return nil
}

func verifyState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()

	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return errors.New("no head block")
	}
	triedb := utils.MakeTrieDatabase(ctx, stack, chaindb, false, true, false)
	defer triedb.Close()

	var (
		err  error
		root = headBlock.Root()
	)
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "err", err)
			return err
		}
	}
	if triedb.Scheme() == rawdb.PathScheme {
		if err := triedb.VerifyState(root); err != nil {
			log.Error("Failed to verify state", "root", root, "err", err)
			return err
		}
		log.Info("Verified the state", "root", root)

		// TODO(rjl493456442) implement dangling checks in pathdb.
		return nil
	} else {
		snapConfig := snapshot.Config{
			CacheSize:  256,
			Recovery:   false,
			NoBuild:    true,
			AsyncBuild: false,
		}
		snaptree, err := snapshot.New(snapConfig, chaindb, triedb, headBlock.Root())
		if err != nil {
			log.Error("Failed to open snapshot tree", "err", err)
			return err
		}
		if err := snaptree.Verify(root); err != nil {
			log.Error("Failed to verify state", "root", root, "err", err)
			return err
		}
		log.Info("Verified the state", "root", root)
		return snapshot.CheckDanglingStorage(chaindb)
	}
}

// checkDanglingStorage iterates the snap storage data, and verifies that all
// storage also has corresponding account data.
func checkDanglingStorage(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	db := utils.MakeChainDatabase(ctx, stack, true)
	defer db.Close()
	return snapshot.CheckDanglingStorage(db)
}

// traverseState is a helper function used for pruning verification.
// Basically it just iterates the trie, ensure all nodes and associated
// contract codes are present.
func traverseState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()

	triedb := utils.MakeTrieDatabase(ctx, stack, chaindb, false, true, false)
	defer triedb.Close()

	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return errors.New("no head block")
	}

	config, err := parseTraverseArgs(ctx)
	if err != nil {
		return err
	}
	if config.root == (common.Hash{}) {
		config.root = headBlock.Root()
	}

	t, err := trie.NewStateTrie(trie.StateTrieID(config.root), triedb)
	if err != nil {
		log.Error("Failed to open trie", "root", config.root, "err", err)
		return err
	}

	var (
		accounts atomic.Uint64
		slots    atomic.Uint64
		codes    atomic.Uint64
		start    = time.Now()
	)

	go func() {
		timer := time.NewTicker(time.Second * 8)
		defer timer.Stop()
		for range timer.C {
			log.Info("Traversing state", "accounts", accounts.Load(), "slots", slots.Load(), "codes", codes.Load(), "elapsed", common.PrettyDuration(time.Since(start)))
		}
	}()

	if config.isAccount {
		log.Info("Start traversing storage trie", "root", config.root.Hex(), "account", config.account.Hex(), "startKey", common.Bytes2Hex(config.startKey), "limitKey", common.Bytes2Hex(config.limitKey))

		acc, err := t.GetAccountByHash(config.account)
		if err != nil {
			log.Error("Get account failed", "account", config.account.Hex(), "err", err)
			return err
		}

		if acc.Root == types.EmptyRootHash {
			log.Info("Account has no storage")
			return nil
		}

		id := trie.StorageTrieID(config.root, config.account, acc.Root)
		storageTrie, err := trie.NewStateTrie(id, triedb)
		if err != nil {
			log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
			return err
		}

		storageIt, err := storageTrie.NodeIterator(config.startKey)
		if err != nil {
			log.Error("Failed to open storage iterator", "root", acc.Root, "err", err)
			return err
		}

		storageIter := trie.NewIterator(storageIt)
		for storageIter.Next() {
			if config.limitKey != nil && bytes.Compare(storageIter.Key, config.limitKey) >= 0 {
				break
			}

			slots.Add(1)
			log.Debug("Storage slot", "key", common.Bytes2Hex(storageIter.Key), "value", common.Bytes2Hex(storageIter.Value))
		}
		if storageIter.Err != nil {
			log.Error("Failed to traverse storage trie", "root", acc.Root, "err", storageIter.Err)
			return storageIter.Err
		}

		log.Info("Storage traversal complete", "slots", slots.Load(), "elapsed", common.PrettyDuration(time.Since(start)))
		return nil
	} else {
		log.Info("Start traversing state trie", "root", config.root.Hex(), "startKey", common.Bytes2Hex(config.startKey), "limitKey", common.Bytes2Hex(config.limitKey))

		return traverseStateParallel(t, triedb, chaindb, config, &accounts, &slots, &codes, start)
	}
}

// traverseStateParallel parallelizes state traversal by dividing work across 16 trie branches
func traverseStateParallel(t *trie.StateTrie, triedb *triedb.Database, chaindb ethdb.Database, config *traverseConfig, accounts, slots, codes *atomic.Uint64, start time.Time) error {
	ctx := context.Background()
	g, ctx := errgroup.WithContext(ctx)

	for i := 0; i < 16; i++ {
		nibble := byte(i)
		g.Go(func() error {
			startKey := config.startKey
			limitKey := config.limitKey

			branchStartKey := make([]byte, len(startKey)+1)
			branchLimitKey := make([]byte, len(startKey)+1)

			if len(startKey) > 0 {
				copy(branchStartKey, startKey)
				copy(branchLimitKey, startKey)
			}

			branchStartKey[len(startKey)] = nibble << 4
			branchLimitKey[len(startKey)] = (nibble + 1) << 4

			if limitKey != nil && bytes.Compare(branchStartKey, limitKey) >= 0 {
				return nil
			}
			if limitKey != nil && bytes.Compare(branchLimitKey, limitKey) > 0 {
				branchLimitKey = make([]byte, len(limitKey))
				copy(branchLimitKey, limitKey)
			}

			return traverseBranch(ctx, t, triedb, chaindb, config.root, branchStartKey, branchLimitKey, accounts, slots, codes)
		})
	}

	if err := g.Wait(); err != nil {
		return err
	}

	log.Info("State traversal complete", "accounts", accounts.Load(), "slots", slots.Load(), "codes", codes.Load(), "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// traverseBranch traverses a specific branch of the state trie
func traverseBranch(ctx context.Context, t *trie.StateTrie, triedb *triedb.Database, chaindb ethdb.Database, root common.Hash, startKey, limitKey []byte, accounts, slots, codes *atomic.Uint64) error {
	acctIt, err := t.NodeIterator(startKey)
	if err != nil {
		return err
	}

	accIter := trie.NewIterator(acctIt)
	for accIter.Next() {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if limitKey != nil && bytes.Compare(accIter.Key, limitKey) >= 0 {
			break
		}

		accounts.Add(1)

		var acc types.StateAccount
		if err := rlp.DecodeBytes(accIter.Value, &acc); err != nil {
			log.Error("Invalid account encountered during traversal", "err", err)
			return err
		}

		if acc.Root != types.EmptyRootHash {
			id := trie.StorageTrieID(root, common.BytesToHash(accIter.Key), acc.Root)
			storageTrie, err := trie.NewStateTrie(id, triedb)
			if err != nil {
				log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
				return err
			}

			localSlots, err := traverseStorageParallel(ctx, storageTrie)
			if err != nil {
				log.Error("Failed to traverse storage trie", "root", acc.Root, "err", err)
				return err
			}
			slots.Add(localSlots)
		}

		if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash.Bytes()) {
			if !rawdb.HasCode(chaindb, common.BytesToHash(acc.CodeHash)) {
				log.Error("Code is missing", "hash", common.BytesToHash(acc.CodeHash))
				return errors.New("missing code")
			}
			codes.Add(1)
		}
	}

	if accIter.Err != nil {
		return accIter.Err
	}

	return nil
}

// traverseStorageParallel parallelizes storage trie traversal by dividing work across 16 trie branches
func traverseStorageParallel(ctx context.Context, storageTrie *trie.StateTrie) (uint64, error) {
	g, ctx := errgroup.WithContext(ctx)
	totalSlots := atomic.Uint64{}

	for i := 0; i < 16; i++ {
		nibble := byte(i)
		g.Go(func() error {
			branchStartKey := []byte{nibble << 4}
			branchLimitKey := []byte{(nibble + 1) << 4}

			localSlots, err := traverseStorageBranch(ctx, storageTrie, branchStartKey, branchLimitKey)
			if err != nil {
				return err
			}
			totalSlots.Add(localSlots)
			return nil
		})
	}

	if err := g.Wait(); err != nil {
		return 0, err
	}

	return totalSlots.Load(), nil
}

// traverseStorageBranch traverses a specific branch of the storage trie
func traverseStorageBranch(ctx context.Context, storageTrie *trie.StateTrie, startKey, limitKey []byte) (uint64, error) {
	storageIt, err := storageTrie.NodeIterator(startKey)
	if err != nil {
		return 0, err
	}

	storageIter := trie.NewIterator(storageIt)
	slots := uint64(0)

	for storageIter.Next() {
		// Check if context was cancelled
		select {
		case <-ctx.Done():
			return 0, ctx.Err()
		default:
		}

		if bytes.Compare(storageIter.Key, limitKey) >= 0 {
			break
		}
		slots++
	}

	if storageIter.Err != nil {
		return 0, storageIter.Err
	}

	return slots, nil
}

type traverseConfig struct {
	root      common.Hash
	startKey  []byte
	limitKey  []byte
	account   common.Hash
	isAccount bool
}

func parseTraverseArgs(ctx *cli.Context) (*traverseConfig, error) {
	if ctx.NArg() > 1 {
		return nil, errors.New("too many arguments, only <root> is required")
	}

	config := &traverseConfig{}
	var err error

	if ctx.NArg() == 1 {
		config.root, err = parseRoot(ctx.Args().First())
		if err != nil {
			return nil, err
		}
	}

	if accountFlag := ctx.String("account"); accountFlag != "" {
		config.isAccount = true
		switch len(accountFlag) {
		case 40, 42:
			config.account = crypto.Keccak256Hash(common.HexToAddress(accountFlag).Bytes())
		case 64, 66:
			config.account = common.HexToHash(accountFlag)
		default:
			return nil, errors.New("account must be 40/42 chars for address or 64/66 chars for hash")
		}
	}

	if startFlag := ctx.String("start"); startFlag != "" {
		if len(startFlag) == 64 || len(startFlag) == 66 {
			config.startKey = common.HexToHash(startFlag).Bytes()
		} else {
			return nil, errors.New("start key must be 64/66 chars hex")
		}
	}

	if limitFlag := ctx.String("limit"); limitFlag != "" {
		if len(limitFlag) == 64 || len(limitFlag) == 66 {
			config.limitKey = common.HexToHash(limitFlag).Bytes()
		} else {
			return nil, errors.New("limit key must be 64/66 chars hex")
		}
	}

	return config, nil
}

// traverseRawState is a helper function used for pruning verification.
// Basically it just iterates the trie, ensure all nodes and associated
// contract codes are present. It's basically identical to traverseState
// but it will check each trie node.
func traverseRawState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()

	triedb := utils.MakeTrieDatabase(ctx, stack, chaindb, false, true, false)
	defer triedb.Close()

	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return errors.New("no head block")
	}

	config, err := parseTraverseArgs(ctx)
	if err != nil {
		log.Error("Failed to parse arguments", "err", err)
		return err
	}
	if config.root == (common.Hash{}) {
		config.root = headBlock.Root()
	}
	t, err := trie.NewStateTrie(trie.StateTrieID(config.root), triedb)
	if err != nil {
		log.Error("Failed to open trie", "root", config.root, "err", err)
		return err
	}

	var (
		accounts int
		nodes    int
		slots    int
		codes    int
		start    = time.Now()
		hasher   = crypto.NewKeccakState()
		got      = make([]byte, 32)
	)

	go func() {
		timer := time.NewTicker(time.Second * 8)
		defer timer.Stop()
		for range timer.C {
			log.Info("Traversing rawstate", "nodes", nodes, "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
		}
	}()

	if config.isAccount {
		log.Info("Start traversing storage trie (raw)", "root", config.root.Hex(), "account", config.account.Hex(), "startKey", common.Bytes2Hex(config.startKey), "limitKey", common.Bytes2Hex(config.limitKey))

		acc, err := t.GetAccountByHash(config.account)
		if err != nil {
			log.Error("Get account failed", "account", config.account.Hex(), "err", err)
			return err
		}

		if acc.Root == types.EmptyRootHash {
			log.Info("Account has no storage")
			return nil
		}

		// Traverse the storage trie with detailed verification
		id := trie.StorageTrieID(config.root, config.account, acc.Root)
		storageTrie, err := trie.NewStateTrie(id, triedb)
		if err != nil {
			log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
			return err
		}

		storageIter, err := storageTrie.NodeIterator(config.startKey)
		if err != nil {
			log.Error("Failed to open storage iterator", "root", acc.Root, "err", err)
			return err
		}

		reader, err := triedb.NodeReader(config.root)
		if err != nil {
			log.Error("State is non-existent", "root", config.root)
			return nil
		}

		for storageIter.Next(true) {
			nodes += 1
			node := storageIter.Hash()

			// Check the presence for non-empty hash node(embedded node doesn't
			// have their own hash).
			if node != (common.Hash{}) {
				blob, _ := reader.Node(config.account, storageIter.Path(), node)
				if len(blob) == 0 {
					log.Error("Missing trie node(storage)", "hash", node)
					return errors.New("missing storage")
				}
				hasher.Reset()
				hasher.Write(blob)
				hasher.Read(got)
				if !bytes.Equal(got, node.Bytes()) {
					log.Error("Invalid trie node(storage)", "hash", node.Hex(), "value", blob)
					return errors.New("invalid storage node")
				}
			}

			// Bump the counter if it's leaf node.
			if storageIter.Leaf() {
				// Check if we've exceeded the limit key for storage
				if config.limitKey != nil && bytes.Compare(storageIter.LeafKey(), config.limitKey) >= 0 {
					break
				}

				slots += 1
				log.Debug("Storage slot", "key", common.Bytes2Hex(storageIter.LeafKey()), "value", common.Bytes2Hex(storageIter.LeafBlob()))
			}
		}
		if storageIter.Error() != nil {
			log.Error("Failed to traverse storage trie", "root", acc.Root, "err", storageIter.Error())
			return storageIter.Error()
		}

		log.Info("Storage traversal complete (raw)", "nodes", nodes, "slots", slots, "elapsed", common.PrettyDuration(time.Since(start)))
		return nil
	} else {
		log.Info("Start traversing the state trie (raw)", "root", config.root.Hex(), "startKey", common.Bytes2Hex(config.startKey), "limitKey", common.Bytes2Hex(config.limitKey))

		accIter, err := t.NodeIterator(config.startKey)
		if err != nil {
			log.Error("Failed to open iterator", "root", config.root, "err", err)
			return err
		}
		reader, err := triedb.NodeReader(config.root)
		if err != nil {
			log.Error("State is non-existent", "root", config.root)
			return nil
		}
		for accIter.Next(true) {
			nodes += 1
			node := accIter.Hash()

			// Check the present for non-empty hash node(embedded node doesn't
			// have their own hash).
			if node != (common.Hash{}) {
				blob, _ := reader.Node(common.Hash{}, accIter.Path(), node)
				if len(blob) == 0 {
					log.Error("Missing trie node(account)", "hash", node)
					return errors.New("missing account")
				}
				hasher.Reset()
				hasher.Write(blob)
				hasher.Read(got)
				if !bytes.Equal(got, node.Bytes()) {
					log.Error("Invalid trie node(account)", "hash", node.Hex(), "value", blob)
					return errors.New("invalid account node")
				}
			}
			// If it's a leaf node, yes we are touching an account,
			// dig into the storage trie further.
			if accIter.Leaf() {
				// Check if we've exceeded the limit key for accounts
				if config.limitKey != nil && bytes.Compare(accIter.LeafKey(), config.limitKey) >= 0 {
					break
				}

				accounts += 1
				var acc types.StateAccount
				if err := rlp.DecodeBytes(accIter.LeafBlob(), &acc); err != nil {
					log.Error("Invalid account encountered during traversal", "err", err)
					return errors.New("invalid account")
				}
				if acc.Root != types.EmptyRootHash {
					id := trie.StorageTrieID(config.root, common.BytesToHash(accIter.LeafKey()), acc.Root)
					storageTrie, err := trie.NewStateTrie(id, triedb)
					if err != nil {
						log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
						return errors.New("missing storage trie")
					}
					storageIter, err := storageTrie.NodeIterator(nil)
					if err != nil {
						log.Error("Failed to open storage iterator", "root", acc.Root, "err", err)
						return err
					}
					for storageIter.Next(true) {
						nodes += 1
						node := storageIter.Hash()

						// Check the presence for non-empty hash node(embedded node doesn't
						// have their own hash).
						if node != (common.Hash{}) {
							blob, _ := reader.Node(common.BytesToHash(accIter.LeafKey()), storageIter.Path(), node)
							if len(blob) == 0 {
								log.Error("Missing trie node(storage)", "hash", node)
								return errors.New("missing storage")
							}
							hasher.Reset()
							hasher.Write(blob)
							hasher.Read(got)
							if !bytes.Equal(got, node.Bytes()) {
								log.Error("Invalid trie node(storage)", "hash", node.Hex(), "value", blob)
								return errors.New("invalid storage node")
							}
						}
						// Bump the counter if it's leaf node.
						if storageIter.Leaf() {
							slots += 1
						}
					}
					if storageIter.Error() != nil {
						log.Error("Failed to traverse storage trie", "root", acc.Root, "err", storageIter.Error())
						return storageIter.Error()
					}
				}
				if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash.Bytes()) {
					if !rawdb.HasCode(chaindb, common.BytesToHash(acc.CodeHash)) {
						log.Error("Code is missing", "account", common.BytesToHash(accIter.LeafKey()))
						return errors.New("missing code")
					}
					codes += 1
				}
			}
		}
		if accIter.Error() != nil {
			log.Error("Failed to traverse state trie", "root", config.root, "err", accIter.Error())
			return accIter.Error()
		}
		log.Info("State traversal complete (raw)", "nodes", nodes, "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
		return nil
	}
}

func parseRoot(input string) (common.Hash, error) {
	var h common.Hash
	if err := h.UnmarshalText([]byte(input)); err != nil {
		return h, err
	}
	return h, nil
}

func dumpState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	db := utils.MakeChainDatabase(ctx, stack, true)
	defer db.Close()

	conf, root, err := parseDumpConfig(ctx, db)
	if err != nil {
		return err
	}
	triedb := utils.MakeTrieDatabase(ctx, stack, db, false, true, false)
	defer triedb.Close()

	snapConfig := snapshot.Config{
		CacheSize:  256,
		Recovery:   false,
		NoBuild:    true,
		AsyncBuild: false,
	}
	snaptree, err := snapshot.New(snapConfig, db, triedb, root)
	if err != nil {
		return err
	}
	accIt, err := snaptree.AccountIterator(root, common.BytesToHash(conf.Start))
	if err != nil {
		return err
	}
	defer accIt.Release()

	log.Info("Snapshot dumping started", "root", root)
	var (
		start    = time.Now()
		logged   = time.Now()
		accounts uint64
	)
	enc := json.NewEncoder(os.Stdout)
	enc.Encode(struct {
		Root common.Hash `json:"root"`
	}{root})
	for accIt.Next() {
		account, err := types.FullAccount(accIt.Account())
		if err != nil {
			return err
		}
		da := &state.DumpAccount{
			Balance:     account.Balance.String(),
			Nonce:       account.Nonce,
			Root:        account.Root.Bytes(),
			CodeHash:    account.CodeHash,
			AddressHash: accIt.Hash().Bytes(),
		}
		if !conf.SkipCode && !bytes.Equal(account.CodeHash, types.EmptyCodeHash.Bytes()) {
			da.Code = rawdb.ReadCode(db, common.BytesToHash(account.CodeHash))
		}
		if !conf.SkipStorage {
			da.Storage = make(map[common.Hash]string)

			stIt, err := snaptree.StorageIterator(root, accIt.Hash(), common.Hash{})
			if err != nil {
				return err
			}
			for stIt.Next() {
				da.Storage[stIt.Hash()] = common.Bytes2Hex(stIt.Slot())
			}
		}
		enc.Encode(da)
		accounts++
		if time.Since(logged) > 8*time.Second {
			log.Info("Snapshot dumping in progress", "at", accIt.Hash(), "accounts", accounts,
				"elapsed", common.PrettyDuration(time.Since(start)))
			logged = time.Now()
		}
		if conf.Max > 0 && accounts >= conf.Max {
			break
		}
	}
	log.Info("Snapshot dumping complete", "accounts", accounts,
		"elapsed", common.PrettyDuration(time.Since(start)))
	return nil
}

// snapshotExportPreimages dumps the preimage data to a flat file.
func snapshotExportPreimages(ctx *cli.Context) error {
	if ctx.NArg() < 1 {
		utils.Fatalf("This command requires an argument.")
	}
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()

	triedb := utils.MakeTrieDatabase(ctx, stack, chaindb, false, true, false)
	defer triedb.Close()

	var root common.Hash
	if ctx.NArg() > 1 {
		rootBytes := common.FromHex(ctx.Args().Get(1))
		if len(rootBytes) != common.HashLength {
			return fmt.Errorf("invalid hash: %s", ctx.Args().Get(1))
		}
		root = common.BytesToHash(rootBytes)
	} else {
		headBlock := rawdb.ReadHeadBlock(chaindb)
		if headBlock == nil {
			log.Error("Failed to load head block")
			return errors.New("no head block")
		}
		root = headBlock.Root()
	}
	snapConfig := snapshot.Config{
		CacheSize:  256,
		Recovery:   false,
		NoBuild:    true,
		AsyncBuild: false,
	}
	snaptree, err := snapshot.New(snapConfig, chaindb, triedb, root)
	if err != nil {
		return err
	}
	return utils.ExportSnapshotPreimages(chaindb, snaptree, ctx.Args().First(), root)
}

// checkAccount iterates the snap data layers, and looks up the given account
// across all layers.
func checkAccount(ctx *cli.Context) error {
	if ctx.NArg() != 1 {
		return errors.New("need <address|hash> arg")
	}
	var (
		hash common.Hash
		addr common.Address
	)
	switch arg := ctx.Args().First(); len(arg) {
	case 40, 42:
		addr = common.HexToAddress(arg)
		hash = crypto.Keccak256Hash(addr.Bytes())
	case 64, 66:
		hash = common.HexToHash(arg)
	default:
		return errors.New("malformed address or hash")
	}
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()
	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()
	start := time.Now()
	log.Info("Checking difflayer journal", "address", addr, "hash", hash)
	if err := snapshot.CheckJournalAccount(chaindb, hash); err != nil {
		return err
	}
	log.Info("Checked the snapshot journalled storage", "time", common.PrettyDuration(time.Since(start)))
	return nil
}
