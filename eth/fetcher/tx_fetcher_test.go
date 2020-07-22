// Copyright 2020 The go-ethereum Authors
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
	"errors"
	"math/big"
	"math/rand"
	"testing"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
)

var (
	// testTxs is a set of transactions to use during testing that have meaningful hashes.
	testTxs = []*types.Transaction{
		types.NewTransaction(5577006791947779410, common.Address{0x0f}, new(big.Int), 0, new(big.Int), nil),
		types.NewTransaction(15352856648520921629, common.Address{0xbb}, new(big.Int), 0, new(big.Int), nil),
		types.NewTransaction(3916589616287113937, common.Address{0x86}, new(big.Int), 0, new(big.Int), nil),
		types.NewTransaction(9828766684487745566, common.Address{0xac}, new(big.Int), 0, new(big.Int), nil),
	}
	// testTxsHashes is the hashes of the test transactions above
	testTxsHashes = []common.Hash{testTxs[0].Hash(), testTxs[1].Hash(), testTxs[2].Hash(), testTxs[3].Hash()}
)

type doTxNotify struct {
	peer   string
	hashes []common.Hash
}
type doTxEnqueue struct {
	peer   string
	txs    []*types.Transaction
	direct bool
}
type doWait struct {
	time time.Duration
	step bool
}
type doDrop string
type doFunc func()

type isWaiting map[string][]common.Hash
type isScheduled struct {
	tracking map[string][]common.Hash
	fetching map[string][]common.Hash
	dangling map[string][]common.Hash
}
type isUnderpriced int

// txFetcherTest represents a test scenario that can be executed by the test
// runner.
type txFetcherTest struct {
	init  func() *TxFetcher
	steps []interface{}
}

// Tests that transaction announcements are added to a waitlist, and none
// of them are scheduled for retrieval until the wait expires.
func TestTransactionFetcherWaiting(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				nil,
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Initial announcement to get something into the waitlist
			doTxNotify{peer: "A", hashes: []common.Hash{{0x01}, {0x02}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x01}, {0x02}},
			}),
			// Announce from a new peer to check that no overwrite happens
			doTxNotify{peer: "B", hashes: []common.Hash{{0x03}, {0x04}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x01}, {0x02}},
				"B": {{0x03}, {0x04}},
			}),
			// Announce clashing hashes but unique new peer
			doTxNotify{peer: "C", hashes: []common.Hash{{0x01}, {0x04}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x01}, {0x02}},
				"B": {{0x03}, {0x04}},
				"C": {{0x01}, {0x04}},
			}),
			// Announce existing and clashing hashes from existing peer
			doTxNotify{peer: "A", hashes: []common.Hash{{0x01}, {0x03}, {0x05}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x01}, {0x02}, {0x03}, {0x05}},
				"B": {{0x03}, {0x04}},
				"C": {{0x01}, {0x04}},
			}),
			isScheduled{tracking: nil, fetching: nil},

			// Wait for the arrival timeout which should move all expired items
			// from the wait list to the scheduler
			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}, {0x03}, {0x05}},
					"B": {{0x03}, {0x04}},
					"C": {{0x01}, {0x04}},
				},
				fetching: map[string][]common.Hash{ // Depends on deterministic test randomizer
					"A": {{0x02}, {0x03}, {0x05}},
					"C": {{0x01}, {0x04}},
				},
			},
			// Queue up a non-fetchable transaction and then trigger it with a new
			// peer (weird case to test 1 line in the fetcher)
			doTxNotify{peer: "C", hashes: []common.Hash{{0x06}, {0x07}}},
			isWaiting(map[string][]common.Hash{
				"C": {{0x06}, {0x07}},
			}),
			doWait{time: txArriveTimeout, step: true},
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}, {0x03}, {0x05}},
					"B": {{0x03}, {0x04}},
					"C": {{0x01}, {0x04}, {0x06}, {0x07}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x02}, {0x03}, {0x05}},
					"C": {{0x01}, {0x04}},
				},
			},
			doTxNotify{peer: "D", hashes: []common.Hash{{0x06}, {0x07}}},
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}, {0x03}, {0x05}},
					"B": {{0x03}, {0x04}},
					"C": {{0x01}, {0x04}, {0x06}, {0x07}},
					"D": {{0x06}, {0x07}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x02}, {0x03}, {0x05}},
					"C": {{0x01}, {0x04}},
					"D": {{0x06}, {0x07}},
				},
			},
		},
	})
}

// Tests that transaction announcements skip the waiting list if they are
// already scheduled.
func TestTransactionFetcherSkipWaiting(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				nil,
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Push an initial announcement through to the scheduled stage
			doTxNotify{peer: "A", hashes: []common.Hash{{0x01}, {0x02}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x01}, {0x02}},
			}),
			isScheduled{tracking: nil, fetching: nil},

			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
			},
			// Announce overlaps from the same peer, ensure the new ones end up
			// in stage one, and clashing ones don't get double tracked
			doTxNotify{peer: "A", hashes: []common.Hash{{0x02}, {0x03}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x03}},
			}),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
			},
			// Announce overlaps from a new peer, ensure new transactions end up
			// in stage one and clashing ones get tracked for the new peer
			doTxNotify{peer: "B", hashes: []common.Hash{{0x02}, {0x03}, {0x04}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x03}},
				"B": {{0x03}, {0x04}},
			}),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
					"B": {{0x02}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
			},
		},
	})
}

// Tests that only a single transaction request gets scheduled to a peer
// and subsequent announces block or get allotted to someone else.
func TestTransactionFetcherSingletonRequesting(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				nil,
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Push an initial announcement through to the scheduled stage
			doTxNotify{peer: "A", hashes: []common.Hash{{0x01}, {0x02}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x01}, {0x02}},
			}),
			isScheduled{tracking: nil, fetching: nil},

			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
			},
			// Announce a new set of transactions from the same peer and ensure
			// they do not start fetching since the peer is already busy
			doTxNotify{peer: "A", hashes: []common.Hash{{0x03}, {0x04}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x03}, {0x04}},
			}),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
			},
			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}, {0x03}, {0x04}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
			},
			// Announce a duplicate set of transactions from a new peer and ensure
			// uniquely new ones start downloading, even if clashing.
			doTxNotify{peer: "B", hashes: []common.Hash{{0x02}, {0x03}, {0x05}, {0x06}}},
			isWaiting(map[string][]common.Hash{
				"B": {{0x05}, {0x06}},
			}),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}, {0x03}, {0x04}},
					"B": {{0x02}, {0x03}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
					"B": {{0x03}},
				},
			},
		},
	})
}

// Tests that if a transaction retrieval fails, all the transactions get
// instantly schedule back to someone else or the announcements dropped
// if no alternate source is available.
func TestTransactionFetcherFailedRescheduling(t *testing.T) {
	// Create a channel to control when tx requests can fail
	proceed := make(chan struct{})

	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				nil,
				func(origin string, hashes []common.Hash) error {
					<-proceed
					return errors.New("peer disconnected")
				},
			)
		},
		steps: []interface{}{
			// Push an initial announcement through to the scheduled stage
			doTxNotify{peer: "A", hashes: []common.Hash{{0x01}, {0x02}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x01}, {0x02}},
			}),
			isScheduled{tracking: nil, fetching: nil},

			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
			},
			// While the original peer is stuck in the request, push in an second
			// data source.
			doTxNotify{peer: "B", hashes: []common.Hash{{0x02}}},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
					"B": {{0x02}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
			},
			// Wait until the original request fails and check that transactions
			// are either rescheduled or dropped
			doFunc(func() {
				proceed <- struct{}{} // Allow peer A to return the failure
			}),
			doWait{time: 0, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"B": {{0x02}},
				},
				fetching: map[string][]common.Hash{
					"B": {{0x02}},
				},
			},
			doFunc(func() {
				proceed <- struct{}{} // Allow peer B to return the failure
			}),
			doWait{time: 0, step: true},
			isWaiting(nil),
			isScheduled{nil, nil, nil},
		},
	})
}

// Tests that if a transaction retrieval succeeds, all alternate origins
// are cleaned up.
func TestTransactionFetcherCleanup(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Push an initial announcement through to the scheduled stage
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0]}},
			isWaiting(map[string][]common.Hash{
				"A": {testTxsHashes[0]},
			}),
			isScheduled{tracking: nil, fetching: nil},

			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
				fetching: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
			},
			// Request should be delivered
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0]}, direct: true},
			isScheduled{nil, nil, nil},
		},
	})
}

// Tests that if a transaction retrieval succeeds, but the response is empty (no
// transactions available, then all are nuked instead of being rescheduled (yes,
// this was a bug)).
func TestTransactionFetcherCleanupEmpty(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Push an initial announcement through to the scheduled stage
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0]}},
			isWaiting(map[string][]common.Hash{
				"A": {testTxsHashes[0]},
			}),
			isScheduled{tracking: nil, fetching: nil},

			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
				fetching: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
			},
			// Deliver an empty response and ensure the transaction is cleared, not rescheduled
			doTxEnqueue{peer: "A", txs: []*types.Transaction{}, direct: true},
			isScheduled{nil, nil, nil},
		},
	})
}

// Tests that non-returned transactions are either re-scheduled from a
// different peer, or self if they are after the cutoff point.
func TestTransactionFetcherMissingRescheduling(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Push an initial announcement through to the scheduled stage
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0], testTxsHashes[1], testTxsHashes[2]}},
			isWaiting(map[string][]common.Hash{
				"A": {testTxsHashes[0], testTxsHashes[1], testTxsHashes[2]},
			}),
			isScheduled{tracking: nil, fetching: nil},

			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[0], testTxsHashes[1], testTxsHashes[2]},
				},
				fetching: map[string][]common.Hash{
					"A": {testTxsHashes[0], testTxsHashes[1], testTxsHashes[2]},
				},
			},
			// Deliver the middle transaction requested, the one before which
			// should be dropped and the one after re-requested.
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0]}, direct: true}, // This depends on the deterministic random
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[2]},
				},
				fetching: map[string][]common.Hash{
					"A": {testTxsHashes[2]},
				},
			},
		},
	})
}

// Tests that out of two transactions, if one is missing and the last is
// delivered, the peer gets properly cleaned out from the internal state.
func TestTransactionFetcherMissingCleanup(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Push an initial announcement through to the scheduled stage
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0], testTxsHashes[1]}},
			isWaiting(map[string][]common.Hash{
				"A": {testTxsHashes[0], testTxsHashes[1]},
			}),
			isScheduled{tracking: nil, fetching: nil},

			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[0], testTxsHashes[1]},
				},
				fetching: map[string][]common.Hash{
					"A": {testTxsHashes[0], testTxsHashes[1]},
				},
			},
			// Deliver the middle transaction requested, the one before which
			// should be dropped and the one after re-requested.
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[1]}, direct: true}, // This depends on the deterministic random
			isScheduled{nil, nil, nil},
		},
	})
}

// Tests that transaction broadcasts properly clean up announcements.
func TestTransactionFetcherBroadcasts(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Set up three transactions to be in different stats, waiting, queued and fetching
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0]}},
			doWait{time: txArriveTimeout, step: true},
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[1]}},
			doWait{time: txArriveTimeout, step: true},
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[2]}},

			isWaiting(map[string][]common.Hash{
				"A": {testTxsHashes[2]},
			}),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[0], testTxsHashes[1]},
				},
				fetching: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
			},
			// Broadcast all the transactions and ensure everything gets cleaned
			// up, but the dangling request is left alone to avoid doing multiple
			// concurrent requests.
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0], testTxs[1], testTxs[2]}, direct: false},
			isWaiting(nil),
			isScheduled{
				tracking: nil,
				fetching: nil,
				dangling: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
			},
			// Deliver the requested hashes
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0], testTxs[1], testTxs[2]}, direct: true},
			isScheduled{nil, nil, nil},
		},
	})
}

// Tests that the waiting list timers properly reset and reschedule.
func TestTransactionFetcherWaitTimerResets(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				nil,
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			doTxNotify{peer: "A", hashes: []common.Hash{{0x01}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x01}},
			}),
			isScheduled{nil, nil, nil},
			doWait{time: txArriveTimeout / 2, step: false},
			isWaiting(map[string][]common.Hash{
				"A": {{0x01}},
			}),
			isScheduled{nil, nil, nil},

			doTxNotify{peer: "A", hashes: []common.Hash{{0x02}}},
			isWaiting(map[string][]common.Hash{
				"A": {{0x01}, {0x02}},
			}),
			isScheduled{nil, nil, nil},
			doWait{time: txArriveTimeout / 2, step: true},
			isWaiting(map[string][]common.Hash{
				"A": {{0x02}},
			}),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}},
				},
			},

			doWait{time: txArriveTimeout / 2, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}},
				},
			},
		},
	})
}

// Tests that if a transaction request is not replied to, it will time
// out and be re-scheduled for someone else.
func TestTransactionFetcherTimeoutRescheduling(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Push an initial announcement through to the scheduled stage
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0]}},
			isWaiting(map[string][]common.Hash{
				"A": {testTxsHashes[0]},
			}),
			isScheduled{tracking: nil, fetching: nil},

			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
				fetching: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
			},
			// Wait until the delivery times out, everything should be cleaned up
			doWait{time: txFetchTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: nil,
				fetching: nil,
				dangling: map[string][]common.Hash{
					"A": {},
				},
			},
			// Ensure that followup announcements don't get scheduled
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[1]}},
			doWait{time: txArriveTimeout, step: true},
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[1]},
				},
				fetching: nil,
				dangling: map[string][]common.Hash{
					"A": {},
				},
			},
			// If the dangling request arrives a bit later, do not choke
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0]}, direct: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[1]},
				},
				fetching: map[string][]common.Hash{
					"A": {testTxsHashes[1]},
				},
			},
		},
	})
}

// Tests that the fetching timeout timers properly reset and reschedule.
func TestTransactionFetcherTimeoutTimerResets(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				nil,
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			doTxNotify{peer: "A", hashes: []common.Hash{{0x01}}},
			doWait{time: txArriveTimeout, step: true},
			doTxNotify{peer: "B", hashes: []common.Hash{{0x02}}},
			doWait{time: txArriveTimeout, step: true},

			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}},
					"B": {{0x02}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}},
					"B": {{0x02}},
				},
			},
			doWait{time: txFetchTimeout - txArriveTimeout, step: true},
			isScheduled{
				tracking: map[string][]common.Hash{
					"B": {{0x02}},
				},
				fetching: map[string][]common.Hash{
					"B": {{0x02}},
				},
				dangling: map[string][]common.Hash{
					"A": {},
				},
			},
			doWait{time: txArriveTimeout, step: true},
			isScheduled{
				tracking: nil,
				fetching: nil,
				dangling: map[string][]common.Hash{
					"A": {},
					"B": {},
				},
			},
		},
	})
}

// Tests that if thousands of transactions are announces, only a small
// number of them will be requested at a time.
func TestTransactionFetcherRateLimiting(t *testing.T) {
	// Create a slew of transactions and to announce them
	var hashes []common.Hash
	for i := 0; i < maxTxAnnounces; i++ {
		hashes = append(hashes, common.Hash{byte(i / 256), byte(i % 256)})
	}

	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				nil,
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Announce all the transactions, wait a bit and ensure only a small
			// percentage gets requested
			doTxNotify{peer: "A", hashes: hashes},
			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": hashes,
				},
				fetching: map[string][]common.Hash{
					"A": hashes[1643 : 1643+maxTxRetrievals],
				},
			},
		},
	})
}

// Tests that then number of transactions a peer is allowed to announce and/or
// request at the same time is hard capped.
func TestTransactionFetcherDoSProtection(t *testing.T) {
	// Create a slew of transactions and to announce them
	var hashesA []common.Hash
	for i := 0; i < maxTxAnnounces+1; i++ {
		hashesA = append(hashesA, common.Hash{0x01, byte(i / 256), byte(i % 256)})
	}
	var hashesB []common.Hash
	for i := 0; i < maxTxAnnounces+1; i++ {
		hashesB = append(hashesB, common.Hash{0x02, byte(i / 256), byte(i % 256)})
	}
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				nil,
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Announce half of the transaction and wait for them to be scheduled
			doTxNotify{peer: "A", hashes: hashesA[:maxTxAnnounces/2]},
			doTxNotify{peer: "B", hashes: hashesB[:maxTxAnnounces/2-1]},
			doWait{time: txArriveTimeout, step: true},

			// Announce the second half and keep them in the wait list
			doTxNotify{peer: "A", hashes: hashesA[maxTxAnnounces/2 : maxTxAnnounces]},
			doTxNotify{peer: "B", hashes: hashesB[maxTxAnnounces/2-1 : maxTxAnnounces-1]},

			// Ensure the hashes are split half and half
			isWaiting(map[string][]common.Hash{
				"A": hashesA[maxTxAnnounces/2 : maxTxAnnounces],
				"B": hashesB[maxTxAnnounces/2-1 : maxTxAnnounces-1],
			}),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": hashesA[:maxTxAnnounces/2],
					"B": hashesB[:maxTxAnnounces/2-1],
				},
				fetching: map[string][]common.Hash{
					"A": hashesA[1643 : 1643+maxTxRetrievals],
					"B": append(append([]common.Hash{}, hashesB[maxTxAnnounces/2-3:maxTxAnnounces/2-1]...), hashesB[:maxTxRetrievals-2]...),
				},
			},
			// Ensure that adding even one more hash results in dropping the hash
			doTxNotify{peer: "A", hashes: []common.Hash{hashesA[maxTxAnnounces]}},
			doTxNotify{peer: "B", hashes: hashesB[maxTxAnnounces-1 : maxTxAnnounces+1]},

			isWaiting(map[string][]common.Hash{
				"A": hashesA[maxTxAnnounces/2 : maxTxAnnounces],
				"B": hashesB[maxTxAnnounces/2-1 : maxTxAnnounces],
			}),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": hashesA[:maxTxAnnounces/2],
					"B": hashesB[:maxTxAnnounces/2-1],
				},
				fetching: map[string][]common.Hash{
					"A": hashesA[1643 : 1643+maxTxRetrievals],
					"B": append(append([]common.Hash{}, hashesB[maxTxAnnounces/2-3:maxTxAnnounces/2-1]...), hashesB[:maxTxRetrievals-2]...),
				},
			},
		},
	})
}

// Tests that underpriced transactions don't get rescheduled after being rejected.
func TestTransactionFetcherUnderpricedDedup(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					errs := make([]error, len(txs))
					for i := 0; i < len(errs); i++ {
						if i%2 == 0 {
							errs[i] = core.ErrUnderpriced
						} else {
							errs[i] = core.ErrReplaceUnderpriced
						}
					}
					return errs
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Deliver a transaction through the fetcher, but reject as underpriced
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0], testTxsHashes[1]}},
			doWait{time: txArriveTimeout, step: true},
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0], testTxs[1]}, direct: true},
			isScheduled{nil, nil, nil},

			// Try to announce the transaction again, ensure it's not scheduled back
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0], testTxsHashes[1], testTxsHashes[2]}}, // [2] is needed to force a step in the fetcher
			isWaiting(map[string][]common.Hash{
				"A": {testTxsHashes[2]},
			}),
			isScheduled{nil, nil, nil},
		},
	})
}

// Tests that underpriced transactions don't get rescheduled after being rejected,
// but at the same time there's a hard cap on the number of transactions that are
// tracked.
func TestTransactionFetcherUnderpricedDoSProtection(t *testing.T) {
	// Temporarily disable fetch timeouts as they massively mess up the simulated clock
	defer func(timeout time.Duration) { txFetchTimeout = timeout }(txFetchTimeout)
	txFetchTimeout = 24 * time.Hour

	// Create a slew of transactions to max out the underpriced set
	var txs []*types.Transaction
	for i := 0; i < maxTxUnderpricedSetSize+1; i++ {
		txs = append(txs, types.NewTransaction(rand.Uint64(), common.Address{byte(rand.Intn(256))}, new(big.Int), 0, new(big.Int), nil))
	}
	hashes := make([]common.Hash, len(txs))
	for i, tx := range txs {
		hashes[i] = tx.Hash()
	}
	// Generate a set of steps to announce and deliver the entire set of transactions
	var steps []interface{}
	for i := 0; i < maxTxUnderpricedSetSize/maxTxRetrievals; i++ {
		steps = append(steps, doTxNotify{peer: "A", hashes: hashes[i*maxTxRetrievals : (i+1)*maxTxRetrievals]})
		steps = append(steps, isWaiting(map[string][]common.Hash{
			"A": hashes[i*maxTxRetrievals : (i+1)*maxTxRetrievals],
		}))
		steps = append(steps, doWait{time: txArriveTimeout, step: true})
		steps = append(steps, isScheduled{
			tracking: map[string][]common.Hash{
				"A": hashes[i*maxTxRetrievals : (i+1)*maxTxRetrievals],
			},
			fetching: map[string][]common.Hash{
				"A": hashes[i*maxTxRetrievals : (i+1)*maxTxRetrievals],
			},
		})
		steps = append(steps, doTxEnqueue{peer: "A", txs: txs[i*maxTxRetrievals : (i+1)*maxTxRetrievals], direct: true})
		steps = append(steps, isWaiting(nil))
		steps = append(steps, isScheduled{nil, nil, nil})
		steps = append(steps, isUnderpriced((i+1)*maxTxRetrievals))
	}
	testTransactionFetcher(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					errs := make([]error, len(txs))
					for i := 0; i < len(errs); i++ {
						errs[i] = core.ErrUnderpriced
					}
					return errs
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: append(steps, []interface{}{
			// The preparation of the test has already been done in `steps`, add the last check
			doTxNotify{peer: "A", hashes: []common.Hash{hashes[maxTxUnderpricedSetSize]}},
			doWait{time: txArriveTimeout, step: true},
			doTxEnqueue{peer: "A", txs: []*types.Transaction{txs[maxTxUnderpricedSetSize]}, direct: true},
			isUnderpriced(maxTxUnderpricedSetSize),
		}...),
	})
}

// Tests that unexpected deliveries don't corrupt the internal state.
func TestTransactionFetcherOutOfBoundDeliveries(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Deliver something out of the blue
			isWaiting(nil),
			isScheduled{nil, nil, nil},
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0]}, direct: false},
			isWaiting(nil),
			isScheduled{nil, nil, nil},

			// Set up a few hashes into various stages
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0]}},
			doWait{time: txArriveTimeout, step: true},
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[1]}},
			doWait{time: txArriveTimeout, step: true},
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[2]}},

			isWaiting(map[string][]common.Hash{
				"A": {testTxsHashes[2]},
			}),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[0], testTxsHashes[1]},
				},
				fetching: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
			},
			// Deliver everything and more out of the blue
			doTxEnqueue{peer: "B", txs: []*types.Transaction{testTxs[0], testTxs[1], testTxs[2], testTxs[3]}, direct: true},
			isWaiting(nil),
			isScheduled{
				tracking: nil,
				fetching: nil,
				dangling: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
			},
		},
	})
}

// Tests that dropping a peer cleans out all internal data structures in all the
// live or danglng stages.
func TestTransactionFetcherDrop(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Set up a few hashes into various stages
			doTxNotify{peer: "A", hashes: []common.Hash{{0x01}}},
			doWait{time: txArriveTimeout, step: true},
			doTxNotify{peer: "A", hashes: []common.Hash{{0x02}}},
			doWait{time: txArriveTimeout, step: true},
			doTxNotify{peer: "A", hashes: []common.Hash{{0x03}}},

			isWaiting(map[string][]common.Hash{
				"A": {{0x03}},
			}),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}, {0x02}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}},
				},
			},
			// Drop the peer and ensure everything's cleaned out
			doDrop("A"),
			isWaiting(nil),
			isScheduled{nil, nil, nil},

			// Push the node into a dangling (timeout) state
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0]}},
			doWait{time: txArriveTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
				fetching: map[string][]common.Hash{
					"A": {testTxsHashes[0]},
				},
			},
			doWait{time: txFetchTimeout, step: true},
			isWaiting(nil),
			isScheduled{
				tracking: nil,
				fetching: nil,
				dangling: map[string][]common.Hash{
					"A": {},
				},
			},
			// Drop the peer and ensure everything's cleaned out
			doDrop("A"),
			isWaiting(nil),
			isScheduled{nil, nil, nil},
		},
	})
}

// Tests that dropping a peer instantly reschedules failed announcements to any
// available peer.
func TestTransactionFetcherDropRescheduling(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Set up a few hashes into various stages
			doTxNotify{peer: "A", hashes: []common.Hash{{0x01}}},
			doWait{time: txArriveTimeout, step: true},
			doTxNotify{peer: "B", hashes: []common.Hash{{0x01}}},

			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"A": {{0x01}},
					"B": {{0x01}},
				},
				fetching: map[string][]common.Hash{
					"A": {{0x01}},
				},
			},
			// Drop the peer and ensure everything's cleaned out
			doDrop("A"),
			isWaiting(nil),
			isScheduled{
				tracking: map[string][]common.Hash{
					"B": {{0x01}},
				},
				fetching: map[string][]common.Hash{
					"B": {{0x01}},
				},
			},
		},
	})
}

// This test reproduces a crash caught by the fuzzer. The root cause was a
// dangling transaction timing out and clashing on readd with a concurrently
// announced one.
func TestTransactionFetcherFuzzCrash01(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Get a transaction into fetching mode and make it dangling with a broadcast
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0]}},
			doWait{time: txArriveTimeout, step: true},
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0]}},

			// Notify the dangling transaction once more and crash via a timeout
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0]}},
			doWait{time: txFetchTimeout, step: true},
		},
	})
}

// This test reproduces a crash caught by the fuzzer. The root cause was a
// dangling transaction getting peer-dropped and clashing on readd with a
// concurrently announced one.
func TestTransactionFetcherFuzzCrash02(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Get a transaction into fetching mode and make it dangling with a broadcast
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0]}},
			doWait{time: txArriveTimeout, step: true},
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0]}},

			// Notify the dangling transaction once more, re-fetch, and crash via a drop and timeout
			doTxNotify{peer: "B", hashes: []common.Hash{testTxsHashes[0]}},
			doWait{time: txArriveTimeout, step: true},
			doDrop("A"),
			doWait{time: txFetchTimeout, step: true},
		},
	})
}

// This test reproduces a crash caught by the fuzzer. The root cause was a
// dangling transaction getting rescheduled via a partial delivery, clashing
// with a concurrent notify.
func TestTransactionFetcherFuzzCrash03(t *testing.T) {
	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error { return nil },
			)
		},
		steps: []interface{}{
			// Get a transaction into fetching mode and make it dangling with a broadcast
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0], testTxsHashes[1]}},
			doWait{time: txFetchTimeout, step: true},
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0], testTxs[1]}},

			// Notify the dangling transaction once more, partially deliver, clash&crash with a timeout
			doTxNotify{peer: "B", hashes: []common.Hash{testTxsHashes[0]}},
			doWait{time: txArriveTimeout, step: true},

			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[1]}, direct: true},
			doWait{time: txFetchTimeout, step: true},
		},
	})
}

// This test reproduces a crash caught by the fuzzer. The root cause was a
// dangling transaction getting rescheduled via a disconnect, clashing with
// a concurrent notify.
func TestTransactionFetcherFuzzCrash04(t *testing.T) {
	// Create a channel to control when tx requests can fail
	proceed := make(chan struct{})

	testTransactionFetcherParallel(t, txFetcherTest{
		init: func() *TxFetcher {
			return NewTxFetcher(
				func(common.Hash) bool { return false },
				func(txs []*types.Transaction) []error {
					return make([]error, len(txs))
				},
				func(string, []common.Hash) error {
					<-proceed
					return errors.New("peer disconnected")
				},
			)
		},
		steps: []interface{}{
			// Get a transaction into fetching mode and make it dangling with a broadcast
			doTxNotify{peer: "A", hashes: []common.Hash{testTxsHashes[0]}},
			doWait{time: txArriveTimeout, step: true},
			doTxEnqueue{peer: "A", txs: []*types.Transaction{testTxs[0]}},

			// Notify the dangling transaction once more, re-fetch, and crash via an in-flight disconnect
			doTxNotify{peer: "B", hashes: []common.Hash{testTxsHashes[0]}},
			doWait{time: txArriveTimeout, step: true},
			doFunc(func() {
				proceed <- struct{}{} // Allow peer A to return the failure
			}),
			doWait{time: 0, step: true},
			doWait{time: txFetchTimeout, step: true},
		},
	})
}

func testTransactionFetcherParallel(t *testing.T, tt txFetcherTest) {
	t.Parallel()
	testTransactionFetcher(t, tt)
}

func testTransactionFetcher(t *testing.T, tt txFetcherTest) {
	// Create a fetcher and hook into it's simulated fields
	clock := new(mclock.Simulated)
	wait := make(chan struct{})

	fetcher := tt.init()
	fetcher.clock = clock
	fetcher.step = wait
	fetcher.rand = rand.New(rand.NewSource(0x3a29))

	fetcher.Start()
	defer fetcher.Stop()

	// Crunch through all the test steps and execute them
	for i, step := range tt.steps {
		switch step := step.(type) {
		case doTxNotify:
			if err := fetcher.Notify(step.peer, step.hashes); err != nil {
				t.Errorf("step %d: %v", i, err)
			}
			<-wait // Fetcher needs to process this, wait until it's done
			select {
			case <-wait:
				panic("wtf")
			case <-time.After(time.Millisecond):
			}

		case doTxEnqueue:
			if err := fetcher.Enqueue(step.peer, step.txs, step.direct); err != nil {
				t.Errorf("step %d: %v", i, err)
			}
			<-wait // Fetcher needs to process this, wait until it's done

		case doWait:
			clock.Run(step.time)
			if step.step {
				<-wait // Fetcher supposed to do something, wait until it's done
			}

		case doDrop:
			if err := fetcher.Drop(string(step)); err != nil {
				t.Errorf("step %d: %v", i, err)
			}
			<-wait // Fetcher needs to process this, wait until it's done

		case doFunc:
			step()

		case isWaiting:
			// We need to check that the waiting list (stage 1) internals
			// match with the expected set. Check the peer->hash mappings
			// first.
			for peer, hashes := range step {
				waiting := fetcher.waitslots[peer]
				if waiting == nil {
					t.Errorf("step %d: peer %s missing from waitslots", i, peer)
					continue
				}
				for _, hash := range hashes {
					if _, ok := waiting[hash]; !ok {
						t.Errorf("step %d, peer %s: hash %x missing from waitslots", i, peer, hash)
					}
				}
				for hash := range waiting {
					if !containsHash(hashes, hash) {
						t.Errorf("step %d, peer %s: hash %x extra in waitslots", i, peer, hash)
					}
				}
			}
			for peer := range fetcher.waitslots {
				if _, ok := step[peer]; !ok {
					t.Errorf("step %d: peer %s extra in waitslots", i, peer)
				}
			}
			// Peer->hash sets correct, check the hash->peer and timeout sets
			for peer, hashes := range step {
				for _, hash := range hashes {
					if _, ok := fetcher.waitlist[hash][peer]; !ok {
						t.Errorf("step %d, hash %x: peer %s missing from waitlist", i, hash, peer)
					}
					if _, ok := fetcher.waittime[hash]; !ok {
						t.Errorf("step %d: hash %x missing from waittime", i, hash)
					}
				}
			}
			for hash, peers := range fetcher.waitlist {
				if len(peers) == 0 {
					t.Errorf("step %d, hash %x: empty peerset in waitlist", i, hash)
				}
				for peer := range peers {
					if !containsHash(step[peer], hash) {
						t.Errorf("step %d, hash %x: peer %s extra in waitlist", i, hash, peer)
					}
				}
			}
			for hash := range fetcher.waittime {
				var found bool
				for _, hashes := range step {
					if containsHash(hashes, hash) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("step %d,: hash %x extra in waittime", i, hash)
				}
			}

		case isScheduled:
			// Check that all scheduled announces are accounted for and no
			// extra ones are present.
			for peer, hashes := range step.tracking {
				scheduled := fetcher.announces[peer]
				if scheduled == nil {
					t.Errorf("step %d: peer %s missing from announces", i, peer)
					continue
				}
				for _, hash := range hashes {
					if _, ok := scheduled[hash]; !ok {
						t.Errorf("step %d, peer %s: hash %x missing from announces", i, peer, hash)
					}
				}
				for hash := range scheduled {
					if !containsHash(hashes, hash) {
						t.Errorf("step %d, peer %s: hash %x extra in announces", i, peer, hash)
					}
				}
			}
			for peer := range fetcher.announces {
				if _, ok := step.tracking[peer]; !ok {
					t.Errorf("step %d: peer %s extra in announces", i, peer)
				}
			}
			// Check that all announces required to be fetching are in the
			// appropriate sets
			for peer, hashes := range step.fetching {
				request := fetcher.requests[peer]
				if request == nil {
					t.Errorf("step %d: peer %s missing from requests", i, peer)
					continue
				}
				for _, hash := range hashes {
					if !containsHash(request.hashes, hash) {
						t.Errorf("step %d, peer %s: hash %x missing from requests", i, peer, hash)
					}
				}
				for _, hash := range request.hashes {
					if !containsHash(hashes, hash) {
						t.Errorf("step %d, peer %s: hash %x extra in requests", i, peer, hash)
					}
				}
			}
			for peer := range fetcher.requests {
				if _, ok := step.fetching[peer]; !ok {
					if _, ok := step.dangling[peer]; !ok {
						t.Errorf("step %d: peer %s extra in requests", i, peer)
					}
				}
			}
			for peer, hashes := range step.fetching {
				for _, hash := range hashes {
					if _, ok := fetcher.fetching[hash]; !ok {
						t.Errorf("step %d, peer %s: hash %x missing from fetching", i, peer, hash)
					}
				}
			}
			for hash := range fetcher.fetching {
				var found bool
				for _, req := range fetcher.requests {
					if containsHash(req.hashes, hash) {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("step %d: hash %x extra in fetching", i, hash)
				}
			}
			for _, hashes := range step.fetching {
				for _, hash := range hashes {
					alternates := fetcher.alternates[hash]
					if alternates == nil {
						t.Errorf("step %d: hash %x missing from alternates", i, hash)
						continue
					}
					for peer := range alternates {
						if _, ok := fetcher.announces[peer]; !ok {
							t.Errorf("step %d: peer %s extra in alternates", i, peer)
							continue
						}
						if _, ok := fetcher.announces[peer][hash]; !ok {
							t.Errorf("step %d, peer %s: hash %x extra in alternates", i, hash, peer)
							continue
						}
					}
					for p := range fetcher.announced[hash] {
						if _, ok := alternates[p]; !ok {
							t.Errorf("step %d, hash %x: peer %s missing from alternates", i, hash, p)
							continue
						}
					}
				}
			}
			for peer, hashes := range step.dangling {
				request := fetcher.requests[peer]
				if request == nil {
					t.Errorf("step %d: peer %s missing from requests", i, peer)
					continue
				}
				for _, hash := range hashes {
					if !containsHash(request.hashes, hash) {
						t.Errorf("step %d, peer %s: hash %x missing from requests", i, peer, hash)
					}
				}
				for _, hash := range request.hashes {
					if !containsHash(hashes, hash) {
						t.Errorf("step %d, peer %s: hash %x extra in requests", i, peer, hash)
					}
				}
			}
			// Check that all transaction announces that are scheduled for
			// retrieval but not actively being downloaded are tracked only
			// in the stage 2 `announced` map.
			var queued []common.Hash
			for _, hashes := range step.tracking {
				for _, hash := range hashes {
					var found bool
					for _, hs := range step.fetching {
						if containsHash(hs, hash) {
							found = true
							break
						}
					}
					if !found {
						queued = append(queued, hash)
					}
				}
			}
			for _, hash := range queued {
				if _, ok := fetcher.announced[hash]; !ok {
					t.Errorf("step %d: hash %x missing from announced", i, hash)
				}
			}
			for hash := range fetcher.announced {
				if !containsHash(queued, hash) {
					t.Errorf("step %d: hash %x extra in announced", i, hash)
				}
			}

		case isUnderpriced:
			if fetcher.underpriced.Cardinality() != int(step) {
				t.Errorf("step %d: underpriced set size mismatch: have %d, want %d", i, fetcher.underpriced.Cardinality(), step)
			}

		default:
			t.Fatalf("step %d: unknown step type %T", i, step)
		}
		// After every step, cross validate the internal uniqueness invariants
		// between stage one and stage two.
		for hash := range fetcher.waittime {
			if _, ok := fetcher.announced[hash]; ok {
				t.Errorf("step %d: hash %s present in both stage 1 and 2", i, hash)
			}
		}
	}
}

// containsHash returns whether a hash is contained within a hash slice.
func containsHash(slice []common.Hash, hash common.Hash) bool {
	for _, have := range slice {
		if have == hash {
			return true
		}
	}
	return false
}
