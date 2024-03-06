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
	"github.com/ethereum/go-ethereum/event"
	"github.com/protolambda/zrnt/eth2/beacon/capella"
	"github.com/protolambda/zrnt/eth2/configs"
	"github.com/protolambda/ztyp/tree"
)

var (
	testServer1 = "testServer1"
	testServer2 = "testServer2"

	testBlock1 = &capella.BeaconBlock{
		Slot: 123,
		Body: capella.BeaconBlockBody{
			ExecutionPayload: capella.ExecutionPayload{BlockNumber: 456},
		},
	}
	testBlock2 = &capella.BeaconBlock{
		Slot: 124,
		Body: capella.BeaconBlockBody{
			ExecutionPayload: capella.ExecutionPayload{BlockNumber: 457},
		},
	}
)

func init() {
	eb1, _ := getExecBlock(testBlock1)
	testBlock1.Body.ExecutionPayload.BlockHash = tree.Root(eb1.Hash())
	eb2, _ := getExecBlock(testBlock2)
	testBlock2.Body.ExecutionPayload.BlockHash = tree.Root(eb2.Hash())
}

func TestBlockSync(t *testing.T) {
	ht := &testHeadTracker{}
	eventFeed := new(event.Feed)
	blockSync := newBeaconBlockSync(ht, eventFeed)
	headCh := make(chan types.ChainHeadEvent, 16)
	eventFeed.Subscribe(headCh)
	ts := sync.NewTestScheduler(t, blockSync)
	ts.AddServer(testServer1, 1)
	ts.AddServer(testServer2, 1)

	expHeadBlock := func(tci int, expHead *capella.BeaconBlock) {
		var expNumber, headNumber uint64
		if expHead != nil {
			expNumber = uint64(expHead.Body.ExecutionPayload.BlockNumber)
		}
		select {
		case event := <-headCh:
			headNumber = event.HeadBlock.Number
		default:
		}
		if headNumber != expNumber {
			t.Errorf("Wrong head block in test case #%d (expected block number %d, got %d)", tci, expNumber, headNumber)
		}
	}

	// no block requests expected until head tracker knows about a head
	ts.Run(1)
	expHeadBlock(1, nil)

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
	expHeadBlock(3, nil)

	// set as validated head, expect no further requests but block 1 set as head block
	ht.validated.Header = blockHeader(testBlock1)
	ts.Run(4)
	expHeadBlock(4, testBlock1)

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
	ht.validated.Header = blockHeader(testBlock2)
	// expect req2 retry to server 2
	ts.Run(7, testServer2, sync.ReqBeaconBlock(head2.BlockRoot))
	// now head block should be unavailable again
	expHeadBlock(4, nil)

	// valid response, now head block should be block 2 immediately as it is already validated
	ts.RequestEvent(request.EvResponse, ts.Request(7, 1), testBlock2)
	ts.Run(8)
	expHeadBlock(5, testBlock2)
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

func (h *testHeadTracker) ValidatedHead() (types.SignedHeader, bool) {
	return h.validated, h.validated.Header != (types.Header{})
}

// TODO add test case for finality
func (h *testHeadTracker) ValidatedFinality() (types.FinalityUpdate, bool) {
	return types.FinalityUpdate{
		Attested:      types.HeaderWithExecProof{Header: h.validated.Header},
		Finalized:     types.HeaderWithExecProof{PayloadHeader: &capella.ExecutionPayloadHeader{}},
		Signature:     h.validated.Signature,
		SignatureSlot: h.validated.SignatureSlot,
	}, h.validated.Header != (types.Header{})
}
