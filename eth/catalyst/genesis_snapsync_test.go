// Copyright 2026 The go-ethereum Authors
// This file is part of the go-ethereum library.

package catalyst

import (
	"context"
	"math/big"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/eth/ethconfig"
	"github.com/ethereum/go-ethereum/params"
)

// TestGenesisSnapSyncBlockImport reproduces issue #28222 where a node starting at genesis
// with the default sync mode (SnapSync) gets stuck when receiving block payloads after genesis.
func TestGenesisSnapSyncBlockImport(t *testing.T) {
	genesis, _ := generateMergeChain(0, false)
	n, ethservice := startEthService(t, genesis, nil, func(cfg *ethconfig.Config) {
		cfg.SyncMode = ethconfig.SnapSync
	})
	defer n.Close()

	api := newConsensusAPIWithoutHeartbeat(ethservice)
	parent := ethservice.BlockChain().CurrentBlock()

	signer := types.LatestSigner(ethservice.BlockChain().Config())
	logCode := common.Hex2Bytes("60606040525b7f24ec1d3ff24c2f6ff210738839dbc339cd45a5294d85c79361016243157aae7b60405180905060405180910390a15b600a8060416000396000f360606040526008565b00")

	// Produce and import block #1 and block #2
	for i := 0; i < 2; i++ {
		statedb, _ := ethservice.BlockChain().StateAt(parent.Root, parent.Number, parent.Time)
		tx := types.MustSignNewTx(testKey, signer, &types.LegacyTx{
			Nonce:    statedb.GetNonce(testAddr),
			Value:    new(big.Int),
			Gas:      1000000,
			GasPrice: big.NewInt(2 * params.InitialBaseFee),
			Data:     logCode,
		})
		ethservice.TxPool().Add([]*types.Transaction{tx}, true)

		fcParams := engine.PayloadAttributes{
			Timestamp:             parent.Time + 1,
			Random:                crypto.Keccak256Hash([]byte{byte(i)}),
			SuggestedFeeRecipient: parent.Coinbase,
		}
		fcState := engine.ForkchoiceStateV1{
			HeadBlockHash:      parent.Hash(),
			SafeBlockHash:      common.Hash{},
			FinalizedBlockHash: common.Hash{},
		}

		resp, err := api.ForkchoiceUpdatedV1(context.Background(), fcState, &fcParams)
		if err != nil {
			t.Fatalf("error preparing payload at block %d: %v", i+1, err)
		}
		if resp.PayloadStatus.Status != engine.VALID {
			t.Fatalf("forkchoice invalid status at block %d: %v", i+1, resp.PayloadStatus.Status)
		}

		payload, err := api.getPayload(*resp.PayloadID, true, nil, nil)
		if err != nil {
			t.Fatalf("can't get payload at block %d: %v", i+1, err)
		}

		execResp, err := api.NewPayloadV1(context.Background(), *payload.ExecutionPayload)
		if err != nil {
			t.Fatalf("can't execute payload at block %d: %v", i+1, err)
		}
		if execResp.Status != engine.VALID {
			t.Fatalf("block %d rejected with status: %v (expected VALID)", i+1, execResp.Status)
		}

		fcState = engine.ForkchoiceStateV1{
			HeadBlockHash:      payload.ExecutionPayload.BlockHash,
			SafeBlockHash:      payload.ExecutionPayload.ParentHash,
			FinalizedBlockHash: payload.ExecutionPayload.ParentHash,
		}
		if _, err := api.ForkchoiceUpdatedV1(context.Background(), fcState, nil); err != nil {
			t.Fatalf("failed to update head at block %d: %v", i+1, err)
		}
		if ethservice.BlockChain().CurrentBlock().Number.Uint64() != payload.ExecutionPayload.Number {
			t.Fatalf("chain head not updated for block %d", i+1)
		}
		parent = ethservice.BlockChain().CurrentBlock()
	}
}
