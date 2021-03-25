// Copyright 2020 The go-ethereum Authors
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
	"math"
	"reflect"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/common/prque"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/p2p/nodestate"
)

const (
	lazyQueueRefresh = time.Second * 10 // refresh period of the active queue
)

// priorityPoolSetup contains node state flags and fields used by priorityPool
// Note: activeFlag and inactiveFlag can be controlled both externally and by the pool,
// see priorityPool description for details.
type priorityPoolSetup struct {
	// controlled by priorityPool
	activeFlag, inactiveFlag       nodestate.Flags
	capacityField, ppNodeInfoField nodestate.Field
	// external connections
	updateFlag    nodestate.Flags
	priorityField nodestate.Field
}

// newPriorityPoolSetup creates a new priorityPoolSetup and initializes the fields
// and flags controlled by priorityPool
func newPriorityPoolSetup(setup *nodestate.Setup) priorityPoolSetup {
	return priorityPoolSetup{
		activeFlag:      setup.NewFlag("active"),
		inactiveFlag:    setup.NewFlag("inactive"),
		capacityField:   setup.NewField("capacity", reflect.TypeOf(uint64(0))),
		ppNodeInfoField: setup.NewField("ppNodeInfo", reflect.TypeOf(&ppNodeInfo{})),
	}
}

// connect sets the fields and flags used by priorityPool as an input
func (pps *priorityPoolSetup) connect(priorityField nodestate.Field, updateFlag nodestate.Flags) {
	pps.priorityField = priorityField // should implement nodePriority
	pps.updateFlag = updateFlag       // triggers an immediate priority update
}

// priorityPool handles a set of nodes where each node has a capacity (a scalar value)
// and a priority (which can change over time and can also depend on the capacity).
// A node is active if it has at least the necessary minimal amount of capacity while
// inactive nodes have 0 capacity (values between 0 and the minimum are not allowed).
// The pool ensures that the number and total capacity of all active nodes are limited
// and the highest priority nodes are active at all times (limits can be changed
// during operation with immediate effect).
//
// When activating clients a priority bias is applied in favor of the already active
// nodes in order to avoid nodes quickly alternating between active and inactive states
// when their priorities are close to each other. The bias is specified in terms of
// duration (time) because priorities are expected to usually get lower over time and
// therefore a future minimum prediction (see EstMinPriority) should monotonously
// decrease with the specified time parameter.
// This time bias can be interpreted as minimum expected active time at the given
// capacity (if the threshold priority stays the same).
//
// Nodes in the pool always have either inactiveFlag or activeFlag set. A new node is
// added to the pool by externally setting inactiveFlag. priorityPool can switch a node
// between inactiveFlag and activeFlag at any time. Nodes can be removed from the pool
// by externally resetting both flags. activeFlag should not be set externally.
//
// The highest priority nodes in "inactive" state are moved to "active" state as soon as
// the minimum capacity can be granted for them. The capacity of lower priority active
// nodes is reduced or they are demoted to "inactive" state if their priority is
// insufficient even at minimal capacity.
type priorityPool struct {
	priorityPoolSetup
	ns                     *nodestate.NodeStateMachine
	clock                  mclock.Clock
	lock                   sync.Mutex
	activeQueue            *prque.LazyQueue
	inactiveQueue          *prque.Prque
	changed                []*ppNodeInfo
	activeCount, activeCap uint64
	maxCount, maxCap       uint64
	minCap                 uint64
	activeBias             time.Duration
	capacityStepDiv        uint64

	cachedCurve    *capacityCurve
	ccUpdatedAt    mclock.AbsTime
	ccUpdateForced bool
}

// ppNodeInfo is the internal node descriptor of priorityPool
type ppNodeInfo struct {
	nodePriority               nodePriority
	node                       *enode.Node
	connected                  bool
	capacity, origCap          uint64
	bias                       time.Duration
	forced, changed            bool
	activeIndex, inactiveIndex int
}

// newPriorityPool creates a new priorityPool
func newPriorityPool(ns *nodestate.NodeStateMachine, setup priorityPoolSetup, clock mclock.Clock, minCap uint64, activeBias time.Duration, capacityStepDiv uint64) *priorityPool {
	pp := &priorityPool{
		ns:                ns,
		priorityPoolSetup: setup,
		clock:             clock,
		inactiveQueue:     prque.New(inactiveSetIndex),
		minCap:            minCap,
		activeBias:        activeBias,
		capacityStepDiv:   capacityStepDiv,
	}
	if pp.activeBias < time.Duration(1) {
		pp.activeBias = time.Duration(1)
	}
	pp.activeQueue = prque.NewLazyQueue(activeSetIndex, activePriority, pp.activeMaxPriority, clock, lazyQueueRefresh)

	ns.SubscribeField(pp.priorityField, func(node *enode.Node, state nodestate.Flags, oldValue, newValue interface{}) {
		if newValue != nil {
			c := &ppNodeInfo{
				node:          node,
				nodePriority:  newValue.(nodePriority),
				activeIndex:   -1,
				inactiveIndex: -1,
			}
			ns.SetFieldSub(node, pp.ppNodeInfoField, c)
		} else {
			ns.SetStateSub(node, nodestate.Flags{}, pp.activeFlag.Or(pp.inactiveFlag), 0)
			if n, _ := pp.ns.GetField(node, pp.ppNodeInfoField).(*ppNodeInfo); n != nil {
				pp.disconnectedNode(n)
			}
			ns.SetFieldSub(node, pp.capacityField, nil)
			ns.SetFieldSub(node, pp.ppNodeInfoField, nil)
		}
	})
	ns.SubscribeState(pp.activeFlag.Or(pp.inactiveFlag), func(node *enode.Node, oldState, newState nodestate.Flags) {
		if c, _ := pp.ns.GetField(node, pp.ppNodeInfoField).(*ppNodeInfo); c != nil {
			if oldState.IsEmpty() {
				pp.connectedNode(c)
			}
			if newState.IsEmpty() {
				pp.disconnectedNode(c)
			}
		}
	})
	ns.SubscribeState(pp.updateFlag, func(node *enode.Node, oldState, newState nodestate.Flags) {
		if !newState.IsEmpty() {
			pp.updatePriority(node)
		}
	})
	return pp
}

// requestCapacity checks whether changing the capacity of a node to the given target
// is possible (bias is applied in favor of other active nodes if the target is higher
// than the current capacity).
// If setCap is true then it also performs the change if possible. The function returns
// the minimum priority needed to do the change and whether it is currently allowed.
// If setCap and allowed are both true then the caller can assume that the change was
// successful.
// Note: priorityField should always be set before calling requestCapacity. If setCap
// is false then both inactiveFlag and activeFlag can be unset and they are not changed
// by this function call either.
// Note 2: this function should run inside a NodeStateMachine operation
func (pp *priorityPool) requestCapacity(node *enode.Node, targetCap uint64, bias time.Duration, setCap bool) (minPriority int64, allowed bool) {
	pp.lock.Lock()
	pp.activeQueue.Refresh()
	var updates []capUpdate
	defer func() {
		pp.lock.Unlock()
		pp.updateFlags(updates)
	}()

	if targetCap < pp.minCap {
		targetCap = pp.minCap
	}
	if bias < pp.activeBias {
		bias = pp.activeBias
	}
	c, _ := pp.ns.GetField(node, pp.ppNodeInfoField).(*ppNodeInfo)
	if c == nil {
		log.Error("requestCapacity called for unknown node", "id", node.ID())
		return math.MaxInt64, false
	}
	var priority int64
	if targetCap > c.capacity {
		priority = c.nodePriority.estimatePriority(targetCap, 0, 0, bias, false)
	} else {
		priority = c.nodePriority.priority(targetCap)
	}
	pp.markForChange(c)
	pp.setCapacity(c, targetCap)
	c.forced = true
	pp.activeQueue.Remove(c.activeIndex)
	pp.inactiveQueue.Remove(c.inactiveIndex)
	pp.activeQueue.Push(c)
	_, minPriority = pp.enforceLimits()
	// if capacity update is possible now then minPriority == math.MinInt64
	// if it is not possible at all then minPriority == math.MaxInt64
	allowed = priority >= minPriority
	updates = pp.finalizeChanges(setCap && allowed)
	return
}

// SetLimits sets the maximum number and total capacity of simultaneously active nodes
func (pp *priorityPool) SetLimits(maxCount, maxCap uint64) {
	pp.lock.Lock()
	pp.activeQueue.Refresh()
	var updates []capUpdate
	defer func() {
		pp.lock.Unlock()
		pp.ns.Operation(func() { pp.updateFlags(updates) })
	}()

	inc := (maxCount > pp.maxCount) || (maxCap > pp.maxCap)
	dec := (maxCount < pp.maxCount) || (maxCap < pp.maxCap)
	pp.maxCount, pp.maxCap = maxCount, maxCap
	if dec {
		pp.enforceLimits()
		updates = pp.finalizeChanges(true)
	}
	if inc {
		updates = append(updates, pp.tryActivate()...)
	}
}

// setActiveBias sets the bias applied when trying to activate inactive nodes
func (pp *priorityPool) setActiveBias(bias time.Duration) {
	pp.lock.Lock()
	var updates []capUpdate
	defer func() {
		pp.lock.Unlock()
		pp.ns.Operation(func() { pp.updateFlags(updates) })
	}()

	pp.activeBias = bias
	if pp.activeBias < time.Duration(1) {
		pp.activeBias = time.Duration(1)
	}
	updates = pp.tryActivate()
}

// Active returns the number and total capacity of currently active nodes
func (pp *priorityPool) Active() (uint64, uint64) {
	pp.lock.Lock()
	defer pp.lock.Unlock()

	return pp.activeCount, pp.activeCap
}

// Limits returns the maximum allowed number and total capacity of active nodes
func (pp *priorityPool) Limits() (uint64, uint64) {
	pp.lock.Lock()
	defer pp.lock.Unlock()

	return pp.maxCount, pp.maxCap
}

// inactiveSetIndex callback updates ppNodeInfo item index in inactiveQueue
func inactiveSetIndex(a interface{}, index int) {
	a.(*ppNodeInfo).inactiveIndex = index
}

// activeSetIndex callback updates ppNodeInfo item index in activeQueue
func activeSetIndex(a interface{}, index int) {
	a.(*ppNodeInfo).activeIndex = index
}

// invertPriority inverts a priority value. The active queue uses inverted priorities
// because the node on the top is the first to be deactivated.
func invertPriority(p int64) int64 {
	if p == math.MinInt64 {
		return math.MaxInt64
	}
	return -p
}

// activePriority callback returns actual priority of ppNodeInfo item in activeQueue
func activePriority(a interface{}) int64 {
	c := a.(*ppNodeInfo)
	if c.forced {
		return math.MinInt64
	}
	if c.bias == 0 {
		return invertPriority(c.nodePriority.priority(c.capacity))
	} else {
		return invertPriority(c.nodePriority.estimatePriority(c.capacity, 0, 0, c.bias, true))
	}
}

// activeMaxPriority callback returns estimated maximum priority of ppNodeInfo item in activeQueue
func (pp *priorityPool) activeMaxPriority(a interface{}, until mclock.AbsTime) int64 {
	c := a.(*ppNodeInfo)
	if c.forced {
		return math.MinInt64
	}
	future := time.Duration(until - pp.clock.Now())
	if future < 0 {
		future = 0
	}
	return invertPriority(c.nodePriority.estimatePriority(c.capacity, 0, future, c.bias, false))
}

// inactivePriority callback returns actual priority of ppNodeInfo item in inactiveQueue
func (pp *priorityPool) inactivePriority(p *ppNodeInfo) int64 {
	return p.nodePriority.priority(pp.minCap)
}

// connectedNode is called when a new node has been added to the pool (inactiveFlag set)
// Note: this function should run inside a NodeStateMachine operation
func (pp *priorityPool) connectedNode(c *ppNodeInfo) {
	pp.lock.Lock()
	pp.activeQueue.Refresh()
	var updates []capUpdate
	defer func() {
		pp.lock.Unlock()
		pp.updateFlags(updates)
	}()

	if c.connected {
		return
	}
	c.connected = true
	pp.inactiveQueue.Push(c, pp.inactivePriority(c))
	updates = pp.tryActivate()
}

// disconnectedNode is called when a node has been removed from the pool (both inactiveFlag
// and activeFlag reset)
// Note: this function should run inside a NodeStateMachine operation
func (pp *priorityPool) disconnectedNode(c *ppNodeInfo) {
	pp.lock.Lock()
	pp.activeQueue.Refresh()
	var updates []capUpdate
	defer func() {
		pp.lock.Unlock()
		pp.updateFlags(updates)
	}()

	if !c.connected {
		return
	}
	c.connected = false
	pp.activeQueue.Remove(c.activeIndex)
	pp.inactiveQueue.Remove(c.inactiveIndex)
	if c.capacity != 0 {
		pp.setCapacity(c, 0)
		updates = pp.tryActivate()
	}
}

// markForChange internally puts a node in a temporary state that can either be reverted
// or confirmed later. This temporary state allows changing the capacity of a node and
// moving it between the active and inactive queue. activeFlag/inactiveFlag and
// capacityField are not changed while the changes are still temporary.
func (pp *priorityPool) markForChange(c *ppNodeInfo) {
	if c.changed {
		return
	}
	c.changed = true
	c.origCap = c.capacity
	pp.changed = append(pp.changed, c)
}

// setCapacity changes the capacity of a node and adjusts activeCap and activeCount
// accordingly. Note that this change is performed in the temporary state so it should
// be called after markForChange and before finalizeChanges.
func (pp *priorityPool) setCapacity(n *ppNodeInfo, cap uint64) {
	pp.activeCap += cap - n.capacity
	if n.capacity == 0 {
		pp.activeCount++
	}
	if cap == 0 {
		pp.activeCount--
	}
	n.capacity = cap
}

// enforceLimits enforces active node count and total capacity limits. It returns the
// lowest active node priority. Note that this function is performed on the temporary
// internal state.
func (pp *priorityPool) enforceLimits() (*ppNodeInfo, int64) {
	if pp.activeCap <= pp.maxCap && pp.activeCount <= pp.maxCount {
		return nil, math.MinInt64
	}
	var (
		c                 *ppNodeInfo
		maxActivePriority int64
	)
	pp.activeQueue.MultiPop(func(data interface{}, priority int64) bool {
		c = data.(*ppNodeInfo)
		pp.markForChange(c)
		maxActivePriority = priority
		if c.capacity == pp.minCap || pp.activeCount > pp.maxCount {
			pp.setCapacity(c, 0)
		} else {
			sub := c.capacity / pp.capacityStepDiv
			if c.capacity-sub < pp.minCap {
				sub = c.capacity - pp.minCap
			}
			pp.setCapacity(c, c.capacity-sub)
			pp.activeQueue.Push(c)
		}
		return pp.activeCap > pp.maxCap || pp.activeCount > pp.maxCount
	})
	return c, invertPriority(maxActivePriority)
}

// finalizeChanges either commits or reverts temporary changes. The necessary capacity
// field and according flag updates are not performed here but returned in a list because
// they should be performed while the mutex is not held.
func (pp *priorityPool) finalizeChanges(commit bool) (updates []capUpdate) {
	for _, c := range pp.changed {
		// always remove and push back in order to update biased/forced priority
		pp.activeQueue.Remove(c.activeIndex)
		pp.inactiveQueue.Remove(c.inactiveIndex)
		c.bias = 0
		c.forced = false
		c.changed = false
		if !commit {
			pp.setCapacity(c, c.origCap)
		}
		if c.connected {
			if c.capacity != 0 {
				pp.activeQueue.Push(c)
			} else {
				pp.inactiveQueue.Push(c, pp.inactivePriority(c))
			}
			if c.capacity != c.origCap && commit {
				updates = append(updates, capUpdate{c.node, c.origCap, c.capacity})
			}
		}
		c.origCap = 0
	}
	pp.changed = nil
	if commit {
		pp.ccUpdateForced = true
	}
	return
}

// capUpdate describes a capacityField and activeFlag/inactiveFlag update
type capUpdate struct {
	node           *enode.Node
	oldCap, newCap uint64
}

// updateFlags performs capacityField and activeFlag/inactiveFlag updates while the
// pool mutex is not held
// Note: this function should run inside a NodeStateMachine operation
func (pp *priorityPool) updateFlags(updates []capUpdate) {
	for _, f := range updates {
		if f.oldCap == 0 {
			pp.ns.SetStateSub(f.node, pp.activeFlag, pp.inactiveFlag, 0)
		}
		if f.newCap == 0 {
			pp.ns.SetStateSub(f.node, pp.inactiveFlag, pp.activeFlag, 0)
			pp.ns.SetFieldSub(f.node, pp.capacityField, nil)
		} else {
			pp.ns.SetFieldSub(f.node, pp.capacityField, f.newCap)
		}
	}
}

// tryActivate tries to activate inactive nodes if possible
func (pp *priorityPool) tryActivate() []capUpdate {
	var commit bool
	for pp.inactiveQueue.Size() > 0 {
		c := pp.inactiveQueue.PopItem().(*ppNodeInfo)
		pp.markForChange(c)
		pp.setCapacity(c, pp.minCap)
		c.bias = pp.activeBias
		pp.activeQueue.Push(c)
		pp.enforceLimits()
		if c.capacity > 0 {
			commit = true
			c.bias = 0
		} else {
			break
		}
	}
	pp.ccUpdateForced = true
	return pp.finalizeChanges(commit)
}

// updatePriority gets the current priority value of the given node from the nodePriority
// interface and performs the necessary changes. It is triggered by updateFlag.
// Note: this function should run inside a NodeStateMachine operation
func (pp *priorityPool) updatePriority(node *enode.Node) {
	pp.lock.Lock()
	pp.activeQueue.Refresh()
	var updates []capUpdate
	defer func() {
		pp.lock.Unlock()
		pp.updateFlags(updates)
	}()

	c, _ := pp.ns.GetField(node, pp.ppNodeInfoField).(*ppNodeInfo)
	if c == nil || !c.connected {
		return
	}
	pp.activeQueue.Remove(c.activeIndex)
	pp.inactiveQueue.Remove(c.inactiveIndex)
	if c.capacity != 0 {
		pp.activeQueue.Push(c)
	} else {
		pp.inactiveQueue.Push(c, pp.inactivePriority(c))
	}
	updates = pp.tryActivate()
}

// capacityCurve is a snapshot of the priority pool contents in a format that can efficiently
// estimate how much capacity could be granted to a given node at a given priority level.
type capacityCurve struct {
	points       []curvePoint       // curve points sorted in descending order of priority
	index        map[enode.ID][]int // curve point indexes belonging to each node
	excludeList  []int              // curve point indexes of excluded node
	excludeFirst bool               // true if activeCount == maxCount
}

type curvePoint struct {
	freeCap uint64 // available capacity and node count at the current priority level
	nextPri int64  // next priority level where more capacity will be available
}

// getCapacityCurve returns a new or recently cached capacityCurve based on the contents of the pool
func (pp *priorityPool) getCapacityCurve() *capacityCurve {
	pp.lock.Lock()
	defer pp.lock.Unlock()

	now := pp.clock.Now()
	dt := time.Duration(now - pp.ccUpdatedAt)
	if !pp.ccUpdateForced && pp.cachedCurve != nil && dt < time.Second*10 {
		return pp.cachedCurve
	}

	pp.ccUpdateForced = false
	pp.ccUpdatedAt = now
	curve := &capacityCurve{
		index: make(map[enode.ID][]int),
	}
	pp.cachedCurve = curve

	var excludeID enode.ID
	excludeFirst := pp.maxCount == pp.activeCount
	// reduce node capacities or remove nodes until nothing is left in the queue;
	// record the available capacity and the necessary priority after each step
	for pp.activeCap > 0 {
		cp := curvePoint{}
		if pp.activeCap > pp.maxCap {
			log.Error("Active capacity is greater than allowed maximum", "active", pp.activeCap, "maximum", pp.maxCap)
		} else {
			cp.freeCap = pp.maxCap - pp.activeCap
		}
		// temporarily increase activeCap to enforce reducing or removing a node capacity
		tempCap := cp.freeCap + 1
		pp.activeCap += tempCap
		var next *ppNodeInfo
		// enforceLimits removes the lowest priority node if it has minimal capacity,
		// otherwise reduces its capacity
		next, cp.nextPri = pp.enforceLimits()
		pp.activeCap -= tempCap
		if next == nil {
			log.Error("getCapacityCurve: cannot remove next element from the priority queue")
			break
		}
		id := next.node.ID()
		if excludeFirst {
			// if the node count limit is already reached then mark the node with the
			// lowest priority for exclusion
			curve.excludeFirst = true
			excludeID = id
			excludeFirst = false
		}
		// multiple curve points and therefore multiple indexes may belong to a node
		// if it was removed in multiple steps (if its capacity was more than the minimum)
		curve.index[id] = append(curve.index[id], len(curve.points))
		curve.points = append(curve.points, cp)
	}
	// restore original state of the queue
	pp.finalizeChanges(false)
	curve.points = append(curve.points, curvePoint{
		freeCap: pp.maxCap,
		nextPri: math.MaxInt64,
	})
	if curve.excludeFirst {
		curve.excludeList = curve.index[excludeID]
	}
	return curve
}

// exclude returns a capacityCurve with the given node excluded from the original curve
func (cc *capacityCurve) exclude(id enode.ID) *capacityCurve {
	if excludeList, ok := cc.index[id]; ok {
		// return a new version of the curve (only one excluded node can be selected)
		// Note: if the first node was excluded by default (excludeFirst == true) then
		// we can forget about that and exclude the node with the given id instead.
		return &capacityCurve{
			points:      cc.points,
			index:       cc.index,
			excludeList: excludeList,
		}
	}
	return cc
}

func (cc *capacityCurve) getPoint(i int) curvePoint {
	cp := cc.points[i]
	if i == 0 && cc.excludeFirst {
		cp.freeCap = 0
		return cp
	}
	for ii := len(cc.excludeList) - 1; ii >= 0; ii-- {
		ei := cc.excludeList[ii]
		if ei < i {
			break
		}
		e1, e2 := cc.points[ei], cc.points[ei+1]
		cp.freeCap += e2.freeCap - e1.freeCap
	}
	return cp
}

// maxCapacity calculates the maximum capacity available for a node with a given
// (monotonically decreasing) priority vs. capacity function. Note that if the requesting
// node is already in the pool then it should be excluded from the curve in order to get
// the correct result.
func (cc *capacityCurve) maxCapacity(priority func(cap uint64) int64) uint64 {
	min, max := 0, len(cc.points)-1 // the curve always has at least one point
	for min < max {
		mid := (min + max) / 2
		cp := cc.getPoint(mid)
		if cp.freeCap == 0 || priority(cp.freeCap) > cp.nextPri {
			min = mid + 1
		} else {
			max = mid
		}
	}
	cp2 := cc.getPoint(min)
	if cp2.freeCap == 0 || min == 0 {
		return cp2.freeCap
	}
	cp1 := cc.getPoint(min - 1)
	if priority(cp2.freeCap) > cp1.nextPri {
		return cp2.freeCap
	}
	minc, maxc := cp1.freeCap, cp2.freeCap-1
	for minc < maxc {
		midc := (minc + maxc + 1) / 2
		if midc == 0 || priority(midc) > cp1.nextPri {
			minc = midc
		} else {
			maxc = midc - 1
		}
	}
	return maxc
}
