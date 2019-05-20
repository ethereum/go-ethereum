package nat

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/jackpal/go-nat-pmp"
)

// natPMPClient adapts the NAT-PMP protocol implementation so it conforms to
// the common interface.
type pmp struct {
	gw net.IP
	c  *natpmp.Client
}

func (n *pmp) String() string {
	return fmt.Sprintf("NAT-PMP(%v)", n.gw)
}

func (n *pmp) ExternalIP() (net.IP, error) {
	response, err := n.c.GetExternalAddress()
	if err != nil {
		return nil, err
	}
	return response.ExternalIPAddress[:], nil
}

func (n *pmp) AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error {
	if lifetime <= 0 {
		return fmt.Errorf("lifetime must not be <= 0")
	}
	// Note order of port arguments is switched between our
	// AddMapping and the client's AddPortMapping.
	_, err := n.c.AddPortMapping(strings.ToLower(protocol), intport, extport, int(lifetime/time.Second))
	return err
}

func (n *pmp) DeleteMapping(protocol string, extport, intport int) (err error) {
	// To destroy a mapping, send an add-port with an internalPort of
	// the internal port to destroy, an external port of zero and a
	// time of zero.
	_, err = n.c.AddPortMapping(strings.ToLower(protocol), intport, 0, 0)
	return err
}

func discoverPMP() Interface {
	// run external address lookups on all potential gateways
	gws := potentialGateways()
	found := make(chan *pmp, len(gws))
	for i := range gws {
		gw := gws[i]
		go func() {
			c := natpmp.NewClient(gw)
			if _, err := c.GetExternalAddress(); err != nil {
				found <- nil
			} else {
				found <- &pmp{gw, c}
			}
		}()
	}
	// return the one that responds first.
	// discovery needs to be quick, so we stop caring about
	// any responses after a very short timeout.
	timeout := time.NewTimer(1 * time.Second)
	defer timeout.Stop()
	for _ = range gws {
		select {
		case c := <-found:
			if c != nil {
				return c
			}
		case <-timeout.C:
			return nil
		}
	}
	return nil
}

var (
	// LAN IP ranges
	_, lan10, _  = net.ParseCIDR("10.0.0.0/8")
	_, lan176, _ = net.ParseCIDR("172.16.0.0/12")
	_, lan192, _ = net.ParseCIDR("192.168.0.0/16")
)

// TODO: improve this. We currently assume that (on most networks)
// the router is X.X.X.1 in a local LAN range.
func potentialGateways() (gws []net.IP) {
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil
	}
	for _, iface := range ifaces {
		ifaddrs, err := iface.Addrs()
		if err != nil {
			return gws
		}
		for _, addr := range ifaddrs {
			switch x := addr.(type) {
			case *net.IPNet:
				if lan10.Contains(x.IP) || lan176.Contains(x.IP) || lan192.Contains(x.IP) {
					ip := x.IP.Mask(x.Mask).To4()
					if ip != nil {
						ip[3] = ip[3] | 0x01
						gws = append(gws, ip)
					}
				}
			}
		}
	}
	return gws
}
