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
	"cmp"
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math/bits"
	"os"
	"slices"
	"strconv"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/cmd/utils"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/node"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/ethereum/go-ethereum/trie/bintrie"
	"github.com/holiman/uint256"
	"github.com/urfave/cli/v2"
	"golang.org/x/sync/errgroup"
)

var (
	verifyOnlyFlag = &cli.BoolFlag{
		Name:  "verify-only",
		Usage: "Run both verification checks without writing anything",
	}
	keepPreimagesFlag = &cli.BoolFlag{
		Name:  "keep-preimages",
		Usage: "Persist the artifact's preimages into the node's preimage store",
	}

	bintrieImportCommand = &cli.Command{
		Name:      "import",
		Usage:     "Import a binary tree state from distribution artifacts",
		ArgsUsage: "<snapshot> <preimages> <anchor block-number-or-hash>",
		Action:    importBinaryTrie,
		Flags: slices.Concat([]cli.Flag{
			verifyOnlyFlag,
			keepPreimagesFlag,
			forceConvertFlag,
			tmpDirFlag,
			workersFlag,
		}, utils.NetworkFlags, utils.DatabaseFlags),
		Description: `
geth bintrie import [flags] <snapshot> <preimages> <anchor block-number-or-hash>

Verifies and ingests an EIP-8347 PBT snapshot and preimage file: the tree is
rebuilt from the leaves against the artifact's claimed root, and the leaves
are re-derived into the merkle root the anchor block commits, resolved from
the local header chain - the artifacts never vouch for themselves. Code is
reassembled from its chunks and pinned to each code hash. A failed check
leaves nothing openable. --verify-only runs both checks and writes nothing.
--workers derive the preimage file's candidate keys in parallel.

Preimages are not persisted unless --keep-preimages is given: nothing on a
binary-tree node reads them.

Memory: the sorters hold at most 4 GB of records, two run buffers each allocated
once, and the database its cache plus memtables, about 1.5 x (--cache x
--cache.database / 100). Go's collector can hold as much again before it runs,
so budget about twice their sum, or cap it with GOMEMLIMIT. Merging the
sorters' spilled runs takes up to 512 KiB per run on top, at most a quarter of
the sorter's budget once runs are many.

Disk: --tmpdir holds the sort spills and a copy of every tree node, each
until it is used up, so it needs disk space, not a tmpfs.

This command needs the node stopped. To import while the node keeps
following the chain, start geth with --pbt.import.snapshot, --pbt.import.preimages
and --pbt.import.anchor instead: the same checks run in the background once the
anchor is canonical and finalized, scratch goes to <datadir>/geth/pbt-import,
its sorters hold 1 GB rather than 4 beside the node's own memory, and the
migration follower adopts the anchor when the import attests. That
needs a chain config that schedules the binary tree (e.g. --override.genesis);
debug_migrationProgress reports "importing" until then.
`,
	}
)

// importBinaryTrie is the CLI action behind "geth bintrie import".
func importBinaryTrie(ctx *cli.Context) error {
	if ctx.NArg() != 3 {
		return errors.New("usage: geth bintrie import <snapshot> <preimages> <anchor block-number-or-hash>")
	}
	stack, _ := makeConfigNode(ctx)
	defer stack.Close()

	// Verification writes nothing, so it takes a read-only handle: a
	// writable open can repair and compact a datadir the operator only
	// wanted checked.
	verifyOnly := ctx.Bool(verifyOnlyFlag.Name)
	chaindb := utils.MakeChainDatabase(ctx, stack, verifyOnly)
	defer chaindb.Close()

	// The anchor's state root comes from the local header chain, never from
	// the operator or the artifacts.
	var (
		arg    = ctx.Args().Get(2)
		header *types.Header
	)
	if hashish(arg) {
		hash := common.HexToHash(arg)
		number, ok := rawdb.ReadHeaderNumber(chaindb, hash)
		if !ok {
			return fmt.Errorf("anchor block %x not found in the local header chain", hash)
		}
		// An anchor must be canonical: a side-chain header at the same height
		// commits a different state, and the EIP anchors at finalized blocks.
		if canonical := rawdb.ReadCanonicalHash(chaindb, number); canonical != hash {
			return fmt.Errorf("anchor block %x is not canonical at height %d", hash, number)
		}
		header = rawdb.ReadHeader(chaindb, hash, number)
	} else {
		number, err := strconv.ParseUint(arg, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid anchor block %q: %v", arg, err)
		}
		hash := rawdb.ReadCanonicalHash(chaindb, number)
		if hash == (common.Hash{}) {
			return fmt.Errorf("no canonical hash for anchor block %d", number)
		}
		header = rawdb.ReadHeader(chaindb, hash, number)
	}
	if header == nil {
		return errors.New("anchor header not found")
	}
	log.Info("Importing binary tree state", "anchor", header.Number, "stateRoot", header.Root)

	if ctx.Bool(forceConvertFlag.Name) {
		if verifyOnly {
			log.Warn("Ignoring --force: verification writes nothing")
		} else {
			head := rawdb.ReadHeadBlock(chaindb)
			if head == nil {
				return errors.New("no head block found")
			}
			// Past the fork the namespace is the node's live state.
			if past, err := headCommitsBinaryTree(chaindb, head); err != nil {
				return err
			} else if past {
				return errors.New("refusing --force: the head block commits the binary tree")
			}
			if err := wipeBinaryTrieState(chaindb, stack.ResolvePath("triedb")); err != nil {
				return fmt.Errorf("failed to wipe binary tree state: %w", err)
			}
		}
	}
	workers := ctx.Int(workersFlag.Name)
	if workers < 1 {
		return errors.New("--workers must be at least 1")
	}
	if _, err := importState(context.Background(), chaindb, importOptions{
		snapshot:      ctx.Args().Get(0),
		preimages:     ctx.Args().Get(1),
		anchor:        header,
		verifyOnly:    verifyOnly,
		keepPreimages: ctx.Bool(keepPreimagesFlag.Name),
		conversionOptions: conversionOptions{
			sortBudget: sortBudget,
			tmpDir:     ctx.String(tmpDirFlag.Name),
			workers:    workers,
		},
	}); err != nil {
		return err
	}
	// The namespace is written, but the node refuses to start until the chain
	// config schedules the tree; say so rather than leave it to be discovered.
	if !verifyOnly {
		if stored := rawdb.ReadChainConfig(chaindb, rawdb.ReadCanonicalHash(chaindb, 0)); stored == nil || !stored.IsPBT() {
			log.Warn("The chain config does not schedule the binary tree; the node will refuse to start until it does")
		}
	}
	return nil
}

// onlineSortBudget is the sort budget of an import running inside the node:
// a quarter of the offline command's, since the node keeps its caches and the
// chain beside it. Nethermind holds 256 MiB per sort buffer in the same place.
const onlineSortBudget = 1024 << 20

// pbtImporter runs an EIP-8347 import inside the node, in the background,
// while the chain keeps following its head (--pbt.import.*). The chain holds
// the binary tree for it from construction (BlockChainConfig.PBTImport) until
// FinishPBTImport releases it to the follower.
type pbtImporter struct {
	chain     *core.BlockChain
	db        ethdb.Database
	cfg       ethconfig.PBTImportConfig
	triedbDir string        // the PBT journal's directory, for wipes
	tmpDir    string        // scratch, owned by the import
	poll      time.Duration // how often the anchor wait looks again

	cancel context.CancelFunc
	done   chan struct{}
}

func newPBTImporter(stack *node.Node, backend *eth.Ethereum, cfg ethconfig.PBTImportConfig) *pbtImporter {
	return &pbtImporter{
		chain:     backend.BlockChain(),
		db:        backend.ChainDb(),
		cfg:       cfg,
		triedbDir: stack.ResolvePath("triedb"),
		tmpDir:    stack.ResolvePath("pbt-import"),
		poll:      5 * time.Second,
	}
}

// Start refuses what a restart cannot fix and leaves the import to run in
// the background.
func (p *pbtImporter) Start() error {
	ref, err := parseAnchorRef(p.cfg.Anchor)
	if err != nil {
		return err
	}
	num, hash, anchored := rawdb.ReadPBTAnchor(rawdb.NewTable(p.db, string(rawdb.PBTPrefix)))
	finished := anchored && rawdb.HasPBTState(p.db)
	if finished && ref.matches(num, hash) {
		log.Info("Binary tree already imported", "anchor", num, "hash", hash)
		p.chain.FinishPBTImport(nil) // releases a --pbt.import.force hold
		return nil
	}
	if !rawdb.ReadPBTImportPending(p.db) {
		return fmt.Errorf("binary tree import refused: the database holds a tree anchored at block %d (%x); --%s replaces it", num, hash, utils.PBTImportForceFlag.Name)
	}
	ctx, cancel := context.WithCancel(context.Background())
	p.cancel, p.done = cancel, make(chan struct{})
	go func() {
		defer close(p.done)
		p.run(ctx, ref)
	}()
	return nil
}

// Stop cancels a running import and waits for it, before the chain closes
// the database under it.
func (p *pbtImporter) Stop() error {
	if p.cancel != nil {
		p.cancel()
		<-p.done
	}
	return nil
}

// run waits for the anchor, imports at it and hands the outcome to the
// chain. An interrupted run leaves the hold and its writes to the next start.
func (p *pbtImporter) run(ctx context.Context, ref anchorRef) {
	anchor, err := p.awaitAnchor(ctx, ref)
	if err == nil {
		err = p.importAt(ctx, anchor)
	}
	switch {
	case err == nil:
		p.chain.FinishPBTImport(nil)
	case ctx.Err() != nil:
		log.Info("Binary tree import interrupted; restart with the same flags to redo it")
	default:
		log.Error("Binary tree import failed", "err", err)
		p.chain.FinishPBTImport(err)
	}
}

// awaitAnchor waits until the anchor is canonical and finalized, the blocks
// EIP-8347 anchors at, and fails once it never can be.
func (p *pbtImporter) awaitAnchor(ctx context.Context, ref anchorRef) (*types.Header, error) {
	var logged time.Time
	for {
		header, waiting, err := p.anchorState(ref)
		if err != nil || header != nil {
			return header, err
		}
		if time.Since(logged) >= time.Minute {
			log.Info("Binary tree import waiting for its anchor", "anchor", p.cfg.Anchor, "state", waiting)
			logged = time.Now()
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(p.poll):
		}
	}
}

// anchorState returns the anchor's header once it is canonical and
// finalized, else what it still waits for, or why it never will qualify: a
// finalized height holding another block, or a block past the fork.
func (p *pbtImporter) anchorState(ref anchorRef) (*types.Header, string, error) {
	number, hash := ref.number, ref.hash
	if ref.byHash {
		n, ok := rawdb.ReadHeaderNumber(p.db, hash)
		if !ok {
			return nil, "unknown block", nil
		}
		number = n
	}
	var (
		canonical = rawdb.ReadCanonicalHash(p.db, number)
		final     = p.chain.CurrentFinalBlock()
		finalized = final != nil && final.Number.Uint64() >= number
	)
	if !ref.byHash {
		hash = canonical
	}
	switch {
	case canonical == (common.Hash{}):
		return nil, "not in the canonical chain yet", nil
	case canonical != hash && finalized:
		return nil, "", fmt.Errorf("anchor block %x can never be canonical: height %d is finalized at %x", hash, number, canonical)
	case canonical != hash:
		return nil, "not canonical", nil
	case !finalized:
		return nil, "not finalized", nil
	}
	header := rawdb.ReadHeader(p.db, hash, number)
	if header == nil {
		return nil, "", fmt.Errorf("anchor header %d (%x) missing", number, hash)
	}
	if p.chain.Config().IsBinaryTrie(header.Number, header.Time) {
		return nil, "", fmt.Errorf("anchor block %d is past the binary tree fork", number)
	}
	return header, "", nil
}

// importAt imports the pair at the anchor. The namespace is the import's
// while the chain holds it: whatever a stopped or failed run, or the tree
// --pbt.import.force replaces, left there is wiped first, and a failure
// wipes its own writes.
func (p *pbtImporter) importAt(ctx context.Context, anchor *types.Header) error {
	if err := os.RemoveAll(p.tmpDir); err != nil {
		return err
	}
	if err := os.MkdirAll(p.tmpDir, 0o700); err != nil {
		return err
	}
	defer os.RemoveAll(p.tmpDir)

	if dirty, err := hasBinaryTrieState(p.db); err != nil {
		return err
	} else if dirty {
		if err := wipeBinaryTrieState(p.db, p.triedbDir); err != nil {
			return err
		}
	}
	log.Info("Starting binary tree import in the background", "anchor", anchor.Number, "hash", anchor.Hash(),
		"stateRoot", anchor.Root, "snapshot", p.cfg.Snapshot, "preimages", p.cfg.Preimages)
	yield := newBlockYield()
	defer followBlockProcessing(p.chain, yield)()
	_, err := importState(ctx, p.db, importOptions{
		snapshot:  p.cfg.Snapshot,
		preimages: p.cfg.Preimages,
		anchor:    anchor,
		yield:     yield,
		conversionOptions: conversionOptions{
			sortBudget: onlineSortBudget,
			tmpDir:     p.tmpDir,
			workers:    cmp.Or(p.cfg.Workers, utils.PBTImportWorkersFlag.Value),
			// Finalized blocks do not reorg, but an attested tree anchored off
			// the canonical chain is refused by the follower anyway: say so now.
			beforeAttest: func() error {
				if rawdb.ReadCanonicalHash(p.db, anchor.Number.Uint64()) != anchor.Hash() {
					return fmt.Errorf("anchor block %d (%x) left the canonical chain during the import", anchor.Number, anchor.Hash())
				}
				return nil
			},
		},
	})
	if err != nil && ctx.Err() == nil {
		if werr := wipeBinaryTrieState(p.db, p.triedbDir); werr != nil {
			log.Error("Failed to wipe a failed binary tree import", "err", werr)
		}
	}
	return err
}

// anchorRef is an anchor argument: a block hash, or else a block number.
type anchorRef struct {
	hash   common.Hash
	number uint64
	byHash bool
}

func parseAnchorRef(arg string) (anchorRef, error) {
	if common.IsHexHash(arg) {
		return anchorRef{hash: common.HexToHash(arg), byHash: true}, nil
	}
	number, err := strconv.ParseUint(arg, 10, 64)
	if err != nil {
		return anchorRef{}, fmt.Errorf("invalid binary tree import anchor %q: want a block number or hash", arg)
	}
	return anchorRef{number: number}, nil
}

// matches reports whether the ref names the given block.
func (r anchorRef) matches(number uint64, hash common.Hash) bool {
	if r.byHash {
		return r.hash == hash
	}
	return r.number == number
}

// The EIP-8347 consumer: importState ingests a distributed PBT snapshot and
// preimage file after proving them with the dual-check. Check 1 rebuilds the
// tree from the leaves and demands the claimed root; check 2 re-derives the
// MPT from the same leaves through the preimages and demands the anchor
// block's state root, with every distinct bytecode reassembled from its
// chunks and pinned to its code hash. The preimage set is held to exactness
// in both directions: a leaf without a preimage and a preimage without a
// leaf both reject. Nothing becomes openable unverified - the flat-state
// attestation lands last - and verify-only mode runs the whole pipeline
// with every writer disarmed.

// maxImportCodeSize bounds the code size an artifact may claim. It is a
// memory bound rather than a consensus rule - the protocol's own ceiling is
// params.MaxCodeSizeAmsterdam, sixteen times smaller - because the claim
// sizes a buffer and a candidate set before anything can be checked against
// it: a code_size field costs an attacker four bytes and would otherwise
// cost the verifier gigabytes.
const maxImportCodeSize = 1 << 20

// importGroup carries one account's header-stem leaves through the join.
type importGroup struct {
	stem        []byte
	addr        common.Address
	basic       *[32]byte // nil, or leaves[0]
	codeHash    *[32]byte // nil, or leaves[1]
	delegation  *[32]byte // nil, or leaves[2]
	leaves      [3][32]byte
	slots       []importSlot // header-range storage, in sub-index order
	headerClaim uint64       // header-range slots the preimage file lists
	headerSeen  uint64       // header-range slots the leaves hold
}

type importSlot struct {
	slot  common.Hash
	value [32]byte
}

// importOptions carries an import's inputs: the two artifacts, the anchor
// whose header the state is proven against, and the conversion tunables the
// two commands share. A self-converter sets own instead of the artifacts.
type importOptions struct {
	snapshot      string
	preimages     string
	anchor        *types.Header
	verifyOnly    bool
	keepPreimages bool
	own           *ownLeaves
	// yield, when set, pauses the import while the node processes a block.
	yield *blockYield
	conversionOptions
}

// ownLeaves carries a self-converter's leaves and join candidates into the
// import in place of the artifacts. EIP-8347 has a self-converter run check 2
// and its code limb over the leaves it derived, but not check 1: its build is
// the tree, and the only claimed root is the one that build computes. Its
// code store holds its code already, so none is written.
type ownLeaves struct {
	leaves     *bintrie.RecordStream         // in tree-key order
	candidates *bintrie.RecordSorter         // derived by the scan, see accountCandidates
	unanchored bool                          // no block names the root: attest no anchor
	onLeaf     func(key, value []byte) error // also hands each leaf to a snapshot artifact
	artifacts  func(root common.Hash) error  // finishes and checks the artifacts before attesting
	count      atomic.Uint64                 // leaves handed out, for progress
}

// next hands the leaf pipeline the conversion's next leaf.
func (o *ownLeaves) next() ([]byte, [32]byte, error) {
	key, value, err := o.leaves.Next()
	if err != nil {
		return nil, [32]byte{}, err
	}
	if o.onLeaf != nil {
		if err := o.onLeaf(key, value); err != nil {
			return nil, [32]byte{}, err
		}
	}
	o.count.Add(1)
	return key, [32]byte(value), nil
}

// importState verifies and ingests the artifacts, returning the imported
// root. With verifyOnly the database is never touched. Cancelling ctx stops
// it between batches, before anything is attested.
func importState(ctx context.Context, chaindb ethdb.Database, opts importOptions) (common.Hash, error) {
	var (
		anchorRoot = opts.anchor.Root
		verifyOnly = opts.verifyOnly
		own        = opts.own
	)
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))
	if !verifyOnly {
		if dirty, err := hasBinaryTrieState(chaindb); err != nil {
			return common.Hash{}, err
		} else if dirty {
			return common.Hash{}, errors.New("binary tree namespace already holds state, whether a finished import or the debris of one; re-run with --force to wipe it")
		}
	}
	// The candidate sorters accumulate alone in Phase 1 - they fully drain
	// via Sort() before Phase 2's other sorters fill in earnest, so they
	// share the whole budget.
	// Phase 2 runs four sorters concurrently (accounts, slots, chunks and,
	// briefly at the account/code-zone boundary, the code stems), so they
	// split the same budget four ways: either phase's peak sorter RAM stays
	// within opts.sortBudget, a sorter's budget being the bytes its two run
	// buffers hold.
	quarter := opts.sortBudget / 4

	// Open both artifacts before the expensive phase: a mistyped path should
	// not cost a full pass over the other file. A self-converter has none.
	var (
		pre    *preimageReader
		snap   *snapshotReader
		leaves leafReader = own
		err    error
	)
	if own == nil {
		if pre, err = openPreimages(opts.preimages); err != nil {
			return common.Hash{}, err
		}
		defer pre.close()
		if snap, err = openSnapshot(opts.snapshot); err != nil {
			return common.Hash{}, err
		}
		defer snap.close()
		leaves = snap
	}
	var (
		pbtBatch ethdb.Batch // tree nodes and flat state, inside the namespace
		rawBatch ethdb.Batch // code and preimages, outside it
	)
	if !verifyOnly {
		pbtBatch = pbtdb.NewBatch()
		if own == nil {
			rawBatch = chaindb.NewBatch()
		}
	}
	flush := func(force bool) error {
		for _, batch := range []ethdb.Batch{pbtBatch, rawBatch} {
			if batch == nil {
				continue
			}
			if force || batch.ValueSize() >= ethdb.IdealBatchSize {
				if err := batch.Write(); err != nil {
					return err
				}
				batch.Reset()
			}
		}
		return nil
	}

	// Phase 1: every candidate tree key the preimages can stand for, from
	// the preimage file, or from the keys a self-converter's scan resolved.
	var cand *bintrie.RecordSorter
	if own != nil {
		cand = own.candidates
	} else if cand, err = preimageCandidates(ctx, pre, rawBatch, flush, opts); err != nil {
		return common.Hash{}, err
	}
	defer cand.Close() // a second Close is a no-op; this one covers the error paths

	// Phase 2: stream the leaves; rebuild the tree; join every account and
	// storage leaf to its preimage candidate; collect the MPT re-derivation
	// inputs.
	var prog *progress
	if own == nil {
		prog = startProgress("Verifying snapshot", "bytes", uint64(snap.size), snap.read.Load)
	} else {
		prog = startProgress("Building the binary tree", "leaves", 0, own.count.Load)
	}
	defer prog.stop()
	candStream, err := cand.Sort()
	if err != nil {
		return common.Hash{}, err
	}
	held := &heldStream{stream: candStream}

	// Tree nodes are staged per depth and written only once check 1 has
	// passed, in key order (see nodeStager).
	var (
		builderErr error
		onNode     func(path []byte, hash common.Hash, blob []byte)
		nodes      *nodeStager
	)
	if !verifyOnly {
		if nodes, err = newNodeStager(opts.tmpDir); err != nil {
			return common.Hash{}, err
		}
		defer nodes.close()
		onNode = func(path []byte, hash common.Hash, blob []byte) {
			if builderErr != nil {
				return
			}
			builderErr = nodes.add(path, blob)
		}
	}
	builder := bintrie.NewStackBuilder(onNode)

	var (
		acctSorter  = bintrie.NewRecordSorter(opts.tmpDir, quarter, nil) // keccak(addr) -> nonce ‖ balance ‖ codeHash
		slotSorter  = bintrie.NewRecordSorter(opts.tmpDir, quarter, nil) // keccak(addr) ‖ keccak(slot) -> rlp value
		chunkSorter = bintrie.NewRecordSorter(opts.tmpDir, quarter, nil) // codeHash ‖ index -> chunk
		codeSizes   = make(map[common.Hash]uint32)
		stats       = newStats("Importing state")
	)
	defer acctSorter.Close()
	defer slotSorter.Close()
	defer chunkSorter.Close()

	// addSlot records one storage slot everywhere it goes but flat state,
	// which the check-2 fold writes in keccak order: the preimage store,
	// and the MPT re-derivation. The sorter copies the key and value, so
	// both are built in buffers reused for every slot.
	var (
		hasher  = newKeccakHasher()
		enc     = rlp.NewEncoderBuffer(nil)
		slotKey [2 * common.HashLength]byte
		slotRLP []byte
	)
	addSlot := func(accountHash common.Hash, slot common.Hash, value [32]byte) error {
		slotHash := hasher.hash(slot[:])
		copy(slotKey[:], accountHash[:])
		copy(slotKey[common.HashLength:], slotHash[:])
		enc.Reset(nil)
		enc.WriteBytes(common.TrimLeftZeroes(value[:]))
		slotRLP = enc.AppendToBytes(slotRLP[:0])
		stats.slots.Add(1)
		return slotSorter.Add(slotKey[:], slotRLP)
	}

	// sealGroup validates one account's header leaves and records it
	// everywhere it goes but flat state, which the check-2 fold writes in
	// keccak order. The account record is built in a reused buffer too.
	var acctRecord []byte
	sealGroup := func(g *importGroup) error {
		// Header-range slots are derived from sub-indices, so the preimage
		// file's list of them is checked rather than believed - the exact
		// match the spec asks for, in both directions.
		if g.headerClaim != g.headerSeen {
			missing := g.headerClaim & ^g.headerSeen
			if missing != 0 {
				return fmt.Errorf("preimage file names slot %d of %x, which the state does not hold",
					bits.TrailingZeros64(missing), g.addr)
			}
			return fmt.Errorf("account %x holds slot %d, which the preimage file does not name",
				g.addr, bits.TrailingZeros64(g.headerSeen&^g.headerClaim))
		}
		// The typed record fixes the basic-data leaf's version and reserved
		// bytes at zero, so a decoded leaf is canonical by construction.
		var (
			nonce    uint64
			balance  = new(uint256.Int)
			codeSize uint32
		)
		if g.basic != nil {
			_, codeSize, nonce, balance = bintrie.DecodeBasicData(g.basic[:])
		}
		// A header record carries kind 0, 1 or 2, and derivation emits
		// exactly one of a code-hash or a delegation leaf for it - g.basic,
		// and one of g.codeHash/g.delegation, are never both nil or both set.
		var codeHash common.Hash
		switch {
		case g.delegation != nil:
			// The delegation leaf derives from a 20-byte target, so its
			// designator is always the 23-byte EIP-7702 indicator.
			designator := g.delegation[:23]
			codeHash = hasher.hash(designator)
			if rawBatch != nil {
				rawdb.WriteCode(rawBatch, codeHash, designator)
			}
		case g.codeHash != nil:
			codeHash = common.BytesToHash(g.codeHash[:])
			if codeSize > maxImportCodeSize {
				return fmt.Errorf("account %x claims %d bytes of code, over the %d-byte import bound", g.addr, codeSize, maxImportCodeSize)
			}
			// Sizes are anchored through the code limb; register every hash
			// that claims code, and hold shared hashes to one size.
			if codeSize > 0 || codeHash != types.EmptyCodeHash {
				if have, ok := codeSizes[codeHash]; ok && have != codeSize {
					return fmt.Errorf("code %x claimed with sizes %d and %d", codeHash, have, codeSize)
				}
				codeSizes[codeHash] = codeSize
			}
		}
		accountHash := hasher.hash(g.addr[:])
		stats.accounts.Add(1)

		acctRecord = appendMerkleAccountRecord(acctRecord[:0], nonce, balance, codeHash)
		if err := acctSorter.Add(accountHash[:], acctRecord); err != nil {
			return err
		}
		for _, s := range g.slots {
			if err := addSlot(accountHash, s.slot, s.value); err != nil {
				return err
			}
		}
		return flush(false)
	}

	// skipCandidate handles a candidate the snapshot stream has moved past:
	// an unmatched slot is a surplus preimage; unmatched header candidates
	// are fine singly (an account holds one of code-hash and delegation, and
	// basic data collapses at zero) but a whole stem of them means the
	// preimage file names an address the state does not hold.
	// Every candidate must match a leaf, and every leaf a candidate: that is
	// the spec's exact set match, in both directions. A header-stem candidate
	// serves its whole group, so it stays held until the leaves move past it,
	// and heldMatched records whether it ever matched.
	heldMatched := false
	surplus := func(ckey, cvalue []byte) error {
		addr := common.BytesToAddress(cvalue[:common.AddressLength])
		if ckey[0] == bintrie.AccountZone {
			return fmt.Errorf("preimage file names address %x, which the state does not hold", addr)
		}
		return fmt.Errorf("preimage file names slot %x of %x, which the state does not hold",
			cvalue[common.AddressLength:], addr)
	}

	// The code zone joins against stems derived from the observed code
	// hashes, built at the account/code zone boundary.
	var (
		codeCand     *bintrie.RecordSorter
		codeHeld     *heldStream
		codeRecord   []byte
		group        *importGroup
		spare        importGroup // the one group, reused for every account
		pastAccounts bool
	)
	defer func() {
		if codeCand != nil {
			codeCand.Close()
		}
	}()
	enterCodeZone := func() error {
		if group != nil {
			if err := sealGroup(group); err != nil {
				return err
			}
			group = nil
		}
		codeCand = bintrie.NewRecordSorter(opts.tmpDir, quarter, nil)
		for hash, size := range codeSizes {
			if size == 0 {
				continue
			}
			chunks := (uint64(size) + 30) / 31
			for ti := uint64(0); ti <= (chunks-1)/256; ti++ {
				value := append(hash.Bytes(), binary.BigEndian.AppendUint64(nil, ti)...)
				if err := codeCand.Add(bintrie.CodeChunkStem(hash, ti), value); err != nil {
					return err
				}
			}
		}
		stream, err := codeCand.Sort()
		if err != nil {
			return err
		}
		codeHeld = &heldStream{stream: stream}
		pastAccounts = true
		return nil
	}

	// The storage zone groups every leaf of one account before the next, so
	// keccak(addr) is computed once per account.
	var (
		lastStorageAddr common.Address
		lastStorageHash common.Hash
		haveStorageHash bool
	)

	// The pass is a pipeline: a decoder goroutine reads the snapshot into
	// batches, which the tree builder's goroutine and this loop each consume
	// in order, so decoding, hashing and joining overlap.
	pipe := startLeafPipeline(leaves, builder, &builderErr, opts.yield)
	defer pipe.stop()
	var (
		batch *leafBatch
		next  int
	)
	for {
		if batch == nil || next == len(batch.values) {
			if err := opts.yield.pause(ctx); err != nil {
				return common.Hash{}, err
			}
			if batch != nil {
				pipe.release(batch)
			}
			var ok bool
			if batch, ok = <-pipe.leaves; !ok {
				break
			}
			next = 0
			stats.leaves.Add(uint64(len(batch.values)))
		}
		key, value := batch.key(next), &batch.values[next]
		next++
		// Zone boundary: the account zone ends where anything else begins.
		if key[0] != bintrie.AccountZone && !pastAccounts {
			if err := enterCodeZone(); err != nil {
				return common.Hash{}, err
			}
		}

		if key[0] == bintrie.CodeZone {
			// Join against the derived code stems; a stem with no candidate
			// is code no account's hash addresses.
			stem := key[:33]
			for {
				ckey, cvalue, err := codeHeld.current()
				if err != nil {
					return common.Hash{}, err
				}
				if ckey == nil || bytes.Compare(ckey, stem) > 0 {
					return common.Hash{}, fmt.Errorf("code leaf %x is addressed by no account's code hash", key)
				}
				if bytes.Equal(ckey, stem) {
					var (
						ti    = binary.BigEndian.Uint64(cvalue[32:])
						index = ti*256 + uint64(key[33])
					)
					// The record is the code hash and the chunk index.
					codeRecord = binary.BigEndian.AppendUint64(append(codeRecord[:0], cvalue[:32]...), index)
					if err := chunkSorter.Add(codeRecord, value[:]); err != nil {
						return common.Hash{}, err
					}
					break
				}
				codeHeld.advance() // matched already, or a fully zero-chunk stem
			}
			continue
		}

		// Account and storage zones join against the preimage candidates.
		// An account leaf matches its stem - one candidate serves the whole
		// group - and a storage leaf its full key.
		joinKey := key
		if key[0] == bintrie.AccountZone {
			joinKey = key[:len(key)-1]
		}
		var matched []byte
		for {
			ckey, cvalue, err := held.current()
			if err != nil {
				return common.Hash{}, err
			}
			if ckey == nil || bytes.Compare(ckey, joinKey) > 0 {
				return common.Hash{}, fmt.Errorf("leaf %x has no preimage", key)
			}
			if bytes.Equal(ckey, joinKey) {
				matched, heldMatched = cvalue, true
				break
			}
			// The held candidate is behind this leaf, so it is finished.
			if !heldMatched {
				return common.Hash{}, surplus(ckey, cvalue)
			}
			held.advance()
			heldMatched = false
		}
		addr := common.BytesToAddress(matched[:common.AddressLength])

		if key[0] == bintrie.StorageZone {
			if !haveStorageHash || addr != lastStorageAddr {
				lastStorageAddr, lastStorageHash, haveStorageHash = addr, hasher.hash(addr[:]), true
			}
			slot := common.BytesToHash(matched[common.AddressLength:])
			if err := addSlot(lastStorageHash, slot, *value); err != nil {
				return common.Hash{}, err
			}
			continue
		}

		// Account zone: gather the stem's leaves.
		stem, sub := key[:33], key[33]
		if group != nil && !bytes.Equal(group.stem, stem) {
			if err := sealGroup(group); err != nil {
				return common.Hash{}, err
			}
			group = nil
		}
		if group == nil {
			spare = importGroup{
				stem:        append(spare.stem[:0], stem...),
				addr:        addr,
				headerClaim: binary.BigEndian.Uint64(matched[common.AddressLength:]),
				slots:       spare.slots[:0],
			}
			group = &spare
		}
		switch {
		case sub == bintrie.BasicDataLeafKey:
			group.leaves[0] = *value
			group.basic = &group.leaves[0]
		case sub == bintrie.CodeHashLeafKey:
			group.leaves[1] = *value
			group.codeHash = &group.leaves[1]
		case sub == bintrie.DelegationLeafKey:
			group.leaves[2] = *value
			group.delegation = &group.leaves[2]
		case sub >= bintrie.HeaderStorageOffset && sub < bintrie.HeaderStorageOffset+bintrie.HeaderStorageSlots:
			// The slot number is the sub-index, not something the preimage
			// file has to be trusted for.
			var slot common.Hash
			slot[31] = sub - bintrie.HeaderStorageOffset
			group.slots = append(group.slots, importSlot{slot: slot, value: *value})
			group.headerSeen |= 1 << (sub - bintrie.HeaderStorageOffset)
		}
	}
	// A pipeline that stopped early leaves the join incomplete: its own
	// error is the one to report.
	if err := pipe.wait(); err != nil {
		return common.Hash{}, err
	}
	if group != nil {
		if err := sealGroup(group); err != nil {
			return common.Hash{}, err
		}
		group = nil
	}
	// Drain what remains of both candidate streams: leftover slots are
	// surplus preimages, leftover code stems are all-zero chunks.
	for {
		ckey, cvalue, err := held.current()
		if err != nil {
			return common.Hash{}, err
		}
		if ckey == nil {
			break
		}
		if !heldMatched {
			return common.Hash{}, surplus(ckey, cvalue)
		}
		held.advance()
		heldMatched = false
	}
	// The code candidates are only ever read inside the loop above, so a
	// snapshot that never left the account zone needs none built.
	cand.Close()

	// Check 1: the leaves must rebuild to the claimed root. A self-converter
	// claims none: its build is the tree.
	root := builder.Finish()
	if builderErr != nil {
		return common.Hash{}, builderErr
	}
	if own == nil {
		if root != snap.root {
			return common.Hash{}, fmt.Errorf("snapshot leaves rebuild to %x, its pbtRoot claims %x", root, snap.root)
		}
		prog.finish("Verified snapshot consistency", "root", root, "leaves", stats.leaves.Load(), "digest", snap.digest())
	} else {
		prog.finish("Built the binary tree", "root", root, "leaves", stats.leaves.Load())
	}
	// One knob for the parallelism: the digests share opts.workers hashing
	// goroutines, and as many families are read back at once.
	var (
		written  = newWrittenState(opts.workers)
		families = written.families()
		readBack errgroup.Group
	)
	defer written.close() // after the read-back below
	readBack.SetLimit(max(opts.workers, 1))
	defer readBack.Wait()
	if nodes != nil {
		if err := writeStagedNodes(ctx, pbtdb, nodes, written.nodes, opts.yield); err != nil {
			return common.Hash{}, err
		}
		// The tree nodes are all written: read them back while the code
		// limb and check 2 run. Reading puts nothing in the memtable.
		readBack.Go(func() error { return verifyWritten(ctx, pbtdb, opts.yield, families[0]) })
		// Likewise the flat storage, once check 2 has written its last row.
		written.storageWritten = func() {
			readBack.Go(func() error { return verifyWritten(ctx, pbtdb, opts.yield, families[2]) })
		}
	}

	// The code limb: every claimed code hash must reassemble from its
	// chunks, and the chunks must be exactly the code's re-chunking.
	chunkStream, err := chunkSorter.Sort()
	if err != nil {
		return common.Hash{}, err
	}
	chunkHeld := &heldStream{stream: chunkStream}
	hashes := make([]common.Hash, 0, len(codeSizes))
	for hash := range codeSizes {
		hashes = append(hashes, hash)
	}
	slices.SortFunc(hashes, func(a, b common.Hash) int { return bytes.Compare(a[:], b[:]) })
	prog = startProgress("Verifying code", "codes", uint64(len(hashes)), stats.codes.Load)
	defer prog.stop()
	var chunks []bintrie.IndexedChunk // one code's, reused for the next
	for _, hash := range hashes {
		if err := opts.yield.pause(ctx); err != nil {
			return common.Hash{}, err
		}
		chunks = chunks[:0]
		for {
			ckey, cvalue, err := chunkHeld.current()
			if err != nil {
				return common.Hash{}, err
			}
			if ckey == nil || !bytes.Equal(ckey[:32], hash[:]) {
				break
			}
			var chunk [32]byte
			copy(chunk[:], cvalue)
			chunks = append(chunks, bintrie.IndexedChunk{Index: binary.BigEndian.Uint64(ckey[32:]), Chunk: chunk})
			chunkHeld.advance()
		}
		code, err := bintrie.AssembleCode(hash, codeSizes[hash], chunks)
		if err != nil {
			return common.Hash{}, err
		}
		// A kind-1 account's bytecode must not itself be an EIP-7702
		// delegation indicator: that account must carry kind 2 instead.
		if _, ok := types.ParseDelegation(code); ok {
			return common.Hash{}, fmt.Errorf("code %x recovers to a delegation indicator; its account must hold kind 2", hash)
		}
		if rawBatch != nil {
			rawdb.WriteCode(rawBatch, hash, code)
			if err := flush(false); err != nil {
				return common.Hash{}, err
			}
		}
		stats.codes.Add(1)
	}
	if ckey, _, err := chunkHeld.current(); err != nil {
		return common.Hash{}, err
	} else if ckey != nil {
		return common.Hash{}, fmt.Errorf("code chunks left over for unclaimed hash %x", ckey[:32])
	}
	prog.finish("Verified code against its hashes", "codes", stats.codes.Load())
	chunkSorter.Close() // its spill is done with: free the disk now, not at return

	// Check 2: re-derive the MPT and demand the anchor root.
	acctStream, err := acctSorter.Sort()
	if err != nil {
		return common.Hash{}, err
	}
	slotStream, err := slotSorter.Sort()
	if err != nil {
		return common.Hash{}, err
	}
	got, err := rederiveMerkleRoot(ctx, acctStream, slotStream, pbtBatch, written, opts.tmpDir, opts.workers, stats.accounts.Load()+stats.slots.Load(), opts.yield)
	if err != nil {
		return common.Hash{}, err
	}
	acctSorter.Close() // likewise, drained by check 2
	slotSorter.Close()
	if got != anchorRoot {
		return common.Hash{}, fmt.Errorf("the leaves re-derive merkle root %x, the anchor commits %x", got, anchorRoot)
	}
	log.Info("Verified consensus anchoring", "stateRoot", anchorRoot, "accounts", stats.accounts.Load(), "slots", stats.slots.Load())

	if verifyOnly {
		log.Info("Verification complete, nothing written", "binaryRoot", root,
			"snapshotDigest", snap.digest(), "preimageDigest", pre.digest())
		return root, nil
	}
	if err := flush(true); err != nil {
		return common.Hash{}, err
	}
	readBack.Go(func() error { return verifyWritten(ctx, pbtdb, opts.yield, families[1]) })
	if err := readBack.Wait(); err != nil {
		return common.Hash{}, err
	}
	logWritten(written)
	// A cancelled import never attests, however far it got: an interrupted
	// run is redone from scratch, never half-trusted.
	if err := ctx.Err(); err != nil {
		return common.Hash{}, err
	}
	if own != nil && own.artifacts != nil {
		if err := own.artifacts(root); err != nil {
			return common.Hash{}, err
		}
	}
	if opts.beforeAttest != nil {
		if err := opts.beforeAttest(); err != nil {
			return common.Hash{}, err
		}
	}
	// The anchor rides with the root: the tree does not say which block it
	// commits, and catching up from here has to start somewhere. The
	// attestation is the completion marker and comes last; the imported tree
	// bases an empty history, like a conversion.
	rawdb.WriteSnapshotRoot(pbtdb, root)
	if own != nil && own.unanchored {
		log.Warn("Converted without an anchor: the namespace is not bootable; reconvert at the head root with --force before starting the node")
	} else {
		rawdb.WritePBTAnchor(pbtdb, opts.anchor.Number.Uint64(), opts.anchor.Hash())
	}
	rawdb.WritePBTFlatState(pbtdb)
	if own == nil {
		log.Info("Import complete", "binaryRoot", root, "anchor", opts.anchor.Number,
			"snapshotDigest", snap.digest(), "preimageDigest", pre.digest())
	}
	return root, nil
}

// preimageCandidates derives every candidate tree key the preimage file can
// stand for: Phase 1 of an import. The reader stays sequential, digesting the
// file; workers check its order, a batch against the one before it, and
// derive the candidates into sorters of their own, which the first then
// absorbs into one merge. Keeping the preimages needs their hashes in order,
// so then the reader checks the order itself. A record's slot keys go out in
// runs of at most a batch's worth, so a contract with many slots is spread
// over the workers rather than held whole; the reader counts its
// header-range slots, which only the record as a whole determines.
func preimageCandidates(ctx context.Context, pre *preimageReader, rawBatch ethdb.Batch, flush func(bool) error, opts importOptions) (*bintrie.RecordSorter, error) {
	keep := rawBatch != nil && opts.keepPreimages
	workers := max(opts.workers, 1)
	cands := make([]*bintrie.RecordSorter, workers)
	for i := range cands {
		cands[i] = bintrie.NewRecordSorter(opts.tmpDir, splitBudget(opts.sortBudget, workers), nil)
	}
	fail := func(err error) (*bintrie.RecordSorter, error) {
		for _, c := range cands {
			c.Close()
		}
		return nil, err
	}
	var (
		preAccounts  uint64
		prePreimages uint64
		batches      = 3*workers + 1 // queued, at the workers and at the reader
		records      = make(chan *preimageBatch, batches)
		free         = make(chan *preimageBatch, batches)
		g, gctx      = errgroup.WithContext(ctx)
	)
	for range batches {
		free <- &preimageBatch{runs: make([]preimageRun, 0, preimageBatchSize), keys: make([]common.Hash, 0, preimageBatchSlots)}
	}
	what, verified := "Verifying preimage file", "Verified preimage file"
	if keep {
		what, verified = "Verifying and storing preimages", "Verified and stored preimages"
	}
	prog := startProgress(what, "bytes", uint64(pre.size), pre.read.Load)
	defer prog.stop()
	for _, cand := range cands {
		g.Go(func() error {
			var (
				buf    candidateBuf
				hasher = newKeccakHasher()
			)
			for batch := range records {
				if err := opts.yield.pause(gctx); err != nil {
					return err
				}
				order := batch.order
				for i := range batch.runs {
					run := &batch.runs[i]
					if !keep {
						if run.first {
							if _, err := order.record(hasher, run.index, run.addr); err != nil {
								return err
							}
						}
						if err := order.slots(hasher, run.slots, nil); err != nil {
							return err
						}
					}
					if err := addSlotCandidates(cand, &buf, run.addr, run.slots); err != nil {
						return err
					}
					if run.last {
						if err := addHeaderCandidate(cand, &buf, run.addr, run.headerSlots); err != nil {
							return err
						}
					}
				}
				batch.runs, batch.keys = batch.runs[:0], batch.keys[:0]
				free <- batch
			}
			return nil
		})
	}
	readErr := func() error {
		defer close(records)
		var (
			batch  *preimageBatch
			order  preimageOrder // the reader's own check, when it keeps the preimages
			hasher = newKeccakHasher()
		)
		take := func() error {
			if err := opts.yield.pause(gctx); err != nil {
				return err
			}
			select {
			case batch = <-free:
				return nil
			case <-gctx.Done():
				return gctx.Err() // a worker failed; its error is the one reported
			}
		}
		// send hands the batch over and takes the next, which continues the
		// order check from this one's last run.
		send := func() error {
			last := &batch.runs[len(batch.runs)-1]
			next := preimageOrder{index: last.index, addr: crypto.Keccak256Hash(last.addr[:]), started: true}
			if !last.last {
				next.slot, next.slotted = crypto.Keccak256Hash(last.slots[len(last.slots)-1][:]), true
			}
			records <- batch // never blocks: the channel holds every batch
			if err := take(); err != nil {
				return err
			}
			batch.order = next
			return nil
		}
		if err := take(); err != nil {
			return err
		}
		batch.order = preimageOrder{}
		for {
			addr, count, err := pre.nextRecord()
			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
			index := pre.records - 1
			// The preimage file is keccak-ordered, which the reader checks on
			// this path, so the preimage store is written in that order.
			if keep {
				addrHash, err := order.record(hasher, index, addr)
				if err != nil {
					return err
				}
				if err := putPreimage(rawBatch, addrHash, addr.Bytes()); err != nil {
					return err
				}
			}
			var headerSlots uint64
			for left, first := count, true; first || left > 0; first = false {
				if len(batch.runs) == cap(batch.runs) || (left > 0 && len(batch.keys) == cap(batch.keys)) {
					if err := send(); err != nil {
						return err
					}
				}
				n := min(left, cap(batch.keys)-len(batch.keys))
				slots := batch.keys[len(batch.keys) : len(batch.keys)+n]
				batch.keys = batch.keys[:len(batch.keys)+n]
				if err := pre.readSlots(slots); err != nil {
					return err
				}
				for i := range slots {
					if sub, ok := bintrie.HeaderSlot((*[32]byte)(&slots[i])); ok {
						headerSlots |= 1 << sub
					}
				}
				if keep {
					err := order.slots(hasher, slots, func(slot, hash common.Hash) error {
						return putPreimage(rawBatch, hash, slot[:])
					})
					if err != nil {
						return err
					}
					if err := flush(false); err != nil {
						return err
					}
				}
				left -= n
				run := preimageRun{index: index, addr: addr, slots: slots, first: first, last: left == 0}
				if run.last {
					run.headerSlots = headerSlots
				}
				batch.runs = append(batch.runs, run)
			}
			preAccounts++
			prePreimages += 1 + uint64(count)
		}
		if len(batch.runs) > 0 {
			records <- batch
		}
		return nil
	}()
	if err := g.Wait(); err != nil {
		return fail(err)
	}
	if readErr != nil {
		return fail(readErr)
	}
	// An absorbed sorter holds nothing, so closing it releases no run.
	cand := cands[0]
	for _, c := range cands[1:] {
		if err := cand.Absorb(c); err != nil {
			return fail(err)
		}
		c.Close()
	}
	prog.finish(verified, "accounts", preAccounts, "preimages", prePreimages, "digest", pre.digest())
	return cand, nil
}

// putPreimage writes one preimage store entry under its keccak hash.
func putPreimage(w ethdb.KeyValueWriter, hash common.Hash, preimage []byte) error {
	return w.Put(append(append([]byte{}, rawdb.PreimagePrefix...), hash[:]...), preimage)
}

// preimageRun is a record of the preimage file, or a run of its slot keys: a
// batch holds at most preimageBatchSlots keys, so a record with more spans
// batches, a run in each, and is never held whole. first marks the run a
// record begins with, last the one it ends with.
type preimageRun struct {
	index       uint64 // the record's, in the file
	addr        common.Address
	slots       []common.Hash // a window of the batch's keys
	first, last bool
	headerSlots uint64 // the record's header-range slots, set on its last run
}

// preimageBatch is a stretch of the preimage file for a phase-1 worker. order
// is the order check's state where the stretch starts, which the worker
// checks the first run against: the record before or, if the run continues a
// record, that record's last slot key before. Batches cycle between the
// reader and the workers, so phase 1 holds a fixed number of slot keys,
// however many one record has.
type preimageBatch struct {
	order preimageOrder
	runs  []preimageRun
	keys  []common.Hash // the runs' slot keys, back to back
}

// preimageBatchSize and preimageBatchSlots bound a phase-1 batch: the records
// it begins or continues, and their slot keys (256 KiB). A batch is a few
// milliseconds of a worker's time, so a block waits no longer for the import
// to stand aside (blockYield).
var (
	preimageBatchSize  = 1024
	preimageBatchSlots = 1 << 13
)

// candidateBuf holds one candidate's key and value while it is built: the
// sorter copies them, so a worker reuses one buffer for every candidate.
type candidateBuf struct {
	key   [bintrie.StorageKeyLength]byte
	value [common.AddressLength + common.HashLength]byte
}

// addSlotCandidates derives the candidate tree keys a run of one record's
// slot keys stands for: one per overflow slot (see accountCandidates). The
// record's header candidate carries the header-range slots
// (addHeaderCandidate).
func addSlotCandidates(cand *bintrie.RecordSorter, buf *candidateBuf, addr common.Address, slots []common.Hash) error {
	c := accountCandidates{cand: cand, buf: buf, addr: addr}
	for _, slot := range slots {
		if err := c.slot(slot); err != nil {
			return err
		}
	}
	return nil
}

// addHeaderCandidate adds the candidate of addr's header stem: the address
// and a bitmap of the header-range slots its record claims.
func addHeaderCandidate(cand *bintrie.RecordSorter, buf *candidateBuf, addr common.Address, headerSlots uint64) error {
	key := bintrie.AppendHeaderStem(buf.key[:0], addr)
	value := binary.BigEndian.AppendUint64(append(buf.value[:0], addr[:]...), headerSlots)
	return cand.Add(key, value)
}

// accountCandidates derives one account's candidates as its slots stream
// past: one per overflow slot, carrying the address and the slot key. A
// header-range slot's number is its sub-index minus the offset, so it needs
// no candidate of its own - unlike an overflow slot, whose stem is a one-way
// hash of the slot number and can only be matched by deriving it forward.
// The self-converter's scan feeds it the keys it resolves; phase 1 the runs
// of a preimage record.
type accountCandidates struct {
	cand          *bintrie.RecordSorter
	buf           *candidateBuf
	addr          common.Address
	headerSlots   uint64
	storagePrefix [32]byte // blake3(address32), hashed once and reused for every overflow slot
	havePrefix    bool
}

// slot adds an overflow slot's candidate, or notes a header-range one.
func (c *accountCandidates) slot(slot common.Hash) error {
	inHeader, treeIndex, sub := bintrie.StorageIndex(slot[:])
	if inHeader {
		c.headerSlots |= 1 << (sub - bintrie.HeaderStorageOffset)
		return nil
	}
	if !c.havePrefix {
		a32 := bintrie.Address32(c.addr)
		c.storagePrefix, c.havePrefix = bintrie.KeyHash(a32[:]), true
	}
	key := append(bintrie.AppendStorageStem(c.buf.key[:0], c.addr, c.storagePrefix[:], &treeIndex), sub)
	value := append(append(c.buf.value[:0], c.addr[:]...), slot[:]...)
	return c.cand.Add(key, value)
}

// finish adds the header stem's candidate, once every slot is in.
func (c *accountCandidates) finish() error {
	return addHeaderCandidate(c.cand, c.buf, c.addr, c.headerSlots)
}

// leafBatchSize and leafBatches size the snapshot pipeline: leaves per batch,
// and batches in flight.
var (
	leafBatchSize = 4096
	leafBatches   = 8
)

// leafBatch carries snapshot leaves from the decoder to its two consumers,
// copied out of the reader's reused buffers.
type leafBatch struct {
	keys   []byte
	ends   []int // each key's end in keys
	values [][32]byte
	refs   atomic.Int32 // consumers yet to release it
}

func (b *leafBatch) add(key []byte, value [32]byte) {
	b.keys = append(b.keys, key...)
	b.ends = append(b.ends, len(b.keys))
	b.values = append(b.values, value)
}

func (b *leafBatch) key(i int) []byte {
	start := 0
	if i > 0 {
		start = b.ends[i-1]
	}
	return b.keys[start:b.ends[i]:b.ends[i]]
}

// leafReader yields leaves in tree-key order: a snapshot's, or a
// self-converter's own.
type leafReader interface {
	next() ([]byte, [32]byte, error)
}

// leafPipeline streams leaves from a decoder goroutine to a tree-building
// goroutine and to the caller, both in tree-key order.
type leafPipeline struct {
	leaves chan *leafBatch // the caller's
	free   chan *leafBatch
	cancel context.CancelFunc
	g      *errgroup.Group
}

// startLeafPipeline starts reading src and feeding builder; *builderErr is
// the error builder's node callback records. Both pause between batches while
// the node processes a block (y).
func startLeafPipeline(src leafReader, builder *bintrie.StackBuilder, builderErr *error, y *blockYield) *leafPipeline {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)
	p := &leafPipeline{
		leaves: make(chan *leafBatch, leafBatches),
		free:   make(chan *leafBatch, leafBatches),
		cancel: cancel,
		g:      g,
	}
	for range leafBatches {
		p.free <- new(leafBatch)
	}
	toBuilder := make(chan *leafBatch, leafBatches)
	g.Go(func() error {
		defer close(p.leaves)
		defer close(toBuilder)
		for eof := false; !eof; {
			if err := y.pause(ctx); err != nil {
				return err
			}
			var b *leafBatch
			select {
			case b = <-p.free:
			case <-ctx.Done():
				return ctx.Err()
			}
			b.keys, b.ends, b.values = b.keys[:0], b.ends[:0], b.values[:0]
			for len(b.values) < leafBatchSize {
				key, value, err := src.next()
				if err == io.EOF {
					eof = true
					break
				}
				if err != nil {
					return err
				}
				b.add(key, value)
			}
			if len(b.values) == 0 {
				return nil
			}
			b.refs.Store(2)
			for _, ch := range []chan *leafBatch{toBuilder, p.leaves} {
				select {
				case ch <- b:
				case <-ctx.Done():
					return ctx.Err()
				}
			}
		}
		return nil
	})
	g.Go(func() error {
		for b := range toBuilder {
			if err := y.pause(ctx); err != nil {
				return err
			}
			for i := range b.values {
				if err := builder.Add(b.key(i), b.values[i][:]); err != nil {
					return err
				}
				if *builderErr != nil {
					return *builderErr
				}
			}
			p.release(b)
		}
		return nil
	})
	return p
}

// release hands a batch back once both consumers are done with it.
func (p *leafPipeline) release(b *leafBatch) {
	if b.refs.Add(-1) == 0 {
		p.free <- b
	}
}

// wait returns once the decoder and the builder are done, with the first
// error either met.
func (p *leafPipeline) wait() error { return p.g.Wait() }

// stop cancels the pipeline and waits for it, for a caller leaving early.
func (p *leafPipeline) stop() {
	p.cancel()
	p.g.Wait()
}
