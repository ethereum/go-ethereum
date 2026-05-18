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
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"slices"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/pruner"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
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
				Flags: slices.Concat([]cli.Flag{
					utils.AccountFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot traverse-state <state-root>
will traverse the whole state from the given state root and will abort if any
referenced trie node or contract code is missing. This command can be used for
state integrity verification. The default checking target is the HEAD state.

It's also usable without snapshot enabled.

If --account is specified, only the storage trie of that account is traversed.
`,
			},
			{
				Name:      "traverse-rawstate",
				Usage:     "Traverse the state with given root hash and perform detailed verification",
				ArgsUsage: "<root>",
				Action:    traverseRawState,
				Flags: slices.Concat([]cli.Flag{
					utils.AccountFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot traverse-rawstate <state-root>
will traverse the whole state from the given root and will abort if any referenced
trie node or contract code is missing. This command can be used for state integrity
verification. The default checking target is the HEAD state. It's basically identical
to traverse-state, but the check granularity is smaller.

It's also usable without snapshot enabled.

If --account is specified, only the storage trie of that account is traversed.
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
			{
				Name:    "list-eip-7610-accounts",
				Aliases: []string{"eip7610"},
				Usage:   "list EIP7610 eligible accounts",
				Action:  listEIP7610EligibleAccounts,
				Flags:   slices.Concat(utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth snapshot list-eip-7610-accounts
traverses the post–EIP-161 state and returns all accounts that are eligible
under EIP-7610: accounts with zero nonce, empty runtime code, and non-empty
storage. The traversal will be aborted immediately if the state is prior to
EIP-161.

The exported accounts are identified by their address.
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

// parseAccount parses the account flag value as either an address (20 bytes)
// or an account hash (32 bytes) and returns the hashed account key.
func parseAccount(input string) (common.Hash, error) {
	switch len(input) {
	case 40, 42: // address
		return crypto.Keccak256Hash(common.HexToAddress(input).Bytes()), nil
	case 64, 66: // hash
		return common.HexToHash(input), nil
	default:
		return common.Hash{}, errors.New("malformed account address or hash")
	}
}

// lookupAccount resolves the account from the state trie using the given
// account hash.
func lookupAccount(accountHash common.Hash, tr *trie.Trie) (*types.StateAccount, error) {
	accData, err := tr.Get(accountHash.Bytes())
	if err != nil {
		return nil, fmt.Errorf("failed to get account %s: %w", accountHash, err)
	}
	if accData == nil {
		return nil, fmt.Errorf("account not found: %s", accountHash)
	}
	var acc types.StateAccount
	if err := rlp.DecodeBytes(accData, &acc); err != nil {
		return nil, fmt.Errorf("invalid account data %s: %w", accountHash, err)
	}
	return &acc, nil
}

func traverseStorage(id *trie.ID, db *triedb.Database, report bool, detail bool) error {
	tr, err := trie.NewStateTrie(id, db)
	if err != nil {
		log.Error("Failed to open storage trie", "account", id.Owner, "root", id.Root, "err", err)
		return err
	}
	var (
		slots      int
		nodes      int
		lastReport time.Time
		start      = time.Now()
	)
	it, err := tr.NodeIterator(nil)
	if err != nil {
		log.Error("Failed to open storage iterator", "account", id.Owner, "root", id.Root, "err", err)
		return err
	}
	logger := log.Debug
	if report {
		logger = log.Info
	}
	logger("Start traversing storage trie", "account", id.Owner, "storageRoot", id.Root)

	if !detail {
		iter := trie.NewIterator(it)
		for iter.Next() {
			slots += 1
			if time.Since(lastReport) > time.Second*8 {
				logger("Traversing storage", "account", id.Owner, "slots", slots, "elapsed", common.PrettyDuration(time.Since(start)))
				lastReport = time.Now()
			}
		}
		if iter.Err != nil {
			log.Error("Failed to traverse storage trie", "root", id.Root, "err", iter.Err)
			return iter.Err
		}
		logger("Storage is complete", "account", id.Owner, "slots", slots, "elapsed", common.PrettyDuration(time.Since(start)))
	} else {
		reader, err := db.NodeReader(id.StateRoot)
		if err != nil {
			log.Error("Failed to open state reader", "err", err)
			return err
		}
		var (
			buffer = make([]byte, 32)
			hasher = crypto.NewKeccakState()
		)
		for it.Next(true) {
			nodes += 1
			node := it.Hash()

			// Check the presence for non-empty hash node(embedded node doesn't
			// have their own hash).
			if node != (common.Hash{}) {
				blob, _ := reader.Node(id.Owner, it.Path(), node)
				if len(blob) == 0 {
					log.Error("Missing trie node(storage)", "hash", node)
					return errors.New("missing storage")
				}
				hasher.Reset()
				hasher.Write(blob)
				hasher.Read(buffer)
				if !bytes.Equal(buffer, node.Bytes()) {
					log.Error("Invalid trie node(storage)", "hash", node.Hex(), "value", blob)
					return errors.New("invalid storage node")
				}
			}
			if it.Leaf() {
				slots += 1
			}
			if time.Since(lastReport) > time.Second*8 {
				logger("Traversing storage", "account", id.Owner, "nodes", nodes, "slots", slots, "elapsed", common.PrettyDuration(time.Since(start)))
				lastReport = time.Now()
			}
		}
		if err := it.Error(); err != nil {
			log.Error("Failed to traverse storage trie", "root", id.Root, "err", err)
			return err
		}
		logger("Storage is complete", "account", id.Owner, "nodes", nodes, "slots", slots, "elapsed", common.PrettyDuration(time.Since(start)))
	}
	return nil
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
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var (
		root common.Hash
		err  error
	)
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "err", err)
			return err
		}
		log.Info("Start traversing the state", "root", root)
	} else {
		root = headBlock.Root()
		log.Info("Start traversing the state", "root", root, "number", headBlock.NumberU64())
	}
	// If --account is specified, only traverse the storage trie of that account.
	if accountStr := ctx.String(utils.AccountFlag.Name); accountStr != "" {
		accountHash, err := parseAccount(accountStr)
		if err != nil {
			log.Error("Failed to parse account", "err", err)
			return err
		}
		// Use raw trie since the account key is already hashed.
		t, err := trie.New(trie.StateTrieID(root), triedb)
		if err != nil {
			log.Error("Failed to open state trie", "root", root, "err", err)
			return err
		}
		acc, err := lookupAccount(accountHash, t)
		if err != nil {
			log.Error("Failed to look up account", "hash", accountHash, "err", err)
			return err
		}
		if acc.Root == types.EmptyRootHash {
			log.Info("Account has no storage", "hash", accountHash)
			return nil
		}
		return traverseStorage(trie.StorageTrieID(root, accountHash, acc.Root), triedb, true, false)
	}
	t, err := trie.NewStateTrie(trie.StateTrieID(root), triedb)
	if err != nil {
		log.Error("Failed to open trie", "root", root, "err", err)
		return err
	}
	var (
		accounts   int
		slots      int
		codes      int
		lastReport time.Time
		start      = time.Now()
	)
	acctIt, err := t.NodeIterator(nil)
	if err != nil {
		log.Error("Failed to open iterator", "root", root, "err", err)
		return err
	}
	accIter := trie.NewIterator(acctIt)
	for accIter.Next() {
		accounts += 1
		var acc types.StateAccount
		if err := rlp.DecodeBytes(accIter.Value, &acc); err != nil {
			log.Error("Invalid account encountered during traversal", "err", err)
			return err
		}
		if acc.Root != types.EmptyRootHash {
			err := traverseStorage(trie.StorageTrieID(root, common.BytesToHash(accIter.Key), acc.Root), triedb, false, false)
			if err != nil {
				return err
			}
		}
		if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash.Bytes()) {
			if !rawdb.HasCode(chaindb, common.BytesToHash(acc.CodeHash)) {
				log.Error("Code is missing", "hash", common.BytesToHash(acc.CodeHash))
				return errors.New("missing code")
			}
			codes += 1
		}
		if time.Since(lastReport) > time.Second*8 {
			log.Info("Traversing state", "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
			lastReport = time.Now()
		}
	}
	if accIter.Err != nil {
		log.Error("Failed to traverse state trie", "root", root, "err", accIter.Err)
		return accIter.Err
	}
	log.Info("State is complete", "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
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
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var (
		root common.Hash
		err  error
	)
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "err", err)
			return err
		}
		log.Info("Start traversing the state", "root", root)
	} else {
		root = headBlock.Root()
		log.Info("Start traversing the state", "root", root, "number", headBlock.NumberU64())
	}
	// If --account is specified, only traverse the storage trie of that account.
	if accountStr := ctx.String(utils.AccountFlag.Name); accountStr != "" {
		accountHash, err := parseAccount(accountStr)
		if err != nil {
			log.Error("Failed to parse account", "err", err)
			return err
		}
		// Use raw trie since the account key is already hashed.
		t, err := trie.New(trie.StateTrieID(root), triedb)
		if err != nil {
			log.Error("Failed to open state trie", "root", root, "err", err)
			return err
		}
		acc, err := lookupAccount(accountHash, t)
		if err != nil {
			log.Error("Failed to look up account", "hash", accountHash, "err", err)
			return err
		}
		if acc.Root == types.EmptyRootHash {
			log.Info("Account has no storage", "hash", accountHash)
			return nil
		}
		return traverseStorage(trie.StorageTrieID(root, accountHash, acc.Root), triedb, true, true)
	}
	t, err := trie.NewStateTrie(trie.StateTrieID(root), triedb)
	if err != nil {
		log.Error("Failed to open trie", "root", root, "err", err)
		return err
	}
	var (
		nodes      int
		accounts   int
		slots      int
		codes      int
		lastReport time.Time
		start      = time.Now()
		hasher     = crypto.NewKeccakState()
		got        = make([]byte, 32)
	)
	accIter, err := t.NodeIterator(nil)
	if err != nil {
		log.Error("Failed to open iterator", "root", root, "err", err)
		return err
	}
	reader, err := triedb.NodeReader(root)
	if err != nil {
		log.Error("State is non-existent", "root", root)
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
			accounts += 1
			var acc types.StateAccount
			if err := rlp.DecodeBytes(accIter.LeafBlob(), &acc); err != nil {
				log.Error("Invalid account encountered during traversal", "err", err)
				return errors.New("invalid account")
			}
			if acc.Root != types.EmptyRootHash {
				err := traverseStorage(trie.StorageTrieID(root, common.BytesToHash(accIter.LeafKey()), acc.Root), triedb, false, true)
				if err != nil {
					return err
				}
			}
			if !bytes.Equal(acc.CodeHash, types.EmptyCodeHash.Bytes()) {
				if !rawdb.HasCode(chaindb, common.BytesToHash(acc.CodeHash)) {
					log.Error("Code is missing", "account", common.BytesToHash(accIter.LeafKey()))
					return errors.New("missing code")
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
		log.Error("Failed to traverse state trie", "root", root, "err", accIter.Error())
		return accIter.Error()
	}
	log.Info("State is complete", "nodes", nodes, "accounts", accounts, "slots", slots, "codes", codes, "elapsed", common.PrettyDuration(time.Since(start)))
	return nil
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

	stateIt, err := utils.NewStateIterator(triedb, db, root)
	if err != nil {
		return err
	}
	accIt, err := stateIt.AccountIterator(root, common.BytesToHash(conf.Start))
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

			stIt, err := stateIt.StorageIterator(root, accIt.Hash(), common.Hash{})
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
		hash := ctx.Args().Get(1)
		if !common.IsHexHash(hash) {
			return fmt.Errorf("invalid hash: %s", ctx.Args().Get(1))
		}
		root = common.HexToHash(hash)
	} else {
		headBlock := rawdb.ReadHeadBlock(chaindb)
		if headBlock == nil {
			log.Error("Failed to load head block")
			return errors.New("no head block")
		}
		root = headBlock.Root()
	}
	stateIt, err := utils.NewStateIterator(triedb, chaindb, root)
	if err != nil {
		return err
	}
	return utils.ExportSnapshotPreimages(chaindb, stateIt, ctx.Args().First(), root)
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

// listEIP7610EligibleAccounts traverses the post–EIP-161 state and returns all
// accounts that are eligible under EIP-7610: accounts with zero nonce, empty
// runtime code, and non-empty storage.
//
// Such accounts could only have been created before EIP-161, since after that
// all newly created contracts are initialized with a nonce of one.
//
// This helper should be generally applicable to all networks, including the
// Ethereum mainnet. For most networks where EIP-161 was enabled from genesis,
// the resulting set is expected to be empty. Otherwise, network operators are
// responsible for generating the eligible account set themselves.
//
// Notably, the exported accounts are identified by their address.
func listEIP7610EligibleAccounts(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
	defer chaindb.Close()

	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return nil
	}
	config, _, err := core.LoadChainConfig(chaindb, utils.MakeGenesis(ctx))
	if err != nil {
		log.Error("Failed to load chain config", "err", err)
		return err
	}
	if !config.IsEIP158(headBlock.Number()) {
		log.Info("Local head is prior to EIP-161", "head", headBlock.Number(), "eip-161", *config.EIP158Block)
		return nil
	}
	triedb := utils.MakeTrieDatabase(ctx, stack, chaindb, false, true, false)
	defer triedb.Close()

	if triedb.Scheme() != rawdb.PathScheme {
		log.Error("Hash scheme is not supported")
		return nil
	}
	iter, err := triedb.AccountIterator(headBlock.Root(), common.Hash{})
	if err != nil {
		log.Error("Failed to get account iterator", "err", err)
		return err
	}
	var (
		start    = time.Now()
		accounts []common.Address
	)
	for iter.Next() {
		blob := iter.Account()
		if blob == nil {
			log.Error("Failed to get account blob")
			return nil
		}
		var account types.SlimAccount
		if err := rlp.DecodeBytes(blob, &account); err != nil {
			log.Error("Failed to decode", "err", err)
			return err
		}
		// EIP-7610 account eligibility:
		// - account.nonce == 0
		// - account.runtime_code == empty
		// - account.storage != empty
		if len(account.CodeHash) == 0 && account.Nonce == 0 && len(account.Root) != 0 {
			preimage := rawdb.ReadPreimage(chaindb, iter.Hash())
			if preimage == nil {
				log.Error("Failed to read preimage", "hash", iter.Hash().Hex())
				return nil
			}
			accounts = append(accounts, common.BytesToAddress(preimage))
		}
	}
	if len(accounts) == 0 {
		log.Info("Traversed state", "eligible", len(accounts), "elapsed", common.PrettyDuration(time.Since(start)))
	} else {
		sort.Slice(accounts, func(i, j int) bool {
			return accounts[i].Cmp(accounts[j]) < 0
		})
		buf := make([]byte, len(accounts)*common.AddressLength)
		for i, h := range accounts {
			copy(buf[i*common.AddressLength:], h[:])
		}
		log.Info("Traversed state", "eligible", len(accounts), "elapsed", common.PrettyDuration(time.Since(start)), "output", hex.EncodeToString(buf))
	}
	return nil
}
