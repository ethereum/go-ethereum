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

	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

/*
 Retrieve Request and store Request handling
*/

// Handler for storage/retrieval related protocol requests
type RequestHandler struct {
	netStore *storage.NetStore
}

// NewEwquestHandler creates a new RequestHandler
// netStore to
func NewRequestHandler(netStore *storage.NetStore) *RequestHandler {
	return &RequestHandler{
		netStore: netStore, // entrypoint internal
	}
}

/*
Retrieve request

MaxSize specifies the maximum size that the peer will accept. This is useful in
particular if we allow storage and delivery of multichunk payload representing
the entire or partial subtree unfolding from the requested root key.
So when only interested in limited part of a stream (infinite trees) or only
testing chunk availability etc etc, we can indicate it by limiting the size here.

Request ID can be newly generated or kept from the request originator.

*/
type retrieveRequestMsg struct {
	Key     storage.Key   // target Key address of chunk to be retrieved
	Id      uint64        // request id, request is a lookup if missing or zero
	MaxSize uint64        // maximum size of delivery accepted
	from    *StreamerPeer //
}

func (self retrieveRequestMsg) String() string {
	var from string
	if self.from == nil {
		from = "ourselves"
	} else {
		from = fmt.Sprintf("%x", self.from.Over())
	}
	var target []byte
	if len(self.Key) > 3 {
		target = self.Key[:4]
	}
	return fmt.Sprintf("Requester: %v, Key: %x; ID: %v, MaxSize: %v", from, target, self.Id, self.MaxSize)
}

// entrypoint for retrieve requests coming from the bzz wire protocol
// checks swap balance - return if peer has no credit
func (self *StreamerPeer) handleRetrieveRequestMsg(msg interface{}) error {
	req := msg.(*retrieveRequestMsg)
	req.from = self
	// TODO:
	// swap - record credit for 1 request
	// note that only charge actual reqsearches

	// call storage.NetStore#Get which
	// blocks until local retrieval finished
	// launches cloud retrieval
	chunk, _ := self.netStore.Get(req.Key)
	rs := chunk.Req
	if rs != nil {
		rs = storage.NewRequestStatus(req.Key)
		addRequester(rs, req)
		chunk.Req = rs
	}

	// check if we can immediately deliver
	if chunk.SData != nil {
		if req.MaxSize == 0 || int64(req.MaxSize) >= chunk.Size {
			err := self.Deliver(chunk, Top)
			if err != nil {
				log.Trace(fmt.Sprintf("%v - content found, delivery error: %v", req.Key.Log(), err))
				return nil
			}
			log.Trace(fmt.Sprintf("%v - content found, delivering...", req.Key.Log()))
		} else {
			log.Trace(fmt.Sprintf("%v - content found, not wanted", req.Key.Log()))
		}
	} else {
		log.Trace(fmt.Sprintf("content not found locally, retrieve via bzz", req.Key.Log()))
	}
	return nil
}

/*
adds a new peer to an existing open request
only add if less than requesterCount peers forwarded the same request id so far
note this is done irrespective of status (searching or found)
*/
func addRequester(rs *storage.RequestStatus, req *retrieveRequestMsg) {
	log.Trace(fmt.Sprintf("Depo.addRequester: key %v - add peer to req.Id %v", req.Key.Log(), req.Id))
	list := rs.Requesters[req.Id]
	rs.Requesters[req.Id] = append(list, req)
}

/*
 store requests are put in netstore so they are stored and then
 forwarded to the peers in their kademlia proximity bin by the syncer
*/
type storeRequestMsg struct {
	Key   storage.Key
	SData []byte // the stored chunk Data (incl size)
	// optional
	Id   uint64 // request ID. if delivery, the ID is retrieve request ID
	from Peer   // [not serialised] protocol registers the requester
}

func (self storeRequestMsg) String() string {
	var from string
	if self.from == nil {
		from = "self"
	} else {
		from = fmt.Sprintf("%x", self.from.Over())
	}
	end := len(self.SData)
	if len(self.SData) > 10 {
		end = 10
	}
	return fmt.Sprintf("from: %v, ID: %v, SData %x", from, self.Id, self.SData[:end])
}

// the entrypoint for store requests coming from the bzz wire protocol
// if key found locally, return. otherwise
// remote is untrusted, so hash is verified and chunk passed on to NetStore
func (self *RequestHandler) handleStoreRequestMsg(msg interface{}, p Peer) error {
	req := msg.(*storeRequestMsg)
	req.from = p
	// TODO: chunk validation
	chunk := storage.NewChunk(req.Key, nil)
	chunk.SData = req.SData
	chunk.Source = p
	self.netStore.Put(chunk)
	log.Trace(fmt.Sprintf("delivery of %v from %v", chunk, p))
	return nil
}
