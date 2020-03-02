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
	"math"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
)

const maxBalance = math.MaxInt64

const (
	balanceCallbackQueue = iota
	balanceCallbackZero
	balanceCallbackCount
)

// expirationController controls the exponential expiration of positive and negative
// balances
type expirationController interface {
	posExpiration(mclock.AbsTime) fixed64
	negExpiration(mclock.AbsTime) fixed64
}

// priceFactors determine the pricing policy (may apply either to positive or
// negative balances which may have different factors).
// - timeFactor is cost unit per nanosecond of connection time
// - capacityFactor is cost unit per nanosecond of connection time per 1000000 capacity
// - requestFactor is cost unit per request "realCost" unit
type priceFactors struct {
	timeFactor, capacityFactor, requestFactor float64
}

func (p priceFactors) timePrice(cap uint64) float64 {
	return p.timeFactor + float64(cap)*p.capacityFactor/1000000
}

func (p priceFactors) reqPrice() float64 {
	return p.requestFactor
}

// balanceTracker keeps track of the positive and negative balances of a connected
// client and calculates actual and projected future priority values required by
// prque.LazyQueue.
type balanceTracker struct {
	lock                             sync.Mutex
	clock                            mclock.Clock
	exp                              expirationController
	stopped                          bool
	capacity                         uint64
	balance                          balance
	posFactor, negFactor             priceFactors
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
	pos, neg expiredValue
}

// balanceCallback represents a single callback that is activated when client priority
// reaches the given threshold
type balanceCallback struct {
	id        int
	threshold int64
	callback  func()
}

// init initializes balanceTracker
// Note: capacity should never be zero
func (bt *balanceTracker) init(clock mclock.Clock, capacity uint64) {
	bt.clock = clock
	bt.initTime, bt.lastUpdate = clock.Now(), clock.Now() // Init timestamps
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
	bt.addBalance(now)
	bt.posFactor = priceFactors{0, 0, 0}
	bt.negFactor = priceFactors{0, 0, 0}
	if bt.updateEvent != nil {
		bt.updateEvent.Stop()
		bt.updateEvent = nil
	}
}

// balanceToPriority converts a balance to a priority value. Higher priority means
// first to disconnect. Positive balance translates to negative priority. If positive
// balance is zero then negative balance translates to a positive priority.
func (bt *balanceTracker) balanceToPriority(b balance) int64 {
	if b.pos.base > 0 {
		return -int64(b.pos.value(bt.exp.posExpiration(bt.clock.Now())) / bt.capacity)
	}
	return int64(b.neg.value(bt.exp.negExpiration(bt.clock.Now())))
}

// posBalanceMissing calculates the missing amount of positive balance in order to
// connect at targetCapacity, stay connected for the given amount of time and then
// still have a priority of targetPriority
func (bt *balanceTracker) posBalanceMissing(targetPriority int64, targetCapacity uint64, after time.Duration) uint64 {
	now := bt.clock.Now()
	if targetPriority > 0 {
		timePrice := bt.negFactor.timePrice(targetCapacity)
		timeCost := uint64(float64(after) * timePrice)
		negBalance := bt.balance.neg.value(bt.exp.negExpiration(now))
		if timeCost+negBalance < uint64(targetPriority) {
			return 0
		}
		if uint64(targetPriority) > negBalance && timePrice > 1e-100 {
			if negTime := time.Duration(float64(uint64(targetPriority)-negBalance) / timePrice); negTime < after {
				after -= negTime
			} else {
				after = 0
			}
		}
		targetPriority = 0
	}
	timePrice := bt.posFactor.timePrice(targetCapacity)
	posRequired := uint64(float64(-targetPriority)*float64(targetCapacity)+float64(after)*timePrice) + 1
	if posRequired >= maxBalance {
		return math.MaxUint64 // target not reachable
	}
	posBalance := bt.balance.pos.value(bt.exp.posExpiration(now))
	if posRequired > posBalance {
		return posRequired - posBalance
	}
	return 0
}

// reducedBalance estimates the reduced balance at a given time in the fututre based
// on the current balance, the time factor and an estimated average request cost per time ratio
func (bt *balanceTracker) reducedBalance(at mclock.AbsTime, avgReqCost float64) balance {
	dt := float64(at - bt.lastUpdate)
	b := bt.balance
	if b.pos.base != 0 {
		factor := bt.posFactor.timePrice(bt.capacity) + bt.posFactor.reqPrice()*avgReqCost
		diff := -int64(dt * factor)
		dd := b.pos.add(diff, bt.exp.posExpiration(at))
		if dd == diff {
			dt = 0
		} else {
			dt += float64(dd) / factor
		}
	}
	if dt > 0 {
		factor := bt.negFactor.timePrice(bt.capacity) + bt.negFactor.reqPrice()*avgReqCost
		b.neg.add(int64(dt*factor), bt.exp.negExpiration(at))
	}
	return b
}

// timeUntil calculates the remaining time needed to reach a given priority level
// assuming that no requests are processed until then. If the given level is never
// reached then (0, false) is returned.
// Note: the function assumes that the balance has been recently updated and
// calculates the time starting from the last update.
func (bt *balanceTracker) timeUntil(priority int64) (time.Duration, bool) {
	now := bt.clock.Now()
	var dt float64
	if bt.balance.pos.base != 0 {
		posBalance := bt.balance.pos.value(bt.exp.posExpiration(now))
		timePrice := bt.posFactor.timePrice(bt.capacity)
		if timePrice < 1e-100 {
			return 0, false
		}
		if priority < 0 {
			newBalance := uint64(-priority) * bt.capacity
			if newBalance > posBalance {
				return 0, false
			}
			dt = float64(posBalance-newBalance) / timePrice
			return time.Duration(dt), true
		} else {
			dt = float64(posBalance) / timePrice
		}
	} else {
		if priority < 0 {
			return 0, false
		}
	}
	// if we have a positive balance then dt equals the time needed to get it to zero
	negBalance := bt.balance.neg.value(bt.exp.negExpiration(now))
	timePrice := bt.negFactor.timePrice(bt.capacity)
	if uint64(priority) > negBalance {
		if timePrice < 1e-100 {
			return 0, false
		}
		dt += float64(uint64(priority)-negBalance) / timePrice
	}
	return time.Duration(dt), true
}

// setCapacity updates the capacity value used for priority calculation
// Note: capacity should never be zero
func (bt *balanceTracker) setCapacity(capacity uint64) {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	bt.capacity = capacity
}

// getPriority returns the actual priority based on the current balance
func (bt *balanceTracker) getPriority(now mclock.AbsTime) int64 {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	bt.addBalance(now)
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

// addBalance updates balance based on the time factor
func (bt *balanceTracker) addBalance(now mclock.AbsTime) {
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
					bt.addBalance(now)
					bt.checkCallbacks(now)
				}
			})
		}
	}
}

// requestCost should be called after serving a request for the given peer
func (bt *balanceTracker) requestCost(cost uint64) uint64 {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	if bt.stopped {
		return 0
	}
	now := bt.clock.Now()
	bt.addBalance(now)
	fcost := float64(cost)

	posExp := bt.exp.posExpiration(now)
	if bt.balance.pos.base != 0 {
		if bt.posFactor.reqPrice() != 0 {
			c := -int64(fcost * bt.posFactor.reqPrice())
			cc := bt.balance.pos.add(c, posExp)
			if c == cc {
				fcost = 0
			} else {
				fcost *= 1 - float64(cc)/float64(c)
			}
			bt.checkCallbacks(now)
		} else {
			fcost = 0
		}
	}
	if fcost > 0 {
		if bt.negFactor.reqPrice() != 0 {
			bt.balance.neg.add(int64(fcost*bt.negFactor.reqPrice()), bt.exp.negExpiration(now))
			bt.checkCallbacks(now)
		}
	}
	bt.sumReqCost += cost
	return bt.balance.pos.value(posExp)
}

// getBalance returns the current positive and negative balance
func (bt *balanceTracker) getBalance(now mclock.AbsTime) (expiredValue, expiredValue) {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	bt.addBalance(now)
	return bt.balance.pos, bt.balance.neg
}

// setBalance sets the positive and negative balance to the given values
func (bt *balanceTracker) setBalance(pos, neg expiredValue) error {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	now := bt.clock.Now()
	bt.addBalance(now)
	bt.balance.pos = pos
	bt.balance.neg = neg
	bt.checkCallbacks(now)
	return nil
}

// setFactors sets the price factors. timeFactor is the price of a nanosecond of
// connection while requestFactor is the price of a "realCost" unit.
func (bt *balanceTracker) setFactors(posFactor, negFactor priceFactors) {
	bt.lock.Lock()
	defer bt.lock.Unlock()

	if bt.stopped {
		return
	}
	now := bt.clock.Now()
	bt.addBalance(now)
	bt.posFactor, bt.negFactor = posFactor, negFactor
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
	bt.addBalance(now)
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
