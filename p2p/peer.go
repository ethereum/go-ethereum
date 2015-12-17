// Copyright 2014 The go-ethereum Authors
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

package p2p

import (
	"errors"
	"fmt"
	"io"
	"net"
	"sort"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/rlp"
)

const pingInterval = 15 * time.Second

// Peer represents a connected remote node.
type Peer struct {
	// contains an element for each running subprotocol (excluding devp2p).
	running []*protoRW

	conn     *conn
	wg       sync.WaitGroup
	protoErr chan error
	closed   chan struct{}
	disc     chan DiscReason
}

// NewPeer returns a peer for testing purposes.
func NewPeer(id discover.NodeID, name string, caps []Cap) *Peer {
	pipe, _ := net.Pipe()
	randomPriv, _ := crypto.GenerateKey()
	dc := newDevConn(pipe, randomPriv, nil)
	conn := &conn{transport: dc, id: id, caps: caps, name: name}
	peer := newPeer(conn, nil)
	close(peer.closed) // ensures Disconnect doesn't block
	return peer
}

// ID returns the node's public key.
func (p *Peer) ID() discover.NodeID {
	return p.conn.id
}

// Name returns the node name that the remote node advertised.
func (p *Peer) Name() string {
	return p.conn.name
}

// Caps returns the capabilities (supported subprotocols) of the remote peer.
func (p *Peer) Caps() []Cap {
	// TODO: maybe return copy
	return p.conn.caps
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
	return fmt.Sprintf("Peer %x %v", p.conn.id[:8], p.RemoteAddr())
}

func newPeer(conn *conn, protocols []Protocol) *Peer {
	protomap := matchProtocols(protocols, conn.caps)
	p := &Peer{
		conn:     conn,
		running:  protomap,
		disc:     make(chan DiscReason),
		protoErr: make(chan error, len(protomap)+1), // protocols + pingLoop
		closed:   make(chan struct{}),
	}
	return p
}

func (p *Peer) run() DiscReason {
	var (
		writeErr  = make(chan error, len(p.running))
		readErr   = make(chan error, 1)
		reason    DiscReason
		requested bool
		// While most of the code works with the transport interface so it
		// can be tested, using the connection requires an actual
		// *devConn.
		devconn = p.conn.transport.(*devConn)
	)

	// Ensure that the RLPx handshake is done. The only time this will
	// actually do anything is while testing because the tests don't
	// trigger the handshake explicitly.
	if err := devconn.Handshake(); err != nil {
		return DiscProtocolError
	}

	p.wg.Add(2)
	go p.readLoop(devconn.protocols[0], readErr)
	go p.pingLoop(devconn.protocols[0])

	// Start all protocol handlers.
	p.startProtocols(devconn, writeErr)

	// Wait for an error or disconnect.
loop:
	for {
		select {
		case err := <-writeErr:
			if err != nil {
				glog.V(logger.Detail).Infof("%v: write error: %v\n", p, err)
				reason = DiscNetworkError
				break loop
			}
		case err := <-readErr:
			if r, ok := err.(DiscReason); ok {
				glog.V(logger.Debug).Infof("%v: remote requested disconnect: %v\n", p, r)
				requested = true
				reason = r
			} else {
				glog.V(logger.Detail).Infof("%v: read error: %v\n", p, err)
				reason = DiscNetworkError
			}
			break loop
		case err := <-p.protoErr:
			reason = discReasonForError(err)
			glog.V(logger.Debug).Infof("%v: protocol error: %v (%v)\n", p, err, reason)
			break loop
		case reason = <-p.disc:
			glog.V(logger.Debug).Infof("%v: locally requested disconnect: %v\n", p, reason)
			break loop
		}
	}

	close(p.closed)
	p.conn.close(reason)
	p.wg.Wait()
	if requested {
		reason = DiscRequested
	}
	return reason
}

func (p *Peer) pingLoop(devp2p *devProtocol) {
	ping := time.NewTicker(pingInterval)
	defer p.wg.Done()
	defer ping.Stop()
	for {
		select {
		case <-ping.C:
			if err := SendItems(devp2p, pingMsg); err != nil {
				p.protoErr <- err
				return
			}
		case <-p.closed:
			return
		}
	}
}

func (p *Peer) readLoop(devp2p *devProtocol, errc chan<- error) {
	defer p.wg.Done()
	for {
		msg, err := devp2p.ReadMsg()
		if err != nil {
			errc <- err
			return
		}
		if err = p.handle(devp2p, msg); err != nil {
			errc <- err
			return
		}
	}
}

func (p *Peer) handle(devp2p *devProtocol, msg Msg) (err error) {
	switch {
	case msg.Code == pingMsg:
		msg.Discard()
		go SendItems(devp2p, pongMsg)
		return
	case msg.Code == discMsg:
		var reason [1]DiscReason
		// This is the last message. We don't need to discard or
		// check errors because, the connection will be closed after it.
		rlp.Decode(msg.Payload, &reason)
		return reason[0]
	case msg.Code < baseProtocolLength:
		// ignore other base protocol messages
		return msg.Discard()
	default:
		// Dispatch as subprotocol message by message code offset.
		// This is how dispatch worked before chunking was implemented.
		proto, err := p.getProto(msg.Code)
		msg.Code -= proto.offset
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

// matchProtocols creates protoRWs for matching named subprotocols.
func matchProtocols(protocols []Protocol, caps []Cap) []*protoRW {
	sort.Sort(capsByNameAndVersion(caps))
	i := 0
	offset := baseProtocolLength
	var result []*protoRW
outer:
	for _, cap := range caps {
		for _, proto := range protocols {
			if proto.Name == cap.Name && proto.Version == cap.Version {
				if i > 0 && result[i-1].Name == cap.Name {
					// If the previous match was for the same protocol
					// (with a lower version), reset the offset and replace it.
					offset -= result[i-1].Protocol.Length
				} else {
					// Otherwise, append a new protocol.
					result = append(result, nil)
					i++
				}
				result[i-1] = &protoRW{Protocol: proto, offset: offset}
				offset += proto.Length
				continue outer
			}
		}
	}
	return result
}

func (p *Peer) startProtocols(dc *devConn, writeErr chan<- error) {
	switch dc.Version() {
	case 5:
		// Acknowledge the protocols on the RLPx layer. This creates
		// *devProtocol wrappers, dc.protocols[i] contains entries in
		// range 1..len(p.running).
		dc.addProtocols(len(p.running))
		for i, proto := range p.running {
			proto.offset = 0
			proto.werr = writeErr
			proto.rw = dc.protocols[i+1]
		}
	case 4:
		// This is a legacy connection with offset-based dispatch.
		for _, proto := range p.running {
			proto.closed = p.closed
			proto.in = make(chan Msg)
			proto.werr = writeErr
			proto.rw = dc.protocols[0]
		}
	default:
		panic("conn has no version")
	}
	// Spawn Run for all protocols.
	p.wg.Add(len(p.running))
	for _, proto := range p.running {
		proto := proto
		glog.V(logger.Detail).Infof("%v: Starting protocol %s/%d\n", p, proto.Name, proto.Version)
		go func() {
			err := proto.Run(p, proto)
			if err == nil {
				glog.V(logger.Detail).Infof("%v: Protocol %s/%d returned\n", p, proto.Name, proto.Version)
				err = errors.New("protocol returned")
			} else if err != io.EOF {
				glog.V(logger.Detail).Infof("%v: Protocol %s/%d error: %v\n", p, proto.Name, proto.Version, err)
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
		if proto.offset > 0 && code >= proto.offset && code < proto.offset+proto.Length {
			return proto, nil
		}
	}
	return nil, newPeerError(errInvalidMsgCode, "%d", code)
}

type protoRW struct {
	Protocol
	offset uint64
	rw     MsgReadWriter
	werr   chan<- error // for write results

	// for RLPx V4 offset-based dispatch
	in     chan Msg        // receices read messages
	closed <-chan struct{} // receives when peer is shutting down
	index  uint16
}

func (rw *protoRW) WriteMsg(msg Msg) error {
	if msg.Code >= rw.Length {
		return newPeerError(errInvalidMsgCode, "not handled")
	}
	msg.Code += rw.offset
	err := rw.rw.WriteMsg(msg)
	// Report write status back to Peer.run. It will initiate shutdown
	// if the error is non-nil otherwise. The calling protocol should
	// exit soon after, but might not return the error correctly.
	if err != nil {
		rw.werr <- err
	}
	// TODO: maybe make the error sticky to prevent further writes
	return err
}

func (rw *protoRW) ReadMsg() (Msg, error) {
	if rw.offset == 0 {
		// RLPx version 5
		return rw.rw.ReadMsg()
	}
	// RLPx version 4
	select {
	case msg := <-rw.in:
		return msg, nil
	case <-rw.closed:
		return Msg{}, io.EOF
	}
}

// PeerInfo represents a short summary of the information known about a connected
// peer. Sub-protocol independent fields are contained and initialized here, with
// protocol specifics delegated to all connected sub-protocols.
type PeerInfo struct {
	ID      string   `json:"id"`   // Unique node identifier (also the encryption key)
	Name    string   `json:"name"` // Name of the node, including client type, version, OS, custom data
	Caps    []string `json:"caps"` // Sum-protocols advertised by this particular peer
	Network struct {
		LocalAddress  string `json:"localAddress"`  // Local endpoint of the TCP data connection
		RemoteAddress string `json:"remoteAddress"` // Remote endpoint of the TCP data connection
	} `json:"network"`
	Protocols map[string]interface{} `json:"protocols"` // Sub-protocol specific metadata fields
}

// Info gathers and returns a collection of metadata known about a peer.
func (p *Peer) Info() *PeerInfo {
	// Gather the protocol capabilities
	var caps []string
	for _, cap := range p.Caps() {
		caps = append(caps, cap.String())
	}
	// Assemble the generic peer metadata
	info := &PeerInfo{
		ID:        p.ID().String(),
		Name:      p.Name(),
		Caps:      caps,
		Protocols: make(map[string]interface{}),
	}
	info.Network.LocalAddress = p.LocalAddr().String()
	info.Network.RemoteAddress = p.RemoteAddr().String()

	// Gather all the running protocol infos
	for _, proto := range p.running {
		protoInfo := interface{}("unknown")
		if query := proto.Protocol.PeerInfo; query != nil {
			if metadata := query(p.ID()); metadata != nil {
				protoInfo = metadata
			} else {
				protoInfo = "handshake"
			}
		}
		info.Protocols[proto.Name] = protoInfo
	}
	return info
}
