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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	ErrNotInitialized     = errors.New("sync committee chain not initialized")
	ErrNeedCommittee      = errors.New("sync committee required")
	ErrInvalidUpdate      = errors.New("invalid committee update")
	ErrInvalidPeriod      = errors.New("invalid update period")
	ErrWrongCommitteeRoot = errors.New("wrong committee root")
	ErrCannotReorg        = errors.New("can not reorg committee chain")
)

var (
	bestUpdateKey    = []byte("update-")    // bigEndian64(syncPeriod) -> RLP(types.LightClientUpdate)  (nextCommittee only referenced by root hash)
	fixedRootKey     = []byte("fixedRoot-") // bigEndian64(syncPeriod) -> committee root hash
	syncCommitteeKey = []byte("committee-") // bigEndian64(syncPeriod) -> serialized committee
)

// CommitteeChain maintains a chain of sync committee updates and a small
// set of best known signed heads. It is used in all client configurations
// operating on a beacon chain. It can sync its update chain and receive signed
// heads from either an ODR or beacon node API backend and propagate/serve this
// data to subscribed peers. Received signed heads are validated based on the
// known sync committee chain and added to the local set if valid or placed in a
// deferred queue if the committees are not synced up to the period of the new
// head yet.
// Sync committee chain is either initialized from a weak subjectivity checkpoint
// or controlled by a BeaconChain that is driven by a trusted source (beacon node API).
type CommitteeChain struct {
	lock               sync.RWMutex
	db                 ethdb.KeyValueStore
	sigVerifier        committeeSigVerifier
	clock              mclock.Clock
	updates            *canonicalStore[*types.LightClientUpdate]
	committees         *canonicalStore[*types.SerializedSyncCommittee]
	fixedRoots         *canonicalStore[common.Hash]
	syncCommitteeCache *lru.Cache[uint64, syncCommittee] // cache deserialized committees
	unixNano           func() int64

	config             types.ChainConfig
	signerThreshold    int
	minimumUpdateScore types.UpdateScore
	enforceTime        bool
}

// NewCommitteeChain creates a new CommitteeChain
func NewCommitteeChain(db ethdb.KeyValueStore, config types.ChainConfig, signerThreshold int, enforceTime bool, sigVerifier committeeSigVerifier, clock mclock.Clock, unixNano func() int64) *CommitteeChain {
	s := &CommitteeChain{
		fixedRoots: newCanonicalStore[common.Hash](db, fixedRootKey, func(root common.Hash) ([]byte, error) {
			return root[:], nil
		}, func(enc []byte) (root common.Hash, err error) {
			if len(enc) == len(root) {
				copy(root[:], enc)
			} else {
				err = errors.New("Incorrect length for committee root entry in the database")
			}
			return
		}),
		committees: newCanonicalStore[*types.SerializedSyncCommittee](db, syncCommitteeKey, func(committee *types.SerializedSyncCommittee) ([]byte, error) {
			return committee[:], nil
		}, func(enc []byte) (*types.SerializedSyncCommittee, error) {
			if len(enc) == types.SerializedSyncCommitteeSize {
				committee := new(types.SerializedSyncCommittee)
				copy(committee[:], enc)
				return committee, nil
			}
			return nil, errors.New("Incorrect length for serialized committee entry in the database")
		}),
		updates: newCanonicalStore[*types.LightClientUpdate](db, bestUpdateKey, func(update *types.LightClientUpdate) ([]byte, error) {
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
	if !s.updates.IsEmpty() {
		if s.fixedRoots.IsEmpty() || s.updates.First < s.fixedRoots.First ||
			s.updates.First >= s.fixedRoots.AfterLast {
			log.Crit("Inconsistent database error: first update is not in the fixed roots range")
		}
		if s.committees.First > s.updates.First || s.committees.AfterLast <= s.updates.AfterLast {
			log.Crit("Inconsistent database error: missing committees in update range")
		}
	}
	if !s.committees.IsEmpty() {
		if s.fixedRoots.IsEmpty() || s.committees.First < s.fixedRoots.First ||
			s.committees.First >= s.fixedRoots.AfterLast {
			log.Crit("Inconsistent database error: first committee is not in the fixed roots range")
		}
		if s.committees.AfterLast > s.fixedRoots.AfterLast && s.committees.AfterLast > s.updates.AfterLast+1 {
			log.Crit("Inconsistent database error: last committee is neither in the fixed roots range nor proven by updates")
		}
		log.Trace("Sync committee chain loaded", "first period", s.committees.First, "last period", s.committees.AfterLast-1)
	}
	// roll back invalid updates (might be necessary if forks have been changed since last time)
	var batch ethdb.Batch
	for !s.updates.IsEmpty() {
		if update := s.updates.get(s.updates.AfterLast - 1); update == nil || s.verifyUpdate(update) {
			if update == nil {
				log.Crit("Sync committee update missing", "period", s.updates.AfterLast-1)
			}
			break
		}
		if batch == nil {
			batch = s.db.NewBatch()
		}
		s.rollback(batch, s.updates.AfterLast)
	}
	if batch != nil {
		if err := batch.Write(); err != nil {
			log.Error("Error writing batch into chain database", "error", err)
		}
	}
	return s
}

func (s *CommitteeChain) Reset() {
	s.lock.Lock()
	defer s.lock.Unlock()

	batch := s.db.NewBatch()
	s.rollback(batch, 0)
	if err := batch.Write(); err != nil {
		log.Error("Error writing batch into chain database", "error", err)
	}
}

func (s *CommitteeChain) AddFixedRoot(period uint64, root common.Hash) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	batch := s.db.NewBatch()
	oldRoot := s.getCommitteeRoot(period)
	if !s.fixedRoots.CanExpand(period) {
		if root != oldRoot {
			return ErrInvalidPeriod
		}
		for p := s.fixedRoots.AfterLast; p <= period; p++ {
			s.fixedRoots.add(batch, p, s.getCommitteeRoot(p))
		}
	}
	if oldRoot != (common.Hash{}) && (oldRoot != root) {
		// existing old root was different, we have to reorg the chain
		s.rollback(batch, period)
	}
	s.fixedRoots.add(batch, period, root)
	if err := batch.Write(); err != nil {
		log.Error("Error writing batch into chain database", "error", err)
		return err
	}
	return nil
}

func (s *CommitteeChain) DeleteFixedRootsFrom(period uint64) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if period >= s.fixedRoots.AfterLast {
		return nil
	}
	batch := s.db.NewBatch()
	s.fixedRoots.deleteFrom(batch, period)
	if s.updates.IsEmpty() || period <= s.updates.First {
		s.updates.deleteFrom(batch, period)
		s.deleteCommitteesFrom(batch, period)
	} else {
		fromPeriod := s.updates.AfterLast + 1
		if period > fromPeriod {
			fromPeriod = period
		}
		s.deleteCommitteesFrom(batch, fromPeriod)
	}
	if err := batch.Write(); err != nil {
		log.Error("Error writing batch into chain database", "error", err)
		return err
	}
	return nil
}

func (s *CommitteeChain) deleteCommitteesFrom(batch ethdb.Batch, period uint64) {
	deleted := s.committees.deleteFrom(batch, period)
	for period := deleted.First; period < deleted.AfterLast; period++ {
		s.syncCommitteeCache.Remove(period)
	}
}

func (s *CommitteeChain) GetCommittee(period uint64) *types.SerializedSyncCommittee {
	return s.committees.get(period)
}

func (s *CommitteeChain) AddCommittee(period uint64, committee *types.SerializedSyncCommittee) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if !s.committees.CanExpand(period) {
		return ErrInvalidPeriod
	}
	root := s.getCommitteeRoot(period)
	if root == (common.Hash{}) {
		return ErrInvalidPeriod
	}
	if root != committee.Root() {
		return ErrWrongCommitteeRoot
	}
	if !s.committees.Includes(period) {
		s.committees.add(nil, period, committee)
		s.syncCommitteeCache.Remove(period)
	}
	return nil
}

func (s *CommitteeChain) GetUpdate(period uint64) *types.LightClientUpdate {
	return s.updates.get(period)
}

func (s *CommitteeChain) InsertUpdate(update *types.LightClientUpdate, nextCommittee *types.SerializedSyncCommittee) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	period := update.AttestedHeader.Header.SyncPeriod()
	if !s.updates.CanExpand(period) || !s.committees.Includes(period) {
		return ErrInvalidPeriod
	}
	if s.minimumUpdateScore.BetterThan(update.Score()) {
		return ErrInvalidUpdate
	}
	oldRoot := s.getCommitteeRoot(period + 1)
	reorg := oldRoot != (common.Hash{}) && oldRoot != update.NextSyncCommitteeRoot
	if oldUpdate := s.updates.get(period); oldUpdate != nil && !update.Score().BetterThan(oldUpdate.Score()) {
		// a better or equal update already exists; no changes, only fail if new one tried to reorg
		if reorg {
			return ErrCannotReorg
		}
		return nil
	}
	if s.fixedRoots.Includes(period+1) && reorg {
		return ErrCannotReorg
	}
	if !s.verifyUpdate(update) {
		return ErrInvalidUpdate
	}
	addCommittee := !s.committees.Includes(period+1) || reorg
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
		s.committees.add(batch, period+1, nextCommittee)
		s.syncCommitteeCache.Remove(period + 1)
	}
	s.updates.add(batch, period, update)
	if err := batch.Write(); err != nil {
		log.Error("Error writing batch into chain database", "error", err)
		return err
	}
	log.Info("Inserted new committee update", "period", period, "next committee root", update.NextSyncCommitteeRoot)
	return nil
}

func (s *CommitteeChain) NextSyncPeriod() (uint64, bool) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	if s.committees.IsEmpty() {
		return 0, false
	}
	if !s.updates.IsEmpty() {
		return s.updates.AfterLast, true
	}
	return s.committees.AfterLast - 1, true
}

func (s *CommitteeChain) rollback(batch ethdb.Batch, period uint64) {
	s.deleteCommitteesFrom(batch, period)
	s.fixedRoots.deleteFrom(batch, period)
	if period > 0 {
		period--
	}
	s.updates.deleteFrom(batch, period)
}

func (s *CommitteeChain) getCommitteeRoot(period uint64) common.Hash {
	if root := s.fixedRoots.get(period); root != (common.Hash{}) || period == 0 {
		return root
	}
	if update := s.updates.get(period - 1); update != nil {
		return update.NextSyncCommitteeRoot
	}
	return common.Hash{}
}

// getSyncCommittee returns the deserialized sync committee at the given period
// of the current local committee chain (tracker mutex lock expected).
func (s *CommitteeChain) getSyncCommittee(period uint64) syncCommittee {
	if c, ok := s.syncCommitteeCache.Get(period); ok {
		return c
	}
	if sc := s.committees.get(period); sc != nil {
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

// VerifySignedHeader returns true if the given signed head has a valid signature
// according to the local committee chain. The caller should ensure that the
// committees advertised by the same source where the signed head came from are
// synced before verifying the signature.
// The age of the header is also returned (the time elapsed since the beginning
// of the given slot, according to the local system clock). If enforceTime is
// true then negative age (future) headers are rejected.
func (s *CommitteeChain) VerifySignedHeader(head types.SignedHeader) (bool, time.Duration) {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.verifySignedHeader(head)
}

// (rlock required)
func (s *CommitteeChain) verifySignedHeader(head types.SignedHeader) (bool, time.Duration) {
	var (
		slotTime = int64(time.Second) * int64(s.config.GenesisTime+head.Header.Slot*12)
		age      = time.Duration(s.unixNano() - slotTime)
	)
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
// (rlock required)
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

type canonicalStore[T any] struct {
	Range
	db        ethdb.KeyValueStore
	keyPrefix []byte
	cache     *lru.Cache[uint64, T]
	encode    func(T) ([]byte, error)
	decode    func([]byte) (T, error)
}

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
		period := binary.BigEndian.Uint64(iter.Key()[kl : kl+8])
		if cs.First == 0 {
			cs.First = period
		} else if cs.AfterLast != period {
			if iter.Next() {
				log.Error("Gap in the canonical chain database")
			}
			break // continuity guaranteed
		}
		cs.AfterLast = period + 1
	}
	iter.Release()
	return cs
}

func (cs *canonicalStore[T]) getDbKey(period uint64) []byte {
	var (
		kl  = len(cs.keyPrefix)
		key = make([]byte, kl+8)
	)
	copy(key[:kl], cs.keyPrefix)
	binary.BigEndian.PutUint64(key[kl:], period)
	return key
}

func (cs *canonicalStore[T]) add(batch ethdb.Batch, period uint64, value T) {
	if !cs.CanExpand(period) {
		log.Error("Cannot expand canonical store", "range.first", cs.First, "range.afterLast", cs.AfterLast, "new period", period)
		return
	}
	enc, err := cs.encode(value)
	if err != nil {
		log.Error("Error encoding canonical store value", "error", err)
		return
	}
	key := cs.getDbKey(period)
	if batch != nil {
		err = batch.Put(key, enc)
	} else {
		err = cs.db.Put(key, enc)
	}
	if err != nil {
		log.Error("Error writing into canonical store value database", "error", err)
	}
	cs.cache.Add(period, value)
	cs.Expand(period)
}

// should only be used in batch mode
func (cs *canonicalStore[T]) deleteFrom(batch ethdb.Batch, fromPeriod uint64) (deleted Range) {
	if fromPeriod >= cs.AfterLast {
		return
	}
	if fromPeriod < cs.First {
		fromPeriod = cs.First
	}
	deleted = Range{First: fromPeriod, AfterLast: cs.AfterLast}
	for period := fromPeriod; period < cs.AfterLast; period++ {
		batch.Delete(cs.getDbKey(period))
		cs.cache.Remove(period)
	}
	if fromPeriod > cs.First {
		cs.AfterLast = fromPeriod
	} else {
		cs.Range = Range{}
	}
	return
}

func (cs *canonicalStore[T]) get(period uint64) T {
	if value, ok := cs.cache.Get(period); ok {
		return value
	}
	var value T
	if enc, err := cs.db.Get(cs.getDbKey(period)); err == nil {
		if v, err := cs.decode(enc); err == nil {
			value = v
		} else {
			log.Error("Error decoding canonical store value", "error", err)
		}
	}
	return value
}
