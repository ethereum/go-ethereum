package p2p

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/event"
	"github.com/ethereum/go-ethereum/logger"
)

// peerAddr is the structure of a peer list element.
// It is also a valid net.Addr.
type peerAddr struct {
	IP     net.IP
	Port   uint64
	Pubkey []byte // optional
}

func newPeerAddr(addr net.Addr, pubkey []byte) *peerAddr {
	n := addr.Network()
	if n != "tcp" && n != "tcp4" && n != "tcp6" {
		// for testing with non-TCP
		return &peerAddr{net.ParseIP("127.0.0.1"), 30303, pubkey}
	}
	ta := addr.(*net.TCPAddr)
	return &peerAddr{ta.IP, uint64(ta.Port), pubkey}
}

func (d peerAddr) Network() string {
	if d.IP.To4() != nil {
		return "tcp4"
	} else {
		return "tcp6"
	}
}

func (d peerAddr) String() string {
	return fmt.Sprintf("%v:%d", d.IP, d.Port)
}

func (d *peerAddr) RlpData() interface{} {
	return []interface{}{string(d.IP), d.Port, d.Pubkey}
}

// Peer represents a remote peer.
type Peer struct {
	// Peers have all the log methods.
	// Use them to display messages related to the peer.
	*logger.Logger

	infolock   sync.Mutex
	identity   ClientIdentity
	caps       []Cap
	listenAddr *peerAddr // what remote peer is listening on
	dialAddr   *peerAddr // non-nil if dialing

	// The mutex protects the connection
	// so only one protocol can write at a time.
	writeMu sync.Mutex
	conn    net.Conn
	bufconn *bufio.ReadWriter

	// These fields maintain the running protocols.
	protocols       []Protocol
	runBaseProtocol bool // for testing
	cryptoHandshake bool // for testing

	runlock sync.RWMutex // protects running
	running map[string]*proto

	protoWG  sync.WaitGroup
	protoErr chan error
	closed   chan struct{}
	disc     chan DiscReason

	activity event.TypeMux // for activity events

	slot int // index into Server peer list

	// These fields are kept so base protocol can access them.
	// TODO: this should be one or more interfaces
	ourID         ClientIdentity        // client id of the Server
	ourListenAddr *peerAddr             // listen addr of Server, nil if not listening
	newPeerAddr   chan<- *peerAddr      // tell server about received peers
	otherPeers    func() []*Peer        // should return the list of all peers
	pubkeyHook    func(*peerAddr) error // called at end of handshake to validate pubkey
}

// NewPeer returns a peer for testing purposes.
func NewPeer(id ClientIdentity, caps []Cap) *Peer {
	conn, _ := net.Pipe()
	peer := newPeer(conn, nil, nil)
	peer.setHandshakeInfo(id, nil, caps)
	close(peer.closed)
	return peer
}

func newServerPeer(server *Server, conn net.Conn, dialAddr *peerAddr) *Peer {
	p := newPeer(conn, server.Protocols, dialAddr)
	p.ourID = server.Identity
	p.newPeerAddr = server.peerConnect
	p.otherPeers = server.Peers
	p.pubkeyHook = server.verifyPeer
	p.runBaseProtocol = true

	// laddr can be updated concurrently by NAT traversal.
	// newServerPeer must be called with the server lock held.
	if server.laddr != nil {
		p.ourListenAddr = newPeerAddr(server.laddr, server.Identity.Pubkey())
	}
	return p
}

func newPeer(conn net.Conn, protocols []Protocol, dialAddr *peerAddr) *Peer {
	p := &Peer{
		Logger:    logger.NewLogger("P2P " + conn.RemoteAddr().String()),
		conn:      conn,
		dialAddr:  dialAddr,
		bufconn:   bufio.NewReadWriter(bufio.NewReader(conn), bufio.NewWriter(conn)),
		protocols: protocols,
		running:   make(map[string]*proto),
		disc:      make(chan DiscReason),
		protoErr:  make(chan error),
		closed:    make(chan struct{}),
	}
	return p
}

// Identity returns the client identity of the remote peer. The
// identity can be nil if the peer has not yet completed the
// handshake.
func (p *Peer) Identity() ClientIdentity {
	p.infolock.Lock()
	defer p.infolock.Unlock()
	return p.identity
}

func (self *Peer) Pubkey() (pubkey []byte) {
	self.infolock.Lock()
	defer self.infolock.Unlock()
	switch {
	case self.identity != nil:
		pubkey = self.identity.Pubkey()
	case self.dialAddr != nil:
		pubkey = self.dialAddr.Pubkey
	case self.listenAddr != nil:
		pubkey = self.listenAddr.Pubkey
	}
	return
}

// Caps returns the capabilities (supported subprotocols) of the remote peer.
func (p *Peer) Caps() []Cap {
	p.infolock.Lock()
	defer p.infolock.Unlock()
	return p.caps
}

func (p *Peer) setHandshakeInfo(id ClientIdentity, laddr *peerAddr, caps []Cap) {
	p.infolock.Lock()
	p.identity = id
	p.listenAddr = laddr
	p.caps = caps
	p.infolock.Unlock()
}

// RemoteAddr returns the remote address of the network connection.
func (p *Peer) RemoteAddr() net.Addr {
	return p.conn.RemoteAddr()
}

// LocalAddr returns the local address of the network connection.
func (p *Peer) LocalAddr() net.Addr {
	return p.conn.LocalAddr()
}

// Disconnect terminates the peer connection with the given reason.
// It returns immediately and does not wait until the connection is closed.
func (p *Peer) Disconnect(reason DiscReason) {
	select {
	case p.disc <- reason:
	case <-p.closed:
	}
}

// String implements fmt.Stringer.
func (p *Peer) String() string {
	kind := "inbound"
	p.infolock.Lock()
	if p.dialAddr != nil {
		kind = "outbound"
	}
	p.infolock.Unlock()
	return fmt.Sprintf("Peer(%p %v %s)", p, p.conn.RemoteAddr(), kind)
}

const (
	// maximum amount of time allowed for reading a message
	msgReadTimeout = 5 * time.Second
	// maximum amount of time allowed for writing a message
	msgWriteTimeout = 5 * time.Second
	// messages smaller than this many bytes will be read at
	// once before passing them to a protocol.
	wholePayloadSize = 64 * 1024
)

var (
	inactivityTimeout     = 2 * time.Second
	disconnectGracePeriod = 2 * time.Second
)

func (p *Peer) loop() (reason DiscReason, err error) {
	defer p.activity.Stop()
	defer p.closeProtocols()
	defer close(p.closed)
	defer p.conn.Close()

	if p.cryptoHandshake {
		if err := p.handleCryptoHandshake(); err != nil {
			return DiscProtocolError, err // no graceful disconnect
		}
	}

	// read loop
	readMsg := make(chan Msg)
	readErr := make(chan error)
	readNext := make(chan bool, 1)
	protoDone := make(chan struct{}, 1)
	go p.readLoop(readMsg, readErr, readNext)
	readNext <- true

	if p.runBaseProtocol {
		p.startBaseProtocol()
	}

loop:
	for {
		select {
		case msg := <-readMsg:
			// a new message has arrived.
			var wait bool
			if wait, err = p.dispatch(msg, protoDone); err != nil {
				p.Errorf("msg dispatch error: %v\n", err)
				reason = discReasonForError(err)
				break loop
			}
			if !wait {
				// Msg has already been read completely, continue with next message.
				readNext <- true
			}
			p.activity.Post(time.Now())
		case <-protoDone:
			// protocol has consumed the message payload,
			// we can continue reading from the socket.
			readNext <- true

		case err := <-readErr:
			// read failed. there is no need to run the
			// polite disconnect sequence because the connection
			// is probably dead anyway.
			// TODO: handle write errors as well
			return DiscNetworkError, err
		case err = <-p.protoErr:
			reason = discReasonForError(err)
			break loop
		case reason = <-p.disc:
			break loop
		}
	}

	// wait for read loop to return.
	close(readNext)
	<-readErr
	// tell the remote end to disconnect
	done := make(chan struct{})
	go func() {
		p.conn.SetDeadline(time.Now().Add(disconnectGracePeriod))
		p.writeMsg(NewMsg(discMsg, reason), disconnectGracePeriod)
		io.Copy(ioutil.Discard, p.conn)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(disconnectGracePeriod):
	}
	return reason, err
}

func (p *Peer) readLoop(msgc chan<- Msg, errc chan<- error, unblock <-chan bool) {
	for _ = range unblock {
		p.conn.SetReadDeadline(time.Now().Add(msgReadTimeout))
		if msg, err := readMsg(p.bufconn); err != nil {
			errc <- err
		} else {
			msgc <- msg
		}
	}
	close(errc)
}

func (p *Peer) dispatch(msg Msg, protoDone chan struct{}) (wait bool, err error) {
	proto, err := p.getProto(msg.Code)
	if err != nil {
		return false, err
	}
	if msg.Size <= wholePayloadSize {
		// optimization: msg is small enough, read all
		// of it and move on to the next message
		buf, err := ioutil.ReadAll(msg.Payload)
		if err != nil {
			return false, err
		}
		msg.Payload = bytes.NewReader(buf)
		proto.in <- msg
	} else {
		wait = true
		pr := &eofSignal{msg.Payload, int64(msg.Size), protoDone}
		msg.Payload = pr
		proto.in <- msg
	}
	return wait, nil
}

func (p *Peer) handleCryptoHandshake() (err error) {

	return nil
}

func (p *Peer) startBaseProtocol() {
	p.runlock.Lock()
	defer p.runlock.Unlock()
	p.running[""] = p.startProto(0, Protocol{
		Length: baseProtocolLength,
		Run:    runBaseProtocol,
	})
}

// startProtocols starts matching named subprotocols.
func (p *Peer) startSubprotocols(caps []Cap) {
	sort.Sort(capsByName(caps))

	p.runlock.Lock()
	defer p.runlock.Unlock()
	offset := baseProtocolLength
outer:
	for _, cap := range caps {
		for _, proto := range p.protocols {
			if proto.Name == cap.Name &&
				proto.Version == cap.Version &&
				p.running[cap.Name] == nil {
				p.running[cap.Name] = p.startProto(offset, proto)
				offset += proto.Length
				continue outer
			}
		}
	}
}

func (p *Peer) startProto(offset uint64, impl Protocol) *proto {
	rw := &proto{
		in:      make(chan Msg),
		offset:  offset,
		maxcode: impl.Length,
		peer:    p,
	}
	p.protoWG.Add(1)
	go func() {
		err := impl.Run(p, rw)
		if err == nil {
			p.Infof("protocol %q returned", impl.Name)
			err = newPeerError(errMisc, "protocol returned")
		} else {
			p.Errorf("protocol %q error: %v\n", impl.Name, err)
		}
		select {
		case p.protoErr <- err:
		case <-p.closed:
		}
		p.protoWG.Done()
	}()
	return rw
}

// getProto finds the protocol responsible for handling
// the given message code.
func (p *Peer) getProto(code uint64) (*proto, error) {
	p.runlock.RLock()
	defer p.runlock.RUnlock()
	for _, proto := range p.running {
		if code >= proto.offset && code < proto.offset+proto.maxcode {
			return proto, nil
		}
	}
	return nil, newPeerError(errInvalidMsgCode, "%d", code)
}

func (p *Peer) closeProtocols() {
	p.runlock.RLock()
	for _, p := range p.running {
		close(p.in)
	}
	p.runlock.RUnlock()
	p.protoWG.Wait()
}

// writeProtoMsg sends the given message on behalf of the given named protocol.
func (p *Peer) writeProtoMsg(protoName string, msg Msg) error {
	p.runlock.RLock()
	proto, ok := p.running[protoName]
	p.runlock.RUnlock()
	if !ok {
		return fmt.Errorf("protocol %s not handled by peer", protoName)
	}
	if msg.Code >= proto.maxcode {
		return newPeerError(errInvalidMsgCode, "code %x is out of range for protocol %q", msg.Code, protoName)
	}
	msg.Code += proto.offset
	return p.writeMsg(msg, msgWriteTimeout)
}

// writeMsg writes a message to the connection.
func (p *Peer) writeMsg(msg Msg, timeout time.Duration) error {
	p.writeMu.Lock()
	defer p.writeMu.Unlock()
	p.conn.SetWriteDeadline(time.Now().Add(timeout))
	if err := writeMsg(p.bufconn, msg); err != nil {
		return newPeerError(errWrite, "%v", err)
	}
	return p.bufconn.Flush()
}

type proto struct {
	name            string
	in              chan Msg
	maxcode, offset uint64
	peer            *Peer
}

func (rw *proto) WriteMsg(msg Msg) error {
	if msg.Code >= rw.maxcode {
		return newPeerError(errInvalidMsgCode, "not handled")
	}
	msg.Code += rw.offset
	return rw.peer.writeMsg(msg, msgWriteTimeout)
}

func (rw *proto) EncodeMsg(code uint64, data ...interface{}) error {
	return rw.WriteMsg(NewMsg(code, data...))
}

func (rw *proto) ReadMsg() (Msg, error) {
	msg, ok := <-rw.in
	if !ok {
		return msg, io.EOF
	}
	msg.Code -= rw.offset
	return msg, nil
}

// eofSignal wraps a reader with eof signaling. the eof channel is
// closed when the wrapped reader returns an error or when count bytes
// have been read.
//
type eofSignal struct {
	wrapped io.Reader
	count   int64
	eof     chan<- struct{}
}

// note: when using eofSignal to detect whether a message payload
// has been read, Read might not be called for zero sized messages.

func (r *eofSignal) Read(buf []byte) (int, error) {
	n, err := r.wrapped.Read(buf)
	r.count -= int64(n)
	if (err != nil || r.count <= 0) && r.eof != nil {
		r.eof <- struct{}{} // tell Peer that msg has been consumed
		r.eof = nil
	}
	return n, err
}
