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
	"context"
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

var (
	retryQueue         = time.Millisecond * 100
	softRequestTimeout = time.Millisecond * 500
	hardRequestTimeout = time.Second * 10
)

// retrieveManager is a layer on top of requestDistributor which takes care of
// matching replies by request ID and handles timeouts and resends if necessary.
type retrieveManager struct {
	dist       *requestDistributor
	serverPool peerSelector
	removePeer peerDropFn

	lock     sync.RWMutex
	sentReqs map[uint64]*sentReq
}

// validatorFunc is a function that processes a reply message
type validatorFunc func(distPeer, *Msg) error

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

// peerSelector receives feedback info about response times and timeouts
type peerSelector interface {
	adjustResponseTime(*poolEntry, time.Duration, bool)
}

type sentReq struct {
	rm                      *retrieveManager
	req                     *distReq
	validate                validatorFunc
	stopChn                 chan struct{}
	stopped                 bool
	err                     error
	lock                    sync.RWMutex               // protect access to sentTo map
	sentTo                  map[distPeer]chan struct{} // channel signaling a reply from the given peer
	reqWg                   sync.WaitGroup             // wait until every peer replied or reached hard timeout and we can forget about the request
	sentCnt, softTimeoutCnt int
}

// reqPeerCallback is called after a request sent to a certain peer has either been
// answered or timed out hard
type reqPeerCallback func(p distPeer, respTime time.Duration, srto, hrto bool)

// newRetrieveManager creates the retrieve manager
func newRetrieveManager(dist *requestDistributor, serverPool peerSelector, removePeer peerDropFn) *retrieveManager {
	return &retrieveManager{
		dist:       dist,
		serverPool: serverPool,
		removePeer: removePeer,
		sentReqs:   make(map[uint64]*sentReq),
	}
}

// retrieve sends a request (to multiple peers if necessary) and waits for an answer
// that is delivered through the deliver function and successfully validated by the
// validator callback. It returns when a valid answer is delivered or the context is
// cancelled.
func (rm *retrieveManager) retrieve(ctx context.Context, reqID uint64, req *distReq, val validatorFunc) error {
	sentReq := rm.sendReq(reqID, req, val)
	select {
	case <-sentReq.stopChn:
	case <-ctx.Done():
		sentReq.stop(ctx.Err())
	}
	return sentReq.getError()
}

// sendReq starts a process that keeps trying to retrieve a valid answer for a
// request from any suitable peers until stopped or succeeded.
func (rm *retrieveManager) sendReq(reqID uint64, req *distReq, val validatorFunc) *sentReq {
	r := &sentReq{
		rm:       rm,
		req:      req,
		sentTo:   make(map[distPeer]chan struct{}),
		stopChn:  make(chan struct{}),
		validate: val,
	}

	canSend := req.canSend
	req.canSend = func(p distPeer) bool {
		r.lock.RLock()
		_, sent := r.sentTo[p]
		r.lock.RUnlock()
		return !sent && canSend(p)
	}

	rm.lock.Lock()
	rm.sentReqs[reqID] = r
	rm.lock.Unlock()

	r.reqWg.Add(1)
	r.tryRetrieve()
	go func() {
		r.reqWg.Wait()
		rm.lock.Lock()
		delete(rm.sentReqs, reqID)
		rm.lock.Unlock()
	}()
	r.reqWg.Done()

	return r
}

// deliver is called by the LES protocol manager to deliver reply messages to waiting requests
func (rm *retrieveManager) deliver(peer distPeer, msg *Msg) error {
	rm.lock.RLock()
	req, ok := rm.sentReqs[msg.ReqID]
	rm.lock.RUnlock()
	if ok {
		ok = req.expectResponseFrom(peer)
	}

	if !ok {
		return errResp(ErrUnexpectedResponse, "reqID = %v", msg.ReqID)
	}

	if err := req.validate(peer, msg); err != nil {
		req.delivered(peer, false)
		return errResp(ErrInvalidResponse, "reqID = %v", msg.ReqID)
	}
	req.delivered(peer, true)
	return nil
}

// tryRetrieve starts a retrieval process for a single peer. If no request has been
// sent yet and no suitable peer found, it sets an ErrNoPeers error code and returns.
// Otherwise it keeps trying until it sends the request to a peer, then waits for an
// answer or a timeout.
func (r *sentReq) tryRetrieve() {
	r.reqWg.Add(1)
	go func() {
		defer r.reqWg.Done()

		for {
			sent := r.rm.dist.queue(r.req)
			var p distPeer
			select {
			case p = <-sent:
			case <-r.stopChn:
				if r.rm.dist.cancel(r.req) {
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
				r.lock.Lock()
				if r.sentTo[p] == nil {
					r.sentTo[p] = make(chan struct{})
					r.sentCnt++
				}
				r.lock.Unlock()
				respTime, srto, hrto := r.runTimer(p)
				// send feedback to server pool and remove peer if hard timeout happened
				pp, ok := p.(*peer)
				if ok && r.rm.serverPool != nil {
					r.rm.serverPool.adjustResponseTime(pp.poolEntry, respTime, srto)
				}
				if hrto {
					pp.Log().Debug("Request timed out hard")
					if r.rm.removePeer != nil {
						r.rm.removePeer(pp.id)
					}
				}
			}
			break
		}
	}()
}

// getError returns any retrieval error (either internally generated or set by the
// stop function) after stopChn has been closed
func (r *sentReq) getError() error {
	return r.err
}

// expectResponseFrom tells if we are expecting a response from a given peer.
// A response to a sent request is expected even after another peer have delivered
// a valid response. If a hard timeout occurs, no response is expected any more.
func (r *sentReq) expectResponseFrom(peer distPeer) bool {
	r.lock.Lock()
	defer r.lock.Unlock()

	return r.sentTo[peer] != nil
}

// delivered notifies the retrieval mechanism that a reply (either valid
// or invalid) has been received from a certain peer
func (r *sentReq) delivered(peer distPeer, valid bool) bool {
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
			r.tryRetrieve()
		}

	}
	return ok
}

// stop stops the retrieval process and sets an error code that will be returned
// by getError
func (r *sentReq) stop(err error) {
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
func (r *sentReq) waiting() bool {
	r.lock.RLock()
	defer r.lock.RUnlock()

	return !r.stopped && r.sentCnt+r.softTimeoutCnt != 0
}

// runTimer starts a request timeout for a single peer. If a certain (short) period
// of time passes with no reply received, it switches from "sent" to "soft timeout"
// state and the retrieval mechanism will start trying to send another request to a
// new peer. If another (longer) timeout period passes, it switches to "hard timeout"
// state, after which a reply is no longer expected and the peer should be dropped.
func (r *sentReq) runTimer(peer distPeer) (respTime time.Duration, srto, hrto bool) {
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
			r.tryRetrieve()
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

func getNextReqID() uint64 {
	var rnd [8]byte
	rand.Read(rnd[:])
	return binary.BigEndian.Uint64(rnd[:])
}
