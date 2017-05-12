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
	"fmt"
	"time"

	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/log"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
	"github.com/ethereum/go-ethereum/p2p/simulations/adapters"
)

const (
	ProtocolName       = "bzz"
	Version            = 0
	NetworkId          = 322 // BZZ in l33t
	ProtocolMaxMsgSize = 10 * 1024 * 1024
)

// the Addr interface that peerPool needs
type Addr interface {
	OverlayPeer
	Over() []byte
	Under() []byte
	String() string
}

// Peer interface represents an live peer connection
type Peer interface {
	Addr                   // the address of a peer
	Conn                   // the live connection (protocols.Peer)
	LastActive() time.Time // last time active
}

// Conn interface represents an live peer connection
type Conn interface {
	ID() discover.NodeID                                       // the key that uniquely identifies the Node for the peerPool
	Handshake(interface{}, time.Duration) (interface{}, error) // can send messages
	Send(interface{}) error                                    // can send messages
	Drop(error)                                                // disconnect this peer
	Register(interface{}, func(interface{}) error) uint64      // register message-handler callbacks
	DisconnectHook(func(error))                                // register message-handler callbacks
	Run() error                                                // the run function to run a protocol
}

// bzzPeer is the bzz protocol view of a protocols.Peer (itself an extension of p2p.Peer)
// implements the Peer interface and all interfaces Peer implements: Addr, OverlayPeer
type bzzPeer struct {
	Conn                 // represents the connection for online peers
	localAddr  *bzzAddr  // local Peers address
	*bzzAddr             // remote address -> implements Addr interface = protocols.Peer
	lastActive time.Time // time is updated whenever mutexes are releasing
}

// Off returns the overlay peer record for offline persistance
func (self *bzzPeer) Off() OverlayAddr {
	return self.bzzAddr
}

// LastActive returns the time the peer was last active
func (self *bzzPeer) LastActive() time.Time {
	return self.lastActive
}

// BzzCodeMap compiles the message codes and message types bzz wire protocol.
// note each call to Register can start a new series (initial code is arg1)
// the initial offset for a series is arbitrary (to ensure u)
func BzzCodeMap(msgs ...interface{}) *protocols.CodeMap {
	ct := protocols.NewCodeMap(ProtocolName, Version, ProtocolMaxMsgSize)
	ct.Register(0, &bzzHandshake{})
	ct.Register(1, msgs...)
	return ct
}

// NewBzz is the protocol constructor
// returns p2p.Protocol that is to be offered by the node.Service
func NewBzz(over, under []byte, ct *protocols.CodeMap, services func(*bzzPeer) error, peerInfo func(id discover.NodeID) interface{}, nodeInfo func() interface{}) *p2p.Protocol {
	run := func(p *protocols.Peer) error {
		bee := &bzzPeer{
			Conn:      p,
			localAddr: &bzzAddr{over, under},
		}
		// protocol handshake and its validation
		// sets remote peer address
		err := bee.bzzHandshake()
		if err != nil {
			log.Error(fmt.Sprintf("handshake error in peer %v: %v", bee.ID(), err))
			return err
		}

		// mount external service models on the peer connection (swap, sync, hive)
		if services != nil {
			err = services(bee)
			if err != nil {
				log.Error(fmt.Sprintf("protocol service error for peer %v: %v", bee.ID(), err))
				return err
			}
		}

		return bee.Run()
	}

	return protocols.NewProtocol(ProtocolName, Version, run, ct, peerInfo, nodeInfo)
}

/*
 Handshake

* Version: 8 byte integer version of the protocol
* NetworkID: 8 byte integer network identifier
* Addr: the address advertised by the node including underlay and overlay connecctions
*/
type bzzHandshake struct {
	Version   uint64
	NetworkId uint64
	Addr      *bzzAddr
}

func (self *bzzHandshake) String() string {
	return fmt.Sprintf("Handshake: Version: %v, NetworkId: %v, Addr: %v", self.Version, self.NetworkId, self.Addr)
}

// bzzAddr implements the PeerAddr interface
type bzzAddr struct {
	OAddr []byte
	UAddr []byte
}

// implements OverlayPeer interface to be used in pot package
func (self *bzzAddr) Address() []byte {
	return self.OAddr
}

func (self *bzzAddr) Over() []byte {
	return self.OAddr
}

func (self *bzzAddr) Under() []byte {
	return self.UAddr
}

func (self *bzzAddr) On(p OverlayConn) OverlayConn {
	bp := p.(*bzzPeer)
	bp.bzzAddr = self
	return bp
}

func (self *bzzAddr) Update(a OverlayAddr) OverlayAddr {
	return &bzzAddr{self.OAddr, a.(Addr).Under()}
}

func (self *bzzAddr) String() string {
	return fmt.Sprintf("%x <%x>", self.OAddr, self.UAddr)
}

// bzzHandshake negotiates the bzz master handshake
// and validates the response, returns error when
// mismatch/incompatibility is evident
func (self *bzzPeer) bzzHandshake() error {

	lhs := &bzzHandshake{
		Version:   uint64(Version),
		NetworkId: uint64(NetworkId),
		Addr:      self.localAddr,
	}

	hs, err := self.Handshake(lhs, time.Second)
	if err != nil {
		log.Error(fmt.Sprintf("handshake failed: %v", err))
		return err
	}

	rhs := hs.(*bzzHandshake)
	self.bzzAddr = rhs.Addr
	err = checkBzzHandshake(rhs)
	if err != nil {
		log.Error(fmt.Sprintf("handshake between %v and %v  failed: %v", self.localAddr, self.bzzAddr, err))
		return err
	}
	return nil
}

// checkBzzHandshake checks for the validity and compatibility of the remote handshake
func checkBzzHandshake(rhs *bzzHandshake) error {

	if NetworkId != rhs.NetworkId {
		return fmt.Errorf("network id mismatch %d (!= %d)", rhs.NetworkId, NetworkId)
	}

	if Version != rhs.Version {
		return fmt.Errorf("version mismatch %d (!= %d)", rhs.Version, Version)
	}

	return nil
}

// RandomAddr is a utility method generating an address from a public key
func RandomAddr() *bzzAddr {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("unable to generate key")
	}
	pubkey := crypto.FromECDSAPub(&key.PublicKey)
	var id discover.NodeID
	copy(id[:], pubkey[1:])
	return &bzzAddr{
		OAddr: crypto.Keccak256(pubkey[1:]),
		UAddr: id[:],
	}
}

// NewNodeIdFromAddr transforms the underlay address to an adapters.NodeId
func NewNodeIdFromAddr(addr Addr) *adapters.NodeId {
	return adapters.NewNodeId(addr.Under())
}

// NewAddrFromNodeId constucts a bzzAddr from an adapters.NodeId
// the overlay address is derived as the hash of the nodeId
func NewAddrFromNodeId(n *adapters.NodeId) *bzzAddr {
	id := n.NodeID
	return &bzzAddr{crypto.Keccak256(id[:]), id[:]}
}
