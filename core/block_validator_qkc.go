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

package core

import (
	"context"
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/consensus"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/params"
)

var (
	// ErrXShardDepositMismatch is returned when a block of a QuarkChain shard
	// does not open with the deposits its inbox has for it.
	ErrXShardDepositMismatch = errors.New("cross-shard deposits mismatch")

	// ErrXShardCursorMismatch is returned when the cross-shard cursor of a block
	// of a QuarkChain shard is not the one after its deposits.
	ErrXShardCursorMismatch = errors.New("cross-shard cursor mismatch")

	// ErrXShardGas is returned when the cross-shard gas of a block of a
	// QuarkChain shard is invalid.
	ErrXShardGas = errors.New("invalid cross-shard gas")
)

// XShardInbox provides a QuarkChain shard with the cross-shard deposits its
// blocks apply. The consensus layer of the shard implements it.
type XShardInbox interface {
	// Deposits returns the deposits a shard block may apply, in the order it
	// applies them: from the cursor of its parent up to its previous root block.
	Deposits(parent *types.Header, prevRootBlock common.Hash) (XShardDeposits, error)
}

// XShardDeposits iterates over the deposits a shard block may apply.
type XShardDeposits interface {
	// Next moves on to the next deposit and returns it, or returns nil at the
	// end of the deposits.
	Next() (*types.DepositTx, error)

	// Cursor returns the cross-shard cursor on the deposit Next last returned,
	// or at the end once Next returned nil.
	Cursor() types.XShardCursor
}

// ProcessXShardDeposits walks the mandatory deposit prefix within gasLimit.
// Admission uses declared gas; apply executes or validates each admitted deposit
// and returns its actual gas usage. The returned cursor advances to EOF only when
// the inbox ends, otherwise it remains on the last admitted deposit.
func ProcessXShardDeposits(ctx context.Context, deposits XShardDeposits, gasLimit uint64, apply func(*types.DepositTx) (uint64, error)) (gasUsed uint64, cursor types.XShardCursor, err error) {
	cursor = deposits.Cursor()
	for {
		if err := ctx.Err(); err != nil {
			return gasUsed, cursor, err
		}
		// Probe even at exact exhaustion to distinguish EOF from pending deposits.
		deposit, err := deposits.Next()
		if err != nil {
			return gasUsed, cursor, err
		}
		if deposit == nil {
			return gasUsed, deposits.Cursor(), nil
		}
		if deposit.Gas > gasLimit-gasUsed {
			return gasUsed, cursor, nil
		}
		used, err := apply(deposit)
		if err != nil {
			return gasUsed, cursor, err
		}
		if used > gasLimit-gasUsed {
			return gasUsed, cursor, fmt.Errorf("%w: deposit used %d with %d left", ErrXShardGas, used, gasLimit-gasUsed)
		}
		gasUsed += used
		cursor = deposits.Cursor()
	}
}

// SetXShardInbox sets the inbox the blocks of a QuarkChain shard are checked
// against once they ran. It must be set before blocks are imported: without it
// the cross-shard deposits of the blocks are not checked.
func (bc *BlockChain) SetXShardInbox(inbox XShardInbox) {
	bc.xshardInbox = inbox
}

// XShardDeposits opens the inbox used by block validation for block construction.
func (bc *BlockChain) XShardDeposits(parent *types.Header, prevRootBlock common.Hash) (XShardDeposits, error) {
	if bc.xshardInbox == nil {
		return nil, errors.New("cross-shard inbox not configured")
	}
	return bc.xshardInbox.Deposits(parent, prevRootBlock)
}

// validateXShardDeposits checks the cross-shard deposits of a block of a
// QuarkChain shard once it ran, given the receipts of its transactions:
//
//  1. its cross-shard gas limit is its gas limit over XShardGasLimitDivisor,
//  2. it opens with the deposits of its inbox, in order,
//  3. its cross-shard gas used is the gas these deposits used, within its
//     cross-shard gas limit, and the next deposit does not fit in the gas left:
//     the block applies as many deposits as it can,
//  4. its cross-shard cursor is the one on its last deposit, or at the end if
//     there is no deposit left.
//
// The cross-shard gas the deposits leave is for the other transactions. It
// checks nothing on other chains, or without an inbox.
func (v *BlockValidator) validateXShardDeposits(block *types.Block, receipts types.Receipts) error {
	if v.config.QuarkChain == nil || v.bc == nil || v.bc.xshardInbox == nil {
		return nil
	}
	header := block.Header()
	extra, err := header.QKCExtra()
	if err != nil {
		return err
	}
	if want := header.GasLimit / params.XShardGasLimitDivisor; extra.XShardGasLimit != want {
		return fmt.Errorf("%w: limit %d, want %d", ErrXShardGas, extra.XShardGasLimit, want)
	}
	parent := v.bc.GetHeader(header.ParentHash, header.Number.Uint64()-1)
	if parent == nil {
		return consensus.ErrUnknownAncestor
	}
	deposits, err := v.bc.xshardInbox.Deposits(parent, extra.PrevRootBlock)
	if err != nil {
		return err
	}
	var (
		txs = block.Transactions()
		i   int
	)
	if extra.XShardGasUsed > extra.XShardGasLimit {
		return fmt.Errorf("%w: used %d, limit %d", ErrXShardGas, extra.XShardGasUsed, extra.XShardGasLimit)
	}
	gasUsed, cursor, err := ProcessXShardDeposits(context.Background(), deposits, extra.XShardGasLimit, func(deposit *types.DepositTx) (uint64, error) {
		if i == len(txs) || txs[i].Type() != types.DepositTxType {
			return 0, fmt.Errorf("%w: missing mandatory deposit of block %x log %d", ErrXShardGas, deposit.SourceBlockHash, deposit.SourceLogIndex)
		}
		if txs[i].Hash() != types.NewTx(deposit).Hash() {
			return 0, fmt.Errorf("%w: tx %d is not the deposit of block %x log %d", ErrXShardDepositMismatch, i, deposit.SourceBlockHash, deposit.SourceLogIndex)
		}
		used := receipts[i].GasUsed
		i++
		return used, nil
	})
	if err != nil {
		return err
	}
	if i < len(txs) && txs[i].Type() == types.DepositTxType {
		return fmt.Errorf("%w: unexpected deposit tx %d", ErrXShardDepositMismatch, i)
	}
	if gasUsed != extra.XShardGasUsed {
		return fmt.Errorf("%w: used %d, want %d, limit %d", ErrXShardGas, extra.XShardGasUsed, gasUsed, extra.XShardGasLimit)
	}
	if cursor != extra.Cursor {
		return fmt.Errorf("%w: have %+v, want %+v", ErrXShardCursorMismatch, extra.Cursor, cursor)
	}
	return nil
}
