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
	"fmt"
	"github.com/ethereum/go-ethereum/p2p/enode"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
)

const (
	MetricsInboundConnects  = "p2p/InboundConnects"  // Name for the registered inbound connects meter
	MetricsInboundTraffic   = "p2p/InboundTraffic"   // Name for the registered inbound traffic meter
	MetricsOutboundConnects = "p2p/OutboundConnects" // Name for the registered outbound connects meter
	MetricsOutboundTraffic  = "p2p/OutboundTraffic"  // Name for the registered outbound traffic meter

	MetricsRegistryIngressPrefix = MetricsInboundTraffic + "/"
	MetricsRegistryEgressPrefix  = MetricsOutboundTraffic + "/"

	MeteredPeerLimit = 1024
)

var (
	ingressConnectMeter = metrics.NewRegisteredMeter(MetricsInboundConnects, nil)  // Meter counting the ingress connections
	ingressTrafficMeter = metrics.NewRegisteredMeter(MetricsInboundTraffic, nil)   // Meter metering the cumulative ingress traffic
	egressConnectMeter  = metrics.NewRegisteredMeter(MetricsOutboundConnects, nil) // Meter counting the egress connections
	egressTrafficMeter  = metrics.NewRegisteredMeter(MetricsOutboundTraffic, nil)  // Meter metering the cumulative egress traffic

	PeerIngressRegistry = metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, MetricsRegistryIngressPrefix) // Registry containing the peer ingress
	PeerEgressRegistry  = metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, MetricsRegistryEgressPrefix)  // Registry containing the peer egress

	metricsFeed      event.Feed // Event feed for peer metrics
	meteredPeerCount uint64     // Actually stored peer connection count
)

// MeteredPeerEventType is the type of peer events emitted by a metered connection.
type MeteredPeerEventType int

const (
	// PeerConnected is the type of event emitted when a peer successfully
	// made the handshake.
	PeerConnected MeteredPeerEventType = iota

	// PeerDisconnected is the type of event emitted when a peer disconnects.
	PeerDisconnected

	// PeerHandshakeFailed is the type of event emitted when a peer fails to
	// make the handshake or disconnects before the handshake.
	PeerHandshakeFailed
)

// MeteredPeerEvent is an event emitted when peers connect or disconnect
type MeteredPeerEvent struct {
	Type    MeteredPeerEventType // Type of peer event
	IP      net.IP               // IP address of the peer
	ID      string               // NodeID of the peer
	Elapsed time.Duration        // Time elapsed between the connection and the handshake/disconnection
	Ingress uint64               // Ingress count in the moment of disconnection
	Egress  uint64               // Egress count in the moment of disconnection
}

// SubscribeMeteredPeerEvent registers a subscription of MeteredPeerEvent
func SubscribeMeteredPeerEvent(ch chan<- MeteredPeerEvent) event.Subscription {
	return metricsFeed.Subscribe(ch)
}

// meteredConn is a wrapper around a net.Conn that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	net.Conn // Network connection to wrap with metering

	connected    time.Time     // Connection time of the peer
	ip           net.IP        // IP address of the peer
	id           string        // NodeID of the peer
	ingressMeter metrics.Meter // Meter for the read bytes of the peer
	egressMeter  metrics.Meter // Meter for the written bytes of the peer

	lock sync.RWMutex // Lock protecting the metered connection's internals
}

// newMeteredConn creates a new metered connection, bumps the ingress or egress
// connection meter and also increases the metered peer count. If the metrics
// system is disabled, the IP address is unspecified or the metered peer count
// reached the limit, this function returns the original object.
func newMeteredConn(conn net.Conn, ingress bool, ip net.IP) net.Conn {
	// Short circuit if metrics are disabled
	if !metrics.Enabled {
		return conn
	}
	if ip.IsUnspecified() {
		log.Warn("Peer IP is unspecified")
		return conn
	}
	if atomic.LoadUint64(&meteredPeerCount) >= MeteredPeerLimit {
		log.Warn("Metered peer count reached the limit")
		return conn
	}
	// Increment the metered peer count
	atomic.AddUint64(&meteredPeerCount, 1)
	// Bump the connection counters and wrap the connection
	if ingress {
		ingressConnectMeter.Mark(1)
	} else {
		egressConnectMeter.Mark(1)
	}
	return &meteredConn{
		Conn:      conn,
		ip:        ip,
		connected: time.Now(),
	}
}

// Read delegates a network read to the underlying connection, bumping the common
// and the peer ingress traffic meters along the way.
func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	ingressTrafficMeter.Mark(int64(n))
	c.lock.RLock()
	if c.ingressMeter != nil {
		c.ingressMeter.Mark(int64(n))
	}
	c.lock.RUnlock()
	return n, err
}

// Write delegates a network write to the underlying connection, bumping the common
// and the peer egress traffic meters along the way.
func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	egressTrafficMeter.Mark(int64(n))
	c.lock.RLock()
	if c.egressMeter != nil {
		c.egressMeter.Mark(int64(n))
	}
	c.lock.RUnlock()
	return n, err
}

// handshakeDone is called when a peer handshake is done. Registers the peer to
// the ingress and the egress traffic registries using the peer's IP and NodeID,
// also emits connect event.
func (c *meteredConn) handshakeDone(id enode.ID) {
	c.lock.Lock()
	c.id = id.String()
	key := fmt.Sprintf("%s/%s", c.ip, c.id)
	c.ingressMeter = metrics.NewRegisteredMeter(key, PeerIngressRegistry)
	c.egressMeter = metrics.NewRegisteredMeter(key, PeerEgressRegistry)
	c.lock.Unlock()

	metricsFeed.Send(MeteredPeerEvent{
		Type:    PeerConnected,
		IP:      c.ip,
		ID:      id.String(),
		Elapsed: time.Now().Sub(c.connected),
	})
}

// Close delegates a close operation to the underlying connection, unregisters
// the peer from the traffic registries and emits close event.
func (c *meteredConn) Close() error {
	// Decrement the metered peer count
	atomic.AddUint64(&meteredPeerCount, ^uint64(0))

	c.lock.RLock()
	// If the peer disconnects before the handshake
	if c.id == "" {
		c.lock.RUnlock()
		metricsFeed.Send(MeteredPeerEvent{
			Type:    PeerHandshakeFailed,
			IP:      c.ip,
			Elapsed: time.Now().Sub(c.connected),
		})
		return c.Conn.Close()
	}
	id, ingress, egress := c.id, uint64(c.ingressMeter.Count()), uint64(c.egressMeter.Count())
	c.lock.RUnlock()

	// Unregister the peer from the traffic registries
	key := fmt.Sprintf("%s/%s", c.ip, id)
	PeerIngressRegistry.Unregister(key)
	PeerEgressRegistry.Unregister(key)

	metricsFeed.Send(MeteredPeerEvent{
		Type:    PeerDisconnected,
		IP:      c.ip,
		ID:      id,
		Ingress: ingress,
		Egress:  egress,
	})
	return c.Conn.Close()
}
