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
	baseProtocolVersion    = 3
	baseProtocolLength     = uint64(16)
	baseProtocolMaxMsgSize = 10 * 1024 * 1024

	pingInterval          = 15 * time.Second
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

// Peer represents a connected remote node.
type Peer struct {
	// Peers have all the log methods.
	// Use them to display messages related to the peer.
	*logger.Logger

	conn    net.Conn
	rw      *conn
	running map[string]*protoRW

	protoWG  sync.WaitGroup
	protoErr chan error
	closed   chan struct{}
	disc     chan DiscReason
}

// NewPeer returns a peer for testing purposes.
func NewPeer(id discover.NodeID, name string, caps []Cap) *Peer {
	pipe, _ := net.Pipe()
	msgpipe, _ := MsgPipe()
	conn := &conn{msgpipe, &protoHandshake{ID: id, Name: name, Caps: caps}}
	peer := newPeer(pipe, conn, nil)
	close(peer.closed) // ensures Disconnect doesn't block
	return peer
}

// ID returns the node's public key.
func (p *Peer) ID() discover.NodeID {
	return p.rw.ID
}

// Name returns the node name that the remote node advertised.
func (p *Peer) Name() string {
	return p.rw.Name
}

// Caps returns the capabilities (supported subprotocols) of the remote peer.
func (p *Peer) Caps() []Cap {
	// TODO: maybe return copy
	return p.rw.Caps
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
	return fmt.Sprintf("Peer %.8x %v", p.rw.ID[:], p.RemoteAddr())
}

func newPeer(fd net.Conn, conn *conn, protocols []Protocol) *Peer {
	logtag := fmt.Sprintf("Peer %.8x %v", conn.ID[:], fd.RemoteAddr())
	p := &Peer{
		Logger:   logger.NewLogger(logtag),
		conn:     fd,
		rw:       conn,
		running:  matchProtocols(protocols, conn.Caps, conn),
		disc:     make(chan DiscReason),
		protoErr: make(chan error),
		closed:   make(chan struct{}),
	}
	return p
}

func (p *Peer) run() DiscReason {
	var readErr = make(chan error, 1)
	defer p.closeProtocols()
	defer close(p.closed)

	p.startProtocols()
	go func() { readErr <- p.readLoop() }()

	ping := time.NewTicker(pingInterval)
	defer ping.Stop()

	// Wait for an error or disconnect.
	var reason DiscReason
loop:
	for {
		select {
		case <-ping.C:
			go func() {
				if err := SendItems(p.rw, pingMsg); err != nil {
					p.protoErr <- err
					return
				}
			}()
		case err := <-readErr:
			// We rely on protocols to abort if there is a write error. It
			// might be more robust to handle them here as well.
			p.DebugDetailf("Read error: %v\n", err)
			p.conn.Close()
			return DiscNetworkError
		case err := <-p.protoErr:
			reason = discReasonForError(err)
			break loop
		case reason = <-p.disc:
			break loop
		}
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
		SendItems(p.rw, discMsg, uint(reason))
		// Wait for the other side to close the connection.
		// Discard any data that they send until then.
		io.Copy(ioutil.Discard, p.conn)
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(disconnectGracePeriod):
	}
	p.conn.Close()
}

func (p *Peer) readLoop() error {
	for {
		p.conn.SetDeadline(time.Now().Add(frameReadTimeout))
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
		go SendItems(p.rw, pongMsg)
	case msg.Code == discMsg:
		var reason [1]DiscReason
		// no need to discard or for error checking, we'll close the
		// connection after this.
		rlp.Decode(msg.Payload, &reason)
		p.Debugf("Disconnect requested: %v\n", reason[0])
		p.Disconnect(DiscRequested)
		return discRequestedError(reason[0])
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

// matchProtocols creates structures for matching named subprotocols.
func matchProtocols(protocols []Protocol, caps []Cap, rw MsgReadWriter) map[string]*protoRW {
	sort.Sort(capsByName(caps))
	offset := baseProtocolLength
	result := make(map[string]*protoRW)
outer:
	for _, cap := range caps {
		for _, proto := range protocols {
			if proto.Name == cap.Name && proto.Version == cap.Version && result[cap.Name] == nil {
				result[cap.Name] = &protoRW{Protocol: proto, offset: offset, in: make(chan Msg), w: rw}
				offset += proto.Length
				continue outer
			}
		}
	}
	return result
}

func (p *Peer) startProtocols() {
	for _, proto := range p.running {
		proto := proto
		p.DebugDetailf("Starting protocol %s/%d\n", proto.Name, proto.Version)
		p.protoWG.Add(1)
		go func() {
			err := proto.Run(p, proto)
			if err == nil {
				p.DebugDetailf("Protocol %s/%d returned\n", proto.Name, proto.Version)
				err = errors.New("protocol returned")
			} else {
				p.DebugDetailf("Protocol %s/%d error: %v\n", proto.Name, proto.Version, err)
			}
			select {
			case p.protoErr <- err:
			case <-p.closed:
			}
			p.protoWG.Done()
		}()
	}
}

// getProto finds the protocol responsible for handling
// the given message code.
func (p *Peer) getProto(code uint64) (*protoRW, error) {
	for _, proto := range p.running {
		if code >= proto.offset && code < proto.offset+proto.Length {
			return proto, nil
		}
	}
	return nil, newPeerError(errInvalidMsgCode, "%d", code)
}

func (p *Peer) closeProtocols() {
	for _, p := range p.running {
		close(p.in)
	}
	p.protoWG.Wait()
}

// writeProtoMsg sends the given message on behalf of the given named protocol.
// this exists because of Server.Broadcast.
func (p *Peer) writeProtoMsg(protoName string, msg Msg) error {
	proto, ok := p.running[protoName]
	if !ok {
		return fmt.Errorf("protocol %s not handled by peer", protoName)
	}
	if msg.Code >= proto.Length {
		return newPeerError(errInvalidMsgCode, "code %x is out of range for protocol %q", msg.Code, protoName)
	}
	msg.Code += proto.offset
	return p.rw.WriteMsg(msg)
}

type protoRW struct {
	Protocol

	in     chan Msg
	offset uint64
	w      MsgWriter
}

func (rw *protoRW) WriteMsg(msg Msg) error {
	if msg.Code >= rw.Length {
		return newPeerError(errInvalidMsgCode, "not handled")
	}
	msg.Code += rw.offset
	return rw.w.WriteMsg(msg)
}

func (rw *protoRW) ReadMsg() (Msg, error) {
	msg, ok := <-rw.in
	if !ok {
		return msg, io.EOF
	}
	msg.Code -= rw.offset
	return msg, nil
}
