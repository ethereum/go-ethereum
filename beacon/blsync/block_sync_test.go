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
}

func (h *testHeadTracker) PrefetchHead() types.HeadInfo {
	return h.prefetch
}

func (h *testHeadTracker) ValidatedOptimistic() (types.OptimisticUpdate, bool) {
	return types.OptimisticUpdate{
		Attested:      types.HeaderWithExecProof{Header: h.validated.Header},
		Signature:     h.validated.Signature,
		SignatureSlot: h.validated.SignatureSlot,
	}, h.validated.Header != (types.Header{})
}

func (h *testHeadTracker) ValidatedFinality() (types.FinalityUpdate, bool) {
	finalized := types.NewExecutionHeader(new(deneb.ExecutionPayloadHeader))
	return types.FinalityUpdate{
		Attested:      types.HeaderWithExecProof{Header: h.validated.Header},
		Finalized:     types.HeaderWithExecProof{PayloadHeader: finalized},
		Signature:     h.validated.Signature,
		SignatureSlot: h.validated.SignatureSlot,
	}, h.validated.Header != (types.Header{})
}

func TestValidatedFinality(t *testing.T) {
	tracker := &testHeadTracker{}

	earlyBlock := types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 42,
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 400,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("1111111111111111111111111111111111111111111111111111111111111111")),
			},
		},
	})

	tracker.validated.Header = earlyBlock.Header()
	tracker.prefetch = blockHeadInfo(earlyBlock)

	_, ok := tracker.ValidatedFinality()
	if !ok {
		t.Fatalf("ValidatedFinality failed for early block")
	}

	lateBlock := types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 200,
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 457,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("011703f39c664efc1c6cf5f49ca09b595581eec572d4dfddd3d6179a9e63e655")),
			},
		},
	})
	tracker.validated.Header = lateBlock.Header()
	tracker.prefetch = blockHeadInfo(lateBlock)

	blockSync := newBeaconBlockSync(tracker)
	headCh := make(chan types.ChainHeadEvent, 16)
	blockSync.SubscribeChainHead(headCh)

	execPayload, _ := lateBlock.ExecutionPayload()
	payloadHash := common.HexToHash("905ac721c4058d9ed40b27b6b9c1bdd10d4333e4f3d9769100bf9dfb80e5d1f6")

	blockSync.chainHeadFeed.Send(types.ChainHeadEvent{
		Block:     execPayload,
		Finalized: payloadHash,
	})

	select {
	case event := <-headCh:
		if event.Finalized != payloadHash {
			t.Errorf("Wrong finalized hash: got %s, want %s",
				event.Finalized.Hex(), payloadHash.Hex())
		}
	default:
		t.Fatalf("No head event received")
	}
}

func (h *testHeadTracker) ValidatedFinalityWithEpoch() (types.FinalityUpdate, bool) {
	currentEpoch := h.validated.Header.Epoch()

	if currentEpoch < 2 {
		return types.FinalityUpdate{
			Attested: types.HeaderWithExecProof{Header: h.validated.Header},
		}, h.validated.Header != (types.Header{})
	}

	payloadHeader := &deneb.ExecutionPayloadHeader{
		BlockNumber: 456,
		BlockHash:   zrntcommon.Hash32(common.HexToHash("905ac721c4058d9ed40b27b6b9c1bdd10d4333e4f3d9769100bf9dfb80e5d1f6")),
	}

	finalized := types.NewExecutionHeader(payloadHeader)

	return types.FinalityUpdate{
		Attested: types.HeaderWithExecProof{Header: h.validated.Header},
		Finalized: types.HeaderWithExecProof{
			Header:        testBlock1.Header(),
			PayloadHeader: finalized,
		},
		Signature:     h.validated.Signature,
		SignatureSlot: h.validated.SignatureSlot,
	}, true
}

func TestValidatedFinalityWithEpoch(t *testing.T) {
	tracker := &testHeadTracker{}

	earlyBlock := types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 42,
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 400,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("1111111111111111111111111111111111111111111111111111111111111111")),
			},
		},
	})
	tracker.validated.Header = earlyBlock.Header()
	tracker.prefetch = blockHeadInfo(earlyBlock)

	finality, ok := tracker.ValidatedFinality()
	if !ok {
		t.Fatalf("ValidatedFinality failed for early block")
	}

	if finality.Finalized.PayloadHeader != nil &&
		finality.Finalized.PayloadHeader.BlockHash() != (common.Hash{}) {
		t.Errorf("Expected no finality for early block (epoch %d), got: %v",
			earlyBlock.Slot()/64, finality.Finalized.PayloadHeader.BlockHash().Hex())
	}

	lateBlock := types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 200,
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 457,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("011703f39c664efc1c6cf5f49ca09b595581eec572d4dfddd3d6179a9e63e655")),
			},
		},
	})
	tracker.validated.Header = lateBlock.Header()
	tracker.prefetch = blockHeadInfo(lateBlock)

	finality, ok = tracker.ValidatedFinalityWithEpoch()
	if !ok {
		t.Fatalf("ValidatedFinality failed for late block")
	}

	expectedHash := common.HexToHash("905ac721c4058d9ed40b27b6b9c1bdd10d4333e4f3d9769100bf9dfb80e5d1f6")
	if finality.Finalized.PayloadHeader.BlockHash() != expectedHash {
		t.Errorf("Expected finalized hash %s for late block (epoch %d), got %s",
			expectedHash.Hex(), lateBlock.Slot()/64, finality.Finalized.PayloadHeader.BlockHash().Hex())
	}

	blockSync := newBeaconBlockSync(tracker)
	headCh := make(chan types.ChainHeadEvent, 16)
	blockSync.SubscribeChainHead(headCh)

	execPayload, _ := lateBlock.ExecutionPayload()

	blockSync.chainHeadFeed.Send(types.ChainHeadEvent{
		Block:     execPayload,
		Finalized: expectedHash,
	})

	select {
	case event := <-headCh:
		if event.Finalized != expectedHash {
			t.Errorf("Event has wrong finalized hash: got %s, want %s",
				event.Finalized.Hex(), expectedHash.Hex())
		}
		if event.Block.NumberU64() != execPayload.NumberU64() {
			t.Errorf("Event has wrong block number: got %d, want %d",
				event.Block.NumberU64(), execPayload.NumberU64())
		}
	default:
		t.Fatalf("No head event received")
	}
}
