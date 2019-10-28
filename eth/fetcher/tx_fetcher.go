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
	"math/rand"
	"time"

	mapset "github.com/deckarep/golang-set"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/log"
)

var (
	// txAnnounceLimit is the maximum number of unique transaction a peer
	// can announce in a short time.
	txAnnounceLimit = 4096

	// txFetchTimeout is the maximum allotted time to return an explicitly
	// requested transaction.
	txFetchTimeout = 5 * time.Second

	// MaxTransactionFetch is the maximum transaction number can be fetched
	// in one request. The rationale to pick this value is:
	// In eth protocol, the softResponseLimit is 2MB. Nowdays according to
	// Etherscan the average transaction size is around 200B, so in theory
	// we can include lots of transaction in a single protocol packet. However
	// the maximum size of a single transaction is raised to 128KB, so pick
	// a middle value here to ensure we can maximize the efficiency of the
	// retrieval and response size overflow won't happen in most cases.
	MaxTransactionFetch = 256

	// underpriceSetSize is the size of underprice set which used for maintaining
	// the set of underprice transactions.
	underpriceSetSize = 4096
)

// txAnnounce is the notification of the availability of a single
// new transaction in the network.
type txAnnounce struct {
	origin   string              // Identifier of the peer originating the notification
	time     time.Time           // Timestamp of the announcement
	fetchTxs func([]common.Hash) // Callback for retrieving transaction from specified peer
}

// txsAnnounce is the notification of the availability of a batch
// of new transactions in the network.
type txsAnnounce struct {
	hashes   []common.Hash       // Batch of transaction hashes being announced
	origin   string              // Identifier of the peer originating the notification
	time     time.Time           // Timestamp of the announcement
	fetchTxs func([]common.Hash) // Callback for retrieving transaction from specified peer
}

// TxFetcher is responsible for retrieving new transaction based
// on the announcement.
type TxFetcher struct {
	notify  chan *txsAnnounce
	cleanup chan []common.Hash
	quit    chan struct{}

	// Announce states
	announces   map[string]int                // Per peer transaction announce counts to prevent memory exhaustion
	announced   map[common.Hash][]*txAnnounce // Announced transactions, scheduled for fetching
	fetching    map[common.Hash]*txAnnounce   // Announced transactions, currently fetching
	underpriced mapset.Set                    // Transaction set whose price is too low for accepting

	// Callbacks
	hasTx    func(common.Hash) bool             // Retrieves a tx from the local txpool
	addTxs   func([]*types.Transaction) []error // Insert a batch of transactions into local txpool
	dropPeer func(string)                       // Drop the specified peer

	// Hooks
	announceHook     func([]common.Hash)        // Hook which is called when a batch transactions are announced
	importTxsHook    func([]*types.Transaction) // Hook which is called when a batch of transactions are imported.
	dropHook         func(string)               // Hook which is called when a peer is dropped
	cleanupHook      func([]common.Hash)        // Hook which is called when internal status is cleaned
	rejectUnderprice func(common.Hash)          // Hook which is called when underprice transaction is rejected
}

// NewTxFetcher creates a transaction fetcher to retrieve transaction
// based on hash announcements.
func NewTxFetcher(hasTx func(common.Hash) bool, addTxs func([]*types.Transaction) []error, dropPeer func(string)) *TxFetcher {
	return &TxFetcher{
		notify:      make(chan *txsAnnounce),
		cleanup:     make(chan []common.Hash),
		quit:        make(chan struct{}),
		announces:   make(map[string]int),
		announced:   make(map[common.Hash][]*txAnnounce),
		fetching:    make(map[common.Hash]*txAnnounce),
		underpriced: mapset.NewSet(),
		hasTx:       hasTx,
		addTxs:      addTxs,
		dropPeer:    dropPeer,
	}
}

// Notify announces the fetcher of the potential availability of a
// new transaction in the network.
func (f *TxFetcher) Notify(peer string, hashes []common.Hash, time time.Time, fetchTxs func([]common.Hash)) error {
	announce := &txsAnnounce{
		hashes:   hashes,
		time:     time,
		origin:   peer,
		fetchTxs: fetchTxs,
	}
	select {
	case f.notify <- announce:
		return nil
	case <-f.quit:
		return errTerminated
	}
}

// EnqueueTxs imports a batch of received transaction into fetcher.
func (f *TxFetcher) EnqueueTxs(peer string, txs []*types.Transaction) error {
	var (
		drop   bool
		hashes []common.Hash
	)
	errs := f.addTxs(txs)
	for i, err := range errs {
		if err != nil {
			// Drop peer if the received transaction isn't signed properly.
			drop = (drop || err == core.ErrInvalidSender)
			txFetchInvalidMeter.Mark(1)

			// Track the transaction hash if the price is too low for us.
			// Avoid re-request this transaction when we receive another
			// announcement.
			if err == core.ErrUnderpriced {
				for f.underpriced.Cardinality() >= underpriceSetSize {
					f.underpriced.Pop()
				}
				f.underpriced.Add(txs[i].Hash())
			}
		}
		hashes = append(hashes, txs[i].Hash())
	}
	if f.importTxsHook != nil {
		f.importTxsHook(txs)
	}
	// Drop the peer if some transaction failed signature verification.
	// We can regard this peer is trying to DOS us by feeding lots of
	// random hashes.
	if drop {
		f.dropPeer(peer)
		if f.dropHook != nil {
			f.dropHook(peer)
		}
	}
	select {
	case f.cleanup <- hashes:
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
	fetchTimer := time.NewTimer(0)

	for {
		// Clean up any expired transaction fetches.
		// There are many cases can lead to it:
		// * We send the request to busy peer which can reply immediately
		// * We send the request to malicious peer which doesn't reply deliberately
		// * We send the request to normal peer for a batch of transaction, but some
		//   transactions have been included into blocks. According to EIP these txs
		//   won't be included.
		// But it's fine to delete the fetching record and reschedule fetching iff we
		// receive the annoucement again.
		for hash, announce := range f.fetching {
			if time.Since(announce.time) > txFetchTimeout {
				delete(f.fetching, hash)
				txFetchTimeoutMeter.Mark(1)
			}
		}
		select {
		case anno := <-f.notify:
			txAnnounceInMeter.Mark(int64(len(anno.hashes)))

			// Drop the new announce if there are too many accumulated.
			count := f.announces[anno.origin] + len(anno.hashes)
			if count > txAnnounceLimit {
				txAnnounceDOSMeter.Mark(int64(count - txAnnounceLimit))
				break
			}
			f.announces[anno.origin] = count

			// All is well, schedule the announce if transaction is not yet downloading
			empty := len(f.announced) == 0
			for _, hash := range anno.hashes {
				if _, ok := f.fetching[hash]; ok {
					continue
				}
				if f.underpriced.Contains(hash) {
					txAnnounceUnderpriceMeter.Mark(1)
					if f.rejectUnderprice != nil {
						f.rejectUnderprice(hash)
					}
					continue
				}
				f.announced[hash] = append(f.announced[hash], &txAnnounce{
					origin:   anno.origin,
					time:     anno.time,
					fetchTxs: anno.fetchTxs,
				})
			}
			if empty && len(f.announced) > 0 {
				f.reschedule(fetchTimer)
			}
			if f.announceHook != nil {
				f.announceHook(anno.hashes)
			}
		case <-fetchTimer.C:
			// At least one tx's timer ran out, check for needing retrieval
			request := make(map[string][]common.Hash)

			for hash, announces := range f.announced {
				if time.Since(announces[0].time) > arriveTimeout-gatherSlack {
					// Pick a random peer to retrieve from, reset all others
					announce := announces[rand.Intn(len(announces))]
					f.forgetHash(hash)

					// Skip fetching if we already receive the transaction.
					if f.hasTx(hash) {
						txAnnounceSkipMeter.Mark(1)
						continue
					}
					// If the transaction still didn't arrive, queue for fetching
					request[announce.origin] = append(request[announce.origin], hash)
					f.fetching[hash] = announce
				}
			}
			// Send out all block header requests
			for peer, hashes := range request {
				log.Trace("Fetching scheduled transactions", "peer", peer, "txs", hashes)
				fetchTxs := f.fetching[hashes[0]].fetchTxs
				fetchTxs(hashes)
				txFetchOutMeter.Mark(int64(len(hashes)))
			}
			// Schedule the next fetch if blocks are still pending
			f.reschedule(fetchTimer)
		case hashes := <-f.cleanup:
			for _, hash := range hashes {
				f.forgetHash(hash)
				anno, exist := f.fetching[hash]
				if !exist {
					txBroadcastInMeter.Mark(1) // Directly transaction propagation
					continue
				}
				txFetchDurationTimer.UpdateSince(anno.time)
				txFetchSuccessMeter.Mark(1)
				delete(f.fetching, hash)
			}
			if f.cleanupHook != nil {
				f.cleanupHook(hashes)
			}
		case <-f.quit:
			return
		}
	}
}

// rescheduleFetch resets the specified fetch timer to the next blockAnnounce timeout.
func (f *TxFetcher) reschedule(fetch *time.Timer) {
	// Short circuit if no transactions are announced
	if len(f.announced) == 0 {
		return
	}
	// Otherwise find the earliest expiring announcement
	earliest := time.Now()
	for _, announces := range f.announced {
		if earliest.After(announces[0].time) {
			earliest = announces[0].time
		}
	}
	fetch.Reset(arriveTimeout - time.Since(earliest))
}

func (f *TxFetcher) forgetHash(hash common.Hash) {
	// Remove all pending announces and decrement DOS counters
	for _, announce := range f.announced[hash] {
		f.announces[announce.origin]--
		if f.announces[announce.origin] <= 0 {
			delete(f.announces, announce.origin)
		}
	}
	delete(f.announced, hash)
}
