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
	eventBufferLimit = 128
	knownPeerLimit   = p2p.MeteredPeerLimit
	unknownPeerLimit = p2p.MeteredPeerLimit
)

// PeersMessage contains information about the node's peers. This data structure
// tries to maintain the metered peer data based on the different behaviours of
// the peers.
//
// Every peer has an IP address, and the peers that manage to make the handshake
// have node IDs. There can appear more peers with the same IP, therefore the
// peer maintainer data structure is a tree consisting of a map of maps, where
// the first key groups the peers by IP, while the second one groups them by the
// node ID. The peers failing before the handshake only have IP addresses, so
// they are stored as part of the value of the outer map. A peer can connect
// multiple times, so an array is needed to store the consecutive sessions.
//
// Another criteria is to limit the number of metered peers so that they don't
// fill the memory. The metered peers are selected based on their activity: the
// peers that are inactive for the longest time are thrown first. For the selection a fifo
// list is used which is linked to the bottom of the peer tree, and when a peer
// is removed from the list, it is also removed from the tree. The active peers
// have priority over the disconnected ones, therefore this list is extended by
// a separator, which is a pointer to a list element. The separator separates the
// active peers from the inactive ones, and it is the entry for the list. If the
// peer that is to be inserted is active, it goes before the separator, otherwise
// it goes after. This way the peers that are active for the longest time are at
// the beginning of the list, and the inactive ones move to the end. When a peer
// has some activity, it is removed from and reinserted into the list.
//
// The peers that don't manage to make handshake are not inserted into the list,
// their sessions are not stored, only their connection attempts are appended
// to the array belonging to their IP. In order to keep the fifo principle, a super
// array contains the order of the attempts, and when the peer count reaches the
// limit, the earliest attempt is removed from the beginning of its array.
type PeersMessage struct {
	// Bundles is the outer map using the peer's IP address as key.
	Bundles map[string]*PeerBundle `json:"bundles,omitempty"`

	RemovedKnown   []string `json:"removedKnown,omitempty"`
	RemovedUnknown []string `json:"removedUnknown,omitempty"`

	// activeSeparator is a pointer to the last active peer element, splitting
	// the list into an active and inactive part, and forming the entry for the
	// peer list.
	activeSeparator *list.Element

	// knownPeers contains the peers that managed to make handshake.
	knownPeers *list.List

	// unknownPeers contains pointers to the peer bundles that belong to the
	// IP addresses, from which the peers attempted to connect then failed.
	// Its values are appended in the moment of the attempt, so in chronological
	// order, which means that the oldest attempt is at the beginning of the array.
	// When the first element is removed, the first element of the linked bundle's
	// attempt array is also removed, ensuring that always the latest attempts are stored.
	unknownPeers []*PeerBundle

	// geodb is used to look up the geographical data based on the IP.
	geodb *GeoDB
}

// NewPeersMessage returns a new instance of the metered peer maintainer data structure.
func NewPeersMessage(geodb *GeoDB) *PeersMessage {
	return &PeersMessage{
		Bundles:      make(map[string]*PeerBundle),
		knownPeers:   list.New(),
		unknownPeers: make([]*PeerBundle, 0, unknownPeerLimit),
		geodb:        geodb,
	}
}

func (m *PeersMessage) hasBundle(ip string) bool {
	_, ok := m.Bundles[ip]
	return ok
}

func (m *PeersMessage) hasPeer(ip, id string) bool {
	if !m.hasBundle(ip) {
		return false
	}
	return m.Bundles[ip].has(id)
}

// getOrInitBundle returns the bundle belonging to the given IP.
// Inserts a new bundle into the map if it doesn't already exist.
func (m *PeersMessage) getOrInitBundle(ip string) *PeerBundle {
	if _, ok := m.Bundles[ip]; !ok {
		m.Bundles[ip] = &PeerBundle{
			Location:   m.geodb.Location(ip),
			KnownPeers: make(map[string]*KnownPeer),
			root:       m,
			ip:         ip,
		}
	}
	return m.Bundles[ip]
}

func (m *PeersMessage) getOrInitKnownPeer(ip, id string) *KnownPeer {
	return m.getOrInitBundle(ip).getOrInitKnownPeer(id)
}

// updateKnownPeer updates the last session of the peer belonging to the given IP
// and ID and returns the IP and the ID of the removed peer if there is any,
// (nil, nil) otherwise.
func (m *PeersMessage) updateKnownPeer(ip, id string, session *PeerSession) (removedKey string) {
	peer := m.getOrInitKnownPeer(ip, id)
	if peer.listElement != nil {
		// If the peer is already part of the list, remove it first.
		if peer.listElement == m.activeSeparator {
			m.activeSeparator = m.activeSeparator.Prev()
		}
		m.knownPeers.Remove(peer.listElement)
	}
	if m.knownPeers.Len() >= knownPeerLimit {
		// If the peer count reached the limit, remove the last element of the
		// list, which is the oldest inactive one.
		removed := m.knownPeers.Remove(m.knownPeers.Back())
		if p, ok := removed.(*KnownPeer); ok {
			removedKey = fmt.Sprintf("%s/%s", p.bundle.ip, p.id)
			p.delete() // Remove the peer from the tree.
		} else {
			log.Warn("Bad value used as peer", "value", removed)
		}
	}
	if m.activeSeparator == nil {
		// If there isn't any active peer in the list, push the new peer to the
		// front of the list.
		peer.listElement = m.knownPeers.PushFront(peer)
	} else {
		// If there are active peers in the list, push the new peer after them.
		peer.listElement = m.knownPeers.InsertAfter(peer, m.activeSeparator)
	}
	peer.update(session)
	if peer.Sessions[len(peer.Sessions)-1].Disconnected == nil {
		// If the new peer is active, step to it with the separator.
		m.activeSeparator = peer.listElement
	}
	return removedKey
}

// updateUnknownPeer inserts a peer connection attempt into the peer tree.
func (m *PeersMessage) updateUnknownPeer(ip string, peer *UnknownPeer) (removedKey string) {
	if len(m.unknownPeers) >= unknownPeerLimit {
		// If the count of the metered unknown peers reached the limit,
		// remove the oldest attempt, which is the first element of the
		// array belonging to the peer bundle pointed by the first element
		// of the super unknown peer array.
		removed := m.unknownPeers[0]
		removed.UnknownPeers = removed.UnknownPeers[1:]
		m.unknownPeers = m.unknownPeers[1:]
		removedKey = removed.ip
	}
	bundle := m.getOrInitBundle(ip)
	bundle.UnknownPeers = append(bundle.UnknownPeers, peer)
	m.unknownPeers = append(m.unknownPeers, bundle)

	return removedKey
}

// clear removes the elements of the metered peer list, and removes the linked
// peers from the peer tree along the way.
func (m *PeersMessage) clear() {
	for m.knownPeers.Front() != nil {
		first := m.knownPeers.Remove(m.knownPeers.Front())
		if p, ok := first.(*KnownPeer); ok {
			p.delete()
		} else {
			log.Warn("Bad value used as peer", "value", first)
		}
	}
	m.activeSeparator = nil
}

func (m *PeersMessage) updateTraffic(ip, id string, ingress, egress float64) {
	if !m.hasPeer(ip, id) {
		m.updateKnownPeer(ip, id, &PeerSession{
			Ingress: ChartEntries{&ChartEntry{
				Value: ingress,
			}},
			Egress: ChartEntries{&ChartEntry{
				Value: egress,
			}},
		})
		return
	}
	m.getOrInitKnownPeer(ip, id).updateTraffic(
		&ChartEntry{Value: ingress},
		&ChartEntry{Value: egress},
	)
}

//func (m *PeersMessage) updateTraffic(ingress, egress *map[string]float64) {
//	now := time.Now()
//	for k, v := range *ingress {
//
//	}
//	for e := m.knownPeers.Front(); e != nil; e = e.Next() {
//		if p, ok := e.Value.(*KnownPeer); ok {
//			key := fmt.Sprintf("%s/%s", p.bundle.ip, p.id)
//			p.updateTraffic(
//				&ChartEntry{Time: now, Value: (*ingress)[key]},
//				&ChartEntry{Time: now, Value: (*egress)[key]},
//			)
//		} else {
//			log.Warn("Bad value used as peer", "value", e.Value)
//		}
//	}
//}

// append
func (m *PeersMessage) append(n *PeersMessage) (removedKnown, removedUnknown []string) {
	for _, bundle := range n.Bundles {
		for _, peer := range bundle.KnownPeers {
			for _, session := range peer.Sessions {
				removedKey := m.updateKnownPeer(bundle.ip, peer.id, session)
				if removedKey != "" {
					removedKnown = append(removedKnown, removedKey)
				}
			}
		}
		for _, peer := range bundle.UnknownPeers {
			removedKey := m.updateUnknownPeer(bundle.ip, peer)
			if removedKey != "" {
				removedUnknown = append(removedUnknown, removedKey)
			}
		}
	}
	return removedKnown, removedUnknown
}

// PeerBundle contains the peers belonging to a given IP address
type PeerBundle struct {
	Location *GeoLocation `json:"location,omitempty"` // Geographical location based on IP

	// KnownPeers is the inner map of the metered peer maintainer data structure
	// using the node ID as key.
	KnownPeers map[string]*KnownPeer `json:"knownPeers,omitempty"`

	// UnknownPeers contains the failed connection attempts of the peers
	// belonging to a given IP address in chronological order.
	UnknownPeers []*UnknownPeer `json:"unknownPeers,omitempty"`

	root *PeersMessage // Pointer to the outer map.
	ip   string        // Key of the bundle in the outer map.
}

func (b *PeerBundle) has(id string) bool {
	_, ok := b.KnownPeers[id]
	return ok
}

// getOrInitKnownPeer returns the peer belonging to the given ID.
// Initializes it if it doesn't already exist.
func (b *PeerBundle) getOrInitKnownPeer(id string) *KnownPeer {
	if _, ok := b.KnownPeers[id]; !ok {
		b.KnownPeers[id] = &KnownPeer{
			sampleCount: sampleLimit,
			bundle:      b,
			id:          id,
		}
	}
	return b.KnownPeers[id]
}

// KnownPeer contains the metered data of a particular peer.
type KnownPeer struct {
	// Sessions contains the metered data in a session of a peer.
	Sessions []*PeerSession `json:"sessions,omitempty"`

	sampleCount int

	bundle *PeerBundle // Pointer to the inner map.
	id     string      // Key of the peer in the inner map.

	listElement *list.Element // Pointer to the peer element in the list.
}

func (peer *KnownPeer) delete() {
	delete(peer.bundle.KnownPeers, peer.id)
	if len(peer.bundle.KnownPeers) < 1 && len(peer.bundle.UnknownPeers) < 1 {
		delete(peer.bundle.root.Bundles, peer.bundle.ip)
	}
	peer.listElement = nil
	peer.bundle = nil
	for i := range peer.Sessions {
		peer.Sessions[i] = nil
	}
	peer.Sessions = nil
}

func (peer *KnownPeer) update(session *PeerSession) {
	if peer.Sessions == nil {
		peer.Sessions = []*PeerSession{session}
		if session.Connected != nil && session.Disconnected != nil && session.Connected.After(*session.Disconnected) {
			peer.Sessions = append(peer.Sessions, &PeerSession{Connected: session.Connected})
			session.Connected = nil
		}
		return
	}
	if session.Connected != nil && session.Disconnected != nil && session.Connected.After(*session.Disconnected) {
		last := peer.Sessions[len(peer.Sessions)-1]
		if last.Connected == nil {
			// This may happen at the beginning, if the connection was established
			// before the initialization of the peer event feed. Otherwise it should
			// not happen to handle a disconnect event before the connect, because
			// both of them are coming on the same channel consecutively.
			log.Warn("Disconnect event appeared without connect")
		}
		last.Disconnected = session.Disconnected
		last.Ingress = append(last.Ingress, session.Ingress...)
		last.Egress = append(last.Egress, session.Egress...)
		session.Disconnected = nil
		session.Ingress = nil
		session.Egress = nil
	}
	if session.Connected != nil {
		peer.Sessions = append(peer.Sessions, session)
	}
	if session.Disconnected != nil {
		peer.Sessions[len(peer.Sessions)-1].Disconnected = session.Disconnected
	}
	for i := 0; i < len(session.Ingress) && i < len(session.Egress); i++ {
		peer.updateTraffic(session.Ingress[i], session.Egress[i])
	}
}

func (peer *KnownPeer) updateTraffic(ingress, egress *ChartEntry) {
	if ingress == nil {
		ingress = new(ChartEntry)
	}
	if egress == nil {
		egress = new(ChartEntry)
	}
	first := peer.Sessions[0]
	if len(first.Ingress) < 1 || len(first.Egress) < 1 {
		first.Ingress = append(first.Ingress, ingress)
		first.Egress = append(first.Egress, egress)
		peer.sampleCount = 1
		return
	}
	if peer.sampleCount >= sampleLimit {
		if len(first.Ingress) < 2 || len(first.Egress) < 2 {
			peer.Sessions = peer.Sessions[1:]
		} else {
			first.Ingress = first.Ingress[1:]
			first.Egress = first.Egress[1:]
		}
		peer.sampleCount--
	}
	last := peer.Sessions[len(peer.Sessions)-1]
	last.Ingress = append(last.Ingress, ingress)
	last.Egress = append(last.Egress, egress)
	peer.sampleCount++
}

func (peer *KnownPeer) len() int {
	return peer.sampleCount
}

type PeerSession struct {
	Connected    *time.Time `json:"connected,omitempty"`
	Disconnected *time.Time `json:"disconnected,omitempty"`

	Ingress ChartEntries `json:"ingress,omitempty"`
	Egress  ChartEntries `json:"egress,omitempty"`
}

type UnknownPeer struct {
	Connected    time.Time `json:"connected"`
	Disconnected time.Time `json:"disconnected"`
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

	db.peerLock.Lock()
	db.history.Peers = NewPeersMessage(db.geodb)
	db.peerLock.Unlock()
	diff := NewPeersMessage(db.geodb)

	trafficCollector := func(prefix string) func(*map[string]float64) func(name string, i interface{}) {
		return func(traffic *map[string]float64) func(name string, i interface{}) {
			return func(name string, i interface{}) {
				if m, ok := i.(metrics.Meter); ok {
					(*traffic)[strings.TrimPrefix(name, prefix)] = float64(m.Count())
				} else {
					log.Warn("Bad value used as meter", "name", name)
				}
			}
		}
	}
	collectIngress := trafficCollector(p2p.MetricsInboundTraffic + "/")
	collectEgress := trafficCollector(p2p.MetricsOutboundTraffic + "/")

	for {
		select {
		case event := <-peerCh:
			now := time.Now()
			switch event.Type {
			case p2p.PeerConnected:
				connected := now.Add(-event.Elapsed)
				diff.updateKnownPeer(event.IP.String(), event.ID, &PeerSession{
					Connected: &connected,
				})
			case p2p.PeerDisconnected:
				diff.updateKnownPeer(event.IP.String(), event.ID, &PeerSession{
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
			case p2p.PeerHandshakeFailed:
				diff.updateUnknownPeer(event.IP.String(), &UnknownPeer{
					Connected:    now.Add(-event.Elapsed),
					Disconnected: now,
				})
			default:
				log.Error("Unknown metered peer event type", "type", event.Type)
			}
		case <-ticker.C:
			ingress, egress := make(map[string]float64), make(map[string]float64)
			db.peerLock.Lock()
			for e := db.history.Peers.knownPeers.Front(); e != nil; e = e.Next() {
				if p, ok := e.Value.(*KnownPeer); ok {
					key := fmt.Sprintf("%s/%s", p.bundle.ip, p.id)
					ingress[key] = 0
					egress[key] = 0
				}
			}
			p2p.PeerIngressRegistry.Each(collectIngress(&ingress))
			p2p.PeerEgressRegistry.Each(collectEgress(&egress))
			//diff.updateTraffic(&ingress, &egress)
			for key := range ingress {
				if k := strings.Split(key, "/"); len(k) == 2 {
					diff.updateTraffic(k[0], k[1], ingress[key], egress[key])
				} else {
					log.Warn("Bad key", "key", key)
				}
			}
			for key := range egress {
				if _, ok := ingress[key]; !ok {
					if k := strings.Split(key, "/"); len(k) == 2 {
						diff.updateTraffic(k[0], k[1], ingress[key], egress[key])
					} else {
						log.Warn("Bad key", "key", key)
					}
				}
			}
			for _, bundle := range diff.Bundles {
				for _, peer := range bundle.KnownPeers {
					var ipExists, idExists bool
					var b *PeerBundle
					if b, ipExists = db.history.Peers.Bundles[bundle.ip]; ipExists {
						_, idExists = b.KnownPeers[peer.id]
					}
					if !idExists || !ipExists {
						fmt.Println("doesn't exist ", bundle.ip, peer.id)
						//t, n := time.Now(), sampleLimit-peer.len()
						//if len(peer.Sessions) > 0 && peer.Sessions[0].Connected != nil {
						//	t = *peer.Sessions[0].Connected
						//}
						//peer.Sessions = append([]*PeerSession{{
						//	Ingress: emptyChartEntries(t, n, db.config.Refresh),
						//	Egress:  emptyChartEntries(t, n, db.config.Refresh),
						//}}, peer.Sessions...)
					} else {
						fmt.Println("does exist    ", bundle.ip, peer.id)
						s, _ := json.MarshalIndent(bundle, "", "  ")
						fmt.Println(string(s))
						fmt.Println(ingress[fmt.Sprintf("%s/%s", bundle.ip, peer.id)])
						fmt.Println(egress[fmt.Sprintf("%s/%s", bundle.ip, peer.id)])
					}
				}
			}
			diff.RemovedKnown, diff.RemovedUnknown = db.history.Peers.append(diff)
			//sh, _ := json.MarshalIndent(db.history.Network, "", "  ")
			//fmt.Println(string(sh))
			db.peerLock.Unlock()
			//s, _ := json.MarshalIndent(diff, "", "  ")
			//fmt.Println(string(s))
			//db.sendToAll(&Message{Peers: diff})
			diff.clear()
		case err := <-subPeer.Err():
			log.Warn("Peer subscription error", "err", err)
			return
		case errc := <-db.quit:
			errc <- nil
			return
		}
	}
}
