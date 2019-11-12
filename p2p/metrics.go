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
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/maticnetwork/bor/event"
	"github.com/maticnetwork/bor/log"
	"github.com/maticnetwork/bor/metrics"
	"github.com/maticnetwork/bor/p2p/enode"
)

const (
	MetricsInboundTraffic   = "p2p/ingress" // Name for the registered inbound traffic meter
	MetricsOutboundTraffic  = "p2p/egress"  // Name for the registered outbound traffic meter
	MetricsOutboundConnects = "p2p/dials"   // Name for the registered outbound connects meter
	MetricsInboundConnects  = "p2p/serves"  // Name for the registered inbound connects meter

	MeteredPeerLimit = 1024 // This amount of peers are individually metered
)

var (
	ingressConnectMeter = metrics.NewRegisteredMeter(MetricsInboundConnects, nil)  // Meter counting the ingress connections
	ingressTrafficMeter = metrics.NewRegisteredMeter(MetricsInboundTraffic, nil)   // Meter metering the cumulative ingress traffic
	egressConnectMeter  = metrics.NewRegisteredMeter(MetricsOutboundConnects, nil) // Meter counting the egress connections
	egressTrafficMeter  = metrics.NewRegisteredMeter(MetricsOutboundTraffic, nil)  // Meter metering the cumulative egress traffic
	activePeerCounter   = metrics.NewRegisteredCounter("p2p/peers", nil)           // Gauge tracking the current peer count

	PeerIngressRegistry = metrics.NewPrefixedChildRegistry(metrics.EphemeralRegistry, MetricsInboundTraffic+"/")  // Registry containing the peer ingress
	PeerEgressRegistry  = metrics.NewPrefixedChildRegistry(metrics.EphemeralRegistry, MetricsOutboundTraffic+"/") // Registry containing the peer egress

	meteredPeerFeed  event.Feed // Event feed for peer metrics
	meteredPeerCount int32      // Actually stored peer connection count
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

// MeteredPeerEvent is an event emitted when peers connect or disconnect.
type MeteredPeerEvent struct {
	Type    MeteredPeerEventType // Type of peer event
	IP      net.IP               // IP address of the peer
	ID      enode.ID             // NodeID of the peer
	Elapsed time.Duration        // Time elapsed between the connection and the handshake/disconnection
	Ingress uint64               // Ingress count at the moment of the event
	Egress  uint64               // Egress count at the moment of the event
}

// SubscribeMeteredPeerEvent registers a subscription for peer life-cycle events
// if metrics collection is enabled.
func SubscribeMeteredPeerEvent(ch chan<- MeteredPeerEvent) event.Subscription {
	return meteredPeerFeed.Subscribe(ch)
}

// meteredConn is a wrapper around a net.Conn that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	net.Conn // Network connection to wrap with metering

	connected time.Time // Connection time of the peer
	ip        net.IP    // IP address of the peer
	id        enode.ID  // NodeID of the peer

	// trafficMetered denotes if the peer is registered in the traffic registries.
	// Its value is true if the metered peer count doesn't reach the limit in the
	// moment of the peer's connection.
	trafficMetered bool
	ingressMeter   metrics.Meter // Meter for the read bytes of the peer
	egressMeter    metrics.Meter // Meter for the written bytes of the peer

	lock sync.RWMutex // Lock protecting the metered connection's internals
}

// newMeteredConn creates a new metered connection, bumps the ingress or egress
// connection meter and also increases the metered peer count. If the metrics
// system is disabled or the IP address is unspecified, this function returns
// the original object.
func newMeteredConn(conn net.Conn, ingress bool, ip net.IP) net.Conn {
	// Short circuit if metrics are disabled
	if !metrics.Enabled {
		return conn
	}
	if ip.IsUnspecified() {
		log.Warn("Peer IP is unspecified")
		return conn
	}
	// Bump the connection counters and wrap the connection
	if ingress {
		ingressConnectMeter.Mark(1)
	} else {
		egressConnectMeter.Mark(1)
	}
	activePeerCounter.Inc(1)

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
	if c.trafficMetered {
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
	if c.trafficMetered {
		c.egressMeter.Mark(int64(n))
	}
	c.lock.RUnlock()
	return n, err
}

// handshakeDone is called when a peer handshake is done. Registers the peer to
// the ingress and the egress traffic registries using the peer's IP and node ID,
// also emits connect event.
func (c *meteredConn) handshakeDone(id enode.ID) {
	// TODO (kurkomisi): use the node URL instead of the pure node ID. (the String() method of *Node)
	if atomic.AddInt32(&meteredPeerCount, 1) >= MeteredPeerLimit {
		// Don't register the peer in the traffic registries.
		atomic.AddInt32(&meteredPeerCount, -1)
		c.lock.Lock()
		c.id, c.trafficMetered = id, false
		c.lock.Unlock()
		log.Warn("Metered peer count reached the limit")
	} else {
		key := fmt.Sprintf("%s/%s", c.ip, id.String())
		c.lock.Lock()
		c.id, c.trafficMetered = id, true
		c.ingressMeter = metrics.NewRegisteredMeter(key, PeerIngressRegistry)
		c.egressMeter = metrics.NewRegisteredMeter(key, PeerEgressRegistry)
		c.lock.Unlock()
	}
	meteredPeerFeed.Send(MeteredPeerEvent{
		Type:    PeerConnected,
		IP:      c.ip,
		ID:      id,
		Elapsed: time.Since(c.connected),
	})
}

// Close delegates a close operation to the underlying connection, unregisters
// the peer from the traffic registries and emits close event.
func (c *meteredConn) Close() error {
	err := c.Conn.Close()
	c.lock.RLock()
	if c.id == (enode.ID{}) {
		// If the peer disconnects before the handshake.
		c.lock.RUnlock()
		meteredPeerFeed.Send(MeteredPeerEvent{
			Type:    PeerHandshakeFailed,
			IP:      c.ip,
			Elapsed: time.Since(c.connected),
		})
		activePeerCounter.Dec(1)
		return err
	}
	id := c.id
	if !c.trafficMetered {
		// If the peer isn't registered in the traffic registries.
		c.lock.RUnlock()
		meteredPeerFeed.Send(MeteredPeerEvent{
			Type: PeerDisconnected,
			IP:   c.ip,
			ID:   id,
		})
		activePeerCounter.Dec(1)
		return err
	}
	ingress, egress := uint64(c.ingressMeter.Count()), uint64(c.egressMeter.Count())
	c.lock.RUnlock()

	// Decrement the metered peer count
	atomic.AddInt32(&meteredPeerCount, -1)

	// Unregister the peer from the traffic registries
	key := fmt.Sprintf("%s/%s", c.ip, id)
	PeerIngressRegistry.Unregister(key)
	PeerEgressRegistry.Unregister(key)

	meteredPeerFeed.Send(MeteredPeerEvent{
		Type:    PeerDisconnected,
		IP:      c.ip,
		ID:      id,
		Ingress: ingress,
		Egress:  egress,
	})
	activePeerCounter.Dec(1)
	return err
}
