package bzz

import (
	"math/rand"
	"sync"
	"time"
)

type netStore struct {
	localStore *localStore
	lock       sync.Mutex
	hive       *hive
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

const (
	requesterCount = 3
)

var (
	searchTimeout = 3 * time.Second
)

type requestStatus struct {
	key        Key
	status     int
	requesters map[int64][]*retrieveRequestMsgData
	C          chan bool
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

func (self *netStore) put(entry *Chunk) {
	self.localStore.Put(entry)
	self.store(entry)
	// only send responses once
	if entry.req != nil && entry.req.status == reqSearching {
		entry.req.status = reqFound
		close(entry.req.C)
		self.propagateResponse(entry)
	}
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

// waits for response or times  out
func (self *netStore) Get(key Key) (chunk *Chunk, err error) {
	chunk = self.get(key)
	id := generateId()
	timeout := time.Now().Add(searchTimeout)
	if chunk.Data == nil {
		self.startSearch(chunk, id, timeout)
	}
	timer := time.After(searchTimeout)
	select {
	case <-timer:
		err = notFound
	case <-chunk.req.C:
	}
	return
}

func (self *netStore) get(key Key) (chunk *Chunk) {
	var err error
	chunk, err = self.localStore.Get(key)
	// we assume that a returned chunk is the one stored in the memory cache
	if err != nil {
		// no data and no request status
		chunk = &Chunk{
			Key: key,
		}
		self.localStore.memStore.Put(chunk)
	}

	if chunk.req == nil {
		chunk.req = new(requestStatus)
	}
	return
}

func (self *netStore) addRetrieveRequest(req *retrieveRequestMsgData) {

	self.lock.Lock()
	defer self.lock.Unlock()

	chunk := self.get(req.Key)

	send, timeout := self.strategyUpdateRequest(chunk.req, req) // may change req status

	if send == storeRequestMsg {
		self.deliver(req, chunk)
	} else {
		// we might need chunk.req to cache relevant peers response, or would it expire?
		self.peers(req, chunk, *timeout)
		if timeout != nil {
			self.startSearch(chunk, req.Id, *timeout)
		}
	}

}

// it's assumed that caller holds the lock
func (self *netStore) startSearch(chunk *Chunk, id int64, timeout time.Time) {
	chunk.req.status = reqSearching
	peers := self.hive.getPeers(chunk.Key)
	req := &retrieveRequestMsgData{
		Key:     chunk.Key,
		Id:      id,
		Timeout: timeout,
	}
	for _, peer := range peers {
		peer.retrieve(req)
	}
}

func generateId() int64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return r.Int63()
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
func (self *netStore) strategyUpdateRequest(rs *requestStatus, req *retrieveRequestMsgData) (msgTyp int, timeout *time.Time) {

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

func (self *netStore) propagateResponse(chunk *Chunk) {
	for id, requesters := range chunk.req.requesters {
		counter := requesterCount
		msg := &storeRequestMsgData{
			Key:  chunk.Key,
			Data: chunk.Data,
			Size: chunk.Size,
			Id:   id,
		}
		for _, req := range requesters {
			if req.Timeout.After(time.Now()) {
				go req.peer.store(msg)
				counter--
				if counter <= 0 {
					break
				}
			}
		}
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
	id := generateId()
	req := &storeRequestMsgData{
		Key:  chunk.Key,
		Data: chunk.Data,
		Id:   id,
		Size: chunk.Size,
	}
	for _, peer := range self.hive.getPeers(chunk.Key) {
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

func (self *netStore) searchTimeout(rs *requestStatus, req *retrieveRequestMsgData) (timeout *time.Time) {
	t := time.Now().Add(searchTimeout)
	if req.Timeout.Before(t) {
		return &req.Timeout
	} else {
		return &t
	}
}
