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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/mohae/deepcopy"
)

const eventBufferLimit = 128 // Maximum number of buffered peer events for each event type

// getOrInitBundle returns the peer bundle belonging to the given IP, or
// initializes the bundle if it doesn't exist.
func getOrInitBundle(m *NetworkMessage, ip string) *PeerBundle {
	if _, ok := m.PeerBundles[ip]; !ok {
		m.PeerBundles[ip] = &PeerBundle{
			Peers: make(map[string]*Peer),
		}
	}
	return m.PeerBundles[ip]
}

// getOrInitPeer returns the peer belonging to the given IP and node id, or
// initializes the peer if it doesn't exist.
func getOrInitPeer(m *NetworkMessage, ip, id string) *Peer {
	b := getOrInitBundle(m, ip)
	if _, ok := b.Peers[id]; !ok {
		b.Peers[id] = new(Peer)
	}
	return b.Peers[id]
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

	var (
		quit = make(chan struct{})

		// Channels used for avoiding the blocking of the event feeds.
		connectCh    = make(chan *p2p.PeerConnectEvent, eventBufferLimit)
		handshakeCh  = make(chan *p2p.PeerHandshakeEvent, eventBufferLimit)
		disconnectCh = make(chan *p2p.PeerDisconnectEvent, eventBufferLimit)
		readCh       = make(chan *p2p.PeerReadEvent, eventBufferLimit)
		writeCh      = make(chan *p2p.PeerWriteEvent, eventBufferLimit)
	)
	go func() {
		var (
			// Peer event channels.
			peerConnectEventCh    = make(chan p2p.PeerConnectEvent, eventBufferLimit)
			peerHandshakeEventCh  = make(chan p2p.PeerHandshakeEvent, eventBufferLimit)
			peerDisconnectEventCh = make(chan p2p.PeerDisconnectEvent, eventBufferLimit)
			peerReadEventCh       = make(chan p2p.PeerReadEvent, eventBufferLimit)
			peerWriteEventCh      = make(chan p2p.PeerWriteEvent, eventBufferLimit)

			// Subscribe to peer events.
			subConnect    = p2p.SubscribePeerConnectEvent(peerConnectEventCh)
			subHandshake  = p2p.SubscribePeerHandshakeEvent(peerHandshakeEventCh)
			subDisconnect = p2p.SubscribePeerDisconnectEvent(peerDisconnectEventCh)
			subRead       = p2p.SubscribePeerReadEvent(peerReadEventCh)
			subWrite      = p2p.SubscribePeerWriteEvent(peerWriteEventCh)
		)
		defer func() {
			// Unsubscribe at the end.
			subConnect.Unsubscribe()
			subHandshake.Unsubscribe()
			subDisconnect.Unsubscribe()
			subRead.Unsubscribe()
			subWrite.Unsubscribe()
		}()
		// Waiting for peer events.
		for {
			select {
			case event := <-peerConnectEventCh:
				select {
				case connectCh <- &event:
				default:
					log.Warn("Failed to handle peer connect event", "event", event)
				}
			case event := <-peerHandshakeEventCh:
				select {
				case handshakeCh <- &event:
				default:
					log.Warn("Failed to handle peer handshake event", "event", event)
				}
			case event := <-peerDisconnectEventCh:
				select {
				case disconnectCh <- &event:
				default:
					log.Warn("Failed to handle peer disconnect event", "event", event)
				}
			case event := <-peerReadEventCh:
				select {
				case readCh <- &event:
				default:
					log.Warn("Failed to handle peer read event", "event", event)
				}
			case event := <-peerWriteEventCh:
				select {
				case writeCh <- &event:
				default:
					log.Warn("Failed to handle peer write event", "event", event)
				}
			case err := <-subConnect.Err():
				log.Warn("Peer connect subscription error", "err", err)
				return
			case err := <-subHandshake.Err():
				log.Warn("Peer handshake subscription error", "err", err)
				return
			case err := <-subDisconnect.Err():
				log.Warn("Peer disconnect subscription error", "err", err)
				return
			case err := <-subRead.Err():
				log.Warn("Peer read subscription error", "err", err)
				return
			case err := <-subWrite.Err():
				log.Warn("Peer write subscription error", "err", err)
				return
			case <-quit:
				return
			}
		}
	}()
	go db.keepPeerHistoryClean(quit)

	ticker := time.NewTicker(db.config.Refresh)
	defer ticker.Stop()

	// Listen for events, and prepare the difference between two metering.
	diff := &NetworkMessage{
		PeerBundles: make(map[string]*PeerBundle),
	}
	for {
		select {
		case event := <-connectCh:
			ip := event.IP.String()
			p := getOrInitPeer(diff, ip, event.ID)
			if diff.PeerBundles[ip].Location == nil {
				db.peerLock.RLock()
				lookup := db.networkHistory.PeerBundles[ip] == nil || db.networkHistory.PeerBundles[ip].Location == nil
				db.peerLock.RUnlock()
				if lookup {
					location := db.geodb.Lookup(event.IP)
					diff.PeerBundles[ip].Location = &GeoLocation{
						Country:   location.Country.Names.English,
						City:      location.City.Names.English,
						Latitude:  location.Location.Latitude,
						Longitude: location.Location.Longitude,
					}
				}
			}
			if p.Connected == nil {
				p.Connected = []time.Time{event.Connected}
			} else {
				p.Connected = append(p.Connected, event.Connected)
			}
		case event := <-handshakeCh:
			ip := event.IP.String()
			p := getOrInitPeer(diff, ip, event.DefaultID)
			p.DefaultID = event.DefaultID
			if p.Handshake == nil {
				p.Handshake = []time.Time{event.Handshake}
			} else {
				p.Handshake = append(p.Handshake, event.Handshake)
			}
			delete(diff.PeerBundles[ip].Peers, event.DefaultID)
			getOrInitPeer(diff, ip, event.ID)
			diff.PeerBundles[ip].Peers[event.ID] = p // TODO (kurkomisi): Merge instead in order to keep the previous connection.
			// Remove the peer from history in case the metering was before the handshake.
			db.peerLock.RLock()
			stored := db.networkHistory.PeerBundles[ip] != nil && db.networkHistory.PeerBundles[ip].Peers[event.DefaultID] != nil
			db.peerLock.RUnlock()
			if stored {
				db.peerLock.Lock()
				hp := getOrInitPeer(db.networkHistory, ip, event.DefaultID)
				delete(db.networkHistory.PeerBundles[ip].Peers, event.DefaultID)
				getOrInitPeer(db.networkHistory, ip, event.ID)
				db.networkHistory.PeerBundles[ip].Peers[event.ID] = hp // TODO (kurkomisi): Merge.
				db.peerLock.Unlock()
			}
		case event := <-disconnectCh:
			p := getOrInitPeer(diff, event.IP.String(), event.ID)
			if p.Disconnected == nil {
				p.Disconnected = []time.Time{event.Disconnected}
			} else {
				p.Disconnected = append(p.Disconnected, event.Disconnected)
			}
		case event := <-readCh:
			// Sum up the ingress between two updates.
			p := getOrInitPeer(diff, event.IP.String(), event.ID)
			if len(p.Ingress) <= 0 {
				p.Ingress = ChartEntries{&ChartEntry{Value: float64(event.Ingress)}}
			} else {
				p.Ingress[0].Value += float64(event.Ingress)
			}
		case event := <-writeCh:
			// Sum up the egress between two updates.
			p := getOrInitPeer(diff, event.IP.String(), event.ID)
			if len(p.Egress) <= 0 {
				p.Egress = ChartEntries{&ChartEntry{Value: float64(event.Egress)}}
			} else {
				p.Egress[0].Value += float64(event.Egress)
			}
		case <-ticker.C:
			now := time.Now()
			// Merge the diff with the history.
			db.peerLock.Lock()
			for ip, bundle := range diff.PeerBundles {
				if bundle.Location != nil {
					b := getOrInitBundle(db.networkHistory, ip)
					b.Location = bundle.Location
				}
				for id, peer := range bundle.Peers {
					peerHistory := getOrInitPeer(db.networkHistory, ip, id)
					if peer.Connected != nil {
						peerHistory.Connected = append(peerHistory.Connected, peer.Connected...)
					}
					if peer.Handshake != nil {
						peerHistory.Handshake = append(peerHistory.Handshake, peer.Handshake...)
					}
					if peer.Disconnected != nil {
						peerHistory.Disconnected = append(peerHistory.Disconnected, peer.Disconnected...)
					}
					ingress := &ChartEntry{
						Time: now,
					}
					if len(peer.Ingress) > 0 {
						ingress.Value = peer.Ingress[0].Value
					}
					if peerHistory.Ingress == nil {
						peer.Ingress = append(emptyChartEntries(now.Add(-db.config.Refresh), peerIngressSampleLimit-1, db.config.Refresh), ingress)
						peerHistory.Ingress = peer.Ingress
					} else {
						peer.Ingress = ChartEntries{ingress}
						peerHistory.Ingress = append(peerHistory.Ingress[1:], ingress)
					}
					egress := &ChartEntry{
						Time: now,
					}
					if len(peer.Egress) > 0 {
						egress.Value = peer.Egress[0].Value
					}
					if peerHistory.Egress == nil {
						peer.Egress = append(emptyChartEntries(now.Add(-db.config.Refresh), peerEgressSampleLimit-1, db.config.Refresh), egress)
						peerHistory.Egress = peer.Egress
					} else {
						peer.Egress = ChartEntries{egress}
						peerHistory.Egress = append(peerHistory.Egress[1:], egress)
					}
				}
			}
			db.peerLock.Unlock()
			// Send the diff to the clients.
			db.sendToAll(&Message{Network: deepcopy.Copy(diff).(*NetworkMessage)})

			// Prepare for the next metering, clear the diff variable.
			for ip, bundle := range diff.PeerBundles {
				for id := range bundle.Peers {
					bundle.Peers[id] = nil
					delete(bundle.Peers, id)
				}
				delete(diff.PeerBundles, ip)
			}
		case errc := <-db.quit:
			close(quit)
			errc <- nil
			return
		}
	}
}

// keepPeerHistoryClean purges the stored peer metrics with a given rate in
// order to decrease the load. The inactive peers that disconnected before
// the calculated time will be deleted. If the total amount of peers exceeds
// the limit, the surplus will be chosen from the disconnected ones in the
// iteration order, and will be deleted as well.
func (db *Dashboard) keepPeerHistoryClean(quit chan struct{}) {
	cleanRate := db.config.Refresh * peerTrafficSampleLimit
	for {
		select {
		case <-time.After(cleanRate):
			validAfter := time.Now().Add(-cleanRate)
			db.peerLock.Lock()
			for ip, bundle := range db.networkHistory.PeerBundles {
				bundle.Location = nil
				for id, peer := range bundle.Peers {
					if len(peer.Disconnected) > 0 && peer.Disconnected[len(peer.Disconnected)-1].Before(validAfter) {
						bundle.Peers[id] = nil
						delete(bundle.Peers, id)
					}
				}
				if len(bundle.Peers) <= 0 {
					delete(db.networkHistory.PeerBundles, ip)
				}
			}
			// TODO (kurkomisi): Check the limit during the insertion.
			var lenCount int
			for _, bundle := range db.networkHistory.PeerBundles {
				lenCount += len(bundle.Peers)
			}
			if lenCount > peerLimit {
			outerLoop:
				for ip, bundle := range db.networkHistory.PeerBundles {
					bundle.Location = nil
					for id, peer := range bundle.Peers {
						if peer.Disconnected != nil {
							bundle.Peers[id] = nil
							delete(bundle.Peers, id)
							lenCount--
							if lenCount <= peerLimit {
								if len(bundle.Peers) <= 0 {
									delete(bundle.Peers, ip)
								}
								break outerLoop
							}
						}
					}
					if len(bundle.Peers) <= 0 {
						delete(db.networkHistory.PeerBundles, ip)
					}
				}
			}
			db.peerLock.Unlock()
		case <-quit:
			return
		}
	}
}
