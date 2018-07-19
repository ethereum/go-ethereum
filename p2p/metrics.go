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
	"github.com/ethereum/go-ethereum/metrics"
)

var (
	ingressConnectMeter  = metrics.NewRegisteredMeter("p2p/InboundConnects", nil)
	ingressTrafficMeter  = metrics.NewRegisteredMeter("p2p/InboundTraffic", nil)
	egressConnectMeter   = metrics.NewRegisteredMeter("p2p/OutboundConnects", nil)
	egressTrafficMeter   = metrics.NewRegisteredMeter("p2p/OutboundTraffic", nil)
	IngressTrafficMeters = make(map[string]metrics.Meter)
	EgressTrafficMeters  = make(map[string]metrics.Meter)
)

// meteredConn is a wrapper around a net.Conn that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	net.Conn        // Network connection to wrap with metering
	id              string
	unmarkedIngress uint
	unmarkedEgress  uint
}

// newMeteredConn creates a new metered connection, also bumping the ingress or
// egress connection meter. If the metrics system is disabled, this function
// returns the original object.
func newMeteredConn(conn net.Conn, ingress bool) net.Conn {
	// Short circuit if metrics are disabled
	if !metrics.Enabled {
		return conn
	}
	// Otherwise bump the connection counters and wrap the connection
	if ingress {
		ingressConnectMeter.Mark(1)
	} else {
		egressConnectMeter.Mark(1)
	}
	return &meteredConn{Conn: conn}
}

// Read delegates a network read to the underlying connection, bumping the ingress
// traffic meter along the way.
func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	ingressTrafficMeter.Mark(int64(n))
	if rm, ok := IngressTrafficMeters[c.id]; ok {
		rm.Mark(int64(n))
		return n, err
	}
	c.unmarkedIngress += uint(n)
	return n, err
}

// Write delegates a network write to the underlying connection, bumping the
// egress traffic meter along the way.
func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	egressTrafficMeter.Mark(int64(n))
	if rm, ok := EgressTrafficMeters[c.id]; ok {
		rm.Mark(int64(n))
		return n, err
	}
	c.unmarkedEgress += uint(n)
	return n, err
}

func (c *meteredConn) meterIndividually(id string) {
	c.id = id
	IngressTrafficMeters[id] = metrics.NewRegisteredMeter(fmt.Sprintf("p2p/InboundTraffic/%s", id), nil)
	EgressTrafficMeters[id] = metrics.NewRegisteredMeter(fmt.Sprintf("p2p/OutboundTraffic/%s", id), nil)
	IngressTrafficMeters[id].Mark(int64(c.unmarkedIngress))
	EgressTrafficMeters[id].Mark(int64(c.unmarkedEgress))
	c.unmarkedIngress, c.unmarkedEgress = 0, 0
}

func (c *meteredConn) Close() error {
	delete(IngressTrafficMeters, c.id)
	delete(EgressTrafficMeters, c.id)
	metrics.Unregister(fmt.Sprintf("p2p/InboundTraffic/%s", c.id))
	metrics.Unregister(fmt.Sprintf("p2p/OutboundTraffic/%s", c.id))
	fmt.Println("Close", c.id)
	return c.Conn.Close()
}
