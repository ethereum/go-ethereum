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

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/rpc"
	"github.com/ethereum/go-ethereum/swarm/log"
	"github.com/ethereum/go-ethereum/swarm/state"
)

const (
	DefaultNetworkID = 3
	// ProtocolMaxMsgSize maximum allowed message size
	ProtocolMaxMsgSize = 10 * 1024 * 1024
	// timeout for waiting
	bzzHandshakeTimeout = 3000 * time.Millisecond
)

// BzzSpec is the spec of the generic swarm handshake
var BzzSpec = &protocols.Spec{
	Name:       "bzz",
	Version:    5,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		HandshakeMsg{},
	},
}

// DiscoverySpec is the spec for the bzz discovery subprotocols
var DiscoverySpec = &protocols.Spec{
	Name:       "hive",
	Version:    5,
	MaxMsgSize: 10 * 1024 * 1024,
	Messages: []interface{}{
		peersMsg{},
		subPeersMsg{},
	},
}

// Addr interface that peerPool needs
type Addr interface {
	OverlayPeer
	Over() []byte
	Under() []byte
	String() string
	Update(OverlayAddr) OverlayAddr
}

// Peer interface represents an live peer connection
type Peer interface {
	Addr                   // the address of a peer
	Conn                   // the live connection (protocols.Peer)
	LastActive() time.Time // last time active
}

// Conn interface represents an live peer connection
type Conn interface {
	ID() discover.NodeID                                                                  // the key that uniquely identifies the Node for the peerPool
	Handshake(context.Context, interface{}, func(interface{}) error) (interface{}, error) // can send messages
	Send(context.Context, interface{}) error                                              // can send messages
	Drop(error)                                                                           // disconnect this peer
	Run(func(context.Context, interface{}) error) error                                   // the run function to run a protocol
	Off() OverlayAddr
}

// BzzConfig captures the config params used by the hive
type BzzConfig struct {
	OverlayAddr  []byte // base address of the overlay network
	UnderlayAddr []byte // node's underlay address
	HiveParams   *HiveParams
	NetworkID    uint64
}

// Bzz is the swarm protocol bundle
type Bzz struct {
	*Hive
	NetworkID    uint64
	localAddr    *BzzAddr
	mtx          sync.Mutex
	handshakes   map[discover.NodeID]*HandshakeMsg
	streamerSpec *protocols.Spec
	streamerRun  func(*BzzPeer) error
}

// NewBzz is the swarm protocol constructor
// arguments
// * bzz config
// * overlay driver
// * peer store
func NewBzz(config *BzzConfig, kad Overlay, store state.Store, streamerSpec *protocols.Spec, streamerRun func(*BzzPeer) error) *Bzz {
	return &Bzz{
		Hive:         NewHive(config.HiveParams, kad, store),
		NetworkID:    config.NetworkID,
		localAddr:    &BzzAddr{config.OverlayAddr, config.UnderlayAddr},
		handshakes:   make(map[discover.NodeID]*HandshakeMsg),
		streamerRun:  streamerRun,
		streamerSpec: streamerSpec,
	}
}

// UpdateLocalAddr updates underlayaddress of the running node
func (b *Bzz) UpdateLocalAddr(byteaddr []byte) *BzzAddr {
	b.localAddr = b.localAddr.Update(&BzzAddr{
		UAddr: byteaddr,
		OAddr: b.localAddr.OAddr,
	}).(*BzzAddr)
	return b.localAddr
}

// NodeInfo returns the node's overlay address
func (b *Bzz) NodeInfo() interface{} {
	return b.localAddr.Address()
}

// Protocols return the protocols swarm offers
// Bzz implements the node.Service interface
// * handshake/hive
// * discovery
func (b *Bzz) Protocols() []p2p.Protocol {
	protocol := []p2p.Protocol{
		{
			Name:     BzzSpec.Name,
			Version:  BzzSpec.Version,
			Length:   BzzSpec.Length(),
			Run:      b.runBzz,
			NodeInfo: b.NodeInfo,
		},
		{
			Name:     DiscoverySpec.Name,
			Version:  DiscoverySpec.Version,
			Length:   DiscoverySpec.Length(),
			Run:      b.RunProtocol(DiscoverySpec, b.Hive.Run),
			NodeInfo: b.Hive.NodeInfo,
			PeerInfo: b.Hive.PeerInfo,
		},
	}
	if b.streamerSpec != nil && b.streamerRun != nil {
		protocol = append(protocol, p2p.Protocol{
			Name:    b.streamerSpec.Name,
			Version: b.streamerSpec.Version,
			Length:  b.streamerSpec.Length(),
			Run:     b.RunProtocol(b.streamerSpec, b.streamerRun),
		})
	}
	return protocol
}

// APIs returns the APIs offered by bzz
// * hive
// Bzz implements the node.Service interface
func (b *Bzz) APIs() []rpc.API {
	return []rpc.API{{
		Namespace: "hive",
		Version:   "3.0",
		Service:   b.Hive,
	}}
}

// RunProtocol is a wrapper for swarm subprotocols
// returns a p2p protocol run function that can be assigned to p2p.Protocol#Run field
// arguments:
// * p2p protocol spec
// * run function taking BzzPeer as argument
//   this run function is meant to block for the duration of the protocol session
//   on return the session is terminated and the peer is disconnected
// the protocol waits for the bzz handshake is negotiated
// the overlay address on the BzzPeer is set from the remote handshake
func (b *Bzz) RunProtocol(spec *protocols.Spec, run func(*BzzPeer) error) func(*p2p.Peer, p2p.MsgReadWriter) error {
	return func(p *p2p.Peer, rw p2p.MsgReadWriter) error {
		// wait for the bzz protocol to perform the handshake
		handshake, _ := b.GetHandshake(p.ID())
		defer b.removeHandshake(p.ID())
		select {
		case <-handshake.done:
		case <-time.After(bzzHandshakeTimeout):
			return fmt.Errorf("%08x: %s protocol timeout waiting for handshake on %08x", b.BaseAddr()[:4], spec.Name, p.ID().Bytes()[:4])
		}
		if handshake.err != nil {
			return fmt.Errorf("%08x: %s protocol closed: %v", b.BaseAddr()[:4], spec.Name, handshake.err)
		}
		// the handshake has succeeded so construct the BzzPeer and run the protocol
		peer := &BzzPeer{
			Peer:       protocols.NewPeer(p, rw, spec),
			localAddr:  b.localAddr,
			BzzAddr:    handshake.peerAddr,
			lastActive: time.Now(),
		}
		return run(peer)
	}
}

// performHandshake implements the negotiation of the bzz handshake
// shared among swarm subprotocols
func (b *Bzz) performHandshake(p *protocols.Peer, handshake *HandshakeMsg) error {
	ctx, cancel := context.WithTimeout(context.Background(), bzzHandshakeTimeout)
	defer func() {
		close(handshake.done)
		cancel()
	}()
	rsh, err := p.Handshake(ctx, handshake, b.checkHandshake)
	if err != nil {
		handshake.err = err
		return err
	}
	handshake.peerAddr = rsh.(*HandshakeMsg).Addr
	return nil
}

// runBzz is the p2p protocol run function for the bzz base protocol
// that negotiates the bzz handshake
func (b *Bzz) runBzz(p *p2p.Peer, rw p2p.MsgReadWriter) error {
	handshake, _ := b.GetHandshake(p.ID())
	if !<-handshake.init {
		return fmt.Errorf("%08x: bzz already started on peer %08x", b.localAddr.Over()[:4], ToOverlayAddr(p.ID().Bytes())[:4])
	}
	close(handshake.init)
	defer b.removeHandshake(p.ID())
	peer := protocols.NewPeer(p, rw, BzzSpec)
	err := b.performHandshake(peer, handshake)
	if err != nil {
		log.Warn(fmt.Sprintf("%08x: handshake failed with remote peer %08x: %v", b.localAddr.Over()[:4], ToOverlayAddr(p.ID().Bytes())[:4], err))

		return err
	}
	// fail if we get another handshake
	msg, err := rw.ReadMsg()
	if err != nil {
		return err
	}
	msg.Discard()
	return errors.New("received multiple handshakes")
}

// BzzPeer is the bzz protocol view of a protocols.Peer (itself an extension of p2p.Peer)
// implements the Peer interface and all interfaces Peer implements: Addr, OverlayPeer
type BzzPeer struct {
	*protocols.Peer           // represents the connection for online peers
	localAddr       *BzzAddr  // local Peers address
	*BzzAddr                  // remote address -> implements Addr interface = protocols.Peer
	lastActive      time.Time // time is updated whenever mutexes are releasing
}

func NewBzzTestPeer(p *protocols.Peer, addr *BzzAddr) *BzzPeer {
	return &BzzPeer{
		Peer:      p,
		localAddr: addr,
		BzzAddr:   NewAddrFromNodeID(p.ID()),
	}
}

// Off returns the overlay peer record for offline persistence
func (p *BzzPeer) Off() OverlayAddr {
	return p.BzzAddr
}

// LastActive returns the time the peer was last active
func (p *BzzPeer) LastActive() time.Time {
	return p.lastActive
}

/*
 Handshake

* Version: 8 byte integer version of the protocol
* NetworkID: 8 byte integer network identifier
* Addr: the address advertised by the node including underlay and overlay connecctions
*/
type HandshakeMsg struct {
	Version   uint64
	NetworkID uint64
	Addr      *BzzAddr

	// peerAddr is the address received in the peer handshake
	peerAddr *BzzAddr

	init chan bool
	done chan struct{}
	err  error
}

// String pretty prints the handshake
func (bh *HandshakeMsg) String() string {
	return fmt.Sprintf("Handshake: Version: %v, NetworkID: %v, Addr: %v", bh.Version, bh.NetworkID, bh.Addr)
}

// Perform initiates the handshake and validates the remote handshake message
func (b *Bzz) checkHandshake(hs interface{}) error {
	rhs := hs.(*HandshakeMsg)
	if rhs.NetworkID != b.NetworkID {
		return fmt.Errorf("network id mismatch %d (!= %d)", rhs.NetworkID, b.NetworkID)
	}
	if rhs.Version != uint64(BzzSpec.Version) {
		return fmt.Errorf("version mismatch %d (!= %d)", rhs.Version, BzzSpec.Version)
	}
	return nil
}

// removeHandshake removes handshake for peer with peerID
// from the bzz handshake store
func (b *Bzz) removeHandshake(peerID discover.NodeID) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	delete(b.handshakes, peerID)
}

// GetHandshake returns the bzz handhake that the remote peer with peerID sent
func (b *Bzz) GetHandshake(peerID discover.NodeID) (*HandshakeMsg, bool) {
	b.mtx.Lock()
	defer b.mtx.Unlock()
	handshake, found := b.handshakes[peerID]
	if !found {
		handshake = &HandshakeMsg{
			Version:   uint64(BzzSpec.Version),
			NetworkID: b.NetworkID,
			Addr:      b.localAddr,
			init:      make(chan bool, 1),
			done:      make(chan struct{}),
		}
		// when handhsake is first created for a remote peer
		// it is initialised with the init
		handshake.init <- true
		b.handshakes[peerID] = handshake
	}

	return handshake, found
}

// BzzAddr implements the PeerAddr interface
type BzzAddr struct {
	OAddr []byte
	UAddr []byte
}

// Address implements OverlayPeer interface to be used in Overlay
func (a *BzzAddr) Address() []byte {
	return a.OAddr
}

// Over returns the overlay address
func (a *BzzAddr) Over() []byte {
	return a.OAddr
}

// Under returns the underlay address
func (a *BzzAddr) Under() []byte {
	return a.UAddr
}

// ID returns the nodeID from the underlay enode address
func (a *BzzAddr) ID() discover.NodeID {
	return discover.MustParseNode(string(a.UAddr)).ID
}

// Update updates the underlay address of a peer record
func (a *BzzAddr) Update(na OverlayAddr) OverlayAddr {
	return &BzzAddr{a.OAddr, na.(Addr).Under()}
}

// String pretty prints the address
func (a *BzzAddr) String() string {
	return fmt.Sprintf("%x <%s>", a.OAddr, a.UAddr)
}

// RandomAddr is a utility method generating an address from a public key
func RandomAddr() *BzzAddr {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("unable to generate key")
	}
	pubkey := crypto.FromECDSAPub(&key.PublicKey)
	var id discover.NodeID
	copy(id[:], pubkey[1:])
	return NewAddrFromNodeID(id)
}

// NewNodeIDFromAddr transforms the underlay address to an adapters.NodeID
func NewNodeIDFromAddr(addr Addr) discover.NodeID {
	log.Info(fmt.Sprintf("uaddr=%s", string(addr.Under())))
	node := discover.MustParseNode(string(addr.Under()))
	return node.ID
}

// NewAddrFromNodeID constucts a BzzAddr from a discover.NodeID
// the overlay address is derived as the hash of the nodeID
func NewAddrFromNodeID(id discover.NodeID) *BzzAddr {
	return &BzzAddr{
		OAddr: ToOverlayAddr(id.Bytes()),
		UAddr: []byte(discover.NewNode(id, net.IP{127, 0, 0, 1}, 30303, 30303).String()),
	}
}

// NewAddrFromNodeIDAndPort constucts a BzzAddr from a discover.NodeID and port uint16
// the overlay address is derived as the hash of the nodeID
func NewAddrFromNodeIDAndPort(id discover.NodeID, host net.IP, port uint16) *BzzAddr {
	return &BzzAddr{
		OAddr: ToOverlayAddr(id.Bytes()),
		UAddr: []byte(discover.NewNode(id, host, port, port).String()),
	}
}

// ToOverlayAddr creates an overlayaddress from a byte slice
func ToOverlayAddr(id []byte) []byte {
	return crypto.Keccak256(id)
}
