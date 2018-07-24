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

	"fmt"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/metrics"
	"github.com/syndtr/goleveldb/leveldb/errors"
	"sync"
	"sync/atomic"
)

const (
	IngressPrefix = "p2p/InboundTraffic"
	EgressPrefix  = "p2p/OutboundTraffic"
)

var (
	ingressConnectMeter = metrics.NewRegisteredMeter("p2p/InboundConnects", nil)
	egressConnectMeter  = metrics.NewRegisteredMeter("p2p/OutboundConnects", nil)

	PeerIngressRegistry   = metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, IngressPrefix+"/")
	PeerEgressRegistry    = metrics.NewPrefixedChildRegistry(metrics.DefaultRegistry, EgressPrefix+"/")
	TrafficMeterCollector = newTrafficMeterCollector()

	nextDefaultID uint32
)

type trafficMeter struct {
	ingress metrics.Meter
	egress  metrics.Meter
}

type trafficMeterCollector struct {
	common  *trafficMeter
	peers   map[string]*trafficMeter
	changed []string

	lock  sync.RWMutex
	cLock sync.Mutex
}

func newTrafficMeterCollector() *trafficMeterCollector {
	return &trafficMeterCollector{
		common: &trafficMeter{
			ingress: metrics.NewRegisteredMeter(IngressPrefix, nil),
			egress:  metrics.NewRegisteredMeter(EgressPrefix, nil),
		},
		peers:   make(map[string]*trafficMeter),
		changed: make([]string, 0, 128),
	}
}

func (tmc *trafficMeterCollector) register(id string, tm *trafficMeter) error {
	if tm == nil {
		peer := &trafficMeter{
			ingress: metrics.NewRegisteredMeter(id, PeerIngressRegistry),
			egress:  metrics.NewRegisteredMeter(id, PeerEgressRegistry),
		}
		tmc.lock.Lock()
		tmc.peers[id] = peer
		tmc.lock.Unlock()
		return nil
	}
	if tm.ingress == nil || tm.egress == nil {
		return errors.New("Meter is not set correctly")
	}
	if err := PeerIngressRegistry.Register(id, tm.ingress); err != nil {
		return err
	}
	if err := PeerEgressRegistry.Register(id, tm.egress); err != nil {
		PeerIngressRegistry.Unregister(id)
		return err
	}
	tmc.lock.Lock()
	tmc.peers[id] = tm
	tmc.lock.Unlock()
	return nil
}

func (tmc *trafficMeterCollector) unregister(old, new string) {
	PeerIngressRegistry.Unregister(old)
	PeerEgressRegistry.Unregister(old)

	tmc.lock.Lock()
	delete(tmc.peers, old)
	tmc.lock.Unlock()

	tmc.cLock.Lock()
	tmc.changed = append(tmc.changed, old, new)
	tmc.cLock.Unlock()
}

func (tmc *trafficMeterCollector) changeID(old, new string) error {
	tmc.lock.RLock()
	peer, ok := tmc.peers[old]
	tmc.lock.RUnlock()
	if !ok {
		return errors.New(fmt.Sprintf("No meter with id %s", old))
	}
	if err := tmc.register(new, peer); err != nil {
		return err
	}
	tmc.unregister(old, new)
	return nil
}

func (tmc *trafficMeterCollector) markIngress(id string, n int64) {
	tmc.common.ingress.Mark(n)
	tmc.lock.RLock()
	peer, ok := tmc.peers[id]
	tmc.lock.RUnlock()
	if ok {
		peer.ingress.Mark(n)
	}
}

func (tmc *trafficMeterCollector) markEgress(id string, n int64) {
	tmc.common.egress.Mark(n)
	tmc.lock.RLock()
	peer, ok := tmc.peers[id]
	tmc.lock.RUnlock()
	if ok {
		peer.egress.Mark(n)
	}
}

func (tmc *trafficMeterCollector) GetIDs() []string {
	tmc.lock.RLock()
	ids := make([]string, 0, len(tmc.peers))
	for id := range tmc.peers {
		ids = append(ids, id)
	}
	tmc.lock.RUnlock()
	return ids
}

func (tmc *trafficMeterCollector) GetAndClearChanged() []string {
	tmc.cLock.Lock()
	defer tmc.cLock.Unlock()

	changed := make([]string, len(tmc.changed))
	copy(changed, tmc.changed)
	tmc.changed = tmc.changed[:0]

	return changed
}

// meteredConn is a wrapper around a net.Conn that meters both the
// inbound and outbound network traffic.
type meteredConn struct {
	net.Conn // Network connection to wrap with metering
	id       string
}

// newMeteredConn creates a new metered connection, also bumping the ingress or
// egress connection meter. If the metrics system is disabled, this function
// returns the original object.
func newMeteredConn(conn net.Conn, ingress bool) net.Conn {
	// Short circuit if metrics are disabled
	if !metrics.Enabled {
		return conn
	}
	// Otherwise bump the connection counters and wrap the connection
	if ingress {
		ingressConnectMeter.Mark(1)
	} else {
		egressConnectMeter.Mark(1)
	}
	id := fmt.Sprintf("unidentified_%d", atomic.AddUint32(&nextDefaultID, 1))
	TrafficMeterCollector.register(id, nil)
	return &meteredConn{Conn: conn, id: id}
}

// Read delegates a network read to the underlying connection, bumping the ingress
// traffic meter along the way.
func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	TrafficMeterCollector.markIngress(c.id, int64(n))
	return n, err
}

// Write delegates a network write to the underlying connection, bumping the
// egress traffic meter along the way.
func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	TrafficMeterCollector.markEgress(c.id, int64(n))
	return n, err
}

func (c *meteredConn) Close() error {
	TrafficMeterCollector.unregister(c.id, "")
	return c.Conn.Close()
}

func (c *meteredConn) setPeerID(id string) {
	if err := TrafficMeterCollector.changeID(c.id, id); err != nil {
		log.Warn("Failed to set peer id", "id", fmt.Sprintf("%s...%s", id[:6], id[len(id)-6:]), "err", err)
		return
	}
	c.id = id
}
