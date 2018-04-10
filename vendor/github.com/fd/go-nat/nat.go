// Package nat implements NAT handling facilities
package nat

import (
	"errors"
	"math"
	"math/rand"
	"net"
	"time"
)

var ErrNoExternalAddress = errors.New("no external address")
var ErrNoInternalAddress = errors.New("no internal address")
var ErrNoNATFound = errors.New("no NAT found")

// protocol is either "udp" or "tcp"
type NAT interface {
	// Type returns the kind of NAT port mapping service that is used
	Type() string

	// GetDeviceAddress returns the internal address of the gateway device.
	GetDeviceAddress() (addr net.IP, err error)

	// GetExternalAddress returns the external address of the gateway device.
	GetExternalAddress() (addr net.IP, err error)

	// GetInternalAddress returns the address of the local host.
	GetInternalAddress() (addr net.IP, err error)

	// AddPortMapping maps a port on the local host to an external port.
	AddPortMapping(protocol string, internalPort int, description string, timeout time.Duration) (mappedExternalPort int, err error)

	// DeletePortMapping removes a port mapping.
	DeletePortMapping(protocol string, internalPort int) (err error)
}

// DiscoverGateway attempts to find a gateway device.
func DiscoverGateway() (NAT, error) {
	select {
	case nat := <-discoverUPNP_IG1():
		return nat, nil
	case nat := <-discoverUPNP_IG2():
		return nat, nil
	case nat := <-discoverNATPMP():
		return nat, nil
	case <-time.After(10 * time.Second):
		return nil, ErrNoNATFound
	}
}

func randomPort() int {
	rand.Seed(time.Now().UnixNano())
	return rand.Intn(math.MaxUint16-10000) + 10000
}
