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

type CheckpointInit struct {
	chain          *light.CommitteeChain
	checkpointHash common.Hash
	pending        bool
	initialized    bool
}

func NewCheckpointInit(chain *light.CommitteeChain, checkpointHash common.Hash) *CheckpointInit {
	return &CheckpointInit{
		chain:          chain,
		checkpointHash: checkpointHash,
	}
}

// Process implements request.Module
func (s *CheckpointInit) Process(tracker *request.RequestTracker, requestEvents []request.RequestEvent, serverEvents []request.ServerEvent) bool {
	if s.initialized {
		return false
	}
	for _, event := range requestEvents {
		if event.Timeout != event.Finalized {
			s.pending = false
		}
		if event.Response != nil {
			if checkpoint, ok := event.Response.(*types.BootstrapData); ok && checkpoint.Validate(common.Hash(event.Request.(ReqCheckpointData))) == nil {
				s.chain.CheckpointInit(*checkpoint) //TODO
				s.initialized = true
				return true
			}
			event.Server.Fail("invalid checkpoint data")
		}
	}
	if !s.pending {
		if _, request := tracker.TryRequest(func(server request.Server) (request.Request, float32) {
			return ReqCheckpointData(s.checkpointHash), 0
		}); request != nil {
			s.pending = true
		}
	}
	return false
}

type ForwardUpdateSync struct {
	chain          *light.CommitteeChain
	rangeLock      rangeLock
	processQueue   []request.RequestEvent
	nextSyncPeriod map[request.Server]uint64
}

func NewForwardUpdateSync(chain *light.CommitteeChain) *ForwardUpdateSync {
	return &ForwardUpdateSync{
		chain:          chain,
		rangeLock:      make(rangeLock),
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

func (s *ForwardUpdateSync) verifyRange(event request.RequestEvent) bool {
	request, ok := event.Request.(ReqUpdates)
	if !ok {
		return false
	}
	response, ok := event.Response.(RespUpdates)
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
func (s *ForwardUpdateSync) processResponse(event request.RequestEvent) (success bool) {
	response, ok := event.Response.(RespUpdates)
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
				event.Server.Fail("invalid update received")
			} else {
				log.Error("Unexpected InsertUpdate error", "error", err)
			}
			return
		}
		success = true
	}
	return
}

type updateResponseList []request.RequestEvent

func (u updateResponseList) Len() int      { return len(u) }
func (u updateResponseList) Swap(i, j int) { u[i], u[j] = u[j], u[i] }
func (u updateResponseList) Less(i, j int) bool {
	return u[i].Request.(ReqUpdates).FirstPeriod < u[j].Request.(ReqUpdates).FirstPeriod
}

// Process implements request.Module
func (s *ForwardUpdateSync) Process(tracker *request.RequestTracker, requestEvents []request.RequestEvent, serverEvents []request.ServerEvent) (trigger bool) {
	// iterate events and add responses to process queue
	for _, event := range requestEvents {
		if event.Response != nil && !s.verifyRange(event) {
			event.Server.Fail("invalid update range")
			event.Response = nil
		}
		req := event.Request.(ReqUpdates)
		if event.Response != nil {
			// there is a response with a valid format; put it in the process queue
			s.processQueue = append(s.processQueue, event)
			if event.Timeout {
				// it was already timed out and unlocked; lock again until processed
				s.rangeLock.lock(req.FirstPeriod, req.Count, 1)
			}
		} else if event.Timeout != event.Finalized {
			// unlock if timed out or returned with an invalid response without
			// previously being unlocked by a timeout
			s.rangeLock.lock(req.FirstPeriod, req.Count, -1)
		}
	}

	// try processing ordered list of available responses
	sort.Sort(updateResponseList(s.processQueue)) //TODO
	for s.processQueue != nil {
		event := s.processQueue[0]
		if !s.processResponse(event) {
			break
		}
		trigger = true
		req := event.Request.(ReqUpdates)
		s.rangeLock.lock(req.FirstPeriod, req.Count, -1)
		s.processQueue = s.processQueue[1:]
		if len(s.processQueue) == 0 {
			s.processQueue = nil
		}
	}

	// update nextSyncPeriod of servers based on server events
	for _, event := range serverEvents {
		switch event.Type {
		case EvNewSignedHead:
			signedHead := event.Data.(types.SignedHeader)
			s.nextSyncPeriod[event.Server] = types.SyncPeriod(signedHead.Header.Slot + 256)
		case request.EvUnregistered:
			delete(s.nextSyncPeriod, event.Server)
		}
	}

	// start new requests if necessary
	startPeriod, chainInit := s.chain.NextSyncPeriod()
	if !chainInit {
		return false
	}
	for {
		firstPeriod, maxCount := s.rangeLock.firstUnlocked(startPeriod, maxUpdateRequest)
		if _, request := tracker.TryRequest(func(server request.Server) (request.Request, float32) {
			nextPeriod := s.nextSyncPeriod[server]
			if nextPeriod <= firstPeriod {
				return nil, 0
			}
			count := maxCount
			if nextPeriod < firstPeriod+maxCount {
				count = nextPeriod - firstPeriod
			}
			return ReqUpdates{FirstPeriod: firstPeriod, Count: count}, float32(count)
		}); request != nil {
			req := request.(ReqUpdates)
			s.rangeLock.lock(req.FirstPeriod, req.Count, 1)
		} else {
			break
		}
	}
	return
}
