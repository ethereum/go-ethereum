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
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/common/mclock"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"github.com/ethereum/go-ethereum/rpc"
)

var (
	ErrMinCap               = errors.New("capacity too small")
	ErrTotalCap             = errors.New("total capacity exceeded")
	ErrUnknownBenchmarkType = errors.New("unknown benchmark type")

	dropCapacityDelay = time.Second // delay applied to decreasing capacity changes
)

// PrivateLightServerAPI provides an API to access the LES light server.
// It offers only methods that operate on public data that is freely available to anyone.
type PrivateLightServerAPI struct {
	server *LesServer
}

// NewPrivateLightServerAPI creates a new LES light server API.
func NewPrivateLightServerAPI(server *LesServer) *PrivateLightServerAPI {
	return &PrivateLightServerAPI{
		server: server,
	}
}

// TotalCapacity queries total available capacity for all clients
func (api *PrivateLightServerAPI) TotalCapacity() hexutil.Uint64 {
	return hexutil.Uint64(api.server.priorityClientPool.totalCapacity())
}

// SubscribeTotalCapacity subscribes to changed total capacity events.
// If onlyUnderrun is true then notification is sent only if the total capacity
// drops under the total capacity of connected priority clients.
//
// Note: actually applying decreasing total capacity values is delayed while the
// notification is sent instantly. This allows lowering the capacity of a priority client
// or choosing which one to drop before the system drops some of them automatically.
func (api *PrivateLightServerAPI) SubscribeTotalCapacity(ctx context.Context, onlyUnderrun bool) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()
	api.server.priorityClientPool.subscribeTotalCapacity(&tcSubscription{notifier, rpcSub, onlyUnderrun})
	return rpcSub, nil
}

type (
	// tcSubscription represents a total capacity subscription
	tcSubscription struct {
		notifier     *rpc.Notifier
		rpcSub       *rpc.Subscription
		onlyUnderrun bool
	}
	tcSubs map[*tcSubscription]struct{}
)

// send sends a changed total capacity event to the subscribers
func (s tcSubs) send(tc uint64, underrun bool) {
	for sub := range s {
		select {
		case <-sub.rpcSub.Err():
			delete(s, sub)
		case <-sub.notifier.Closed():
			delete(s, sub)
		default:
			if underrun || !sub.onlyUnderrun {
				sub.notifier.Notify(sub.rpcSub.ID, tc)
			}
		}
	}
}

// MinimumCapacity queries minimum assignable capacity for a single client
func (api *PrivateLightServerAPI) MinimumCapacity() hexutil.Uint64 {
	return hexutil.Uint64(minCapacity)
}

// FreeClientCapacity queries the capacity provided for free clients
func (api *PrivateLightServerAPI) FreeClientCapacity() hexutil.Uint64 {
	return hexutil.Uint64(api.server.freeClientCap)
}

// SetClientCapacity sets the priority capacity assigned to a given client.
// If the assigned capacity is bigger than zero then connection is always
// guaranteed. The sum of capacity assigned to priority clients can not exceed
// the total available capacity.
//
// Note: assigned capacity can be changed while the client is connected with
// immediate effect.
func (api *PrivateLightServerAPI) SetClientCapacity(id enode.ID, cap uint64) error {
	if cap != 0 && cap < minCapacity {
		return ErrMinCap
	}
	return api.server.priorityClientPool.setClientCapacity(id, cap)
}

// GetClientCapacity returns the capacity assigned to a given client
func (api *PrivateLightServerAPI) GetClientCapacity(id enode.ID) hexutil.Uint64 {
	api.server.priorityClientPool.lock.Lock()
	defer api.server.priorityClientPool.lock.Unlock()

	return hexutil.Uint64(api.server.priorityClientPool.clients[id].cap)
}

// clientPool is implemented by both the free and priority client pools
type clientPool interface {
	peerSetNotify
	setLimits(count int, totalCap uint64)
}

// priorityClientPool stores information about prioritized clients
type priorityClientPool struct {
	lock                             sync.Mutex
	child                            clientPool
	ps                               *peerSet
	clients                          map[enode.ID]priorityClientInfo
	totalCap, totalCapAnnounced      uint64
	totalConnectedCap, freeClientCap uint64
	maxPeers, priorityCount          int

	subs            tcSubs
	updateSchedule  []scheduledUpdate
	scheduleCounter uint64
}

// scheduledUpdate represents a delayed total capacity update
type scheduledUpdate struct {
	time         mclock.AbsTime
	totalCap, id uint64
}

// priorityClientInfo entries exist for all prioritized clients and currently connected non-priority clients
type priorityClientInfo struct {
	cap       uint64 // zero for non-priority clients
	connected bool
	peer      *peer
}

// newPriorityClientPool creates a new priority client pool
func newPriorityClientPool(freeClientCap uint64, ps *peerSet, child clientPool) *priorityClientPool {
	return &priorityClientPool{
		clients:       make(map[enode.ID]priorityClientInfo),
		freeClientCap: freeClientCap,
		ps:            ps,
		child:         child,
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
	if c.connected {
		return
	}
	if c.cap == 0 && v.child != nil {
		v.child.registerPeer(p)
	}
	if c.cap != 0 && v.totalConnectedCap+c.cap > v.totalCap {
		go v.ps.Unregister(p.id)
		return
	}

	c.connected = true
	c.peer = p
	v.clients[id] = c
	if c.cap != 0 {
		v.priorityCount++
		v.totalConnectedCap += c.cap
		if v.child != nil {
			v.child.setLimits(v.maxPeers-v.priorityCount, v.totalCap-v.totalConnectedCap)
		}
		p.updateCapacity(c.cap)
	}
}

// unregisterPeer is called when a client is disconnected. If the client has no
// priority assigned then it is also removed from the child pool.
func (v *priorityClientPool) unregisterPeer(p *peer) {
	v.lock.Lock()
	defer v.lock.Unlock()

	id := p.ID()
	c := v.clients[id]
	if !c.connected {
		return
	}
	if c.cap != 0 {
		c.connected = false
		v.clients[id] = c
		v.priorityCount--
		v.totalConnectedCap -= c.cap
		if v.child != nil {
			v.child.setLimits(v.maxPeers-v.priorityCount, v.totalCap-v.totalConnectedCap)
		}
	} else {
		if v.child != nil {
			v.child.unregisterPeer(p)
		}
		delete(v.clients, id)
	}
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
		v.subs.send(totalCap, false)
		return
	}
	v.setLimitsNow(count, v.totalCap)
	if totalCap < v.totalCap {
		v.subs.send(totalCap, totalCap < v.totalConnectedCap)
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
				c.connected = false
				v.totalConnectedCap -= c.cap
				v.priorityCount--
				v.clients[id] = c
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

// totalCapacity queries total available capacity for all clients
func (v *priorityClientPool) totalCapacity() uint64 {
	v.lock.Lock()
	defer v.lock.Unlock()

	return v.totalCapAnnounced
}

// subscribeTotalCapacity subscribes to changed total capacity events
func (v *priorityClientPool) subscribeTotalCapacity(sub *tcSubscription) {
	v.lock.Lock()
	defer v.lock.Unlock()

	v.subs[sub] = struct{}{}
}

// setClientCapacity sets the priority capacity assigned to a given client
func (v *priorityClientPool) setClientCapacity(id enode.ID, cap uint64) error {
	v.lock.Lock()
	defer v.lock.Unlock()

	c := v.clients[id]
	if c.cap == cap {
		return nil
	}
	if c.connected {
		if v.totalConnectedCap+cap > v.totalCap+c.cap {
			return ErrTotalCap
		}
		if c.cap == 0 {
			if v.child != nil {
				v.child.unregisterPeer(c.peer)
			}
			v.priorityCount++
		}
		if cap == 0 {
			v.priorityCount--
		}
		v.totalConnectedCap += cap - c.cap
		if v.child != nil {
			v.child.setLimits(v.maxPeers-v.priorityCount, v.totalCap-v.totalConnectedCap)
		}
		if cap == 0 {
			if v.child != nil {
				v.child.registerPeer(c.peer)
			}
			c.peer.updateCapacity(v.freeClientCap)
		} else {
			c.peer.updateCapacity(cap)
		}
	}
	if cap != 0 || c.connected {
		c.cap = cap
		v.clients[id] = c
	} else {
		delete(v.clients, id)
	}
	return nil
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
				return nil, ErrUnknownBenchmarkType
			}
		} else {
			return nil, ErrUnknownBenchmarkType
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
