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

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/types"
)

type TestTracker struct {
	servers   []request.Server
	allowance map[request.Server]int
	sent      []request.RequestWithID
	lastId    request.ID
}

func (tt *TestTracker) AddServer(server request.Server, allowance int) {
	tt.servers = append(tt.servers, server)
	if tt.allowance == nil {
		tt.allowance = make(map[request.Server]int)
	}
	tt.allowance[server] = allowance
}

func (tt *TestTracker) AddAllowance(server request.Server, allowance int) {
	tt.allowance[server] += allowance
}

func (tt *TestTracker) TryRequest(requestFn func(server request.Server) (request.Request, float32)) (request.RequestWithID, bool) {
	var (
		bestServer request.Server
		bestReq    request.Request
		bestPri    float32
	)
	for _, server := range tt.servers {
		if tt.allowance[server] == 0 {
			continue
		}
		req, pri := requestFn(server)
		if req != nil && (bestReq == nil || pri > bestPri) {
			bestServer, bestReq, bestPri = server, req, pri
		}
	}
	if bestServer == nil {
		return request.RequestWithID{}, false
	}
	tt.allowance[bestServer]--
	tt.lastId++
	req := request.RequestWithID{
		ServerAndID: request.ServerAndID{Server: bestServer, ID: tt.lastId},
		Request:     bestReq,
	}
	tt.sent = append(tt.sent, req)
	return req, true
}

func (tt *TestTracker) ExpRequests(t *testing.T, tci int, expSent []request.RequestWithID) {
	for i, expReq := range expSent {
		if i >= len(tt.sent) {
			t.Errorf("Missing sent request in test case #%d index #%d (expected %v, got none)", tci, i, expReq)
			continue
		}
		if tt.sent[i] != expReq {
			t.Errorf("Wrong sent request in test case #%d index #%d (expected %v, got %v)", tci, i, expReq, tt.sent[i])
		}
	}
	for i := len(expSent); i < len(tt.sent); i++ {
		t.Errorf("Unexpected sent request in test case #%d index #%d (expected none, got %v)", tci, i, tt.sent[i])
	}
	tt.sent = nil
}

func (tt *TestTracker) InvalidResponse(id request.ServerAndID, desc string) {
	return
}

func ExpTrigger(t *testing.T, tci int, expTrigger, trigger bool) {
	if trigger != expTrigger {
		t.Errorf("Invalid process trigger output in test case #%d (expected %v, got %v)", tci, expTrigger, trigger)
	}
}

type TestCommitteeChain struct {
	fsp, nsp uint64
	init     bool
}

func (t *TestCommitteeChain) CheckpointInit(bootstrap types.BootstrapData) error {
	t.fsp, t.nsp, t.init = bootstrap.Header.SyncPeriod(), bootstrap.Header.SyncPeriod()+2, true
	return nil
}

func (t *TestCommitteeChain) InsertUpdate(update *types.LightClientUpdate, nextCommittee *types.SerializedSyncCommittee) error {
	period := update.AttestedHeader.Header.SyncPeriod()
	if period < t.fsp || period > t.nsp || !t.init {
		return light.ErrInvalidPeriod
	}
	if period == t.nsp {
		t.nsp++
	}
	return nil
}

func (t *TestCommitteeChain) NextSyncPeriod() (uint64, bool) {
	return t.nsp, t.init
}

func (tc *TestCommitteeChain) ExpInit(t *testing.T, ExpInit bool) {
	if tc.init != ExpInit {
		t.Errorf("Incorrect init flag (expected %v, got %v)", ExpInit, tc.init)
	}
}

func (t *TestCommitteeChain) SetNextSyncPeriod(nsp uint64) {
	t.init, t.nsp = true, nsp
}

func (tc *TestCommitteeChain) ExpNextSyncPeriod(t *testing.T, expNsp uint64) {
	tc.ExpInit(t, true)
	if tc.nsp != expNsp {
		t.Errorf("Incorrect NextSyncPeriod (expected %d, got %d)", expNsp, tc.nsp)
	}
}

type TestHeadTracker struct {
	phead     types.HeadInfo
	validated []types.SignedHeader
}

func (ht *TestHeadTracker) Validate(head types.SignedHeader) (bool, error) {
	ht.validated = append(ht.validated, head)
	return true, nil
}

func (ht *TestHeadTracker) ExpValidated(t *testing.T, tci int, expHeads []types.SignedHeader) {
	for i, expHead := range expHeads {
		if i >= len(ht.validated) {
			t.Errorf("Missing validated head in test case #%d index #%d (expected {slot %d blockRoot %x}, got none)", tci, i, expHead.Header.Slot, expHead.Header.Hash())
			continue
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

func (ht *TestHeadTracker) SetPrefetchHead(head types.HeadInfo) {
	ht.phead = head
}

func (ht *TestHeadTracker) ExpPrefetch(t *testing.T, tci int, exp types.HeadInfo) {
	if ht.phead != exp {
		t.Errorf("Wrong prefetch head in test case #%d (expected {slot %d blockRoot %x}, got {slot %d blockRoot %x})", tci, exp.Slot, exp.BlockRoot, ht.phead.Slot, ht.phead.BlockRoot)
	}
}
