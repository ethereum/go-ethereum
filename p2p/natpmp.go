package p2p

import (
	"fmt"
	"net"
	"time"

	natpmp "github.com/jackpal/go-nat-pmp"
)

// Adapt the NAT-PMP protocol to the NAT interface

// TODO:
//  + Register for changes to the external address.
//  + Re-register port mapping when router reboots.
//  + A mechanism for keeping a port mapping registered.
//  + Discover gateway address automatically.

type natPMPClient struct {
	client *natpmp.Client
}

// PMP returns a NAT traverser that uses NAT-PMP. The provided gateway
// address should be the IP of your router.
func PMP(gateway net.IP) (nat NAT) {
	return &natPMPClient{natpmp.NewClient(gateway)}
}

func (*natPMPClient) String() string {
	return "NAT-PMP"
}

func (n *natPMPClient) GetExternalAddress() (net.IP, error) {
	response, err := n.client.GetExternalAddress()
	if err != nil {
		return nil, err
	}
	return response.ExternalIPAddress[:], nil
}

func (n *natPMPClient) AddPortMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error {
	if lifetime <= 0 {
		return fmt.Errorf("lifetime must not be <= 0")
	}
	// Note order of port arguments is switched between our AddPortMapping and the client's AddPortMapping.
	_, err := n.client.AddPortMapping(protocol, intport, extport, int(lifetime/time.Second))
	return err
}

func (n *natPMPClient) DeletePortMapping(protocol string, externalPort, internalPort int) (err error) {
	// To destroy a mapping, send an add-port with
	// an internalPort of the internal port to destroy, an external port of zero and a time of zero.
	_, err = n.client.AddPortMapping(protocol, internalPort, 0, 0)
	return
}
