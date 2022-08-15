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
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/state/pruner"
	"github.com/ethereum/go-ethereum/core/state/snapshot"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	trieUtils "github.com/ethereum/go-ethereum/trie/utils"
	"github.com/gballet/go-verkle"
	"github.com/holiman/uint256"
	"github.com/shirou/gopsutil/v3/mem"
	cli "github.com/urfave/cli/v2"
)

var (
	// emptyRoot is the known root hash of an empty trie.
	emptyRoot = common.HexToHash("56e81f171bcc55a6ff8345e692c0f86e5b48e01b996cadc001622fb5e363b421")

	// emptyCode is the known hash of the empty EVM bytecode.
	emptyCode = crypto.Keccak256(nil)
)

var (
	snapshotCommand = &cli.Command{
		Name:        "snapshot",
		Usage:       "A set of commands based on the snapshot",
		Category:    "MISCELLANEOUS COMMANDS",
		Description: "",
		Subcommands: []*cli.Command{
			{
				Name:      "prune-state",
				Usage:     "Prune stale ethereum state data based on the snapshot",
				ArgsUsage: "<root>",
				Action:    pruneState,
				Flags: utils.GroupFlags([]cli.Flag{
					utils.CacheTrieJournalFlag,
					utils.BloomFilterSizeFlag,
				}, utils.NetworkFlags, utils.DatabasePathFlags),
				Description: `
geth snapshot prune-state <state-root>
will prune historical state data with the help of the state snapshot.
All trie nodes and contract codes that do not belong to the specified
version state will be deleted from the database. After pruning, only
two version states are available: genesis and the specific one.

The default pruning target is the HEAD-127 state.

WARNING: It's necessary to delete the trie clean cache after the pruning.
If you specify another directory for the trie clean cache via "--cache.trie.journal"
during the use of Geth, please also specify it here for correct deletion. Otherwise
the trie clean cache with default directory will be deleted.
`,
			},
			{
				Name:      "verify-state",
				Usage:     "Recalculate state hash based on the snapshot for verification",
				ArgsUsage: "<root>",
				Action:    verifyState,
				Flags:     utils.GroupFlags(utils.NetworkFlags, utils.DatabasePathFlags),
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
				Flags:     utils.GroupFlags(utils.NetworkFlags, utils.DatabasePathFlags),
				Description: `
geth snapshot check-dangling-storage <state-root> traverses the snap storage 
data, and verifies that all snapshot storage data has a corresponding account. 
`,
			},
			{
				Name:      "inspect-account",
				Usage:     "Check all snapshot layers for the a specific account",
				ArgsUsage: "<address | hash>",
				Action:    checkAccount,
				Flags:     utils.GroupFlags(utils.NetworkFlags, utils.DatabasePathFlags),
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
				Flags:     utils.GroupFlags(utils.NetworkFlags, utils.DatabasePathFlags),
				Description: `
geth snapshot traverse-state <state-root>
will traverse the whole state from the given state root and will abort if any
referenced trie node or contract code is missing. This command can be used for
state integrity verification. The default checking target is the HEAD state.

It's also usable without snapshot enabled.
`,
			},
			{
				Name:      "traverse-rawstate",
				Usage:     "Traverse the state with given root hash and perform detailed verification",
				ArgsUsage: "<root>",
				Action:    traverseRawState,
				Flags:     utils.GroupFlags(utils.NetworkFlags, utils.DatabasePathFlags),
				Description: `
geth snapshot traverse-rawstate <state-root>
will traverse the whole state from the given root and will abort if any referenced
trie node or contract code is missing. This command can be used for state integrity
verification. The default checking target is the HEAD state. It's basically identical
to traverse-state, but the check granularity is smaller. 

It's also usable without snapshot enabled.
`,
			},
			{
				Name:      "dump",
				Usage:     "Dump a specific block from storage (same as 'geth dump' but using snapshots)",
				ArgsUsage: "[? <blockHash> | <blockNum>]",
				Action:    dumpState,
				Flags: utils.GroupFlags([]cli.Flag{
					utils.ExcludeCodeFlag,
					utils.ExcludeStorageFlag,
					utils.StartKeyFlag,
					utils.DumpLimitFlag,
				}, utils.NetworkFlags, utils.DatabasePathFlags),
				Description: `
This command is semantically equivalent to 'geth dump', but uses the snapshots
as the backend data source, making this command a lot faster. 

The argument is interpreted as block number or hash. If none is provided, the latest
block is used.
`,
			},
			{
				Name:      "to-verkle",
				Usage:     "use the snapshot to compute a translation of a MPT into a verkle tree",
				ArgsUsage: "<root>",
				Action:    convertToVerkle,
				Flags: utils.GroupFlags([]cli.Flag{
					utils.VerkleConversionInsertRangeStartFlag,
					utils.VerkleConversionInsertRangeSizeFlag,
				}, utils.NetworkFlags, utils.DatabasePathFlags),
				Description: `
geth snapshot to-verkle <state-root>
This command takes a snapshot and inserts its values in a fresh verkle tree.

The argument is interpreted as the root hash. If none is provided, the latest
block is used.
 `,
			},
			{
				Name:      "verify-verkle",
				Usage:     "verify the translation of a MPT into a verkle tree",
				ArgsUsage: "<root>",
				Action:    verifyVerkle,
				Flags: utils.GroupFlags([]cli.Flag{
					utils.VerkleConversionInsertRangeStartFlag,
					utils.VerkleConversionInsertRangeSizeFlag,
				}, utils.NetworkFlags, utils.DatabasePathFlags),
				Description: `
geth snapshot to-verkle <state-root>
This command takes a snapshot and inserts its values in a fresh verkle tree.

The argument is interpreted as the root hash. If none is provided, the latest
block is used.
 `,
			},
		},
	}
)

func pruneState(ctx *cli.Context) error {
	stack, config := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, false)
	pruner, err := pruner.NewPruner(chaindb, stack.ResolvePath(""), stack.ResolvePath(config.Eth.TrieCleanCacheJournal), ctx.Uint64(utils.BloomFilterSizeFlag.Name))
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
	headBlock := rawdb.ReadHeadBlock(chaindb)
	if headBlock == nil {
		log.Error("Failed to load head block")
		return errors.New("no head block")
	}
	snaptree, err := snapshot.New(chaindb, trie.NewDatabase(chaindb), 256, headBlock.Root(), false, false, false)
	if err != nil {
		log.Error("Failed to open snapshot tree", "err", err)
		return err
	}
	if ctx.NArg() > 1 {
		log.Error("Too many arguments given")
		return errors.New("too many arguments")
	}
	var root = headBlock.Root()
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "err", err)
			return err
		}
	}
	if err := snaptree.Verify(root); err != nil {
		log.Error("Failed to verify state", "root", root, "err", err)
		return err
	}
	log.Info("Verified the state", "root", root)
	return snapshot.CheckDanglingStorage(chaindb)
}

// checkDanglingStorage iterates the snap storage data, and verifies that all
// storage also has corresponding account data.
func checkDanglingStorage(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	return snapshot.CheckDanglingStorage(utils.MakeChainDatabase(ctx, stack, true))
}

// traverseState is a helper function used for pruning verification.
// Basically it just iterates the trie, ensure all nodes and associated
// contract codes are present.
func traverseState(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
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
	triedb := trie.NewDatabase(chaindb)
	t, err := trie.NewSecure(common.Hash{}, root, triedb)
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
	accIter := trie.NewIterator(t.NodeIterator(nil))
	for accIter.Next() {
		accounts += 1
		var acc types.StateAccount
		if err := rlp.DecodeBytes(accIter.Value, &acc); err != nil {
			log.Error("Invalid account encountered during traversal", "err", err)
			return err
		}
		if acc.Root != emptyRoot {
			storageTrie, err := trie.NewSecure(common.BytesToHash(accIter.Key), acc.Root, triedb)
			if err != nil {
				log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
				return err
			}
			storageIter := trie.NewIterator(storageTrie.NodeIterator(nil))
			for storageIter.Next() {
				slots += 1
			}
			if storageIter.Err != nil {
				log.Error("Failed to traverse storage trie", "root", acc.Root, "err", storageIter.Err)
				return storageIter.Err
			}
		}
		if !bytes.Equal(acc.CodeHash, emptyCode) {
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
	triedb := trie.NewDatabase(chaindb)
	t, err := trie.NewSecure(common.Hash{}, root, triedb)
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
	accIter := t.NodeIterator(nil)
	for accIter.Next(true) {
		nodes += 1
		node := accIter.Hash()

		// Check the present for non-empty hash node(embedded node doesn't
		// have their own hash).
		if node != (common.Hash{}) {
			blob := rawdb.ReadTrieNode(chaindb, node)
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
			if acc.Root != emptyRoot {
				storageTrie, err := trie.NewSecure(common.BytesToHash(accIter.LeafKey()), acc.Root, triedb)
				if err != nil {
					log.Error("Failed to open storage trie", "root", acc.Root, "err", err)
					return errors.New("missing storage trie")
				}
				storageIter := storageTrie.NodeIterator(nil)
				for storageIter.Next(true) {
					nodes += 1
					node := storageIter.Hash()

					// Check the present for non-empty hash node(embedded node doesn't
					// have their own hash).
					if node != (common.Hash{}) {
						blob := rawdb.ReadTrieNode(chaindb, node)
						if len(blob) == 0 {
							log.Error("Missing trie node(storage)", "hash", node)
							return errors.New("missing storage")
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
			if !bytes.Equal(acc.CodeHash, emptyCode) {
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

	conf, db, root, err := parseDumpConfig(ctx, stack)
	if err != nil {
		return err
	}
	snaptree, err := snapshot.New(db, trie.NewDatabase(db), 256, root, false, false, false)
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
		account, err := snapshot.FullAccount(accIt.Account())
		if err != nil {
			return err
		}
		da := &state.DumpAccount{
			Balance:   account.Balance.String(),
			Nonce:     account.Nonce,
			Root:      account.Root,
			CodeHash:  account.CodeHash,
			SecureKey: accIt.Hash().Bytes(),
		}
		if !conf.SkipCode && !bytes.Equal(account.CodeHash, emptyCode) {
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

func convertToVerkle(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, false)
	if chaindb == nil {
		return errors.New("nil chaindb")
	}
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
			log.Error("Failed to resolve state root", "error", err)
			return err
		}
		log.Info("Start traversing the state", "root", root)
	} else {
		root = headBlock.Root()
		log.Info("Start traversing the state", "root", root, "number", headBlock.NumberU64())
	}

	var (
		accounts   int
		lastReport time.Time
		start      = time.Now()
		rangeStart = ctx.Uint64(utils.VerkleConversionInsertRangeStartFlag.Name)
		rangeEnd   = rangeStart + ctx.Uint64(utils.VerkleConversionInsertRangeSizeFlag.Name)
		wg         sync.WaitGroup
		flushError error
	)

	if rangeEnd > 256 {
		rangeEnd = 256
	}

	flushCh := make(chan verkle.VerkleNode)
	saveverkle := func(node verkle.VerkleNode) {
		flushCh <- node
	}
	var flushWg sync.WaitGroup
	flushWg.Add(1)
	go func() {
		for node := range flushCh {
			comm := node.ComputeCommitment()
			s, err := node.Serialize()
			if err != nil {
				panic(err)
			}
			commB := comm.Bytes()
			if err := chaindb.Put(commB[:], s); err != nil {
				flushError = err
				break
			}
		}
		flushWg.Done()
	}()

	snaptree, err := snapshot.New(chaindb, trie.NewDatabase(chaindb), 256, root, false, false, false)
	if err != nil {
		return err
	}
	accIt, err := snaptree.AccountIterator(root, common.Hash{})
	if err != nil {
		return err
	}
	defer accIt.Release()

	type treeHugger struct {
		node *verkle.LeafNode
		stem []byte
	}
	treeHuggers := make([]chan *treeHugger, runtime.NumCPU())
	subRoots := make([]*verkle.InternalNode, runtime.NumCPU())
	rootPerCPU := (256 + runtime.NumCPU() - 1) / runtime.NumCPU()
	for i := range treeHuggers {
		treeHuggers[i] = make(chan *treeHugger, 128)
		subRoots[i] = verkle.New().(*verkle.InternalNode)

		// save references for the goroutine to capture
		hugger := treeHuggers[i]
		root := subRoots[i]
		wg.Add(1)

		go func() {
			ticker := time.NewTicker(10 * time.Second)
			defer ticker.Stop()
			for hug := range hugger {
				select {
				case <-ticker.C:
					// Check the memory usage every 10 seconds. If it
					// goes above a given watermark, flush the tree's
					// lower nodes to disk.
					v, _ := mem.VirtualMemory()

					// Compute flushing depth
					// 1 if > 80%, 2 if 80% < && > 60%...
					// don't bother cleaning up above 60%
					depth := 5 - uint8(v.UsedPercent/20)

					if depth < 3 {
						root.FlushAtDepth(depth, saveverkle)
					}
				default:
				}

				if uint64(hug.stem[0]) < rangeStart && uint64(hug.stem[0]) > rangeEnd {
					// skip stem outside the insertion range
					continue
				}

				hug.node.ComputeCommitment()
				hashed := hug.node.ToHashedNode()
				flushCh <- hug.node
				root.InsertStem(hug.stem, hashed, chaindb.Get)
			}
			wg.Done()
		}()
	}

	// Process all accounts sequentially
	for accIt.Next() {
		accounts += 1
		acc, err := snapshot.FullAccount(accIt.Account())
		if err != nil {
			log.Error("Invalid account encountered during traversal", "error", err)
			return err
		}

		// Store the basic account data
		var (
			nonce, balance, version [32]byte
			newValues               = make([][]byte, 256)
		)
		newValues[0] = version[:]
		newValues[1] = balance[:]
		newValues[2] = nonce[:]
		newValues[4] = version[:] // memory-saving trick: by default, an account has 0 size
		binary.LittleEndian.PutUint64(nonce[:8], acc.Nonce)
		for i, b := range acc.Balance.Bytes() {
			balance[len(acc.Balance.Bytes())-1-i] = b
		}
		addr := rawdb.ReadPreimage(chaindb, accIt.Hash())
		if addr == nil {
			return fmt.Errorf("could not find preimage for address %x %v %v", accIt.Hash(), acc, accIt.Error())
		}
		stem := trieUtils.GetTreeKeyVersion(addr)

		// Store the account code if present
		if !bytes.Equal(acc.CodeHash, emptyCode) {
			code := rawdb.ReadCode(chaindb, common.BytesToHash(acc.CodeHash))
			chunks := trie.ChunkifyCode(code)

			for i := 0; i < 128 && i < len(chunks)/32; i++ {
				newValues[128+i] = chunks[32*i : 32*(i+1)]
			}

			for i := 128; i < len(chunks)/32; {
				values := make([][]byte, 256)
				chunkkey := trieUtils.GetTreeKeyCodeChunk(addr, uint256.NewInt(uint64(i)))
				j := i
				for ; (j-i) < 256 && j < len(chunks)/32; j++ {
					values[(j-128)%256] = chunks[32*j : 32*(j+1)]
				}
				i = j

				// Otherwise, store the previous group in the tree with a
				// stem insertion.
				treeHuggers[int(chunkkey[0])/rootPerCPU] <- &treeHugger{stem: chunkkey[:31], node: verkle.NewLeafNode(chunkkey[:31], values)}
			}

			// Write the code size in the account header group
			var size [32]byte
			newValues[4] = size[:]
			binary.LittleEndian.PutUint64(size[:8], uint64(len(code)))
		}

		// Save every slot into the tree
		if !bytes.Equal(acc.Root, emptyRoot[:]) {
			var (
				laststem [31]byte
				values   = make([][]byte, 256)
			)
			copy(laststem[:], stem)

			storageIt, err := snaptree.StorageIterator(root, accIt.Hash(), common.Hash{})
			if err != nil {
				log.Error("Failed to open storage trie", "root", acc.Root, "error", err)
				return err
			}
			for storageIt.Next() {
				slotnr := rawdb.ReadPreimage(chaindb, storageIt.Hash())
				if slotnr == nil {
					return fmt.Errorf("could not find preimage for slot %x", storageIt.Hash())
				}
				slotkey := trieUtils.GetTreeKeyStorageSlot(addr, uint256.NewInt(0).SetBytes(slotnr))

				var value [32]byte
				copy(value[:len(storageIt.Slot())-1], storageIt.Slot())

				// if the slot belongs to the header group, store it there
				if bytes.Equal(slotkey[:31], stem) {
					newValues[int(slotkey[31])] = value[:]
					continue
				}

				// if the slot belongs to the same group as the previous
				// one, add it to the current group of values.
				if bytes.Equal(laststem[:], slotkey[:31]) {
					values[slotkey[31]] = value[:]
					continue
				}

				// flush the previous group, iff it's not the header group
				if !bytes.Equal(stem[:31], laststem[:]) {
					treeHuggers[int(laststem[0])/rootPerCPU] <- &treeHugger{stem: laststem[:], node: verkle.NewLeafNode(laststem[:], values)}
				}
			}
			if !bytes.Equal(laststem[:31], stem[:31]) {
				treeHuggers[int(laststem[0])/rootPerCPU] <- &treeHugger{stem: laststem[:], node: verkle.NewLeafNode(laststem[:], values)}
			}
			storageIt.Release()
			if storageIt.Error() != nil {
				log.Error("Failed to traverse storage trie", "root", acc.Root, "error", storageIt.Error())
				return storageIt.Error()
			}
		}
		// Finish with storing the complete account header group inside the tree.
		treeHuggers[int(stem[0])/rootPerCPU] <- &treeHugger{stem: stem[:], node: verkle.NewLeafNode(stem[:31], newValues)}

		if time.Since(lastReport) > time.Second*8 {
			log.Info("Traversing state", "accounts", accounts, "elapsed", common.PrettyDuration(time.Since(start)))
			lastReport = time.Now()
		}
	}
	if accIt.Error() != nil {
		log.Error("Failed to compute commitment", "root", root, "error", accIt.Error())
		return accIt.Error()
	}
	log.Info("Wrote all leaves", "accounts", accounts, "elapsed", common.PrettyDuration(time.Since(start)))
	for _, hugger := range treeHuggers {
		close(hugger)
	}
	wg.Wait()
	if flushError != nil {
		log.Error("Error encountered by the flusing goroutine", "error", flushError)
	}

	vRoot := verkle.MergeTrees(subRoots)
	vRoot.ComputeCommitment()
	vRoot.(*verkle.InternalNode).Flush(saveverkle)
	close(flushCh)
	flushWg.Wait()

	if rangeStart != 0 || rangeEnd != 256 {
		children := vRoot.(*verkle.InternalNode).Children()
		// Print partial subtree root commitments, as only a partial tree has been built
		log.Info("Conversion complete", "accounts", accounts, "elapsed", common.PrettyDuration(time.Since(start)))
		for i := rangeStart; i < rangeEnd; i++ {
			log.Info("Root commitment at depth 1", "offset", i, "commitment", fmt.Sprintf("%x", children[i].ComputeCommitment().Bytes()))
		}
	} else {
		log.Info("Conversion complete", "root commitment", fmt.Sprintf("%x", vRoot.ComputeCommitment().Bytes()), "accounts", accounts, "elapsed", common.PrettyDuration(time.Since(start)))
	}
	return nil
}

var zero [32]byte

// recurse into each child to ensure they can be loaded from the db. The tree isn't rebuilt
// (only its nodes are loaded) so there is no need to flush them, the garbage collector should
// take care of that for us.
func checkChildren(root verkle.VerkleNode, resolver verkle.NodeResolverFn) error {
	switch node := root.(type) {
	case *verkle.InternalNode:
		for i, child := range node.Children() {
			childC := child.ComputeCommitment().Bytes()

			childS, err := resolver(childC[:])
			if bytes.Equal(childC[:], zero[:]) {
				continue
			}
			if err != nil {
				return fmt.Errorf("could not find child %x in db: %w", childC, err)
			}
			// depth is set to 0, the tree isn't rebuilt so it's not a problem
			childN, err := verkle.ParseNode(childS, 0, childC[:])
			if err != nil {
				return fmt.Errorf("decode error child %x in db: %w", child.ComputeCommitment().Bytes(), err)
			}
			if err := checkChildren(childN, resolver); err != nil {
				return fmt.Errorf("%x%w", i, err) // write the path to the erroring node
			}
		}
	case *verkle.LeafNode:
		// sanity check: ensure at least one value is non-zero

		for i := 0; i < verkle.NodeWidth; i++ {
			if len(node.Value(i)) != 0 {
				return nil
			}
		}
		return fmt.Errorf("Both balance and nonce are 0")
	case verkle.Empty:
		// nothing to do
	default:
		return fmt.Errorf("unsupported type encountered %v", root)
	}

	return nil
}

func verifyVerkle(ctx *cli.Context) error {
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	chaindb := utils.MakeChainDatabase(ctx, stack, true)
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
		rootC common.Hash
		err   error
	)
	if ctx.NArg() == 1 {
		rootC, err = parseRoot(ctx.Args().First())
		if err != nil {
			log.Error("Failed to resolve state root", "error", err)
			return err
		}
		log.Info("Rebuilding the tree", "root", rootC)
	} else {
		rootC = headBlock.Root()
		log.Info("Rebuilding the tree", "root", rootC, "number", headBlock.NumberU64())
	}

	var (
		//start      = time.Now()
		rangeStart = ctx.Uint64(utils.VerkleConversionInsertRangeStartFlag.Name)
		rangeEnd   = rangeStart + ctx.Uint64(utils.VerkleConversionInsertRangeSizeFlag.Name)
	)

	if rangeEnd > 256 {
		rangeEnd = 256
	}

	serializedRoot, err := chaindb.Get(rootC[:])
	if err != nil {
		return err
	}
	root, err := verkle.ParseNode(serializedRoot, 0, rootC[:])
	if err != nil {
		return err
	}

	if err := checkChildren(root, chaindb.Get); err != nil {
		log.Error("Could not rebuild the tree from the database", "err", err)
		return err
	}

	log.Info("Tree was rebuilt from the database")
	return nil
}
