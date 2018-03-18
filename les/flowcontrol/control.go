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

// Package flowcontrol implements a client side flow control mechanism
package flowcontrol

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

// fcTimeConst is the time constant applied for MinRecharge during linear
// buffer recharge period
const fcTimeConst = time.Millisecond

// ServerParams are the flow control parameters specified by a server for a client
//
// Note: a server can assign different amounts of bandwidth to each client by giving
// different parameters to them.
type ServerParams struct {
	BufLimit, MinRecharge uint64
}

// ClientNode is the flow control system's representation of a client
// (used in server mode only)
type ClientNode struct {
	params   *ServerParams
	bufValue uint64
	lastTime mclock.AbsTime
	sumCost  uint64            // sum of req costs received from this client
	accepted map[uint64]uint64 // value = sumCost after accepting the given req
	lock     sync.Mutex
	cm       *ClientManager
	cmNodeFields
}

// NewClientNode returns a new ClientNode
func NewClientNode(cm *ClientManager, params *ServerParams) *ClientNode {
	node := &ClientNode{
		cm:       cm,
		params:   params,
		bufValue: params.BufLimit,
		lastTime: cm.clock.Now(),
		accepted: make(map[uint64]uint64),
	}
	cm.init(node)
	return node
}

func (peer *ClientNode) recalcBV(time mclock.AbsTime) {
	dt := uint64(time - peer.lastTime)
	if time < peer.lastTime {
		dt = 0
	}
	peer.bufValue += peer.params.MinRecharge * dt / uint64(fcTimeConst)
	if peer.bufValue > peer.params.BufLimit {
		peer.bufValue = peer.params.BufLimit
	}
	peer.lastTime = time
}

// AcceptRequest returns whether a new request can be accepted and the missing
// buffer amount if it was rejected due to a buffer underrun. If accepted, maxCost
// is deducted from the flow control buffer.
func (peer *ClientNode) AcceptRequest(index, maxCost uint64) (accepted bool, bufShort uint64, priority int64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	time := peer.cm.clock.Now()
	peer.recalcBV(time)
	if maxCost > peer.bufValue {
		return false, maxCost - peer.bufValue, 0
	}
	peer.bufValue -= maxCost
	peer.sumCost += maxCost
	peer.accepted[index] = peer.sumCost
	return true, 0, peer.cm.accepted(peer, maxCost, time)
}

// RequestProcessed should be called when the request has been processed
func (peer *ClientNode) RequestProcessed(index, maxCost, servingTime uint64) (bv, realCost uint64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	time := peer.cm.clock.Now()
	peer.recalcBV(time)
	realCost = peer.cm.processed(peer, maxCost, servingTime, time)
	bv = peer.bufValue + peer.sumCost - peer.accepted[index]
	delete(peer.accepted, index)
	return
}

// ServerNode is the flow control system's representation of a server
// (used in client mode only)
type ServerNode struct {
	clock       mclock.Clock
	bufEstimate uint64
	bufRecharge bool
	lastTime    mclock.AbsTime
	params      *ServerParams
	sumCost     uint64            // sum of req costs sent to this server
	pending     map[uint64]uint64 // value = sumCost after sending the given req
	lock        sync.RWMutex
}

// NewServerNode returns a new ServerNode
func NewServerNode(params *ServerParams, clock mclock.Clock) *ServerNode {
	return &ServerNode{
		clock:       clock,
		bufEstimate: params.BufLimit,
		bufRecharge: false,
		lastTime:    clock.Now(),
		params:      params,
		pending:     make(map[uint64]uint64),
	}
}

func (peer *ServerNode) recalcBLE(time mclock.AbsTime) {
	if time < peer.lastTime {
		return
	}
	if peer.bufRecharge {
		dt := uint64(time - peer.lastTime)
		peer.bufEstimate += peer.params.MinRecharge * dt / uint64(fcTimeConst)
		if peer.bufEstimate >= peer.params.BufLimit {
			peer.bufEstimate = peer.params.BufLimit
			peer.bufRecharge = false
		}
	}
	peer.lastTime = time
}

// safetyMargin is added to the flow control waiting time when estimated buffer value is low
const safetyMargin = time.Millisecond

func (peer *ServerNode) canSend(maxCost uint64) (time.Duration, float64) {
	peer.recalcBLE(peer.clock.Now())
	maxCost += uint64(safetyMargin) * peer.params.MinRecharge / uint64(fcTimeConst)
	if maxCost > peer.params.BufLimit {
		maxCost = peer.params.BufLimit
	}
	if peer.bufEstimate >= maxCost {
		return 0, float64(peer.bufEstimate-maxCost) / float64(peer.params.BufLimit)
	}
	return time.Duration((maxCost - peer.bufEstimate) * uint64(fcTimeConst) / peer.params.MinRecharge), 0
}

// CanSend returns the minimum waiting time required before sending a request
// with the given maximum estimated cost. Second return value is the relative
// estimated buffer level after sending the request (divided by BufLimit).
func (peer *ServerNode) CanSend(maxCost uint64) (time.Duration, float64) {
	peer.lock.RLock()
	defer peer.lock.RUnlock()

	return peer.canSend(maxCost)
}

// QueuedRequest should be called when the request has been assigned to the given
// server node, before putting it in the send queue. It is mandatory that requests
// are sent in the same order as the QueuedRequest calls are made.
func (peer *ServerNode) QueuedRequest(reqID, maxCost uint64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	peer.recalcBLE(peer.clock.Now())
	// Note: we do not know when requests actually arrive to the server so bufRecharge
	// is not turned on here if buffer was full; in this case it is going to be turned
	// on by the first reply's bufValue feedback
	peer.bufEstimate -= maxCost
	peer.sumCost += maxCost
	peer.pending[reqID] = peer.sumCost
}

// ReceivedReply adjusts estimated buffer value according to the value included in
// the latest request reply.
func (peer *ServerNode) ReceivedReply(reqID, bv uint64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	peer.recalcBLE(peer.clock.Now())
	if bv > peer.params.BufLimit {
		bv = peer.params.BufLimit
	}
	sc, ok := peer.pending[reqID]
	if !ok {
		return
	}
	delete(peer.pending, reqID)
	cc := peer.sumCost - sc
	peer.bufEstimate = 0
	if bv > cc {
		peer.bufEstimate = bv - cc
	}
	peer.bufRecharge = peer.bufEstimate < peer.params.BufLimit
	peer.lastTime = peer.clock.Now()
}
