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
	"github.com/mohae/deepcopy"
	"strings"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	eventBufferLimit = 128
	knownPeerLimit   = 3 //p2p.MeteredPeerLimit
	unknownPeerLimit = 3 //p2p.MeteredPeerLimit
)

type knownPeerDiff struct {
	*KnownPeer
	activeListElement *list.Element
	listElement       *list.Element // Pointer to the peer element in the list.
	ip, id            string
}

type peerDiff struct {
	*PeersMessage
	root     *PeersMessage
	rootLock *sync.RWMutex

	knownActivePeerList   *list.List
	knownInactivePeerList *list.List
	unknownPeers          []string

	geodb   *GeoDB
	refresh time.Duration
}

func newPeerDiff(root *PeersMessage, rootLock *sync.RWMutex, geodb *GeoDB, refresh time.Duration) *peerDiff {
	return &peerDiff{
		PeersMessage:          NewPeersMessage(),
		root:                  root,
		rootLock:              rootLock,
		knownActivePeerList:   list.New(),
		knownInactivePeerList: list.New(),
		unknownPeers:          make([]string, 0, unknownPeerLimit),
		geodb:                 geodb,
		refresh:               refresh,
	}
}

func (diff *peerDiff) insert(ip, id string, session *PeerSession) {
	newIP, newID := diff.initKnownPeer(ip, id)
	bundle := diff.Bundles[ip]
	if newIP {
		bundle.Location = diff.geodb.Location(ip)
	}
	peer := &knownPeerDiff{
		KnownPeer: bundle.KnownPeers[id],
		ip:        ip,
		id:        id,
	}
	if newID {
		now := time.Now()
		peer.append(&PeerSession{
			Ingress: emptyChartEntries(now, sampleLimit, diff.refresh),
			Egress:  emptyChartEntries(now, sampleLimit, diff.refresh),
		})
	}
	peer.append(session)
	if peer.activeListElement != nil {
		diff.knownActivePeerList.Remove(peer.activeListElement)
	}
	if peer.listElement != nil {
		diff.knownInactivePeerList.Remove(peer.listElement)
	}
	// Set peer activity
	if len(peer.Sessions) > 0 {
		peer.Active = peer.Sessions[len(peer.Sessions)-1].Disconnected == nil
	} else {
		diff.rootLock.RLock()
		if diff.root.hasKnownPeer(ip, id) {
			rootSessions := diff.root.Bundles[ip].KnownPeers[id].Sessions
			peer.Active = len(rootSessions) > 0 && rootSessions[len(rootSessions)-1].Disconnected == nil
		} else {
			peer.Active = false
		}
		diff.rootLock.RUnlock()
	}
	if peer.Active {
		peer.activeListElement = diff.knownActivePeerList.PushBack(peer)
	} else {
		peer.listElement = diff.knownInactivePeerList.PushBack(peer)
	}
	for diff.knownActivePeerList.Len()+diff.knownInactivePeerList.Len() > knownPeerLimit {
		var removed interface{}
		if diff.knownInactivePeerList.Len() > 0 {
			removed = diff.knownInactivePeerList.Remove(diff.knownInactivePeerList.Front())
		} else {
			removed = diff.knownActivePeerList.Remove(diff.knownActivePeerList.Front())
		}
		if p, ok := removed.(*knownPeerDiff); ok {
			diff.removeKnownPeer(p.ip, p.id)
			diff.rootLock.RLock()
			if diff.root.hasKnownPeer(p.ip, p.id) {
				diff.RemovedKnownIP = append(diff.RemovedKnownIP, p.ip)
				diff.RemovedKnownID = append(diff.RemovedKnownID, p.id)
			}
			diff.rootLock.RUnlock()
		}
	}
}

func (diff *peerDiff) insertUnknown(ip string, peer *UnknownPeer) {
	newBundle := diff.initBundle(ip)
	bundle := diff.Bundles[ip]
	if newBundle {
		bundle.Location = diff.geodb.Location(ip)
	}
	diff.unknownPeers = append(diff.unknownPeers, ip)
	bundle.UnknownPeers = append(bundle.UnknownPeers, peer)
	for len(diff.unknownPeers) > unknownPeerLimit {
		rip := diff.unknownPeers[0]
		diff.RemovedUnknownIP = append(diff.RemovedUnknownIP, rip)
		diff.removeUnknownPeer(rip)
		diff.unknownPeers = diff.unknownPeers[1:]
	}
}

func (diff *peerDiff) dump() {
	diff.rootLock.Lock()
	for i := 0; i < len(diff.RemovedKnownIP); i++ {
		diff.root.removeKnownPeer(diff.RemovedKnownIP[i], diff.RemovedKnownID[i])
	}
	for _, rip := range diff.RemovedUnknownIP {
		diff.root.removeUnknownPeer(rip)
	}
	for e := diff.knownActivePeerList.Front(); e != nil; e = e.Next() {
		if peer, ok := e.Value.(*knownPeerDiff); ok {
			diff.root.getOrInitKnownPeer(peer.ip, peer.id).upgrade(peer.KnownPeer)
		} else {
			log.Warn("Invalid value in the active peer metrics list")
		}
	}
	for e := diff.knownInactivePeerList.Front(); e != nil; e = e.Next() {
		if peer, ok := e.Value.(*knownPeerDiff); ok {
			diff.root.getOrInitKnownPeer(peer.ip, peer.id).upgrade(peer.KnownPeer)
		} else {
			log.Warn("Invalid value in the inactive peer metrics list")
		}
	}
	diff.rootLock.Unlock()
	diff.clear()
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
	type collectorFunc func(traffic *map[string]float64) registryFunc

	trafficCollector := func(prefix string) collectorFunc {
		return func(traffic *map[string]float64) registryFunc {
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

	db.peerLock.Lock()
	db.history.Network = &NetworkMessage{Peers: NewPeersMessage()}
	diff := newPeerDiff(db.history.Network.Peers, &db.peerLock, db.geodb, db.config.Refresh)
	db.peerLock.Unlock()

	for {
		select {
		case event := <-peerCh:
			now := time.Now()
			switch event.Type {
			case p2p.PeerConnected:
				connected := now.Add(-event.Elapsed)
				diff.insert(event.IP.String(), event.ID, &PeerSession{
					Connected: &connected,
				})
			case p2p.PeerDisconnected:
				diff.insert(event.IP.String(), event.ID, &PeerSession{
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
				diff.insertUnknown(event.IP.String(), &UnknownPeer{
					Connected:    now.Add(-event.Elapsed),
					Disconnected: now,
				})
			default:
				log.Error("Unknown metered peer event type", "type", event.Type)
			}
		case <-ticker.C:
			ingress, egress := make(map[string]float64), make(map[string]float64)
			p2p.PeerIngressRegistry.Each(collectIngress(&ingress))
			p2p.PeerEgressRegistry.Each(collectEgress(&egress))

			now := time.Now()
			appendSample := func(key string, ingress, egress float64) {
				if k := strings.Split(key, "/"); len(k) == 2 {
					diff.insert(k[0], k[1], &PeerSession{
						Ingress: ChartEntries{&ChartEntry{
							Time:  now,
							Value: ingress,
						}},
						Egress: ChartEntries{&ChartEntry{
							Time:  now,
							Value: egress,
						}},
					})
				} else {
					log.Warn("Invalid traffic key", "key", key)
				}
			}
			for key, val := range ingress {
				appendSample(key, val, egress[key])
			}
			for key, val := range egress {
				if _, ok := ingress[key]; ok {
					continue
				}
				appendSample(key, ingress[key], val)
			}
			for e := diff.knownInactivePeerList.Front(); e != nil; e = e.Next() {
				if peer, ok := e.Value.(*knownPeerDiff); ok {
					diff.insert(peer.ip, peer.id, &PeerSession{
						Ingress: ChartEntries{&ChartEntry{
							Time: now,
						}},
						Egress: ChartEntries{&ChartEntry{
							Time: now,
						}},
					})
				}
			}
			db.sendToAll(&Message{Network: &NetworkMessage{Peers: deepcopy.Copy(diff.PeersMessage).(*PeersMessage)}})
			s, _ := json.MarshalIndent(deepcopy.Copy(diff), "", "  ")
			fmt.Println(string(s))
			diff.dump()
		case err := <-subPeer.Err():
			log.Warn("Peer subscription error", "err", err)
			return
		case errc := <-db.quit:
			errc <- nil
			return
		}
	}
}
