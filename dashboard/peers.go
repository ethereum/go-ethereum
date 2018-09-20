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
	"time"

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
)

const (
	eventBufferLimit = 128 // Maximum number of buffered peer events
	connectionLimit  = 16
)

var autoID int64

// maintainedPeer is the element of the peer maintainer's linked list.
// Similarly to an ordinary linked list element it knows the previous
// and the next elements, and also has a pointer to its parent map in
// order to remove itself from there when it is removed from the list.
type maintainedPeer struct {
	// To simplify the implementation, the list is implemented as a ring,
	// such that 'root' is both the next element of the last list element
	// and the previous element of the first list element.
	prev   *maintainedPeer // Pointer to the previous element
	next   *maintainedPeer // Pointer to the next element
	parent *idContainer    // Pointer to the parent
	id     string          // NodeID of the peer
}

// idContainer contains the NodeIDs belonging to an IP address.
// The pointer to the parent makes it possible for the container
// to remove itself from the parent's map when becomes empty.
type idContainer struct {
	parent *PeerMaintainer            // Pointer to the parent of the container
	ip     string                     // IP of the peer
	peers  map[string]*maintainedPeer // NodeIDs belonging to the given IP
}

// update moves the list element belonging to the given ID to the end,
// or inserts a new element to the end of the list if the given ID didn't
// appear yet in the container. Returns the IP and the ID of the removed
// peer if an element removal happened, or nil otherwise.
func (idc *idContainer) update(id string) *removedPeer {
	maintainer := idc.parent
	if _, ok := idc.peers[id]; !ok {
		e := &maintainedPeer{
			parent: idc,
			id:     id,
		}
		maintainer.insert(e, maintainer.root.prev)
		idc.peers[id] = e
	} else {
		maintainer.insert(maintainer.remove(idc.peers[id]), maintainer.root.prev)
	}
	if maintainer.len > maintainer.limit {
		first := maintainer.remove(maintainer.root.next)
		return &removedPeer{
			ip: first.parent.ip,
			id: first.id,
		}
	}
	return nil
}

// remove removes the peer belonging to the given ID and removes itself too
// from the parent container if becomes empty.
func (idc *idContainer) remove(id string) {
	delete(idc.peers, id)
	if len(idc.peers) < 1 {
		idc.parent.removeIP(idc.ip)
		idc.parent = nil
	}
}

// PeerMaintainer is an abstract layer above the metered peer container,
// maintaining the peers. i.e. sorting them based on their activity and
// removing the oldest inactive ones when their count reaches the limit.
//
// Consists of a map of maps which represent the peers grouped by the IP
// address then by the NodeID. The elements on the bottom of the tree are
// doubly linked and sorted by their activity. i.e. the first element is
// the peer that is inactive for the longest time. This makes it possible
// to count the peers and remove the oldest one effectively.
//
// When a peer event appears, the active peer goes to the end of the list
// and in case of removal the IP and the ID of the removed peer is returned.
type PeerMaintainer struct {
	ids   map[string]*idContainer // NodeIDs grouped by IP
	root  maintainedPeer          // Sentinel list element
	len   int                     // Current list length excluding the sentinel element
	limit int                     // Maximum number of maintained peers
}

// init initializes the peer maintainer.
func (pm *PeerMaintainer) init(limit int) *PeerMaintainer {
	if limit < 0 {
		limit = 0
	}
	pm.ids = make(map[string]*idContainer)
	pm.root.prev = &pm.root
	pm.root.next = &pm.root
	pm.len = 0
	pm.limit = limit
	return pm
}

// NewPeerMaintainer returns an initialized peer maintainer.
func NewPeerMaintainer(limit int) *PeerMaintainer {
	return new(PeerMaintainer).init(limit)
}

// Update updates the peer element belonging to the given IP and ID.
func (pm *PeerMaintainer) Update(ip, id string) *removedPeer {
	return pm.getOrInitIDs(ip).update(id)
}

// getOrInitIDs returns the ID map. Initializes it if it doesn't exist.
func (pm *PeerMaintainer) getOrInitIDs(ip string) *idContainer {
	if _, ok := pm.ids[ip]; !ok {
		pm.ids[ip] = &idContainer{
			parent: pm,
			ip:     ip,
			peers:  make(map[string]*maintainedPeer),
		}
	}
	return pm.ids[ip]
}

// insert inserts 'e' after 'at' and increments the current list length.
func (pm *PeerMaintainer) insert(e, at *maintainedPeer) {
	n := at.next
	at.next = e
	e.prev = at
	e.next = n
	n.prev = e
	pm.len++
}

// remove removes e from the list and from the maintainer tree.
func (pm *PeerMaintainer) remove(e *maintainedPeer) *maintainedPeer {
	e.next.prev = e.prev
	e.prev.next = e.next
	e.prev = nil
	e.next = nil
	e.parent.remove(e.id)
	e.parent = nil
	pm.len--
	return e
}

// clear cleans up the peer maintainer.
func (pm *PeerMaintainer) clear() {
	for pm.root.next != &pm.root {
		next := pm.root.next
		pm.root.next = next.next
		next.prev = nil
		next.next = nil
		next.parent.remove(next.id)
		next.parent = nil
	}
}

// removeIP removes the peers belonging to the given IP. It is supposed that
// 'ids' is empty, because the clearing direction is from bottom to top.
func (pm *PeerMaintainer) removeIP(ip string) {
	delete(pm.ids, ip)
}

// removedPeer contains the IP and the ID of the peer that was removed from
// the peer maintainer.
type removedPeer struct {
	ip, id string
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

	db.peerLock.RLock()
	//historyMaintainer := NewPeerMaintainer(p2p.MeteredPeerLimit)
	//historyHandshakeFailedMaintainer := NewPeerMaintainer(p2p.MeteredPeerLimit)
	//diffMaintainer := NewPeerMaintainer(p2p.MeteredPeerLimit)
	//diffHandshakeFailedMaintainer := NewPeerMaintainer(p2p.MeteredPeerLimit)
	for {
		select {
		case event := <-peerCh:
			fmt.Println(event)
			//diffMaintainer.Update(event.IP.String(), event.ID)
		case err := <-subPeer.Err():
			log.Warn("Peer subscription error", "err", err)
			return
		case errc := <-db.quit:
			errc <- nil
			return
		}
	}
}
