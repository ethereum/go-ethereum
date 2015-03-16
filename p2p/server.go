package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
)

const (
	defaultDialTimeout   = 10 * time.Second
	refreshPeersInterval = 30 * time.Second

	// total timeout for encryption handshake and protocol
	// handshake in both directions.
	handshakeTimeout = 5 * time.Second
	// maximum time allowed for reading a complete message.
	// this is effectively the amount of time a connection can be idle.
	frameReadTimeout = 1 * time.Minute
	// maximum amount of time allowed for writing a complete message.
	frameWriteTimeout = 5 * time.Second
)

var srvlog = logger.NewLogger("P2P Server")
var srvjslog = logger.NewJsonLogger()

// Server manages all peer connections.
//
// The fields of Server are used as configuration parameters.
// You should set them before starting the Server. Fields may not be
// modified while the server is running.
type Server struct {
	// This field must be set to a valid secp256k1 private key.
	PrivateKey *ecdsa.PrivateKey

	// MaxPeers is the maximum number of peers that can be
	// connected. It must be greater than zero.
	MaxPeers int

	// Name sets the node name of this server.
	// Use common.MakeName to create a name that follows existing conventions.
	Name string

	// Bootstrap nodes are used to establish connectivity
	// with the rest of the network.
	BootstrapNodes []*discover.Node

	// Protocols should contain the protocols supported
	// by the server. Matching protocols are launched for
	// each peer.
	Protocols []Protocol

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
	NAT nat.Interface

	// If Dialer is set to a non-nil value, the given Dialer
	// is used to dial outbound peer connections.
	Dialer *net.Dialer

	// If NoDial is true, the server will not dial any peers.
	NoDial bool

	// Hooks for testing. These are useful because we can inhibit
	// the whole protocol stack.
	setupFunc
	newPeerHook

	ourHandshake *protoHandshake

	lock     sync.RWMutex
	running  bool
	listener net.Listener
	peers    map[discover.NodeID]*Peer

	ntab *discover.Table

	quit        chan struct{}
	loopWG      sync.WaitGroup // {dial,listen,nat}Loop
	peerWG      sync.WaitGroup // active peer goroutines
	peerConnect chan *discover.Node
}

type setupFunc func(net.Conn, *ecdsa.PrivateKey, *protoHandshake, *discover.Node) (*conn, error)
type newPeerHook func(*Peer)

// Peers returns all connected peers.
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

// PeerCount returns the number of connected peers.
func (srv *Server) PeerCount() int {
	srv.lock.RLock()
	n := len(srv.peers)
	srv.lock.RUnlock()
	return n
}

// SuggestPeer creates a connection to the given Node if it
// is not already connected.
func (srv *Server) SuggestPeer(n *discover.Node) {
	srv.peerConnect <- n
}

// Broadcast sends an RLP-encoded message to all connected peers.
// This method is deprecated and will be removed later.
func (srv *Server) Broadcast(protocol string, code uint64, data ...interface{}) {
	var payload []byte
	if data != nil {
		payload = common.Encode(data)
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

	// static fields
	if srv.PrivateKey == nil {
		return fmt.Errorf("Server.PrivateKey must be set to a non-nil key")
	}
	if srv.MaxPeers <= 0 {
		return fmt.Errorf("Server.MaxPeers must be > 0")
	}
	srv.quit = make(chan struct{})
	srv.peers = make(map[discover.NodeID]*Peer)
	srv.peerConnect = make(chan *discover.Node)
	if srv.setupFunc == nil {
		srv.setupFunc = setupConn
	}

	// node table
	ntab, err := discover.ListenUDP(srv.PrivateKey, srv.ListenAddr, srv.NAT)
	if err != nil {
		return err
	}
	srv.ntab = ntab

	// handshake
	srv.ourHandshake = &protoHandshake{Version: baseProtocolVersion, Name: srv.Name, ID: ntab.Self()}
	for _, p := range srv.Protocols {
		srv.ourHandshake.Caps = append(srv.ourHandshake.Caps, p.cap())
	}

	// listen/dial
	if srv.ListenAddr != "" {
		if err := srv.startListening(); err != nil {
			return err
		}
	}
	if srv.Dialer == nil {
		srv.Dialer = &net.Dialer{Timeout: defaultDialTimeout}
	}
	if !srv.NoDial {
		srv.loopWG.Add(1)
		go srv.dialLoop()
	}
	if srv.NoDial && srv.ListenAddr == "" {
		srvlog.Warnln("I will be kind-of useless, neither dialing nor listening.")
	}

	srv.running = true
	return nil
}

func (srv *Server) startListening() error {
	listener, err := net.Listen("tcp", srv.ListenAddr)
	if err != nil {
		return err
	}
	laddr := listener.Addr().(*net.TCPAddr)
	srv.ListenAddr = laddr.String()
	srv.listener = listener
	srv.loopWG.Add(1)
	go srv.listenLoop()
	if !laddr.IP.IsLoopback() && srv.NAT != nil {
		srv.loopWG.Add(1)
		go func() {
			nat.Map(srv.NAT, srv.quit, "tcp", laddr.Port, laddr.Port, "ethereum p2p")
			srv.loopWG.Done()
		}()
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

	srvlog.Infoln("Stopping Server")
	srv.ntab.Close()
	if srv.listener != nil {
		// this unblocks listener Accept
		srv.listener.Close()
	}
	close(srv.quit)
	srv.loopWG.Wait()

	// No new peers can be added at this point because dialLoop and
	// listenLoop are down. It is safe to call peerWG.Wait because
	// peerWG.Add is not called outside of those loops.
	for _, peer := range srv.peers {
		peer.Disconnect(DiscQuitting)
	}
	srv.peerWG.Wait()
}

// main loop for adding connections via listening
func (srv *Server) listenLoop() {
	defer srv.loopWG.Done()
	srvlog.Infoln("Listening on", srv.listener.Addr())
	for {
		conn, err := srv.listener.Accept()
		if err != nil {
			return
		}
		srvlog.Debugf("Accepted conn %v\n", conn.RemoteAddr())
		srv.peerWG.Add(1)
		go srv.startPeer(conn, nil)
	}
}

func (srv *Server) dialLoop() {
	defer srv.loopWG.Done()
	refresh := time.NewTicker(refreshPeersInterval)
	defer refresh.Stop()

	srv.ntab.Bootstrap(srv.BootstrapNodes)
	go srv.findPeers()

	dialed := make(chan *discover.Node)
	dialing := make(map[discover.NodeID]bool)

	// TODO: limit number of active dials
	// TODO: ensure only one findPeers goroutine is running
	// TODO: pause findPeers when we're at capacity

	for {
		select {
		case <-refresh.C:

			go srv.findPeers()

		case dest := <-srv.peerConnect:
			// avoid dialing nodes that are already connected.
			// there is another check for this in addPeer,
			// which runs after the handshake.
			srv.lock.Lock()
			_, isconnected := srv.peers[dest.ID]
			srv.lock.Unlock()
			if isconnected || dialing[dest.ID] || dest.ID == srv.ntab.Self() {
				continue
			}

			dialing[dest.ID] = true
			srv.peerWG.Add(1)
			go func() {
				srv.dialNode(dest)
				// at this point, the peer has been added
				// or discarded. either way, we're not dialing it anymore.
				dialed <- dest
			}()

		case dest := <-dialed:
			delete(dialing, dest.ID)

		case <-srv.quit:
			// TODO: maybe wait for active dials
			return
		}
	}
}

func (srv *Server) dialNode(dest *discover.Node) {
	addr := &net.TCPAddr{IP: dest.IP, Port: dest.TCPPort}
	srvlog.Debugf("Dialing %v\n", dest)
	conn, err := srv.Dialer.Dial("tcp", addr.String())
	if err != nil {
		srvlog.DebugDetailf("dial error: %v", err)
		return
	}
	srv.startPeer(conn, dest)
}

func (srv *Server) findPeers() {
	far := srv.ntab.Self()
	for i := range far {
		far[i] = ^far[i]
	}
	closeToSelf := srv.ntab.Lookup(srv.ntab.Self())
	farFromSelf := srv.ntab.Lookup(far)

	for i := 0; i < len(closeToSelf) || i < len(farFromSelf); i++ {
		if i < len(closeToSelf) {
			srv.peerConnect <- closeToSelf[i]
		}
		if i < len(farFromSelf) {
			srv.peerConnect <- farFromSelf[i]
		}
	}
}

func (srv *Server) startPeer(fd net.Conn, dest *discover.Node) {
	// TODO: handle/store session token
	fd.SetDeadline(time.Now().Add(handshakeTimeout))
	conn, err := srv.setupFunc(fd, srv.PrivateKey, srv.ourHandshake, dest)
	if err != nil {
		fd.Close()
		srvlog.Debugf("Handshake with %v failed: %v", fd.RemoteAddr(), err)
		return
	}

	conn.MsgReadWriter = &netWrapper{
		wrapped: conn.MsgReadWriter,
		conn:    fd, rtimeout: frameReadTimeout, wtimeout: frameWriteTimeout,
	}
	p := newPeer(fd, conn, srv.Protocols)
	if ok, reason := srv.addPeer(conn.ID, p); !ok {
		srvlog.DebugDetailf("Not adding %v (%v)\n", p, reason)
		p.politeDisconnect(reason)
		return
	}

	srvlog.Debugf("Added %v\n", p)
	srvjslog.LogJson(&logger.P2PConnected{
		RemoteId:            fmt.Sprintf("%x", conn.ID[:]),
		RemoteAddress:       fd.RemoteAddr().String(),
		RemoteVersionString: conn.Name,
		NumConnections:      srv.PeerCount(),
	})

	if srv.newPeerHook != nil {
		srv.newPeerHook(p)
	}
	discreason := p.run()
	srv.removePeer(p)

	srvlog.Debugf("Removed %v (%v)\n", p, discreason)
	srvjslog.LogJson(&logger.P2PDisconnected{
		RemoteId:       fmt.Sprintf("%x", conn.ID[:]),
		NumConnections: srv.PeerCount(),
	})
}

func (srv *Server) addPeer(id discover.NodeID, p *Peer) (bool, DiscReason) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	switch {
	case !srv.running:
		return false, DiscQuitting
	case len(srv.peers) >= srv.MaxPeers:
		return false, DiscTooManyPeers
	case srv.peers[id] != nil:
		return false, DiscAlreadyConnected
	case id == srv.ntab.Self():
		return false, DiscSelf
	}
	srv.peers[id] = p
	return true, 0
}

func (srv *Server) removePeer(p *Peer) {
	srv.lock.Lock()
	delete(srv.peers, p.ID())
	srv.lock.Unlock()
	srv.peerWG.Done()
}
