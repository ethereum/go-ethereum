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

const fcTimeConst = time.Millisecond

type ServerParams struct {
	BufLimit, MinRecharge uint64
}

type ClientNode struct {
	params   *ServerParams
	bufValue uint64
	lastTime mclock.AbsTime
	lock     sync.Mutex
	cm       *ClientManager
	cmNode   *cmNode
}

func NewClientNode(cm *ClientManager, params *ServerParams) *ClientNode {
	node := &ClientNode{
		cm:       cm,
		params:   params,
		bufValue: params.BufLimit,
		lastTime: mclock.Now(),
	}
	node.cmNode = cm.addNode(node)
	return node
}

func (peer *ClientNode) Remove(cm *ClientManager) {
	cm.removeNode(peer.cmNode)
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

func (peer *ClientNode) AcceptRequest() (uint64, bool) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	time := mclock.Now()
	peer.recalcBV(time)
	return peer.bufValue, peer.cm.accept(peer.cmNode, time)
}

func (peer *ClientNode) RequestProcessed(cost uint64) (bv, realCost uint64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	time := mclock.Now()
	peer.recalcBV(time)
	peer.bufValue -= cost
	peer.recalcBV(time)
	rcValue, rcost := peer.cm.processed(peer.cmNode, time)
	if rcValue < peer.params.BufLimit {
		bv := peer.params.BufLimit - rcValue
		if bv > peer.bufValue {
			peer.bufValue = bv
		}
	}
	return peer.bufValue, rcost
}

type ServerNode struct {
	bufEstimate     uint64
	lastTime        mclock.AbsTime
	params          *ServerParams
	sumCost         uint64            // sum of req costs sent to this server
	pending         map[uint64]uint64 // value = sumCost after sending the given req
	assignedRequest uint64            // when != 0, only the request with the given ID can be sent to this peer
	assignToken     chan struct{}     // send to this channel before assigning, read from it after deassigning
	lock            sync.RWMutex
}

func NewServerNode(params *ServerParams) *ServerNode {
	return &ServerNode{
		bufEstimate: params.BufLimit,
		lastTime:    mclock.Now(),
		params:      params,
		pending:     make(map[uint64]uint64),
		assignToken: make(chan struct{}, 1),
	}
}

func (peer *ServerNode) recalcBLE(time mclock.AbsTime) {
	dt := uint64(time - peer.lastTime)
	if time < peer.lastTime {
		dt = 0
	}
	peer.bufEstimate += peer.params.MinRecharge * dt / uint64(fcTimeConst)
	if peer.bufEstimate > peer.params.BufLimit {
		peer.bufEstimate = peer.params.BufLimit
	}
	peer.lastTime = time
}

// safetyMargin is added to the flow control waiting time when estimated buffer value is low
const safetyMargin = time.Millisecond * 200

func (peer *ServerNode) canSend(maxCost uint64) time.Duration {
	maxCost += uint64(safetyMargin) * peer.params.MinRecharge / uint64(fcTimeConst)
	if maxCost > peer.params.BufLimit {
		maxCost = peer.params.BufLimit
	}
	if peer.bufEstimate >= maxCost {
		return 0
	}
	return time.Duration((maxCost - peer.bufEstimate) * uint64(fcTimeConst) / peer.params.MinRecharge)
}

// CanSend returns the minimum waiting time required before sending a request
// with the given maximum estimated cost
func (peer *ServerNode) CanSend(maxCost uint64) time.Duration {
	peer.lock.RLock()
	defer peer.lock.RUnlock()

	return peer.canSend(maxCost)
}

// AssignRequest tries to assign the server node to the given request, guaranteeing
// that once it returns true, no request will be sent to the node before this one
func (peer *ServerNode) AssignRequest(reqID uint64) bool {
	select {
	case peer.assignToken <- struct{}{}:
	default:
		return false
	}
	peer.lock.Lock()
	peer.assignedRequest = reqID
	peer.lock.Unlock()
	return true
}

// MustAssignRequest waits until the node can be assigned to the given request.
// It is always guaranteed that assignments are released in a short amount of time.
func (peer *ServerNode) MustAssignRequest(reqID uint64) {
	peer.assignToken <- struct{}{}
	peer.lock.Lock()
	peer.assignedRequest = reqID
	peer.lock.Unlock()
}

// DeassignRequest releases a request assignment in case the planned request
// is not being sent.
func (peer *ServerNode) DeassignRequest(reqID uint64) {
	peer.lock.Lock()
	if peer.assignedRequest == reqID {
		peer.assignedRequest = 0
		<-peer.assignToken
	}
	peer.lock.Unlock()
}

// IsAssigned returns true if the server node has already been assigned to a request
// (note that this function returning false does not guarantee that you can assign a request
// immediately afterwards, its only purpose is to help peer selection)
func (peer *ServerNode) IsAssigned() bool {
	peer.lock.RLock()
	locked := peer.assignedRequest != 0
	peer.lock.RUnlock()
	return locked
}

// blocks until request can be sent
func (peer *ServerNode) SendRequest(reqID, maxCost uint64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	if peer.assignedRequest != reqID {
		peer.lock.Unlock()
		peer.MustAssignRequest(reqID)
		peer.lock.Lock()
	}

	peer.recalcBLE(mclock.Now())
	wait := peer.canSend(maxCost)
	for wait > 0 {
		peer.lock.Unlock()
		time.Sleep(wait)
		peer.lock.Lock()
		peer.recalcBLE(mclock.Now())
		wait = peer.canSend(maxCost)
	}
	peer.assignedRequest = 0
	<-peer.assignToken
	peer.bufEstimate -= maxCost
	peer.sumCost += maxCost
	if reqID >= 0 {
		peer.pending[reqID] = peer.sumCost
	}
}

func (peer *ServerNode) GotReply(reqID, bv uint64) {

	peer.lock.Lock()
	defer peer.lock.Unlock()

	if bv > peer.params.BufLimit {
		bv = peer.params.BufLimit
	}
	sc, ok := peer.pending[reqID]
	if !ok {
		return
	}
	delete(peer.pending, reqID)
	peer.bufEstimate = bv - (peer.sumCost - sc)
	peer.lastTime = mclock.Now()
}
