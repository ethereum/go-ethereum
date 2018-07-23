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

var (
	ingressConnectMeter = metrics.NewRegisteredMeter("p2p/InboundConnects", nil)
	egressConnectMeter  = metrics.NewRegisteredMeter("p2p/OutboundConnects", nil)
	TrafficMeter        = newTrafficMeter()
	nextDefaultID       uint32
)

const (
	IngressPrefix = "p2p/InboundTraffic"
	EgressPrefix  = "p2p/OutboundTraffic"
)

type trafficMeter struct {
	peerIngress   map[string]metrics.Meter
	peerEgress    map[string]metrics.Meter
	commonIngress metrics.Meter
	commonEgress  metrics.Meter
	changed       map[string]string
	lock          sync.RWMutex
	cLock         sync.RWMutex
}

// Create a new registry.
func newTrafficMeter() *trafficMeter {
	return &trafficMeter{
		peerIngress:   make(map[string]metrics.Meter),
		peerEgress:    make(map[string]metrics.Meter),
		commonIngress: metrics.NewRegisteredMeter("p2p/InboundTraffic", nil),
		commonEgress:  metrics.NewRegisteredMeter("p2p/OutboundTraffic", nil),
		changed:       make(map[string]string),
	}
}

func (tm *trafficMeter) register(id string, ingressMeter, egressMeter metrics.Meter) error {
	if ingressMeter == nil && egressMeter == nil {
		tm.lock.Lock()
		tm.peerIngress[id] = metrics.NewRegisteredMeter(fmt.Sprintf("%s/%s", IngressPrefix, id), nil)
		tm.peerEgress[id] = metrics.NewRegisteredMeter(fmt.Sprintf("%s/%s", EgressPrefix, id), nil)
		tm.lock.Unlock()
		return nil
	}
	if ingressMeter == nil || egressMeter == nil {
		return errors.New("Meter is not set")
	}
	if err := metrics.Register(fmt.Sprintf("%s/%s", IngressPrefix, id), ingressMeter); err != nil {
		return err
	}
	if err := metrics.Register(fmt.Sprintf("%s/%s", EgressPrefix, id), egressMeter); err != nil {
		metrics.Unregister(fmt.Sprintf("%s/%s", IngressPrefix, id))
		return err
	}
	tm.lock.Lock()
	tm.peerIngress[id] = ingressMeter
	tm.peerEgress[id] = egressMeter
	tm.lock.Unlock()
	return nil
}

func (tm *trafficMeter) delete(id string) {
	tm.lock.Lock()
	delete(tm.peerIngress, id)
	delete(tm.peerEgress, id)
	tm.lock.Unlock()
}
func (tm *trafficMeter) unregister(old, new string) {
	metrics.Unregister(fmt.Sprintf("%s/%s", IngressPrefix, old))
	metrics.Unregister(fmt.Sprintf("%s/%s", EgressPrefix, old))
	tm.delete(old)
	tm.cLock.Lock()
	tm.changed[old] = new
	tm.cLock.Unlock()
}

func (tm *trafficMeter) changeID(old, new string) error {
	tm.lock.RLock()
	irm, oki := tm.peerIngress[old]
	erm, oke := tm.peerEgress[old]
	tm.lock.RUnlock()
	if !oki || !oke {
		return errors.New(fmt.Sprintf("No meter with id %s", old))
	}
	if err := tm.register(new, irm, erm); err != nil {
		return err
	}
	tm.unregister(old, new)
	return nil
}

func (tm *trafficMeter) markIngress(id string, n int64) {
	tm.commonIngress.Mark(n)
	tm.lock.RLock()
	rm, ok := tm.peerIngress[id]
	tm.lock.RUnlock()
	if ok {
		rm.Mark(n)
	}
}

func (tm *trafficMeter) markEgress(id string, n int64) {
	tm.commonEgress.Mark(n)
	tm.lock.RLock()
	rm, ok := tm.peerEgress[id]
	tm.lock.RUnlock()
	if ok {
		rm.Mark(n)
	}
}

func (tm *trafficMeter) GetIDs() []string {
	ids := make([]string, 0, len(tm.peerIngress))
	tm.lock.RLock()
	for id := range tm.peerIngress {
		ids = append(ids, id)
	}
	tm.lock.RUnlock()
	return ids
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
	TrafficMeter.register(id, nil, nil)
	return &meteredConn{Conn: conn, id: id}
}

// Read delegates a network read to the underlying connection, bumping the ingress
// traffic meter along the way.
func (c *meteredConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	TrafficMeter.markIngress(c.id, int64(n))
	return n, err
}

// Write delegates a network write to the underlying connection, bumping the
// egress traffic meter along the way.
func (c *meteredConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	TrafficMeter.markEgress(c.id, int64(n))
	return n, err
}

func (c *meteredConn) setPeerID(id string) {
	if err := TrafficMeter.changeID(c.id, id); err != nil {
		log.Warn("Failed to set peer id", "id", fmt.Sprintf("%s...%s", id[:6], id[len(id)-6:]), "err", err)
		return
	}
	c.id = id
}

func (c *meteredConn) Close() error {
	TrafficMeter.unregister(c.id, "")
	fmt.Println("Close", c.id)
	return c.Conn.Close()
}
