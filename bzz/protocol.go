package bzz

/*
BZZ implements the bzz wire protocol of swarm
the protocol instance is launched on each peer by the network layer if the
BZZ protocol handler is registered on the p2p server.

The protocol takes care of actually communicating the bzz protocol
encoding and decoding requests for storage and retrieval
handling the protocol handshake
dispaching to netstore for handling the DHT logic
registering peers in the KΛÐΞMLIΛ table via the hive logistic manager
*/

import (
	"bytes"
	"fmt"
	"net"
	"path"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/kademlia"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/errs"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/syndtr/goleveldb/leveldb/iterator"
)

const (
	Version            = 0
	ProtocolLength     = uint64(8)
	ProtocolMaxMsgSize = 10 * 1024 * 1024
	NetworkId          = 322
)

// bzz protocol message codes
const (
	statusMsg          = iota // 0x01
	storeRequestMsg           // 0x02
	retrieveRequestMsg        // 0x03
	peersMsg                  // 0x04
)

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrVersionMismatch
	ErrNetworkIdMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
)

var errorToString = map[int]string{
	ErrMsgTooLarge:       "Message too long",
	ErrDecode:            "Invalid message",
	ErrInvalidMsgCode:    "Invalid message code",
	ErrVersionMismatch:   "Protocol version mismatch",
	ErrNetworkIdMismatch: "NetworkId mismatch",
	ErrNoStatusMsg:       "No status message",
	ErrExtraStatusMsg:    "Extra status message",
}

// bzzProtocol represents the swarm wire protocol
// an instance is running on each peer
type bzzProtocol struct {
	netStore   *netStore
	peer       *p2p.Peer
	remoteAddr *peerAddr
	key        Key
	rw         p2p.MsgReadWriter
	errors     *errs.Errors
	requestDb  *LDBDatabase
	quitC      chan bool
}

/*
 Handshake

 [0x01, Version: B_8, ID: B, Addr: [NodeID: B_64, IP: B_4 or B_6, Port: P], NetworkID; B_8, Caps: [[cap1: B_3, capVersion1: P], [cap2: B_3, capVersion2: P], ...]]

* Version: 8 byte integer version of the protocol
* ID: arbitrary byte sequence client identifier human readable
* Addr: the address advertised by the node, format identical to DEVp2p wire protocol
* NetworkID: 8 byte integer network identifier
* Caps: swarm-specific capabilities, format identical to devp2p

*/
type statusMsgData struct {
	Version   uint64
	ID        string
	Addr      *peerAddr
	NetworkId uint64
	Caps      []p2p.Cap
	// Strategy  uint64
}

func (self *statusMsgData) String() string {
	return fmt.Sprintf("Status: Version: %v, ID: %v, Addr: %v, NetworkId: %v, Caps: %v", self.Version, self.ID, self.Addr, self.NetworkId, self.Caps)
}

/*
 Given the chunker I see absolutely no reason why not allow storage and delivery
 of larger data . See my discussion on flexible chunking.
 store requests are forwarded to the peers in their kademlia proximity bin
 if they are distant
 if they are within our storage radius or have any incentive to store it
 then attach your nodeID to the metadata
 if the storage request is sufficiently close (within our proxLimit, i. e., the
 last row of the routing table), then sending it to all peers will not guarantee convergence, so there needs to be an absolute expiry of the request too.
 Maybe the protocol should specify a forward probability exponentially
 declining with age.

Store request

[+0x02, key: B_32, metadata: [], data: B_4k]: the data chunk to be stored, preceded by its key.


*/
type storeRequestMsgData struct {
	Key   Key    // hash of datasize | data
	SData []byte // the actual chunk Data
	// optional
	Id             uint64     // request ID. if delivery, the ID is retrieve request ID
	requestTimeout *time.Time // expiry for forwarding - [not serialised][not currently used]
	storageTimeout *time.Time // expiry of content - [not serialised][not currently used]
	Metadata       metaData   // routing and accounting metadata [not currently used]
	peer           *peer      // [not serialised] protocol registers the requester
}

func (self storeRequestMsgData) String() string {
	var from string
	if self.peer == nil {
		from = "self"
	} else {
		from = self.peer.Addr().String()
	}
	return fmt.Sprintf("From: %v, Key: %x; ID: %v, requestTimeout: %v, storageTimeout: %v, SData %x", from, self.Key[:4], self.Id, self.requestTimeout, self.storageTimeout, self.SData[:10])
}

/*
Root key retrieve request
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

Retrieve request

[0x03, key: B_32, Id: B_8, MaxSize: B_8, MaxPeers: B_8, Timeout: B_8, metadata: B]: key of the data chunk to be retrieved, timeout in milliseconds. Note that zero timeout retrievals serve also as messages to retrieve peers.

*/
type retrieveRequestMsgData struct {
	Key Key
	// optional
	Id       uint64     // request id, request is a lookup if missing or zero
	MaxSize  uint64     // maximum size of delivery accepted
	MaxPeers uint64     // maximum number of peers returned
	Timeout  uint64     // the longest time we are expecting a response
	timeout  *time.Time // [not serialised]
	peer     *peer      // [not serialised] protocol registers the requester
}

func (self retrieveRequestMsgData) String() string {
	var from string
	if self.peer == nil {
		from = "ourselves"
	} else {
		from = self.peer.Addr().String()
	}
	var target []byte
	if len(self.Key) > 3 {
		target = self.Key[:4]
	}
	return fmt.Sprintf("From: %v, Key: %x; ID: %v, MaxSize: %v, MaxPeers: %d", from, target, self.Id, self.MaxSize, self.MaxPeers)
}

// lookups are encoded by missing request ID
func (self retrieveRequestMsgData) isLookup() bool {
	return self.Id == 0
}

func isZeroKey(key Key) bool {
	return len(key) == 0 || bytes.Equal(key, zeroKey)
}

func (self retrieveRequestMsgData) setTimeout(t *time.Time) {
	self.timeout = t
	if t != nil {
		self.Timeout = uint64(t.UnixNano())
	} else {
		self.Timeout = 0
	}
}

func (self retrieveRequestMsgData) getTimeout() (t *time.Time) {
	if self.Timeout > 0 && self.timeout == nil {
		timeout := time.Unix(int64(self.Timeout), 0)
		t = &timeout
		self.timeout = t
	}
	return
}

// peerAddr is sent in StatusMsg as part of the handshake
type peerAddr struct {
	IP    net.IP
	Port  uint16
	ID    []byte      // the 64 byte NodeID (ECDSA Public Key)
	hash  common.Hash // [not serialised] Sha3 hash of NodeID
	enode string      // [not serialised] the enode URL of the peers Address
}

func (self peerAddr) String() string {
	return self.new().enode
}

func (self *peerAddr) new() *peerAddr {
	self.hash = crypto.Sha3Hash(self.ID)
	self.enode = fmt.Sprintf("enode://%x@%v:%d", self.ID, self.IP, self.Port)
	return self
}

/*
peers Msg is one response to retrieval; it is always encouraged after a retrieval
request to respond with a list of peers in the same kademlia proximity bin.
The encoding of a peer is identical to that in the devp2p base protocol peers
messages: [IP, Port, NodeID]
note that a node's DPA address is not the NodeID but the hash of the NodeID.

Timeout serves to indicate whether the responder is forwarding the query within
the timeout or not.

The Key is the target (if response to a retrieval request) or missing (zero value)
peers address (hash of NodeID) if retrieval request was a self lookup.

Peers message is requested by retrieval requests with a missing or zero value request ID

[0x04, Key: B_32, peers: [[IP, Port, NodeID], [IP, Port, NodeID], .... ], Timeout: B_8, Id: B_8 ]

*/
type peersMsgData struct {
	Peers   []*peerAddr //
	Timeout uint64
	timeout *time.Time // indicate whether responder is expected to deliver content
	Key     Key        // present if a response to a retrieval request
	Id      uint64     // present if a response to a retrieval request
	//
	peer *peer
}

func (self peersMsgData) String() string {
	var from string
	if self.peer == nil {
		from = "ourselves"
	} else {
		from = self.peer.Addr().String()
	}
	var target []byte
	if len(self.Key) > 3 {
		target = self.Key[:4]
	}
	return fmt.Sprintf("From: %v, Key: %x; ID: %v, Peers: %v", from, target, self.Id, self.Peers)
}

func (self peersMsgData) setTimeout(t *time.Time) {
	self.timeout = t
	if t != nil {
		self.Timeout = uint64(t.UnixNano())
	} else {
		self.Timeout = 0
	}
}

func (self peersMsgData) getTimeout() (t *time.Time) {
	if self.Timeout > 0 && self.timeout == nil {
		timeout := time.Unix(int64(self.Timeout), 0)
		t = &timeout
		self.timeout = t
	}
	return
}

/*
metadata is as yet a placeholder
it will likely contain info about hops or the entire forward chain of node IDs
this may allow some interesting schemes to evolve optimal routing strategies
metadata for storage and retrieval requests could specify format parameters
relevant for the (blockhashing) chunking scheme used (for chunks corresponding
to a treenode). For instance all runtime params for the chunker (hashing
algorithm used, branching etc.)
Finally metadata can hold accounting info relevant to incentivisation scheme
*/
type metaData struct{}

/*
main entrypoint, wrappers starting a server running the bzz protocol
use this constructor to attach the protocol ("class") to server caps
the Dev p2p layer then runs the protocol instance on each peer
*/
func BzzProtocol(netstore *netStore) (p2p.Protocol, error) {

	db, err := NewLDBDatabase(path.Join(netstore.path, "requests"))
	if err != nil {
		return p2p.Protocol{}, err
	}
	return p2p.Protocol{
		Name:    "bzz",
		Version: Version,
		Length:  ProtocolLength,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			return runBzzProtocol(db, netstore, p, rw)
		},
	}, nil
}

// the main loop that handles incoming messages
// note RemovePeer in the post-disconnect hook
func runBzzProtocol(db *LDBDatabase, netstore *netStore, p *p2p.Peer, rw p2p.MsgReadWriter) (err error) {
	self := &bzzProtocol{
		netStore: netstore,
		rw:       rw,
		peer:     p,
		errors: &errs.Errors{
			Package: "BZZ",
			Errors:  errorToString,
		},
		requestDb: db,
		quitC:     make(chan bool),
	}
	glog.V(logger.Debug).Infof("[BZZ] listening address: %v", self.netStore.addr())

	go self.storeRequestLoop()

	err = self.handleStatus()
	if err == nil {
		for {
			err = self.handle()
			if err != nil {
				// if the handler loop exits, the peer is disconnecting
				// deregister the peer in the hive
				self.netStore.hive.removePeer(peer{bzzProtocol: self})
				break
			}
		}
		close(self.quitC)
	}
	return
}

func (self *bzzProtocol) handle() error {
	msg, err := self.rw.ReadMsg()
	glog.V(logger.Debug).Infof("[BZZ] Incoming MSG: %v", msg)
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return self.protoError(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// make sure that the payload has been fully consumed
	defer msg.Discard()
	/*
	   statusMsg          = iota // 0x01
	   storeRequestMsg           // 0x02
	   retrieveRequestMsg        // 0x03
	   peersMsg                  // 0x04
	*/

	switch msg.Code {
	case statusMsg:
		// no extra status message allowed. The one needed already handled by
		// handleStatus
		glog.V(logger.Debug).Infof("[BZZ] Status message: %v", msg)
		return self.protoError(ErrExtraStatusMsg, "")

	case storeRequestMsg:
		// store requests are dispatched to netStore
		var req storeRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "msg %v: %v", msg, err)
		}
		req.peer = &peer{bzzProtocol: self}
		self.netStore.addStoreRequest(&req)

	case retrieveRequestMsg:
		// retrieve Requests are dispatched to netStore
		var req retrieveRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		if req.Key == nil {
			return self.protoError(ErrDecode, "protocol handler: req.Key == nil || req.Timeout == nil")
		}
		req.peer = &peer{bzzProtocol: self}
		glog.V(logger.Debug).Infof("[BZZ] Receiving retrieve request: %s", req.String())
		self.netStore.addRetrieveRequest(&req)

	case peersMsg:
		// response to lookups and immediate response to retrieve requests
		// dispatches new peer data to the hive that adds them to KADDB
		var req peersMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		req.peer = &peer{bzzProtocol: self}
		glog.V(logger.Debug).Infof("[BZZ] Receiving peer addresses: %s", req.String())
		self.netStore.hive.addPeerEntries(&req)

	default:
		// no other message is allowed
		return self.protoError(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

func (self *bzzProtocol) handleStatus() (err error) {
	// send precanned status message
	handshake := &statusMsgData{
		Version:   uint64(Version),
		ID:        "honey",
		Addr:      self.netStore.addr(),
		NetworkId: uint64(NetworkId),
		Caps:      []p2p.Cap{},
		// Sync:      self.syncer.lastSynced(),
	}

	if err = p2p.Send(self.rw, statusMsg, handshake); err != nil {
		return err
	}

	// read and handle remote status
	var msg p2p.Msg
	msg, err = self.rw.ReadMsg()
	if err != nil {
		return err
	}

	if msg.Code != statusMsg {
		self.protoError(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, statusMsg)
	}

	if msg.Size > ProtocolMaxMsgSize {
		return self.protoError(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}

	var status statusMsgData
	if err := msg.Decode(&status); err != nil {
		return self.protoError(ErrDecode, "msg %v: %v", msg, err)
	}

	if status.NetworkId != NetworkId {
		return self.protoError(ErrNetworkIdMismatch, "%d (!= %d)", status.NetworkId, NetworkId)
	}

	if Version != status.Version {
		return self.protoError(ErrVersionMismatch, "%d (!= %d)", status.Version, Version)
	}

	self.remoteAddr = status.Addr.new()
	glog.V(logger.Detail).Infof("[BZZ] self: advertised IP: %v, local address: %v\npeer: advertised IP: %v, remote address: %v\n", self.netStore.addr().IP, self.peer.LocalAddr(), status.Addr.IP, self.peer.RemoteAddr())

	glog.V(logger.Info).Infof("[BZZ] Peer %08x is [bzz] capable (%d/%d)\n", self.remoteAddr.hash[:4], status.Version, status.NetworkId)
	self.netStore.hive.addPeer(peer{bzzProtocol: self})

	return nil
}

func (self *bzzProtocol) addrKey() []byte {
	id := self.peer.ID()
	if self.key == nil {
		self.key = Key(crypto.Sha3(id[:]))
	}
	return self.key
}

// protocol instance implements kademlia.Node interface (embedded hive.peer)
func (self *bzzProtocol) Addr() kademlia.Address {
	return kademlia.Address(self.remoteAddr.hash)
}

func (self *bzzProtocol) Url() string {
	return self.remoteAddr.enode
}

// TODO:
func (self *bzzProtocol) LastActive() time.Time {
	return time.Now()
}

// may need to implement protocol drop only? don't want to kick off the peer
// if they are useful for other protocols
func (self *bzzProtocol) Drop() {
	self.peer.Disconnect(p2p.DiscSubprotocolError)
}

func (self *bzzProtocol) String() string {
	return fmt.Sprintf("%08x: %v\n", self.remoteAddr.hash.Bytes()[:4], self.Url())
}

func (self *bzzProtocol) peerAddr() *peerAddr {
	p := self.peer
	id := p.ID()
	host, port, _ := net.SplitHostPort(p.RemoteAddr().String())
	intport, _ := strconv.Atoi(port)
	return &peerAddr{
		ID:   id[:],
		IP:   net.ParseIP(host),
		Port: uint16(intport),
	}
}

// outgoing messages
func (self *bzzProtocol) retrieve(req *retrieveRequestMsgData) {
	glog.V(logger.Debug).Infof("[BZZ] Sending retrieve request: %v", req)
	err := p2p.Send(self.rw, retrieveRequestMsg, req)
	if err != nil {
		glog.V(logger.Error).Infof("[BZZ] EncodeMsg error: %v", err)
	}
}

// storeRequestLoop is buffering store requests to be sent over to the peer
// this is to prevent crashes due to network output buffer contention (???)
// the messages are supposed to be sent in the p2p priority queue.
// TODO: as soon as there is API for that feature, adjust.
// TODO: when peer drops the iterator position is not persisted
// the request DB is shared between peers, keys are prefixed by the peers address
// and the iterator
func (self *bzzProtocol) storeRequestLoop() {

	start := make([]byte, 64)
	copy(start, self.addrKey())

	key := make([]byte, 64)
	copy(key, start)
	var n int
	var it iterator.Iterator
LOOP:
	for {
		if n == 0 {
			it = self.requestDb.NewIterator()
			// glog.V(logger.Debug).Infof("[BZZ] seek iterator: %x", key)
			it.Seek(key)
			if !it.Valid() {
				// glog.V(logger.Debug).Infof("[BZZ] not valid, sleep, continue: %x", key)
				time.Sleep(1 * time.Second)
				continue
			}
			key = it.Key()
			// glog.V(logger.Debug).Infof("[BZZ] found db key: %x", key)
			n = 100
		}
		// glog.V(logger.Debug).Infof("[BZZ] checking key: %x <> %x ", key, self.key())

		// reached the end of this peers range
		if !bytes.Equal(key[:32], self.addrKey()) {
			// glog.V(logger.Debug).Infof("[BZZ] reached the end of this peers range: %x", key)
			n = 0
			continue
		}

		chunk, err := self.netStore.localStore.dbStore.Get(key[32:])
		if err != nil {
			self.requestDb.Delete(key)
			continue
		}
		// glog.V(logger.Debug).Infof("[BZZ] sending chunk: %x", chunk.Key)

		id := generateId()
		req := &storeRequestMsgData{
			Key:   chunk.Key,
			SData: chunk.SData,
			Id:    uint64(id),
		}
		self.store(req)

		n--
		self.requestDb.Delete(key)
		it.Next()
		key = it.Key()
		if len(key) == 0 {
			key = start
			if n == 0 {
				time.Sleep(1 * time.Second)
			}
			n = 0
		}

		select {
		case <-self.quitC:
			break LOOP
		default:
		}
	}
}

func (self *bzzProtocol) store(req *storeRequestMsgData) {
	p2p.Send(self.rw, storeRequestMsg, req)
}

func (self *bzzProtocol) storeRequest(key Key) {
	peerKey := make([]byte, 64)
	copy(peerKey, self.addrKey())
	copy(peerKey[32:], key[:])
	glog.V(logger.Debug).Infof("[BZZ] enter store request %x into db", peerKey)
	self.requestDb.Put(peerKey, []byte{0})
}

func (self *bzzProtocol) peers(req *peersMsgData) {
	p2p.Send(self.rw, peersMsg, req)
}

func (self *bzzProtocol) protoError(code int, format string, params ...interface{}) (err *errs.Error) {
	err = self.errors.New(code, format, params...)
	err.Log(glog.V(logger.Info))
	return
}

func (self *bzzProtocol) protoErrorDisconnect(err *errs.Error) {
	err.Log(glog.V(logger.Info))
	if err.Fatal() {
		self.peer.Disconnect(p2p.DiscSubprotocolError)
	}
}
