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
	blockSync := newBeaconBlockSync(ht)
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

type testHeadTracker struct {
	prefetch         types.HeadInfo
	validated        types.SignedHeader
	finalized        types.Header
	finalizedPayload *types.ExecutionHeader
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
