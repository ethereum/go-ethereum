// Copyright 2023 The go-ethereum Authors
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
	"sync/atomic"

	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
)

// beaconBlockSync implements request.Module; it fetches the beacon blocks belonging
// to the validated and prefetch heads. With p2pBlocks it fetches no blocks: the head
// events carry only the execution block hash proven by the light client update, and the
// execution client retrieves the block from its own peers. If it can't, the engine client
// asks for the block (fetchBlock), and it is fetched and sent after all.
type beaconBlockSync struct {
	recentBlocks *lru.Cache[common.Hash, *types.BeaconBlock]
	recentSlots  *lru.Cache[common.Hash, uint64] // p2pBlocks: slots of recently validated heads, by block root
	locked       map[common.Hash]request.ServerAndID
	serverHeads  map[request.Server]common.Hash
	headTracker  headTracker
	p2pBlocks    bool

	// p2pBlocks fallback: fallbackReq is set by fetchBlock; fallback is the head event
	// whose execution block is being fetched, from the beacon block fallbackRoot.
	fallbackReq  atomic.Pointer[types.ChainHeadEvent]
	fallback     *types.ChainHeadEvent
	fallbackRoot common.Hash
	fallbackSent bool // requested (again after a failed request, until a newer head)
	trigger      func()

	lastHeadInfo  types.HeadInfo
	chainHeadFeed event.FeedOf[types.ChainHeadEvent]
}

type headTracker interface {
	PrefetchHead() types.HeadInfo
	ValidatedOptimistic() (types.OptimisticUpdate, bool)
	ValidatedFinality() (types.FinalityUpdate, bool)
}

// newBeaconBlockSync returns a new beaconBlockSync.
func newBeaconBlockSync(headTracker headTracker, p2pBlocks bool) *beaconBlockSync {
	return &beaconBlockSync{
		headTracker:  headTracker,
		recentBlocks: lru.NewCache[common.Hash, *types.BeaconBlock](10),
		recentSlots:  lru.NewCache[common.Hash, uint64](10),
		locked:       make(map[common.Hash]request.ServerAndID),
		serverHeads:  make(map[request.Server]common.Hash),
		p2pBlocks:    p2pBlocks,
	}
}

func (s *beaconBlockSync) SubscribeChainHead(ch chan<- types.ChainHeadEvent) event.Subscription {
	return s.chainHeadFeed.Subscribe(ch)
}

// Process implements request.Module.
func (s *beaconBlockSync) Process(requester request.Requester, events []request.Event) {
	for _, event := range events {
		switch event.Type {
		case request.EvResponse, request.EvFail, request.EvTimeout:
			sid, req, resp := event.RequestInfo()
			blockRoot := common.Hash(req.(sync.ReqBeaconBlock))
			log.Debug("Beacon block event", "type", event.Type.Name, "hash", blockRoot)
			if resp != nil {
				s.recentBlocks.Add(blockRoot, resp.(*types.BeaconBlock))
			}
			if s.locked[blockRoot] == sid {
				delete(s.locked, blockRoot)
			}
			if resp == nil && s.fallback != nil && blockRoot == s.fallbackRoot {
				s.fallbackSent = false // failed or timed out: ask again (maybe another server)
			}
		case sync.EvNewHead:
			s.serverHeads[event.Server] = event.Data.(types.HeadInfo).BlockRoot
		case request.EvUnregistered:
			delete(s.serverHeads, event.Server)
		}
	}
	s.updateEventFeed()
	if s.p2pBlocks {
		s.updateFallback(requester)
		return
	}
	// request validated head block if unavailable and not yet requested
	if vh, ok := s.headTracker.ValidatedOptimistic(); ok {
		s.tryRequestBlock(requester, vh.Attested.Hash(), false)
	}
	// request prefetch head if the given server has announced it
	if prefetchHead := s.headTracker.PrefetchHead().BlockRoot; prefetchHead != (common.Hash{}) {
		s.tryRequestBlock(requester, prefetchHead, true)
	}
}

// fetchBlock asks for the execution block of a head event sent without it, which the
// execution client couldn't get from its peers (p2pBlocks). It is fetched from the beacon
// API and the event sent again with it, unless a newer head was sent meanwhile.
func (s *beaconBlockSync) fetchBlock(head types.ChainHeadEvent) {
	s.fallbackReq.Store(&head)
	if s.trigger != nil {
		s.trigger()
	}
}

// updateFallback requests the beacon block carrying the execution block asked for by
// fetchBlock (again if a request fails, until a newer head is sent), and sends the head
// again with it when it arrives.
func (s *beaconBlockSync) updateFallback(requester request.Requester) {
	if head := s.fallbackReq.Swap(nil); head != nil {
		s.fallback, s.fallbackRoot, s.fallbackSent = head, head.BeaconHead.Hash(), false
	}
	if s.fallback == nil {
		return
	}
	if s.lastHeadInfo.BlockRoot != s.fallback.BeaconHead.Hash() {
		s.fallback = nil // a newer head was sent
		return
	}
	block, ok := s.recentBlocks.Get(s.fallbackRoot)
	if !ok {
		if !s.fallbackSent {
			s.fallbackSent = s.tryRequestBlock(requester, s.fallbackRoot, false)
		}
		return
	}
	head := *s.fallback
	s.fallback = nil
	payload, err := block.ExecutionPayload()
	if err != nil || payload.Hash() != head.ExecHash {
		log.Warn("Beacon block doesn't carry the head's execution block", "root", s.fallbackRoot, "head", head.ExecHash, "error", err)
		return
	}
	head.BeaconHead, head.Block, head.ExecRequests = block.Header(), payload, block.ExecutionRequestsList()
	s.chainHeadFeed.Send(head)
}

// tryRequestBlock requests a beacon block unless it is at hand or already requested, and
// returns whether it sent a request.
func (s *beaconBlockSync) tryRequestBlock(requester request.Requester, blockRoot common.Hash, needSameHead bool) bool {
	if _, ok := s.recentBlocks.Get(blockRoot); ok {
		return false
	}
	if _, ok := s.locked[blockRoot]; ok {
		return false
	}
	for _, server := range requester.CanSendTo() {
		if needSameHead && (s.serverHeads[server] != blockRoot) {
			continue
		}
		id := requester.Send(server, sync.ReqBeaconBlock(blockRoot))
		s.locked[blockRoot] = request.ServerAndID{Server: server, ID: id}
		return true
	}
	return false
}

func blockHeadInfo(block *types.BeaconBlock) types.HeadInfo {
	if block == nil {
		return types.HeadInfo{}
	}
	return types.HeadInfo{Slot: block.Slot(), BlockRoot: block.Root()}
}

func (s *beaconBlockSync) updateEventFeed() {
	optimistic, ok := s.headTracker.ValidatedOptimistic()
	if !ok {
		return
	}

	validatedHead := optimistic.Attested.Hash()
	var headBlock *types.BeaconBlock
	if s.p2pBlocks {
		s.recentSlots.Add(validatedHead, optimistic.Attested.Slot)
	} else if headBlock, ok = s.recentBlocks.Get(validatedHead); !ok {
		return
	}

	var finalizedHash common.Hash
	if finality, ok := s.headTracker.ValidatedFinality(); ok {
		he := optimistic.Attested.Epoch()
		fe := finality.Attested.Header.Epoch()
		switch {
		case he == fe:
			finalizedHash = finality.Finalized.PayloadHeader.BlockHash()
		case he < fe:
			return
		case he == fe+1:
			parentSlot, ok := s.parentSlot(optimistic.Attested.ParentRoot)
			if !ok || parentSlot/params.EpochLength == fe {
				return // head is at first slot of next epoch, wait for finality update
			}
		}
	}

	headInfo := types.HeadInfo{Slot: optimistic.Attested.Slot, BlockRoot: validatedHead}
	if headInfo == s.lastHeadInfo {
		return
	}
	s.lastHeadInfo = headInfo

	if s.p2pBlocks {
		// only the execution block hash, proven by the light client update
		var execHash common.Hash
		if optimistic.Attested.PayloadHeader != nil {
			execHash = optimistic.Attested.PayloadHeader.BlockHash()
		}
		if execHash == (common.Hash{}) {
			log.Error("Validated beacon head has no execution block hash", "slot", headInfo.Slot, "root", validatedHead)
			return
		}
		s.chainHeadFeed.Send(types.ChainHeadEvent{
			BeaconHead: optimistic.Attested.Header,
			ExecHash:   execHash,
			Finalized:  finalizedHash,
		})
		return
	}
	// new head block and finality info available; extract executable data and send event to feed
	execBlock, err := headBlock.ExecutionPayload()
	if err != nil {
		log.Error("Error extracting execution block from validated beacon block", "error", err)
		return
	}
	s.chainHeadFeed.Send(types.ChainHeadEvent{
		BeaconHead:   optimistic.Attested.Header,
		ExecHash:     execBlock.Hash(),
		Block:        execBlock,
		ExecRequests: headBlock.ExecutionRequestsList(),
		Finalized:    finalizedHash,
	})
}

// parentSlot returns the slot of a recent beacon block, from the fetched blocks or (with
// p2pBlocks) the validated heads.
func (s *beaconBlockSync) parentSlot(root common.Hash) (uint64, bool) {
	if block, ok := s.recentBlocks.Get(root); ok {
		return block.Slot(), true
	}
	return s.recentSlots.Get(root)
}
