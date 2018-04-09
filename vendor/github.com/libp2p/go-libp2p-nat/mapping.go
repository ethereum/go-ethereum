package nat

import (
	"fmt"
	"sync"
	"time"

	"github.com/jbenet/goprocess"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

// Mapping represents a port mapping in a NAT.
type Mapping interface {
	// NAT returns the NAT object this Mapping belongs to.
	NAT() *NAT

	// Protocol returns the protocol of this port mapping. This is either
	// "tcp" or "udp" as no other protocols are likely to be NAT-supported.
	Protocol() string

	// InternalPort returns the internal device port. Mapping will continue to
	// try to map InternalPort() to an external facing port.
	InternalPort() int

	// ExternalPort returns the external facing port. If the mapping is not
	// established, port will be 0
	ExternalPort() int

	// InternalAddr returns the internal address.
	InternalAddr() ma.Multiaddr

	// ExternalAddr returns the external facing address. If the mapping is not
	// established, addr will be nil, and and ErrNoMapping will be returned.
	ExternalAddr() (addr ma.Multiaddr, err error)

	// Close closes the port mapping
	Close() error
}

// keeps republishing
type mapping struct {
	sync.Mutex // guards all fields

	nat       *NAT
	proto     string
	intport   int
	extport   int
	permanent bool
	intaddr   ma.Multiaddr
	proc      goprocess.Process

	comment string

	cached    ma.Multiaddr
	cacheTime time.Time
	cacheLk   sync.Mutex
}

func (m *mapping) NAT() *NAT {
	m.Lock()
	defer m.Unlock()
	return m.nat
}

func (m *mapping) Protocol() string {
	m.Lock()
	defer m.Unlock()
	return m.proto
}

func (m *mapping) InternalPort() int {
	m.Lock()
	defer m.Unlock()
	return m.intport
}

func (m *mapping) ExternalPort() int {
	m.Lock()
	defer m.Unlock()
	return m.extport
}

func (m *mapping) setExternalPort(p int) {
	m.Lock()
	defer m.Unlock()
	m.extport = p
}

func (m *mapping) InternalAddr() ma.Multiaddr {
	m.Lock()
	defer m.Unlock()
	return m.intaddr
}

func (m *mapping) ExternalAddr() (ma.Multiaddr, error) {
	m.cacheLk.Lock()
	ctime := m.cacheTime
	cval := m.cached
	m.cacheLk.Unlock()
	if time.Since(ctime) < CacheTime {
		return cval, nil
	}

	if m.ExternalPort() == 0 { // dont even try right now.
		return nil, ErrNoMapping
	}

	m.nat.natmu.Lock()
	ip, err := m.nat.nat.GetExternalAddress()
	m.nat.natmu.Unlock()
	if err != nil {
		return nil, err
	}

	ipmaddr, err := manet.FromIP(ip)
	if err != nil {
		return nil, fmt.Errorf("error parsing ip")
	}

	// call m.ExternalPort again, as mapping may have changed under our feet. (tocttou)
	extport := m.ExternalPort()
	if extport == 0 {
		return nil, ErrNoMapping
	}

	tcp, err := ma.NewMultiaddr(fmt.Sprintf("/%s/%d", m.Protocol(), extport))
	if err != nil {
		return nil, err
	}

	maddr2 := ipmaddr.Encapsulate(tcp)

	m.cacheLk.Lock()
	m.cached = maddr2
	m.cacheTime = time.Now()
	m.cacheLk.Unlock()
	return maddr2, nil
}

func (m *mapping) Close() error {
	return m.proc.Close()
}
