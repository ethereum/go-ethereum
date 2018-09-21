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

	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

const (
	MetricsInboundConnects  = "p2p/InboundConnects"  // Name for the registered inbound connects meter
	MetricsInboundTraffic   = "p2p/InboundTraffic"   // Name for the registered inbound traffic meter
	MetricsOutboundConnects = "p2p/OutboundConnects" // Name for the registered outbound connects meter
	MetricsOutboundTraffic  = "p2p/OutboundTraffic"  // Name for the registered outbound traffic meter
)

var (
	ingressConnectMeter = metrics.NewRegisteredMeter(MetricsInboundConnects, nil)  // meter counting the ingress connections
	ingressTrafficMeter = metrics.NewRegisteredMeter(MetricsInboundTraffic, nil)   // meter metering the cumulative ingress traffic
	egressConnectMeter  = metrics.NewRegisteredMeter(MetricsOutboundConnects, nil) // meter counting the egress connections
	egressTrafficMeter  = metrics.NewRegisteredMeter(MetricsOutboundTraffic, nil)  // meter metering the cumulative egress traffic

	metricsFeed = new(peerMetricsFeed) // Peer event feed for metrics

	defaultMeteredPeerID uint64 // Used to create unique id for the metered connection before the handshake
)

// peerMetricsFeed delivers the peer metrics to the subscribed channels.
type peerMetricsFeed struct {
	connect    event.Feed // Event feed to notify the connection of a peer
	handshake  event.Feed // Event feed to notify the handshake with a peer
	disconnect event.Feed // Event feed to notify the disconnection of a peer
	read       event.Feed // Event feed to notify the amount of read bytes of a peer
	write      event.Feed // Event feed to notify the amount of written bytes of a peer

	scope event.SubscriptionScope // Facility to unsubscribe all the subscriptions at once
}

// PeerConnectEvent contains information about the connection of a peer.
type PeerConnectEvent struct {
	IP        net.IP
	ID        string
	Connected time.Time
}

// PeerHandshakeEvent contains information about the handshake with a peer.
type PeerHandshakeEvent struct {
	IP        net.IP
	DefaultID string
	ID        string
	Handshake time.Time
}

// PeerDisconnectEvent contains information about the disconnection of a peer.
type PeerDisconnectEvent struct {
	IP           net.IP
	ID           string
	Disconnected time.Time
}

// PeerReadEvent contains information about the read operation of a peer.
type PeerReadEvent struct {
	IP      net.IP
	ID      string
	Ingress int
}

// PeerWriteEvent contains information about the write operation of a peer.
type PeerWriteEvent struct {
	IP     net.IP
	ID     string
	Egress int
}

// SubscribePeerConnectEvent registers a subscription of PeerConnectEvent
func SubscribePeerConnectEvent(ch chan<- PeerConnectEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.connect.Subscribe(ch))
}

// SubscribePeerHandshakeEvent registers a subscription of PeerHandshakeEvent
func SubscribePeerHandshakeEvent(ch chan<- PeerHandshakeEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.handshake.Subscribe(ch))
}

// SubscribePeerDisconnectEvent registers a subscription of PeerDisconnectEvent
func SubscribePeerDisconnectEvent(ch chan<- PeerDisconnectEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.disconnect.Subscribe(ch))
}

// SubscribePeerReadEvent registers a subscription of PeerReadEvent
func SubscribePeerReadEvent(ch chan<- PeerReadEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.read.Subscribe(ch))
}

// SubscribePeerWriteEvent registers a subscription of PeerWriteEvent
func SubscribePeerWriteEvent(ch chan<- PeerWriteEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.write.Subscribe(ch))
}

// closeMetricsFeed closes all the tracked subscriptions.
func closeMetricsFeed() {
	metricsFeed.scope.Close()
}

// meteredConn is a wrapper around a net.Conn that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	net.Conn        // Network connection to wrap with metering
	ip       net.IP // The IP address of the peer
	id       string // The node id of the peer

	lock sync.RWMutex // Lock protecting the metered connection's internals
}

// newMeteredConn creates a new metered connection, also bumping the ingress or
// egress connection meter. If the metrics system is disabled, this function
// returns the original object.
func newMeteredConn(conn net.Conn, ingress bool, ip net.IP) net.Conn {
	// Short circuit if metrics are disabled
	if !metrics.Enabled {
		return conn
	}
	if ip.IsUnspecified() {
		log.Warn("peer IP is unspecified")
		return conn
	}
	// Otherwise bump the connection counters and wrap the connection
	if ingress {
		ingressConnectMeter.Mark(1)
	} else {
		egressConnectMeter.Mark(1)
	}
	id := fmt.Sprintf("peer_%d", atomic.AddUint64(&defaultMeteredPeerID, 1))
	metricsFeed.connect.Send(PeerConnectEvent{
		IP:        ip,
		ID:        id,
		Connected: time.Now(),
	})
	return &meteredConn{
		Conn: conn,
		ip:   ip,
		id:   id,
	}
}

// Read delegates a network read to the underlying connection, bumping the ingress
// traffic meter along the way.
func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	ingressTrafficMeter.Mark(int64(n))
	c.lock.RLock()
	id := c.id
	c.lock.RUnlock()
	metricsFeed.read.Send(PeerReadEvent{
		IP:      c.ip,
		ID:      id,
		Ingress: n,
	})
	return n, err
}

// Write delegates a network write to the underlying connection, bumping the
// egress traffic meter along the way.
func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	egressTrafficMeter.Mark(int64(n))
	c.lock.RLock()
	id := c.id
	c.lock.RUnlock()
	metricsFeed.write.Send(PeerWriteEvent{
		IP:     c.ip,
		ID:     id,
		Egress: n,
	})
	return n, err
}

// Close closes the underlying connection.
func (c *meteredConn) Close() error {
	c.lock.RLock()
	id := c.id
	c.lock.RUnlock()
	metricsFeed.disconnect.Send(PeerDisconnectEvent{
		IP:           c.ip,
		ID:           id,
		Disconnected: time.Now(),
	})
	return c.Conn.Close()
}

// handshakeDone changes the default id to the peer's node id.
func (c *meteredConn) handshakeDone(id discover.NodeID) {
	c.lock.Lock()
	defaultID := c.id
	c.id = id.String()
	c.lock.Unlock()
	metricsFeed.handshake.Send(PeerHandshakeEvent{
		IP:        c.ip,
		DefaultID: defaultID,
		ID:        id.String(),
		Handshake: time.Now(),
	})
}
