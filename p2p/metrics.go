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
	"errors"
	"net"

	"github.com/XinFinOrg/XDPoSChain/metrics"
)

var (
	MetricsInboundTraffic  = "p2p/ingress" // Name for the registered inbound traffic meter
	MetricsOutboundTraffic = "p2p/egress"  // Name for the registered outbound traffic meter
	ingressTrafficMeter    = metrics.NewRegisteredMeter("p2p/InboundTraffic", nil)
	egressTrafficMeter     = metrics.NewRegisteredMeter("p2p/OutboundTraffic", nil)
)

var (
	activePeerGauge = metrics.NewRegisteredGauge("p2p/peers", nil)

	serveMeter          = metrics.NewRegisteredMeter("p2p/serves", nil)
	serveSuccessMeter   = metrics.NewRegisteredMeter("p2p/serves/success", nil)
	dialMeter           = metrics.NewRegisteredMeter("p2p/dials", nil)
	dialSuccessMeter    = metrics.NewRegisteredMeter("p2p/dials/success", nil)
	dialConnectionError = metrics.NewRegisteredMeter("p2p/dials/error/connection", nil)

	// handshake error meters
	dialTooManyPeers        = metrics.NewRegisteredMeter("p2p/dials/error/saturated", nil)
	dialAlreadyConnected    = metrics.NewRegisteredMeter("p2p/dials/error/known", nil)
	dialSelf                = metrics.NewRegisteredMeter("p2p/dials/error/self", nil)
	dialUselessPeer         = metrics.NewRegisteredMeter("p2p/dials/error/useless", nil)
	dialUnexpectedIdentity  = metrics.NewRegisteredMeter("p2p/dials/error/id/unexpected", nil)
	dialEncHandshakeError   = metrics.NewRegisteredMeter("p2p/dials/error/rlpx/enc", nil)
	dialProtoHandshakeError = metrics.NewRegisteredMeter("p2p/dials/error/rlpx/proto", nil)
)

func markDialError(err error) {
	if !metrics.Enabled() {
		return
	}
	if err2 := errors.Unwrap(err); err2 != nil {
		err = err2
	}
	switch err {
	case DiscTooManyPeers:
		dialTooManyPeers.Mark(1)
	case DiscAlreadyConnected:
		dialAlreadyConnected.Mark(1)
	case DiscSelf:
		dialSelf.Mark(1)
	case DiscUselessPeer:
		dialUselessPeer.Mark(1)
	case DiscUnexpectedIdentity:
		dialUnexpectedIdentity.Mark(1)
	case errEncHandshakeError:
		dialEncHandshakeError.Mark(1)
	case errProtoHandshakeError:
		dialProtoHandshakeError.Mark(1)
	}
}

// meteredConn is a wrapper around a network TCP connection that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	*net.TCPConn // Network connection to wrap with metering
}

// newMeteredConn creates a new metered connection, also bumping the ingress or
// egress connection meter. If the metrics system is disabled, this function
// returns the original object.
func newMeteredConn(conn net.Conn) net.Conn {
	// Short circuit if metrics are disabled
	if !metrics.Enabled() {
		return conn
	}
	return &meteredConn{conn.(*net.TCPConn)}
}

// Read delegates a network read to the underlying connection, bumping the ingress
// traffic meter along the way.
func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.TCPConn.Read(b)
	ingressTrafficMeter.Mark(int64(n))
	return
}

// Write delegates a network write to the underlying connection, bumping the
// egress traffic meter along the way.
func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.TCPConn.Write(b)
	egressTrafficMeter.Mark(int64(n))
	return
}
