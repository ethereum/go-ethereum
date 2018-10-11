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
	"github.com/ethereum/go-ethereum/metrics"
	"strings"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	eventBufferLimit = 128                  // Maximum number of events of buffered peer events.
	knownPeerLimit   = p2p.MeteredPeerLimit // Maximum number of stored peers, which successfully made the handshake.
	unknownPeerLimit = p2p.MeteredPeerLimit // Maximum number of stored peers, which failed to make the handshake.

	// eventLimit is the maximum number of the dashboard's custom peer events,
	// that are collected between two metering period and sent to the clients
	// as one message.
	// TODO (kurkomisi): Limit the number of events.
	eventLimit = knownPeerLimit << 2
)

// PeerContainer contains information about the node's peers. This data structure
// maintains the metered peer data based on the different behaviours of the peers.
//
// Every peer has an IP address, and the peers that manage to make the handshake
// (known peers) have node IDs too. There can appear more peers with the same IP,
// therefore the peer maintainer data structure is a tree consisting of a map of
// maps, where the first key groups the peers by IP, while the second one groups
// them by the node ID. The peers failing before the handshake (unknown peers)
// only have IP addresses, so their connection attempts are stored as part of the
// value of the outer map.
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
// The active peers that are still connected have priority over the disconnected
// ones, therefore the list is extended by a separator, which is a pointer to a
// list element. The separator separates the active peers from the inactive ones,
// and it is the entry for the list. If the peer that is to be inserted is active,
// it goes after the separator, otherwise it goes before. This way the active peers
// never move to the front before the inactive ones.
//
// The peers that don't manage to make handshake are not inserted into the list,
// only their connection attempts are appended to the array belonging to their IP.
// In order to keep the fifo principle, a super array contains the order of the
// attempts, and when the overall count reaches the limit, the earliest attempt is
// removed from the beginning of its array.
//
// This data structure makes it possible to marshal the peer
// history simply by passing it to the JSON marshaler.
type PeerContainer struct {
	// Bundles is the outer map using the peer's IP address as key.
	Bundles map[string]*PeerBundle `json:"bundles,omitempty"`

	// activeSeparator is a pointer to the last inactive peer element, splitting
	// the list into an inactive and an active part, and forming the entry for
	// the peer list.
	activeSeparator *list.Element

	// knownPeers contains the peers that managed to make handshake.
	knownPeers *list.List

	// unknownPeers is the super array containing the IP addresses, from which
	// the peers attempted to connect then failed before/during the handshake.
	// Its values are appended in chronological order, which means that the
	// oldest attempt is at the beginning of the array. When the first element
	// is removed, the first element of the related bundle's attempt array is
	// removed too, ensuring that always the latest attempts are stored.
	unknownPeers []string

	// geodb is the geoip database used to retrieve the peers' geographical location.
	geodb *GeoDB

	// refresh is the refresh rate used to generate the
	// initial auxiliary traffic samples' time stamps.
	refresh time.Duration
}

// NewPeerContainer returns a new instance of the peer container.
func NewPeerContainer(geodb *GeoDB, refresh time.Duration) *PeerContainer {
	return &PeerContainer{
		Bundles:      make(map[string]*PeerBundle),
		knownPeers:   list.New(),
		unknownPeers: make([]string, 0, unknownPeerLimit),
		geodb:        geodb,
		refresh:      refresh,
	}
}

// getOrInitBundle inserts a new peer bundle into the map, if the peer belonging
// to the given IP wasn't metered so far. In this case retrieves the location of
// the IP address from the database and creates a corresponding peer event.
// Returns the bundle belonging to the given IP and the events occurring during
// the initialization.
func (pc *PeerContainer) getOrInitBundle(ip string) (*PeerBundle, []*PeerEvent) {
	var events []*PeerEvent
	if _, ok := pc.Bundles[ip]; !ok {
		location := pc.geodb.Location(ip)
		events = append(events, &PeerEvent{
			IP:       ip,
			Location: location,
		})
		pc.Bundles[ip] = &PeerBundle{
			Location:   location,
			KnownPeers: make(map[string]*KnownPeer),
		}
	}
	return pc.Bundles[ip], events
}

// extendKnown handles the events of the successfully connected peers.
// Returns the events occurring during the extension.
func (pc *PeerContainer) extendKnown(event *PeerEvent) (events []*PeerEvent) {
	bundle, bundleInitEvents := pc.getOrInitBundle(event.IP)
	events = append(events, bundleInitEvents...)
	peer, peerInitEvents := bundle.getOrInitKnownPeer(event.IP, event.ID, pc.refresh)
	events = append(events, peerInitEvents...)
	if peer.listElement != nil {
		// If the peer is already in the list, remove it and reinsert later.
		pc.knownPeers.Remove(peer.listElement)
		peer.listElement = nil
	}
	for pc.knownPeers.Len() >= knownPeerLimit {
		// While the length of the list is greater than or equal to the limit before
		// the insertion, remove the first element from the list and from the map.
		if removedPeer, ok := pc.knownPeers.Remove(pc.knownPeers.Front()).(*KnownPeer); ok {
			if event := pc.removeKnown(removedPeer.ip, removedPeer.id); event != nil {
				events = append(events, event)
			}
		}
	}
	// Append the connect and the disconnect events to
	// the corresponding arrays keeping the limit.
	if event.Connected != nil {
		peer.Connected = append(peer.Connected, event.Connected)
		if first := len(peer.Connected) - sampleLimit; first > 0 {
			peer.Connected = peer.Connected[first:]
		}
	}
	if event.Disconnected != nil {
		peer.Disconnected = append(peer.Disconnected, event.Disconnected)
		if first := len(peer.Disconnected) - sampleLimit; first > 0 {
			peer.Disconnected = peer.Disconnected[first:]
		}
	}
	// Insert the peer into the list.
	if pc.activeSeparator == nil {
		// If there isn't active peer in the list
		peer.listElement = pc.knownPeers.PushBack(peer)
		pc.activeSeparator = peer.listElement
	} else {
		// Insert the peer after the last inactive peer, then increment the separator.
		peer.listElement = pc.knownPeers.InsertAfter(peer, pc.activeSeparator)
		pc.activeSeparator = pc.activeSeparator.Next()
	}
	return events
}

// extendUnknown handles the events of the peers failing before/during the handshake.
// Returns the events occurring during the extension.
func (pc *PeerContainer) extendUnknown(event *PeerEvent) (events []*PeerEvent) {
	bundle, initEvents := pc.getOrInitBundle(event.IP)
	events = append(events, initEvents...)
	bundle.UnknownPeers = append(bundle.UnknownPeers, &UnknownPeer{
		Connected:    *event.Connected,
		Disconnected: *event.Disconnected,
	})
	pc.unknownPeers = append(pc.unknownPeers, event.IP)
	for len(pc.unknownPeers) > unknownPeerLimit {
		// While the length of the connection attempt order array is greater
		// than the limit, remove the first element from the involved peer's
		// array and also from the super array.
		if r := pc.removeUnknown(pc.unknownPeers[0]); r != nil {
			events = append(events, r)
		}
		pc.unknownPeers = pc.unknownPeers[1:]
	}
	return events
}

// setActive moves the peer denoted by the given IP address and node ID after
// the list's active separator.
func (pc *PeerContainer) setActive(ip, id string) {
	if bundle, ok := pc.Bundles[ip]; ok {
		if peer, ok := bundle.KnownPeers[id]; ok {
			if peer.listElement != nil {
				// If the peer is already in the list, remove it first.
				// Theoretically this should always happen, because all
				// the peers are inserted into the list.
				pc.knownPeers.Remove(peer.listElement)
			}
			if pc.activeSeparator == nil {
				// If there isn't active peer yet.
				peer.listElement = pc.knownPeers.PushBack(peer)
				pc.activeSeparator = peer.listElement
			} else {
				peer.listElement = pc.knownPeers.InsertAfter(peer, pc.activeSeparator)
			}
		}
	}
}

// resetActiveSeparator resets the active separator, denoting
// that active peers are not considered active anymore.
func (pc *PeerContainer) resetActiveSeparator() {
	pc.activeSeparator = nil
}

// PeerBundle contains the peers belonging to a given IP address.
type PeerBundle struct {
	// Location contains the geographical location based on the bundle's IP address.
	Location *GeoLocation `json:"location,omitempty"`

	// KnownPeers is the inner map of the metered peer
	// maintainer data structure using the node ID as key.
	KnownPeers map[string]*KnownPeer `json:"knownPeers,omitempty"`

	// UnknownPeers contains the failed connection attempts of the
	// peers belonging to a given IP address in chronological order.
	UnknownPeers []*UnknownPeer `json:"unknownPeers,omitempty"`
}

// removeKnown removes the known peer belonging to the
// given IP address and node ID from the peer tree.
func (pc *PeerContainer) removeKnown(ip, id string) (removed *PeerEvent) {
	if bundle, ok := pc.Bundles[ip]; ok {
		if _, ok := bundle.KnownPeers[id]; ok {
			removed = &PeerEvent{
				RemoveID: id,
			}
			delete(bundle.KnownPeers, id)
		}
		if len(bundle.KnownPeers) < 1 && len(bundle.UnknownPeers) < 1 {
			if removed == nil {
				removed = &PeerEvent{
					RemoveIP: ip,
				}
			} else {
				removed.RemoveIP = ip
			}
			delete(pc.Bundles, ip)
		}
	}
	return removed
}

// removeUnknown removes the unknown peer belonging to the
// given IP address and node ID from the peer tree.
func (pc *PeerContainer) removeUnknown(ip string) (removed *PeerEvent) {
	if bundle, ok := pc.Bundles[ip]; ok {
		if len(bundle.UnknownPeers) > 0 {
			bundle.UnknownPeers = bundle.UnknownPeers[1:]
		}
		if len(bundle.KnownPeers) < 1 && len(bundle.UnknownPeers) < 1 {
			removed = &PeerEvent{RemoveIP: ip}
			delete(pc.Bundles, ip)
		}
	}
	return removed
}

// getOrInitKnownPeer inserts a new peer into the map, if the peer belonging
// to the given IP address and node ID wasn't metered so far. Returns the peer
// belonging to the given IP and ID as well as the events occurring during the
// initialization.
func (bundle *PeerBundle) getOrInitKnownPeer(ip, id string, refresh time.Duration) (*KnownPeer, []*PeerEvent) {
	var events []*PeerEvent
	if _, ok := bundle.KnownPeers[id]; !ok {
		now := time.Now()
		ingress := emptyChartEntries(now, sampleLimit, refresh)
		egress := emptyChartEntries(now, sampleLimit, refresh)
		events = append(events, &PeerEvent{
			IP:      ip,
			ID:      id,
			Ingress: ingress,
			Egress:  egress,
		})
		bundle.KnownPeers[id] = &KnownPeer{
			ip:      ip,
			id:      id,
			Ingress: ingress,
			Egress:  egress,
		}
	}
	return bundle.KnownPeers[id], events
}

// KnownPeer contains the metered data of a particular peer.
type KnownPeer struct {
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

	listElement *list.Element // Pointer to the peer element in the list.
	ip, id      string        // The IP and the ID by which the peer can be accessed in the tree.
}

// UnknownPeer contains a failed peer connection attempt's attributes.
type UnknownPeer struct {
	// Connected contains the timestamp of the connection attempt's moment.
	Connected time.Time `json:"connected"`

	// Disconnected contains the timestamp of the
	// moment when the connection attempt failed.
	Disconnected time.Time `json:"disconnected"`
}

// PeerEvent contains the attributes of a peer event.
type PeerEvent struct {
	IP           string       `json:"ip,omitempty"`           // IP address of the peer.
	ID           string       `json:"id,omitempty"`           // Node ID of the peer.
	RemoveIP     string       `json:"removeIP,omitempty"`     // IP address of the peer that is to be removed.
	RemoveID     string       `json:"removeID,omitempty"`     // Node ID of the peer that is to be removed.
	Location     *GeoLocation `json:"location,omitempty"`     // Geographical location of the peer.
	Connected    *time.Time   `json:"connected,omitempty"`    // Timestamp of the connection moment.
	Disconnected *time.Time   `json:"disconnected,omitempty"` // Timestamp of the disonnection moment.
	Ingress      ChartEntries `json:"ingress,omitempty"`      // Ingress samples.
	Egress       ChartEntries `json:"egress,omitempty"`       // Egress samples.
}

// trafficMap
type trafficMap map[string]map[string]float64

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
					// The name of the meter has the format: <common traffic prefix><IP>/<ID>
					if k := strings.Split(strings.TrimPrefix(name, prefix), "/"); len(k) == 2 {
						traffic.insert(k[0], k[1], float64(m.Count()))
					} else {
						log.Warn("Invalid meter name", "name", name, "prefix", prefix)
					}
				} else {
					log.Warn("Invalid meter type", "name", name)
				}
			}
		}
	}
	collectIngress := trafficCollector(p2p.MetricsInboundTraffic + "/")
	collectEgress := trafficCollector(p2p.MetricsOutboundTraffic + "/")

	peers := NewPeerContainer(db.geodb, db.config.Refresh)
	db.peerLock.Lock()
	db.history.Network = &NetworkMessage{
		Peers: peers,
	}
	db.peerLock.Unlock()

	// diff contains peer events, which trigger operations that
	// will be executed on the peer tree after a metering period.
	diff := make([]*PeerEvent, 0, eventLimit)
	ingress, egress := new(trafficMap), new(trafficMap)
	*ingress, *egress = make(trafficMap), make(trafficMap)

	for {
		select {
		case event := <-peerCh:
			now := time.Now()
			switch event.Type {
			case p2p.PeerConnected:
				connected := now.Add(-event.Elapsed)
				diff = append(diff, &PeerEvent{
					IP:        event.IP.String(),
					ID:        event.ID,
					Connected: &connected,
				})
			case p2p.PeerDisconnected:
				diff = append(diff, &PeerEvent{
					IP:           event.IP.String(),
					ID:           event.ID,
					Disconnected: &now,
				})
				// The disconnect event comes with the last metered traffic count,
				// because after the disconnection the peer's meter is removed
				// from the registry. It can happen, that between two metering
				// period the same peer disconnects multiple times, and appending
				// all the samples to the traffic arrays would shift the metering,
				// so only the last metering is stored, overwriting the previous one.
				ingress.insert(event.IP.String(), event.ID, float64(event.Ingress))
				egress.insert(event.IP.String(), event.ID, float64(event.Egress))
			case p2p.PeerHandshakeFailed:
				connected := now.Add(-event.Elapsed)
				diff = append(diff, &PeerEvent{
					IP:           event.IP.String(),
					Connected:    &connected,
					Disconnected: &now,
				})
			default:
				log.Error("Unknown metered peer event type", "type", event.Type)
			}
		case <-ticker.C:
			// Collect the traffic samples from the registry.
			p2p.PeerIngressRegistry.Each(collectIngress(ingress))
			p2p.PeerEgressRegistry.Each(collectEgress(egress))

			db.peerLock.Lock()
			// Usually the active peers don't produce events, and marking
			// them as active makes it sure that they won't be removed from
			// the tree. Only the active peers are registered into the peer
			// registry, so after the traffic collection the ingress and the
			// egress maps contain all the active peers.
			//
			// It is important to mark the active ones before the merge with
			// the diff, otherwise the active peers can be removed.
			//
			// After a metering period the active peers can become inactive,
			// so resetting the separator makes it sure, that only the active
			// peers move to the protected part of the list.
			peers.resetActiveSeparator()
			for ip, bundle := range *ingress {
				for id := range bundle {
					if _, ok := (*egress)[ip][id]; ok {
						peers.setActive(ip, id)
					}
				}
			}
			var events []*PeerEvent
			for i := 0; i < len(diff); i++ {
				if diff[i].IP == "" {
					log.Warn("Peer event without IP", "event", *diff[i])
					continue
				}
				// There are two main branches of peer events coming from the event
				// feed, one belongs to the known peers, one to the unknown peers.
				// If the event has node ID, it belongs to a known peer, otherwise
				// to an unknown one.
				if diff[i].ID == "" {
					events = append(events, peers.extendUnknown(diff[i])...)
					continue
				}
				events = append(events, peers.extendKnown(diff[i])...)
			}
			// The insertion can produce additional peer events, such
			// as remove, location and initial samples events. These
			// are stored in the 'events', then appended to the 'diff'.
			diff = append(diff, events...)
			now := time.Now()
			// Update the peer tree using the events.
			for ip, bundle := range peers.Bundles {
				for id, peer := range bundle.KnownPeers {
					// Value is 0 if the traffic map doesn't have the
					// entry corresponding to the given IP and ID.
					i := &ChartEntry{
						Time:  now,
						Value: (*ingress)[ip][id],
					}
					e := &ChartEntry{
						Time:  now,
						Value: (*egress)[ip][id],
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
					diff = append(diff, &PeerEvent{
						IP:      ip,
						ID:      id,
						Ingress: ChartEntries{i},
						Egress:  ChartEntries{e},
					})
				}
			}
			db.peerLock.Unlock()

			db.sendToAll(&Message{Network: &NetworkMessage{
				Diff: append([]*PeerEvent{}, diff...)},
			})
			// Clear the traffic maps, and the event array,
			// prepare them for the next metering.
			*ingress, *egress = make(trafficMap), make(trafficMap)
			diff = diff[:0]
		case err := <-subPeer.Err():
			log.Warn("Peer subscription error", "err", err)
			return
		case errc := <-db.quit:
			errc <- nil
			return
		}
	}
}
