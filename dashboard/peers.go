// Copyright 2018 The go-ethereum Authors
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

package dashboard

import (
	"container/list"
	"encoding/json"
	"fmt"
	"github.com/ethereum/go-ethereum/metrics"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	eventBufferLimit = 128 // Maximum number of buffered peer events
	connectionLimit  = 16
)

var autoID int64

// maintainedPeer is the element of the peer maintainer's linked list.
// Similarly to an ordinary linked list element it knows the previous
// and the next elements, and also has a pointer to its parent map in
// order to remove itself from there when it is removed from the list.
type maintainedPeer struct {
	// To simplify the implementation, the list is implemented as a ring,
	// such that 'root' is both the next element of the last list element
	// and the previous element of the first list element.
	prev   *maintainedPeer // Pointer to the previous element
	next   *maintainedPeer // Pointer to the next element
	parent *idContainer    // Pointer to the parent
	id     string          // NodeID of the peer
}

// idContainer contains the NodeIDs belonging to an IP address.
// The pointer to the parent makes it possible for the container
// to remove itself from the parent's map when becomes empty.
type idContainer struct {
	parent *PeerMaintainer            // Pointer to the parent of the container
	ip     string                     // IP of the peer
	peers  map[string]*maintainedPeer // NodeIDs belonging to the given IP
}

// update moves the list element belonging to the given ID to the end,
// or inserts a new element to the end of the list if the given ID didn't
// appear yet in the container. Returns the IP and the ID of the removed
// peer if an element removal happened, or nil otherwise.
func (idc *idContainer) update(id string) *removedPeer {
	maintainer := idc.parent
	if _, ok := idc.peers[id]; !ok {
		e := &maintainedPeer{
			parent: idc,
			id:     id,
		}
		maintainer.insert(e, maintainer.root.prev)
		idc.peers[id] = e
	} else {
		maintainer.insert(maintainer.remove(idc.peers[id]), maintainer.root.prev)
	}
	if maintainer.len > maintainer.limit {
		first := maintainer.remove(maintainer.root.next)
		return &removedPeer{
			ip: first.parent.ip,
			id: first.id,
		}
	}
	return nil
}

// remove removes the peer belonging to the given ID and removes itself too
// from the parent container if becomes empty.
func (idc *idContainer) remove(id string) {
	delete(idc.peers, id)
	if len(idc.peers) < 1 {
		idc.parent.removeIP(idc.ip)
		idc.parent = nil
	}
}

// PeerMaintainer is an abstract layer above the metered peer container,
// maintaining the peers. i.e. sorting them based on their activity and
// removing the oldest inactive ones when their count reaches the limit.
//
// Consists of a map of maps which represent the peers grouped by the IP
// address then by the NodeID. The elements on the bottom of the tree are
// doubly linked and sorted by their activity. i.e. the first element is
// the peer that is inactive for the longest time. This makes it possible
// to count the peers and remove the oldest one effectively.
//
// When a peer event appears, the active peer goes to the end of the list
// and in case of removal the IP and the ID of the removed peer is returned.
type PeerMaintainer struct {
	ids   map[string]*idContainer // NodeIDs grouped by IP
	root  maintainedPeer          // Sentinel list element
	len   int                     // Current list length excluding the sentinel element
	limit int                     // Maximum number of maintained peers
}

// init initializes the peer maintainer.
func (pm *PeerMaintainer) init(limit int) *PeerMaintainer {
	if limit < 0 {
		limit = 0
	}
	pm.ids = make(map[string]*idContainer)
	pm.root.prev = &pm.root
	pm.root.next = &pm.root
	pm.len = 0
	pm.limit = limit
	return pm
}

// NewPeerMaintainer returns an initialized peer maintainer.
func NewPeerMaintainer(limit int) *PeerMaintainer {
	return new(PeerMaintainer).init(limit)
}

// Update updates the peer element belonging to the given IP and ID.
func (pm *PeerMaintainer) Update(ip, id string) *removedPeer {
	return pm.getOrInitIDs(ip).update(id)
}

// getOrInitIDs returns the ID map. Initializes it if it doesn't exist.
func (pm *PeerMaintainer) getOrInitIDs(ip string) *idContainer {
	if _, ok := pm.ids[ip]; !ok {
		pm.ids[ip] = &idContainer{
			parent: pm,
			ip:     ip,
			peers:  make(map[string]*maintainedPeer),
		}
	}
	return pm.ids[ip]
}

// insert inserts 'e' after 'at' and increments the current list length.
func (pm *PeerMaintainer) insert(e, at *maintainedPeer) {
	n := at.next
	at.next = e
	e.prev = at
	e.next = n
	n.prev = e
	pm.len++
}

// remove removes e from the list and from the maintainer tree.
func (pm *PeerMaintainer) remove(e *maintainedPeer) *maintainedPeer {
	e.next.prev = e.prev
	e.prev.next = e.next
	e.prev = nil
	e.next = nil
	e.parent.remove(e.id)
	e.parent = nil
	pm.len--
	return e
}

// clear cleans up the peer maintainer.
func (pm *PeerMaintainer) clear() {
	for pm.root.next != &pm.root {
		next := pm.root.next
		pm.root.next = next.next
		next.prev = nil
		next.next = nil
		next.parent.remove(next.id)
		next.parent = nil
	}
}

// removeIP removes the peers belonging to the given IP. It is supposed that
// 'ids' is empty, because the clearing direction is from bottom to top.
func (pm *PeerMaintainer) removeIP(ip string) {
	delete(pm.ids, ip)
}

// removedPeer contains the IP and the ID of the peer that was removed from
// the peer maintainer.
type removedPeer struct {
	ip, id string
}









const (
	knownPeerLimit   = p2p.MeteredPeerLimit
	unknownPeerLimit = p2p.MeteredPeerLimit
)

type PeerContainer struct {
	Bundles map[string]*PeerBundle `json:"peerBundles,omitempty"`

	activeSeparator list.Element
	knownPeers      *list.List
	unknownPeers    *list.List

	refresh time.Duration
}

func NewPeerContainer(refresh time.Duration) *PeerContainer {
	return &PeerContainer{
		Bundles:      make(map[string]*PeerBundle),
		knownPeers:   list.New(),
		unknownPeers: list.New(),
		refresh: refresh,
	}
}

func (pc *PeerContainer) getOrInit(ip string) *PeerBundle {
	if _, ok := pc.Bundles[ip]; !ok {
		pc.Bundles[ip] = &PeerBundle{
			KnownPeers: make(map[string]*KnownPeer),
			parent: pc,
			ip: ip,
		}
	}
	return pc.Bundles[ip]
}

func (pc *PeerContainer) updateKnown(ip, id string, cycle *PeerCycle) {
	pc.getOrInit(ip).updateKnown(id, cycle)
}

func (pc *PeerContainer) updateUnknown(ip string, peer *UnknownPeer) {
	pc.getOrInit(ip).updateUnknown(peer)
}

func (pc *PeerContainer) Update(event *peerEvent) {
	switch event.t {
	case peerConnected:
		connected := time.Now().Add(-event.Elapsed)
		pc.updateKnown(event.IP.String(), event.ID, &PeerCycle{
			Connected: &connected,
		})
	case peerDisconnected:
		now := time.Now()
		pc.updateKnown(event.IP.String(), event.ID, &PeerCycle{
			Disconnected: &now,
			Ingress: ChartEntries{
				&ChartEntry{
					Time:  now,
					Value: float64(event.Ingress),
				},
			},
			Egress: ChartEntries{
				&ChartEntry{
					Time:  now,
					Value: float64(event.Egress),
				},
			},
		})
	case peerHandshakeFailed:
		now := time.Now()
		pc.updateUnknown(event.IP.String(), &UnknownPeer{
			Connected: now.Add(-event.Elapsed),
			Disconnected: now,
		})
	case peerIngress:
		fmt.Println("Ingress:", event)
		pc.updateKnown(event.ip, event.id, &PeerCycle{
			Ingress: ChartEntries{
				&ChartEntry{
					Time: time.Now(),
					Value: float64(event.traffic),
				},
			},
		})
	case peerEgress:
		fmt.Println("Egress:", event)
		pc.updateKnown(event.ip, event.id, &PeerCycle{
			Egress: ChartEntries{
				&ChartEntry{
					Time: time.Now(),
					Value: float64(event.traffic),
				},
			},
		})
	default:
		log.Error("Unknown peer event type", "type", event.Type)
	}
}

type PeerBundle struct {
	Location     *GeoLocation          `json:"location,omitempty"`
	KnownPeers   map[string]*KnownPeer `json:"knownPeers,omitempty"`
	UnknownPeers []*UnknownPeer         `json:"unknownPeers,omitempty"`

	parent *PeerContainer
	ip     string
}

func (b *PeerBundle) getOrInit(id string) *KnownPeer {
	if _, ok := b.KnownPeers[id]; !ok {
		b.KnownPeers[id] = &KnownPeer{
			parent: b,
			id: id,
		}
	}
	return b.KnownPeers[id]
}

func (b *PeerBundle) updateKnown(id string, cycle *PeerCycle) {
	b.getOrInit(id).update(cycle)
}

func (b *PeerBundle) updateUnknown(peer *UnknownPeer) {
	b.UnknownPeers = append(b.UnknownPeers, peer)
}

type KnownPeer struct {
	Cycles []*PeerCycle `json:"cycles,omitempty"`

	parent *PeerBundle
	id     string
}

func (peer *KnownPeer) update(cycle *PeerCycle) {
	if cycle.Connected != nil {
		if peer.Cycles == nil {
			peer.Cycles = append(peer.Cycles, &PeerCycle{
				Ingress: emptyChartEntries(time.Now(), 3, peer.parent.parent.refresh),
				Egress: emptyChartEntries(time.Now(), 3, peer.parent.parent.refresh),
			})
		}
		peer.Cycles = append(peer.Cycles, cycle)
		return
	}
	if cycle.Disconnected != nil {
		if len(peer.Cycles) < 1 {
			log.Error("Peer disconnect event appeared without connect")
			return
		}
		last := peer.Cycles[len(peer.Cycles)-1]
		last.Disconnected = cycle.Disconnected
		last.Ingress = append(last.Ingress, cycle.Ingress...)
		last.Egress = append(last.Egress, cycle.Egress...)
		return
	}
}

type PeerCycle struct {
	Connected    *time.Time `json:"connected,omitempty"`
	Disconnected *time.Time `json:"disconnected,omitempty"`

	Ingress ChartEntries `json:"ingress,omitempty"`
	Egress  ChartEntries `json:"egress,omitempty"`
}

type UnknownPeer struct {
	Connected    time.Time `json:"connected,omitempty"`
	Disconnected time.Time `json:"disconnected,omitempty"`
}

type peerContainerUpdateType int

const (
	peerConnected       = peerContainerUpdateType(p2p.PeerConnected)
	peerDisconnected    = peerContainerUpdateType(p2p.PeerDisconnected)
	peerHandshakeFailed = peerContainerUpdateType(p2p.PeerHandshakeFailed)
	peerIngress         = peerConnected + peerDisconnected + peerHandshakeFailed + iota
	peerEgress
)

type peerEvent struct {
	*p2p.MeteredPeerEvent
	ip string
	id string
	t peerContainerUpdateType
	traffic uint64
}














// collectPeerData gathers data about the peers and sends it to the clients.
func (db *Dashboard) collectPeerData() {
	defer db.wg.Done()

	// Open the geodb database for IP to geographical information conversions.
	var err error
	db.geodb, err = OpenGeoDB()
	if err != nil {
		log.Warn("Failed to open geodb", "err", err)
		return
	}
	defer db.geodb.Close()

	peerCh := make(chan p2p.MeteredPeerEvent, eventBufferLimit) // Peer event channel.
	subPeer := p2p.SubscribeMeteredPeerEvent(peerCh)            // Subscribe to peer events.
	defer subPeer.Unsubscribe()                                 // Unsubscribe at the end.

	ticker := time.NewTicker(db.config.Refresh)
	defer ticker.Stop()

	pc := NewPeerContainer(db.config.Refresh)

	trafficUpdater := func(prefix string) (func (*map[string]int64) func(name string, i interface{})) {
		return func (entryMap *map[string]int64) func(name string, i interface{}) {
			return func(name string, i interface{}) {
				if m, ok := i.(metrics.Meter); ok {
					(*entryMap)[strings.TrimPrefix(name, prefix)] = m.Count()
				}
			}
		}
	}
	updateIngress := trafficUpdater(p2p.MetricsInboundTraffic+"/")
	updateEgress := trafficUpdater(p2p.MetricsOutboundTraffic+"/")

	for {
		select {
		case event := <-peerCh:
			pc.Update(&peerEvent{
				MeteredPeerEvent: &event,
				t: peerContainerUpdateType(event.Type),
			})
		case <-ticker.C:
			ingress := make(map[string]int64)
			egress := make(map[string]int64)
			p2p.PeerIngressRegistry.Each(updateIngress(&ingress))
			p2p.PeerEgressRegistry.Each(updateEgress(&egress))
			//fmt.Println(ingress)
			s, _ := json.MarshalIndent(pc, "", "  ")
			fmt.Println(string(s))
		case err := <-subPeer.Err():
			log.Warn("Peer subscription error", "err", err)
			return
		case errc := <-db.quit:
			errc <- nil
			return
		}
	}
}
