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

package beacon

import (
	"bufio"
	"bytes"
	"context"
	"encoding/binary"

	"fmt"
	"io"
	"math"
	"math/bits"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/rawdb"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
	lru "github.com/hashicorp/golang-lru"
	"github.com/minio/sha256-simd"
	bls "github.com/protolambda/bls12-381-util"
)

var (
	initDataKey      = []byte("init") // RLP(lightClientInitData)
	bestUpdateKey    = []byte("bu-")  // bigEndian64(syncPeriod) -> RLP(LightClientUpdate)  (nextCommittee only referenced by root hash)
	syncCommitteeKey = []byte("sc-")  // bigEndian64(syncPeriod) + committee root hash -> serialized committee
)

const (
	maxUpdateInfoLength     = 128
	MaxCommitteeUpdateFetch = 128
	CommitteeCostFactor     = 16
	broadcastFrequencyLimit = time.Millisecond * 200
	advertiseDelay          = time.Second * 10
)

func (s *SignedHead) calculateScore() (int, uint64) {
	if len(s.BitMask) != 64 {
		return 0, 0 // signature check will filter it out later but we calculate score before sig check
	}
	var signerCount int
	for _, v := range s.BitMask {
		signerCount += bits.OnesCount8(v)
	}
	if signerCount <= 170 {
		return signerCount, 0
	}
	baseScore := uint64(s.Header.Slot) * 0x400
	if signerCount <= 256 {
		return signerCount, baseScore + uint64(signerCount)*24 - 0x1000 // 0..0x800; 0..2 slots offset
	}
	if signerCount <= 341 {
		return signerCount, baseScore + uint64(signerCount)*12 - 0x400 // 0x800..0xC00; 2..3 slots offset
	}
	return signerCount, baseScore + uint64(signerCount)*3 + 0x800 // 0..0xC00..0xE00; 3..3.5 slots offset
}

type SignedHead struct {
	BitMask   []byte
	Signature []byte
	Header    Header
}

func (s *SignedHead) Equal(s2 *SignedHead) bool {
	return s.Header == s2.Header && bytes.Equal(s.BitMask, s2.BitMask) && bytes.Equal(s.Signature, s2.Signature)
}

type syncCommittee struct {
	keys      [512]*bls.Pubkey
	aggregate *bls.Pubkey
}

type sctClient interface {
	SendSignedHeads(heads []SignedHead)
	SendUpdateInfo(updateInfo *UpdateInfo)
}

type sctServer interface {
	GetBestCommitteeProofs(ctx context.Context, req CommitteeRequest) (CommitteeReply, error)
	ClosedChannel() chan struct{}
	WrongReply(description string)
}

const lastProcessedCount = 4

type SyncCommitteeTracker struct {
	lock                                                                              sync.RWMutex
	db                                                                                ethdb.Database
	clock                                                                             mclock.Clock
	bestUpdateCache, serializedCommitteeCache, syncCommitteeCache, committeeRootCache *lru.Cache

	forks       Forks
	constraints SctConstraints
	initDone    bool

	firstPeriod uint64 // first bestUpdate stored in db (can be <= initData.Period)
	nextPeriod  uint64 // first bestUpdate not stored in db

	updateInfo                             *UpdateInfo
	connected                              map[sctServer]*sctPeerInfo
	requestQueue                           []*sctPeerInfo
	broadcastTo, advertisedTo              map[sctClient]struct{}
	lastBroadcast                          mclock.AbsTime
	advertiseScheduled, broadcastScheduled bool
	triggerCh, stopCh                      chan struct{}
	receivedList, processedList            headList
	lastProcessed                          [lastProcessedCount]common.Hash
	lastProcessedIndex                     int

	headSubs []func(Header)
}

func NewSyncCommitteeTracker(db ethdb.Database, forks Forks, constraints SctConstraints, clock mclock.Clock) *SyncCommitteeTracker {
	db = rawdb.NewTable(db, "sct-")
	s := &SyncCommitteeTracker{
		db:            db,
		clock:         clock,
		forks:         forks,
		constraints:   constraints,
		connected:     make(map[sctServer]*sctPeerInfo),
		broadcastTo:   make(map[sctClient]struct{}),
		triggerCh:     make(chan struct{}, 1),
		stopCh:        make(chan struct{}),
		receivedList:  newHeadList(4),
		processedList: newHeadList(2),
	}
	s.bestUpdateCache, _ = lru.New(1000)
	s.serializedCommitteeCache, _ = lru.New(100)
	s.syncCommitteeCache, _ = lru.New(100)
	s.committeeRootCache, _ = lru.New(100)

	/*var np [8]byte
	binary.BigEndian.PutUint64(np[:], s.nextPeriod)*/
	iter := s.db.NewIterator(bestUpdateKey, nil) //np[:])
	kl := len(bestUpdateKey)
	//fmt.Println("NewSyncCommitteeTracker")
	// iterate through them all for simplicity; at most a few hundred items
	first := true
	for iter.Next() {
		period := binary.BigEndian.Uint64(iter.Key()[kl : kl+8])
		//fmt.Println(" period", period)
		if first {
			s.firstPeriod = period
			first = false
		} else if s.nextPeriod != period {
			break // continuity guaranteed
		}
		s.nextPeriod = period + 1
	}
	iter.Release()
	constraints.SetCallbacks(s.Init, s.EnforceForksAndConstraints)
	return s
}

func (s *SyncCommitteeTracker) Stop() {
	close(s.stopCh)
}

/*func (s *SyncCommitteeTracker) clearDb() {
	//fmt.Println("clearDb")
	iter := s.db.NewIterator(nil, nil)
	for iter.Next() {
		s.db.Delete(iter.Key())
	}
	iter.Release()
	s.bestUpdateCache.Purge()
	s.serializedCommitteeCache.Purge()
	s.syncCommitteeCache.Purge()
	s.committeeRootCache.Purge()
	s.updateInfo = nil
}*/

func getBestUpdateKey(period uint64) []byte {
	kl := len(bestUpdateKey)
	key := make([]byte, kl+8)
	copy(key[:kl], bestUpdateKey)
	binary.BigEndian.PutUint64(key[kl:], period)
	return key
}

/*func (s *SyncCommitteeTracker) getNextPeriod() uint64 {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.nextPeriod
}*/

func (s *SyncCommitteeTracker) GetBestUpdate(period uint64) *LightClientUpdate {
	//fmt.Println("GetBestUpdate", period)
	if v, ok := s.bestUpdateCache.Get(period); ok {
		update, _ := v.(*LightClientUpdate)
		if update != nil {
			//fmt.Println(" cache", update.Header.Slot>>13)
			if uint64(update.Header.Slot>>13) != period {
				log.Error("Best update from wrong period found in cache")
			}
		}
		return update
	}
	if updateEnc, err := s.db.Get(getBestUpdateKey(period)); err == nil {
		update := new(LightClientUpdate)
		if err := rlp.DecodeBytes(updateEnc, update); err == nil {
			update.CalculateScore()
			s.bestUpdateCache.Add(period, update)
			//fmt.Println(" db", update.Header.Slot>>13)
			if uint64(update.Header.Slot>>13) != period {
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

func (s *SyncCommitteeTracker) storeBestUpdate(update *LightClientUpdate) {
	period := uint64(update.Header.Slot) >> 13 //TODO which header?
	//fmt.Println("storeBestUpdate", period, update)
	updateEnc, err := rlp.EncodeToBytes(update)
	if err != nil {
		log.Error("Error encoding LightClientUpdate", "error", err)
		return
	}
	s.bestUpdateCache.Add(period, update)
	s.db.Put(getBestUpdateKey(period), updateEnc)
	s.committeeRootCache.Remove(period + 1)
	s.updateInfoChanged()
}

func (s *SyncCommitteeTracker) deleteBestUpdate(period uint64) {
	s.db.Delete(getBestUpdateKey(period))
	s.bestUpdateCache.Remove(period)
	s.committeeRootCache.Remove(period + 1)
	s.updateInfoChanged()
}

const (
	sciSuccess = iota
	sciNeedCommittee
	sciWrongUpdate
	sciUnexpectedError
)

func (s *SyncCommitteeTracker) verifyUpdate(update *LightClientUpdate) bool {
	if update.Header.Slot&0x1fff == 0x1fff {
		// last slot of each period is not suitable for an update because it is signed by the next period's sync committee, proves the same committee it is signed by
		return false
	}
	if !s.checkForksAndConstraints(update) {
		return false
	}
	var checkRoot common.Hash
	if update.hasFinality() {
		if update.FinalizedHeader.Slot>>13 != update.Header.Slot>>13 {
			// finalized header is from the previous period, proves the same committee it is signed by
			return false
		}
		if root, ok := verifySingleProof(update.FinalityBranch, BsiFinalBlock, MerkleValue(update.FinalizedHeader.Hash()), 0); !ok || root != update.Header.StateRoot {
			//fmt.Println("wrong finality proof", ok, root, update.FinalizedHeader.StateRoot)
			return false
		}
		checkRoot = update.FinalizedHeader.StateRoot
	} else {
		checkRoot = update.Header.StateRoot
	}
	if root, ok := verifySingleProof(update.NextSyncCommitteeBranch, BsiNextSyncCommittee, MerkleValue(update.NextSyncCommitteeRoot), 0); !ok || root != checkRoot {
		//fmt.Println("wrong nsc proof", ok, root, checkRoot)
		return false
	}
	return s.verifySignature(SignedHead{Header: update.Header, Signature: update.SyncCommitteeSignature, BitMask: update.SyncCommitteeBits})
}

// returns true if nextCommittee is needed; call again
// verifies update before inserting
func (s *SyncCommitteeTracker) insertUpdate(update *LightClientUpdate, nextCommittee []byte) int {
	//fmt.Println("insertUpdate", update, nextCommittee != nil)
	period := uint64(update.Header.Slot) >> 13
	//fmt.Println(" before insert (first, next, update):", s.firstPeriod, s.nextPeriod, period)
	if !s.verifyUpdate(update) {
		//fmt.Println("wrong update")
		return sciWrongUpdate
	}

	if s.nextPeriod == 0 || period > s.nextPeriod || period+1 < s.firstPeriod || period < 1 {
		log.Error("Unexpected insertUpdate", "period", period, "firstPeriod", s.firstPeriod, "nextPeriod", s.nextPeriod)
		return sciUnexpectedError
	}
	update.CalculateScore()
	var rollback bool
	if period+1 == s.firstPeriod {
		if s.getSyncCommitteeLocked(period-1) == nil || update.NextSyncCommitteeRoot != s.getSyncCommitteeRoot(period+1) { //TODO kell itt ez a plusz check, vagy checkForksAndConstraints mindig eleg?
			return sciWrongUpdate
		}
	} else if period < s.nextPeriod {
		// update should already exist
		oldUpdate := s.GetBestUpdate(period)
		if oldUpdate == nil {
			log.Error("Update expected to exist but missing from db")
			return sciUnexpectedError
		}
		rollback = update.NextSyncCommitteeRoot != oldUpdate.NextSyncCommitteeRoot
		if !update.score.betterThan(oldUpdate.score) {
			// not better that existing one, nothing to do
			//fmt.Println(" not better than existing update, nothing changed")
			return sciSuccess
		}
	}

	//fmt.Println(" insertUpdate(period, nextPeriod, p+1 committee root, exists):", period, s.nextPeriod, update.NextSyncCommitteeRoot, s.GetSerializedSyncCommittee(period+1, update.NextSyncCommitteeRoot) != nil)
	if (period == s.nextPeriod || rollback) && s.GetSerializedSyncCommittee(period+1, update.NextSyncCommitteeRoot) == nil {
		//fmt.Println("111")
		// committee is not yet stored in db
		if nextCommittee == nil {
			//fmt.Println("need committee")
			return sciNeedCommittee
		}
		if SerializedCommitteeRoot(nextCommittee) != update.NextSyncCommitteeRoot {
			//fmt.Println("wrong committee root")
			return sciWrongUpdate
		}
		//fmt.Println("222")
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
	//fmt.Println(" after insert (first, next):", s.firstPeriod, s.nextPeriod)
	log.Info("Synced new committee update", "period", period, "nextCommitteeRoot", update.NextSyncCommitteeRoot)
	return sciSuccess
}

func getSyncCommitteeKey(period uint64, committeeRoot common.Hash) []byte {
	kl := len(syncCommitteeKey)
	key := make([]byte, kl+8+32)
	copy(key[:kl], syncCommitteeKey)
	binary.BigEndian.PutUint64(key[kl:kl+8], period)
	copy(key[kl+8:], committeeRoot[:])
	return key
}

func (s *SyncCommitteeTracker) GetSerializedSyncCommittee(period uint64, committeeRoot common.Hash) []byte {
	//fmt.Println("GetSerializedSyncCommittee", period, committeeRoot)
	key := getSyncCommitteeKey(period, committeeRoot)
	if v, ok := s.serializedCommitteeCache.Get(string(key)); ok {
		committee, _ := v.([]byte)
		if len(committee) == 513*48 {
			//fmt.Println(" in cache")
			return committee
		} else {
			log.Error("Serialized committee with invalid size found in cache")
		}
	}
	if committee, err := s.db.Get(key); err == nil {
		if len(committee) == 513*48 {
			//fmt.Println(" in db", len(committee))
			s.serializedCommitteeCache.Add(string(key), committee)
			return committee
		} else {
			log.Error("Serialized committee with invalid size found in database")
		}
	}
	return nil
}

func (s *SyncCommitteeTracker) storeSerializedSyncCommittee(period uint64, committeeRoot common.Hash, committee []byte) {
	//fmt.Println("storeSerializedSyncCommittee", period, committeeRoot)
	key := getSyncCommitteeKey(period, committeeRoot)
	s.serializedCommitteeCache.Add(string(key), committee)
	s.syncCommitteeCache.Remove(string(key)) // a nil entry for "not found" might have been stored here earlier
	s.db.Put(key, committee)
}

// caller should ensure that previous advertised committees are synced before checking signature
func (s *SyncCommitteeTracker) verifySignature(head SignedHead) bool {
	if len(head.Signature) != 96 || len(head.BitMask) != 64 {
		fmt.Println("wrong sig size")
		return false
	}
	var signature bls.Signature
	var sigBytes [96]byte
	copy(sigBytes[:], head.Signature)
	if err := signature.Deserialize(&sigBytes); err != nil {
		fmt.Println("sig deserialize error", err)
		return false
	}

	committee := s.getSyncCommitteeLocked(uint64(head.Header.Slot+1) >> 13) // signed with the next slot's committee
	if committee == nil {
		fmt.Println("sig check: committee not found", uint64(head.Header.Slot+1)>>13)
		return false
	}
	var signerKeys [512]*bls.Pubkey
	signerCount := 0
	for i, key := range committee.keys {
		if head.BitMask[i/8]&(byte(1)<<(i%8)) != 0 {
			signerKeys[signerCount] = key
			signerCount++
		}
	}

	var signingRoot common.Hash
	hasher := sha256.New()
	headerHash := head.Header.Hash()
	hasher.Write(headerHash[:])
	domain := s.forks.domain(uint64(head.Header.Slot) >> 5)
	fmt.Println("sig check domain", domain)
	hasher.Write(domain[:])
	hasher.Sum(signingRoot[:0])
	fmt.Println("signingRoot", signingRoot, "signerCount", signerCount)

	vvv := bls.FastAggregateVerify(signerKeys[:signerCount], signingRoot[:], &signature)
	fmt.Println("sig bls check", vvv)
	return vvv
}

func computeDomain(forkVersion []byte, genesisValidatorsRoot common.Hash) MerkleValue {
	var forkVersion32, forkDataRoot, domain MerkleValue
	hasher := sha256.New()
	copy(forkVersion32[:len(forkVersion)], forkVersion)
	hasher.Write(forkVersion32[:])
	hasher.Write(genesisValidatorsRoot[:])
	hasher.Sum(forkDataRoot[:0])
	domain[0] = 7
	copy(domain[4:], forkDataRoot[:28])
	//fmt.Println("computeDomain", forkVersion, genesisValidatorsRoot, domain)
	return domain
}

func SerializedCommitteeRoot(enc []byte) common.Hash {
	if len(enc) != 513*48 {
		return common.Hash{}
	}
	hasher := sha256.New()
	var (
		padding [16]byte
		data    [512]common.Hash
	)
	for i := range data {
		hasher.Reset()
		hasher.Write(enc[i*48 : (i+1)*48])
		hasher.Write(padding[:])
		hasher.Sum(data[i][:0])
	}
	l := 512
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
	hasher.Write(enc[512*48 : 513*48])
	hasher.Write(padding[:])
	hasher.Sum(data[1][:0])
	hasher.Reset()
	hasher.Write(data[0][:])
	hasher.Write(data[1][:])
	hasher.Sum(data[0][:0])
	return data[0]
}

func deserializeSyncCommittee(enc []byte) *syncCommittee {
	if len(enc) != 513*48 {
		log.Error("Wrong input size for deserializeSyncCommittee", "expected", 513*48, "got", len(enc))
		return nil
	}
	sc := new(syncCommittee)
	for i := 0; i <= 512; i++ {
		pk := new(bls.Pubkey)
		var sk [48]byte
		copy(sk[:], enc[i*48:(i+1)*48])
		if err := pk.Deserialize(&sk); err != nil {
			log.Error("bls.Pubkey.Deserialize failed", "error", err, "data", sk)
			return nil
		}
		if i < 512 {
			sc.keys[i] = pk
		} else {
			sc.aggregate = pk
		}
	}
	return sc
}

func (s *SyncCommitteeTracker) getSyncCommitteeRoot(period uint64) (root common.Hash) {
	//fmt.Println("getSyncCommitteeRoot", period)
	if v, ok := s.committeeRootCache.Get(period); ok {
		//fmt.Println(" cached", v.(common.Hash))
		return v.(common.Hash)
	}
	defer func() {
		s.committeeRootCache.Add(period, root)
	}()

	/*if s.nextPeriod > 0 && period > 0 {
		if update := s.GetBestUpdate(period - 1); update != nil {
			//fmt.Println(" update", period-1, update)
			return update.NextSyncCommitteeRoot
		}
	}*/

	if r, matchAll := s.constraints.CommitteeRoot(period); !matchAll {
		//fmt.Println(" from constraints", r)
		return r
	}
	if s.nextPeriod == 0 || period == 0 {
		//fmt.Println(" not found", s.nextPeriod, period)
		return common.Hash{}
	}
	if update := s.GetBestUpdate(period - 1); update != nil {
		//fmt.Println(" found in update", period-1, update)
		return update.NextSyncCommitteeRoot
	}
	//fmt.Println(" no update, root not found")
	return common.Hash{}
}

func (s *SyncCommitteeTracker) GetSyncCommitteeRoot(period uint64) common.Hash {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.getSyncCommitteeRoot(period)
}

func (s *SyncCommitteeTracker) getSyncCommitteeLocked(period uint64) *syncCommittee {
	////fmt.Println("sct.getSyncCommittee", period)
	if committeeRoot := s.getSyncCommitteeRoot(period); committeeRoot != (common.Hash{}) {
		key := string(getSyncCommitteeKey(period, committeeRoot))
		if v, ok := s.syncCommitteeCache.Get(key); ok {
			////fmt.Println(" cached")
			sc, _ := v.(*syncCommittee)
			return sc
		}
		if sc := s.GetSerializedSyncCommittee(period, committeeRoot); sc != nil {
			c := deserializeSyncCommittee(sc)
			s.syncCommitteeCache.Add(key, c)
			return c
		} else {
			////fmt.Println(" no committee")
			log.Error("Missing serialized sync committee", "period", period, "committeeRoot", committeeRoot)
		}
	}
	return nil
}

func (s *SyncCommitteeTracker) getSyncCommittee(period uint64) *syncCommittee {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.getSyncCommitteeLocked(period)
}

var minimumScore = UpdateScore{
	signerCount:    200, //257,		//TODO server resync utan is ilyen vacakok a score-ok? lodestar a hibas?
	subPeriodIndex: 10,  //0x1000,
}

type UpdateScore struct {
	signerCount, subPeriodIndex uint32
	finalized                   bool
}

type UpdateScores []UpdateScore //TODO RLP encode to single []byte

func (u *UpdateScore) penalty() uint64 {
	if u.signerCount == 0 {
		return math.MaxUint64
	}
	a := uint64(512 - u.signerCount)
	b := a * a
	c := b * b
	p := a * b * c // signerCount ^ 7 (between 0 and 2^63)
	if !u.finalized {
		//		p += 0x2000000000000000 / uint64(u.subPeriodIndex+1)
		p += 0x1000000000000000 / uint64(u.subPeriodIndex+1)
	}
	return p
}

func (u UpdateScore) betterThan(w UpdateScore) bool {
	return u.penalty() < w.penalty()
}

func (u *UpdateScore) Encode(data []byte) {
	v := u.signerCount + u.subPeriodIndex<<10
	if u.finalized {
		v += 0x800000
	}
	var enc [4]byte
	binary.LittleEndian.PutUint32(enc[:], v)
	copy(data, enc[:3])
}

func (u *UpdateScore) Decode(data []byte) {
	var enc [4]byte
	copy(enc[:3], data)
	v := binary.LittleEndian.Uint32(enc[:])
	u.signerCount = v & 0x3ff
	u.subPeriodIndex = (v >> 10) & 0x1fff
	u.finalized = (v & 0x800000) != 0
}

type UpdateInfo struct {
	AfterLastPeriod uint64
	Scores          []byte // encoded; 3 bytes per period		//TODO use decoded version in memory
	//NextCommitteeRoot common.Hash	//TODO not needed?
}

// 0 if not initialized
func (s *SyncCommitteeTracker) NextPeriod() uint64 {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.nextPeriod
}

func (s *SyncCommitteeTracker) GetUpdateInfo() *UpdateInfo {
	s.lock.Lock()
	defer s.lock.Unlock()

	return s.getUpdateInfo()
}

func (s *SyncCommitteeTracker) getUpdateInfo() *UpdateInfo {
	if s.updateInfo != nil {
		return s.updateInfo
	}
	l := s.nextPeriod - s.firstPeriod
	if l > maxUpdateInfoLength {
		l = maxUpdateInfoLength
	}
	firstPeriod := s.nextPeriod - l

	u := &UpdateInfo{
		AfterLastPeriod: s.nextPeriod,
		Scores:          make([]byte, int(l)*3),
		//NextCommitteeRoot: s.getSyncCommitteeRoot(s.nextPeriod),
	}

	for period := firstPeriod; period < s.nextPeriod; period++ {
		if update := s.GetBestUpdate(uint64(period)); update != nil {
			scoreIndex := int(period-firstPeriod) * 3
			update.score.Encode(u.Scores[scoreIndex : scoreIndex+3])
		} else {
			log.Error("Update missing from database", "period", period)
		}
	}

	s.updateInfo = u
	return u
}

type CommitteeRequest struct {
	UpdatePeriods, CommitteePeriods []uint64
}

func (req CommitteeRequest) empty() bool {
	return req.UpdatePeriods == nil && req.CommitteePeriods == nil
}

type CommitteeReply struct {
	Updates    []LightClientUpdate
	Committees [][]byte
}

type sctPeerInfo struct {
	peer               sctServer
	remoteInfo         UpdateInfo
	requesting, queued bool
	forkPeriod         uint64
	deferredHeads      []SignedHead
}

func (s *SyncCommitteeTracker) startRequest(sp *sctPeerInfo) bool {
	req := s.nextRequest(sp)
	if req.empty() {
		//fmt.Println("startRequest: no more requests")
		if sp.deferredHeads != nil {
			s.addSignedHeads(sp.peer, sp.deferredHeads)
			sp.deferredHeads = nil
		}
		return false
	}
	sp.requesting = true
	//fmt.Println("startRequest: requesting")
	go func() {
		ctx, _ := context.WithTimeout(context.Background(), time.Second*20)
		reply, err := sp.peer.GetBestCommitteeProofs(ctx, req) // expected to return with error in case of shutdown
		if err != nil {
			//fmt.Println("startRequest: request error", err)
			s.lock.Lock()
			sp.requesting = false
			s.lock.Unlock()
			return
		}
		s.lock.Lock()
		sp.requesting = false
		if s.processReply(sp, req, reply) {
			s.requestQueue = append(s.requestQueue, sp)
			sp.queued = true
			select {
			case s.triggerCh <- struct{}{}:
			default:
			}
		} else {
			sp.peer.WrongReply("Invalid committee proof")
		}
		s.lock.Unlock()
	}()
	return true
}

func (s *SyncCommitteeTracker) syncLoop() {
	s.lock.Lock()
	for {
		if len(s.requestQueue) > 0 {
			sp := s.requestQueue[0]
			s.requestQueue = s.requestQueue[1:]
			if len(s.requestQueue) == 0 {
				s.requestQueue = nil
			}
			sp.queued = false
			if s.startRequest(sp) {
				s.lock.Unlock()
				select {
				case <-s.triggerCh:
				case <-s.clock.After(time.Second):
				case <-s.stopCh:
					return
				}
				s.lock.Lock()
			}
		} else {
			s.lock.Unlock()
			select {
			case <-s.triggerCh:
			case <-s.stopCh:
				return
			}
			s.lock.Lock()
		}
	}
}

func (s *SyncCommitteeTracker) SyncWithPeer(peer sctServer, remoteInfo *UpdateInfo) {
	//fmt.Println("SyncWithPeer", remoteInfo)

	s.lock.Lock()
	sp := s.connected[peer]
	if sp == nil {
		sp = &sctPeerInfo{peer: peer}
		s.connected[peer] = sp
	}
	if remoteInfo != nil {
		sp.remoteInfo = *remoteInfo
		sp.forkPeriod = remoteInfo.AfterLastPeriod
		if !sp.queued && !sp.requesting {
			s.requestQueue = append(s.requestQueue, sp)
			sp.queued = true
			select {
			case s.triggerCh <- struct{}{}:
			default:
			}
		}
	}
	s.lock.Unlock()
}

func (s *SyncCommitteeTracker) Disconnect(peer sctServer) {
	s.lock.Lock()
	delete(s.connected, peer)
	s.lock.Unlock()
}

func (s *SyncCommitteeTracker) retrySyncAllPeers() {
	for _, sp := range s.connected {
		if !sp.queued && !sp.requesting {
			s.requestQueue = append(s.requestQueue, sp)
			sp.queued = true
		}
	}
	select {
	case s.triggerCh <- struct{}{}:
	default:
	}
}

func (s *SyncCommitteeTracker) nextRequest(sp *sctPeerInfo) CommitteeRequest {
	if sp.remoteInfo.AfterLastPeriod <= uint64(len(sp.remoteInfo.Scores)/3) {
		return CommitteeRequest{}
	}
	localInfo := s.getUpdateInfo()
	//fmt.Println("nextRequest")
	//fmt.Println(" localInfo", localInfo)
	//fmt.Println(" remoteInfo", sp.remoteInfo)
	localFirstPeriod := localInfo.AfterLastPeriod - uint64(len(localInfo.Scores)/3)
	remoteFirstPeriod := sp.remoteInfo.AfterLastPeriod - uint64(len(sp.remoteInfo.Scores)/3)

	constraintsFirst, constraintsAfterFixed, constraintsAfterLast := s.constraints.PeriodRange()
	//fmt.Println(" local range:", s.firstPeriod, s.nextPeriod)
	//fmt.Println(" remote range:", remoteFirstPeriod, sp.remoteInfo.AfterLastPeriod)
	//fmt.Println(" committee constraint range:", constraintsFirst, constraintsAfterFixed, constraintsAfterLast)

	if constraintsAfterLast < constraintsFirst+2 {
		return CommitteeRequest{}
	}
	if s.nextPeriod != 0 && (constraintsAfterFixed <= s.firstPeriod || constraintsFirst >= s.nextPeriod) {
		log.Error("Gap between local updates and fixed committee constraints, cannot sync", "local first", s.firstPeriod, "local afterLast", s.nextPeriod, "constraints first", constraintsFirst, "constraints afterFixed", constraintsAfterFixed)
		return CommitteeRequest{}
	}

	// committee period range -> update period range
	syncFirst, syncAfterFixed, syncAfterLast := constraintsFirst+1, constraintsAfterFixed-1, constraintsAfterLast-1
	// find intersection with remote advertised update range
	/*if remoteFirstPeriod > syncFirst {
		syncFirst = remoteFirstPeriod
	}*/
	if sp.remoteInfo.AfterLastPeriod < syncAfterFixed {
		syncAfterFixed = sp.remoteInfo.AfterLastPeriod
	}
	if sp.remoteInfo.AfterLastPeriod < syncAfterLast {
		syncAfterLast = sp.remoteInfo.AfterLastPeriod
	}
	if syncAfterLast < syncFirst {
		return CommitteeRequest{}
	}
	//fmt.Println(" sync range:", syncFirst, syncAfterFixed, syncAfterLast)

	var (
		request  CommitteeRequest
		reqCount int
	)

	for period := syncAfterFixed - 1; period <= syncAfterFixed; period++ {
		if s.nextPeriod == 0 || period+1 < s.firstPeriod || period > s.nextPeriod {
			request.CommitteePeriods = append(request.CommitteePeriods, period)
			reqCount += CommitteeCostFactor
		}
	}
	addUpdate := func(updatePeriod, committeePeriod uint64) bool {
		if reqCount >= MaxCommitteeUpdateFetch {
			return false
		}
		// decode remote score
		var remoteScore UpdateScore
		if updatePeriod >= remoteFirstPeriod {
			remotePtr := int(updatePeriod-remoteFirstPeriod) * 3
			remoteScore.Decode(sp.remoteInfo.Scores[remotePtr : remotePtr+3])
		} else {
			// if earlier than advertised range then assume minimum score; fail later if delivered score is too low
			remoteScore = minimumScore
		}

		if s.nextPeriod == 0 || updatePeriod < s.firstPeriod || updatePeriod >= s.nextPeriod {
			// outside already synced range, request update+committee if possible (try even if it is before the remote advertised range)
			if reqCount+CommitteeCostFactor+1 > MaxCommitteeUpdateFetch {
				return false
			}
			if minimumScore.betterThan(remoteScore) {
				// do not sync further if advertised score is less than our minimum requirement
				return false
			}
			request.UpdatePeriods = append(request.UpdatePeriods, updatePeriod)
			request.CommitteePeriods = append(request.CommitteePeriods, committeePeriod)
			reqCount += CommitteeCostFactor + 1
		} else {
			// inside already synced range, request update only if advertised remote score is better than local
			if updatePeriod < localFirstPeriod || updatePeriod < remoteFirstPeriod {
				// do not update already synced range before the remote advertised range
				return true
			}
			var localScore UpdateScore
			localPtr := int(updatePeriod-localFirstPeriod) * 3
			localScore.Decode(localInfo.Scores[localPtr : localPtr+3])
			//fmt.Println(" known period", updatePeriod, "remoteScore", remoteScore, "localScore", localScore, remoteScore.betterThan(localScore))
			if remoteScore.betterThan(localScore) {
				request.UpdatePeriods = append(request.UpdatePeriods, updatePeriod)
				reqCount++
			}
		}
		return true

	}
	origin := syncAfterFixed
	if s.nextPeriod != 0 && origin > s.nextPeriod {
		origin = s.nextPeriod
	}
	if syncFirst > origin {
		return CommitteeRequest{}
	}
	for period := origin; period < syncAfterLast; period++ {
		if !addUpdate(period, period+1) {
			break
		}
	}
	for period := origin - 1; period >= syncFirst; period-- {
		if !addUpdate(period, period-1) {
			break
		}
	}

	//fmt.Println(" request", request)
	return request
}

/*func (s *SyncCommitteeTracker) canInsertUpdate(period uint64) bool {
	return (period == 0 || s.getSyncCommitteeLocked(period-1) != nil) && s.getSyncCommitteeLocked(period) != nil && s.getSyncCommitteeLocked(period+1) != nil
}*/

func (s *SyncCommitteeTracker) processReply(sp *sctPeerInfo, sentRequest CommitteeRequest, reply CommitteeReply) bool {
	//fmt.Println("Processing committee reply", len(reply.Updates), len(reply.Committees))
	//for i, u := range reply.Updates {
	//fmt.Println(" update", sentRequest.UpdatePeriods[i], u.Header.Slot>>13, u.NextSyncCommitteeRoot)
	//}
	/*	for i, c := range reply.Committees {
		//fmt.Println(" committees", sentRequest.CommitteePeriods[i], SerializedCommitteeRoot(c))
	}*/
	if len(reply.Updates) != len(sentRequest.UpdatePeriods) || len(reply.Committees) != len(sentRequest.CommitteePeriods) {
		//fmt.Println(" length mismatch", len(sentRequest.UpdatePeriods), len(sentRequest.CommitteePeriods))
		return false
	}

	futureCommittees := make(map[uint64][]byte)
	var lastStoredCommittee uint64
	for i, c := range reply.Committees {
		if len(c) != 513*48 {
			//fmt.Println(" wrong committee size")
			return false
		}
		period := sentRequest.CommitteePeriods[i]
		//fmt.Println(" committees", period, s.getSyncCommitteeRoot(period), SerializedCommitteeRoot(c))
		if root := s.getSyncCommitteeRoot(period); root != (common.Hash{}) {
			if SerializedCommitteeRoot(c) == root {
				s.storeSerializedSyncCommittee(period, root, c)
				if period > lastStoredCommittee {
					lastStoredCommittee = period
				}
			} else {
				//fmt.Println(" committee root mismatch")
				return false
			}
		} else {
			//fmt.Println("  future committee", period)
			futureCommittees[period] = c
		}
	}

	//fmt.Println(" committees processed; nextPeriod ==", s.nextPeriod)
	if s.nextPeriod == 0 {
		//fmt.Println(" init chain", lastStoredCommittee)
		// chain not initialized
		if lastStoredCommittee > 0 && s.getSyncCommitteeLocked(lastStoredCommittee-1) != nil {
			s.firstPeriod, s.nextPeriod = lastStoredCommittee, lastStoredCommittee
			s.updateInfoChanged()
		} else {
			//fmt.Println("  can not init")
			return false
		}
	}

	/*	for len(sentRequest.CommitteePeriods) > committeeIndex && (len(sentRequest.UpdatePeriods) == 0 || sentRequest.CommitteePeriods[committeeIndex] <= sentRequest.UpdatePeriods[0]) {
		period, committee := sentRequest.CommitteePeriods[committeeIndex], reply.Committees[committeeIndex]
		committeeRoot := s.getSyncCommitteeRoot(period)
		if SerializedCommitteeRoot(committee) != committeeRoot {
			//fmt.Println(" wrong committee root")
			return false
		}
		//fmt.Println(" storing init committee for period", period)
		s.storeSerializedSyncCommittee(period, committeeRoot, committee)
		committeeIndex++
	}*/
	firstPeriod := sp.remoteInfo.AfterLastPeriod - uint64(len(sp.remoteInfo.Scores)/3)
	for i, update := range reply.Updates {
		update := update // updates are cached by reference, do not overwrite
		period := uint64(update.Header.Slot) >> 13
		/*for !s.canInsertUpdate(period) {

		}*/
		if period != sentRequest.UpdatePeriods[i] {
			//fmt.Println(" wrong update period")
			return false
		}
		if period > s.nextPeriod { // a previous insertUpdate could have reduced nextPeriod since the request was created
			continue // skip but do not fail because it is not the remote side's fault; retry with new request
		}

		var remoteInfoScore UpdateScore
		if period >= firstPeriod {
			remoteInfoIndex := int(period - firstPeriod)
			remoteInfoScore.Decode(sp.remoteInfo.Scores[remoteInfoIndex*3 : remoteInfoIndex*3+3])
		} else {
			remoteInfoScore = minimumScore
		}
		update.CalculateScore()
		if remoteInfoScore.betterThan(update.score) {
			//fmt.Println(" update has lower score than promised")
			return false // remote did not deliver an update with the promised score
		}

		switch s.insertUpdate(&update, futureCommittees[period+1]) {
		case sciSuccess:
			if sp.forkPeriod == period {
				// if local chain is successfully updated to the remote fork then remote is not on a different fork anymore
				sp.forkPeriod = sp.remoteInfo.AfterLastPeriod
			}
		case sciWrongUpdate:
			//fmt.Println(" insert: wrong update")
			return false
		case sciNeedCommittee:
			// remember that remote is on a different and more valuable fork; do not fail but construct next request accordingly
			sp.forkPeriod = period
			continue
		case sciUnexpectedError:
			// local error, insertUpdate has already printed an error log
			//fmt.Println(" insert: unexpected error")
			return false // though not the remote's fault, fail here to avoid infinite retries
		}
	}
	return true
}

type LightClientUpdate struct {
	Header                  Header
	NextSyncCommitteeRoot   common.Hash
	NextSyncCommitteeBranch MerkleValues
	FinalizedHeader         Header
	FinalityBranch          MerkleValues
	SyncCommitteeBits       []byte
	SyncCommitteeSignature  []byte
	ForkVersion             []byte
	score                   UpdateScore
}

func (l *LightClientUpdate) hasFinality() bool {
	return l.FinalizedHeader.BodyRoot != (common.Hash{})
}

func (l *LightClientUpdate) CalculateScore() *UpdateScore {
	l.score.signerCount = 0
	for _, v := range l.SyncCommitteeBits {
		l.score.signerCount += uint32(bits.OnesCount8(v))
	}
	l.score.subPeriodIndex = uint32(l.Header.Slot & 0x1fff)
	l.score.finalized = l.hasFinality()
	return &l.score
}

func trimZeroes(data []byte) []byte {
	l := len(data)
	for l > 0 && data[l-1] == 0 {
		l--
	}
	return data[:l]
}

func (l *LightClientUpdate) checkForkVersion(forks Forks) bool {
	return bytes.Equal(trimZeroes(l.ForkVersion), trimZeroes(forks.version(uint64(l.Header.Slot>>5))))
}

type headInfo struct {
	head         SignedHead
	hash         common.Hash
	signerCount  int
	score        uint64
	receivedFrom map[sctServer]struct{}
	sentTo       map[sctClient]struct{}
	processed    bool
}

type headList struct {
	list []*headInfo // highest score first
	//hashMap map[common.Hash]*headInfo
	limit int
	//updateCallback func(head *headInfo, position int)
}

func newHeadList(limit int) headList {
	return headList{ /*hashMap: make(map[common.Hash]*headInfo), */ limit: limit}
}

func (h *headList) getHead(hash common.Hash) *headInfo {
	//return h.hashMap[hash]
	for _, headInfo := range h.list {
		if headInfo.hash == hash {
			return headInfo
		}
	}
	return nil
}

func (h *headList) updateHead(head *headInfo) {
	pos := len(h.list)
	for i, hh := range h.list {
		if hh == head {
			pos = i
			break
		}
	}
	for pos > 0 && head.score > h.list[pos-1].score {
		if pos < len(h.list) {
			h.list[pos] = h.list[pos-1]
		}
		pos--
	}
	if pos < len(h.list) {
		h.list[pos] = head
	} else if len(h.list) < h.limit {
		h.list = append(h.list, head)
	}
}

func (s *SyncCommitteeTracker) AddSignedHeads(peer sctServer, heads []SignedHead) {
	s.lock.Lock()
	defer s.lock.Unlock()

	if sp := s.connected[peer]; sp != nil && (sp.requesting || sp.queued) {
		fmt.Println("deferred heads", len(heads))
		sp.deferredHeads = append(sp.deferredHeads, heads...)
		return
	}
	s.addSignedHeads(peer, heads)
}

func (s *SyncCommitteeTracker) addSignedHeads(peer sctServer, heads []SignedHead) {
	var (
		broadcast   bool
		oldHeadHash common.Hash
	)
	if len(s.receivedList.list) > 0 {
		oldHeadHash = s.receivedList.list[0].hash
	}
	for _, head := range heads {
		fmt.Println("*** addSignedHead", head.Header.Slot, head.Header.Hash())
		if !s.verifySignature(head) {
			peer.WrongReply("invalid header signature")
			continue
		}
		signerCount, score := head.calculateScore()
		hash := head.Header.Hash()
		if h := s.receivedList.getHead(hash); h != nil {
			fmt.Println(" already in receivedList   processed ==", h.processed, "   new score ==", score, "   old score ==", h.score)
			h.receivedFrom[peer] = struct{}{}
			if score > h.score {
				h.head = head
				h.score = score
				h.signerCount = signerCount
				h.sentTo = nil
				s.receivedList.updateHead(h)
				if h.processed {
					s.processedList.updateHead(h)
					broadcast = true
				}
			}
		} else {
			var processed bool
			for _, p := range s.lastProcessed {
				if p == hash {
					processed = true
					break
				}
			}
			h := &headInfo{
				head:         head,
				hash:         hash,
				score:        score,
				sentTo:       make(map[sctClient]struct{}),
				receivedFrom: map[sctServer]struct{}{peer: struct{}{}},
				processed:    processed,
			}
			fmt.Println(" added to in receivedList   processed ==", h.processed, "   hash ==", hash, "   lastProcessed ==", s.lastProcessed)
			s.receivedList.updateHead(h)
			if h.processed {
				s.processedList.updateHead(h)
				broadcast = true
			}
		}
		fmt.Println(" broadcast ==", broadcast)
	}
	if broadcast {
		fmt.Println(" broadcast")
		s.broadcastHeads()
	}
	if len(s.receivedList.list) > 0 && oldHeadHash != s.receivedList.list[0].hash {
		head := s.receivedList.list[0].head.Header
		for _, subFn := range s.headSubs {
			subFn(head)
		}
	}
}

func (s *SyncCommitteeTracker) SubscribeToNewHeads(subFn func(Header)) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.headSubs = append(s.headSubs, subFn)
}

func (s *SyncCommitteeTracker) ProcessedBeaconHead(hash common.Hash) {
	s.lock.Lock()
	defer s.lock.Unlock()

	fmt.Println("ProcessedBeaconHead", hash)
	if headInfo := s.receivedList.getHead(hash); headInfo != nil {

		fmt.Println(" processed head in received list, broadcasting")
		headInfo.processed = true
		s.processedList.updateHead(headInfo)
		fmt.Println(" processedList", s.processedList)
		s.broadcastHeads()
	} else {
		fmt.Println(" not found in received list")
	}
	s.lastProcessed[s.lastProcessedIndex] = hash
	s.lastProcessedIndex++
	if s.lastProcessedIndex == lastProcessedCount {
		s.lastProcessedIndex = 0
	}
}

func (s *SyncCommitteeTracker) Activate(peer sctClient) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.broadcastTo[peer] = struct{}{}
	fmt.Println("sct.Activate   len(broadcastTo) =", len(s.broadcastTo))
	s.broadcastHeadsTo(peer, true)
}

func (s *SyncCommitteeTracker) Deactivate(peer sctClient, remove bool) {
	s.lock.Lock()
	defer s.lock.Unlock()

	delete(s.broadcastTo, peer)
	fmt.Println("sct.Deactivate   len(broadcastTo) =", len(s.broadcastTo))
	if remove {
		for _, headInfo := range s.processedList.list {
			delete(headInfo.sentTo, peer)
		}
	}
}

// frequency limited
func (s *SyncCommitteeTracker) broadcastHeads() {
	now := s.clock.Now()
	sinceLast := time.Duration(now - s.lastBroadcast)
	if sinceLast < broadcastFrequencyLimit {
		if !s.broadcastScheduled {
			s.broadcastScheduled = true
			s.clock.AfterFunc(broadcastFrequencyLimit-sinceLast, func() {
				s.lock.Lock()
				s.broadcastHeadsNow()
				s.broadcastScheduled = false
				s.lock.Unlock()
			})
		}
		return
	}
	s.broadcastHeadsNow()
	s.lastBroadcast = now
}

func (s *SyncCommitteeTracker) broadcastHeadsNow() {
	fmt.Println(" broadcastHeadsNow")
	for peer := range s.broadcastTo {
		s.broadcastHeadsTo(peer, false)
	}
}

// broadcast to all if peer == nil
func (s *SyncCommitteeTracker) broadcastHeadsTo(peer sctClient, sendEmpty bool) {
	fmt.Println(" broadcastHeadsTo")
	heads := make([]SignedHead, 0, len(s.processedList.list))
	for _, headInfo := range s.processedList.list {
		if _, ok := headInfo.sentTo[peer]; !ok {
			heads = append(heads, headInfo.head)
			headInfo.sentTo[peer] = struct{}{}
		}
		if headInfo.signerCount > 341 {
			break
		}
	}
	if sendEmpty || len(heads) > 0 {
		fmt.Println("  send", len(heads))
		peer.SendSignedHeads(heads)
	}
}

func (s *SyncCommitteeTracker) updateInfoChanged() {
	s.updateInfo = nil
	if s.advertiseScheduled {
		return
	}
	s.advertiseScheduled = true
	s.advertisedTo = nil

	s.clock.AfterFunc(advertiseDelay, func() {
		s.lock.Lock()
		s.advertiseCommitteesNow()
		s.advertiseScheduled = false
		s.lock.Unlock()
	})
}

func (s *SyncCommitteeTracker) advertiseCommitteesNow() {
	info := s.getUpdateInfo()
	for peer := range s.broadcastTo {
		if _, ok := s.advertisedTo[peer]; !ok {
			peer.SendUpdateInfo(info)
		}
	}
}

func (s *SyncCommitteeTracker) advertiseCommitteesTo(peer sctClient) {
	peer.SendUpdateInfo(s.getUpdateInfo())
	if s.advertisedTo == nil {
		s.advertisedTo = make(map[sctClient]struct{})
	}
	s.advertisedTo[peer] = struct{}{}
}

func (s *SyncCommitteeTracker) checkForksAndConstraints(update *LightClientUpdate) bool {
	if !s.initDone {
		log.Error("SyncCommitteeTracker not initialized")
		return false
	}
	if !update.checkForkVersion(s.forks) {
		return false
	}
	root, matchAll := s.constraints.CommitteeRoot(uint64(update.Header.Slot)>>13 + 1)
	return matchAll || root == update.NextSyncCommitteeRoot
}

func (s *SyncCommitteeTracker) EnforceForksAndConstraints() {
	s.lock.Lock()
	s.enforceForksAndConstraints()
	s.lock.Unlock()
}

func (s *SyncCommitteeTracker) enforceForksAndConstraints() {
	//fmt.Println("sct.EnforceForksAndConstraints", s.initDone)
	if !s.initDone {
		return
	}
	s.committeeRootCache.Purge()
	for s.nextPeriod > s.firstPeriod {
		if update := s.GetBestUpdate(s.nextPeriod - 1); update == nil || s.checkForksAndConstraints(update) {
			if update == nil {
				log.Error("Sync committee update missing", "period", s.nextPeriod-1)
			}
			break
		}
		s.nextPeriod--
		//fmt.Println(" roll back head", s.nextPeriod)
		s.deleteBestUpdate(s.nextPeriod)
	}
	for s.nextPeriod > s.firstPeriod {
		if s.firstPeriod > 0 && s.getSyncCommitteeLocked(s.firstPeriod-1) != nil {
			break
		}
		//fmt.Println(" delete tail", s.firstPeriod)
		s.deleteBestUpdate(s.firstPeriod)
		s.firstPeriod++
	}
	if s.nextPeriod == s.firstPeriod && s.nextPeriod > 0 && (s.getSyncCommitteeLocked(s.nextPeriod-1) == nil || s.getSyncCommitteeLocked(s.nextPeriod) == nil) {
		//fmt.Println(" reset chain")
		s.nextPeriod, s.firstPeriod = 0, 0
	}

	s.retrySyncAllPeers()
	//fmt.Println(" finished sct.EnforceForksAndConstraints")
}

func (s *SyncCommitteeTracker) Init(genesisData genesisData) {
	//fmt.Println("sct.Init")
	s.lock.Lock()
	s.forks.computeDomains(genesisData.GenesisValidatorsRoot)
	s.initDone = true
	s.enforceForksAndConstraints()
	s.lock.Unlock()

	go s.syncLoop()
}

type Fork struct {
	Epoch   uint64
	Name    string
	Version []byte
	domain  MerkleValue
}

type Forks []Fork

func (bf Forks) version(epoch uint64) []byte {
	for i := len(bf) - 1; i >= 0; i-- {
		if epoch >= bf[i].Epoch {
			return bf[i].Version
		}
	}
	log.Error("Fork version unknown", "epoch", epoch)
	return nil
}

func (bf Forks) domain(epoch uint64) MerkleValue {
	for i := len(bf) - 1; i >= 0; i-- {
		if epoch >= bf[i].Epoch {
			return bf[i].domain
		}
	}
	log.Error("Fork domain unknown", "epoch", epoch)
	return MerkleValue{}
}

func (bf Forks) computeDomains(genesisValidatorsRoot common.Hash) {
	for i := range bf {
		bf[i].domain = computeDomain(bf[i].Version, genesisValidatorsRoot)
	}
}

func (bf Forks) epoch(name string) (uint64, bool) {
	for _, fork := range bf {
		if fork.Name == name {
			return fork.Epoch, true
		}
	}
	return 0, false
}

func (f Forks) Len() int           { return len(f) }
func (f Forks) Swap(i, j int)      { f[i], f[j] = f[j], f[i] }
func (f Forks) Less(i, j int) bool { return f[i].Epoch < f[j].Epoch }

func fieldValue(line, field string) (name, value string, ok bool) {
	if pos := strings.Index(line, field); pos >= 0 {
		return line[:pos], strings.TrimSpace(line[pos+len(field):]), true
	}
	return "", "", false
}

func LoadForks(fileName string) (Forks, error) {
	file, err := os.Open(fileName)
	if err != nil {
		return nil, fmt.Errorf("Error opening beacon chain config file: %v", err)
	}
	defer file.Close()

	forkVersions := make(map[string][]byte)
	forkEpochs := make(map[string]uint64)
	reader := bufio.NewReader(file)
	for {
		l, _, err := reader.ReadLine()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("Error reading beacon chain config file: %v", err)
		}
		line := string(l)
		if name, value, ok := fieldValue(line, "_FORK_VERSION:"); ok {
			if v, err := hexutil.Decode(value); err == nil {
				forkVersions[name] = v
			} else {
				return nil, fmt.Errorf("Error decoding hex fork id \"%s\" in beacon chain config file: %v", value, err)
			}
		}
		if name, value, ok := fieldValue(line, "_FORK_EPOCH:"); ok {
			if v, err := strconv.ParseUint(value, 10, 64); err == nil {
				forkEpochs[name] = v
			} else {
				return nil, fmt.Errorf("Error parsing epoch number \"%s\" in beacon chain config file: %v", value, err)
			}
		}
	}

	var forks Forks
	forkEpochs["GENESIS"] = 0
	for name, epoch := range forkEpochs {
		if version, ok := forkVersions[name]; ok {
			delete(forkVersions, name)
			forks = append(forks, Fork{Epoch: epoch, Name: name, Version: version})
		} else {
			return nil, fmt.Errorf("Fork id missing for \"%s\" in beacon chain config file", name)
		}
	}
	for name := range forkVersions {
		return nil, fmt.Errorf("Epoch number missing for fork \"%s\" in beacon chain config file", name)
	}
	sort.Sort(forks)
	return forks, nil
}

type SctConstraints interface {
	PeriodRange() (first, afterFixed, afterLast uint64)
	CommitteeRoot(period uint64) (root common.Hash, matchAll bool)
	SetCallbacks(initCallback func(genesisData), updateCallback func())
}

type genesisData struct {
	GenesisTime           uint64
	GenesisValidatorsRoot common.Hash
}

type checkpointData struct {
	Checkpoint                       common.Hash
	Period                           uint64
	CommitteeRoot, NextCommitteeRoot common.Hash
}

type lightClientInitData struct {
	genesisData
	checkpointData
}

func (block *BlockData) getGenesisData() (genesisData, bool) {
	if block.ProofFormat&HspInitData == 0 {
		//log.Error("Genesis data not available in beacon state")
		return genesisData{}, false
	}
	gt := block.mustGetStateValue(BsiGenesisTime)
	return genesisData{
		GenesisTime:           binary.LittleEndian.Uint64(gt[:]), //TODO check if encoding is correct
		GenesisValidatorsRoot: common.Hash(block.mustGetStateValue(BsiGenesisValidators)),
	}, true
}

func (block *BlockData) getCheckpointData() (checkpointData, bool) {
	if block.ProofFormat&HspInitData == 0 {
		log.Error("Checkpoint data not available in beacon state")
		return checkpointData{}, false
	}
	return checkpointData{
		Checkpoint:        block.BlockRoot,
		Period:            block.Header.Slot >> 13,
		CommitteeRoot:     common.Hash(block.mustGetStateValue(BsiSyncCommittee)),
		NextCommitteeRoot: common.Hash(block.mustGetStateValue(BsiNextSyncCommittee)),
	}, true
}

func (block *BlockData) getInitData() (lightClientInitData, bool) {
	genesisData, ok1 := block.getGenesisData()
	checkpointData, ok2 := block.getCheckpointData()
	if !(ok1 && ok2) {
		return lightClientInitData{}, false
	}
	return lightClientInitData{
		genesisData:    genesisData,
		checkpointData: checkpointData,
	}, true
}

type sctInitBackend interface {
	GetInitBlock(ctx context.Context, checkpoint common.Hash) (*BlockData, error)
}

type WeakSubjectivityCheckpoint struct {
	lock sync.RWMutex

	parent                              SctConstraints // constraints applied to pre-checkpoint history (no old committees synced if nil)
	db                                  ethdb.Database
	initData                            lightClientInitData
	initialized                         bool
	initTriggerCh, parentInitCh, stopCh chan struct{}
	initCallback                        func(genesisData)
	updateCallback                      func()
}

func NewWeakSubjectivityCheckpoint(db ethdb.Database, backend sctInitBackend, checkpoint common.Hash, parent SctConstraints) *WeakSubjectivityCheckpoint {
	wsc := &WeakSubjectivityCheckpoint{
		parent:        parent,
		db:            db,
		initTriggerCh: make(chan struct{}, 1),
		stopCh:        make(chan struct{}),
	}
	if parent != nil {
		wsc.parentInitCh = make(chan struct{})
	}

	var haveInitData bool
	if enc, err := db.Get(initDataKey); err == nil {
		var initData lightClientInitData
		if err := rlp.DecodeBytes(enc, &initData); err == nil {
			//fmt.Println("Found stored initData", initData.Checkpoint, checkpoint)
			if initData.Checkpoint == checkpoint || initData.Checkpoint == (common.Hash{}) {
				log.Info("Beacon chain initialized with stored checkpoint", "checkpoint", initData.Checkpoint)
				wsc.initData = initData
				haveInitData = true
			} /* else {
				log.Info("Ignoring stored beacon checkpoint")
			}*/
		} else {
			log.Error("Error decoding stored beacon checkpoint", "error", err)
		}
	}
	if !haveInitData && checkpoint == (common.Hash{}) {
		return nil
	}
	go func() {
		var initData lightClientInitData
		if !haveInitData {
		loop:
			for {
				select {
				case <-wsc.stopCh:
					return
				case <-wsc.initTriggerCh:
					ctx, _ := context.WithTimeout(context.Background(), time.Second*20)
					//fmt.Println("Fetching init block at checkpoint", checkpoint)
					log.Info("Requesting beacon init block", "checkpoint", checkpoint)
					if block, err := backend.GetInitBlock(ctx, checkpoint); err == nil {
						//fmt.Println("Received init block:", block)
						var ok bool
						if initData, ok = block.getInitData(); ok {
							log.Info("Successfully initialized beacon chain", "checkpoint", checkpoint, "slot", block.Header.Slot)
							break loop
						} else {
							log.Error("Init data not found in validated checkpoint block") // should not happen
						}
					} else {
						log.Warn("Failed to retrieve beacon init block", "error", err)
					}
				}
			}
		}
		//fmt.Println("wsc 2")
		if wsc.parentInitCh != nil {
			select {
			case <-wsc.stopCh:
				return
			case <-wsc.parentInitCh:
			}
		}
		//fmt.Println("wsc 3")
		wsc.lock.Lock()
		wsc.initData = initData
		wsc.initialized = true
		initCallback := wsc.initCallback
		wsc.initCallback = nil
		wsc.lock.Unlock()
		if initCallback != nil {
			//fmt.Println("wsc 4")
			initCallback(initData.genesisData)
		}
		//fmt.Println("wsc 5")
	}()
	return wsc
}

func (wsc *WeakSubjectivityCheckpoint) init(initData lightClientInitData) {
	wsc.lock.Lock()
	if enc, err := rlp.EncodeToBytes(&initData); err == nil {
		wsc.db.Put(initDataKey, enc)
	} else {
		log.Error("Error encoding initData", "error", err)
	}
	wsc.initData, wsc.initialized = initData, true
	updateCallback, initCallback := wsc.updateCallback, wsc.initCallback
	wsc.lock.Unlock()
	if initCallback != nil {
		initCallback(initData.genesisData)
	}
	updateCallback()
}

func (wsc *WeakSubjectivityCheckpoint) PeriodRange() (first, afterFixed, afterLast uint64) {
	wsc.lock.RLock()
	defer wsc.lock.RUnlock()

	if !wsc.initialized {
		return
	}
	if wsc.parent != nil {
		first, afterFixed, afterLast = wsc.parent.PeriodRange()
	}
	if afterFixed < wsc.initData.Period {
		first = wsc.initData.Period
	}
	if afterFixed < wsc.initData.Period+2 {
		afterFixed = wsc.initData.Period + 2
	}
	afterLast = math.MaxUint64 // no constraints on valid committee updates after the checkpoint
	return
}

// aktualis root-ot az elozo update next-jehez hasonlitjuk, next-et a constraint-re matcheljuk
/*func (wsc *WeakSubjectivityCheckpoint) CommitteeRootMatch(period uint64, root common.Hash) bool {
	wsc.lock.RLock()
	defer wsc.lock.RUnlock()

	if !wsc.initialized {
		return false
	}
	switch {
	case period < wsc.initData.Period:
		return wsc.parent != nil && wsc.parent.CommitteeRootMatch(period, root)
	case period == wsc.initData.Period:
		return root == wsc.initData.CommitteeRoot
	case period == wsc.initData.Period+1:
		return root == wsc.initData.NextCommitteeRoot
	default:
		return true // match all, no constraints on valid committee updates after the checkpoint
	}
}*/

func (wsc *WeakSubjectivityCheckpoint) CommitteeRoot(period uint64) (root common.Hash, matchAll bool) {
	wsc.lock.RLock()
	defer wsc.lock.RUnlock()

	if !wsc.initialized {
		return common.Hash{}, false
	}
	switch {
	case period < wsc.initData.Period:
		if wsc.parent != nil {
			return wsc.parent.CommitteeRoot(period)
		}
		return common.Hash{}, false
	case period == wsc.initData.Period:
		return wsc.initData.CommitteeRoot, false
	case period == wsc.initData.Period+1:
		return wsc.initData.NextCommitteeRoot, false
	default:
		return common.Hash{}, true // match all, no constraints on valid committee updates after the checkpoint
	}
}

func (wsc *WeakSubjectivityCheckpoint) SetCallbacks(initCallback func(genesisData), updateCallback func()) {
	wsc.lock.Lock()
	if wsc.initialized {
		wsc.lock.Unlock()
		initCallback(wsc.initData.genesisData)
	} else {
		wsc.initCallback = initCallback
		wsc.updateCallback = updateCallback
		wsc.lock.Unlock()
	}
	if wsc.parent != nil {
		wsc.parent.SetCallbacks(func(genesisData) { close(wsc.parentInitCh) }, updateCallback)
	}
}

func (wsc *WeakSubjectivityCheckpoint) TriggerFetch() {
	select {
	case wsc.initTriggerCh <- struct{}{}:
	default:
	}
}

// call after ODR backend shutdown to ensure that init request does not get stuck
func (wsc *WeakSubjectivityCheckpoint) Stop() {
	close(wsc.stopCh)
}
