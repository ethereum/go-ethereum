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

package sync

import (
	"encoding/binary"
	"sync"

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	"github.com/minio/sha256-simd"
)

var (
	initDataKey      = []byte("ct.init") // RLP(LightClientInitData)
	bestUpdateKey    = []byte("ct.bu-")  // bigEndian64(syncPeriod) -> RLP(types.LightClientUpdate)  (nextCommittee only referenced by root hash)
	syncCommitteeKey = []byte("ct.sc-")  // bigEndian64(syncPeriod) + committee root hash -> serialized committee
)

const SerializedCommitteeSize = (params.SyncCommitteeSize + 1) * params.BlsPubkeySize

// ChainConfig contains built-in chain configuration presets for certain networks
type ChainConfig struct {
	GenesisData
	Forks      Forks
	Checkpoint common.Hash
}

// GenesisData is required for signature verification and is set by the
// CommitteeTracker.Init function.
type GenesisData struct {
	GenesisTime           uint64      // unix time (in seconds) of slot 0
	GenesisValidatorsRoot common.Hash // root hash of the genesis validator set, used for signature domain calculation
}

// Constraints defines constraints on the synced update chain. These constraints
// include the GenesisData, a range of periods (first <= period < afterFixed)
// where committee roots are fixed and another "free" range (afterFixed <= period < afterLast)
// where committee roots are determined by the best known update chain.
// An implementation of Constraints should call initCallback to pass
// GenesisData whenever it is available (either durinrg SetCallbacks or later).
// If the constraints are changed then it should call updateCallback.
//
// Note: this interface can be used either for light syncing mode (in which case
// only the checkpoint is fixed and any valid update chain can be synced starting
// from there) or full syncing light service mode (in which case a full beacon
// header chain is synced based on the externally driven consensus and the update
// chain is fully restricted based on that).
type Constraints interface {
	SyncRange() (syncRange types.UpdateRange, lastFixed uint64)
	CommitteeRoot(period uint64) (root common.Hash, matchAll bool) // matchAll is true in the free range where any committee root matches the constraints
	SetCallbacks(initCallback func(GenesisData), updateCallback func())
}

// CommitteeTracker maintains a chain of sync committee updates and a small
// set of best known signed heads. It is used in all client configurations
// operating on a beacon chain. It can sync its update chain and receive signed
// heads from either an ODR or beacon node API backend and propagate/serve this
// data to subscribed peers. Received signed heads are validated based on the
// known sync committee chain and added to the local set if valid or placed in a
// deferred queue if the committees are not synced up to the period of the new
// head yet.
// Sync committee chain is either initialized from a weak subjectivity checkpoint
// or controlled by a BeaconChain that is driven by a trusted source (beacon node API).
type CommitteeTracker struct {
	lock                     sync.RWMutex
	db                       ethdb.KeyValueStore
	sigVerifier              committeeSigVerifier
	clock                    mclock.Clock
	bestUpdateCache          *lru.Cache[uint64, *types.LightClientUpdate]
	serializedCommitteeCache *lru.Cache[string, []byte]
	syncCommitteeCache       *lru.Cache[string, syncCommittee]
	committeeRootCache       *lru.Cache[uint64, common.Hash]
	unixNano                 func() int64

	forks              Forks
	constraints        Constraints
	signerThreshold    int
	minimumUpdateScore types.UpdateScore
	enforceTime        bool

	genesisInit bool   // genesis data initialized (signature check possible)
	genesisTime uint64 // unix time (seconds)
	chainInit   bool   // update and committee chain initialized
	// if chain is initialized then best updates for periods between firstPeriod to nextPeriod-1
	// and committees for periods between firstPeriod to nextPeriod are available
	firstPeriod, nextPeriod uint64

	updateInfo                *types.UpdateInfo
	connected                 map[ctServer]*ctPeerInfo
	requestQueue              []*ctPeerInfo
	broadcastTo, advertisedTo map[ctClient]struct{}
	advertiseScheduled        bool
	triggerCh, initCh, stopCh chan struct{}
	acceptedList              headList

	headSubs []func(types.Header)
}

// NewCommitteeTracker creates a new CommitteeTracker
func NewCommitteeTracker(db ethdb.KeyValueStore, forks Forks, constraints Constraints, signerThreshold int, enforceTime bool, sigVerifier committeeSigVerifier, clock mclock.Clock, unixNano func() int64) *CommitteeTracker {
	s := &CommitteeTracker{
		bestUpdateCache:          lru.NewCache[uint64, *types.LightClientUpdate](1000),
		serializedCommitteeCache: lru.NewCache[string, []byte](100),
		syncCommitteeCache:       lru.NewCache[string, syncCommittee](100),
		committeeRootCache:       lru.NewCache[uint64, common.Hash](100),
		db:                       db,
		sigVerifier:              sigVerifier,
		clock:                    clock,
		unixNano:                 unixNano,
		forks:                    forks,
		constraints:              constraints,
		signerThreshold:          signerThreshold,
		enforceTime:              enforceTime,
		minimumUpdateScore: types.UpdateScore{
			SignerCount:    uint32(signerThreshold),
			SubPeriodIndex: params.SyncPeriodLength / 16,
		},
		connected:    make(map[ctServer]*ctPeerInfo),
		broadcastTo:  make(map[ctClient]struct{}),
		triggerCh:    make(chan struct{}, 1),
		initCh:       make(chan struct{}),
		stopCh:       make(chan struct{}),
		acceptedList: newHeadList(4),
	}
	var (
		iter = s.db.NewIterator(bestUpdateKey, nil)
		kl   = len(bestUpdateKey)
	)
	// iterate through them all for simplicity; at most a few hundred items
	for iter.Next() {
		period := binary.BigEndian.Uint64(iter.Key()[kl : kl+8])
		if !s.chainInit {
			s.chainInit = true
			s.firstPeriod = period
		} else if s.nextPeriod != period {
			break // continuity guaranteed
		}
		s.nextPeriod = period + 1
	}
	iter.Release()
	constraints.SetCallbacks(s.Init, s.EnforceForksAndConstraints)
	return s
}

// Init initializes the tracker with the given GenesisData and starts the update
// syncing process.
// Note that Init may be called either at startup or later if it has to be
// fetched from the network based on a checkpoint hash.
func (s *CommitteeTracker) Init(genesisData GenesisData) {
	if s.genesisInit {
		log.Error("CommitteeTracker already initialized")
		return
	}
	s.lock.Lock()
	s.forks.computeDomains(genesisData.GenesisValidatorsRoot)
	s.genesisTime = genesisData.GenesisTime
	s.genesisInit = true
	s.enforceForksAndConstraints()
	s.lock.Unlock()
	close(s.initCh)

	go s.syncLoop()
}

// GetInitChannel returns a channel that gets closed when the tracker has been initialized
func (s *CommitteeTracker) GetInitChannel() chan struct{} {
	return s.initCh
}

const (
	sciSuccess = iota
	sciNeedCommittee
	sciWrongUpdate
	sciUnexpectedError
)

// insertUpdate verifies the update and stores it in the update chain if possible.
// The serialized version of the next committee should also be supplied if it is
// not already stored in the database.
func (s *CommitteeTracker) insertUpdate(update *types.LightClientUpdate, nextCommittee []byte) int {
	var (
		period   = update.Header.SyncPeriod()
		rollback bool
	)
	if !s.verifyUpdate(update) {
		return sciWrongUpdate
	}

	if !s.chainInit || period > s.nextPeriod || period+1 < s.firstPeriod {
		log.Error("Unexpected insertUpdate", "period", period, "firstPeriod", s.firstPeriod, "nextPeriod", s.nextPeriod)
		return sciUnexpectedError
	}
	if period+1 == s.firstPeriod {
		if update.NextSyncCommitteeRoot != s.getSyncCommitteeRoot(period+1) {
			return sciWrongUpdate
		}
	} else if period < s.nextPeriod {
		// update should already exist
		oldUpdate := s.GetBestUpdate(period)
		if oldUpdate == nil {
			log.Error("Update expected to exist but missing from db")
			return sciUnexpectedError
		}
		if !update.Score().BetterThan(oldUpdate.Score()) {
			// not better that existing one, nothing to do
			return sciSuccess
		}
		rollback = update.NextSyncCommitteeRoot != oldUpdate.NextSyncCommitteeRoot
	}

	if (period == s.nextPeriod || rollback) && s.GetSerializedSyncCommittee(period+1, update.NextSyncCommitteeRoot) == nil {
		// committee is not yet stored in db
		if nextCommittee == nil {
			return sciNeedCommittee
		}
		if SerializedCommitteeRoot(nextCommittee) != update.NextSyncCommitteeRoot {
			return sciWrongUpdate
		}
		s.storeSerializedSyncCommittee(period+1, update.NextSyncCommitteeRoot, nextCommittee)
	}

	if rollback {
		for p := s.nextPeriod - 1; p >= period; p-- {
			s.deleteBestUpdate(p)
		}
		s.nextPeriod = period
	}
	s.storeBestUpdate(update)
	if period == s.nextPeriod {
		s.nextPeriod++
	}
	if period+1 == s.firstPeriod {
		s.firstPeriod--
	}
	log.Info("Synced new committee update", "period", period, "nextCommitteeRoot", update.NextSyncCommitteeRoot)
	return sciSuccess
}

// verifyUpdate checks whether the header signature is correct and the update
// fits into the specified constraints (assumes that the update has been
// successfully validated previously)
func (s *CommitteeTracker) verifyUpdate(update *types.LightClientUpdate) bool {
	if !s.checkConstraints(update) {
		return false
	}
	ok, age := s.verifySignature(SignedHead{Header: update.Header, Signature: update.SyncCommitteeSignature, BitMask: update.SyncCommitteeBits, SignatureSlot: update.Header.Slot})
	if age < 0 {
		log.Warn("Future committee update received", "age", age)
	}
	return ok
}

// getBestUpdateKey returns the database key for the canonical sync committee
// update at the given period
func getBestUpdateKey(period uint64) []byte {
	var (
		kl  = len(bestUpdateKey)
		key = make([]byte, kl+8)
	)
	copy(key[:kl], bestUpdateKey)
	binary.BigEndian.PutUint64(key[kl:], period)
	return key
}

// GetBestUpdate returns the best known canonical sync committee update at the given period
func (s *CommitteeTracker) GetBestUpdate(period uint64) *types.LightClientUpdate {
	if update, ok := s.bestUpdateCache.Get(period); ok {
		if update != nil {
			if update.Header.SyncPeriod() != period {
				log.Error("Best update from wrong period found in cache")
			}
		}
		return update
	}
	if updateEnc, err := s.db.Get(getBestUpdateKey(period)); err == nil {
		update := new(types.LightClientUpdate)
		if err := rlp.DecodeBytes(updateEnc, update); err == nil {
			update.Score() // ensure that canonical updates in memory always have their score calculated and therefore are thread safe
			s.bestUpdateCache.Add(period, update)
			if update.Header.SyncPeriod() != period {
				log.Error("Best update from wrong period found in database")
			}
			return update
		} else {
			log.Error("Error decoding best update", "error", err)
		}
	}
	s.bestUpdateCache.Add(period, nil)
	return nil
}

// storeBestUpdate stores a sync committee update in the canonical update chain
func (s *CommitteeTracker) storeBestUpdate(update *types.LightClientUpdate) {
	period := update.Header.SyncPeriod()
	updateEnc, err := rlp.EncodeToBytes(update)
	if err != nil {
		log.Error("Error encoding types.LightClientUpdate", "error", err)
		return
	}
	s.bestUpdateCache.Add(period, update)
	s.db.Put(getBestUpdateKey(period), updateEnc)
	s.committeeRootCache.Remove(period + 1)
	s.updateInfoChanged()
}

// deleteBestUpdate deletes a sync committee update from the canonical update chain
func (s *CommitteeTracker) deleteBestUpdate(period uint64) {
	s.db.Delete(getBestUpdateKey(period))
	s.bestUpdateCache.Remove(period)
	s.committeeRootCache.Remove(period + 1)
	s.updateInfoChanged()
}

// getSyncCommitteeKey returns the database key for the specified sync committee
func getSyncCommitteeKey(period uint64, committeeRoot common.Hash) []byte {
	var (
		kl  = len(syncCommitteeKey)
		key = make([]byte, kl+8+32)
	)
	copy(key[:kl], syncCommitteeKey)
	binary.BigEndian.PutUint64(key[kl:kl+8], period)
	copy(key[kl+8:], committeeRoot[:])
	return key
}

// GetSerializedSyncCommittee fetches the serialized version of a sync committee
// from cache or database
func (s *CommitteeTracker) GetSerializedSyncCommittee(period uint64, committeeRoot common.Hash) []byte {
	key := getSyncCommitteeKey(period, committeeRoot)
	if committee, ok := s.serializedCommitteeCache.Get(string(key)); ok {
		if len(committee) == SerializedCommitteeSize {
			return committee
		} else {
			log.Error("Serialized committee with invalid size found in cache")
		}
	}
	if committee, err := s.db.Get(key); err == nil {
		if len(committee) == SerializedCommitteeSize {
			s.serializedCommitteeCache.Add(string(key), committee)
			return committee
		} else {
			log.Error("Serialized committee with invalid size found in database")
		}
	}
	return nil
}

// storeSerializedSyncCommittee stores the serialized version of a sync committee
// to cache and database
func (s *CommitteeTracker) storeSerializedSyncCommittee(period uint64, committeeRoot common.Hash, committee []byte) {
	key := getSyncCommitteeKey(period, committeeRoot)
	s.serializedCommitteeCache.Add(string(key), committee)
	s.syncCommitteeCache.Remove(string(key)) // a nil entry for "not found" might have been stored here earlier
	s.db.Put(key, committee)
}

// SerializedCommitteeRoot calculates the root hash of the binary tree representation
// of a sync committee provided in serialized format
func SerializedCommitteeRoot(enc []byte) common.Hash {
	if len(enc) != SerializedCommitteeSize {
		return common.Hash{}
	}
	var (
		hasher  = sha256.New()
		padding [64 - params.BlsPubkeySize]byte
		data    [params.SyncCommitteeSize]common.Hash
		l       = params.SyncCommitteeSize
	)
	for i := range data {
		hasher.Reset()
		hasher.Write(enc[i*params.BlsPubkeySize : (i+1)*params.BlsPubkeySize])
		hasher.Write(padding[:])
		hasher.Sum(data[i][:0])
	}
	for l > 1 {
		for i := 0; i < l/2; i++ {
			hasher.Reset()
			hasher.Write(data[i*2][:])
			hasher.Write(data[i*2+1][:])
			hasher.Sum(data[i][:0])
		}
		l /= 2
	}
	hasher.Reset()
	hasher.Write(enc[SerializedCommitteeSize-params.BlsPubkeySize : SerializedCommitteeSize])
	hasher.Write(padding[:])
	hasher.Sum(data[1][:0])
	hasher.Reset()
	hasher.Write(data[0][:])
	hasher.Write(data[1][:])
	hasher.Sum(data[0][:0])
	return data[0]
}

// getSyncCommitteeRoot returns the sync committee root at the given period of
// the current local committee root constraints or update chain (tracker mutex
// lock expected).
func (s *CommitteeTracker) getSyncCommitteeRoot(period uint64) (root common.Hash) {
	if r, ok := s.committeeRootCache.Get(period); ok {
		return r
	}
	defer func() {
		s.committeeRootCache.Add(period, root)
	}()

	if r, matchAll := s.constraints.CommitteeRoot(period); !matchAll {
		return r
	}
	if !s.chainInit || period <= s.firstPeriod || period > s.nextPeriod {
		return common.Hash{}
	}
	if update := s.GetBestUpdate(period - 1); update != nil {
		return update.NextSyncCommitteeRoot
	}
	return common.Hash{}
}

// GetSyncCommitteeRoot returns the sync committee root at the given period of the
// current local committee root constraints or update chain (tracker mutex locked).
func (s *CommitteeTracker) GetSyncCommitteeRoot(period uint64) common.Hash {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.getSyncCommitteeRoot(period)
}

// getSyncCommittee returns the deserialized sync committee at the given period
// of the current local committee chain (tracker mutex lock expected).
func (s *CommitteeTracker) getSyncCommittee(period uint64) syncCommittee {
	if committeeRoot := s.getSyncCommitteeRoot(period); committeeRoot != (common.Hash{}) {
		key := string(getSyncCommitteeKey(period, committeeRoot))
		if sc, ok := s.syncCommitteeCache.Get(key); ok {
			return sc
		}
		if sc := s.GetSerializedSyncCommittee(period, committeeRoot); sc != nil {
			c := s.sigVerifier.deserializeSyncCommittee(sc)
			s.syncCommitteeCache.Add(key, c)
			return c
		} else {
			log.Error("Missing serialized sync committee", "period", period, "committeeRoot", committeeRoot)
		}
	}
	return nil
}

// EnforceForksAndConstraints rolls back committee updates that do not match the
// tracker's forks and constraints and also starts new requests if possible
// (tracker mutex locked)
func (s *CommitteeTracker) EnforceForksAndConstraints() {
	s.lock.Lock()
	s.enforceForksAndConstraints()
	s.lock.Unlock()
}

// enforceForksAndConstraints rolls back committee updates that do not match the
// tracker's forks and constraints and also starts new requests if possible
// (tracker mutex expected)
func (s *CommitteeTracker) enforceForksAndConstraints() {
	if !s.genesisInit || !s.chainInit {
		return
	}
	s.committeeRootCache.Purge()
	for s.nextPeriod > s.firstPeriod {
		if update := s.GetBestUpdate(s.nextPeriod - 1); update == nil || s.verifyUpdate(update) { // check constraints and signature
			if update == nil {
				log.Error("Sync committee update missing", "period", s.nextPeriod-1)
			}
			break
		}
		s.nextPeriod--
		s.deleteBestUpdate(s.nextPeriod)
	}
	if s.nextPeriod == s.firstPeriod {
		if root, matchAll := s.constraints.CommitteeRoot(s.firstPeriod); matchAll || s.getSyncCommitteeRoot(s.firstPeriod) != root || s.getSyncCommittee(s.firstPeriod) == nil {
			s.nextPeriod, s.firstPeriod, s.chainInit = 0, 0, false
		}
	}

	s.retrySyncAllPeers()
}

// checkConstraints checks whether the signed headers of the given committee
// update is on the right fork and the proven NextSyncCommitteeRoot matches the
// update chain constraints.
func (s *CommitteeTracker) checkConstraints(update *types.LightClientUpdate) bool {
	if !s.genesisInit {
		log.Error("CommitteeTracker not initialized")
		return false
	}
	root, matchAll := s.constraints.CommitteeRoot(update.Header.SyncPeriod() + 1)
	return matchAll || root == update.NextSyncCommitteeRoot
}
