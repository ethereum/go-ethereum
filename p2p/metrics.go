// Copyright 2015 The go-ethereum Authors
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

// Contains the meters and timers used by the networking layer.

package p2p

import (
	"net"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/mohae/deepcopy"
	"sync"
	"sync/atomic"
	"time"
)

const (
	MetricsInboundTraffic   = "p2p/InboundTraffic"
	MetricsInboundConnects  = "p2p/InboundConnects"
	MetricsOutboundTraffic  = "p2p/OutboundTraffic"
	MetricsOutboundConnects = "p2p/OutboundConnects"

	MetricsRegistryIngressPrefix = MetricsInboundTraffic + "/"
	MetricsRegistryEgressPrefix  = MetricsOutboundTraffic + "/"

	MeteredPeerLimit = 16384
)

var (
	ingressConnectMeter = metrics.NewRegisteredMeter(MetricsInboundConnects, nil)
	ingressTrafficMeter = metrics.NewRegisteredMeter(MetricsInboundTraffic, nil)
	egressConnectMeter  = metrics.NewRegisteredMeter(MetricsOutboundConnects, nil)
	egressTrafficMeter  = metrics.NewRegisteredMeter(MetricsOutboundTraffic, nil)

	PeerIngressRegistry = metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, MetricsRegistryIngressPrefix)
	PeerEgressRegistry  = metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, MetricsRegistryEgressPrefix)
	PeerTrafficMeters   = newPeerTrafficMeters()

	nextDefaultID uint32
)

type PeerMetrics struct {
	ID string
	IP net.IP

	// TODO: -*
	Connected    *time.Time
	Handshake    *time.Time
	Disconnected *time.Time

	Ingress int64
	Egress  int64

	traffic func() (ingress, egress int64)
}

type peerTrafficMeters struct {
	peers map[uint]*PeerMetrics
	lock  sync.RWMutex
}

func newPeerTrafficMeters() *peerTrafficMeters {
	return &peerTrafficMeters{
		peers: make(map[uint]*PeerMetrics),
	}
}

func (m *peerTrafficMeters) register(id uint, ip net.IP, traffic func() (ingress, egress int64)) error {
	now := time.Now()
	peer := &PeerMetrics{
		IP:           ip,
		Connected:    &now,
		traffic:      traffic,
	}
	m.lock.Lock()
	m.peers[id] = peer
	m.lock.Unlock()

	return nil
}

func (m *peerTrafficMeters) handshakeDone(id uint, peerID string, traffic func() (ingress, egress int64)) {
	now := time.Now()
	m.lock.Lock()
	if peer, ok := m.peers[id]; ok {
		peer.Handshake = &now
		peer.ID = peerID
		peer.traffic = traffic
	}
	m.lock.Unlock()
}

func (m *peerTrafficMeters) close(id uint) {
	now := time.Now()
	m.lock.Lock()
	m.peers[id].Disconnected = &now
	m.lock.Unlock()
}

func (m *peerTrafficMeters) Peers() map[uint]*PeerMetrics {
	peers := make(map[uint]*PeerMetrics)
	m.lock.Lock()
	for id, peer := range m.peers {
		peer.Ingress, peer.Egress = peer.traffic()
		peers[id] = deepcopy.Copy(peer).(*PeerMetrics)
		if peer.Disconnected != nil {
			PeerIngressRegistry.Unregister(peer.ID)
			PeerEgressRegistry.Unregister(peer.ID)
			delete(m.peers, id)
		}
	}
	m.lock.Unlock()
	return peers
}

type networkMeter struct {
	ingress metrics.Meter
	egress  metrics.Meter
}

// meteredConn is a wrapper around a net.Conn that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	net.Conn // Network connection to wrap with metering
	id       uint
	meter    *networkMeter

	ingressBeforeHandshake int64
	egressBeforeHandshake  int64

	lock sync.RWMutex
}

// newMeteredConn creates a new metered connection, also bumping the ingress or
// egress connection meter. If the metrics system is disabled, this function
// returns the original object.
func newMeteredConn(conn net.Conn, ingress bool, ip net.IP) net.Conn {
	// Short circuit if metrics are disabled
	if !metrics.Enabled {
		return conn
	}
	if len(PeerTrafficMeters.peers) >= MeteredPeerLimit {
		log.Warn("Metered peer limit exceeded")
		return conn
	}
	// Otherwise bump the connection counters and wrap the connection
	if ingress {
		ingressConnectMeter.Mark(1)
	} else {
		egressConnectMeter.Mark(1)
	}
	id := uint(atomic.AddUint32(&nextDefaultID, 1))
	c := &meteredConn{
		Conn: conn,
		id:   id,
	}
	PeerTrafficMeters.register(id, ip, func() (ingress, egress int64) {
		return atomic.LoadInt64(&c.ingressBeforeHandshake), atomic.LoadInt64(&c.egressBeforeHandshake)
	})
	return c
}

// Read delegates a network read to the underlying connection, bumping the ingress
// traffic meter along the way.
func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	ingressTrafficMeter.Mark(int64(n))
	c.lock.RLock()
	if c.meter == nil {
		atomic.AddInt64(&c.ingressBeforeHandshake, int64(n))
	} else {
		c.meter.ingress.Mark(int64(n))
	}
	c.lock.RUnlock()
	return n, err
}

// Write delegates a network write to the underlying connection, bumping the
// egress traffic meter along the way.
func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	egressTrafficMeter.Mark(int64(n))
	c.lock.RLock()
	if c.meter == nil {
		atomic.AddInt64(&c.egressBeforeHandshake, int64(n))
	} else {
		c.meter.egress.Mark(int64(n))
	}
	c.lock.RUnlock()
	return n, err
}

func (c *meteredConn) Close() error {
	PeerTrafficMeters.close(c.id)
	return c.Conn.Close()
}

func (c *meteredConn) handshakeDone(peerID string) {
	m := &networkMeter{
		ingress: metrics.NewRegisteredMeter(peerID, PeerIngressRegistry),
		egress:  metrics.NewRegisteredMeter(peerID, PeerEgressRegistry),
	}
	c.lock.Lock()
	m.ingress.Mark(atomic.LoadInt64(&c.ingressBeforeHandshake))
	m.egress.Mark(atomic.LoadInt64(&c.egressBeforeHandshake))
	c.meter = m
	c.lock.Unlock()

	ingressMeter, oki := PeerIngressRegistry.Get(peerID).(metrics.Meter)
	egressMeter, oke := PeerEgressRegistry.Get(peerID).(metrics.Meter)
	traffic := func() (ingress, egress int64) {
		return 0, 0
	}
	if oki && oke {
		traffic = func() (ingress, egress int64) {
			return ingressMeter.Count(), egressMeter.Count()
		}
	} else {
		log.Warn("Failed to get traffic meter", "peerID", peerID)
	}
	PeerTrafficMeters.handshakeDone(c.id, peerID, traffic)
}
