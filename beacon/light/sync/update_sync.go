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
	"github.com/ethereum/go-ethereum/beacon/params"
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
	// per-server state is used to track the state of requesting checkpoint header
	// info. Part of this info (canonical and finalized state) is not validated
	// and therefore it is requested from each server separately after it has
	// reported a missing checkpoint (which is also not validated info).
	serverState map[request.Server]serverState
	// the following fields are used to determine whether the checkpoint is on
	// epoch boundary. This information is validated and therefore stored globally.
	parentHash                  common.Hash
	hasEpochInfo, epochBoundary bool
	cpSlot, parentSlot          uint64
}

const (
	ssDefault         = iota // no action yet or checkpoint requested
	ssNeedHeader             // checkpoint req failed, need cp header
	ssHeaderRequested        // cp header requested
	ssNeedParent             // cp header slot %32 != 0, need parent to check epoch boundary
	ssParentRequested        // cp parent header requested
	ssPrintStatus            // has all necessary info, print log message if init still not successful
	ssDone                   // log message printed, no more action required
)

type serverState struct {
	state                           int
	hasHeader, canonical, finalized bool // stored per server because not validated
}

// NewCheckpointInit creates a new CheckpointInit.
func NewCheckpointInit(chain committeeChain, checkpointHash common.Hash) *CheckpointInit {
	return &CheckpointInit{
		chain:          chain,
		checkpointHash: checkpointHash,
		serverState:    make(map[request.Server]serverState),
	}
}

// Process implements request.Module.
func (s *CheckpointInit) Process(requester request.Requester, events []request.Event) {
	if s.initialized {
		return
	}

	for _, event := range events {
		switch event.Type {
		case request.EvResponse, request.EvFail, request.EvTimeout:
			sid, req, resp := event.RequestInfo()
			if s.locked == sid {
				s.locked = request.ServerAndID{}
			}
			if event.Type == request.EvTimeout {
				continue
			}
			switch s.serverState[sid.Server].state {
			case ssDefault:
				if resp != nil {
					if checkpoint := resp.(*types.BootstrapData); checkpoint.Header.Hash() == common.Hash(req.(ReqCheckpointData)) {
						s.chain.CheckpointInit(*checkpoint)
						s.initialized = true
						return
					}
					requester.Fail(event.Server, "invalid checkpoint data")
				}
				s.serverState[sid.Server] = serverState{state: ssNeedHeader}
			case ssHeaderRequested:
				if resp == nil {
					s.serverState[sid.Server] = serverState{state: ssPrintStatus}
					continue
				}
				newState := serverState{
					hasHeader: true,
					canonical: resp.(RespHeader).Canonical,
					finalized: resp.(RespHeader).Finalized,
				}
				s.cpSlot, s.parentHash = resp.(RespHeader).Header.Slot, resp.(RespHeader).Header.ParentRoot
				if s.cpSlot%params.EpochLength == 0 {
					s.hasEpochInfo, s.epochBoundary = true, true
				}
				if s.hasEpochInfo {
					newState.state = ssPrintStatus
				} else {
					newState.state = ssNeedParent
				}
				s.serverState[sid.Server] = newState
			case ssParentRequested:
				s.parentSlot = resp.(RespHeader).Header.Slot
				s.hasEpochInfo, s.epochBoundary = true, s.cpSlot/params.EpochLength > s.parentSlot/params.EpochLength
				newState := s.serverState[sid.Server]
				newState.state = ssPrintStatus
				s.serverState[sid.Server] = newState
			}

		case request.EvUnregistered:
			delete(s.serverState, event.Server)
		}
	}

	// start a request if possible
	for _, server := range requester.CanSendTo() {
		switch s.serverState[server].state {
		case ssDefault:
			if s.locked == (request.ServerAndID{}) {
				id := requester.Send(server, ReqCheckpointData(s.checkpointHash))
				s.locked = request.ServerAndID{Server: server, ID: id}
			}
		case ssNeedHeader:
			requester.Send(server, ReqHeader(s.checkpointHash))
			newState := s.serverState[server]
			newState.state = ssHeaderRequested
			s.serverState[server] = newState
		case ssNeedParent:
			requester.Send(server, ReqHeader(s.parentHash))
			newState := s.serverState[server]
			newState.state = ssParentRequested
			s.serverState[server] = newState
		}
	}

	// print log message if necessary
	for server, state := range s.serverState {
		if state.state != ssPrintStatus {
			continue
		}
		switch {
		case !state.hasHeader:
			log.Error("blsync: checkpoint block is not available, reported as unknown", "server", server.Name())
		case !state.canonical:
			log.Error("blsync: checkpoint block is not available, reported as non-canonical", "server", server.Name())
		case !s.hasEpochInfo:
			// should be available if hasHeader is true and state is ssPrintStatus
			panic("checkpoint epoch info not available when printing retrieval status")
		case !s.epochBoundary:
			log.Error("blsync: checkpoint block is not first of epoch", "slot", s.cpSlot, "parent", s.parentSlot, "server", server.Name())
		case !state.finalized:
			log.Error("blsync: checkpoint block is reported as non-finalized", "server", server.Name())
		default:
			log.Error("blsync: checkpoint not available, but reported as finalized; specified checkpoint hash might be too old", "server", server.Name())
		}
		s.serverState[server] = serverState{state: ssDone}
	}
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
	processQueue   []updateResponse
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
func (s *ForwardUpdateSync) lockRange(sid request.ServerAndID, req ReqUpdates) {
	if _, ok := s.lockedIDs[sid]; ok {
		return
	}
	s.lockedIDs[sid] = struct{}{}
	s.rangeLock.lock(req.FirstPeriod, req.Count, 1)
}

// unlockRange unlocks the range belonging to the given update request, unless
// same request has already been unlocked
func (s *ForwardUpdateSync) unlockRange(sid request.ServerAndID, req ReqUpdates) {
	if _, ok := s.lockedIDs[sid]; !ok {
		return
	}
	delete(s.lockedIDs, sid)
	s.rangeLock.lock(req.FirstPeriod, req.Count, -1)
}

// verifyRange returns true if the number of updates and the individual update
// periods in the response match the requested section.
func (s *ForwardUpdateSync) verifyRange(request ReqUpdates, response RespUpdates) bool {
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

// updateResponse is a response that has passed initial verification and has been
// queued for processing. Note that an update response cannot be processed until
// the previous updates have also been added to the chain.
type updateResponse struct {
	sid      request.ServerAndID
	request  ReqUpdates
	response RespUpdates
}

// updateResponseList implements sort.Sort and sorts update request/response events by FirstPeriod.
type updateResponseList []updateResponse

func (u updateResponseList) Len() int      { return len(u) }
func (u updateResponseList) Swap(i, j int) { u[i], u[j] = u[j], u[i] }
func (u updateResponseList) Less(i, j int) bool {
	return u[i].request.FirstPeriod < u[j].request.FirstPeriod
}

// Process implements request.Module.
func (s *ForwardUpdateSync) Process(requester request.Requester, events []request.Event) {
	for _, event := range events {
		switch event.Type {
		case request.EvResponse, request.EvFail, request.EvTimeout:
			sid, rq, rs := event.RequestInfo()
			req := rq.(ReqUpdates)
			var queued bool
			if event.Type == request.EvResponse {
				resp := rs.(RespUpdates)
				if s.verifyRange(req, resp) {
					// there is a response with a valid format; put it in the process queue
					s.processQueue = append(s.processQueue, updateResponse{sid: sid, request: req, response: resp})
					s.lockRange(sid, req)
					queued = true
				} else {
					requester.Fail(event.Server, "invalid update range")
				}
			}
			if !queued {
				s.unlockRange(sid, req)
			}
		case EvNewOptimisticUpdate:
			update := event.Data.(types.OptimisticUpdate)
			s.nextSyncPeriod[event.Server] = types.SyncPeriod(update.SignatureSlot + 256)
		case request.EvUnregistered:
			delete(s.nextSyncPeriod, event.Server)
		}
	}

	// try processing ordered list of available responses
	sort.Sort(updateResponseList(s.processQueue))
	for s.processQueue != nil {
		u := s.processQueue[0]
		if !s.processResponse(requester, u) {
			break
		}
		s.unlockRange(u.sid, u.request)
		s.processQueue = s.processQueue[1:]
		if len(s.processQueue) == 0 {
			s.processQueue = nil
		}
	}

	// start new requests if possible
	startPeriod, chainInit := s.chain.NextSyncPeriod()
	if !chainInit {
		return
	}
	for {
		firstPeriod, maxCount := s.rangeLock.firstUnlocked(startPeriod, maxUpdateRequest)
		var (
			sendTo    request.Server
			bestCount uint64
		)
		for _, server := range requester.CanSendTo() {
			nextPeriod := s.nextSyncPeriod[server]
			if nextPeriod <= firstPeriod {
				continue
			}
			count := maxCount
			if nextPeriod < firstPeriod+maxCount {
				count = nextPeriod - firstPeriod
			}
			if count > bestCount {
				sendTo, bestCount = server, count
			}
		}
		if sendTo == nil {
			return
		}
		req := ReqUpdates{FirstPeriod: firstPeriod, Count: bestCount}
		id := requester.Send(sendTo, req)
		s.lockRange(request.ServerAndID{Server: sendTo, ID: id}, req)
	}
}

// processResponse adds the fetched updates and committees to the committee chain.
// Returns true in case of full or partial success.
func (s *ForwardUpdateSync) processResponse(requester request.Requester, u updateResponse) (success bool) {
	for i, update := range u.response.Updates {
		if err := s.chain.InsertUpdate(update, u.response.Committees[i]); err != nil {
			if err == light.ErrInvalidPeriod {
				// there is a gap in the update periods; stop processing without
				// failing and try again next time
				return
			}
			if err == light.ErrInvalidUpdate || err == light.ErrWrongCommitteeRoot || err == light.ErrCannotReorg {
				requester.Fail(u.sid.Server, "invalid update received")
			} else {
				log.Error("Unexpected InsertUpdate error", "error", err)
			}
			return
		}
		success = true
	}
	return
}
