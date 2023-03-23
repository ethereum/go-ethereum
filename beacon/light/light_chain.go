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
	"sync"

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/merkle"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/lru"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/log"
	//"github.com/ethereum/go-ethereum/rlp"
)

var (
	ErrNotFound           = errors.New("not found")
	ErrEmptySlot          = errors.New("empty slot")
	ErrInvalidProofFormat = errors.New("invalid proof format")
	ErrInvalidStateRoot   = errors.New("invalid state root")
)

type LightChain struct {
	lock                 sync.RWMutex
	db                   ethdb.KeyValueStore //TODO implement database
	chainHead, chainTail types.Header
	chainInit            bool
	stateHead, stateTail types.Header
	stateInit            bool
	headerCache          *lru.Cache[slotAndHash, types.Header]
	canonicalCache       *lru.Cache[uint64, common.Hash]
	slotCache            *lru.Cache[common.Hash, uint64]
	stateCache           *lru.Cache[slotAndHash, merkle.Values]
	stateProofFormat     merkle.ProofFormat //TODO slot/parentSlot dependent format
}

func NewLightChain(db ethdb.KeyValueStore, stateProofFormat merkle.ProofFormat) *LightChain {
	//TODO init from db
	return &LightChain{
		db:               db,
		stateProofFormat: stateProofFormat,
		headerCache:      lru.NewCache[slotAndHash, types.Header](10000), //TODO use smaller cache when db is implemented
		canonicalCache:   lru.NewCache[uint64, common.Hash](10000),
		slotCache:        lru.NewCache[common.Hash, uint64](10000),
		stateCache:       lru.NewCache[slotAndHash, merkle.Values](10000),
	}
}

type slotAndHash struct {
	slot uint64
	hash common.Hash
}

func (lc *LightChain) AddHeader(header types.Header) {
	lc.lock.Lock()
	defer lc.lock.Unlock()

	blockRoot := header.Hash()
	lc.headerCache.Add(slotAndHash{header.Slot, blockRoot}, header)
	lc.slotCache.Add(blockRoot, header.Slot)
	if lc.chainInit && blockRoot == lc.chainTail.ParentRoot {
		var err error
		for err == nil {
			lc.canonicalCache.Add(header.Slot, header.Hash())
			for slot := header.Slot + 1; slot < lc.chainTail.Slot; slot++ {
				lc.canonicalCache.Add(slot, common.Hash{})
			}
			lc.chainTail = header
			header, err = lc.GetHeaderByHash(header.ParentRoot)
		}
	}
}

func (lc *LightChain) SetChainHead(head types.Header) {
	lc.lock.Lock()
	defer lc.lock.Unlock()

	if !lc.chainInit {
		lc.chainInit = true
		lc.chainHead = head
		lc.chainTail = head
	}
	for slot := head.Slot + 1; slot <= lc.chainHead.Slot; slot++ {
		lc.canonicalCache.Remove(slot)
	}
	lc.chainHead = head
	for !lc.IsCanonical(head) {
		lc.canonicalCache.Add(head.Slot, head.Hash())
		parent, err := lc.GetParent(head)
		if err != nil {
			for slot := lc.chainTail.Slot; slot < head.Slot; slot++ {
				lc.canonicalCache.Remove(slot)
			}
			lc.chainTail = head
			lc.stateInit = false
			lc.reinitStateChain(head)
			return
		}
		for slot := parent.Slot + 1; slot < head.Slot; slot++ {
			lc.canonicalCache.Add(slot, common.Hash{})
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
		lc.extendStateHead()
	} else {
		lc.reinitStateChain(head)
	}
}

func (lc *LightChain) extendStateHead() {
	for slot := lc.stateHead.Slot + 1; slot <= lc.chainHead.Slot; slot++ {
		if header, err := lc.GetHeaderBySlot(slot); err == nil {
			if lc.HasStateProof(header) {
				lc.stateHead = header
			} else {
				break
			}
		}
	}
}

func (lc *LightChain) extendStateTail() {
	if lc.stateTail.Slot == 0 {
		return
	}
	for slot := lc.stateTail.Slot - 1; slot >= lc.chainTail.Slot; slot-- {
		if header, err := lc.GetHeaderBySlot(slot); err == nil {
			if lc.HasStateProof(header) {
				lc.stateTail = header
			} else {
				break
			}
		}
	}
}

func (lc *LightChain) reinitStateChain(header types.Header) {
	for slot := header.Slot; slot <= lc.chainHead.Slot; slot++ {
		if header, err := lc.GetHeaderBySlot(slot); err == nil && lc.HasStateProof(header) {
			lc.stateInit = true
			lc.stateHead = header
			lc.stateTail = header
			lc.extendStateHead()
			return
		}
	}

}

func (lc *LightChain) HeaderRange() (head, tail types.Header, init bool) {
	lc.lock.RLock()
	defer lc.lock.RUnlock()

	return lc.chainHead, lc.chainTail, lc.chainInit
}

func (lc *LightChain) HasHeader(blockRoot common.Hash) bool {
	_, ok := lc.slotCache.Get(blockRoot)
	return ok
}

func (lc *LightChain) GetHeaderByHash(blockRoot common.Hash) (types.Header, error) {
	if slot, ok := lc.slotCache.Get(blockRoot); ok {
		if header, ok := lc.headerCache.Get(slotAndHash{slot, blockRoot}); ok {
			return header, nil
		}
		log.Error("LightChain slot -> blockRoot entry found but header is missing", "slot", slot, "blockRoot", blockRoot)
	}
	return types.Header{}, ErrNotFound
}

func (lc *LightChain) GetHeaderBySlot(slot uint64) (types.Header, error) {
	if blockRoot, ok := lc.canonicalCache.Get(slot); ok {
		if blockRoot == (common.Hash{}) {
			return types.Header{}, ErrEmptySlot
		}
		if header, ok := lc.headerCache.Get(slotAndHash{slot, blockRoot}); ok {
			return header, nil
		}
		log.Error("LightChain canonical blockRoot entry found but header is missing", "slot", slot, "blockRoot", blockRoot)
	}
	return types.Header{}, ErrNotFound
}

func (lc *LightChain) GetParent(header types.Header) (types.Header, error) {
	return lc.GetHeaderByHash(header.ParentRoot)
}

func (lc *LightChain) IsCanonical(header types.Header) bool {
	blockRoot, ok := lc.canonicalCache.Get(header.Slot)
	return ok && blockRoot == header.Hash()
}

func (lc *LightChain) StateProofRange() (head, tail types.Header, init bool) {
	lc.lock.RLock()
	defer lc.lock.RUnlock()

	return lc.stateHead, lc.stateTail, lc.stateInit
}

func (lc *LightChain) HasStateProof(header types.Header) bool {
	_, ok := lc.stateCache.Get(slotAndHash{header.Slot, header.StateRoot})
	return ok
}

func (lc *LightChain) GetStateProof(header types.Header) (merkle.MultiProof, error) {
	values, ok := lc.stateCache.Get(slotAndHash{header.Slot, header.StateRoot})
	if !ok {
		return merkle.MultiProof{}, ErrNotFound
	}
	return merkle.MultiProof{Format: lc.stateProofFormat, Values: values}, nil
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
		lc.extendStateHead()
	} else if header.Slot < lc.stateTail.Slot && header.Slot >= lc.chainTail.Slot {
		lc.extendStateTail()
	}
	return nil
}
