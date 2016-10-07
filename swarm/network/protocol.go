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
type bzz struct {
	*protocols.Peer
	hive       PeerPool
	network    adapters.NetAdapter
	localAddr  *peerAddr
	*peerAddr  // remote address
	lastActive time.Time
}

func (self *bzz) LastActive() time.Time {
	return self.lastActive
}

// implemented by peerAddr and peerAddr
type NodeAddr interface {
	RemoteOverlayAddr() []byte
	RemoteUnderlayAddr() []byte
}

// the Node interface that peerPool needs
type Node interface {
	NodeAddr
	String() string      // pretty printable the Node
	ID() discover.NodeID // the key that uniquely identifies the Node for the peerPool

	Send(interface{}) error                             // can send messages
	Drop()                                              // disconnect this peer
	Register(interface{}, func(interface{}) error) uint // register message-handler callbacks
}

// PeerPool is the interface for the connectivity manager
// directly interacts with the p2p server to suggest connections
type PeerPool interface {
	Add(Node) error
	Remove(Node)
}

type PeerInfo interface {
	Info() interface{}
	PeerInfo(discover.NodeID) interface{}
}

func bzzCodeMap(msgs ...interface{}) *protocols.CodeMap {
	ct := protocols.NewCodeMap(ProtocolName, Version, ProtocolMaxMsgSize)
	ct.Register(&bzzHandshake{})
	ct.Register(msgs...)
	return ct
}

// Bzz is the protocol constructor
// returns p2p.Protocol that is to be offered by the node.Service
func Bzz(localAddr []byte, hive PeerPool, na adapters.NetAdapter, m adapters.Messenger, ct *protocols.CodeMap, services func(Node) error) *p2p.Protocol {
	// handle handshake

	run := func(p *p2p.Peer, rw p2p.MsgReadWriter) error {

		peer := protocols.NewPeer(p, rw, ct, m, func() { na.Disconnect(p, rw) })
		addr := &peerAddr{localAddr, na.LocalAddr()}
		glog.V(6).Infof("local addr: %v", addr)

		bee := &bzz{Peer: peer, hive: hive, network: na, localAddr: addr}
		// protocol handshake and its validation
		// sets remote peer address
		glog.V(6).Infof("launch handshake")
		err := bee.bzzHandshake()
		if err != nil {
			return err
		}

		// mount external service models on the peer connection (swap, sync)
		if services != nil {
			err = services(bee)
			if err != nil {
				return err
			}
		}

		err = hive.Add(bee)
		if err != nil {
			return err
		}
		glog.V(6).Infof("added peer '%v' to hive", bee.ID())

		defer hive.Remove(bee)
		// # syncer
		// # request handler
		// # swap

		return bee.Run()
	}
	var info func() interface{}
	var peerInfo func(discover.NodeID) interface{}

	if o, ok := hive.(PeerInfo); ok {
		info = o.Info
		peerInfo = o.PeerInfo
	}

	return &p2p.Protocol{
		Name:     ProtocolName,
		Version:  Version,
		Length:   ct.Length(),
		Run:      run,
		NodeInfo: info,
		PeerInfo: peerInfo,
	}
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

type peerAddr struct {
	OverlayAddr  []byte
	UnderlayAddr []byte
}

func (self *peerAddr) RemoteOverlayAddr() []byte {
	return self.OverlayAddr
}

func (self *peerAddr) RemoteUnderlayAddr() []byte {
	return self.UnderlayAddr
}

func (self *peerAddr) String() string {
	return fmt.Sprintf("%x <%x>", self.OverlayAddr, self.UnderlayAddr)
}

// bzzHandshake negotiates the bzz master handshake
// and validates the response, returns error when
// mismatch/incompatibility is evident
func (self *bzz) bzzHandshake() error {

	lhs := &bzzHandshake{
		Version:   uint64(Version),
		NetworkId: uint64(NetworkId),
		Addr:      self.localAddr,
	}

	glog.V(6).Infof("launch handshake")
	hs, err := self.Handshake(lhs)
	if err != nil {
		return err
	}

	rhs := hs.(*bzzHandshake)
	glog.V(6).Infof("check handshake")
	err = checkBzzHandshake(rhs)
	if err != nil {
		return err
	}

	addr := rhs.Addr
	// RemoteAddr returns the remote address of the network connection.
	// with rlpx use this to set adverrtised IP
	self.localAddr.UnderlayAddr, err = self.network.ParseAddr(self.localAddr.UnderlayAddr, self.RemoteAddr().String())
	glog.V(6).Infof("error is %v", err)
	if err != nil {
		return err
	}

	glog.V(logger.Debug).Infof("self: advertised net address: %x, local address: %v\npeer: advertised: %v, remote address: %v\n", self.network.LocalAddr(), self.LocalAddr(), NodeID(addr), self.RemoteAddr())
	self.peerAddr = addr
	return nil

}

func checkBzzHandshake(rhs *bzzHandshake) error {

	if NetworkId != rhs.NetworkId {
		return fmt.Errorf("network id mismatch %d (!= %d)", rhs.NetworkId, NetworkId)
	}

	if Version != rhs.Version {
		return fmt.Errorf("version mismatch %d (!= %d)", rhs.Version, Version)
	}

	return nil
}

func randomAddr() *peerAddr {
	key, err := crypto.GenerateKey()
	if err != nil {
		panic("unable to generate key")
	}
	pubkey := crypto.FromECDSAPub(&key.PublicKey)
	var id discover.NodeID
	copy(id[:], pubkey[1:])
	return &peerAddr{
		OverlayAddr:  crypto.Keccak256(pubkey[1:]),
		UnderlayAddr: id[:],
	}
}

func NodeID(addr NodeAddr) *discover.NodeID {
	// id := discover.MustHexID(string(addr.UnderlayAddr))
	var id discover.NodeID
	copy(id[:], addr.RemoteUnderlayAddr())
	return &id
}

func nodeID2addr(id *discover.NodeID) *peerAddr {
	return &peerAddr{
		OverlayAddr:  crypto.Keccak256(id[:]),
		UnderlayAddr: id[:],
	}
}
