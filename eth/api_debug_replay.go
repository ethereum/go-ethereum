// Copyright 2024 The go-ethereum Authors
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

package eth

import (
	"context"
	"fmt"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/core/state"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/core/types/bal"
	"github.com/ethereum/go-ethereum/core/vm"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

// ReplayBadBlock re-executes a stored bad block using all the information
// captured during the block construction, and returns execution details
// side by side for diffing.
func (api *DebugAPI) ReplayBadBlock(ctx context.Context, hash common.Hash) (map[string]interface{}, error) {
	details := rawdb.ReadBadBlockWithDetails(api.eth.chainDb, hash)
	if details == nil {
		return nil, fmt.Errorf("bad block %#x not found", hash)
	}
	block := details.Block

	parent := api.eth.blockchain.GetBlock(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, fmt.Errorf("parent block %#x not found", block.ParentHash())
	}
	statedb, release, err := api.eth.stateAtBlock(ctx, parent, nil, false, false)
	if err != nil {
		return nil, fmt.Errorf("parent state unavailable: %w", err)
	}
	defer release()

	importState := statedb.Copy() // snapshot before the build replay mutates it

	// Build side: same serial execution, with the reverted transactions injected.
	buildBAL, buildReceipts, buildGas, buildRoot, err := api.replayBuild(ctx, block, statedb, details.Reverted)
	if err != nil {
		return nil, fmt.Errorf("build replay: %w", err)
	}
	// Import side: identical execution without the reverted transactions.
	importBAL, importReceipts, importGas, importRoot, err := api.replayBuild(ctx, block, importState, nil)
	if err != nil {
		return nil, fmt.Errorf("import replay: %w", err)
	}
	buildHash, importHash := buildBAL.Hash(), importBAL.Hash()

	out := map[string]interface{}{
		"reason":        details.Reason,
		"revertedCount": len(details.Reverted),
		"reverted":      details.Reverted,

		// Gas accounting.
		"headerGasUsed": block.GasUsed(),
		"buildGasUsed":  buildGas,
		"importGasUsed": importGas,

		// Block-level access list.
		"headerBALHash": block.Header().BlockAccessListHash,
		"buildBALHash":  buildHash,
		"importBALHash": importHash,
		"buildBAL":      buildBAL,
		"importBAL":     importBAL,

		// State root.
		"headerStateRoot": block.Root(),
		"buildStateRoot":  buildRoot,
		"importStateRoot": importRoot,

		// Receipts: the ones recorded at build time, plus both replays.
		"storedBuildReceipts": details.Receipts,
		"buildReceipts":       buildReceipts,
		"importReceipts":      importReceipts,
	}
	return out, nil
}

// replayBuild re-executes a block's construction against the provided state.
func (api *DebugAPI) replayBuild(ctx context.Context, block *types.Block, statedb *state.StateDB, reverted []*rawdb.RevertedTx) (*bal.BlockAccessList, types.Receipts, uint64, common.Hash, error) {
	bc := api.eth.blockchain
	config := bc.Config()

	parent := bc.GetHeader(block.ParentHash(), block.NumberU64()-1)
	if parent == nil {
		return nil, nil, 0, common.Hash{}, fmt.Errorf("parent header %#x not found", block.ParentHash())
	}
	// Reconstruct the sealing header, resetting the fields accumulated during
	// execution so the replay starts from the same point as building/import.
	header := types.CopyHeader(block.Header())
	header.GasUsed = 0
	if header.BlobGasUsed != nil {
		header.BlobGasUsed = new(uint64)
	}
	header.RequestsHash = nil
	header.BlockAccessListHash = nil

	coinbase := block.Coinbase()
	evm := vm.NewEVM(core.NewEVMBlockContext(header, bc, &coinbase), statedb, config, vm.Config{})
	defer evm.Release()

	var (
		signer    = types.MakeSigner(config, header.Number, header.Time)
		gp        = core.NewGasPool(header.GasLimit)
		blockAL   = bal.NewConstructionBlockAccessList()
		blockHash = block.Hash()
		committed = block.Transactions()
		receipts  types.Receipts
		tcount    = 0
	)
	// Pre-execution system calls.
	blockAL.Merge(core.PreExecution(ctx, header.ParentBeaconRoot, parent, config, evm, header.Number, header.Time))

	// Group the reverted transactions by build slot.
	revBySlot := make(map[int][]*types.Transaction)
	for _, r := range reverted {
		revBySlot[int(r.Index)] = append(revBySlot[int(r.Index)], r.Tx)
	}

	// applyReverted executes a tx that was reverted during building.
	applyReverted := func(tx *types.Transaction) {
		msg, err := core.TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			return
		}
		statedb.SetTxContext(tx.Hash(), tcount, uint32(tcount+1))
		snap, gpSnap := statedb.Snapshot(), gp.Snapshot()
		_, _, err = core.ApplyTransactionWithEVM(msg, gp, statedb, header.Number, blockHash, header.Time, tx, evm)
		if err == nil {
			txRLP, _ := rlp.EncodeToBytes(tx)
			log.Warn("Expect the transaction to be failed", "index", tcount, "hash", tx.Hash(), "rlp", txRLP)
		}
		statedb.RevertToSnapshot(snap)
		gp.Set(gpSnap)
	}

	for k := 0; k <= len(committed); k++ {
		for _, tx := range revBySlot[k] {
			applyReverted(tx)
		}
		if k == len(committed) {
			break
		}
		tx := committed[k]
		msg, err := core.TransactionToMessage(tx, signer, header.BaseFee)
		if err != nil {
			return nil, nil, 0, common.Hash{}, fmt.Errorf("could not convert tx %d [%v]: %w", k, tx.Hash().Hex(), err)
		}
		statedb.SetTxContext(tx.Hash(), tcount, uint32(tcount+1))
		receipt, txBal, err := core.ApplyTransactionWithEVM(msg, gp, statedb, header.Number, blockHash, header.Time, tx, evm)
		if err != nil {
			return nil, nil, 0, common.Hash{}, fmt.Errorf("could not apply committed tx %d [%v]: %w", k, tx.Hash().Hex(), err)
		}
		receipts = append(receipts, receipt)
		if tx.Type() == types.BlobTxType && header.BlobGasUsed != nil {
			*header.BlobGasUsed += receipt.BlobGasUsed
		}
		tcount++
		blockAL.Merge(txBal)
	}

	// Post-execution system calls and finalize.
	var allLogs []*types.Log
	for _, r := range receipts {
		allLogs = append(allLogs, r.Logs...)
	}
	_, postBal, err := core.PostExecution(ctx, config, header.Number, header.Time, allLogs, evm, uint32(tcount+1))
	if err != nil {
		return nil, nil, 0, common.Hash{}, err
	}
	blockAL.Merge(postBal)

	body := types.Body{
		Transactions: committed,
		Withdrawals:  block.Withdrawals(),
	}
	bc.Engine().Finalize(bc, header, statedb, &body, uint32(tcount+1), blockAL)
	root := statedb.IntermediateRoot(evm.GetRules())

	return blockAL.ToEncodingObj(), receipts, gp.Used(), root, nil
}
