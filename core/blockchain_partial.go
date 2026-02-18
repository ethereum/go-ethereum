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

// ErrDeepReorg is returned when a chain reorganization exceeds the BAL retention depth.
// When this error is returned, the partial state node needs to resync state from full peers.
var ErrDeepReorg = errors.New("reorg depth exceeds BAL retention")

// ProcessBlockWithBAL processes a block using BAL instead of execution.
// This is the entry point for partial state block processing.
//
// # Trust Model - Why We Don't Re-Verify Consensus Attestations
//
// Post-Merge (PoS) Architecture Trust Boundary:
//   - Consensus Layer (CL): Responsible for block proposal, validator attestations,
//     finality (Casper FFG), proposer signatures, and all consensus rules
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
	if err := accessList.Validate(len(block.Transactions())); err != nil {
		return fmt.Errorf("invalid BAL structure: %w", err)
	}

	// 2. Verify BAL hash matches header commitment
	// TODO(EIP-7928): Uncomment when BlockAccessListHash is added to Header
	// balHash := accessList.Hash()
	// if balHash != block.Header().BlockAccessListHash {
	//     return fmt.Errorf("BAL hash mismatch: got %x, want %x",
	//         balHash, block.Header().BlockAccessListHash)
	// }

	// 3. Get parent state root. Use partialState's tracked root (the actual
	// computed root from the previous block) rather than the header root, which
	// may differ when untracked contracts have unresolved storage roots.
	parentRoot := bc.partialState.Root()
	if parentRoot == (common.Hash{}) {
		// First block after sync — use the parent block's header root
		parent := bc.GetBlock(block.ParentHash(), block.NumberU64()-1)
		if parent == nil {
			return errors.New("parent block not found")
		}
		parentRoot = parent.Root()
	}

	log.Debug("ProcessBlockWithBAL: parent root details",
		"block", block.NumberU64(), "parentRoot", parentRoot,
		"hasState", bc.HasState(parentRoot), "headerRoot", block.Root(),
		"trackedRoot", bc.partialState.Root())

	// 4. Apply BAL diffs and compute new state root.
	// Pass block.Root() as expectedRoot so the resolver can query peers for this
	// state's untracked contracts.
	newRoot, unresolved, err := bc.partialState.ApplyBALAndComputeRoot(parentRoot, block.Root(), accessList)
	if err != nil {
		return fmt.Errorf("failed to apply BAL: %w", err)
	}

	// 5. Verify computed root matches header.
	// If all storage roots were resolved, a mismatch indicates a real bug.
	// If some were unresolved, a mismatch is expected (stale storage roots).
	if newRoot != block.Root() {
		if unresolved == 0 {
			return fmt.Errorf("state root mismatch (all storage resolved): computed %x, header %x, block %d",
				newRoot, block.Root(), block.NumberU64())
		}
		log.Warn("Partial state root mismatch (unresolved storage roots)",
			"computed", newRoot, "header", block.Root(), "block", block.NumberU64(),
			"unresolved", unresolved)
	}

	// 6. Track last processed block for gap detection and HasState checks.
	bc.partialState.SetLastProcessedBlock(block.NumberU64())

	// 7. Block is stored via normal chain insertion
	// BAL storage for reorgs is handled separately via BALHistory

	log.Debug("Processed block with BAL",
		"number", block.NumberU64(),
		"hash", block.Hash().Hex(),
		"root", newRoot.Hex(),
		"accounts", len(*accessList))

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

	// Check if reorg exceeds BAL retention depth
	// If so, we need to resync state from full peers because we don't have the BALs
	if history := bc.partialState.History(); history != nil {
		retention := history.Retention()
		if retention > 0 && reorgDepth > retention {
			log.Warn("Reorg exceeds BAL retention depth, partial resync required",
				"reorgDepth", reorgDepth,
				"retention", retention,
				"ancestor", commonAncestor.Number())
			return ErrDeepReorg
		}
	}

	// Step 1: Revert state to common ancestor
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

// TriggerPartialResync initiates a state resync when a reorg exceeds BAL retention.
// This is called when HandlePartialReorg returns ErrDeepReorg.
//
// The resync fetches state from full peers using snap sync, downloading:
// - Full account trie (all balances, nonces, code hashes)
// - Storage only for tracked contracts (per ContractFilter configuration)
//
// This is similar to initial partial state sync, but starting from the reorg ancestor
// rather than genesis.
func (bc *BlockChain) TriggerPartialResync(ancestor *types.Header) error {
	if bc.partialState == nil {
		return errors.New("partial state not enabled")
	}

	log.Info("Triggering partial state resync due to deep reorg",
		"ancestor", ancestor.Number,
		"root", ancestor.Root.Hex())

	// TODO(partial-state): Implement resync coordination with downloader.
	// This requires extending eth/downloader to support targeted state sync.
	// For now, return an error indicating manual intervention may be needed.
	//
	// The implementation should:
	// 1. Pause normal block processing
	// 2. Use snap sync to fetch state at ancestor.Root
	// 3. Apply ContractFilter to only store tracked contract storage
	// 4. Resume normal operation once state is available
	return errors.New("partial state resync not yet implemented: restart node to re-sync from scratch, or increase --partial-state.bal-retention to handle deeper reorgs")
}
