// Package nat provides access to common port mapping protocols.
package nat

import (
	"errors"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/jackpal/go-nat-pmp"
)

// An implementation of nat.Interface can map local ports to ports
// accessible from the Internet.
type Interface interface {
	// These methods manage a mapping between a port on the local
	// machine to a port that can be connected to from the internet.
	//
	// protocol is "UDP" or "TCP". Some implementations allow setting
	// a display name for the mapping. The mapping may be removed by
	// the gateway when its lifetime ends.
	AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error
	DeleteMapping(protocol string, extport, intport int) error

	// This method should return the external (Internet-facing)
	// address of the gateway device.
	ExternalIP() (net.IP, error)

	// Should return name of the method. This is used for logging.
	String() string
}

// Parse parses a NAT interface description.
// The following formats are currently accepted.
// Note that mechanism names are not case-sensitive.
//
//     "" or "none"         return nil
//     "extip:77.12.33.4"   will assume the local machine is reachable on the given IP
//     "any"                uses the first auto-detected mechanism
//     "upnp"               uses the Universal Plug and Play protocol
//     "pmp"                uses NAT-PMP with an auto-detected gateway address
//     "pmp:192.168.0.1"    uses NAT-PMP with the given gateway address
func Parse(spec string) (Interface, error) {
	var (
		parts = strings.SplitN(spec, ":", 2)
		mech  = strings.ToLower(parts[0])
		ip    net.IP
	)
	if len(parts) > 1 {
		ip = net.ParseIP(parts[1])
		if ip == nil {
			return nil, errors.New("invalid IP address")
		}
	}
	switch mech {
	case "", "none", "off":
		return nil, nil
	case "any", "auto", "on":
		return Any(), nil
	case "extip", "ip":
		if ip == nil {
			return nil, errors.New("missing IP address")
		}
		return ExtIP(ip), nil
	case "upnp":
		return UPnP(), nil
	case "pmp", "natpmp", "nat-pmp":
		return PMP(ip), nil
	default:
		return nil, fmt.Errorf("unknown mechanism %q", parts[0])
	}
}

const (
	mapTimeout        = 20 * time.Minute
	mapUpdateInterval = 15 * time.Minute
)

// Map adds a port mapping on m and keeps it alive until c is closed.
// This function is typically invoked in its own goroutine.
func Map(m Interface, c chan struct{}, protocol string, extport, intport int, name string) {
	refresh := time.NewTimer(mapUpdateInterval)
	defer func() {
		refresh.Stop()
		glog.V(logger.Debug).Infof("Deleting port mapping: %s %d -> %d (%s) using %s\n", protocol, extport, intport, name, m)
		m.DeleteMapping(protocol, extport, intport)
	}()
	glog.V(logger.Debug).Infof("add mapping: %s %d -> %d (%s) using %s\n", protocol, extport, intport, name, m)
	if err := m.AddMapping(protocol, intport, extport, name, mapTimeout); err != nil {
		glog.V(logger.Warn).Infof("network port %d could not be mapped: %v\n", intport, err)
		glog.V(logger.Debug).Infof("mapping with %v returned %v\n", m, err)
	}
	for {
		select {
		case _, ok := <-c:
			if !ok {
				return
			}
		case <-refresh.C:
			glog.V(logger.Detail).Infof("refresh mapping: %s %d -> %d (%s) using %s\n", protocol, extport, intport, name, m)
			if err := m.AddMapping(protocol, intport, extport, name, mapTimeout); err != nil {
				glog.V(logger.Warn).Infof("network port %d could not be mapped: %v\n", intport, err)
				glog.V(logger.Debug).Infof("mapping with %v returned %v\n", m, err)
			}
			refresh.Reset(mapUpdateInterval)
		}
	}
}

// ExtIP assumes that the local machine is reachable on the given
// external IP address, and that any required ports were mapped manually.
// Mapping operations will not return an error but won't actually do anything.
func ExtIP(ip net.IP) Interface {
	if ip == nil {
		panic("IP must not be nil")
	}
	return extIP(ip)
}

type extIP net.IP

func (n extIP) ExternalIP() (net.IP, error) { return net.IP(n), nil }
func (n extIP) String() string              { return fmt.Sprintf("ExtIP(%v)", net.IP(n)) }

// These do nothing.
func (extIP) AddMapping(string, int, int, string, time.Duration) error { return nil }
func (extIP) DeleteMapping(string, int, int) error                     { return nil }

// Any returns a port mapper that tries to discover any supported
// mechanism on the local network.
func Any() Interface {
	// TODO: attempt to discover whether the local machine has an
	// Internet-class address. Return ExtIP in this case.
	return startautodisc("UPnP or NAT-PMP", func() Interface {
		found := make(chan Interface, 2)
		go func() { found <- discoverUPnP() }()
		go func() { found <- discoverPMP() }()
		for i := 0; i < cap(found); i++ {
			if c := <-found; c != nil {
				return c
			}
		}
		return nil
	})
}

// UPnP returns a port mapper that uses UPnP. It will attempt to
// discover the address of your router using UDP broadcasts.
func UPnP() Interface {
	return startautodisc("UPnP", discoverUPnP)
}

// PMP returns a port mapper that uses NAT-PMP. The provided gateway
// address should be the IP of your router. If the given gateway
// address is nil, PMP will attempt to auto-discover the router.
func PMP(gateway net.IP) Interface {
	if gateway != nil {
		return &pmp{gw: gateway, c: natpmp.NewClient(gateway)}
	}
	return startautodisc("NAT-PMP", discoverPMP)
}

// autodisc represents a port mapping mechanism that is still being
// auto-discovered. Calls to the Interface methods on this type will
// wait until the discovery is done and then call the method on the
// discovered mechanism.
//
// This type is useful because discovery can take a while but we
// want return an Interface value from UPnP, PMP and Auto immediately.
type autodisc struct {
	what string // type of interface being autodiscovered
	once sync.Once
	doit func() Interface

	mu    sync.Mutex
	found Interface
}

func startautodisc(what string, doit func() Interface) Interface {
	// TODO: monitor network configuration and rerun doit when it changes.
	ad := &autodisc{what: what, doit: doit}
	// Start the auto discovery as early as possible so it is already
	// in progress when the rest of the stack calls the methods.
	go ad.wait()
	return ad
}

func (n *autodisc) AddMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error {
	if err := n.wait(); err != nil {
		return err
	}
	return n.found.AddMapping(protocol, extport, intport, name, lifetime)
}

func (n *autodisc) DeleteMapping(protocol string, extport, intport int) error {
	if err := n.wait(); err != nil {
		return err
	}
	return n.found.DeleteMapping(protocol, extport, intport)
}

func (n *autodisc) ExternalIP() (net.IP, error) {
	if err := n.wait(); err != nil {
		return nil, err
	}
	return n.found.ExternalIP()
}

func (n *autodisc) String() string {
	n.mu.Lock()
	defer n.mu.Unlock()
	if n.found == nil {
		return n.what
	} else {
		return n.found.String()
	}
}

// wait blocks until auto-discovery has been performed.
func (n *autodisc) wait() error {
	n.once.Do(func() {
		n.mu.Lock()
		n.found = n.doit()
		n.mu.Unlock()
	})
	if n.found == nil {
		return fmt.Errorf("no %s router discovered", n.what)
	}
	return nil
}
