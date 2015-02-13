package p2p

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	baseProtocolVersion    = 2
	baseProtocolLength     = uint64(16)
	baseProtocolMaxMsgSize = 10 * 1024 * 1024

	disconnectGracePeriod = 2 * time.Second
)

const (
	// devp2p message codes
	handshakeMsg = 0x00
	discMsg      = 0x01
	pingMsg      = 0x02
	pongMsg      = 0x03
	getPeersMsg  = 0x04
	peersMsg     = 0x05
)

// handshake is the RLP structure of the protocol handshake.
type handshake struct {
	Version    uint64
	Name       string
	Caps       []Cap
	ListenPort uint64
	NodeID     discover.NodeID
}

// Peer represents a connected remote node.
type Peer struct {
	// Peers have all the log methods.
	// Use them to display messages related to the peer.
	*logger.Logger

	infoMu sync.Mutex
	name   string
	caps   []Cap

	ourID, remoteID *discover.NodeID
	ourName         string

	rw *frameRW

	// These fields maintain the running protocols.
	protocols []Protocol
	runlock   sync.RWMutex // protects running
	running   map[string]*proto

	// disables protocol handshake, for testing
	noHandshake bool

	protoWG  sync.WaitGroup
	protoErr chan error
	closed   chan struct{}
	disc     chan DiscReason
}

// NewPeer returns a peer for testing purposes.
func NewPeer(id discover.NodeID, name string, caps []Cap) *Peer {
	conn, _ := net.Pipe()
	peer := newPeer(conn, nil, "", nil, &id)
	peer.setHandshakeInfo(name, caps)
	close(peer.closed) // ensures Disconnect doesn't block
	return peer
}

// ID returns the node's public key.
func (p *Peer) ID() discover.NodeID {
	return *p.remoteID
}

// Name returns the node name that the remote node advertised.
func (p *Peer) Name() string {
	// this needs a lock because the information is part of the
	// protocol handshake.
	p.infoMu.Lock()
	name := p.name
	p.infoMu.Unlock()
	return name
}

// Caps returns the capabilities (supported subprotocols) of the remote peer.
func (p *Peer) Caps() []Cap {
	// this needs a lock because the information is part of the
	// protocol handshake.
	p.infoMu.Lock()
	caps := p.caps
	p.infoMu.Unlock()
	return caps
}

// RemoteAddr returns the remote address of the network connection.
func (p *Peer) RemoteAddr() net.Addr {
	return p.rw.RemoteAddr()
}

// LocalAddr returns the local address of the network connection.
func (p *Peer) LocalAddr() net.Addr {
	return p.rw.LocalAddr()
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
	return fmt.Sprintf("Peer %.8x %v", p.remoteID[:], p.RemoteAddr())
}

func newPeer(conn net.Conn, protocols []Protocol, ourName string, ourID, remoteID *discover.NodeID) *Peer {
	logtag := fmt.Sprintf("Peer %.8x %v", remoteID[:], conn.RemoteAddr())
	return &Peer{
		Logger:    logger.NewLogger(logtag),
		rw:        newFrameRW(conn, msgWriteTimeout),
		ourID:     ourID,
		ourName:   ourName,
		remoteID:  remoteID,
		protocols: protocols,
		running:   make(map[string]*proto),
		disc:      make(chan DiscReason),
		protoErr:  make(chan error),
		closed:    make(chan struct{}),
	}
}

func (p *Peer) setHandshakeInfo(name string, caps []Cap) {
	p.infoMu.Lock()
	p.name = name
	p.caps = caps
	p.infoMu.Unlock()
}

func (p *Peer) run() DiscReason {
	var readErr = make(chan error, 1)
	defer p.closeProtocols()
	defer close(p.closed)

	go func() { readErr <- p.readLoop() }()

	if !p.noHandshake {
		if err := writeProtocolHandshake(p.rw, p.ourName, *p.ourID, p.protocols); err != nil {
			p.DebugDetailf("Protocol handshake error: %v\n", err)
			p.rw.Close()
			return DiscProtocolError
		}
	}

	// Wait for an error or disconnect.
	var reason DiscReason
	select {
	case err := <-readErr:
		// We rely on protocols to abort if there is a write error. It
		// might be more robust to handle them here as well.
		p.DebugDetailf("Read error: %v\n", err)
		p.rw.Close()
		return DiscNetworkError

	case err := <-p.protoErr:
		reason = discReasonForError(err)
	case reason = <-p.disc:
	}
	p.politeDisconnect(reason)

	// Wait for readLoop. It will end because conn is now closed.
	<-readErr
	p.Debugf("Disconnected: %v\n", reason)
	return reason
}

func (p *Peer) politeDisconnect(reason DiscReason) {
	done := make(chan struct{})
	go func() {
		EncodeMsg(p.rw, discMsg, uint(reason))
		// Wait for the other side to close the connection.
		// Discard any data that they send until then.
		io.Copy(ioutil.Discard, p.rw)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(disconnectGracePeriod):
	}
	p.rw.Close()
}

func (p *Peer) readLoop() error {
	if !p.noHandshake {
		if err := readProtocolHandshake(p, p.rw); err != nil {
			return err
		}
	}
	for {
		msg, err := p.rw.ReadMsg()
		if err != nil {
			return err
		}
		if err = p.handle(msg); err != nil {
			return err
		}
	}
	return nil
}

func (p *Peer) handle(msg Msg) error {
	switch {
	case msg.Code == pingMsg:
		msg.Discard()
		go EncodeMsg(p.rw, pongMsg)
	case msg.Code == discMsg:
		var reason DiscReason
		// no need to discard or for error checking, we'll close the
		// connection after this.
		rlp.Decode(msg.Payload, &reason)
		p.Disconnect(DiscRequested)
		return discRequestedError(reason)
	case msg.Code < baseProtocolLength:
		// ignore other base protocol messages
		return msg.Discard()
	default:
		// it's a subprotocol message
		proto, err := p.getProto(msg.Code)
		if err != nil {
			return fmt.Errorf("msg code out of range: %v", msg.Code)
		}
		proto.in <- msg
	}
	return nil
}

func readProtocolHandshake(p *Peer, rw MsgReadWriter) error {
	// read and handle remote handshake
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}
	if msg.Code == discMsg {
		// disconnect before protocol handshake is valid according to the
		// spec and we send it ourself if Server.addPeer fails.
		var reason DiscReason
		rlp.Decode(msg.Payload, &reason)
		return discRequestedError(reason)
	}
	if msg.Code != handshakeMsg {
		return newPeerError(errProtocolBreach, "expected handshake, got %x", msg.Code)
	}
	if msg.Size > baseProtocolMaxMsgSize {
		return newPeerError(errInvalidMsg, "message too big")
	}
	var hs handshake
	if err := msg.Decode(&hs); err != nil {
		return err
	}
	// validate handshake info
	if hs.Version != baseProtocolVersion {
		return newPeerError(errP2PVersionMismatch, "required version %d, received %d\n",
			baseProtocolVersion, hs.Version)
	}
	if hs.NodeID == *p.remoteID {
		return newPeerError(errPubkeyForbidden, "node ID mismatch")
	}
	// TODO: remove Caps with empty name
	p.setHandshakeInfo(hs.Name, hs.Caps)
	p.startSubprotocols(hs.Caps)
	return nil
}

func writeProtocolHandshake(w MsgWriter, name string, id discover.NodeID, ps []Protocol) error {
	var caps []interface{}
	for _, proto := range ps {
		caps = append(caps, proto.cap())
	}
	return EncodeMsg(w, handshakeMsg, baseProtocolVersion, name, caps, 0, id)
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
	p.DebugDetailf("Starting protocol %s/%d\n", impl.Name, impl.Version)
	rw := &proto{
		name:    impl.Name,
		in:      make(chan Msg),
		offset:  offset,
		maxcode: impl.Length,
		w:       p.rw,
	}
	p.protoWG.Add(1)
	go func() {
		err := impl.Run(p, rw)
		if err == nil {
			p.DebugDetailf("Protocol %s/%d returned\n", impl.Name, impl.Version)
			err = errors.New("protocol returned")
		} else {
			p.DebugDetailf("Protocol %s/%d error: %v\n", impl.Name, impl.Version, err)
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
// this exists because of Server.Broadcast.
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
	return p.rw.WriteMsg(msg)
}

type proto struct {
	name            string
	in              chan Msg
	maxcode, offset uint64
	w               MsgWriter
}

func (rw *proto) WriteMsg(msg Msg) error {
	if msg.Code >= rw.maxcode {
		return newPeerError(errInvalidMsgCode, "not handled")
	}
	msg.Code += rw.offset
	return rw.w.WriteMsg(msg)
}

func (rw *proto) ReadMsg() (Msg, error) {
	msg, ok := <-rw.in
	if !ok {
		return msg, io.EOF
	}
	msg.Code -= rw.offset
	return msg, nil
}
