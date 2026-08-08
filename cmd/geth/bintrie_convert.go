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
	"os"
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
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/ethereum/go-ethereum/triedb"
	"github.com/ethereum/go-ethereum/triedb/database"
	"github.com/holiman/uint256"
	"github.com/urfave/cli/v2"
)

var (
	deleteSourceFlag = &cli.BoolFlag{
		Name:  "delete-source",
		Usage: "Delete MPT trie nodes after the conversion verifies",
	}
	memoryLimitFlag = &cli.Uint64Flag{
		Name:  "memory-limit",
		Usage: "Sort-buffer budget in MB before leaves spill to disk",
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
					tmpDirFlag,
					forceConvertFlag,
					snapshotOutFlag,
					preimagesOutFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth bintrie convert [--delete-source] [--memory-limit MB] [--tmpdir DIR]
                     [--force] [--snapshot-out FILE] [--preimages-out FILE]
                     [state-root]

Converts the Merkle Patricia Trie state into the EIP-8297 binary tree,
offline, following the EIP-8347 pipeline: every state leaf is derived,
sorted in tree-key order (spilling to disk past the memory budget), and the
tree is built bottom-up in one pass. The flat state the binary tree reads
through is written alongside, and the completion marker lands last, so an
interrupted conversion leaves a database that refuses to open rather than
one that silently reads as empty.

The optional state-root argument selects the state to convert; the head
block's root is used when omitted. The source database must hold the
account and storage key preimages, which only a node synced with
--cache.preimages has; every preimage consumed is verified against the
hash it was found under.

With --snapshot-out and --preimages-out, the conversion also emits the
EIP-8347 distribution artifacts: the byte-canonical PBT snapshot and the
address-sorted preimage file, along with the keccak digest of each, which
every correct producer reproduces for the same anchor state.

Flags:
  --delete-source    Delete MPT trie nodes once the conversion verifies
  --memory-limit     Sort-buffer budget in MB before spilling (default: 4096)
  --tmpdir           Spill-file directory (default: the OS temp dir)
  --force            Wipe existing binary tree state first
  --snapshot-out     Write the PBT snapshot artifact to this file
  --preimages-out    Write the preimage file to this file
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
	log.Info("Starting MPT to binary trie conversion", "root", root, "block", headBlock.NumberU64())

	// Check the namespace before MakeTrieDatabase does: its guard fatals on a
	// completed conversion without saying what to do about it, and misses the
	// debris of an interrupted one entirely.
	if ctx.Bool(forceConvertFlag.Name) {
		if err := wipeBinaryTrieState(chaindb); err != nil {
			return fmt.Errorf("failed to wipe binary tree state: %w", err)
		}
	} else if dirty, err := hasBinaryTrieState(chaindb); err != nil {
		return fmt.Errorf("failed to probe the binary tree namespace: %w", err)
	} else if dirty {
		return errors.New("database already holds binary tree state, complete or from an interrupted conversion; re-run with --force to wipe and reconvert")
	}
	srcTriedb := utils.MakeTrieDatabase(ctx, stack, chaindb, true, true, false)
	defer srcTriedb.Close()

	binRoot, err := convertState(chaindb, srcTriedb, root, conversionOptions{
		sortBudget:   int(ctx.Uint64(memoryLimitFlag.Name)) * 1024 * 1024,
		tmpDir:       ctx.String(tmpDirFlag.Name),
		snapshotPath: ctx.String(snapshotOutFlag.Name),
		preimagePath: ctx.String(preimagesOutFlag.Name),
	})
	if err != nil {
		return err
	}
	log.Info("Conversion complete", "binaryRoot", binRoot)

	if ctx.Bool(deleteSourceFlag.Name) {
		log.Info("Deleting source MPT data")
		if err := deleteMPTData(chaindb, srcTriedb, root); err != nil {
			return fmt.Errorf("MPT deletion failed: %w", err)
		}
		log.Info("Source MPT data deleted")
	}
	return nil
}

// conversionOptions carries the tunables of a conversion run.
type conversionOptions struct {
	sortBudget   int    // bytes of buffered records before a sort spills
	tmpDir       string // spill directory; empty means the OS temp dir
	snapshotPath string // PBT snapshot artifact destination; empty writes none
	preimagePath string // preimage file destination; empty writes none
}

// conversionStats tracks progress for the periodic report.
type conversionStats struct {
	accounts   uint64
	slots      uint64
	codes      uint64
	leaves     uint64
	start      time.Time
	lastReport time.Time
}

func (s *conversionStats) report(force bool) {
	if !force && time.Since(s.lastReport) < 8*time.Second {
		return
	}
	s.lastReport = time.Now()
	log.Info("Converting state", "accounts", s.accounts, "slots", s.slots,
		"codes", s.codes, "leaves", s.leaves, "elapsed", common.PrettyDuration(time.Since(s.start)))
}

// convertState converts the MPT state at root into the binary tree namespace
// of chaindb and returns the binary root. It is the conversion engine behind
// the CLI command, factored so tests drive the whole pipeline without a node.
//
// The shape is EIP-8347's: one scan of the source derives every tree leaf,
// the leaves are sorted in tree-key order - plain byte order, with spills to
// disk past the memory budget - and the tree is built bottom-up in a single
// pass, its records streaming straight into the database. No intermediate
// tree ever exists, so there is nothing to commit-and-reload and the memory
// high-water mark is the sort buffer.
//
// Flat state is written during the scan: the binary tree reads accounts and
// slots through the flat store first, and treats a miss as authoritative
// absence, so a conversion writing trie nodes alone would hash to the right
// root and read as empty. The accounts are slimmed with the storage root
// normalized to the empty root - the binary tree has no per-account storage
// roots, and a replaying node records exactly that - so converted and
// replayed flat state stay byte-identical.
//
// Nothing becomes openable or publishable unverified. Both stores are
// checked from what actually landed on disk - the tree by walking its
// persisted records back into a root, the flat state by re-deriving the
// whole leaf set from it - before the artifacts are finalized, and the
// flat-state attestation is written last of all: opening a binary tree
// database checks it, so a conversion that dies or fails verification at
// any point leaves a namespace that refuses to open ("resync required")
// rather than an attested half-state that reads as empty. A re-run after
// such a death needs --force, which wipes the namespace.
func convertState(chaindb ethdb.Database, srcTriedb *triedb.Database, root common.Hash, opts conversionOptions) (common.Hash, error) {
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))

	// Refuse a namespace that is not virgin: either a completed conversion
	// (attested) or the debris of an interrupted one. Both need --force.
	if dirty, err := hasBinaryTrieState(chaindb); err != nil {
		return common.Hash{}, err
	} else if dirty {
		return common.Hash{}, errors.New("binary tree namespace is not empty; re-run with --force to wipe it")
	}
	stats := &conversionStats{start: time.Now(), lastReport: time.Now()}

	sorter := bintrie.NewLeafSorter(opts.tmpDir, opts.sortBudget)
	defer sorter.Close()

	// The distribution artifacts ride the pipeline: the snapshot writer taps
	// the sorted leaf stream, the preimage file the scan. Both are finalized
	// only after the converted state verifies, and before the database's own
	// completion markers, so neither a failed conversion nor a failed
	// verification leaves a publishable file or an openable namespace.
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
		preimages = newPreimageFile(opts.tmpDir, opts.sortBudget)
		defer preimages.close()
	}
	// Phase 1: scan the merkle state, deriving tree leaves into the sorter
	// and streaming flat state to disk as it goes.
	if err := deriveLeaves(chaindb, pbtdb, srcTriedb, root, sorter, preimages, stats); err != nil {
		return common.Hash{}, err
	}
	stats.report(true)

	// Phase 2+3: sort and build. The builder emits each database record
	// exactly once, children before parents, and the batch flushes on size.
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
	// Finish returns the zero hash for an empty stream, which is exactly
	// types.EmptyBinaryHash.
	binRoot := builder.Finish()
	if builderErr != nil {
		return common.Hash{}, builderErr
	}
	if err := batch.Write(); err != nil {
		return common.Hash{}, err
	}
	log.Info("Built binary tree", "nodes", written, "leaves", stats.leaves)

	// Verify before anything becomes publishable or openable: the tree from
	// its persisted records, the flat state by re-deriving the full leaf set
	// from it. A failure leaves a namespace with no completion marker.
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

	// Finalize. The root marker is what a reopening path database reads the
	// disk-layer root from; no state id is written, so the converted tree is
	// the base of an empty history and live operation numbers layers from 1.
	// The attestation comes last: it is the completion marker.
	rawdb.WriteSnapshotRoot(pbtdb, binRoot)
	rawdb.WritePBTFlatState(pbtdb)
	return binRoot, nil
}

// deriveLeaves walks the merkle state at root and derives every binary tree
// leaf into the sorter, writing flat state alongside. Code is emitted once
// per distinct code hash - the leaves are content-addressed and shared, and
// the sorter treats a duplicate key as corruption.
func deriveLeaves(chaindb ethdb.Database, pbtdb ethdb.Database, srcTriedb *triedb.Database, root common.Hash, sorter *bintrie.RecordSorter, preimages *preimageFile, stats *conversionStats) error {
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
		// The state layer resolves 32 zero bytes to absence: such a leaf is
		// not written, and reads recover the zero it stood for.
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

		// Flat state: the slim account, with the storage root normalized to
		// the empty root the way every replaying writer records it. The
		// source's merkle storage root means nothing in this namespace.
		accountHash := common.BytesToHash(accIter.Key)
		slim := acc
		slim.Root = types.EmptyRootHash
		rawdb.WriteAccountSnapshot(flatBatch, accountHash, types.SlimAccountRLP(slim))

		// Storage, from both of its homes.
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
				_, content, _, err := rlp.Split(storageIter.Value)
				if err != nil {
					return fmt.Errorf("invalid storage RLP for key %x (account %x): %w", slotKey, addr, err)
				}
				var padded [32]byte
				copy(padded[32-len(content):], content)
				if err := emit(bintrie.StorageSlotKey(addr, slotKey), padded); err != nil {
					return err
				}
				rawdb.WriteStorageSnapshot(flatBatch, accountHash, common.BytesToHash(storageIter.Key), common.CopyBytes(storageIter.Value))
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

// checkPreimage guards a preimage lookup: the stored value must be the
// stated length and must hash back to the key it was found under. The store
// is a bare hash-to-value table with no integrity of its own, so a corrupt
// entry would otherwise become a wrong-stem tree that every later check -
// the verifiers included, since they re-derive from the same value -
// happily confirms. One short keccak per leaf, noise against the blake3
// derivations each leaf already pays for.
func checkPreimage(preimage []byte, hash common.Hash, wantLen int) error {
	if len(preimage) != wantLen {
		return fmt.Errorf("corrupt preimage for %x: %d bytes, want %d", hash, len(preimage), wantLen)
	}
	if crypto.Keccak256Hash(preimage) != hash {
		return fmt.Errorf("corrupt preimage for %x: value hashes to %x", hash, crypto.Keccak256(preimage))
	}
	return nil
}

// emitAccountHeader derives the header-stem and code-zone leaves of one
// account: the basic data, then either the delegation indicator - exclusive
// with the code hash, per the account's bytecode being an EIP-7702
// designator - or the code hash and, once per distinct hash, the code
// chunks. It is shared between the conversion scan and the flat-state
// verifier, so both derive from one set of rules; what stays independently
// pinned is the rules themselves, by the reference-vector parity tests.
// Storage is the caller's business: it lives in other stems, and the two
// callers walk it from different stores.
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
	// A delegated account holds its designator in a header leaf in place of
	// the code hash - the two are exclusive - and produces no code-zone
	// leaves at all.
	if _, isDelegation := types.ParseDelegation(code); isDelegation {
		return emit(bintrie.DelegationKey(addr), [32]byte(bintrie.EncodeDelegation(code)))
	}
	if err := emit(bintrie.CodeHashKey(addr), codeHash); err != nil {
		return err
	}
	// Code, once per distinct hash: the leaves are content-addressed, so
	// every holder of this bytecode shares them.
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

// emitCodeChunks derives the code-zone leaves of one bytecode. All-zero
// chunks - 31 zero code bytes carrying no push data - are skipped: the state
// layer resolves them to absence, and code_size delimits the code rather
// than chunk presence.
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

// hasBinaryTrieState reports whether anything at all sits in the binary tree
// namespace: a completed conversion, a live tree, or the debris of a run that
// died. Which keys the debris consists of depends on where the writer
// stopped, so every key family is probed - and it has to be by family rather
// than a bare scan of the namespace prefix, which is shared with block bodies.
func hasBinaryTrieState(chaindb ethdb.Database) (bool, error) {
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))
	for _, family := range rawdb.PBTKeyFamilies {
		it := pbtdb.NewIterator(family, nil)
		found := it.Next()
		err := it.Error()
		it.Release()
		if err != nil {
			// An unreadable namespace is not a clean one.
			return false, fmt.Errorf("failed to probe the binary tree namespace: %w", err)
		}
		if found {
			return true, nil
		}
	}
	return false, nil
}

// wipeBinaryTrieState removes everything under the binary tree namespace so a
// conversion can start over. Like the probe, it works family by family: the
// namespace prefix is shared with block bodies, which a bare prefix sweep
// would delete along with the tree.
func wipeBinaryTrieState(chaindb ethdb.Database) error {
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
	log.Info("Wiped binary tree state", "records", wiped)
	return nil
}

// rawBinaryNodes serves the converted tree's records straight from the
// database namespace. The verifiers read through it rather than through a
// path database: opening one requires the flat-state attestation that the
// verification gates, and a read-only open would demand history freezers
// that a conversion never creates.
type rawBinaryNodes struct {
	pbtdb ethdb.Database
}

func (r rawBinaryNodes) NodeReader(common.Hash) (database.NodeReader, error) {
	return r, nil
}

// Node returns the record at path; a missing record returns nil, which the
// trie reader turns into a missing-node error. The hash goes unchecked: the
// verification walk re-derives the root from the leaves, which subsumes
// per-node hash equality.
func (r rawBinaryNodes) Node(_ common.Hash, path []byte, _ common.Hash) ([]byte, error) {
	return rawdb.ReadAccountTrieNode(r.pbtdb, path), nil
}

// verifyConvertedState re-derives the converted root from what is actually
// on disk: every leaf is walked out of the persisted tree and folded through
// an independent bottom-up build, which must reproduce the root. Together
// with the flat-state verification this is what stands between a conversion
// and its completion marker - and what gates --delete-source: the source is
// only dropped once the converted bytes alone can stand in for it.
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

// verifyFlatState proves the flat store can stand in for the tree: the
// entire leaf set is re-derived from the flat records - accounts and slots
// resolved back through the preimage store, code re-read per distinct hash -
// and must rebuild to the converted root. The flat store is the authoritative
// read path (a miss there is an absent account, with the trie never
// consulted), so trie-side verification alone would bless a database that
// reads as empty; this is the check that makes the attestation honest.
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
	// The flat accounts, walked in hashed order and resolved back to their
	// addresses through the preimage store.
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
		// The binary tree has no per-account storage roots; a flat record
		// carrying one was not written by this conversion.
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

	// The flat slots. The iteration is ordered by account hash, so the
	// address preimage resolves once per account.
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
		_, content, _, err := rlp.Split(slotIt.Value())
		if err != nil {
			slotIt.Release()
			return fmt.Errorf("invalid flat storage value for key %x (account %x): %w", slotHash, lastAddr, err)
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

	// The re-derived leaves must rebuild to the converted root.
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
