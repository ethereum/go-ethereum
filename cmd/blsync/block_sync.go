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

package main

import (
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
)

// beaconBlockSync implements request.Module; it fetches the beacon blocks belonging
// to the validated and prefetch heads.
type beaconBlockSync struct {
	recentBlocks *lru.Cache[common.Hash, *capella.BeaconBlock]
	locked       map[common.Hash]struct{}
	serverHeads  map[request.Server]common.Hash
	headTracker  headTracker

	lastHeadBlock *capella.BeaconBlock
	headBlockCh   chan *capella.BeaconBlock
}

type headTracker interface {
	PrefetchHead() types.HeadInfo
	ValidatedHead() types.SignedHeader
}

// newBeaconBlockSync returns a new beaconBlockSync.
func newBeaconBlockSync(headTracker headTracker) *beaconBlockSync {
	return &beaconBlockSync{
		headTracker:  headTracker,
		recentBlocks: lru.NewCache[common.Hash, *capella.BeaconBlock](10),
		locked:       make(map[common.Hash]struct{}),
		serverHeads:  make(map[request.Server]common.Hash),
		headBlockCh:  make(chan *capella.BeaconBlock, 1),
	}
}

// Process implements request.Module
func (s *beaconBlockSync) Process(tracker request.Tracker, events []request.Event) {
	// iterate events and add valid responses to recentBlocks
	for _, event := range events {
		switch event.Type {
		case request.EvResponse, request.EvFail, request.EvTimeout:
			_, req, resp := event.RequestInfo()
			blockRoot := common.Hash(req.(sync.ReqBeaconBlock))
			if resp != nil {
				block := resp.(*capella.BeaconBlock)
				s.recentBlocks.Add(blockRoot, block)
			}
			delete(s.locked, blockRoot)
		case sync.EvNewHead:
			s.serverHeads[event.Server] = event.Data.(types.HeadInfo).BlockRoot
		case request.EvUnregistered:
			delete(s.serverHeads, event.Server)
		}
	}

	// send validated head block or request it if unavailable
	if vh := s.headTracker.ValidatedHead(); vh != (types.SignedHeader{}) {
		validatedHead := vh.Header.Hash()
		if headBlock, ok := s.recentBlocks.Get(validatedHead); ok && headBlock != s.lastHeadBlock {
			select {
			case s.headBlockCh <- headBlock:
				s.lastHeadBlock = headBlock
			default:
			}
		} else {
			s.tryRequestBlock(tracker, validatedHead, false)
		}
	}
	// request prefetch head
	if prefetchHead := s.headTracker.PrefetchHead().BlockRoot; prefetchHead != (common.Hash{}) {
		s.tryRequestBlock(tracker, prefetchHead, true)
	}
}

// tryRequestBlock tries to send a block request for the given root if the block
// is not available and the root is not locked by another pending request.
// If prefetch is true then the request is only sent to a server whose latest
// announced head has the same block root. If prefetch is false then a validated
// block is requested which is expected to be available at every properly synced
// server, therefore no such restriction is applied.
func (s *beaconBlockSync) tryRequestBlock(tracker request.Tracker, blockRoot common.Hash, prefetch bool) {
	if _, ok := s.recentBlocks.Get(blockRoot); ok {
		return
	}
	if _, ok := s.locked[blockRoot]; ok {
		return
	}
	if _, ok := tracker.TryRequest(func(server request.Server) (request.Request, float32) {
		if prefetch && s.serverHeads[server] != blockRoot {
			// when requesting a not yet validated head, request it from someone
			// who has announced it already
			return nil, 0
		}
		return sync.ReqBeaconBlock(blockRoot), 0
	}); ok {
		s.locked[blockRoot] = struct{}{}
	}
}
