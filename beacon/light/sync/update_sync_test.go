// Copyright 2024 The go-ethereum Authors
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

func TestCheckpointInit(t *testing.T) {
	chain := &TestCommitteeChain{}
	checkpoint := &types.BootstrapData{Header: types.Header{Slot: 0x2000*4 + 0x1000}} // period 4
	checkpointHash := checkpoint.Header.Hash()
	chkInit := NewCheckpointInit(chain, checkpointHash)
	ts := NewTestScheduler(t, chkInit)
	// add 2 servers
	ts.AddServer(testServer1, 1)
	ts.AddServer(testServer2, 1)

	chkInit.Process([]request.Event{
		{Server: testServer1, Type: request.EvRegistered},
		{Server: testServer2, Type: request.EvRegistered},
	})
	// expect bootstrap request to server 1
	req1 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer1, ID: 1}, Request: ReqCheckpointData(checkpointHash)}
	ts.ExpRequests(t, 1, []request.RequestWithID{req1})
	// req1 times out; expect request to server 2
	chkInit.Process([]request.Event{
		TestReqEvent(request.EvTimeout, req1, nil),
	})
	req2 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 2}, Request: ReqCheckpointData(checkpointHash)}
	ts.ExpRequests(t, 2, []request.RequestWithID{req2})
	// invalid response to req2; expect init state to still be false
	wrongCheckpoint := &types.BootstrapData{Header: types.Header{Slot: 123456}}
	chkInit.Process([]request.Event{
		TestReqEvent(request.EvResponse, req2, wrongCheckpoint),
	})
	// req1 fails (hard timeout)
	chkInit.Process([]request.Event{
		TestReqEvent(request.EvFail, req1, nil),
	})
	chain.ExpInit(t, false)
	// server 3 is registered
	ts.AddServer(testServer3, 1)
	chkInit.Process([]request.Event{
		{Server: testServer3, Type: request.EvRegistered},
	})
	// expect bootstrap request to server 3
	req3 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer3, ID: 3}, Request: ReqCheckpointData(checkpointHash)}
	ts.ExpRequests(t, 3, []request.RequestWithID{req3})
	// valid response to req3; expect chain to be initialized
	chkInit.Process([]request.Event{
		TestReqEvent(request.EvResponse, req3, checkpoint),
	})
	chain.ExpInit(t, true)
}

func TestUpdateSyncParallel(t *testing.T) {
	chain := &TestCommitteeChain{}
	chain.SetNextSyncPeriod(0)
	updateSync := NewForwardUpdateSync(chain)
	ts := NewTestScheduler(t, updateSync)
	// add 2 servers, head at period 100; allow 3-3 parallel requests for each
	ts.AddServer(testServer1, 3)
	ts.AddServer(testServer2, 3)

	updateSync.Process([]request.Event{
		{Server: testServer1, Type: request.EvRegistered},
		{Server: testServer1, Type: EvNewSignedHead, Data: types.SignedHeader{SignatureSlot: 0x2000*100 + 0x1000}},
		{Server: testServer2, Type: request.EvRegistered},
		{Server: testServer2, Type: EvNewSignedHead, Data: types.SignedHeader{SignatureSlot: 0x2000*100 + 0x1000}},
	})
	// expect 6 requests to be sent
	req1 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer1, ID: 1}, Request: ReqUpdates{FirstPeriod: 0, Count: 8}}
	req2 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer1, ID: 2}, Request: ReqUpdates{FirstPeriod: 8, Count: 8}}
	req3 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer1, ID: 3}, Request: ReqUpdates{FirstPeriod: 16, Count: 8}}
	req4 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 4}, Request: ReqUpdates{FirstPeriod: 24, Count: 8}}
	req5 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 5}, Request: ReqUpdates{FirstPeriod: 32, Count: 8}}
	req6 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 6}, Request: ReqUpdates{FirstPeriod: 40, Count: 8}}
	ts.ExpRequests(t, 1, []request.RequestWithID{req1, req2, req3, req4, req5, req6})
	// valid response to request 1
	ts.AddAllowance(testServer1, 1)
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvResponse, req1, testRespUpdate(req1)),
	})
	// expect 8 periods synced and a new request started
	chain.ExpNextSyncPeriod(t, 8)
	req7 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer1, ID: 7}, Request: ReqUpdates{FirstPeriod: 48, Count: 8}}
	ts.ExpRequests(t, 2, []request.RequestWithID{req7})
	// valid response to requests 4 and 5
	ts.AddAllowance(testServer2, 2)
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvResponse, req4, testRespUpdate(req4)),
		TestReqEvent(request.EvResponse, req5, testRespUpdate(req5)),
	})
	// expect 2 more requests but no sync progress (responses 4 and 5 cannot be added before 2 and 3)
	chain.ExpNextSyncPeriod(t, 8)
	req8 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 8}, Request: ReqUpdates{FirstPeriod: 56, Count: 8}}
	req9 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 9}, Request: ReqUpdates{FirstPeriod: 64, Count: 8}}
	ts.ExpRequests(t, 3, []request.RequestWithID{req8, req9})
	// soft timeout for requests 2 and 3 (server 1 is overloaded)
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvTimeout, req2, nil),
		TestReqEvent(request.EvTimeout, req3, nil),
	})
	// no allowance, no more requests
	ts.ExpRequests(t, 4, nil)
	// valid response to requests 6 and 8 and 9
	ts.AddAllowance(testServer2, 3)
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvResponse, req6, testRespUpdate(req6)),
		TestReqEvent(request.EvResponse, req8, testRespUpdate(req8)),
		TestReqEvent(request.EvResponse, req9, testRespUpdate(req9)),
	})
	// server 2 can now resend requests 2 and 3 (timed out by server 1) and also send a new one
	req2r := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 10}, Request: ReqUpdates{FirstPeriod: 8, Count: 8}}
	req3r := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 11}, Request: ReqUpdates{FirstPeriod: 16, Count: 8}}
	req10 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 12}, Request: ReqUpdates{FirstPeriod: 72, Count: 8}}
	ts.ExpRequests(t, 5, []request.RequestWithID{req2r, req3r, req10})
	// server 1 finally answers timed out request 2
	ts.AddAllowance(testServer1, 1)
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvResponse, req2, testRespUpdate(req2)),
	})
	// expect sync progress and one new request
	chain.ExpNextSyncPeriod(t, 16)
	req11 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer1, ID: 13}, Request: ReqUpdates{FirstPeriod: 80, Count: 8}}
	ts.ExpRequests(t, 6, []request.RequestWithID{req11})
	// server 2 answers re-sent requests 2 and 3
	ts.AddAllowance(testServer2, 2)
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvResponse, req2r, testRespUpdate(req2r)),
		TestReqEvent(request.EvResponse, req3r, testRespUpdate(req3r)),
	})
	// finally the gap is filled, update can process responses up to req6
	chain.ExpNextSyncPeriod(t, 48)
	// expect 2 new requests from server 2 (now the available range is covered)
	req12 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 14}, Request: ReqUpdates{FirstPeriod: 88, Count: 8}}
	req13 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 15}, Request: ReqUpdates{FirstPeriod: 96, Count: 4}}
	ts.ExpRequests(t, 7, []request.RequestWithID{req12, req13})
	// all remaining requests are answered
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvResponse, req3, testRespUpdate(req3)),
		TestReqEvent(request.EvResponse, req7, testRespUpdate(req7)),
		TestReqEvent(request.EvResponse, req10, testRespUpdate(req10)),
		TestReqEvent(request.EvResponse, req11, testRespUpdate(req11)),
		TestReqEvent(request.EvResponse, req12, testRespUpdate(req12)),
		TestReqEvent(request.EvResponse, req13, testRespUpdate(req13)),
	})
	// expect chain to be fully synced
	chain.ExpNextSyncPeriod(t, 100)
}

func TestUpdateSyncDifferentHeads(t *testing.T) {
	chain := &TestCommitteeChain{}
	chain.SetNextSyncPeriod(10)
	updateSync := NewForwardUpdateSync(chain)
	ts := NewTestScheduler(t, updateSync)
	// add 3 servers with different announced head periods
	ts.AddServer(testServer1, 1)
	ts.AddServer(testServer2, 1)
	ts.AddServer(testServer3, 1)

	updateSync.Process([]request.Event{
		{Server: testServer1, Type: request.EvRegistered},
		{Server: testServer1, Type: EvNewSignedHead, Data: types.SignedHeader{SignatureSlot: 0x2000*15 + 0x1000}},
		{Server: testServer2, Type: request.EvRegistered},
		{Server: testServer2, Type: EvNewSignedHead, Data: types.SignedHeader{SignatureSlot: 0x2000*16 + 0x1000}},
		{Server: testServer3, Type: request.EvRegistered},
		{Server: testServer3, Type: EvNewSignedHead, Data: types.SignedHeader{SignatureSlot: 0x2000*17 + 0x1000}},
	})
	// expect request to the best announced head
	req1 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer3, ID: 1}, Request: ReqUpdates{FirstPeriod: 10, Count: 7}}
	ts.ExpRequests(t, 1, []request.RequestWithID{req1})
	// request times out, expect request to the next best head
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvTimeout, req1, nil),
	})
	req2 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 2}, Request: ReqUpdates{FirstPeriod: 10, Count: 6}}
	ts.ExpRequests(t, 2, []request.RequestWithID{req2})
	// request times out, expect request to the last available server
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvTimeout, req2, nil),
	})
	req3 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer1, ID: 3}, Request: ReqUpdates{FirstPeriod: 10, Count: 5}}
	ts.ExpRequests(t, 3, []request.RequestWithID{req3})
	// valid response to request 3, expect chain synced to period 15
	ts.AddAllowance(testServer1, 1)
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvResponse, req3, testRespUpdate(req3)),
	})
	chain.ExpNextSyncPeriod(t, 15)
	// invalid response to request 1, server can only deliver updates up to period 15 despite announced head
	req1x := request.RequestWithID{ServerAndID: req1.ServerAndID, Request: ReqUpdates{FirstPeriod: 10, Count: 5}}
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvResponse, req1, testRespUpdate(req1x)),
	})
	// expect no progress of chain head
	chain.ExpNextSyncPeriod(t, 15)
	// valid response to request 2, expect chain synced to period 16
	ts.AddAllowance(testServer2, 1)
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvResponse, req2, testRespUpdate(req2)),
	})
	chain.ExpNextSyncPeriod(t, 16)
	// a new server is registered with announced head period 17
	ts.AddServer(testServer4, 1)
	updateSync.Process([]request.Event{
		{Server: testServer4, Type: request.EvRegistered},
		{Server: testServer4, Type: EvNewSignedHead, Data: types.SignedHeader{SignatureSlot: 0x2000*17 + 0x1000}},
	})
	// expect request to sync one more period
	req4 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer4, ID: 4}, Request: ReqUpdates{FirstPeriod: 16, Count: 1}}
	ts.ExpRequests(t, 4, []request.RequestWithID{req4})
	// valid response, expect chain synced to period 17
	ts.AddAllowance(testServer1, 1)
	updateSync.Process([]request.Event{
		TestReqEvent(request.EvResponse, req4, testRespUpdate(req4)),
	})
	chain.ExpNextSyncPeriod(t, 17)
}

func testRespUpdate(request request.RequestWithID) request.Response {
	var resp RespUpdates
	req := request.Request.(ReqUpdates)
	resp.Updates = make([]*types.LightClientUpdate, int(req.Count))
	resp.Committees = make([]*types.SerializedSyncCommittee, int(req.Count))
	period := req.FirstPeriod
	for i := range resp.Updates {
		resp.Updates[i] = &types.LightClientUpdate{AttestedHeader: types.SignedHeader{Header: types.Header{Slot: 0x2000*period + 0x1000}}}
		resp.Committees[i] = new(types.SerializedSyncCommittee)
		period++
	}
	return resp
}
