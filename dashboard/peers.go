// Copyright 2019 The go-ethereum Authors
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
	"reflect"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/metrics"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	knownPeerLimit = 100 // Maximum number of stored peers, which successfully made the handshake.

	// eventLimit is the maximum number of the dashboard's custom peer events,
	// that are collected between two metering period and sent to the clients
	// as one message.
	// TODO (kurkomisi): Limit the number of events.
	eventLimit = knownPeerLimit << 2
)

// peerContainer contains information about the node's peers. This data structure
// maintains the metered peer data based on the different behaviours of the peers.
//
// Every peer has an IP address, and the peers that manage to make the handshake
// (known peers) have node IDs too. There can appear more peers with the same IP,
// therefore the peer container data structure is a tree consisting of a map of
// maps, where the first key groups the peers by IP, while the second one groups
// them by the node ID. The known peers can be active if their connection is still
// open, or inactive otherwise. The peers failing before the handshake (unknown
// peers) only have IP addresses, so their connection attempts are stored as part
// of the value of the outer map.
//
// Another criteria is to limit the number of metered peers so that
// they don't fill the memory. The selection order is based on the
// peers activity: the peers that are inactive for the longest time
// are thrown first. For the selection a fifo list is used which is
// linked to the bottom of the peer tree in a way that every activity
// of the peer pushes the peer to the end of the list, so the inactive
// ones come to the front. When a peer has some activity, it is removed
// from and reinserted into the list. When the length of the list reaches
// the limit, the first element is removed from the list, as well as from
// the tree.
//
// The active peers have priority over the inactive ones, therefore
// they have their own list. The separation makes it sure that the
// inactive peers are always removed before the active ones.
//
// The peers that don't manage to make handshake are not inserted into the list,
// only their connection attempts are appended to the array belonging to their IP.
// In order to keep the fifo principle, a super array contains the order of the
// attempts, and when the overall count reaches the limit, the earliest attempt is
// removed from the beginning of its array.
//
// This data structure makes it possible to marshal the peer
// history simply by passing it to the JSON marshaler.
type peerContainer struct {
	// Bundles is the outer map using the peer's IP address as key.
	Bundles map[string]*peerBundle `json:"bundles,omitempty"`

	activeCount int // Number of the still connected peers

	// inactivePeers contains the peers with closed connection in chronological order.
	inactivePeers *list.List

	// geodb is the geoip database used to retrieve the peers' geographical location.
	geodb *geoDB
}

// newPeerContainer returns a new instance of the peer container.
func newPeerContainer(geodb *geoDB) *peerContainer {
	return &peerContainer{
		Bundles:       make(map[string]*peerBundle),
		inactivePeers: list.New(),
		geodb:         geodb,
	}
}

// bundle inserts a new peer bundle into the map, if the peer belonging
// to the given IP wasn't metered so far. In this case retrieves the location of
// the IP address from the database and creates a corresponding peer event.
// Returns the bundle belonging to the given IP and the events occurring during
// the initialization.
func (pc *peerContainer) bundle(addr string) (*peerBundle, []*peerEvent) {
	var events []*peerEvent
	if _, ok := pc.Bundles[addr]; !ok {
		i := strings.IndexByte(addr, ':')
		if i < 0 {
			i = len(addr)
		}
		location := pc.geodb.location(addr[:i])
		events = append(events, &peerEvent{
			Addr:     addr,
			Location: location,
		})
		pc.Bundles[addr] = &peerBundle{
			Location:   location,
			KnownPeers: make(map[string]*knownPeer),
		}
	}
	return pc.Bundles[addr], events
}

// extendKnown handles the events of the successfully connected peers.
// Returns the events occurring during the extension.
func (pc *peerContainer) extendKnown(event *peerEvent) []*peerEvent {
	bundle, events := pc.bundle(event.Addr)
	peer, peerEvents := bundle.knownPeer(event.Addr, event.Enode)
	events = append(events, peerEvents...)
	// Append the connect and the disconnect events to
	// the corresponding arrays keeping the limit.
	switch {
	case event.Connected != nil: // Handshake succeeded
		peer.Connected = append(peer.Connected, event.Connected)
		if first := len(peer.Connected) - sampleLimit; first > 0 {
			peer.Connected = peer.Connected[first:]
		}
		if event.peer == nil {
			log.Warn("Peer handshake succeeded event without peer instance", "addr", event.Addr, "enode", event.Enode)
		}
		peer.peer = event.peer
		info := event.peer.Info()
		peer.Name = info.Name
		peer.Protocols = info.Protocols
		peer.Active = true
		e := &peerEvent{
			Activity:  Active,
			Name:      info.Name,
			Addr:      peer.addr,
			Enode:     peer.enode,
			Protocols: peer.Protocols,
		}
		events = append(events, e)
		pc.activeCount++
		if peer.listElement != nil {
			_ = pc.inactivePeers.Remove(peer.listElement)
			peer.listElement = nil
		}
	case event.Disconnected != nil: // Peer disconnected
		peer.Disconnected = append(peer.Disconnected, event.Disconnected)
		if first := len(peer.Disconnected) - sampleLimit; first > 0 {
			peer.Disconnected = peer.Disconnected[first:]
		}
		peer.Active = false
		events = append(events, &peerEvent{
			Activity: Inactive,
			Addr:     peer.addr,
			Enode:    peer.enode,
		})
		pc.activeCount--
		if peer.listElement != nil {
			// If the peer is already in the list, remove and reinsert it.
			_ = pc.inactivePeers.Remove(peer.listElement)
		}
		// Insert the peer into the list.
		peer.listElement = pc.inactivePeers.PushBack(peer)
	default:
		log.Warn("Unexpected known peer event", "event", *event)
	}
	for pc.inactivePeers.Len() > 0 && pc.activeCount+pc.inactivePeers.Len() > knownPeerLimit {
		// While the count of the known peers is greater than the limit,
		// remove the first element from the inactive peer list and from the map.
		if removedPeer, ok := pc.inactivePeers.Remove(pc.inactivePeers.Front()).(*knownPeer); ok {
			events = append(events, pc.removeKnown(removedPeer.addr, removedPeer.enode)...)
		} else {
			log.Warn("Failed to parse the removed peer")
		}
	}
	if pc.activeCount > knownPeerLimit {
		log.Warn("Number of active peers is greater than the limit")
	}
	return events
}

// peerBundle contains the peers belonging to a given IP address.
type peerBundle struct {
	// Location contains the geographical location based on the bundle's IP address.
	Location *geoLocation `json:"location,omitempty"`

	// KnownPeers is the inner map of the metered peer
	// maintainer data structure using the node ID as key.
	KnownPeers map[string]*knownPeer `json:"knownPeers,omitempty"`

	// Attempts contains the count of the failed connection
	// attempts of the peers belonging to a given IP address.
	Attempts uint `json:"attempts,omitempty"`
}

// removeKnown removes the known peer belonging to the
// given IP address and node ID from the peer tree.
func (pc *peerContainer) removeKnown(addr, enode string) (events []*peerEvent) {
	// TODO (kurkomisi): Remove peers that don't have traffic samples anymore.
	if bundle, ok := pc.Bundles[addr]; ok {
		if _, ok := bundle.KnownPeers[enode]; ok {
			events = append(events, &peerEvent{
				Remove: RemoveKnown,
				Addr:   addr,
				Enode:  enode,
			})
			delete(bundle.KnownPeers, enode)
		} else {
			log.Warn("No peer to remove", "addr", addr, "enode", enode)
		}
		if len(bundle.KnownPeers) < 1 && bundle.Attempts < 1 {
			events = append(events, &peerEvent{
				Remove: RemoveBundle,
				Addr:   addr,
			})
			delete(pc.Bundles, addr)
		}
	} else {
		log.Warn("No bundle to remove", "addr", addr)
	}
	return events
}

// knownPeer inserts a new peer into the map, if the peer belonging
// to the given IP address and node ID wasn't metered so far. Returns the peer
// belonging to the given IP and ID as well as the events occurring during the
// initialization.
func (bundle *peerBundle) knownPeer(addr, enode string) (*knownPeer, []*peerEvent) {
	var events []*peerEvent
	if _, ok := bundle.KnownPeers[enode]; !ok {
		ingress := emptyChartEntries(sampleLimit)
		egress := emptyChartEntries(sampleLimit)
		events = append(events, &peerEvent{
			Addr:    addr,
			Enode:   enode,
			Ingress: append([]*ChartEntry{}, ingress...),
			Egress:  append([]*ChartEntry{}, egress...),
		})
		bundle.KnownPeers[enode] = &knownPeer{
			addr:    addr,
			enode:   enode,
			Ingress: ingress,
			Egress:  egress,
		}
	}
	return bundle.KnownPeers[enode], events
}

// knownPeer contains the metered data of a particular peer.
type knownPeer struct {
	// Connected contains the timestamps of the peer's connection events.
	Connected []*time.Time `json:"connected,omitempty"`

	// Disconnected contains the timestamps of the peer's disconnection events.
	Disconnected []*time.Time `json:"disconnected,omitempty"`

	// Ingress and Egress contain the peer's traffic samples, which are collected
	// periodically from the metrics registry.
	//
	// A peer can connect multiple times, and we want to visualize the time
	// passed between two connections, so after the first connection a 0 value
	// is appended to the traffic arrays even if the peer is inactive until the
	// peer is removed.
	Ingress ChartEntries `json:"ingress,omitempty"`
	Egress  ChartEntries `json:"egress,omitempty"`

	Name      string                 `json:"name,omitempty"`      // Name of the node, including client type, version, OS, custom data
	Enode     string                 `json:"enode,omitempty"`     // Node URL
	Protocols map[string]interface{} `json:"protocols,omitempty"` // Sub-protocol specific metadata fields

	Active bool `json:"active"` // Denotes if the peer is still connected.

	listElement *list.Element // Pointer to the peer element in the list.
	addr, enode string        // The IP and the ID by which the peer can be accessed in the tree.
	prevIngress float64
	prevEgress  float64

	peer *p2p.Peer // Connected remote node instance
}

type RemovedPeerType string
type ActivityType string

const (
	RemoveKnown  RemovedPeerType = "known"
	RemoveBundle RemovedPeerType = "bundle"

	Active   ActivityType = "active"
	Inactive ActivityType = "inactive"
)

// peerEvent contains the attributes of a peer event.
type peerEvent struct {
	Name         string                 `json:"name,omitempty"`         // Name of the node, including client type, version, OS, custom data
	Addr         string                 `json:"addr,omitempty"`         // TCP address of the peer.
	Enode        string                 `json:"enode,omitempty"`        // Node URL
	Protocols    map[string]interface{} `json:"protocols,omitempty"`    // Sub-protocol specific metadata fields
	Remove       RemovedPeerType        `json:"remove,omitempty"`       // Type of the peer that is to be removed.
	Location     *geoLocation           `json:"location,omitempty"`     // Geographical location of the peer.
	Connected    *time.Time             `json:"connected,omitempty"`    // Timestamp of the connection moment.
	Disconnected *time.Time             `json:"disconnected,omitempty"` // Timestamp of the disonnection moment.
	Ingress      ChartEntries           `json:"ingress,omitempty"`      // Ingress samples.
	Egress       ChartEntries           `json:"egress,omitempty"`       // Egress samples.
	Activity     ActivityType           `json:"activity,omitempty"`     // Connection status change.

	peer *p2p.Peer // Connected remote node instance.
}

// trafficMap is a container for the periodically collected peer traffic.
type trafficMap map[string]map[string]float64

// insert inserts a new value to the traffic map. Overwrites
// the value at the given ip and id if that already exists.
func (m *trafficMap) insert(ip, id string, val float64) {
	if _, ok := (*m)[ip]; !ok {
		(*m)[ip] = make(map[string]float64)
	}
	(*m)[ip][id] = val
}

// collectPeerData gathers data about the peers and sends it to the clients.
func (db *Dashboard) collectPeerData() {
	defer db.wg.Done()

	// Open the geodb database for IP to geographical information conversions.
	var err error
	db.geodb, err = openGeoDB()
	if err != nil {
		log.Warn("Failed to open geodb", "err", err)
		errc := <-db.quit
		errc <- nil
		return
	}
	defer db.geodb.close()

	ticker := time.NewTicker(db.config.Refresh)
	defer ticker.Stop()

	type registryFunc func(name string, i interface{})
	type collectorFunc func(traffic *trafficMap) registryFunc

	// trafficCollector generates a function that can be passed to
	// the prefixed peer registry in order to collect the metered
	// traffic data from each peer meter.
	trafficCollector := func(prefix string) collectorFunc {
		// This part makes is possible to collect the
		// traffic data into a map from outside.
		return func(traffic *trafficMap) registryFunc {
			// The function which can be passed to the registry.
			return func(name string, i interface{}) {
				if m, ok := i.(metrics.Meter); ok {
					enode := strings.TrimPrefix(name, prefix)
					if addr := strings.Split(enode, "@"); len(addr) == 2 {
						traffic.insert(addr[1], enode, float64(m.Count()))
					} else {
						log.Warn("Invalid enode", "enode", enode)
					}
				} else {
					log.Warn("Invalid meter type", "name", name)
				}
			}
		}
	}
	collectIngress := trafficCollector(p2p.MetricsInboundTraffic + "/")
	collectEgress := trafficCollector(p2p.MetricsOutboundTraffic + "/")

	peers := newPeerContainer(db.geodb)
	db.peerLock.Lock()
	db.history.Network = &NetworkMessage{
		Peers: peers,
	}
	db.peerLock.Unlock()

	// newPeerEvents contains peer events, which trigger operations that
	// will be executed on the peer tree after a metering period.
	newPeerEvents := make([]*peerEvent, 0, eventLimit)
	ingress, egress := new(trafficMap), new(trafficMap)
	*ingress, *egress = make(trafficMap), make(trafficMap)

	defer db.subPeer.Unsubscribe()
	for {
		select {
		case event := <-db.peerCh:
			now := time.Now()
			switch event.Type {
			case p2p.PeerHandshakeFailed:
				connected := now.Add(-event.Elapsed)
				newPeerEvents = append(newPeerEvents, &peerEvent{
					Addr:         event.Addr,
					Connected:    &connected,
					Disconnected: &now,
				})
			case p2p.PeerHandshakeSucceeded:
				connected := now.Add(-event.Elapsed)
				newPeerEvents = append(newPeerEvents, &peerEvent{
					Addr:      event.Addr,
					Enode:     event.Peer.Node().String(),
					peer:      event.Peer,
					Connected: &connected,
				})
			case p2p.PeerDisconnected:
				addr, enode := event.Addr, event.Peer.Node().String()
				newPeerEvents = append(newPeerEvents, &peerEvent{
					Addr:         addr,
					Enode:        enode,
					Disconnected: &now,
				})
				// The disconnect event comes with the last metered traffic count,
				// because after the disconnection the peer's meter is removed
				// from the registry. It can happen, that between two metering
				// period the same peer disconnects multiple times, and appending
				// all the samples to the traffic arrays would shift the metering,
				// so only the last metering is stored, overwriting the previous one.
				ingress.insert(addr, enode, float64(event.Ingress))
				egress.insert(addr, enode, float64(event.Egress))
			default:
				log.Error("Unknown metered peer event type", "type", event.Type)
			}
		case <-ticker.C:
			// Collect the traffic samples from the registry.
			p2p.PeerIngressRegistry.Each(collectIngress(ingress))
			p2p.PeerEgressRegistry.Each(collectEgress(egress))

			// Protect 'peers', because it is part of the history.
			db.peerLock.Lock()

			var diff []*peerEvent
			for i := 0; i < len(newPeerEvents); i++ {
				if newPeerEvents[i].Addr == "" {
					log.Warn("Peer event without IP", "event", *newPeerEvents[i])
					continue
				}
				diff = append(diff, newPeerEvents[i])
				// There are two main branches of peer events coming from the event
				// feed, one belongs to the known peers, one to the unknown peers.
				// If the event has node ID, it belongs to a known peer, otherwise
				// to an unknown one, which is considered as connection attempt.
				//
				// The extension can produce additional peer events, such
				// as remove, location and initial samples events.
				if newPeerEvents[i].Enode == "" {
					bundle, events := peers.bundle(newPeerEvents[i].Addr)
					bundle.Attempts++
					diff = append(diff, events...)
					continue
				}
				diff = append(diff, peers.extendKnown(newPeerEvents[i])...)
			}
			// Update the peer tree using the traffic maps.
			for addr, bundle := range peers.Bundles {
				for enode, peer := range bundle.KnownPeers {
					// Value is 0 if the traffic map doesn't have the
					// entry corresponding to the given IP and ID.
					curIngress, curEgress := (*ingress)[addr][enode], (*egress)[addr][enode]
					deltaIngress, deltaEgress := curIngress, curEgress
					if deltaIngress >= peer.prevIngress {
						deltaIngress -= peer.prevIngress
					}
					if deltaEgress >= peer.prevEgress {
						deltaEgress -= peer.prevEgress
					}
					peer.prevIngress, peer.prevEgress = curIngress, curEgress
					i := &ChartEntry{
						Value: deltaIngress,
					}
					e := &ChartEntry{
						Value: deltaEgress,
					}
					peer.Ingress = append(peer.Ingress, i)
					peer.Egress = append(peer.Egress, e)
					if first := len(peer.Ingress) - sampleLimit; first > 0 {
						peer.Ingress = peer.Ingress[first:]
					}
					if first := len(peer.Egress) - sampleLimit; first > 0 {
						peer.Egress = peer.Egress[first:]
					}
					// Creating the traffic sample events.
					diff = append(diff, &peerEvent{
						Addr:    addr,
						Enode:   enode,
						Ingress: ChartEntries{i},
						Egress:  ChartEntries{e},
					})
					if peer.peer != nil {
						info := peer.peer.Info()
						if !reflect.DeepEqual(peer.Protocols, info.Protocols) {
							peer.Protocols = info.Protocols
							diff = append(diff, &peerEvent{
								Addr:      addr,
								Enode:     enode,
								Protocols: peer.Protocols,
							})
						}
					}
				}
			}
			db.peerLock.Unlock()

			if len(diff) > 0 {
				db.sendToAll(&Message{Network: &NetworkMessage{
					Diff: diff,
				}})
			}
			// Clear the traffic maps, and the event array,
			// prepare them for the next metering.
			*ingress, *egress = make(trafficMap), make(trafficMap)
			newPeerEvents = newPeerEvents[:0]
		case err := <-db.subPeer.Err():
			log.Warn("Peer subscription error", "err", err)
			errc := <-db.quit
			errc <- nil
			return
		case errc := <-db.quit:
			errc <- nil
			return
		}
	}
}
