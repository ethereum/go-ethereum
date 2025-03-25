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

package blsync

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type engineClient struct {
	config     *params.ClientConfig
	rpc        *rpc.Client
	rootCtx    context.Context
	cancelRoot context.CancelFunc
	wg         sync.WaitGroup
}

func startEngineClient(config *params.ClientConfig, rpc *rpc.Client, headCh <-chan types.ChainHeadEvent) *engineClient {
	ctx, cancel := context.WithCancel(context.Background())
	ec := &engineClient{
		config:     config,
		rpc:        rpc,
		rootCtx:    ctx,
		cancelRoot: cancel,
	}
	ec.wg.Add(1)
	go ec.updateLoop(headCh)
	return ec
}

func (ec *engineClient) stop() {
	ec.cancelRoot()
	ec.wg.Wait()
}

func (ec *engineClient) updateLoop(headCh <-chan types.ChainHeadEvent) {
	defer ec.wg.Done()

	for {
		select {
		case <-ec.rootCtx.Done():
			log.Debug("Stopping engine API update loop")
			return

		case event := <-headCh:
			if ec.rpc == nil { // dry run, no engine API specified
				log.Info("New execution block retrieved", "number", event.Block.NumberU64(), "hash", event.Block.Hash(), "finalized", event.Finalized)
				continue
			}

			fork := ec.config.ForkAtEpoch(event.BeaconHead.Epoch())
			forkName := strings.ToLower(fork.Name)

			log.Debug("Calling NewPayload", "number", event.Block.NumberU64(), "hash", event.Block.Hash())
			if status, err := ec.callNewPayload(forkName, event); err == nil {
				log.Info("Successful NewPayload", "number", event.Block.NumberU64(), "hash", event.Block.Hash(), "status", status)
			} else {
				log.Error("Failed NewPayload", "number", event.Block.NumberU64(), "hash", event.Block.Hash(), "error", err)
			}

			log.Debug("Calling ForkchoiceUpdated", "head", event.Block.Hash())
			if status, err := ec.callForkchoiceUpdated(forkName, event); err == nil {
				log.Info("Successful ForkchoiceUpdated", "head", event.Block.Hash(), "status", status)
			} else {
				log.Error("Failed ForkchoiceUpdated", "head", event.Block.Hash(), "error", err)
			}
		}
	}
}

func (ec *engineClient) callNewPayload(fork string, event types.ChainHeadEvent) (string, error) {
	execData := engine.BlockToExecutableData(event.Block, nil, nil, nil).ExecutionPayload

	var (
		method string
		params = []any{execData}
	)
	switch fork {
	case "electra":
		method = "engine_newPayloadV4"
		parentBeaconRoot := event.BeaconHead.ParentRoot
		blobHashes := collectBlobHashes(event.Block)
		params = append(params, blobHashes, parentBeaconRoot, event.ExecRequests)
	case "deneb":
		method = "engine_newPayloadV3"
		parentBeaconRoot := event.BeaconHead.ParentRoot
		blobHashes := collectBlobHashes(event.Block)
		params = append(params, blobHashes, parentBeaconRoot)
	case "capella":
		method = "engine_newPayloadV2"
	default:
		method = "engine_newPayloadV1"
	}

	ctx, cancel := context.WithTimeout(ec.rootCtx, time.Second*5)
	defer cancel()
	var resp engine.PayloadStatusV1
	err := ec.rpc.CallContext(ctx, &resp, method, params...)
	return resp.Status, err
}

func collectBlobHashes(b *ctypes.Block) []common.Hash {
	list := make([]common.Hash, 0)
	for _, tx := range b.Transactions() {
		list = append(list, tx.BlobHashes()...)
	}
	return list
}

func (ec *engineClient) callForkchoiceUpdated(fork string, event types.ChainHeadEvent) (string, error) {
	update := engine.ForkchoiceStateV1{
		HeadBlockHash:      event.Block.Hash(),
		SafeBlockHash:      event.Finalized,
		FinalizedBlockHash: event.Finalized,
	}

	var method string
	switch fork {
	case "deneb", "electra":
		method = "engine_forkchoiceUpdatedV3"
	case "capella":
		method = "engine_forkchoiceUpdatedV2"
	default:
		method = "engine_forkchoiceUpdatedV1"
	}

	ctx, cancel := context.WithTimeout(ec.rootCtx, time.Second*5)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err := ec.rpc.CallContext(ctx, &resp, method, update, nil)
	return resp.PayloadStatus.Status, err
}
