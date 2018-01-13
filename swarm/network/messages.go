// Copyright 2016 The go-ethereum Authors
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

package network

import (
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/contracts/chequebook"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/swarm/network/kademlia"
	"github.com/ethereum/go-ethereum/swarm/services/swap"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

/*
BZZ protocol Message Types and Message Data Types
*/

// bzz protocol message codes
const (
	statusMsg          = iota // 0x01
	storeRequestMsg           // 0x02
	retrieveRequestMsg        // 0x03
	peersMsg                  // 0x04
	syncRequestMsg            // 0x05
	deliveryRequestMsg        // 0x06
	unsyncedKeysMsg           // 0x07
	paymentMsg                // 0x08
)

/*
 Handshake

* Version: 8 byte integer version of the protocol
* ID: arbitrary byte sequence client identifier human readable
* Addr: the address advertised by the node, format similar to DEVp2p wire protocol
* Swap: info for the swarm accounting protocol
* NetworkID: 8 byte integer network identifier
* Caps: swarm-specific capabilities, format identical to devp2p
* SyncState: syncronisation state (db iterator key and address space etc) persisted about the peer

*/
type statusMsgData struct {
	Version   uint64
	ID        string
	Addr      *peerAddr
	Swap      *swap.SwapProfile
	NetworkId uint64
}

func (self *statusMsgData) String() string {
	return fmt.Sprintf("Status: Version: %v, ID: %v, Addr: %v, Swap: %v, NetworkId: %v", self.Version, self.ID, self.Addr, self.Swap, self.NetworkId)
}

/*
 store requests are forwarded to the peers in their kademlia proximity bin
 if they are distant
 if they are within our storage radius or have any incentive to store it
 then attach your nodeID to the metadata
 if the storage request is sufficiently close (within our proxLimit, i. e., the
 last row of the routing table)
*/
type storeRequestMsgData struct {
	Key   storage.Key // hash of datasize | data
	SData []byte      // the actual chunk Data
	// optional
	Id             uint64     // request ID. if delivery, the ID is retrieve request ID
	requestTimeout *time.Time // expiry for forwarding - [not serialised][not currently used]
	storageTimeout *time.Time // expiry of content - [not serialised][not currently used]
	from           *peer      // [not serialised] protocol registers the requester
}

func (self storeRequestMsgData) String() string {
	var from string
	if self.from == nil {
		from = "self"
	} else {
		from = self.from.Addr().String()
	}
	end := len(self.SData)
	if len(self.SData) > 10 {
		end = 10
	}
	return fmt.Sprintf("from: %v, Key: %v; ID: %v, requestTimeout: %v, storageTimeout: %v, SData %x", from, self.Key, self.Id, self.requestTimeout, self.storageTimeout, self.SData[:end])
}

/*
Retrieve request

Timeout in milliseconds. Note that zero timeout retrieval requests do not request forwarding, but prompt for a peers message response. therefore they serve also
as messages to retrieve peers.

MaxSize specifies the maximum size that the peer will accept. This is useful in
particular if we allow storage and delivery of multichunk payload representing
the entire or partial subtree unfolding from the requested root key.
So when only interested in limited part of a stream (infinite trees) or only
testing chunk availability etc etc, we can indicate it by limiting the size here.

Request ID can be newly generated or kept from the request originator.
If request ID Is missing or zero, the request is handled as a lookup only
prompting a peers response but not launching a search. Lookup requests are meant
to be used to bootstrap kademlia tables.

In the special case that the key is the zero value as well, the remote peer's
address is assumed (the message is to be handled as a self lookup request).
The response is a PeersMsg with the peers in the kademlia proximity bin
corresponding to the address.
*/

type retrieveRequestMsgData struct {
	Key      storage.Key // target Key address of chunk to be retrieved
	Id       uint64      // request id, request is a lookup if missing or zero
	MaxSize  uint64      // maximum size of delivery accepted
	MaxPeers uint64      // maximum number of peers returned
	Timeout  uint64      // the longest time we are expecting a response
	timeout  *time.Time  // [not serialied]
	from     *peer       //
}

func (self *retrieveRequestMsgData) String() string {
	var from string
	if self.from == nil {
		from = "ourselves"
	} else {
		from = self.from.Addr().String()
	}
	var target []byte
	if len(self.Key) > 3 {
		target = self.Key[:4]
	}
	return fmt.Sprintf("from: %v, Key: %x; ID: %v, MaxSize: %v, MaxPeers: %d", from, target, self.Id, self.MaxSize, self.MaxPeers)
}

// lookups are encoded by missing request ID
func (self *retrieveRequestMsgData) isLookup() bool {
	return self.Id == 0
}

// sets timeout fields
func (self *retrieveRequestMsgData) setTimeout(t *time.Time) {
	self.timeout = t
	if t != nil {
		self.Timeout = uint64(t.UnixNano())
	} else {
		self.Timeout = 0
	}
}

func (self *retrieveRequestMsgData) getTimeout() (t *time.Time) {
	if self.Timeout > 0 && self.timeout == nil {
		timeout := time.Unix(int64(self.Timeout), 0)
		t = &timeout
		self.timeout = t
	}
	return
}

// peerAddr is sent in StatusMsg as part of the handshake
type peerAddr struct {
	IP   net.IP
	Port uint16
	ID   []byte // the 64 byte NodeID (ECDSA Public Key)
	Addr kademlia.Address
}

// peerAddr pretty prints as enode
func (self *peerAddr) String() string {
	var nodeid discover.NodeID
	copy(nodeid[:], self.ID)
	return discover.NewNode(nodeid, self.IP, 0, self.Port).String()
}

/*
peers Msg is one response to retrieval; it is always encouraged after a retrieval
request to respond with a list of peers in the same kademlia proximity bin.
The encoding of a peer is identical to that in the devp2p base protocol peers
messages: [IP, Port, NodeID]
note that a node's DPA address is not the NodeID but the hash of the NodeID.

Timeout serves to indicate whether the responder is forwarding the query within
the timeout or not.

NodeID serves as the owner of payment contracts and signer of proofs of transfer.

The Key is the target (if response to a retrieval request) or missing (zero value)
peers address (hash of NodeID) if retrieval request was a self lookup.

Peers message is requested by retrieval requests with a missing or zero value request ID
*/
type peersMsgData struct {
	Peers   []*peerAddr //
	Timeout uint64      //
	timeout *time.Time  // indicate whether responder is expected to deliver content
	Key     storage.Key // present if a response to a retrieval request
	Id      uint64      // present if a response to a retrieval request
	from    *peer
}

// peers msg pretty printer
func (self *peersMsgData) String() string {
	var from string
	if self.from == nil {
		from = "ourselves"
	} else {
		from = self.from.Addr().String()
	}
	var target []byte
	if len(self.Key) > 3 {
		target = self.Key[:4]
	}
	return fmt.Sprintf("from: %v, Key: %x; ID: %v, Peers: %v", from, target, self.Id, self.Peers)
}

func (self *peersMsgData) setTimeout(t *time.Time) {
	self.timeout = t
	if t != nil {
		self.Timeout = uint64(t.UnixNano())
	} else {
		self.Timeout = 0
	}
}

/*
syncRequest

is sent after the handshake to initiate syncing
the syncState of the remote node is persisted in kaddb and set on the
peer/protocol instance when the node is registered by hive as online{
*/

type syncRequestMsgData struct {
	SyncState *syncState `rlp:"nil"`
}

func (self *syncRequestMsgData) String() string {
	return fmt.Sprintf("%v", self.SyncState)
}

/*
deliveryRequest

is sent once a batch of sync keys is filtered. The ones not found are
sent as a list of syncReuest (hash, priority) in the Deliver field.
When the source receives the sync request it continues to iterate
and fetch at most N items as yet unsynced.
At the same time responds with deliveries of the items.
*/
type deliveryRequestMsgData struct {
	Deliver []*syncRequest
}

func (self *deliveryRequestMsgData) String() string {
	return fmt.Sprintf("sync request for new chunks\ndelivery request for %v chunks", len(self.Deliver))
}

/*
unsyncedKeys

is sent first after the handshake if SyncState iterator brings up hundreds, thousands?
and subsequently sent as a response to deliveryRequestMsgData.

Syncing is the iterative process of exchanging unsyncedKeys and deliveryRequestMsgs
both ways.

State contains the sync state sent by the source. When the source receives the
sync state it continues to iterate and fetch at most N items as yet unsynced.
At the same time responds with deliveries of the items.
*/
type unsyncedKeysMsgData struct {
	Unsynced []*syncRequest
	State    *syncState
}

func (self *unsyncedKeysMsgData) String() string {
	return fmt.Sprintf("sync: keys of %d new chunks (state %v) => synced: %v", len(self.Unsynced), self.State, self.State.Synced)
}

/*
payment

is sent when the swap balance is tilted in favour of the remote peer
and in absolute units exceeds the PayAt parameter in the remote peer's profile
*/

type paymentMsgData struct {
	Units   uint               // units actually paid for (checked against amount by swap)
	Promise *chequebook.Cheque // payment with cheque
}

func (self *paymentMsgData) String() string {
	return fmt.Sprintf("payment for %d units: %v", self.Units, self.Promise)
}
