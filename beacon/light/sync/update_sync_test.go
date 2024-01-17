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

	// expect bootstrap request to server 1
	ts.Run(1, testServer1, ReqCheckpointData(checkpointHash))

	// server 1 times out; expect request to server 2
	ts.RequestEvent(request.EvTimeout, 1, nil)
	ts.Run(2, testServer2, ReqCheckpointData(checkpointHash))

	// invalid response from server 2; expect init state to still be false
	ts.RequestEvent(request.EvResponse, 2, &types.BootstrapData{Header: types.Header{Slot: 123456}})
	ts.ExpFail(testServer2)
	ts.Run(3, nil, nil)
	chain.ExpInit(t, false)

	// server 1 fails (hard timeout)
	ts.RequestEvent(request.EvFail, 1, nil)
	ts.Run(4, nil, nil)
	chain.ExpInit(t, false)

	// server 3 is registered; expect bootstrap request to server 3
	ts.AddServer(testServer3, 1)
	ts.Run(5, testServer3, ReqCheckpointData(checkpointHash))

	// valid response from server 3; expect chain to be initialized
	ts.RequestEvent(request.EvResponse, 5, checkpoint)
	ts.Run(6, nil, nil)
	chain.ExpInit(t, true)
}

func TestUpdateSyncParallel(t *testing.T) {
	chain := &TestCommitteeChain{}
	chain.SetNextSyncPeriod(0)
	updateSync := NewForwardUpdateSync(chain)
	ts := NewTestScheduler(t, updateSync)
	// add 2 servers, head at period 100; allow 3-3 parallel requests for each
	ts.AddServer(testServer1, 3)
	ts.ServerEvent(EvNewSignedHead, testServer1, types.SignedHeader{SignatureSlot: 0x2000*100 + 0x1000})
	ts.AddServer(testServer2, 3)
	ts.ServerEvent(EvNewSignedHead, testServer2, types.SignedHeader{SignatureSlot: 0x2000*100 + 0x1000})

	// expect 6 requests to be sent
	ts.Run(1, testServer1, ReqUpdates{FirstPeriod: 0, Count: 8})
	ts.Run(2, testServer1, ReqUpdates{FirstPeriod: 8, Count: 8})
	ts.Run(3, testServer1, ReqUpdates{FirstPeriod: 16, Count: 8})
	ts.Run(4, testServer2, ReqUpdates{FirstPeriod: 24, Count: 8})
	ts.Run(5, testServer2, ReqUpdates{FirstPeriod: 32, Count: 8})
	ts.Run(6, testServer2, ReqUpdates{FirstPeriod: 40, Count: 8})

	// valid response to request 1; expect 8 periods synced and a new request started
	ts.RequestEvent(request.EvResponse, 1, testRespUpdate(ts.Request(1)))
	ts.AddAllowance(testServer1, 1)
	ts.Run(7, testServer1, ReqUpdates{FirstPeriod: 48, Count: 8})
	chain.ExpNextSyncPeriod(t, 8)

	// valid response to requests 4 and 5
	ts.RequestEvent(request.EvResponse, 4, testRespUpdate(ts.Request(4)))
	ts.RequestEvent(request.EvResponse, 5, testRespUpdate(ts.Request(5)))
	ts.AddAllowance(testServer2, 2)
	// expect 2 more requests but no sync progress (responses 4 and 5 cannot be added before 2 and 3)
	ts.Run(8, testServer2, ReqUpdates{FirstPeriod: 56, Count: 8})
	ts.Run(9, testServer2, ReqUpdates{FirstPeriod: 64, Count: 8})
	chain.ExpNextSyncPeriod(t, 8)

	// soft timeout for requests 2 and 3 (server 1 is overloaded)
	ts.RequestEvent(request.EvTimeout, 2, nil)
	ts.RequestEvent(request.EvTimeout, 3, nil)
	// no allowance, no more requests
	ts.Run(10, nil, nil)

	// valid response to requests 6 and 8 and 9
	ts.RequestEvent(request.EvResponse, 6, testRespUpdate(ts.Request(6)))
	ts.RequestEvent(request.EvResponse, 8, testRespUpdate(ts.Request(8)))
	ts.RequestEvent(request.EvResponse, 9, testRespUpdate(ts.Request(9)))
	ts.AddAllowance(testServer2, 3)
	// server 2 can now resend requests 2 and 3 (timed out by server 1) and also send a new one
	ts.Run(11, testServer2, ReqUpdates{FirstPeriod: 8, Count: 8})
	ts.Run(12, testServer2, ReqUpdates{FirstPeriod: 16, Count: 8})
	ts.Run(13, testServer2, ReqUpdates{FirstPeriod: 72, Count: 8})

	// server 1 finally answers timed out request 2
	ts.RequestEvent(request.EvResponse, 2, testRespUpdate(ts.Request(2)))
	ts.AddAllowance(testServer1, 1)
	// expect sync progress and one new request
	ts.Run(14, testServer1, ReqUpdates{FirstPeriod: 80, Count: 8})
	chain.ExpNextSyncPeriod(t, 16)

	// server 2 answers requests 11 and 12 (resends of requests 2 and 3)
	ts.RequestEvent(request.EvResponse, 11, testRespUpdate(ts.Request(11)))
	ts.RequestEvent(request.EvResponse, 12, testRespUpdate(ts.Request(12)))
	ts.AddAllowance(testServer2, 2)
	ts.Run(15, testServer2, ReqUpdates{FirstPeriod: 88, Count: 8})
	ts.Run(16, testServer2, ReqUpdates{FirstPeriod: 96, Count: 4})
	// finally the gap is filled, update can process responses up to req6
	chain.ExpNextSyncPeriod(t, 48)

	// all remaining requests are answered
	ts.RequestEvent(request.EvResponse, 3, testRespUpdate(ts.Request(3)))
	ts.RequestEvent(request.EvResponse, 7, testRespUpdate(ts.Request(7)))
	ts.RequestEvent(request.EvResponse, 13, testRespUpdate(ts.Request(13)))
	ts.RequestEvent(request.EvResponse, 14, testRespUpdate(ts.Request(14)))
	ts.RequestEvent(request.EvResponse, 15, testRespUpdate(ts.Request(15)))
	ts.RequestEvent(request.EvResponse, 16, testRespUpdate(ts.Request(16)))
	ts.Run(17, nil, nil)
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
	ts.ServerEvent(EvNewSignedHead, testServer1, types.SignedHeader{SignatureSlot: 0x2000*15 + 0x1000})
	ts.AddServer(testServer2, 1)
	ts.ServerEvent(EvNewSignedHead, testServer2, types.SignedHeader{SignatureSlot: 0x2000*16 + 0x1000})
	ts.AddServer(testServer3, 1)
	ts.ServerEvent(EvNewSignedHead, testServer3, types.SignedHeader{SignatureSlot: 0x2000*17 + 0x1000})

	// expect request to the best announced head
	ts.Run(1, testServer3, ReqUpdates{FirstPeriod: 10, Count: 7})

	// request times out, expect request to the next best head
	ts.RequestEvent(request.EvTimeout, 1, nil)
	ts.Run(2, testServer2, ReqUpdates{FirstPeriod: 10, Count: 6})

	// request times out, expect request to the last available server
	ts.RequestEvent(request.EvTimeout, 2, nil)
	ts.Run(3, testServer1, ReqUpdates{FirstPeriod: 10, Count: 5})

	// valid response to request 3, expect chain synced to period 15
	ts.RequestEvent(request.EvResponse, 3, testRespUpdate(ts.Request(3)))
	ts.AddAllowance(testServer1, 1)
	ts.Run(4, nil, nil)
	chain.ExpNextSyncPeriod(t, 15)

	// invalid response to request 1, server can only deliver updates up to period 15 despite announced head
	req1x := ts.Request(1)
	req1x.Request = ReqUpdates{FirstPeriod: 10, Count: 5}
	ts.RequestEvent(request.EvResponse, 1, testRespUpdate(req1x))
	ts.ExpFail(testServer3)
	ts.Run(5, nil, nil)
	// expect no progress of chain head
	chain.ExpNextSyncPeriod(t, 15)

	// valid response to request 2, expect chain synced to period 16
	ts.RequestEvent(request.EvResponse, 2, testRespUpdate(ts.Request(2)))
	ts.AddAllowance(testServer2, 1)
	ts.Run(6, nil, nil)
	chain.ExpNextSyncPeriod(t, 16)

	// a new server is registered with announced head period 17
	ts.AddServer(testServer4, 1)
	ts.ServerEvent(EvNewSignedHead, testServer4, types.SignedHeader{SignatureSlot: 0x2000*17 + 0x1000})
	// expect request to sync one more period
	ts.Run(7, testServer4, ReqUpdates{FirstPeriod: 16, Count: 1})

	// valid response, expect chain synced to period 17
	ts.RequestEvent(request.EvResponse, 7, testRespUpdate(ts.Request(7)))
	ts.AddAllowance(testServer4, 1)
	ts.Run(8, nil, nil)
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
