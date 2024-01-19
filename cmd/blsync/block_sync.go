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
	headCh   chan headData
}

type headData struct {
	block  *capella.BeaconBlock
	update types.FinalityUpdate
}

type headTracker interface {
	PrefetchHead() types.HeadInfo
	ValidatedHead() (types.SignedHeader, bool)
	ValidatedFinality() (types.FinalityUpdate, bool)
}

// newBeaconBlockSync returns a new beaconBlockSync.
func newBeaconBlockSync(headTracker headTracker) *beaconBlockSync {
	return &beaconBlockSync{
		headTracker:  headTracker,
		recentBlocks: lru.NewCache[common.Hash, *capella.BeaconBlock](10),
		locked:       make(map[common.Hash]struct{}),
		serverHeads:  make(map[request.Server]common.Hash),
		headCh:  make(chan headData, 1),
	}
}

func (s *beaconBlockSync) Process(events []request.Event) {
	for _, event := range events {
		switch event.Type {
		case request.EvRequest:
			_, req, _ := event.RequestInfo()
			blockRoot := common.Hash(req.(sync.ReqBeaconBlock))
			s.locked[blockRoot] = struct{}{}
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

	// send validated head block
	head, ok := s.headTracker.ValidatedHead()
	if !ok {
		return
	}
	finality, ok := s.headTracker.ValidatedFinality()	//TODO fetch directly if subscription does not deliver
	if !ok || head.Header.Epoch() != finality.Attested.Header.Epoch() {
		return
	}
	validatedHead := head.Header.Hash()
	headBlock, ok := s.recentBlocks.Get(validatedHead)
	if !ok || headBlock == s.lastHeadBlock {
		return
	}
	select {
	case s.headCh <- headData{block: headBlock, update: finality}:
		s.lastHeadBlock = headBlock
	default:
	}
}

func (s *beaconBlockSync) MakeRequest(server request.Server) (request.Request, float32) {
	// request validated head block if unavailable and not yet requested
	if vh, ok := s.headTracker.ValidatedHead(); ok {
		validatedHead := vh.Header.Hash()
		if _, ok := s.recentBlocks.Get(validatedHead); !ok {
			if _, ok := s.locked[validatedHead]; !ok {
				return sync.ReqBeaconBlock(validatedHead), 1
			}
		}
	}
	// request prefetch head if the given server has announced it
	if prefetchHead := s.headTracker.PrefetchHead().BlockRoot; prefetchHead != (common.Hash{}) && prefetchHead == s.serverHeads[server] {
		if _, ok := s.recentBlocks.Get(prefetchHead); !ok {
			if _, ok := s.locked[prefetchHead]; !ok {
				return sync.ReqBeaconBlock(prefetchHead), 0
			}
		}
	}
	return nil, 0
}
