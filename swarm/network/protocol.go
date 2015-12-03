package network

/*
BZZ implements the bzz wire protocol of swarm
the protocol instance is launched on each peer by the network layer if the
BZZ protocol handler is registered on the p2p server.

The protocol takes care of actually communicating the bzz protocol
* encoding and decoding requests for storage and retrieval
* handling the s§protocol handshake
* dispaching to netstore for handling the DHT logic
* registering peers in the KΛÐΞMLIΛ table via the hive logistic manager
* handling sync protocol messages via the syncer
* talks the SWAP payent protocol (swap accounting is done within NetStore)
*/

import (
	"fmt"
	"net"
	"strconv"

	bzzswap "github.com/ethereum/go-ethereum/swarm/services/swap"
	"github.com/ethereum/go-ethereum/swarm/storage"
	"github.com/ethereum/go-ethereum/common/chequebook"
	"github.com/ethereum/go-ethereum/common/swap"
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
	NetworkId          = 322
)

const (
	ErrMsgTooLarge = iota
	ErrDecode
	ErrInvalidMsgCode
	ErrVersionMismatch
	ErrNetworkIdMismatch
	ErrNoStatusMsg
	ErrExtraStatusMsg
	ErrSwap
	ErrSync
)

var errorToString = map[int]string{
	ErrMsgTooLarge:       "Message too long",
	ErrDecode:            "Invalid message",
	ErrInvalidMsgCode:    "Invalid message code",
	ErrVersionMismatch:   "Protocol version mismatch",
	ErrNetworkIdMismatch: "NetworkId mismatch",
	ErrNoStatusMsg:       "No status message",
	ErrExtraStatusMsg:    "Extra status message",
	ErrSwap:              "SWAP error",
	ErrSync:              "Sync error",
}

// bzz represents the swarm wire protocol
// an instance is running on each peer
type bzz struct {
	selfID     discover.NodeID      // peer's node id used in peer advertising in handshake
	key        storage.Key          // baseaddress as storage.Key
	storage    StorageHandler       // handler storage/retrieval related requests coming via the bzz wire protocol
	hive       *Hive                // the logistic manager, peerPool, routing servicec and peer handler
	dbAccess   *DbAccess            // access to db storage counter and iterator for syncing
	requestDb  *storage.LDBDatabase // db to persist backlog of deliveries to aid syncing
	remoteAddr *peerAddr            // remote peers address
	peer       *p2p.Peer            // the p2p peer object
	rw         p2p.MsgReadWriter    // messageReadWriter to send messages to
	errors     *errs.Errors         // errors table

	swap        *swap.Swap          // swap instance for the peer connection
	swapParams  *bzzswap.SwapParams // swap settings both local and remote
	swapEnabled bool                // flag to switch off SWAP (will be via Caps in handshake)
	syncer      *syncer             // syncer instance for the peer connection
	syncParams  *SyncParams         // syncer params
	syncState   *syncState          // outgoing syncronisation state (contains reference to remote peers db counter)
	syncEnabled bool                // flag to enable syncing
}

// interface type for handler of storage/retrieval related requests coming
// via the bzz wire protocol
// messages: UnsyncedKeys, DeliveryRequest, StoreRequest, RetrieveRequest
type StorageHandler interface {
	HandleUnsyncedKeysMsg(req *unsyncedKeysMsgData, p *peer) error
	HandleDeliveryRequestMsg(req *deliveryRequestMsgData, p *peer) error
	HandleStoreRequestMsg(req *storeRequestMsgData, p *peer)
	HandleRetrieveRequestMsg(req *retrieveRequestMsgData, p *peer)
}

/*
main entrypoint, wrappers starting a server that will run the bzz protocol
use this constructor to attach the protocol ("class") to server caps
This is done by node.Node#Register(func(node.ServiceContext) (Service, error))
Service implements Protocols() which is an array of protocol constructors
at node startup the protocols are initialised
the Dev p2p layer then calls Run(p *p2p.Peer, rw p2p.MsgReadWriter) error
on each peer connection
The Run function of the Bzz protocol class creates a bzz instance
which will represent the peer for the swarm hive and all peer-aware components
*/
func Bzz(cloud StorageHandler, hive *Hive, dbaccess *DbAccess, sp *bzzswap.SwapParams, sy *SyncParams) (p2p.Protocol, error) {

	// a single global request db is created for all peer connections
	// this is to persist delivery backlog and aid syncronisation
	requestDb, err := storage.NewLDBDatabase(sy.RequestDbPath)
	if err != nil {
		return p2p.Protocol{}, fmt.Errorf("error setting up request db: %v", err)
	}

	return p2p.Protocol{
		Name:    "bzz",
		Version: Version,
		Length:  ProtocolLength,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			return run(requestDb, cloud, hive, dbaccess, sp, sy, p, rw)
		},
	}, nil
}

/*
the main protocol loop that
 * does the handshake by exchanging statusMsg
 * if peer is valid and accepted, registers with the hive
 * then enters into a forever loop handling incoming messages
 * storage and retrieval related queries coming via bzz are dispatched to StorageHandler
 * peer-related messages are dispatched to the hive
 * payment related messages are relayed to SWAP service
 * on disconnect, unregister the peer in the hive (note RemovePeer in the post-disconnect hook)
 * whenever the loop terminates, the peer will disconnect with Subprotocol error
 * whenever handlers return an error the loop terminates
*/
func run(requestDb *storage.LDBDatabase, depo StorageHandler, hive *Hive, dbaccess *DbAccess, sp *bzzswap.SwapParams, sy *SyncParams, p *p2p.Peer, rw p2p.MsgReadWriter) (err error) {

	self := &bzz{
		storage:   depo,
		hive:      hive,
		dbAccess:  dbaccess,
		requestDb: requestDb,
		peer:      p,
		rw:        rw,
		errors: &errs.Errors{
			Package: "BZZ",
			Errors:  errorToString,
		},
		swapParams:  sp,
		syncParams:  sy,
		swapEnabled: true,
		syncEnabled: true,
	}

	// handle handshake
	err = self.handleStatus()
	if err != nil {
		return err
	}
	defer func() {
		// if the handler loop exits, the peer is disconnecting
		// deregister the peer in the hive
		self.hive.removePeer(&peer{bzz: self})
		if self.syncer != nil {
			self.syncer.stop() // quits request db and delivery loops, save requests
		}
		if self.swap != nil {
			self.swap.Stop() // quits chequebox autocash etc
		}
	}()

	// the main forever loop that handles incoming requests
	for {
		err = self.handle()
		if err != nil {
			return
		}
	}
	return
}

// may need to implement protocol drop only? don't want to kick off the peer
// if they are useful for other protocols
func (self *bzz) Drop() {
	self.peer.Disconnect(p2p.DiscSubprotocolError)
}

// one cycle of the main forever loop that handles and dispatches incoming messages
func (self *bzz) handle() error {
	msg, err := self.rw.ReadMsg()
	glog.V(logger.Debug).Infof("[BZZ] <- %v", msg)
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return self.protoError(ErrMsgTooLarge, "%v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// make sure that the payload has been fully consumed
	defer msg.Discard()

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
		glog.V(logger.Debug).Infof("[BZZ] incoming store request: %s", req.String())
		// swap accounting is done within forwarding
		self.storage.HandleStoreRequestMsg(&req, &peer{bzz: self})

	case retrieveRequestMsg:
		// retrieve Requests are dispatched to netStore
		var req retrieveRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		req.from = &peer{bzz: self}
		// if request is lookup and not to be delivered
		if req.isLookup() {
			glog.V(logger.Detail).Infof("[BZZ] 	self lookup for %v: responding with peers only...", req.from)
		} else if req.Key == nil {
			return self.protoError(ErrDecode, "protocol handler: req.Key == nil || req.Timeout == nil")
		} else {
			// swap accounting is done within netStore
			self.storage.HandleRetrieveRequestMsg(&req, &peer{bzz: self})
		}
		// direct response with peers, TODO: sort this out
		self.hive.peers(&req)

	case peersMsg:
		// response to lookups and immediate response to retrieve requests
		// dispatches new peer data to the hive that adds them to KADDB
		var req peersMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		req.from = &peer{bzz: self}
		glog.V(logger.Debug).Infof("[BZZ] incoming peer addresses: %v", req)
		self.hive.HandlePeersMsg(&req, &peer{bzz: self})

	case syncRequestMsg:
		var req syncRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		glog.V(logger.Debug).Infof("[BZZ] sync request received: %v", req)
		self.sync(req.SyncState)

	case unsyncedKeysMsg:
		// coming from parent node offering
		var req unsyncedKeysMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		glog.V(logger.Debug).Infof("[BZZ] incoming unsynced keys msg: %s", req.String())
		err := self.storage.HandleUnsyncedKeysMsg(&req, &peer{bzz: self})
		if err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		// set peers state to persist
		self.syncState = req.State

	case deliveryRequestMsg:
		// response to syncKeysMsg hashes filtered not existing in db
		// also relays the last synced state to the source
		var req deliveryRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		glog.V(logger.Debug).Infof("[BZZ] incoming delivery request: %s", req.String())
		err := self.storage.HandleDeliveryRequestMsg(&req, &peer{bzz: self})
		if err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}

	case paymentMsg:
		// swap protocol message for payment, Units paid for, Cheque paid with
		var req paymentMsgData
		if err := msg.Decode(&req); err != nil {
			return self.protoError(ErrDecode, "->msg %v: %v", msg, err)
		}
		glog.V(logger.Debug).Infof("[BZZ] incoming payment: %s", req.String())
		self.swap.Receive(int(req.Units), req.Promise)

	default:
		// no other message is allowed
		return self.protoError(ErrInvalidMsgCode, "%v", msg.Code)
	}
	return nil
}

func (self *bzz) handleStatus() (err error) {

	handshake := &statusMsgData{
		Version:   uint64(Version),
		ID:        "honey",
		Addr:      self.selfAddr(),
		NetworkId: uint64(NetworkId),
		Swap: &bzzswap.SwapProfile{
			Profile:    self.swapParams.Profile,
			PayProfile: self.swapParams.PayProfile,
		},
	}

	err = p2p.Send(self.rw, statusMsg, handshake)
	if err != nil {
		self.protoError(ErrNoStatusMsg, err.Error())
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

	self.remoteAddr = self.peerAddr(status.Addr)
	glog.V(logger.Detail).Infof("[BZZ] self: advertised IP: %v, peer advertised: %v, local address: %v\npeer: advertised IP: %v, remote address: %v\n", self.selfAddr(), self.remoteAddr, self.peer.LocalAddr(), status.Addr.IP, self.peer.RemoteAddr())

	if self.swapEnabled {
		// set remote profile for accounting
		self.swap, err = bzzswap.NewSwap(self.swapParams, status.Swap, self)
		if err != nil {
			return self.protoError(ErrSwap, "%v", err)
		}
	}

	glog.V(logger.Info).Infof("[BZZ] Peer %08x is [bzz] capable (%d/%d)", self.remoteAddr.Addr[:4], status.Version, status.NetworkId)
	self.hive.addPeer(&peer{bzz: self})

	// hive sets syncstate so sync should start after node added
	if self.syncEnabled {
		glog.V(logger.Info).Infof("[BZZ] syncronisation request sent with %v", self.syncState)
		self.syncRequest()
	}
	return nil
}

func (self *bzz) sync(state *syncState) error {
	// syncer setup
	if self.syncer != nil {
		return self.protoError(ErrSync, "sync request can only be sent once")
	}

	cnt := self.dbAccess.counter()
	remoteaddr := self.remoteAddr.Addr
	start, stop := self.hive.kad.KeyRange(remoteaddr)
	if state == nil {
		state = newSyncState(start, stop, cnt)
		glog.V(logger.Warn).Infof("[BZZ] peer %08x provided no sync state, setting up full sync: %v\n", remoteaddr[:4], state)
	} else {
		state.synced = make(chan bool)
		state.SessionAt = cnt
		if storage.IsZeroKey(state.Stop) && state.Synced {
			state.Start = storage.Key(start[:])
			state.Stop = storage.Key(stop[:])
		}
	}
	var err error
	self.syncer, err = newSyncer(
		self.requestDb,
		storage.Key(remoteaddr[:]),
		self.dbAccess,
		self.unsyncedKeys, self.store,
		self.syncParams, state,
	)
	if err != nil {
		return self.protoError(ErrSync, "%v", err)
	}
	return nil
}

func (self *bzz) String() string {
	return self.remoteAddr.String()
}

// repair reported address if IP missing
func (self *bzz) peerAddr(base *peerAddr) *peerAddr {
	if base.IP.IsUnspecified() {
		host, _, _ := net.SplitHostPort(self.peer.RemoteAddr().String())
		base.IP = net.ParseIP(host)
	}
	return base
}

// returns self advertised node connection info (listening address w enodes)
// IP will get repaired on the other end if missing
// or resolved via ID by discovery at dialout
func (self *bzz) selfAddr() *peerAddr {
	id := self.hive.id
	host, port, _ := net.SplitHostPort(self.hive.listenAddr())
	intport, _ := strconv.Atoi(port)
	addr := &peerAddr{
		Addr: self.hive.addr,
		ID:   id[:],
		IP:   net.ParseIP(host),
		Port: uint16(intport),
	}
	return addr
}

// outgoing messages
// send retrieveRequestMsg
func (self *bzz) retrieve(req *retrieveRequestMsgData) error {
	return self.send(retrieveRequestMsg, req)
}

// send storeRequestMsg
func (self *bzz) store(req *storeRequestMsgData) error {
	return self.send(storeRequestMsg, req)
}

func (self *bzz) syncRequest() error {
	req := &syncRequestMsgData{
		SyncState: self.syncState,
	}
	return self.send(syncRequestMsg, req)
}

// queue storeRequestMsg in request db
func (self *bzz) deliveryRequest(reqs []*syncRequest) error {
	req := &deliveryRequestMsgData{
		Deliver: reqs,
	}
	return self.send(deliveryRequestMsg, req)
}

// batch of syncRequests to send off
func (self *bzz) unsyncedKeys(reqs []*syncRequest, state *syncState) error {
	req := &unsyncedKeysMsgData{
		Unsynced: reqs,
		State:    state,
	}
	return self.send(unsyncedKeysMsg, req)
}

// send paymentMsg
func (self *bzz) Pay(units int, promise swap.Promise) {
	req := &paymentMsgData{uint(units), promise.(*chequebook.Cheque)}
	self.payment(req)
}

// send paymentMsg
func (self *bzz) payment(req *paymentMsgData) error {
	return self.send(paymentMsg, req)
}

// sends peersMsg
func (self *bzz) peers(req *peersMsgData) error {
	return self.send(peersMsg, req)
}

func (self *bzz) protoError(code int, format string, params ...interface{}) (err *errs.Error) {
	err = self.errors.New(code, format, params...)
	err.Log(glog.V(logger.Info))
	return
}

func (self *bzz) protoErrorDisconnect(err *errs.Error) {
	err.Log(glog.V(logger.Info))
	if err.Fatal() {
		self.peer.Disconnect(p2p.DiscSubprotocolError)
	}
}

func (self *bzz) send(msg uint64, data interface{}) error {
	glog.V(logger.Debug).Infof("[BZZ] -> %v: %v (%T) to %v", msg, data, data, self)
	err := p2p.Send(self.rw, msg, data)
	if err != nil {
		self.Drop()
	}
	return err
}
