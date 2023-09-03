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
	"encoding/binary"
	"errors"
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
	"github.com/ethereum/go-ethereum/rlp"
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
// be set either based on a CheckpointData or a trusted source (a local beacon
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
	chainmu            sync.RWMutex // locks database, cache and canonicalStore access
	db                 ethdb.KeyValueStore
	updates            *canonicalStore[*types.LightClientUpdate]
	committees         *canonicalStore[*types.SerializedSyncCommittee]
	fixedRoots         *canonicalStore[common.Hash]
	syncCommitteeCache *lru.Cache[uint64, syncCommittee] // cache deserialized committees

	clock       mclock.Clock         // monotonic clock (simulated clock in tests)
	unixNano    func() int64         // system clock (simulated clock in tests)
	sigVerifier committeeSigVerifier // BLS sig verifier (dummy verifier in tests)

	config             *types.ChainConfig
	signerThreshold    int
	minimumUpdateScore types.UpdateScore
	enforceTime        bool
}

// NewCommitteeChain creates a new CommitteeChain.
func NewCommitteeChain(db ethdb.KeyValueStore, config *types.ChainConfig, signerThreshold int, enforceTime bool, sigVerifier committeeSigVerifier, clock mclock.Clock, unixNano func() int64) *CommitteeChain {
	s := &CommitteeChain{
		fixedRoots: newCanonicalStore[common.Hash](db, rawdb.FixedRootKey, func(root common.Hash) ([]byte, error) {
			return root[:], nil
		}, func(enc []byte) (root common.Hash, err error) {
			if len(enc) != common.HashLength {
				return common.Hash{}, errors.New("incorrect length for committee root entry in the database")
			}
			return common.BytesToHash(enc), nil
		}),
		committees: newCanonicalStore[*types.SerializedSyncCommittee](db, rawdb.SyncCommitteeKey, func(committee *types.SerializedSyncCommittee) ([]byte, error) {
			return committee[:], nil
		}, func(enc []byte) (*types.SerializedSyncCommittee, error) {
			if len(enc) == types.SerializedSyncCommitteeSize {
				committee := new(types.SerializedSyncCommittee)
				copy(committee[:], enc)
				return committee, nil
			}
			return nil, errors.New("incorrect length for serialized committee entry in the database")
		}),
		updates: newCanonicalStore[*types.LightClientUpdate](db, rawdb.BestUpdateKey, func(update *types.LightClientUpdate) ([]byte, error) {
			return rlp.EncodeToBytes(update)
		}, func(enc []byte) (*types.LightClientUpdate, error) {
			update := new(types.LightClientUpdate)
			if err := rlp.DecodeBytes(enc, update); err != nil {
				return nil, err
			}
			return update, nil
		}),
		syncCommitteeCache: lru.NewCache[uint64, syncCommittee](10),
		db:                 db,
		sigVerifier:        sigVerifier,
		clock:              clock,
		unixNano:           unixNano,
		config:             config,
		signerThreshold:    signerThreshold,
		enforceTime:        enforceTime,
		minimumUpdateScore: types.UpdateScore{
			SignerCount:    uint32(signerThreshold),
			SubPeriodIndex: params.SyncPeriodLength / 16,
		},
	}

	// check validity constraints
	if !s.updates.periods.IsEmpty() {
		if s.fixedRoots.periods.IsEmpty() || s.updates.periods.First < s.fixedRoots.periods.First ||
			s.updates.periods.First >= s.fixedRoots.periods.AfterLast {
			log.Error("Inconsistent database error: first update is not in the fixed roots range")
		}
		if s.committees.periods.First > s.updates.periods.First || s.committees.periods.AfterLast <= s.updates.periods.AfterLast {
			log.Error("Inconsistent database error: missing committees in update range")
		}
	}
	if !s.committees.periods.IsEmpty() {
		if s.fixedRoots.periods.IsEmpty() || s.committees.periods.First < s.fixedRoots.periods.First ||
			s.committees.periods.First >= s.fixedRoots.periods.AfterLast {
			log.Error("Inconsistent database error: first committee is not in the fixed roots range")
		}
		if s.committees.periods.AfterLast > s.fixedRoots.periods.AfterLast && s.committees.periods.AfterLast > s.updates.periods.AfterLast+1 {
			log.Error("Inconsistent database error: last committee is neither in the fixed roots range nor proven by updates")
		}
		log.Trace("Sync committee chain loaded", "first period", s.committees.periods.First, "last period", s.committees.periods.AfterLast-1)
	}
	// roll back invalid updates (might be necessary if forks have been changed since last time)
	var batch ethdb.Batch
	for !s.updates.periods.IsEmpty() {
		if update, ok := s.updates.get(s.updates.periods.AfterLast - 1); !ok || s.verifyUpdate(update) {
			if update == nil {
				log.Error("Sync committee update missing", "period", s.updates.periods.AfterLast-1)
			}
			break
		}
		if batch == nil {
			batch = s.db.NewBatch()
		}
		s.rollback(batch, s.updates.periods.AfterLast)
	}
	if batch != nil {
		if err := batch.Write(); err != nil {
			log.Error("Error writing batch into chain database", "error", err)
		}
	}
	return s
}

// Reset resets the committee chain.
func (s *CommitteeChain) Reset() {
	s.chainmu.Lock()
	defer s.chainmu.Unlock()

	batch := s.db.NewBatch()
	s.rollback(batch, 0)
	if err := batch.Write(); err != nil {
		log.Error("Error writing batch into chain database", "error", err)
	}
}

// AddFixedRoot sets a fixed committee root at the given period.
// Note that the period where the first committee is added has to have a fixed
// root which can either come from a CheckpointData or a trusted source.
func (s *CommitteeChain) AddFixedRoot(period uint64, root common.Hash) error {
	s.chainmu.Lock()
	defer s.chainmu.Unlock()

	batch := s.db.NewBatch()
	oldRoot := s.getCommitteeRoot(period)
	if !s.fixedRoots.periods.CanExpand(period) {
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
		for p := s.fixedRoots.periods.AfterLast; p < period; p++ {
			if err := s.fixedRoots.add(batch, p, s.getCommitteeRoot(p)); err != nil {
				return err
			}
		}
	}
	if oldRoot != (common.Hash{}) && (oldRoot != root) {
		// existing old root was different, we have to reorg the chain
		s.rollback(batch, period)
	}
	if err := s.fixedRoots.add(batch, period, root); err != nil {
		return err
	}
	if err := batch.Write(); err != nil {
		log.Error("Error writing batch into chain database", "error", err)
		return err
	}
	return nil
}

// DeleteFixedRootsFrom deletes fixed roots starting from the given period.
// It also maintains chain consistency, meaning that it also deletes updates and
// committees if they are no longer supported by a valid update chain.
func (s *CommitteeChain) DeleteFixedRootsFrom(period uint64) error {
	s.chainmu.Lock()
	defer s.chainmu.Unlock()

	if period >= s.fixedRoots.periods.AfterLast {
		return nil
	}
	batch := s.db.NewBatch()
	s.fixedRoots.deleteFrom(batch, period)
	if s.updates.periods.IsEmpty() || period <= s.updates.periods.First {
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
		fromPeriod := s.updates.periods.AfterLast + 1 // not proven by updates
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
	for period := deleted.First; period < deleted.AfterLast; period++ {
		s.syncCommitteeCache.Remove(period)
	}
}

// GetCommittee returns the committee at the given period.
// Note: GetCommittee can be called either with locked or unlocked chain mutex.
func (s *CommitteeChain) GetCommittee(period uint64) *types.SerializedSyncCommittee {
	committee, _ := s.committees.get(period)
	return committee
}

// AddCommittee adds a committee at the given period if possible.
func (s *CommitteeChain) AddCommittee(period uint64, committee *types.SerializedSyncCommittee) error {
	s.chainmu.Lock()
	defer s.chainmu.Unlock()

	if !s.committees.periods.CanExpand(period) {
		return ErrInvalidPeriod
	}
	root := s.getCommitteeRoot(period)
	if root == (common.Hash{}) {
		return ErrInvalidPeriod
	}
	if root != committee.Root() {
		return ErrWrongCommitteeRoot
	}
	if !s.committees.periods.Includes(period) {
		if err := s.committees.add(s.db, period, committee); err != nil {
			return err
		}
		s.syncCommitteeCache.Remove(period)
	}
	return nil
}

// GetUpdate returns the update at the given period.
// Note: GetUpdate can be called either with locked or unlocked chain mutex.
func (s *CommitteeChain) GetUpdate(period uint64) *types.LightClientUpdate {
	update, _ := s.updates.get(period)
	return update
}

// InsertUpdate adds a new update if possible.
func (s *CommitteeChain) InsertUpdate(update *types.LightClientUpdate, nextCommittee *types.SerializedSyncCommittee) error {
	s.chainmu.Lock()
	defer s.chainmu.Unlock()

	period := update.AttestedHeader.Header.SyncPeriod()
	if !s.updates.periods.CanExpand(period) || !s.committees.periods.Includes(period) {
		return ErrInvalidPeriod
	}
	if s.minimumUpdateScore.BetterThan(update.Score()) {
		return ErrInvalidUpdate
	}
	oldRoot := s.getCommitteeRoot(period + 1)
	reorg := oldRoot != (common.Hash{}) && oldRoot != update.NextSyncCommitteeRoot
	if oldUpdate, ok := s.updates.get(period); ok && !update.Score().BetterThan(oldUpdate.Score()) {
		// a better or equal update already exists; no changes, only fail if new one tried to reorg
		if reorg {
			return ErrCannotReorg
		}
		return nil
	}
	if s.fixedRoots.periods.Includes(period+1) && reorg {
		return ErrCannotReorg
	}
	if !s.verifyUpdate(update) {
		return ErrInvalidUpdate
	}
	addCommittee := !s.committees.periods.Includes(period+1) || reorg
	if addCommittee {
		if nextCommittee == nil {
			return ErrNeedCommittee
		}
		if nextCommittee.Root() != update.NextSyncCommitteeRoot {
			return ErrWrongCommitteeRoot
		}
	}
	batch := s.db.NewBatch()
	if reorg {
		s.rollback(batch, period+1)
	}
	if addCommittee {
		if err := s.committees.add(batch, period+1, nextCommittee); err != nil {
			return err
		}
		s.syncCommitteeCache.Remove(period + 1)
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

	if s.committees.periods.IsEmpty() {
		return 0, false
	}
	if !s.updates.periods.IsEmpty() {
		return s.updates.periods.AfterLast, true
	}
	return s.committees.periods.AfterLast - 1, true
}

// rollback removes all committees and fixed roots from the given period and updates
// starting from the previous period.
func (s *CommitteeChain) rollback(batch ethdb.Batch, period uint64) {
	s.deleteCommitteesFrom(batch, period)
	s.fixedRoots.deleteFrom(batch, period)
	if period > 0 {
		period--
	}
	s.updates.deleteFrom(batch, period)
}

// getCommitteeRoot returns the committee root at the given period, either fixed,
// proven by a previous update or both. It returns an empty hash if the committee
// root is unknown.
func (s *CommitteeChain) getCommitteeRoot(period uint64) common.Hash {
	if root, ok := s.fixedRoots.get(period); ok || period == 0 {
		return root
	}
	if update, ok := s.updates.get(period - 1); ok {
		return update.NextSyncCommitteeRoot
	}
	return common.Hash{}
}

// getSyncCommittee returns the deserialized sync committee at the given period.
func (s *CommitteeChain) getSyncCommittee(period uint64) syncCommittee {
	if c, ok := s.syncCommitteeCache.Get(period); ok {
		return c
	}
	if sc, ok := s.committees.get(period); ok {
		c, err := s.sigVerifier.deserializeSyncCommittee(sc)
		if err != nil {
			log.Error("Sync committee deserialization error", "error", err)
			return nil
		}
		s.syncCommitteeCache.Add(period, c)
		return c
	}
	log.Error("Missing serialized sync committee", "period", period)
	return nil
}

// VerifySignedHeader returns true if the given signed header has a valid signature
// according to the local committee chain. The caller should ensure that the
// committees advertised by the same source where the signed header came from are
// synced before verifying the signature.
// The age of the header is also returned (the time elapsed since the beginning
// of the given slot, according to the local system clock). If enforceTime is
// true then negative age (future) headers are rejected.
func (s *CommitteeChain) VerifySignedHeader(head types.SignedHeader) (bool, time.Duration) {
	s.chainmu.RLock()
	defer s.chainmu.RUnlock()

	return s.verifySignedHeader(head)
}

func (s *CommitteeChain) verifySignedHeader(head types.SignedHeader) (bool, time.Duration) {
	var age time.Duration
	now := s.unixNano()
	if head.Header.Slot < (uint64(now-math.MinInt64)/uint64(time.Second)-s.config.GenesisTime)/12 {
		age = time.Duration(now - int64(time.Second)*int64(s.config.GenesisTime+head.Header.Slot*12))
	} else {
		age = time.Duration(math.MinInt64)
	}
	if s.enforceTime && age < 0 {
		return false, age
	}
	committee := s.getSyncCommittee(types.SyncPeriod(head.SignatureSlot))
	if committee == nil {
		return false, age
	}
	if signingRoot, err := s.config.Forks.SigningRoot(head.Header); err == nil {
		return s.sigVerifier.verifySignature(committee, signingRoot, &head.Signature), age
	}
	return false, age
}

// verifyUpdate checks whether the header signature is correct and the update
// fits into the specified constraints (assumes that the update has been
// successfully validated previously)
func (s *CommitteeChain) verifyUpdate(update *types.LightClientUpdate) bool {
	// Note: SignatureSlot determines the sync period of the committee used for signature
	// verification. Though in reality SignatureSlot is always bigger than update.Header.Slot,
	// setting them as equal here enforces the rule that they have to be in the same sync
	// period in order for the light client update proof to be meaningful.
	ok, age := s.verifySignedHeader(update.AttestedHeader)
	if age < 0 {
		log.Warn("Future committee update received", "age", age)
	}
	return ok
}

// canonicalStore stores instances of the given type in a database and caches
// them in memory, associated with a continuous range of period numbers.
type canonicalStore[T any] struct {
	db        ethdb.KeyValueStore
	keyPrefix []byte
	periods   Range
	cache     *lru.Cache[uint64, T]
	encode    func(T) ([]byte, error)
	decode    func([]byte) (T, error)
}

// newCanonicalStore creates a new canonicalStore.
func newCanonicalStore[T any](db ethdb.KeyValueStore, keyPrefix []byte,
	encode func(T) ([]byte, error), decode func([]byte) (T, error)) *canonicalStore[T] {
	cs := &canonicalStore[T]{
		db:        db,
		keyPrefix: keyPrefix,
		encode:    encode,
		decode:    decode,
		cache:     lru.NewCache[uint64, T](100),
	}
	var (
		iter = db.NewIterator(keyPrefix, nil)
		kl   = len(keyPrefix)
	)
	for iter.Next() {
		if len(iter.Key()) != kl+8 {
			log.Error("Invalid key length in the canonical chain database")
			continue
		}
		period := binary.BigEndian.Uint64(iter.Key()[kl : kl+8])
		if cs.periods.First == 0 {
			cs.periods.First = period
		} else if cs.periods.AfterLast != period {
			if iter.Next() {
				log.Error("Gap in the canonical chain database")
			}
			break // continuity guaranteed
		}
		cs.periods.AfterLast = period + 1
	}
	iter.Release()
	return cs
}

// databaseKey returns the database key belonging to the given period.
func (cs *canonicalStore[T]) databaseKey(period uint64) []byte {
	var (
		kl  = len(cs.keyPrefix)
		key = make([]byte, kl+8)
	)
	copy(key[:kl], cs.keyPrefix)
	binary.BigEndian.PutUint64(key[kl:], period)
	return key
}

// add adds the given item to the database. It also ensures that the range remains
// continuous. Can be used either with a batch or database backend.
func (cs *canonicalStore[T]) add(backend ethdb.KeyValueWriter, period uint64, value T) error {
	if !cs.periods.CanExpand(period) {
		log.Error("Cannot expand canonical store", "range.first", cs.periods.First, "range.afterLast", cs.periods.AfterLast, "new period", period)
		return errors.New("Cannot expand canonical store")
	}
	enc, err := cs.encode(value)
	if err != nil {
		log.Error("Error encoding canonical store value", "error", err)
		return err
	}
	if err := backend.Put(cs.databaseKey(period), enc); err != nil {
		log.Error("Error writing into canonical store value database", "error", err)
		return err
	}
	cs.cache.Add(period, value)
	cs.periods.Expand(period)
	return nil
}

// deleteFrom removes items starting from the given period. Should be used with a
// batch backend.
func (cs *canonicalStore[T]) deleteFrom(backend ethdb.KeyValueWriter, fromPeriod uint64) (deleted Range) {
	if fromPeriod >= cs.periods.AfterLast {
		return
	}
	if fromPeriod < cs.periods.First {
		fromPeriod = cs.periods.First
	}
	deleted = Range{First: fromPeriod, AfterLast: cs.periods.AfterLast}
	for period := fromPeriod; period < cs.periods.AfterLast; period++ {
		backend.Delete(cs.databaseKey(period))
		cs.cache.Remove(period)
	}
	if fromPeriod > cs.periods.First {
		cs.periods.AfterLast = fromPeriod
	} else {
		cs.periods = Range{}
	}
	return
}

// get returns the item at the given period or the null value of the given type
// if no item is present.
// Note: get is thread safe in itself and therefore can be called either with
// locked or unlocked chain mutex.
func (cs *canonicalStore[T]) get(period uint64) (value T, ok bool) {
	if value, ok = cs.cache.Get(period); ok {
		return
	}
	if enc, err := cs.db.Get(cs.databaseKey(period)); err == nil {
		if v, err := cs.decode(enc); err == nil {
			value, ok = v, true
		} else {
			log.Error("Error decoding canonical store value", "error", err)
		}
	}
	return
}
