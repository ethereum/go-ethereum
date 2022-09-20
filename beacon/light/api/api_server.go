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

package api

import (
	"sync/atomic"

	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

type ApiServer struct {
	api           *BeaconLightApi
	eventCallback func(event request.Event)
	unsubscribe   func()
	lastId        uint64
}

func NewApiServer(api *BeaconLightApi) *ApiServer {
	return &ApiServer{api: api}
}

func (s *ApiServer) Subscribe(eventCallback func(event request.Event)) {
	s.eventCallback = eventCallback
	s.unsubscribe = s.api.StartHeadListener(func(slot uint64, blockRoot common.Hash) {
		eventCallback(request.Event{Type: sync.EvNewHead, Data: types.HeadInfo{Slot: slot, BlockRoot: blockRoot}})
	}, func(head types.SignedHeader) {
		eventCallback(request.Event{Type: sync.EvNewSignedHead, Data: head})
	}, func(err error) {
		log.Warn("Head event stream error", "err", err)
	})
}

func (s *ApiServer) SendRequest(req request.Request) request.ID {
	id := request.ID(atomic.AddUint64(&s.lastId, 1))
	go func() {
		var resp request.Response
		switch data := req.(type) {
		case sync.ReqUpdates:
			if updates, committees, err := s.api.GetBestUpdatesAndCommittees(data.FirstPeriod, data.Count); err == nil {
				resp = sync.RespUpdates{Updates: updates, Committees: committees}
			}
		/*case sync.ReqOptimisticHead:
		if signedHead, err := s.api.GetOptimisticHeadUpdate(); err == nil {
			resp = signedHead
		}*/ //TODO ???
		case sync.ReqHeader:
			if header, err := s.api.GetHeader(common.Hash(data)); err == nil {
				resp = header
			}
		case sync.ReqCheckpointData:
			if bootstrap, err := s.api.GetCheckpointData(common.Hash(data)); err == nil {
				resp = bootstrap
			}
		case sync.ReqBeaconBlock:
			if block, err := s.api.GetBeaconBlock(common.Hash(data)); err == nil {
				resp = block
			}
		default:
		}
		if resp != nil {
			s.eventCallback(request.Event{Type: request.EvResponse, Data: request.IdAndResponse{ID: id, Response: resp}})
		} else {
			s.eventCallback(request.Event{Type: request.EvFail, Data: id})
		}
	}()
	return id
}

// Note: UnsubscribeHeads should not be called concurrently with SubscribeHeads
func (s *ApiServer) Unsubscribe() {
	if s.unsubscribe != nil {
		s.unsubscribe()
		s.unsubscribe = nil
	}
}
