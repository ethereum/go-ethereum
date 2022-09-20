// Copyright 2022 The go-ethereum Authors
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
	"context"
	"errors"
	"time"

	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/log"
)

const (
	headPollFrequency = time.Millisecond * 200
	headPollCount     = 50
	maxRequest        = 8
)

// committee update syncing is initiated in each period for each syncPeriodOffsets[i]
// when slot (period+1)*params.SyncPeriodLength+syncPeriodOffsets[i] has been reached.
// This ensures that a close-to-best update for each period can be synced and
// propagated well in advance before the next period begins but later (when it's
// very unlikely that even a reorg could change the given period) the absolute
// best update will also be propagated if it's different from the previous one.
var syncPeriodOffsets = []int{-256, -16, 64}

// CommitteeSyncer syncs committee updates and signed heads from BeaconLightApi
// to CommitteeTracker
type CommitteeSyncer struct {
	api *BeaconLightApi

	genesisData         sync.GenesisData
	checkpointPeriod    uint64
	checkpointCommittee []byte
	committeeTracker    *sync.CommitteeTracker

	lastAdvertisedPeriod uint64
	lastPeriodOffset     int

	updateCache    *lru.Cache[uint64, types.LightClientUpdate]
	committeeCache *lru.Cache[uint64, []byte]
	closeCh        chan struct{}
	stopFn         func()
}

// NewCommitteeSyncer creates a new CommitteeSyncer
// Note: genesisData is only needed when light syncing (using GetInitData for bootstrap)
func NewCommitteeSyncer(api *BeaconLightApi, genesisData sync.GenesisData) *CommitteeSyncer {
	return &CommitteeSyncer{
		api:            api,
		genesisData:    genesisData,
		closeCh:        make(chan struct{}),
		updateCache:    lru.NewCache[uint64, types.LightClientUpdate](maxRequest),
		committeeCache: lru.NewCache[uint64, []byte](maxRequest),
	}
}

// Start starts the syncing of the given CommitteeTracker
func (cs *CommitteeSyncer) Start(committeeTracker *sync.CommitteeTracker) {
	cs.committeeTracker = committeeTracker
	committeeTracker.SyncWithPeer(cs, nil)
	stopFn := cs.api.StartHeadListener(
		func(slot uint64, blockRoot common.Hash) {
			cs.updateCache.Purge()
			cs.committeeCache.Purge()
			cs.syncUpdates(slot, false)
		}, func(signedHead sync.SignedHead) {
			if cs.committeeTracker.AddSignedHeads(cs, []sync.SignedHead{signedHead}) != nil {
				cs.syncUpdates(signedHead.Header.Slot, true)
				if err := cs.committeeTracker.AddSignedHeads(cs, []sync.SignedHead{signedHead}); err != nil {
					log.Error("Error adding new signed head", "error", err)
				}
			}
		}, func(err error) {
			log.Warn("Head event stream error", "err", err)
		})
	cs.stopFn = stopFn
}

// Stop stops the syncing process
func (cs *CommitteeSyncer) Stop() {
	cs.committeeTracker.Disconnect(cs)
	close(cs.closeCh)
	if cs.stopFn != nil {
		cs.stopFn()
	}
}

// syncUpdates checks whether one of the syncPeriodOffsets for the latest period
// has been reached by the current head and initiates az update sync if necessary.
// If retry is true then syncing is tried again even if no new syncing offset
// point has been reached.
func (cs *CommitteeSyncer) syncUpdates(slot uint64, retry bool) {
	nextPeriod := types.PeriodOfSlot(slot + uint64(-syncPeriodOffsets[0]))
	if nextPeriod == 0 {
		return
	}
	var (
		nextPeriodStart = types.PeriodStart(nextPeriod)
		lastPeriod      = nextPeriod - 1
		offset          = 1
	)
	for offset < len(syncPeriodOffsets) && slot >= nextPeriodStart+uint64(syncPeriodOffsets[offset]) {
		offset++
	}
	if (retry || lastPeriod != cs.lastAdvertisedPeriod || offset != cs.lastPeriodOffset) && cs.syncUpdatesUntil(lastPeriod) {
		cs.lastAdvertisedPeriod, cs.lastPeriodOffset = lastPeriod, offset
	}
}

// syncUpdatesUntil queries committee updates that the tracker does not have or
// might have improved since the last query and advertises them to the tracker.
// The tracker can then fetch the actual updates and committees via GetBestCommitteeProofs.
func (cs *CommitteeSyncer) syncUpdatesUntil(lastPeriod uint64) bool {
	ptr := int(types.MaxUpdateInfoLength)
	if lastPeriod+1 < uint64(ptr) {
		ptr = int(lastPeriod + 1)
	}
	var (
		updateInfo = &types.UpdateInfo{
			AfterLastPeriod: lastPeriod + 1,
			Scores:          make(types.UpdateScores, ptr),
		}
		localNextPeriod = cs.committeeTracker.NextPeriod()
		period          = lastPeriod
	)
	for {
		remoteUpdate, err := cs.getBestUpdate(period)
		if err != nil {
			break
		}
		ptr--
		updateInfo.Scores[ptr] = remoteUpdate.Score()
		if ptr == 0 || period == 0 {
			break
		}
		if period < localNextPeriod {
			localUpdate := cs.committeeTracker.GetBestUpdate(period)
			if localUpdate == nil || localUpdate.NextSyncCommitteeRoot == remoteUpdate.NextSyncCommitteeRoot {
				break
			}
		}
		period--
	}
	updateInfo.Scores = updateInfo.Scores[ptr:]
	log.Info("Fetched committee updates", "localNext", localNextPeriod, "count", len(updateInfo.Scores))
	if len(updateInfo.Scores) == 0 {
		log.Error("Could not fetch last committee update")
		return false
	}
	select {
	case <-cs.committeeTracker.SyncWithPeer(cs, updateInfo):
		localNextPeriod = cs.committeeTracker.NextPeriod()
		if localNextPeriod <= lastPeriod {
			log.Error("Failed to sync all API committee updates", "local next period", localNextPeriod, "remote next period", lastPeriod+1)
		}
	case <-cs.closeCh:
		return false
	}
	return true
}

// GetBestCommitteeProofs fetches updates and committees for the specified periods
func (cs *CommitteeSyncer) GetBestCommitteeProofs(ctx context.Context, req types.CommitteeRequest) (types.CommitteeReply, error) {
	reply := types.CommitteeReply{
		Updates:    make([]types.LightClientUpdate, len(req.UpdatePeriods)),
		Committees: make([][]byte, len(req.CommitteePeriods)),
	}
	var err error
	for i, period := range req.UpdatePeriods {
		if reply.Updates[i], err = cs.getBestUpdate(period); err != nil {
			return types.CommitteeReply{}, err
		}
	}
	for i, period := range req.CommitteePeriods {
		if reply.Committees[i], err = cs.getCommittee(period); err != nil {
			return types.CommitteeReply{}, err
		}
	}
	return reply, nil
}

// CanRequest returns true if a request for the given amount of items can be processed
func (cs *CommitteeSyncer) CanRequest(updateCount, committeeCount int) bool {
	return updateCount <= maxRequest && committeeCount <= maxRequest
}

// getBestUpdate returns the best update for the given period
func (cs *CommitteeSyncer) getBestUpdate(period uint64) (types.LightClientUpdate, error) {
	if c, ok := cs.updateCache.Get(period); ok {
		return c, nil
	}
	update, _, err := cs.getBestUpdateAndCommittee(period)
	return update, err
}

// getCommittee returns the committee for the given period
// Note: cannot return committee altair fork period; this is always same as the
// committee of the next period
func (cs *CommitteeSyncer) getCommittee(period uint64) ([]byte, error) {
	if period == 0 {
		return nil, errors.New("no committee available for period 0")
	}
	if cs.checkpointCommittee != nil && period == cs.checkpointPeriod {
		return cs.checkpointCommittee, nil
	}
	if c, ok := cs.committeeCache.Get(period); ok {
		return c, nil
	}
	_, committee, err := cs.getBestUpdateAndCommittee(period - 1)
	return committee, err
}

// getBestUpdateAndCommittee fetches the best update for period and corresponding
// committee for period+1 and caches the results until a new head is received by
// headPollLoop
func (cs *CommitteeSyncer) getBestUpdateAndCommittee(period uint64) (types.LightClientUpdate, []byte, error) {
	update, committee, err := cs.api.GetBestUpdateAndCommittee(period)
	if err != nil {
		return types.LightClientUpdate{}, nil, err
	}
	cs.updateCache.Add(period, update)
	cs.committeeCache.Add(period+1, committee)
	return update, committee, nil
}

// GetInitData fetches the bootstrap data and returns LightClientInitData (the
// corresponding committee is stored so that a subsequent GetBestCommitteeProofs
// can return it when requested)
func (cs *CommitteeSyncer) GetInitData(ctx context.Context, checkpoint common.Hash) (types.Header, sync.LightClientInitData, error) {
	if cs.genesisData == (sync.GenesisData{}) {
		return types.Header{}, sync.LightClientInitData{}, errors.New("missing genesis data")
	}
	header, checkpointData, committee, err := cs.api.GetCheckpointData(ctx, checkpoint)
	if err != nil {
		return types.Header{}, sync.LightClientInitData{}, err
	}
	cs.checkpointPeriod, cs.checkpointCommittee = checkpointData.Period, committee
	return header, sync.LightClientInitData{GenesisData: cs.genesisData, CheckpointData: checkpointData}, nil
}

// ProtocolError is called by the tracker when the BeaconLightApi has provided
// wrong committee updates or signed heads
func (cs *CommitteeSyncer) ProtocolError(description string) {
	log.Error("Beacon node API data source delivered wrong reply", "error", description)
}
