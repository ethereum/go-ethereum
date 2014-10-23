package eth

import (
	"net"
)

// protocol is either "udp" or "tcp"
type NAT interface {
	GetExternalAddress() (addr net.IP, err error)
	AddPortMapping(protocol string, externalPort, internalPort int, description string, timeout int) (mappedExternalPort int, err error)
	DeletePortMapping(protocol string, externalPort, internalPort int) (err error)
}
