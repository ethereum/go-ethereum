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
	ErrMinCap   = errors.New("capacity too small")
	ErrTotalCap = errors.New("total capacity exceeded")

	dropCapacityDelay = time.Second
)

// PublicLesServerAPI  provides an API to access the les server.
// It offers only methods that operate on public data that is freely available to anyone.
type PrivateLesServerAPI struct {
	server *LesServer
}

// NewPublicLesServerAPI creates a new les server API.
func NewPrivateLesServerAPI(server *LesServer) *PrivateLesServerAPI {
	return &PrivateLesServerAPI{
		server: server,
	}
}

// TotalCapacity queries total available capacity for all clients
func (api *PrivateLesServerAPI) TotalCapacity() hexutil.Uint64 {
	return hexutil.Uint64(api.server.priorityClientPool.totalCapacity())
}

func (api *PrivateLesServerAPI) SubscribeTotalCapacity(ctx context.Context, onlyUnderrun bool) (*rpc.Subscription, error) {
	notifier, supported := rpc.NotifierFromContext(ctx)
	if !supported {
		return &rpc.Subscription{}, rpc.ErrNotificationsUnsupported
	}
	rpcSub := notifier.CreateSubscription()
	api.server.priorityClientPool.subscribeTotalCapacity(&tcSubscription{notifier, rpcSub, onlyUnderrun})
	return rpcSub, nil
}

type (
	tcSubscription struct {
		notifier     *rpc.Notifier
		rpcSub       *rpc.Subscription
		onlyUnderrun bool
	}
	tcSubs map[*tcSubscription]struct{}
)

func (s tcSubs) send(tc uint64, underrun bool) {
	for sub, _ := range s {
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
func (api *PrivateLesServerAPI) MinimumCapacity() hexutil.Uint64 {
	return hexutil.Uint64(minCapacity)
}

type clientPool interface {
	peerSetNotify
	setLimits(count int, totalCap uint64)
}

// priorityClientPool stores information about prioritized clients
type priorityClientPool struct {
	lock                                       sync.Mutex
	child                                      clientPool
	ps                                         *peerSet
	clients                                    map[enode.ID]priorityClientInfo
	totalCap, totalConnectedCap, freeClientCap uint64
	maxPeers, priorityCount                    int

	subs            tcSubs
	updateSchedule  []scheduledUpdate
	scheduleCounter uint64
}

type scheduledUpdate struct {
	time         mclock.AbsTime
	totalCap, id uint64
}

// priorityClientInfo entries exist for all prioritized clients and currently connected free clients
type priorityClientInfo struct {
	cap       uint64 // zero for non-priority clients
	connected bool
	peer      *peer
}

// SetClientCapacity sets the priority capacity assigned to a given client.
// If the assigned capacity is bigger than zero then connection is always
// guaranteed. The sum of capacity assigned to priority clients can not exceed
// the total available capacity.
//
// Note: assigned capacity can be changed while the client is connected with
// immediate effect.
func (api *PrivateLesServerAPI) SetClientCapacity(id enode.ID, cap uint64) error {
	if cap != 0 && cap < minCapacity {
		return ErrMinCap
	}
	return api.server.priorityClientPool.setClientCapacity(id, cap)
}

// GetClientCapacity returns the capacity assigned to a given client
func (api *PrivateLesServerAPI) GetClientCapacity(id enode.ID) hexutil.Uint64 {
	api.server.priorityClientPool.lock.Lock()
	defer api.server.priorityClientPool.lock.Unlock()

	return hexutil.Uint64(api.server.priorityClientPool.clients[id].cap)
}

func newPriorityClientPool(freeClientCap uint64, ps *peerSet, child clientPool) *priorityClientPool {
	return &priorityClientPool{
		clients:       make(map[enode.ID]priorityClientInfo),
		freeClientCap: freeClientCap,
		ps:            ps,
		child:         child,
	}
}

// connect should be called when a new client is connected. The callback function
// is called when the assigned capacity is changed while the client is connected.
// It returns the priority capacity or zero if the client is not prioritized.
// It also returns whether the client can be accepted.
//
// Note: priorityClientPool also stores a record about free clients while they are
// connected in order to be able to assign priority to them later with the callback
// function if necessary.
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
		v.ps.Unregister(p.id)
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

// disconnect should be called when a client is disconnected.
// It should be called for all clients accepted by connect even if not prioritized.
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

func (v *priorityClientPool) setLimits(count int, totalCap uint64) {
	v.lock.Lock()
	defer v.lock.Unlock()

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

func (v *priorityClientPool) setLimitsNow(count int, totalCap uint64) {
	if v.priorityCount > count || v.totalConnectedCap > totalCap {
		for id, c := range v.clients {
			if c.connected {
				c.connected = false
				v.totalConnectedCap -= c.cap
				v.priorityCount--
				v.clients[id] = c
				v.ps.Unregister(c.peer.id)
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

func (v *priorityClientPool) totalCapacity() uint64 {
	v.lock.Lock()
	defer v.lock.Unlock()

	return v.totalCap
}

func (v *priorityClientPool) subscribeTotalCapacity(sub *tcSubscription) {
	v.lock.Lock()
	defer v.lock.Unlock()

	v.subs[sub] = struct{}{}
}

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
			c.peer.updateCapacity(cap)
		}
		v.totalConnectedCap += cap - c.cap
		if v.child != nil {
			v.child.setLimits(v.maxPeers-v.priorityCount, v.totalCap-v.totalConnectedCap)
		}
		if cap == 0 {
			if v.child != nil {
				v.child.registerPeer(c.peer)
			}
			v.priorityCount--
			c.peer.updateCapacity(v.freeClientCap)
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
