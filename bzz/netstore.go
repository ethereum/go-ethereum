package bzz

import (
	"math/rand"
	"sync"
	"time"
)

type NetStore struct {
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

func NewNetStore(path string) *NetStore {
	dbStore, _ := newDbStore(path)
	return &NetStore{
		localStore: &localStore{
			memStore: newMemStore(dbStore),
			dbStore:  dbStore,
		}, hive: newHive(),
	}
}

func (self *NetStore) Put(entry *Chunk) {
	chunk, err := self.localStore.Get(entry.Key)
	dpaLogger.Debugf("NetStore.Put: localStore.Get returned with %v.", err)
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

func (self *NetStore) put(entry *Chunk) {
	self.localStore.Put(entry)
	dpaLogger.Debugf("NetStore.put: localStore.Put of %064x completed.", entry.Key)
	go self.store(entry)
	// only send responses once
	dpaLogger.Debugf("NetStore.put: req: %#v", entry.req)
	if entry.req != nil && entry.req.status == reqSearching {
		entry.req.status = reqFound
		close(entry.req.C)
		self.propagateResponse(entry)
	}
}

func (self *NetStore) addStoreRequest(req *storeRequestMsgData) {
	self.lock.Lock()
	defer self.lock.Unlock()
	chunk, err := self.localStore.Get(req.Key)
	// we assume that a returned chunk is the one stored in the memory cache
	if err != nil {
		chunk = &Chunk{
			Key:  req.Key,
			Data: req.Data,
			Size: int64(req.Size),
		}
	} else if chunk.Data == nil {
		chunk.Data = req.Data
		chunk.Size = int64(req.Size)
	} else {
		return
	}
	self.put(chunk)
}

// waits for response or times  out
func (self *NetStore) Get(key Key) (chunk *Chunk, err error) {
	chunk = self.get(key)
	id := generateId()
	timeout := time.Now().Add(searchTimeout)
	if chunk.Data == nil {
		self.startSearch(chunk, id, &timeout)
	} else {
		return
	}
	// TODO: use self.timer time.Timer and reset with defer disableTimer
	timer := time.After(searchTimeout)
	select {
	case <-timer:
		dpaLogger.Debugf("NetStore.Get: %064x request time out ", key)
		err = notFound
	case <-chunk.req.C:
		dpaLogger.Debugf("NetStore.get: %064x retrieved", key)

	}
	return
}

func (self *NetStore) get(key Key) (chunk *Chunk) {
	var err error
	chunk, err = self.localStore.Get(key)
	dpaLogger.Debugf("NetStore.get: localStore.Get of %064x returned with %v.", key, err)
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

func (self *NetStore) addRetrieveRequest(req *retrieveRequestMsgData) {

	self.lock.Lock()
	defer self.lock.Unlock()

	chunk := self.get(req.Key)
	if chunk.Data == nil {
		chunk.req.status = reqSearching
	} else {
		chunk.req.status = reqFound
	}

	t := time.Now().Add(10 * time.Second)
	req.timeout = &t

	send, timeout := self.strategyUpdateRequest(chunk.req, req) // may change req status

	if send == storeRequestMsg {
		dpaLogger.Debugf("NetStore.addRetrieveRequest: %064x - content found, delivering...", req.Key)
		self.deliver(req, chunk)
	} else {
		// we might need chunk.req to cache relevant peers response, or would it expire?
		self.peers(req, chunk, timeout)
		dpaLogger.Debugf("NetStore.addRetrieveRequest: %064x - searching.... responding with peers...", req.Key)

		if timeout != nil {
			self.startSearch(chunk, int64(req.Id), timeout)
		}
	}

}

// it's assumed that caller holds the lock
func (self *NetStore) startSearch(chunk *Chunk, id int64, timeout *time.Time) {
	chunk.req.status = reqSearching
	dpaLogger.Debugf("NetStore.startSearch: %064x - getting peers from cademlia...", chunk.Key)
	peers := self.hive.getPeers(chunk.Key)
	req := &retrieveRequestMsgData{
		Key:     chunk.Key,
		Id:      uint64(id),
		timeout: timeout,
	}
	for _, peer := range peers {
		dpaLogger.Debugf("NetStore.startSearch: sending retrieveRequests to peer [%064x]", req.Key)
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
func (self *NetStore) addRequester(rs *requestStatus, req *retrieveRequestMsgData) {
	dpaLogger.Debugf("NetStore.addRequester: key %064x - add peer [%#v] to req.Id %064x", req.Key, req.peer, req.Id)
	list := rs.requesters[int64(req.Id)]
	rs.requesters[int64(req.Id)] = append(list, req)
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
func (self *NetStore) strategyUpdateRequest(rs *requestStatus, req *retrieveRequestMsgData) (msgTyp int, timeout *time.Time) {
	dpaLogger.Debugf("NetStore.strategyUpdateRequest: key %064x", req.Key)

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

func (self *NetStore) propagateResponse(chunk *Chunk) {
	dpaLogger.Debugf("NetStore.propagateResponse: key %064x", chunk.Key)
	for id, requesters := range chunk.req.requesters {
		counter := requesterCount
		dpaLogger.Debugf("NetStore.propagateResponse id %064x", id)
		msg := &storeRequestMsgData{
			Key:  chunk.Key,
			Data: chunk.Data,
			Size: uint64(chunk.Size),
			Id:   uint64(id),
		}
		for _, req := range requesters {
			if req.timeout.After(time.Now()) {
				dpaLogger.Debugf("NetStore.propagateResponse store -> %064x with %v", req.Id, req.peer)
				go req.peer.store(msg)
				counter--
				if counter <= 0 {
					break
				}
			}
		}
	}
}

func (self *NetStore) deliver(req *retrieveRequestMsgData, chunk *Chunk) {
	storeReq := &storeRequestMsgData{
		Key:            req.Key,
		Id:             req.Id,
		Data:           chunk.Data,
		Size:           uint64(chunk.Size),
		requestTimeout: req.timeout, //
		// StorageTimeout *time.Time // expiry of content
		// Metadata       metaData
	}
	req.peer.store(storeReq)
}

func (self *NetStore) store(chunk *Chunk) {
	id := generateId()
	req := &storeRequestMsgData{
		Key:  chunk.Key,
		Data: chunk.Data,
		Id:   uint64(id),
		Size: uint64(chunk.Size),
	}
	for _, peer := range self.hive.getPeers(chunk.Key) {
		go peer.store(req)
	}
}

func (self *NetStore) peers(req *retrieveRequestMsgData, chunk *Chunk, timeout *time.Time) {
	peersData := &peersMsgData{
		Peers:   []*peerAddr{}, // get proximity bin from cademlia routing table
		Key:     req.Key,
		Id:      req.Id,
		timeout: timeout,
	}
	req.peer.peers(peersData)
}

func (self *NetStore) searchTimeout(rs *requestStatus, req *retrieveRequestMsgData) (timeout *time.Time) {
	t := time.Now().Add(searchTimeout)
	if req.timeout != nil && req.timeout.Before(t) {
		return req.timeout
	} else {
		return &t
	}
}
