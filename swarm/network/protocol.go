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

/*
bzz implements the swarm wire protocol [bzz] (sister of eth and shh)
the protocol instance is launched on each peer by the network layer if the
bzz protocol handler is registered on the p2p server.

The bzz protocol component speaks the bzz protocol
* handle the protocol handshake
* register peers in the KΛÐΞMLIΛ table via the hive logistic manager
* dispatch to hive for handling the DHT logic
* encode and decode requests for storage and retrieval
* handle sync protocol messages via the syncer
* talks the SWAP payment protocol (swap accounting is done within NetStore)
*/

import (
	"errors"
	"fmt"
	"net"
	"strconv"
	"time"

	"github.com/ethereum/go-ethereum/contracts/chequebook"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	bzzswap "github.com/ethereum/go-ethereum/swarm/services/swap"
	"github.com/ethereum/go-ethereum/swarm/services/swap/swap"
	"github.com/ethereum/go-ethereum/swarm/storage"
)

const (
	Version            = 0
	ProtocolLength     = uint64(8)
	ProtocolMaxMsgSize = 10 * 1024 * 1024
	NetworkId          = 3
)

// bzz represents the swarm wire protocol
// an instance is running on each peer
type bzz struct {
	storage    StorageHandler       // handler storage/retrieval related requests coming via the bzz wire protocol
	hive       *Hive                // the logistic manager, peerPool, routing service and peer handler
	dbAccess   *DbAccess            // access to db storage counter and iterator for syncing
	requestDb  *storage.LDBDatabase // db to persist backlog of deliveries to aid syncing
	remoteAddr *peerAddr            // remote peers address
	peer       *p2p.Peer            // the p2p peer object
	rw         p2p.MsgReadWriter    // messageReadWriter to send messages to
	backend    chequebook.Backend
	lastActive time.Time
	NetworkId  uint64

	swap        *swap.Swap          // swap instance for the peer connection
	swapParams  *bzzswap.SwapParams // swap settings both local and remote
	swapEnabled bool                // flag to enable SWAP (will be set via Caps in handshake)
	syncEnabled bool                // flag to enable SYNC (will be set via Caps in handshake)
	syncer      *syncer             // syncer instance for the peer connection
	syncParams  *SyncParams         // syncer params
	syncState   *syncState          // outgoing syncronisation state (contains reference to remote peers db counter)
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
func Bzz(cloud StorageHandler, backend chequebook.Backend, hive *Hive, dbaccess *DbAccess, sp *bzzswap.SwapParams, sy *SyncParams, networkId uint64) (p2p.Protocol, error) {

	// a single global request db is created for all peer connections
	// this is to persist delivery backlog and aid syncronisation
	requestDb, err := storage.NewLDBDatabase(sy.RequestDbPath)
	if err != nil {
		return p2p.Protocol{}, fmt.Errorf("error setting up request db: %v", err)
	}
	if networkId == 0 {
		networkId = NetworkId
	}
	return p2p.Protocol{
		Name:    "bzz",
		Version: Version,
		Length:  ProtocolLength,
		Run: func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
			return run(requestDb, cloud, backend, hive, dbaccess, sp, sy, networkId, p, rw)
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
func run(requestDb *storage.LDBDatabase, depo StorageHandler, backend chequebook.Backend, hive *Hive, dbaccess *DbAccess, sp *bzzswap.SwapParams, sy *SyncParams, networkId uint64, p *p2p.Peer, rw p2p.MsgReadWriter) (err error) {

	self := &bzz{
		storage:     depo,
		backend:     backend,
		hive:        hive,
		dbAccess:    dbaccess,
		requestDb:   requestDb,
		peer:        p,
		rw:          rw,
		swapParams:  sp,
		syncParams:  sy,
		swapEnabled: hive.swapEnabled,
		syncEnabled: true,
		NetworkId:   networkId,
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
		if self.hive.blockRead {
			log.Warn(fmt.Sprintf("Cannot read network"))
			time.Sleep(100 * time.Millisecond)
			continue
		}
		err = self.handle()
		if err != nil {
			return
		}
	}
}

// TODO: may need to implement protocol drop only? don't want to kick off the peer
// if they are useful for other protocols
func (self *bzz) Drop() {
	self.peer.Disconnect(p2p.DiscSubprotocolError)
}

// one cycle of the main forever loop that handles and dispatches incoming messages
func (self *bzz) handle() error {
	msg, err := self.rw.ReadMsg()
	log.Debug(fmt.Sprintf("<- %v", msg))
	if err != nil {
		return err
	}
	if msg.Size > ProtocolMaxMsgSize {
		return fmt.Errorf("message too long: %v > %v", msg.Size, ProtocolMaxMsgSize)
	}
	// make sure that the payload has been fully consumed
	defer msg.Discard()

	switch msg.Code {

	case statusMsg:
		// no extra status message allowed. The one needed already handled by
		// handleStatus
		log.Debug(fmt.Sprintf("Status message: %v", msg))
		return errors.New("extra status message")

	case storeRequestMsg:
		// store requests are dispatched to netStore
		var req storeRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return fmt.Errorf("<- %v: %v", msg, err)
		}
		if n := len(req.SData); n < 9 {
			return fmt.Errorf("<- %v: Data too short (%v)", msg, n)
		}
		// last Active time is set only when receiving chunks
		self.lastActive = time.Now()
		log.Trace(fmt.Sprintf("incoming store request: %s", req.String()))
		// swap accounting is done within forwarding
		self.storage.HandleStoreRequestMsg(&req, &peer{bzz: self})

	case retrieveRequestMsg:
		// retrieve Requests are dispatched to netStore
		var req retrieveRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return fmt.Errorf("<- %v: %v", msg, err)
		}
		req.from = &peer{bzz: self}
		// if request is lookup and not to be delivered
		if req.isLookup() {
			log.Trace(fmt.Sprintf("self lookup for %v: responding with peers only...", req.from))
		} else if req.Key == nil {
			return fmt.Errorf("protocol handler: req.Key == nil || req.Timeout == nil")
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
			return fmt.Errorf("<- %v: %v", msg, err)
		}
		req.from = &peer{bzz: self}
		log.Trace(fmt.Sprintf("<- peer addresses: %v", req))
		self.hive.HandlePeersMsg(&req, &peer{bzz: self})

	case syncRequestMsg:
		var req syncRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return fmt.Errorf("<- %v: %v", msg, err)
		}
		log.Debug(fmt.Sprintf("<- sync request: %v", req))
		self.lastActive = time.Now()
		self.sync(req.SyncState)

	case unsyncedKeysMsg:
		// coming from parent node offering
		var req unsyncedKeysMsgData
		if err := msg.Decode(&req); err != nil {
			return fmt.Errorf("<- %v: %v", msg, err)
		}
		log.Debug(fmt.Sprintf("<- unsynced keys : %s", req.String()))
		err := self.storage.HandleUnsyncedKeysMsg(&req, &peer{bzz: self})
		self.lastActive = time.Now()
		if err != nil {
			return fmt.Errorf("<- %v: %v", msg, err)
		}

	case deliveryRequestMsg:
		// response to syncKeysMsg hashes filtered not existing in db
		// also relays the last synced state to the source
		var req deliveryRequestMsgData
		if err := msg.Decode(&req); err != nil {
			return fmt.Errorf("<-msg %v: %v", msg, err)
		}
		log.Debug(fmt.Sprintf("<- delivery request: %s", req.String()))
		err := self.storage.HandleDeliveryRequestMsg(&req, &peer{bzz: self})
		self.lastActive = time.Now()
		if err != nil {
			return fmt.Errorf("<- %v: %v", msg, err)
		}

	case paymentMsg:
		// swap protocol message for payment, Units paid for, Cheque paid with
		if self.swapEnabled {
			var req paymentMsgData
			if err := msg.Decode(&req); err != nil {
				return fmt.Errorf("<- %v: %v", msg, err)
			}
			log.Debug(fmt.Sprintf("<- payment: %s", req.String()))
			self.swap.Receive(int(req.Units), req.Promise)
		}

	default:
		// no other message is allowed
		return fmt.Errorf("invalid message code: %v", msg.Code)
	}
	return nil
}

func (self *bzz) handleStatus() (err error) {

	handshake := &statusMsgData{
		Version:   uint64(Version),
		ID:        "honey",
		Addr:      self.selfAddr(),
		NetworkId: self.NetworkId,
		Swap: &bzzswap.SwapProfile{
			Profile:    self.swapParams.Profile,
			PayProfile: self.swapParams.PayProfile,
		},
	}

	err = p2p.Send(self.rw, statusMsg, handshake)
	if err != nil {
		return err
	}

	// read and handle remote status
	var msg p2p.Msg
	msg, err = self.rw.ReadMsg()
	if err != nil {
		return err
	}

	if msg.Code != statusMsg {
		return fmt.Errorf("first msg has code %x (!= %x)", msg.Code, statusMsg)
	}

	if msg.Size > ProtocolMaxMsgSize {
		return fmt.Errorf("message too long: %v > %v", msg.Size, ProtocolMaxMsgSize)
	}

	var status statusMsgData
	if err := msg.Decode(&status); err != nil {
		return fmt.Errorf("<- %v: %v", msg, err)
	}

	if status.NetworkId != self.NetworkId {
		return fmt.Errorf("network id mismatch: %d (!= %d)", status.NetworkId, self.NetworkId)
	}

	if Version != status.Version {
		return fmt.Errorf("protocol version mismatch: %d (!= %d)", status.Version, Version)
	}

	self.remoteAddr = self.peerAddr(status.Addr)
	log.Trace(fmt.Sprintf("self: advertised IP: %v, peer advertised: %v, local address: %v\npeer: advertised IP: %v, remote address: %v\n", self.selfAddr(), self.remoteAddr, self.peer.LocalAddr(), status.Addr.IP, self.peer.RemoteAddr()))

	if self.swapEnabled {
		// set remote profile for accounting
		self.swap, err = bzzswap.NewSwap(self.swapParams, status.Swap, self.backend, self)
		if err != nil {
			return err
		}
	}

	log.Info(fmt.Sprintf("Peer %08x is capable (%d/%d)", self.remoteAddr.Addr[:4], status.Version, status.NetworkId))
	err = self.hive.addPeer(&peer{bzz: self})
	if err != nil {
		return err
	}

	// hive sets syncstate so sync should start after node added
	log.Info(fmt.Sprintf("syncronisation request sent with %v", self.syncState))
	self.syncRequest()

	return nil
}

func (self *bzz) sync(state *syncState) error {
	// syncer setup
	if self.syncer != nil {
		return errors.New("sync request can only be sent once")
	}

	cnt := self.dbAccess.counter()
	remoteaddr := self.remoteAddr.Addr
	start, stop := self.hive.kad.KeyRange(remoteaddr)

	// an explicitly received nil syncstate disables syncronisation
	if state == nil {
		self.syncEnabled = false
		log.Warn(fmt.Sprintf("syncronisation disabled for peer %v", self))
		state = &syncState{DbSyncState: &storage.DbSyncState{}, Synced: true}
	} else {
		state.synced = make(chan bool)
		state.SessionAt = cnt
		if storage.IsZeroKey(state.Stop) && state.Synced {
			state.Start = storage.Key(start[:])
			state.Stop = storage.Key(stop[:])
		}
		log.Debug(fmt.Sprintf("syncronisation requested by peer %v at state %v", self, state))
	}
	var err error
	self.syncer, err = newSyncer(
		self.requestDb,
		storage.Key(remoteaddr[:]),
		self.dbAccess,
		self.unsyncedKeys, self.store,
		self.syncParams, state, func() bool { return self.syncEnabled },
	)
	if err != nil {
		return nil
	}
	log.Trace(fmt.Sprintf("syncer set for peer %v", self))
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
	req := &syncRequestMsgData{}
	if self.hive.syncEnabled {
		log.Debug(fmt.Sprintf("syncronisation request to peer %v at state %v", self, self.syncState))
		req.SyncState = self.syncState
	}
	if self.syncState == nil {
		log.Warn(fmt.Sprintf("syncronisation disabled for peer %v at state %v", self, self.syncState))
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

func (self *bzz) send(msg uint64, data interface{}) error {
	if self.hive.blockWrite {
		return fmt.Errorf("network write blocked")
	}
	log.Trace(fmt.Sprintf("-> %v: %v (%T) to %v", msg, data, data, self))
	err := p2p.Send(self.rw, msg, data)
	if err != nil {
		self.Drop()
	}
	return err
}
