package bzz

/*
BZZ implements the bzz wire protocol of swarm
routing decoded storage and retrieval requests
registering peers with the DHT
*/

import (
	"fmt"
	"net"
	"time"

	"github.com/ethereum/go-ethereum/common/kademlia"
	"github.com/ethereum/go-ethereum/errs"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
)

const (
	Version            = 0
	ProtocolLength     = uint64(8)
	ProtocolMaxMsgSize = 10 * 1024 * 1024
	NetworkId          = 0
	strategy           = 0
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
// instance is running on each peer
type bzzProtocol struct {
	node     *discover.Node
	netStore *NetStore
	peer     *p2p.Peer
	rw       p2p.MsgReadWriter
	errors   *errs.Errors
}

/*
 message structs used for rlp decoding
Handshake

[0x01, Version: B_32, strategy: B_32, capacity: B_64, peers: B_8]

Storing

[+0x02, key: B_256, metadata: [], data: B_4k]: the data chunk to be stored, preceded by its key.

Retrieving

[0x03, key: B_256, timeout: B_64, metadata: []]: key of the data chunk to be retrieved, timeout in milliseconds. Note that zero timeout retrievals serve also as messages to retrieve peers.

Peers

[0x04, key: B_256, timeout: B_64, peers: [[peer], [peer], .... ]] the encoding of a peer is identical to that in the devp2p base protocol peers messages: [IP, Port, NodeID] note that a node's DPA address is not the NodeID but the hash of the NodeID. Timeout serves to indicate whether the responder is forwarding the query within the timeout or not.

*/

type statusMsgData struct {
	Version   uint64
	ID        string
	NodeID    []byte
	Addr      *peerAddr
	NetworkId uint64
	Caps      []p2p.Cap
	// Strategy  uint64
}

func (self *statusMsgData) String() string {
	return fmt.Sptintf("Status: Version: %v, ID: %v, NodeID: %v, Addr: %v, NetworkId: %v, Caps: %v", self.Version, self.ID, self.NodeID, self.Addr, self.NetworkId, self.Caps)
}

/*
 Given the chunker I see absolutely no reason why not allow storage and delivery of larger data . See my discussion on flexible chunking.
 store requests are forwarded to the peers in their cademlia proximity bin if they are distant
 if they are within our storage radius or have any incentive to store it then attach your nodeID to the metadata
 if the storage request is sufficiently close (within our proximity range (the last row of the routing table), then sending it to all peers will not guarantee convergence, so there needs to be an absolute expiry of the request too. Maybe the protocol should specify a forward probability exponentially declining with age.
*/
type storeRequestMsgData struct {
	Key   Key    // hash of datasize | data
	SData []byte // is this needed?
	// optional
	Id             uint64     //
	requestTimeout *time.Time // expiry for forwarding
	storageTimeout *time.Time // expiry of content
	Metadata       metaData   //
	//
	peer peer
}

func (self *storeRequestMsgData) String() string {
	return fmt.Sprintf("From: %v, Key: %x; ID: %v, requestTimeout: %v, storageTimeout: %v, SData %x", self.peer.Addr(), self.Key[:4], self.Id, self.requestTimeout, self.storageTimeout, self.SData[:10])
}

/*
Root key retrieve request
Timeout in milliseconds. Note that zero timeout retrieval requests do not request forwarding, but prompt for a peers message response. therefore they also serve also as messages to retrieve peers.
MaxSize specifies the maximum size that the peer will accept. This is useful in particular if we allow storage and delivery of multichunk payload representing the entire or partial subtree unfolding from the requested root key. So when only interested in limited part of a stream (infinite trees) or only testing chunk availability etc etc, we can indicate it by limiting the size here.
In the special case that the key is identical to the peers own address (hash of NodeID) the message is to be handled as a self lookup. The response is a PeersMsg with the peers in the cademlia proximity bin corresponding to the address.
It is unclear if a retrieval request with an empty target is the same as a self lookup
*/
type retrieveRequestMsgData struct {
	Key Key
	// optional
	Id       uint64     // request id
	MaxSize  uint64     // maximum size of delivery accepted
	MaxPeers uint64     // maximum number of peers returned
	timeout  *time.Time //
	//Metadata metaData  //
	//
	peer peer // protocol registers the requester
}

func (self *retrieveRequestMsgData) String() string {
	return fmt.Sprintf("From: %v, Key: %x; ID: %v, MaxSize: %v, MaxPeers: %v", self.peer.Addr(), self.Key[:4], self.Id, self.MaxSize, self.MaxPeers)
}

type peerAddr struct {
	IP   net.IP
	Port uint16
	ID   []byte
	n    *discover.Node
}

func (self *peerAddr) node() *discover.Node {
	if self.n == nil {
		var nodeid discover.NodeID
		copy(nodeid[:], self.ID)
		self.n = discover.NewNode(nodeid, self.IP, self.Port, self.Port)
	}
	return self.n
}

func (self *peerAddr) addr() kademlia.Address {
	return kademlia.Address(self.node().Sha())
}

func (self *peerAddr) url() string {
	return self.node().String()
}

/*
one response to retrieval, always encouraged after a retrieval request to respond with a list of peers in the same cademlia proximity bin.
The encoding of a peer is identical to that in the devp2p base protocol peers messages: [IP, Port, NodeID]
note that a node's DPA address is not the NodeID but the hash of the NodeID.
Timeout serves to indicate whether the responder is forwarding the query within the timeout or not.
The Key is the target (if response to a retrieval request) or peers address (hash of NodeID) if retrieval request was a self lookup.
It is unclear if PeersMsg with an empty Key has a special meaning or just mean the same as with the peers address as Key (cademlia bin)
*/
type peersMsgData struct {
	Peers   []*peerAddr //
	timeout *time.Time  // indicate whether responder is expected to deliver content
	Key     Key         // if a response to a retrieval request
	Id      uint64      // if a response to a retrieval request
	//
	peer peer
}

/*
metadata is as yet a placeholder
it will likely contain info about hops or the entire forward chain of node IDs
this may allow some interesting schemes to evolve optimal routing strategies
metadata for storage and retrieval requests could specify format parameters relevant for the (blockhashing) chunking scheme used (for chunks corresponding to a treenode). For instance all runtime params for the chunker (hashing algorithm used, branching etc.)
Finally metadata can hold info relevant to some reward or compensation scheme that may be used to incentivise peers.
*/
type metaData struct{}

/*
main entrypoint, wrappers starting a server running the bzz protocol
use this constructor to attach the protocol ("class") to server caps
the Dev p2p layer then runs the protocol instance on each peer
*/
func BzzProtocol(netStore *NetStore) p2p.Protocol {
	return p2p.Protocol{
		Name:    "bzz",
		Version: Version,
		Length:  ProtocolLength,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			return runBzzProtocol(netStore, p, rw)
		},
	}
}

// the main loop that handles incoming messages
// note RemovePeer in the post-disconnect hook
func runBzzProtocol(netStore *NetStore, p *p2p.Peer, rw p2p.MsgReadWriter) (err error) {
	self := &bzzProtocol{
		netStore: netStore,
		rw:       rw,
		peer:     p,
		errors: &errs.Errors{
			Package: "BZZ",
			Errors:  errorToString,
		},
	}
	err = self.handleStatus()
	if err == nil {
		for {
			err = self.handle()
			if err != nil {
				self.netStore.hive.removePeer(peer{bzzProtocol: self})
				break
			}
		}
	}
	return
}

func (self *bzzProtocol) handle() error {
	msg, err := self.rw.ReadMsg()
	dpaLogger.Debugf("Incoming MSG: %v", msg)
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
		dpaLogger.Debugf("Status message: %v", msg)
		return self.protoError(ErrExtraStatusMsg, "")

	case storeRequestMsg:
		var req storeRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "msg %v: %v", msg, err)
		}
		req.peer = peer{bzzProtocol: self}
		self.netStore.addStoreRequest(&req)

	case retrieveRequestMsg:
		var req retrieveRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		dpaLogger.Debugf("Request message: %v", req)
		if req.Key == nil {
			return self.protoError(ErrDecode, "protocol handler: req.Key == nil || req.Timeout == nil")
		}
		req.peer = peer{bzzProtocol: self}
		self.netStore.addRetrieveRequest(&req)

	case peersMsg:
		var req peersMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		req.peer = peer{bzzProtocol: self}
		self.netStore.hive.addPeerEntries(&req)

	default:
		return self.protoError(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

func (self *bzzProtocol) handleStatus() (err error) {
	// send precanned status message
	sliceNodeID := self.netStore.self.ID
	handshake := &statusMsgData{
		Version:   uint64(Version),
		ID:        "honey",
		NodeID:    sliceNodeID[:],
		Addr:      newPeerAddrFromNode(self.netStore.self),
		NetworkId: uint64(NetworkId),
		Caps:      []p2p.Cap{},
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
		return self.protoError(ErrNoStatusMsg, "first msg has code %x (!= %x)", msg.Code, statusMsg)
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

	glog.V(logger.Info).Infof("Peer is [bzz] capable (%d/%d)\n", status.Version, status.NetworkId)

	self.node = status.Addr.node()

	self.netStore.hive.addPeer(peer{bzzProtocol: self})

	return nil
}

// protocol instance implements kademlia.Node interface (embedded hive.peer)
func (self *bzzProtocol) Addr() (a kademlia.Address) {
	return kademlia.Address(self.node.Sha())
}

func (self *bzzProtocol) Url() string {
	return self.node.String()
}

func (self *bzzProtocol) LastActive() time.Time {
	return time.Now()
}

func (self *bzzProtocol) Drop() {
}

func (self *bzzProtocol) String() string {
	return fmt.Sprintf("%4x: %v\n", self.node.Sha().Bytes()[:4], self.Url())
}

func newPeerAddrFromNode(node *discover.Node) *peerAddr {
	return &peerAddr{
		ID:   node.ID[:],
		IP:   node.IP,
		Port: node.TCP,
	}
}

func (self *bzzProtocol) peerAddr() *peerAddr {
	return newPeerAddrFromNode(self.peer.Node())
}

// outgoing messages
func (self *bzzProtocol) retrieve(req *retrieveRequestMsgData) {
	dpaLogger.Debugf("Request message: %v", req)
	err := p2p.Send(self.rw, retrieveRequestMsg, req)
	if err != nil {
		dpaLogger.Errorf("EncodeMsg error: %v", err)
	}
}

func (self *bzzProtocol) store(req *storeRequestMsgData) {
	p2p.Send(self.rw, storeRequestMsg, req)
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
