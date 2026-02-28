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
	"reflect"
	"testing"

	"github.com/ethereum/go-ethereum/beacon/light"
	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/types"
)

type requestWithID struct {
	sid     request.ServerAndID
	request request.Request
}

type TestScheduler struct {
	t         *testing.T
	module    request.Module
	events    []request.Event
	servers   []request.Server
	allowance map[request.Server]int
	sent      map[int][]requestWithID
	testIndex int
	expFail   map[request.Server]int // expected Server.Fail calls during next Run
	lastId    request.ID
}

func NewTestScheduler(t *testing.T, module request.Module) *TestScheduler {
	return &TestScheduler{
		t:         t,
		module:    module,
		allowance: make(map[request.Server]int),
		expFail:   make(map[request.Server]int),
		sent:      make(map[int][]requestWithID),
	}
}

func (ts *TestScheduler) Run(testIndex int, exp ...any) {
	expReqs := make([]requestWithID, len(exp)/2)
	id := ts.lastId
	for i := range expReqs {
		id++
		expReqs[i] = requestWithID{
			sid:     request.ServerAndID{Server: exp[i*2].(request.Server), ID: id},
			request: exp[i*2+1].(request.Request),
		}
	}
	if len(expReqs) == 0 {
		expReqs = nil
	}

	ts.testIndex = testIndex
	ts.module.Process(ts, ts.events)
	ts.events = nil

	for server, count := range ts.expFail {
		delete(ts.expFail, server)
		if count == 0 {
			continue
		}
		ts.t.Errorf("Missing %d Server.Fail(s) from server %s in test case #%d", count, server.Name(), testIndex)
	}

	if !reflect.DeepEqual(ts.sent[testIndex], expReqs) {
		ts.t.Errorf("Wrong sent requests in test case #%d (expected %v, got %v)", testIndex, expReqs, ts.sent[testIndex])
	}
}

func (ts *TestScheduler) CanSendTo() (cs []request.Server) {
	for _, server := range ts.servers {
		if ts.allowance[server] > 0 {
			cs = append(cs, server)
		}
	}
	return
}

func (ts *TestScheduler) Send(server request.Server, req request.Request) request.ID {
	ts.lastId++
	ts.sent[ts.testIndex] = append(ts.sent[ts.testIndex], requestWithID{
		sid:     request.ServerAndID{Server: server, ID: ts.lastId},
		request: req,
	})
	ts.allowance[server]--
	return ts.lastId
}

func (ts *TestScheduler) Fail(server request.Server, desc string) {
	if ts.expFail[server] == 0 {
		ts.t.Errorf("Unexpected Fail from server %s in test case #%d: %s", server.Name(), ts.testIndex, desc)
		return
	}
	ts.expFail[server]--
}

func (ts *TestScheduler) Request(testIndex, reqIndex int) requestWithID {
	if len(ts.sent[testIndex]) < reqIndex {
		ts.t.Errorf("Missing request from test case %d index %d", testIndex, reqIndex)
		return requestWithID{}
	}
	return ts.sent[testIndex][reqIndex-1]
}

func (ts *TestScheduler) ServerEvent(evType *request.EventType, server request.Server, data any) {
	ts.events = append(ts.events, request.Event{
		Type:   evType,
		Server: server,
		Data:   data,
	})
}

func (ts *TestScheduler) RequestEvent(evType *request.EventType, req requestWithID, resp request.Response) {
	if req.request == nil {
		return
	}
	ts.events = append(ts.events, request.Event{
		Type:   evType,
		Server: req.sid.Server,
		Data: request.RequestResponse{
			ID:       req.sid.ID,
			Request:  req.request,
			Response: resp,
		},
	})
}

func (ts *TestScheduler) AddServer(server request.Server, allowance int) {
	ts.servers = append(ts.servers, server)
	ts.allowance[server] = allowance
	ts.ServerEvent(request.EvRegistered, server, nil)
}

func (ts *TestScheduler) RemoveServer(server request.Server) {
	ts.servers = append(ts.servers, server)
	for i, s := range ts.servers {
		if s == server {
			copy(ts.servers[i:len(ts.servers)-1], ts.servers[i+1:])
			ts.servers = ts.servers[:len(ts.servers)-1]
			break
		}
	}
	delete(ts.allowance, server)
	ts.ServerEvent(request.EvUnregistered, server, nil)
}

func (ts *TestScheduler) AddAllowance(server request.Server, allowance int) {
	ts.allowance[server] += allowance
}

func (ts *TestScheduler) ExpFail(server request.Server) {
	ts.expFail[server]++
}

type TestCommitteeChain struct {
	fsp, nsp uint64
	init     bool
}

func (tc *TestCommitteeChain) CheckpointInit(bootstrap types.BootstrapData) error {
	tc.fsp, tc.nsp, tc.init = bootstrap.Header.SyncPeriod(), bootstrap.Header.SyncPeriod()+2, true
	return nil
}

func (tc *TestCommitteeChain) InsertUpdate(update *types.LightClientUpdate, nextCommittee *types.SerializedSyncCommittee) error {
	period := update.AttestedHeader.Header.SyncPeriod()
	if period < tc.fsp || period > tc.nsp || !tc.init {
		return light.ErrInvalidPeriod
	}
	if period == tc.nsp {
		tc.nsp++
	}
	return nil
}

func (tc *TestCommitteeChain) NextSyncPeriod() (uint64, bool) {
	return tc.nsp, tc.init
}

func (tc *TestCommitteeChain) ExpInit(t *testing.T, ExpInit bool) {
	if tc.init != ExpInit {
		t.Errorf("Incorrect init flag (expected %v, got %v)", ExpInit, tc.init)
	}
}

func (tc *TestCommitteeChain) SetNextSyncPeriod(nsp uint64) {
	tc.init, tc.nsp = true, nsp
}

func (tc *TestCommitteeChain) ExpNextSyncPeriod(t *testing.T, expNsp uint64) {
	tc.ExpInit(t, true)
	if tc.nsp != expNsp {
		t.Errorf("Incorrect NextSyncPeriod (expected %d, got %d)", expNsp, tc.nsp)
	}
}

type TestHeadTracker struct {
	phead     types.HeadInfo
	validated []types.OptimisticUpdate
	finality  types.FinalityUpdate
}

func (ht *TestHeadTracker) ValidateOptimistic(update types.OptimisticUpdate) (bool, error) {
	ht.validated = append(ht.validated, update)
	return true, nil
}

func (ht *TestHeadTracker) ValidateFinality(update types.FinalityUpdate) (bool, error) {
	ht.finality = update
	return true, nil
}

func (ht *TestHeadTracker) ValidatedFinality() (types.FinalityUpdate, bool) {
	return ht.finality, ht.finality.Attested.Header != (types.Header{})
}

func (ht *TestHeadTracker) ExpValidated(t *testing.T, tci int, expHeads []types.OptimisticUpdate) {
	for i, expHead := range expHeads {
		if i >= len(ht.validated) {
			t.Errorf("Missing validated head in test case #%d index #%d (expected {slot %d blockRoot %x}, got none)", tci, i, expHead.Attested.Header.Slot, expHead.Attested.Header.Hash())
			continue
		}
		if !reflect.DeepEqual(ht.validated[i], expHead) {
			vhead := ht.validated[i].Attested.Header
			t.Errorf("Wrong validated head in test case #%d index #%d (expected {slot %d blockRoot %x}, got {slot %d blockRoot %x})", tci, i, expHead.Attested.Header.Slot, expHead.Attested.Header.Hash(), vhead.Slot, vhead.Hash())
		}
	}
	for i := len(expHeads); i < len(ht.validated); i++ {
		vhead := ht.validated[i].Attested.Header
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
