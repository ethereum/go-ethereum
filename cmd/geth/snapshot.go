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
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/database"
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

func traverseState(ctx *cli.Context) error {
	ts, err := setupTraversal(ctx)
	if err != nil {
		return err
	}
	defer ts.Close()

	var (
		counters     = &traverseCounters{start: time.Now()}
		cctx, cancel = context.WithCancel(context.Background())
	)
	defer cancel()

	go func() {
		timer := time.NewTicker(time.Second * 8)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				log.Info("Traversing state", "accounts", counters.accounts.Load(), "slots", counters.slots.Load(), "codes", counters.codes.Load(), "elapsed", common.PrettyDuration(time.Since(counters.start)))
			case <-ctx.Done():
				return
			}
		}
	}()

	if ts.config.isAccount {
		return ts.traverseAccount(cctx, counters, false)
	} else {
		return ts.traverseState(cctx, counters, false)
	}
}

type OnStorageNodeHook func(node common.Hash, path []byte) error

// createRawStorageHook creates hooks for raw storage traversal with verification
func createRawStorageHook(reader database.NodeReader, accountHash common.Hash) OnStorageNodeHook {
	return func(node common.Hash, path []byte) error {
		if node != (common.Hash{}) {
			blob, _ := reader.Node(accountHash, path, node)
			if len(blob) == 0 {
				log.Error("Missing trie node(storage)", "hash", node)
				return errors.New("missing storage")
			}
			if !bytes.Equal(crypto.Keccak256(blob), node.Bytes()) {
				log.Error("Invalid trie node(storage)", "hash", node.Hex(), "value", blob)
				return errors.New("invalid storage node")
			}
		}
		return nil
	}
}

// traverseStorage parallelizes storage trie traversal
func traverseStorage(ctx context.Context, storageTrie *trie.StateTrie, startKey, limitKey []byte, raw bool, hook OnStorageNodeHook) (uint64, uint64, error) {
	var (
		eg, cctx = errgroup.WithContext(ctx)
		slots    atomic.Uint64
		nodes    atomic.Uint64
	)

	for i := 0; i < 16; i++ {
		nibble := byte(i)
		eg.Go(func() error {
			// Calculate this branch's natural boundaries
			var (
				branchStart = []byte{nibble << 4}
				branchLimit []byte
			)
			if nibble < 15 {
				branchLimit = []byte{(nibble + 1) << 4}
			}

			// Skip branches that are entirely before startKey
			if startKey != nil && branchLimit != nil && bytes.Compare(branchLimit, startKey) <= 0 {
				return nil
			}

			// Skip branches that are entirely after limitKey
			if limitKey != nil && bytes.Compare(branchStart, limitKey) >= 0 {
				return nil
			}

			// Use the more restrictive start boundary
			if startKey != nil && bytes.Compare(branchStart, startKey) < 0 {
				branchStart = startKey
			}
			if limitKey != nil && (branchLimit == nil || bytes.Compare(branchLimit, limitKey) > 0) {
				branchLimit = limitKey
			}

			// Skip if branch range is empty
			if branchLimit != nil && bytes.Compare(branchStart, branchLimit) >= 0 {
				return nil
			}

			localSlots, localNodes, err := traverseStorageBranchWithHooks(cctx, storageTrie, branchStart, branchLimit, raw, hook)
			if err != nil {
				return err
			}
			slots.Add(localSlots)
			nodes.Add(localNodes)
			return nil
		})
	}

	if err := eg.Wait(); err != nil {
		return 0, 0, err
	}

	return slots.Load(), nodes.Load(), nil
}

// traverseStorageBranchWithHooks traverses a specific range of the storage trie using hooks
func traverseStorageBranchWithHooks(ctx context.Context, storageTrie *trie.StateTrie, startKey, limitKey []byte, raw bool, hook OnStorageNodeHook) (slots, nodes uint64, err error) {
	nodeIter, err := storageTrie.NodeIterator(startKey)
	if err != nil {
		return 0, 0, err
	}

	if raw {
		// Raw traversal with detailed node checking
		for nodeIter.Next(true) {
			select {
			case <-ctx.Done():
				return 0, 0, ctx.Err()
			default:
			}

			nodes++
			if hook != nil {
				if err := hook(nodeIter.Hash(), nodeIter.Path()); err != nil {
					return 0, 0, err
				}
			}

			if nodeIter.Leaf() {
				if limitKey != nil && bytes.Compare(nodeIter.LeafKey(), limitKey) >= 0 {
					break
				}
				slots++
			}
		}
	} else {
		// Simple traversal - just iterate through leaf nodes
		storageIter := trie.NewIterator(nodeIter)
		for storageIter.Next() {
			select {
			case <-ctx.Done():
				return 0, 0, ctx.Err()
			default:
			}

			if limitKey != nil && bytes.Compare(storageIter.Key, limitKey) >= 0 {
				break
			}
			slots++
		}

		if storageIter.Err != nil {
			return 0, 0, storageIter.Err
		}
	}

	if err := nodeIter.Error(); err != nil {
		return 0, 0, err
	}

	return slots, nodes, nil
}

type traverseConfig struct {
	root      common.Hash
	startKey  []byte
	limitKey  []byte
	account   common.Hash
	isAccount bool
}

type traverseSetup struct {
	stack   *node.Node
	chaindb ethdb.Database
	triedb  *triedb.Database
	trie    *trie.StateTrie
	config  *traverseConfig
}

type traverseCounters struct {
	accounts atomic.Uint64
	slots    atomic.Uint64
	codes    atomic.Uint64
	nodes    atomic.Uint64
	start    time.Time
}

func setupTraversal(ctx *cli.Context) (*traverseSetup, error) {
	stack, _ := makeConfigNode(ctx)

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	triedb := utils.MakeTrieDatabase(ctx, stack, chaindb, false, true, false)

	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return nil, errors.New("no head block")
	}

	config, err := parseTraverseArgs(ctx)
	if err != nil {
		return nil, err
	}
	if config.root == (common.Hash{}) {
		config.root = headBlock.Root()
	}

	t, err := trie.NewStateTrie(trie.StateTrieID(config.root), triedb)
	if err != nil {
		log.Error("Failed to open trie", "root", config.root, "err", err)
		return nil, err
	}

	return &traverseSetup{
		stack:   stack,
		chaindb: chaindb,
		triedb:  triedb,
		trie:    t,
		config:  config,
	}, nil
}

func (ts *traverseSetup) Close() {
	ts.triedb.Close()
	ts.chaindb.Close()
	ts.stack.Close()
}

func (ts *traverseSetup) traverseAccount(ctx context.Context, counters *traverseCounters, raw bool) error {
	log.Info("Start traversing storage trie", "root", ts.config.root.Hex(), "account", ts.config.account.Hex(), "startKey", common.Bytes2Hex(ts.config.startKey), "limitKey", common.Bytes2Hex(ts.config.limitKey))

	acc, err := ts.trie.GetAccountByHash(ts.config.account)
	if err != nil {
		log.Error("Get account failed", "account", ts.config.account.Hex(), "err", err)
		return err
	}

	if acc.Root == types.EmptyRootHash {
		log.Info("Account has no storage")
		return nil
	}

	id := trie.StorageTrieID(ts.config.root, ts.config.account, acc.Root)
	storageTrie, err := trie.NewStateTrie(id, ts.triedb)
	if err != nil {
		log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
		return err
	}

	var hook OnStorageNodeHook
	if raw {
		reader, err := ts.triedb.NodeReader(ts.config.root)
		if err != nil {
			log.Error("State is non-existent", "root", ts.config.root)
			return nil
		}
		hook = createRawStorageHook(reader, ts.config.account)
	}

	slots, nodes, err := traverseStorage(ctx, storageTrie, ts.config.startKey, ts.config.limitKey, raw, hook)
	if err != nil {
		log.Error("Failed to traverse storage trie", "root", acc.Root, "err", err)
		return err
	}

	counters.slots.Add(slots)
	counters.nodes.Add(nodes)

	if raw {
		log.Info("Storage traversal complete (raw)", "nodes", counters.nodes.Load(), "slots", counters.slots.Load(), "elapsed", common.PrettyDuration(time.Since(counters.start)))
	} else {
		log.Info("Storage traversal complete", "slots", counters.slots.Load(), "elapsed", common.PrettyDuration(time.Since(counters.start)))
	}
	return nil
}

func (ts *traverseSetup) traverseState(ctx context.Context, counters *traverseCounters, raw bool) error {
	log.Info("Start traversing state trie", "root", ts.config.root.Hex(), "startKey", common.Bytes2Hex(ts.config.startKey), "limitKey", common.Bytes2Hex(ts.config.limitKey))

	eg, ctx := errgroup.WithContext(ctx)
	var reader database.NodeReader
	if raw {
		var err error
		reader, err = ts.triedb.NodeReader(ts.config.root)
		if err != nil {
			log.Error("State is non-existent", "root", ts.config.root)
			return nil
		}
	}

	for i := 0; i < 16; i++ {
		nibble := byte(i)
		eg.Go(func() error {
			var (
				startKey = []byte{nibble << 4}
				limitKey []byte
			)
			if nibble < 15 {
				limitKey = []byte{(nibble + 1) << 4}
			}

			if ts.config != nil {
				// Skip branches that are entirely before startKey
				if ts.config.startKey != nil && limitKey != nil && bytes.Compare(limitKey, ts.config.startKey) <= 0 {
					return nil
				}

				// Skip branches that are entirely after limitKey
				if ts.config.limitKey != nil && bytes.Compare(startKey, ts.config.limitKey) >= 0 {
					return nil
				}

				if ts.config.startKey != nil && bytes.Compare(startKey, ts.config.startKey) < 0 {
					startKey = ts.config.startKey
				}
				if ts.config.limitKey != nil && (limitKey == nil || bytes.Compare(limitKey, ts.config.limitKey) > 0) {
					limitKey = ts.config.limitKey
				}
			}

			if limitKey != nil && bytes.Compare(startKey, limitKey) >= 0 {
				return nil
			}

			var hooks *TraverseHooks
			if raw {
				hooks = ts.createRawHooks(reader)
			} else {
				hooks = ts.createSimpleHooks()
			}

			return ts.traverseStateBranchWithHooks(ctx, startKey, limitKey, raw, counters, hooks)
		})
	}

	if err := eg.Wait(); err != nil {
		return err
	}

	log.Info("State traversal complete", "accounts", counters.accounts.Load(), "slots", counters.slots.Load(), "codes", counters.codes.Load(), "elapsed", common.PrettyDuration(time.Since(counters.start)))
	return nil
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

// TraverseHooks defines the hooks for different traversal modes
type TraverseHooks struct {
	// Called for each trie account node (only for raw mode)
	OnAccountNode func(node common.Hash, path []byte) error
	// Called for each storage trie
	OnStorageTrie func(ctx context.Context, storageTrie *trie.StateTrie, accountHash common.Hash) (slots, nodes uint64, err error)
	// Called for each code
	OnCode func(codeHash []byte, accountHash common.Hash) error
}

// createSimpleHooks creates hooks for simple traversal mode
func (ts *traverseSetup) createSimpleHooks() *TraverseHooks {
	return &TraverseHooks{
		OnStorageTrie: func(ctx context.Context, storageTrie *trie.StateTrie, accountHash common.Hash) (slots, nodes uint64, err error) {
			return traverseStorage(ctx, storageTrie, nil, nil, false, nil)
		},
		OnCode: func(codeHash []byte, accountHash common.Hash) error {
			if !rawdb.HasCode(ts.chaindb, common.BytesToHash(codeHash)) {
				log.Error("Code is missing", "hash", common.BytesToHash(codeHash))
				return errors.New("missing code")
			}
			return nil
		},
	}
}

// createRawHooks creates hooks for raw traversal mode with detailed verification
func (ts *traverseSetup) createRawHooks(reader database.NodeReader) *TraverseHooks {
	return &TraverseHooks{
		OnAccountNode: func(node common.Hash, path []byte) error {
			if node != (common.Hash{}) {
				blob, _ := reader.Node(common.Hash{}, path, node)
				if len(blob) == 0 {
					log.Error("Missing trie node(account)", "hash", node)
					return errors.New("missing account")
				}
				if !bytes.Equal(crypto.Keccak256(blob), node.Bytes()) {
					log.Error("Invalid trie node(account)", "hash", node.Hex(), "value", blob)
					return errors.New("invalid account node")
				}
			}
			return nil
		},
		OnStorageTrie: func(ctx context.Context, storageTrie *trie.StateTrie, accountHash common.Hash) (slots, nodes uint64, err error) {
			hook := createRawStorageHook(reader, accountHash)
			return traverseStorage(ctx, storageTrie, nil, nil, true, hook)
		},
		OnCode: func(codeHash []byte, accountHash common.Hash) error {
			if !rawdb.HasCode(ts.chaindb, common.BytesToHash(codeHash)) {
				log.Error("Code is missing", "account", accountHash)
				return errors.New("missing code")
			}
			return nil
		},
	}
}

// traverseStateBranchWithHooks provides common branch traversal logic using hooks
func (ts *traverseSetup) traverseStateBranchWithHooks(ctx context.Context, startKey, limitKey []byte, raw bool, counters *traverseCounters, hooks *TraverseHooks) error {
	accIter, err := ts.trie.NodeIterator(startKey)
	if err != nil {
		return err
	}

	if raw {
		// Raw traversal with detailed node checking
		for accIter.Next(true) {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			counters.nodes.Add(1)
			if hooks != nil && hooks.OnAccountNode != nil {
				if err := hooks.OnAccountNode(accIter.Hash(), accIter.Path()); err != nil {
					return err
				}
			}

			// If it's a leaf node, process the account
			if accIter.Leaf() {
				if limitKey != nil && bytes.Compare(accIter.LeafKey(), limitKey) >= 0 {
					break
				}
				counters.accounts.Add(1)

				var acc types.StateAccount
				if err := rlp.DecodeBytes(accIter.LeafBlob(), &acc); err != nil {
					log.Error("Invalid account encountered during traversal", "err", err)
					return errors.New("invalid account")
				}

				accountHash := common.BytesToHash(accIter.LeafKey())

				if acc.Root != types.EmptyRootHash {
					id := trie.StorageTrieID(ts.config.root, accountHash, acc.Root)
					storageTrie, err := trie.NewStateTrie(id, ts.triedb)
					if err != nil {
						log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
						return err
					}

					if hooks != nil && hooks.OnStorageTrie != nil {
						slots, nodes, err := hooks.OnStorageTrie(ctx, storageTrie, accountHash)
						if err != nil {
							return err
						}
						counters.slots.Add(slots)
						counters.nodes.Add(nodes)
					}
				}

				if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash.Bytes()) {
					if err := hooks.OnCode(acc.CodeHash, accountHash); err != nil {
						return err
					}
					counters.codes.Add(1)
				}
			}
		}
	} else {
		// Simple traversal - just iterate through leaf nodes
		acctIt, err := ts.trie.NodeIterator(startKey)
		if err != nil {
			return err
		}

		accIter := trie.NewIterator(acctIt)
		for accIter.Next() {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			if limitKey != nil && bytes.Compare(accIter.Key, limitKey) >= 0 {
				break
			}

			counters.accounts.Add(1)

			var acc types.StateAccount
			if err := rlp.DecodeBytes(accIter.Value, &acc); err != nil {
				log.Error("Invalid account encountered during traversal", "err", err)
				return err
			}

			accountHash := common.BytesToHash(accIter.Key)

			// Process storage if present
			if acc.Root != types.EmptyRootHash {
				id := trie.StorageTrieID(ts.config.root, accountHash, acc.Root)
				storageTrie, err := trie.NewStateTrie(id, ts.triedb)
				if err != nil {
					log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
					return err
				}

				slots, nodes, err := hooks.OnStorageTrie(ctx, storageTrie, accountHash)
				if err != nil {
					return err
				}
				counters.slots.Add(slots)
				counters.nodes.Add(nodes)
			}

			// Process code if present
			if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash.Bytes()) {
				if err := hooks.OnCode(acc.CodeHash, accountHash); err != nil {
					return err
				}
				counters.codes.Add(1)
			}
		}

		if accIter.Err != nil {
			return accIter.Err
		}
	}

	if accIter.Error() != nil {
		return accIter.Error()
	}

	return nil
}

// traverseRawState is a helper function used for pruning verification.
// Basically it just iterates the trie, ensure all nodes and associated
// contract codes are present. It's basically identical to traverseState
// but it will check each trie node.
func traverseRawState(ctx *cli.Context) error {
	ts, err := setupTraversal(ctx)
	if err != nil {
		return err
	}
	defer ts.Close()

	var (
		counters     = &traverseCounters{start: time.Now()}
		cctx, cancel = context.WithCancel(context.Background())
	)
	defer cancel()

	go func() {
		timer := time.NewTicker(time.Second * 8)
		defer timer.Stop()
		for {
			select {
			case <-timer.C:
				log.Info("Traversing rawstate", "nodes", counters.nodes.Load(), "accounts", counters.accounts.Load(), "slots", counters.slots.Load(), "codes", counters.codes.Load(), "elapsed", common.PrettyDuration(time.Since(counters.start)))
			case <-cctx.Done():
				return
			}
		}
	}()

	if ts.config.isAccount {
		return ts.traverseAccount(cctx, counters, true)
	} else {
		return ts.traverseState(cctx, counters, true)
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
