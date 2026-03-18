// Copyright 2026 The go-ethereum Authors
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

package fetcher

import (
	"slices"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

// makeTestBlobSidecar is a helper method to create random blob sidecar
// with certain number of blobs.
func makeTestCellSidecar(blobCount int) *types.BlobTxCellSidecar {
	var (
		blobs       []kzg4844.Blob
		commitments []kzg4844.Commitment
		proofs      []kzg4844.Proof
	)

	for i := 0; i < blobCount; i++ {
		blob := &kzg4844.Blob{}
		blob[0] = byte(i)
		blobs = append(blobs, *blob)

		commit, _ := kzg4844.BlobToCommitment(blob)
		commitments = append(commitments, commit)

		cellProofs, _ := kzg4844.ComputeCellProofs(blob)
		proofs = append(proofs, cellProofs...)
	}

	sidecar, _ := types.NewBlobTxSidecar(types.BlobSidecarVersion1, blobs, commitments, proofs).ToBlobTxCellSidecar()

	return sidecar
}

func selectCells(cells []kzg4844.Cell, custody *types.CustodyBitmap) []kzg4844.Cell {
	custodyIndices := custody.Indices()
	result := make([]kzg4844.Cell, 0)

	for _, idx := range custodyIndices {
		result = append(result, cells[idx])
	}

	return result
}

const (
	testBlobAvailabilityTimeout = 500 * time.Millisecond
	testBlobFetchTimeout        = 5 * time.Second
)

var (
	testBlobTxHashes = []common.Hash{
		{0x01}, {0x02}, {0x03}, {0x04}, {0x05}, {0x06}, {0x07}, {0x08},
	}

	testBlobSidecars = []*types.BlobTxCellSidecar{
		makeTestCellSidecar(1),
		makeTestCellSidecar(2),
		makeTestCellSidecar(3),
		makeTestCellSidecar(4),
	}

	custody = types.NewCustodyBitmap([]uint64{0, 1, 2, 3, 4, 5, 6, 7})

	fullCustody      = *types.CustodyBitmapAll
	halfCustody      = *types.CustodyBitmapData
	frontCustody     = types.NewCustodyBitmap([]uint64{0, 1, 2, 3, 8, 9, 10, 11})
	backCustody      = types.NewCustodyBitmap([]uint64{4, 5, 6, 7, 8, 9, 10, 11})
	differentCustody = types.NewCustodyBitmap([]uint64{8, 9, 10, 11, 12, 13, 14, 15})
)

type doBlobNotify struct {
	peer    string
	hashes  []common.Hash
	custody types.CustodyBitmap
}

type doBlobEnqueue struct {
	peer    string
	hashes  []common.Hash
	cells   [][]kzg4844.Cell
	custody types.CustodyBitmap
}

type blobDoFunc func(*BlobFetcher)

type isWaitingAvailability map[common.Hash]map[string]struct{}

type isDecidedFull map[common.Hash]struct{}
type isDecidedPartial map[common.Hash]struct{}

type blobAnnounce struct {
	hash    common.Hash
	custody types.CustodyBitmap
}

type isBlobScheduled struct {
	announces map[string][]blobAnnounce // announces에 있는 것들 (peer -> hash+custody)
	fetching  map[string][]blobAnnounce // requests에 있는 것들 (peer -> hash+custody)
}

type isCompleted []common.Hash
type isDropped []string

type isFetching struct {
	hashes map[common.Hash]fetchInfo
}

type fetchInfo struct {
	fetching *types.CustodyBitmap
	fetched  []uint64
}

type blobFetcherTest struct {
	init  func() *BlobFetcher
	steps []interface{}
}

type mockRand struct {
	value int
}

func (r *mockRand) Intn(n int) int {
	return r.value
}

// TestBlobFetcherFullSchedule tests scheduling full payload decision
// Blob should be fetched immediately when its availability is announced
// by idle peer, if the client decided to pull the full payload
// Additional announcements should be recorded as alternates during the fetch
func TestBlobFetcherFullFetch(t *testing.T) {
	testBlobFetcher(t, blobFetcherTest{
		init: func() *BlobFetcher {
			return NewBlobFetcher(
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash, *types.CustodyBitmap) error { return nil },
				func(string) {},
				&custody,
				&mockRand{value: 5}, // to force full requests (5 < 15)
			)
		},
		steps: []interface{}{
			// A announced full custody blob (should make full decision & start fetching)
			doBlobNotify{peer: "A", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			isDecidedFull{testBlobTxHashes[0]: struct{}{}},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},
			isFetching{
				hashes: map[common.Hash]fetchInfo{
					testBlobTxHashes[0]: {
						fetching: &halfCustody,
						fetched:  []uint64{},
					},
				},
			},

			// Same hash announced by another peer(B) -> should be added to alternatives
			doBlobNotify{peer: "B", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},

			// Announce partial custody by C -> should be ignored
			doBlobNotify{peer: "C", hashes: []common.Hash{testBlobTxHashes[1]}, custody: custody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},

			// Additional hashes announced by A -> should not be fetched
			doBlobNotify{peer: "A", hashes: []common.Hash{testBlobTxHashes[1]}, custody: fullCustody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}, {hash: testBlobTxHashes[1], custody: halfCustody}},
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},

			// Announce of multiple transactions
			doBlobNotify{peer: "D", hashes: []common.Hash{testBlobTxHashes[2], testBlobTxHashes[3]}, custody: fullCustody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}, {hash: testBlobTxHashes[1], custody: halfCustody}},
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
					"D": {{hash: testBlobTxHashes[2], custody: halfCustody}, {hash: testBlobTxHashes[3], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
					"D": {{hash: testBlobTxHashes[2], custody: halfCustody}, {hash: testBlobTxHashes[3], custody: halfCustody}},
				},
			},
		},
	})
}

// TestBlobFetcherPartialFetching tests partial request decision and availability check flow
func TestBlobFetcherPartialFetch(t *testing.T) {
	testBlobFetcher(t, blobFetcherTest{
		init: func() *BlobFetcher {
			return NewBlobFetcher(
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash, *types.CustodyBitmap) error { return nil },
				func(string) {},
				&custody,
				&mockRand{value: 20}, // Force partial requests (20 >= 15)
			)
		},
		steps: []interface{}{
			// First full announce for tx 0, 1 -> should make partial decision and go to waitlist
			doBlobNotify{peer: "A", hashes: []common.Hash{testBlobTxHashes[0], testBlobTxHashes[1]}, custody: fullCustody},
			isDecidedPartial{testBlobTxHashes[0]: struct{}{}, testBlobTxHashes[1]: struct{}{}},
			isWaitingAvailability{testBlobTxHashes[0]: map[string]struct{}{"A": {}}, testBlobTxHashes[1]: map[string]struct{}{"A": {}}},
			isBlobScheduled{announces: nil, fetching: nil},

			// Partial announce for tx 0 (waitlist) -> should be dropped
			doBlobNotify{peer: "B", hashes: []common.Hash{testBlobTxHashes[0]}, custody: custody},
			isWaitingAvailability{testBlobTxHashes[0]: map[string]struct{}{"A": {}}, testBlobTxHashes[1]: map[string]struct{}{"A": {}}},
			isBlobScheduled{announces: nil, fetching: nil},

			// Second full announce for tx 0 -> should make tx 0 available & fetched
			doBlobNotify{peer: "C", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			isWaitingAvailability{testBlobTxHashes[1]: map[string]struct{}{"A": {}}},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
					"C": {{hash: testBlobTxHashes[0], custody: custody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
				},
			},
			isFetching{
				hashes: map[common.Hash]fetchInfo{
					testBlobTxHashes[0]: {
						fetching: &custody,
						fetched:  []uint64{},
					},
				},
			},

			// Partial announce for tx 0, overlapped custody -> overlapping part should be accepted
			doBlobNotify{peer: "B", hashes: []common.Hash{testBlobTxHashes[0]}, custody: frontCustody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
					"B": {{hash: testBlobTxHashes[0], custody: *frontCustody.Intersection(&custody)}},
					"C": {{hash: testBlobTxHashes[0], custody: custody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
				},
			},

			// Partial announce for tx 0, with additional custody -> don't update
			doBlobNotify{peer: "B", hashes: []common.Hash{testBlobTxHashes[0]}, custody: custody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
					"B": {{hash: testBlobTxHashes[0], custody: *frontCustody.Intersection(&custody)}},
					"C": {{hash: testBlobTxHashes[0], custody: custody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
				},
			},

			// Partial announce for tx 0, without any overlapped custody -> should be dropped
			doBlobNotify{peer: "D", hashes: []common.Hash{testBlobTxHashes[0]}, custody: differentCustody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
					"B": {{hash: testBlobTxHashes[0], custody: *frontCustody.Intersection(&custody)}},
					"C": {{hash: testBlobTxHashes[0], custody: custody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
				},
			},
		},
	})
}

// todo wait timeout
// todo drop

// TestBlobFetcherFullDelivery tests cell delivery and fetch completion logic (full fetch)
func TestBlobFetcherFullDelivery(t *testing.T) {
	testBlobFetcher(t, blobFetcherTest{
		init: func() *BlobFetcher {
			return NewBlobFetcher(
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash, *types.CustodyBitmap) error { return nil },
				func(string) {},
				&custody,
				&mockRand{value: 5}, // Force full requests for simplicity
			)
		},
		steps: []interface{}{
			// Full announce by two peers (A, B) -> schedule fetch
			doBlobNotify{peer: "A", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			doBlobNotify{peer: "B", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},
			isFetching{
				hashes: map[common.Hash]fetchInfo{
					testBlobTxHashes[0]: {
						fetching: &halfCustody,
						fetched:  []uint64{},
					},
				},
			},

			// All alternates should be clean up on delivery
			doBlobEnqueue{peer: "A", hashes: []common.Hash{testBlobTxHashes[0]}, cells: [][]kzg4844.Cell{selectCells(testBlobSidecars[0].Cells, &halfCustody)}, custody: halfCustody},
			isBlobScheduled{announces: nil, fetching: nil},
			isFetching{hashes: nil}, // fetches should be empty after completion
			isCompleted{testBlobTxHashes[0]},
		},
	})
}

// TestBlobFetcherPartialDelivery tests cell delivery and fetch completion logic (partial fetch)
func TestBlobFetcherPartialDelivery(t *testing.T) {
	testBlobFetcher(t, blobFetcherTest{
		init: func() *BlobFetcher {
			return NewBlobFetcher(
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash, *types.CustodyBitmap) error { return nil },
				func(string) {},
				&custody,
				&mockRand{value: 20},
			)
		},
		steps: []interface{}{
			// Full announce by two peers (A, B) -> schedule fetch
			doBlobNotify{peer: "A", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			doBlobNotify{peer: "B", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			isWaitingAvailability(nil),
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
					"B": {{hash: testBlobTxHashes[0], custody: custody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
				},
			},
			isFetching{
				hashes: map[common.Hash]fetchInfo{
					testBlobTxHashes[0]: {
						fetching: &custody,
						fetched:  []uint64{},
					},
				},
			},

			// Partial announce by C, D -> alternates
			doBlobNotify{peer: "C", hashes: []common.Hash{testBlobTxHashes[0]}, custody: frontCustody},
			doBlobNotify{peer: "D", hashes: []common.Hash{testBlobTxHashes[0]}, custody: backCustody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
					"B": {{hash: testBlobTxHashes[0], custody: custody}},
					"C": {{hash: testBlobTxHashes[0], custody: *frontCustody.Intersection(&custody)}},
					"D": {{hash: testBlobTxHashes[0], custody: *backCustody.Intersection(&custody)}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: custody}},
				},
			},

			// Drop A, B -> schedule fetch from C, D
			doDrop("A"),
			doDrop("B"),
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"C": {{hash: testBlobTxHashes[0], custody: *frontCustody.Intersection(&custody)}},
					"D": {{hash: testBlobTxHashes[0], custody: *backCustody.Intersection(&custody)}},
				},
				fetching: map[string][]blobAnnounce{
					"C": {{hash: testBlobTxHashes[0], custody: *frontCustody.Intersection(&custody)}},
					"D": {{hash: testBlobTxHashes[0], custody: *backCustody.Intersection(&custody)}},
				},
			},

			// Delivery from C -> wait for D
			doBlobEnqueue{
				peer:    "C",
				hashes:  []common.Hash{testBlobTxHashes[0]},
				cells:   [][]kzg4844.Cell{selectCells(testBlobSidecars[0].Cells, frontCustody.Intersection(&custody))},
				custody: *frontCustody.Intersection(&custody),
			},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"D": {{hash: testBlobTxHashes[0], custody: *backCustody.Intersection(&custody)}},
				},
				fetching: map[string][]blobAnnounce{
					"D": {{hash: testBlobTxHashes[0], custody: *backCustody.Intersection(&custody)}},
				},
			},
			isFetching{
				hashes: map[common.Hash]fetchInfo{
					testBlobTxHashes[0]: {
						fetching: &custody,
						fetched:  frontCustody.Intersection(&custody).Indices(),
					},
				},
			},

			// Announce already delivered cells + fetching cells -> leave fetching cells only
			doBlobNotify{peer: "E", hashes: []common.Hash{testBlobTxHashes[0]}, custody: custody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"D": {{hash: testBlobTxHashes[0], custody: *backCustody.Intersection(&custody)}},
					"E": {{hash: testBlobTxHashes[0], custody: custody}},
				},
				fetching: map[string][]blobAnnounce{
					"D": {{hash: testBlobTxHashes[0], custody: *backCustody.Intersection(&custody)}},
				},
			},

			// Not delivered -> reschedule to E
			doWait{time: blobFetchTimeout, step: true},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"E": {{hash: testBlobTxHashes[0], custody: custody}},
				},
				fetching: map[string][]blobAnnounce{
					"E": {{hash: testBlobTxHashes[0], custody: *backCustody.Intersection(&custody)}},
				},
			},
			isFetching{
				hashes: map[common.Hash]fetchInfo{
					testBlobTxHashes[0]: {
						fetching: &custody,
						fetched:  frontCustody.Intersection(&custody).Indices(),
					},
				},
			},
			// Delivery from E -> complete
			doWait{time: blobFetchTimeout / 2, step: true},
			doBlobEnqueue{
				peer:    "E",
				hashes:  []common.Hash{testBlobTxHashes[0]},
				cells:   [][]kzg4844.Cell{selectCells(testBlobSidecars[0].Cells, backCustody.Intersection(&custody))},
				custody: *backCustody.Intersection(&custody),
			},
			isCompleted{testBlobTxHashes[0]},
		},
	})
}

// TestBlobFetcherAvailabilityTimeout tests availability timeout for partial requests
func TestBlobFetcherAvailabilityTimeout(t *testing.T) {
	testBlobFetcher(t, blobFetcherTest{
		init: func() *BlobFetcher {
			return NewBlobFetcher(
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash, *types.CustodyBitmap) error { return nil },
				func(string) {},
				&custody,
				&mockRand{value: 20},
			)
		},
		steps: []interface{}{
			// First full announce for tx 0 -> should make partial decision and go to waitlist
			doBlobNotify{peer: "A", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			isDecidedPartial{testBlobTxHashes[0]: struct{}{}},
			isWaitingAvailability{testBlobTxHashes[0]: map[string]struct{}{"A": {}}},
			isBlobScheduled{announces: nil, fetching: nil},

			// Run clock for timeout
			doWait{time: testBlobAvailabilityTimeout, step: true},

			// After timeout, waitlist should be empty
			isWaitingAvailability{},
			isBlobScheduled{announces: nil, fetching: nil},
		},
	})
}

// TestBlobFetcherPeerDrop tests peer drop scenarios
func TestBlobFetcherPeerDrop(t *testing.T) {
	testBlobFetcher(t, blobFetcherTest{
		init: func() *BlobFetcher {
			return NewBlobFetcher(
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash, *types.CustodyBitmap) error { return nil },
				func(string) {},
				&custody,
				&mockRand{value: 5},
			)
		},
		steps: []interface{}{
			// Full announce by peer A -> should schedule fetch
			doBlobNotify{peer: "A", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			isDecidedFull{testBlobTxHashes[0]: struct{}{}},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},
			isFetching{
				hashes: map[common.Hash]fetchInfo{
					testBlobTxHashes[0]: {
						fetching: &halfCustody,
						fetched:  []uint64{},
					},
				},
			},

			// Another peer B announces same hash -> should be added to alternates
			doBlobNotify{peer: "B", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},

			// Drop peer A -> should reschedule fetch to peer B
			doDrop("A"),
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},
			isFetching{
				hashes: map[common.Hash]fetchInfo{
					testBlobTxHashes[0]: {
						fetching: &halfCustody,
						fetched:  []uint64{},
					},
				},
			},

			// Drop peer B -> should drop the transaction, remove all traces
			doDrop("B"),
			isBlobScheduled{announces: nil, fetching: nil},
			isFetching{hashes: nil},
		},
	})
}

// TestBlobFetcherFetchTimeout tests fetch timeout and rescheduling, full request case
func TestBlobFetcherFetchTimeout(t *testing.T) {
	testBlobFetcher(t, blobFetcherTest{
		init: func() *BlobFetcher {
			return NewBlobFetcher(
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(txs []common.Hash, _ [][]kzg4844.Cell, _ *types.CustodyBitmap) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash, *types.CustodyBitmap) error { return nil },
				func(string) {},
				&custody,
				&mockRand{value: 5},
			)
		},
		steps: []interface{}{
			// Full announce by peer A -> schedule fetch
			doBlobNotify{peer: "A", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			isDecidedFull{testBlobTxHashes[0]: struct{}{}},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},
			isFetching{
				hashes: map[common.Hash]fetchInfo{
					testBlobTxHashes[0]: {
						fetching: &halfCustody,
						fetched:  []uint64{},
					},
				},
			},

			// Another peer announces same hash -> should be added to alternates
			doBlobNotify{peer: "B", hashes: []common.Hash{testBlobTxHashes[0]}, custody: fullCustody},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"A": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},

			// Wait for fetch timeout -> should reschedule to peer B
			doWait{time: testBlobFetchTimeout, step: true},
			isBlobScheduled{
				announces: map[string][]blobAnnounce{
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
				fetching: map[string][]blobAnnounce{
					"B": {{hash: testBlobTxHashes[0], custody: halfCustody}},
				},
			},
			isFetching{
				hashes: map[common.Hash]fetchInfo{
					testBlobTxHashes[0]: {
						fetching: &halfCustody,
						fetched:  []uint64{},
					},
				},
			},

			// Wait for timeout -> should drop transaction
			doWait{time: testBlobFetchTimeout, step: true},
			isBlobScheduled{announces: nil, fetching: nil},
			isFetching{hashes: nil},
		},
	})
}

// testBlobFetcher is the generic test runner for blob fetcher tests
func testBlobFetcher(t *testing.T, tt blobFetcherTest) {
	clock := new(mclock.Simulated)
	wait := make(chan struct{})

	// Create a fetcher and boot it up
	fetcher := tt.init()
	fetcher.clock = clock
	fetcher.step = wait

	fetcher.Start()
	defer fetcher.Stop()

	defer func() {
		for {
			select {
			case <-wait:
			default:
				return
			}
		}
	}()

	// Iterate through all the test steps and execute them
	for i, step := range tt.steps {
		// Clear the channel if anything is left over
		for len(wait) > 0 {
			<-wait
		}
		// Process the next step of the test
		switch step := step.(type) {
		case doBlobNotify:
			if err := fetcher.Notify(step.peer, step.hashes, step.custody); err != nil {
				t.Errorf("step %d: failed to notify fetcher: %v", i, err)
				return
			}
			<-wait

		case doBlobEnqueue:
			if err := fetcher.Enqueue(step.peer, step.hashes, step.cells, step.custody); err != nil {
				t.Errorf("step %d: failed to enqueue blobs: %v", i, err)
				return
			}
			<-wait

		case blobDoFunc:
			step(fetcher)

		case isWaitingAvailability:
			// Check expected hashes and peers are present
			for hash, peers := range step {
				if waitPeers, ok := fetcher.waitlist[hash]; !ok {
					t.Errorf("step %d: hash %x not in waitlist", i, hash)
					return
				} else {
					// Check expected peers are present
					for peer := range peers {
						if _, ok := waitPeers[peer]; !ok {
							t.Errorf("step %d: peer %s not waiting for hash %x", i, peer, hash)
							return
						}
					}
					// Check no unexpected peers are present
					for peer := range waitPeers {
						if _, ok := peers[peer]; !ok {
							t.Errorf("step %d: unexpected peer %s waiting for hash %x", i, peer, hash)
							return
						}
					}
				}
			}
			// Check no unexpected hashes in waitlist
			for hash := range fetcher.waitlist {
				if _, ok := step[hash]; !ok {
					t.Errorf("step %d: unexpected hash %x in waitlist", i, hash)
					return
				}
			}

		case isDecidedFull:
			for hash := range step {
				if _, ok := fetcher.full[hash]; !ok {
					t.Errorf("step %d: hash %x not decided for full request", i, hash)
					return
				}
			}

		case isDecidedPartial:
			for hash := range step {
				if _, ok := fetcher.partial[hash]; !ok {
					t.Errorf("step %d: hash %x not decided for partial request", i, hash)
					return
				}
			}

		case isBlobScheduled:
			// todo fetches
			// Check tracking (announces) - bidirectional verification
			for peer, announces := range step.announces {
				peerAnnounces := fetcher.announces[peer]
				if peerAnnounces == nil {
					t.Errorf("step %d: peer %s missing from announces", i, peer)
					continue
				}
				// Check expected announces are present
				for _, ann := range announces {
					if cellWithSeq, ok := peerAnnounces[ann.hash]; !ok {
						t.Errorf("step %d, peer %s: hash %x missing from announces", i, peer, ann.hash)
					} else if *cellWithSeq.cells != ann.custody {
						t.Errorf("step %d, peer %s, hash %x: custody mismatch in announces", i, peer, ann.hash)
					}
				}
				// Check no unexpected announces are present
				for hash := range peerAnnounces {
					found := false
					for _, ann := range announces {
						if ann.hash == hash {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("step %d, peer %s: unexpected hash %x in announces", i, peer, hash)
					}
				}
			}
			// Check no unexpected peers in announces
			for peer := range fetcher.announces {
				if _, ok := step.announces[peer]; !ok {
					t.Errorf("step %d: unexpected peer %s in announces", i, peer)
				}
			}

			// Check fetching (requests)
			for peer, requests := range step.fetching {
				peerRequests := fetcher.requests[peer]
				if peerRequests == nil {
					t.Errorf("step %d: peer %s missing from requests", i, peer)
					continue
				}
				// Check expected requests are present
				for _, req := range requests {
					found := false
					for _, cellReq := range peerRequests {
						for _, hash := range cellReq.txs {
							if hash == req.hash && *cellReq.cells == req.custody {
								found = true
								break
							}
						}
						if found {
							break
						}
					}
					if !found {
						t.Errorf("step %d, peer %s: hash %x with custody not found in requests", i, peer, req.hash)
					}
				}
				// (bidirectional) Check no unexpected requests are present
				for _, cellReq := range peerRequests {
					for _, hash := range cellReq.txs {
						found := false
						for _, req := range requests {
							if req.hash == hash && *cellReq.cells == req.custody {
								found = true
								break
							}
						}
						if !found {
							t.Errorf("step %d, peer %s: unexpected hash %x in requests", i, peer, hash)
						}
					}
				}
			}
			// Check no unexpected peers in requests
			for peer := range fetcher.requests {
				if _, ok := step.fetching[peer]; !ok {
					t.Errorf("step %d: unexpected peer %s in requests", i, peer)
				}
			}

			// Check internal consistency: alternates should match announces
			// For every hash being fetched, alternates should contain all peers who announced it
			for _, announces := range step.fetching {
				for _, announce := range announces {
					hash := announce.hash
					alternates := fetcher.alternates[hash]
					if alternates == nil {
						t.Errorf("step %d: hash %x missing from alternates", i, hash)
						continue
					}

					// Check that all peers with this hash in announces are in alternates with matching custody
					for peer, peerAnnounces := range fetcher.announces {
						if cellWithSeq := peerAnnounces[hash]; cellWithSeq != nil {
							if altCustody, ok := alternates[peer]; !ok {
								t.Errorf("step %d, hash %x: peer %s missing from alternates", i, hash, peer)
							} else if *altCustody != *cellWithSeq.cells {
								t.Errorf("step %d, hash %x, peer %s: custody bitmap mismatch in alternates", i, hash, peer)
							}
						}
					}

					// Check that all peers in alternates actually have this hash announced with matching custody
					for peer, altCustody := range alternates {
						if fetcher.announces[peer] == nil || fetcher.announces[peer][hash] == nil {
							t.Errorf("step %d, hash %x: peer %s extra in alternates", i, hash, peer)
						} else if cellWithSeq := fetcher.announces[peer][hash]; *cellWithSeq.cells != *altCustody {
							t.Errorf("step %d, hash %x, peer %s: custody bitmap mismatch between announces and alternates", i, hash, peer)
						}
					}
				}
			}

		case isFetching:
			// Check expected hashes are present in fetches
			for hash, expected := range step.hashes {
				if fetchStatus, ok := fetcher.fetches[hash]; !ok {
					t.Errorf("step %d: hash %x missing from fetches", i, hash)
				} else {
					// Check fetching bitmap
					if expected.fetching != nil {
						if fetchStatus.fetching == nil {
							t.Errorf("step %d, hash %x: fetching bitmap is nil", i, hash)
						} else if *fetchStatus.fetching != *expected.fetching {
							t.Errorf("step %d, hash %x: fetching bitmap mismatch", i, hash)
						}
					}

					// Check fetched indices
					if expected.fetched != nil {
						if len(fetchStatus.fetched) != len(expected.fetched) {
							t.Errorf("step %d, hash %x: fetched length mismatch, got %d, want %d", i, hash, len(fetchStatus.fetched), len(expected.fetched))
						} else {
							// Sort both slices before comparing
							gotFetched := make([]uint64, len(fetchStatus.fetched))
							copy(gotFetched, fetchStatus.fetched)
							slices.Sort(gotFetched)

							expectedFetched := make([]uint64, len(expected.fetched))
							copy(expectedFetched, expected.fetched)
							slices.Sort(expectedFetched)

							if !slices.Equal(gotFetched, expectedFetched) {
								t.Errorf("step %d, hash %x: fetched indices mismatch", i, hash)
							}
						}
					}
				}
			}
			// Check no unexpected hashes in fetches
			for hash := range fetcher.fetches {
				if _, ok := step.hashes[hash]; !ok {
					t.Errorf("step %d: unexpected hash %x in fetches", i, hash)
				}
			}

		case isCompleted:
			for _, hash := range step {
				if _, ok := fetcher.fetches[hash]; ok {
					t.Errorf("step %d: hash %x still in fetches (should be completed)", i, hash)
					return
				}
			}

		case isDropped:
			for _, peer := range step {
				if _, ok := fetcher.announces[peer]; ok {
					t.Errorf("step %d: peer %s still has announces (should be dropped)", i, peer)
					return
				}
			}

		case doWait:
			clock.Run(step.time)
			if step.step {
				<-wait
			}

		case doDrop:
			if err := fetcher.Drop(string(step)); err != nil {
				t.Errorf("step %d: %v", i, err)
			}
			<-wait

		default:
			t.Errorf("step %d: unknown step type %T", i, step)
			return
		}
	}
}
