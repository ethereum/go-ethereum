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
	"github.com/mohae/deepcopy"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
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

	purgeOrder := list.New()
	failedPurgeOrder := list.New()
	update := func(peer *Peer, l *list.List) {
		if peer.element == nil {
			peer.element = purgeOrder.PushBack(peer)
		} else {
			purgeOrder.MoveToBack(peer.element)
		}
	}
	// Listen for events, and prepare the difference between two metering.
	diff := &NetworkMessage{
		PeerBundles: make(map[string]*PeerBundle),
	}
	for {
		select {
		case event := <-connectCh:
			diffBundle := diff.getOrInitBundle(event.IP)
			diffBundle.Location = db.geodb.Location(event.IP)
			diffPeer := diffBundle.getOrInitPeer(event.ID)
			diffPeer.Connected = append(diffPeer.Connected, event.Connected)
		case event := <-failedCh:
			diffBundle := diff.getOrInitBundle(event.IP)
			diffBundle.Location = db.geodb.Location(event.IP)
			diffBundle.FailedPeers = append(diffBundle.FailedPeers, &Peer{
				Connected: []time.Time{event.Connected},
				Disconnected: []time.Time{event.Disconnected},
			})
		case event := <-disconnectCh:
			diffPeer := diff.getOrInitPeer(event.IP, event.ID)
			diffPeer.Disconnected = append(diffPeer.Disconnected, event.Disconnected)
		case event := <-ingressCh:
			diffPeer := diff.getOrInitPeer(event.IP, event.ID)
			if len(diffPeer.Ingress) != 1 {
				diffPeer.Ingress = ChartEntries{&ChartEntry{Value: float64(event.Amount)}}
			} else {
				diffPeer.Ingress[0].Value = float64(event.Amount)
			}
		case event := <-egressCh:
			diffPeer := diff.getOrInitPeer(event.IP, event.ID)
			if len(diffPeer.Egress) != 1 {
				diffPeer.Egress = ChartEntries{&ChartEntry{Value: float64(event.Amount)}}
			} else {
				diffPeer.Egress[0].Value = float64(event.Amount)
			}
		case <-ticker.C:
			now := time.Now()
			// Merge the diff with the history.
			db.peerLock.Lock()
			for ip, diffBundle := range diff.PeerBundles {
				historyBundle := db.history.Network.getOrInitBundle(ip)
				historyBundle.Location = diffBundle.Location
				for id, diffPeer := range diffBundle.Peers {
					historyPeer := historyBundle.getOrInitPeer(id)
					historyPeer.Connected = append(historyPeer.Connected, diffPeer.Connected...)
					historyPeer.Disconnected = append(historyPeer.Disconnected, diffPeer.Disconnected...)
					if len(diffPeer.Ingress) == 1 {
						diffPeer.Ingress[0].Time = now
						if historyPeer.Ingress == nil {
							historyPeer.Ingress = append(emptyChartEntries(now.Add(-db.config.Refresh), sampleLimit-1, db.config.Refresh), diffPeer.Ingress[0])
							// The first message about a diffPeer should contain the whole list
							diffPeer.Ingress = historyPeer.Ingress
						} else {
							historyPeer.Ingress = append(historyPeer.Ingress, diffPeer.Ingress[0])[1:]
						}
					}
					if len(diffPeer.Egress) == 1 {
						diffPeer.Egress[0].Time = now
						if historyPeer.Egress == nil {
							historyPeer.Egress = append(emptyChartEntries(now.Add(-db.config.Refresh), sampleLimit-1, db.config.Refresh), diffPeer.Egress[0])
							// The first message about a diffPeer should contain the whole list
							diffPeer.Egress = historyPeer.Egress
						} else {
							historyPeer.Egress = append(historyPeer.Egress, diffPeer.Egress[0])[1:]
						}
					}
					update(historyPeer, purgeOrder)
				}
				historyBundle.FailedPeers = append(historyBundle.FailedPeers, diffBundle.FailedPeers...)
				for _, fp := range diffBundle.FailedPeers {
					update(fp, failedPurgeOrder)
				}
			}
			for purgeOrder.Len() > p2p.MeteredPeerLimit {
				purgeOrder.Remove(purgeOrder.Front())
			}
			for failedPurgeOrder.Len() > p2p.MeteredPeerLimit {
				failedPurgeOrder.Remove(failedPurgeOrder.Front())
			}

			//ss, _ := json.MarshalIndent(db.history.Network, "", "   ")
			//fmt.Println(string(ss))
			db.peerLock.Unlock()
			// Send the diff to the clients.
			db.sendToAll(&Message{Network: deepcopy.Copy(diff).(*NetworkMessage)})
			//s, _ := json.MarshalIndent(diff, "", "   ")
			//fmt.Println(string(s))

			s, _ := json.MarshalIndent(diff, "", "  ")
			fmt.Println(string(s))
			// Prepare for the next metering, clear the diff variable.
			diff = &NetworkMessage{
				PeerBundles: make(map[string]*PeerBundle),
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

