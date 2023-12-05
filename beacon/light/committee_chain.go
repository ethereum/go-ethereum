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

package light

import (
	"errors"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
)

var (
	ErrNeedCommittee      = errors.New("sync committee required")
	ErrInvalidUpdate      = errors.New("invalid committee update")
	ErrInvalidPeriod      = errors.New("invalid update period")
	ErrWrongCommitteeRoot = errors.New("wrong committee root")
	ErrCannotReorg        = errors.New("can not reorg committee chain")
)

// CommitteeChain is a passive data structure that can validate, hold and update
// a chain of beacon light sync committees and updates. It requires at least one
// externally set fixed committee root at the beginning of the chain which can
// be set either based on a BootstrapData or a trusted source (a local beacon
// full node). This makes the structure useful for both light client and light
// server setups.
//
// It always maintains the following consistency constraints:
//   - a committee can only be present if its root hash matches an existing fixed
//     root or if it is proven by an update at the previous period
//   - an update can only be present if a committee is present at the same period
//     and the update signature is valid and has enough participants.
//     The committee at the next period (proven by the update) should also be
//     present (note that this means they can only be added together if neither
//     is present yet). If a fixed root is present at the next period then the
//     update can only be present if it proves the same committee root.
//
// Once synced to the current sync period, CommitteeChain can also validate
// signed beacon headers.
type CommitteeChain struct {
	// chainmu guards against concurrent access to the canonicalStore structures
	// (updates, committees, fixedCommitteeRoots) and ensures that they stay consistent
	// with each other and with committeeCache.
	chainmu             sync.RWMutex
	db                  ethdb.KeyValueStore
	updates             *canonicalStore[*types.LightClientUpdate]
	committees          *canonicalStore[*types.SerializedSyncCommittee]
	fixedCommitteeRoots *canonicalStore[common.Hash]
	committeeCache      *lru.Cache[uint64, syncCommittee] // cache deserialized committees

	clock       mclock.Clock         // monotonic clock (simulated clock in tests)
	unixNano    func() int64         // system clock (simulated clock in tests)
	sigVerifier committeeSigVerifier // BLS sig verifier (dummy verifier in tests)

	config             *types.ChainConfig
	signerThreshold    int
	minimumUpdateScore types.UpdateScore
	enforceTime        bool // enforceTime specifies whether the age of a signed header should be checked
}

// NewCommitteeChain creates a new CommitteeChain.
func NewCommitteeChain(db ethdb.KeyValueStore, config *types.ChainConfig, signerThreshold int, enforceTime bool) *CommitteeChain {
	return newCommitteeChain(db, config, signerThreshold, enforceTime, blsVerifier{}, &mclock.System{}, func() int64 { return time.Now().UnixNano() })
}

// newCommitteeChain creates a new CommitteeChain with the option of replacing the
// clock source and signature verification for testing purposes.
func newCommitteeChain(db ethdb.KeyValueStore, config *types.ChainConfig, signerThreshold int, enforceTime bool, sigVerifier committeeSigVerifier, clock mclock.Clock, unixNano func() int64) *CommitteeChain {
	s := &CommitteeChain{
		committeeCache:  lru.NewCache[uint64, syncCommittee](10),
		db:              db,
		sigVerifier:     sigVerifier,
		clock:           clock,
		unixNano:        unixNano,
		config:          config,
		signerThreshold: signerThreshold,
		enforceTime:     enforceTime,
		minimumUpdateScore: types.UpdateScore{
			SignerCount:    uint32(signerThreshold),
			SubPeriodIndex: params.SyncPeriodLength / 16,
		},
	}

	var err1, err2, err3 error
	if s.fixedCommitteeRoots, err1 = newCanonicalStore[common.Hash](db, rawdb.FixedCommitteeRootKey); err1 != nil {
		log.Error("Error creating fixed committee root store", "error", err1)
	}
	if s.committees, err2 = newCanonicalStore[*types.SerializedSyncCommittee](db, rawdb.SyncCommitteeKey); err2 != nil {
		log.Error("Error creating committee store", "error", err2)
	}
	if s.updates, err3 = newCanonicalStore[*types.LightClientUpdate](db, rawdb.BestUpdateKey); err3 != nil {
		log.Error("Error creating update store", "error", err3)
	}
	if err1 != nil || err2 != nil || err3 != nil || !s.checkConstraints() {
		log.Info("Resetting invalid committee chain")
		s.Reset()
	}
	// roll back invalid updates (might be necessary if forks have been changed since last time)
	for !s.updates.periods.isEmpty() {
		update, ok := s.updates.get(s.db, s.updates.periods.End-1)
		if !ok {
			log.Error("Sync committee update missing", "period", s.updates.periods.End-1)
			s.Reset()
			break
		}
		if valid, err := s.verifyUpdate(update); err != nil {
			log.Error("Error validating update", "period", s.updates.periods.End-1, "error", err)
		} else if valid {
			break
		}
		if err := s.rollback(s.updates.periods.End); err != nil {
			log.Error("Error writing batch into chain database", "error", err)
		}
	}
	if !s.committees.periods.isEmpty() {
		log.Trace("Sync committee chain loaded", "first period", s.committees.periods.Start, "last period", s.committees.periods.End-1)
	}
	return s
}

// checkConstraints checks committee chain validity constraints
func (s *CommitteeChain) checkConstraints() bool {
	isNotInFixedCommitteeRootRange := func(r periodRange) bool {
		return s.fixedCommitteeRoots.periods.isEmpty() ||
			r.Start < s.fixedCommitteeRoots.periods.Start ||
			r.Start >= s.fixedCommitteeRoots.periods.End
	}

	valid := true
	if !s.updates.periods.isEmpty() {
		if isNotInFixedCommitteeRootRange(s.updates.periods) {
			log.Error("Start update is not in the fixed roots range")
			valid = false
		}
		if s.committees.periods.Start > s.updates.periods.Start || s.committees.periods.End <= s.updates.periods.End {
			log.Error("Missing committees in update range")
			valid = false
		}
	}
	if !s.committees.periods.isEmpty() {
		if isNotInFixedCommitteeRootRange(s.committees.periods) {
			log.Error("Start committee is not in the fixed roots range")
			valid = false
		}
		if s.committees.periods.End > s.fixedCommitteeRoots.periods.End && s.committees.periods.End > s.updates.periods.End+1 {
			log.Error("Last committee is neither in the fixed roots range nor proven by updates")
			valid = false
		}
	}
	return valid
}

// Reset resets the committee chain.
func (s *CommitteeChain) Reset() {
	s.chainmu.Lock()
	defer s.chainmu.Unlock()

	if err := s.rollback(0); err != nil {
		log.Error("Error writing batch into chain database", "error", err)
	}
}

// CheckpointInit initializes a CommitteeChain based on the checkpoint.
// Note: if the chain is already initialized and the committees proven by the
// checkpoint do match the existing chain then the chain is retained and the
// new checkpoint becomes fixed.
func (s *CommitteeChain) CheckpointInit(bootstrap *types.BootstrapData) error {
	s.chainmu.Lock()
	defer s.chainmu.Unlock()

	if err := bootstrap.Validate(); err != nil {
		return err
	}

	period := bootstrap.Header.SyncPeriod()
	if err := s.deleteFixedCommitteeRootsFrom(period + 2); err != nil {
		s.Reset()
		return err
	}
	if s.addFixedCommitteeRoot(period, bootstrap.CommitteeRoot) != nil {
		s.Reset()
		if err := s.addFixedCommitteeRoot(period, bootstrap.CommitteeRoot); err != nil {
			s.Reset()
			return err
		}
	}
	if err := s.addFixedCommitteeRoot(period+1, common.Hash(bootstrap.CommitteeBranch[0])); err != nil {
		s.Reset()
		return err
	}
	if err := s.addCommittee(period, bootstrap.Committee); err != nil {
		s.Reset()
		return err
	}
	return nil
}

// addFixedCommitteeRoot sets a fixed committee root at the given period.
// Note that the period where the first committee is added has to have a fixed
// root which can either come from a BootstrapData or a trusted source.
func (s *CommitteeChain) addFixedCommitteeRoot(period uint64, root common.Hash) error {
	if root == (common.Hash{}) {
		return ErrWrongCommitteeRoot
	}

	batch := s.db.NewBatch()
	oldRoot := s.getCommitteeRoot(period)
	if !s.fixedCommitteeRoots.periods.canExpand(period) {
		// Note: the fixed committee root range should always be continuous and
		// therefore the expected syncing method is to forward sync and optionally
		// backward sync periods one by one, starting from a checkpoint. The only
		// case when a root that is not adjacent to the already fixed ones can be
		// fixed is when the same root has already been proven by an update chain.
		// In this case the all roots in between can and should be fixed.
		// This scenario makes sense when a new trusted checkpoint is added to an
		// existing chain, ensuring that it will not be rolled back (might be
		// important in case of low signer participation rate).
		if root != oldRoot {
			return ErrInvalidPeriod
		}
		// if the old root exists and matches the new one then it is guaranteed
		// that the given period is after the existing fixed range and the roots
		// in between can also be fixed.
		for p := s.fixedCommitteeRoots.periods.End; p < period; p++ {
			if err := s.fixedCommitteeRoots.add(batch, p, s.getCommitteeRoot(p)); err != nil {
				return err
			}
		}
	}
	if oldRoot != (common.Hash{}) && (oldRoot != root) {
		// existing old root was different, we have to reorg the chain
		if err := s.rollback(period); err != nil {
			return err
		}
	}
	if err := s.fixedCommitteeRoots.add(batch, period, root); err != nil {
		return err
	}
	if err := batch.Write(); err != nil {
		log.Error("Error writing batch into chain database", "error", err)
		return err
	}
	return nil
}

// deleteFixedCommitteeRootsFrom deletes fixed roots starting from the given period.
// It also maintains chain consistency, meaning that it also deletes updates and
// committees if they are no longer supported by a valid update chain.
func (s *CommitteeChain) deleteFixedCommitteeRootsFrom(period uint64) error {
	if period >= s.fixedCommitteeRoots.periods.End {
		return nil
	}
	batch := s.db.NewBatch()
	s.fixedCommitteeRoots.deleteFrom(batch, period)
	if s.updates.periods.isEmpty() || period <= s.updates.periods.Start {
		// Note: the first period of the update chain should always be fixed so if
		// the fixed root at the first update is removed then the entire update chain
		// and the proven committees have to be removed. Earlier committees in the
		// remaining fixed root range can stay.
		s.updates.deleteFrom(batch, period)
		s.deleteCommitteesFrom(batch, period)
	} else {
		// The update chain stays intact, some previously fixed committee roots might
		// get unfixed but are still proven by the update chain. If there were
		// committees present after the range proven by updates, those should be
		// removed if the belonging fixed roots are also removed.
		fromPeriod := s.updates.periods.End + 1 // not proven by updates
		if period > fromPeriod {
			fromPeriod = period // also not justified by fixed roots
		}
		s.deleteCommitteesFrom(batch, fromPeriod)
	}
	if err := batch.Write(); err != nil {
		log.Error("Error writing batch into chain database", "error", err)
		return err
	}
	return nil
}

// deleteCommitteesFrom deletes committees starting from the given period.
func (s *CommitteeChain) deleteCommitteesFrom(batch ethdb.Batch, period uint64) {
	deleted := s.committees.deleteFrom(batch, period)
	for period := deleted.Start; period < deleted.End; period++ {
		s.committeeCache.Remove(period)
	}
}

// addCommittee adds a committee at the given period if possible.
func (s *CommitteeChain) addCommittee(period uint64, committee *types.SerializedSyncCommittee) error {
	if !s.committees.periods.canExpand(period) {
		return ErrInvalidPeriod
	}
	root := s.getCommitteeRoot(period)
	if root == (common.Hash{}) {
		return ErrInvalidPeriod
	}
	if root != committee.Root() {
		return ErrWrongCommitteeRoot
	}
	if !s.committees.periods.contains(period) {
		if err := s.committees.add(s.db, period, committee); err != nil {
			return err
		}
		s.committeeCache.Remove(period)
	}
	return nil
}

// InsertUpdate adds a new update if possible.
func (s *CommitteeChain) InsertUpdate(update *types.LightClientUpdate, nextCommittee *types.SerializedSyncCommittee) error {
	s.chainmu.Lock()
	defer s.chainmu.Unlock()

	period := update.AttestedHeader.Header.SyncPeriod()
	if !s.updates.periods.canExpand(period) || !s.committees.periods.contains(period) {
		return ErrInvalidPeriod
	}
	if s.minimumUpdateScore.BetterThan(update.Score()) {
		return ErrInvalidUpdate
	}
	oldRoot := s.getCommitteeRoot(period + 1)
	reorg := oldRoot != (common.Hash{}) && oldRoot != update.NextSyncCommitteeRoot
	if oldUpdate, ok := s.updates.get(s.db, period); ok && !update.Score().BetterThan(oldUpdate.Score()) {
		// a better or equal update already exists; no changes, only fail if new one tried to reorg
		if reorg {
			return ErrCannotReorg
		}
		return nil
	}
	if s.fixedCommitteeRoots.periods.contains(period+1) && reorg {
		return ErrCannotReorg
	}
	if ok, err := s.verifyUpdate(update); err != nil {
		return err
	} else if !ok {
		return ErrInvalidUpdate
	}
	addCommittee := !s.committees.periods.contains(period+1) || reorg
	if addCommittee {
		if nextCommittee == nil {
			return ErrNeedCommittee
		}
		if nextCommittee.Root() != update.NextSyncCommitteeRoot {
			return ErrWrongCommitteeRoot
		}
	}
	if reorg {
		if err := s.rollback(period + 1); err != nil {
			return err
		}
	}
	batch := s.db.NewBatch()
	if addCommittee {
		if err := s.committees.add(batch, period+1, nextCommittee); err != nil {
			return err
		}
		s.committeeCache.Remove(period + 1)
	}
	if err := s.updates.add(batch, period, update); err != nil {
		return err
	}
	if err := batch.Write(); err != nil {
		log.Error("Error writing batch into chain database", "error", err)
		return err
	}
	log.Info("Inserted new committee update", "period", period, "next committee root", update.NextSyncCommitteeRoot)
	return nil
}

// NextSyncPeriod returns the next period where an update can be added and also
// whether the chain is initialized at all.
func (s *CommitteeChain) NextSyncPeriod() (uint64, bool) {
	s.chainmu.RLock()
	defer s.chainmu.RUnlock()

	if s.committees.periods.isEmpty() {
		return 0, false
	}
	if !s.updates.periods.isEmpty() {
		return s.updates.periods.End, true
	}
	return s.committees.periods.End - 1, true
}

// rollback removes all committees and fixed roots from the given period and updates
// starting from the previous period.
func (s *CommitteeChain) rollback(period uint64) error {
	max := s.updates.periods.End + 1
	if s.committees.periods.End > max {
		max = s.committees.periods.End
	}
	if s.fixedCommitteeRoots.periods.End > max {
		max = s.fixedCommitteeRoots.periods.End
	}
	for max > period {
		max--
		batch := s.db.NewBatch()
		s.deleteCommitteesFrom(batch, max)
		s.fixedCommitteeRoots.deleteFrom(batch, max)
		if max > 0 {
			s.updates.deleteFrom(batch, max-1)
		}
		if err := batch.Write(); err != nil {
			log.Error("Error writing batch into chain database", "error", err)
			return err
		}
	}
	return nil
}

// getCommitteeRoot returns the committee root at the given period, either fixed,
// proven by a previous update or both. It returns an empty hash if the committee
// root is unknown.
func (s *CommitteeChain) getCommitteeRoot(period uint64) common.Hash {
	if root, ok := s.fixedCommitteeRoots.get(s.db, period); ok || period == 0 {
		return root
	}
	if update, ok := s.updates.get(s.db, period-1); ok {
		return update.NextSyncCommitteeRoot
	}
	return common.Hash{}
}

// getSyncCommittee returns the deserialized sync committee at the given period.
func (s *CommitteeChain) getSyncCommittee(period uint64) (syncCommittee, error) {
	if c, ok := s.committeeCache.Get(period); ok {
		return c, nil
	}
	if sc, ok := s.committees.get(s.db, period); ok {
		c, err := s.sigVerifier.deserializeSyncCommittee(sc)
		if err != nil {
			return nil, fmt.Errorf("Sync committee #%d deserialization error: %v", period, err)
		}
		s.committeeCache.Add(period, c)
		return c, nil
	}
	return nil, fmt.Errorf("Missing serialized sync committee #%d", period)
}

// VerifySignedHeader returns true if the given signed header has a valid signature
// according to the local committee chain. The caller should ensure that the
// committees advertised by the same source where the signed header came from are
// synced before verifying the signature.
// The age of the header is also returned (the time elapsed since the beginning
// of the given slot, according to the local system clock). If enforceTime is
// true then negative age (future) headers are rejected.
func (s *CommitteeChain) VerifySignedHeader(head types.SignedHeader) (bool, time.Duration, error) {
	s.chainmu.RLock()
	defer s.chainmu.RUnlock()

	return s.verifySignedHeader(head)
}

func (s *CommitteeChain) verifySignedHeader(head types.SignedHeader) (bool, time.Duration, error) {
	var age time.Duration
	now := s.unixNano()
	if head.Header.Slot < (uint64(now-math.MinInt64)/uint64(time.Second)-s.config.GenesisTime)/12 {
		age = time.Duration(now - int64(time.Second)*int64(s.config.GenesisTime+head.Header.Slot*12))
	} else {
		age = time.Duration(math.MinInt64)
	}
	if s.enforceTime && age < 0 {
		return false, age, nil
	}
	committee, err := s.getSyncCommittee(types.SyncPeriod(head.SignatureSlot))
	if err != nil {
		return false, 0, err
	}
	if committee == nil {
		return false, age, nil
	}
	if signingRoot, err := s.config.Forks.SigningRoot(head.Header); err == nil {
		return s.sigVerifier.verifySignature(committee, signingRoot, &head.Signature), age, nil
	}
	return false, age, nil
}

// verifyUpdate checks whether the header signature is correct and the update
// fits into the specified constraints (assumes that the update has been
// successfully validated previously)
func (s *CommitteeChain) verifyUpdate(update *types.LightClientUpdate) (bool, error) {
	// Note: SignatureSlot determines the sync period of the committee used for signature
	// verification. Though in reality SignatureSlot is always bigger than update.Header.Slot,
	// setting them as equal here enforces the rule that they have to be in the same sync
	// period in order for the light client update proof to be meaningful.
	ok, age, err := s.verifySignedHeader(update.AttestedHeader)
	if age < 0 {
		log.Warn("Future committee update received", "age", age)
	}
	return ok, err
}
