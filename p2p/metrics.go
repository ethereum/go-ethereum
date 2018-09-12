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

	meteredPeerAutoID uint64 // Used to create unique id for the metered connection before the handshake
	meteredPeerCount  uint64
)

// peerMetricsFeed delivers the peer metrics to the subscribed channels.
type peerMetricsFeed struct {
	connect    event.Feed // Event feed to notify the connection of a peer
	handshake  event.Feed // Event feed to notify the handshake with a peer
	disconnect event.Feed // Event feed to notify the disconnection of a peer
	read       event.Feed // Event feed to notify the amount of read bytes of a peer
	write      event.Feed // Event feed to notify the amount of written bytes of a peer

	scope event.SubscriptionScope // Facility to unsubscribe all the subscriptions at once

	quit chan chan error
}

// PeerConnectEvent contains information about the connection of a peer.
type PeerConnectEvent struct {
	Key       string
	Connected time.Time
}

// PeerHandshakeEvent contains information about the handshake with a peer.
type PeerHandshakeEvent struct {
	AutoKey   string
	Key       string
	Ingress   int64
	Egress    int64
	Handshake time.Time
}

// PeerDisconnectEvent contains information about the disconnection of a peer.
type PeerDisconnectEvent struct {
	Key          string
	Ingress      int64
	Egress       int64
	Disconnected time.Time
}

// PeerReadEvent contains information about the read operation of a peer.
type PeerTrafficEvent map[string]int64

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
func SubscribePeerReadEvent(ch chan<- PeerTrafficEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.read.Subscribe(ch))
}

// SubscribePeerWriteEvent registers a subscription of PeerWriteEvent
func SubscribePeerWriteEvent(ch chan<- PeerTrafficEvent) event.Subscription {
	return metricsFeed.scope.Track(metricsFeed.write.Subscribe(ch))
}

func startTrafficNotifier(refresh time.Duration) {
	metricsFeed.quit = make(chan chan error)
	ticker := time.NewTicker(refresh)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			// send read and write
			ingressEvents, egressEvents := make(PeerTrafficEvent), make(PeerTrafficEvent)
			PeerIngressRegistry.Each(func(name string, i interface{}) {
				if m, ok := i.(metrics.Meter); ok {
					ingressEvents[strings.TrimPrefix(name, MetricsRegistryIngressPrefix)] = m.Count()
				}
			})
			PeerEgressRegistry.Each(func(name string, i interface{}) {
				if m, ok := i.(metrics.Meter); ok {
					egressEvents[strings.TrimPrefix(name, MetricsRegistryEgressPrefix)] = m.Count()
				}
			})
			metricsFeed.read.Send(ingressEvents)
			metricsFeed.write.Send(egressEvents)
			//fmt.Println(ingressEvents)
			//fmt.Println(egressEvents)
			//fmt.Println()
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
	net.Conn        // Network connection to wrap with metering
	ip       string // The IP address of the peer

	key          string
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
	atomic.AddUint64(&meteredPeerCount, 1)
	// Otherwise bump the connection counters and wrap the connection
	if ingress {
		ingressConnectMeter.Mark(1)
	} else {
		egressConnectMeter.Mark(1)
	}
	key := fmt.Sprintf("%s/%s", ip.String(), fmt.Sprintf("peer_%d", atomic.AddUint64(&meteredPeerAutoID, 1)))
	metricsFeed.connect.Send(PeerConnectEvent{
		Key:       key,
		Connected: time.Now(),
	})
	return &meteredConn{
		Conn:         conn,
		key:          key,
		ip:           ip.String(),
		ingressMeter: metrics.NewRegisteredMeter(key, PeerIngressRegistry),
		egressMeter:  metrics.NewRegisteredMeter(key, PeerEgressRegistry),
	}
}

// Read delegates a network read to the underlying connection, bumping the ingress
// traffic meter along the way.
func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	ingressTrafficMeter.Mark(int64(n))
	c.ingressMeter.Mark(int64(n))
	return n, err
}

// Write delegates a network write to the underlying connection, bumping the
// egress traffic meter along the way.
func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	egressTrafficMeter.Mark(int64(n))
	c.egressMeter.Mark(int64(n))
	return n, err
}

// Close closes the underlying connection.
func (c *meteredConn) Close() error {
	// Decrement the metered peer count.
	atomic.AddUint64(&meteredPeerCount, ^uint64(0))
	c.ingressMeter.Stop()
	c.egressMeter.Stop()
	c.lock.RLock()
	key := c.key
	metricsFeed.disconnect.Send(PeerDisconnectEvent{
		Key:          key,
		Ingress:      c.ingressMeter.Count(),
		Egress:       c.egressMeter.Count(),
		Disconnected: time.Now(),
	})
	c.lock.RUnlock()
	PeerIngressRegistry.Unregister(key)
	PeerEgressRegistry.Unregister(key)
	return c.Conn.Close()
}

// handshakeDone changes the default id to the peer's node id.
func (c *meteredConn) handshakeDone(id discover.NodeID) {
	c.ingressMeter.Stop()
	c.egressMeter.Stop()
	c.lock.Lock()

	autoKey := c.key
	key := fmt.Sprintf("%s/%s", c.ip, id.String())
	ingressMeter := metrics.NewRegisteredMeter(key, PeerIngressRegistry)
	egressMeter := metrics.NewRegisteredMeter(key, PeerEgressRegistry)
	ingressMeter.Mark(c.ingressMeter.Count())
	egressMeter.Mark(c.egressMeter.Count())
	PeerIngressRegistry.Unregister(c.key)
	PeerEgressRegistry.Unregister(c.key)
	c.key = key
	c.ingressMeter = ingressMeter
	c.egressMeter = egressMeter

	c.lock.Unlock()

	metricsFeed.handshake.Send(PeerHandshakeEvent{
		AutoKey:   autoKey,
		Key:       key,
		//Ingress: nil,
		//Egress: nil,
		Handshake: time.Now(),
	})
}
