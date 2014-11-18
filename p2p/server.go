package p2p

import (
	"bytes"
	"fmt"
	"net"
	"sort"
	"strconv"
	"sync"
	"time"

	logpkg "github.com/ethereum/go-ethereum/logger"
)

const (
	outboundAddressPoolSize = 10
	disconnectGracePeriod   = 2
)

type Blacklist interface {
	Get([]byte) (bool, error)
	Put([]byte) error
	Delete([]byte) error
	Exists(pubkey []byte) (ok bool)
}

type BlacklistMap struct {
	blacklist map[string]bool
	lock      sync.RWMutex
}

func NewBlacklist() *BlacklistMap {
	return &BlacklistMap{
		blacklist: make(map[string]bool),
	}
}

func (self *BlacklistMap) Get(pubkey []byte) (bool, error) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	v, ok := self.blacklist[string(pubkey)]
	var err error
	if !ok {
		err = fmt.Errorf("not found")
	}
	return v, err
}

func (self *BlacklistMap) Exists(pubkey []byte) (ok bool) {
	self.lock.RLock()
	defer self.lock.RUnlock()
	_, ok = self.blacklist[string(pubkey)]
	return
}

func (self *BlacklistMap) Put(pubkey []byte) error {
	self.lock.RLock()
	defer self.lock.RUnlock()
	self.blacklist[string(pubkey)] = true
	return nil
}

func (self *BlacklistMap) Delete(pubkey []byte) error {
	self.lock.RLock()
	defer self.lock.RUnlock()
	delete(self.blacklist, string(pubkey))
	return nil
}

type Server struct {
	network   Network
	listening bool //needed?
	dialing   bool //needed?
	closed    bool
	identity  ClientIdentity
	addr      net.Addr
	port      uint16
	protocols []string

	quit      chan chan bool
	peersLock sync.RWMutex

	maxPeers           int
	peers              []*Peer
	peerSlots          chan int
	peersTable         map[string]int
	peerCount          int
	cachedEncodedPeers []byte

	peerConnect    chan net.Addr
	peerDisconnect chan DisconnectRequest
	blacklist      Blacklist
	handlers       Handlers
}

var logger = logpkg.NewLogger("P2P")

func New(network Network, addr net.Addr, identity ClientIdentity, handlers Handlers, maxPeers int, blacklist Blacklist) *Server {
	// get alphabetical list of protocol names from handlers map
	protocols := []string{}
	for protocol := range handlers {
		protocols = append(protocols, protocol)
	}
	sort.Strings(protocols)

	_, port, _ := net.SplitHostPort(addr.String())
	intport, _ := strconv.Atoi(port)

	self := &Server{
		// NewSimpleClientIdentity(clientIdentifier, version, customIdentifier)
		network:   network,
		identity:  identity,
		addr:      addr,
		port:      uint16(intport),
		protocols: protocols,

		quit: make(chan chan bool),

		maxPeers:   maxPeers,
		peers:      make([]*Peer, maxPeers),
		peerSlots:  make(chan int, maxPeers),
		peersTable: make(map[string]int),

		peerConnect:    make(chan net.Addr, outboundAddressPoolSize),
		peerDisconnect: make(chan DisconnectRequest),
		blacklist:      blacklist,

		handlers: handlers,
	}
	for i := 0; i < maxPeers; i++ {
		self.peerSlots <- i // fill up with indexes
	}
	return self
}

func (self *Server) NewAddr(host string, port int) (addr net.Addr, err error) {
	addr, err = self.network.NewAddr(host, port)
	return
}

func (self *Server) ParseAddr(address string) (addr net.Addr, err error) {
	addr, err = self.network.ParseAddr(address)
	return
}

func (self *Server) ClientIdentity() ClientIdentity {
	return self.identity
}

func (self *Server) Peers() (peers []*Peer) {
	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	for _, peer := range self.peers {
		if peer != nil {
			peers = append(peers, peer)
		}
	}
	return
}

func (self *Server) PeerCount() int {
	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	return self.peerCount
}

func (self *Server) PeerConnect(addr net.Addr) {
	// TODO: should buffer, filter and uniq
	// send GetPeersMsg if not blocking
	select {
	case self.peerConnect <- addr: // not enough peers
		self.Broadcast("", getPeersMsg)
	default: // we dont care
	}
}

func (self *Server) PeerDisconnect() chan DisconnectRequest {
	return self.peerDisconnect
}

func (self *Server) Blacklist() Blacklist {
	return self.blacklist
}

func (self *Server) Handlers() Handlers {
	return self.handlers
}

func (self *Server) Broadcast(protocol string, code MsgCode, data ...interface{}) {
	var payload []byte
	if data != nil {
		payload = encodePayload(data...)
	}
	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	for _, peer := range self.peers {
		if peer != nil {
			var msg = Msg{Code: code}
			if data != nil {
				msg.Payload = bytes.NewReader(payload)
				msg.Size = uint32(len(payload))
			}
			peer.messenger.writeProtoMsg(protocol, msg)
		}
	}
}

// Start the server
func (self *Server) Start(listen bool, dial bool) {
	self.network.Start()
	if listen {
		listener, err := self.network.Listener(self.addr)
		if err != nil {
			logger.Warnf("Error initializing listener: %v", err)
			logger.Warnf("Connection listening disabled")
			self.listening = false
		} else {
			self.listening = true
			logger.Infoln("Listen on %v: ready and accepting connections", listener.Addr())
			go self.inboundPeerHandler(listener)
		}
	}
	if dial {
		dialer, err := self.network.Dialer(self.addr)
		if err != nil {
			logger.Warnf("Error initializing dialer: %v", err)
			logger.Warnf("Connection dialout disabled")
			self.dialing = false
		} else {
			self.dialing = true
			logger.Infoln("Dial peers watching outbound address pool")
			go self.outboundPeerHandler(dialer)
		}
	}
	logger.Infoln("server started")
}

func (self *Server) Stop() {
	logger.Infoln("server stopping...")
	// // quit one loop if dialing
	if self.dialing {
		logger.Infoln("stop dialout...")
		dialq := make(chan bool)
		self.quit <- dialq
		<-dialq
		fmt.Println("quit another")
	}
	// quit the other loop if listening
	if self.listening {
		logger.Infoln("stop listening...")
		listenq := make(chan bool)
		self.quit <- listenq
		<-listenq
		fmt.Println("quit one")
	}

	fmt.Println("quit waited")

	logger.Infoln("stopping peers...")
	peers := []net.Addr{}
	self.peersLock.RLock()
	self.closed = true
	for _, peer := range self.peers {
		if peer != nil {
			peers = append(peers, peer.Address)
		}
	}
	self.peersLock.RUnlock()
	for _, address := range peers {
		go self.removePeer(DisconnectRequest{
			addr:   address,
			reason: DiscQuitting,
		})
	}
	// wait till they actually disconnect
	// this is checked by draining the peerSlots (slots are released back if a peer is removed)
	i := 0
	fmt.Println("draining peers")

FOR:
	for {
		select {
		case slot := <-self.peerSlots:
			i++
			fmt.Printf("%v: found slot %v\n", i, slot)
			if i == self.maxPeers {
				break FOR
			}
		}
	}
	logger.Infoln("server stopped")
}

// main loop for adding connections via listening
func (self *Server) inboundPeerHandler(listener net.Listener) {
	for {
		select {
		case slot := <-self.peerSlots:
			go self.connectInboundPeer(listener, slot)
		case errc := <-self.quit:
			listener.Close()
			fmt.Println("quit listenloop")
			errc <- true
			return
		}
	}
}

// main loop for adding outbound peers based on peerConnect address pool
// this same loop handles peer disconnect requests as well
func (self *Server) outboundPeerHandler(dialer Dialer) {
	// addressChan initially set to nil (only watches peerConnect if we need more peers)
	var addressChan chan net.Addr
	slots := self.peerSlots
	var slot *int
	for {
		select {
		case i := <-slots:
			// we need a peer in slot i, slot reserved
			slot = &i
			// now we can watch for candidate peers in the next loop
			addressChan = self.peerConnect
			// do not consume more until candidate peer is found
			slots = nil
		case address := <-addressChan:
			// candidate peer found, will dial out asyncronously
			// if connection fails slot will be released
			go self.connectOutboundPeer(dialer, address, *slot)
			// we can watch if more peers needed in the next loop
			slots = self.peerSlots
			// until then we dont care about candidate peers
			addressChan = nil
		case request := <-self.peerDisconnect:
			go self.removePeer(request)
		case errc := <-self.quit:
			if addressChan != nil && slot != nil {
				self.peerSlots <- *slot
			}
			fmt.Println("quit dialloop")
			errc <- true
			return
		}
	}
}

// check if peer address already connected
func (self *Server) isConnected(address net.Addr) bool {
	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	_, found := self.peersTable[address.String()]
	return found
}

// connect to peer via listener.Accept()
func (self *Server) connectInboundPeer(listener net.Listener, slot int) {
	var address net.Addr
	conn, err := listener.Accept()
	if err != nil {
		logger.Debugln(err)
		self.peerSlots <- slot
		return
	}
	address = conn.RemoteAddr()
	// XXX: this won't work because the remote socket
	// address does not identify the peer. we should
	// probably get rid of this check and rely on public
	// key detection in the base protocol.
	if self.isConnected(address) {
		conn.Close()
		self.peerSlots <- slot
		return
	}
	fmt.Printf("adding %v\n", address)
	go self.addPeer(conn, address, true, slot)
}

// connect to peer via dial out
func (self *Server) connectOutboundPeer(dialer Dialer, address net.Addr, slot int) {
	if self.isConnected(address) {
		return
	}
	conn, err := dialer.Dial(address.Network(), address.String())
	if err != nil {
		self.peerSlots <- slot
		return
	}
	go self.addPeer(conn, address, false, slot)
}

// creates the new peer object and inserts it into its slot
func (self *Server) addPeer(conn net.Conn, address net.Addr, inbound bool, slot int) *Peer {
	self.peersLock.Lock()
	defer self.peersLock.Unlock()
	if self.closed {
		fmt.Println("oopsy, not no longer need peer")
		conn.Close()           //oopsy our bad
		self.peerSlots <- slot // release slot
		return nil
	}
	logger.Infoln("adding new peer", address)
	peer := NewPeer(conn, address, inbound, self)
	self.peers[slot] = peer
	self.peersTable[address.String()] = slot
	self.peerCount++
	self.cachedEncodedPeers = nil
	fmt.Printf("added peer %v %v (slot %v)\n", address, peer, slot)
	peer.Start()
	return peer
}

// removes peer: sending disconnect msg, stop peer, remove rom list/table, release slot
func (self *Server) removePeer(request DisconnectRequest) {
	self.peersLock.Lock()

	address := request.addr
	slot := self.peersTable[address.String()]
	peer := self.peers[slot]
	fmt.Printf("removing peer %v %v (slot %v)\n", address, peer, slot)
	if peer == nil {
		logger.Debugf("already removed peer on %v", address)
		self.peersLock.Unlock()
		return
	}
	// remove from list and index
	self.peerCount--
	self.peers[slot] = nil
	delete(self.peersTable, address.String())
	self.cachedEncodedPeers = nil
	fmt.Printf("removed peer %v (slot %v)\n", peer, slot)
	self.peersLock.Unlock()

	// sending disconnect message
	disconnectMsg := NewMsg(discMsg, request.reason)
	peer.Write("", disconnectMsg)
	// be nice and wait
	time.Sleep(disconnectGracePeriod * time.Second)
	// switch off peer and close connections etc.
	fmt.Println("stopping peer")
	peer.Stop()
	fmt.Println("stopped peer")
	// release slot to signal need for a new peer, last!
	self.peerSlots <- slot
}

// encodedPeerList returns an RLP-encoded list of peers.
// the returned slice will be nil if there are no peers.
func (self *Server) encodedPeerList() []byte {
	// TODO: memoize and reset when peers change
	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	if self.cachedEncodedPeers == nil && self.peerCount > 0 {
		var peerData []interface{}
		for _, i := range self.peersTable {
			peer := self.peers[i]
			peerData = append(peerData, peer.Encode())
		}
		self.cachedEncodedPeers = encodePayload(peerData)
	}
	return self.cachedEncodedPeers
}

// fix handshake message to push to peers
func (self *Server) handshakeMsg() Msg {
	return NewMsg(handshakeMsg,
		p2pVersion,
		[]byte(self.identity.String()),
		[]interface{}{self.protocols},
		self.port,
		self.identity.Pubkey()[1:],
	)
}

func (self *Server) RegisterPubkey(candidate *Peer, pubkey []byte) error {
	// Check for blacklisting
	if self.blacklist.Exists(pubkey) {
		return fmt.Errorf("blacklisted")
	}

	self.peersLock.RLock()
	defer self.peersLock.RUnlock()
	for _, peer := range self.peers {
		if peer != nil && peer != candidate && bytes.Compare(peer.Pubkey, pubkey) == 0 {
			return fmt.Errorf("already connected")
		}
	}
	candidate.Pubkey = pubkey
	return nil
}
