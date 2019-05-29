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

package les

import (
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

const (
	balanceCallbackQueue = iota
	balanceCallbackZero
	balanceCallbackCount
)

// balanceTracker keeps track of the positive and negative balances of a connected
// client and calculates actual and projected future priority values required by
// prque.LazyQueue.
type balanceTracker struct {
	lock                             sync.Mutex
	clock                            mclock.Clock
	stopped                          bool
	capacity                         uint64
	balance                          balance
	timeFactor, requestFactor        float64
	negTimeFactor, negRequestFactor  float64
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
	pos, neg uint64
}

// balanceCallback represents a single callback that is activated when client priority
// reaches the given threshold
type balanceCallback struct {
	id        int
	threshold int64
	callback  func()
}

// init initializes balanceTracker
func (bt *balanceTracker) init(clock mclock.Clock, capacity uint64) {
	bt.clock = clock
	bt.initTime = clock.Now()
	for i := range bt.callbackIndex {
		bt.callbackIndex[i] = -1
	}
	bt.capacity = capacity
}

// stop shuts down the balance tracker
func (bt *balanceTracker) stop(now mclock.AbsTime) {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	bt.stopped = true
	bt.updateBalance(now)
	bt.negTimeFactor = 0
	bt.negRequestFactor = 0
	bt.timeFactor = 0
	bt.requestFactor = 0
	if bt.updateEvent != nil {
		bt.updateEvent.Stop()
		bt.updateEvent = nil
	}
}

// balanceToPriority converts a balance to a priority value. Higher priority means
// first to disconnect. Positive balance translates to negative priority. If positive
// balance is zero then negative balance translates to a positive priority.
func (bt *balanceTracker) balanceToPriority(b balance) int64 {
	if b.pos > 0 {
		return ^int64(b.pos / bt.capacity)
	}
	return int64(b.neg)
}

// reducedBalance estimates the reduced balance at a given time in the fututre based
// on the current balance, the time factor and an estimated average request cost per time ratio
func (bt *balanceTracker) reducedBalance(at mclock.AbsTime, avgReqCost float64) balance {
	dt := float64(at - bt.lastUpdate)
	b := bt.balance
	if b.pos != 0 {
		factor := bt.timeFactor + bt.requestFactor*avgReqCost
		diff := uint64(dt * factor)
		if diff <= b.pos {
			b.pos -= diff
			dt = 0
		} else {
			dt -= float64(b.pos) / factor
			b.pos = 0
		}
	}
	if dt != 0 {
		factor := bt.negTimeFactor + bt.negRequestFactor*avgReqCost
		b.neg += uint64(dt * factor)
	}
	return b
}

// timeUntil calculates the remaining time needed to reach a given priority level
// assuming that no requests are processed until then. If the given level is never
// reached then (0, false) is returned.
// Note: the function assumes that the balance has been recently updated and
// calculates the time starting from the last update.
func (bt *balanceTracker) timeUntil(priority int64) (time.Duration, bool) {
	var dt float64
	if bt.balance.pos != 0 {
		if bt.timeFactor < 1e-100 {
			return 0, false
		}
		if priority < 0 {
			newBalance := uint64(^priority) * bt.capacity
			if newBalance > bt.balance.pos {
				return 0, false
			}
			dt = float64(bt.balance.pos-newBalance) / bt.timeFactor
			return time.Duration(dt), true
		} else {
			dt = float64(bt.balance.pos) / bt.timeFactor
		}
	} else {
		if priority < 0 {
			return 0, false
		}
	}
	// if we have a positive balance then dt equals the time needed to get it to zero
	if uint64(priority) > bt.balance.neg {
		if bt.negTimeFactor < 1e-100 {
			return 0, false
		}
		dt += float64(uint64(priority)-bt.balance.neg) / bt.negTimeFactor
	}
	return time.Duration(dt), true
}

// getPriority returns the actual priority based on the current balance
func (bt *balanceTracker) getPriority(now mclock.AbsTime) int64 {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	bt.updateBalance(now)
	return bt.balanceToPriority(bt.balance)
}

// estimatedPriority gives an upper estimate for the priority at a given time in the future.
// If addReqCost is true then an average request cost per time is assumed that is twice the
// average cost per time in the current session. If false, zero request cost is assumed.
func (bt *balanceTracker) estimatedPriority(at mclock.AbsTime, addReqCost bool) int64 {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	var avgReqCost float64
	if addReqCost {
		dt := time.Duration(bt.lastUpdate - bt.initTime)
		if dt > time.Second {
			avgReqCost = float64(bt.sumReqCost) * 2 / float64(dt)
		}
	}
	return bt.balanceToPriority(bt.reducedBalance(at, avgReqCost))
}

// updateBalance updates balance based on the time factor
func (bt *balanceTracker) updateBalance(now mclock.AbsTime) {
	if now > bt.lastUpdate {
		bt.balance = bt.reducedBalance(now, 0)
		bt.lastUpdate = now
	}
}

// checkCallbacks checks whether the threshold of any of the active callbacks
// have been reached and calls them if necessary. It also sets up or updates
// a scheduled event to ensure that is will be called again just after the next
// threshold has been reached.
// Note: checkCallbacks assumes that the balance has been recently updated.
func (bt *balanceTracker) checkCallbacks(now mclock.AbsTime) {
	if bt.callbackCount == 0 {
		return
	}
	pri := bt.balanceToPriority(bt.balance)
	for bt.callbackCount != 0 && bt.callbacks[bt.callbackCount-1].threshold <= pri {
		bt.callbackCount--
		bt.callbackIndex[bt.callbacks[bt.callbackCount].id] = -1
		go bt.callbacks[bt.callbackCount].callback()
	}
	if bt.callbackCount != 0 {
		d, ok := bt.timeUntil(bt.callbacks[bt.callbackCount-1].threshold)
		if !ok {
			bt.nextUpdate = 0
			bt.updateAfter(0)
			return
		}
		if bt.nextUpdate == 0 || bt.nextUpdate > now+mclock.AbsTime(d) {
			if d > time.Second {
				// Note: if the scheduled update is not in the very near future then we
				// schedule the update a bit earlier. This way we do need to update a few
				// extra times but don't need to reschedule every time a processed request
				// brings the expected firing time a little bit closer.
				d = ((d - time.Second) * 7 / 8) + time.Second
			}
			bt.nextUpdate = now + mclock.AbsTime(d)
			bt.updateAfter(d)
		}
	} else {
		bt.nextUpdate = 0
		bt.updateAfter(0)
	}
}

// updateAfter schedules a balance update and callback check in the future
func (bt *balanceTracker) updateAfter(dt time.Duration) {
	if bt.updateEvent == nil || bt.updateEvent.Stop() {
		if dt == 0 {
			bt.updateEvent = nil
		} else {
			bt.updateEvent = bt.clock.AfterFunc(dt, func() {
				bt.lock.Lock()
				defer bt.lock.Unlock()

				if bt.callbackCount != 0 {
					now := bt.clock.Now()
					bt.updateBalance(now)
					bt.checkCallbacks(now)
				}
			})
		}
	}
}

// requestCost should be called after serving a request for the given peer
func (bt *balanceTracker) requestCost(cost uint64) {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	if bt.stopped {
		return
	}
	now := bt.clock.Now()
	bt.updateBalance(now)
	fcost := float64(cost)

	if bt.balance.pos != 0 {
		if bt.requestFactor != 0 {
			c := uint64(fcost * bt.requestFactor)
			if bt.balance.pos >= c {
				bt.balance.pos -= c
				fcost = 0
			} else {
				fcost *= 1 - float64(bt.balance.pos)/float64(c)
				bt.balance.pos = 0
			}
			bt.checkCallbacks(now)
		} else {
			fcost = 0
		}
	}
	if fcost > 0 {
		if bt.negRequestFactor != 0 {
			bt.balance.neg += uint64(fcost * bt.negRequestFactor)
			bt.checkCallbacks(now)
		}
	}
	bt.sumReqCost += cost
}

// getBalance returns the current positive and negative balance
func (bt *balanceTracker) getBalance(now mclock.AbsTime) (uint64, uint64) {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	bt.updateBalance(now)
	return bt.balance.pos, bt.balance.neg
}

// setBalance sets the positive and negative balance to the given values
func (bt *balanceTracker) setBalance(pos, neg uint64) error {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	now := bt.clock.Now()
	bt.updateBalance(now)
	bt.balance.pos = pos
	bt.balance.neg = neg
	bt.checkCallbacks(now)
	return nil
}

// setFactors sets the price factors. timeFactor is the price of a nanosecond of
// connection while requestFactor is the price of a "realCost" unit.
func (bt *balanceTracker) setFactors(neg bool, timeFactor, requestFactor float64) {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	if bt.stopped {
		return
	}
	now := bt.clock.Now()
	bt.updateBalance(now)
	if neg {
		bt.negTimeFactor = timeFactor
		bt.negRequestFactor = requestFactor
	} else {
		bt.timeFactor = timeFactor
		bt.requestFactor = requestFactor
	}
	bt.checkCallbacks(now)
}

// setCallback sets up a one-time callback to be called when priority reaches
// the threshold. If it has already reached the threshold the callback is called
// immediately.
func (bt *balanceTracker) addCallback(id int, threshold int64, callback func()) {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	bt.removeCb(id)
	idx := 0
	for idx < bt.callbackCount && threshold < bt.callbacks[idx].threshold {
		idx++
	}
	for i := bt.callbackCount - 1; i >= idx; i-- {
		bt.callbackIndex[bt.callbacks[i].id]++
		bt.callbacks[i+1] = bt.callbacks[i]
	}
	bt.callbackCount++
	bt.callbackIndex[id] = idx
	bt.callbacks[idx] = balanceCallback{id, threshold, callback}
	now := bt.clock.Now()
	bt.updateBalance(now)
	bt.checkCallbacks(now)
}

// removeCallback removes the given callback and returns true if it was active
func (bt *balanceTracker) removeCallback(id int) bool {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	return bt.removeCb(id)
}

// removeCb removes the given callback and returns true if it was active
// Note: should be called while bt.lock is held
func (bt *balanceTracker) removeCb(id int) bool {
	idx := bt.callbackIndex[id]
	if idx == -1 {
		return false
	}
	bt.callbackIndex[id] = -1
	for i := idx; i < bt.callbackCount-1; i++ {
		bt.callbackIndex[bt.callbacks[i+1].id]--
		bt.callbacks[i] = bt.callbacks[i+1]
	}
	bt.callbackCount--
	return true
}
