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
	"time"

	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/log"
)

type HeadTracker struct {
	lock            sync.RWMutex
	committeeChain  *CommitteeChain
	minSignerCount  int
	signedHead      types.SignedHeader
	headSignerCount int
	prefetchHead    types.HeadInfo
}

func NewHeadTracker(committeeChain *CommitteeChain, minSignerCount int) *HeadTracker {
	return &HeadTracker{
		committeeChain: committeeChain,
		minSignerCount: minSignerCount,
	}
}

func (h *HeadTracker) ValidatedHead() types.SignedHeader {
	h.lock.RLock()
	defer h.lock.RUnlock()

	return h.signedHead
}

func (h *HeadTracker) Validate(head types.SignedHeader) (bool, error) {
	h.lock.Lock()
	defer h.lock.Unlock()

	signerCount := head.Signature.SignerCount()
	if signerCount < h.minSignerCount {
		return false, errors.New("low signer count")
	}
	if head.Header.Slot < h.signedHead.Header.Slot || (head.Header.Slot == h.signedHead.Header.Slot && signerCount <= h.headSignerCount) {
		return false, nil
	}
	sigOk, age, err := h.committeeChain.VerifySignedHeader(head)
	if err != nil {
		return false, err
	}
	if age < 0 {
		log.Warn("Future signed head received", "age", age)
	}
	if age > time.Minute*2 {
		log.Warn("Old signed head received", "age", age)
	}
	if !sigOk {
		return false, errors.New("invalid header signature")
	}
	h.signedHead, h.headSignerCount = head, signerCount
	return true, nil
}

func (h *HeadTracker) PrefetchHead() types.HeadInfo {
	h.lock.RLock()
	defer h.lock.RUnlock()

	return h.prefetchHead
}

func (h *HeadTracker) SetPrefetchHead(head types.HeadInfo) {
	h.lock.Lock()
	defer h.lock.Unlock()

	h.prefetchHead = head
}
