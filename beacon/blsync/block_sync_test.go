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
	finalityUpdate types.FinalityUpdate
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

// TODO add test case for finality
func (h *testHeadTracker) ValidatedFinality() (types.FinalityUpdate, bool) {
	finalized := types.NewExecutionHeader(new(deneb.ExecutionPayloadHeader))
	return types.FinalityUpdate{
		Attested:      types.HeaderWithExecProof{Header: h.validated.Header},
		Finalized:     types.HeaderWithExecProof{PayloadHeader: finalized},
		Signature:     h.validated.Signature,
		SignatureSlot: h.validated.SignatureSlot,
	}, h.validated.Header != (types.Header{})
}

func TestBlockSyncFinality(t *testing.T) {
	ht := &testHeadTracker{}
	blockSync := newBeaconBlockSync(ht)
	headCh := make(chan types.ChainHeadEvent, 16)
	blockSync.SubscribeChainHead(headCh)
	ts := sync.NewTestScheduler(t, blockSync)
	ts.AddServer(testServer1, 1)
	ts.AddServer(testServer2, 1)

	// Helper function to check finality hash in chain head events
	expFinalizedHash := func(expFinalizedHash common.Hash) {
		t.Helper()
		select {
		case event := <-headCh:
			if event.Finalized != expFinalizedHash {
				t.Errorf("Wrong finalized hash, expected %x, got %x", expFinalizedHash, event.Finalized)
			}
		default:
			t.Error("Expected chain head event with finalized hash, but none received")
		}
	}

	// Helper function to check no chain head event
	expNoChainHeadEvent := func() {
		t.Helper()
		select {
		case event := <-headCh:
			t.Errorf("Unexpected chain head event received: finalized hash %x", event.Finalized)
		default:
			// Expected: no event
		}
	}

	// Create test blocks with different slot configurations
	testBlock3 := types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 127, // Slot 127 (epoch 3, slot 63)
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 460,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("a0b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1")),
			},
		},
	})

	testBlock4 := types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 128, // Slot 128 (epoch 4, slot 0)
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 461,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("b1c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2")),
			},
		},
	})

	// Test case 1: No finality update available
	// Set block 3 as validated head, no finality update
	ht.validated.Header = testBlock3.Header()
	ts.Run(1)
	// Should not emit chain head event because no finality update is available
	expNoChainHeadEvent()

	// Test case 2: Finality update available, same epoch
	// Set finality update for the same epoch as attested header
	ht.finalityUpdate = types.FinalityUpdate{
		Attested: types.HeaderWithExecProof{Header: testBlock3.Header()},
		Finalized: types.HeaderWithExecProof{
			Header: types.Header{
				Slot:      127,
				StateRoot: common.HexToHash("f1e2d3c4b5a6f7e8d9c0b1a2f3e4d5c6b7a8f9e0d1c2b3a4f5e6d7c8b9a0f1e2"),
			},
			PayloadHeader: types.NewExecutionHeader(&deneb.ExecutionPayloadHeader{
				BlockHash: zrntcommon.Hash32(common.HexToHash("f1e2d3c4b5a6f7e8d9c0b1a2f3e4d5c6b7a8f9e0d1c2b3a4f5e6d7c8b9a0f1e2")),
			}),
		},
		Signature:     ht.validated.Signature,
		SignatureSlot: ht.validated.SignatureSlot,
	}
	ts.Run(2)
	// Should emit chain head event with finalized hash
	expFinalizedHash(common.HexToHash("f1e2d3c4b5a6f7e8d9c0b1a2f3e4d5c6b7a8f9e0d1c2b3a4f5e6d7c8b9a0f1e2"))

	// Test case 3: Finality update available, attested epoch < finalized epoch
	// This should not emit chain head event
	ht.validated.Header = testBlock3.Header() // Slot 127 (epoch 3)
	ht.finalityUpdate = types.FinalityUpdate{
		Attested: types.HeaderWithExecProof{Header: testBlock3.Header()},
		Finalized: types.HeaderWithExecProof{
			Header: types.Header{
				Slot:      128,
				StateRoot: common.HexToHash("e2d3c4b5a6f7e8d9c0b1a2f3e4d5c6b7a8f9e0d1c2b3a4f5e6d7c8b9a0f1e2d3"),
			},
			PayloadHeader: types.NewExecutionHeader(&deneb.ExecutionPayloadHeader{
				BlockHash: zrntcommon.Hash32(common.HexToHash("e2d3c4b5a6f7e8d9c0b1a2f3e4d5c6b7a8f9e0d1c2b3a4f5e6d7c8b9a0f1e2d3")),
			}),
		},
		Signature:     ht.validated.Signature,
		SignatureSlot: ht.validated.SignatureSlot,
	}
	ts.Run(3)
	// Should not emit chain head event because attested epoch < finalized epoch
	expNoChainHeadEvent()

	// Test case 4: Finality update available, attested epoch == finalized epoch + 1
	// Set block 4 as validated head (epoch 4, slot 0)
	ht.validated.Header = testBlock4.Header()
	// Set finalized header to epoch 3 (slot 127)
	ht.finalityUpdate.Finalized.Header.Slot = 127
	ts.Run(4)
	// Should not emit chain head event because head is at first slot of next epoch
	// and we need to wait for finality update
	expNoChainHeadEvent()

	// Test case 5: Finality update available, attested epoch == finalized epoch + 1
	// but parent block is not in the same epoch as finalized
	// Set block 4 as validated head (epoch 4, slot 0)
	ht.validated.Header = testBlock4.Header()
	// Set finalized header to epoch 3 (slot 127)
	ht.finalityUpdate.Finalized.Header.Slot = 127
	// Set finalized payload header for test case 5
	ht.finalityUpdate.Finalized.PayloadHeader = types.NewExecutionHeader(&deneb.ExecutionPayloadHeader{
		BlockHash: zrntcommon.Hash32(common.HexToHash("e2d3c4b5a6f7e8d9c0b1a2f3e4d5c6b7a8f9e0d1c2b3a4f5e6d7c8b9a0f1e2d3")),
	})
	// Add parent block (block 3) to recent blocks
	blockSync.recentBlocks.Add(testBlock3.Root(), testBlock3)
	ts.Run(5)
	// Should emit chain head event because parent block is available and not in finalized epoch
	expFinalizedHash(common.HexToHash("e2d3c4b5a6f7e8d9c0b1a2f3e4d5c6b7a8f9e0d1c2b3a4f5e6d7c8b9a0f1e2d3"))

	// Test case 6: Finality update available, attested epoch == finalized epoch + 1
	// but parent block is in the same epoch as finalized
	// Create a block in epoch 2 (slot 95)
	testBlock5 := types.NewBeaconBlock(&deneb.BeaconBlock{
		Slot: 95, // Slot 95 (epoch 2, slot 63)
		Body: deneb.BeaconBlockBody{
			ExecutionPayload: deneb.ExecutionPayload{
				BlockNumber: 462,
				BlockHash:   zrntcommon.Hash32(common.HexToHash("c2d3e4f5a6b7c8d9e0f1a2b3c4d5e6f7a8b9c0d1e2f3a4b5c6d7e8f9a0b1c2d3")),
			},
		},
	})
	ht.validated.Header = testBlock4.Header() // Slot 128 (epoch 4, slot 0)
	ht.finalityUpdate.Finalized.Header.Slot = 127 // Epoch 3, slot 63
	// Add parent block (block 5) to recent blocks - this is in epoch 2
	blockSync.recentBlocks.Add(testBlock5.Root(), testBlock5)
	ts.Run(6)
	// Should not emit chain head event because parent is in finalized epoch
	expNoChainHeadEvent()
}
