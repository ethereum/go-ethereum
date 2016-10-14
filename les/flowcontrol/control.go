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

// Package flowcontrol implements a client side flow control mechanism
package flowcontrol

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

const fcTimeConst = 1000000

type ServerParams struct {
	BufLimit, MinRecharge uint64
}

type ClientNode struct {
	params   *ServerParams
	bufValue uint64
	lastTime int64
	lock     sync.Mutex
	cm       *ClientManager
	cmNode   *cmNode
}

func NewClientNode(cm *ClientManager, params *ServerParams) *ClientNode {
	node := &ClientNode{
		cm:       cm,
		params:   params,
		bufValue: params.BufLimit,
		lastTime: getTime(),
	}
	node.cmNode = cm.addNode(node)
	return node
}

func (peer *ClientNode) Remove(cm *ClientManager) {
	cm.removeNode(peer.cmNode)
}

func (peer *ClientNode) recalcBV(time int64) {
	dt := uint64(time - peer.lastTime)
	if time < peer.lastTime {
		dt = 0
	}
	peer.bufValue += peer.params.MinRecharge * dt / fcTimeConst
	if peer.bufValue > peer.params.BufLimit {
		peer.bufValue = peer.params.BufLimit
	}
	peer.lastTime = time
}

func (peer *ClientNode) AcceptRequest() (uint64, bool) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	time := getTime()
	peer.recalcBV(time)
	return peer.bufValue, peer.cm.accept(peer.cmNode, time)
}

func (peer *ClientNode) RequestProcessed(cost uint64) (bv, realCost uint64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	time := getTime()
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
	bufEstimate uint64
	lastTime    int64
	params      *ServerParams
	sumCost     uint64            // sum of req costs sent to this server
	pending     map[uint64]uint64 // value = sumCost after sending the given req
	lock        sync.Mutex
}

func NewServerNode(params *ServerParams) *ServerNode {
	return &ServerNode{
		bufEstimate: params.BufLimit,
		lastTime:    getTime(),
		params:      params,
		pending:     make(map[uint64]uint64),
	}
}

func getTime() int64 {
	return int64(mclock.Now())
}

func (peer *ServerNode) recalcBLE(time int64) {
	dt := uint64(time - peer.lastTime)
	if time < peer.lastTime {
		dt = 0
	}
	peer.bufEstimate += peer.params.MinRecharge * dt / fcTimeConst
	if peer.bufEstimate > peer.params.BufLimit {
		peer.bufEstimate = peer.params.BufLimit
	}
	peer.lastTime = time
}

func (peer *ServerNode) canSend(maxCost uint64) uint64 {
	if peer.bufEstimate >= maxCost {
		return 0
	}
	return (maxCost - peer.bufEstimate) * fcTimeConst / peer.params.MinRecharge
}

func (peer *ServerNode) CanSend(maxCost uint64) uint64 {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	return peer.canSend(maxCost)
}

// blocks until request can be sent
func (peer *ServerNode) SendRequest(reqID, maxCost uint64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	peer.recalcBLE(getTime())
	for peer.bufEstimate < maxCost {
		time.Sleep(time.Duration(peer.canSend(maxCost)))
		peer.recalcBLE(getTime())
	}
	peer.bufEstimate -= maxCost
	peer.sumCost += maxCost
	if reqID >= 0 {
		peer.pending[reqID] = peer.sumCost
	}
}

func (peer *ServerNode) GotReply(reqID, bv uint64) {
	peer.lock.Lock()
	defer peer.lock.Unlock()

	sc, ok := peer.pending[reqID]
	if !ok {
		return
	}
	delete(peer.pending, reqID)
	peer.bufEstimate = bv - (peer.sumCost - sc)
	peer.lastTime = getTime()
}
