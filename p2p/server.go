package p2p

import (
	"bytes"
	"crypto/ecdsa"
	"crypto/rand"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/nat"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	defaultDialTimeout      = 10 * time.Second
	refreshPeersInterval    = 30 * time.Second
	staticPeerCheckInterval = 15 * time.Second

	// This is the maximum number of inbound connection
	// that are allowed to linger between 'accepted' and
	// 'added as peer'.
	maxAcceptConns = 50

	// total timeout for encryption handshake and protocol
	// handshake in both directions.
	handshakeTimeout = 5 * time.Second
	// maximum time allowed for reading a complete message.
	// this is effectively the amount of time a connection can be idle.
	frameReadTimeout = 1 * time.Minute
	// maximum amount of time allowed for writing a complete message.
	frameWriteTimeout = 5 * time.Second
)

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

	// Static nodes are used as pre-configured connections which are always
	// maintained and re-connected on disconnects.
	StaticNodes []*discover.Node

	// Trusted nodes are used as pre-configured connections which are always
	// allowed to connect, even above the peer limit.
	TrustedNodes []*discover.Node

	// NodeDatabase is the path to the database containing the previously seen
	// live nodes in the network.
	NodeDatabase string

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

	lock         sync.RWMutex // protects running, peers and the trust fields
	running      bool
	peers        map[discover.NodeID]*Peer
	staticNodes  map[discover.NodeID]*discover.Node // Map of currently maintained static remote nodes
	staticDial   chan *discover.Node                // Dial request channel reserved for the static nodes
	staticCycle  time.Duration                      // Overrides staticPeerCheckInterval, used for testing
	trustedNodes map[discover.NodeID]bool           // Set of currently trusted remote nodes

	ntab     *discover.Table
	listener net.Listener

	quit   chan struct{}
	loopWG sync.WaitGroup // {dial,listen,nat}Loop
	peerWG sync.WaitGroup // active peer goroutines
}

type setupFunc func(net.Conn, *ecdsa.PrivateKey, *protoHandshake, *discover.Node, bool, map[discover.NodeID]bool) (*conn, error)
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

// AddPeer connects to the given node and maintains the connection until the
// server is shut down. If the connection fails for any reason, the server will
// attempt to reconnect the peer.
func (srv *Server) AddPeer(node *discover.Node) {
	srv.lock.Lock()
	defer srv.lock.Unlock()

	srv.staticNodes[node.ID] = node
}

// Broadcast sends an RLP-encoded message to all connected peers.
// This method is deprecated and will be removed later.
func (srv *Server) Broadcast(protocol string, code uint64, data interface{}) error {
	return srv.BroadcastLimited(protocol, code, func(i float64) float64 { return i }, data)
}

// BroadcastsRange an RLP-encoded message to a random set of peers using the limit function to limit the amount
// of peers.
func (srv *Server) BroadcastLimited(protocol string, code uint64, limit func(float64) float64, data interface{}) error {
	var payload []byte
	if data != nil {
		var err error
		payload, err = rlp.EncodeToBytes(data)
		if err != nil {
			return err
		}
	}
	srv.lock.RLock()
	defer srv.lock.RUnlock()

	i, max := 0, int(limit(float64(len(srv.peers))))
	for _, peer := range srv.peers {
		if i >= max {
			break
		}

		if peer != nil {
			var msg = Msg{Code: code}
			if data != nil {
				msg.Payload = bytes.NewReader(payload)
				msg.Size = uint32(len(payload))
			}
			peer.writeProtoMsg(protocol, msg)
			i++
		}
	}
	return nil
}

// Start starts running the server.
// Servers can be re-used and started again after stopping.
func (srv *Server) Start() (err error) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if srv.running {
		return errors.New("server already running")
	}
	glog.V(logger.Info).Infoln("Starting Server")

	// static fields
	if srv.PrivateKey == nil {
		return fmt.Errorf("Server.PrivateKey must be set to a non-nil key")
	}
	if srv.MaxPeers <= 0 {
		return fmt.Errorf("Server.MaxPeers must be > 0")
	}
	srv.quit = make(chan struct{})
	srv.peers = make(map[discover.NodeID]*Peer)

	// Create the current trust maps, and the associated dialing channel
	srv.trustedNodes = make(map[discover.NodeID]bool)
	for _, node := range srv.TrustedNodes {
		srv.trustedNodes[node.ID] = true
	}
	srv.staticNodes = make(map[discover.NodeID]*discover.Node)
	for _, node := range srv.StaticNodes {
		srv.staticNodes[node.ID] = node
	}
	srv.staticDial = make(chan *discover.Node)

	if srv.setupFunc == nil {
		srv.setupFunc = setupConn
	}

	// node table
	ntab, err := discover.ListenUDP(srv.PrivateKey, srv.ListenAddr, srv.NAT, srv.NodeDatabase)
	if err != nil {
		return err
	}
	srv.ntab = ntab

	// handshake
	srv.ourHandshake = &protoHandshake{Version: baseProtocolVersion, Name: srv.Name, ID: ntab.Self().ID}
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
		glog.V(logger.Warn).Infoln("I will be kind-of useless, neither dialing nor listening.")
	}
	// maintain the static peers
	go srv.staticNodesLoop()

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

	glog.V(logger.Info).Infoln("Stopping Server")
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
	srv.lock.Lock()
	for _, peer := range srv.peers {
		peer.Disconnect(DiscQuitting)
	}
	srv.lock.Unlock()
	srv.peerWG.Wait()
}

// Self returns the local node's endpoint information.
func (srv *Server) Self() *discover.Node {
	srv.lock.RLock()
	defer srv.lock.RUnlock()
	if !srv.running {
		return &discover.Node{IP: net.ParseIP("0.0.0.0")}
	}
	return srv.ntab.Self()
}

// main loop for adding connections via listening
func (srv *Server) listenLoop() {
	defer srv.loopWG.Done()

	// This channel acts as a semaphore limiting
	// active inbound connections that are lingering pre-handshake.
	// If all slots are taken, no further connections are accepted.
	slots := make(chan struct{}, maxAcceptConns)
	for i := 0; i < maxAcceptConns; i++ {
		slots <- struct{}{}
	}

	glog.V(logger.Info).Infoln("Listening on", srv.listener.Addr())
	for {
		<-slots
		conn, err := srv.listener.Accept()
		if err != nil {
			return
		}
		glog.V(logger.Debug).Infof("Accepted conn %v\n", conn.RemoteAddr())
		srv.peerWG.Add(1)
		go func() {
			srv.startPeer(conn, nil)
			slots <- struct{}{}
		}()
	}
}

// staticNodesLoop is responsible for periodically checking that static
// connections are actually live, and requests dialing if not.
func (srv *Server) staticNodesLoop() {
	// Create a default maintenance ticker, but override it requested
	cycle := staticPeerCheckInterval
	if srv.staticCycle != 0 {
		cycle = srv.staticCycle
	}
	tick := time.NewTicker(cycle)

	for {
		select {
		case <-srv.quit:
			return

		case <-tick.C:
			// Collect all the non-connected static nodes
			needed := []*discover.Node{}
			srv.lock.RLock()
			for id, node := range srv.staticNodes {
				if _, ok := srv.peers[id]; !ok {
					needed = append(needed, node)
				}
			}
			srv.lock.RUnlock()

			// Try to dial each of them (don't hang if server terminates)
			for _, node := range needed {
				glog.V(logger.Debug).Infof("Dialing static peer %v", node)
				select {
				case srv.staticDial <- node:
				case <-srv.quit:
					return
				}
			}
		}
	}
}

func (srv *Server) dialLoop() {
	var (
		dialed      = make(chan *discover.Node)
		dialing     = make(map[discover.NodeID]bool)
		findresults = make(chan []*discover.Node)
		refresh     = time.NewTimer(0)
	)
	defer srv.loopWG.Done()
	defer refresh.Stop()

	// TODO: maybe limit number of active dials
	dial := func(dest *discover.Node) {
		// Don't dial nodes that would fail the checks in addPeer.
		// This is important because the connection handshake is a lot
		// of work and we'd rather avoid doing that work for peers
		// that can't be added.
		srv.lock.RLock()
		ok, _ := srv.checkPeer(dest.ID)
		srv.lock.RUnlock()
		if !ok || dialing[dest.ID] {
			return
		}

		dialing[dest.ID] = true
		srv.peerWG.Add(1)
		go func() {
			srv.dialNode(dest)
			dialed <- dest
		}()
	}

	srv.ntab.Bootstrap(srv.BootstrapNodes)
	for {
		select {
		case <-refresh.C:
			// Grab some nodes to connect to if we're not at capacity.
			srv.lock.RLock()
			needpeers := len(srv.peers) < srv.MaxPeers/2
			srv.lock.RUnlock()
			if needpeers {
				go func() {
					var target discover.NodeID
					rand.Read(target[:])
					findresults <- srv.ntab.Lookup(target)
				}()
			} else {
				// Make sure we check again if the peer count falls
				// below MaxPeers.
				refresh.Reset(refreshPeersInterval)
			}
		case dest := <-srv.staticDial:
			dial(dest)
		case dests := <-findresults:
			for _, dest := range dests {
				dial(dest)
			}
			refresh.Reset(refreshPeersInterval)
		case dest := <-dialed:
			delete(dialing, dest.ID)
			if len(dialing) == 0 {
				// Check again immediately after dialing all current candidates.
				refresh.Reset(0)
			}
		case <-srv.quit:
			// TODO: maybe wait for active dials
			return
		}
	}
}

func (srv *Server) dialNode(dest *discover.Node) {
	addr := &net.TCPAddr{IP: dest.IP, Port: int(dest.TCP)}
	glog.V(logger.Debug).Infof("Dialing %v\n", dest)
	conn, err := srv.Dialer.Dial("tcp", addr.String())
	if err != nil {
		// dialLoop adds to the wait group counter when launching
		// dialNode, so we need to count it down again. startPeer also
		// does that when an error occurs.
		srv.peerWG.Done()
		glog.V(logger.Detail).Infof("dial error: %v", err)
		return
	}
	srv.startPeer(conn, dest)
}

func (srv *Server) startPeer(fd net.Conn, dest *discover.Node) {
	// TODO: handle/store session token

	// Run setupFunc, which should create an authenticated connection
	// and run the capability exchange. Note that any early error
	// returns during that exchange need to call peerWG.Done because
	// the callers of startPeer added the peer to the wait group already.
	fd.SetDeadline(time.Now().Add(handshakeTimeout))

	// Check capacity, but override for static nodes
	srv.lock.RLock()
	atcap := len(srv.peers) == srv.MaxPeers
	if dest != nil {
		if _, ok := srv.staticNodes[dest.ID]; ok {
			atcap = false
		}
	}
	srv.lock.RUnlock()

	conn, err := srv.setupFunc(fd, srv.PrivateKey, srv.ourHandshake, dest, atcap, srv.trustedNodes)
	if err != nil {
		fd.Close()
		glog.V(logger.Debug).Infof("Handshake with %v failed: %v", fd.RemoteAddr(), err)
		srv.peerWG.Done()
		return
	}
	conn.MsgReadWriter = &netWrapper{
		wrapped: conn.MsgReadWriter,
		conn:    fd, rtimeout: frameReadTimeout, wtimeout: frameWriteTimeout,
	}
	p := newPeer(fd, conn, srv.Protocols)
	if ok, reason := srv.addPeer(conn.ID, p); !ok {
		glog.V(logger.Detail).Infof("Not adding %v (%v)\n", p, reason)
		p.politeDisconnect(reason)
		srv.peerWG.Done()
		return
	}
	// The handshakes are done and it passed all checks.
	// Spawn the Peer loops.
	go srv.runPeer(p)
}

func (srv *Server) runPeer(p *Peer) {
	glog.V(logger.Debug).Infof("Added %v\n", p)
	srvjslog.LogJson(&logger.P2PConnected{
		RemoteId:            p.ID().String(),
		RemoteAddress:       p.RemoteAddr().String(),
		RemoteVersionString: p.Name(),
		NumConnections:      srv.PeerCount(),
	})
	if srv.newPeerHook != nil {
		srv.newPeerHook(p)
	}
	discreason := p.run()
	srv.removePeer(p)
	glog.V(logger.Debug).Infof("Removed %v (%v)\n", p, discreason)
	srvjslog.LogJson(&logger.P2PDisconnected{
		RemoteId:       p.ID().String(),
		NumConnections: srv.PeerCount(),
	})
}

func (srv *Server) addPeer(id discover.NodeID, p *Peer) (bool, DiscReason) {
	srv.lock.Lock()
	defer srv.lock.Unlock()
	if ok, reason := srv.checkPeer(id); !ok {
		return false, reason
	}
	srv.peers[id] = p
	return true, 0
}

// checkPeer verifies whether a peer looks promising and should be allowed/kept
// in the pool, or if it's of no use.
func (srv *Server) checkPeer(id discover.NodeID) (bool, DiscReason) {
	// First up, figure out if the peer is static or trusted
	_, static := srv.staticNodes[id]
	trusted := srv.trustedNodes[id]

	// Make sure the peer passes all required checks
	switch {
	case !srv.running:
		return false, DiscQuitting
	case !static && !trusted && len(srv.peers) >= srv.MaxPeers:
		return false, DiscTooManyPeers
	case srv.peers[id] != nil:
		return false, DiscAlreadyConnected
	case id == srv.ntab.Self().ID:
		return false, DiscSelf
	default:
		return true, 0
	}
}

func (srv *Server) removePeer(p *Peer) {
	srv.lock.Lock()
	delete(srv.peers, p.ID())
	srv.lock.Unlock()
	srv.peerWG.Done()
}
