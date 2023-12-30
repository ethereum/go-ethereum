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

package sync

import (
	"testing"

	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/types"
)

type testTracker struct{}

func (t *testTracker) TryRequest(requestFn func(server any) (request.Request, float32)) (request.ServerAndId, request.Request) {
	return request.ServerAndId{}, nil
}

func (t *testTracker) InvalidResponse(id request.ServerAndId, desc string) {
	return
}

type testCommitteeChain struct {
	nsp  uint64
	init bool
}

func (t *testCommitteeChain) CheckpointInit(bootstrap types.BootstrapData) error {
	return nil
}

func (t *testCommitteeChain) InsertUpdate(update *types.LightClientUpdate, nextCommittee *types.SerializedSyncCommittee) error {
	return nil
}

func (t *testCommitteeChain) NextSyncPeriod() (uint64, bool) {
	return t.nsp, t.init
}

type testHeadTracker struct {
	phead     types.HeadInfo
	validated []types.SignedHeader
}

func (ht *testHeadTracker) Validate(head types.SignedHeader) (bool, error) {
	ht.validated = append(ht.validated, head)
	return true, nil
}

func (ht *testHeadTracker) expValidated(t *testing.T, tci int, expHeads []types.SignedHeader) {
	for i, expHead := range expHeads {
		if i >= len(ht.validated) {
			t.Errorf("Missing validated head in test case #%d index #%d (expected {slot %d blockRoot %x}, got none)", tci, i, expHead.Header.Slot, expHead.Header.Hash())
		}
		if ht.validated[i] != expHead {
			vhead := ht.validated[i].Header
			t.Errorf("Wrong validated head in test case #%d index #%d (expected {slot %d blockRoot %x}, got {slot %d blockRoot %x})", tci, i, expHead.Header.Slot, expHead.Header.Hash(), vhead.Slot, vhead.Hash())
		}
	}
	for i := len(expHeads); i < len(ht.validated); i++ {
		vhead := ht.validated[i].Header
		t.Errorf("Unexpected validated head in test case #%d index #%d (expected none, got {slot %d blockRoot %x})", tci, i, vhead.Slot, vhead.Hash())
	}
	ht.validated = nil
}

func (ht *testHeadTracker) SetPrefetchHead(head types.HeadInfo) {
	ht.phead = head
}

func (ht *testHeadTracker) expPrefetch(t *testing.T, tci int, exp types.HeadInfo) {
	if ht.phead != exp {
		t.Errorf("Wrong prefetch head in test case #%d (expected {slot %d blockRoot %x}, got {slot %d blockRoot %x})", tci, exp.Slot, exp.BlockRoot, ht.phead.Slot, ht.phead.BlockRoot)
	}
}
