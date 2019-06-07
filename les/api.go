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

package les

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/les/csvlogger"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	errNoCheckpoint         = errors.New("no local checkpoint provided")
	errNotActivated         = errors.New("checkpoint registrar is not activated")
	errInvalidTag           = errors.New("invalid client tag")
	errInvalidParam         = errors.New("invalid client parameter")
	errInvalidValue         = errors.New("invalid parameter value")
	errTotalCap             = errors.New("total capacity exceeded")
	errUnknownBenchmarkType = errors.New("unknown benchmark type")
	errMultiple             = errors.New("multiple errors")
	errClientNotConnected   = errors.New("client is not connected")

	dropCapacityDelay = time.Second // delay applied to decreasing capacity changes
)

// PrivateLightServerAPI provides an API to access the LES light server.
type PrivateLightServerAPI struct {
	server *LesServer
}

// NewPrivateLightServerAPI creates a new LES light server API.
func NewPrivateLightServerAPI(server *LesServer) *PrivateLightServerAPI {
	return &PrivateLightServerAPI{
		server: server,
	}
}

// ServerInfo returns global server parameters
func (api *PrivateLightServerAPI) ServerInfo() map[string]interface{} {
	res := make(map[string]interface{})
	res["minimumCapacity"] = uint64(api.server.minCapacity)
	res["freeClientCapacity"] = api.server.freeClientCap
	res["totalCapacity"], res["totalConnectedCapacity"], res["priorityConnectedCapacity"], res["totalPriorityCapacity"] = api.server.priorityClientPool.capacityInfo()
	return res
}

// ClientInfo returns information about clients listed in the ids list or matching the given tags
func (api *PrivateLightServerAPI) ClientInfo(ids []enode.ID, tags []string) map[enode.ID]map[string]interface{} {
	res := make(map[enode.ID]map[string]interface{})
	api.server.priorityClientPool.matchClients(ids, tags, func(client *priorityClientInfo) {
		res[client.id] = api.server.priorityClientPool.clientInfo(client)
	})
	return res
}

// SetClientParams sets client parameters for all clients listed in the ids list or matching the given tags
func (api *PrivateLightServerAPI) SetClientParams(ids []enode.ID, tags []string, params map[string]interface{}) error {
	var finalErr error
	api.server.priorityClientPool.matchClients(ids, tags, func(client *priorityClientInfo) {
		var (
			err         error
			updatePrice bool
		)
		for name, value := range params {
			switch name {
			case "userTags":
				if t, ok := value.([]interface{}); ok {
					tags := make([]string, len(t))
					for i, tag := range t {
						if tt, ok := tag.(string); ok {
							tags[i] = tt
						} else {
							err = errInvalidTag
						}
					}
					if err == nil {
						err = api.server.priorityClientPool.setClientTags(client, tags)
					}
				} else {
					err = errInvalidTag
				}
			case "capacity":
				if capacity, ok := value.(float64); ok && (capacity == 0 || uint64(capacity) >= api.server.minCapacity) {
					err = api.server.priorityClientPool.setClientCapacity(client, uint64(capacity))
					updatePrice = true
				} else {
					err = errInvalidValue
				}
			case "pricing/timeFactor":
				if val, ok := value.(float64); ok && val >= 0 {
					client.timeFactor = val / 1000000000
					updatePrice = true
				} else {
					err = errInvalidValue
				}
			case "pricing/capacityFactor":
				if val, ok := value.(float64); ok && val >= 0 {
					client.capacityFactor = val / 1000000000
					updatePrice = true
				} else {
					err = errInvalidValue
				}
			case "pricing/requestCostFactor":
				if val, ok := value.(float64); ok && val >= 0 {
					client.requestCostFactor = val / 1000000000
					updatePrice = true
				} else {
					err = errInvalidValue
				}
			case "pricing/alert":
				if val, ok := value.(float64); ok && val >= 0 {
					api.server.priorityClientPool.setPriceUpdate(client, uint64(val), false)
				} else {
					err = errInvalidValue
				}
			case "pricing/periodicUpdate":
				if val, ok := value.(float64); ok && val >= 0 {
					api.server.priorityClientPool.setPriceUpdate(client, uint64(val), true)
				} else {
					err = errInvalidValue
				}
			default:
				err = errInvalidParam
			}
			if err != nil {
				if finalErr == nil {
					finalErr = err
				} else {
					finalErr = errMultiple
				}
			}
		}
		if updatePrice && client.connected {
			api.server.priorityClientPool.updatePriceFactors(client)
		}
	})
	return finalErr
}

// eventSub represents an event subscription
type eventSub struct {
	notifier         *rpc.Notifier
	rpcSub           *rpc.Subscription
	clientTags       []string
	totalCapUnderrun bool
}

// SubscribeEvent subscribes to global events and client events related to the clients matching the given tags.
// If totalCapUnderrun is true then totalCapacity updates are only sent when totalCapacity drops under totalConnectedCapacity.
func (api *PrivateLightServerAPI) SubscribeEvent(ctx context.Context, clientTags []string, totalCapUnderrun bool) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()
	api.server.priorityClientPool.subscribeEvents(&eventSub{notifier, rpcSub, clientTags, totalCapUnderrun})
	return rpcSub, nil
}

// Benchmark runs a request performance benchmark with a given set of measurement setups
// in multiple passes specified by passCount. The measurement time for each setup in each
// pass is specified in milliseconds by length.
//
// Note: measurement time is adjusted for each pass depending on the previous ones.
// Therefore a controlled total measurement time is achievable in multiple passes.
func (api *PrivateLightServerAPI) Benchmark(setups []map[string]interface{}, passCount, length int) ([]map[string]interface{}, error) {
	benchmarks := make([]requestBenchmark, len(setups))
	for i, setup := range setups {
		if t, ok := setup["type"].(string); ok {
			getInt := func(field string, def int) int {
				if value, ok := setup[field].(float64); ok {
					return int(value)
				}
				return def
			}
			getBool := func(field string, def bool) bool {
				if value, ok := setup[field].(bool); ok {
					return value
				}
				return def
			}
			switch t {
			case "header":
				benchmarks[i] = &benchmarkBlockHeaders{
					amount:  getInt("amount", 1),
					skip:    getInt("skip", 1),
					byHash:  getBool("byHash", false),
					reverse: getBool("reverse", false),
				}
			case "body":
				benchmarks[i] = &benchmarkBodiesOrReceipts{receipts: false}
			case "receipts":
				benchmarks[i] = &benchmarkBodiesOrReceipts{receipts: true}
			case "proof":
				benchmarks[i] = &benchmarkProofsOrCode{code: false}
			case "code":
				benchmarks[i] = &benchmarkProofsOrCode{code: true}
			case "cht":
				benchmarks[i] = &benchmarkHelperTrie{
					bloom:    false,
					reqCount: getInt("amount", 1),
				}
			case "bloom":
				benchmarks[i] = &benchmarkHelperTrie{
					bloom:    true,
					reqCount: getInt("amount", 1),
				}
			case "txSend":
				benchmarks[i] = &benchmarkTxSend{}
			case "txStatus":
				benchmarks[i] = &benchmarkTxStatus{}
			default:
				return nil, errUnknownBenchmarkType
			}
		} else {
			return nil, errUnknownBenchmarkType
		}
	}
	rs := api.server.protocolManager.runBenchmark(benchmarks, passCount, time.Millisecond*time.Duration(length))
	result := make([]map[string]interface{}, len(setups))
	for i, r := range rs {
		res := make(map[string]interface{})
		if r.err == nil {
			res["totalCount"] = r.totalCount
			res["avgTime"] = r.avgTime
			res["maxInSize"] = r.maxInSize
			res["maxOutSize"] = r.maxOutSize
		} else {
			res["error"] = r.err.Error()
		}
		result[i] = res
	}
	return result, nil
}

// PrivateDebugAPI provides an API to debug LES light server functionality.
type PrivateDebugAPI struct {
	server *LesServer
}

// NewPrivateDebugAPI creates a new LES light server debug API.
func NewPrivateDebugAPI(server *LesServer) *PrivateDebugAPI {
	return &PrivateDebugAPI{
		server: server,
	}
}

// FreezeClient forces a temporary client freeze which normally happens when the server is overloaded
func (api *PrivateDebugAPI) FreezeClient(id enode.ID) error {
	return api.server.priorityClientPool.freezeClient(id)
}

// priorityClientInfo entries exist for all prioritized clients and currently connected non-priority clients
type priorityClientInfo struct {
	capacity                                      uint64 // zero for non-priority clients
	connected                                     bool
	userTags                                      map[string]struct{}
	peer                                          *peer
	id                                            enode.ID
	timeFactor, capacityFactor, requestCostFactor float64
	priceUpdatePeriod                             uint64
	priceTracker                                  priceTracker
}

// matchTags checks whether the client matches the given tags
func (c *priorityClientInfo) matchTags(tags []string) bool {
	for _, tag := range tags {
		if len(tag) > 0 && tag[0] == '$' {
			switch tag {
			case "$all":
			case "$connected":
				if !c.connected {
					return false
				}
			case "$disconnected":
				if c.connected {
					return false
				}
			case "$priority":
				if c.capacity == 0 {
					return false
				}
			case "$free":
				if c.capacity != 0 {
					return false
				}
			default:
				return false
			}
		} else {
			if _, ok := c.userTags[tag]; !ok {
				return false
			}
		}
	}
	return true
}

type (
	// clientPool is implemented by both the free and priority client pools
	clientPool interface {
		peerSetNotify
		setLimits(count int, totalCap uint64)
	}
	// priorityClientPool stores information about prioritized clients
	priorityClientPool struct {
		lock                                               sync.Mutex
		child                                              clientPool
		ps                                                 *peerSet
		clients                                            map[enode.ID]*priorityClientInfo
		totalCap, totalCapAnnounced                        uint64
		totalConnectedCap, totalAssignedCap, freeClientCap uint64
		maxPeers, freeCount, priorityCount                 int
		logger                                             *csvlogger.Logger
		logTotalPriConn                                    *csvlogger.Channel

		subs            map[*eventSub]struct{}
		updateSchedule  []scheduledUpdate
		scheduleCounter uint64
	}
	// scheduledUpdate represents a delayed total capacity update
	scheduledUpdate struct {
		time         mclock.AbsTime
		totalCap, id uint64
	}
)

// newPriorityClientPool creates a new priority client pool
func newPriorityClientPool(freeClientCap uint64, ps *peerSet, child clientPool, metricsLogger, eventLogger *csvlogger.Logger) *priorityClientPool {
	return &priorityClientPool{
		clients:         make(map[enode.ID]*priorityClientInfo),
		freeClientCap:   freeClientCap,
		ps:              ps,
		child:           child,
		logger:          eventLogger,
		logTotalPriConn: metricsLogger.NewChannel("totalPriConn", 0),
	}
}

// registerPeer is called when a new client is connected. If the client has no
// priority assigned then it is passed to the child pool which may either keep it
// or disconnect it.
//
// Note: priorityClientPool also stores a record about free clients while they are
// connected in order to be able to assign priority to them later.
func (v *priorityClientPool) registerPeer(p *peer) {
	v.lock.Lock()
	defer v.lock.Unlock()

	id := p.ID()
	c := v.clients[id]
	if c == nil {
		c = &priorityClientInfo{id: id}
		v.clients[id] = c
	}
	v.logger.Event(fmt.Sprintf("priorityClientPool: registerPeer  capacity=%d  connected=%v, %x", c.capacity, c.connected, id.Bytes()))
	if c.connected {
		return
	}
	if c.capacity == 0 && v.child != nil {
		v.child.registerPeer(p)
		v.freeCount++
	}
	if c.capacity != 0 && v.totalConnectedCap+c.capacity > v.totalCap {
		v.logger.Event(fmt.Sprintf("priorityClientPool: rejected, %x", id.Bytes()))
		go v.ps.Unregister(p.id)
		return
	}

	c.connected = true
	v.updatePriceFactors(c)
	c.peer = p
	p.priceTracker = &c.priceTracker
	if c.capacity != 0 {
		v.priorityCount++
		v.totalConnectedCap += c.capacity
		v.logger.Event(fmt.Sprintf("priorityClientPool: accepted with %d capacity, %x", c.capacity, id.Bytes()))
		v.logTotalPriConn.Update(float64(v.totalConnectedCap))
		if v.child != nil {
			v.child.setLimits(v.maxPeers-v.priorityCount, v.totalCap-v.totalConnectedCap)
		}
		p.updateCapacity(c.capacity)
	}
	v.sendEvent("connect", c)
}

// unregisterPeer is called when a client is disconnected. If the client has no
// priority assigned then it is also removed from the child pool.
func (v *priorityClientPool) unregisterPeer(p *peer) {
	v.lock.Lock()
	defer v.lock.Unlock()

	id := p.ID()
	c := v.clients[id]
	if c == nil {
		return
	}
	v.logger.Event(fmt.Sprintf("priorityClientPool: unregisterPeer  capacity=%d  connected=%v, %x", c.capacity, c.connected, id.Bytes()))
	if !c.connected {
		return
	}
	c.priceTracker.setFactors(0, 0)
	if c.capacity != 0 {
		c.connected = false
		v.priorityCount--
		v.totalConnectedCap -= c.capacity
		v.logTotalPriConn.Update(float64(v.totalConnectedCap))
		if v.child != nil {
			v.child.setLimits(v.maxPeers-v.priorityCount, v.totalCap-v.totalConnectedCap)
		}
	} else {
		if v.child != nil {
			v.child.unregisterPeer(p)
			v.freeCount--
		}
		delete(v.clients, id)
	}
	v.sendEvent("disconnect", c)
}

// clientInfo creates a client info data structure
func (v *priorityClientPool) clientInfo(c *priorityClientInfo) map[string]interface{} {
	clientInfo := make(map[string]interface{})
	clientInfo["isConnected"] = c.connected
	capacity := c.capacity
	pri := true
	if capacity == 0 {
		capacity = v.freeClientCap
		pri = false
	}
	clientInfo["capacity"] = capacity
	clientInfo["hasPriority"] = pri
	tags := make([]string, 0, len(c.userTags))
	for tag, _ := range c.userTags {
		tags = append(tags, tag)
	}
	clientInfo["userTags"] = tags
	clientInfo["pricing/totalAmount"] = c.priceTracker.getTotalAmount()
	return clientInfo
}

// sendEvent sends an event to the subscribers interested in it. For global events client == nil.
func (v *priorityClientPool) sendEvent(clientEvent string, client *priorityClientInfo) {
	var event map[string]interface{}
	makeEvent := func() {
		event = make(map[string]interface{})
		event["totalCapacity"] = v.totalCap
		event["totalConnectedCapacity"] = v.totalConnectedCap + uint64(v.freeCount)*v.freeClientCap
		event["priorityConnectedCapacity"] = v.totalConnectedCap
		if client != nil {
			event["clientEvent"] = clientEvent
			event["clientId"] = client.id
			event["clientInfo"] = v.clientInfo(client)
		}
	}

	for sub := range v.subs {
		select {
		case <-sub.rpcSub.Err():
			delete(v.subs, sub)
		case <-sub.notifier.Closed():
			delete(v.subs, sub)
		default:
			var send bool
			if client == nil {
				send = v.totalCap < v.totalConnectedCap || !sub.totalCapUnderrun
			} else {
				send = client.matchTags(sub.clientTags)
			}
			if send {
				if event == nil {
					makeEvent()
				}
				sub.notifier.Notify(sub.rpcSub.ID, event)
			}
		}
	}
}

// matchClients calls the given callback for all clients in the ids list or matching the
// given tags. If an unknown client is listed in ids it is temporarily created but only
// kept if the callback has assigned a priority to it.
func (v *priorityClientPool) matchClients(ids []enode.ID, tags []string, cb func(client *priorityClientInfo)) {
	v.lock.Lock()
	defer v.lock.Unlock()

	for _, id := range ids {
		c := v.clients[id]
		if c == nil {
			c = &priorityClientInfo{id: id}
			v.clients[id] = c
		}
		cb(c)
		if c.capacity == 0 && !c.connected {
			delete(v.clients, id)
		}
	}
	if len(tags) > 0 {
		for _, info := range v.clients {
			if info.matchTags(tags) {
				cb(info)
			}
		}
	}
}

// priceUpdate sends a price update client event and schedules a new update with the
// price tracker if necessary.
func (v *priorityClientPool) priceUpdate(client *priorityClientInfo) {
	v.lock.Lock()
	defer v.lock.Unlock()

	v.sendEvent("priceUpdate", client)
	if client.priceUpdatePeriod != 0 {
		v.setPriceUpdate(client, client.priceUpdatePeriod, true)
	}
}

// setPriceUpdate schedules a price update when the total price reaches the given limit.
// If periodic is false then the limit is interpreted as an absolute value while if true
// it is relative to the current totalAmount value or the its value at the last future update.
func (v *priorityClientPool) setPriceUpdate(client *priorityClientInfo, value uint64, periodic bool) {
	if value == 0 {
		client.priceTracker.setCallback(0, nil)
		return
	}
	if periodic {
		client.priceUpdatePeriod = value
		value += client.priceTracker.getTotalAmount()
	} else {
		client.priceUpdatePeriod = 0
	}
	client.priceTracker.setCallback(value, func() { v.priceUpdate(client) })
}

// setLimits updates the allowed peer count and total capacity of the priority
// client pool. Since the free client pool is a child of the priority pool the
// remaining peer count and capacity is assigned to the free pool by calling its
// own setLimits function.
//
// Note: a decreasing change of the total capacity is applied with a delay.
func (v *priorityClientPool) setLimits(count int, totalCap uint64) {
	v.lock.Lock()
	defer v.lock.Unlock()

	v.totalCapAnnounced = totalCap
	if totalCap > v.totalCap {
		v.setLimitsNow(count, totalCap)
		v.sendEvent("", nil)
		return
	}
	v.setLimitsNow(count, v.totalCap)
	if totalCap < v.totalCap {
		v.sendEvent("", nil)
		for i, s := range v.updateSchedule {
			if totalCap >= s.totalCap {
				s.totalCap = totalCap
				v.updateSchedule = v.updateSchedule[:i+1]
				return
			}
		}
		v.updateSchedule = append(v.updateSchedule, scheduledUpdate{time: mclock.Now() + mclock.AbsTime(dropCapacityDelay), totalCap: totalCap})
		if len(v.updateSchedule) == 1 {
			v.scheduleCounter++
			id := v.scheduleCounter
			v.updateSchedule[0].id = id
			time.AfterFunc(dropCapacityDelay, func() { v.checkUpdate(id) })
		}
	} else {
		v.updateSchedule = nil
	}
}

// checkUpdate performs the next scheduled update if possible and schedules
// the one after that
func (v *priorityClientPool) checkUpdate(id uint64) {
	v.lock.Lock()
	defer v.lock.Unlock()

	if len(v.updateSchedule) == 0 || v.updateSchedule[0].id != id {
		return
	}
	v.setLimitsNow(v.maxPeers, v.updateSchedule[0].totalCap)
	v.updateSchedule = v.updateSchedule[1:]
	if len(v.updateSchedule) != 0 {
		v.scheduleCounter++
		id := v.scheduleCounter
		v.updateSchedule[0].id = id
		dt := time.Duration(v.updateSchedule[0].time - mclock.Now())
		time.AfterFunc(dt, func() { v.checkUpdate(id) })
	}
}

// setLimits updates the allowed peer count and total capacity immediately
func (v *priorityClientPool) setLimitsNow(count int, totalCap uint64) {
	if v.priorityCount > count || v.totalConnectedCap > totalCap {
		for id, c := range v.clients {
			if c.connected {
				v.logger.Event(fmt.Sprintf("priorityClientPool: setLimitsNow kicked out, %x", id.Bytes()))
				c.connected = false
				v.totalConnectedCap -= c.capacity
				v.logTotalPriConn.Update(float64(v.totalConnectedCap))
				v.priorityCount--
				v.sendEvent("disconnect", c)
				go v.ps.Unregister(c.peer.id)
				if v.priorityCount <= count && v.totalConnectedCap <= totalCap {
					break
				}
			}
		}
	}
	v.maxPeers = count
	v.totalCap = totalCap
	if v.child != nil {
		v.child.setLimits(v.maxPeers-v.priorityCount, v.totalCap-v.totalConnectedCap)
	}
}

// capacityInfo queries total available, connected and assigned capacity
func (v *priorityClientPool) capacityInfo() (total, conn, priConn, priAssigned uint64) {
	v.lock.Lock()
	defer v.lock.Unlock()

	return v.totalCapAnnounced, v.totalConnectedCap + uint64(v.freeCount)*v.freeClientCap, v.totalConnectedCap, v.totalAssignedCap
}

// subscribeEvents subscribes to events
func (v *priorityClientPool) subscribeEvents(sub *eventSub) {
	v.lock.Lock()
	defer v.lock.Unlock()

	v.subs[sub] = struct{}{}
}

// setClientTags sets the user tags associated with the client
func (v *priorityClientPool) setClientTags(c *priorityClientInfo, tags []string) error {
	if len(tags) == 0 {
		c.userTags = nil
	} else {
		c.userTags = make(map[string]struct{})
		for _, tag := range tags {
			if len(tag) > 0 && tag[0] == '$' {
				return errInvalidTag
			}
			c.userTags[tag] = struct{}{}
		}
	}
	return nil
}

// setClientCapacity sets the priority capacity assigned to a given client
func (v *priorityClientPool) setClientCapacity(c *priorityClientInfo, capacity uint64) error {
	if c.capacity == capacity {
		return nil
	}
	v.totalAssignedCap += capacity - c.capacity
	if c.connected {
		if v.totalConnectedCap+capacity > v.totalCap+c.capacity {
			return errTotalCap
		}
		if c.capacity == 0 {
			if v.child != nil {
				v.child.unregisterPeer(c.peer)
				v.freeCount--
			}
			v.priorityCount++
		}
		if capacity == 0 {
			v.priorityCount--
		}
		v.totalConnectedCap += capacity - c.capacity
		v.logTotalPriConn.Update(float64(v.totalConnectedCap))
		if v.child != nil {
			v.child.setLimits(v.maxPeers-v.priorityCount, v.totalCap-v.totalConnectedCap)
		}
		if capacity == 0 {
			if v.child != nil {
				v.child.registerPeer(c.peer)
				v.freeCount++
			}
			c.peer.updateCapacity(v.freeClientCap)
		} else {
			c.peer.updateCapacity(capacity)
		}
	}
	if capacity != 0 || c.connected {
		c.capacity = capacity
	} else {
		delete(v.clients, c.id)
	}
	v.sendEvent("updateCapacity", c)
	if c.connected {
		v.logger.Event(fmt.Sprintf("priorityClientPool: changed capacity to %d, %x", capacity, c.id.Bytes()))
	}
	return nil
}

// freezeClient forces a temporary client freeze which normally happens when the server is overloaded
func (v *priorityClientPool) freezeClient(id enode.ID) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	c := v.clients[id]
	if c != nil && c.peer != nil {
		c.peer.freezeClient()
		return nil
	} else {
		return errClientNotConnected
	}
}

// updatePriceFactors updates the price tracker for a client based on current parameters
func (v *priorityClientPool) updatePriceFactors(c *priorityClientInfo) {
	capacity := c.capacity
	if capacity == 0 {
		capacity = v.freeClientCap
	}
	c.priceTracker.setFactors(c.timeFactor+c.capacityFactor*float64(capacity)/1000000, c.requestCostFactor)
}

// priceTracker calculates service price for a single client based on connection time
// and/or requests served. It also provides a callback function when a given total
// price threshold has been reached.
type priceTracker struct {
	lock                           sync.Mutex
	totalAmount, callbackThreshold uint64
	callback                       func()
	timeFactor, requestFactor      float64
	lastUpdate, nextUpdate         mclock.AbsTime
	updateTimer                    *time.Timer
}

// update recalculates the total price, calls the callback and/or modifies
// the next scheduled update if necessary
func (pt *priceTracker) update() {
	now := mclock.Now()
	if pt.lastUpdate != 0 {
		dt := now - pt.lastUpdate
		if dt > 0 {
			pt.totalAmount += uint64(pt.timeFactor * float64(dt))
		}
	}
	pt.lastUpdate = now
	if pt.callbackThreshold == 0 || pt.callback == nil {
		pt.nextUpdate = 0
		pt.updateAfter(0)
	} else {
		if pt.totalAmount >= pt.callbackThreshold {
			pt.callbackThreshold = 0
			pt.nextUpdate = 0
			go pt.callback()
		} else {
			if pt.timeFactor > 1e-100 {
				dt := float64(pt.callbackThreshold-pt.totalAmount) / pt.timeFactor
				if dt > 1e15 {
					dt = 1e15
				}
				d := time.Duration(dt)
				if pt.nextUpdate == 0 || pt.nextUpdate > now+mclock.AbsTime(d) {
					if d > time.Second {
						d = ((d - time.Second) * 7 / 8) + time.Second
					}
					pt.nextUpdate = now + mclock.AbsTime(d)
					pt.updateAfter(d)
				}
			} else {
				pt.nextUpdate = 0
				pt.updateAfter(0)
			}
		}
	}
}

// updateAfter schedules an update in the future
func (pt *priceTracker) updateAfter(dt time.Duration) {
	if pt.updateTimer == nil || pt.updateTimer.Stop() {
		if dt == 0 {
			pt.updateTimer = nil
		} else {
			pt.updateTimer = time.AfterFunc(dt, func() {
				pt.lock.Lock()
				defer pt.lock.Unlock()

				if pt.callbackThreshold != 0 {
					pt.update()
				}
			})
		}
	}
}

// requestCost should be called after serving a request for the given peer
func (pt *priceTracker) requestCost(cost uint64) {
	pt.lock.Lock()
	defer pt.lock.Unlock()

	if pt.requestFactor != 0 {
		pt.totalAmount += uint64(float64(cost) * pt.requestFactor)
		pt.update()
	}
}

// getTotalAmount returns the current total cost accumulated.
func (pt *priceTracker) getTotalAmount() uint64 {
	pt.lock.Lock()
	defer pt.lock.Unlock()

	pt.update()
	return pt.totalAmount
}

// setFactors sets the price factors. timeFactor is the price of a nanosecond of
// connection while requestFactor is the price of a "realCost" unit.
func (pt *priceTracker) setFactors(timeFactor, requestFactor float64) {
	pt.lock.Lock()
	defer pt.lock.Unlock()

	pt.update()
	pt.timeFactor = timeFactor
	pt.requestFactor = requestFactor
}

// setCallback sets up a one-time callback to be called when totalAmount reaches
// the threshold. If it has already reached the threshold the callback is called
// immediately.
func (pt *priceTracker) setCallback(threshold uint64, callback func()) {
	pt.lock.Lock()
	defer pt.lock.Unlock()

	pt.callbackThreshold = threshold
	pt.callback = callback
	pt.update()
}

// PrivateLightAPI provides an API to access the LES light server or light client.
type PrivateLightAPI struct {
	backend *lesCommons
	reg     *checkpointRegistrar
}

// NewPrivateLightAPI creates a new LES service API.
func NewPrivateLightAPI(backend *lesCommons, reg *checkpointRegistrar) *PrivateLightAPI {
	return &PrivateLightAPI{
		backend: backend,
		reg:     reg,
	}
}

// LatestCheckpoint returns the latest local checkpoint package.
//
// The checkpoint package consists of 4 strings:
//   result[0], hex encoded latest section index
//   result[1], 32 bytes hex encoded latest section head hash
//   result[2], 32 bytes hex encoded latest section canonical hash trie root hash
//   result[3], 32 bytes hex encoded latest section bloom trie root hash
func (api *PrivateLightAPI) LatestCheckpoint() ([4]string, error) {
	var res [4]string
	cp := api.backend.latestLocalCheckpoint()
	if cp.Empty() {
		return res, errNoCheckpoint
	}
	res[0] = hexutil.EncodeUint64(cp.SectionIndex)
	res[1], res[2], res[3] = cp.SectionHead.Hex(), cp.CHTRoot.Hex(), cp.BloomRoot.Hex()
	return res, nil
}

// GetLocalCheckpoint returns the specific local checkpoint package.
//
// The checkpoint package consists of 3 strings:
//   result[0], 32 bytes hex encoded latest section head hash
//   result[1], 32 bytes hex encoded latest section canonical hash trie root hash
//   result[2], 32 bytes hex encoded latest section bloom trie root hash
func (api *PrivateLightAPI) GetCheckpoint(index uint64) ([3]string, error) {
	var res [3]string
	cp := api.backend.getLocalCheckpoint(index)
	if cp.Empty() {
		return res, errNoCheckpoint
	}
	res[0], res[1], res[2] = cp.SectionHead.Hex(), cp.CHTRoot.Hex(), cp.BloomRoot.Hex()
	return res, nil
}

// GetCheckpointContractAddress returns the contract contract address in hex format.
func (api *PrivateLightAPI) GetCheckpointContractAddress() (string, error) {
	if api.reg == nil {
		return "", errNotActivated
	}
	return api.reg.config.ContractAddr.Hex(), nil
}
