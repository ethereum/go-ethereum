package nat

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	nat "github.com/fd/go-nat"
	logging "github.com/ipfs/go-log"
	goprocess "github.com/jbenet/goprocess"
	periodic "github.com/jbenet/goprocess/periodic"
	ma "github.com/multiformats/go-multiaddr"
	manet "github.com/multiformats/go-multiaddr-net"
)

var (
	// ErrNoMapping signals no mapping exists for an address
	ErrNoMapping = errors.New("mapping not established")
)

var log = logging.Logger("nat")

// MappingDuration is a default port mapping duration.
// Port mappings are renewed every (MappingDuration / 3)
const MappingDuration = time.Second * 60

// CacheTime is the time a mapping will cache an external address for
const CacheTime = time.Second * 15

// DiscoverNAT looks for a NAT device in the network and
// returns an object that can manage port mappings.
func DiscoverNAT() *NAT {
	nat, err := nat.DiscoverGateway()
	if err != nil {
		log.Debug("DiscoverGateway error:", err)
		return nil
	}
	addr, err := nat.GetDeviceAddress()
	if err != nil {
		log.Debug("DiscoverGateway address error:", err)
	} else {
		log.Debug("DiscoverGateway address:", addr)
	}
	return newNAT(nat)
}

// NAT is an object that manages address port mappings in
// NATs (Network Address Translators). It is a long-running
// service that will periodically renew port mappings,
// and keep an up-to-date list of all the external addresses.
type NAT struct {
	natmu sync.Mutex
	nat   nat.NAT
	proc  goprocess.Process // manages nat mappings lifecycle

	mappingmu sync.RWMutex // guards mappings
	mappings  map[*mapping]struct{}

	Notifier
}

func newNAT(realNAT nat.NAT) *NAT {
	return &NAT{
		nat:      realNAT,
		proc:     goprocess.WithParent(goprocess.Background()),
		mappings: make(map[*mapping]struct{}),
	}
}

// Close shuts down all port mappings. NAT can no longer be used.
func (nat *NAT) Close() error {
	return nat.proc.Close()
}

// Process returns the nat's life-cycle manager, for making it listen
// to close signals.
func (nat *NAT) Process() goprocess.Process {
	return nat.proc
}

// Mappings returns a slice of all NAT mappings
func (nat *NAT) Mappings() []Mapping {
	nat.mappingmu.Lock()
	maps2 := make([]Mapping, 0, len(nat.mappings))
	for m := range nat.mappings {
		maps2 = append(maps2, m)
	}
	nat.mappingmu.Unlock()
	return maps2
}

func (nat *NAT) addMapping(m *mapping) {
	// make mapping automatically close when nat is closed.
	nat.proc.AddChild(m.proc)

	nat.mappingmu.Lock()
	nat.mappings[m] = struct{}{}
	nat.mappingmu.Unlock()
}

func (nat *NAT) rmMapping(m *mapping) {
	nat.mappingmu.Lock()
	delete(nat.mappings, m)
	nat.mappingmu.Unlock()
}

// NewMapping attemps to construct a mapping on protocol and internal port
// It will also periodically renew the mapping until the returned Mapping
// -- or its parent NAT -- is Closed.
//
// May not succeed, and mappings may change over time;
// NAT devices may not respect our port requests, and even lie.
// Clients should not store the mapped results, but rather always
// poll our object for the latest mappings.
func (nat *NAT) NewMapping(maddr ma.Multiaddr) (Mapping, error) {
	if nat == nil {
		return nil, fmt.Errorf("no nat available")
	}

	network, addr, err := manet.DialArgs(maddr)
	if err != nil {
		return nil, fmt.Errorf("DialArgs failed on addr: %s", maddr.String())
	}

	switch network {
	case "tcp", "tcp4", "tcp6":
		network = "tcp"
	case "udp", "udp4", "udp6":
		network = "udp"
	default:
		return nil, fmt.Errorf("transport not supported by NAT: %s", network)
	}

	intports := strings.Split(addr, ":")[1]
	intport, err := strconv.Atoi(intports)
	if err != nil {
		return nil, err
	}

	m := &mapping{
		nat:     nat,
		proto:   network,
		intport: intport,
		intaddr: maddr,
	}
	m.proc = goprocess.WithTeardown(func() error {
		nat.rmMapping(m)
		return nil
	})
	nat.addMapping(m)

	m.proc.AddChild(periodic.Every(MappingDuration/3, func(worker goprocess.Process) {
		nat.establishMapping(m)
	}))

	// do it once synchronously, so first mapping is done right away, and before exiting,
	// allowing users -- in the optimistic case -- to use results right after.
	nat.establishMapping(m)
	return m, nil
}

func (nat *NAT) establishMapping(m *mapping) {
	oldport := m.ExternalPort()

	log.Debugf("Attempting port map: %s/%d", m.Protocol(), m.InternalPort())
	comment := "libp2p"
	if m.comment != "" {
		comment = "libp2p-" + m.comment
	}

	nat.natmu.Lock()
	newport, err := nat.nat.AddPortMapping(m.Protocol(), m.InternalPort(), comment, MappingDuration)
	if err != nil {
		// Some hardware does not support mappings with timeout, so try that
		newport, err = nat.nat.AddPortMapping(m.Protocol(), m.InternalPort(), comment, 0)
	}
	nat.natmu.Unlock()

	failure := func() {
		m.setExternalPort(0) // clear mapping
		// TODO: log.Event
		log.Warningf("failed to establish port mapping: %s", err)
		nat.Notifier.notifyAll(func(n Notifiee) {
			n.MappingFailed(nat, m, oldport, err)
		})

		// we do not close if the mapping failed,
		// because it may work again next time.
	}

	if err != nil || newport == 0 {
		failure()
		return
	}

	m.setExternalPort(newport)
	ext, err := m.ExternalAddr()
	if err != nil {
		log.Debugf("NAT Mapping addr error: %s %s", m.InternalAddr(), err)
		failure()
		return
	}

	log.Debugf("NAT Mapping: %s --> %s", m.InternalAddr(), ext)
	if oldport != 0 && newport != oldport {
		log.Debugf("failed to renew same port mapping: ch %d -> %d", oldport, newport)
		nat.Notifier.notifyAll(func(n Notifiee) {
			n.MappingChanged(nat, m, oldport, newport)
		})
	}

	nat.Notifier.notifyAll(func(n Notifiee) {
		n.MappingSuccess(nat, m)
	})
}

// PortMapAddrs attempts to open (and continue to keep open)
// port mappings for given addrs. This function blocks until
// all  addresses have been tried. This allows clients to
// retrieve results immediately after:
//
//   nat.PortMapAddrs(addrs)
//   mapped := nat.ExternalAddrs()
//
// Some may not succeed, and mappings may change over time;
// NAT devices may not respect our port requests, and even lie.
// Clients should not store the mapped results, but rather always
// poll our object for the latest mappings.
func (nat *NAT) PortMapAddrs(addrs []ma.Multiaddr) {
	// spin off addr mappings independently.
	var wg sync.WaitGroup
	for _, addr := range addrs {
		// do all of them concurrently
		wg.Add(1)
		go func(addr ma.Multiaddr) {
			defer wg.Done()
			nat.NewMapping(addr)
		}(addr)
	}
	wg.Wait()
}

// MappedAddrs returns address mappings NAT believes have been
// successfully established. Unsuccessful mappings are nil. This is:
//
// 		map[internalAddr]externalAddr
//
// This set of mappings _may not_ be correct, as NAT devices are finicky.
// Consider this with _best effort_ semantics.
func (nat *NAT) MappedAddrs() map[ma.Multiaddr]ma.Multiaddr {

	mappings := nat.Mappings()
	addrmap := make(map[ma.Multiaddr]ma.Multiaddr, len(mappings))

	for _, m := range mappings {
		i := m.InternalAddr()
		e, err := m.ExternalAddr()
		if err != nil {
			addrmap[i] = nil
		} else {
			addrmap[i] = e
		}
	}
	return addrmap
}

// ExternalAddrs returns a list of addresses that NAT believes have
// been successfully established. Unsuccessful mappings are omitted,
// so nat.ExternalAddrs() may return less addresses than nat.InternalAddrs().
// To see which addresses are mapped, use nat.MappedAddrs().
//
// This set of mappings _may not_ be correct, as NAT devices are finicky.
// Consider this with _best effort_ semantics.
func (nat *NAT) ExternalAddrs() []ma.Multiaddr {
	mappings := nat.Mappings()
	addrs := make([]ma.Multiaddr, 0, len(mappings))
	for _, m := range mappings {
		a, err := m.ExternalAddr()
		if err != nil {
			continue // this mapping not currently successful.
		}
		addrs = append(addrs, a)
	}
	return addrs
}
