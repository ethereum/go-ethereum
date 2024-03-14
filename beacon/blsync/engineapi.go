package blsync

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/engine"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type engineClient struct {
	config     *lightClientConfig
	rpc        *rpc.Client
	rootCtx    context.Context
	cancelRoot context.CancelFunc
	wg         sync.WaitGroup
}

func startEngineClient(config *lightClientConfig, rpc *rpc.Client, headCh <-chan types.ChainHeadEvent) *engineClient {
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
			return

		case event := <-headCh:
			if ec.rpc == nil { // dry run, no engine API specified
				log.Info("New execution block retrieved", "number", event.Block.NumberU64(), "hash", event.Block.Hash(), "finalized", event.Finalized)
				continue
			}

			fork := ec.config.ForkAtEpoch(event.BeaconHead.Epoch())
			forkName := strings.ToLower(fork.Name)

			if status, err := ec.callNewPayload(forkName, event); err == nil {
				log.Info("Successful NewPayload", "number", event.Block.NumberU64(), "hash", event.Block.Hash(), "status", status)
			} else {
				log.Error("Failed NewPayload", "number", event.Block.NumberU64(), "hash", event.Block.Hash(), "error", err)
			}

			if status, err := ec.callForkchoiceUpdated(forkName, event); err == nil {
				log.Info("Successful ForkchoiceUpdated", "head", event.Block.Hash(), "status", status)
			} else {
				log.Error("Failed ForkchoiceUpdated", "head", event.Block.Hash(), "error", err)
			}
		}
	}
}

func (ec *engineClient) callNewPayload(fork string, event types.ChainHeadEvent) (string, error) {
	execData := engine.BlockToExecutableData(event.Block, nil, nil).ExecutionPayload

	var (
		method string
		params = []any{execData}
	)
	switch fork {
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

func collectBlobHashes(b *ctypes.Block) (list []common.Hash) {
	for _, tx := range b.Transactions() {
		for _, h := range tx.BlobHashes() {
			list = append(list, h)
		}
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
	case "deneb":
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
