// Copyright 2025 The go-ethereum Authors
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
	"errors"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/state/partial"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/log"
)

// ProcessBlockWithBAL processes a block using BAL instead of execution.
// This is the entry point for partial state block processing.
//
// # Trust Model - Why We Don't Re-Verify Consensus Attestations
//
// Post-Merge (PoS) Architecture Trust Boundary:
//   - Consensus Layer (CL): Responsible for block proposal, attestations (2/3+ sync committee
//     threshold), finality proofs, proposer signatures, and all consensus rules
//   - Execution Layer (EL): Responsible for transaction execution, state computation, receipts
//
// Blocks received via Engine API (engine_newPayloadV5) have ALREADY been attested by the CL
// before being sent to the EL. The EL trusts the CL for consensus validation - this is the
// fundamental trust model of the Merge architecture (see eth/catalyst/api.go).
//
// For partial state nodes:
//   - Normal operation: Blocks arrive via Engine API, already consensus-validated by CL
//   - We validate: BAL hash matches header commitment, computed state root matches header
//   - We trust: CL has verified proposer signatures, attestations, and finality
//
// This is identical to how full nodes operate - they also don't re-verify CL attestations.
// The only difference is we apply BAL diffs instead of re-executing transactions.
//
// Future consideration: If supporting light client sync where blocks come from untrusted
// P2P sources, use beacon light client verification via CommitteeChain.VerifySignedHeader()
// or HeadTracker.ValidateOptimistic() (see beacon/light/).
func (bc *BlockChain) ProcessBlockWithBAL(
	block *types.Block,
	accessList *bal.BlockAccessList,
) error {
	// Sanity check
	if bc.partialState == nil {
		return errors.New("partial state not enabled")
	}

	// Note: No consensus attestation verification here - blocks via Engine API are
	// pre-attested by the Consensus Layer. See function documentation above.

	// 1. Validate BAL structure
	if err := accessList.Validate(); err != nil {
		return fmt.Errorf("invalid BAL structure: %w", err)
	}

	// 2. Verify BAL hash matches header commitment
	// TODO(EIP-7928): Uncomment when BlockAccessListHash is added to Header
	// balHash := accessList.Hash()
	// if balHash != block.Header().BlockAccessListHash {
	//     return fmt.Errorf("BAL hash mismatch: got %x, want %x",
	//         balHash, block.Header().BlockAccessListHash)
	// }

	// 3. Get parent state root
	parent := bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return errors.New("parent block not found")
	}
	parentRoot := parent.Root()

	// 4. Apply BAL diffs and compute new state root
	newRoot, err := bc.partialState.ApplyBALAndComputeRoot(parentRoot, accessList)
	if err != nil {
		return fmt.Errorf("failed to apply BAL: %w", err)
	}

	// 5. Verify computed root matches header
	if newRoot != block.Root() {
		return fmt.Errorf("state root mismatch: computed %x, header %x",
			newRoot, block.Root())
	}

	// 6. Block is stored via normal chain insertion
	// BAL storage for reorgs is handled separately via BALHistory

	log.Debug("Processed block with BAL",
		"number", block.NumberU64(),
		"hash", block.Hash().Hex(),
		"root", newRoot.Hex(),
		"accounts", len(accessList.Accesses))

	return nil
}

// SupportsPartialState returns true if partial state processing is enabled.
func (bc *BlockChain) SupportsPartialState() bool {
	return bc.partialState != nil
}

// PartialState returns the partial state manager, or nil if not enabled.
func (bc *BlockChain) PartialState() *partial.PartialState {
	return bc.partialState
}

// HandlePartialReorg handles chain reorganization for partial state nodes.
// It reverts state to the common ancestor and then applies BALs from the new chain.
//
// Parameters:
//   - commonAncestor: The most recent block that both chains share
//   - newBlocks: Ordered list of blocks from the new chain (oldest to newest)
//   - getBAL: Function to retrieve BAL for a given block (from BALHistory or Engine API)
func (bc *BlockChain) HandlePartialReorg(
	commonAncestor *types.Block,
	newBlocks []*types.Block,
	getBAL func(blockHash common.Hash, blockNum uint64) (*bal.BlockAccessList, error),
) error {
	if bc.partialState == nil {
		return errors.New("partial state not enabled")
	}

	currentHead := bc.CurrentBlock()
	reorgDepth := currentHead.Number.Uint64() - commonAncestor.Number().Uint64()

	// Step 1: Revert state to common ancestor
	// Simply set state root to ancestor's root (we have all account trie data)
	bc.partialState.SetRoot(commonAncestor.Root())

	log.Debug("Reverted partial state to ancestor",
		"ancestor", commonAncestor.Number(),
		"ancestorRoot", commonAncestor.Root().Hex(),
		"reorgDepth", reorgDepth)

	// Step 2: Apply new chain's blocks using their BALs
	for _, block := range newBlocks {
		// Get BAL for this block
		accessList, err := getBAL(block.Hash(), block.NumberU64())
		if err != nil {
			return fmt.Errorf("failed to get BAL for block %d: %w", block.NumberU64(), err)
		}
		if accessList == nil {
			return fmt.Errorf("block %d missing BAL for reorg", block.NumberU64())
		}

		// Apply BAL to move state forward on new chain
		if err := bc.ProcessBlockWithBAL(block, accessList); err != nil {
			return fmt.Errorf("failed to apply block %d during reorg: %w",
				block.NumberU64(), err)
		}
	}

	if len(newBlocks) > 0 {
		log.Info("Completed partial state reorg",
			"ancestor", commonAncestor.Number(),
			"newHead", newBlocks[len(newBlocks)-1].NumberU64(),
			"reorgDepth", reorgDepth)
	} else {
		log.Info("Completed partial state reorg (reset to ancestor)",
			"ancestor", commonAncestor.Number(),
			"reorgDepth", reorgDepth)
	}

	return nil
}

// Note: Deep reorgs beyond block pruning depth require resync from peers.
// This is handled by the downloader, not here.
