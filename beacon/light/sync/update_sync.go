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

package sync

import (
	"sort"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

const maxUpdateRequest = 8

type committeeChain interface {
	CheckpointInit(bootstrap types.BootstrapData) error
	InsertUpdate(update *types.LightClientUpdate, nextCommittee *types.SerializedSyncCommittee) error
	NextSyncPeriod() (uint64, bool)
}

type CheckpointInit struct {
	chain          committeeChain
	checkpointHash common.Hash
	locked         bool
	initialized    bool
}

func NewCheckpointInit(chain committeeChain, checkpointHash common.Hash) *CheckpointInit {
	return &CheckpointInit{
		chain:          chain,
		checkpointHash: checkpointHash,
	}
}

// Process implements request.Module
func (s *CheckpointInit) Process(tracker request.Tracker, events []request.Event) bool {
	if s.initialized {
		return false
	}
	for _, event := range events {
		if !event.IsRequestEvent() {
			continue
		}
		s.locked = false
		sid, request, response := event.RequestInfo()
		if response != nil {
			if checkpoint, ok := response.(*types.BootstrapData); ok && checkpoint.Header.Hash() == common.Hash(request.(ReqCheckpointData)) {
				s.chain.CheckpointInit(*checkpoint) //TODO
				s.initialized = true
				return true
			}
			tracker.InvalidResponse(sid, "invalid checkpoint data")
		}
	}
	if !s.locked {
		if _, ok := tracker.TryRequest(func(server request.Server) (request.Request, float32) {
			return ReqCheckpointData(s.checkpointHash), 0
		}); ok {
			s.locked = true
		}
	}
	return false
}

type ForwardUpdateSync struct {
	chain          committeeChain
	rangeLock      rangeLock
	lockedIDs      map[request.ServerAndID]struct{}
	processQueue   []request.Event
	nextSyncPeriod map[request.Server]uint64
}

func NewForwardUpdateSync(chain committeeChain) *ForwardUpdateSync {
	return &ForwardUpdateSync{
		chain:          chain,
		rangeLock:      make(rangeLock),
		lockedIDs:      make(map[request.ServerAndID]struct{}),
		nextSyncPeriod: make(map[request.Server]uint64),
	}
}

type rangeLock map[uint64]int

func (r rangeLock) lock(first, count uint64, add int) {
	for i := first; i < first+count; i++ {
		if v := r[i] + add; v > 0 {
			r[i] = v
		} else {
			delete(r, i)
		}
	}
}

func (r rangeLock) firstUnlocked(start, maxCount uint64) (first, count uint64) {
	first = start
	for {
		if _, ok := r[first]; !ok {
			break
		}
		first++
	}
	for {
		count++
		if count == maxCount {
			break
		}
		if _, ok := r[first+count]; ok {
			break
		}
	}
	return
}

func (s *ForwardUpdateSync) lockRange(sid request.ServerAndID, req request.Request) {
	if _, ok := s.lockedIDs[sid]; ok {
		return
	}
	s.lockedIDs[sid] = struct{}{}
	r := req.(ReqUpdates)
	s.rangeLock.lock(r.FirstPeriod, r.Count, 1)
}

func (s *ForwardUpdateSync) unlockRange(sid request.ServerAndID, req request.Request) {
	if _, ok := s.lockedIDs[sid]; !ok {
		return
	}
	delete(s.lockedIDs, sid)
	r := req.(ReqUpdates)
	s.rangeLock.lock(r.FirstPeriod, r.Count, -1)
}

func (s *ForwardUpdateSync) verifyRange(req request.Request, resp request.Response) bool {
	request, ok := req.(ReqUpdates)
	if !ok {
		return false
	}
	response, ok := resp.(RespUpdates)
	if !ok {
		return false
	}
	if uint64(len(response.Updates)) != request.Count || uint64(len(response.Committees)) != request.Count {
		return false
	}
	for i, update := range response.Updates {
		if update.AttestedHeader.Header.SyncPeriod() != request.FirstPeriod+uint64(i) {
			return false
		}
	}
	return true
}

// returns true for partial success
func (s *ForwardUpdateSync) processResponse(tracker request.Tracker, event request.Event) (success bool) {
	sid, _, resp := event.RequestInfo()
	response, ok := resp.(RespUpdates)
	if !ok {
		return false
	}
	for i, update := range response.Updates {
		if err := s.chain.InsertUpdate(update, response.Committees[i]); err != nil {
			if err == light.ErrInvalidPeriod {
				// there is a gap in the update periods; stop processing without
				// failing and try again next time
				return
			}
			if err == light.ErrInvalidUpdate || err == light.ErrWrongCommitteeRoot || err == light.ErrCannotReorg {
				tracker.InvalidResponse(sid, "invalid update received")
			} else {
				log.Error("Unexpected InsertUpdate error", "error", err)
			}
			return
		}
		success = true
	}
	return
}

type updateResponseList []request.Event

func (u updateResponseList) Len() int      { return len(u) }
func (u updateResponseList) Swap(i, j int) { u[i], u[j] = u[j], u[i] }
func (u updateResponseList) Less(i, j int) bool {
	return u[i].Data.(request.RequestResponse).Request.(ReqUpdates).FirstPeriod <
		u[j].Data.(request.RequestResponse).Request.(ReqUpdates).FirstPeriod
}

// Process implements request.Module
func (s *ForwardUpdateSync) Process(tracker request.Tracker, events []request.Event) (trigger bool) {
	// iterate events and add responses to process queue
	for _, event := range events {
		switch event.Type {
		case request.EvResponse, request.EvFail, request.EvTimeout:
			sid, req, resp := event.RequestInfo()
			if event.Type == request.EvResponse && !s.verifyRange(req, resp) {
				tracker.InvalidResponse(sid, "invalid update range")
				resp = nil
			}
			if resp != nil {
				// there is a response with a valid format; put it in the process queue
				s.processQueue = append(s.processQueue, event)
				s.lockRange(sid, req)
			} else {
				s.unlockRange(sid, req)
			}
		case EvNewSignedHead:
			signedHead := event.Data.(types.SignedHeader)
			s.nextSyncPeriod[event.Server] = types.SyncPeriod(signedHead.SignatureSlot + 256)
		case request.EvUnregistered:
			delete(s.nextSyncPeriod, event.Server)
		}
	}

	// try processing ordered list of available responses
	sort.Sort(updateResponseList(s.processQueue)) //TODO
	for s.processQueue != nil {
		event := s.processQueue[0]
		if !s.processResponse(tracker, event) {
			break
		}
		trigger = true
		sid, req, _ := event.RequestInfo()
		s.unlockRange(sid, req)
		s.processQueue = s.processQueue[1:]
		if len(s.processQueue) == 0 {
			s.processQueue = nil
		}
	}

	// start new requests if necessary
	startPeriod, chainInit := s.chain.NextSyncPeriod()
	if !chainInit {
		return false
	}
	for {
		firstPeriod, maxCount := s.rangeLock.firstUnlocked(startPeriod, maxUpdateRequest)
		if reqWithID, ok := tracker.TryRequest(func(server request.Server) (request.Request, float32) {
			nextPeriod := s.nextSyncPeriod[server]
			if nextPeriod <= firstPeriod {
				return nil, 0
			}
			count := maxCount
			if nextPeriod < firstPeriod+maxCount {
				count = nextPeriod - firstPeriod
			}
			return ReqUpdates{FirstPeriod: firstPeriod, Count: count}, float32(count)
		}); ok {
			s.lockRange(reqWithID.ServerAndID, reqWithID.Request)
		} else {
			break
		}
	}
	return
}
