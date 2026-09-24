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
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/holiman/uint256"
	"github.com/urfave/cli/v2"
)

var (
	deleteSourceFlag = &cli.BoolFlag{
		Name:  "delete-source",
		Usage: "Delete the merkle state after the conversion verifies",
	}
	memoryLimitFlag = &cli.Uint64Flag{
		Name:  "memory-limit",
		Usage: "Total sort-buffer budget in MB before records spill to disk",
		Value: 4096,
	}
	tmpDirFlag = &cli.StringFlag{
		Name:  "tmpdir",
		Usage: "Directory for the sort's spill files (default: the OS temp dir)",
	}
	forceConvertFlag = &cli.BoolFlag{
		Name:  "force",
		Usage: "Wipe any existing binary tree state before converting",
	}
	snapshotOutFlag = &cli.StringFlag{
		Name:  "snapshot-out",
		Usage: "File to write the byte-canonical PBT snapshot artifact to",
	}
	preimagesOutFlag = &cli.StringFlag{
		Name:  "preimages-out",
		Usage: "File to write the address-sorted preimage file to",
	}
	dropPreimagesFlag = &cli.BoolFlag{
		Name:  "drop-preimages",
		Usage: "Delete the preimage store after the conversion verifies",
	}

	bintrieCommand = &cli.Command{
		Name:        "bintrie",
		Usage:       "A set of commands for binary trie operations",
		Description: "",
		Subcommands: []*cli.Command{
			bintrieImportCommand,
			{
				Name:      "convert",
				Usage:     "Convert MPT state to binary trie",
				ArgsUsage: "[state-root]",
				Action:    convertToBinaryTrie,
				Flags: slices.Concat([]cli.Flag{
					deleteSourceFlag,
					memoryLimitFlag,
					tmpDirFlag,
					forceConvertFlag,
					snapshotOutFlag,
					preimagesOutFlag,
					dropPreimagesFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth bintrie convert [flags] [state-root]

Converts the MPT state into the EIP-8297 binary tree offline, following the
EIP-8347 pipeline, and verifies the result before marking it complete.
Defaults to the head root. The source must hold preimages
(--cache.preimages). --snapshot-out and --preimages-out emit the EIP-8347
distribution artifacts. --drop-preimages deletes the preimage store
afterwards, which also rules out a re-conversion.
`,
			},
		},
	}
)

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
	// --delete-source is settled before anything is wiped or converted: it
	// disposes of state the chain may still be running on, and learning that
	// only after a full conversion would be a long wait for a refusal.
	if ctx.Bool(deleteSourceFlag.Name) {
		if !rawdb.ReadPBTMigrationDone(chaindb) {
			return errors.New("refusing --delete-source: the migration window has not closed, so the merkle trie is still the chain's own state")
		}
		stored := rawdb.ReadChainConfig(chaindb, rawdb.ReadCanonicalHash(chaindb, 0))
		if stored == nil || !stored.IsBinaryTrie(headBlock.Number(), headBlock.Time()) {
			return errors.New("refusing --delete-source: the head block commits the merkle trie, which the node still executes on")
		}
	}
	log.Info("Starting MPT to binary trie conversion", "root", root, "block", headBlock.NumberU64())

	snapshotPath, preimagePath := ctx.String(snapshotOutFlag.Name), ctx.String(preimagesOutFlag.Name)
	if snapshotPath != "" && snapshotPath == preimagePath {
		return errors.New("--snapshot-out and --preimages-out must name different files")
	}
	if (snapshotPath == "") != (preimagePath == "") {
		log.Warn("Emitting one distribution artifact without the other; EIP-8347 distributes the snapshot and the preimage file together")
	}
	// MB to bytes without wrapping the platform int.
	budgetMB := ctx.Uint64(memoryLimitFlag.Name)
	if budgetMB > math.MaxInt>>20 {
		budgetMB = math.MaxInt >> 20
	}
	// Probe before MakeTrieDatabase: its guard fatals on a completed
	// conversion and cannot see interrupted debris.
	if ctx.Bool(forceConvertFlag.Name) {
		if !hasMerkleSource(chaindb, root) {
			return errors.New("refusing --force: no merkle state to reconvert from")
		}
		if err := wipeBinaryTrieState(chaindb, stack.ResolvePath("triedb")); err != nil {
			return fmt.Errorf("failed to wipe binary tree state: %w", err)
		}
	} else if dirty, err := hasBinaryTrieState(chaindb); err != nil {
		return fmt.Errorf("failed to probe the binary tree namespace: %w", err)
	} else if dirty {
		if !hasMerkleSource(chaindb, root) {
			return errors.New("binary tree state present, merkle state gone; re-import rather than reconvert")
		}
		return errors.New("database already holds binary tree state, complete or from an interrupted conversion; re-run with --force to wipe and reconvert")
	}
	srcTriedb := utils.MakeTrieDatabase(ctx, stack, chaindb, true, true, false)
	// Closed once, by whichever path gets there first: --delete-source has to
	// close it early to reset the freezers it holds open.
	closeSource := sync.OnceValue(srcTriedb.Close)
	defer closeSource()
	// The disposal deletes whole key families; on the hash scheme they also
	// hold bare-hash code the converted node still reads.
	if ctx.Bool(deleteSourceFlag.Name) && srcTriedb.Scheme() != rawdb.PathScheme {
		return errors.New("refusing --delete-source: needs the path scheme")
	}

	binRoot, err := convertState(chaindb, srcTriedb, root, conversionOptions{
		sortBudget:   int(budgetMB << 20),
		tmpDir:       ctx.String(tmpDirFlag.Name),
		snapshotPath: snapshotPath,
		preimagePath: preimagePath,
	})
	if err != nil {
		return err
	}
	log.Info("Conversion complete", "binaryRoot", binRoot)

	if ctx.Bool(deleteSourceFlag.Name) {
		log.Info("Deleting source MPT data")
		// Marker first: a killed deletion must not pass for an intact store.
		rawdb.WritePBTMerkleDisposed(chaindb)
		// The source holds the history freezers open.
		if err := closeSource(); err != nil {
			return fmt.Errorf("failed to close the source trie database: %w", err)
		}
		if err := core.DisposeMerkleState(chaindb, stack.ResolvePath("triedb"), nil); err != nil {
			return fmt.Errorf("MPT deletion failed: %w", err)
		}
		log.Info("Source MPT data deleted")
	}
	if ctx.Bool(dropPreimagesFlag.Name) {
		// The trie database flushes on close; wipe once it has let go.
		if err := closeSource(); err != nil {
			return fmt.Errorf("failed to close the source trie database: %w", err)
		}
		if err := wipePreimages(chaindb); err != nil {
			return fmt.Errorf("preimage deletion failed: %w", err)
		}
	}
	return nil
}

// conversionOptions carries the tunables of a conversion run.
type conversionOptions struct {
	sortBudget   int    // total bytes of buffered records before the sorts spill
	tmpDir       string // spill directory; empty means the OS temp dir
	snapshotPath string // PBT snapshot artifact destination; empty writes none
	preimagePath string // preimage file destination; empty writes none
}

// conversionStats tracks progress for the periodic report. The message names
// the work, since the converter and the importer share the counters.
type conversionStats struct {
	what       string
	accounts   uint64
	slots      uint64
	codes      uint64
	leaves     uint64
	start      time.Time
	lastReport time.Time
}

func newStats(what string) *conversionStats {
	return &conversionStats{what: what, start: time.Now(), lastReport: time.Now()}
}

func (s *conversionStats) report(force bool) {
	if !force && time.Since(s.lastReport) < 8*time.Second {
		return
	}
	s.lastReport = time.Now()
	log.Info(s.what, "accounts", s.accounts, "slots", s.slots,
		"codes", s.codes, "leaves", s.leaves, "elapsed", common.PrettyDuration(time.Since(s.start)))
}

// convertState converts the MPT state at root into the binary tree namespace
// per EIP-8347: one scan derives every leaf and streams flat state, an
// external sort orders the leaves in tree-key order, and the tree builds
// bottom-up in one pass. Three checks gate the result: the scan against the
// source state root, the tree and the flat store against the converted one.
// The attestation lands last, so an interrupted run refuses to open.
func convertState(chaindb ethdb.Database, srcTriedb *triedb.Database, root common.Hash, opts conversionOptions) (common.Hash, error) {
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))

	// A non-virgin namespace, completed or debris, needs --force.
	if dirty, err := hasBinaryTrieState(chaindb); err != nil {
		return common.Hash{}, err
	} else if dirty {
		return common.Hash{}, errors.New("binary tree namespace is not empty; re-run with --force to wipe it")
	}
	stats := newStats("Converting state")

	// Four sorters can be alive at once, and a sealed one keeps its buffer
	// while its stream drains, so the budget is split rather than reused.
	parts := 3
	if opts.preimagePath != "" {
		parts = 4
	}
	share := opts.sortBudget / parts

	sorter := bintrie.NewLeafSorter(opts.tmpDir, share)
	defer sorter.Close()

	// The re-derivation limbs, keyed by account hash and by account‖slot.
	accounts := bintrie.NewRecordSorter(opts.tmpDir, share, nil)
	defer accounts.Close()
	slots := bintrie.NewRecordSorter(opts.tmpDir, share, nil)
	defer slots.Close()

	var (
		snapshot  *snapshotWriter
		preimages *preimageFile
	)
	if opts.snapshotPath != "" {
		sw, err := newSnapshotWriter(opts.snapshotPath)
		if err != nil {
			return common.Hash{}, err
		}
		snapshot = sw
		defer func() {
			if snapshot != nil {
				snapshot.abort()
			}
		}()
	}
	if opts.preimagePath != "" {
		preimages = newPreimageFile(opts.tmpDir, share)
		defer preimages.close()
	}
	// Phase 1: scan, deriving leaves and streaming flat state.
	if err := deriveLeaves(chaindb, pbtdb, srcTriedb, root, sorter, accounts, slots, preimages, stats); err != nil {
		return common.Hash{}, err
	}
	stats.report(true)

	// An empty state would finalize a zero root; not worth supporting.
	if stats.leaves == 0 {
		return common.Hash{}, errors.New("refusing to convert an empty state")
	}

	// Check 1, before the build: the other two checks are fed by this same
	// record set, so only this one ties the output to the source state.
	if err := verifySourceRoot(accounts, slots, root, stats.start); err != nil {
		return common.Hash{}, err
	}
	accounts.Close()
	slots.Close()

	// Phase 2+3: sort and build, streaming records to disk.
	stream, err := sorter.Sort()
	if err != nil {
		return common.Hash{}, err
	}
	var (
		batch      = pbtdb.NewBatch()
		written    uint64
		builderErr error
	)
	builder := bintrie.NewStackBuilder(func(path []byte, hash common.Hash, blob []byte) {
		if builderErr != nil {
			return
		}
		rawdb.WriteAccountTrieNode(batch, path, blob)
		written++
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				builderErr = err
				return
			}
			batch.Reset()
		}
	})
	for {
		key, value, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return common.Hash{}, err
		}
		if snapshot != nil {
			if err := snapshot.add(key, value); err != nil {
				return common.Hash{}, err
			}
		}
		if err := builder.Add(key, value); err != nil {
			return common.Hash{}, err
		}
	}
	binRoot := builder.Finish()
	if builderErr != nil {
		return common.Hash{}, builderErr
	}
	if err := batch.Write(); err != nil {
		return common.Hash{}, err
	}
	log.Info("Built binary tree", "nodes", written, "leaves", stats.leaves)

	// Verify both stores before anything becomes publishable or openable.
	if err := verifyConvertedState(chaindb, binRoot); err != nil {
		return common.Hash{}, fmt.Errorf("converted tree failed verification: %w", err)
	}
	if err := verifyFlatState(chaindb, pbtdb, srcTriedb, binRoot, opts); err != nil {
		return common.Hash{}, fmt.Errorf("converted flat state failed verification: %w", err)
	}

	if snapshot != nil {
		digest, err := snapshot.finalize(binRoot)
		if err != nil {
			return common.Hash{}, err
		}
		log.Info("Wrote PBT snapshot artifact", "path", opts.snapshotPath,
			"leaves", snapshot.count, "digest", digest)
		snapshot = nil // finalized; disarm the abort
	}
	if preimages != nil {
		digest, err := preimages.write(opts.preimagePath)
		if err != nil {
			os.Remove(opts.preimagePath)
			return common.Hash{}, err
		}
		log.Info("Wrote preimage file", "path", opts.preimagePath,
			"accounts", preimages.accounts, "digest", digest)
	}

	// No state id: the converted tree bases an empty history and live commits
	// number from 1. The attestation is the completion marker and comes last.
	rawdb.WriteSnapshotRoot(pbtdb, binRoot)
	rawdb.WritePBTFlatState(pbtdb)
	return binRoot, nil
}

// deriveLeaves walks the merkle state at root, deriving every tree leaf into
// the sorter, writing flat state alongside, and recording the merkle
// re-derivation limbs the source-root check folds back up.
func deriveLeaves(chaindb ethdb.Database, pbtdb ethdb.Database, srcTriedb *triedb.Database, root common.Hash, sorter, accounts, slots *bintrie.RecordSorter, preimages *preimageFile, stats *conversionStats) error {
	srcTrie, err := trie.NewStateTrie(trie.StateTrieID(root), srcTriedb)
	if err != nil {
		return fmt.Errorf("failed to open source trie: %w", err)
	}
	acctIt, err := srcTrie.NodeIterator(nil)
	if err != nil {
		return fmt.Errorf("failed to create account iterator: %w", err)
	}
	var (
		accIter   = trie.NewIterator(acctIt)
		flatBatch = pbtdb.NewBatch()
		seenCode  = make(map[common.Hash]struct{})
	)
	emit := func(key []byte, value [32]byte) error {
		// Zero values resolve to absence and are never written.
		if value == ([32]byte{}) {
			return nil
		}
		stats.leaves++
		return sorter.Add(key, value[:])
	}
	for accIter.Next() {
		var acc types.StateAccount
		if err := rlp.DecodeBytes(accIter.Value, &acc); err != nil {
			return fmt.Errorf("invalid account RLP: %w", err)
		}
		addrBytes := srcTrie.GetKey(accIter.Key)
		if addrBytes == nil {
			return fmt.Errorf("missing preimage for account hash %x (the source node must have synced with --cache.preimages)", accIter.Key)
		}
		if err := checkPreimage(addrBytes, common.BytesToHash(accIter.Key), common.AddressLength); err != nil {
			return err
		}
		addr := common.BytesToAddress(addrBytes)
		if preimages != nil {
			if err := preimages.beginAccount(addr); err != nil {
				return err
			}
		}

		if err := emitAccountHeader(chaindb, addr, acc.Nonce, acc.Balance, common.BytesToHash(acc.CodeHash), seenCode, stats, emit); err != nil {
			return err
		}

		// Normalize the storage root: replaying nodes record EmptyRootHash.
		accountHash := common.BytesToHash(accIter.Key)
		slim := acc
		slim.Root = types.EmptyRootHash
		rawdb.WriteAccountSnapshot(flatBatch, accountHash, types.SlimAccountRLP(slim))
		if err := accounts.Add(accountHash.Bytes(), merkleAccountRecord(acc.Nonce, acc.Balance, common.BytesToHash(acc.CodeHash))); err != nil {
			return err
		}

		if acc.Root != types.EmptyRootHash {
			storageTrie, err := trie.NewStateTrie(trie.StorageTrieID(root, accountHash, acc.Root), srcTriedb)
			if err != nil {
				return fmt.Errorf("failed to open storage trie for %x: %w", addr, err)
			}
			storageIt, err := storageTrie.NodeIterator(nil)
			if err != nil {
				return fmt.Errorf("failed to create storage iterator for %x: %w", addr, err)
			}
			storageIter := trie.NewIterator(storageIt)
			for storageIter.Next() {
				slotKey := storageTrie.GetKey(storageIter.Key)
				if slotKey == nil {
					return fmt.Errorf("missing preimage for storage key %x (account %x)", storageIter.Key, addr)
				}
				if err := checkPreimage(slotKey, common.BytesToHash(storageIter.Key), common.HashLength); err != nil {
					return err
				}
				if preimages != nil {
					preimages.addSlot(slotKey)
				}
				_, content, rest, err := rlp.Split(storageIter.Value)
				if err != nil {
					return fmt.Errorf("invalid storage RLP for key %x (account %x): %w", slotKey, addr, err)
				}
				if len(rest) != 0 || len(content) > 32 {
					return fmt.Errorf("malformed storage value for key %x (account %x)", slotKey, addr)
				}
				var padded [32]byte
				copy(padded[32-len(content):], content)
				if err := emit(bintrie.StorageSlotKey(addr, slotKey), padded); err != nil {
					return err
				}
				rawdb.WriteStorageSnapshot(flatBatch, accountHash, common.BytesToHash(storageIter.Key), common.CopyBytes(storageIter.Value))
				if err := slots.Add(append(accountHash.Bytes(), storageIter.Key...), storageIter.Value); err != nil {
					return err
				}
				stats.slots++

				if flatBatch.ValueSize() >= ethdb.IdealBatchSize {
					if err := flatBatch.Write(); err != nil {
						return err
					}
					flatBatch.Reset()
				}
			}
			if err := storageIter.Err; err != nil {
				return fmt.Errorf("storage iteration failed for %x: %w", addr, err)
			}
		}
		stats.accounts++
		if flatBatch.ValueSize() >= ethdb.IdealBatchSize {
			if err := flatBatch.Write(); err != nil {
				return err
			}
			flatBatch.Reset()
		}
		stats.report(false)
	}
	if err := accIter.Err; err != nil {
		return fmt.Errorf("account iteration failed: %w", err)
	}
	return flatBatch.Write()
}

// checkPreimage rejects a preimage of the wrong length or one that does not
// hash back to its key: the store has no integrity of its own, and a corrupt
// entry would convert into a wrong-stem tree the verifiers confirm.
func checkPreimage(preimage []byte, hash common.Hash, wantLen int) error {
	if len(preimage) != wantLen {
		return fmt.Errorf("corrupt preimage for %x: %d bytes, want %d", hash, len(preimage), wantLen)
	}
	if crypto.Keccak256Hash(preimage) != hash {
		return fmt.Errorf("corrupt preimage for %x: value hashes to %x", hash, crypto.Keccak256(preimage))
	}
	return nil
}

// emitAccountHeader derives an account's header and code-zone leaves. Shared
// by the scan and the flat-state verifier; the derivation rules stay pinned
// by the reference-vector parity tests.
func emitAccountHeader(chaindb ethdb.Database, addr common.Address, nonce uint64, balance *uint256.Int, codeHash common.Hash, seenCode map[common.Hash]struct{}, stats *conversionStats, emit func([]byte, [32]byte) error) error {
	var code []byte
	if codeHash != types.EmptyCodeHash {
		code = rawdb.ReadCode(chaindb, codeHash)
		if code == nil {
			return fmt.Errorf("missing code for hash %x (account %x)", codeHash, addr)
		}
	}
	basic, err := bintrie.EncodeBasicData(uint32(len(code)), nonce, balance)
	if err != nil {
		return fmt.Errorf("account %x: %w", addr, err)
	}
	if err := emit(bintrie.BasicDataKey(addr), basic); err != nil {
		return err
	}
	// A delegation replaces the code-hash leaf and has no code-zone leaves.
	if _, isDelegation := types.ParseDelegation(code); isDelegation {
		return emit(bintrie.DelegationKey(addr), [32]byte(bintrie.EncodeDelegation(code)))
	}
	if err := emit(bintrie.CodeHashKey(addr), codeHash); err != nil {
		return err
	}
	// Code once per distinct hash: the leaves are content-addressed.
	if len(code) > 0 {
		if _, seen := seenCode[codeHash]; !seen {
			seenCode[codeHash] = struct{}{}
			if stats != nil {
				stats.codes++
			}
			return emitCodeChunks(codeHash, code, emit)
		}
	}
	return nil
}

// emitCodeChunks derives one bytecode's code-zone leaves; all-zero chunks
// resolve to absence and are skipped.
func emitCodeChunks(codeHash common.Hash, code []byte, emit func([]byte, [32]byte) error) error {
	chunks := bintrie.ChunkifyCode(code)
	for i := 0; i < len(chunks)/32; i++ {
		var chunk [32]byte
		copy(chunk[:], chunks[32*i:32*(i+1)])
		if err := emit(bintrie.CodeChunkKey(codeHash, uint64(i)), chunk); err != nil {
			return err
		}
	}
	return nil
}

// hasBinaryTrieState reports whether anything sits in the binary tree
// namespace. Probed per key family: the bare prefix is shared with block
// bodies.
func hasBinaryTrieState(chaindb ethdb.Database) (bool, error) {
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))
	for _, family := range rawdb.PBTKeyFamilies {
		it := pbtdb.NewIterator(family, nil)
		found := it.Next()
		err := it.Error()
		it.Release()
		if err != nil {
			// Unreadable is not clean.
			return false, fmt.Errorf("failed to probe the binary tree namespace: %w", err)
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

// hasMerkleSource reports merkle state to convert from: no disposal marker (a
// begun disposal leaves nodes behind) and a root node on disk (an imported
// datadir never had one).
func hasMerkleSource(chaindb ethdb.Database, root common.Hash) bool {
	return !rawdb.ReadPBTMerkleDisposed(chaindb) &&
		(rawdb.HasAccountTrieNode(chaindb, nil) || rawdb.HasLegacyTrieNode(chaindb, root))
}

// wipeBinaryTrieState clears the binary tree state: the key families (the
// bare prefix is shared with block bodies), the PBT history freezers, and
// the journal file in triedbDir.
func wipeBinaryTrieState(chaindb ethdb.Database, triedbDir string) error {
	// Bookkeeping before state: any crash prefix leaves stale state the next
	// import detects, never a position that shadows a fresh anchor.
	if err := rawdb.WipeMigrationState(chaindb); err != nil {
		return err
	}
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))
	batch := pbtdb.NewBatch()
	wiped := 0
	for _, family := range rawdb.PBTKeyFamilies {
		it := pbtdb.NewIterator(family, nil)
		for it.Next() {
			if err := batch.Delete(common.CopyBytes(it.Key())); err != nil {
				it.Release()
				return err
			}
			wiped++
			if batch.ValueSize() >= ethdb.IdealBatchSize {
				if err := batch.Write(); err != nil {
					it.Release()
					return err
				}
				batch.Reset()
			}
		}
		err := it.Error()
		it.Release()
		if err != nil {
			return err
		}
	}
	if err := batch.Write(); err != nil {
		return err
	}
	if ancient, err := chaindb.AncientDatadir(); err == nil {
		for _, open := range []func(string, bool, bool) (ethdb.ResettableAncientStore, error){
			rawdb.NewStateFreezer,
			rawdb.NewTrienodeFreezer,
		} {
			store, err := open(ancient, true, false)
			if err != nil {
				return err
			}
			err = store.Reset()
			store.Close()
			if err != nil {
				return err
			}
		}
	}
	if triedbDir != "" {
		if err := os.Remove(filepath.Join(triedbDir, "pbt.journal")); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	log.Info("Wiped binary tree state", "records", wiped)
	return nil
}

// rawBinaryNodes reads tree records straight from the namespace: a path
// database would demand the attestation the verifiers gate, and history
// freezers a conversion never creates.
type rawBinaryNodes struct {
	pbtdb ethdb.Database
}

func (r rawBinaryNodes) NodeReader(common.Hash) (database.NodeReader, error) {
	return r, nil
}

// Node returns the record at path, unverified: the walk re-derives the root
// from the leaves, which subsumes per-node hash checks.
func (r rawBinaryNodes) Node(_ common.Hash, path []byte, _ common.Hash) ([]byte, error) {
	return rawdb.ReadAccountTrieNode(r.pbtdb, path), nil
}

// verifySourceRoot demands that the scanned records re-derive the state root
// they were read from, the converter's counterpart to the importer's anchor
// check.
func verifySourceRoot(accounts, slots *bintrie.RecordSorter, want common.Hash, start time.Time) error {
	acctStream, err := accounts.Sort()
	if err != nil {
		return err
	}
	slotStream, err := slots.Sort()
	if err != nil {
		return err
	}
	got, err := rederiveMerkleRoot(acctStream, slotStream, start)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("the scanned state re-derives merkle root %x, the source commits %x", got, want)
	}
	log.Info("Verified the scan against the source state root", "stateRoot", want)
	return nil
}

// verifyConvertedState refolds every persisted leaf and requires the rebuilt
// root to match. Gates the completion marker and --delete-source.
func verifyConvertedState(chaindb ethdb.Database, root common.Hash) error {
	tr, err := bintrie.NewBinaryTrie(root, rawBinaryNodes{pbtdb: rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))})
	if err != nil {
		return fmt.Errorf("cannot open the converted tree at %x: %w", root, err)
	}
	it, err := tr.NodeIterator(nil)
	if err != nil {
		return err
	}
	rebuild := bintrie.NewStackBuilder(nil)
	leaves := 0
	for it.Next(true) {
		if !it.Leaf() {
			continue
		}
		if err := rebuild.Add(common.CopyBytes(it.LeafKey()), common.CopyBytes(it.LeafBlob())); err != nil {
			return fmt.Errorf("leaf %x: %w", it.LeafKey(), err)
		}
		leaves++
	}
	if err := it.Error(); err != nil {
		return err
	}
	if got := rebuild.Finish(); got != root && !(leaves == 0 && root == types.EmptyBinaryHash) {
		return fmt.Errorf("disk leaves rebuild to %x, converted root is %x", got, root)
	}
	log.Info("Verified converted tree", "leaves", leaves, "root", root)
	return nil
}

// verifyFlatState re-derives the whole leaf set from the flat records and
// requires it to rebuild to the converted root. Flat state is the
// authoritative read path - a miss is an absent account - so the tree walk
// alone would bless a database that reads as empty.
func verifyFlatState(chaindb ethdb.Database, pbtdb ethdb.Database, srcTriedb *triedb.Database, root common.Hash, opts conversionOptions) error {
	sorter := bintrie.NewLeafSorter(opts.tmpDir, opts.sortBudget)
	defer sorter.Close()

	var (
		leaves   uint64
		seenCode = make(map[common.Hash]struct{})
		emit     = func(key []byte, value [32]byte) error {
			if value == ([32]byte{}) {
				return nil
			}
			leaves++
			return sorter.Add(key, value[:])
		}
	)
	acctIt := pbtdb.NewIterator(rawdb.SnapshotAccountPrefix, nil)
	for acctIt.Next() {
		accountHash := common.BytesToHash(acctIt.Key()[len(rawdb.SnapshotAccountPrefix):])
		addrBytes := srcTriedb.Preimage(accountHash)
		if addrBytes == nil {
			acctIt.Release()
			return fmt.Errorf("missing preimage for flat account %x", accountHash)
		}
		if err := checkPreimage(addrBytes, accountHash, common.AddressLength); err != nil {
			acctIt.Release()
			return err
		}
		addr := common.BytesToAddress(addrBytes)

		var slim types.SlimAccount
		if err := rlp.DecodeBytes(acctIt.Value(), &slim); err != nil {
			acctIt.Release()
			return fmt.Errorf("invalid flat account %x: %w", accountHash, err)
		}
		// Converted flat accounts never carry a storage root.
		if len(slim.Root) != 0 {
			acctIt.Release()
			return fmt.Errorf("flat account %x carries a storage root %x", accountHash, slim.Root)
		}
		codeHash := types.EmptyCodeHash
		if len(slim.CodeHash) != 0 {
			codeHash = common.BytesToHash(slim.CodeHash)
		}
		balance := slim.Balance
		if balance == nil {
			balance = new(uint256.Int)
		}
		if err := emitAccountHeader(chaindb, addr, slim.Nonce, balance, codeHash, seenCode, nil, emit); err != nil {
			acctIt.Release()
			return err
		}
	}
	if err := acctIt.Error(); err != nil {
		acctIt.Release()
		return err
	}
	acctIt.Release()

	// Slots are ordered by account hash: the address resolves once per account.
	var (
		slotIt   = pbtdb.NewIterator(rawdb.SnapshotStoragePrefix, nil)
		lastHash common.Hash
		lastAddr common.Address
	)
	for slotIt.Next() {
		key := slotIt.Key()
		if len(key) != len(rawdb.SnapshotStoragePrefix)+2*common.HashLength {
			slotIt.Release()
			return fmt.Errorf("malformed flat storage key %x", key)
		}
		accountHash := common.BytesToHash(key[len(rawdb.SnapshotStoragePrefix) : len(rawdb.SnapshotStoragePrefix)+common.HashLength])
		slotHash := common.BytesToHash(key[len(key)-common.HashLength:])

		if accountHash != lastHash || lastHash == (common.Hash{}) {
			addrBytes := srcTriedb.Preimage(accountHash)
			if addrBytes == nil {
				slotIt.Release()
				return fmt.Errorf("missing preimage for flat account %x", accountHash)
			}
			if err := checkPreimage(addrBytes, accountHash, common.AddressLength); err != nil {
				slotIt.Release()
				return err
			}
			lastHash, lastAddr = accountHash, common.BytesToAddress(addrBytes)
		}
		slotKey := srcTriedb.Preimage(slotHash)
		if slotKey == nil {
			slotIt.Release()
			return fmt.Errorf("missing preimage for flat storage key %x (account %x)", slotHash, lastAddr)
		}
		if err := checkPreimage(slotKey, slotHash, common.HashLength); err != nil {
			slotIt.Release()
			return err
		}
		_, content, rest, err := rlp.Split(slotIt.Value())
		if err != nil {
			slotIt.Release()
			return fmt.Errorf("invalid flat storage value for key %x (account %x): %w", slotHash, lastAddr, err)
		}
		if len(rest) != 0 || len(content) > 32 {
			slotIt.Release()
			return fmt.Errorf("malformed flat storage value for key %x (account %x)", slotHash, lastAddr)
		}
		var padded [32]byte
		copy(padded[32-len(content):], content)
		if err := emit(bintrie.StorageSlotKey(lastAddr, slotKey), padded); err != nil {
			slotIt.Release()
			return err
		}
	}
	if err := slotIt.Error(); err != nil {
		slotIt.Release()
		return err
	}
	slotIt.Release()

	stream, err := sorter.Sort()
	if err != nil {
		return err
	}
	rebuild := bintrie.NewStackBuilder(nil)
	for {
		key, value, err := stream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := rebuild.Add(key, value); err != nil {
			return err
		}
	}
	if got := rebuild.Finish(); got != root && !(leaves == 0 && root == types.EmptyBinaryHash) {
		return fmt.Errorf("flat state re-derives to %x, converted root is %x", got, root)
	}
	log.Info("Verified converted flat state", "leaves", leaves, "root", root)
	return nil
}

// wipePreimages deletes the preimage store, which a binary tree never
// consults. The whole key family goes, --vmdebug's SHA3 preimages included:
// they share the key space, and a 32-byte one looks like a slot key.
func wipePreimages(chaindb ethdb.Database) error {
	it := rawdb.NewKeyLengthIterator(chaindb.NewIterator(rawdb.PreimagePrefix, nil), len(rawdb.PreimagePrefix)+common.HashLength)
	defer it.Release()

	batch := chaindb.NewBatch()
	wiped := 0
	for it.Next() {
		if err := batch.Delete(common.CopyBytes(it.Key())); err != nil {
			return err
		}
		wiped++
		if batch.ValueSize() >= ethdb.IdealBatchSize {
			if err := batch.Write(); err != nil {
				return err
			}
			batch.Reset()
		}
	}
	if err := it.Error(); err != nil {
		return err
	}
	if err := batch.Write(); err != nil {
		return err
	}
	log.Warn("Deleted the preimage store", "records", wiped)
	return nil
}
