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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
	"github.com/ethereum/go-ethereum/qkc/rawdb"
	qkctypes "github.com/ethereum/go-ethereum/qkc/types"
	"github.com/holiman/uint256"
)

var (
	// ErrUnknownCursorRoot is returned when the requested root tip of a cursor
	// is not known.
	ErrUnknownCursorRoot = errors.New("unknown root block of the cross-shard cursor")

	// ErrInconsistentClusterState is returned when local data is missing or
	// inconsistent with a known root chain or an imported shard block's cursor.
	ErrInconsistentClusterState = errors.New("inconsistent cluster state")

	// ErrXShardDepositsMissing is returned when advancing to a source block
	// whose deposits have not been received yet.
	ErrXShardDepositsMissing = errors.New("cross-shard deposits not received")
)

// XShardChain is the view a shard has of the root chain and of the deposits the
// other shards sent it, which the cross-shard cursor walks.
type XShardChain interface {
	// RootBlock returns the root block with the given hash, or nil if unknown.
	RootBlock(hash common.Hash) *qkctypes.RootBlock

	// RootBlockByNumber returns the root block at the given height on the root
	// chain ending at the root block with the given hash, or nil if there is
	// none, e.g. above it.
	RootBlockByNumber(tip common.Hash, number uint64) *qkctypes.RootBlock

	// XShardDeposits returns the deposits sent to the shard by the block of
	// another shard with the given hash, or nil if they are not received.
	XShardDeposits(hash common.Hash) []*types.DepositTx
}

// RootBlock reads a root block received by the shard.
func (c *ShardChain) RootBlock(hash common.Hash) *qkctypes.RootBlock {
	return rawdb.ReadRootBlock(c.db, hash)
}

// RootBlockByNumber reads a root block on the ancestry of tip, including forks.
func (c *ShardChain) RootBlockByNumber(tip common.Hash, number uint64) *qkctypes.RootBlock {
	for header := rawdb.ReadRootBlockHeader(c.db, tip); header != nil; header = rawdb.ReadRootBlockHeader(c.db, header.ParentHash) {
		if uint64(header.Number) == number {
			return c.RootBlock(header.Hash())
		}
		if uint64(header.Number) < number {
			return nil
		}
	}
	return nil
}

// XShardDeposits reads the incoming deposits of a confirmed source block.
// A nil result means the deposit list has not been received, even if empty.
func (c *ShardChain) XShardDeposits(hash common.Hash) []*types.DepositTx {
	return rawdb.ReadXShardDeposits(c.db, hash)
}

// xShardCursor walks the deposits of a shard in the order the shard applies
// them, as the XshardTxCursor of pyquarkchain: root block by root block, the
// coinbase of the root block first, then the deposits of each source block the
// root block confirms, in order. Its position is the deposit it last returned:
//
//	(x, 0, 0)  the start of root block x, or the end if x is past the walk
//	(x, 0, 1)  the coinbase of root block x
//	(x, y, z)  deposit z of source block y, counted from 1, of root block x
//
// The walk ends at the previous root block of the shard block applying the
// deposits.
type xShardCursor struct {
	config  *params.ChainConfig
	chain   XShardChain
	maxRoot *qkctypes.RootBlock // last root block of the walk

	root         *qkctypes.RootBlock // root block walked, nil at the end
	shardIndex   uint64              // source block walked in the root block, from 1
	depositIndex uint64              // deposit walked in the source block
	deposits     []*types.DepositTx  // deposits of the source block walked
}

// newXShardCursor returns a cursor at the given position, walking up to the
// root block maxRoot.
func newXShardCursor(config *params.ChainConfig, chain XShardChain, pos types.XShardCursor, maxRoot common.Hash) (*xShardCursor, error) {
	c := &xShardCursor{
		config:       config,
		chain:        chain,
		maxRoot:      chain.RootBlock(maxRoot),
		shardIndex:   pos.ShardBlockIndex,
		depositIndex: pos.XShardDepositIndex,
	}
	if c.maxRoot == nil {
		return nil, fmt.Errorf("%w: %x", ErrUnknownCursorRoot, maxRoot)
	}
	// EOF is exactly one root past the tip, with no source or deposit index.
	if pos.RootBlockHeight > c.maxRoot.NumberU64() {
		if pos.RootBlockHeight != c.maxRoot.NumberU64()+1 || c.shardIndex != 0 || c.depositIndex != 0 {
			return nil, fmt.Errorf("%w: invalid EOF cursor %+v after root #%d", ErrInconsistentClusterState, pos, c.maxRoot.NumberU64())
		}
		return c, nil
	}
	// The cursor may lag behind the parent's root reference. Missing root data
	// below the tip is an inconsistency, not EOF.
	if c.root = chain.RootBlockByNumber(maxRoot, pos.RootBlockHeight); c.root == nil {
		return nil, fmt.Errorf("%w: missing root block #%d below %x", ErrInconsistentClusterState, pos.RootBlockHeight, maxRoot)
	}
	// The imported parent's cursor must fit the root block it points to.
	if c.shardIndex > uint64(len(c.root.ShardBlockHeaders())) {
		return nil, fmt.Errorf("%w: invalid cross-shard cursor %+v: root block #%d has %d source blocks", ErrInconsistentClusterState, pos, c.root.NumberU64(), len(c.root.ShardBlockHeaders()))
	}
	if c.shardIndex > 0 {
		header := c.root.ShardBlockHeaders()[c.shardIndex-1]
		// The parent already consumed deposits from this source block.
		if c.deposits = chain.XShardDeposits(header.Hash()); c.deposits == nil {
			return nil, fmt.Errorf("%w: missing deposits of block %x", ErrInconsistentClusterState, header.Hash())
		}
	}
	return c, nil
}

// Cursor returns the position of the cursor, implementing core.XShardDeposits.
func (c *xShardCursor) Cursor() types.XShardCursor {
	if c.root == nil {
		return types.XShardCursor{RootBlockHeight: c.maxRoot.NumberU64() + 1, ShardBlockIndex: c.shardIndex, XShardDepositIndex: c.depositIndex}
	}
	return types.XShardCursor{RootBlockHeight: c.root.NumberU64(), ShardBlockIndex: c.shardIndex, XShardDepositIndex: c.depositIndex}
}

// Next moves the cursor onto the next deposit and returns it, or returns nil at
// the end of the walk, implementing core.XShardDeposits.
func (c *xShardCursor) Next() (*types.DepositTx, error) {
	if c.root == nil {
		return nil, nil
	}
	c.depositIndex++
	for {
		if deposit := c.current(); deposit != nil {
			return deposit, nil
		}
		// Move on to the next source block of the root block, skipping the
		// blocks of the shard itself.
		if headers := c.root.ShardBlockHeaders(); c.shardIndex < uint64(len(headers)) {
			c.shardIndex++
			c.depositIndex = 0
			c.deposits = nil

			header := headers[c.shardIndex-1]
			extra, err := header.QKCExtra()
			if err != nil {
				return nil, fmt.Errorf("source block %x: %w", header.Hash(), err)
			}
			if extra.ChainID.Cmp(c.config.ChainID) == 0 {
				continue
			}
			if c.deposits = c.chain.XShardDeposits(header.Hash()); c.deposits == nil {
				return nil, fmt.Errorf("%w: block %x of shard %d", ErrXShardDepositsMissing, header.Hash(), extra.ChainID)
			}
			continue
		}
		// Move on to the next root block, starting with its coinbase.
		nextHeight := c.root.NumberU64() + 1
		if nextHeight > c.maxRoot.NumberU64() {
			c.root, c.shardIndex, c.depositIndex, c.deposits = nil, 0, 0, nil
			return nil, nil
		}
		root := c.chain.RootBlockByNumber(c.maxRoot.Hash(), nextHeight)
		if root == nil {
			// A gap before maxRoot is missing data, not the end of the walk.
			return nil, fmt.Errorf("%w: missing root block #%d below %x", ErrInconsistentClusterState, nextHeight, c.maxRoot.Hash())
		}
		c.root, c.shardIndex, c.depositIndex, c.deposits = root, 0, 1, nil
	}
}

// current returns the deposit at the position of the cursor, or nil if there
// is none.
func (c *xShardCursor) current() *types.DepositTx {
	if c.shardIndex == 0 {
		if c.depositIndex == 1 && c.config.IsRootCoinbaseChain() {
			return rootCoinbaseDeposit(c.root)
		}
		return nil
	}
	if c.depositIndex < uint64(len(c.deposits)) {
		return c.deposits[c.depositIndex]
	}
	return nil
}

// rootCoinbaseDeposit returns the deposit paying the coinbase of a root block to
// its miner, on the configured root coinbase chain. It is a move, with no gas cost.
func rootCoinbaseDeposit(root *qkctypes.RootBlock) *types.DepositTx {
	coinbase := root.Coinbase()
	return &types.DepositTx{
		Version:         types.DepositTxVersion,
		SourceBlockHash: root.Hash(),
		SourceLogIndex:  0,
		From:            coinbase,
		To:              &coinbase,
		Value:           root.CoinbaseAmount(),
		GasPrice:        new(uint256.Int),
		Gas:             params.XShardSendGas,
		IsMove:          true,
	}
}

var _ core.XShardInbox = (*XShardInbox)(nil)

// XShardInbox serves a shard the deposits its blocks apply, walking the
// cross-shard cursor over the root chain and the deposits received from the
// other shards. It implements core.XShardInbox.
type XShardInbox struct {
	config *params.ChainConfig
	chain  XShardChain
}

// NewXShardInbox returns the inbox of the shard with the given configuration.
func NewXShardInbox(config *params.ChainConfig, chain XShardChain) *XShardInbox {
	return &XShardInbox{config: config, chain: chain}
}

// Deposits returns the deposits a shard block may apply, in order: from the
// cursor of its parent up to its previous root block.
// The root reference must equal or extend the parent's, except at shard genesis.
func (in *XShardInbox) Deposits(parent *types.Header, prevRootBlock common.Hash) (core.XShardDeposits, error) {
	extra, err := parent.QKCExtra()
	if err != nil {
		return nil, fmt.Errorf("parent: %w", err)
	}
	if parent.Number.Sign() != 0 {
		root := in.chain.RootBlock(prevRootBlock)
		if root == nil {
			return nil, fmt.Errorf("%w: %x", ErrUnknownCursorRoot, prevRootBlock)
		}
		parentRoot := in.chain.RootBlock(extra.PrevRootBlock)
		if parentRoot == nil {
			return nil, fmt.Errorf("%w: missing parent root %x", ErrInconsistentClusterState, extra.PrevRootBlock)
		}
		if root.NumberU64() < parentRoot.NumberU64() {
			return nil, ErrRootNotDescendant
		}
		ancestor := in.chain.RootBlockByNumber(prevRootBlock, parentRoot.NumberU64())
		if ancestor == nil {
			return nil, fmt.Errorf("%w: missing root block #%d below %x", ErrInconsistentClusterState, parentRoot.NumberU64(), prevRootBlock)
		}
		if ancestor.Hash() != parentRoot.Hash() {
			return nil, ErrRootNotDescendant
		}
	}
	cursor, err := newXShardCursor(in.config, in.chain, extra.Cursor, prevRootBlock)
	if err != nil {
		return nil, err
	}
	return cursor, nil
}
