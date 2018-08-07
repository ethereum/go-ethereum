package dashboard

import (
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"time"
	"github.com/mohae/deepcopy"
)

const eventBufferLimit = 128

func getOrInitPeer(m *NetworkMessage, ip, id string) *Peer {
	if _, ok := m.Peers[ip]; !ok {
		m.Peers[ip] = make(map[string]*Peer)
	}
	if _, ok := m.Peers[ip][id]; !ok {
		m.Peers[ip][id] = new(Peer)
	}
	return m.Peers[ip][id]
}

func (db *Dashboard) collectPeerData() {
	defer db.wg.Done()

	var err error
	db.geodb, err = OpenGeoDB()
	if err != nil {
		log.Warn("Failed to open geodb", "err", err)
		return
	}
	defer db.geodb.Close()

	var (
		quit         = make(chan struct{})
		connectCh    = make(chan *p2p.PeerConnectEvent, eventBufferLimit)
		handshakeCh  = make(chan *p2p.PeerHandshakeEvent, eventBufferLimit)
		disconnectCh = make(chan *p2p.PeerDisconnectEvent, eventBufferLimit)
		readCh       = make(chan *p2p.PeerReadEvent, eventBufferLimit)
		writeCh      = make(chan *p2p.PeerWriteEvent, eventBufferLimit)
	)
	go func() {
		var (
			peerConnectEventCh    = make(chan p2p.PeerConnectEvent, eventBufferLimit)
			peerHandshakeEventCh  = make(chan p2p.PeerHandshakeEvent, eventBufferLimit)
			peerDisconnectEventCh = make(chan p2p.PeerDisconnectEvent, eventBufferLimit)
			peerReadEventCh       = make(chan p2p.PeerReadEvent, eventBufferLimit)
			peerWriteEventCh      = make(chan p2p.PeerWriteEvent, eventBufferLimit)

			subConnect    = p2p.SubscribePeerConnectEvent(peerConnectEventCh)
			subHandshake  = p2p.SubscribePeerHandshakeEvent(peerHandshakeEventCh)
			subDisconnect = p2p.SubscribePeerDisconnectEvent(peerDisconnectEventCh)
			subRead       = p2p.SubscribePeerReadEvent(peerReadEventCh)
			subWrite      = p2p.SubscribePeerWriteEvent(peerWriteEventCh)
		)
		defer func() {
			subConnect.Unsubscribe()
			subHandshake.Unsubscribe()
			subDisconnect.Unsubscribe()
			subRead.Unsubscribe()
			subWrite.Unsubscribe()
		}()
		for {
			select {
			case event := <-peerConnectEventCh:
				select {
				case connectCh <- &event:
				default:
					log.Warn("Failed to handle connect event", "event", event)
				}
			case event := <-peerHandshakeEventCh:
				select {
				case handshakeCh <- &event:
				default:
					log.Warn("Failed to handle handshake event", "event", event)
				}
			case event := <-peerDisconnectEventCh:
				select {
				case disconnectCh <- &event:
				default:
					log.Warn("Failed to handle disconnect event", "event", event)
				}
			case event := <-peerReadEventCh:
				select {
				case readCh <- &event:
				default:
					log.Warn("Failed to handle read event", "event", event)
				}
			case event := <-peerWriteEventCh:
				select {
				case writeCh <- &event:
				default:
					log.Warn("Failed to handle write event", "event", event)
				}
			case <-quit:
				return
			}
		}
	}()
	go db.cleanPeerHistory(quit)

	ticker := time.NewTicker(db.config.Refresh)
	defer ticker.Stop()

	network := &NetworkMessage{
		Peers: make(map[string]map[string]*Peer),
	}
	for {
		select {
		case event := <-connectCh:
			ip := event.IP.String()
			p := getOrInitPeer(network, ip, event.ID)
			if p.Location == nil {
				db.peerLock.RLock()
				peers := db.peerHistory.Peers
				lookup := peers[ip] == nil || peers[ip][event.ID] == nil || peers[ip][event.ID].Location == nil
				db.peerLock.RUnlock()
				if lookup {
					location := db.geodb.Lookup(event.IP)
					p.Location = &PeerLocation{
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
			p := getOrInitPeer(network, ip, event.DefaultID)
			if p.Handshake == nil {
				p.Handshake = []time.Time{event.Handshake}
			} else {
				p.Handshake = append(p.Handshake, event.Handshake)
			}
			delete(network.Peers[ip], event.DefaultID)
			getOrInitPeer(network, ip, event.ID)
			network.Peers[ip][event.ID] = p // interleave instead
			// Remove the peer from history in case the metering was before the handshake.
			db.peerLock.RLock()
			stored := db.peerHistory.Peers[ip] != nil && db.peerHistory.Peers[ip][event.DefaultID] != nil
			db.peerLock.RUnlock()
			if stored {
				db.peerLock.Lock()
				hp := getOrInitPeer(db.peerHistory, ip, event.DefaultID)
				delete(db.peerHistory.Peers[ip], event.DefaultID)
				getOrInitPeer(db.peerHistory, ip, event.ID)
				db.peerHistory.Peers[ip][event.ID] = hp // interleave instead
				db.peerLock.Unlock()
			}
		case event := <-disconnectCh:
			p := getOrInitPeer(network, event.IP.String(), event.ID)
			if p.Disconnected == nil {
				p.Disconnected = []time.Time{event.Disconnected}
			} else {
				p.Disconnected = append(p.Disconnected, event.Disconnected)
			}
		case event := <-readCh:
			// Sum up the ingress between two updates.
			p := getOrInitPeer(network, event.IP.String(), event.ID)
			if len(p.Ingress) <= 0 {
				p.Ingress = ChartEntries{&ChartEntry{Value: float64(event.Ingress)}}
			} else {
				p.Ingress[0].Value += float64(event.Ingress)
			}
		case event := <-writeCh:
			// Sum up the egress between two updates.
			p := getOrInitPeer(network, event.IP.String(), event.ID)
			if len(p.Egress) <= 0 {
				p.Egress = ChartEntries{&ChartEntry{Value: float64(event.Egress)}}
			} else {
				p.Egress[0].Value += float64(event.Egress)
			}
		case <-ticker.C:
			now := time.Now()
			db.peerLock.Lock()
			for ip, peers := range network.Peers {
				for id, peer := range peers {
					peerHistory := getOrInitPeer(db.peerHistory, ip, id)
					if peer.Location != nil {
						peerHistory.Location = peer.Location
					}
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
						//peerHistory.Ingress = ChartEntries{ingress}
					} else {
						peer.Ingress = ChartEntries{ingress}
						peerHistory.Ingress = append(peerHistory.Ingress[1:], ingress)
						//peerHistory.Ingress = append(peerHistory.Ingress, ingress)
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
						//peerHistory.Egress = ChartEntries{egress}
					} else {
						peer.Egress = ChartEntries{egress}
						peerHistory.Egress = append(peerHistory.Egress[1:], egress)
						//peerHistory.Egress = append(peerHistory.Egress, egress)
					}
				}
			}
			db.peerLock.Unlock()
			db.sendToAll(&Message{Network: deepcopy.Copy(network).(*NetworkMessage)})

			//fmt.Println()
			//s, _ := json.MarshalIndent(network, "", "    ")
			//fmt.Println(string(s))

			for ip, peers := range network.Peers {
				for id := range peers {
					peers[id] = nil
					delete(peers, id)
				}
				delete(network.Peers, ip)
			}
		case errc := <-db.quit:
			close(quit)
			errc <- nil
			return
		}
	}
}

func (db *Dashboard) cleanPeerHistory(quit chan struct{}) {
	cleanRate := db.config.Refresh * peerTrafficSampleLimit
	for {
		select {
		case <-time.After(cleanRate):
			// clear disconnected
			validAfter := time.Now().Add(-cleanRate)
			db.peerLock.Lock()
			for ip, peers := range db.peerHistory.Peers {
				for id, peer := range peers {
					if len(peer.Disconnected) > 0 && peer.Disconnected[len(peer.Disconnected)-1].Before(validAfter) {
						db.peerHistory.Peers[ip][id].Location = nil
						db.peerHistory.Peers[ip][id] = nil
						delete(db.peerHistory.Peers[ip], id)
					}
				}
				if len(peers) <= 0 {
					delete(db.peerHistory.Peers, ip)
				}
			}
			var lenCount int
			for _, peers := range db.peerHistory.Peers {
				lenCount += len(peers)
			}
			if lenCount > p2p.MeteredPeerLimit {
			outerLoop:
				for ip, peers := range db.peerHistory.Peers {
					for id, peer := range peers {
						if peer.Disconnected != nil {
							db.peerHistory.Peers[ip][id].Location = nil
							db.peerHistory.Peers[ip][id] = nil
							delete(db.peerHistory.Peers[ip], id)
							lenCount--
							if lenCount <= p2p.MeteredPeerLimit {
								if len(peers) <= 0 {
									delete(db.peerHistory.Peers, ip)
								}
								break outerLoop
							}
						}
					}
					if len(peers) <= 0 {
						delete(db.peerHistory.Peers, ip)
					}
				}
			}
			db.peerLock.Unlock()
		case <-quit:
			return
		}
	}
}
