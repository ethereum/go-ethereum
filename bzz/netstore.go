package bzz

import (
	"bytes"
	"encoding/binary"
	"math/rand"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
)

/*
netStore is a network storage for chunks (a dht = distributed hash table of sorts)
it is the entrypoint for chunk store/retrieval requests
both local (coming from DPA api) and network (coming from peers via bzz protocol)
it implements the ChunkStore interface and embeds local storage

its called by the bzz protocol instance running on each peer, so this is heavily
parallelised.
For routing and peer selection it embeds the hive which is a kademlia-driven
logistics engine for swarm.
It is aware of the node's network address
*/
type netStore struct {
	localStore *localStore
	lock       sync.Mutex
	hive       *hive
	self       *peerAddr
	ready      chan bool
	path       string
}

/*
request status values:
- started searching
- found
*/

const (
	reqSearching = iota // after search for chunk started until found or timed out
	reqFound            // chunk found search terminated
)

const (
	// maximum number of peers that a retrieved message is delivered to
	requesterCount = 3
)

var (
	// timeout interval before retrieval is timed out
	searchTimeout = 3 * time.Second
	zeroKey       = Key(common.Hash{}.Bytes())
)

// each chunk when first requested opens a record associated with the request
// next time a request for the same chunk arrives, this record is updated
// this request status keeps track of the request ID-s as well as the requesting
// peers and has a channel that is closed when the chunk is retrieved. Multiple
// local callers can wait on this channel (or combined with a timeout, block with a
// select).
type requestStatus struct {
	key        Key
	status     int
	requesters map[uint64][]*retrieveRequestMsgData
	C          chan bool
}

func newRequestStatus() *requestStatus {
	return &requestStatus{
		requesters: make(map[uint64][]*retrieveRequestMsgData),
		C:          make(chan bool),
	}
}

// netstore contructor, takes path argument that is used to initialise dbStore,
// the persistent (disk) storage component of localStore
// the second argument is the hive, the connection/logistics manager for the node
func newNetStore(path string, h *hive) (netstore *netStore, err error) {
	dbStore, err := newDbStore(path)
	if err != nil {
		return
	}
	netstore = &netStore{
		localStore: &localStore{
			memStore: newMemStore(dbStore),
			dbStore:  dbStore,
		},
		path:  path,
		hive:  h,
		ready: make(chan bool),
	}
	return
}

// netStore is started only when the network is started and the node has a
// network address advertised to peers via the bzz protocol handshake (Status Msg)
func (self *netStore) start(addr *peerAddr) (err error) {
	self.self = addr
	close(self.ready)
	return
}

// The bzz protocol instance is started by the network when a bzz capable peer
// is connected. Since this is on a different thread, the protocol needs to
// wait for the netStore to be started before it sends out the handshake (Status Msg)
// by closing the ready channel by start addr will block until network address
// is available to the netStore also.
func (self *netStore) addr() *peerAddr {
	<-self.ready
	return self.self
}

// not relevant as of yet
// but will quit the synchronisation loop(s)
func (self *netStore) stop() (err error) {
	return
}

// called from dpa, entrypoint for *local* chunk store requests
func (self *netStore) Put(entry *Chunk) {
	chunk, err := self.localStore.Get(entry.Key)
	glog.V(logger.Debug).Infof("[BZZ] netStore.Put: localStore.Get returned with %v.", err)
	if err != nil {
		chunk = entry
	} else if chunk.SData == nil {
		chunk.SData = entry.SData
		chunk.Size = entry.Size
	} else {
		return
	}
	// from this point on the storage logic is the same with network storage requests
	self.put(chunk)
}

// store logic common to local and network chunk store requests
func (self *netStore) put(entry *Chunk) {
	self.localStore.Put(entry)
	glog.V(logger.Debug).Infof("[BZZ] netStore.put: localStore.Put of %064x completed, %d bytes (%p).", entry.Key, len(entry.SData), entry)
	if entry.req != nil {
		// if entry had a requst status, it means it has recently been requested
		// by at least one peer
		if entry.req.status == reqSearching {
			// the status is set to found
			entry.req.status = reqFound
			// closing C singals to other routines (local requests)
			// that the chunk is has been retrieved
			close(entry.req.C)
			// deliver the chunk to requesters upstream
			self.propagateResponse(entry)
		}
	} else {
		go self.store(entry)
	}
}

// store propagates store requests to specific peers given by the kademlia hive
// except for peers that the store request came from (if any)
func (self *netStore) store(chunk *Chunk) {
	for _, peer := range self.hive.getPeers(chunk.Key, 0) {
		if chunk.source == nil || peer.Addr() != chunk.source.Addr() {
			peer.storeRequest(chunk.Key)
		}
	}
}

// the entrypoint for network store requests
func (self *netStore) addStoreRequest(req *storeRequestMsgData) {
	self.lock.Lock()
	defer self.lock.Unlock()
	glog.V(logger.Debug).Infof("[BZZ] netStore.addStoreRequest: req = %v", req)
	chunk, err := self.localStore.Get(req.Key)
	glog.V(logger.Debug).Infof("[BZZ] netStore.addStoreRequest: chunk reference %p", chunk)
	// we assume that a returned chunk is the one stored in the memory cache
	if err != nil {
		chunk = &Chunk{
			Key:   req.Key,
			SData: req.SData,
			Size:  int64(binary.LittleEndian.Uint64(req.SData[0:8])),
		}
	} else if chunk.SData == nil {
		// need data, validate now
		hasher := hasherfunc.New()
		hasher.Write(req.SData)
		if !bytes.Equal(hasher.Sum(nil), req.Key) {
			// data does not validate, ignore
			glog.V(logger.Warn).Infof("[BZZ] netStore.addStoreRequest: chunk invalid. store request ignored: %v", req)
			return
		}

		chunk.SData = req.SData
		chunk.Size = int64(binary.LittleEndian.Uint64(req.SData[0:8]))
	} else {
		// data is found, store request ignored
		return
	}
	chunk.source = req.peer
	self.put(chunk)
}

// Get is the entrypoint for local retrieve requests
// waits for response or times out
func (self *netStore) Get(key Key) (chunk *Chunk, err error) {
	chunk = self.get(key)
	timeout := time.Now().Add(searchTimeout)
	if chunk.SData == nil {
		req := &retrieveRequestMsgData{
			Key: chunk.Key,
			Id:  generateId(),
		}
		req.setTimeout(&timeout)
		self.startSearch(req, chunk)
	} else {
		return
	}
	// TODO: use self.timer time.Timer and reset with defer disableTimer
	timer := time.After(searchTimeout)
	select {
	case <-timer:
		glog.V(logger.Debug).Infof("[BZZ] netStore.Get: %064x request time out ", key)
		err = notFound
	case <-chunk.req.C:
		glog.V(logger.Debug).Infof("[BZZ] netStore.Get: %064x retrieved, %d bytes (%p)", key, len(chunk.SData), chunk)
	}
	return
}

// retrieve logic common for local and network chunk retrieval
func (self *netStore) get(key Key) (chunk *Chunk) {
	var err error
	chunk, err = self.localStore.Get(key)
	glog.V(logger.Debug).Infof("[BZZ] netStore.get: localStore.Get of %064x returned with %v.", key, err)
	// we assume that a returned chunk is the one stored in the memory cache
	if err != nil {
		// no data and no request status
		chunk = &Chunk{
			Key: key,
		}
		self.localStore.memStore.Put(chunk)
	}

	if chunk.req == nil {
		chunk.req = newRequestStatus()
	}
	return
}

// entrypoint for network retrieve requests
func (self *netStore) addRetrieveRequest(req *retrieveRequestMsgData) {

	self.lock.Lock()
	defer self.lock.Unlock()
	// if key is zero or nil or matches requester address, then the request
	// is a self lookup and not to be forwarded
	if !req.isLookup() {
		chunk := self.get(req.Key)
		if chunk.SData != nil {
			chunk.req.status = reqFound
		}

		req = self.strategyUpdateRequest(chunk.req, req) // may change req status

		if chunk.req.status == reqFound {
			glog.V(logger.Debug).Infof("[BZZ] netStore.addRetrieveRequest: %064x - content found, delivering...", req.Key)
			self.deliver(req, chunk)
			return
		}

		glog.V(logger.Debug).Infof("[BZZ] netStore.addRetrieveRequest: %064x. Start net search by forwarding retrieve request. For now responding with peers.", req.Key)
		self.startSearch(req, chunk)

	} else {
		glog.V(logger.Debug).Infof("[BZZ] netStore.addRetrieveRequest: self lookup for %v: responding with peers...", req.peer)
	}

	self.peers(req)
}

// logic propagating retrieve requests to peers given by the kademlia hive
// it's assumed that caller holds the lock
func (self *netStore) startSearch(req *retrieveRequestMsgData, chunk *Chunk) {
	chunk.req.status = reqSearching
	peers := self.hive.getPeers(chunk.Key, 0)
	glog.V(logger.Debug).Infof("[BZZ] netStore.startSearch: %064x - received %d peers from KΛÐΞMLIΛ...", chunk.Key, len(peers))
	for _, peer := range peers {
		glog.V(logger.Debug).Infof("[BZZ] netStore.startSearch: sending retrieveRequests to peer [%064x]", req.Key)
		glog.V(logger.Debug).Infof("[BZZ] req.requesters: %v", chunk.req.requesters)
		var requester bool
	OUT:
		for _, recipients := range chunk.req.requesters {
			for _, recipient := range recipients {
				if recipient.peer.Addr() == peer.Addr() {
					requester = true
					break OUT
				}
			}
		}
		if !requester {
			peer.retrieve(req)
		}
	}
}

func generateId() uint64 {
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	return uint64(r.Int63())
}

/*
adds a new peer to an existing open request
only add if less than requesterCount peers forwarded the same request id so far
note this is done irrespective of status (searching or found)
*/
func (self *netStore) addRequester(rs *requestStatus, req *retrieveRequestMsgData) {
	glog.V(logger.Debug).Infof("[BZZ] netStore.addRequester: key %064x - add peer [%v] to req.Id %v", req.Key, req.peer, req.Id)
	list := rs.requesters[req.Id]
	rs.requesters[req.Id] = append(list, req)
}

// add peer request the chunk and decides the timeout for the response if still searching
func (self *netStore) strategyUpdateRequest(rs *requestStatus, origReq *retrieveRequestMsgData) (req *retrieveRequestMsgData) {
	glog.V(logger.Debug).Infof("[BZZ] netStore.strategyUpdateRequest: key %064x", origReq.Key)
	// we do not create an alternative one
	req = origReq
	if rs != nil {
		self.addRequester(rs, req)
		if rs.status == reqSearching {
			req.setTimeout(self.searchTimeout(rs, req))
		}
	}
	return
}

// once a chunk is found propagate it its requesters unless timed out
func (self *netStore) propagateResponse(chunk *Chunk) {
	glog.V(logger.Debug).Infof("[BZZ] netStore.propagateResponse: key %064x", chunk.Key)
	for id, requesters := range chunk.req.requesters {
		counter := requesterCount
		glog.V(logger.Debug).Infof("[BZZ] netStore.propagateResponse id %064x", id)
		msg := &storeRequestMsgData{
			Key:   chunk.Key,
			SData: chunk.SData,
			Id:    uint64(id),
		}
		for _, req := range requesters {
			if req.timeout == nil || req.timeout.After(time.Now()) {
				glog.V(logger.Debug).Infof("[BZZ] netStore.propagateResponse store -> %064x with %v", req.Id, req.peer)
				go req.peer.store(msg)
				counter--
				if counter <= 0 {
					break
				}
			}
		}
	}
}

// called on each request when a chunk is found,
// delivery is done by sending a request to the requesting peer
func (self *netStore) deliver(req *retrieveRequestMsgData, chunk *Chunk) {
	// here we should check MaxSize if set to a value lower than chunk.Size
	// we withhold
	// also check for timeout and withhold if expired
	if req.MaxSize == 0 || int64(req.MaxSize) >= chunk.Size {
		storeReq := &storeRequestMsgData{
			Key:            req.Key,
			Id:             req.Id,
			SData:          chunk.SData,
			requestTimeout: req.timeout, //
			// StorageTimeout *time.Time // expiry of content
			// Metadata       metaData
		}
		req.peer.store(storeReq)
	}
}

// the immediate response to a retrieve request,
// sends relevant peer data given by the kademlia hive to the requester
func (self *netStore) peers(req *retrieveRequestMsgData) {
	// FIXME: should check req.MaxPeers but then should not default to zero or make sure we set it when sending retrieveRequests
	// we might need chunk.req to cache relevant peers response, or would it expire?
	if req != nil && req.MaxPeers >= 0 {
		var addrs []*peerAddr
		if req.timeout == nil || time.Now().Before(*(req.timeout)) {
			key := req.Key
			if isZeroKey(key) {
				key = req.peer.addrKey()
				req.Key = nil
			}
			for _, peer := range self.hive.getPeers(key, int(req.MaxPeers)) {
				addrs = append(addrs, peer.remoteAddr)
			}
			glog.V(logger.Debug).Infof("[BZZ] netStore.peers sending %d addresses. req.Id: %v, req.Key: %v", len(addrs), req.Id, req.Key)

			peersData := &peersMsgData{
				Peers: addrs,
				Key:   req.Key,
				Id:    req.Id,
			}
			peersData.setTimeout(req.timeout)
			req.peer.peers(peersData)
		}
	}
}

// decides the timeout promise sent with the immediate peers response to a retrieve request
// if timeout is explicitly set and expired
func (self *netStore) searchTimeout(rs *requestStatus, req *retrieveRequestMsgData) (timeout *time.Time) {
	reqt := req.getTimeout()
	t := time.Now().Add(searchTimeout)
	if reqt != nil && reqt.Before(t) {
		return reqt
	} else {
		return &t
	}
}
