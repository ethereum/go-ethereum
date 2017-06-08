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
	"context"
	"crypto/rand"
	"encoding/binary"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/light"
	"github.com/ethereum/go-ethereum/log"
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
	valFunc validatorFunc
	req     *distReq
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
	self.mlock.Lock()
	req, ok := self.sentReqs[msg.ReqID]
	self.mlock.Unlock()
	if ok {
		ok = req.req.expectResponseFrom(peer)
	}

	if !ok {
		return errResp(ErrUnexpectedResponse, "reqID = %v", msg.ReqID)
	}

	if err := req.valFunc(self.db, msg); err != nil {
		peer.Log().Warn("Invalid odr response", "err", err)
		req.req.delivered(peer, false)
		return errResp(ErrInvalidResponse, "reqID = %v", msg.ReqID)
	}
	req.req.delivered(peer, true)
	return nil
}

// Retrieve tries to fetch an object from the LES network.
// If the network retrieval was successful, it stores the object in local db.
func (self *LesOdr) Retrieve(ctx context.Context, req light.OdrRequest) (err error) {
	lreq := LesRequest(req)

	reqWg := new(sync.WaitGroup)
	reqWg.Add(1)
	defer reqWg.Done()

	reqID := getNextReqID()
	rq := &distReq{
		getCost: func(dp distPeer) uint64 {
			return lreq.GetCost(dp.(*peer))
		},
		canSend: func(dp distPeer) bool {
			p := dp.(*peer)
			return lreq.CanSend(p)
		},
		request: func(dp distPeer) func() {
			p := dp.(*peer)
			reqWg.Add(1)
			cost := lreq.GetCost(p)
			p.fcServer.QueueRequest(reqID, cost)
			return func() { lreq.Request(reqID, p) }
		},
	}

	sreq := &sentReq{
		valFunc: lreq.Validate,
		req:     rq,
	}

	self.mlock.Lock()
	self.sentReqs[reqID] = sreq
	self.mlock.Unlock()

	go func() {
		reqWg.Wait()
		self.mlock.Lock()
		delete(self.sentReqs, reqID)
		self.mlock.Unlock()
	}()

	stopChn := self.reqDist.retrieve(rq, func(p distPeer, respTime time.Duration, srto, hrto bool) {
		reqWg.Done()
		pp := p.(*peer)
		if self.serverPool != nil {
			self.serverPool.adjustResponseTime(pp.poolEntry, respTime, srto)
		}
		if hrto {
			pp.Log().Debug("Request timed out hard")
			self.removePeer(pp.id)
		}
	})

	select {
	case <-stopChn:
	case <-ctx.Done():
		rq.stop(ctx.Err())
	}

	if err = rq.getError(); err == nil {
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
