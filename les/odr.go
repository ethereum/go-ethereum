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

	"github.com/ubiq/go-ubiq/ethdb"
	"github.com/ubiq/go-ubiq/light"
	"github.com/ubiq/go-ubiq/log"
)

// LesOdr implements light.OdrBackend
type LesOdr struct {
	db        ethdb.Database
	stop      chan struct{}
	retriever *retrieveManager
}

func NewLesOdr(db ethdb.Database, retriever *retrieveManager) *LesOdr {
	return &LesOdr{
		db:        db,
		retriever: retriever,
		stop:      make(chan struct{}),
	}
}

func (odr *LesOdr) Stop() {
	close(odr.stop)
}

func (odr *LesOdr) Database() ethdb.Database {
	return odr.db
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

// Retrieve tries to fetch an object from the LES network.
// If the network retrieval was successful, it stores the object in local db.
func (self *LesOdr) Retrieve(ctx context.Context, req light.OdrRequest) (err error) {
	lreq := LesRequest(req)

	reqID := genReqID()
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
			cost := lreq.GetCost(p)
			p.fcServer.QueueRequest(reqID, cost)
			return func() { lreq.Request(reqID, p) }
		},
	}

	if err = self.retriever.retrieve(ctx, reqID, rq, func(p distPeer, msg *Msg) error { return lreq.Validate(self.db, msg) }); err == nil {
		// retrieved from network, store in db
		req.StoreResult(self.db)
	} else {
		log.Debug("Failed to retrieve data from network", "err", err)
	}
	return
}
