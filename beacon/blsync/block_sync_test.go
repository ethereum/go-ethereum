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
	"github.com/protolambda/ztyp/view"
)

var (
	testServer1 = testServer("testServer1")
	testServer2 = testServer("testServer2")

	testBlock1 = types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 127,
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 456,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("905ac721c4058d9ed40b27b6b9c1bdd10d4333e4f3d9769100bf9dfb80e5d1f6")),
			},
		},
	})
	testBlock2 = types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 128,
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 457,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("011703f39c664efc1c6cf5f49ca09b595581eec572d4dfddd3d6179a9e63e655")),
			},
		},
	})
	testFinal1 = types.NewExecutionHeader(&deneb.ExecutionPayloadHeader{
		BlockNumber: 395,
		BlockHash:   zrntcommon.Hash32(common.HexToHash("abbe7625624bf8ddd84723709e2758956289465dd23475f02387e0854942666")),
	})
	testFinal2 = types.NewExecutionHeader(&deneb.ExecutionPayloadHeader{
		BlockNumber: 420,
		BlockHash:   zrntcommon.Hash32(common.HexToHash("9182a6ef8723654de174283750932ccc092378549836bf4873657eeec474598")),
	})
)

type testServer string

func (t testServer) Name() string {
	return string(t)
}

func TestBlockSync(t *testing.T) {
	ht := &testHeadTracker{}
	blockSync := newBeaconBlockSync(ht, false)
	headCh := make(chan types.ChainHeadEvent, 16)
	blockSync.SubscribeChainHead(headCh)
	ts := sync.NewTestScheduler(t, blockSync)
	ts.AddServer(testServer1, 1)
	ts.AddServer(testServer2, 1)

	expHeadEvent := func(expHead *types.BeaconBlock, expFinal *types.ExecutionHeader) {
		t.Helper()
		var expNumber, headNumber uint64
		var expFinalHash, finalHash common.Hash
		if expHead != nil {
			p, err := expHead.ExecutionPayload()
			if err != nil {
				t.Fatalf("expHead.ExecutionPayload() failed: %v", err)
			}
			expNumber = p.NumberU64()
		}
		if expFinal != nil {
			expFinalHash = expFinal.BlockHash()
		}
		select {
		case event := <-headCh:
			headNumber = event.Block.NumberU64()
			finalHash = event.Finalized
		default:
		}
		if headNumber != expNumber {
			t.Errorf("Wrong head block, expected block number %d, got %d)", expNumber, headNumber)
		}
		if finalHash != expFinalHash {
			t.Errorf("Wrong finalized block, expected block hash %064x, got %064x)", expFinalHash[:], finalHash[:])
		}
	}

	// no block requests expected until head tracker knows about a head
	ts.Run(1)
	expHeadEvent(nil, nil)

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
	expHeadEvent(nil, nil)

	// set as validated head, expect no further requests but block 1 set as head block
	ht.validated.Header = testBlock1.Header()
	ht.finalized, ht.finalizedPayload = testBlock1.Header(), testFinal1
	ts.Run(4)
	expHeadEvent(testBlock1, testFinal1)

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
	expHeadEvent(nil, nil)

	// valid response, now head block should be block 2 immediately as it is already validated
	// but head event is still not expected because an epoch boundary was crossed and the
	// expected finality update has not arrived yet
	ts.RequestEvent(request.EvResponse, ts.Request(7, 1), testBlock2)
	ts.Run(8)
	expHeadEvent(nil, nil)

	// expected finality update arrived, now a head event is expected
	ht.finalized, ht.finalizedPayload = testBlock2.Header(), testFinal2
	ts.Run(9)
	expHeadEvent(testBlock2, testFinal2)
}

// TestBlockSyncP2PBlocks tests that with p2pBlocks no blocks are requested and the head
// events carry the execution block hash proven by the light client update.
func TestBlockSyncP2PBlocks(t *testing.T) {
	ht := &testHeadTracker{}
	blockSync := newBeaconBlockSync(ht, true)
	headCh := make(chan types.ChainHeadEvent, 16)
	blockSync.SubscribeChainHead(headCh)
	ts := sync.NewTestScheduler(t, blockSync)
	ts.AddServer(testServer1, 1)

	expHeadEvent := func(expHash, expFinal common.Hash) {
		t.Helper()
		var event types.ChainHeadEvent
		select {
		case event = <-headCh:
		default:
		}
		if event.ExecHash != expHash {
			t.Errorf("Wrong head hash, expected %064x, got %064x", expHash[:], event.ExecHash[:])
		}
		if event.Block != nil {
			t.Errorf("Unexpected execution block in head event")
		}
		if event.Finalized != expFinal {
			t.Errorf("Wrong finalized block, expected block hash %064x, got %064x", expFinal[:], event.Finalized[:])
		}
	}
	payload := func(number uint64, hash string) *types.ExecutionHeader {
		return types.NewExecutionHeader(&deneb.ExecutionPayloadHeader{
			BlockNumber: view.Uint64View(number),
			BlockHash:   zrntcommon.Hash32(common.HexToHash(hash)),
		})
	}
	var (
		head1    = types.Header{Slot: 127}
		head2    = types.Header{Slot: 128, ParentRoot: head1.Hash()} // first slot of epoch 4
		head3    = types.Header{Slot: 129, ParentRoot: head2.Hash()}
		payload1 = payload(456, "905ac721c4058d9ed40b27b6b9c1bdd10d4333e4f3d9769100bf9dfb80e5d1f6")
		payload2 = payload(457, "011703f39c664efc1c6cf5f49ca09b595581eec572d4dfddd3d6179a9e63e655")
		payload3 = payload(458, "5d5b1f6a3e2c4b7e8f90a1b2c3d4e5f60718293a4b5c6d7e8f9a0b1c2d3e4f50")
	)

	// an announced head doesn't trigger a block request
	ht.prefetch = types.HeadInfo{Slot: head1.Slot, BlockRoot: head1.Hash()}
	ts.ServerEvent(sync.EvNewHead, testServer1, ht.prefetch)
	ts.Run(1)
	expHeadEvent(common.Hash{}, common.Hash{})

	// a validated head is sent right away, with its proven execution hash; no request
	ht.validated.Header, ht.validatedPayload = head1, payload1
	ht.finalized, ht.finalizedPayload = head1, testFinal1
	ts.Run(2)
	expHeadEvent(payload1.BlockHash(), testFinal1.BlockHash())

	// the first block of the next epoch waits for the finality update
	ht.validated.Header, ht.validatedPayload = head2, payload2
	ts.Run(3)
	expHeadEvent(common.Hash{}, common.Hash{})

	// the next one doesn't: its parent's slot is known from the validated heads
	ht.validated.Header, ht.validatedPayload = head3, payload3
	ts.Run(4)
	expHeadEvent(payload3.BlockHash(), common.Hash{})
}

// TestBlockSyncP2PBlocksFallback tests that with p2pBlocks the execution block of a head is
// fetched when asked for (the execution client couldn't get it from its peers), and the head
// sent again with it.
func TestBlockSyncP2PBlocksFallback(t *testing.T) {
	ht := &testHeadTracker{}
	blockSync := newBeaconBlockSync(ht, true)
	headCh := make(chan types.ChainHeadEvent, 16)
	blockSync.SubscribeChainHead(headCh)
	ts := sync.NewTestScheduler(t, blockSync)
	ts.AddServer(testServer1, 1)

	nextEvent := func() (types.ChainHeadEvent, bool) {
		select {
		case event := <-headCh:
			return event, true
		default:
			return types.ChainHeadEvent{}, false
		}
	}
	payload1, _ := testBlock1.ExecutionPayload()
	head := testBlock1.Header()
	ht.validated.Header = head
	ht.validatedPayload = types.NewExecutionHeader(&deneb.ExecutionPayloadHeader{
		BlockNumber: view.Uint64View(payload1.NumberU64()),
		BlockHash:   zrntcommon.Hash32(payload1.Hash()),
	})

	// the validated head is sent without its block
	ts.Run(1)
	event, ok := nextEvent()
	if !ok || event.Block != nil || event.ExecHash != payload1.Hash() {
		t.Fatalf("expected the head without its block, got %v (event: %v)", event, ok)
	}

	// asked for its block: requested once, then the head is sent again with it, once
	blockSync.fetchBlock(event)
	ts.Run(2, testServer1, sync.ReqBeaconBlock(testBlock1.Root()))
	ts.RequestEvent(request.EvResponse, ts.Request(2, 1), testBlock1)
	ts.AddAllowance(testServer1, 1)
	ts.Run(3)
	again, ok := nextEvent()
	if !ok || again.Block == nil || again.Block.Hash() != payload1.Hash() || again.ExecHash != payload1.Hash() ||
		again.BeaconHead != testBlock1.Header() {
		t.Fatalf("expected the head with its block, got %v (event: %v)", again, ok)
	}
	ts.Run(4)
	if e, ok := nextEvent(); ok {
		t.Fatalf("unexpected head event %v", e)
	}

	// a failed request is repeated (while it is still the latest head)
	head2 := types.Header{Slot: head.Slot + 1, ParentRoot: head.Hash()}
	ht.validated.Header = head2
	ts.Run(5)
	event2, _ := nextEvent()
	blockSync.fetchBlock(event2)
	ts.Run(6, testServer1, sync.ReqBeaconBlock(head2.Hash()))
	ts.RequestEvent(request.EvFail, ts.Request(6, 1), nil)
	ts.AddAllowance(testServer1, 1)
	ts.Run(7, testServer1, sync.ReqBeaconBlock(head2.Hash()))

	// a block that doesn't carry the head's execution block isn't sent, nor requested again
	ts.RequestEvent(request.EvResponse, ts.Request(7, 1), testBlock2)
	ts.AddAllowance(testServer1, 1)
	ts.Run(8)
	if e, ok := nextEvent(); ok {
		t.Fatalf("unexpected head event %v", e)
	}

	// a request for a head older than the last one sent is dropped (its block is at hand)
	blockSync.fetchBlock(event)
	ts.Run(9)
	if e, ok := nextEvent(); ok {
		t.Fatalf("unexpected head event %v", e)
	}
}

type testHeadTracker struct {
	prefetch         types.HeadInfo
	validated        types.SignedHeader
	validatedPayload *types.ExecutionHeader
	finalized        types.Header
	finalizedPayload *types.ExecutionHeader
}

func (h *testHeadTracker) PrefetchHead() types.HeadInfo {
	return h.prefetch
}

func (h *testHeadTracker) ValidatedOptimistic() (types.OptimisticUpdate, bool) {
	attested := types.HeaderWithExecProof{Header: h.validated.Header, PayloadHeader: h.validatedPayload}
	return types.OptimisticUpdate{
		Attested:      attested,
		Signature:     h.validated.Signature,
		SignatureSlot: h.validated.SignatureSlot,
	}, h.validated.Header != (types.Header{})
}

func (h *testHeadTracker) ValidatedFinality() (types.FinalityUpdate, bool) {
	if h.validated.Header == (types.Header{}) || h.finalizedPayload == nil {
		return types.FinalityUpdate{}, false
	}
	return types.FinalityUpdate{
		Attested:      types.HeaderWithExecProof{Header: h.finalized},
		Finalized:     types.HeaderWithExecProof{Header: h.finalized, PayloadHeader: h.finalizedPayload},
		Signature:     h.validated.Signature,
		SignatureSlot: h.validated.SignatureSlot,
	}, true
}
