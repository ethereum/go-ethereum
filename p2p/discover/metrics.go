package discover

import (
	"net"

	"github.com/ethereum/go-ethereum/metrics"
)

const (
	moduleName = "discover"
	// ingressMeterName is the prefix of the per-packet inbound metrics.
	ingressMeterName = moduleName + "/ingress"

	// egressMeterName is the prefix of the per-packet outbound metrics.
	egressMeterName = moduleName + "/egress"
)

var (
	ingressTrafficMeter = metrics.NewRegisteredMeter(ingressMeterName, nil)
	egressTrafficMeter  = metrics.NewRegisteredMeter(egressMeterName, nil)
)

// meteredConn is a wrapper around a net.UDPConn that meters both the
// inbound and outbound network traffic.
type meteredUdpConn struct {
	UDPConn
}

func newMeteredConn(conn UDPConn) UDPConn {
	// Short circuit if metrics are disabled
	if !metrics.Enabled {
		return conn
	}

	return &meteredUdpConn{UDPConn: conn}
}

// Read delegates a network read to the underlying connection, bumping the udp ingress traffic meter along the way.
func (c *meteredUdpConn) ReadFromUDP(b []byte) (n int, addr *net.UDPAddr, err error) {
	n, addr, err = c.UDPConn.ReadFromUDP(b)
	ingressTrafficMeter.Mark(int64(n))
	return n, addr, err
}

// Write delegates a network write to the underlying connection, bumping the udp egress traffic meter along the way.
func (c *meteredUdpConn) WriteToUDP(b []byte, addr *net.UDPAddr) (n int, err error) {
	n, err = c.UDPConn.WriteToUDP(b, addr)
	egressTrafficMeter.Mark(int64(n))
	return n, err
}
