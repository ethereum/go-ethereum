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
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/mohae/deepcopy"
)

const eventBufferLimit = 128 // Maximum number of buffered peer events
const trafficEventBufferLimit = p2p.MeteredPeerLimit

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
		// Peer event channels.
		connectCh    = make(chan p2p.PeerConnectEvent, eventBufferLimit)
		failedCh     = make(chan p2p.PeerFailedEvent, eventBufferLimit)
		disconnectCh = make(chan p2p.PeerDisconnectEvent, eventBufferLimit)
		ingressCh    = make(chan p2p.PeerTrafficEvent, trafficEventBufferLimit)
		egressCh     = make(chan p2p.PeerTrafficEvent, trafficEventBufferLimit)

		// Subscribe to peer events.
		subConnect    = p2p.SubscribePeerConnectEvent(connectCh)
		subFailed     = p2p.SubscribePeerFailedEvent(failedCh)
		subDisconnect = p2p.SubscribePeerDisconnectEvent(disconnectCh)
		subIngress    = p2p.SubscribePeerIngressEvent(ingressCh)
		subEgress     = p2p.SubscribePeerEgressEvent(egressCh)
	)
	defer func() {
		// Unsubscribe at the end.
		subConnect.Unsubscribe()
		subFailed.Unsubscribe()
		subDisconnect.Unsubscribe()
		subIngress.Unsubscribe()
		subEgress.Unsubscribe()
	}()

	ticker := time.NewTicker(db.config.Refresh)
	defer ticker.Stop()

	// Listen for events, and prepare the difference between two metering.
	diff := &NetworkMessage{
		PeerBundles: make(map[string]*PeerBundle),
	}
	for {
		select {
		case event := <-connectCh:
			ip := event.IP
			p := diff.getOrInitPeer(ip, event.ID)
			if diff.PeerBundles[ip].Location == nil {
				db.peerLock.RLock()
				lookup := db.history.Network.PeerBundles[ip] == nil || db.history.Network.PeerBundles[ip].Location == nil
				db.peerLock.RUnlock()
				if lookup {
					location := db.geodb.Lookup(net.ParseIP(event.IP))
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
		//case event := <-failedCh:
		//	ip := event.IP
		//	p := diff.getOrInitPeer(ip, event.AutoID)
		//	p.DefaultID = event.AutoID
		//	if p.Handshake == nil {
		//		p.Handshake = []time.Time{event.Handshake}
		//	} else {
		//		p.Handshake = append(p.Handshake, event.Handshake)
		//	}
		//	delete(diff.PeerBundles[ip].Peers, event.AutoID)
		//	diff.getOrInitPeer(ip, event.ID)
		//	diff.PeerBundles[ip].Peers[event.ID] = p // TODO (kurkomisi): Merge instead in order to keep the previous connection.
		//	// Remove the peer from history in case the metering was before the handshake.
		//	db.peerLock.RLock()
		//	stored := db.history.Network.PeerBundles[ip] != nil && db.history.Network.PeerBundles[ip].Peers[event.AutoID] != nil
		//	db.peerLock.RUnlock()
		//	if stored {
		//		db.peerLock.Lock()
		//		hp := db.history.Network.getOrInitPeer(ip, event.AutoID)
		//		delete(db.history.Network.PeerBundles[ip].Peers, event.AutoID)
		//		db.history.Network.getOrInitPeer(ip, event.ID)
		//		db.history.Network.PeerBundles[ip].Peers[event.ID] = hp // TODO (kurkomisi): Merge.
		//		db.peerLock.Unlock()
		//	}
		case event := <-disconnectCh:
			p := diff.getOrInitPeer(event.IP, event.ID)
			if p.Disconnected == nil {
				p.Disconnected = []time.Time{event.Disconnected}
			} else {
				p.Disconnected = append(p.Disconnected, event.Disconnected)
			}
		case event := <-ingressCh:
			fmt.Println("ingress", event.IP, event.Amount)
			// Sum up the ingress between two updates.
			p := diff.getOrInitPeer(event.IP, event.ID)
			if len(p.Ingress) <= 0 {
				p.Ingress = ChartEntries{&ChartEntry{Value: float64(event.Amount)}}
			} else {
				p.Ingress[0].Value += float64(event.Amount)
			}
		case event := <-egressCh:
			fmt.Println("egress ", event.IP, event.Amount)
			// Sum up the egress between two updates.
			p := diff.getOrInitPeer(event.IP, event.ID)
			if len(p.Egress) <= 0 {
				p.Egress = ChartEntries{&ChartEntry{Value: float64(event.Amount)}}
			} else {
				p.Egress[0].Value += float64(event.Amount)
			}
		case <-ticker.C:
			now := time.Now()
			// Merge the diff with the history.
			db.peerLock.Lock()
			for ip, bundle := range diff.PeerBundles {
				if bundle.Location != nil {
					b := db.history.Network.getOrInitBundle(ip)
					b.Location = bundle.Location
				}
				for id, peer := range bundle.Peers {
					peerHistory := db.history.Network.getOrInitPeer(ip, id)
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
		case err := <-subConnect.Err():
			log.Warn("Peer connect subscription error", "err", err)
			return
		case err := <-subFailed.Err():
			log.Warn("Peer failed subscription error", "err", err)
			return
		case err := <-subDisconnect.Err():
			log.Warn("Peer disconnect subscription error", "err", err)
			return
		case err := <-subIngress.Err():
			log.Warn("Peer ingress subscription error", "err", err)
			return
		case err := <-subEgress.Err():
			log.Warn("Peer egress subscription error", "err", err)
			return
		case errc := <-db.quit:
			errc <- nil
			return
		}
	}
}

