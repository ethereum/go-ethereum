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

// Package access provides a layer to handle local blockchain database and
// on-demand network retrieval
package access

import (
	"errors"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/ethdb"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"golang.org/x/net/context"
)

var (
	errNotInDb = errors.New("object not found in database")
)

const LogLevel = logger.Debug

var (
	requestTimeout = time.Millisecond * 300
	retryPeers     = time.Second * 1
)

// ChainAccess provides access to blockchain and state data through local
// database and optionally also on-demand network retrieval
type ChainAccess struct {
	db         ethdb.Database
	odr        bool // light client mode, odr enabled
	lock       sync.Mutex
	sentReqs   map[uint64]*sentReq
	sentReqCnt uint64
	peers      *peerSet
}

// requestFunc is a function that requests some data from a peer
type requestFunc func(*Peer) error

// validatorFunc is a function that processes a message and returns true if
// it was a meaningful answer to a given request
type validatorFunc func(*Msg) bool

// sentReq is a request waiting for an answer that satisfies its valFunc
type sentReq struct {
	valFunc     validatorFunc
	deliverChan chan *Msg
}

// NewDbChainAccess creates a ChainAccess with ODR disabled
func NewDbChainAccess(db ethdb.Database) *ChainAccess {
	return NewChainAccess(db, false)
}

// NewChainAccess create a ChainAccess with optional ODR
func NewChainAccess(db ethdb.Database, odr bool) *ChainAccess {
	return &ChainAccess{
		db:       db,
		peers:    newPeerSet(),
		sentReqs: make(map[uint64]*sentReq),
		odr:      odr,
	}
}

// Db returns the local database assigned to the ChainAccess object
func (self *ChainAccess) Db() ethdb.Database {
	return self.db
}

// OdrEnabled returns true if this ChainAccess is capable of doing ODR requests
func (self *ChainAccess) OdrEnabled() bool {
	return self.odr
}

// RegisterPeer registers a new LES peer to the ODR capable peer set
func (self *ChainAccess) RegisterPeer(id string, version int, head common.Hash, getBlockBodies getBlockBodiesFn, getNodeData getNodeDataFn, getReceipts getReceiptsFn, getProofs getProofsFn) error {
	glog.V(logger.Detail).Infoln("Registering peer", id)
	if err := self.peers.Register(newPeer(id, version, head, getBlockBodies, getNodeData, getReceipts, getProofs)); err != nil {
		glog.V(logger.Error).Infoln("Register failed:", err)
		return err
	}
	return nil
}

// UnregisterPeer removes a peer from the ODR capable peer set
func (self *ChainAccess) UnregisterPeer(id string) {
	self.peers.Unregister(id)
}

const (
	MsgBlockBodies = iota
	MsgNodeData
	MsgReceipts
	MsgProofs
)

// Msg encodes a LES message that delivers reply data for a request
type Msg struct {
	MsgType int
	Obj     interface{}
}

// ObjectAccess is the ODR request interface (passed to Retrieve, functions called by Retrieve and Deliver)
// 		DbGet() tries to retrieve the object from the local database (object is stored by the request struct in memory if retrieved)
//		DbPut() stores it in the local database
//		Request(*Peer) requests it from a LES peer
//		Valid(*Msg) checks if a message is a valid answer to this request and stores the retrieved object in memory
type ObjectAccess interface {
	// database storage
	DbGet() bool
	DbPut()
	// network retrieval
	Request(*Peer) error
	Valid(*Msg) bool // if true, keeps the retrieved object
}

// Deliver is called by the LES protocol manager to deliver ODR reply messages to waiting requests
func (self *ChainAccess) Deliver(id string, msg *Msg) (processed bool) {
	self.lock.Lock()
	defer self.lock.Unlock()

	for i, req := range self.sentReqs {
		if req.valFunc(msg) {
			req.deliverChan <- msg
			delete(self.sentReqs, i)
			return true
		}
	}
	return false
}

// networkRequest sends a request to known peers until an answer is received
// or the context is cancelled
func (self *ChainAccess) networkRequest(ctx context.Context, rqFunc requestFunc, valFunc validatorFunc) (*Msg, error) {
	req := &sentReq{
		deliverChan: make(chan *Msg),
		valFunc:     valFunc,
	}
	self.lock.Lock()
	reqCnt := self.sentReqCnt
	self.sentReqCnt++
	self.sentReqs[reqCnt] = req
	self.lock.Unlock()

	defer func() {
		self.lock.Lock()
		delete(self.sentReqs, reqCnt)
		self.lock.Unlock()
	}()

	var msg *Msg

	for {
		peers := self.peers.BestPeers()
		if len(peers) == 0 {
			select {
			case <-ctx.Done():
				setTerminated(ctx)
				return nil, ctx.Err()
			case <-time.After(retryPeers):
			}
		}
		for _, peer := range peers {
			rqFunc(peer)
			select {
			case <-ctx.Done():
				setTerminated(ctx)
				return nil, ctx.Err()
			case msg = <-req.deliverChan:
				peer.Promote()
				glog.V(LogLevel).Infof("networkRequest success")
				return msg, nil
			case <-time.After(requestTimeout):
				peer.Demote()
				glog.V(LogLevel).Infof("networkRequest timeout")
			}
		}
	}
}

// Retrieve tries to fetch an object from the local db, then from the LES network.
// If the network retrieval was successful, it stores the object in local db.
func (self *ChainAccess) Retrieve(ctx context.Context, obj ObjectAccess) (err error) {
	// look in db
	if obj.DbGet() {
		return nil
	}
	if IsOdrContext(ctx) {
		// not found in db, trying the network
		_, err = self.networkRequest(ctx, obj.Request, obj.Valid)
		if err == nil {
			// retrieved from network, store in db
			obj.DbPut()
		} else {
			glog.V(LogLevel).Infof("networkRequest  err = %v", err)
		}
		return
	} else {
		return errNotInDb
	}
}
