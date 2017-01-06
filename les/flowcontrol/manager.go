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
)

const rcConst = 1000000

type cmNode struct {
	node                       *ClientNode
	lastUpdate                 int64
	reqAccepted                int64
	serving, recharging        bool
	rcWeight                   uint64
	rcValue, rcDelta           int64
	finishRecharge, startValue int64
}

func (node *cmNode) update(time int64) {
	dt := time - node.lastUpdate
	node.rcValue += node.rcDelta * dt / rcConst
	node.lastUpdate = time
	if node.recharging && time >= node.finishRecharge {
		node.recharging = false
		node.rcDelta = 0
		node.rcValue = 0
	}
}

func (node *cmNode) set(serving bool, simReqCnt, sumWeight uint64) {
	if node.serving && !serving {
		node.recharging = true
		sumWeight += node.rcWeight
	}
	node.serving = serving
	if node.recharging && serving {
		node.recharging = false
		sumWeight -= node.rcWeight
	}

	node.rcDelta = 0
	if serving {
		node.rcDelta = int64(rcConst / simReqCnt)
	}
	if node.recharging {
		node.rcDelta = -int64(node.node.cm.rcRecharge * node.rcWeight / sumWeight)
		node.finishRecharge = node.lastUpdate + node.rcValue*rcConst/(-node.rcDelta)
	}
}

type ClientManager struct {
	lock                             sync.Mutex
	nodes                            map[*cmNode]struct{}
	simReqCnt, sumWeight, rcSumValue uint64
	maxSimReq, maxRcSum              uint64
	rcRecharge                       uint64
	resumeQueue                      chan chan bool
	time                             int64
}

func NewClientManager(rcTarget, maxSimReq, maxRcSum uint64) *ClientManager {
	cm := &ClientManager{
		nodes:       make(map[*cmNode]struct{}),
		resumeQueue: make(chan chan bool),
		rcRecharge:  rcConst * rcConst / (100*rcConst/rcTarget - rcConst),
		maxSimReq:   maxSimReq,
		maxRcSum:    maxRcSum,
	}
	go cm.queueProc()
	return cm
}

func (self *ClientManager) Stop() {
	self.lock.Lock()
	defer self.lock.Unlock()

	// signal any waiting accept routines to return false
	self.nodes = make(map[*cmNode]struct{})
	close(self.resumeQueue)
}

func (self *ClientManager) addNode(cnode *ClientNode) *cmNode {
	time := getTime()
	node := &cmNode{
		node:           cnode,
		lastUpdate:     time,
		finishRecharge: time,
		rcWeight:       1,
	}
	self.lock.Lock()
	defer self.lock.Unlock()

	self.nodes[node] = struct{}{}
	self.update(getTime())
	return node
}

func (self *ClientManager) removeNode(node *cmNode) {
	self.lock.Lock()
	defer self.lock.Unlock()

	time := getTime()
	self.stop(node, time)
	delete(self.nodes, node)
	self.update(time)
}

// recalc sumWeight
func (self *ClientManager) updateNodes(time int64) (rce bool) {
	var sumWeight, rcSum uint64
	for node := range self.nodes {
		rc := node.recharging
		node.update(time)
		if rc && !node.recharging {
			rce = true
		}
		if node.recharging {
			sumWeight += node.rcWeight
		}
		rcSum += uint64(node.rcValue)
	}
	self.sumWeight = sumWeight
	self.rcSumValue = rcSum
	return
}

func (self *ClientManager) update(time int64) {
	for {
		firstTime := time
		for node := range self.nodes {
			if node.recharging && node.finishRecharge < firstTime {
				firstTime = node.finishRecharge
			}
		}
		if self.updateNodes(firstTime) {
			for node := range self.nodes {
				if node.recharging {
					node.set(node.serving, self.simReqCnt, self.sumWeight)
				}
			}
		} else {
			self.time = time
			return
		}
	}
}

func (self *ClientManager) canStartReq() bool {
	return self.simReqCnt < self.maxSimReq && self.rcSumValue < self.maxRcSum
}

func (self *ClientManager) queueProc() {
	for rc := range self.resumeQueue {
		for {
			time.Sleep(time.Millisecond * 10)
			self.lock.Lock()
			self.update(getTime())
			cs := self.canStartReq()
			self.lock.Unlock()
			if cs {
				break
			}
		}
		close(rc)
	}
}

func (self *ClientManager) accept(node *cmNode, time int64) bool {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.update(time)
	if !self.canStartReq() {
		resume := make(chan bool)
		self.lock.Unlock()
		self.resumeQueue <- resume
		<-resume
		self.lock.Lock()
		if _, ok := self.nodes[node]; !ok {
			return false // reject if node has been removed or manager has been stopped
		}
	}
	self.simReqCnt++
	node.set(true, self.simReqCnt, self.sumWeight)
	node.startValue = node.rcValue
	self.update(self.time)
	return true
}

func (self *ClientManager) stop(node *cmNode, time int64) {
	if node.serving {
		self.update(time)
		self.simReqCnt--
		node.set(false, self.simReqCnt, self.sumWeight)
		self.update(time)
	}
}

func (self *ClientManager) processed(node *cmNode, time int64) (rcValue, rcCost uint64) {
	self.lock.Lock()
	defer self.lock.Unlock()

	self.stop(node, time)
	return uint64(node.rcValue), uint64(node.rcValue - node.startValue)
}
