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
	"strings"

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

	MetricsRegistryIngressPrefix = MetricsInboundTraffic + "/"
	MetricsRegistryEgressPrefix  = MetricsOutboundTraffic + "/"

	MeteredPeerLimit = 1024
)

var (
	ingressConnectMeter = metrics.NewRegisteredMeter(MetricsInboundConnects, nil)  // meter counting the ingress connections
	ingressTrafficMeter = metrics.NewRegisteredMeter(MetricsInboundTraffic, nil)   // meter metering the cumulative ingress traffic
	egressConnectMeter  = metrics.NewRegisteredMeter(MetricsOutboundConnects, nil) // meter counting the egress connections
	egressTrafficMeter  = metrics.NewRegisteredMeter(MetricsOutboundTraffic, nil)  // meter metering the cumulative egress traffic

	PeerIngressRegistry = metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, MetricsRegistryIngressPrefix)
	PeerEgressRegistry  = metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, MetricsRegistryEgressPrefix)

	metricsFeed = new(peerMetricsFeed) // Peer event feed for metrics

	meteredPeerCount uint64
)

// peerMetricsFeed delivers the peer metrics to the subscribed channels.
type peerMetricsFeed struct {
	connect    event.Feed // Event feed to notify the connection and the successful handshake of a peer
	ingress    event.Feed // Event feed to notify the amount of read bytes of a peer
	egress     event.Feed // Event feed to notify the amount of written bytes of a peer
	disconnect event.Feed // Event feed to notify the disconnection of a peer
	failed     event.Feed // Event feed to notify the connection of a peer and its disconnection before the handshake

	scope event.SubscriptionScope // Facility to unsubscribe all the subscriptions at once

	quit chan chan error
}

// PeerConnectEvent contains information about the connection of a peer.
type PeerConnectEvent struct {
	IP        string
	ID        string
	Connected time.Time
}

// PeerDisconnectEvent contains information about the disconnection of a peer.
type PeerDisconnectEvent struct {
	IP           string
	ID           string
	Disconnected time.Time
}

type PeerTrafficEvent struct {
	IP     string
	ID     string
	Amount int64
}

type PeerFailedEvent struct {
	IP           string
	Connected    time.Time
	Disconnected time.Time
}

// SubscribePeerConnectEvent registers a subscription of PeerConnectEvent
func SubscribePeerConnectEvent(ch chan<- PeerConnectEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.connect.Subscribe(ch))
}

// SubscribePeerDisconnectEvent registers a subscription of PeerDisconnectEvent
func SubscribePeerDisconnectEvent(ch chan<- PeerDisconnectEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.disconnect.Subscribe(ch))
}

// SubscribePeerTrafficEvent registers a subscription of PeerTrafficEvent

func SubscribePeerIngressEvent(ch chan<- PeerTrafficEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.ingress.Subscribe(ch))
}

func SubscribePeerEgressEvent(ch chan<- PeerTrafficEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.egress.Subscribe(ch))
}

// SubscribePeerFailedEvent registers a subscription of PeerFailedEvent
func SubscribePeerFailedEvent(ch chan<- PeerFailedEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.failed.Subscribe(ch))
}

func runMetricsFeedHelper(refresh time.Duration) {
	metricsFeed.quit = make(chan chan error)
	ticker := time.NewTicker(refresh)
	defer ticker.Stop()

	// It is possible to send all of the traffic events together, but it is risky to use pointers in the events.
	trafficEventSender := func(prefix string, feed *event.Feed) func(name string, i interface{}) {
		return func(name string, i interface{}) {
			if m, ok := i.(metrics.Meter); ok {
				// Trim the common prefix and split the peer specific part in order to get the ip and the node id.
				if key := strings.Split(strings.TrimPrefix(name, prefix), "/"); len(key) == 2 {
					feed.Send(PeerTrafficEvent{
						IP:     key[0],
						ID:     key[1],
						Amount: m.Count(),
					})
				} else {
					log.Warn("Invalid peer metrics name", "name", name)
				}
			}
		}
	}
	sendIngress := trafficEventSender(MetricsRegistryIngressPrefix, &metricsFeed.ingress)
	sendEgress := trafficEventSender(MetricsRegistryEgressPrefix, &metricsFeed.egress)

	for {
		select {
		case <-ticker.C:
			PeerIngressRegistry.Each(sendIngress)
			PeerEgressRegistry.Each(sendEgress)
		case errc := <-metricsFeed.quit:
			errc <- nil
			return
		}
	}
}

// closeMetricsFeed closes all the tracked subscriptions.
func closeMetricsFeed() {
	if metricsFeed.quit != nil {
		errc := make(chan error)
		metricsFeed.quit <- errc
		<-errc
	}
	metricsFeed.scope.Close()
	PeerIngressRegistry.UnregisterAll()
	PeerEgressRegistry.UnregisterAll()
}

// meteredConn is a wrapper around a net.Conn that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	net.Conn // Network connection to wrap with metering

	connected    time.Time
	ip           string // The IP address of the peer
	id           string // The NodeID of the peer
	ingressMeter metrics.Meter
	egressMeter  metrics.Meter

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
		ip:        ip.String(),
		connected: time.Now(),
	}
}

// Read delegates a network read to the underlying connection, bumping the ingress
// traffic meter along the way.
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

// Write delegates a network write to the underlying connection, bumping the
// egress traffic meter along the way.
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

// Close closes the underlying connection.
func (c *meteredConn) Close() error {
	// Decrement the metered peer count
	atomic.AddUint64(&meteredPeerCount, ^uint64(0))
	err, now := c.Conn.Close(), time.Now()

	c.lock.RLock()
	ip, id := c.ip, c.id
	c.lock.RUnlock()

	// If the peer disconnects before the handshake
	if id == "" {
		metricsFeed.failed.Send(PeerFailedEvent{
			IP:           ip,
			Connected:    c.connected,
			Disconnected: now,
		})
		return err
	}
	c.lock.RLock()
	//ingress, egress := c.ingressMeter.Count(), c.egressMeter.Count()
	c.lock.RUnlock()

	// Unregister the peer from the metrics registry
	key := fmt.Sprintf("%s/%s", ip, id)
	PeerIngressRegistry.Unregister(key)
	PeerEgressRegistry.Unregister(key)

	//metricsFeed.ingress.Send(PeerTrafficEvent{
	//	IP:     ip,
	//	ID:     id,
	//	Amount: ingress,
	//})
	//metricsFeed.egress.Send(PeerTrafficEvent{
	//	IP:     ip,
	//	ID:     id,
	//	Amount: egress,
	//})
	metricsFeed.disconnect.Send(PeerDisconnectEvent{
		IP:           ip,
		ID:           id,
		Disconnected: now,
	})
	return err
}

// handshakeDone changes the default id to the peer's node id.
func (c *meteredConn) handshakeDone(id discover.NodeID) {
	c.lock.Lock()
	c.id = id.String()
	key := fmt.Sprintf("%s/%s", c.ip, c.id)
	c.ingressMeter = metrics.NewRegisteredMeter(key, PeerIngressRegistry)
	c.egressMeter = metrics.NewRegisteredMeter(key, PeerEgressRegistry)
	c.lock.Unlock()

	metricsFeed.connect.Send(PeerConnectEvent{
		IP:        c.ip,
		ID:        id.String(),
		Connected: c.connected,
	})
}
