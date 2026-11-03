// Copyright 2026-2027 The quarkchain Authors
// This file is part of the quarkchain library.
//
// The quarkchain library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The quarkchain library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the quarkchain library. If not, see <http://www.gnu.org/licenses/>.

package shard

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/miner"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/qkc/rawdb"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
)

var (
	// ErrClusterStateInconsistency marks failures of guarantees that other cluster
	// components must establish before calling ShardChain. These defensive
	// checks should never fail here; a failure means an upstream guarantee was
	// violated. They include pyquarkchain's check assertions and the requirement
	// that confirmed local blocks be available before importing a root block.
	// Keep the specific cause wrapped alongside it for diagnosis in logs.
	ErrClusterStateInconsistency = errors.New("cluster state inconsistency")

	// ErrShardStateInconsistency marks missing or inconsistent local data that was
	// already validated when stored. Incoming validation errors do not use it.
	ErrShardStateInconsistency = errors.New("shard state inconsistency")

	// ErrRootChainUninitialized is returned when no root tip is known yet.
	ErrRootChainUninitialized = errors.New("root chain not initialized")

	// ErrRootChainAlreadyInitialized is returned when Init is called after a
	// root tip has already been selected.
	ErrRootChainAlreadyInitialized = errors.New("root chain already initialized")

	// ErrUnknownRootBlock is returned when a required root block is not stored
	// locally.
	ErrUnknownRootBlock = errors.New("unknown root block")

	// ErrRootNotDescendant is returned when the root block referenced by a shard
	// block does not extend the one referenced by its parent.
	ErrRootNotDescendant = errors.New("previous root block does not extend the one of the parent")

	// ErrNotConfirmedDescendant is returned when a shard block does not extend
	// the latest shard block confirmed by the root block it references.
	ErrNotConfirmedDescendant = errors.New("block does not extend the last confirmed block")

	// ErrInvalidRootBlock is returned when a root block has invalid genesis
	// contents or lacks the required confirmed shard checkpoint.
	ErrInvalidRootBlock = errors.New("invalid root block")

	// ErrUnknownConfirmedBlock is returned when the root chain confirms a shard
	// block that is not in the local chain.
	ErrUnknownConfirmedBlock = errors.New("unknown confirmed block")

	// ErrRootNotAfterGenesis is returned when a normal root import has block
	// number zero. Root genesis must be handled by Init.
	ErrRootNotAfterGenesis = errors.New("root block number must be greater than zero")

	// ErrRootBlockVersion is returned when a root block has an unsupported
	// version.
	ErrRootBlockVersion = errors.New("unsupported root block version")

	// ErrTooManyConfirmedBlocks is returned when a root block confirms more
	// blocks of the shard than MaxBlocksPerRootBlock.
	ErrTooManyConfirmedBlocks = errors.New("too many shard blocks in the root block")

	// ErrConfirmedNotContiguous is returned when the first block of the shard
	// confirmed by a root block does not follow the latest one confirmed by its
	// parent.
	ErrConfirmedNotContiguous = errors.New("confirmed blocks do not follow the last confirmed block")

	// ErrConfirmedRootNotAncestor is returned when the latest shard block
	// confirmed by a root block references a root block that is not one of its
	// ancestors.
	ErrConfirmedRootNotAncestor = errors.New("last confirmed block references a root block off the root chain")

	// ErrMissingXShardDeposits is returned when a root block confirms a block of
	// another shard whose cross-shard deposits to this shard are not stored.
	ErrMissingXShardDeposits = errors.New("missing cross-shard deposits")

	// ErrUnexpectedXShardDeposits is returned when cross-shard deposits are
	// stored for a block of another shard that sends none to this shard.
	ErrUnexpectedXShardDeposits = errors.New("unexpected cross-shard deposits")

	errNotQuarkChain = errors.New("not a QuarkChain shard chain")
)

// ShardChain is the consensus layer of a shard. It imports shard blocks into
// the underlying chain under the root chain rules, imports the root blocks, and
// selects the canonical shard head.
//
// ShardChain exclusively manages the underlying chain's canonical head and
// canonical hash mappings. Callers must not independently change the head or
// import canonical headers through core.BlockChain.
type ShardChain struct {
	db      ethdb.KeyValueStore // Database of the root blocks seen by the shard
	chain   *core.BlockChain
	builder *miner.Miner // The node's shared builder; nil for an import-only shard

	// NewShardChain creates a shard chain on top of the given chain, reading the
	// root blocks and incoming deposits from db. For a QuarkChain configuration,
	// construction and validation share the inbox backed by this database.
	rootTip *qkctypes.RootBlockHeader // Nil until Init succeeds; updated by root imports
	mu      sync.Mutex                // Serializes block imports and shard/root tip selection and updates
}

// NewShardChain binds the node's existing blockchain and builder, reading root
// blocks and incoming deposits from db. The builder must use this blockchain
// and its engine; nil supports import-only shards. Call Init before use.
func NewShardChain(db ethdb.KeyValueStore, chain *core.BlockChain, builder *miner.Miner) *ShardChain {
	c := &ShardChain{db: db, chain: chain, builder: builder}
	if config := chain.Config().QuarkChain; config != nil {
		chain.SetXShardInbox(NewXShardInbox(config, c))
	}
	return c
}

// Init sets the shard head and root tip from a root checkpoint.
//
// For root genesis, R0 is sent by the master and carries the shards' genesis
// headers. Check that it contains exactly one header for this shard, matching
// the existing shard genesis G0. G0.PrevRootBlock must be empty to avoid a
// circular hash dependency between G0 and R0. Store R0 and its confirmation
// of G0 in one batch; no stored root is required.
//
// For a non-genesis root, its header and shard confirmation must already be
// stored locally. Restore the shard head to the stored confirmation.
//
// Set the canonical shard head before publishing rootTip. Until initialization
// succeeds, errors leave rootTip nil and allow retry. Init does not stop the blockchain.
//
// The shard service handles master connections, missing-data synchronization
// and genesis creation. It can start mining after Init succeeds.
func (c *ShardChain) Init(root *qkctypes.RootBlock) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Step 1: reject repeat initialization.
	if c.rootTip != nil {
		return ErrRootChainAlreadyInitialized
	}
	if root == nil {
		return fmt.Errorf("%w: nil root block", ErrUnknownRootBlock)
	}

	var rootHeader *qkctypes.RootBlockHeader
	var confirmed *types.Header
	if root.Number() == 0 {
		// Step 2a: validate the master's genesis body against the local genesis.
		var err error
		confirmed, err = c.validateGenesisRoot(root)
		if err != nil {
			return err
		}
		rootHeader = root.Header()

		// Persist the root and its non-nil confirmation atomically. On a later
		// head failure, the same validated bootstrap can safely be retried.
		batch := c.db.NewBatch()
		defer batch.Close()
		rawdb.WriteRootBlock(batch, root)
		rawdb.WriteLastConfirmedBlockHash(batch, root.Hash(), confirmed.Hash())
		if err := batch.Write(); err != nil {
			return fmt.Errorf("store genesis root %x: %w", root.Hash(), err)
		}
	} else {
		// Step 2b: recover only locally stored checkpoints. The received body
		// is not a source of recovery data.
		rootHeader = rawdb.ReadRootBlockHeader(c.db, root.Hash())
		if rootHeader == nil {
			return fmt.Errorf("%w: %w: %x", ErrClusterStateInconsistency, ErrUnknownRootBlock, root.Hash())
		}
		var err error
		confirmed, err = c.lastConfirmed(rootHeader)
		if err != nil {
			return err
		}
	}

	// Step 3: restore the confirmed head and publish rootTip after success.
	if err := c.setHeadForRootTip(rootHeader, confirmed); err != nil {
		return err
	}
	c.rootTip = rootHeader
	// TODO: wire Init into the shard service's master handshake, data sync and
	// mining startup. Genesis construction/state setup stays in that service.
	return nil
}

// RootTip returns the root tip, or nil before Init. It waits for an import in
// progress.
func (c *ShardChain) RootTip() *qkctypes.RootBlockHeader {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.rootTip
}

// validateGenesisRoot verifies the body commitment and this shard's sole
// genesis header. Other shards' genesis hashes and heights are their own
// responsibility; this shard only decodes their shard IDs to skip them.
func (c *ShardChain) validateGenesisRoot(root *qkctypes.RootBlock) (*types.Header, error) {
	config := c.chain.Config().QuarkChain
	if config == nil {
		return nil, errNotQuarkChain
	}
	if root.Number() != 0 || root.ParentHash() != (common.Hash{}) {
		return nil, fmt.Errorf("%w: expected genesis root", ErrInvalidRootBlock)
	}
	if root.ShardHeaderHash() != qkctypes.CalcShardHeaderHash(root.ShardBlockHeaders()) {
		return nil, fmt.Errorf("%w: genesis body commitment mismatch", ErrInvalidRootBlock)
	}
	genesis := c.chain.Genesis().Header()
	extra, err := genesis.QKCExtra()
	if err != nil {
		return nil, fmt.Errorf("%w: genesis: %w", ErrShardStateInconsistency, err)
	}
	if extra.ShardID != config.ShardID || extra.PrevRootBlock != (common.Hash{}) {
		return nil, fmt.Errorf("%w: genesis must have the local shard ID and an empty previous root", ErrShardStateInconsistency)
	}
	found := false
	for _, header := range root.ShardBlockHeaders() {
		extra, err := header.QKCExtra()
		if err != nil {
			return nil, fmt.Errorf("%w: genesis shard header: %w", ErrInvalidRootBlock, err)
		}
		if extra.ShardID != config.ShardID {
			continue
		}
		if found || header.Number.Sign() != 0 || header.Hash() != genesis.Hash() {
			return nil, fmt.Errorf("%w: expected exactly one matching genesis for shard %d", ErrInvalidRootBlock, config.ShardID)
		}
		found = true
	}
	if !found {
		return nil, fmt.Errorf("%w: missing genesis for shard %d", ErrInvalidRootBlock, config.ShardID)
	}
	return genesis, nil
}

// InsertShardBlock validates a block against the root chain, imports it, and
// makes it the canonical head if the fork choice rule selects it. Known blocks
// are ignored.
func (c *ShardChain) InsertShardBlock(block *types.Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Shard genesis is already committed before Init; it is not imported here.
	if block.NumberU64() == 0 {
		return consensus.ErrInvalidNumber
	}

	// TODO 1. check shard id here 2. check future block in engine.VerifyBlockHeader

	if c.chain.HasBlock(block.Hash(), block.NumberU64()) {
		return nil
	}
	rootTip := c.rootTip
	if rootTip == nil {
		return ErrRootChainUninitialized
	}
	parent := c.chain.GetHeader(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	prevRoot, err := c.validateRootReference(block.Header(), parent)
	if err != nil {
		return err
	}

	updateHead, err := c.shouldUpdateHead(block.Header(), prevRoot, rootTip)
	if err != nil {
		return err
	}
	// post cross shard execution will be validated in InsertBlockWithoutSetHead
	if _, err := c.chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		return err
	}
	if updateHead {
		if _, err := c.chain.SetCanonical(block); err != nil {
			return fmt.Errorf("set canonical head %x: %w", block.Hash(), err)
		}
	}
	return nil
}

// validateRootReference checks the root block referenced by a shard block, its
// previous root block, against the shard and root chains, and returns it. It
// mirrors the root checks of ShardState.validate_block in pyquarkchain, whose
// error messages are quoted below. Two rules apply.
//
// Rule 1: a shard block must reference the same root block as its parent,
// or a descendant of that root block. The shard chain's view of the root chain
// can only stay unchanged or move forward. Skip this comparison when the parent
// is shard genesis, because genesis has an empty PrevRootBlock.
//
//	              +----- R1'
//	              v
//	root chain    R0 <-- R1 <-- R2 <-- R3
//	                     ^
//	                     |
//	shard chain   ... <- P <--- B
//
//	P references R1, so B may reference R1, R2 or R3, but neither R0, which is
//	older, nor R1', which is on another root chain.
//
// A fork after R1 passes this rule, as it still extends R1. Whether the block
// then becomes the head is up to the fork choice, see shouldUpdateHead.
//
// Rule 2: let C be the latest shard block confirmed up to the referenced root.
// The candidate must extend C, so its parent must be C or a descendant of C.
//
//	root chain    R0 <-- R1
//	                     :  confirms up to C
//	                     v
//	shard chain   G <--- C <-- P <-- B     B extends C: valid
//	              ^
//	              +--- P' <-- B'           B' forks before C: invalid
//
//	B and B' both reference R1.
func (c *ShardChain) validateRootReference(header, parent *types.Header) (*qkctypes.RootBlockHeader, error) {
	// The block's referenced root must be known ("cannot find root block for
	// the minor block"). A non-genesis parent's reference was checked on import.
	prevRoot, err := c.prevRootBlock(header)
	if err != nil {
		return nil, err
	}
	// G0 has no previous-root reference to compare.
	if parent.Number.Sign() != 0 {
		parentPrevRoot, err := c.prevRootBlock(parent)
		if err != nil {
			return nil, fmt.Errorf("%w: parent: %w", ErrShardStateInconsistency, err)
		}
		// Rule 1. pyquarkchain checks the heights first, which is implied here, as
		// a root block is never lower than its ancestors.
		if !c.isRootDescendant(prevRoot, parentPrevRoot) {
			return nil, fmt.Errorf("%w: root #%d [%x..] after #%d [%x..]", ErrRootNotDescendant,
				prevRoot.Number, prevRoot.Hash().Bytes()[:4], parentPrevRoot.Number, parentPrevRoot.Hash().Bytes()[:4])
		}
	}
	// Rule 2. Initialization establishes a confirmed baseline, and root
	// imports carry it forward even when they confirm no additional blocks.
	confirmed, err := c.lastConfirmed(prevRoot)
	if err != nil {
		return nil, err
	}
	// The block itself is not stored yet, so check its parent instead: the
	// block extends C if and only if its parent is C or extends it.
	if !c.isShardDescendant(parent, confirmed) {
		return nil, fmt.Errorf("%w: confirmed #%d [%x..]", ErrNotConfirmedDescendant,
			confirmed.Number, confirmed.Hash().Bytes()[:4])
	}
	return prevRoot, nil
}

// shouldUpdateHead reports whether a new shard block should become the head.
// It follows ShardState.add_block in pyquarkchain. Apply the rules below in order.
//
//  1. The block must reference the root tip or one of its ancestors. A block
//     referencing another root chain, like R1' below, is never selected.
//  2. A block extending the head is selected.
//  3. Otherwise the block is on a fork, which must extend C, the latest shard
//     block confirmed by the root tip.
//  4. The higher of the fork and the head wins. At the same height, the fork
//     wins if it references a higher root block than the head, as it has seen
//     more of the root chain.
//
// For example, with R2 as the root tip and H as the head:
//
//	              +----- R1'
//	              v
//	root chain    R0 <-- R1 <-- R2
//	                            :  confirms up to C
//	                            v
//	shard chain   G <-- ... <-- C <-- H
//	              ^             ^
//	              |             +---- X <-- Y
//	              +---- F <-- ... <-- Z
//
//	A child of H is selected (rule 2). Y, higher than H, is selected, and so
//	is X, as high as H, if it references a higher root block than H does
//	(rule 4). Z is never selected, however high, as its fork does not extend
//	C (rule 3).
//
// As in pyquarkchain, forks are compared by height, not by difficulty. Errors
// reading local chain data are returned instead of silently rejecting the fork.
func (c *ShardChain) shouldUpdateHead(header *types.Header, prevRoot, rootTip *qkctypes.RootBlockHeader) (bool, error) {
	// Rule 1: "Don't update tip if the block depends on a root block that is
	// not root_tip or root_tip's ancestor".
	if !c.isRootDescendant(rootTip, prevRoot) {
		return false, nil
	}
	// Rule 2. The head always extends C, so the block does too. The root tip
	// update must keep it so, by moving the head onto the shard chain confirmed
	// by the new root tip.
	current := c.chain.CurrentBlock()
	if header.ParentHash == current.Hash() {
		return true, nil
	}
	// Rule 3, as __is_minor_block_linked_to_root_tip: "Determine whether a
	// minor block is a descendant of a minor block confirmed by root tip".
	// Every initialized root has a confirmed baseline. pyquarkchain first
	// checks that the block is higher than C, which is implied here: the block
	// is new, so it is not C, and it only extends C if it is higher.
	confirmed, err := c.lastConfirmed(rootTip)
	if err != nil {
		return false, err
	}
	if !c.isShardDescendant(header, confirmed) {
		return false, nil
	}
	// Rule 4. On a full tie, the head stays.
	if header.Number.Uint64() != current.Number.Uint64() {
		return header.Number.Uint64() > current.Number.Uint64(), nil
	}

	// At equal shard height, prefer the block with the higher root reference.
	currentRoot, err := c.prevRootBlock(current)
	if err != nil {
		return false, fmt.Errorf("%w: current head: %w", ErrShardStateInconsistency, err)
	}
	return prevRoot.Number > currentRoot.Number, nil
}

// InsertRootBlock validates a root block against the shard chain, stores it,
// and makes it the root tip if the fork choice rule selects it, moving the
// canonical head onto the shard chain it confirms. It reports whether the root
// tip changed. Known root blocks are ignored. Head and root writes are not one
// transaction; a failed root-batch write may leave the head changed. Subsequent
// root imports or service initialization may restore consistency.
// It mirrors ShardState.add_root_block in pyquarkchain.
//
// Unlike shard blocks, root blocks are chosen by total difficulty: "Switch to
// the root block with higher total diff". On a tie, the root tip stays.
func (c *ShardChain) InsertRootBlock(root *qkctypes.RootBlock) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	rootTip := c.rootTip
	if rootTip == nil {
		return false, ErrRootChainUninitialized
	}
	if rawdb.HasRootBlock(c.db, root.Hash()) {
		return false, nil
	}
	// Step 1: validate the root and find its latest confirmed shard block.
	confirmed, err := c.validateRootBlock(root)
	if err != nil {
		return false, err
	}
	// Step 2: the fork choice.
	switched := root.TotalDifficulty().Cmp(&rootTip.TotalDifficulty) > 0
	// Step 3: reconcile the shard head before storing the root.
	previousHead := c.chain.CurrentBlock().Hash()
	if switched {
		if err := c.setHeadForRootTip(root.Header(), confirmed); err != nil {
			log.Error("Failed to set shard head for root block", "root", root.Hash(), "rootTip", rootTip.Hash(), "confirmed", confirmed.Hash(), "previousHead", previousHead, "head", c.chain.CurrentBlock().Hash(), "err", err)
			return false, err
		}
	}
	// Step 4: store the root and its confirmation together, then publish the tip.
	batch := c.db.NewBatch()
	defer batch.Close()
	rawdb.WriteRootBlock(batch, root)
	rawdb.WriteLastConfirmedBlockHash(batch, root.Hash(), confirmed.Hash())
	if err := batch.Write(); err != nil {
		// TODO: Handle the edge case where the shard head switches successfully but
		// the root batch fails. For now, log and return the error. Subsequent root
		// imports, including re-importing the failed root, may restore consistency.
		// If normal imports cannot resume, restart the service and recover through
		// Init using a valid root checkpoint already stored locally.
		log.Error("Failed to store root block", "root", root.Hash(), "rootTip", rootTip.Hash(), "confirmed", confirmed.Hash(), "previousHead", previousHead, "head", c.chain.CurrentBlock().Hash(), "err", err)
		return false, fmt.Errorf("store root block %x: %w", root.Hash(), err)
	}
	if switched {
		c.rootTip = root.Header()
	}
	return switched, nil
}

// validateRootBlock checks a root block against the shard and root chains, and
// returns C, the latest shard block confirmed by the root chain up to it.
// A root carrying no new local blocks inherits its parent's non-nil confirmation.
// It mirrors the shard-specific checks of ShardState.add_root_block in
// pyquarkchain. The master must validate the root
// block and its body before forwarding it to shards. Each shard validates its
// minor blocks and distributes outgoing cross-shard deposits. This method checks
// availability and consistency of local shard data.
// Failures of these cluster guarantees wrap ErrClusterStateInconsistency;
// missing already-validated local data wraps ErrShardStateInconsistency.
// Ordinary input validation errors keep their specific error classification.
//
// Rule 1: the master must validate root consensus, including proof of work
// and total difficulty. Root genesis is handled by Init. Normal root imports
// must have a non-zero block number and a parent already stored locally.
//
// Rule 2: the master must validate the commitment to the carried shard headers
// and their complete ordering/continuity. Locally, blocks for this shard must
// already be present ("cannot find minor block in local shard"), and their count
// must respect MaxBlocksPerRootBlock ("too many minor blocks in the root block").
// The cluster must ensure confirmed local blocks are available before importing
// this root block. The source shards must also have delivered all required
// cross-shard deposits. Missing confirmed local blocks or missing/unexpected
// deposit lists violate these cluster guarantees ("cannot find x_shard tx list"),
// see validateXShardDeposits.
//
// Rule 3: locally, the first carried block of this shard must extend C', the
// latest one confirmed by the parent, and C, the last carried block (or C' if
// there are none), must reference an ancestor of the root block. G0 is exempt
// from the root-reference check because its previous-root hash is empty. Only
// the first carried block is checked against C' here; the master validates
// intermediate ordering and continuity ("Master should assure this check
// will not fail").
//
//	root chain    R0 <-- R1 <------------ R
//	                     :                :
//	                     v                v
//	shard chain   G <--- C' <-- A <------ C
//
//	R1 confirms the shard chain up to C', and R, on top of R1, up to C, so
//	R carries A and C. A must be a child of C', and C must reference R1 or
//	an ancestor of it.
func (c *ShardChain) validateRootBlock(root *qkctypes.RootBlock) (*types.Header, error) {
	config := c.chain.Config().QuarkChain
	if config == nil {
		return nil, errNotQuarkChain
	}
	header := root.Header()
	// Master validation is the first guard for root consensus (including PoW
	// and total difficulty), body commitments and complete header continuity.
	// This shard checks local availability and consistency below; it does not
	// repeat those upstream checks or accept missing required local data.
	// The master is expected to reject unsupported versions too; check it
	// locally so malformed or misrouted root blocks cannot enter this shard DB.
	if header.Version != 0 {
		return nil, fmt.Errorf("%w: %d", ErrRootBlockVersion, header.Version)
	}
	// Rule 1: Init handles root genesis at height 0. Normal root imports
	// must be above it and have a known parent.
	if header.Number == 0 {
		return nil, fmt.Errorf("%w: %w: root #0", ErrClusterStateInconsistency, ErrRootNotAfterGenesis)
	}
	parent := rawdb.ReadRootBlockHeader(c.db, header.ParentHash)
	if parent == nil {
		return nil, fmt.Errorf("%w: parent %x", ErrUnknownRootBlock, header.ParentHash)
	}
	// Rule 2: the master or peer shard must ensure every carried local-shard
	// block and relevant cross-shard deposit list is available before this root
	// block is imported. Check these guarantees defensively here; missing required
	// data is an ErrClusterStateInconsistency. Preserve the local headers'
	// carried order for the first-block check in rule 3.
	var blocks []*types.Header
	for _, block := range root.ShardBlockHeaders() {
		extra, err := block.QKCExtra()
		if err != nil {
			return nil, fmt.Errorf("shard block %x: %w", block.Hash(), err)
		}
		if extra.ShardID != config.ShardID {
			if err := c.validateXShardDeposits(block, extra, config); err != nil {
				return nil, err
			}
			continue
		}
		if !c.chain.HasBlock(block.Hash(), block.Number.Uint64()) {
			return nil, fmt.Errorf("%w: %w: #%d [%x..]", ErrClusterStateInconsistency, ErrUnknownConfirmedBlock, block.Number, block.Hash().Bytes()[:4])
		}
		blocks = append(blocks, block)
	}
	if limit := config.MaxBlocksPerRootBlock; limit != 0 && uint64(len(blocks)) > limit {
		return nil, fmt.Errorf("%w: %d, max %d", ErrTooManyConfirmedBlocks, len(blocks), limit)
	}
	// Rule 3 local continuity check. The master must validate the complete
	// carried-header ordering and continuity; here we check the first local block
	// against C', the parent's latest confirmation, guaranteed by initialization
	// and inherited by every imported root.
	confirmed, err := c.lastConfirmed(parent)
	if err != nil {
		return nil, err
	}
	if len(blocks) > 0 {
		first := blocks[0]
		if first.ParentHash != confirmed.Hash() {
			return nil, fmt.Errorf("%w: %w: first #%d [%x..]", ErrClusterStateInconsistency, ErrConfirmedNotContiguous, first.Number, first.Hash().Bytes()[:4])
		}
		confirmed = blocks[len(blocks)-1]
	}
	// Rule 3 local ancestry check: C, the last block of this confirmed shard,
	// or C' if there are none, references an ancestor of the root block. Even
	// after master validation, verify this against the locally stored root chain.
	// G0 has EmptyHash rather than a previous root to compare.
	if confirmed.Number.Sign() != 0 {
		prevRoot, err := c.prevRootBlock(confirmed)
		if err != nil {
			return nil, fmt.Errorf("%w: confirmed block: %w", ErrShardStateInconsistency, err)
		}
		if !c.isRootDescendant(header, prevRoot) {
			return nil, fmt.Errorf("%w: %w: confirmed #%d [%x..] references root #%d [%x..]", ErrClusterStateInconsistency, ErrConfirmedRootNotAncestor,
				confirmed.Number, confirmed.Hash().Bytes()[:4], prevRoot.Number, prevRoot.Hash().Bytes()[:4])
		}
	}
	return confirmed, nil
}

// validateXShardDeposits checks a remote block's root reference and deposit list.
// The referenced root must be stored locally.
// Sources in this shard's network must have a deposit list, even if empty.
// Sources outside the network must have no list.
// These rules also apply to ordinary blocks referencing R0.
func (c *ShardChain) validateXShardDeposits(block *types.Header, extra *types.QKCExtra, config *params.QuarkChainConfig) error {
	// All shards start at R0, so an unknown root cannot be explained by this
	// shard having been created after the referenced root, as in pyquarkchain.
	exist := rawdb.HasRootBlock(c.db, extra.PrevRootBlock)
	if !exist {
		return fmt.Errorf("%w: %w: shard %d block #%d [%x..] references root %x", ErrClusterStateInconsistency, ErrUnknownRootBlock, extra.ShardID, block.Number, block.Hash().Bytes()[:4], extra.PrevRootBlock)
	}
	sends := config.IsXShardDestination(extra.ShardID)
	stored := rawdb.HasXShardDeposits(c.db, block.Hash())
	switch {
	case sends && !stored:
		return fmt.Errorf("%w: %w: shard %d block #%d [%x..]", ErrClusterStateInconsistency, ErrMissingXShardDeposits, extra.ShardID, block.Number, block.Hash().Bytes()[:4])
	case !sends && stored:
		return fmt.Errorf("%w: %w: shard %d block #%d [%x..]", ErrClusterStateInconsistency, ErrUnexpectedXShardDeposits, extra.ShardID, block.Number, block.Hash().Bytes()[:4])
	}
	return nil
}

// setHeadForRootTip moves the canonical head onto the shard chain of the target
// root block, given C, the latest shard block confirmed by the root chain up to
// it. C must be a non-nil, validated checkpoint compatible with rootTip. The caller
// publishes rootTip after all required writes succeed. It mirrors the tip reset
// of ShardState.add_root_block in pyquarkchain. The head must keep the two
// properties shouldUpdateHead relies on: it is C or extends C, and, unless it
// is G0, it references the root tip or one of its ancestors.
//
// Init starts directly at its genesis or recovered checkpoint C. Once initialized, root
// imports apply the two cases below; an empty root body still inherits C.
//
// Case 1: C is not on the canonical chain, for example on another fork.
// The head moves to C, which has both properties, see rule 3 of
// validateRootBlock. A descendant such as D can also be the shard tip if it
// references the new root tip or one of its ancestors; D need not itself be
// root-confirmed. As a simplification, switch to C instead of searching for
// the best compatible descendant, which would require additional fork indexing
// and selection logic ("shard_header might not be the tip of the longest chain"
// in pyquarkchain).
//
//	root chain    R0 <-- R1 <-- R2
//	                     :      :
//	                     v      v
//	shard chain   G <--- A <--- C <--- D
//	                     ^
//	                     +----- X <--- H
//
//	R1 confirms the shard chain up to A, and the new root tip R2 up to C.
//	C is on a fork of the head H, so the head moves from H to C, not to D.
//
// Case 2: C is canonical. Since ShardChain manages both the head and canonical
// mappings, C is the current head or one of its ancestors. Starting at the
// current head, walk back to the nearest shard block whose referenced root is
// the new root tip or one of its ancestors. C is the earliest block the walk
// can reach. This follows "we need to scan back until finding a minor block
// pointing to the same root chain r_block is on".
//
//	                     +----- R2'
//	                     v
//	root chain    R0 <-- R1 <-- R2 <-- R3
//	                     :
//	                     v
//	shard chain   G <--- C <--- X[R1] <--- Y[R2'] <--- H[R2']
//
//	R1 and its descendants confirm the shard chain up to C. X references
//	R1, and Y and H reference R2', the former root tip. With R3 as the new
//	root tip, the head moves back from H to X.
func (c *ShardChain) setHeadForRootTip(rootTip *qkctypes.RootBlockHeader, confirmed *types.Header) error {
	if c.rootTip == nil || c.chain.GetCanonicalHash(confirmed.Number.Uint64()) != confirmed.Hash() {
		// Initialization and case 1 select C directly. Init must also ask core
		// to accept an unchanged head before publishing the root tip.
		if err := c.setCanonicalByHead(confirmed); err != nil {
			return err
		}
		return nil
	}
	// Case 2: scan above C, which is already compatible with rootTip. Stopping
	// at C also avoids looking up G0's empty previous-root reference.
	current := c.chain.CurrentBlock()
	head := current
	for head.Number.Cmp(confirmed.Number) > 0 {
		prevRoot, err := c.prevRootBlock(head)
		if err != nil {
			return fmt.Errorf("%w: shard head: %w", ErrShardStateInconsistency, err)
		}
		if c.isRootDescendant(rootTip, prevRoot) {
			break
		}
		if head = c.chain.GetHeader(head.ParentHash, head.Number.Uint64()-1); head == nil {
			return fmt.Errorf("%w: %w: shard head parent missing", ErrShardStateInconsistency, consensus.ErrUnknownAncestor)
		}
	}
	if head.Hash() != current.Hash() {
		if err := c.setCanonicalByHead(head); err != nil {
			return err
		}
	}
	return nil
}

// setCanonicalByHead loads the block for head and makes it canonical.
func (c *ShardChain) setCanonicalByHead(head *types.Header) error {
	block := c.chain.GetBlock(head.Hash(), head.Number.Uint64())
	if block == nil {
		return fmt.Errorf("%w: %w: head #%d [%x..]", ErrShardStateInconsistency, consensus.ErrUnknownAncestor, head.Number, head.Hash().Bytes()[:4])
	}
	if _, err := c.chain.SetCanonical(block); err != nil {
		return fmt.Errorf("set canonical head %x: %w", head.Hash(), err)
	}
	return nil
}

// prevRootBlock returns the header of the root block referenced by a shard block.
// Callers must handle genesis separately because G0's previous-root hash is empty.
func (c *ShardChain) prevRootBlock(header *types.Header) (*qkctypes.RootBlockHeader, error) {
	extra, err := header.QKCExtra()
	if err != nil {
		return nil, err
	}
	root := rawdb.ReadRootBlockHeader(c.db, extra.PrevRootBlock)
	if root == nil {
		return nil, fmt.Errorf("%w: %x", ErrUnknownRootBlock, extra.PrevRootBlock)
	}
	return root, nil
}

// lastConfirmed returns the header of the latest shard block confirmed by the
// root chain up to the given root block. A successful read always returns a
// non-nil header. A missing confirmation makes the root invalid. A recorded
// confirmation whose local block is missing is a shard-state inconsistency.
func (c *ShardChain) lastConfirmed(root *qkctypes.RootBlockHeader) (*types.Header, error) {
	hash := rawdb.ReadLastConfirmedBlockHash(c.db, root.Hash())
	if hash == (common.Hash{}) {
		return nil, fmt.Errorf("%w: root #%d [%x..] has no confirmation", ErrInvalidRootBlock, root.Number, root.Hash().Bytes()[:4])
	}
	header := c.chain.GetHeaderByHash(hash)
	if header == nil {
		// InsertRootBlock only stores confirmations of locally available blocks.
		return nil, fmt.Errorf("%w: %w: %x", ErrShardStateInconsistency, ErrUnknownConfirmedBlock, hash)
	}
	return header, nil
}

// isRootDescendant reports whether the root block descendant extends, or is,
// the root block ancestor. It returns false if an ancestor is missing.
func (c *ShardChain) isRootDescendant(descendant, ancestor *qkctypes.RootBlockHeader) bool {
	for descendant.Number > ancestor.Number {
		if descendant = rawdb.ReadRootBlockHeader(c.db, descendant.ParentHash); descendant == nil {
			return false
		}
	}
	return descendant.Hash() == ancestor.Hash()
}

// isShardDescendant reports whether the shard block descendant extends, or is,
// the shard block ancestor. It returns false if an ancestor is missing.
func (c *ShardChain) isShardDescendant(descendant, ancestor *types.Header) bool {
	for descendant.Number.Uint64() > ancestor.Number.Uint64() {
		if descendant = c.chain.GetHeader(descendant.ParentHash, descendant.Number.Uint64()-1); descendant == nil {
			return false
		}
	}
	return descendant.Hash() == ancestor.Hash()
}
