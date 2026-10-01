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
	"github.com/ethereum/go-ethereum/common/hexutil"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type engineClient struct {
	config     *params.ClientConfig
	rpc        *rpc.Client
	fetchBlock func(types.ChainHeadEvent) // P2PBlocks: get a head's block from the beacon API instead
	rootCtx    context.Context
	cancelRoot context.CancelFunc
	wg         sync.WaitGroup
}

func startEngineClient(config *params.ClientConfig, rpc *rpc.Client, headCh <-chan types.ChainHeadEvent, fetchBlock func(types.ChainHeadEvent)) *engineClient {
	ctx, cancel := context.WithCancel(context.Background())
	ec := &engineClient{
		config:     config,
		rpc:        rpc,
		fetchBlock: fetchBlock,
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

// Without the block (P2PBlocks), geth answers the forkchoice update with SYNCING while it
// fetches the block, and records the head, safe and finalized blocks only for a known head:
// repeat the update until it is VALID (or the next head arrives). If it isn't VALID after
// p2pFallbackAfter repeats, the execution client's peers didn't provide the block (or it is
// still catching up): fetch the block from the beacon API and send it with newPayload.
const (
	p2pRetryInterval = time.Second
	p2pRetries       = 10
	p2pFallbackAfter = 4
)

func (ec *engineClient) updateLoop(headCh <-chan types.ChainHeadEvent) {
	defer ec.wg.Done()

	var (
		pending types.ChainHeadEvent // P2PBlocks: head still being fetched by the execution client
		retry   <-chan time.Time
		retries int
	)
	for {
		select {
		case <-ec.rootCtx.Done():
			log.Debug("Stopping engine API update loop")
			return

		case event := <-headCh:
			retry = nil
			if ec.rpc == nil { // dry run, no engine API specified
				log.Info("New execution block retrieved", "hash", event.ExecHash, "finalized", event.Finalized)
				continue
			}
			if status := ec.sendHead(event); event.Block == nil && status != engine.VALID {
				pending, retries, retry = event, 0, time.After(p2pRetryInterval)
			}

		case <-retry:
			retry = nil
			retries++
			log.Debug("Repeating ForkchoiceUpdated", "head", pending.ExecHash, "attempt", retries)
			if status := ec.forkchoiceUpdated(pending); status != engine.VALID && retries < p2pRetries {
				if retries == p2pFallbackAfter && ec.fetchBlock != nil {
					log.Info("Execution client lacks the head block, fetching it from the beacon API", "head", pending.ExecHash, "status", status)
					ec.fetchBlock(pending)
				}
				retry = time.After(p2pRetryInterval)
			}
		}
	}
}

// sendHead hands a new head to the execution client: the block if there is one, then the
// forkchoice update. It returns the forkchoice status.
func (ec *engineClient) sendHead(event types.ChainHeadEvent) string {
	// Without the block (P2PBlocks), the forkchoice update alone makes the execution client
	// fetch it from its peers.
	if event.Block != nil {
		forkName := ec.forkName(event)
		log.Debug("Calling NewPayload", "number", event.Block.NumberU64(), "hash", event.ExecHash)
		if status, err := ec.callNewPayload(forkName, event); err == nil {
			log.Info("Successful NewPayload", "number", event.Block.NumberU64(), "hash", event.ExecHash, "status", status)
		} else {
			log.Error("Failed NewPayload", "number", event.Block.NumberU64(), "hash", event.ExecHash, "error", err)
		}
	}
	return ec.forkchoiceUpdated(event)
}

// forkchoiceUpdated sends the forkchoice update for a head and returns its status.
func (ec *engineClient) forkchoiceUpdated(event types.ChainHeadEvent) string {
	log.Debug("Calling ForkchoiceUpdated", "head", event.ExecHash)
	status, err := ec.callForkchoiceUpdated(ec.forkName(event), event)
	if err == nil {
		log.Info("Successful ForkchoiceUpdated", "head", event.ExecHash, "status", status)
		return status
	}
	if err.Error() == "beacon syncer reorging" {
		log.Debug("Failed ForkchoiceUpdated", "head", event.ExecHash, "error", err)
		return status // ignore beacon syncer reorging errors, this error can occur if the blsync is skipping a block
	}
	log.Error("Failed ForkchoiceUpdated", "head", event.ExecHash, "error", err)
	return status
}

func (ec *engineClient) forkName(event types.ChainHeadEvent) string {
	return strings.ToLower(ec.config.ForkAtEpoch(event.BeaconHead.Epoch()).Name)
}

func (ec *engineClient) callNewPayload(fork string, event types.ChainHeadEvent) (string, error) {
	execData := engine.BlockToExecutableData(event.Block, nil, nil, nil).ExecutionPayload

	var (
		method string
		params = []any{execData}
	)
	switch fork {
	case "altair", "bellatrix":
		method = "engine_newPayloadV1"
	case "capella":
		method = "engine_newPayloadV2"
	case "deneb":
		method = "engine_newPayloadV3"
		parentBeaconRoot := event.BeaconHead.ParentRoot
		blobHashes := collectBlobHashes(event.Block)
		params = append(params, blobHashes, parentBeaconRoot)
	default: // electra, fulu and above
		method = "engine_newPayloadV4"
		parentBeaconRoot := event.BeaconHead.ParentRoot
		blobHashes := collectBlobHashes(event.Block)
		hexRequests := make([]hexutil.Bytes, len(event.ExecRequests))
		for i := range event.ExecRequests {
			hexRequests[i] = hexutil.Bytes(event.ExecRequests[i])
		}
		params = append(params, blobHashes, parentBeaconRoot, hexRequests)
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
		HeadBlockHash:      event.ExecHash,
		SafeBlockHash:      event.Finalized,
		FinalizedBlockHash: event.Finalized,
	}

	var method string
	switch fork {
	case "altair", "bellatrix":
		method = "engine_forkchoiceUpdatedV1"
	case "capella":
		method = "engine_forkchoiceUpdatedV2"
	default: // deneb, electra, fulu and above
		method = "engine_forkchoiceUpdatedV3"
	}

	ctx, cancel := context.WithTimeout(ec.rootCtx, time.Second*5)
	defer cancel()
	var resp engine.ForkChoiceResponse
	err := ec.rpc.CallContext(ctx, &resp, method, update, nil)
	return resp.PayloadStatus.Status, err
}
