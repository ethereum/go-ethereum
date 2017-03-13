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
	"github.com/ethereum/go-ethereum/logger"
	"github.com/ethereum/go-ethereum/logger/glog"
	"github.com/ethereum/go-ethereum/p2p"
	"github.com/ethereum/go-ethereum/p2p/adapters"
	"github.com/ethereum/go-ethereum/p2p/discover"
	"github.com/ethereum/go-ethereum/p2p/protocols"
)

const (
	ProtocolName       = "bzz"
	Version            = 0
	NetworkId          = 322 // BZZ in l33t
	ProtocolMaxMsgSize = 10 * 1024 * 1024
)

// bzz is the bzz protocol view of a protocols.Peer (itself an extension of p2p.Peer)
type bzzPeer struct {
	*protocols.Peer
	network    adapters.NodeAdapter
	localAddr  *peerAddr
	*peerAddr  // remote address
	lastActive time.Time
	//peers      map[discover.NodeID]bool
}

func (self *bzzPeer) LastActive() time.Time {
	return self.lastActive
}


// implemented by peerAddr
type PeerAddr interface {
	OverlayAddr() []byte
	UnderlayAddr() []byte
}

// the Peer interface that peerPool needs
type Peer interface {
	PeerAddr
	String() string      // pretty printable the Node
	ID() discover.NodeID // the key that uniquely identifies the Node for the peerPool
	Send(interface{}) error                               // can send messages
	Drop(error)                                           // disconnect this peer
	Register(interface{}, func(interface{}) error) uint64 // register message-handler callbacks
	DisconnectHook(func(interface{}) error)
}

func BzzCodeMap(msgs ...interface{}) *protocols.CodeMap {
	ct := protocols.NewCodeMap(ProtocolName, Version, ProtocolMaxMsgSize)
	ct.Register(&bzzHandshake{})
	ct.Register(msgs...)
	return ct
}

// Bzz is the protocol constructor
// returns p2p.Protocol that is to be offered by the node.Service
func Bzz(localAddr []byte, na adapters.NodeAdapter, ct *protocols.CodeMap, services func(Peer) error, peerInfo func(id discover.NodeID) interface{}, nodeInfo func() interface{}) *p2p.Protocol {
	run := func(p *protocols.Peer) error {
		addr := &peerAddr{localAddr, na.LocalAddr()}

		bee := &bzzPeer{
			Peer:      p,
			network:   na,
			localAddr: addr,
		}
		// protocol handshake and its validation
		// sets remote peer address
		err := bee.bzzHandshake()
		if err != nil {
			glog.V(6).Infof("handshake error in peer %v: %v", bee.ID(), err)
			return err
		}

		// mount external service models on the peer connection (swap, sync, hive)
		if services != nil {
			err = services(bee)
			if err != nil {
				glog.V(6).Infof("protocol service error for peer %v: %v", bee.ID(), err)
				return err
			}
		}

		return bee.Run()
	}

	return protocols.NewProtocol(ProtocolName, Version, run, na, ct, peerInfo, nodeInfo)
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
	Addr      *peerAddr
}

func (self *bzzHandshake) String() string {
	return fmt.Sprintf("Handshake: Version: %v, NetworkId: %v, Addr: %v", self.Version, self.NetworkId, self.Addr)
}

// peerAddr implements the PeerAddress interface
type peerAddr struct {
	OAddr []byte
	UAddr []byte
}

func (self *peerAddr) OverlayAddr() []byte {
	return self.OAddr
}

func (self *peerAddr) UnderlayAddr() []byte {
	return self.UAddr
}

func (self *peerAddr) String() string {
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

	hs, err := self.Handshake(lhs)
	if err != nil {
		glog.V(6).Infof("handshake failed: %v", err)
		return err
	}

	rhs := hs.(*bzzHandshake)
	err = checkBzzHandshake(rhs)
	if err != nil {
		glog.V(6).Infof("handshake between %v and %v  failed: %v", self.localAddr, self.peerAddr, err)
		return err
	}

	addr := rhs.Addr
	// Addr returns the remote address of the network connection.
	// with rlpx use this to set adverrtised IP
	self.localAddr.UAddr, err = self.network.ParseAddr(self.localAddr.UAddr, self.RemoteAddr().String())
	if err != nil {
		return err
	}

	glog.V(logger.Debug).Infof("self: advertised net address: %x, local address: %v\npeer: advertised: %v, remote address: %v\n", self.network.LocalAddr(), self.LocalAddr(), NodeId(addr), self.RemoteAddr())
	self.peerAddr = addr
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
func RandomAddr() *peerAddr {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("unable to generate key")
	}
	pubkey := crypto.FromECDSAPub(&key.PublicKey)
	var id discover.NodeID
	copy(id[:], pubkey[1:])
	return &peerAddr{
		OAddr: crypto.Keccak256(pubkey[1:]),
		UAddr: id[:],
	}
}

// NodeId transforms the underlay address to an adapters.NodeId
func NodeId(addr PeerAddr) *adapters.NodeId {
	return adapters.NewNodeId(addr.UnderlayAddr())
}

// NewPeerAddrFromNodeId constucts a peerAddr from an adapters.NodeId
// the overlay address is derived as the hash of the nodeId
func NewPeerAddrFromNodeId(n *adapters.NodeId) *peerAddr {
	id := n.NodeID
	return &peerAddr{
		OAddr: crypto.Keccak256(id[:]),
		UAddr: id[:],
	}
}
