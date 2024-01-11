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

package main

import (
	"testing"

	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/tree"
)

var (
	testServer1 = 1
	testServer2 = 2

	testBlock1 = &capella.BeaconBlock{Slot: 123}
	testBlock2 = &capella.BeaconBlock{Slot: 124}
)

func TestBlockSync(t *testing.T) {
	tracker := &sync.TestTracker{}
	tracker.AddServer(testServer1, 1)
	tracker.AddServer(testServer2, 1)
	ht := &testHeadTracker{}
	blockSync := newBeaconBlockSync(ht)

	expHeadBlock := func(tci int, expHead *capella.BeaconBlock) {
		expInfo := blockHeadInfo(expHead)
		headInfo := blockHeadInfo(blockSync.getHeadBlock())
		if headInfo != expInfo {
			t.Errorf("Wrong head block in test case #%d (expected {slot %d blockRoot %x}, got {slot %d blockRoot %x})", tci, expInfo.Slot, expInfo.BlockRoot, headInfo.Slot, headInfo.BlockRoot)
		}
	}

	blockSync.Process(tracker, []request.Event{
		{Server: testServer1, Type: request.EvRegistered},
		{Server: testServer2, Type: request.EvRegistered},
	})
	// no block requests expected until head tracker knows about a head
	tracker.ExpRequests(t, 1, nil)
	expHeadBlock(1, nil)
	// set block 1 as prefetch head, announced by server 2
	head1 := blockHeadInfo(testBlock1)
	ht.prefetch = head1
	blockSync.Process(tracker, []request.Event{
		{Server: testServer2, Type: sync.EvNewHead, Data: head1},
	})
	// expect request to server 2 which has announced the head
	req1 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 1}, Request: sync.ReqBeaconBlock(head1.BlockRoot)}
	tracker.ExpRequests(t, 2, []request.RequestWithID{req1})
	// valid response
	tracker.AddAllowance(testServer2, 1)
	blockSync.Process(tracker, []request.Event{
		sync.TestReqEvent(request.EvResponse, req1, testBlock1),
	})
	// head block still not expected as the fetched block is not the validated head yet
	expHeadBlock(2, nil)
	// set as validated head, expect no further requests but block 1 set as head block
	ht.validated.Header = blockHeader(testBlock1)
	blockSync.Process(tracker, nil)
	tracker.ExpRequests(t, 3, nil)
	expHeadBlock(3, testBlock1)

	// set block 2 as prefetch head, announced by server 1
	head2 := blockHeadInfo(testBlock2)
	ht.prefetch = head2
	blockSync.Process(tracker, []request.Event{
		{Server: testServer1, Type: sync.EvNewHead, Data: head2},
	})
	// expect request to server 1
	req2 := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer1, ID: 2}, Request: sync.ReqBeaconBlock(head2.BlockRoot)}
	tracker.ExpRequests(t, 4, []request.RequestWithID{req2})
	// req2 fails, no further requests expected because server 2 has not announced it
	blockSync.Process(tracker, []request.Event{
		sync.TestReqEvent(request.EvFail, req2, nil),
	})
	tracker.ExpRequests(t, 5, nil)
	// set as validated head before retrieving block; now it's assumed to be available from server 2 too
	ht.validated.Header = blockHeader(testBlock2)
	blockSync.Process(tracker, nil)
	// now head block is unavailable again
	expHeadBlock(4, nil)
	// expect req2 retry to server 2
	req2r := request.RequestWithID{ServerAndID: request.ServerAndID{Server: testServer2, ID: 3}, Request: sync.ReqBeaconBlock(head2.BlockRoot)}
	tracker.ExpRequests(t, 6, []request.RequestWithID{req2r})
	// valid response, now head block should be block 2 immediately as it is already validated
	blockSync.Process(tracker, []request.Event{
		sync.TestReqEvent(request.EvResponse, req2r, testBlock2),
	})
	expHeadBlock(5, testBlock2)
}

func blockHeadInfo(block *capella.BeaconBlock) types.HeadInfo {
	if block == nil {
		return types.HeadInfo{}
	}
	return types.HeadInfo{Slot: uint64(block.Slot), BlockRoot: beaconBlockHash(block)}
}

func blockHeader(block *capella.BeaconBlock) types.Header {
	return types.Header{
		Slot:          uint64(block.Slot),
		ProposerIndex: uint64(block.ProposerIndex),
		ParentRoot:    common.Hash(block.ParentRoot),
		StateRoot:     common.Hash(block.StateRoot),
		BodyRoot:      common.Hash(block.Body.HashTreeRoot(configs.Mainnet, tree.GetHashFn())),
	}
}

type testHeadTracker struct {
	prefetch  types.HeadInfo
	validated types.SignedHeader
}

func (h *testHeadTracker) PrefetchHead() types.HeadInfo {
	return h.prefetch
}

func (h *testHeadTracker) ValidatedHead() types.SignedHeader {
	return h.validated
}
