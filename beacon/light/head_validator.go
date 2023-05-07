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

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/log"
)

type HeadValidator struct {
	lock           sync.Mutex
	committeeChain *CommitteeChain
	subs           []*headSub
}

func NewHeadValidator(committeeChain *CommitteeChain) *HeadValidator {
	return &HeadValidator{committeeChain: committeeChain}
}

type headSub struct {
	minSignerCount int
	nextSlot       uint64
	callbacks      []func(types.SignedHeader)
}

func (h *HeadValidator) Subscribe(minSignerCount int, callback func(types.SignedHeader)) {
	h.lock.Lock()
	defer h.lock.Unlock()

	insertAt := len(h.subs)
	for i, sub := range h.subs {
		if sub.minSignerCount == minSignerCount {
			sub.callbacks = append(sub.callbacks, callback)
			return
		}
		if sub.minSignerCount > minSignerCount {
			insertAt = i
			break
		}
	}
	h.subs = append(h.subs, nil)
	copy(h.subs[insertAt+1:], h.subs[insertAt:len(h.subs)-1])
	h.subs[insertAt] = &headSub{
		minSignerCount: minSignerCount,
		callbacks:      []func(types.SignedHeader){callback},
	}
}

func (h *HeadValidator) Add(head types.SignedHeader) error {
	h.lock.Lock()
	defer h.lock.Unlock()

	sigOk, age := h.committeeChain.VerifySignedHeader(head)
	if age < 0 {
		log.Warn("Future signed head received", "age", age)
	}
	if age > time.Minute*2 {
		log.Warn("Old signed head received", "age", age)
	}
	if !sigOk {
		return errors.New("invalid header signature")
	}

	signerCount := head.SyncAggregate.SignerCount()
	for _, sub := range h.subs {
		if sub.minSignerCount > signerCount {
			break
		}
		if head.Header.Slot >= sub.nextSlot {
			for _, cb := range sub.callbacks {
				cb(head)
			}
			sub.nextSlot = head.Header.Slot + 1
		}
	}
	return nil
}
