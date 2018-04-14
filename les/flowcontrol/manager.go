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

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/les/flowcontrol/prque"
)

const (
	CmDisabled        = iota // client manager is disabled, no requests are accepted
	CmNormal                 // normal operation, maximum available bandwidth can be allocated
	CmBlockProcessing        // requests are accepted but the buffers are only recharged with the guaranteed minimum rate
)

// cmNodeFields are ClientNode fields used by the client manager
// Note: these fields are locked by the client manager's mutex
type cmNodeFields struct {
	servingStarted                 mclock.AbsTime
	servingMaxCost                 uint64
	corrBufValue                   int64
	rcLastUpdate                   mclock.AbsTime
	rcLastIntValue, rcNextIntValue int64
}

// rcQueueItem represents an integrator threshold value where a certain client's buffer is recharged
type rcQueueItem struct {
	node     *ClientNode
	intValue int64
}

// Before implements prque.item
//
// Note: intValue is interpreted as mod 2^64, the difference between the highest
// and lowest value at any moment is always less than 2^63.
func rcQueueCompare(i, j interface{}) bool {
	return (j.(rcQueueItem).intValue - i.(rcQueueItem).intValue) > 0
}

// Note: valid is called under client manager mutex lock
func (rcq rcQueueItem) valid() bool {
	return rcq.intValue == rcq.node.rcNextIntValue
}

// servingQueueItem represents a queued request (prioritized by BufValue/BufLimit)
type servingQueueItem struct {
	start    func() bool
	priority float64
}

// Before implements prque.item
func servingQueueCompare(i, j interface{}) bool {
	return i.(servingQueueItem).priority > j.(servingQueueItem).priority
}

// ClientManager controls the bandwidth assigned to the clients of a server.
// Since ServerParams guarantee a safe lower estimate for processable requests
// even in case of all clients being active, ClientManager calculates a
// corrigated buffer value and usually allows a higher remaining buffer value
// to be returned with each reply.
type ClientManager struct {
	clock     mclock.Clock
	child     *ClientManager
	lock      sync.RWMutex
	nodes     map[*ClientNode]struct{}
	enabledCh chan struct{}

	parallelReqs, maxParallelReqs int
	targetParallelReqs            float64
	servingQueue                  *prque.Prque

	mode                             int
	totalRecharge                    float64
	forceMinRecharge, bufCorrEnabled bool

	sumRecharge    uint64
	rcLastUpdate   mclock.AbsTime
	rcLastIntValue int64 // normalized to MRR=1000000
	rcQueue        *prque.Prque
}

// NewClientManager returns a new client manager. Multiple client managers can
// be chained to realize priority levels. Each level has its own manager, the
// parent has the higher priority (while the parent is processing a request
// the child is disabled).
func NewClientManager(maxParallelReqs int, targetParallelReqs float64, clock mclock.Clock, child *ClientManager) *ClientManager {
	cm := &ClientManager{
		clock:        clock,
		nodes:        make(map[*ClientNode]struct{}),
		child:        child,
		servingQueue: prque.New(servingQueueCompare),
		rcQueue:      prque.New(rcQueueCompare),

		maxParallelReqs:    maxParallelReqs,
		targetParallelReqs: targetParallelReqs,
	}
	cm.SetMode(CmNormal)
	return cm
}

// SetMode changes the operating mode of the manager and its children. When
// multiple priority levels are used, mode should be changed at the manager
// of the highest level.
func (cm *ClientManager) SetMode(newMode int) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if newMode == cm.mode {
		return
	}
	cm.updateRecharge(cm.clock.Now())

	enabled := cm.mode != CmDisabled
	newEnabled := cm.mode != CmDisabled
	if !enabled && newEnabled && cm.enabledCh != nil {
		close(cm.enabledCh)
		cm.enabledCh = nil
	}
	if enabled && !newEnabled {
		cm.enabledCh = make(chan struct{})
	}

	switch newMode {
	case CmDisabled:
		cm.totalRecharge = 0
		cm.bufCorrEnabled = false
		cm.forceMinRecharge = false
	case CmNormal:
		cm.totalRecharge = cm.targetParallelReqs * 1000000
		cm.bufCorrEnabled = true
		cm.forceMinRecharge = false
	case CmBlockProcessing:
		cm.totalRecharge = 0
		cm.bufCorrEnabled = false
		cm.forceMinRecharge = true
	}

	cm.mode = newMode

	if cm.child != nil {
		if cm.parallelReqs == 0 {
			cm.child.SetMode(newMode)
		} else {
			cm.child.SetMode(CmDisabled)
		}
	}
}

func (cm *ClientManager) setParallelReqs(p int, time mclock.AbsTime) {
	if p == cm.parallelReqs {
		return
	}
	if cm.child != nil && cm.mode != CmDisabled {
		if cm.parallelReqs == 0 {
			cm.child.SetMode(CmDisabled)
		}
		if p == 0 {
			cm.child.SetMode(cm.mode)
		}
	}
	cm.parallelReqs = p
}

func (cm *ClientManager) updateRecharge(time mclock.AbsTime) {
	lastUpdate := cm.rcLastUpdate
	cm.rcLastUpdate = time
	if cm.totalRecharge == 0 {
		return
	}
	for cm.sumRecharge > 0 {
		var slope float64
		if cm.forceMinRecharge {
			slope = 1
		} else {
			slope = cm.totalRecharge / float64(cm.sumRecharge)
		}
		dt := time - lastUpdate
		q := cm.rcQueue.Pop()
		for q != nil && !q.(rcQueueItem).valid() {
			q = cm.rcQueue.Pop()
		}
		if q == nil {
			cm.rcLastIntValue += int64(slope * float64(dt))
			return
		}
		rcqItem := q.(rcQueueItem)
		dtNext := mclock.AbsTime(float64(rcqItem.intValue-cm.rcLastIntValue) / slope)
		if dt < dtNext {
			cm.rcQueue.Push(q)
			cm.rcLastIntValue += int64(slope * float64(dt))
			return
		}
		if rcqItem.node.corrBufValue < int64(rcqItem.node.params.BufLimit) {
			rcqItem.node.corrBufValue = int64(rcqItem.node.params.BufLimit)
			cm.sumRecharge -= rcqItem.node.params.MinRecharge
		}
		lastUpdate += dtNext
		cm.rcLastIntValue = rcqItem.intValue
	}
}

func (cm *ClientManager) updateNodeRc(node *ClientNode, bvc int64, time mclock.AbsTime) {
	cm.updateRecharge(time)
	wasFull := true
	if node.corrBufValue != int64(node.params.BufLimit) {
		wasFull = false
		node.corrBufValue += (cm.rcLastIntValue - node.rcLastIntValue) * int64(node.params.MinRecharge) / 1000000
		if node.corrBufValue > int64(node.params.BufLimit) {
			node.corrBufValue = int64(node.params.BufLimit)
		}
		node.rcLastIntValue = cm.rcLastIntValue
	}
	node.corrBufValue += bvc
	if node.corrBufValue < 0 {
		node.corrBufValue = 0
	}
	isFull := false
	if node.corrBufValue >= int64(node.params.BufLimit) {
		node.corrBufValue = int64(node.params.BufLimit)
		isFull = true
	}
	if wasFull && !isFull {
		cm.sumRecharge += node.params.MinRecharge
	}
	if !wasFull && isFull {
		cm.sumRecharge -= node.params.MinRecharge
	}
	if !isFull {
		node.rcLastIntValue = cm.rcLastIntValue
		node.rcNextIntValue = cm.rcLastIntValue + (int64(node.params.BufLimit)-node.corrBufValue)*1000000/int64(node.params.MinRecharge)
		cm.rcQueue.Push(rcQueueItem{node: node, intValue: node.rcNextIntValue})
	}
}

// waitOrStop blocks while request processing is disabled and returns true if
// it should be cancelled because the client has been disconnected.
func (cm *ClientManager) waitOrStop(node *ClientNode) bool {
	cm.lock.RLock()
	_, ok := cm.nodes[node]
	stop := !ok
	ch := cm.enabledCh
	cm.lock.RUnlock()

	if !stop && ch != nil {
		<-ch
		cm.lock.RLock()
		_, ok = cm.nodes[node]
		stop = !ok
		cm.lock.RUnlock()
	}

	return stop
}

func (cm *ClientManager) Stop() {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.nodes = nil
}

func (cm *ClientManager) addNode(node *ClientNode) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	node.corrBufValue = int64(node.params.BufLimit)
	node.rcLastIntValue = cm.rcLastIntValue

	if cm.nodes != nil {
		cm.nodes[node] = struct{}{}
	}
}

func (cm *ClientManager) removeNode(node *ClientNode) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if cm.nodes != nil {
		delete(cm.nodes, node)
	}
}

func (cm *ClientManager) accept(node *ClientNode, maxCost uint64, time mclock.AbsTime) chan bool {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if cm.parallelReqs == cm.maxParallelReqs {
		ch := make(chan bool, 1)
		start := func() bool {
			// always called while client manager lock is held
			_, started := cm.nodes[node]
			ch <- started
			return started
		}
		cm.servingQueue.Push(servingQueueItem{start, float64(node.bufValue) / float64(node.params.BufLimit)})
		return ch
	}

	cm.setParallelReqs(cm.parallelReqs+1, time)
	node.servingStarted = time
	node.servingMaxCost = maxCost
	cm.updateNodeRc(node, -int64(maxCost), time)
	return nil
}

func (cm *ClientManager) started(node *ClientNode, maxCost uint64, time mclock.AbsTime) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	node.servingStarted = time
	node.servingMaxCost = maxCost
	cm.updateNodeRc(node, -int64(maxCost), time)
}

func (cm *ClientManager) processed(node *ClientNode, time mclock.AbsTime) (realCost uint64) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	realCost = uint64(time - node.servingStarted)
	if realCost > node.servingMaxCost {
		realCost = node.servingMaxCost
	}
	if !cm.forceMinRecharge {
		cm.updateNodeRc(node, int64(node.servingMaxCost-realCost), time)
	}
	if cm.bufCorrEnabled {
		if uint64(node.corrBufValue) > node.bufValue {
			node.bufValue = uint64(node.corrBufValue)
		}
	}

	for !cm.servingQueue.Empty() {
		if cm.servingQueue.Pop().(servingQueueItem).start() {
			return
		}
	}
	cm.setParallelReqs(cm.parallelReqs-1, time)
	return
}
