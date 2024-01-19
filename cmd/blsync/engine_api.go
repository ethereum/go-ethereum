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

package main

import (
	"context"
	"fmt"
	"math/big"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	ctypes "github.com/ethereum/go-ethereum/core/types"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/trie"
	"github.com/holiman/uint256"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/tree"
)

func updateEngineApi(client *rpc.Client, headCh chan headData) {
	for headData := range headCh {
		execBlock, err := getExecBlock(headData.block)
		if err != nil {
			log.Error("Error extracting execution block from validated beacon block", "error", err)
			continue
		}
		execRoot := execBlock.Hash()
		finalizedRoot := common.Hash(headData.update.Finalized.PayloadHeader.BlockHash)
		if client == nil { // dry run, no engine API specified
			log.Info("New execution block retrieved", "block number", execBlock.NumberU64(), "block hash", execRoot, "finalized block hash", finalizedRoot)
		} else {
			if status, err := callNewPayloadV2(client, execBlock); err == nil {
				log.Info("Successful NewPayload", "block number", execBlock.NumberU64(), "block hash", execRoot, "status", status)
			} else {
				log.Error("Failed NewPayload", "block number", execBlock.NumberU64(), "block hash", execRoot, "error", err)
			}
			if status, err := callForkchoiceUpdatedV1(client, execRoot, finalizedRoot); err == nil {
				log.Info("Successful ForkchoiceUpdated", "head", execRoot, "status", status)
			} else {
				log.Error("Failed ForkchoiceUpdated", "head", execRoot, "error", err)
			}
		}
	}
}

// getExecBlock extracts the execution block from the beacon block's payload.
func getExecBlock(beaconBlock *capella.BeaconBlock) (*ctypes.Block, error) {
	payload := &beaconBlock.Body.ExecutionPayload
	txs := make([]*ctypes.Transaction, len(payload.Transactions))
	for i, opaqueTx := range payload.Transactions {
		var tx ctypes.Transaction
		if err := tx.UnmarshalBinary(opaqueTx); err != nil {
			return nil, fmt.Errorf("failed to parse tx %d: %v", i, err)
		}
		txs[i] = &tx
	}
	withdrawals := make([]*ctypes.Withdrawal, len(payload.Withdrawals))
	for i, w := range payload.Withdrawals {
		withdrawals[i] = &ctypes.Withdrawal{
			Index:     uint64(w.Index),
			Validator: uint64(w.ValidatorIndex),
			Address:   common.Address(w.Address),
			Amount:    uint64(w.Amount),
		}
	}
	wroot := ctypes.DeriveSha(ctypes.Withdrawals(withdrawals), trie.NewStackTrie(nil))
	execHeader := &ctypes.Header{
		ParentHash:      common.Hash(payload.ParentHash),
		UncleHash:       ctypes.EmptyUncleHash,
		Coinbase:        common.Address(payload.FeeRecipient),
		Root:            common.Hash(payload.StateRoot),
		TxHash:          ctypes.DeriveSha(ctypes.Transactions(txs), trie.NewStackTrie(nil)),
		ReceiptHash:     common.Hash(payload.ReceiptsRoot),
		Bloom:           ctypes.Bloom(payload.LogsBloom),
		Difficulty:      common.Big0,
		Number:          new(big.Int).SetUint64(uint64(payload.BlockNumber)),
		GasLimit:        uint64(payload.GasLimit),
		GasUsed:         uint64(payload.GasUsed),
		Time:            uint64(payload.Timestamp),
		Extra:           []byte(payload.ExtraData),
		MixDigest:       common.Hash(payload.PrevRandao), // reused in merge
		Nonce:           ctypes.BlockNonce{},             // zero
		BaseFee:         (*uint256.Int)(&payload.BaseFeePerGas).ToBig(),
		WithdrawalsHash: &wroot,
	}
	execBlock := ctypes.NewBlockWithHeader(execHeader).WithBody(txs, nil).WithWithdrawals(withdrawals)
	if execBlockHash := execBlock.Hash(); execBlockHash != common.Hash(payload.BlockHash) {
		return nil, fmt.Errorf("Sanity check failed, payload hash does not match (expected %x, got %x)", common.Hash(payload.BlockHash), execBlockHash)
	}
	return execBlock, nil
}

// beaconBlockHash calculates the hash of a beacon block.
func beaconBlockHash(beaconBlock *capella.BeaconBlock) common.Hash {
	return common.Hash(beaconBlock.HashTreeRoot(configs.Mainnet, tree.GetHashFn()))
}

func callNewPayloadV2(client *rpc.Client, block *ctypes.Block) (string, error) {
	var resp engine.PayloadStatusV1
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err := client.CallContext(ctx, &resp, "engine_newPayloadV2", *engine.BlockToExecutableData(block, nil, nil).ExecutionPayload)
	cancel()
	return resp.Status, err
}

func callForkchoiceUpdatedV1(client *rpc.Client, headHash, finalizedHash common.Hash) (string, error) {
	var resp engine.ForkChoiceResponse
	update := engine.ForkchoiceStateV1{
		HeadBlockHash:      headHash,
		SafeBlockHash:      finalizedHash,
		FinalizedBlockHash: finalizedHash,
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err := client.CallContext(ctx, &resp, "engine_forkchoiceUpdatedV1", update, nil)
	cancel()
	return resp.PayloadStatus.Status, err
}
