package p2p

import (
	"bytes"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
)

const (
	outboundAddressPoolSize   = 500
	defaultDialTimeout        = 10 * time.Second
	portMappingUpdateInterval = 15 * time.Minute
	portMappingTimeout        = 20 * time.Minute
)

var srvlog = logger.NewLogger("P2P Server")

// Server manages all peer connections.
//
// The fields of Server are used as configuration parameters.
// You should set them before starting the Server. Fields may not be
// modified while the server is running.
type Server struct {
	// This field must be set to a valid client identity.
	Identity ClientIdentity

	// MaxPeers is the maximum number of peers that can be
	// connected. It must be greater than zero.
	MaxPeers int

	// Protocols should contain the protocols supported
	// by the server. Matching protocols are launched for
	// each peer.
	Protocols []Protocol

	// If Blacklist is set to a non-nil value, the given Blacklist
	// is used to verify peer connections.
	Blacklist Blacklist

	// If ListenAddr is set to a non-nil address, the server
	// will listen for incoming connections.
	//
	// If the port is zero, the operating system will pick a port. The
	// ListenAddr field will be updated with the actual address when
	// the server is started.
	ListenAddr string

	// If set to a non-nil value, the given NAT port mapper
	// is used to make the listening port available to the
	// Internet.
	NAT NAT

	// If Dialer is set to a non-nil value, the given Dialer
	// is used to dial outbound peer connections.
	Dialer *net.Dialer

	// If NoDial is true, the server will not dial any peers.
	NoDial bool

	// peer selector
	PeerSelector peerSelector

	// Hook for testing. This is useful because we can inhibit
	// the whole protocol stack.
	// newPeerFunc peerFunc

	lock      sync.RWMutex
	running   bool
	listener  net.Listener
	laddr     *net.TCPAddr // real listen addr
	peers     []*Peer
	peerSlots chan int
	peerCount int

	connectFunc func(*Peer, net.Conn)

	quit           chan struct{}
	wg             sync.WaitGroup
	peerDisconnect chan *Peer
}

// NAT is implemented by NAT traversal methods.
type NAT interface {
	GetExternalAddress() (net.IP, error)
	AddPortMapping(protocol string, extport, intport int, name string, lifetime time.Duration) error
	DeletePortMapping(protocol string, extport, intport int) error

	// Should return name of the method.
	String() string
}

// Peers returns all currently connected peers.
func (srv *Server) Peers() (peers []*Peer) {
	srv.lock.RLock()
	defer srv.lock.RUnlock()
	for _, peer := range srv.peers {
		if peer != nil {
			peers = append(peers, peer)
		}
	}
	return
}

// GetPeers returns addresses near target if given or supported by the client
// or falls back to all actively connected peers
func (srv *Server) GetPeers(target ...[]byte) []*peerAddr {
	if len(target) == 1 { // delegate to selector
		return ActiveAddresses(srv.PeerSelector.GetPeers(target[0])...)
	} else {
		// in fact it is not clear why the selector would not want to reply to this case as well
		var peers []peerInfo
		for _, peer := range srv.Peers() {
			peers = append(peers, peer)
		}
		return ActiveAddresses(peers...)
	}
}

// PeerCount returns the number of connected peers.
func (srv *Server) PeerCount() int {
	srv.lock.RLock()
	defer srv.lock.RUnlock()
	return srv.peerCount
}

// SuggestPeer is a convenient method that does dns resolution
// and passes on the request to AddPeer
func (srv *Server) SuggestPeer(addr string, pubkey []byte) error {
	netaddr, err := net.ResolveTCPAddr("tcp", addr)
	if err != nil {
		srvlog.Errorf("couldn't resolve %s:", addr, err)
		return err
	}
	peerAddr := &peerAddr{netaddr.IP, uint64(netaddr.Port), pubkey}
	return srv.AddPeer(peerAddr)
}

// AddPeer takes a peerAddr address as argument.
// If not found among connected peers turns to the peerSelector
// to decide if it is a worthwhile connection
func (srv *Server) AddPeer(addr *peerAddr) (err error) {
	// need to look up nodeID first
	srvlog.Infof("checking peer %v", addr)

	peer := &Peer{
		dialAddr:    addr,
		lastActiveC: make(chan time.Time),
		lastActive:  time.Now().Add(-24 * time.Hour),
	}
	if srv.PeerSelector.AddPeer(peer) {
		srvlog.Infof("peer %v accepted by peer selection", addr)
		err = srv.dialPeer(peer)
	} else {
		srvlog.Infof("peer %v rejected by peer selection", addr)
	}
	return
}

func (srv *Server) dialPeer(peer *Peer) (err error) {
	timeout := time.After(5 * time.Second)
	select {
	case <-timeout:
		err = fmt.Errorf("Too many connections. No slot available")
	case slot := <-srv.peerSlots: // there is a slot available
		srvlog.Debugf("Dialing %v (slot %d)\n", peer.dialAddr, slot)
		conn, dialErr := srv.Dialer.Dial(peer.dialAddr.Network(), peer.dialAddr.String())
		if dialErr != nil {
			err = fmt.Errorf("Dial error: %v", dialErr)
			srvlog.Errorln(err)
			srv.peerSlots <- slot
			return
		}
		srvlog.Debugf("Connected to %v (slot %d)\n", peer.dialAddr, slot)
		peer.slot = slot
		srv.connectFunc(peer, conn)
		go srv.addPeer(peer)
	}
	return

}

func (srv *Server) connect(p *Peer, conn net.Conn) {
	p.init(conn)
	p.ourID = srv.Identity
	p.addPeer = srv.AddPeer
	p.getPeers = srv.GetPeers
	p.pubkeyHook = srv.verifyPeer
	p.runBaseProtocol = true
	p.protocols = srv.Protocols

	// laddr can be updated concurrently by NAT traversal.
	if srv.laddr != nil {
		p.ourListenAddr = newPeerAddr(srv.laddr, srv.Identity.Pubkey())
	}
}

// Broadcast sends an RLP-encoded message to all connected peers.
// This method is deprecated and will be removed later.
func (srv *Server) Broadcast(protocol string, code uint64, data ...interface{}) {
	var payload []byte
	if data != nil {
		payload = encodePayload(data...)
	}
	srv.lock.RLock()
	defer srv.lock.RUnlock()
	for _, peer := range srv.peers {
		if peer != nil {
			var msg = Msg{Code: code}
			if data != nil {
				msg.Payload = bytes.NewReader(payload)
				msg.Size = uint32(len(payload))
			}
			peer.writeProtoMsg(protocol, msg)
		}
	}
}

// Start starts running the server.
// Servers can be re-used and started again after stopping.
func (srv *Server) Start() (err error) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.running {
		return errors.New("server already running")
	}
	srvlog.Infoln("Starting Server")

	// initialize fields
	if srv.Identity == nil {
		return fmt.Errorf("Server.Identity must be set to a non-nil identity")
	}
	if srv.MaxPeers <= 0 {
		return fmt.Errorf("Server.MaxPeers must be > 0")
	}
	srv.quit = make(chan struct{})
	srv.peers = make([]*Peer, srv.MaxPeers)
	srv.peerSlots = make(chan int, srv.MaxPeers)
	srv.peerDisconnect = make(chan *Peer)
	// if srv.newPeerFunc == nil {
	// 	srv.newPeerFunc = newServerPeer
	// }
	if srv.Blacklist == nil {
		srv.Blacklist = NewBlacklist()
	}
	if srv.Dialer == nil {
		srv.Dialer = &net.Dialer{Timeout: defaultDialTimeout}
	}

	if srv.ListenAddr != "" {
		if err := srv.startListening(); err != nil {
			return err
		}
	}
	// if !srv.NoDial {
	// 	srv.wg.Add(1)
	// 	go srv.dialLoop()
	// }
	if srv.NoDial && srv.ListenAddr == "" {
		srvlog.Warnln("I will be kind-of useless, neither dialing nor listening.")
	}

	if srv.PeerSelector == nil {
		srv.PeerSelector = &BaseSelector{}
	}

	if srv.connectFunc == nil {
		srv.connectFunc = srv.connect
	}

	// make all slots available
	for i := range srv.peers {
		srv.peerSlots <- i
	}
	// note: discLoop is not part of WaitGroup
	go srv.discLoop()
	srv.running = true
	return nil
}

func (srv *Server) startListening() error {
	listener, err := net.Listen("tcp", srv.ListenAddr)
	if err != nil {
		return err
	}
	srv.ListenAddr = listener.Addr().String()
	srv.laddr = listener.Addr().(*net.TCPAddr)
	srv.listener = listener
	srv.wg.Add(1)
	go srv.listenLoop()
	if !srv.laddr.IP.IsLoopback() && srv.NAT != nil {
		srv.wg.Add(1)
		go srv.natLoop(srv.laddr.Port)
	}
	return nil
}

// Stop terminates the server and all active peer connections.
// It blocks until all active connections have been closed.
func (srv *Server) Stop() {
	srv.lock.Lock()
	if !srv.running {
		srv.lock.Unlock()
		return
	}
	srv.running = false
	srv.lock.Unlock()

	srvlog.Infoln("Stopping server")
	if srv.listener != nil {
		// this unblocks listener Accept
		srv.listener.Close()
	}
	close(srv.quit)
	for _, peer := range srv.Peers() {
		peer.Disconnect(DiscQuitting)
	}
	srv.wg.Wait()

	// wait till they actually disconnect
	// this is checked by claiming all peerSlots.
	// slots become available as the peers disconnect.
	for i := 0; i < cap(srv.peerSlots); i++ {
		<-srv.peerSlots
	}
	// terminate discLoop
	close(srv.peerDisconnect)
}

func (srv *Server) discLoop() {
	for peer := range srv.peerDisconnect {
		srv.removePeer(peer)
	}
}

// main loop for adding connections via listening
func (srv *Server) listenLoop() {
	defer srv.wg.Done()

	srvlog.Infoln("Listening on", srv.listener.Addr())
	for {
		select {
		case slot := <-srv.peerSlots:
			srvlog.Debugf("grabbed slot %v for listening", slot)
			conn, err := srv.listener.Accept()
			if err != nil {
				srv.peerSlots <- slot
				return
			}
			srvlog.Debugf("Accepted conn %v (slot %d) - peer selector check after handshake", conn.RemoteAddr(), slot)
			peer := &Peer{slot: slot}
			srv.connectFunc(peer, conn)
			srv.addPeer(peer)
		case <-srv.quit:
			return
		}
	}
}

func (srv *Server) natLoop(port int) {
	defer srv.wg.Done()
	for {
		srv.updatePortMapping(port)
		select {
		case <-time.After(portMappingUpdateInterval):
			// one more round
		case <-srv.quit:
			srv.removePortMapping(port)
			return
		}
	}
}

func (srv *Server) updatePortMapping(port int) {
	srvlog.Infoln("Attempting to map port", port, "with", srv.NAT)
	err := srv.NAT.AddPortMapping("tcp", port, port, "ethereum p2p", portMappingTimeout)
	if err != nil {
		srvlog.Errorln("Port mapping error:", err)
		return
	}
	extip, err := srv.NAT.GetExternalAddress()
	if err != nil {
		srvlog.Errorln("Error getting external IP:", err)
		return
	}
	srv.lock.Lock()
	extaddr := *(srv.listener.Addr().(*net.TCPAddr))
	extaddr.IP = extip
	srvlog.Infoln("Mapped port, external addr is", &extaddr)
	srv.laddr = &extaddr
	srv.lock.Unlock()
}

func (srv *Server) removePortMapping(port int) {
	srvlog.Infoln("Removing port mapping for", port, "with", srv.NAT)
	srv.NAT.DeletePortMapping("tcp", port, port)
}

// creates the new peer object and inserts it into its slot
func (srv *Server) addPeer(peer *Peer) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	srvlog.Debugf("Add peer %v (slot %v)\n", peer, peer.slot)
	srv.peers[peer.slot] = peer
	srv.peerCount++
	go func() { peer.loop(); srv.peerDisconnect <- peer }()
	return
}

// removes peer: sending disconnect msg, stop peer, remove rom list/table, release slot
func (srv *Server) removePeer(peer *Peer) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	srvlog.Debugf("Remove peer %v (slot %v)\n", peer, peer.slot)
	if srv.peers[peer.slot] != peer {
		srvlog.Warnln("Invalid peer to remove:", peer)
		return
	}
	// remove from list and index
	srv.peerCount--
	srv.peers[peer.slot] = nil
	// release slot to signal need for a new peer, last!
	srv.peerSlots <- peer.slot
}

func (srv *Server) verifyPeer(addr *peerAddr) error {
	if srv.Blacklist.Exists(addr.Pubkey) {
		return errors.New("blacklisted")
	}
	if bytes.Equal(srv.Identity.Pubkey()[1:], addr.Pubkey) {
		return newPeerError(errPubkeyForbidden, "not allowed to connect to srv")
	}
	srv.lock.RLock()
	defer srv.lock.RUnlock()
	for _, peer := range srv.peers {
		if peer != nil {
			id := peer.Identity()
			if id != nil && bytes.Equal(id.Pubkey(), addr.Pubkey) {
				return errors.New("already connected")
			}
		}
	}
	return nil
}

// TODO replace with "Set"
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
