// Copyright 2015 The go-ethereum Authors
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
package les

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"golang.org/x/net/context"
)

var (
	softRequestTimeout = time.Millisecond * 500
	hardRequestTimeout = time.Second * 10
	retryPeers         = time.Second * 1
)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

type LesOdr struct {
	light.OdrBackend
	db           ethdb.Database
	stop         chan struct{}
	removePeer   peerDropFn
	mlock, clock sync.Mutex
	sentReqs     map[uint64]*sentReq
	peers        *odrPeerSet
	lastReqID    uint64
}

func NewLesOdr(db ethdb.Database) *LesOdr {
	return &LesOdr{
		db:       db,
		stop:     make(chan struct{}),
		peers:    newOdrPeerSet(),
		sentReqs: make(map[uint64]*sentReq),
	}
}

func (odr *LesOdr) Stop() {
	close(odr.stop)
}

func (odr *LesOdr) Database() ethdb.Database {
	return odr.db
}

// validatorFunc is a function that processes a message and returns true if
// it was a meaningful answer to a given request
type validatorFunc func(ethdb.Database, *Msg) bool

// sentReq is a request waiting for an answer that satisfies its valFunc
type sentReq struct {
	valFunc  validatorFunc
	sentTo   map[*peer]chan struct{}
	lock     sync.RWMutex  // protects acces to sentTo
	answered chan struct{} // closed and set to nil when any peer answers it
}

// RegisterPeer registers a new LES peer to the ODR capable peer set
func (self *LesOdr) RegisterPeer(p *peer) error {
	return self.peers.register(p)
}

// UnregisterPeer removes a peer from the ODR capable peer set
func (self *LesOdr) UnregisterPeer(p *peer) {
	self.peers.unregister(p)
}

const (
	MsgBlockBodies = iota
	MsgCode
	MsgReceipts
	MsgProofs
	MsgHeaderProofs
)

// Msg encodes a LES message that delivers reply data for a request
type Msg struct {
	MsgType int
	ReqID   uint64
	Obj     interface{}
}

// Deliver is called by the LES protocol manager to deliver ODR reply messages to waiting requests
func (self *LesOdr) Deliver(peer *peer, msg *Msg) error {
	var delivered chan struct{}
	self.mlock.Lock()
	req, ok := self.sentReqs[msg.ReqID]
	self.mlock.Unlock()
	if ok {
		req.lock.Lock()
		delivered, ok = req.sentTo[peer]
		req.lock.Unlock()
	}

	if !ok {
		return errResp(ErrUnexpectedResponse, "reqID = %v", msg.ReqID)
	}

	if req.valFunc(self.db, msg) {
		close(delivered)
		req.lock.Lock()
		if req.answered != nil {
			close(req.answered)
			req.answered = nil
		}
		req.lock.Unlock()
		return nil
	}
	return errResp(ErrInvalidResponse, "reqID = %v", msg.ReqID)
}

func (self *LesOdr) requestPeer(req *sentReq, peer *peer, delivered, timeout chan struct{}, reqWg *sync.WaitGroup) {
	stime := mclock.Now()
	defer func() {
		req.lock.Lock()
		delete(req.sentTo, peer)
		req.lock.Unlock()
		reqWg.Done()
	}()

	select {
	case <-delivered:
		servTime := uint64(mclock.Now() - stime)
		self.peers.updateTimeout(peer, false)
		self.peers.updateServTime(peer, servTime)
		return
	case <-time.After(softRequestTimeout):
		close(timeout)
		if self.peers.updateTimeout(peer, true) {
			self.removePeer(peer.id)
		}
	case <-self.stop:
		return
	}

	select {
	case <-delivered:
		servTime := uint64(mclock.Now() - stime)
		self.peers.updateServTime(peer, servTime)
		return
	case <-time.After(hardRequestTimeout):
		self.removePeer(peer.id)
	case <-self.stop:
		return
	}
}

// networkRequest sends a request to known peers until an answer is received
// or the context is cancelled
func (self *LesOdr) networkRequest(ctx context.Context, lreq LesOdrRequest) error {
	answered := make(chan struct{})
	req := &sentReq{
		valFunc:  lreq.Valid,
		sentTo:   make(map[*peer]chan struct{}),
		answered: answered, // reply delivered by any peer
	}
	reqID := self.getNextReqID()
	self.mlock.Lock()
	self.sentReqs[reqID] = req
	self.mlock.Unlock()

	reqWg := new(sync.WaitGroup)
	reqWg.Add(1)
	defer reqWg.Done()
	go func() {
		reqWg.Wait()
		self.mlock.Lock()
		delete(self.sentReqs, reqID)
		self.mlock.Unlock()
	}()

	exclude := make(map[*peer]struct{})
	for {
		if peer := self.peers.bestPeer(lreq, exclude); peer == nil {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-req.answered:
				return nil
			case <-time.After(retryPeers):
			}
		} else {
			exclude[peer] = struct{}{}
			delivered := make(chan struct{})
			timeout := make(chan struct{})
			req.lock.Lock()
			req.sentTo[peer] = delivered
			req.lock.Unlock()
			reqWg.Add(1)
			cost := lreq.GetCost(peer)
			peer.fcServer.SendRequest(reqID, cost)
			go self.requestPeer(req, peer, delivered, timeout, reqWg)
			lreq.Request(reqID, peer)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-answered:
				return nil
			case <-timeout:
			}
		}
	}
}

// Retrieve tries to fetch an object from the local db, then from the LES network.
// If the network retrieval was successful, it stores the object in local db.
func (self *LesOdr) Retrieve(ctx context.Context, req light.OdrRequest) (err error) {
	lreq := LesRequest(req)
	err = self.networkRequest(ctx, lreq)
	if err == nil {
		// retrieved from network, store in db
		req.StoreResult(self.db)
	} else {
		glog.V(logger.Debug).Infof("networkRequest  err = %v", err)
	}
	return
}

func (self *LesOdr) getNextReqID() uint64 {
	self.clock.Lock()
	defer self.clock.Unlock()

	self.lastReqID++
	return self.lastReqID
}
