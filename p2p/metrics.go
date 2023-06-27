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
	ingressTrafficMeter = metrics.NewRegisteredMeter("p2p/ingress", nil)
	egressTrafficMeter  = metrics.NewRegisteredMeter("p2p/egress", nil)

	ingressDialMeter        metrics.Meter = metrics.NilMeter{}
	ingressDialSuccessMeter metrics.Meter = metrics.NilMeter{}
	egressDialAttemptMeter  metrics.Meter = metrics.NilMeter{}
	egressDialSuccessMeter  metrics.Meter = metrics.NilMeter{}
	activePeerGauge         metrics.Gauge = metrics.NilGauge{}

	dialConnectionError          metrics.Meter = metrics.NilMeter{}
	dialTooManyPeers             metrics.Meter = metrics.NilMeter{}
	dialAlreadyConnected         metrics.Meter = metrics.NilMeter{}
	dialSelf                     metrics.Meter = metrics.NilMeter{}
	dialUselessPeer              metrics.Meter = metrics.NilMeter{}
	dialNoSecp256k1Key           metrics.Meter = metrics.NilMeter{}
	dialFailedRLPXEncHandshake   metrics.Meter = metrics.NilMeter{}
	dialFailedRLPXProtoHandshake metrics.Meter = metrics.NilMeter{}
)

func init() {
	if !metrics.Enabled {
		return
	}

	ingressDialMeter = metrics.NewRegisteredMeter("p2p/serves", nil)
	ingressDialSuccessMeter = metrics.NewRegisteredMeter("p2p/serves/success", nil)
	egressDialAttemptMeter = metrics.NewRegisteredMeter("p2p/dials", nil)
	egressDialSuccessMeter = metrics.NewRegisteredMeter("p2p/dials/success", nil)
	activePeerGauge = metrics.NewRegisteredGauge("p2p/peers", nil)

	dialConnectionError = metrics.NewRegisteredMeter("p2p/dials/error/connection", nil)
	dialTooManyPeers = metrics.NewRegisteredMeter("p2p/dials/error/saturated", nil)
	dialAlreadyConnected = metrics.NewRegisteredMeter("p2p/dials/error/known", nil)
	dialSelf = metrics.NewRegisteredMeter("p2p/dials/error/self", nil)
	dialUselessPeer = metrics.NewRegisteredMeter("p2p/dials/error/useless", nil)
	dialFailedRLPXEncHandshake = metrics.NewRegisteredMeter("p2p/dials/error/rlpx/enc", nil)
	dialFailedRLPXProtoHandshake = metrics.NewRegisteredMeter("p2p/dials/error/rlpx/proto", nil)
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
	if !metrics.Enabled {
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
