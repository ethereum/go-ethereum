// Copyright 2016 The go-ethereum Authors
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

	"github.com/ethereum/go-ethereum/common/mclock"
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

var (
	retryQueue         = time.Millisecond * 100
	softRequestTimeout = time.Millisecond * 500
	hardRequestTimeout = time.Second * 10
)

// reqPeerCallback is called after a request sent to a certain peer has either been
// answered or timed out hard
type reqPeerCallback func(p distPeer, respTime time.Duration, srto, hrto bool)

// distReq is the request abstraction used by the distributor. It is based on
// three callback functions:
// - getCost returns the upper estimate of the cost of sending the request to a given peer
// - canSend tells if the server peer is suitable to serve the request
// - request prepares sending the request to the given peer and returns a function that
// does the actual sending. Request order should be preserved but the callback itself should not
// block until it is sent because other peers might still be able to receive requests while
// one of them is blocking. Instead, the returned function is put in the peer's send queue.
//
// Requests can either be queued manually or by the retrieval management mechanism which
// takes care of timeouts, resends and not sending to the same peer again.
type distReq struct {
	getCost func(distPeer) uint64
	canSend func(distPeer) bool
	request func(distPeer) func()

	reqOrder uint64
	sentChn  chan distPeer
	element  *list.Element

	// retrieval management fields (optional)
	dist                    *requestDistributor
	peerCallback            reqPeerCallback
	lock                    sync.RWMutex
	stopChn                 chan struct{}
	stopped                 bool
	err                     error
	sentTo                  map[distPeer]chan struct{} // channel signaling a reply from the given peer
	sentCnt, softTimeoutCnt int
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
					req.lock.Lock()
					if req.sentTo != nil {
						// if the retrieval manager is used, create deliver channel
						req.sentTo[peer] = make(chan struct{})
						req.sentCnt++
					}
					req.lock.Unlock()
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
		req.lock.RLock()
		for peer, _ := range peers {
			// if retrieve manager is not used and sentTo is nil, ok is always false
			if _, ok := req.sentTo[peer]; !ok && peer.canQueue() && req.canSend(peer) {
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
		req.lock.RUnlock()
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

// retrieve starts a process that keeps trying to retrieve a valid answer for a
// request from any suitable peers until stopped or succeeded.
func (d *requestDistributor) retrieve(r *distReq, cb reqPeerCallback) chan struct{} {
	r.dist = d
	r.sentTo = make(map[distPeer]chan struct{})
	r.stopChn = make(chan struct{})
	r.peerCallback = cb

	go r.retrieve()
	return r.stopChn
}

// retrieve starts a retrieval process for a single peer. If no request has been
// sent yet and no suitable peer found, it sets an ErrNoPeers error code and returns.
// Otherwise it keeps trying until it sends the request to a peer, then waits for an
// answer or a timeout.
func (r *distReq) retrieve() {
	for {
		sent := r.dist.queue(r)
		var p distPeer
		select {
		case p = <-sent:
		case <-r.stopChn:
			if r.dist.cancel(r) {
				p = nil
			} else {
				p = <-sent
			}
		}
		if p == nil {
			if r.waiting() {
				time.Sleep(retryQueue)
				if r.waiting() {
					continue
				}
			} else {
				r.stop(ErrNoPeers) // no effect if already stopped for another reason
			}
		} else {
			respTime, srto, hrto := r.runTimer(p)
			r.peerCallback(p, respTime, srto, hrto)
		}
		break
	}
}

// getError returns any retrieval error (either internally generated or set by the
// stop function) after stopChn has been closed
func (r *distReq) getError() error {
	return r.err
}

// expectResponseFrom tells if we are expecting a response from a given peer.
// A response to a sent request is expected even after another peer have delivered
// a valid response. If a hard timeout occurs, no response is expected any more.
func (r *distReq) expectResponseFrom(peer distPeer) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.sentTo[peer] != nil
}

// delivered notifies the retrieval mechanism that a reply (either valid
// or invalid) has been received from a certain peer
func (r *distReq) delivered(peer distPeer, valid bool) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	delivered, ok := r.sentTo[peer]
	if ok {
		close(delivered)
		delete(r.sentTo, peer)
		if valid && !r.stopped {
			r.stopped = true
			close(r.stopChn)
		}
		if !r.stopped && r.sentCnt == 0 {
			go r.retrieve()
		}

	}
	return ok
}

// stop stops the retrieval process and sets an error code that will be returned
// by getError
func (r *distReq) stop(err error) {
	r.lock.Lock()
	if !r.stopped {
		r.stopped = true
		r.err = err
		close(r.stopChn)
	}
	r.lock.Unlock()
}

// waiting returns true if the retrieval mechanism is waiting for an answer from
// any peer
func (r *distReq) waiting() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return !r.stopped && r.sentCnt+r.softTimeoutCnt != 0
}

// runTimer starts a request timeout for a single peer. If a certain (short) period
// of time passes with no reply received, it switches from "sent" to "soft timeout"
// state and the retrieval mechanism will start trying to send another request to a
// new peer. If another (longer) timeout period passes, it switches to "hard timeout"
// state, after which a reply is no longer expected and the peer should be dropped.
func (r *distReq) runTimer(peer distPeer) (respTime time.Duration, srto, hrto bool) {
	start := mclock.Now()

	r.lock.RLock()
	delivered := r.sentTo[peer]
	r.lock.RUnlock()
	if delivered == nil {
		panic(nil)
	}

	select {
	case <-delivered:
		r.lock.Lock()
		r.sentCnt--
		r.lock.Unlock()
		return time.Duration(mclock.Now() - start), false, false
	case <-time.After(softRequestTimeout):
		r.lock.Lock()
		r.sentCnt--
		resend := !r.stopped && r.sentCnt == 0
		r.softTimeoutCnt++
		r.lock.Unlock()
		if resend {
			go r.retrieve()
		}
	}

	hrto = false
	select {
	case <-delivered:
	case <-time.After(hardRequestTimeout):
		hrto = true
	}

	r.lock.Lock()
	r.softTimeoutCnt--
	// do not delete sentTo entry, nil means that we are not expecting an answer
	// but we also do not want to send to that peer again
	r.sentTo[peer] = nil
	r.lock.Unlock()
	return time.Duration(mclock.Now() - start), true, hrto
}
