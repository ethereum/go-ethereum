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

	"github.com/ethereum/go-ethereum/event"
		"github.com/ethereum/go-ethereum/metrics"
		"sync"
		"time"
	"github.com/ethereum/go-ethereum/log"
	"sync/atomic"
	"fmt"
)

const (
	MetricsInboundTraffic   = "p2p/InboundTraffic"
	MetricsInboundConnects  = "p2p/InboundConnects"
	MetricsOutboundTraffic  = "p2p/OutboundTraffic"
	MetricsOutboundConnects = "p2p/OutboundConnects"

	MeteredPeerLimit = 16384
)

var (
	ingressConnectMeter = metrics.NewRegisteredMeter(MetricsInboundConnects, nil)
	ingressTrafficMeter = metrics.NewRegisteredMeter(MetricsInboundTraffic, nil)
	egressConnectMeter  = metrics.NewRegisteredMeter(MetricsOutboundConnects, nil)
	egressTrafficMeter  = metrics.NewRegisteredMeter(MetricsOutboundTraffic, nil)

	NME = &networkMeterEvents{}
)

type networkMeterEvents struct {
	connectFeed    event.Feed
	handshakeFeed  event.Feed
	disconnectFeed event.Feed

	readFeed  event.Feed
	writeFeed event.Feed

	scope event.SubscriptionScope

	defaultID uint64
}

type PeerConnectEvent struct {
	IP        net.IP
	ID        string
	Connected time.Time
}

type PeerHandshakeEvent struct {
	IP        net.IP
	DefaultID string
	ID        string
	Handshake time.Time
}

type PeerDisconnectEvent struct {
	IP           net.IP
	ID           string
	Disconnected time.Time
}

type PeerReadEvent struct {
	IP      net.IP
	ID      string
	Ingress int
}

type PeerWriteEvent struct {
	IP     net.IP
	ID     string
	Egress int
}

func SubscribePeerConnectEvent(ch chan<- PeerConnectEvent) event.Subscription {
	return NME.scope.Track(NME.connectFeed.Subscribe(ch))
}
func SubscribePeerHandshakeEvent(ch chan<- PeerHandshakeEvent) event.Subscription {
	return NME.scope.Track(NME.handshakeFeed.Subscribe(ch))
}
func SubscribePeerDisconnectEvent(ch chan<- PeerDisconnectEvent) event.Subscription {
	return NME.scope.Track(NME.disconnectFeed.Subscribe(ch))
}
func SubscribePeerReadEvent(ch chan<- PeerReadEvent) event.Subscription {
	return NME.scope.Track(NME.readFeed.Subscribe(ch))
}
func SubscribePeerWriteEvent(ch chan<- PeerWriteEvent) event.Subscription {
	return NME.scope.Track(NME.writeFeed.Subscribe(ch))
}

func closeNME() {
	NME.scope.Close()
}

// meteredConn is a wrapper around a net.Conn that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	net.Conn // Network connection to wrap with metering
	ip net.IP
	id string

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
	id := fmt.Sprintf("peer_%d", atomic.AddUint64(&NME.defaultID, 1))
	NME.connectFeed.Send(PeerConnectEvent{
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
	NME.readFeed.Send(PeerReadEvent{
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
	NME.writeFeed.Send(PeerWriteEvent{
		IP:     c.ip,
		ID:     id,
		Egress: n,
	})
	return n, err
}

func (c *meteredConn) Close() error {
	c.lock.RLock()
	id := c.id
	c.lock.RUnlock()
	NME.disconnectFeed.Send(PeerDisconnectEvent{
		IP:           c.ip,
		ID:           id,
		Disconnected: time.Now(),
	})
	return c.Conn.Close()
}

func (c *meteredConn) handshakeDone(peerID string) {
	c.lock.Lock()
	defaultID := c.id
	c.id = peerID
	c.lock.Unlock()
	NME.handshakeFeed.Send(PeerHandshakeEvent{
		IP:        c.ip,
		DefaultID: defaultID,
		ID:        peerID,
		Handshake: time.Now(),
	})
}
