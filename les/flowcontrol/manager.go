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

const rcConst = 1000000

type cmNode struct {
	node                         *ClientNode
	lastUpdate                   mclock.AbsTime
	serving, recharging          bool
	rcWeight                     uint64
	rcValue, rcDelta, startValue int64
	finishRecharge               mclock.AbsTime
}

func (node *cmNode) update(time mclock.AbsTime) {
	dt := int64(time - node.lastUpdate)
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
		node.finishRecharge = node.lastUpdate + mclock.AbsTime(node.rcValue*rcConst/(-node.rcDelta))
	}
}

type ClientManager struct {
	lock                             sync.Mutex
	nodes                            map[*cmNode]struct{}
	simReqCnt, sumWeight, rcSumValue uint64
	maxSimReq, maxRcSum              uint64
	rcRecharge                       uint64
	resumeQueue                      chan chan bool
	time                             mclock.AbsTime
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

func (cm *ClientManager) Stop() {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	// signal any waiting accept routines to return false
	cm.nodes = make(map[*cmNode]struct{})
	close(cm.resumeQueue)
}

func (cm *ClientManager) addNode(cnode *ClientNode) *cmNode {
	time := mclock.Now()
	node := &cmNode{
		node:           cnode,
		lastUpdate:     time,
		finishRecharge: time,
		rcWeight:       1,
	}
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.nodes[node] = struct{}{}
	cm.update(mclock.Now())
	return node
}

func (cm *ClientManager) removeNode(node *cmNode) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	time := mclock.Now()
	cm.stop(node, time)
	delete(cm.nodes, node)
	cm.update(time)
}

// recalc sumWeight
func (cm *ClientManager) updateNodes(time mclock.AbsTime) (rce bool) {
	var sumWeight, rcSum uint64
	for node := range cm.nodes {
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
	cm.sumWeight = sumWeight
	cm.rcSumValue = rcSum
	return
}

func (cm *ClientManager) update(time mclock.AbsTime) {
	for {
		firstTime := time
		for node := range cm.nodes {
			if node.recharging && node.finishRecharge < firstTime {
				firstTime = node.finishRecharge
			}
		}
		if cm.updateNodes(firstTime) {
			for node := range cm.nodes {
				if node.recharging {
					node.set(node.serving, cm.simReqCnt, cm.sumWeight)
				}
			}
		} else {
			cm.time = time
			return
		}
	}
}

func (cm *ClientManager) canStartReq() bool {
	return cm.simReqCnt < cm.maxSimReq && cm.rcSumValue < cm.maxRcSum
}

func (cm *ClientManager) queueProc() {
	for rc := range cm.resumeQueue {
		for {
			time.Sleep(time.Millisecond * 10)
			cm.lock.Lock()
			cm.update(mclock.Now())
			cs := cm.canStartReq()
			cm.lock.Unlock()
			if cs {
				break
			}
		}
		close(rc)
	}
}

func (cm *ClientManager) accept(node *cmNode, time mclock.AbsTime) bool {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.update(time)
	if !cm.canStartReq() {
		resume := make(chan bool)
		cm.lock.Unlock()
		cm.resumeQueue <- resume
		<-resume
		cm.lock.Lock()
		if _, ok := cm.nodes[node]; !ok {
			return false // reject if node has been removed or manager has been stopped
		}
	}
	cm.simReqCnt++
	node.set(true, cm.simReqCnt, cm.sumWeight)
	node.startValue = node.rcValue
	cm.update(cm.time)
	return true
}

func (cm *ClientManager) stop(node *cmNode, time mclock.AbsTime) {
	if node.serving {
		cm.update(time)
		cm.simReqCnt--
		node.set(false, cm.simReqCnt, cm.sumWeight)
		cm.update(time)
	}
}

func (cm *ClientManager) processed(node *cmNode, time mclock.AbsTime) (rcValue, rcCost uint64) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.stop(node, time)
	return uint64(node.rcValue), uint64(node.rcValue - node.startValue)
}
