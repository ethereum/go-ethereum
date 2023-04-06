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

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/rlp"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrEmptySlot          = errors.New("empty slot")
	ErrInvalidProofFormat = errors.New("invalid proof format")
	ErrInvalidStateRoot   = errors.New("invalid state root")
)

var (
	chainRangeKey = []byte("range-")     // RLP(chainRangeData)
	headerKey     = []byte("header-")    // bigEndian64(slot) + blockRoot -> RLP(types.Header)
	stateKey      = []byte("state-")     // bigEndian64(slot) + stateRoot -> RLP(stateProofData)
	canonicalKey  = []byte("canonical-") // bigEndian64(slot) -> canonical root
	hashToSlotKey = []byte("hash2slot-") // blockRoot -> RLP(slot)
)

type LightChain struct {
	lock sync.RWMutex
	db   ethdb.KeyValueStore

	chainHead, chainTail types.Header
	chainInit            bool
	stateHead, stateTail types.Header
	stateInit            bool
	lastStoredRange      chainRangeData

	headerCache      *lru.Cache[slotAndHash, types.Header]
	canonicalCache   *lru.Cache[uint64, common.Hash]
	hashToSlotCache  *lru.Cache[common.Hash, uint64]
	stateCache       *lru.Cache[slotAndHash, merkle.Values]
	stateProofFormat merkle.ProofFormat //TODO slot/parentSlot dependent format
}

type slotAndHash struct {
	slot uint64
	hash common.Hash
}

type chainRangeData struct {
	ChainInit            bool
	ChainHead, ChainTail uint64
	StateInit            bool
	StateHead, StateTail uint64
}

type stateProofData struct {
	FormatId uint //TODO compact binary format?
	Values   merkle.Values
}

func NewLightChain(db ethdb.KeyValueStore, stateProofFormat merkle.ProofFormat) *LightChain {
	lc := &LightChain{
		db:               db,
		stateProofFormat: stateProofFormat,
		headerCache:      lru.NewCache[slotAndHash, types.Header](500),
		canonicalCache:   lru.NewCache[uint64, common.Hash](2000),
		hashToSlotCache:  lru.NewCache[common.Hash, uint64](2000),
		stateCache:       lru.NewCache[slotAndHash, merkle.Values](100),
	}
	lc.loadChainRange()
	return lc
}

func (lc *LightChain) loadChainRange() {
	if rangeEnc, err := lc.db.Get(chainRangeKey); err == nil {
		var cr chainRangeData
		if err := rlp.DecodeBytes(rangeEnc, &cr); err != nil {
			log.Error("Failed to decode chain range data", "error", err)
			return
		}
		if cr.ChainInit {
			if lc.chainHead, err = lc.getHeaderBySlot(cr.ChainHead); err != nil {
				log.Error("Chain head not found")
				return
			}
			if lc.chainTail, err = lc.getHeaderBySlot(cr.ChainTail); err != nil {
				log.Error("Chain tail not found")
				return
			}
			lc.chainInit = true
		}
		if cr.StateInit {
			if lc.stateHead, err = lc.getHeaderBySlot(cr.StateHead); err != nil || !lc.HasStateProof(lc.stateHead) {
				log.Error("State head not found")
				return
			}
			if lc.stateTail, err = lc.getHeaderBySlot(cr.StateTail); err != nil || !lc.HasStateProof(lc.stateTail) {
				log.Error("State tail not found")
				return
			}
			lc.stateInit = true
		}
		lc.lastStoredRange = cr
	}
}

func (lc *LightChain) storeChainRange(batch ethdb.Batch) {
	cr := chainRangeData{
		ChainInit: lc.chainInit,
		ChainHead: lc.chainHead.Slot,
		ChainTail: lc.chainTail.Slot,
		StateInit: lc.stateInit,
		StateHead: lc.stateHead.Slot,
		StateTail: lc.stateTail.Slot,
	}
	if cr == lc.lastStoredRange {
		return
	}
	rangeEnc, err := rlp.EncodeToBytes(&cr)
	if err != nil {
		log.Error("Failed to encode chain range data", "error", err)
		return
	}
	batch.Put(chainRangeKey, rangeEnc)
}

func getHeaderKey(slot uint64, blockRoot common.Hash) []byte {
	var (
		kl  = len(headerKey)
		key = make([]byte, kl+8+32)
	)
	copy(key[:kl], headerKey)
	binary.BigEndian.PutUint64(key[kl:kl+8], slot)
	copy(key[kl+8:], blockRoot[:])
	return key
}

func getStateKey(slot uint64, stateRoot common.Hash) []byte {
	var (
		kl  = len(stateKey)
		key = make([]byte, kl+8+32)
	)
	copy(key[:kl], stateKey)
	binary.BigEndian.PutUint64(key[kl:kl+8], slot)
	copy(key[kl+8:], stateRoot[:])
	return key
}

func getCanonicalKey(slot uint64) []byte {
	var (
		kl  = len(canonicalKey)
		key = make([]byte, kl+8)
	)
	copy(key[:kl], canonicalKey)
	binary.BigEndian.PutUint64(key[kl:kl+8], slot)
	return key
}

func getHashToSlotKey(blockRoot common.Hash) []byte {
	var (
		kl  = len(hashToSlotKey)
		key = make([]byte, kl+32)
	)
	copy(key[:kl], hashToSlotKey)
	copy(key[kl:], blockRoot[:])
	return key
}

func (lc *LightChain) getCanonicalHash(slot uint64) common.Hash {
	if blockRoot, ok := lc.canonicalCache.Get(slot); ok {
		return blockRoot
	}
	var blockRoot common.Hash
	if data, err := lc.db.Get(getCanonicalKey(slot)); err == nil && len(data) == len(blockRoot) {
		copy(blockRoot[:], data)
	}
	lc.canonicalCache.Add(slot, blockRoot)
	return blockRoot
}

func (lc *LightChain) storeCanonicalHash(batch ethdb.Batch, slot uint64, blockRoot common.Hash) {
	if blockRoot == (common.Hash{}) {
		lc.deleteCanonicalHash(batch, slot)
		return
	}
	batch.Put(getCanonicalKey(slot), blockRoot[:])
	lc.canonicalCache.Add(slot, blockRoot)
}

func (lc *LightChain) deleteCanonicalHash(batch ethdb.Batch, slot uint64) {
	batch.Delete(getCanonicalKey(slot))
	lc.canonicalCache.Add(slot, common.Hash{})
}

func (lc *LightChain) AddHeader(header types.Header) {
	lc.lock.Lock()
	defer lc.lock.Unlock()

	batch := lc.db.NewBatch()
	blockRoot := header.Hash()
	headerEnc, err := rlp.EncodeToBytes(&header)
	if err != nil {
		log.Error("Failed to encode beacon header", "error", err)
		return
	}
	batch.Put(getHeaderKey(header.Slot, blockRoot), headerEnc)
	lc.headerCache.Add(slotAndHash{header.Slot, blockRoot}, header)
	slotEnc, err := rlp.EncodeToBytes(&header.Slot)
	if err != nil {
		log.Error("Failed to encode slot number", "error", err)
		return
	}
	batch.Put(getHashToSlotKey(blockRoot), slotEnc)
	lc.hashToSlotCache.Add(blockRoot, header.Slot)
	if lc.chainInit && blockRoot == lc.chainTail.ParentRoot {
		var err error
		for err == nil {
			lc.storeCanonicalHash(batch, header.Slot, header.Hash())
			for slot := header.Slot + 1; slot < lc.chainTail.Slot; slot++ {
				lc.deleteCanonicalHash(batch, slot)
			}
			lc.chainTail = header
			header, err = lc.GetParent(header)
		}
		lc.storeChainRange(batch)
	}
	if err := batch.Write(); err != nil {
		log.Error("Failed to write batch to database", "error", err)
	}
}

func (lc *LightChain) SetChainHead(head types.Header) {
	lc.lock.Lock()
	defer lc.lock.Unlock()

	batch := lc.db.NewBatch()
	defer func() {
		lc.storeChainRange(batch)
		if err := batch.Write(); err != nil {
			log.Error("Failed to write batch to database", "error", err)
		}
	}()

	if !lc.chainInit {
		lc.chainInit = true
		lc.chainHead = head
		lc.chainTail = head
	}
	for slot := head.Slot + 1; slot <= lc.chainHead.Slot; slot++ {
		lc.deleteCanonicalHash(batch, slot)
	}
	lc.chainHead = head
	for !lc.IsCanonical(head) {
		lc.storeCanonicalHash(batch, head.Slot, head.Hash())
		parent, err := lc.GetParent(head)
		if err != nil {
			for slot := lc.chainTail.Slot; slot < head.Slot; slot++ {
				lc.deleteCanonicalHash(batch, slot)
			}
			lc.chainTail = head
			lc.stateInit = false
			lc.reinitStateChain(batch, head)
			return
		}
		for slot := parent.Slot + 1; slot < head.Slot; slot++ {
			lc.deleteCanonicalHash(batch, slot)
		}
		head = parent
	}
	if lc.stateInit && lc.stateHead.Slot >= head.Slot {
		if head.Slot >= lc.stateTail.Slot {
			lc.stateHead = head
		} else {
			lc.stateInit = false
		}
	}
	if lc.stateInit {
		lc.extendStateHead(batch)
	} else {
		lc.reinitStateChain(batch, head)
	}
}

func (lc *LightChain) HeaderRange() (head, tail types.Header, init bool) {
	lc.lock.RLock()
	defer lc.lock.RUnlock()

	return lc.chainHead, lc.chainTail, lc.chainInit
}

func (lc *LightChain) getHeader(slot uint64, blockRoot common.Hash) (types.Header, error) {
	if header, ok := lc.headerCache.Get(slotAndHash{slot, blockRoot}); ok {
		return header, nil
	}
	headerEnc, err := lc.db.Get(getHeaderKey(slot, blockRoot))
	if err != nil {
		return types.Header{}, ErrNotFound
	}
	var header types.Header
	if err := rlp.DecodeBytes(headerEnc, &header); err != nil {
		log.Error("Failed to decode beacon header", "error", err)
		return types.Header{}, ErrNotFound
	}
	return header, nil
}

func (lc *LightChain) getSlotByHash(blockRoot common.Hash) (uint64, bool) {
	if slot, ok := lc.hashToSlotCache.Get(blockRoot); ok {
		return slot, true
	}
	slotEnc, err := lc.db.Get(getHashToSlotKey(blockRoot))
	if err != nil {
		return 0, false
	}
	var slot uint64
	if err := rlp.DecodeBytes(slotEnc, &slot); err != nil {
		log.Error("Failed to decode slot number", "error", err)
		return 0, false
	}
	return slot, true
}

func (lc *LightChain) HasHeader(blockRoot common.Hash) bool {
	_, ok := lc.getSlotByHash(blockRoot)
	return ok
}

func (lc *LightChain) GetHeaderByHash(blockRoot common.Hash) (types.Header, error) {
	if slot, ok := lc.getSlotByHash(blockRoot); ok {
		header, err := lc.getHeader(slot, blockRoot)
		if err != nil {
			log.Error("LightChain slot -> blockRoot entry found but header is missing", "slot", slot, "blockRoot", blockRoot)
		}
		return header, err
	}
	return types.Header{}, ErrNotFound
}

func (lc *LightChain) GetHeaderBySlot(slot uint64) (types.Header, error) {
	lc.lock.RLock()
	defer lc.lock.RUnlock()

	return lc.getHeaderBySlot(slot)
}

func (lc *LightChain) getHeaderBySlot(slot uint64) (types.Header, error) {
	if !lc.chainInit || slot < lc.chainTail.Slot || slot > lc.chainHead.Slot {
		return types.Header{}, ErrNotFound
	}

	blockRoot := lc.getCanonicalHash(slot)
	if blockRoot == (common.Hash{}) {
		return types.Header{}, ErrEmptySlot
	}
	header, err := lc.getHeader(slot, blockRoot)
	if err != nil {
		log.Error("LightChain canonical blockRoot entry found but header is missing", "slot", slot, "blockRoot", blockRoot)
	}
	return header, err
}

func (lc *LightChain) GetParent(header types.Header) (types.Header, error) {
	return lc.GetHeaderByHash(header.ParentRoot)
}

func (lc *LightChain) IsCanonical(header types.Header) bool {
	return lc.getCanonicalHash(header.Slot) == header.Hash()
}

func (lc *LightChain) StateProofRange() (head, tail types.Header, init bool) {
	lc.lock.RLock()
	defer lc.lock.RUnlock()

	return lc.stateHead, lc.stateTail, lc.stateInit
}

func (lc *LightChain) extendStateHead(batch ethdb.Batch) {
	for slot := lc.stateHead.Slot + 1; slot <= lc.chainHead.Slot; slot++ {
		if header, err := lc.getHeaderBySlot(slot); err == nil {
			if lc.HasStateProof(header) {
				lc.stateHead = header
			} else {
				break
			}
		}
	}
}

func (lc *LightChain) extendStateTail(batch ethdb.Batch) {
	if lc.stateTail.Slot == 0 {
		return
	}
	for slot := lc.stateTail.Slot - 1; slot >= lc.chainTail.Slot; slot-- {
		if header, err := lc.getHeaderBySlot(slot); err == nil {
			if lc.HasStateProof(header) {
				lc.stateTail = header
			} else {
				break
			}
		}
	}
}

func (lc *LightChain) reinitStateChain(batch ethdb.Batch, header types.Header) {
	for slot := header.Slot; slot <= lc.chainHead.Slot; slot++ {
		if header, err := lc.getHeaderBySlot(slot); err == nil && lc.HasStateProof(header) {
			lc.stateInit = true
			lc.stateHead = header
			lc.stateTail = header
			lc.extendStateHead(batch)
			return
		}
	}
}

func (lc *LightChain) HasStateProof(header types.Header) bool {
	if _, ok := lc.stateCache.Get(slotAndHash{header.Slot, header.StateRoot}); ok {
		return true
	}
	ok, err := lc.db.Has(getStateKey(header.Slot, header.StateRoot))
	return ok && err == nil
}

func (lc *LightChain) GetStateProof(header types.Header) (merkle.MultiProof, error) {
	if values, ok := lc.stateCache.Get(slotAndHash{header.Slot, header.StateRoot}); ok {
		return merkle.MultiProof{Format: lc.stateProofFormat, Values: values}, nil
	}
	stateEnc, err := lc.db.Get(getStateKey(header.Slot, header.StateRoot))
	if err != nil {
		return merkle.MultiProof{}, ErrNotFound
	}
	var state stateProofData
	if err := rlp.DecodeBytes(stateEnc, &state); err != nil {
		log.Error("Failed to decode state proof data", "error", err)
		return merkle.MultiProof{}, ErrNotFound
	}
	return merkle.MultiProof{Format: lc.stateProofFormat, Values: state.Values}, nil
}

func (lc *LightChain) StateProofFormat(header types.Header) merkle.ProofFormat {
	return lc.stateProofFormat
}

func (lc *LightChain) AddStateProof(header types.Header, proof merkle.MultiProof) error {
	lc.lock.Lock()
	defer lc.lock.Unlock()

	if !merkle.IsEqual(proof.Format, lc.StateProofFormat(header)) {
		return ErrInvalidProofFormat
	}
	if proof.RootHash() != header.StateRoot {
		return ErrInvalidStateRoot
	}
	batch := lc.db.NewBatch()

	stateEnc, err := rlp.EncodeToBytes(&stateProofData{Values: proof.Values})
	if err != nil {
		log.Error("Failed to encode state proof data", "error", err)
		return err
	}
	batch.Put(getStateKey(header.Slot, header.StateRoot), stateEnc)
	lc.stateCache.Add(slotAndHash{header.Slot, header.StateRoot}, proof.Values)
	if !lc.IsCanonical(header) {
		return nil
	}
	if !lc.stateInit {
		lc.stateInit = true
		lc.stateHead = header
		lc.stateTail = header
		return nil
	}
	if header.Slot > lc.stateHead.Slot && header.Slot <= lc.chainHead.Slot {
		lc.extendStateHead(batch)
	} else if header.Slot < lc.stateTail.Slot && header.Slot >= lc.chainTail.Slot {
		lc.extendStateTail(batch)
	}

	lc.storeChainRange(batch)
	if err := batch.Write(); err != nil {
		log.Error("Failed to write batch to database", "error", err)
		return err
	}
	return nil
}

func (lc *LightChain) DeleteBefore(beforeSlot uint64) {
	lc.lock.Lock()
	defer lc.lock.Unlock()

	lc.deleteBefore(beforeSlot, true)
}

func (lc *LightChain) DeleteNonCanonical(beforeSlot uint64) {
	lc.lock.Lock()
	defer lc.lock.Unlock()

	lc.deleteBefore(beforeSlot, false)
}

func (lc *LightChain) deleteBefore(beforeSlot uint64, removeCanonical bool) {
	if !lc.chainInit {
		return
	}
	batch := lc.db.NewBatch()
	defer func() {
		if err := batch.Write(); err != nil {
			log.Error("Failed to write batch to database", "error", err)
		}
	}()

	if removeCanonical {
		// remove canonical hashes
		iter := lc.db.NewIterator(canonicalKey, nil)
		kl := len(canonicalKey)
		for {
			if !iter.Next() {
				lc.chainInit = false
				break
			}
			key := iter.Key()
			if len(key) != kl+8 {
				log.Error("Canonical hash entry found with invalid key length")
				continue
			}
			slot := binary.BigEndian.Uint64(key[kl:])
			if slot >= beforeSlot {
				var err error
				if lc.chainTail, err = lc.getHeaderBySlot(slot); err != nil {
					log.Error("Could not find new chain tail")
					lc.chainInit = false
					break
				}
				if lc.stateInit && lc.chainTail.Slot > lc.stateTail.Slot {
					lc.stateTail = lc.chainTail
				}
				break
			}
			batch.Delete(key)
			lc.canonicalCache.Remove(slot)
		}
		lc.storeChainRange(batch)
	}
	// remove headers and hash-to-slot entries
	iter := lc.db.NewIterator(headerKey, nil)
	kl := len(headerKey)
	for iter.Next() {
		key := iter.Key()
		if len(key) != kl+8+32 {
			log.Error("Header entry found with invalid key length")
			break
		}
		slot := binary.BigEndian.Uint64(key[kl : kl+8])
		if slot >= beforeSlot {
			break
		}
		var blockRoot common.Hash
		copy(blockRoot[:], key[kl+8:])
		if removeCanonical || blockRoot != lc.getCanonicalHash(slot) {
			batch.Delete(getHashToSlotKey(blockRoot))
			lc.hashToSlotCache.Remove(blockRoot)
			batch.Delete(key)
			lc.headerCache.Remove(slotAndHash{slot: slot, hash: blockRoot})
		}
	}
	if !lc.stateInit {
		return
	}
	// remove states
	iter = lc.db.NewIterator(stateKey, nil)
	kl = len(stateKey)
	for iter.Next() {
		key := iter.Key()
		if len(key) != kl+8+32 {
			log.Error("State entry found with invalid key length")
			break
		}
		slot := binary.BigEndian.Uint64(key[kl : kl+8])
		if slot >= beforeSlot {
			break
		}
		var stateRoot common.Hash
		copy(stateRoot[:], key[kl+8:])
		if !removeCanonical {
			if header, err := lc.getHeaderBySlot(slot); err != nil && header.StateRoot == stateRoot {
				continue
			}
		}
		batch.Delete(key)
		lc.stateCache.Remove(slotAndHash{slot: slot, hash: stateRoot})
	}
}
