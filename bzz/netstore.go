package bzz

/*
TODO:
- put Data -> Reader logic to chunker
- clarify dpa / localStore / hive / netstore naming and division of labour and entry points for local/remote requests
- figure out if its a problem that peers on requester list may disconnect while searching
- Id (nonce/requester map key) should probs be random byte slice or (hash of) originator's address to avoid collisions
- rework protocol errors using errs after PR merged
- integrate cademlia as peer pool
- finish the net/dht logic, startSearch and storage
*/

import (
	"math/rand"
	"sync"
	"time"
)

// This is a mock implementation with a fixed peer pool with no distinction between peers
type peerPool struct {
	pool map[string]peer
}

func (self *peerPool) addPeer(p peer) {
	self.pool[string(p.pubkey)] = p
}

func (self *peerPool) removePeer(p peer) {
	delete(self.pool, string(p.pubkey))
}

func (self *peerPool) getPeers(target Key) (peers []peer) {
	for _, value := range self.pool {
		peers = append(peers, value)
	}
	return
}

type netStore struct {
	localStore *localStore
	lock       sync.Mutex
	peerPool   *peerPool
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
	pubkey []byte
}

type requestStatus struct {
	key        Key
	status     int
	requesters map[int64][]*retrieveRequestMsgData
}

// it's assumed that caller holds the lock
func (self *netStore) startSearch(chunk *Chunk) {
	chunk.req.status = reqSearching
	// implement search logic here
}

/*
adds a new peer to an existing open request
only add if less than requesterCount peers forwarded the same request id so far
note this is done irrespective of status (searching or found/timedOut)
*/
func (self *netStore) addRequester(rs *requestStatus, req *retrieveRequestMsgData) {
	list := rs.requesters[req.Id]
	rs.requesters[req.Id] = append(list, req)
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
func (self *netStore) strategyUpdateRequest(rs *requestStatus, req *retrieveRequestMsgData) (msgTyp int, timeout time.Time) {

	switch rs.status {
	case reqSearching:
		msgTyp = peersMsg
		timeout = self.searchTimeout(rs, req)
	case reqTimedOut:
		msgTyp = peersMsg
	case reqFound:
		msgTyp = storeRequestMsg
	}
	return

}

func (self *netStore) put(entry *Chunk) {
	self.localStore.Put(entry)
	self.store(entry)
	// only send responses once
	if entry.req != nil && entry.req.status == reqSearching {
		entry.req.status = reqFound
		self.propagateResponse(entry)
	}
}

func (self *netStore) Put(entry *Chunk) {
	chunk, err := self.localStore.Get(entry.Key)
	if err != nil {
		chunk = entry
	} else if chunk.Data == nil {
		chunk.Data = entry.Data
		chunk.Size = entry.Size
	} else {
		return
	}
	self.put(chunk)
}

func (self *netStore) addStoreRequest(req *storeRequestMsgData) {
	self.lock.Lock()
	defer self.lock.Unlock()
	chunk, err := self.localStore.Get(req.Key)
	// we assume that a returned chunk is the one stored in the memory cache
	if err != nil {
		chunk = &Chunk{
			Key:  req.Key,
			Data: req.Data,
			Size: req.Size,
		}
	} else if chunk.Data == nil {
		chunk.Data = req.Data
		chunk.Size = req.Size
	} else {
		return
	}
	self.put(chunk)
}

func (self *netStore) propagateResponse(chunk *Chunk) {
	// send chunk to first requesterCount peer of each Id
}

func (self *netStore) addRetrieveRequest(req *retrieveRequestMsgData) {

	self.lock.Lock()
	defer self.lock.Unlock()

	chunk, err := self.localStore.Get(req.Key)
	// we assume that a returned chunk is the one stored in the memory cache
	if err != nil {
		// no data and no request status
		chunk = &Chunk{
			Key: req.Key,
		}
		self.localStore.memStore.Put(chunk)
	}

	if chunk.req == nil {
		chunk.req = new(requestStatus)
		if chunk.Data == nil {
			self.startSearch(chunk)
		}
	}

	send, timeout := self.strategyUpdateRequest(chunk.req, req) // may change req status

	if send == storeRequestMsg {
		self.deliver(req, chunk)
	} else {
		// we might need chunk.req to cache relevant peers response, or would it expire?
		self.peers(req, chunk, timeout)
	}

}

func (self *netStore) deliver(req *retrieveRequestMsgData, chunk *Chunk) {
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

func (self *netStore) store(chunk *Chunk) {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	req := &storeRequestMsgData{
		Key:  chunk.Key,
		Data: chunk.Data,
		Id:   r.Int63(),
		Size: chunk.Size,
	}
	for _, peer := range self.peerPool.getPeers(chunk.Key) {
		go peer.store(req)
	}
}

func (self *netStore) peers(req *retrieveRequestMsgData, chunk *Chunk, timeout time.Time) {
	peersData := &peersMsgData{
		Peers:   []*peerAddr{}, // get proximity bin from cademlia routing table
		Key:     req.Key,
		Id:      req.Id,
		Timeout: timeout,
	}
	req.peer.peers(peersData)
}

func (self *netStore) searchTimeout(rs *requestStatus, req *retrieveRequestMsgData) (timeout time.Time) {
	return
}

// these should go to cademlia
func (self *netStore) addPeers(req *peersMsgData) (err error) {
	return
}

func (self *netStore) removePeer(p peer) {
	return
}
