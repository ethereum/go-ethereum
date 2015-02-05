package bzz

/*
TODO:
- put Data -> Reader logic to chunker
- clarify dpa / hive / netstore naming and division of labour and entry points for local/remote requests
- figure out if its a problem that peers on requester list may disconnect while searching
- Id (nonce/requester map key) should probs be random byte slice or (hash of) originator's address to avoid collisions
- rework protocol errors using errs after PR merged
- integrate cademlia as peer pool
- finish the net/dht logic, startSearch and storage
*/

import (
	"sync"
	"time"
)

type Hive struct {
	dpa      *DPA
	memstore *memStore
	lock     sync.Mutex
}

/*
request status values:
- blank
- started searching
- timed out
- found
*/

const (
	reqBlank = iota
	reqSearching
	reqTimedOut
	reqFound
)

const requesterCount = 3

type peer struct {
	*bzzProtocol
}

type requestStatus struct {
	key        Key
	status     int
	requesters map[uint64][]*retrieveRequestMsgData
}

// it's assumed that caller holds the lock
func (self *Hive) startSearch(chunk *Chunk) {
	chunk.req.status = reqSearching
	// implement search logic here
}

/*
adds a new peer to an existing open request
only add if less than requesterCount peers forwarded the same request id so far
note this is done irrespective of status (searching or found/timedOut)
*/
func (self *Hive) addRequester(rs *requestStatus, req *retrieveRequestMsgData) (added bool) {
	list := rs.requesters[req.Id]
	if len(list) < requesterCount {
		rs.requesters[req.Id] = append(list, req)
		added = true
	}
	return
}

/*
decides how to respond to a retrieval request
updates the request status if needed
returns
send bool: true if chunk is to be delivered, false if respond with peers (as for now)
timeout: if respond with peers, timeout indicates our bet
this is the most simplistic implementation:
 - respond with delivery iff less than requesterCount peers forwarded the same request id so far and chunk is found
 - respond with reject (peers and zero timeout) if given up
 - respond with peers and timeout if still searching
! in the last case as well, we should respond with reject if already got requesterCount peers with that exact id
*/
func (self *Hive) strategyUpdateRequest(rs *requestStatus, req *retrieveRequestMsgData) (msgTyp int, timeout time.Time) {

	switch rs.status {
	case reqSearching:
		if self.addRequester(rs, req) {
			msgTyp = peersMsg
			timeout = self.searchTimeout(rs, req)
		}
	case reqTimedOut:
		msgTyp = peersMsg
	case reqFound:
		if self.addRequester(rs, req) {
			msgTyp = storeRequestMsg
		}
	}
	return

}

func (self *Hive) addStoreRequest(req *storeRequestMsgData) (err error) {

	self.lock.Lock()
	defer self.lock.Unlock()
	// TODO:
	return
}

func (self *Hive) addRetrieveRequest(req *retrieveRequestMsgData) {

	self.lock.Lock()
	defer self.lock.Unlock()

	chunk, err := self.dpa.Get(req.Key)
	// we assume that a returned chunk is the one stored in the memory cache
	if err != nil {
		// no data and no request status
		chunk = &Chunk{
			Key: req.Key,
		}
		self.memstore.Put(chunk)
	}

	if chunk.req == nil {
		chunk.req = new(requestStatus)
		if chunk.Data == nil {
			self.startSearch(chunk)
		}
	}

	send, timeout := self.strategyUpdateRequest(chunk.req, req) // may change req status

	if send {
		self.deliver(req, chunk)
	} else {
		// we might need chunk.req to cache relevant peers response, or would it expire?
		self.peers(req, chunk, timeout)
	}

}

func (self *Hive) deliver(req *retrieveRequestMsgData, chunk *Chunk) {
	storeReq := &storeRequestMsgData{
		Key:            req.Key,
		Id:             req.Id,
		Data:           chunk.Data,
		Size:           chunk.Size,
		RequestTimeout: req.Timeout, //
		// StorageTimeout time.Time // expiry of content
		// Metadata       metaData
	}
	req.peer.store(storeReq)
}

func (self *Hive) peers(req *retrieveRequestMsgData, chunk *Chunk, timeout time.Time) {
	peersData := &peersMsgData{
		Peers:   []*peerAddr{}, // get proximity bin from cademlia routing table
		Key:     req.Key,
		Id:      req.Id,
		Timeout: timeout,
	}
	req.peer.peers(peersData)
}

func (self *Hive) searchTimeout(rs *requestStatus, req *retrieveRequestMsgData) (timeout time.Time) {
	return
}

// these should go to cademlia
func (self *Hive) addPeers(req *peersMsgData) (err error) {
	return
}

func (self *Hive) removePeer(p peer) {
	return
}
