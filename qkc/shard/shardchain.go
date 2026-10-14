// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.
//
// The go-ethereum library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-ethereum library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-ethereum library. If not, see <http://www.gnu.org/licenses/>.

// Package shard implements the consensus layer of a QuarkChain shard.
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
	"github.com/ethereum/go-ethereum/qkc/rawdb"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
)

var (
	// ErrRootChainUninitialized is returned when no root tip is known yet.
	ErrRootChainUninitialized = errors.New("root chain not initialized")

	// ErrUnknownRootBlock is returned when a shard block references a root block
	// that is not known yet.
	ErrUnknownRootBlock = errors.New("unknown root block")

	// ErrRootNotDescendant is returned when the root block referenced by a shard
	// block does not extend the one referenced by its parent.
	ErrRootNotDescendant = errors.New("previous root block does not extend the one of the parent")

	// ErrNotConfirmedDescendant is returned when a shard block does not extend
	// the latest shard block confirmed by the root block it references.
	ErrNotConfirmedDescendant = errors.New("block does not extend the last confirmed block")

	// ErrUnknownConfirmedBlock is returned when the root chain confirms a shard
	// block that is not in the local chain.
	ErrUnknownConfirmedBlock = errors.New("unknown confirmed block")
)

// ShardChain is the consensus layer of a shard. It imports shard blocks into
// the underlying chain under the root chain rules, and selects the canonical
// shard head.
type ShardChain struct {
	db    ethdb.KeyValueStore // Database of the root blocks seen by the shard
	chain *core.BlockChain

	mu sync.Mutex // Serializes block imports and head updates
}

// NewShardChain creates a shard chain on top of the given chain, reading the
// root blocks from db.
func NewShardChain(db ethdb.KeyValueStore, chain *core.BlockChain) *ShardChain {
	return &ShardChain{db: db, chain: chain}
}

// InsertShardBlock validates a block against the root chain, imports it, and
// makes it the canonical head if the fork choice rule selects it. Known blocks
// are ignored.
func (c *ShardChain) InsertShardBlock(block *types.Block) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// TODO 1. check shard id here 2. check future block in engine.VerifyBlockHeader

	if c.chain.HasBlock(block.Hash(), block.NumberU64()) {
		return nil
	}
	rootTip := c.rootTip()
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

	// post cross shard execution will be validated in InsertBlockWithoutSetHead
	if _, err := c.chain.InsertBlockWithoutSetHead(context.Background(), block, false); err != nil {
		return err
	}
	if c.shouldUpdateHead(block.Header(), prevRoot, rootTip) {
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
// Rule 1: the previous root block of the block must be the one of its parent,
// or a descendant of it ("prev root block height must be non-decreasing",
// "prev root blocks are not on the same chain"). The view a shard chain has of
// the root chain only moves forward, along a single root chain. The
// cross-shard cursor relies on it: a block applies the deposits confirmed by
// the root blocks from the previous root block of its parent up to its own.
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
// Rule 2: the block must extend the latest shard block confirmed by the root
// chain up to its previous root block ("prev root block's minor block is not in
// the same chain as the minor block"). Once a root block confirms a shard
// block, the shard blocks that have seen that root block cannot fork away from
// it.
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
	// Both root blocks must be known ("cannot find root block for the minor
	// block"). The one of the parent was already checked on its import.
	prevRoot, err := c.prevRootBlock(header)
	if err != nil {
		return nil, err
	}
	parentPrevRoot, err := c.prevRootBlock(parent)
	if err != nil {
		return nil, fmt.Errorf("parent: %w", err)
	}
	// Rule 1. pyquarkchain checks the heights first, which is implied here, as
	// a root block is never lower than its ancestors.
	if !c.isRootDescendant(prevRoot, parentPrevRoot) {
		return nil, fmt.Errorf("%w: root #%d [%x..] after #%d [%x..]", ErrRootNotDescendant,
			prevRoot.Number, prevRoot.Hash().Bytes()[:4], parentPrevRoot.Number, parentPrevRoot.Hash().Bytes()[:4])
	}
	// Rule 2, skipped while the root chain confirms no block of the shard yet,
	// as after genesis.
	confirmed, err := c.lastConfirmed(prevRoot)
	if err != nil {
		return nil, err
	}
	// The block itself is not stored yet, so check its parent instead: the
	// block extends C if and only if its parent is C or extends it.
	if confirmed != nil && !c.isShardDescendant(parent, confirmed) {
		return nil, fmt.Errorf("%w: confirmed #%d [%x..]", ErrNotConfirmedDescendant,
			confirmed.Number, confirmed.Hash().Bytes()[:4])
	}
	return prevRoot, nil
}

// shouldUpdateHead reports whether a newly imported shard block, referencing
// the root block prevRoot, should become the canonical head. It mirrors the tip
// update of ShardState.add_block in pyquarkchain: "Update tip if a block is
// appended or a fork is longer (with the same ancestor confirmed by root block
// tip) or they are equal length but the root height confirmed by the block is
// longer". The rules apply in order:
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
// As in pyquarkchain, forks are compared by height, not by difficulty.
func (c *ShardChain) shouldUpdateHead(header *types.Header, prevRoot, rootTip *qkctypes.RootBlockHeader) bool {
	// Rule 1: "Don't update tip if the block depends on a root block that is
	// not root_tip or root_tip's ancestor".
	if !c.isRootDescendant(rootTip, prevRoot) {
		return false
	}
	// Rule 2. The head always extends C, so the block does too. The root tip
	// update must keep it so, by moving the head onto the shard chain confirmed
	// by the new root tip.
	current := c.chain.CurrentBlock()
	if header.ParentHash == current.Hash() {
		return true
	}
	// Rule 3, as __is_minor_block_linked_to_root_tip: "Determine whether a
	// minor block is a descendant of a minor block confirmed by root tip". Any
	// fork qualifies while the root chain confirms no block of the shard yet,
	// as after genesis. pyquarkchain first checks that the block is higher than
	// C, which is implied here: the block is new, so it is not C, and it only
	// extends C if it is higher.
	confirmed, err := c.lastConfirmed(rootTip)
	if err != nil {
		return false
	}
	if confirmed != nil && !c.isShardDescendant(header, confirmed) {
		return false
	}
	// Rule 4. On a full tie, the head stays.
	if header.Number.Uint64() != current.Number.Uint64() {
		return header.Number.Uint64() > current.Number.Uint64()
	}

	// If both shard headers has the same block number,
	// compare the view of the root block and switch to the minor
	// block with longer root block for more cross shard tx fee.
	currentRoot, err := c.prevRootBlock(current)
	return err == nil && prevRoot.Number > currentRoot.Number
}

// rootTip returns the header of the root tip, or nil if it is not known.
func (c *ShardChain) rootTip() *qkctypes.RootBlockHeader {
	hash := rawdb.ReadRootHeadHash(c.db)
	if hash == (common.Hash{}) {
		return nil
	}
	return rawdb.ReadRootBlockHeader(c.db, hash)
}

// prevRootBlock returns the header of the root block referenced by a shard
// block.
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
// root chain up to the given root block, or nil if none is confirmed yet.
func (c *ShardChain) lastConfirmed(root *qkctypes.RootBlockHeader) (*types.Header, error) {
	hash := rawdb.ReadLastConfirmedBlockHash(c.db, root.Hash())
	if hash == (common.Hash{}) {
		return nil, nil
	}
	header := c.chain.GetHeaderByHash(hash)
	if header == nil {
		// Unexpected error: AddRootBlock should write confirmed minor blocks before writing root blocks.
		// TODO: the shard may be stale, and upper layer should handle this error, likely resync the cluster
		return nil, fmt.Errorf("%w: %x", ErrUnknownConfirmedBlock, hash)
	}
	return header, nil
}

// isRootDescendant reports whether the root block descendant extends, or is,
// the root block ancestor.
func (c *ShardChain) isRootDescendant(descendant, ancestor *qkctypes.RootBlockHeader) bool {
	for descendant.Number > ancestor.Number {
		if descendant = rawdb.ReadRootBlockHeader(c.db, descendant.ParentHash); descendant == nil {
			return false
		}
	}
	return descendant.Hash() == ancestor.Hash()
}

// isShardDescendant reports whether the shard block descendant extends, or is,
// the shard block ancestor.
func (c *ShardChain) isShardDescendant(descendant, ancestor *types.Header) bool {
	for descendant.Number.Uint64() > ancestor.Number.Uint64() {
		if descendant = c.chain.GetHeader(descendant.ParentHash, descendant.Number.Uint64()-1); descendant == nil {
			return false
		}
	}
	return descendant.Hash() == ancestor.Hash()
}
