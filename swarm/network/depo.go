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
	"bytes"
	"encoding/binary"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

// Handler for storage/retrieval related protocol requests
// implements the StorageHandler interface used by the bzz protocol
type Depo struct {
	hashfunc   storage.Hasher
	localStore storage.ChunkStore
	netStore   storage.ChunkStore
}

func NewDepo(hash storage.Hasher, localStore, remoteStore storage.ChunkStore) *Depo {
	return &Depo{
		hashfunc:   hash,
		localStore: localStore,
		netStore:   remoteStore, // entrypoint internal
	}
}

// Handles UnsyncedKeysMsg after msg decoding - unsynced hashes upto sync state
// * the remote sync state is just stored and handled in protocol
// * filters through the new syncRequests and send the ones missing
// * back immediately as a deliveryRequest message
// * empty message just pings back for more (is this needed?)
// * strict signed sync states may be needed.
func (self *Depo) HandleUnsyncedKeysMsg(req *unsyncedKeysMsgData, p *peer) error {
	unsynced := req.Unsynced
	var missing []*syncRequest
	var chunk *storage.Chunk
	var err error
	for _, req := range unsynced {
		// skip keys that are found,
		chunk, err = self.localStore.Get(storage.Key(req.Key[:]))
		if err != nil || chunk.SData == nil {
			missing = append(missing, req)
		}
	}
	glog.V(logger.Debug).Infof("Depo.HandleUnsyncedKeysMsg: received %v unsynced keys: %v missing. new state: %v", len(unsynced), len(missing), req.State)
	glog.V(logger.Detail).Infof("Depo.HandleUnsyncedKeysMsg: received %v", unsynced)
	// send delivery request with missing keys
	err = p.deliveryRequest(missing)
	if err != nil {
		return err
	}
	// set peers state to persist
	p.syncState = req.State
	return nil
}

// Handles deliveryRequestMsg
// * serves actual chunks asked by the remote peer
// by pushing to the delivery queue (sync db) of the correct priority
// (remote peer is free to reprioritize)
// * the message implies remote peer wants more, so trigger for
// * new outgoing unsynced keys message is fired
func (self *Depo) HandleDeliveryRequestMsg(req *deliveryRequestMsgData, p *peer) error {
	deliver := req.Deliver
	// queue the actual delivery of a chunk ()
	glog.V(logger.Detail).Infof("Depo.HandleDeliveryRequestMsg: received %v delivery requests: %v", len(deliver), deliver)
	for _, sreq := range deliver {
		// TODO: look up in cache here or in deliveries
		// priorities are taken from the message so the remote party can
		// reprioritise to at their leisure
		// r = self.pullCached(sreq.Key) // pulls and deletes from cache
		Push(p, sreq.Key, sreq.Priority)
	}

	// sends it out as unsyncedKeysMsg
	p.syncer.sendUnsyncedKeys()
	return nil
}

// the entrypoint for store requests coming from the bzz wire protocol
// if key found locally, return. otherwise
// remote is untrusted, so hash is verified and chunk passed on to NetStore
func (self *Depo) HandleStoreRequestMsg(req *storeRequestMsgData, p *peer) {
	req.from = p
	chunk, err := self.localStore.Get(req.Key)
	switch {
	case err != nil:
		glog.V(logger.Detail).Infof("Depo.handleStoreRequest: %v not found locally. create new chunk/request", req.Key)
		// not found in memory cache, ie., a genuine store request
		// create chunk
		chunk = storage.NewChunk(req.Key, nil)

	case chunk.SData == nil:
		// found chunk in memory store, needs the data, validate now
		hasher := self.hashfunc()
		hasher.Write(req.SData)
		if !bytes.Equal(hasher.Sum(nil), req.Key) {
			// data does not validate, ignore
			// TODO: peer should be penalised/dropped?
			glog.V(logger.Warn).Infof("Depo.HandleStoreRequest: chunk invalid. store request ignored: %v", req)
			return
		}
		glog.V(logger.Detail).Infof("Depo.HandleStoreRequest: %v. request entry found", req)

	default:
		// data is found, store request ignored
		// this should update access count?
		glog.V(logger.Detail).Infof("Depo.HandleStoreRequest: %v found locally. ignore.", req)
		return
	}

	// update chunk with size and data
	chunk.SData = req.SData // protocol validates that SData is minimum 9 bytes long (int64 size  + at least one byte of data)
	chunk.Size = int64(binary.LittleEndian.Uint64(req.SData[0:8]))
	glog.V(logger.Detail).Infof("delivery of %p from %v", chunk, p)
	chunk.Source = p
	self.netStore.Put(chunk)
}

// entrypoint for retrieve requests coming from the bzz wire protocol
// checks swap balance - return if peer has no credit
func (self *Depo) HandleRetrieveRequestMsg(req *retrieveRequestMsgData, p *peer) {
	req.from = p
	// swap - record credit for 1 request
	// note that only charge actual reqsearches
	var err error
	if p.swap != nil {
		err = p.swap.Add(1)
	}
	if err != nil {
		glog.V(logger.Warn).Infof("Depo.HandleRetrieveRequest: %v - cannot process request: %v", req.Key.Log(), err)
		return
	}

	// call storage.NetStore#Get which
	// blocks until local retrieval finished
	// launches cloud retrieval
	chunk, _ := self.netStore.Get(req.Key)
	req = self.strategyUpdateRequest(chunk.Req, req)
	// check if we can immediately deliver
	if chunk.SData != nil {
		glog.V(logger.Detail).Infof("Depo.HandleRetrieveRequest: %v - content found, delivering...", req.Key.Log())

		if req.MaxSize == 0 || int64(req.MaxSize) >= chunk.Size {
			sreq := &storeRequestMsgData{
				Id:             req.Id,
				Key:            chunk.Key,
				SData:          chunk.SData,
				requestTimeout: req.timeout, //
			}
			p.syncer.addRequest(sreq, DeliverReq)
		} else {
			glog.V(logger.Detail).Infof("Depo.HandleRetrieveRequest: %v - content found, not wanted", req.Key.Log())
		}
	} else {
		glog.V(logger.Detail).Infof("Depo.HandleRetrieveRequest: %v - content not found locally. asked swarm for help. will get back", req.Key.Log())
	}
}

// add peer request the chunk and decides the timeout for the response if still searching
func (self *Depo) strategyUpdateRequest(rs *storage.RequestStatus, origReq *retrieveRequestMsgData) (req *retrieveRequestMsgData) {
	glog.V(logger.Detail).Infof("Depo.strategyUpdateRequest: key %v", origReq.Key.Log())
	// we do not create an alternative one
	req = origReq
	if rs != nil {
		self.addRequester(rs, req)
		req.setTimeout(self.searchTimeout(rs, req))
	}
	return
}

// decides the timeout promise sent with the immediate peers response to a retrieve request
// if timeout is explicitly set and expired
func (self *Depo) searchTimeout(rs *storage.RequestStatus, req *retrieveRequestMsgData) (timeout *time.Time) {
	reqt := req.getTimeout()
	t := time.Now().Add(searchTimeout)
	if reqt != nil && reqt.Before(t) {
		return reqt
	} else {
		return &t
	}
}

/*
adds a new peer to an existing open request
only add if less than requesterCount peers forwarded the same request id so far
note this is done irrespective of status (searching or found)
*/
func (self *Depo) addRequester(rs *storage.RequestStatus, req *retrieveRequestMsgData) {
	glog.V(logger.Detail).Infof("Depo.addRequester: key %v - add peer to req.Id %v", req.Key.Log(), req.from, req.Id)
	list := rs.Requesters[req.Id]
	rs.Requesters[req.Id] = append(list, req)
}
