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
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sync"
	"sync/atomic"
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
	"github.com/ethereum/go-ethereum/triedb/pathdb"
	"github.com/holiman/uint256"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

var (
	memoryLimitFlag = &cli.Uint64Flag{
		Name:  "memory-limit",
		Usage: "Total sort-buffer budget in MB before records spill to disk",
		Value: 4096,
	}
	tmpDirFlag = &cli.StringFlag{
		Name:  "tmpdir",
		Usage: "Directory for scratch files, the sort spills and a copy of every tree node; needs disk, not tmpfs (default: the OS temp dir)",
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
		Usage: "File to write the EIP-8347 preimage file to",
	}
	preimagesInFlag = &cli.StringFlag{
		Name:  "preimages",
		Usage: "EIP-8347 preimage file to take the state's keys from instead of the node's preimage store; it must hold exactly the state's keys",
	}
	dropPreimagesFlag = &cli.BoolFlag{
		Name:  "drop-preimages",
		Usage: "Delete the preimage store after the conversion verifies",
	}
	workersFlag = &cli.IntFlag{
		Name:  "workers",
		Usage: "Parallel workers: convert splits its scan among them by account hash, import its candidate key derivation; --memory-limit is split among them",
		Value: 4,
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
					memoryLimitFlag,
					tmpDirFlag,
					forceConvertFlag,
					snapshotOutFlag,
					preimagesOutFlag,
					preimagesInFlag,
					dropPreimagesFlag,
					workersFlag,
				}, utils.NetworkFlags, utils.DatabaseFlags),
				Description: `
geth bintrie convert [flags] [state-root]

Converts the MPT state into the EIP-8297 binary tree offline, following the
EIP-8347 pipeline, and verifies the result before marking it complete.
Defaults to the head root. The addresses and slot keys behind the state's
hashes come from the node's preimage store (--cache.preimages) or from an
EIP-8347 preimage file of the same state (--preimages). The scan runs on
--workers workers. --snapshot-out and --preimages-out emit the
EIP-8347 distribution artifacts. --drop-preimages deletes the preimage store
afterwards, which also rules out a re-conversion from it.
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
	// Past the fork the head commits the binary tree: no merkle state exists
	// at its root, and the namespace a conversion wipes is the live tree.
	if past, err := headCommitsBinaryTree(chaindb, headBlock); err != nil {
		return err
	} else if past {
		return errors.New("the head block commits the binary tree; nothing to convert")
	}
	var (
		root   common.Hash
		anchor *types.Header
		err    error
	)
	if ctx.NArg() == 1 {
		root, err = parseRoot(ctx.Args().First())
		if err != nil {
			return fmt.Errorf("invalid state root: %w", err)
		}
		// Only the head root is anchored; any other root converts into
		// artifacts, not a bootable namespace.
		if root == headBlock.Root() {
			anchor = headBlock.Header()
		}
	} else {
		root = headBlock.Root()
		anchor = headBlock.Header()
	}
	// Without preimages the scan fails, and only after --force has wiped. An
	// empty state has nothing to resolve.
	preimageIn := ctx.String(preimagesInFlag.Name)
	if preimageIn != "" {
		in, err := os.Stat(preimageIn)
		if err != nil {
			return fmt.Errorf("--preimages: %w", err)
		}
		// The outputs are created, and truncated, before the input is read.
		for _, out := range []string{ctx.String(snapshotOutFlag.Name), ctx.String(preimagesOutFlag.Name)} {
			if fi, err := os.Stat(out); err == nil && os.SameFile(in, fi) {
				return errors.New("--preimages must not name an output file")
			}
		}
	} else if root != types.EmptyRootHash && !hasPreimages(chaindb) {
		return errors.New("no preimages to convert from; the source must sync with --cache.preimages, or pass an EIP-8347 preimage file with --preimages")
	}
	log.Info("Starting MPT to binary trie conversion", "root", root, "block", headBlock.NumberU64())

	snapshotPath, preimagePath := ctx.String(snapshotOutFlag.Name), ctx.String(preimagesOutFlag.Name)
	if snapshotPath != "" && snapshotPath == preimagePath {
		return errors.New("--snapshot-out and --preimages-out must name different files")
	}
	if (snapshotPath == "") != (preimagePath == "") {
		log.Warn("Emitting one distribution artifact without the other; EIP-8347 distributes the snapshot and the preimage file together")
	}
	workers := ctx.Int(workersFlag.Name)
	if workers < 1 {
		return errors.New("--workers must be at least 1")
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
	// Closed once: --drop-preimages closes it early, before its wipe.
	closeSource := sync.OnceValue(srcTriedb.Close)
	defer closeSource()

	binRoot, err := convertState(chaindb, srcTriedb, root, conversionOptions{
		sortBudget:   int(budgetMB << 20),
		tmpDir:       ctx.String(tmpDirFlag.Name),
		snapshotPath: snapshotPath,
		preimagePath: preimagePath,
		anchor:       anchor,
		workers:      workers,
		preimageIn:   preimageIn,
	})
	if err != nil {
		return err
	}
	log.Info("Conversion complete", "binaryRoot", binRoot)

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
	sortBudget   int           // total bytes of buffered records before the sorts spill
	tmpDir       string        // spill directory; empty means the OS temp dir
	snapshotPath string        // PBT snapshot artifact destination; empty writes none
	preimagePath string        // preimage file destination; empty writes none
	anchor       *types.Header // block committing root; nil leaves the namespace unbootable
	workers      int           // parallel workers; below 1 means one
	preimageIn   string        // EIP-8347 preimage file to resolve keys from; empty reads the node's store
}

// conversionStats tracks progress for the periodic report. The message names
// the work, since the converter and the importer share the counters. Every
// method is safe for concurrent use: the converter's scan workers share it.
type conversionStats struct {
	what                           string
	accounts, slots, codes, leaves atomic.Uint64
	start                          time.Time
	lastReport                     atomic.Int64 // unix nanoseconds
}

func newStats(what string) *conversionStats {
	s := &conversionStats{what: what, start: time.Now()}
	s.lastReport.Store(s.start.UnixNano())
	return s
}

// report logs progress, unforced at most every 8 seconds across callers.
func (s *conversionStats) report(force bool) {
	now, last := time.Now().UnixNano(), s.lastReport.Load()
	if !force && (now-last < int64(8*time.Second) || !s.lastReport.CompareAndSwap(last, now)) {
		return
	}
	s.lastReport.Store(now)
	log.Info(s.what, "accounts", s.accounts.Load(), "slots", s.slots.Load(),
		"codes", s.codes.Load(), "leaves", s.leaves.Load(), "elapsed", common.PrettyDuration(time.Since(s.start)))
}

// convertState converts the MPT state at root into the binary tree namespace
// per EIP-8347: one scan, split by account hash across workers, derives
// every leaf, an external sort orders the leaves in tree-key order, and the
// tree builds bottom-up in one pass. The scan must re-derive the source
// state root, and both stores must read back exactly what was written. The
// attestation lands last, so an interrupted run refuses to open.
func convertState(chaindb ethdb.Database, srcTriedb *triedb.Database, root common.Hash, opts conversionOptions) (common.Hash, error) {
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))

	// A non-virgin namespace, completed or debris, needs --force.
	if dirty, err := hasBinaryTrieState(chaindb); err != nil {
		return common.Hash{}, err
	} else if dirty {
		return common.Hash{}, errors.New("binary tree namespace is not empty; re-run with --force to wipe it")
	}
	stats := newStats("Converting state")
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
		pf, err := newPreimageFile(opts.preimagePath)
		if err != nil {
			return common.Hash{}, err
		}
		preimages = pf
		defer func() {
			if preimages != nil {
				preimages.abort()
			}
		}()
	}
	// Phase 1: scan, deriving leaves. Each worker scans its own range of
	// account hashes into sorters of its own, resolving keys from the node's
	// preimage store or from its own piece of a preimage file.
	workers := max(opts.workers, 1)
	ranges := splitHashSpace(workers)
	keys := make([]keySource, workers)
	for i := range keys {
		keys[i] = storeKeys{srcTriedb}
	}
	if opts.preimageIn != "" {
		f, err := os.Open(opts.preimageIn)
		if err != nil {
			return common.Hash{}, err
		}
		defer f.Close()
		info, err := f.Stat()
		if err != nil {
			return common.Hash{}, err
		}
		// The file's pieces set the ranges, so the work splits by data volume;
		// a small file may make fewer pieces than workers.
		offsets, starts, err := splitPreimageFile(f, info.Size(), workers)
		if err != nil {
			return common.Hash{}, err
		}
		workers = len(offsets)
		ranges, keys = make([]hashRange, workers), make([]keySource, workers)
		for i := range offsets {
			end := info.Size()
			if i+1 < workers {
				end = offsets[i+1]
				ranges[i].end = starts[i+1]
			}
			ranges[i].start = starts[i]
			keys[i] = &fileKeys{r: preimagePiece(f, offsets[i], end-offsets[i])}
		}
	}
	// Every worker's three sorters are alive at once, and a sealed one keeps
	// its buffer while its stream drains, so the budget is split rather than
	// reused.
	share := splitBudget(opts.sortBudget, 3*workers)
	parts := make([]scanPart, workers)
	for i, r := range ranges {
		parts[i] = scanPart{
			hashRange: r,
			leaves:    bintrie.NewLeafSorter(opts.tmpDir, share),
			// The re-derivation limbs, keyed by account hash and by account‖slot.
			accounts: bintrie.NewRecordSorter(opts.tmpDir, share, nil),
			slots:    bintrie.NewRecordSorter(opts.tmpDir, share, nil),
		}
		defer parts[i].leaves.Close()
		defer parts[i].accounts.Close()
		defer parts[i].slots.Close()
	}
	// Every worker writes its range of the preimage file: the first straight
	// into the artifact, the others into fragments joined after it.
	writers := make([]*preimageFile, workers)
	writers[0] = preimages
	if preimages != nil {
		for i := 1; i < workers; i++ {
			frag, err := newPreimageFragment(opts.tmpDir)
			if err != nil {
				return common.Hash{}, err
			}
			defer frag.abort()
			writers[i] = frag
		}
	}
	src := leafSource{srcTriedb, root, srcTriedb.Scheme() == rawdb.PathScheme && srcTriedb.SnapshotCompleted()}
	codes := newCodeSet()
	g, gctx := errgroup.WithContext(context.Background())
	for i := range parts {
		g.Go(func() error {
			return deriveLeaves(gctx, chaindb, src, &parts[i], keys[i], codes, writers[i], stats)
		})
	}
	if err := g.Wait(); err != nil {
		return common.Hash{}, err
	}
	if preimages != nil {
		if err := preimages.join(writers[1:]); err != nil {
			return common.Hash{}, err
		}
	}
	stats.report(true)

	// One merge serves every worker's records.
	sorter, accounts, slots := parts[0].leaves, parts[0].accounts, parts[0].slots
	for _, p := range parts[1:] {
		if err := errors.Join(sorter.Absorb(p.leaves), accounts.Absorb(p.accounts), slots.Absorb(p.slots)); err != nil {
			return common.Hash{}, err
		}
	}

	// Check 1, before the build: everything written is fed by this same
	// record set, so only this check ties the output to the source state.
	// The fold writes the flat state as it goes, digesting what it writes.
	written := newWrittenState()
	if err := verifySourceRoot(accounts, slots, root, pbtdb.NewBatch(), written, opts.tmpDir, stats.start); err != nil {
		return common.Hash{}, err
	}
	accounts.Close()
	slots.Close()

	// Phase 2+3: sort and build. Tree nodes are staged per depth and written
	// once the tree is built, in key order (see nodeStager).
	stream, err := sorter.Sort()
	if err != nil {
		return common.Hash{}, err
	}
	nodes, err := newNodeStager(opts.tmpDir)
	if err != nil {
		return common.Hash{}, err
	}
	defer nodes.close()
	var builderErr error
	builder := bintrie.NewStackBuilder(func(path []byte, hash common.Hash, blob []byte) {
		if builderErr == nil {
			builderErr = nodes.add(path, blob)
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
	if err := writeStagedNodes(pbtdb, nodes, written.nodes); err != nil {
		return common.Hash{}, err
	}
	log.Info("Built binary tree", "nodes", written.nodes.count, "leaves", stats.leaves.Load())

	// Both stores must hold exactly what was written before anything becomes
	// publishable or openable.
	if err := verifyWritten(pbtdb, written); err != nil {
		return common.Hash{}, err
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
		digest, err := preimages.finalize()
		if err != nil {
			return common.Hash{}, err
		}
		log.Info("Wrote preimage file", "path", opts.preimagePath,
			"accounts", preimages.accounts, "digest", digest)
		preimages = nil // finalized; disarm the abort
	}

	// No state id: the tree bases an empty history and live commits number
	// from 1. The attestation comes last.
	rawdb.WriteSnapshotRoot(pbtdb, binRoot)
	if opts.anchor != nil {
		rawdb.WritePBTAnchor(pbtdb, opts.anchor.Number.Uint64(), opts.anchor.Hash())
	} else {
		log.Warn("Converted without an anchor: the namespace is not bootable; reconvert at the head root with --force before starting the node")
	}
	rawdb.WritePBTFlatState(pbtdb)
	return binRoot, nil
}

// leafSource yields the source state's accounts from a seek hash on, then
// each account's storage, in keccak order, as RLP the way the state stores
// it. flat reads pathdb's flat state, which its iterators allow only once
// complete; otherwise the trie nodes are walked: a hash-scheme source, or a
// pathdb still syncing or generating. Either way verifySourceRoot
// authenticates the scanned leaves against root.
type leafSource struct {
	db   *triedb.Database
	root common.Hash
	flat bool // flat accounts are slim RLP, trie leaves full RLP
}

func (s leafSource) accounts(seek common.Hash) (pathdb.AccountIterator, error) {
	if s.flat {
		return s.db.AccountIterator(s.root, seek)
	}
	return s.walk(trie.StateTrieID(s.root), seek[:])
}

func (s leafSource) storage(accountHash, storageRoot common.Hash) (pathdb.StorageIterator, error) {
	if s.flat {
		return s.db.StorageIterator(s.root, accountHash, common.Hash{})
	}
	return s.walk(trie.StorageTrieID(s.root, accountHash, storageRoot), nil)
}

func (s leafSource) walk(id *trie.ID, start []byte) (*trieLeafIterator, error) {
	tr, err := trie.New(id, s.db)
	if err != nil {
		return nil, err
	}
	it, err := tr.NodeIterator(start)
	if err != nil {
		return nil, err
	}
	return &trieLeafIterator{trie.NewIterator(it)}, nil
}

// trieLeafIterator adapts a trie leaf iterator to both flat iterator shapes.
type trieLeafIterator struct{ it *trie.Iterator }

func (t *trieLeafIterator) Next() bool        { return t.it.Next() }
func (t *trieLeafIterator) Error() error      { return t.it.Err }
func (t *trieLeafIterator) Hash() common.Hash { return common.BytesToHash(t.it.Key) }
func (t *trieLeafIterator) Release()          {}
func (t *trieLeafIterator) Account() []byte   { return t.it.Value }
func (t *trieLeafIterator) Slot() []byte      { return t.it.Value }

// hashRange is one worker's share of the account hashes, [start, end); the
// zero end stands for the top of the hash space.
type hashRange struct {
	start, end common.Hash
}

// past reports whether hash lies beyond the range, in the next one.
func (r hashRange) past(hash common.Hash) bool {
	return r.end != (common.Hash{}) && hash.Cmp(r.end) >= 0
}

// splitHashSpace splits the account hashes into n ascending ranges of equal
// width. Hashes are uniform, so the ranges hold about equal state, bar one
// outsized contract's storage.
func splitHashSpace(n int) []hashRange {
	width := new(uint256.Int).Div(new(uint256.Int).SetAllOne(), uint256.NewInt(uint64(n)))
	ranges := make([]hashRange, n)
	for i := range ranges {
		ranges[i].start = new(uint256.Int).Mul(width, uint256.NewInt(uint64(i))).Bytes32()
		if i > 0 {
			ranges[i-1].end = ranges[i].start
		}
	}
	return ranges
}

// splitBudget divides a sort budget among n sorters. A positive budget stays
// positive: a non-positive one never spills.
func splitBudget(budget, n int) int {
	if budget <= 0 {
		return budget
	}
	return max(budget/n, 1)
}

// scanPart is one scan worker's share: its range of account hashes and the
// sorters it derives into.
type scanPart struct {
	hashRange
	leaves, accounts, slots *bintrie.RecordSorter
}

// codeInfo is what an account header needs of its code: the size, and the
// designator when the code is an EIP-7702 delegation.
type codeInfo struct {
	size       uint32
	delegation []byte
}

// codeSet caches each distinct code's info across the scan workers, so the
// store is read once per code, and lets one worker emit its chunks: a chunk
// emitted twice is a duplicate key.
type codeSet struct {
	mu   sync.Mutex
	seen map[common.Hash]codeInfo
}

func newCodeSet() *codeSet {
	return &codeSet{seen: make(map[common.Hash]codeInfo)}
}

// lookup returns the info of a code already met.
func (s *codeSet) lookup(hash common.Hash) (codeInfo, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, ok := s.seen[hash]
	return info, ok
}

// claim records hash and its info and reports whether it was new: of all
// callers claiming one hash, exactly one wins.
func (s *codeSet) claim(hash common.Hash, info codeInfo) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.seen[hash]; ok {
		return false
	}
	s.seen[hash] = info
	return true
}

// keySource resolves one scan worker's hashed keys to the addresses and slot
// keys behind them, in scan order.
type keySource interface {
	account(hash common.Hash) (common.Address, error)
	slot(addr common.Address, hash common.Hash) (common.Hash, error)
	// done rejects anything the source holds for addr beyond the slots the
	// scan resolved; close, anything left once the worker's range is done.
	done(addr common.Address) error
	close() error
}

// storeKeys looks keys up in the node's preimage store, which may hold more
// than the state, as EIP-8347 allows.
type storeKeys struct{ db *triedb.Database }

func (s storeKeys) account(hash common.Hash) (common.Address, error) {
	preimage := s.db.Preimage(hash)
	if preimage == nil {
		return common.Address{}, fmt.Errorf("missing preimage for account hash %x (the source node must have synced with --cache.preimages, or pass --preimages)", hash)
	}
	if err := checkPreimage(preimage, hash, common.AddressLength); err != nil {
		return common.Address{}, err
	}
	return common.BytesToAddress(preimage), nil
}

func (s storeKeys) slot(addr common.Address, hash common.Hash) (common.Hash, error) {
	preimage := s.db.Preimage(hash)
	if preimage == nil {
		return common.Hash{}, fmt.Errorf("missing preimage for storage key %x (account %x)", hash, addr)
	}
	if err := checkPreimage(preimage, hash, common.HashLength); err != nil {
		return common.Hash{}, err
	}
	return common.BytesToHash(preimage), nil
}

func (storeKeys) done(common.Address) error { return nil }
func (storeKeys) close() error              { return nil }

// fileKeys merges one piece of an EIP-8347 preimage file against the
// worker's scan range, as the EIP's converter step 2 does: both run in keccak
// order, so each record must match the next account and each slot key the
// next slot. The reader hashes every key, which proves the preimages.
type fileKeys struct {
	r     *preimageReader
	slots []common.Hash
	next  int
}

func (f *fileKeys) account(hash common.Hash) (common.Address, error) {
	addr, addrHash, slots, err := f.r.next()
	switch {
	case err == io.EOF:
		return common.Address{}, fmt.Errorf("missing preimage for account hash %x: the preimage file holds no record for it", hash)
	case err != nil:
		return common.Address{}, err
	case addrHash == hash:
		f.slots, f.next = slots, 0
		return addr, nil
	case bytes.Compare(addrHash[:], hash[:]) < 0:
		return common.Address{}, fmt.Errorf("preimage file names address %x, which the state does not hold", addr)
	default:
		return common.Address{}, fmt.Errorf("missing preimage for account hash %x: the preimage file holds no record for it", hash)
	}
}

func (f *fileKeys) slot(addr common.Address, hash common.Hash) (common.Hash, error) {
	if f.next < len(f.slots) && f.r.slotHashes[f.next] == hash {
		f.next++
		return f.slots[f.next-1], nil
	}
	if f.next < len(f.slots) && bytes.Compare(f.r.slotHashes[f.next][:], hash[:]) < 0 {
		return common.Hash{}, fmt.Errorf("preimage file names slot %x of %x, which the state does not hold", f.slots[f.next], addr)
	}
	return common.Hash{}, fmt.Errorf("missing preimage for storage key %x (account %x)", hash, addr)
}

func (f *fileKeys) done(addr common.Address) error {
	if f.next < len(f.slots) {
		return fmt.Errorf("preimage file names slot %x of %x, which the state does not hold", f.slots[f.next], addr)
	}
	return nil
}

func (f *fileKeys) close() error {
	addr, _, _, err := f.r.next()
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("preimage file names address %x, which the state does not hold", addr)
}

// deriveLeaves scans part's range of the source state at root, deriving
// every tree leaf into the part's sorter and recording the merkle
// re-derivation limbs the source-root check folds back up. It gives up once
// ctx is cancelled, by another worker failing.
func deriveLeaves(ctx context.Context, chaindb ethdb.Database, src leafSource, part *scanPart, keys keySource, codes *codeSet, preimages *preimageFile, stats *conversionStats) error {
	// ponytail: a flat scan holds one pebble read snapshot per worker
	// throughout, pinning a memtable and every table compacted meanwhile
	// (the old flat state on a --force reconvert). Reopen from the last hash
	// every 30s, as pathdb's index pruner does, if producer RAM or disk gets
	// tight.
	accIter, err := src.accounts(part.start)
	if err != nil {
		return fmt.Errorf("failed to open source accounts: %w", err)
	}
	defer accIter.Release()
	var (
		// RecordSorter.Add copies its inputs, so one key buffer serves every slot.
		slotKeyBuf [2 * common.HashLength]byte
		// The current account's counts, published to stats with it.
		leaves, slots uint64
	)
	emit := func(key []byte, value [32]byte) error {
		// Zero values resolve to absence and are never written.
		if value == ([32]byte{}) {
			return nil
		}
		leaves++
		return part.leaves.Add(key, value[:])
	}
	// scanStorage derives one account's slots; the deferred release keeps
	// the iterator from leaking on any error path.
	scanStorage := func(accountHash common.Hash, addr common.Address, storageRoot common.Hash) error {
		storageIter, err := src.storage(accountHash, storageRoot)
		if err != nil {
			return fmt.Errorf("failed to open storage for %x: %w", addr, err)
		}
		defer storageIter.Release()
		for storageIter.Next() {
			slotHash := storageIter.Hash()
			slotKey, err := keys.slot(addr, slotHash)
			if err != nil {
				return err
			}
			if preimages != nil {
				if err := preimages.addSlot(slotKey, slotHash); err != nil {
					return err
				}
			}
			slotVal := storageIter.Slot()
			_, content, rest, err := rlp.Split(slotVal)
			if err != nil {
				return fmt.Errorf("invalid storage RLP for key %x (account %x): %w", slotKey, addr, err)
			}
			if len(rest) != 0 || len(content) > 32 {
				return fmt.Errorf("malformed storage value for key %x (account %x)", slotKey, addr)
			}
			var padded [32]byte
			copy(padded[32-len(content):], content)
			if err := emit(bintrie.StorageSlotKey(addr, slotKey[:]), padded); err != nil {
				return err
			}
			copy(slotKeyBuf[:common.HashLength], accountHash[:])
			copy(slotKeyBuf[common.HashLength:], slotHash[:])
			if err := part.slots.Add(slotKeyBuf[:], slotVal); err != nil {
				return err
			}
			slots++
		}
		if err := storageIter.Error(); err != nil {
			return fmt.Errorf("storage iteration failed for %x: %w", addr, err)
		}
		return nil
	}
	for accIter.Next() {
		accountHash := accIter.Hash()
		if part.past(accountHash) {
			break
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		// Full and slim account RLP both decode here.
		acc, err := types.FullAccount(accIter.Account())
		if err != nil {
			return fmt.Errorf("invalid account RLP: %w", err)
		}
		addr, err := keys.account(accountHash)
		if err != nil {
			return err
		}
		if preimages != nil {
			if err := preimages.beginAccount(addr, accountHash); err != nil {
				return err
			}
		}

		if err := emitAccountHeader(chaindb, addr, acc.Nonce, acc.Balance, common.BytesToHash(acc.CodeHash), codes, stats, emit); err != nil {
			return err
		}
		if err := part.accounts.Add(accountHash.Bytes(), merkleAccountRecord(acc.Nonce, acc.Balance, common.BytesToHash(acc.CodeHash))); err != nil {
			return err
		}

		if acc.Root != types.EmptyRootHash {
			if err := scanStorage(accountHash, addr, acc.Root); err != nil {
				return err
			}
		}
		if err := keys.done(addr); err != nil {
			return err
		}
		stats.accounts.Add(1)
		stats.slots.Add(slots)
		stats.leaves.Add(leaves)
		leaves, slots = 0, 0
		stats.report(false)
	}
	if err := accIter.Error(); err != nil {
		return fmt.Errorf("account iteration failed: %w", err)
	}
	return keys.close()
}

// checkPreimage rejects a preimage of the wrong length or one that does not
// hash back to its key: the store has no integrity of its own, and a corrupt
// entry would convert into a wrong-stem tree.
func checkPreimage(preimage []byte, hash common.Hash, wantLen int) error {
	if len(preimage) != wantLen {
		return fmt.Errorf("corrupt preimage for %x: %d bytes, want %d", hash, len(preimage), wantLen)
	}
	if crypto.Keccak256Hash(preimage) != hash {
		return fmt.Errorf("corrupt preimage for %x: value hashes to %x", hash, crypto.Keccak256(preimage))
	}
	return nil
}

// emitAccountHeader derives an account's header and code-zone leaves; the
// derivation rules stay pinned by the reference-vector parity tests.
func emitAccountHeader(chaindb ethdb.Database, addr common.Address, nonce uint64, balance *uint256.Int, codeHash common.Hash, codes *codeSet, stats *conversionStats, emit func([]byte, [32]byte) error) error {
	var (
		info  codeInfo
		code  []byte
		first bool // this account's code is new to the scan, and its chunks this worker's to emit
	)
	if codeHash != types.EmptyCodeHash {
		var known bool
		if info, known = codes.lookup(codeHash); !known {
			code = rawdb.ReadCode(chaindb, codeHash)
			if code == nil {
				return fmt.Errorf("missing code for hash %x (account %x)", codeHash, addr)
			}
			info.size = uint32(len(code))
			if _, isDelegation := types.ParseDelegation(code); isDelegation {
				info.delegation = code
			}
			first = codes.claim(codeHash, info)
		}
	}
	basic, err := bintrie.EncodeBasicData(info.size, nonce, balance)
	if err != nil {
		return fmt.Errorf("account %x: %w", addr, err)
	}
	if err := emit(bintrie.BasicDataKey(addr), basic); err != nil {
		return err
	}
	// A delegation replaces the code-hash leaf and has no code-zone leaves.
	if info.delegation != nil {
		return emit(bintrie.DelegationKey(addr), [32]byte(bintrie.EncodeDelegation(info.delegation)))
	}
	if err := emit(bintrie.CodeHashKey(addr), codeHash); err != nil {
		return err
	}
	// Code once per distinct hash: the leaves are content-addressed.
	if first && info.size > 0 {
		stats.codes.Add(1)
		return emitCodeChunks(codeHash, code, emit)
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
	// Anchor, then bookkeeping, then state: any crash prefix leaves state the
	// follower refuses, never an anchor or a position it would bind to it.
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))
	rawdb.DeletePBTAnchor(pbtdb)
	if err := rawdb.WipeMigrationState(chaindb); err != nil {
		return err
	}
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

// verifySourceRoot demands that the scanned records re-derive the state root
// they were read from, the converter's counterpart to the importer's anchor
// check. With a batch it also writes the flat state, see rederiveMerkleRoot.
func verifySourceRoot(accounts, slots *bintrie.RecordSorter, want common.Hash, batch ethdb.Batch, written *writtenState, tmpDir string, start time.Time) error {
	acctStream, err := accounts.Sort()
	if err != nil {
		return err
	}
	slotStream, err := slots.Sort()
	if err != nil {
		return err
	}
	got, err := rederiveMerkleRoot(acctStream, slotStream, batch, written, tmpDir, start)
	if err != nil {
		return err
	}
	if got != want {
		return fmt.Errorf("the scanned state re-derives merkle root %x, the source commits %x", got, want)
	}
	log.Info("Verified the scan against the source state root", "stateRoot", want)
	return nil
}

// wipePreimages deletes the preimage store, which a binary tree never
// consults. The whole key family goes, --vmdebug's SHA3 preimages included:
// they share the key space, and a 32-byte one looks like a slot key.
func wipePreimages(chaindb ethdb.Database) error {
	it := preimageKeys(chaindb)
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

// headCommitsBinaryTree reports whether head is past the fork, where the
// binary tree namespace holds the node's live state.
func headCommitsBinaryTree(chaindb ethdb.Database, head *types.Block) (bool, error) {
	stored := rawdb.ReadChainConfig(chaindb, rawdb.ReadCanonicalHash(chaindb, 0))
	if stored == nil {
		return false, errors.New("no chain config stored for the genesis block")
	}
	return stored.IsBinaryTrie(head.Number(), head.Time()), nil
}

// hasPreimages reports whether the preimage store holds anything.
func hasPreimages(chaindb ethdb.Database) bool {
	it := preimageKeys(chaindb)
	defer it.Release()
	return it.Next()
}

// preimageKeys iterates the preimage store: hash-length keys under its prefix.
func preimageKeys(chaindb ethdb.Database) ethdb.Iterator {
	return rawdb.NewKeyLengthIterator(chaindb.NewIterator(rawdb.PreimagePrefix, nil), len(rawdb.PreimagePrefix)+common.HashLength)
}
