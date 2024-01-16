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

const maxUpdateRequest = 8 // maximum number of updates requested in a single request

type committeeChain interface {
	CheckpointInit(bootstrap types.BootstrapData) error
	InsertUpdate(update *types.LightClientUpdate, nextCommittee *types.SerializedSyncCommittee) error
	NextSyncPeriod() (uint64, bool)
}

// CheckpointInit implements request.Module; it fetches the light client bootstrap
// data belonging to the given checkpoint hash and initializes the committee chain
// if successful.
type CheckpointInit struct {
	chain          committeeChain
	checkpointHash common.Hash
	locked         request.ServerAndID
	initialized    bool
}

// NewCheckpointInit creates a new CheckpointInit.
func NewCheckpointInit(chain committeeChain, checkpointHash common.Hash) *CheckpointInit {
	return &CheckpointInit{
		chain:          chain,
		checkpointHash: checkpointHash,
	}
}

func (s *CheckpointInit) Process(events []request.Event) {
	for _, event := range events {
		if !event.IsRequestEvent() {
			return
		}
		sid, req, resp := event.RequestInfo()
		if event.Type == request.EvRequest {
			s.locked = sid
			return
		}
		if s.locked == sid {
			s.locked = request.ServerAndID{}
		}
		if resp != nil {
			if checkpoint, ok := resp.(*types.BootstrapData); ok && checkpoint.Header.Hash() == common.Hash(req.(ReqCheckpointData)) {
				s.chain.CheckpointInit(*checkpoint)
				s.initialized = true
				return
			}
			event.Server.Fail("invalid checkpoint data")
		}
	}
}

func (s *CheckpointInit) MakeRequest(server request.Server) (request.Request, float32) {
	if s.initialized || s.locked != (request.ServerAndID{}) {
		return nil, 0
	}
	return ReqCheckpointData(s.checkpointHash), 0
}

// ForwardUpdateSync implements request.Module; it fetches updates between the
// committee chain head and each server's announced head. Updates are fetched
// in batches and multiple batches can also be requested in parallel.
// Out of order responses are also handled; if a batch of updates cannot be added
// to the chain immediately because of a gap then the future updates are
// remembered until they can be processed.
type ForwardUpdateSync struct {
	chain          committeeChain
	rangeLock      rangeLock
	lockedIDs      map[request.ServerAndID]struct{}
	processQueue   []request.Event
	nextSyncPeriod map[request.Server]uint64
}

// NewForwardUpdateSync creates a new ForwardUpdateSync.
func NewForwardUpdateSync(chain committeeChain) *ForwardUpdateSync {
	return &ForwardUpdateSync{
		chain:          chain,
		rangeLock:      make(rangeLock),
		lockedIDs:      make(map[request.ServerAndID]struct{}),
		nextSyncPeriod: make(map[request.Server]uint64),
	}
}

// rangeLock allows locking sections of an integer space, preventing the syncing
// mechanism from making requests again for sections where a not timed out request
// is already pending or where already fetched and unprocessed data is available.
type rangeLock map[uint64]int

// lock locks or unlocks the given section, depending on the sign of the add parameter.
func (r rangeLock) lock(first, count uint64, add int) {
	for i := first; i < first+count; i++ {
		if v := r[i] + add; v > 0 {
			r[i] = v
		} else {
			delete(r, i)
		}
	}
}

// firstUnlocked returns the first unlocked section starting at or after start
// and not longer than maxCount.
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

// lockRange locks the range belonging to the given update request, unless the
// same request has already been locked
func (s *ForwardUpdateSync) lockRange(sid request.ServerAndID, req request.Request) {
	if _, ok := s.lockedIDs[sid]; ok {
		return
	}
	s.lockedIDs[sid] = struct{}{}
	r := req.(ReqUpdates)
	s.rangeLock.lock(r.FirstPeriod, r.Count, 1)
}

// unlockRange unlocks the range belonging to the given update request, unless
// same request has already been unlocked
func (s *ForwardUpdateSync) unlockRange(sid request.ServerAndID, req request.Request) {
	if _, ok := s.lockedIDs[sid]; !ok {
		return
	}
	delete(s.lockedIDs, sid)
	r := req.(ReqUpdates)
	s.rangeLock.lock(r.FirstPeriod, r.Count, -1)
}

// verifyRange returns true if the number of updates and the individual update
// periods in the response match the requested section.
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

// processResponse adds the fetched updates and committees to the committee chain.
// Returns true in case of full or partial success.
func (s *ForwardUpdateSync) processResponse(event request.Event) (success bool) {
	_, _, resp := event.RequestInfo()
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

// updateResponseList implements sort.Sort and sorts update request/response events by FirstPeriod.
type updateResponseList []request.Event

func (u updateResponseList) Len() int      { return len(u) }
func (u updateResponseList) Swap(i, j int) { u[i], u[j] = u[j], u[i] }
func (u updateResponseList) Less(i, j int) bool {
	return u[i].Data.(request.RequestResponse).Request.(ReqUpdates).FirstPeriod <
		u[j].Data.(request.RequestResponse).Request.(ReqUpdates).FirstPeriod
}

func (s *ForwardUpdateSync) Process(events []request.Event) {
	for _, event := range events {
		switch event.Type {
		case request.EvRequest:
			sid, req, _ := event.RequestInfo()
			s.lockRange(sid, req)
		case request.EvResponse, request.EvFail, request.EvTimeout:
			sid, req, resp := event.RequestInfo()
			if event.Type == request.EvResponse && !s.verifyRange(req, resp) {
				event.Server.Fail("invalid update range")
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
		if !s.processResponse(event) {
			return
		}
		sid, req, _ := event.RequestInfo()
		s.unlockRange(sid, req)
		s.processQueue = s.processQueue[1:]
		if len(s.processQueue) == 0 {
			s.processQueue = nil
		}
	}
}

func (s *ForwardUpdateSync) MakeRequest(server request.Server) (request.Request, float32) {
	startPeriod, chainInit := s.chain.NextSyncPeriod()
	if !chainInit {
		return nil, 0
	}
	firstPeriod, maxCount := s.rangeLock.firstUnlocked(startPeriod, maxUpdateRequest)
	nextPeriod := s.nextSyncPeriod[server]
	if nextPeriod <= firstPeriod {
		return nil, 0
	}
	count := maxCount
	if nextPeriod < firstPeriod+maxCount {
		count = nextPeriod - firstPeriod
	}
	return ReqUpdates{FirstPeriod: firstPeriod, Count: count}, float32(count)
}
