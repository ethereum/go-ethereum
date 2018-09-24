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

type PeerContainer struct {
	Bundles map[string]*PeerBundle `json:"peerBundles,omitempty"`

	activeSeparator *list.Element
	knownPeers      *list.List
	unknownPeers    []*PeerBundle

	geodb   *GeoDB
	refresh time.Duration
}

func NewPeerContainer(geodb *GeoDB, refresh time.Duration) *PeerContainer {
	return &PeerContainer{
		Bundles:      make(map[string]*PeerBundle),
		knownPeers:   list.New(),
		unknownPeers: make([]*PeerBundle, 0, unknownPeerLimit),
		geodb:        geodb,
		refresh:      refresh,
	}
}

func (pc *PeerContainer) getOrInitBundle(ip string) *PeerBundle {
	if _, ok := pc.Bundles[ip]; !ok {
		pc.Bundles[ip] = &PeerBundle{
			Location:   pc.geodb.Location(ip),
			KnownPeers: make(map[string]*KnownPeer),
			parent:     pc,
			ip:         ip,
		}
	}
	return pc.Bundles[ip]
}

func (pc *PeerContainer) updateKnown(ip, id string, session *PeerSession) (removedIP, removedID string) {
	peer := pc.getOrInitBundle(ip).getOrInitKnown(id)
	if peer.Sessions == nil {
		peer.Sessions = []*PeerSession{{
			Ingress: emptyChartEntries(time.Now(), 5, pc.refresh),
			Egress:  emptyChartEntries(time.Now(), 5, pc.refresh),
		}}
	}
	if peer.listElement != nil {
		if peer.listElement == pc.activeSeparator {
			pc.activeSeparator = pc.activeSeparator.Prev()
		}
		pc.knownPeers.Remove(peer.listElement)
	}
	if pc.knownPeers.Len() >= knownPeerLimit {
		removed := pc.knownPeers.Remove(pc.knownPeers.Back())
		if p, ok := removed.(*KnownPeer); ok {
			removedIP, removedID = p.parent.ip, p.id
			p.delete()
		} else {
			log.Warn("Bad value used as peer", "value", removed)
		}
	}
	if pc.activeSeparator == nil {
		peer.listElement = pc.knownPeers.PushFront(peer)
	} else {
		peer.listElement = pc.knownPeers.InsertAfter(peer, pc.activeSeparator)
	}
	peer.update(session)
	if peer.Sessions[len(peer.Sessions)-1].Disconnected == nil {
		pc.activeSeparator = peer.listElement
	}
	return removedIP, removedID
}

func (pc *PeerContainer) updateUnknown(ip string, peer *UnknownPeer) {
	if len(pc.unknownPeers) >= unknownPeerLimit {
		removed := pc.unknownPeers[0]
		removed.UnknownPeers = removed.UnknownPeers[1:]
		pc.unknownPeers = pc.unknownPeers[1:]
	}
	bundle := pc.getOrInitBundle(ip)
	bundle.UnknownPeers = append(bundle.UnknownPeers, peer)
	pc.unknownPeers = append(pc.unknownPeers, bundle)
}

func (pc *PeerContainer) clear() {
	for pc.knownPeers.Front() != nil {
		first := pc.knownPeers.Front()
		if p, ok := first.Value.(*KnownPeer); ok {
			p.delete()
		} else {
			log.Warn("Bad value used as peer", "value", first)
		}
	}
	pc.activeSeparator = nil
}

func (pc *PeerContainer) updateTraffic(traffic *peerTraffic) {
	now := time.Now()
	for e := pc.knownPeers.Front(); e != nil; e = e.Next() {
		if p, ok := e.Value.(*KnownPeer); ok {
			key := fmt.Sprintf("%s/%s", p.parent.ip, p.id)
			p.updateTraffic(
				&ChartEntry{Time: now, Value: float64(traffic.ingress[key])},
				&ChartEntry{Time: now, Value: float64(traffic.egress[key])},
			)
		} else {
			log.Warn("Bad value used as peer", "value", e.Value)
		}
	}
}

type PeerBundle struct {
	Location     *GeoLocation          `json:"location,omitempty"`
	KnownPeers   map[string]*KnownPeer `json:"knownPeers,omitempty"`
	UnknownPeers []*UnknownPeer        `json:"unknownPeers,omitempty"`

	parent  *PeerContainer
	ip      string
	element list.Element
}

func (b *PeerBundle) getOrInitKnown(id string) *KnownPeer {
	if _, ok := b.KnownPeers[id]; !ok {
		b.KnownPeers[id] = &KnownPeer{
			parent: b,
			id:     id,
		}
	}
	return b.KnownPeers[id]
}

type KnownPeer struct {
	Sessions []*PeerSession `json:"sessions,omitempty"`

	parent *PeerBundle
	id     string

	listElement *list.Element
}

func (peer *KnownPeer) delete() {
	delete(peer.parent.KnownPeers, peer.id)
	fmt.Println(peer, "to make sure p is not cleared")
	if len(peer.parent.KnownPeers) < 1 && len(peer.parent.UnknownPeers) < 1 {
		delete(peer.parent.parent.Bundles, peer.parent.ip)
	}
	peer.listElement = nil
	peer.parent = nil
	for i := range peer.Sessions {
		peer.Sessions[i] = nil
	}
	peer.Sessions = nil
}

func (peer *KnownPeer) update(session *PeerSession) {
	if session.Connected != nil && session.Disconnected != nil && session.Connected.After(*session.Disconnected) {
		peer.Sessions[len(peer.Sessions)-1].Disconnected = session.Disconnected
		session.Disconnected = nil
	}
	if session.Connected != nil {
		peer.Sessions = append(peer.Sessions, session)
	}
	last := peer.Sessions[len(peer.Sessions)-1]
	if session.Disconnected != nil {
		last.Disconnected = session.Disconnected
	}
	for i := 0; i < len(session.Ingress) && i < len(session.Egress); i++ {
		peer.updateTraffic(session.Ingress[i], session.Egress[i])
	}
}

func (peer *KnownPeer) updateTraffic(ingress, egress *ChartEntry) {
	first, last := peer.Sessions[0], peer.Sessions[len(peer.Sessions)-1]
	if len(first.Ingress) < 2 || len(first.Egress) < 2 {
		peer.Sessions = peer.Sessions[1:]
	} else {
		first.Ingress = first.Ingress[1:]
		first.Egress = first.Egress[1:]
	}
	last.Ingress = append(last.Ingress, ingress)
	last.Egress = append(last.Egress, egress)
}

type PeerSession struct {
	Connected    *time.Time `json:"connected,omitempty"`
	Disconnected *time.Time `json:"disconnected,omitempty"`

	Ingress ChartEntries `json:"ingress,omitempty"`
	Egress  ChartEntries `json:"egress,omitempty"`
}

type UnknownPeer struct {
	Connected    time.Time `json:"connected,omitempty"`
	Disconnected time.Time `json:"disconnected,omitempty"`
}

type peerTraffic struct {
	ingress map[string]int64
	egress  map[string]int64
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

	pc := NewPeerContainer(db.geodb, db.config.Refresh)

	trafficUpdater := func(prefix string) func(*map[string]int64) func(name string, i interface{}) {
		return func(entryMap *map[string]int64) func(name string, i interface{}) {
			return func(name string, i interface{}) {
				if m, ok := i.(metrics.Meter); ok {
					(*entryMap)[strings.TrimPrefix(name, prefix)] = m.Count()
				}
			}
		}
	}
	updateIngress := trafficUpdater(p2p.MetricsInboundTraffic + "/")
	updateEgress := trafficUpdater(p2p.MetricsOutboundTraffic + "/")

	for {
		select {
		case event := <-peerCh:
			now := time.Now()
			switch event.Type {
			case p2p.PeerConnected:
				connected := now.Add(-event.Elapsed)
				pc.updateKnown(event.IP.String(), event.ID, &PeerSession{
					Connected: &connected,
				})
			case p2p.PeerDisconnected:
				pc.updateKnown(event.IP.String(), event.ID, &PeerSession{
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
				pc.updateUnknown(event.IP.String(), &UnknownPeer{
					Connected:    now.Add(-event.Elapsed),
					Disconnected: now,
				})
			default:
				log.Error("Unknown metered peer event type", "type", event.Type)
			}
		case <-ticker.C:
			traffic := peerTraffic{
				ingress: make(map[string]int64),
				egress:  make(map[string]int64),
			}
			p2p.PeerIngressRegistry.Each(updateIngress(&traffic.ingress))
			p2p.PeerEgressRegistry.Each(updateEgress(&traffic.egress))
			pc.updateTraffic(&traffic)
			s, _ := json.MarshalIndent(pc, "", "  ")
			fmt.Println(string(s))
			//db.sendToAll(&Message{Network: pc})
		case err := <-subPeer.Err():
			log.Warn("Peer subscription error", "err", err)
			return
		case errc := <-db.quit:
			errc <- nil
			return
		}
	}
}
