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

package flowcontrol

import (
	"fmt"
	"math"
	"sync"
	"time"

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

var (
	capFactorDropTC         = 1 / float64(time.Second*10) // time constant for dropping the capacity factor
	capFactorRaiseTC        = 1 / float64(time.Hour)      // time constant for raising the capacity factor
	capFactorRaiseThreshold = 0.75                        // connected / total capacity ratio threshold for raising the capacity factor
)

// ClientManager controls the capacity assigned to the clients of a server.
// Since ServerParams guarantee a safe lower estimate for processable requests
// even in case of all clients being active, ClientManager calculates a
// corrigated buffer value and usually allows a higher remaining buffer value
// to be returned with each reply.
type ClientManager struct {
	clock     mclock.Clock
	lock      sync.Mutex
	enabledCh chan struct{}

	curve                                      PieceWiseLinear
	sumRecharge, totalRecharge, totalConnected uint64
	capLogFactor, totalCapacity                float64
	capLastUpdate                              mclock.AbsTime
	totalCapacityCh                            chan uint64

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
		clock:         clock,
		rcQueue:       prque.New(func(a interface{}, i int) { a.(*ClientNode).queueIndex = i }),
		capLastUpdate: clock.Now(),
	}
	if curve != nil {
		cm.SetRechargeCurve(curve)
	}
	return cm
}

// SetRechargeCurve updates the recharge curve
func (cm *ClientManager) SetRechargeCurve(curve PieceWiseLinear) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	now := cm.clock.Now()
	cm.updateRecharge(now)
	cm.updateCapFactor(now, false)
	cm.curve = curve
	if len(curve) > 0 {
		cm.totalRecharge = curve[len(curve)-1].Y
	} else {
		cm.totalRecharge = 0
	}
	cm.refreshCapacity()
}

// connect should be called when a client is connected, before passing it to any
// other ClientManager function
func (cm *ClientManager) connect(node *ClientNode) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	now := cm.clock.Now()
	cm.updateRecharge(now)
	node.corrBufValue = int64(node.params.BufLimit)
	node.rcLastIntValue = cm.rcLastIntValue
	node.queueIndex = -1
	cm.updateCapFactor(now, true)
	cm.totalConnected += node.params.MinRecharge
}

// disconnect should be called when a client is disconnected
func (cm *ClientManager) disconnect(node *ClientNode) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	now := cm.clock.Now()
	cm.updateRecharge(cm.clock.Now())
	cm.updateCapFactor(now, true)
	cm.totalConnected -= node.params.MinRecharge
}

// accepted is called when a request with given maximum cost is accepted.
// It returns a priority indicator for the request which is used to determine placement
// in the serving queue. Older requests have higher priority by default. If the client
// is almost out of buffer, request priority is reduced.
func (cm *ClientManager) accepted(node *ClientNode, maxCost uint64, now mclock.AbsTime) (priority int64) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.updateNodeRc(node, -int64(maxCost), &node.params, now)
	rcTime := (node.params.BufLimit - uint64(node.corrBufValue)) * FixedPointMultiplier / node.params.MinRecharge
	return -int64(now) - int64(rcTime)
}

// processed updates the client buffer according to actual request cost after
// serving has been finished.
//
// Note: processed should always be called for all accepted requests
func (cm *ClientManager) processed(node *ClientNode, maxCost, realCost uint64, now mclock.AbsTime) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	if realCost > maxCost {
		realCost = maxCost
	}
	cm.updateNodeRc(node, int64(maxCost-realCost), &node.params, now)
	if uint64(node.corrBufValue) > node.bufValue {
		if node.log != nil {
			node.log.add(now, fmt.Sprintf("corrected  bv=%d  oldBv=%d", node.corrBufValue, node.bufValue))
		}
		node.bufValue = uint64(node.corrBufValue)
	}
}

// updateParams updates the flow control parameters of a client node
func (cm *ClientManager) updateParams(node *ClientNode, params ServerParams, now mclock.AbsTime) {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.updateRecharge(now)
	cm.updateCapFactor(now, true)
	cm.totalConnected += params.MinRecharge - node.params.MinRecharge
	cm.updateNodeRc(node, 0, &params, now)
}

// updateRecharge updates the recharge integrator and checks the recharge queue
// for nodes with recently filled buffers
func (cm *ClientManager) updateRecharge(now mclock.AbsTime) {
	lastUpdate := cm.rcLastUpdate
	cm.rcLastUpdate = now
	// updating is done in multiple steps if node buffers are filled and sumRecharge
	// is decreased before the given target time
	for cm.sumRecharge > 0 {
		bonusRatio := cm.curve.ValueAt(cm.sumRecharge) / float64(cm.sumRecharge)
		if bonusRatio < 1 {
			bonusRatio = 1
		}
		dt := now - lastUpdate
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
		lastUpdate += dtNext
		// finished recharging, update corrBufValue and sumRecharge if necessary and do next step
		if rcqNode.corrBufValue < int64(rcqNode.params.BufLimit) {
			rcqNode.corrBufValue = int64(rcqNode.params.BufLimit)
			cm.updateCapFactor(lastUpdate, true)
			cm.sumRecharge -= rcqNode.params.MinRecharge
		}
		cm.rcLastIntValue = rcqNode.rcFullIntValue
	}
}

// updateNodeRc updates a node's corrBufValue and adds an external correction value.
// It also adds or removes the rcQueue entry and updates ServerParams and sumRecharge if necessary.
func (cm *ClientManager) updateNodeRc(node *ClientNode, bvc int64, params *ServerParams, now mclock.AbsTime) {
	cm.updateRecharge(now)
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
	diff := int64(params.BufLimit - node.params.BufLimit)
	if diff > 0 {
		node.corrBufValue += diff
	}
	isFull := false
	if node.corrBufValue >= int64(params.BufLimit) {
		node.corrBufValue = int64(params.BufLimit)
		isFull = true
	}
	sumRecharge := cm.sumRecharge
	if !wasFull {
		sumRecharge -= node.params.MinRecharge
	}
	if params != &node.params {
		node.params = *params
	}
	if !isFull {
		sumRecharge += node.params.MinRecharge
		if node.queueIndex != -1 {
			cm.rcQueue.Remove(node.queueIndex)
		}
		node.rcLastIntValue = cm.rcLastIntValue
		node.rcFullIntValue = cm.rcLastIntValue + (int64(node.params.BufLimit)-node.corrBufValue)*FixedPointMultiplier/int64(node.params.MinRecharge)
		cm.rcQueue.Push(node, -node.rcFullIntValue)
	}
	if sumRecharge != cm.sumRecharge {
		cm.updateCapFactor(now, true)
		cm.sumRecharge = sumRecharge
	}

}

// updateCapFactor updates the total capacity factor. The capacity factor allows
// the total capacity of the system to go over the allowed total recharge value
// if the sum of momentarily recharging clients only exceeds the total recharge
// allowance in a very small fraction of time.
// The capacity factor is dropped quickly (with a small time constant) if sumRecharge
// exceeds totalRecharge. It is raised slowly (with a large time constant) if most
// of the total capacity is used by connected clients (totalConnected is larger than
// totalCapacity*capFactorRaiseThreshold) and sumRecharge stays under
// totalRecharge*totalConnected/totalCapacity.
func (cm *ClientManager) updateCapFactor(now mclock.AbsTime, refresh bool) {
	if cm.totalRecharge == 0 {
		return
	}
	dt := now - cm.capLastUpdate
	cm.capLastUpdate = now

	var d float64
	if cm.sumRecharge > cm.totalRecharge {
		d = (1 - float64(cm.sumRecharge)/float64(cm.totalRecharge)) * capFactorDropTC
	} else {
		totalConnected := float64(cm.totalConnected)
		var connRatio float64
		if totalConnected < cm.totalCapacity {
			connRatio = totalConnected / cm.totalCapacity
		} else {
			connRatio = 1
		}
		if connRatio > capFactorRaiseThreshold {
			sumRecharge := float64(cm.sumRecharge)
			limit := float64(cm.totalRecharge) * connRatio
			if sumRecharge < limit {
				d = (1 - sumRecharge/limit) * (connRatio - capFactorRaiseThreshold) * (1 / (1 - capFactorRaiseThreshold)) * capFactorRaiseTC
			}
		}
	}
	if d != 0 {
		cm.capLogFactor += d * float64(dt)
		if cm.capLogFactor < 0 {
			cm.capLogFactor = 0
		}
		if refresh {
			cm.refreshCapacity()
		}
	}
}

// refreshCapacity recalculates the total capacity value and sends an update to the subscription
// channel if the relative change of the value since the last update is more than 0.1 percent
func (cm *ClientManager) refreshCapacity() {
	totalCapacity := float64(cm.totalRecharge) * math.Exp(cm.capLogFactor)
	if totalCapacity >= cm.totalCapacity*0.999 && totalCapacity <= cm.totalCapacity*1.001 {
		return
	}
	cm.totalCapacity = totalCapacity
	if cm.totalCapacityCh != nil {
		select {
		case cm.totalCapacityCh <- uint64(cm.totalCapacity):
		default:
		}
	}
}

// SubscribeTotalCapacity returns all future updates to the total capacity value
// through a channel and also returns the current value
func (cm *ClientManager) SubscribeTotalCapacity(ch chan uint64) uint64 {
	cm.lock.Lock()
	defer cm.lock.Unlock()

	cm.totalCapacityCh = ch
	return uint64(cm.totalCapacity)
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
