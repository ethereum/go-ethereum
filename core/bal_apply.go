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

package core

import (
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/tracing"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
)

// ApplyBlockAccessList reconstructs a block's post-state by writing the
// post-block values recorded in its EIP-7928 block access list into statedb,
// without executing the block's transactions.
//
// For every account the list enumerates, the final entry of each change list is
// the post-block value (entries are ordered ascending by block-access index).
// Balance, nonce, code and storage writes are applied; a storage slot whose
// post-value is zero is cleared. Storage reads and touched-only accounts carry
// no post-values and leave the state untouched. Empty accounts are pruned by the
// subsequent commit according to the block's rules (EIP-158), exactly as they
// are during execution.
//
// Correctness relies on the access list enumerating every state change as an
// explicit write. This holds from EIP-6780 onwards, where SELFDESTRUCT can no
// longer wipe a contract's storage without individual slot writes. On chains
// where the legacy SELFDESTRUCT wipe is still reachable, the reconstructed state
// may retain stale slots, so callers MUST verify the committed state root
// against the block header before trusting the result.
//
// ApplyBlockAccessList only mutates the in-memory state; finalising, committing
// and verifying the resulting root are the caller's responsibility.
func ApplyBlockAccessList(statedb *state.StateDB, list *bal.BlockAccessList) {
	for _, account := range *list {
		addr := account.Address

		// Apply the post-block value (the last write) of every changed slot.
		for _, slot := range account.StorageChanges {
			writes := slot.SlotChanges
			if len(writes) == 0 {
				continue
			}
			key := common.Hash(slot.Slot.Bytes32())
			value := common.Hash(writes[len(writes)-1].PostValue.Bytes32())
			statedb.SetState(addr, key, value)
		}
		if n := len(account.BalanceChanges); n > 0 {
			statedb.SetBalance(addr, account.BalanceChanges[n-1].PostBalance, tracing.BalanceChangeUnspecified)
		}
		if n := len(account.NonceChanges); n > 0 {
			statedb.SetNonce(addr, account.NonceChanges[n-1].PostNonce, tracing.NonceChangeUnspecified)
		}
		if n := len(account.CodeChanges); n > 0 {
			statedb.SetCode(addr, account.CodeChanges[n-1].NewCode, tracing.CodeChangeUnspecified)
		}
	}
}

// useAccessListReconstruction reports whether block's post-state may be rebuilt
// from its EIP-7928 access list instead of executing its transactions.
//
// This is an opt-in acceleration for catching up. It is deliberately confined to
// the region where it is sound and useful:
//
//   - Enabled only via BlockChainConfig.BALStateReconstruction.
//   - Never while a live tracer is attached; a tracer must observe every state
//     transition, which reconstruction skips.
//   - Only Amsterdam blocks that carry an access list. Amsterdam is strictly
//     after EIP-6780 (Cancun), so every storage change is an explicit write.
//   - Only at or below the consensus-finalized block, whose execution the
//     network has already validated. The unfinalized tip is always re-executed,
//     so fork choice and attestation remain fully self-validated.
func (bc *BlockChain) useAccessListReconstruction(block *types.Block, vmConfig vm.Config) bool {
	if !bc.cfg.BALStateReconstruction || vmConfig.Tracer != nil {
		return false
	}
	if !bc.chainConfig.IsAmsterdam(block.Number(), block.Time()) || block.AccessList() == nil {
		return false
	}
	final := bc.CurrentFinalBlock()
	return final != nil && block.NumberU64() <= final.Number.Uint64()
}

// processBlockFromAccessList rebuilds the post-state of block from its access
// list and, if the reconstructed state root matches the header, persists the
// block with that state. It is the executionless counterpart of ProcessBlock's
// normal execute-and-validate flow and is only invoked once
// useAccessListReconstruction has confirmed the block is eligible.
//
// The access list has already been checked against the header's
// blockAccessListHash by ValidateBody, so the applied values are exactly those
// the block producer committed. The state-root comparison is the cryptographic
// anchor: a wrong or incomplete list yields a different root and is rejected,
// letting the caller fall back to full execution. Because receipts are a product
// of execution, blocks stored this way carry none — the accepted receipt gap for
// the finalized region, matching a snap-synced node's pre-pivot history.
func (bc *BlockChain) processBlockFromAccessList(parentRoot common.Hash, block *types.Block, config ExecuteConfig) (*blockProcessingResult, error) {
	start := time.Now()

	// Build a plain reader over the parent state; no prefetcher is needed since
	// nothing is executed. This mirrors setupExecutionState's non-prefetch path.
	var sdb state.Database
	if bc.chainConfig.IsUBT(block.Number(), block.Time()) {
		sdb = state.NewUBTDatabase(bc.triedb, bc.codedb)
	} else {
		sdb = state.NewMPTDatabase(bc.triedb, bc.codedb).WithSnapshot(bc.snaps)
	}
	statedb, err := state.New(parentRoot, sdb)
	if err != nil {
		return nil, err
	}
	ApplyBlockAccessList(statedb, block.AccessList())

	rules := bc.chainConfig.Rules(block.Number(), block.Difficulty().Sign() == 0, block.Time())
	if root := statedb.IntermediateRoot(rules); root != block.Root() {
		return nil, fmt.Errorf("reconstructed state root mismatch (have %x, want %x)", root, block.Root())
	}

	// Persist the block and its reconstructed state through the same write path
	// as normal execution, but without receipts or logs.
	var status WriteStatus
	if config.WriteState {
		if config.WriteHead {
			status, err = bc.writeBlockAndSetHead(block, nil, nil, statedb, false)
		} else {
			err = bc.writeBlockWithState(block, nil, statedb)
		}
		if err != nil {
			return nil, err
		}
	}
	return &blockProcessingResult{
		usedGas:  block.GasUsed(),
		procTime: time.Since(start),
		status:   status,
		stats:    &ExecuteStats{},
	}, nil
}
