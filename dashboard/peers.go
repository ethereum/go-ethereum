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
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

const eventBufferLimit = 128 // Maximum number of buffered peer events
const trafficEventBufferLimit = p2p.MeteredPeerLimit
const connectionLimit = 100

var autoID int64

type peerLimiter struct {
	underlying *NetworkMessage
	failed bool
	l *list.List
}

func NewPeerLimiter(underlying *NetworkMessage, failed bool) *peerLimiter {
	return &peerLimiter{l: list.New(), underlying: underlying, failed: failed}
}

func (pl *peerLimiter) update(peer *Peer) {
	return
	if peer.element == nil {
		peer.element = pl.l.PushBack(peer)
	} else {
		pl.l.MoveToBack(peer.element)
	}
	for pl.l.Len() > 2 {//p2p.MeteredPeerLimit {
		pl.remove(pl.l.Front())
	}
}

func (pl *peerLimiter) remove(e *list.Element) {
	return
	elem := pl.l.Remove(e)
	if peer, ok := elem.(*Peer); ok {
		if pl.failed {
			fmt.Println(peer.ip, peer.id)
			pl.underlying.PeerBundles[peer.ip].FailedPeers.remove(peer.id)
		} else {
			fmt.Println(peer.ip, peer.id[:10])
			pl.underlying.PeerBundles[peer.ip].Peers.remove(peer.id)
		}
	}
}

func (pl *peerLimiter) clear() {
	for pl.l.Front() != nil {
		pl.remove(pl.l.Front())
	}
}
//
//func tail(arr []time.Time) []time.Time {
//	if first := len(arr)-connectionLimit; first > 0 {
//		return arr[first:]
//	}
//	return arr
//}

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
		peerCh    = make(chan p2p.MeteredPeerEvent, eventBufferLimit)
		// Subscribe to peer events.
		subPeer    = p2p.SubscribePeerEvent(peerCh)
	)
	defer func() {
		// Unsubscribe at the end.
		subPeer.Unsubscribe()
	}()

	ticker := time.NewTicker(db.config.Refresh)
	defer ticker.Stop()

	db.peerLock.RLock()
	//historyPeerLimiter := NewPeerLimiter(db.history.Network, false)
	//historyFailedPeerLimiter := NewPeerLimiter(db.history.Network, true)
	//db.peerLock.RUnlock()
	//// Listen for events, and prepare the difference between two metering.
	//diff := &NetworkMessage{
	//	PeerBundles: make(map[string]*PeerBundle),
	//}
	//// Needed in order to keep the limit in the diff
	//diffPeerLimiter := NewPeerLimiter(diff, false)
	//diffFailedPeerLimiter := NewPeerLimiter(diff, true)
	for {
		select {
		case event := <-peerCh:
			fmt.Println(event)
		//case event := <-connectCh:
		//	diffBundle := diff.getOrInitBundle(event.IP)
		//	diffBundle.Location = db.geodb.Location(event.IP)
		//	diffPeer := diffBundle.getOrInitPeer(event.ID)
		//	diffPeer.Connected = append(diffPeer.Connected, event.Connected)
		//	if first := len(diffPeer.Connected)-connectionLimit; first > 0 {
		//		diffPeer.Connected = diffPeer.Connected[first:]
		//	}
		//	diffPeer.ip = event.IP
		//	diffPeer.id = event.ID
		//	diffPeerLimiter.update(diffPeer)
		//case event := <-disconnectCh:
		//	diffPeer := diff.getOrInitPeer(event.IP, event.ID)
		//	diffPeer.Disconnected = append(diffPeer.Disconnected, event.Time)
		//	if first := len(diffPeer.Connected)-connectionLimit; first > 0 {
		//		diffPeer.Connected = diffPeer.Connected[first:]
		//	}
		//	diffPeer.ip = event.IP
		//	diffPeer.id = event.ID
		//	diffPeerLimiter.update(diffPeer)
		//case event := <-ingressCh:
		//	diffPeer := diff.getOrInitPeer(event.IP, event.ID)
		//	if len(diffPeer.Ingress) != 1 {
		//		diffPeer.Ingress = ChartEntries{&ChartEntry{Value: float64(event.Amount)}}
		//	} else {
		//		diffPeer.Ingress[0].Value = float64(event.Amount)
		//	}
		//	diffPeer.ip = event.IP
		//	diffPeer.id = event.ID
		//	diffPeerLimiter.update(diffPeer)
		//case event := <-egressCh:
		//	diffPeer := diff.getOrInitPeer(event.IP, event.ID)
		//	if len(diffPeer.Egress) != 1 {
		//		diffPeer.Egress = ChartEntries{&ChartEntry{Value: float64(event.Amount)}}
		//	} else {
		//		diffPeer.Egress[0].Value = float64(event.Amount)
		//	}
		//	diffPeer.ip = event.IP
		//	diffPeer.id = event.ID
		//	diffPeerLimiter.update(diffPeer)
		//case event := <-failedCh:
		//	diffBundle := diff.getOrInitBundle(event.IP)
		//	diffBundle.Location = db.geodb.Location(event.IP)
		//	id := fmt.Sprintf("peer_%d", atomic.AddInt64(&autoID, 1))
		//	failedPeer := diffBundle.FailedPeers.getOrInit(id)
		//	failedPeer.Connected = []time.Time{event.Connected}
		//	failedPeer.Disconnected = []time.Time{event.Disconnected}
		//	failedPeer.ip = event.IP
		//	failedPeer.id = id
		//	diffFailedPeerLimiter.update(failedPeer)
		//case <-ticker.C:
		//	now := time.Now()
		//	// Merge the diff with the history.
		//	db.peerLock.Lock()
		//	for ip, diffBundle := range diff.PeerBundles {
		//		historyBundle := db.history.Network.getOrInitBundle(ip)
		//		historyBundle.Location = diffBundle.Location
		//		for id, diffPeer := range diffBundle.Peers {
		//			historyPeer := historyBundle.getOrInitPeer(id)
		//			historyPeer.Connected = append(historyPeer.Connected, diffPeer.Connected...)
		//			if first := len(historyPeer.Connected)-connectionLimit; first > 0 {
		//				historyPeer.Connected = historyPeer.Connected[first:]
		//			}
		//			historyPeer.Disconnected = append(historyPeer.Disconnected, diffPeer.Disconnected...)
		//			if first := len(historyPeer.Disconnected)-connectionLimit; first > 0 {
		//				historyPeer.Disconnected = historyPeer.Disconnected[first:]
		//			}
		//			if len(diffPeer.Ingress) == 1 {
		//				diffPeer.Ingress[0].Time = now
		//				if historyPeer.Ingress == nil {
		//					historyPeer.Ingress = append(emptyChartEntries(now.Add(-db.config.Refresh), 3/*sampleLimit-1*/, db.config.Refresh), diffPeer.Ingress[0])
		//					// The first message about a diffPeer should contain the whole list
		//					diffPeer.Ingress = historyPeer.Ingress
		//				} else {
		//					historyPeer.Ingress = append(historyPeer.Ingress, diffPeer.Ingress[0])[1:]
		//				}
		//			}
		//			if len(diffPeer.Egress) == 1 {
		//				diffPeer.Egress[0].Time = now
		//				if historyPeer.Egress == nil {
		//					historyPeer.Egress = append(emptyChartEntries(now.Add(-db.config.Refresh), 3/*sampleLimit-1*/, db.config.Refresh), diffPeer.Egress[0])
		//					// The first message about a diffPeer should contain the whole list
		//					diffPeer.Egress = historyPeer.Egress
		//				} else {
		//					historyPeer.Egress = append(historyPeer.Egress, diffPeer.Egress[0])[1:]
		//				}
		//			}
		//			historyPeer.ip = diffPeer.ip
		//			historyPeer.id = diffPeer.id
		//			historyPeerLimiter.update(historyPeer)
		//		}
		//		for id, diffFailedPeer := range diffBundle.FailedPeers {
		//			historyFailedPeer := historyBundle.getOrInitFailedPeer(id)
		//			historyFailedPeer.Connected = diffFailedPeer.Connected
		//			historyFailedPeer.Disconnected = diffFailedPeer.Disconnected
		//			historyFailedPeer.ip = diffFailedPeer.ip
		//			historyFailedPeer.id = diffFailedPeer.id
		//			historyFailedPeerLimiter.update(historyFailedPeer)
		//		}
		//	}
		//	//for elem := historyPeerLimiter.l.Front(); elem != nil; elem = elem.Next() {
		//	//	s, _ := json.MarshalIndent(elem.Value, "", "  ")
		//	//	fmt.Println(string(s))
		//	//}
		//	//fmt.Println()
		//	db.peerLock.Unlock()
		//
		//	//s, _ := json.MarshalIndent(deepcopy.Copy(diff), "", "  ")
		//	//fmt.Println(string(s))
		//	//fmt.Println()
		//	// Send the diff to the clients.
		//	db.sendToAll(&Message{Network: deepcopy.Copy(diff).(*NetworkMessage)})
		//
		//	// Prepare for the next metering, clear the diff variable.
		//	diffPeerLimiter.clear()
		//	diffFailedPeerLimiter.clear()
		//	diff = &NetworkMessage{
		//		PeerBundles: make(map[string]*PeerBundle),
		//	}
		case err := <-subPeer.Err():
			log.Warn("Peer subscription error", "err", err)
			return
		case errc := <-db.quit:
			errc <- nil
			return
		}
	}
}

