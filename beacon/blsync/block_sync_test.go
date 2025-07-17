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

package blsync

import (
	"testing"

	"github.com/ethereum/go-ethereum/beacon/light/request"
	"github.com/ethereum/go-ethereum/beacon/light/sync"
	"github.com/ethereum/go-ethereum/beacon/types"
	"github.com/ethereum/go-ethereum/common"
	zrntcommon "github.com/protolambda/zrnt/eth2/beacon/common"
	"github.com/protolambda/zrnt/eth2/beacon/deneb"
)

var (
	testServer1 = testServer("testServer1")
	testServer2 = testServer("testServer2")

	testBlock1 = types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 123,
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 456,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("905ac721c4058d9ed40b27b6b9c1bdd10d4333e4f3d9769100bf9dfb80e5d1f6")),
			},
		},
	})
	testBlock2 = types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 124,
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 457,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("011703f39c664efc1c6cf5f49ca09b595581eec572d4dfddd3d6179a9e63e655")),
			},
		},
	})
)

type testServer string

func (t testServer) Name() string {
	return string(t)
}

func TestBlockSync(t *testing.T) {
	ht := &testHeadTracker{}
	blockSync := newBeaconBlockSync(ht)
	headCh := make(chan types.ChainHeadEvent, 16)
	blockSync.SubscribeChainHead(headCh)
	ts := sync.NewTestScheduler(t, blockSync)
	ts.AddServer(testServer1, 1)
	ts.AddServer(testServer2, 1)

	expHeadBlock := func(expHead *types.BeaconBlock) {
		t.Helper()
		var expNumber, headNumber uint64
		if expHead != nil {
			p, err := expHead.ExecutionPayload()
			if err != nil {
				t.Fatalf("expHead.ExecutionPayload() failed: %v", err)
			}
			expNumber = p.NumberU64()
		}
		select {
		case event := <-headCh:
			headNumber = event.Block.NumberU64()
		default:
		}
		if headNumber != expNumber {
			t.Errorf("Wrong head block, expected block number %d, got %d)", expNumber, headNumber)
		}
	}

	// no block requests expected until head tracker knows about a head
	ts.Run(1)
	expHeadBlock(nil)

	// set block 1 as prefetch head, announced by server 2
	head1 := blockHeadInfo(testBlock1)
	ht.prefetch = head1
	ts.ServerEvent(sync.EvNewHead, testServer2, head1)

	// expect request to server 2 which has announced the head
	ts.Run(2, testServer2, sync.ReqBeaconBlock(head1.BlockRoot))

	// valid response
	ts.RequestEvent(request.EvResponse, ts.Request(2, 1), testBlock1)
	ts.AddAllowance(testServer2, 1)
	ts.Run(3)
	// head block still not expected as the fetched block is not the validated head yet
	expHeadBlock(nil)

	// set as validated head, expect no further requests but block 1 set as head block
	ht.validated.Header = testBlock1.Header()
	ts.Run(4)
	expHeadBlock(testBlock1)

	// set block 2 as prefetch head, announced by server 1
	head2 := blockHeadInfo(testBlock2)
	ht.prefetch = head2
	ts.ServerEvent(sync.EvNewHead, testServer1, head2)
	// expect request to server 1
	ts.Run(5, testServer1, sync.ReqBeaconBlock(head2.BlockRoot))

	// req2 fails, no further requests expected because server 2 has not announced it
	ts.RequestEvent(request.EvFail, ts.Request(5, 1), nil)
	ts.Run(6)

	// set as validated head before retrieving block; now it's assumed to be available from server 2 too
	ht.validated.Header = testBlock2.Header()
	// expect req2 retry to server 2
	ts.Run(7, testServer2, sync.ReqBeaconBlock(head2.BlockRoot))
	// now head block should be unavailable again
	expHeadBlock(nil)

	// valid response, now head block should be block 2 immediately as it is already validated
	ts.RequestEvent(request.EvResponse, ts.Request(7, 1), testBlock2)
	ts.Run(8)
	expHeadBlock(testBlock2)
}

type testHeadTracker struct {
	prefetch  types.HeadInfo
	validated types.SignedHeader
	finalized types.HeaderWithExecProof
}

func (h *testHeadTracker) PrefetchHead() types.HeadInfo {
	return h.prefetch
}

func (h *testHeadTracker) ValidatedOptimistic() (types.OptimisticUpdate, bool) {
	if h.validated.Header == (types.Header{}) {
		return types.OptimisticUpdate{}, false
	}
	
	// Create execution header for the attested block
	execHeader := types.NewExecutionHeader(&deneb.ExecutionPayloadHeader{
		BlockNumber: 456,
		BlockHash:   zrntcommon.Hash32(common.HexToHash("905ac721c4058d9ed40b27b6b9c1bdd10d4333e4f3d9769100bf9dfb80e5d1f6")),
	})
	
	return types.OptimisticUpdate{
		Attested:      types.HeaderWithExecProof{Header: h.validated.Header, PayloadHeader: execHeader},
		Signature:     h.validated.Signature,
		SignatureSlot: h.validated.SignatureSlot,
	}, true
}

func (h *testHeadTracker) ValidatedFinality() (types.FinalityUpdate, bool) {
	if h.validated.Header == (types.Header{}) {
		return types.FinalityUpdate{}, false
	}
	
	// Use the finalized field if set, otherwise create a default one
	finalized := h.finalized
	if finalized.PayloadHeader == nil {
		finalized = types.HeaderWithExecProof{PayloadHeader: types.NewExecutionHeader(new(deneb.ExecutionPayloadHeader))}
		return types.FinalityUpdate{
			Attested:      types.HeaderWithExecProof{Header: h.validated.Header},
			Finalized:     finalized,
			Signature:     h.validated.Signature,
			SignatureSlot: h.validated.SignatureSlot,
		}, false // Return false when no real finalized data
	}
	
	// Create execution header for the attested block
	execHeader := types.NewExecutionHeader(&deneb.ExecutionPayloadHeader{
		BlockNumber: 456,
		BlockHash:   zrntcommon.Hash32(common.HexToHash("905ac721c4058d9ed40b27b6b9c1bdd10d4333e4f3d9769100bf9dfb80e5d1f6")),
	})
	
	return types.FinalityUpdate{
		Attested:      types.HeaderWithExecProof{Header: h.validated.Header, PayloadHeader: execHeader},
		Finalized:     finalized,
		Signature:     h.validated.Signature,
		SignatureSlot: h.validated.SignatureSlot,
	}, true // Return true when finalized data is available
}

// TestBlockSyncFinality tests the finality update functionality in block sync
func TestBlockSyncFinality(t *testing.T) {
	ht := &testHeadTracker{}
	blockSync := newBeaconBlockSync(ht)
	headCh := make(chan types.ChainHeadEvent, 16)
	blockSync.SubscribeChainHead(headCh)
	ts := sync.NewTestScheduler(t, blockSync)
	ts.AddServer(testServer1, 1)
	ts.AddServer(testServer2, 1)

	// Helper function to check chain head event
	checkChainHeadEvent := func(expectedFinalizedHash common.Hash) {
		t.Helper()
		select {
		case event := <-headCh:
			if event.Finalized != expectedFinalizedHash {
				t.Errorf("Wrong finalized hash, expected %v, got %v", expectedFinalizedHash, event.Finalized)
			}
			t.Logf("Chain head event received: block=%d, finalized=%v", event.Block.NumberU64(), event.Finalized)
		default:
			t.Error("Expected chain head event, but got none")
		}
	}

	// Set block 1 as prefetch head
	head1 := blockHeadInfo(testBlock1)
	ht.prefetch = head1
	ts.ServerEvent(sync.EvNewHead, testServer1, head1)

	// Request and receive block 1
	ts.Run(1, testServer1, sync.ReqBeaconBlock(head1.BlockRoot))
	ts.RequestEvent(request.EvResponse, ts.Request(1, 1), testBlock1)
	ts.AddAllowance(testServer1, 1)
	ts.Run(2)

	// Set as validated head - this should trigger a chain head event without finality
	ht.validated.Header = testBlock1.Header()
	ts.Run(3)

	// Should get a chain head event without finalized hash
	checkChainHeadEvent(common.Hash{})

	// Now test finality update with block 2 to ensure different head info
	head2 := blockHeadInfo(testBlock2)
	ht.prefetch = head2
	ts.ServerEvent(sync.EvNewHead, testServer2, head2)

	// Request and receive block 2
	ts.Run(4, testServer2, sync.ReqBeaconBlock(head2.BlockRoot))
	ts.RequestEvent(request.EvResponse, ts.Request(4, 1), testBlock2)
	ts.AddAllowance(testServer2, 1)
	ts.Run(5)

	// Create a finalized execution header
	finalizedExecHeader := types.NewExecutionHeader(&deneb.ExecutionPayloadHeader{
		BlockNumber: 400, // Earlier block number
		BlockHash:   zrntcommon.Hash32(common.HexToHash("abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890")),
	})

	// Create a finalized header with the same epoch as the attested header
	finalizedHeader := types.Header{
		Slot: testBlock2.Header().Slot, // Same epoch as block 2
	}

	// Update the testHeadTracker to return finality info and set block 2 as validated
	ht.finalized = types.HeaderWithExecProof{
		Header:        finalizedHeader,
		PayloadHeader: finalizedExecHeader,
	}
	ht.validated.Header = testBlock2.Header()

	// Trigger another update to check finality
	ts.Run(6)

	// Should now have a finalized hash in the chain head event
	checkChainHeadEvent(finalizedExecHeader.BlockHash())
}
