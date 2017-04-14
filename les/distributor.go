// Copyright 2017 The go-ethereum Authors
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

// Package light implements on-demand retrieval capable state and chain objects
// for the Ethereum Light Client.
package les

import (
	"container/list"
	"errors"
	"sync"
	"time"
)

// ErrNoPeers is returned if no peers capable of serving a queued request are available
var ErrNoPeers = errors.New("no suitable peers available")

// requestDistributor implements a mechanism that distributes requests to
// suitable peers, obeying flow control rules and prioritizing them in creation
// order (even when a resend is necessary).
type requestDistributor struct {
	reqQueue         *list.List
	lastReqOrder     uint64
	stopChn, loopChn chan struct{}
	loopNextSent     bool
	lock             sync.Mutex

	getAllPeers func() map[distPeer]struct{}
}

// distPeer is an LES server peer interface for the request distributor.
// waitBefore returns either the necessary waiting time before sending a request
// with the given upper estimated cost or the estimated remaining relative buffer
// value after sending such a request (in which case the request can be sent
// immediately). At least one of these values is always zero.
type distPeer interface {
	waitBefore(uint64) (time.Duration, float64)
	canQueue() bool
	queueSend(f func())
}

// distReq is the request abstraction used by the distributor. It is based on
// three callback functions:
// - getCost returns the upper estimate of the cost of sending the request to a given peer
// - canSend tells if the server peer is suitable to serve the request
// - request prepares sending the request to the given peer and returns a function that
// does the actual sending. Request order should be preserved but the callback itself should not
// block until it is sent because other peers might still be able to receive requests while
// one of them is blocking. Instead, the returned function is put in the peer's send queue.
type distReq struct {
	getCost func(distPeer) uint64
	canSend func(distPeer) bool
	request func(distPeer) func()

	reqOrder uint64
	sentChn  chan distPeer
	element  *list.Element
}

// newRequestDistributor creates a new request distributor
func newRequestDistributor(getAllPeers func() map[distPeer]struct{}, stopChn chan struct{}) *requestDistributor {
	r := &requestDistributor{
		reqQueue:    list.New(),
		loopChn:     make(chan struct{}, 2),
		stopChn:     stopChn,
		getAllPeers: getAllPeers,
	}
	go r.loop()
	return r
}

// distMaxWait is the maximum waiting time after which further necessary waiting
// times are recalculated based on new feedback from the servers
const distMaxWait = time.Millisecond * 10

// main event loop
func (d *requestDistributor) loop() {
	for {
		select {
		case <-d.stopChn:
			d.lock.Lock()
			elem := d.reqQueue.Front()
			for elem != nil {
				close(elem.Value.(*distReq).sentChn)
				elem = elem.Next()
			}
			d.lock.Unlock()
			return
		case <-d.loopChn:
			d.lock.Lock()
			d.loopNextSent = false
		loop:
			for {
				peer, req, wait := d.nextRequest()
				if req != nil && wait == 0 {
					chn := req.sentChn // save sentChn because remove sets it to nil
					d.remove(req)
					send := req.request(peer)
					if send != nil {
						peer.queueSend(send)
					}
					chn <- peer
					close(chn)
				} else {
					if wait == 0 {
						// no request to send and nothing to wait for; the next
						// queued request will wake up the loop
						break loop
					}
					d.loopNextSent = true // a "next" signal has been sent, do not send another one until this one has been received
					if wait > distMaxWait {
						// waiting times may be reduced by incoming request replies, if it is too long, recalculate it periodically
						wait = distMaxWait
					}
					go func() {
						time.Sleep(wait)
						d.loopChn <- struct{}{}
					}()
					break loop
				}
			}
			d.lock.Unlock()
		}
	}
}

// selectPeerItem represents a peer to be selected for a request by weightedRandomSelect
type selectPeerItem struct {
	peer   distPeer
	req    *distReq
	weight int64
}

// Weight implements wrsItem interface
func (sp selectPeerItem) Weight() int64 {
	return sp.weight
}

// nextRequest returns the next possible request from any peer, along with the
// associated peer and necessary waiting time
func (d *requestDistributor) nextRequest() (distPeer, *distReq, time.Duration) {
	peers := d.getAllPeers()

	elem := d.reqQueue.Front()
	var (
		bestPeer distPeer
		bestReq  *distReq
		bestWait time.Duration
		sel      *weightedRandomSelect
	)

	for (len(peers) > 0 || elem == d.reqQueue.Front()) && elem != nil {
		req := elem.Value.(*distReq)
		canSend := false
		for peer, _ := range peers {
			if peer.canQueue() && req.canSend(peer) {
				canSend = true
				cost := req.getCost(peer)
				wait, bufRemain := peer.waitBefore(cost)
				if wait == 0 {
					if sel == nil {
						sel = newWeightedRandomSelect()
					}
					sel.update(selectPeerItem{peer: peer, req: req, weight: int64(bufRemain*1000000) + 1})
				} else {
					if bestReq == nil || wait < bestWait {
						bestPeer = peer
						bestReq = req
						bestWait = wait
					}
				}
				delete(peers, peer)
			}
		}
		next := elem.Next()
		if !canSend && elem == d.reqQueue.Front() {
			close(req.sentChn)
			d.remove(req)
		}
		elem = next
	}

	if sel != nil {
		c := sel.choose().(selectPeerItem)
		return c.peer, c.req, 0
	}
	return bestPeer, bestReq, bestWait
}

// queue adds a request to the distribution queue, returns a channel where the
// receiving peer is sent once the request has been sent (request callback returned).
// If the request is cancelled or timed out without suitable peers, the channel is
// closed without sending any peer references to it.
func (d *requestDistributor) queue(r *distReq) chan distPeer {
	d.lock.Lock()
	defer d.lock.Unlock()

	if r.reqOrder == 0 {
		d.lastReqOrder++
		r.reqOrder = d.lastReqOrder
	}

	back := d.reqQueue.Back()
	if back == nil || r.reqOrder > back.Value.(*distReq).reqOrder {
		r.element = d.reqQueue.PushBack(r)
	} else {
		before := d.reqQueue.Front()
		for before.Value.(*distReq).reqOrder < r.reqOrder {
			before = before.Next()
		}
		r.element = d.reqQueue.InsertBefore(r, before)
	}

	if !d.loopNextSent {
		d.loopNextSent = true
		d.loopChn <- struct{}{}
	}

	r.sentChn = make(chan distPeer, 1)
	return r.sentChn
}

// cancel removes a request from the queue if it has not been sent yet (returns
// false if it has been sent already). It is guaranteed that the callback functions
// will not be called after cancel returns.
func (d *requestDistributor) cancel(r *distReq) bool {
	d.lock.Lock()
	defer d.lock.Unlock()

	if r.sentChn == nil {
		return false
	}

	close(r.sentChn)
	d.remove(r)
	return true
}

// remove removes a request from the queue
func (d *requestDistributor) remove(r *distReq) {
	r.sentChn = nil
	if r.element != nil {
		d.reqQueue.Remove(r.element)
		r.element = nil
	}
}
