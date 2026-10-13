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
	"github.com/ethereum/go-ethereum/common/mclock"
	ctypes "github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rpc"
)

type engineClient struct {
	config     *params.ClientConfig
	rpc        *rpc.Client
	clock      mclock.Clock
	fetchBlock blockFetcher // P2PBlocks: get a block from the beacon API instead

	// fallbackPaused is set when the execution client couldn't import a block fetched for
	// it (it is still syncing, or lacks the parent or its state), until a head is VALID.
	fallbackPaused bool
	rootCtx        context.Context
	cancelRoot     context.CancelFunc
	wg             sync.WaitGroup
}

// blockFetcher asks for an execution block from the beacon API (P2PBlocks): the one with the
// hash execHash, in the beacon block root. It is sent back as a head event with the block,
// unless a head newer than head was sent meanwhile.
type blockFetcher func(head types.ChainHeadEvent, root, execHash common.Hash)

func startEngineClient(config *params.ClientConfig, rpc *rpc.Client, headCh <-chan types.ChainHeadEvent, fetchBlock blockFetcher) *engineClient {
	return startEngineClientWithClock(config, rpc, headCh, fetchBlock, mclock.System{})
}

func startEngineClientWithClock(config *params.ClientConfig, rpc *rpc.Client, headCh <-chan types.ChainHeadEvent, fetchBlock blockFetcher, clock mclock.Clock) *engineClient {
	ctx, cancel := context.WithCancel(context.Background())
	ec := &engineClient{
		config:     config,
		rpc:        rpc,
		clock:      clock,
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
// repeat the update until it is VALID or a new head arrives, at these times after the head
// and then every p2pRepeatEvery. A synced node has the block within a second or two; a node
// that is still syncing stays SYNCING, so the repeats back off and log at debug level only.
//
// If the head isn't VALID at the p2pFallbackAt-th repeat, the execution client's peers didn't
// provide the block: fetch it from the beacon API and send it with newPayload (once per head).
// If its parent isn't the latest VALID head but a block after it, as when blsync skipped a
// head, the execution client lacks that too: fetch the blocks back to the latest VALID head
// as well, if at most p2pFallbackDepth slots lie between them, and send them oldest first. A
// new head starts over, with the blocks fetched so far at hand in the block cache; if they
// are fetched much slower than the heads come, a large gap can outgrow the bound before it is
// filled in, and only peers can fill it. If the execution client answers that it can't import
// the blocks (SYNCING or ACCEPTED: it is still syncing, or lacks a parent or its state),
// fetching blocks is paused until a head is VALID again.
var p2pRepeats = []time.Duration{time.Second, 2 * time.Second, 4 * time.Second, 8 * time.Second}

const (
	p2pRepeatEvery   = 8 * time.Second
	p2pFallbackAt    = 3
	p2pFallbackDepth = 32 // the most slots between the latest VALID head and a head to fill in: an epoch
)

func (ec *engineClient) updateLoop(headCh <-chan types.ChainHeadEvent) {
	defer ec.wg.Done()

	var (
		pending types.ChainHeadEvent // P2PBlocks: the latest head
		valid   types.ChainHeadEvent // P2PBlocks: the latest head that was VALID
		repeat  mclock.ChanTimer     // P2PBlocks: the next repeat of the pending head's update
		repeats int
		fetched []types.ChainHeadEvent // P2PBlocks: the pending head's block, then the ones before it
	)
	setRepeat := func(d time.Duration) {
		if repeat != nil {
			repeat.Stop()
			repeat = nil
		}
		if d > 0 {
			repeat = ec.clock.NewTimer(d)
		}
	}
	defer setRepeat(0)

	for {
		var repeatCh <-chan mclock.AbsTime
		if repeat != nil {
			repeatCh = repeat.C()
		}
		select {
		case <-ec.rootCtx.Done():
			log.Debug("Stopping engine API update loop")
			return

		case event := <-headCh:
			if ec.rpc == nil { // dry run, no engine API specified
				logCtx := []any{"hash", event.ExecHash, "finalized", event.Finalized}
				if event.Block != nil {
					logCtx = append([]any{"number", event.Block.NumberU64()}, logCtx...)
				}
				log.Info("New execution block retrieved", logCtx...)
				continue
			}
			if ec.fetchBlock != nil && event.Block != nil {
				// With P2PBlocks, only the fallback sends blocks: the pending head's, then
				// the ones before it that the execution client lacks too.
				want := pending.ExecHash
				if len(fetched) > 0 {
					want = fetched[len(fetched)-1].Block.ParentHash()
				}
				if event.ExecHash != want || pending.ExecHash == valid.ExecHash {
					log.Debug("Dropping a fetched block no longer asked for", "hash", event.ExecHash)
					continue
				}
				if fetched = append(fetched, event); ec.fetchParent(event, pending, valid) {
					continue
				}
				if event.Block.ParentHash() != valid.ExecHash {
					// They don't lead back to the latest VALID head (no VALID head that
					// recent, or another branch): send the head's block only. An earlier
					// block whose parent geth lacks would make its beacon sync go back to it.
					fetched = fetched[:1]
				}
				payloadStatus, status := ec.sendBlocks(fetched, pending)
				fetched = nil
				switch {
				case status == engine.VALID:
					valid = pending
					ec.resumeFallback()
					setRepeat(0)
				case (payloadStatus == engine.SYNCING || payloadStatus == engine.ACCEPTED) && !ec.fallbackPaused:
					// The pending head's repeats go on; its block was asked for once already.
					log.Info("Execution client is catching up, not fetching head blocks until it has synced", "status", payloadStatus)
					ec.fallbackPaused = true
				}
				continue
			}
			setRepeat(0) // a new head ends the repeats of the previous one
			pending, fetched = event, nil
			if status := ec.sendHead(event); status == engine.VALID {
				valid = event
				ec.resumeFallback()
			} else if event.Block == nil {
				repeats = 0
				setRepeat(p2pRepeats[0])
			}

		case <-repeatCh:
			repeat = nil
			repeats++
			log.Debug("Repeating ForkchoiceUpdated", "head", pending.ExecHash, "attempt", repeats)
			status := ec.forkchoiceUpdated(pending, true)
			if status == engine.VALID {
				valid, fetched = pending, nil
				ec.resumeFallback()
				continue
			}
			if repeats == p2pFallbackAt && ec.fetchBlock != nil && !ec.fallbackPaused {
				log.Info("Execution client lacks the head block, fetching it from the beacon API", "head", pending.ExecHash, "status", status)
				ec.fetchBlock(pending, pending.BeaconHead.Hash(), pending.ExecHash)
			}
			next := p2pRepeatEvery
			if repeats < len(p2pRepeats) {
				next = p2pRepeats[repeats] - p2pRepeats[repeats-1]
			}
			setRepeat(next)
		}
	}
}

// fetchParent asks for the parent of a block fetched for the head (P2PBlocks) if the
// execution client lacks it too: if it isn't the latest VALID head but a block after it, and
// at most p2pFallbackDepth slots lie between the latest VALID head and the head. It returns
// whether it did. It doesn't without a VALID head that recent (the execution client is
// syncing), or past its slot (the block is on another branch).
func (ec *engineClient) fetchParent(block, head, valid types.ChainHeadEvent) bool {
	if valid.ExecHash == (common.Hash{}) || block.Block.ParentHash() == valid.ExecHash ||
		block.BeaconHead.Slot <= valid.BeaconHead.Slot+1 || head.BeaconHead.Slot > valid.BeaconHead.Slot+1+p2pFallbackDepth {
		return false
	}
	if block.ExecHash == head.ExecHash {
		log.Info("Execution client lacks blocks before the head too, fetching them from the beacon API", "head", head.ExecHash, "valid", valid.ExecHash)
	}
	log.Debug("Fetching a missing parent from the beacon API", "number", block.Block.NumberU64()-1, "hash", block.Block.ParentHash())
	ec.fetchBlock(head, block.BeaconHead.ParentRoot, block.Block.ParentHash())
	return true
}

func (ec *engineClient) resumeFallback() {
	if ec.fallbackPaused {
		log.Info("Execution client has synced, fetching missing head blocks again")
		ec.fallbackPaused = false
	}
}

// sendHead hands a new head to the execution client: the block if there is one, then the
// forkchoice update. It returns the forkchoice status.
func (ec *engineClient) sendHead(event types.ChainHeadEvent) string {
	// Without the block (P2PBlocks), the forkchoice update alone makes the execution client
	// fetch it from its peers.
	if event.Block != nil {
		ec.newPayload(event)
	}
	return ec.forkchoiceUpdated(event, false)
}

// sendBlocks sends blocks fetched for a head, given newest first, with newPayload: oldest
// first, up to the first one that isn't VALID; then the head's forkchoice update. It returns
// the last newPayload status and the forkchoice status.
func (ec *engineClient) sendBlocks(blocks []types.ChainHeadEvent, head types.ChainHeadEvent) (payloadStatus, status string) {
	for i := len(blocks) - 1; i >= 0; i-- {
		if payloadStatus = ec.newPayload(blocks[i]); payloadStatus != engine.VALID {
			break
		}
	}
	return payloadStatus, ec.forkchoiceUpdated(head, false)
}

// newPayload sends a block to the execution client and returns its status (empty after an
// error).
func (ec *engineClient) newPayload(event types.ChainHeadEvent) string {
	log.Debug("Calling NewPayload", "number", event.Block.NumberU64(), "hash", event.ExecHash)
	status, err := ec.callNewPayload(ec.forkName(event), event)
	if err != nil {
		log.Error("Failed NewPayload", "number", event.Block.NumberU64(), "hash", event.ExecHash, "error", err)
		return ""
	}
	log.Info("Successful NewPayload", "number", event.Block.NumberU64(), "hash", event.ExecHash, "status", status)
	return status
}

// forkchoiceUpdated sends the forkchoice update for a head and returns its status. A
// repeat that doesn't make the head VALID logs at debug level only.
func (ec *engineClient) forkchoiceUpdated(event types.ChainHeadEvent, repeat bool) string {
	log.Debug("Calling ForkchoiceUpdated", "head", event.ExecHash)
	status, err := ec.callForkchoiceUpdated(ec.forkName(event), event)
	if err == nil {
		if repeat && status != engine.VALID {
			log.Debug("Successful ForkchoiceUpdated", "head", event.ExecHash, "status", status)
		} else {
			log.Info("Successful ForkchoiceUpdated", "head", event.ExecHash, "status", status)
		}
		return status
	}
	if repeat || err.Error() == "beacon syncer reorging" {
		log.Debug("Failed ForkchoiceUpdated", "head", event.ExecHash, "error", err)
		// a repeat can time out while the execution client is busy syncing; "beacon syncer
		// reorging" can occur if blsync is skipping a block
		return status
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
