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
	"math/rand"
	"slices"
	"sort"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto/kzg4844"
)

// todo remove partial / full

type random interface {
	Intn(n int) int
}

// BlobFetcher fetches blobs of new type-3 transactions with probability p,
// and for the remaining (1-p) transactions, it performs availability checks.
// For availability checks, it fetches cells from each blob in the transaction
// according to the custody cell indices provided by the consensus client
// connected to this execution client.

var blobFetchTimeout = 5 * time.Second

// todo tuning
const (
	availabilityThreshold         = 2
	maxPayloadRetrievals          = 128
	maxPayloadAnnounces           = 4096
	MAX_CELLS_PER_PARTIAL_REQUEST = 8
	blobAvailabilityTimeout       = 500 * time.Millisecond
)

type blobTxAnnounce struct {
	origin string              // Identifier of the peer that sent the announcement
	txs    []common.Hash       // Hashes of transactions announced
	cells  types.CustodyBitmap // Custody information of transactions being announced
}

type cellRequest struct {
	txs   []common.Hash        // Transactions that have been requested for their cells
	cells *types.CustodyBitmap // Requested cell indices
	time  mclock.AbsTime       // Timestamp when the request was made
}

type payloadDelivery struct {
	origin     string        // Peer from which the payloads were delivered
	txs        []common.Hash // Hashes of transactions that were delivered
	cells      [][]kzg4844.Cell
	cellBitmap *types.CustodyBitmap
}

type cellWithSeq struct {
	seq   uint64
	cells *types.CustodyBitmap
}

type fetchStatus struct {
	fetching *types.CustodyBitmap // To avoid fetching cells which had already been fetched / currently being fetched
	fetched  []uint64             // To sort cells
	cells    []kzg4844.Cell
}

// BlobFetcher is responsible for managing type 3 transactions based on peer announcements.
//
// BlobFetcher manages three buffers:
//   - Transactions not to be fetched are moved to "waitlist"
//     if a payload(blob) seems to be possessed by D(threshold) other peers, request custody cells for that.
//     Accept it when the cells are received. Otherwise, it is dropped.
//   - Transactions queued to be fetched are moved to "announces"
//     if a payload is received, it is added to the blob pool. Otherwise, the transaction is dropped.
//   - Transactions to be fetched are moved to "fetching"
//     if a payload/cell announcement is received during fetch, the peer is recorded as an alternate source.
type BlobFetcher struct {
	notify  chan *blobTxAnnounce
	cleanup chan *payloadDelivery
	drop    chan *txDrop
	quit    chan struct{}
	custody *types.CustodyBitmap

	txSeq uint64 // To make transactions fetched in arrival order

	full    map[common.Hash]struct{}
	partial map[common.Hash]struct{}

	// Buffer 1: Set of blob txs whose blob data is waiting for availability confirmation (not pull decision)
	waitlist  map[common.Hash]map[string]struct{} // Peer set that announced blob availability
	waittime  map[common.Hash]mclock.AbsTime      // Timestamp when added to waitlist
	waitslots map[string]map[common.Hash]struct{} // Waiting announcements grouped by peer (DoS protection)
	// waitSlots should also include announcements with partial cells

	// Buffer 2: Transactions queued for fetching (pull decision + not pull decision)
	// "announces" is shared with stage 3, for DoS protection
	announces map[string]map[common.Hash]*cellWithSeq // Set of announced transactions, grouped by origin peer

	// Buffer 2
	// Stage 3: Transactions whose payloads/cells are currently being fetched (pull decision + not pull decision)
	fetches  map[common.Hash]*fetchStatus // Hash -> Bitmap, in-flight transaction cells
	requests map[string][]*cellRequest    // In-flight transaction retrievals
	// todo simplify
	alternates map[common.Hash]map[string]*types.CustodyBitmap // In-flight transaction alternate origins (in case the peer is dropped)

	// Callbacks
	validateCells func([]common.Hash, [][]kzg4844.Cell, *types.CustodyBitmap) []error
	addPayload    func([]common.Hash, [][]kzg4844.Cell, *types.CustodyBitmap) []error
	fetchPayloads func(string, []common.Hash, *types.CustodyBitmap) error
	dropPeer      func(string)

	step     chan struct{}    // Notification channel when the fetcher loop iterates
	clock    mclock.Clock     // Monotonic clock or simulated clock for tests
	realTime func() time.Time // Real system time or simulated time for tests
	rand     random           // Randomizer
}

func NewBlobFetcher(
	validateCells func([]common.Hash, [][]kzg4844.Cell, *types.CustodyBitmap) []error,
	addPayload func([]common.Hash, [][]kzg4844.Cell, *types.CustodyBitmap) []error,
	fetchPayloads func(string, []common.Hash, *types.CustodyBitmap) error, dropPeer func(string),
	custody *types.CustodyBitmap, rand random) *BlobFetcher {
	return &BlobFetcher{
		notify:        make(chan *blobTxAnnounce),
		cleanup:       make(chan *payloadDelivery),
		drop:          make(chan *txDrop),
		quit:          make(chan struct{}),
		full:          make(map[common.Hash]struct{}),
		partial:       make(map[common.Hash]struct{}),
		waitlist:      make(map[common.Hash]map[string]struct{}),
		waittime:      make(map[common.Hash]mclock.AbsTime),
		waitslots:     make(map[string]map[common.Hash]struct{}),
		announces:     make(map[string]map[common.Hash]*cellWithSeq),
		fetches:       make(map[common.Hash]*fetchStatus),
		requests:      make(map[string][]*cellRequest),
		alternates:    make(map[common.Hash]map[string]*types.CustodyBitmap),
		validateCells: validateCells,
		addPayload:    addPayload,
		fetchPayloads: fetchPayloads,
		dropPeer:      dropPeer,
		custody:       custody,
		clock:         mclock.System{},
		realTime:      time.Now,
		rand:          rand,
	}
}

// Notify is called when a Type 3 transaction is observed on the network. (TransactionPacket / NewPooledTransactionHashesPacket)
func (f *BlobFetcher) Notify(peer string, txs []common.Hash, cells types.CustodyBitmap) error {
	blobAnnounce := &blobTxAnnounce{origin: peer, txs: txs, cells: cells}
	select {
	case f.notify <- blobAnnounce:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// Enqueue inserts a batch of received blob payloads into the blob pool.
// This is triggered by ethHandler upon receiving direct request responses.
func (f *BlobFetcher) Enqueue(peer string, hashes []common.Hash, cells [][]kzg4844.Cell, cellBitmap types.CustodyBitmap) error {
	blobReplyInMeter.Mark(int64(len(hashes)))

	select {
	case f.cleanup <- &payloadDelivery{origin: peer, txs: hashes, cells: cells, cellBitmap: &cellBitmap}:
	case <-f.quit:
		return errTerminated
	}
	return nil
}

func (f *BlobFetcher) Drop(peer string) error {
	select {
	case f.drop <- &txDrop{peer: peer}:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

func (f *BlobFetcher) UpdateCustody(cells types.CustodyBitmap) {
	f.custody = &cells
}

func (f *BlobFetcher) Start() {
	go f.loop()
}

func (f *BlobFetcher) Stop() {
	close(f.quit)
}

func (f *BlobFetcher) loop() {
	var (
		waitTimer      = new(mclock.Timer) // Timer for waitlist (availability)
		waitTrigger    = make(chan struct{}, 1)
		timeoutTimer   = new(mclock.Timer) // Timer for payload fetch request
		timeoutTrigger = make(chan struct{}, 1)
	)
	for {
		select {
		case ann := <-f.notify:
			// Drop part of the announcements if too many have accumulated from that peer
			// This prevents a peer from dominating the queue with txs without responding to the request
			// todo maxPayloadAnnounces -> according to the number of blobs
			used := len(f.waitslots[ann.origin]) + len(f.announces[ann.origin])
			if used >= maxPayloadAnnounces {
				// Already full
				break
			}

			want := used + len(ann.txs)
			if want >= maxPayloadAnnounces {
				// drop part of announcements
				ann.txs = ann.txs[:maxPayloadAnnounces-used]
			}

			var (
				idleWait   = len(f.waittime) == 0
				_, oldPeer = f.announces[ann.origin]
				nextSeq    = func() uint64 {
					seq := f.txSeq
					f.txSeq++
					return seq
				}
				reschedule = make(map[string]struct{})
			)
			for _, hash := range ann.txs {
				if oldPeer && f.announces[ann.origin][hash] != nil {
					// Ignore already announced information
					// We also have to prevent reannouncement by changing cells field.
					// Considering cell custody transition is notified in advance of its finalization by consensus client,
					// there is no reason to reannounce cells, and it has to be prevented.
					continue
				}
				// Decide full or partial request
				if _, ok := f.full[hash]; !ok {
					if _, ok := f.partial[hash]; !ok {
						// Not decided yet
						var randomValue int
						if f.rand == nil {
							randomValue = rand.Intn(100)
						} else {
							randomValue = f.rand.Intn(100)
						}
						if randomValue < 15 {
							f.full[hash] = struct{}{}
						} else {
							f.partial[hash] = struct{}{}
							// Register for availability check
							f.waitlist[hash] = make(map[string]struct{})
							f.waittime[hash] = f.clock.Now()
						}
					}
				}
				if _, ok := f.full[hash]; ok {
					// 1) Decided to send full request of the tx
					if ann.cells != *types.CustodyBitmapAll {
						continue
					}
					if f.announces[ann.origin] == nil {
						f.announces[ann.origin] = make(map[common.Hash]*cellWithSeq)
					}
					f.announces[ann.origin][hash] = &cellWithSeq{
						cells: types.CustodyBitmapData,
						seq:   nextSeq(),
					}
					reschedule[ann.origin] = struct{}{}
					continue
				}
				if _, ok := f.partial[hash]; ok {
					// 2) Decided to send partial request of the tx
					if f.waitlist[hash] != nil {
						if ann.cells != *types.CustodyBitmapAll {
							continue
						}
						// Transaction is at the stage of availability check
						// Add the peer to the peer list with full availability (waitlist)
						f.waitlist[hash][ann.origin] = struct{}{}
						if waitslots := f.waitslots[ann.origin]; waitslots != nil {
							waitslots[hash] = struct{}{}
						} else {
							f.waitslots[ann.origin] = map[common.Hash]struct{}{
								hash: {},
							}
						}
						if len(f.waitlist[hash]) >= availabilityThreshold {
							for peer := range f.waitlist[hash] {
								if f.announces[peer] == nil {
									f.announces[peer] = make(map[common.Hash]*cellWithSeq)
								}
								f.announces[peer][hash] = &cellWithSeq{
									cells: f.custody,
									seq:   nextSeq(),
								}
								delete(f.waitslots[peer], hash)
								if len(f.waitslots[peer]) == 0 {
									delete(f.waitslots, peer)
								}
								reschedule[peer] = struct{}{}
							}
							delete(f.waitlist, hash)
						}
						continue
					}
					if ann.cells.Intersection(f.custody).OneCount() == 0 {
						// No custody overlapping
						continue
					}

					if f.announces[ann.origin] == nil {
						f.announces[ann.origin] = make(map[common.Hash]*cellWithSeq)
					}
					f.announces[ann.origin][hash] = &cellWithSeq{
						cells: ann.cells.Intersection(f.custody),
						seq:   nextSeq(),
					}
					reschedule[ann.origin] = struct{}{}
				}
			}

			// If a new item was added to the waitlist, schedule its timeout
			if idleWait && len(f.waittime) > 0 {
				f.rescheduleWait(waitTimer, waitTrigger)
			}

			// If this is a new peer and that peer sent transaction with payload flag,
			// schedule transaction fetches from it
			//todo
			if !oldPeer && len(f.announces[ann.origin]) > 0 {
				f.scheduleFetches(timeoutTimer, timeoutTrigger, reschedule)
			}

		case <-waitTrigger:
			// At least one transaction's waiting time ran out, pop all expired ones
			// and update the blobpool according to availability
			// Availability failure case
			for hash, instance := range f.waittime {
				if time.Duration(f.clock.Now()-instance)+txGatherSlack > blobAvailabilityTimeout {
					// Check if enough peers have announced availability
					for peer := range f.waitlist[hash] {
						delete(f.waitslots[peer], hash)
						if len(f.waitslots[peer]) == 0 {
							delete(f.waitslots, peer)
						}
					}
					delete(f.waittime, hash)
					delete(f.waitlist, hash)
				}
			}

			// If transactions are still waiting for availability, reschedule the wait timer
			if len(f.waittime) > 0 {
				f.rescheduleWait(waitTimer, waitTrigger)
			}

		case <-timeoutTrigger:
			// Clean up any expired retrievals and avoid re-requesting them from the
			// same peer (either overloaded or malicious, useless in both cases).
			// Update blobpool according to availability result.
			for peer, requests := range f.requests {
				newRequests := make([]*cellRequest, 0)
				for _, req := range requests {
					if time.Duration(f.clock.Now()-req.time)+txGatherSlack > blobFetchTimeout {
						// Reschedule all timeout cells to alternate peers
						for _, hash := range req.txs {
							// Do not request the same tx from this peer
							delete(f.announces[peer], hash)
							delete(f.alternates[hash], peer)
							// Allow other candidates to be requested these cells
							f.fetches[hash].fetching = f.fetches[hash].fetching.Difference(req.cells)

							// Drop cells if there is no alternate source to fetch cells from
							if len(f.alternates[hash]) == 0 {
								delete(f.alternates, hash)
								delete(f.fetches, hash)
							}
						}
						if len(f.announces[peer]) == 0 {
							delete(f.announces, peer)
						}
					} else {
						newRequests = append(newRequests, req)
					}
				}
				f.requests[peer] = newRequests
				if len(f.requests[peer]) == 0 {
					delete(f.requests, peer)
				}
			}

			// Schedule a new transaction retrieval
			f.scheduleFetches(timeoutTimer, timeoutTrigger, nil)

			// Trigger timeout for new schedule
			f.rescheduleTimeout(timeoutTimer, timeoutTrigger)
		case delivery := <-f.cleanup:
			// Remove from announce
			addedHashes := make([]common.Hash, 0)
			addedCells := make([][]kzg4844.Cell, 0)

			var requestId int
			request := new(cellRequest)
			for _, hash := range delivery.txs {
				// Find the request
				for i, req := range f.requests[delivery.origin] {
					if slices.Contains(req.txs, hash) && *req.cells == *delivery.cellBitmap {
						request = req
						requestId = i
						break
					}
				}
				if request != nil {
					break
				}
			}
			if request == nil {
				// peer sent cells not requested. ignore
				break
			}

			for i, hash := range delivery.txs {
				if !slices.Contains(request.txs, hash) {
					// Unexpected hash, ignore
					continue
				}
				// Update fetch status
				f.fetches[hash].fetched = append(f.fetches[hash].fetched, delivery.cellBitmap.Indices()...)
				f.fetches[hash].cells = append(f.fetches[hash].cells, delivery.cells[i]...)

				// Update announces of this peer
				delete(f.announces[delivery.origin], hash)
				if len(f.announces[delivery.origin]) == 0 {
					delete(f.announces, delivery.origin)
				}
				delete(f.alternates[hash], delivery.origin)
				if len(f.alternates[hash]) == 0 {
					delete(f.alternates, hash)
				}

				// Check whether the all required cells are fetched
				completed := false
				if _, ok := f.full[hash]; ok && len(f.fetches[hash].fetched) >= kzg4844.DataPerBlob {
					completed = true
				} else if _, ok := f.partial[hash]; ok {
					fetched := make([]uint64, len(f.fetches[hash].fetched))
					copy(fetched, f.fetches[hash].fetched)
					slices.Sort(fetched)

					custodyIndices := f.custody.Indices()

					completed = slices.Equal(fetched, custodyIndices)
				}

				if completed {
					addedHashes = append(addedHashes, hash)
					fetchStatus := f.fetches[hash]
					sort.Slice(fetchStatus.cells, func(i, j int) bool {
						return fetchStatus.fetched[i] < fetchStatus.fetched[j]
					})
					addedCells = append(addedCells, fetchStatus.cells)

					// remove announces from other peers
					for peer, txset := range f.announces {
						delete(txset, hash)
						if len(txset) == 0 {
							delete(f.announces, peer)
						}
					}
					delete(f.alternates, hash)
					delete(f.fetches, hash)
				}
			}
			// Update mempool status for arrived hashes
			if len(addedHashes) > 0 {
				f.addPayload(addedHashes, addedCells, delivery.cellBitmap)
			}

			// Remove the request
			f.requests[delivery.origin][requestId] = f.requests[delivery.origin][len(f.requests[delivery.origin])-1]
			f.requests[delivery.origin] = f.requests[delivery.origin][:len(f.requests[delivery.origin])-1]
			if len(f.requests[delivery.origin]) == 0 {
				delete(f.requests, delivery.origin)
			}

			// Reschedule missing transactions in the request
			// Anything not delivered should be re-scheduled (with or without
			// this peer, depending on the response cutoff)
			delivered := make(map[common.Hash]struct{})
			for _, hash := range delivery.txs {
				delivered[hash] = struct{}{}
			}
			cutoff := len(request.txs)
			for i, hash := range request.txs {
				if _, ok := delivered[hash]; ok {
					cutoff = i
					continue
				}
			}
			// Reschedule missing hashes from alternates, not-fulfilled from alt+self
			for i, hash := range request.txs {
				if _, ok := delivered[hash]; !ok {
					// Not delivered
					if i < cutoff {
						// Remove origin from candidate sources for partial responses
						delete(f.alternates[hash], delivery.origin)
						delete(f.announces[delivery.origin], hash)
						if len(f.announces[delivery.origin]) == 0 {
							delete(f.announces, delivery.origin)
						}
					}
					// Mark cells deliverable by other peers
					if f.fetches[hash] != nil {
						f.fetches[hash].fetching = f.fetches[hash].fetching.Difference(delivery.cellBitmap)
					}
				}
			}
			// Something was delivered, try to reschedule requests
			f.scheduleFetches(timeoutTimer, timeoutTrigger, nil) // Partial delivery may enable others to deliver too
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
			// Clean up general announcement tracking
			if _, ok := f.announces[drop.peer]; ok {
				for hash := range f.announces[drop.peer] {
					delete(f.alternates[hash], drop.peer)
					if len(f.alternates[hash]) == 0 {
						delete(f.alternates, hash)
					}
				}
				delete(f.announces, drop.peer)
			}
			delete(f.announces, drop.peer)

			// Clean up any active requests
			if request, ok := f.requests[drop.peer]; ok && len(request) != 0 {
				for _, req := range request {
					for _, hash := range req.txs {
						// Undelivered hash, reschedule if there's an alternative origin available
						f.fetches[hash].fetching = f.fetches[hash].fetching.Difference(req.cells)
						delete(f.alternates[hash], drop.peer)
						if len(f.alternates[hash]) == 0 {
							delete(f.alternates, hash)
							delete(f.fetches, hash)
						}
					}
				}
				delete(f.requests, drop.peer)
				// If a request was cancelled, check if anything needs to be rescheduled
				f.scheduleFetches(timeoutTimer, timeoutTrigger, nil)
				f.rescheduleTimeout(timeoutTimer, timeoutTrigger)
			}

		case <-f.quit:
			return
		}
		// Loop did something, ping the step notifier if needed (tests)
		if f.step != nil {
			f.step <- struct{}{}
		}
	}
}

func (f *BlobFetcher) rescheduleWait(timer *mclock.Timer, trigger chan struct{}) {
	if *timer != nil {
		(*timer).Stop()
	}
	now := f.clock.Now()

	earliest := now
	for _, instance := range f.waittime {
		if earliest > instance {
			earliest = instance
			if txArriveTimeout-time.Duration(now-earliest) < txGatherSlack {
				break
			}
		}
	}
	*timer = f.clock.AfterFunc(txArriveTimeout-time.Duration(now-earliest), func() {
		trigger <- struct{}{}
	})
}

// Exactly same as the one in TxFetcher
func (f *BlobFetcher) rescheduleTimeout(timer *mclock.Timer, trigger chan struct{}) {
	if *timer != nil {
		(*timer).Stop()
	}
	now := f.clock.Now()

	earliest := now
	for _, requests := range f.requests {
		for _, req := range requests {
			// If this request already timed out, skip it altogether
			if req.txs == nil {
				continue
			}
			if earliest > req.time {
				earliest = req.time
				if blobFetchTimeout-time.Duration(now-earliest) < txGatherSlack {
					break
				}
			}
		}
	}
	*timer = f.clock.AfterFunc(blobFetchTimeout-time.Duration(now-earliest), func() {
		trigger <- struct{}{}
	})
}
func (f *BlobFetcher) scheduleFetches(timer *mclock.Timer, timeout chan struct{}, whitelist map[string]struct{}) {
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
	// For each active peer, try to schedule some payload fetches
	idle := len(f.requests) == 0

	f.forEachPeer(actives, func(peer string) {
		if len(f.announces[peer]) == 0 || len(f.requests[peer]) != 0 {
			return // continue
		}
		var (
			hashes    = make([]common.Hash, 0, maxTxRetrievals)
			custodies = make([]*types.CustodyBitmap, 0, maxTxRetrievals)
		)
		f.forEachAnnounce(f.announces[peer], func(hash common.Hash, cells *types.CustodyBitmap) bool {
			var difference *types.CustodyBitmap

			if f.fetches[hash] == nil {
				// tx is not being fetched
				difference = cells
			} else {
				difference = cells.Difference(f.fetches[hash].fetching)
			}

			// Mark fetching for differences
			if difference.OneCount() != 0 {
				if f.fetches[hash] == nil {
					f.fetches[hash] = &fetchStatus{
						fetching: difference,
						fetched:  make([]uint64, 0),
						cells:    make([]kzg4844.Cell, 0),
					}
				} else {
					f.fetches[hash].fetching = f.fetches[hash].fetching.Union(difference)
				}
				// Accumulate the hash and stop if the limit was reached
				hashes = append(hashes, hash)
				custodies = append(custodies, difference)
			}

			// Mark alternatives
			if f.alternates[hash] == nil {
				f.alternates[hash] = map[string]*types.CustodyBitmap{
					peer: cells,
				}
			} else {
				f.alternates[hash][peer] = cells
			}

			return len(hashes) < maxPayloadRetrievals
		})
		// If any hashes were allocated, request them from the peer
		if len(hashes) > 0 {
			// Group hashes by custody bitmap
			requestByCustody := make(map[string]*cellRequest)

			for i, hash := range hashes {
				custody := custodies[i]

				key := string(custody[:])

				if _, ok := requestByCustody[key]; !ok {
					requestByCustody[key] = &cellRequest{
						txs:   []common.Hash{},
						cells: custody,
						time:  f.clock.Now(),
					}
				}
				requestByCustody[key].txs = append(requestByCustody[key].txs, hash)
			}
			// construct request
			var request []*cellRequest
			for _, cr := range requestByCustody {
				request = append(request, cr)
			}
			f.requests[peer] = request
			go func(peer string, request []*cellRequest) {
				for _, req := range request {
					if err := f.fetchPayloads(peer, req.txs, req.cells); err != nil {
						f.Drop(peer)
						break
					}
				}
			}(peer, request)
		}
	})
	// If a new request was fired, schedule a timeout timer
	if idle && len(f.requests) > 0 {
		f.rescheduleTimeout(timer, timeout)
	}
}

// forEachAnnounce loops over the given announcements in arrival order, invoking
// the do function for each until it returns false. We enforce an arrival
// ordering to minimize the chances of transaction nonce-gaps, which result in
// transactions being rejected by the txpool.
func (f *BlobFetcher) forEachAnnounce(announces map[common.Hash]*cellWithSeq, do func(hash common.Hash, cells *types.CustodyBitmap) bool) {
	type announcement struct {
		hash  common.Hash
		cells *types.CustodyBitmap
		seq   uint64
	}
	// Process announcements by their arrival order
	list := make([]announcement, 0, len(announces))
	for hash, entry := range announces {
		list = append(list, announcement{hash: hash, cells: entry.cells, seq: entry.seq})
	}
	sort.Slice(list, func(i, j int) bool {
		return list[i].seq < list[j].seq
	})
	for i := range list {
		if !do(list[i].hash, list[i].cells) {
			return
		}
	}
}

// forEachPeer does a range loop over a map of peers in production, but during
// testing it does a deterministic sorted random to allow reproducing issues.
func (f *BlobFetcher) forEachPeer(peers map[string]struct{}, do func(peer string)) {
	// If we're running production(step == nil), use whatever Go's map gives us
	if f.step == nil {
		for peer := range peers {
			do(peer)
		}
		return
	}
	// We're running the test suite, make iteration deterministic (sorted by peer id)
	list := make([]string, 0, len(peers))
	for peer := range peers {
		list = append(list, peer)
	}
	sort.Strings(list)
	for _, peer := range list {
		do(peer)
	}
}
