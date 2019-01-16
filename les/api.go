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
	"errors"
	"sync"

	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/p2p/enode"
)

var (
	ErrMinCap   = errors.New("capacity too small")
	ErrTotalCap = errors.New("total capacity exceeded")
)

// PublicLesServerAPI  provides an API to access the les server.
// It offers only methods that operate on public data that is freely available to anyone.
type PrivateLesServerAPI struct {
	server *LesServer
	pm     *ProtocolManager
	vip    *vipClientPool
}

// NewPublicLesServerAPI creates a new les server API.
func NewPrivateLesServerAPI(server *LesServer) *PrivateLesServerAPI {
	vip := &vipClientPool{
		clients: make(map[enode.ID]vipClientInfo),
		totalCap: server.totalCapacity,
		pm:      server.protocolManager,
	}
	server.protocolManager.vipClientPool = vip
	return &PrivateLesServerAPI{
		server: server,
		pm:     server.protocolManager,
		vip:    vip,
	}
}

// TotalCapacity queries total available capacity for all clients
func (api *PrivateLesServerAPI) TotalCapacity() hexutil.Uint64 {
	return hexutil.Uint64(api.server.totalCapacity)
}

// MinimumCapacity queries minimum assignable capacity for a single client
func (api *PrivateLesServerAPI) MinimumCapacity() hexutil.Uint64 {
	return hexutil.Uint64(api.server.minCapacity)
}

// vipClientPool stores information about prioritized clients
type vipClientPool struct {
	lock                                  sync.Mutex
	pm                                    *ProtocolManager
	clients                               map[enode.ID]vipClientInfo
	totalCap, totalVipCap, totalConnectedCap uint64
	vipCount                              int
}

// vipClientInfo entries exist for all prioritized clients and currently connected free clients
type vipClientInfo struct {
	cap        uint64 // zero for non-vip clients
	connected bool
	updateCap  func(uint64)
}

// SetClientCapacity sets the priority capacity assigned to a given client.
// If the assigned capacity is bigger than zero then connection is always
// guaranteed. The sum of capacity assigned to priority clients can not exceed
// the total available capacity.
//
// Note: assigned capacity can be changed while the client is connected with
// immediate effect.
func (api *PrivateLesServerAPI) SetClientCapacity(id enode.ID, cap uint64) error {
	if cap != 0 && cap < api.server.minCapacity {
		return ErrMinCap
	}

	api.vip.lock.Lock()
	defer api.vip.lock.Unlock()

	c := api.vip.clients[id]
	if api.vip.totalVipCap+cap > api.vip.totalCap+c.cap {
		return ErrTotalCap
	}
	api.vip.totalVipCap += cap - c.cap
	if c.updateCap != nil && cap != 0 {
		c.updateCap(cap)
	}
	if c.connected {
		if c.cap != 0 {
			api.vip.vipCount--
		}
		if cap != 0 {
			api.vip.vipCount++
		}
		api.vip.totalConnectedCap += cap - c.cap
		api.pm.clientPool.setConnLimit(api.pm.maxFreePeers(api.vip.vipCount, api.vip.totalConnectedCap))
	}
	if c.updateCap != nil && cap == 0 {
		c.updateCap(cap)
	}
	if cap != 0 || c.connected {
		c.cap = cap
		api.vip.clients[id] = c
	} else {
		delete(api.vip.clients, id)
	}
	return nil
}

// GetClientCapacity returns the capacity assigned to a given client
func (api *PrivateLesServerAPI) GetClientCapacity(id enode.ID) hexutil.Uint64 {
	api.vip.lock.Lock()
	defer api.vip.lock.Unlock()

	return hexutil.Uint64(api.vip.clients[id].cap)
}

// connect should be called when a new client is connected. The callback function
// is called when the assigned capacity is changed while the client is connected.
// It returns the priority capacity or zero if the client is not prioritized.
// It also returns whether the client can be accepted.
//
// Note: vipClientPool also stores a record about free clients while they are
// connected in order to be able to assign priority to them later with the callback
// function if necessary.
func (v *vipClientPool) connect(id enode.ID, updateCap func(uint64)) (uint64, bool) {
	v.lock.Lock()
	defer v.lock.Unlock()

	c := v.clients[id]
	if c.connected {
		return 0, false
	}
	c.connected = true
	c.updateCap = updateCap
	v.clients[id] = c
	if c.cap != 0 {
		v.vipCount++
	}
	v.totalConnectedCap += c.cap
	v.pm.clientPool.setConnLimit(v.pm.maxFreePeers(v.vipCount, v.totalConnectedCap))
	return c.cap, true
}

// disconnect should be called when a client is disconnected.
// It should be called for all clients accepted by connect even if not prioritized.
func (v *vipClientPool) disconnect(id enode.ID) {
	v.lock.Lock()
	defer v.lock.Unlock()

	c := v.clients[id]
	c.connected = false
	if c.cap != 0 {
		v.clients[id] = c
		v.vipCount--
	} else {
		delete(v.clients, id)
	}
	v.totalConnectedCap -= c.cap
	v.pm.clientPool.setConnLimit(v.pm.maxFreePeers(v.vipCount, v.totalConnectedCap))
}
