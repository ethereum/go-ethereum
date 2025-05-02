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

	"github.com/ethereum/go-ethereum/metrics"
)

const (
	// HandleHistName is the prefix of the per-packet serving time histograms.
	HandleHistName = "p2p/handle"

	// ingressMeterName is the prefix of the per-packet inbound metrics.
	ingressMeterName = "p2p/ingress"

	// egressMeterName is the prefix of the per-packet outbound metrics.
	egressMeterName = "p2p/egress"
)

var (
	activePeerGauge         = metrics.NewRegisteredGauge("p2p/peers", nil)
	activeInboundPeerGauge  = metrics.NewRegisteredGauge("p2p/peers/inbound", nil)
	activeOutboundPeerGauge = metrics.NewRegisteredGauge("p2p/peers/outbound", nil)

	ingressTrafficMeter = metrics.NewRegisteredMeter("p2p/ingress", nil)
	egressTrafficMeter  = metrics.NewRegisteredMeter("p2p/egress", nil)

	// general ingress/egress connection meters
	serveMeter          = metrics.NewRegisteredMeter("p2p/serves", nil)
	serveSuccessMeter   = metrics.NewRegisteredMeter("p2p/serves/success", nil)
	dialMeter           = metrics.NewRegisteredMeter("p2p/dials", nil)
	dialSuccessMeter    = metrics.NewRegisteredMeter("p2p/dials/success", nil)
	dialConnectionError = metrics.NewRegisteredMeter("p2p/dials/error/connection", nil) // dial timeout; no route to host; connection refused; network is unreachable

	// count peers that stayed connected for at least 1 min
	serve1MinSuccessMeter = metrics.NewRegisteredMeter("p2p/serves/success/1min", nil)
	dial1MinSuccessMeter  = metrics.NewRegisteredMeter("p2p/dials/success/1min", nil)

	// handshake error meters
	dialTooManyPeers        = metrics.NewRegisteredMeter("p2p/dials/error/saturated", nil)
	dialAlreadyConnected    = metrics.NewRegisteredMeter("p2p/dials/error/known", nil)
	dialSelf                = metrics.NewRegisteredMeter("p2p/dials/error/self", nil)
	dialUselessPeer         = metrics.NewRegisteredMeter("p2p/dials/error/useless", nil)
	dialUnexpectedIdentity  = metrics.NewRegisteredMeter("p2p/dials/error/id/unexpected", nil)
	dialEncHandshakeError   = metrics.NewRegisteredMeter("p2p/dials/error/rlpx/enc", nil)   // EOF; connection reset during handshake; message too big; i/o timeout
	dialProtoHandshakeError = metrics.NewRegisteredMeter("p2p/dials/error/rlpx/proto", nil) // EOF

	// capture the rest of errors that are not handled by the above meters
	dialOtherError = metrics.NewRegisteredMeter("p2p/dials/error/other", nil)
)

// markDialError matches errors that occur while setting up a dial connection to the
// corresponding meter. We don't maintain meters for evert possible error, just for
// the most interesting ones.
func markDialError(err error) {
	if !metrics.Enabled() {
		return
	}

	var reason DiscReason
	var handshakeErr *protoHandshakeError
	d := errors.As(err, &reason)
	switch {
	case d && reason == DiscTooManyPeers:
		dialTooManyPeers.Mark(1)
	case d && reason == DiscAlreadyConnected:
		dialAlreadyConnected.Mark(1)
	case d && reason == DiscSelf:
		dialSelf.Mark(1)
	case d && reason == DiscUselessPeer:
		dialUselessPeer.Mark(1)
	case d && reason == DiscUnexpectedIdentity:
		dialUnexpectedIdentity.Mark(1)
	case errors.As(err, &handshakeErr):
		dialProtoHandshakeError.Mark(1)
	case errors.Is(err, errEncHandshakeError):
		dialEncHandshakeError.Mark(1)
	default:
		dialOtherError.Mark(1)
	}
}

// meteredConn is a wrapper around a net.Conn that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	net.Conn
}

// newMeteredConn creates a new metered connection, bumps the ingress or egress
// connection meter and also increases the metered peer count. If the metrics
// system is disabled, function returns the original connection.
func newMeteredConn(conn net.Conn) net.Conn {
	if !metrics.Enabled() {
		return conn
	}
	return &meteredConn{Conn: conn}
}

// Read delegates a network read to the underlying connection, bumping the common
// and the peer ingress traffic meters along the way.
func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	ingressTrafficMeter.Mark(int64(n))
	return n, err
}

// Write delegates a network write to the underlying connection, bumping the common
// and the peer egress traffic meters along the way.
func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	egressTrafficMeter.Mark(int64(n))
	return n, err
}
