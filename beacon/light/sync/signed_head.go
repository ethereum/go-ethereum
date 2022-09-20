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
	"bytes"
	"errors"
	"math/bits"
	"time"

	"github.com/ethereum/go-ethereum/beacon/light/types"
	"github.com/ethereum/go-ethereum/beacon/params"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/log"
)

// SignedHead represents a beacon header signed by a sync committee
//
// Note: this structure is created from either an optimistic update or an instant update:
//  https://github.com/ethereum/consensus-specs/blob/dev/specs/altair/light-client/sync-protocol.md#lightclientoptimisticupdate
//  https://github.com/zsfelfoldi/beacon-APIs/blob/instant_update/apis/beacon/light_client/instant_update.yaml
type SignedHead struct {
	Header        types.Header // signed beacon header
	BitMask       []byte       // bit vector (LSB first) encoding the subset of the relevant sync committee that signed the header
	Signature     []byte       // BLS sync aggregate validating the signingRoot (Forks.signintypes.Header))
	SignatureSlot uint64       // slot in which the signature has been created (newertypes.Header.Slot, determines the signing sync committee)
}

// SignerCount returns the number of individual signers in the signature aggregate
func (s *SignedHead) SignerCount() int {
	if len(s.BitMask) != params.SyncCommitteeBitmaskSize {
		return 0 // signature check will filter it out later but we calculate this before sig check
	}
	var signerCount int
	for _, v := range s.BitMask {
		signerCount += bits.OnesCount8(v)
	}
	return signerCount
}

// Equal returns true if both the headers and the signer sets are the same
func (s *SignedHead) Equal(s2 *SignedHead) bool {
	return s.Header == s2.Header && bytes.Equal(s.BitMask, s2.BitMask) && bytes.Equal(s.Signature, s2.Signature)
}

// AddSignedHeads adds signed heads to the tracker if the syncing process has
// been finished; adds them to a deferred list otherwise that is processed when
// the syncing is finished.
func (s *CommitteeTracker) AddSignedHeads(peer ctServer, heads []SignedHead) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if sp := s.connected[peer]; sp != nil && (sp.requesting || sp.queued) {
		sp.deferredHeads = append(sp.deferredHeads, heads...)
		return nil
	}
	return s.addSignedHeads(peer, heads)
}

// addSignedHeads adds signed heads to the tracker after a successful verification
// (it is assumed that the local update chain has been synced with the given peer)
func (s *CommitteeTracker) addSignedHeads(peer ctServer, heads []SignedHead) error {
	var (
		oldHeadHash common.Hash
		err         error
	)
	if len(s.acceptedList.list) > 0 {
		oldHeadHash = s.acceptedList.list[0].hash
	}
	for _, head := range heads {
		signerCount := head.SignerCount()
		if signerCount < s.signerThreshold {
			continue
		}
		sigOk, age := s.verifySignature(head)
		if age < 0 {
			log.Warn("Future signed head received", "age", age)
		}
		if age > time.Minute*2 {
			log.Warn("Old signed head received", "age", age)
		}
		if !sigOk {
			err = errors.New("invalid header signature")
			continue
		}
		hash := head.Header.Hash()
		if h := s.acceptedList.getHead(hash); h != nil {
			h.receivedFrom[peer] = struct{}{}
			if signerCount > h.signerCount {
				h.head = head
				h.signerCount = signerCount
				h.sentTo = nil
				s.acceptedList.updateHead(h)
			}
		} else {
			h := &headInfo{
				head:         head,
				hash:         hash,
				sentTo:       make(map[ctClient]struct{}),
				receivedFrom: map[ctServer]struct{}{peer: struct{}{}},
			}
			s.acceptedList.updateHead(h)
		}
	}
	if len(s.acceptedList.list) > 0 && oldHeadHash != s.acceptedList.list[0].hash {
		head := s.acceptedList.list[0].head.Header
		for _, subFn := range s.headSubs {
			subFn(head)
		}
	}
	return err
}

// verifySignature returns true if the given signed head has a valid signature
// according to the local committee chain. The caller should ensure that the
// committees advertised by the same source where the signed head came from are
// synced before verifying the signature.
// The age of the header is also returned (the time elapsed since the beginning
// of the given slot, according to the local system clock). If enforceTime is
// true then negative age (future) headers are rejected.
func (s *CommitteeTracker) verifySignature(head SignedHead) (bool, time.Duration) {
	var (
		slotTime = int64(time.Second) * int64(s.genesisTime+head.Header.Slot*12)
		age      = time.Duration(s.unixNano() - slotTime)
	)
	if s.enforceTime && age < 0 {
		return false, age
	}
	committee := s.getSyncCommittee(types.PeriodOfSlot(head.SignatureSlot)) // signed with the next slot's committee
	if committee == nil {
		return false, age
	}
	return s.sigVerifier.verifySignature(committee, s.forks.signingRoot(head.Header), head.BitMask, head.Signature), age
}

// SubscribeToNewHeads subscribes the given callback function to head beacon headers with a verified valid sync committee signature.
func (s *CommitteeTracker) SubscribeToNewHeads(subFn func(types.Header)) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.headSubs = append(s.headSubs, subFn)
}

// headInfo contains the best signed header and the state of propagation belonging
// to a given block root
type headInfo struct {
	head         SignedHead
	hash         common.Hash
	signerCount  int
	receivedFrom map[ctServer]struct{}
	sentTo       map[ctClient]struct{}
}

// headList is a list of best known heads for the few most recent slots
// Note: usually only the highest slot is interesting but in case of low signer
// participation or slow propagation/aggregation of signatures it might make
// sense to keep track of multiple heads as different clients might have
// different tradeoff preferences between delay and security.
type headList struct {
	list  []*headInfo // highest slot first
	limit int
}

// newHeadList creates a new headList
func newHeadList(limit int) headList {
	return headList{limit: limit}
}

// getHead returns the headInfo belonging to the given block root
func (h *headList) getHead(hash common.Hash) *headInfo {
	//return h.hashMap[hash]
	for _, headInfo := range h.list {
		if headInfo.hash == hash {
			return headInfo
		}
	}
	return nil
}

// updateHead adds or updates the given headInfo in the list
func (h *headList) updateHead(head *headInfo) {
	for i, hh := range h.list {
		if hh.head.Header.Slot <= head.head.Header.Slot {
			if hh.head.Header.Slot < head.head.Header.Slot {
				if len(h.list) < h.limit {
					h.list = append(h.list, nil)
				}
				copy(h.list[i+1:len(h.list)], h.list[i:len(h.list)-1])
			}
			h.list[i] = head
			return
		}
	}
	if len(h.list) < h.limit {
		h.list = append(h.list, head)
	}
}
