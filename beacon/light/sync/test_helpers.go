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

type TestServer struct {
	ts *TestScheduler
	ID int
}

func (s *TestServer) Fail(desc string) {
	s.ts.serverFail(s)
}

type TestScheduler struct {
	t         *testing.T
	module    request.Module
	events    []request.Event
	servers   []request.Server
	allowance map[request.Server]int
	sent      map[int]request.RequestWithID
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
		sent:      make(map[int]request.RequestWithID),
	}
}

func (ts *TestScheduler) Run(testIndex int, expServer request.Server, expReq request.Request) {
	ts.testIndex = testIndex
	ts.module.Process(ts.events)
	ts.events = nil

	for server, count := range ts.expFail {
		delete(ts.expFail, server)
		if count == 0 {
			continue
		}
		ts.t.Errorf("Missing %d Server.Fail(s) from server %d in test case #%d", count, server.(*TestServer).ID, testIndex)
	}

	expReqWithID := request.RequestWithID{
		ServerAndID: request.ServerAndID{Server: expServer, ID: ts.lastId + 1},
		Request:     expReq,
	}
	req, ok := ts.tryRequest(testIndex, ts.module.MakeRequest)
	if expReq == nil {
		if ok {
			ts.t.Errorf("Unexpected request in test case #%d (expected none, got %v)", testIndex, req)
		}
		return
	}
	if !ok {
		ts.t.Errorf("Missing request in test case #%d (expected %v, got none)", testIndex, expReqWithID)
		return
	}
	if req != expReqWithID {
		ts.t.Errorf("Wrong request in test case #%d (expected %v, got %v)", testIndex, expReqWithID, req)
	}
}

func (ts *TestScheduler) Request(testIndex int) request.RequestWithID {
	return ts.sent[testIndex]
}

func (ts *TestScheduler) ServerEvent(evType *request.EventType, server request.Server, data any) {
	ts.events = append(ts.events, request.Event{
		Type:   evType,
		Server: server,
		Data:   data,
	})
}

func (ts *TestScheduler) RequestEvent(evType *request.EventType, testIndex int, resp request.Response) {
	req, ok := ts.sent[testIndex]
	if !ok {
		ts.t.Errorf("Missing request from test case %v", testIndex)
		return
	}
	ts.events = append(ts.events, request.Event{
		Type:   evType,
		Server: req.ServerAndID.Server,
		Data: request.RequestResponse{
			ID:       req.ServerAndID.ID,
			Request:  req.Request,
			Response: resp,
		},
	})
}

func (ts *TestScheduler) AddServer(server request.Server, allowance int) {
	server.(*TestServer).ts = ts
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

func (ts *TestScheduler) serverFail(server request.Server) {
	if ts.expFail[server] == 0 {
		ts.t.Errorf("Unexpected Server.Fail from server %d in test case #%d", server.(*TestServer).ID, ts.testIndex)
		return
	}
	ts.expFail[server]--
}

func (ts *TestScheduler) tryRequest(testIndex int, requestFn func(server request.Server) (request.Request, float32)) (request.RequestWithID, bool) {
	var (
		bestServer request.Server
		bestReq    request.Request
		bestPri    float32
	)
	for _, server := range ts.servers {
		if ts.allowance[server] == 0 {
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
	ts.allowance[bestServer]--
	ts.lastId++
	req := request.RequestWithID{
		ServerAndID: request.ServerAndID{Server: bestServer, ID: ts.lastId},
		Request:     bestReq,
	}
	ts.sent[testIndex] = req
	ts.RequestEvent(request.EvRequest, testIndex, nil)
	return req, true
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
