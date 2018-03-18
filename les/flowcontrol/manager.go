// Copyright 2018 The go-ethereum Authors
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
	"github.com/ethereum/go-ethereum/common/prque"
)

// cmNodeFields are ClientNode fields used by the client manager
// Note: these fields are locked by the client manager's mutex
type cmNodeFields struct {
	corrBufValue   int64 // buffer value adjusted with the extra recharge amount
	rcLastIntValue int64 // past recharge integrator value when corrBufValue was last updated
	rcFullIntValue int64 // future recharge integrator value when corrBufValue will reach maximum
	queueIndex     int   // position in the recharge queue (-1 if not queued)
}

// FixedPointMultiplier is applied to the recharge integrator and the recharge curve.
//
// Note: fixed point arithmetic is required for the integrator because it is a
// constantly increasing value that can wrap around int64 limits (which behavior is
// also supported by the priority queue). A floating point value would gradually lose
// precision in this application.
// The recharge curve and all recharge values are encoded as fixed point because
// sumRecharge is frequently updated by adding or subtracting individual recharge
// values and perfect precision is required.
const FixedPointMultiplier = 1000000

// ClientManager controls the bandwidth assigned to the clients of a server.
// Since ServerParams guarantee a safe lower estimate for processable requests
// even in case of all clients being active, ClientManager calculates a
// corrigated buffer value and usually allows a higher remaining buffer value
// to be returned with each reply.
type ClientManager struct {
	clock     mclock.Clock
	lock      sync.Mutex
	nodes     map[*ClientNode]struct{}
	enabledCh chan struct{}

	curve       PieceWiseLinear
	sumRecharge uint64
	// recharge integrator is increasing in each moment with a rate of
	// (totalRecharge / sumRecharge)*FixedPointMultiplier or 0 if sumRecharge==0
	rcLastUpdate   mclock.AbsTime // last time the recharge integrator was updated
	rcLastIntValue int64          // last updated value of the recharge integrator
	// recharge queue is a priority queue with currently recharging client nodes
	// as elements. The priority value is rcFullIntValue which allows to quickly
	// determine which client will first finish recharge.
	rcQueue *prque.Prque
}

// NewClientManager returns a new client manager.
// Client manager enhances flow control performance by allowing client buffers
// to recharge quicker than the minimum guaranteed recharge rate if possible.
// The sum of all minimum recharge rates (sumRecharge) is updated each time
// a clients starts or finishes buffer recharging. Then an adjusted total
// recharge rate is calculated using a piecewise linear recharge curve:
//
// totalRecharge = curve(sumRecharge)
// (totalRecharge >= sumRecharge is enforced)
//
// Then the "bonus" buffer recharge is distributed between currently recharging
// clients proportionally to their minimum recharge rates.
//
// Note: total recharge is proportional to the average number of parallel running
// serving threads. A recharge value of 1000000 corresponds to one thread in average.
// The maximum number of allowed serving threads should always be considerably
// higher than the targeted average number.
//
// Note 2: although it is possible to specify a curve allowing the total target
// recharge starting from zero sumRecharge, it makes sense to add a linear ramp
// starting from zero in order to not let a single low-priority client use up
// the entire server capacity and thus ensure quick availability for others at
// any moment.
func NewClientManager(curve PieceWiseLinear, clock mclock.Clock) *ClientManager {
	cm := &ClientManager{
		clock:   clock,
		nodes:   make(map[*ClientNode]struct{}),
		rcQueue: prque.New(func(a interface{}, i int) { a.(*ClientNode).queueIndex = i }),
		curve:   curve,
	}
	return cm
}

// SetRechargeCurve updates the recharge curve
func (cm *ClientManager) SetRechargeCurve(curve PieceWiseLinear) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.updateRecharge(cm.clock.Now())
	cm.curve = curve
}

// init initializes the ClientManager specific fields of a ClientNode structure
func (cm *ClientManager) init(node *ClientNode) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	node.corrBufValue = int64(node.params.BufLimit)
	node.rcLastIntValue = cm.rcLastIntValue
	node.queueIndex = -1
}

// accepted deduces the upper estimate for request cost from the buffer and returns a priority
// value based on current buffer status which is used by the serving queue.
func (cm *ClientManager) accepted(node *ClientNode, maxCost uint64, now mclock.AbsTime) (priority int64) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.updateNodeRc(node, -int64(maxCost), now)
	rcTime := (node.params.BufLimit - uint64(node.corrBufValue)) * FixedPointMultiplier / node.params.MinRecharge
	return -int64(now) - int64(rcTime)
}

// processed updates the client buffer according to actual request cost after
// serving has been finished.
//
// Note: processed should always be called for all accepted requests
func (cm *ClientManager) processed(node *ClientNode, maxCost, servingTime uint64, now mclock.AbsTime) (realCost uint64) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	realCost = servingTime
	if realCost > maxCost {
		realCost = maxCost
	}
	cm.updateNodeRc(node, int64(maxCost-realCost), now)
	if uint64(node.corrBufValue) > node.bufValue {
		node.bufValue = uint64(node.corrBufValue)
	}
	return
}

// updateRecharge updates the recharge integrator and checks the recharge queue
// for nodes with recently filled buffers
func (cm *ClientManager) updateRecharge(time mclock.AbsTime) {
	lastUpdate := cm.rcLastUpdate
	cm.rcLastUpdate = time
	// updating is done in multiple steps if node buffers are filled and sumRecharge
	// is decreased before the given target time
	for cm.sumRecharge > 0 {
		bonusRatio := cm.curve.ValueAt(cm.sumRecharge) / float64(cm.sumRecharge)
		if bonusRatio < 1 {
			bonusRatio = 1
		}
		dt := time - lastUpdate
		// fetch the client that finishes first
		rcqNode := cm.rcQueue.PopItem().(*ClientNode) // if sumRecharge > 0 then the queue cannot be empty
		// check whether it has already finished
		dtNext := mclock.AbsTime(float64(rcqNode.rcFullIntValue-cm.rcLastIntValue) / bonusRatio)
		if dt < dtNext {
			// not finished yet, put it back, update integrator according
			// to current bonusRatio and return
			cm.rcQueue.Push(rcqNode, -rcqNode.rcFullIntValue)
			cm.rcLastIntValue += int64(bonusRatio * float64(dt))
			return
		}
		// finished recharging, update corrBufValue and sumRecharge if necessary and do next step
		if rcqNode.corrBufValue < int64(rcqNode.params.BufLimit) {
			rcqNode.corrBufValue = int64(rcqNode.params.BufLimit)
			cm.sumRecharge -= rcqNode.params.MinRecharge
		}
		lastUpdate += dtNext
		cm.rcLastIntValue = rcqNode.rcFullIntValue
	}
}

// updateNodeRc updates a node's corrBufValue and adds an external correction value.
// It also adds or removes the rcQueue entry and updates sumRecharge if necessary.
func (cm *ClientManager) updateNodeRc(node *ClientNode, bvc int64, time mclock.AbsTime) {
	cm.updateRecharge(time)
	wasFull := true
	if node.corrBufValue != int64(node.params.BufLimit) {
		wasFull = false
		node.corrBufValue += (cm.rcLastIntValue - node.rcLastIntValue) * int64(node.params.MinRecharge) / FixedPointMultiplier
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
		if node.queueIndex != -1 {
			cm.rcQueue.Remove(node.queueIndex)
		}
		node.rcLastIntValue = cm.rcLastIntValue
		node.rcFullIntValue = cm.rcLastIntValue + (int64(node.params.BufLimit)-node.corrBufValue)*FixedPointMultiplier/int64(node.params.MinRecharge)
		cm.rcQueue.Push(node, -node.rcFullIntValue)
	}
}

// PieceWiseLinear is used to describe recharge curves
type PieceWiseLinear []struct{ X, Y uint64 }

// ValueAt returns the curve's value at a given point
func (pwl PieceWiseLinear) ValueAt(x uint64) float64 {
	l := 0
	h := len(pwl)
	if h == 0 {
		return 0
	}
	for h != l {
		m := (l + h) / 2
		if x > pwl[m].X {
			l = m + 1
		} else {
			h = m
		}
	}
	if l == 0 {
		return float64(pwl[0].Y)
	}
	l--
	if h == len(pwl) {
		return float64(pwl[l].Y)
	}
	dx := pwl[h].X - pwl[l].X
	if dx < 1 {
		return float64(pwl[l].Y)
	}
	return float64(pwl[l].Y) + float64(pwl[h].Y-pwl[l].Y)*float64(x-pwl[l].X)/float64(dx)
}

// Valid returns true if the X coordinates of the curve points are non-strictly monotonic
func (pwl PieceWiseLinear) Valid() bool {
	var lastX uint64
	for _, i := range pwl {
		if i.X < lastX {
			return false
		}
		lastX = i.X
	}
	return true
}
