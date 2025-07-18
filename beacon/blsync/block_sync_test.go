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

// TestBlockSyncFinality tests the beacon block sync's handling of finality updates.
//
// Beacon chain finality works as follows:
// - An "attested" header is the latest block that has been attested to by validators
// - A "finalized" header is a block that has been finalized (cannot be reverted)
// - ChainHeadEvents should include the finalized block hash when finality data is available
// - This enables the execution client to know which blocks are safe from reorgs
func TestBlockSyncFinality(t *testing.T) {
	ht := &testHeadTracker{}
	blockSync := newBeaconBlockSync(ht)
	headCh := make(chan types.ChainHeadEvent, 16)
	blockSync.SubscribeChainHead(headCh)
	ts := sync.NewTestScheduler(t, blockSync)
	ts.AddServer(testServer1, 1)
	ts.AddServer(testServer2, 1)

	// expChainHeadEvent is a helper function that validates ChainHeadEvent emissions.
	// It checks that:
	// 1. An event is emitted when expected (or not emitted when expHead is nil)
	// 2. The event contains the correct execution block number
	// 3. The event contains the expected finalized hash (or empty hash when no finality)
	expChainHeadEvent := func(expHead *types.BeaconBlock, expFinalizedHash common.Hash) {
		t.Helper()
		var event types.ChainHeadEvent
		var hasEvent bool
		select {
		case event = <-headCh:
			hasEvent = true
		default:
		}

		if expHead == nil {
			if hasEvent {
				t.Errorf("Expected no chain head event, but got one with block number %d", event.Block.NumberU64())
			}
			return
		}

		if !hasEvent {
			t.Errorf("Expected chain head event with block number %d, but got none", expHead.Header().Slot)
			return
		}

		expPayload, err := expHead.ExecutionPayload()
		if err != nil {
			t.Fatalf("expHead.ExecutionPayload() failed: %v", err)
		}

		if event.Block.NumberU64() != expPayload.NumberU64() {
			t.Errorf("Wrong head block number, expected %d, got %d", expPayload.NumberU64(), event.Block.NumberU64())
		}

		if event.Finalized != expFinalizedHash {
			t.Errorf("Wrong finalized hash, expected %x, got %x", expFinalizedHash, event.Finalized)
		}
	}

	// ═══════════════════════════════════════════════════════════════════════════════════
	// Test Scenario 1: Basic finality with proper finality update
	// ═══════════════════════════════════════════════════════════════════════════════════
	// This tests the normal case where we have both an attested block (testBlock1) and
	// a finalized block (testBlock2). The ChainHeadEvent should include the finalized
	// block's execution hash, indicating to the execution client that testBlock2 is safe.

	head1 := blockHeadInfo(testBlock1)
	ht.prefetch = head1
	ht.validated.Header = testBlock1.Header()

	// Configure finality update: testBlock1 is attested, testBlock2 is finalized
	ht.finalized.Attested.Header = testBlock1.Header()
	ht.finalized.Finalized.Header = testBlock2.Header()
	ht.finalized.Finalized.PayloadHeader = createTestExecutionHeader(testBlock2)

	// Simulate the block sync process
	ts.ServerEvent(sync.EvNewHead, testServer1, head1)
	ts.Run(1, testServer1, sync.ReqBeaconBlock(head1.BlockRoot))
	ts.RequestEvent(request.EvResponse, ts.Request(1, 1), testBlock1)
	ts.AddAllowance(testServer1, 1)
	ts.Run(2)

	// Verify that ChainHeadEvent includes the finalized block's execution hash
	finalizedPayload, err := testBlock2.ExecutionPayload()
	if err != nil {
		t.Fatalf("Failed to get finalized payload: %v", err)
	}
	expFinalizedHash := finalizedPayload.Hash()
	expChainHeadEvent(testBlock1, expFinalizedHash)

	// ═══════════════════════════════════════════════════════════════════════════════════
	// Test Scenario 2: No finality update available
	// ═══════════════════════════════════════════════════════════════════════════════════
	// This tests the case where we have a new head block but no finality information.
	// The ChainHeadEvent should be emitted but with an empty finalized hash.

	// Clear any pending events from the previous test
	select {
	case <-headCh:
	default:
	}

	// Set up scenario: new head (testBlock2) but no finality update
	ht.validated.Header = testBlock2.Header()
	ht.finalized = types.FinalityUpdate{} // Explicitly clear finality data
	head2 := blockHeadInfo(testBlock2)
	ht.prefetch = head2

	// Simulate block sync process
	ts.ServerEvent(sync.EvNewHead, testServer1, head2)
	ts.Run(3, testServer1, sync.ReqBeaconBlock(head2.BlockRoot))
	ts.RequestEvent(request.EvResponse, ts.Request(3, 1), testBlock2)
	ts.AddAllowance(testServer1, 1)
	ts.Run(4)

	// Verify ChainHeadEvent is emitted but with empty finalized hash
	expChainHeadEvent(testBlock2, common.Hash{})

	// ═══════════════════════════════════════════════════════════════════════════════════
	// Test Scenario 3: Direct ValidatedFinality method testing
	// ═══════════════════════════════════════════════════════════════════════════════════
	// This tests the ValidatedFinality method directly to ensure it returns the correct
	// finality update structure and availability flag.

	// Clear any pending events
	select {
	case <-headCh:
	default:
	}

	// Set up a proper finality update structure
	ht.validated.Header = testBlock1.Header()
	ht.finalized.Attested.Header = testBlock1.Header()
	ht.finalized.Finalized.Header = testBlock2.Header()
	ht.finalized.Finalized.PayloadHeader = createTestExecutionHeader(testBlock2)

	// Test the ValidatedFinality method directly
	finalityUpdate, hasFinalityUpdate := ht.ValidatedFinality()
	if !hasFinalityUpdate {
		t.Error("Expected finality update to be available")
	}

	if finalityUpdate.Attested.Header != testBlock1.Header() {
		t.Error("Finality update attested header doesn't match expected testBlock1")
	}

	if finalityUpdate.Finalized.Header != testBlock2.Header() {
		t.Error("Finality update finalized header doesn't match expected testBlock2")
	}

	// Test that the sync logic properly uses this finality update
	// Since testBlock1 is already in cache, we can just run the sync logic
	ts.Run(5)

	// Verify that the finality information is properly included in the ChainHeadEvent
	expChainHeadEvent(testBlock1, expFinalizedHash)
}

// createTestExecutionHeader creates a minimal ExecutionHeader for testing purposes.
//
// In production, ExecutionHeaders contain many fields (parent hash, state root, receipts root, etc.)
// but for testing beacon chain finality logic, we only need the block hash to verify that
// the correct finalized block is referenced in ChainHeadEvents.
//
// This simplified approach allows us to test the finality propagation logic without
// dealing with the complexity of constructing full execution payloads.
func createTestExecutionHeader(block *types.BeaconBlock) *types.ExecutionHeader {
	payload, err := block.ExecutionPayload()
	if err != nil {
		panic(err)
	}
	// Create a minimal ExecutionHeader with only the block hash populated
	// This is sufficient for testing finality hash propagation
	execHeader := &deneb.ExecutionPayloadHeader{
		BlockHash: [32]byte(payload.Hash()),
	}
	return types.NewExecutionHeader(execHeader)
}

// testHeadTracker is a mock implementation of the HeadTracker interface for testing.
// It allows tests to simulate different beacon chain states and finality conditions.
type testHeadTracker struct {
	prefetch  types.HeadInfo       // The head info to return from PrefetchHead()
	validated types.SignedHeader   // The validated header for optimistic updates
	finalized types.FinalityUpdate // The finality update data for comprehensive finality testing
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

// ValidatedFinality returns the most recent finality update if available.
//
// This method implements a two-tier approach:
//  1. Primary: If explicit finality data is set (h.finalized), return it directly
//  2. Fallback: For backward compatibility with existing tests, create a minimal
//     finality update from the validated header
//
// The fallback ensures that existing tests continue to work while new tests
// can take advantage of the more comprehensive finality testing capabilities.
func (h *testHeadTracker) ValidatedFinality() (types.FinalityUpdate, bool) {
	// Primary path: Return explicit finality data if available
	if h.finalized.Attested.Header != (types.Header{}) {
		return h.finalized, true
	}

	// Fallback path: Create minimal finality update for backward compatibility
	// This ensures existing tests continue to work without modification
	if h.validated.Header != (types.Header{}) {
		finalized := types.NewExecutionHeader(new(deneb.ExecutionPayloadHeader))
		return types.FinalityUpdate{
			Attested:      types.HeaderWithExecProof{Header: h.validated.Header},
			Finalized:     types.HeaderWithExecProof{PayloadHeader: finalized},
			Signature:     h.validated.Signature,
			SignatureSlot: h.validated.SignatureSlot,
		}, true
	}

	// No finality data available
	return types.FinalityUpdate{}, false
}
