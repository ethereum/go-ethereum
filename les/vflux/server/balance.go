// Copyright 2019 The go-ethereum Authors
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

package server

import (
	"errors"
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/les/utils"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
)

var errBalanceOverflow = errors.New("balance overflow")

const maxBalance = math.MaxInt64 // maximum allowed balance value

const (
	balanceCallbackUpdate = iota // called when priority drops below the last minimum estimate
	balanceCallbackZero          // called when priority drops to zero (positive balance exhausted)
	balanceCallbackCount         // total number of balance callbacks
)

// PriceFactors determine the pricing policy (may apply either to positive or
// negative balances which may have different factors).
// - TimeFactor is cost unit per nanosecond of connection time
// - CapacityFactor is cost unit per nanosecond of connection time per 1000000 capacity
// - RequestFactor is cost unit per request "realCost" unit
type PriceFactors struct {
	TimeFactor, CapacityFactor, RequestFactor float64
}

// timePrice returns the price of connection per nanosecond at the given capacity
func (p PriceFactors) timePrice(cap uint64) float64 {
	return p.TimeFactor + float64(cap)*p.CapacityFactor/1000000
}

// NodeBalance keeps track of the positive and negative balances of a connected
// client and calculates actual and projected future priority values.
// Implements nodePriority interface.
type NodeBalance struct {
	bt                               *BalanceTracker
	lock                             sync.RWMutex
	node                             *enode.Node
	connAddress                      string
	active                           bool
	priority                         bool
	capacity                         uint64
	balance                          balance
	posFactor, negFactor             PriceFactors
	sumReqCost                       uint64
	lastUpdate, nextUpdate, initTime mclock.AbsTime
	updateEvent                      mclock.Timer
	// since only a limited and fixed number of callbacks are needed, they are
	// stored in a fixed size array ordered by priority threshold.
	callbacks [balanceCallbackCount]balanceCallback
	// callbackIndex maps balanceCallback constants to callbacks array indexes (-1 if not active)
	callbackIndex [balanceCallbackCount]int
	callbackCount int // number of active callbacks
}

// balance represents a pair of positive and negative balances
type balance struct {
	pos, neg utils.ExpiredValue
}

// balanceCallback represents a single callback that is activated when client priority
// reaches the given threshold
type balanceCallback struct {
	id        int
	threshold int64
	callback  func()
}

// GetBalance returns the current positive and negative balance.
func (n *NodeBalance) GetBalance() (uint64, uint64) {
	n.lock.Lock()
	defer n.lock.Unlock()

	now := n.bt.clock.Now()
	n.updateBalance(now)
	return n.balance.pos.Value(n.bt.posExp.LogOffset(now)), n.balance.neg.Value(n.bt.negExp.LogOffset(now))
}

// GetRawBalance returns the current positive and negative balance
// but in the raw(expired value) format.
func (n *NodeBalance) GetRawBalance() (utils.ExpiredValue, utils.ExpiredValue) {
	n.lock.Lock()
	defer n.lock.Unlock()

	now := n.bt.clock.Now()
	n.updateBalance(now)
	return n.balance.pos, n.balance.neg
}

// AddBalance adds the given amount to the positive balance and returns the balance
// before and after the operation. Exceeding maxBalance results in an error (balance is
// unchanged) while adding a negative amount higher than the current balance results in
// zero balance.
func (n *NodeBalance) AddBalance(amount int64) (uint64, uint64, error) {
	var (
		err      error
		old, new uint64
	)
	n.bt.ns.Operation(func() {
		var (
			callbacks   []func()
			setPriority bool
		)
		n.bt.updateTotalBalance(n, func() bool {
			now := n.bt.clock.Now()
			n.updateBalance(now)

			// Ensure the given amount is valid to apply.
			offset := n.bt.posExp.LogOffset(now)
			old = n.balance.pos.Value(offset)
			if amount > 0 && (amount > maxBalance || old > maxBalance-uint64(amount)) {
				err = errBalanceOverflow
				return false
			}

			// Update the total positive balance counter.
			n.balance.pos.Add(amount, offset)
			callbacks = n.checkCallbacks(now)
			setPriority = n.checkPriorityStatus()
			new = n.balance.pos.Value(offset)
			n.storeBalance(true, false)
			return true
		})
		for _, cb := range callbacks {
			cb()
		}
		if setPriority {
			n.bt.ns.SetStateSub(n.node, n.bt.PriorityFlag, nodestate.Flags{}, 0)
		}
		n.signalPriorityUpdate()
	})
	if err != nil {
		return old, old, err
	}

	return old, new, nil
}

// SetBalance sets the positive and negative balance to the given values
func (n *NodeBalance) SetBalance(pos, neg uint64) error {
	if pos > maxBalance || neg > maxBalance {
		return errBalanceOverflow
	}
	n.bt.ns.Operation(func() {
		var (
			callbacks   []func()
			setPriority bool
		)
		n.bt.updateTotalBalance(n, func() bool {
			now := n.bt.clock.Now()
			n.updateBalance(now)

			var pb, nb utils.ExpiredValue
			pb.Add(int64(pos), n.bt.posExp.LogOffset(now))
			nb.Add(int64(neg), n.bt.negExp.LogOffset(now))
			n.balance.pos = pb
			n.balance.neg = nb
			callbacks = n.checkCallbacks(now)
			setPriority = n.checkPriorityStatus()
			n.storeBalance(true, true)
			return true
		})
		for _, cb := range callbacks {
			cb()
		}
		if setPriority {
			n.bt.ns.SetStateSub(n.node, n.bt.PriorityFlag, nodestate.Flags{}, 0)
		}
		n.signalPriorityUpdate()
	})
	return nil
}

// RequestServed should be called after serving a request for the given peer
func (n *NodeBalance) RequestServed(cost uint64) uint64 {
	n.lock.Lock()
	var callbacks []func()
	defer func() {
		n.lock.Unlock()
		if callbacks != nil {
			n.bt.ns.Operation(func() {
				for _, cb := range callbacks {
					cb()
				}
			})
		}
	}()

	now := n.bt.clock.Now()
	n.updateBalance(now)
	fcost := float64(cost)

	posExp := n.bt.posExp.LogOffset(now)
	var check bool
	if !n.balance.pos.IsZero() {
		if n.posFactor.RequestFactor != 0 {
			c := -int64(fcost * n.posFactor.RequestFactor)
			cc := n.balance.pos.Add(c, posExp)
			if c == cc {
				fcost = 0
			} else {
				fcost *= 1 - float64(cc)/float64(c)
			}
			check = true
		} else {
			fcost = 0
		}
	}
	if fcost > 0 {
		if n.negFactor.RequestFactor != 0 {
			n.balance.neg.Add(int64(fcost*n.negFactor.RequestFactor), n.bt.negExp.LogOffset(now))
			check = true
		}
	}
	if check {
		callbacks = n.checkCallbacks(now)
	}
	n.sumReqCost += cost
	return n.balance.pos.Value(posExp)
}

// Priority returns the actual priority based on the current balance
func (n *NodeBalance) Priority(now mclock.AbsTime, capacity uint64) int64 {
	n.lock.Lock()
	defer n.lock.Unlock()

	n.updateBalance(now)
	return n.balanceToPriority(n.balance, capacity)
}

// EstMinPriority gives a lower estimate for the priority at a given time in the future.
// An average request cost per time is assumed that is twice the average cost per time
// in the current session.
// If update is true then a priority callback is added that turns UpdateFlag on and off
// in case the priority goes below the estimated minimum.
func (n *NodeBalance) EstMinPriority(at mclock.AbsTime, capacity uint64, update bool) int64 {
	n.lock.Lock()
	defer n.lock.Unlock()

	var avgReqCost float64
	dt := time.Duration(n.lastUpdate - n.initTime)
	if dt > time.Second {
		avgReqCost = float64(n.sumReqCost) * 2 / float64(dt)
	}
	pri := n.balanceToPriority(n.reducedBalance(at, capacity, avgReqCost), capacity)
	if update {
		n.addCallback(balanceCallbackUpdate, pri, n.signalPriorityUpdate)
	}
	return pri
}

// PosBalanceMissing calculates the missing amount of positive balance in order to
// connect at targetCapacity, stay connected for the given amount of time and then
// still have a priority of targetPriority
func (n *NodeBalance) PosBalanceMissing(targetPriority int64, targetCapacity uint64, after time.Duration) uint64 {
	n.lock.Lock()
	defer n.lock.Unlock()

	now := n.bt.clock.Now()
	if targetPriority < 0 {
		timePrice := n.negFactor.timePrice(targetCapacity)
		timeCost := uint64(float64(after) * timePrice)
		negBalance := n.balance.neg.Value(n.bt.negExp.LogOffset(now))
		if timeCost+negBalance < uint64(-targetPriority) {
			return 0
		}
		if uint64(-targetPriority) > negBalance && timePrice > 1e-100 {
			if negTime := time.Duration(float64(uint64(-targetPriority)-negBalance) / timePrice); negTime < after {
				after -= negTime
			} else {
				after = 0
			}
		}
		targetPriority = 0
	}
	timePrice := n.posFactor.timePrice(targetCapacity)
	posRequired := uint64(float64(targetPriority)*float64(targetCapacity)+float64(after)*timePrice) + 1
	if posRequired >= maxBalance {
		return math.MaxUint64 // target not reachable
	}
	posBalance := n.balance.pos.Value(n.bt.posExp.LogOffset(now))
	if posRequired > posBalance {
		return posRequired - posBalance
	}
	return 0
}

// SetPriceFactors sets the price factors. TimeFactor is the price of a nanosecond of
// connection while RequestFactor is the price of a request cost unit.
func (n *NodeBalance) SetPriceFactors(posFactor, negFactor PriceFactors) {
	n.lock.Lock()
	now := n.bt.clock.Now()
	n.updateBalance(now)
	n.posFactor, n.negFactor = posFactor, negFactor
	callbacks := n.checkCallbacks(now)
	n.lock.Unlock()
	if callbacks != nil {
		n.bt.ns.Operation(func() {
			for _, cb := range callbacks {
				cb()
			}
		})
	}
}

// GetPriceFactors returns the price factors
func (n *NodeBalance) GetPriceFactors() (posFactor, negFactor PriceFactors) {
	n.lock.Lock()
	defer n.lock.Unlock()

	return n.posFactor, n.negFactor
}

// activate starts time/capacity cost deduction.
func (n *NodeBalance) activate() {
	n.bt.updateTotalBalance(n, func() bool {
		if n.active {
			return false
		}
		n.active = true
		n.lastUpdate = n.bt.clock.Now()
		return true
	})
}

// deactivate stops time/capacity cost deduction and saves the balances in the database
func (n *NodeBalance) deactivate() {
	n.bt.updateTotalBalance(n, func() bool {
		if !n.active {
			return false
		}
		n.updateBalance(n.bt.clock.Now())
		if n.updateEvent != nil {
			n.updateEvent.Stop()
			n.updateEvent = nil
		}
		n.storeBalance(true, true)
		n.active = false
		return true
	})
}

// updateBalance updates balance based on the time factor
func (n *NodeBalance) updateBalance(now mclock.AbsTime) {
	if n.active && now > n.lastUpdate {
		n.balance = n.reducedBalance(now, n.capacity, 0)
		n.lastUpdate = now
	}
}

// storeBalance stores the positive and/or negative balance of the node in the database
func (n *NodeBalance) storeBalance(pos, neg bool) {
	if pos {
		n.bt.storeBalance(n.node.ID().Bytes(), false, n.balance.pos)
	}
	if neg {
		n.bt.storeBalance([]byte(n.connAddress), true, n.balance.neg)
	}
}

// addCallback sets up a one-time callback to be called when priority reaches
// the threshold. If it has already reached the threshold the callback is called
// immediately.
// Note: should be called while n.lock is held
// Note 2: the callback function runs inside a NodeStateMachine operation
func (n *NodeBalance) addCallback(id int, threshold int64, callback func()) {
	n.removeCallback(id)
	idx := 0
	for idx < n.callbackCount && threshold > n.callbacks[idx].threshold {
		idx++
	}
	for i := n.callbackCount - 1; i >= idx; i-- {
		n.callbackIndex[n.callbacks[i].id]++
		n.callbacks[i+1] = n.callbacks[i]
	}
	n.callbackCount++
	n.callbackIndex[id] = idx
	n.callbacks[idx] = balanceCallback{id, threshold, callback}
	now := n.bt.clock.Now()
	n.updateBalance(now)
	n.scheduleCheck(now)
}

// removeCallback removes the given callback and returns true if it was active
// Note: should be called while n.lock is held
func (n *NodeBalance) removeCallback(id int) bool {
	idx := n.callbackIndex[id]
	if idx == -1 {
		return false
	}
	n.callbackIndex[id] = -1
	for i := idx; i < n.callbackCount-1; i++ {
		n.callbackIndex[n.callbacks[i+1].id]--
		n.callbacks[i] = n.callbacks[i+1]
	}
	n.callbackCount--
	return true
}

// checkCallbacks checks whether the threshold of any of the active callbacks
// have been reached and returns triggered callbacks.
// Note: checkCallbacks assumes that the balance has been recently updated.
func (n *NodeBalance) checkCallbacks(now mclock.AbsTime) (callbacks []func()) {
	if n.callbackCount == 0 || n.capacity == 0 {
		return
	}
	pri := n.balanceToPriority(n.balance, n.capacity)
	for n.callbackCount != 0 && n.callbacks[n.callbackCount-1].threshold >= pri {
		n.callbackCount--
		n.callbackIndex[n.callbacks[n.callbackCount].id] = -1
		callbacks = append(callbacks, n.callbacks[n.callbackCount].callback)
	}
	n.scheduleCheck(now)
	return
}

// scheduleCheck sets up or updates a scheduled event to ensure that it will be called
// again just after the next threshold has been reached.
func (n *NodeBalance) scheduleCheck(now mclock.AbsTime) {
	if n.callbackCount != 0 {
		d, ok := n.timeUntil(n.callbacks[n.callbackCount-1].threshold)
		if !ok {
			n.nextUpdate = 0
			n.updateAfter(0)
			return
		}
		if n.nextUpdate == 0 || n.nextUpdate > now+mclock.AbsTime(d) {
			if d > time.Second {
				// Note: if the scheduled update is not in the very near future then we
				// schedule the update a bit earlier. This way we do need to update a few
				// extra times but don't need to reschedule every time a processed request
				// brings the expected firing time a little bit closer.
				d = ((d - time.Second) * 7 / 8) + time.Second
			}
			n.nextUpdate = now + mclock.AbsTime(d)
			n.updateAfter(d)
		}
	} else {
		n.nextUpdate = 0
		n.updateAfter(0)
	}
}

// updateAfter schedules a balance update and callback check in the future
func (n *NodeBalance) updateAfter(dt time.Duration) {
	if n.updateEvent == nil || n.updateEvent.Stop() {
		if dt == 0 {
			n.updateEvent = nil
		} else {
			n.updateEvent = n.bt.clock.AfterFunc(dt, func() {
				var callbacks []func()
				n.lock.Lock()
				if n.callbackCount != 0 {
					now := n.bt.clock.Now()
					n.updateBalance(now)
					callbacks = n.checkCallbacks(now)
				}
				n.lock.Unlock()
				if callbacks != nil {
					n.bt.ns.Operation(func() {
						for _, cb := range callbacks {
							cb()
						}
					})
				}
			})
		}
	}
}

// balanceExhausted should be called when the positive balance is exhausted (priority goes to zero/negative)
// Note: this function should run inside a NodeStateMachine operation
func (n *NodeBalance) balanceExhausted() {
	n.lock.Lock()
	n.storeBalance(true, false)
	n.priority = false
	n.lock.Unlock()
	n.bt.ns.SetStateSub(n.node, nodestate.Flags{}, n.bt.PriorityFlag, 0)
}

// checkPriorityStatus checks whether the node has gained priority status and sets the priority
// callback and flag if necessary. It assumes that the balance has been recently updated.
// Note that the priority flag has to be set by the caller after the mutex has been released.
func (n *NodeBalance) checkPriorityStatus() bool {
	if !n.priority && !n.balance.pos.IsZero() {
		n.priority = true
		n.addCallback(balanceCallbackZero, 0, func() { n.balanceExhausted() })
		return true
	}
	return false
}

// signalPriorityUpdate signals that the priority fell below the previous minimum estimate
// Note: this function should run inside a NodeStateMachine operation
func (n *NodeBalance) signalPriorityUpdate() {
	n.bt.ns.SetStateSub(n.node, n.bt.UpdateFlag, nodestate.Flags{}, 0)
	n.bt.ns.SetStateSub(n.node, nodestate.Flags{}, n.bt.UpdateFlag, 0)
}

// setCapacity updates the capacity value used for priority calculation
// Note: capacity should never be zero
// Note 2: this function should run inside a NodeStateMachine operation
func (n *NodeBalance) setCapacity(capacity uint64) {
	n.lock.Lock()
	now := n.bt.clock.Now()
	n.updateBalance(now)
	n.capacity = capacity
	callbacks := n.checkCallbacks(now)
	n.lock.Unlock()
	for _, cb := range callbacks {
		cb()
	}
}

// balanceToPriority converts a balance to a priority value. Lower priority means
// first to disconnect. Positive balance translates to positive priority. If positive
// balance is zero then negative balance translates to a negative priority.
func (n *NodeBalance) balanceToPriority(b balance, capacity uint64) int64 {
	if !b.pos.IsZero() {
		return int64(b.pos.Value(n.bt.posExp.LogOffset(n.bt.clock.Now())) / capacity)
	}
	return -int64(b.neg.Value(n.bt.negExp.LogOffset(n.bt.clock.Now())))
}

// reducedBalance estimates the reduced balance at a given time in the fututre based
// on the current balance, the time factor and an estimated average request cost per time ratio
func (n *NodeBalance) reducedBalance(at mclock.AbsTime, capacity uint64, avgReqCost float64) balance {
	dt := float64(at - n.lastUpdate)
	b := n.balance
	if !b.pos.IsZero() {
		factor := n.posFactor.timePrice(capacity) + n.posFactor.RequestFactor*avgReqCost
		diff := -int64(dt * factor)
		dd := b.pos.Add(diff, n.bt.posExp.LogOffset(at))
		if dd == diff {
			dt = 0
		} else {
			dt += float64(dd) / factor
		}
	}
	if dt > 0 {
		factor := n.negFactor.timePrice(capacity) + n.negFactor.RequestFactor*avgReqCost
		b.neg.Add(int64(dt*factor), n.bt.negExp.LogOffset(at))
	}
	return b
}

// timeUntil calculates the remaining time needed to reach a given priority level
// assuming that no requests are processed until then. If the given level is never
// reached then (0, false) is returned.
// Note: the function assumes that the balance has been recently updated and
// calculates the time starting from the last update.
func (n *NodeBalance) timeUntil(priority int64) (time.Duration, bool) {
	now := n.bt.clock.Now()
	var dt float64
	if !n.balance.pos.IsZero() {
		posBalance := n.balance.pos.Value(n.bt.posExp.LogOffset(now))
		timePrice := n.posFactor.timePrice(n.capacity)
		if timePrice < 1e-100 {
			return 0, false
		}
		if priority > 0 {
			newBalance := uint64(priority) * n.capacity
			if newBalance > posBalance {
				return 0, false
			}
			dt = float64(posBalance-newBalance) / timePrice
			return time.Duration(dt), true
		}
		dt = float64(posBalance) / timePrice
	} else {
		if priority > 0 {
			return 0, false
		}
	}
	// if we have a positive balance then dt equals the time needed to get it to zero
	negBalance := n.balance.neg.Value(n.bt.negExp.LogOffset(now))
	timePrice := n.negFactor.timePrice(n.capacity)
	if uint64(-priority) > negBalance {
		if timePrice < 1e-100 {
			return 0, false
		}
		dt += float64(uint64(-priority)-negBalance) / timePrice
	}
	return time.Duration(dt), true
}
