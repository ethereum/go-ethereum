package eth

import (
	"fmt"
	"net"

	natpmp "github.com/jackpal/go-nat-pmp"
)

// Adapt the NAT-PMP protocol to the NAT interface

// TODO:
//  + Register for changes to the external address.
//  + Re-register port mapping when router reboots.
//  + A mechanism for keeping a port mapping registered.

type natPMPClient struct {
	client *natpmp.Client
}

func NewNatPMP(gateway net.IP) (nat NAT) {
	return &natPMPClient{natpmp.NewClient(gateway)}
}

func (n *natPMPClient) GetExternalAddress() (addr net.IP, err error) {
	response, err := n.client.GetExternalAddress()
	if err != nil {
		return
	}
	ip := response.ExternalIPAddress
	addr = net.IPv4(ip[0], ip[1], ip[2], ip[3])
	return
}

func (n *natPMPClient) AddPortMapping(protocol string, externalPort, internalPort int,
	description string, timeout int) (mappedExternalPort int, err error) {
	if timeout <= 0 {
		err = fmt.Errorf("timeout must not be <= 0")
		return
	}
	// Note order of port arguments is switched between our AddPortMapping and the client's AddPortMapping.
	response, err := n.client.AddPortMapping(protocol, internalPort, externalPort, timeout)
	if err != nil {
		return
	}
	mappedExternalPort = int(response.MappedExternalPort)
	return
}

func (n *natPMPClient) DeletePortMapping(protocol string, externalPort, internalPort int) (err error) {
	// To destroy a mapping, send an add-port with
	// an internalPort of the internal port to destroy, an external port of zero and a time of zero.
	_, err = n.client.AddPortMapping(protocol, internalPort, 0, 0)
	return
}
