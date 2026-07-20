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

package downloader

import (
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/eth/protocols/eth"
	"github.com/ethereum/go-ethereum/log"
)

// balQueue implements typedQueue and is a type adapter between the generic
// concurrent fetcher and the downloader. It fetches EIP-7928 block access
// lists on a best effort basis: only eth/71+ peers can serve them and blocks
// are imported without one if it doesn't arrive in time.
type balQueue Downloader

// waker returns a notification channel that gets pinged in case more access
// list fetches have been queued up, so the fetcher might assign it to idle peers.
func (q *balQueue) waker() chan bool {
	return q.queue.balWakeCh
}

// pending returns the number of access lists that are currently queued for
// fetching by the concurrent downloader.
func (q *balQueue) pending() int {
	return q.queue.PendingBALs()
}

// capacity is responsible for calculating how many access lists a particular
// peer is estimated to be able to retrieve within the allotted round trip time.
func (q *balQueue) capacity(peer *peerConnection, rtt time.Duration) int {
	if peer.version < eth.ETH71 {
		return 0
	}
	return peer.BALCapacity(rtt)
}

// updateCapacity is responsible for updating how many access lists a particular
// peer is estimated to be able to retrieve in a unit time.
func (q *balQueue) updateCapacity(peer *peerConnection, items int, span time.Duration) {
	peer.UpdateBALRate(items, span)
}

// reserve is responsible for allocating a requested number of pending access
// lists from the download queue to the specified peer. Peers below eth/71
// cannot serve access lists and are never assigned any.
func (q *balQueue) reserve(peer *peerConnection, items int) (*fetchRequest, bool, bool) {
	if peer.version < eth.ETH71 {
		return nil, false, false
	}
	return q.queue.ReserveBALs(peer, items)
}

// unreserve is responsible for removing the current access list retrieval
// allocation assigned to a specific peer and placing it back into the pool to
// allow reassigning to some other peer.
func (q *balQueue) unreserve(peer string) int {
	fails := q.queue.ExpireBALs(peer)
	if fails > 2 {
		log.Trace("Access list delivery timed out", "peer", peer)
	} else {
		log.Debug("Access list delivery stalling", "peer", peer)
	}
	return fails
}

// request is responsible for converting a generic fetch request into an access
// list one and sending it to the remote peer for fulfillment.
func (q *balQueue) request(peer *peerConnection, req *fetchRequest, resCh chan *eth.Response) (*eth.Request, error) {
	peer.log.Trace("Requesting new batch of access lists", "count", len(req.Headers), "from", req.Headers[0].Number)
	if q.balFetchHook != nil {
		q.balFetchHook(req.Headers)
	}
	hashes := make([]common.Hash, 0, len(req.Headers))
	for _, header := range req.Headers {
		hashes = append(hashes, header.Hash())
	}
	return peer.peer.RequestBALs(hashes, resCh)
}

// deliver is responsible for taking a generic response packet from the
// concurrent fetcher, unpacking the access list data and delivering it to the
// downloader's queue.
func (q *balQueue) deliver(peer *peerConnection, packet *eth.Response) (int, error) {
	bals := *packet.Res.(*eth.BlockAccessListResponse)
	hashes := packet.Meta.([]common.Hash) // {keccak256 hash per entry, zero hash if unavailable}

	accepted, err := q.queue.DeliverBALs(peer.id, bals, hashes)
	switch {
	case err == nil && len(bals) == 0:
		peer.log.Trace("Requested access lists delivered")
	case err == nil:
		peer.log.Trace("Delivered new batch of access lists", "count", len(bals), "accepted", accepted)
	default:
		peer.log.Debug("Failed to deliver retrieved access lists", "err", err)
	}
	return accepted, err
}
