// Copyright 2019 The go-ethereum Authors
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
	"bytes"
	"errors"
	"fmt"
	mrand "math/rand"
	"sort"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/txpool"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

const (
	// maxTxAnnounces is the maximum number of unique transaction a peer
	// can announce in a short time.
	maxTxAnnounces = 4096

	// maxTxRetrievals is the maximum transaction number can be fetched in one
	// request. The rationale to pick 256 is:
	//   - In eth protocol, the softResponseLimit is 2MB. Nowadays according to
	//     Etherscan the average transaction size is around 200B, so in theory
	//     we can include lots of transaction in a single protocol packet.
	//   - However the maximum size of a single transaction is raised to 128KB,
	//     so pick a middle value here to ensure we can maximize the efficiency
	//     of the retrieval and response size overflow won't happen in most cases.
	maxTxRetrievals = 256

	// maxTxUnderpricedSetSize is the size of the underpriced transaction set that
	// is used to track recent transactions that have been dropped so we don't
	// re-request them.
	maxTxUnderpricedSetSize = 32768

	// txArriveTimeout is the time allowance before an announced transaction is
	// explicitly requested.
	txArriveTimeout = 500 * time.Millisecond

	// txGatherSlack is the interval used to collate almost-expired announces
	// with network fetches.
	txGatherSlack = 100 * time.Millisecond
)

var (
	// txFetchTimeout is the maximum allotted time to return an explicitly
	// requested transaction.
	txFetchTimeout = 5 * time.Second
)

var (
	txAnnounceInMeter          = metrics.NewRegisteredMeter("eth/fetcher/transaction/announces/in", nil)
	txAnnounceKnownMeter       = metrics.NewRegisteredMeter("eth/fetcher/transaction/announces/known", nil)
	txAnnounceUnderpricedMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/announces/underpriced", nil)
	txAnnounceDOSMeter         = metrics.NewRegisteredMeter("eth/fetcher/transaction/announces/dos", nil)

	txBroadcastInMeter          = metrics.NewRegisteredMeter("eth/fetcher/transaction/broadcasts/in", nil)
	txBroadcastKnownMeter       = metrics.NewRegisteredMeter("eth/fetcher/transaction/broadcasts/known", nil)
	txBroadcastUnderpricedMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/broadcasts/underpriced", nil)
	txBroadcastOtherRejectMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/broadcasts/otherreject", nil)

	txRequestOutMeter     = metrics.NewRegisteredMeter("eth/fetcher/transaction/request/out", nil)
	txRequestFailMeter    = metrics.NewRegisteredMeter("eth/fetcher/transaction/request/fail", nil)
	txRequestDoneMeter    = metrics.NewRegisteredMeter("eth/fetcher/transaction/request/done", nil)
	txRequestTimeoutMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/request/timeout", nil)

	txReplyInMeter          = metrics.NewRegisteredMeter("eth/fetcher/transaction/replies/in", nil)
	txReplyKnownMeter       = metrics.NewRegisteredMeter("eth/fetcher/transaction/replies/known", nil)
	txReplyUnderpricedMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/replies/underpriced", nil)
	txReplyOtherRejectMeter = metrics.NewRegisteredMeter("eth/fetcher/transaction/replies/otherreject", nil)

	txFetcherWaitingPeers   = metrics.NewRegisteredGauge("eth/fetcher/transaction/waiting/peers", nil)
	txFetcherWaitingHashes  = metrics.NewRegisteredGauge("eth/fetcher/transaction/waiting/hashes", nil)
	txFetcherQueueingPeers  = metrics.NewRegisteredGauge("eth/fetcher/transaction/queueing/peers", nil)
	txFetcherQueueingHashes = metrics.NewRegisteredGauge("eth/fetcher/transaction/queueing/hashes", nil)
	txFetcherFetchingPeers  = metrics.NewRegisteredGauge("eth/fetcher/transaction/fetching/peers", nil)
	txFetcherFetchingHashes = metrics.NewRegisteredGauge("eth/fetcher/transaction/fetching/hashes", nil)
)

// txAnnounce is the notification of the availability of a batch
// of new transactions in the network.
type txAnnounce struct {
	origin string        // Identifier of the peer originating the notification
	hashes []common.Hash // Batch of transaction hashes being announced
}

// txRequest represents an in-flight transaction retrieval request destined to
// a specific peers.
type txRequest struct {
	hashes []common.Hash            // Transactions having been requested
	stolen map[common.Hash]struct{} // Deliveries by someone else (don't re-request)
	time   mclock.AbsTime           // Timestamp of the request
}

// txDelivery is the notification that a batch of transactions have been added
// to the pool and should be untracked.
type txDelivery struct {
	origin string        // Identifier of the peer originating the notification
	hashes []common.Hash // Batch of transaction hashes having been delivered
	direct bool          // Whether this is a direct reply or a broadcast
}

// txDrop is the notification that a peer has disconnected.
type txDrop struct {
	peer string
}

// TxFetcher is responsible for retrieving new transaction based on announcements.
//
// The fetcher operates in 3 stages:
//   - Transactions that are newly discovered are moved into a wait list.
//   - After ~500ms passes, transactions from the wait list that have not been
//     broadcast to us in whole are moved into a queueing area.
//   - When a connected peer doesn't have in-flight retrieval requests, any
//     transaction queued up (and announced by the peer) are allocated to the
//     peer and moved into a fetching status until it's fulfilled or fails.
//
// The invariants of the fetcher are:
//   - Each tracked transaction (hash) must only be present in one of the
//     three stages. This ensures that the fetcher operates akin to a finite
//     state automata and there's do data leak.
//   - Each peer that announced transactions may be scheduled retrievals, but
//     only ever one concurrently. This ensures we can immediately know what is
//     missing from a reply and reschedule it.
type TxFetcher struct {
	notify  chan *txAnnounce
	cleanup chan *txDelivery
	drop    chan *txDrop
	quit    chan struct{}

	underpriced mapset.Set[common.Hash] // Transactions discarded as too cheap (don't re-fetch)

	// Stage 1: Waiting lists for newly discovered transactions that might be
	// broadcast without needing explicit request/reply round trips.
	waitlist  map[common.Hash]map[string]struct{} // Transactions waiting for an potential broadcast
	waittime  map[common.Hash]mclock.AbsTime      // Timestamps when transactions were added to the waitlist
	waitslots map[string]map[common.Hash]struct{} // Waiting announcements grouped by peer (DoS protection)

	// Stage 2: Queue of transactions that waiting to be allocated to some peer
	// to be retrieved directly.
	announces map[string]map[common.Hash]struct{} // Set of announced transactions, grouped by origin peer
	announced map[common.Hash]map[string]struct{} // Set of download locations, grouped by transaction hash

	// Stage 3: Set of transactions currently being retrieved, some which may be
	// fulfilled and some rescheduled. Note, this step shares 'announces' from the
	// previous stage to avoid having to duplicate (need it for DoS checks).
	fetching   map[common.Hash]string              // Transaction set currently being retrieved
	requests   map[string]*txRequest               // In-flight transaction retrievals
	alternates map[common.Hash]map[string]struct{} // In-flight transaction alternate origins if retrieval fails

	// Callbacks
	hasTx    func(common.Hash) bool              // Retrieves a tx from the local txpool
	addTxs   func([]*txpool.Transaction) []error // Insert a batch of transactions into local txpool
	fetchTxs func(string, []common.Hash) error   // Retrieves a set of txs from a remote peer

	step  chan struct{} // Notification channel when the fetcher loop iterates
	clock mclock.Clock  // Time wrapper to simulate in tests
	rand  *mrand.Rand   // Randomizer to use in tests instead of map range loops (soft-random)
}

// NewTxFetcher creates a transaction fetcher to retrieve transaction
// based on hash announcements.
func NewTxFetcher(hasTx func(common.Hash) bool, addTxs func([]*txpool.Transaction) []error, fetchTxs func(string, []common.Hash) error) *TxFetcher {
	return NewTxFetcherForTests(hasTx, addTxs, fetchTxs, mclock.System{}, nil)
}

// NewTxFetcherForTests is a testing method to mock out the realtime clock with
// a simulated version and the internal randomness with a deterministic one.
func NewTxFetcherForTests(
	hasTx func(common.Hash) bool, addTxs func([]*txpool.Transaction) []error, fetchTxs func(string, []common.Hash) error,
	clock mclock.Clock, rand *mrand.Rand) *TxFetcher {
	return &TxFetcher{
		notify:      make(chan *txAnnounce),
		cleanup:     make(chan *txDelivery),
		drop:        make(chan *txDrop),
		quit:        make(chan struct{}),
		waitlist:    make(map[common.Hash]map[string]struct{}),
		waittime:    make(map[common.Hash]mclock.AbsTime),
		waitslots:   make(map[string]map[common.Hash]struct{}),
		announces:   make(map[string]map[common.Hash]struct{}),
		announced:   make(map[common.Hash]map[string]struct{}),
		fetching:    make(map[common.Hash]string),
		requests:    make(map[string]*txRequest),
		alternates:  make(map[common.Hash]map[string]struct{}),
		underpriced: mapset.NewSet[common.Hash](),
		hasTx:       hasTx,
		addTxs:      addTxs,
		fetchTxs:    fetchTxs,
		clock:       clock,
		rand:        rand,
	}
}

// Notify announces the fetcher of the potential availability of a new batch of
// transactions in the network.
func (f *TxFetcher) Notify(peer string, hashes []common.Hash) error {
	// Keep track of all the announced transactions
	txAnnounceInMeter.Mark(int64(len(hashes)))

	// Skip any transaction announcements that we already know of, or that we've
	// previously marked as cheap and discarded. This check is of course racy,
	// because multiple concurrent notifies will still manage to pass it, but it's
	// still valuable to check here because it runs concurrent  to the internal
	// loop, so anything caught here is time saved internally.
	var (
		unknowns               = make([]common.Hash, 0, len(hashes))
		duplicate, underpriced int64
	)
	for _, hash := range hashes {
		switch {
		case f.hasTx(hash):
			duplicate++

		case f.underpriced.Contains(hash):
			underpriced++

		default:
			unknowns = append(unknowns, hash)
		}
	}
	txAnnounceKnownMeter.Mark(duplicate)
	txAnnounceUnderpricedMeter.Mark(underpriced)

	// If anything's left to announce, push it into the internal loop
	if len(unknowns) == 0 {
		return nil
	}
	announce := &txAnnounce{
		origin: peer,
		hashes: unknowns,
	}
	select {
	case f.notify <- announce:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Enqueue imports a batch of received transaction into the transaction pool
// and the fetcher. This method may be called by both transaction broadcasts and
// direct request replies. The differentiation is important so the fetcher can
// re-schedule missing transactions as soon as possible.
func (f *TxFetcher) Enqueue(peer string, txs []*types.Transaction, direct bool) error {
	var (
		inMeter          = txReplyInMeter
		knownMeter       = txReplyKnownMeter
		underpricedMeter = txReplyUnderpricedMeter
		otherRejectMeter = txReplyOtherRejectMeter
	)
	if !direct {
		inMeter = txBroadcastInMeter
		knownMeter = txBroadcastKnownMeter
		underpricedMeter = txBroadcastUnderpricedMeter
		otherRejectMeter = txBroadcastOtherRejectMeter
	}
	// Keep track of all the propagated transactions
	inMeter.Mark(int64(len(txs)))

	// Push all the transactions into the pool, tracking underpriced ones to avoid
	// re-requesting them and dropping the peer in case of malicious transfers.
	var (
		added = make([]common.Hash, 0, len(txs))
	)
	// proceed in batches
	for i := 0; i < len(txs); i += 128 {
		end := i + 128
		if end > len(txs) {
			end = len(txs)
		}
		var (
			duplicate   int64
			underpriced int64
			otherreject int64
		)
		batch := txs[i:end]

		wrapped := make([]*txpool.Transaction, len(batch))
		for j, tx := range batch {
			wrapped[j] = &txpool.Transaction{Tx: tx}
		}
		for j, err := range f.addTxs(wrapped) {
			// Track the transaction hash if the price is too low for us.
			// Avoid re-request this transaction when we receive another
			// announcement.
			if errors.Is(err, txpool.ErrUnderpriced) || errors.Is(err, txpool.ErrReplaceUnderpriced) {
				for f.underpriced.Cardinality() >= maxTxUnderpricedSetSize {
					f.underpriced.Pop()
				}
				f.underpriced.Add(batch[j].Hash())
			}
			// Track a few interesting failure types
			switch {
			case err == nil: // Noop, but need to handle to not count these

			case errors.Is(err, txpool.ErrAlreadyKnown):
				duplicate++

			case errors.Is(err, txpool.ErrUnderpriced) || errors.Is(err, txpool.ErrReplaceUnderpriced):
				underpriced++

			default:
				otherreject++
			}
			added = append(added, batch[j].Hash())
		}
		knownMeter.Mark(duplicate)
		underpricedMeter.Mark(underpriced)
		otherRejectMeter.Mark(otherreject)

		// If 'other reject' is >25% of the deliveries in any batch, sleep a bit.
		if otherreject > 128/4 {
			time.Sleep(200 * time.Millisecond)
			log.Warn("Peer delivering stale transactions", "peer", peer, "rejected", otherreject)
		}
	}
	select {
	case f.cleanup <- &txDelivery{origin: peer, hashes: added, direct: direct}:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Drop should be called when a peer disconnects. It cleans up all the internal
// data structures of the given node.
func (f *TxFetcher) Drop(peer string) error {
	select {
	case f.drop <- &txDrop{peer: peer}:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Start boots up the announcement based synchroniser, accepting and processing
// hash notifications and block fetches until termination requested.
func (f *TxFetcher) Start() {
	go f.loop()
}

// Stop terminates the announcement based synchroniser, canceling all pending
// operations.
func (f *TxFetcher) Stop() {
	close(f.quit)
}

func (f *TxFetcher) loop() {
	var (
		waitTimer    = new(mclock.Timer)
		timeoutTimer = new(mclock.Timer)

		waitTrigger    = make(chan struct{}, 1)
		timeoutTrigger = make(chan struct{}, 1)
	)
	for {
		select {
		case ann := <-f.notify:
			// Drop part of the new announcements if there are too many accumulated.
			// Note, we could but do not filter already known transactions here as
			// the probability of something arriving between this call and the pre-
			// filter outside is essentially zero.
			used := len(f.waitslots[ann.origin]) + len(f.announces[ann.origin])
			if used >= maxTxAnnounces {
				// This can happen if a set of transactions are requested but not
				// all fulfilled, so the remainder are rescheduled without the cap
				// check. Should be fine as the limit is in the thousands and the
				// request size in the hundreds.
				txAnnounceDOSMeter.Mark(int64(len(ann.hashes)))
				break
			}
			want := used + len(ann.hashes)
			if want > maxTxAnnounces {
				txAnnounceDOSMeter.Mark(int64(want - maxTxAnnounces))
				ann.hashes = ann.hashes[:want-maxTxAnnounces]
			}
			// All is well, schedule the remainder of the transactions
			idleWait := len(f.waittime) == 0
			_, oldPeer := f.announces[ann.origin]

			for _, hash := range ann.hashes {
				// If the transaction is already downloading, add it to the list
				// of possible alternates (in case the current retrieval fails) and
				// also account it for the peer.
				if f.alternates[hash] != nil {
					f.alternates[hash][ann.origin] = struct{}{}

					// Stage 2 and 3 share the set of origins per tx
					if announces := f.announces[ann.origin]; announces != nil {
						announces[hash] = struct{}{}
					} else {
						f.announces[ann.origin] = map[common.Hash]struct{}{hash: {}}
					}
					continue
				}
				// If the transaction is not downloading, but is already queued
				// from a different peer, track it for the new peer too.
				if f.announced[hash] != nil {
					f.announced[hash][ann.origin] = struct{}{}

					// Stage 2 and 3 share the set of origins per tx
					if announces := f.announces[ann.origin]; announces != nil {
						announces[hash] = struct{}{}
					} else {
						f.announces[ann.origin] = map[common.Hash]struct{}{hash: {}}
					}
					continue
				}
				// If the transaction is already known to the fetcher, but not
				// yet downloading, add the peer as an alternate origin in the
				// waiting list.
				if f.waitlist[hash] != nil {
					f.waitlist[hash][ann.origin] = struct{}{}

					if waitslots := f.waitslots[ann.origin]; waitslots != nil {
						waitslots[hash] = struct{}{}
					} else {
						f.waitslots[ann.origin] = map[common.Hash]struct{}{hash: {}}
					}
					continue
				}
				// Transaction unknown to the fetcher, insert it into the waiting list
				f.waitlist[hash] = map[string]struct{}{ann.origin: {}}
				f.waittime[hash] = f.clock.Now()

				if waitslots := f.waitslots[ann.origin]; waitslots != nil {
					waitslots[hash] = struct{}{}
				} else {
					f.waitslots[ann.origin] = map[common.Hash]struct{}{hash: {}}
				}
			}
			// If a new item was added to the waitlist, schedule it into the fetcher
			if idleWait && len(f.waittime) > 0 {
				f.rescheduleWait(waitTimer, waitTrigger)
			}
			// If this peer is new and announced something already queued, maybe
			// request transactions from them
			if !oldPeer && len(f.announces[ann.origin]) > 0 {
				f.scheduleFetches(timeoutTimer, timeoutTrigger, map[string]struct{}{ann.origin: {}})
			}

		case <-waitTrigger:
			// At least one transaction's waiting time ran out, push all expired
			// ones into the retrieval queues
			actives := make(map[string]struct{})
			for hash, instance := range f.waittime {
				if time.Duration(f.clock.Now()-instance)+txGatherSlack > txArriveTimeout {
					// Transaction expired without propagation, schedule for retrieval
					if f.announced[hash] != nil {
						panic("announce tracker already contains waitlist item")
					}
					f.announced[hash] = f.waitlist[hash]
					for peer := range f.waitlist[hash] {
						if announces := f.announces[peer]; announces != nil {
							announces[hash] = struct{}{}
						} else {
							f.announces[peer] = map[common.Hash]struct{}{hash: {}}
						}
						delete(f.waitslots[peer], hash)
						if len(f.waitslots[peer]) == 0 {
							delete(f.waitslots, peer)
						}
						actives[peer] = struct{}{}
					}
					delete(f.waittime, hash)
					delete(f.waitlist, hash)
				}
			}
			// If transactions are still waiting for propagation, reschedule the wait timer
			if len(f.waittime) > 0 {
				f.rescheduleWait(waitTimer, waitTrigger)
			}
			// If any peers became active and are idle, request transactions from them
			if len(actives) > 0 {
				f.scheduleFetches(timeoutTimer, timeoutTrigger, actives)
			}

		case <-timeoutTrigger:
			// Clean up any expired retrievals and avoid re-requesting them from the
			// same peer (either overloaded or malicious, useless in both cases). We
			// could also penalize (Drop), but there's nothing to gain, and if could
			// possibly further increase the load on it.
			for peer, req := range f.requests {
				if time.Duration(f.clock.Now()-req.time)+txGatherSlack > txFetchTimeout {
					txRequestTimeoutMeter.Mark(int64(len(req.hashes)))

					// Reschedule all the not-yet-delivered fetches to alternate peers
					for _, hash := range req.hashes {
						// Skip rescheduling hashes already delivered by someone else
						if req.stolen != nil {
							if _, ok := req.stolen[hash]; ok {
								continue
							}
						}
						// Move the delivery back from fetching to queued
						if _, ok := f.announced[hash]; ok {
							panic("announced tracker already contains alternate item")
						}
						if f.alternates[hash] != nil { // nil if tx was broadcast during fetch
							f.announced[hash] = f.alternates[hash]
						}
						delete(f.announced[hash], peer)
						if len(f.announced[hash]) == 0 {
							delete(f.announced, hash)
						}
						delete(f.announces[peer], hash)
						delete(f.alternates, hash)
						delete(f.fetching, hash)
					}
					if len(f.announces[peer]) == 0 {
						delete(f.announces, peer)
					}
					// Keep track of the request as dangling, but never expire
					f.requests[peer].hashes = nil
				}
			}
			// Schedule a new transaction retrieval
			f.scheduleFetches(timeoutTimer, timeoutTrigger, nil)

			// No idea if we scheduled something or not, trigger the timer if needed
			// TODO(karalabe): this is kind of lame, can't we dump it into scheduleFetches somehow?
			f.rescheduleTimeout(timeoutTimer, timeoutTrigger)

		case delivery := <-f.cleanup:
			// Independent if the delivery was direct or broadcast, remove all
			// traces of the hash from internal trackers
			for _, hash := range delivery.hashes {
				if _, ok := f.waitlist[hash]; ok {
					for peer, txset := range f.waitslots {
						delete(txset, hash)
						if len(txset) == 0 {
							delete(f.waitslots, peer)
						}
					}
					delete(f.waitlist, hash)
					delete(f.waittime, hash)
				} else {
					for peer, txset := range f.announces {
						delete(txset, hash)
						if len(txset) == 0 {
							delete(f.announces, peer)
						}
					}
					delete(f.announced, hash)
					delete(f.alternates, hash)

					// If a transaction currently being fetched from a different
					// origin was delivered (delivery stolen), mark it so the
					// actual delivery won't double schedule it.
					if origin, ok := f.fetching[hash]; ok && (origin != delivery.origin || !delivery.direct) {
						stolen := f.requests[origin].stolen
						if stolen == nil {
							f.requests[origin].stolen = make(map[common.Hash]struct{})
							stolen = f.requests[origin].stolen
						}
						stolen[hash] = struct{}{}
					}
					delete(f.fetching, hash)
				}
			}
			// In case of a direct delivery, also reschedule anything missing
			// from the original query
			if delivery.direct {
				// Mark the requesting successful (independent of individual status)
				txRequestDoneMeter.Mark(int64(len(delivery.hashes)))

				// Make sure something was pending, nuke it
				req := f.requests[delivery.origin]
				if req == nil {
					log.Warn("Unexpected transaction delivery", "peer", delivery.origin)
					break
				}
				delete(f.requests, delivery.origin)

				// Anything not delivered should be re-scheduled (with or without
				// this peer, depending on the response cutoff)
				delivered := make(map[common.Hash]struct{})
				for _, hash := range delivery.hashes {
					delivered[hash] = struct{}{}
				}
				cutoff := len(req.hashes) // If nothing is delivered, assume everything is missing, don't retry!!!
				for i, hash := range req.hashes {
					if _, ok := delivered[hash]; ok {
						cutoff = i
					}
				}
				// Reschedule missing hashes from alternates, not-fulfilled from alt+self
				for i, hash := range req.hashes {
					// Skip rescheduling hashes already delivered by someone else
					if req.stolen != nil {
						if _, ok := req.stolen[hash]; ok {
							continue
						}
					}
					if _, ok := delivered[hash]; !ok {
						if i < cutoff {
							delete(f.alternates[hash], delivery.origin)
							delete(f.announces[delivery.origin], hash)
							if len(f.announces[delivery.origin]) == 0 {
								delete(f.announces, delivery.origin)
							}
						}
						if len(f.alternates[hash]) > 0 {
							if _, ok := f.announced[hash]; ok {
								panic(fmt.Sprintf("announced tracker already contains alternate item: %v", f.announced[hash]))
							}
							f.announced[hash] = f.alternates[hash]
						}
					}
					delete(f.alternates, hash)
					delete(f.fetching, hash)
				}
				// Something was delivered, try to reschedule requests
				f.scheduleFetches(timeoutTimer, timeoutTrigger, nil) // Partial delivery may enable others to deliver too
			}

		case drop := <-f.drop:
			// A peer was dropped, remove all traces of it
			if _, ok := f.waitslots[drop.peer]; ok {
				for hash := range f.waitslots[drop.peer] {
					delete(f.waitlist[hash], drop.peer)
					if len(f.waitlist[hash]) == 0 {
						delete(f.waitlist, hash)
						delete(f.waittime, hash)
					}
				}
				delete(f.waitslots, drop.peer)
				if len(f.waitlist) > 0 {
					f.rescheduleWait(waitTimer, waitTrigger)
				}
			}
			// Clean up any active requests
			var request *txRequest
			if request = f.requests[drop.peer]; request != nil {
				for _, hash := range request.hashes {
					// Skip rescheduling hashes already delivered by someone else
					if request.stolen != nil {
						if _, ok := request.stolen[hash]; ok {
							continue
						}
					}
					// Undelivered hash, reschedule if there's an alternative origin available
					delete(f.alternates[hash], drop.peer)
					if len(f.alternates[hash]) == 0 {
						delete(f.alternates, hash)
					} else {
						f.announced[hash] = f.alternates[hash]
						delete(f.alternates, hash)
					}
					delete(f.fetching, hash)
				}
				delete(f.requests, drop.peer)
			}
			// Clean up general announcement tracking
			if _, ok := f.announces[drop.peer]; ok {
				for hash := range f.announces[drop.peer] {
					delete(f.announced[hash], drop.peer)
					if len(f.announced[hash]) == 0 {
						delete(f.announced, hash)
					}
				}
				delete(f.announces, drop.peer)
			}
			// If a request was cancelled, check if anything needs to be rescheduled
			if request != nil {
				f.scheduleFetches(timeoutTimer, timeoutTrigger, nil)
				f.rescheduleTimeout(timeoutTimer, timeoutTrigger)
			}

		case <-f.quit:
			return
		}
		// No idea what happened, but bump some sanity metrics
		txFetcherWaitingPeers.Update(int64(len(f.waitslots)))
		txFetcherWaitingHashes.Update(int64(len(f.waitlist)))
		txFetcherQueueingPeers.Update(int64(len(f.announces) - len(f.requests)))
		txFetcherQueueingHashes.Update(int64(len(f.announced)))
		txFetcherFetchingPeers.Update(int64(len(f.requests)))
		txFetcherFetchingHashes.Update(int64(len(f.fetching)))

		// Loop did something, ping the step notifier if needed (tests)
		if f.step != nil {
			f.step <- struct{}{}
		}
	}
}

// rescheduleWait iterates over all the transactions currently in the waitlist
// and schedules the movement into the fetcher for the earliest.
//
// The method has a granularity of 'gatherSlack', since there's not much point in
// spinning over all the transactions just to maybe find one that should trigger
// a few ms earlier.
func (f *TxFetcher) rescheduleWait(timer *mclock.Timer, trigger chan struct{}) {
	if *timer != nil {
		(*timer).Stop()
	}
	now := f.clock.Now()

	earliest := now
	for _, instance := range f.waittime {
		if earliest > instance {
			earliest = instance
			if txArriveTimeout-time.Duration(now-earliest) < gatherSlack {
				break
			}
		}
	}
	*timer = f.clock.AfterFunc(txArriveTimeout-time.Duration(now-earliest), func() {
		trigger <- struct{}{}
	})
}

// rescheduleTimeout iterates over all the transactions currently in flight and
// schedules a cleanup run when the first would trigger.
//
// The method has a granularity of 'gatherSlack', since there's not much point in
// spinning over all the transactions just to maybe find one that should trigger
// a few ms earlier.
//
// This method is a bit "flaky" "by design". In theory the timeout timer only ever
// should be rescheduled if some request is pending. In practice, a timeout will
// cause the timer to be rescheduled every 5 secs (until the peer comes through or
// disconnects). This is a limitation of the fetcher code because we don't trac
// pending requests and timed out requests separately. Without double tracking, if
// we simply didn't reschedule the timer on all-timeout then the timer would never
// be set again since len(request) > 0 => something's running.
func (f *TxFetcher) rescheduleTimeout(timer *mclock.Timer, trigger chan struct{}) {
	if *timer != nil {
		(*timer).Stop()
	}
	now := f.clock.Now()

	earliest := now
	for _, req := range f.requests {
		// If this request already timed out, skip it altogether
		if req.hashes == nil {
			continue
		}
		if earliest > req.time {
			earliest = req.time
			if txFetchTimeout-time.Duration(now-earliest) < gatherSlack {
				break
			}
		}
	}
	*timer = f.clock.AfterFunc(txFetchTimeout-time.Duration(now-earliest), func() {
		trigger <- struct{}{}
	})
}

// scheduleFetches starts a batch of retrievals for all available idle peers.
func (f *TxFetcher) scheduleFetches(timer *mclock.Timer, timeout chan struct{}, whitelist map[string]struct{}) {
	// Gather the set of peers we want to retrieve from (default to all)
	actives := whitelist
	if actives == nil {
		actives = make(map[string]struct{})
		for peer := range f.announces {
			actives[peer] = struct{}{}
		}
	}
	if len(actives) == 0 {
		return
	}
	// For each active peer, try to schedule some transaction fetches
	idle := len(f.requests) == 0

	f.forEachPeer(actives, func(peer string) {
		if f.requests[peer] != nil {
			return // continue in the for-each
		}
		if len(f.announces[peer]) == 0 {
			return // continue in the for-each
		}
		hashes := make([]common.Hash, 0, maxTxRetrievals)
		f.forEachHash(f.announces[peer], func(hash common.Hash) bool {
			if _, ok := f.fetching[hash]; !ok {
				// Mark the hash as fetching and stash away possible alternates
				f.fetching[hash] = peer

				if _, ok := f.alternates[hash]; ok {
					panic(fmt.Sprintf("alternate tracker already contains fetching item: %v", f.alternates[hash]))
				}
				f.alternates[hash] = f.announced[hash]
				delete(f.announced, hash)

				// Accumulate the hash and stop if the limit was reached
				hashes = append(hashes, hash)
				if len(hashes) >= maxTxRetrievals {
					return false // break in the for-each
				}
			}
			return true // continue in the for-each
		})
		// If any hashes were allocated, request them from the peer
		if len(hashes) > 0 {
			f.requests[peer] = &txRequest{hashes: hashes, time: f.clock.Now()}
			txRequestOutMeter.Mark(int64(len(hashes)))

			go func(peer string, hashes []common.Hash) {
				// Try to fetch the transactions, but in case of a request
				// failure (e.g. peer disconnected), reschedule the hashes.
				if err := f.fetchTxs(peer, hashes); err != nil {
					txRequestFailMeter.Mark(int64(len(hashes)))
					f.Drop(peer)
				}
			}(peer, hashes)
		}
	})
	// If a new request was fired, schedule a timeout timer
	if idle && len(f.requests) > 0 {
		f.rescheduleTimeout(timer, timeout)
	}
}

// forEachPeer does a range loop over a map of peers in production, but during
// testing it does a deterministic sorted random to allow reproducing issues.
func (f *TxFetcher) forEachPeer(peers map[string]struct{}, do func(peer string)) {
	// If we're running production, use whatever Go's map gives us
	if f.rand == nil {
		for peer := range peers {
			do(peer)
		}
		return
	}
	// We're running the test suite, make iteration deterministic
	list := make([]string, 0, len(peers))
	for peer := range peers {
		list = append(list, peer)
	}
	sort.Strings(list)
	rotateStrings(list, f.rand.Intn(len(list)))
	for _, peer := range list {
		do(peer)
	}
}

// forEachHash does a range loop over a map of hashes in production, but during
// testing it does a deterministic sorted random to allow reproducing issues.
func (f *TxFetcher) forEachHash(hashes map[common.Hash]struct{}, do func(hash common.Hash) bool) {
	// If we're running production, use whatever Go's map gives us
	if f.rand == nil {
		for hash := range hashes {
			if !do(hash) {
				return
			}
		}
		return
	}
	// We're running the test suite, make iteration deterministic
	list := make([]common.Hash, 0, len(hashes))
	for hash := range hashes {
		list = append(list, hash)
	}
	sortHashes(list)
	rotateHashes(list, f.rand.Intn(len(list)))
	for _, hash := range list {
		if !do(hash) {
			return
		}
	}
}

// rotateStrings rotates the contents of a slice by n steps. This method is only
// used in tests to simulate random map iteration but keep it deterministic.
func rotateStrings(slice []string, n int) {
	orig := make([]string, len(slice))
	copy(orig, slice)

	for i := 0; i < len(orig); i++ {
		slice[i] = orig[(i+n)%len(orig)]
	}
}

// sortHashes sorts a slice of hashes. This method is only used in tests in order
// to simulate random map iteration but keep it deterministic.
func sortHashes(slice []common.Hash) {
	for i := 0; i < len(slice); i++ {
		for j := i + 1; j < len(slice); j++ {
			if bytes.Compare(slice[i][:], slice[j][:]) > 0 {
				slice[i], slice[j] = slice[j], slice[i]
			}
		}
	}
}

// rotateHashes rotates the contents of a slice by n steps. This method is only
// used in tests to simulate random map iteration but keep it deterministic.
func rotateHashes(slice []common.Hash, n int) {
	orig := make([]common.Hash, len(slice))
	copy(orig, slice)

	for i := 0; i < len(orig); i++ {
		slice[i] = orig[(i+n)%len(orig)]
	}
}
