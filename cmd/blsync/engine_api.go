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
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

func updateEngineApi(client *rpc.Client, headCh chan types.ChainHeadEvent) {
	for event := range headCh {
		if client == nil { // dry run, no engine API specified
			log.Info("New execution block retrieved", "block number", event.HeadBlock.Number, "block hash", event.HeadBlock.BlockHash, "finalized block hash", event.Finalized)
		} else {
			if status, err := callNewPayloadV2(client, event.HeadBlock); err == nil {
				log.Info("Successful NewPayload", "block number", event.HeadBlock.Number, "block hash", event.HeadBlock.BlockHash, "status", status)
			} else {
				log.Error("Failed NewPayload", "block number", event.HeadBlock.Number, "block hash", event.HeadBlock.BlockHash, "error", err)
			}
			if status, err := callForkchoiceUpdatedV1(client, event.HeadBlock.BlockHash, event.Finalized); err == nil {
				log.Info("Successful ForkchoiceUpdated", "head", event.HeadBlock.BlockHash, "status", status)
			} else {
				log.Error("Failed ForkchoiceUpdated", "head", event.HeadBlock.BlockHash, "error", err)
			}
		}
	}
}

func callNewPayloadV2(client *rpc.Client, execData *engine.ExecutableData) (string, error) {
	var resp engine.PayloadStatusV1
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	err := client.CallContext(ctx, &resp, "engine_newPayloadV2", execData)
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
