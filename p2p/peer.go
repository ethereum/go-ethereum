package p2p

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rlp"
)

const (
	baseProtocolVersion    = 4
	baseProtocolLength     = uint64(16)
	baseProtocolMaxMsgSize = 10 * 1024 * 1024

	pingInterval = 15 * time.Second
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
	conn    net.Conn
	rw      *conn
	running map[string]*protoRW

	wg       sync.WaitGroup
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
	protomap := matchProtocols(protocols, conn.Caps, conn)
	p := &Peer{
		conn:     fd,
		rw:       conn,
		running:  protomap,
		disc:     make(chan DiscReason),
		protoErr: make(chan error, len(protomap)+1), // protocols + pingLoop
		closed:   make(chan struct{}),
	}
	return p
}

func (p *Peer) run() DiscReason {
	readErr := make(chan error, 1)
	p.wg.Add(2)
	go p.readLoop(readErr)
	go p.pingLoop()

	p.startProtocols()

	// Wait for an error or disconnect.
	var reason DiscReason
	select {
	case err := <-readErr:
		if r, ok := err.(DiscReason); ok {
			reason = r
		} else {
			// Note: We rely on protocols to abort if there is a write
			// error. It might be more robust to handle them here as well.
			glog.V(logger.Detail).Infof("%v: Read error: %v\n", p, err)
			reason = DiscNetworkError
		}
	case err := <-p.protoErr:
		reason = discReasonForError(err)
	case reason = <-p.disc:
		p.politeDisconnect(reason)
		reason = DiscRequested
	}

	close(p.closed)
	p.wg.Wait()
	glog.V(logger.Debug).Infof("%v: Disconnected: %v\n", p, reason)
	return reason
}

func (p *Peer) politeDisconnect(reason DiscReason) {
	if reason != DiscNetworkError {
		SendItems(p.rw, discMsg, uint(reason))
	}
	p.conn.Close()
}

func (p *Peer) pingLoop() {
	ping := time.NewTicker(pingInterval)
	defer p.wg.Done()
	defer ping.Stop()
	for {
		select {
		case <-ping.C:
			if err := SendItems(p.rw, pingMsg); err != nil {
				p.protoErr <- err
				return
			}
		case <-p.closed:
			return
		}
	}
}

func (p *Peer) readLoop(errc chan<- error) {
	defer p.wg.Done()
	for {
		msg, err := p.rw.ReadMsg()
		if err != nil {
			errc <- err
			return
		}
		msg.ReceivedAt = time.Now()
		if err = p.handle(msg); err != nil {
			errc <- err
			return
		}
	}
}

func (p *Peer) handle(msg Msg) error {
	switch {
	case msg.Code == pingMsg:
		msg.Discard()
		go SendItems(p.rw, pongMsg)
	case msg.Code == discMsg:
		var reason [1]DiscReason
		// This is the last message. We don't need to discard or
		// check errors because, the connection will be closed after it.
		rlp.Decode(msg.Payload, &reason)
		glog.V(logger.Debug).Infof("%v: Disconnect Requested: %v\n", p, reason[0])
		return reason[0]
	case msg.Code < baseProtocolLength:
		// ignore other base protocol messages
		return msg.Discard()
	default:
		// it's a subprotocol message
		proto, err := p.getProto(msg.Code)
		if err != nil {
			return fmt.Errorf("msg code out of range: %v", msg.Code)
		}
		select {
		case proto.in <- msg:
			return nil
		case <-p.closed:
			return io.EOF
		}
	}
	return nil
}

func countMatchingProtocols(protocols []Protocol, caps []Cap) int {
	n := 0
	for _, cap := range caps {
		for _, proto := range protocols {
			if proto.Name == cap.Name && proto.Version == cap.Version {
				n++
			}
		}
	}
	return n
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
	p.wg.Add(len(p.running))
	for _, proto := range p.running {
		proto := proto
		proto.closed = p.closed
		glog.V(logger.Detail).Infof("%v: Starting protocol %s/%d\n", p, proto.Name, proto.Version)
		go func() {
			err := proto.Run(p, proto)
			if err == nil {
				glog.V(logger.Detail).Infof("%v: Protocol %s/%d returned\n", p, proto.Name, proto.Version)
				err = errors.New("protocol returned")
			} else if err != io.EOF {
				glog.V(logger.Detail).Infof("%v: Protocol %s/%d error: \n", p, proto.Name, proto.Version, err)
			}
			p.protoErr <- err
			p.wg.Done()
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

type protoRW struct {
	Protocol
	in     chan Msg
	closed <-chan struct{}
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
	select {
	case msg := <-rw.in:
		msg.Code -= rw.offset
		return msg, nil
	case <-rw.closed:
		return Msg{}, io.EOF
	}
}
