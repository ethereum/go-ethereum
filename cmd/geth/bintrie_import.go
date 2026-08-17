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
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"math"
	"math/bits"
	"slices"
	"strconv"
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
	"github.com/holiman/uint256"
	"github.com/urfave/cli/v2"
)

var (
	verifyOnlyFlag = &cli.BoolFlag{
		Name:  "verify-only",
		Usage: "Run both verification checks without writing anything",
	}

	bintrieImportCommand = &cli.Command{
		Name:      "import",
		Usage:     "Import a binary tree state from distribution artifacts",
		ArgsUsage: "<snapshot> <preimages> <anchor block-number-or-hash>",
		Action:    importBinaryTrie,
		Flags: slices.Concat([]cli.Flag{
			verifyOnlyFlag,
			forceConvertFlag,
			memoryLimitFlag,
			tmpDirFlag,
		}, utils.NetworkFlags, utils.DatabaseFlags),
		Description: `
geth bintrie import [flags] <snapshot> <preimages> <anchor block-number-or-hash>

Verifies and ingests an EIP-8347 PBT snapshot and preimage file: the tree is
rebuilt from the leaves against the artifact's claimed root, and the leaves
are re-derived into the merkle root the anchor block commits, resolved from
the local header chain - the artifacts never vouch for themselves. Code is
reassembled from its chunks and pinned to each code hash. A failed check
leaves nothing openable. --verify-only runs both checks and writes nothing.
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
		} else if err := wipeBinaryTrieState(chaindb, stack.ResolvePath("triedb")); err != nil {
			return fmt.Errorf("failed to wipe binary tree state: %w", err)
		}
	}
	budgetMB := ctx.Uint64(memoryLimitFlag.Name)
	if budgetMB > math.MaxInt>>20 {
		budgetMB = math.MaxInt >> 20
	}
	if _, err := importState(chaindb, importOptions{
		snapshot:   ctx.Args().Get(0),
		preimages:  ctx.Args().Get(1),
		anchor:     header,
		verifyOnly: verifyOnly,
		conversionOptions: conversionOptions{
			sortBudget: int(budgetMB << 20),
			tmpDir:     ctx.String(tmpDirFlag.Name),
		},
	}); err != nil {
		return err
	}
	// The namespace is written, but nothing reads it until the chain commits
	// its state with the tree; say so rather than leave it to be discovered.
	if !verifyOnly {
		if stored := rawdb.ReadChainConfig(chaindb, rawdb.ReadCanonicalHash(chaindb, 0)); stored == nil || !stored.IsPBT() {
			log.Warn("The chain config does not commit state with the binary tree; the imported state stays dormant until it does")
		}
	}
	return nil
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

// heldStream wraps a sorted record stream with one record of lookahead, the
// shape a merge-join needs.
type heldStream struct {
	stream     *bintrie.RecordStream
	key, value []byte
	done       bool
}

// current returns the held record, loading the next one if none is held.
func (h *heldStream) current() ([]byte, []byte, error) {
	if h.done || h.key != nil {
		return h.key, h.value, nil
	}
	key, value, err := h.stream.Next()
	if err == io.EOF {
		h.done = true
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	h.key, h.value = key, value
	return key, value, nil
}

// advance drops the held record.
func (h *heldStream) advance() { h.key, h.value = nil, nil }

// preimageWriter batches preimage-store writes, or discards them when nil.
type preimageWriter struct {
	batch ethdb.Batch
	buf   map[common.Hash][]byte
}

func (pw *preimageWriter) add(hash common.Hash, preimage []byte) {
	if pw == nil {
		return
	}
	pw.buf[hash] = common.CopyBytes(preimage)
	if len(pw.buf) >= 1024 {
		rawdb.WritePreimages(pw.batch, pw.buf)
		pw.buf = make(map[common.Hash][]byte, 1024)
	}
}

func (pw *preimageWriter) flush() {
	if pw == nil || len(pw.buf) == 0 {
		return
	}
	rawdb.WritePreimages(pw.batch, pw.buf)
	pw.buf = make(map[common.Hash][]byte)
}

// importGroup carries one account's header-stem leaves through the join.
type importGroup struct {
	stem        []byte
	addr        common.Address
	basic       *[32]byte
	codeHash    *[32]byte
	delegation  *[32]byte
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
// two commands share.
type importOptions struct {
	snapshot   string
	preimages  string
	anchor     *types.Header
	verifyOnly bool
	conversionOptions
}

// importState verifies and ingests the artifacts, returning the imported
// root. With verifyOnly the database is never touched.
func importState(chaindb ethdb.Database, opts importOptions) (common.Hash, error) {
	var (
		anchorRoot   = opts.anchor.Root
		snapshotPath = opts.snapshot
		preimagePath = opts.preimages
		verifyOnly   = opts.verifyOnly
	)
	pbtdb := rawdb.NewTable(chaindb, string(rawdb.PBTPrefix))
	if !verifyOnly {
		if dirty, err := hasBinaryTrieState(chaindb); err != nil {
			return common.Hash{}, err
		} else if dirty {
			return common.Hash{}, errors.New("binary tree namespace already holds state, whether a finished import or the debris of one; re-run with --force to wipe it")
		}
	}
	// Five sorters live across the pipeline - the candidates, the accounts,
	// the slots, the chunks and the code stems - and a sealed one keeps its
	// buffer while its stream drains, so the budget is split five ways rather
	// than sized for one at a time.
	quarter := opts.sortBudget / 5

	// Open both artifacts before the expensive phase: a mistyped path should
	// not cost a full pass over the other file.
	pre, err := openPreimages(preimagePath)
	if err != nil {
		return common.Hash{}, err
	}
	defer pre.close()

	snap, err := openSnapshot(snapshotPath)
	if err != nil {
		return common.Hash{}, err
	}
	defer snap.close()
	if snap.count == 0 {
		return common.Hash{}, errors.New("refusing to import an empty snapshot")
	}
	// Phase 1: derive every candidate tree key the preimages can stand for.

	cand := bintrie.NewRecordSorter(opts.tmpDir, quarter, nil)
	defer cand.Close() // a second Close is a no-op; this one covers the error paths
	var (
		start        = time.Now()
		preAccounts  uint64
		prePreimages uint64
	)
	for {
		addr, slots, err := pre.next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return common.Hash{}, err
		}
		// One candidate per header stem, carrying the address and a bitmap of
		// the header-range slots this record claims. A header-range slot's
		// number is its sub-index minus the offset, so it needs no candidate
		// of its own - unlike an overflow slot, whose stem is a one-way hash
		// of the slot number and can only be matched by deriving it forward.
		var headerSlots uint64
		for _, slot := range slots {
			if inHeader, _, sub := bintrie.StorageIndex(slot[:]); inHeader {
				headerSlots |= 1 << (sub - bintrie.HeaderStorageOffset)
				continue
			}
			if err := cand.Add(bintrie.StorageSlotKey(addr, slot[:]), append(addr.Bytes(), slot[:]...)); err != nil {
				return common.Hash{}, err
			}
		}
		value := binary.BigEndian.AppendUint64(addr.Bytes(), headerSlots)
		if err := cand.Add(bintrie.HeaderStem(addr), value); err != nil {
			return common.Hash{}, err
		}
		preAccounts++
		prePreimages += 1 + uint64(len(slots))
		if preAccounts%50_000 == 0 {
			log.Info("Indexing preimage file", "accounts", preAccounts, "preimages", prePreimages,
				"elapsed", common.PrettyDuration(time.Since(start)))
		}
	}
	log.Info("Indexed preimage file", "accounts", preAccounts, "preimages", prePreimages,
		"digest", pre.digest(), "elapsed", common.PrettyDuration(time.Since(start)))

	// Phase 2: stream the snapshot; rebuild the tree; join every account and
	// storage leaf to its preimage candidate; collect the MPT re-derivation
	// inputs.
	candStream, err := cand.Sort()
	if err != nil {
		return common.Hash{}, err
	}
	held := &heldStream{stream: candStream}

	var (
		pbtBatch ethdb.Batch // tree nodes and flat state, inside the namespace
		rawBatch ethdb.Batch // code and preimages, outside it
		preims   *preimageWriter
	)
	if !verifyOnly {
		pbtBatch = pbtdb.NewBatch()
		rawBatch = chaindb.NewBatch()
		preims = &preimageWriter{batch: rawBatch, buf: make(map[common.Hash][]byte, 1024)}
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

	var (
		builderErr error
		onNode     func(path []byte, hash common.Hash, blob []byte)
	)
	if !verifyOnly {
		onNode = func(path []byte, hash common.Hash, blob []byte) {
			if builderErr != nil {
				return
			}
			rawdb.WriteAccountTrieNode(pbtBatch, path, blob)
			if err := flush(false); err != nil {
				builderErr = err
			}
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

	// addSlot records one storage slot everywhere it goes: flat state, the
	// preimage store, and the MPT re-derivation.
	addSlot := func(accountHash common.Hash, slot common.Hash, value [32]byte) error {
		var (
			slotHash = crypto.Keccak256Hash(slot[:])
			enc, _   = rlp.EncodeToBytes(common.TrimLeftZeroes(value[:]))
		)
		if pbtBatch != nil {
			rawdb.WriteStorageSnapshot(pbtBatch, accountHash, slotHash, enc)
		}
		preims.add(slotHash, slot[:])
		stats.slots++
		return slotSorter.Add(append(accountHash.Bytes(), slotHash.Bytes()...), enc)
	}

	// sealGroup validates one account's header leaves and records the
	// account everywhere it goes.
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
		var (
			nonce    uint64
			balance  = new(uint256.Int)
			codeSize uint32
			version  byte
		)
		if g.basic != nil {
			version, codeSize, nonce, balance = bintrie.DecodeBasicData(g.basic[:])
			if version != 0 {
				return fmt.Errorf("account %x carries basic-data version %d, must be 0", g.addr, version)
			}
			// Re-encode rather than check field by field: the leaf's reserved
			// bytes take part in no check but the tree's own hash, and the
			// claimed root is the artifact's to choose, so anything the
			// decoder ignores would ride through both checks and leave a root
			// no honest converter produces.
			want, err := bintrie.EncodeBasicData(codeSize, nonce, balance)
			if err != nil {
				return fmt.Errorf("account %x: %w", g.addr, err)
			}
			if want != *g.basic {
				return fmt.Errorf("account %x basic-data leaf is not the canonical encoding of its fields", g.addr)
			}
		}
		var codeHash common.Hash
		switch {
		case g.delegation != nil && g.codeHash != nil:
			return fmt.Errorf("account %x holds both a code-hash and a delegation leaf", g.addr)
		case g.delegation != nil:
			// GetAccount's rules: the designator is the leading code_size
			// bytes, and a zero size is malformed.
			if codeSize != 23 {
				return fmt.Errorf("account %x holds a delegation with code size %d, must be 23", g.addr, codeSize)
			}
			designator := g.delegation[:23]
			if _, ok := types.ParseDelegation(designator); !ok {
				return fmt.Errorf("account %x holds a malformed delegation leaf", g.addr)
			}
			// The nine bytes after the designator are padding, and nothing
			// downstream reads them; see the basic-data note above.
			if *g.delegation != [32]byte(bintrie.EncodeDelegation(designator)) {
				return fmt.Errorf("account %x delegation leaf is not the canonical encoding of its designator", g.addr)
			}
			codeHash = crypto.Keccak256Hash(designator)
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
		default:
			return fmt.Errorf("account %x holds neither a code-hash nor a delegation leaf", g.addr)
		}
		accountHash := crypto.Keccak256Hash(g.addr.Bytes())
		if pbtBatch != nil {
			rawdb.WriteAccountSnapshot(pbtBatch, accountHash, types.SlimAccountRLP(types.StateAccount{
				Nonce:    nonce,
				Balance:  balance,
				Root:     types.EmptyRootHash,
				CodeHash: codeHash.Bytes(),
			}))
		}
		preims.add(accountHash, g.addr.Bytes())
		stats.accounts++

		value := make([]byte, 0, 72)
		value = binary.BigEndian.AppendUint64(value, nonce)
		balance32 := balance.Bytes32()
		value = append(value, balance32[:]...)
		value = append(value, codeHash.Bytes()...)
		if err := acctSorter.Add(accountHash.Bytes(), value); err != nil {
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
		group        *importGroup
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

	for {
		key, value, err := snap.next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return common.Hash{}, err
		}
		if err := builder.Add(key, value[:]); err != nil {
			return common.Hash{}, err
		}
		if builderErr != nil {
			return common.Hash{}, builderErr
		}
		stats.leaves++
		stats.report(false)

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
						codeHash = common.BytesToHash(cvalue[:32])
						ti       = binary.BigEndian.Uint64(cvalue[32:])
						index    = ti*256 + uint64(key[33])
					)
					record := append(codeHash.Bytes(), binary.BigEndian.AppendUint64(nil, index)...)
					if err := chunkSorter.Add(record, value[:]); err != nil {
						return common.Hash{}, err
					}
					break
				}
				codeHeld.advance() // a fully zero-chunk stem: legitimate
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
			accountHash := crypto.Keccak256Hash(addr.Bytes())
			slot := common.BytesToHash(matched[common.AddressLength:])
			if err := addSlot(accountHash, slot, value); err != nil {
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
			group = &importGroup{
				stem:        bytes.Clone(stem),
				addr:        addr,
				headerClaim: binary.BigEndian.Uint64(matched[common.AddressLength:]),
			}
		}
		leaf := value
		switch {
		case sub == bintrie.BasicDataLeafKey:
			group.basic = &leaf
		case sub == bintrie.CodeHashLeafKey:
			group.codeHash = &leaf
		case sub == bintrie.DelegationLeafKey:
			group.delegation = &leaf
		case sub >= bintrie.HeaderStorageOffset && sub < bintrie.HeaderStorageOffset+bintrie.HeaderStorageSlots:
			// The slot number is the sub-index, not something the preimage
			// file has to be trusted for.
			var slot common.Hash
			slot[31] = sub - bintrie.HeaderStorageOffset
			group.slots = append(group.slots, importSlot{slot: slot, value: leaf})
			group.headerSeen |= 1 << (sub - bintrie.HeaderStorageOffset)
		default:
			return common.Hash{}, fmt.Errorf("account leaf %x sits at reserved sub-index %d", key, sub)
		}
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

	// Check 1: the leaves must rebuild to the claimed root.
	rebuilt := builder.Finish()
	if builderErr != nil {
		return common.Hash{}, builderErr
	}
	if rebuilt != snap.root {
		return common.Hash{}, fmt.Errorf("snapshot leaves rebuild to %x, its header claims %x", rebuilt, snap.root)
	}
	log.Info("Verified snapshot consistency", "root", rebuilt, "leaves", stats.leaves, "digest", snap.digest())

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
	for _, hash := range hashes {
		var chunks []bintrie.IndexedChunk
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
		if rawBatch != nil {
			rawdb.WriteCode(rawBatch, hash, code)
			if err := flush(false); err != nil {
				return common.Hash{}, err
			}
		}
		stats.codes++
	}
	if ckey, _, err := chunkHeld.current(); err != nil {
		return common.Hash{}, err
	} else if ckey != nil {
		return common.Hash{}, fmt.Errorf("code chunks left over for unclaimed hash %x", ckey[:32])
	}
	log.Info("Verified code against its hashes", "codes", stats.codes)

	// Check 2: re-derive the MPT and demand the anchor root.
	acctStream, err := acctSorter.Sort()
	if err != nil {
		return common.Hash{}, err
	}
	slotStream, err := slotSorter.Sort()
	if err != nil {
		return common.Hash{}, err
	}
	var (
		slotHeld    = &heldStream{stream: slotStream}
		accountTrie = trie.NewStackTrie(nil)
		storageTrie = trie.NewStackTrie(nil)
		rederived   uint64
	)
	for {
		akey, avalue, err := acctStream.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return common.Hash{}, err
		}
		storageTrie.Reset()
		storageRoot := types.EmptyRootHash
		hasStorage := false
		for {
			skey, svalue, err := slotHeld.current()
			if err != nil {
				return common.Hash{}, err
			}
			if skey == nil {
				break
			}
			switch bytes.Compare(skey[:32], akey) {
			case -1:
				return common.Hash{}, fmt.Errorf("storage under account hash %x, which holds no account", skey[:32])
			case 1:
			default:
				if err := storageTrie.Update(skey[32:], svalue); err != nil {
					return common.Hash{}, err
				}
				hasStorage = true
				slotHeld.advance()
				continue
			}
			break
		}
		if hasStorage {
			storageRoot = storageTrie.Hash()
		}
		full, err := rlp.EncodeToBytes(&types.StateAccount{
			Nonce:    binary.BigEndian.Uint64(avalue[:8]),
			Balance:  new(uint256.Int).SetBytes(avalue[8:40]),
			Root:     storageRoot,
			CodeHash: common.CopyBytes(avalue[40:72]),
		})
		if err != nil {
			return common.Hash{}, err
		}
		if err := accountTrie.Update(akey, full); err != nil {
			return common.Hash{}, err
		}
		rederived++
		if rederived%100_000 == 0 {
			log.Info("Re-deriving merkle state", "accounts", rederived,
				"elapsed", common.PrettyDuration(time.Since(stats.start)))
		}
	}
	if skey, _, err := slotHeld.current(); err != nil {
		return common.Hash{}, err
	} else if skey != nil {
		return common.Hash{}, fmt.Errorf("storage under account hash %x, which holds no account", skey[:32])
	}
	if got := accountTrie.Hash(); got != anchorRoot {
		return common.Hash{}, fmt.Errorf("the leaves re-derive merkle root %x, the anchor commits %x", got, anchorRoot)
	}
	log.Info("Verified consensus anchoring", "stateRoot", anchorRoot, "accounts", stats.accounts, "slots", stats.slots)

	if verifyOnly {
		log.Info("Verification complete, nothing written", "binaryRoot", snap.root,
			"snapshotDigest", snap.digest(), "preimageDigest", pre.digest())
		return snap.root, nil
	}
	preims.flush()
	if err := flush(true); err != nil {
		return common.Hash{}, err
	}
	// The anchor rides with the root: the tree does not say which block it
	// commits, and catching up from here has to start somewhere. The
	// attestation is the completion marker and comes last; the imported tree
	// bases an empty history, like a conversion.
	rawdb.WriteSnapshotRoot(pbtdb, snap.root)
	rawdb.WritePBTAnchor(pbtdb, opts.anchor.Number.Uint64(), opts.anchor.Hash())
	rawdb.WritePBTFlatState(pbtdb)
	log.Info("Import complete", "binaryRoot", snap.root, "anchor", opts.anchor.Number,
		"snapshotDigest", snap.digest(), "preimageDigest", pre.digest())
	return snap.root, nil
}
