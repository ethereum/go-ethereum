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

package les

import (
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
	"golang.org/x/net/context"
)

var (
	softRequestTimeout = time.Millisecond * 500
	hardRequestTimeout = time.Second * 10
)

// peerDropFn is a callback type for dropping a peer detected as malicious.
type peerDropFn func(id string)

type odrPeerSelector interface {
	adjustResponseTime(*poolEntry, time.Duration, bool)
}

type LesOdr struct {
	light.OdrBackend
	db           ethdb.Database
	stop         chan struct{}
	removePeer   peerDropFn
	mlock, clock sync.Mutex
	sentReqs     map[uint64]*sentReq
	serverPool   odrPeerSelector
	reqDist      *requestDistributor
}

func NewLesOdr(db ethdb.Database) *LesOdr {
	return &LesOdr{
		db:       db,
		stop:     make(chan struct{}),
		sentReqs: make(map[uint64]*sentReq),
	}
}

func (odr *LesOdr) Stop() {
	close(odr.stop)
}

func (odr *LesOdr) Database() ethdb.Database {
	return odr.db
}

// validatorFunc is a function that processes a message.
type validatorFunc func(ethdb.Database, *Msg) error

// sentReq is a request waiting for an answer that satisfies its valFunc
type sentReq struct {
	valFunc  validatorFunc
	sentTo   map[*peer]chan struct{}
	lock     sync.RWMutex  // protects acces to sentTo
	answered chan struct{} // closed and set to nil when any peer answers it
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

	if err := req.valFunc(self.db, msg); err != nil {
		peer.Log().Warn("Invalid odr response", "err", err)
		return errResp(ErrInvalidResponse, "reqID = %v", msg.ReqID)
	}
	close(delivered)
	req.lock.Lock()
	delete(req.sentTo, peer)
	if req.answered != nil {
		close(req.answered)
		req.answered = nil
	}
	req.lock.Unlock()
	return nil
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
		if self.serverPool != nil {
			self.serverPool.adjustResponseTime(peer.poolEntry, time.Duration(mclock.Now()-stime), false)
		}
		return
	case <-time.After(softRequestTimeout):
		close(timeout)
	case <-self.stop:
		return
	}

	select {
	case <-delivered:
	case <-time.After(hardRequestTimeout):
		peer.Log().Debug("Request timed out hard")
		go self.removePeer(peer.id)
	case <-self.stop:
		return
	}
	if self.serverPool != nil {
		self.serverPool.adjustResponseTime(peer.poolEntry, time.Duration(mclock.Now()-stime), true)
	}
}

// networkRequest sends a request to known peers until an answer is received
// or the context is cancelled
func (self *LesOdr) networkRequest(ctx context.Context, lreq LesOdrRequest) error {
	answered := make(chan struct{})
	req := &sentReq{
		valFunc:  lreq.Validate,
		sentTo:   make(map[*peer]chan struct{}),
		answered: answered, // reply delivered by any peer
	}

	exclude := make(map[*peer]struct{})

	reqWg := new(sync.WaitGroup)
	reqWg.Add(1)
	defer reqWg.Done()

	var timeout chan struct{}
	reqID := getNextReqID()
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			return lreq.GetCost(dp.(*peer))
		},
		canSend: func(dp distPeer) bool {
			p := dp.(*peer)
			_, ok := exclude[p]
			return !ok && lreq.CanSend(p)
		},
		request: func(dp distPeer) func() {
			p := dp.(*peer)
			exclude[p] = struct{}{}
			delivered := make(chan struct{})
			timeout = make(chan struct{})
			req.lock.Lock()
			req.sentTo[p] = delivered
			req.lock.Unlock()
			reqWg.Add(1)
			cost := lreq.GetCost(p)
			p.fcServer.QueueRequest(reqID, cost)
			go self.requestPeer(req, p, delivered, timeout, reqWg)
			return func() { lreq.Request(reqID, p) }
		},
	}

	self.mlock.Lock()
	self.sentReqs[reqID] = req
	self.mlock.Unlock()

	go func() {
		reqWg.Wait()
		self.mlock.Lock()
		delete(self.sentReqs, reqID)
		self.mlock.Unlock()
	}()

	for {
		peerChn := self.reqDist.queue(rq)
		select {
		case <-ctx.Done():
			self.reqDist.cancel(rq)
			return ctx.Err()
		case <-answered:
			self.reqDist.cancel(rq)
			return nil
		case _, ok := <-peerChn:
			if !ok {
				return ErrNoPeers
			}
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-answered:
			return nil
		case <-timeout:
		}
	}
}

// Retrieve tries to fetch an object from the LES network.
// If the network retrieval was successful, it stores the object in local db.
func (self *LesOdr) Retrieve(ctx context.Context, req light.OdrRequest) (err error) {
	lreq := LesRequest(req)
	err = self.networkRequest(ctx, lreq)
	if err == nil {
		// retrieved from network, store in db
		req.StoreResult(self.db)
	} else {
		log.Debug("Failed to retrieve data from network", "err", err)
	}
	return
}

func getNextReqID() uint64 {
	var rnd [8]byte
	rand.Read(rnd[:])
	return binary.BigEndian.Uint64(rnd[:])
}
